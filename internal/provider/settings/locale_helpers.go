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
	errUnableToReadLocale = "Unable to Read YouTrack Locale Settings"
	errUpdatingLocale     = "Error updating locale settings"
)

// getLocaleSettingsAndHandleError fetches the locale settings via API and handles errors.
func (r *localeSettingsResource) getLocaleSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.LocaleSettings, bool) {
	localeSettings, err := r.client.GetLocaleSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadLocale,
			err.Error(),
		)
		return youtrack.LocaleSettings{}, false
	}
	return localeSettings, true
}

// updateLocaleSettingsModelWithTimestamp updates the model from response and sets timestamp.
func updateLocaleSettingsModelWithTimestamp(localeSettings youtrack.LocaleSettings, resourceModel *localeSettingsResourceModel) {
	*resourceModel = convertLocaleSettingsToModel(localeSettings)
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// convertModelToLocaleSettings converts a localeSettingsResourceModel to youtrack.LocaleSettings.
func convertModelToLocaleSettings(model localeSettingsResourceModel) youtrack.LocaleSettings {
	return youtrack.LocaleSettings{
		Locale: youtrack.LocaleDescriptor{
			ID:        model.ID.ValueString(),
			Locale:    model.Locale.ValueString(),
			Language:  model.Language.ValueString(),
			Community: model.Community.ValueBool(),
			Name:      model.Name.ValueString(),
		},
	}
}

// convertLocaleSettingsToModel converts a youtrack.LocaleSettings to localeSettingsResourceModel.
func convertLocaleSettingsToModel(ls youtrack.LocaleSettings) localeSettingsResourceModel {
	return localeSettingsResourceModel{
		ID:        types.StringValue(ls.Locale.ID),
		Locale:    types.StringValue(ls.Locale.Locale),
		Language:  types.StringValue(ls.Locale.Language),
		Community: types.BoolValue(ls.Locale.Community),
		Name:      types.StringValue(ls.Locale.Name),
	}
}

// updateLocaleSettingsAndHandleError updates the locale settings via API and handles errors.
func (r *localeSettingsResource) updateAndHandleError(ctx context.Context, ls youtrack.LocaleSettings, diagnostics *diag.Diagnostics) (youtrack.LocaleSettings, bool) {
	localeSettings, err := r.client.UpdateLocaleSettings(ctx, ls)
	if err != nil {
		diagnostics.AddError(
			errUpdatingLocale,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "locale settings", err),
		)
		return youtrack.LocaleSettings{}, false
	}
	return localeSettings, true
}
