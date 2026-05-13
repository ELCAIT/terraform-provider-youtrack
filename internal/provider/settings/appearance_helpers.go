package settings

import (
	"context"
	"fmt"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadAppearance = "Unable to Read YouTrack Appearance Settings"
	errUpdatingAppearance     = "Error updating appearance settings"
)

// getAppearanceSettingsAndHandleError fetches the appearance settings via API and handles errors.
func (r *appearanceSettingsResource) getAppearanceSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.AppearanceSettings, bool) {
	appearanceSettings, err := r.client.GetAppearanceSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadAppearance,
			err.Error(),
		)
		return youtrack.AppearanceSettings{}, false
	}
	return appearanceSettings, true
}

// updateAppearanceSettingsModelWithTimestamp updates the model from response and sets timestamp.
func updateAppearanceSettingsModelWithTimestamp(appearanceSettings youtrack.AppearanceSettings, resourceModel *appearanceSettingsResourceModel) {
	*resourceModel = convertAppearanceSettingsToModel(appearanceSettings)
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// convertModelToAppearanceSettings converts an appearanceSettingsResourceModel to youtrack.AppearanceSettings.
func convertModelToAppearanceSettings(model appearanceSettingsResourceModel) youtrack.AppearanceSettings {
	return youtrack.AppearanceSettings{
		DateFormat: youtrack.DateFormatDescriptor{
			ID: model.DateFormatID.ValueString(),
		},
		TimeZone: youtrack.TimeZoneDescriptor{
			ID: model.TimeZoneID.ValueString(),
		},
	}
}

// convertAppearanceSettingsToModel converts a youtrack.AppearanceSettings to appearanceSettingsResourceModel.
func convertAppearanceSettingsToModel(as youtrack.AppearanceSettings) appearanceSettingsResourceModel {
	return appearanceSettingsResourceModel{
		ID:                     types.StringValue(as.ID),
		DateFormatID:           types.StringValue(as.DateFormat.ID),
		DateFormatPresentation: types.StringValue(as.DateFormat.Presentation),
		DateFormatPattern:      types.StringValue(as.DateFormat.Pattern),
		DateFormatDatePattern:  types.StringValue(as.DateFormat.DatePattern),
		TimeZoneID:             types.StringValue(as.TimeZone.ID),
		TimeZonePresentation:   types.StringValue(as.TimeZone.Presentation),
		TimeZoneOffset:         types.Int64Value(int64(as.TimeZone.Offset)),
	}
}

// updateAppearanceSettingsAndHandleError updates the appearance settings via API and handles errors.
func (r *appearanceSettingsResource) updateAndHandleError(ctx context.Context, as youtrack.AppearanceSettings, diagnostics *diag.Diagnostics) (youtrack.AppearanceSettings, bool) {
	appearanceSettings, err := r.client.UpdateAppearanceSettings(ctx, as)
	if err != nil {
		diagnostics.AddError(
			errUpdatingAppearance,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "appearance settings", err),
		)
		return youtrack.AppearanceSettings{}, false
	}
	return appearanceSettings, true
}
