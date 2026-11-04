// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errCreatingAppGlobalConfig = "Error configuring app"
	errReadingAppGlobalConfig  = "Error reading app configuration"
	errUpdatingAppGlobalConfig = "Error updating app configuration"
	errInvalidAppGlobalConfig  = "Invalid app configuration"
	errMissingAppGlobalConfig  = "Missing app name for app configuration"

	errAppGlobalConfigAppNameRequired = "App name is required to manage the app configuration"
	errAppGlobalConfigReadFmt         = "Could not read app configuration: %v"

	errAppGlobalConfigSettingsInvalid = "settings must be a JSON object, for example the result of jsonencode()"
	errAppGlobalConfigNoVisibility    = "YouTrack did not report who may use the app, so its permitted groups were not written"
	errAppGlobalConfigReinstalledFmt  = "app %q was reinstalled outside of Terraform (ID %s, now %s), so the plan no longer applies; " +
		"refresh the state, for example with an apply without -refresh=false"

	appGlobalConfigAttrAppName  = "app_name"
	appGlobalConfigAttrSettings = "settings"

	// maskedSecret is what YouTrack returns in place of a setting declared
	// with "format": "secret".
	maskedSecret = "<***>"

	// entityIDKey identifies an entity setting, such as a user group, that
	// YouTrack expands with its name, type and other fields when it reads it.
	entityIDKey = "id"
)

// settingsChanged reports whether the settings have to be written: they are
// managed and differ from the prior state, or there is no prior state. Writing
// them replaces the whole document, so an unrelated change must not do it.
func (m *appGlobalConfigResourceModel) settingsChanged(prior *appGlobalConfigResourceModel) bool {
	return !m.Settings.IsNull() && (prior == nil || !m.Settings.Equal(prior.Settings))
}

// groupsChanged reports whether the permitted groups have to be written, on the
// same terms as settingsChanged.
func (m *appGlobalConfigResourceModel) groupsChanged(prior *appGlobalConfigResourceModel) bool {
	return !m.PermittedGroupIDs.IsNull() && (prior == nil || !m.PermittedGroupIDs.Equal(prior.PermittedGroupIDs))
}

// apply writes the configuration in m to the app named in it, then records the
// result in m. prior is the current state, nil on create: settings and permitted
// groups are only written when they changed. The app is looked up by name on
// every call, so an app that was reinstalled under a new ID is still found.
func (r *appGlobalConfigResource) apply(ctx context.Context, m, prior *appGlobalConfigResourceModel) error {
	app, err := r.client.GetAppByName(ctx, m.AppName.ValueString())
	if err != nil {
		return fmt.Errorf("could not find app %q installed on YouTrack: %w", m.AppName.ValueString(), err)
	}

	if err = m.checkNotReinstalled(prior, app.ID); err != nil {
		return err
	}

	enabled := m.Enabled.ValueBool()
	update := youtrack.AppGlobalConfigUpdate{Enabled: &enabled}
	if m.settingsChanged(prior) {
		update.GlobalSettings = json.RawMessage(m.Settings.ValueString())
	}

	current, err := r.client.SetAppGlobalConfig(ctx, app.ID, update)
	if err != nil {
		return err
	}

	if m.groupsChanged(prior) {
		if err = r.applyVisibility(ctx, app.ID, m, current); err != nil {
			return err
		}
	}

	m.ID = types.StringValue(app.ID)
	return nil
}

// checkNotReinstalled fails when the app in prior, the current state, is no
// longer the installed one: it was reinstalled outside of Terraform and the plan
// was made without a refresh. apply calls it before any write, because the
// framework would only reject the new ID after the app was changed.
func (m *appGlobalConfigResourceModel) checkNotReinstalled(prior *appGlobalConfigResourceModel, appID string) error {
	if prior == nil || !helpers.HasResourceID(prior.ID) || prior.ID.ValueString() == appID {
		return nil
	}

	return fmt.Errorf(errAppGlobalConfigReinstalledFmt, m.AppName.ValueString(), prior.ID.ValueString(), appID)
}

// applyVisibility replaces the permitted groups of the app. The permitted users
// are not managed here, so the ones in current, the configuration the app has
// now, are written back.
func (r *appGlobalConfigResource) applyVisibility(ctx context.Context, appID string, m *appGlobalConfigResourceModel, current *youtrack.AppGlobalConfig) error {
	var diags diag.Diagnostics
	groupIDs, ok := helpers.SetToStringSlice(ctx, m.PermittedGroupIDs, &diags)
	if !ok {
		return fmt.Errorf("could not read permitted_group_ids: %s", diags)
	}

	// without them, writing the groups would also remove every permitted user
	if current.VisibilitySettings == nil {
		return errors.New(errAppGlobalConfigNoVisibility)
	}

	var userIDs []string
	for _, user := range current.VisibilitySettings.PermittedUsers {
		userIDs = append(userIDs, user.ID)
	}

	_, err := r.client.SetAppVisibility(ctx, appID, groupIDs, userIDs)
	return err
}

