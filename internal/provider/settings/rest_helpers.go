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
	errUnableToReadREST = "Unable to Read YouTrack REST Settings"
)

// getRestSettingsAndHandleError fetches the REST settings via API and handles errors.
func (r *restSettingsResource) getRestSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.RestSettings, bool) {
	restSettings, err := r.client.GetRestSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadREST,
			err.Error(),
		)
		return youtrack.RestSettings{}, false
	}
	return restSettings, true
}

// updateRestSettingsModelWithTimestamp updates the model from response and sets timestamp.
func updateRestSettingsModelWithTimestamp(ctx context.Context, restSettings youtrack.RestSettings, resourceModel *restSettingsResourceModel) bool {
	converted := convertRestSettingsToModel(ctx, restSettings)
	if converted == nil {
		return false
	}
	*resourceModel = *converted
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	return true
}

// convertModelToRestSettings converts a restSettingsResourceModel to youtrack.RestSettings.
func convertModelToRestSettings(model restSettingsResourceModel) (youtrack.RestSettings, bool) {
	var allowedOrigins []string
	diags := model.AllowedOrigins.ElementsAs(context.Background(), &allowedOrigins, false)
	if diags.HasError() {
		return youtrack.RestSettings{}, false
	}

	return youtrack.RestSettings{
		AllowAllOrigins: model.AllowAllOrigins.ValueBool(),
		AllowedOrigins:  allowedOrigins,
		ID:              model.ID.ValueString(),
	}, true
}

// convertRestSettingsToModel converts a youtrack.RestSettings to restSettingsResourceModel.
func convertRestSettingsToModel(ctx context.Context, rs youtrack.RestSettings) *restSettingsResourceModel {
	allowedOriginsList, diags := types.ListValueFrom(ctx, types.StringType, rs.AllowedOrigins)
	if diags.HasError() {
		return nil
	}

	return &restSettingsResourceModel{
		ID:              types.StringValue(rs.ID),
		AllowAllOrigins: types.BoolValue(rs.AllowAllOrigins),
		AllowedOrigins:  allowedOriginsList,
	}
}

// updateRestSettingsAndHandleError updates the REST settings via API and handles errors.
func (r *restSettingsResource) updateAndHandleError(ctx context.Context, rs youtrack.RestSettings, diagnostics *diag.Diagnostics) (youtrack.RestSettings, bool) {
	restSettings, err := r.client.UpdateRestSettings(ctx, rs)
	if err != nil {
		diagnostics.AddError(
			errUpdatingRESTSettings,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "REST settings", err),
		)
		return youtrack.RestSettings{}, false
	}
	return restSettings, true
}
