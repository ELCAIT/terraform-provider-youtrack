package settings

import (
	"context"
	"fmt"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadGlobal = "Unable to Read YouTrack Global Settings"
	errUpdatingGlobal     = "Error updating global settings"
)

// getGlobalSettingsAndHandleError fetches the global settings via API and handles errors.
func (r *globalSettingsResource) getGlobalSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.GlobalSettings, bool) {
	globalSettings, err := r.client.GetGlobalSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadGlobal,
			err.Error(),
		)
		return youtrack.GlobalSettings{}, false
	}
	return globalSettings, true
}

// updateGlobalSettingsModelWithTimestamp updates the model from response and sets timestamp.
func updateGlobalSettingsModelWithTimestamp(globalSettings youtrack.GlobalSettings, resourceModel *globalSettingsResourceModel) {
	*resourceModel = convertGlobalSettingsToModel(globalSettings)
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// getLicenseString extracts the license string from the GlobalSettings.
func getLicenseString(gs youtrack.GlobalSettings) string {
	if gs.License == nil {
		return ""
	}
	return gs.License.License
}

// convertModelToGlobalSettings converts a globalSettingsResourceModel to youtrack.GlobalSettings.
func convertModelToGlobalSettings(model globalSettingsResourceModel) youtrack.GlobalSettings {
	gs := youtrack.GlobalSettings{
		ID: model.ID.ValueString(),
	}

	if !model.License.IsNull() && model.License.ValueString() != "" {
		gs.License = &youtrack.License{
			Type:    "jetbrains.charisma.persistent.globalSettings.License",
			License: model.License.ValueString(),
		}
	}

	return gs
}

// convertGlobalSettingsToModel converts a youtrack.GlobalSettings to globalSettingsResourceModel.
func convertGlobalSettingsToModel(gs youtrack.GlobalSettings) globalSettingsResourceModel {
	licenseValue := getLicenseString(gs)

	model := globalSettingsResourceModel{
		ID: types.StringValue(gs.ID),
	}

	if licenseValue == "" {
		model.License = types.StringNull()
	} else {
		model.License = types.StringValue(licenseValue)
	}

	return model
}

// updateGlobalSettingsAndHandleError updates the global settings via API and handles errors.
func (r *globalSettingsResource) updateAndHandleError(ctx context.Context, gs youtrack.GlobalSettings, diagnostics *diag.Diagnostics) (youtrack.GlobalSettings, bool) {
	globalSettings, err := r.client.UpdateGlobalSettings(ctx, gs)
	if err != nil {
		diagnostics.AddError(
			errUpdatingGlobal,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "global settings", err),
		)
		return youtrack.GlobalSettings{}, false
	}
	return globalSettings, true
}

// buildGlobalSettings builds the API struct from plan, reading the current license from API
// if the plan has no license set (to avoid sending nil license which the API rejects).
func (r *globalSettingsResource) buildGlobalSettings(ctx context.Context, plan globalSettingsResourceModel, diagnostics *diag.Diagnostics) (youtrack.GlobalSettings, bool) {
	gs := convertModelToGlobalSettings(plan)

	if gs.License == nil {
		current, ok := r.getGlobalSettingsAndHandleError(ctx, diagnostics)
		if !ok {
			return youtrack.GlobalSettings{}, false
		}
		gs.License = convertModelToGlobalSettings(convertGlobalSettingsToModel(current)).License
	}

	return gs, true
}