// readConfig returns the app ID and global configuration for the app in m. It
// tries the app ID from state first and falls back to the name when that ID no
// longer exists, which is the case after the app was reinstalled. An error
// matching youtrack.IsNotFound means the app is no longer installed at all.
func (r *appGlobalConfigResource) readConfig(ctx context.Context, m appGlobalConfigResourceModel) (string, *youtrack.AppGlobalConfig, error) {
	if appID := m.ID.ValueString(); appID != "" {
		config, err := r.client.GetAppGlobalConfig(ctx, appID)
		if !youtrack.IsNotFound(err) {
			return appID, config, err
		}
	}

	app, err := r.client.GetAppByName(ctx, m.AppName.ValueString())
	if err != nil {
		return "", nil, err
	}

	config, err := r.client.GetAppGlobalConfig(ctx, app.ID)
	return app.ID, config, err
}

// fromAPIModel records what YouTrack reports in m. app_name keeps its
// configured value: the app lookup also matches names case-insensitively, and
// overwriting it would turn that into a replacement on the next plan. settings
// and permitted_group_ids are only read when the configuration manages them.
func (m *appGlobalConfigResourceModel) fromAPIModel(ctx context.Context, appID string, config *youtrack.AppGlobalConfig) error {
	m.ID = types.StringValue(appID)
	m.Enabled = types.BoolValue(config.Enabled)

	if !m.Settings.IsNull() {
		settings, err := reconcileSettings(m.Settings.ValueString(), string(config.GlobalSettings))
		if err != nil {
			return err
		}
		m.Settings = types.StringValue(settings)
	}

	if !m.PermittedGroupIDs.IsNull() {
		// not nil: SetValueFrom turns a nil slice into a null set, and "no groups" is an empty one
		ids := []string{}
		if config.VisibilitySettings != nil {
			for _, group := range config.VisibilitySettings.PermittedGroups {
				ids = append(ids, group.ID)
			}
		}
		sort.Strings(ids)

		set, diags := types.SetValueFrom(ctx, types.StringType, ids)
		if diags.HasError() {
			return fmt.Errorf("could not record permitted groups: %s", diags)
		}
		m.PermittedGroupIDs = set
	}

	return nil
}

// reconcileSettings compares the settings written in the configuration with the
// ones YouTrack reports and returns the document to keep in state. When every
// key that was written still holds, it returns written untouched, so formatting
// and key order do not show up as a difference. Otherwise it returns written
// with the keys that differ replaced by what YouTrack reports, which makes the
// next plan show the drift.
//
// Only the keys that were written are compared: a key that is not in written is
// not managed. Secret settings come back masked and always count as holding, so
// a secret changed outside of Terraform cannot be detected.
func reconcileSettings(written, read string) (string, error) {
	var writtenDoc, readDoc map[string]any
	if err := json.Unmarshal([]byte(written), &writtenDoc); err != nil {
		return "", fmt.Errorf("could not parse the settings in state: %w", err)
	}
	if read != "" {
		if err := json.Unmarshal([]byte(read), &readDoc); err != nil {
			return "", fmt.Errorf("could not parse the settings YouTrack returned: %w", err)
		}
	}

	drifted := false
	for key, writtenValue := range writtenDoc {
		readValue, present := readDoc[key]
		if present && settingMatches(writtenValue, readValue) {
			continue
		}
		// YouTrack does not store a setting written as null or "", it leaves it out
		if !present && (writtenValue == nil || writtenValue == "") {
			continue
		}
		drifted = true
		if present {
			writtenDoc[key] = readValue
		} else {
			delete(writtenDoc, key)
		}
	}

	if !drifted {
		return written, nil
	}

	out, err := json.Marshal(writtenDoc)
	if err != nil {
		return "", fmt.Errorf("could not encode the settings: %w", err)
	}

	return string(out), nil
}

// settingMatches reports whether the value YouTrack returned for a setting holds
// what was written. An entity (an object with an id) matches on its id alone,
// because YouTrack returns it expanded. A masked secret always matches.
func settingMatches(written, read any) bool {
	if s, ok := read.(string); ok && s == maskedSecret {
		return true
	}

	switch w := written.(type) {
	case map[string]any:
		return objectMatches(w, read)
	case []any:
		return arrayMatches(w, read)
	default:
		return reflect.DeepEqual(written, read)
	}
}

func objectMatches(written map[string]any, read any) bool {
	r, ok := read.(map[string]any)
	if !ok {
		return false
	}

	if id, isEntity := written[entityIDKey]; isEntity {
		return reflect.DeepEqual(id, r[entityIDKey])
	}

	for key, writtenValue := range written {
		readValue, present := r[key]
		if !present || !settingMatches(writtenValue, readValue) {
			return false
		}
	}

	return true
}

func arrayMatches(written []any, read any) bool {
	r, ok := read.([]any)
	if !ok {
		return false
	}

	// YouTrack stores a list of entities sorted by id and without duplicates
	if writtenIDs, isEntityList := entityIDs(written); isEntityList {
		readIDs, _ := entityIDs(r)
		return reflect.DeepEqual(writtenIDs, readIDs)
	}

	if len(r) != len(written) {
		return false
	}

	for i := range written {
		if !settingMatches(written[i], r[i]) {
			return false
		}
	}

	return true
}

// entityIDs returns the set of ids in a non-empty list made only of entities,
// and false for any other list.
func entityIDs(list []any) (map[any]bool, bool) {
	if len(list) == 0 {
		return nil, false
	}

	ids := make(map[any]bool, len(list))
	for _, element := range list {
		entity, ok := element.(map[string]any)
		if !ok {
			return nil, false
		}
		id, ok := entity[entityIDKey]
		if !ok {
			return nil, false
		}
		ids[id] = true
	}

	return ids, true
}
