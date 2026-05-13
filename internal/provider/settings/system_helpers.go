package settings

import (
	"context"
	"fmt"
	"net/url"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadSystem = "Unable to Read YouTrack System Settings"
	errUpdatingSystem     = "Error updating system settings"
)

// getSystemSettingsAndHandleError fetches the system settings via API and handles errors.
func (r *systemSettingsResource) getSystemSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.SystemSettings, bool) {
	systemSettings, err := r.client.GetSystemSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadSystem,
			err.Error(),
		)
		return youtrack.SystemSettings{}, false
	}
	return systemSettings, true
}

// updateSystemSettingsModelWithTimestamp updates the model from response and sets timestamp.
func updateSystemSettingsModelWithTimestamp(systemSettings youtrack.SystemSettings, resourceModel *systemSettingsResourceModel) {
	*resourceModel = convertSystemSettingsToModel(systemSettings)
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// convertModelToSystemSettings converts a systemSettingsResourceModel to youtrack.SystemSettings.
func convertModelToSystemSettings(model systemSettingsResourceModel) youtrack.SystemSettings {
	return youtrack.SystemSettings{
		AdministratorEmail:        model.AdministratorEmail.ValueString(),
		MaxExportItems:            int(model.MaxExportItems.ValueInt64()),
		MaxUploadFileSize:         int(model.MaxUploadFileSize.ValueInt64()),
		AllowStatisticsCollection: model.AllowStatisticsCollection.ValueBool(),
		IsApplicationReadOnly:     model.IsApplicationReadOnly.ValueBool(),
		BaseUrl:                   model.BaseURL.ValueString(),
	}
}

// convertSystemSettingsToModel converts a youtrack.SystemSettings to systemSettingsResourceModel.
func convertSystemSettingsToModel(ss youtrack.SystemSettings) systemSettingsResourceModel {
	return systemSettingsResourceModel{
		ID:                        types.StringValue("global"),
		AdministratorEmail:        types.StringValue(ss.AdministratorEmail),
		MaxExportItems:            types.Int64Value(int64(ss.MaxExportItems)),
		MaxUploadFileSize:         types.Int64Value(int64(ss.MaxUploadFileSize)),
		AllowStatisticsCollection: types.BoolValue(ss.AllowStatisticsCollection),
		IsApplicationReadOnly:     types.BoolValue(ss.IsApplicationReadOnly),
		BaseURL:                   types.StringValue(ss.BaseUrl),
	}
}

// updateSystemSettingsAndHandleError updates the system settings via API and handles errors.
func (r *systemSettingsResource) updateAndHandleError(ctx context.Context, ss youtrack.SystemSettings, diagnostics *diag.Diagnostics) (youtrack.SystemSettings, bool) {
	systemSettings, err := r.client.UpdateSystemSettings(ctx, ss)
	if err != nil {
		diagnostics.AddError(
			errUpdatingSystem,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "system settings", err),
		)
		return youtrack.SystemSettings{}, false
	}
	return systemSettings, true
}

// Validation helper functions to reduce cognitive complexity

// validatePositiveInt64 validates that an int64 field is positive (> 0).
func validatePositiveInt64(value types.Int64, fieldPath path.Path, fieldDescription string, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return
	}

	if value.ValueInt64() <= 0 {
		diagnostics.AddAttributeError(
			fieldPath,
			fmt.Sprintf("Invalid %s Value", fieldDescription),
			fmt.Sprintf("The %s must be greater than 0.", fieldDescription),
		)
	}
}

// validateURLField validates that a string field contains a valid URL if not empty.
func validateURLField(value types.String, fieldPath path.Path, fieldDescription string, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return
	}

	urlStr := value.ValueString()
	if urlStr == "" {
		return
	}

	if _, err := url.Parse(urlStr); err != nil {
		diagnostics.AddAttributeError(
			fieldPath,
			helpers.ErrInvalidURL,
			fmt.Sprintf("The %s must be a valid URL: %s", fieldDescription, err.Error()),
		)
	}
}
