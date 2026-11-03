// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
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

	errAppGlobalConfigAttributeEmptyFmt = "%s must not be empty"
	errAppGlobalConfigSettingsInvalid   = "settings must be a JSON object, for example the result of jsonencode()"

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

	enabled := m.Enabled.ValueBool()
	update := youtrack.AppGlobalConfigUpdate{Enabled: &enabled}
	if m.settingsChanged(prior) {
		update.GlobalSettings = json.RawMessage(m.Settings.ValueString())
	}

	if _, err = r.client.SetAppGlobalConfig(ctx, app.ID, update); err != nil {
		return err
	}

	if m.groupsChanged(prior) {
		if err = r.applyVisibility(ctx, app.ID, m); err != nil {
			return err
		}
	}

	m.ID = types.StringValue(app.ID)
	m.AppID = types.StringValue(app.ID)
	return nil
}

// applyVisibility replaces the permitted groups of the app. The permitted users
// are not managed here, so the ones the app already has are written back.
func (r *appGlobalConfigResource) applyVisibility(ctx context.Context, appID string, m *appGlobalConfigResourceModel) error {
	var diags diag.Diagnostics
	groupIDs, ok := helpers.SetToStringSlice(ctx, m.PermittedGroupIDs, &diags)
	if !ok {
		return fmt.Errorf("could not read permitted_group_ids: %s", diags)
	}

	current, err := r.client.GetAppGlobalConfig(ctx, appID)
	if err != nil {
		return err
	}

	var userIDs []string
	if current.VisibilitySettings != nil {
		for _, user := range current.VisibilitySettings.PermittedUsers {
			userIDs = append(userIDs, user.ID)
		}
	}

	_, err = r.client.SetAppVisibility(ctx, appID, groupIDs, userIDs)
	return err
}

// readConfig returns the app ID and global configuration for the app in m. It
// tries the app ID from state first and falls back to the name when that ID no
// longer exists, which is the case after the app was reinstalled. An error
// matching youtrack.IsNotFound means the app is no longer installed at all.
func (r *appGlobalConfigResource) readConfig(ctx context.Context, m appGlobalConfigResourceModel) (string, *youtrack.AppGlobalConfig, error) {
	if appID := m.AppID.ValueString(); appID != "" {
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
	m.AppID = types.StringValue(appID)
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
	if !ok || len(r) != len(written) {
		return false
	}

	for i := range written {
		if !settingMatches(written[i], r[i]) {
			return false
		}
	}

	return true
}
