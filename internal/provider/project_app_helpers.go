// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errCreatingProjectApp         = "Error attaching project app"
	errReadingProjectApp          = "Error reading project app"
	errUpdatingProjectApp         = "Error updating project app"
	errDeletingProjectApp         = "Error detaching project app"
	errMissingProjectAppProjectID = "Missing project ID for project app"
	errInvalidProjectAppConfig    = "Invalid project app configuration"

	errProjectAppProjectIDRequired = "Project ID is required to manage the project app"
	errProjectAppImportIDInvalid   = "Invalid import ID format. Expected: {project_id}/{app_name}"
	errProjectAppReplacedFmt       = "app %q was %s outside of Terraform (%s, now %s), so the plan no longer applies; " +
		"refresh the state, for example with an apply without -refresh=false"
	errProjectAppUsageMismatchFmt = "YouTrack attached app %q to project %q instead of %q; project_id must be the project entity ID (for example 0-1), not its short name"

	projectAppImportSeparator = "/"
	projectAppImportParts     = 2

	projectAppAttrProjectID = "project_id"
	projectAppAttrAppName   = "app_name"
)

// apply attaches the app named in m to its project and sets it enabled or
// disabled, then records the result in m. The app is looked up by name on every
// call, so an app that was reinstalled under a new ID is still found.
func (r *projectAppResource) apply(ctx context.Context, m *projectAppResourceModel) error {
	app, err := r.client.GetAppByName(ctx, m.AppName.ValueString())
	if err != nil {
		return fmt.Errorf("could not find app %q installed on YouTrack: %w", m.AppName.ValueString(), err)
	}

	projectID := m.ProjectID.ValueString()
	if err = r.checkNotReplaced(ctx, m, app.ID); err != nil {
		return err
	}

	usage, err := r.setAppUsage(ctx, app.ID, projectID, m.Enabled.ValueBool())
	if err != nil {
		return err
	}

	if usage.Project != nil && usage.Project.ID != "" && usage.Project.ID != projectID {
		return fmt.Errorf(errProjectAppUsageMismatchFmt, m.AppName.ValueString(), usage.Project.ID, projectID)
	}

	m.fromAPIModel(app.ID, usage)
	return nil
}

// setAppUsage attaches the app to the project when needed and brings its usage
// to the wanted enabled state. An existing usage is reused, never duplicated.
func (r *projectAppResource) setAppUsage(ctx context.Context, appID, projectID string, enabled bool) (*youtrack.AppUsage, error) {
	if enabled {
		return r.client.EnableAppForProject(ctx, appID, projectID)
	}

	usage, err := r.client.DisableAppForProject(ctx, appID, projectID)
	if err != nil || usage != nil {
		return usage, err
	}

	attached, err := r.client.AttachAppToProject(ctx, appID, projectID)
	if err != nil {
		return nil, err
	}
	if !attached.Enabled {
		return attached, nil
	}

	return r.client.SetAppUsageEnabled(ctx, appID, attached.ID, false)
}

// checkNotReplaced fails when the app or its usage in the project is no longer
// the one in the plan, which happens when the app was reinstalled or detached
// outside of Terraform and the plan was made without a refresh. It runs before
// any write: the framework would only reject the new ID after the change was
// made. On create, the plan has no IDs yet and there is nothing to check.
func (r *projectAppResource) checkNotReplaced(ctx context.Context, m *projectAppResourceModel, appID string) error {
	if !helpers.HasResourceID(m.ID) {
		return nil
	}

	if helpers.HasResourceID(m.AppID) && m.AppID.ValueString() != appID {
		return fmt.Errorf(errProjectAppReplacedFmt, m.AppName.ValueString(), "reinstalled", "app "+m.AppID.ValueString(), appID)
	}

	usage, err := r.client.GetAppUsageForProject(ctx, appID, m.ProjectID.ValueString())
	if err != nil {
		return err
	}

	now := "none"
	if usage != nil {
		now = usage.ID
	}
	if now != m.ID.ValueString() {
		return fmt.Errorf(errProjectAppReplacedFmt, m.AppName.ValueString(), "detached from the project", "usage "+m.ID.ValueString(), now)
	}

	return nil
}

// findUsage returns the app ID and the app's usage for the project in m, or a
// nil usage when the app is not attached to it. It tries the app ID from state
// first and falls back to the name when that ID no longer exists, which is the
// case after the app was reinstalled. An error matching youtrack.IsNotFound
// means the app is no longer installed at all.
func (r *projectAppResource) findUsage(ctx context.Context, m projectAppResourceModel) (string, *youtrack.AppUsage, error) {
	projectID := m.ProjectID.ValueString()

	if appID := m.AppID.ValueString(); appID != "" {
		usage, err := r.client.GetAppUsageForProject(ctx, appID, projectID)
		if !youtrack.IsNotFound(err) {
			return appID, usage, err
		}
	}

	app, err := r.client.GetAppByName(ctx, m.AppName.ValueString())
	if err != nil {
		return "", nil, err
	}

	usage, err := r.client.GetAppUsageForProject(ctx, app.ID, projectID)
	return app.ID, usage, err
}

// detach removes the app from the project. An app that is no longer installed
// or no longer attached is treated as already detached.
func (r *projectAppResource) detach(ctx context.Context, m projectAppResourceModel) error {
	appID, usage, err := r.findUsage(ctx, m)
	if err != nil {
		if youtrack.IsNotFound(err) {
			return nil
		}
		return err
	}

	if usage == nil {
		return nil
	}

	return r.client.DeleteAppUsage(ctx, appID, usage.ID)
}

// parseProjectAppImportID splits an import ID of the form {project_id}/{app_name}.
// Only the first separator splits, because app names such as
// @jetbrains/youtrack-workflow-due-date contain one while project IDs never do.
func parseProjectAppImportID(id string) (projectID, appName string, ok bool) {
	parts := strings.SplitN(id, projectAppImportSeparator, projectAppImportParts)
	if len(parts) != projectAppImportParts {
		return "", "", false
	}

	projectID, appName = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if projectID == "" || appName == "" {
		return "", "", false
	}

	return projectID, appName, true
}

// fromAPIModel records the usage in m. app_name and project_id keep their
// configured values: the app lookup also matches names case-insensitively, and
// overwriting them would turn that into a replacement on the next plan.
func (m *projectAppResourceModel) fromAPIModel(appID string, usage *youtrack.AppUsage) {
	m.ID = types.StringValue(usage.ID)
	m.AppID = types.StringValue(appID)
	m.Enabled = types.BoolValue(usage.Enabled)
}
