package provider

import (
	"context"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	bundleValueColorIDDescription = "ID of the color style of the value, a number from 0 (the default style) to 35. " +
		"The palette is fixed by YouTrack and a custom color cannot be set. " +
		"When unset, the color YouTrack has is kept; to go back to the default style, set it to \"0\"."
	errInvalidBundleValueColorID = "Invalid color ID"
)

// bundleColorIDAttribute is the color of a bundle value. It is Optional+Computed: a value created without it
// gets the default style, and one left unset in configuration keeps the color YouTrack has.
func bundleColorIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: bundleValueColorIDDescription,
	}
}

// ValidateConfig refuses at plan time a color ID the client would refuse at apply time.
func (r *enumBundleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	colorIDs, diags := bundleColorIDsFromConfig(ctx, req.Config, func(value *enumBundleValueModel) types.String { return value.ColorID })
	resp.Diagnostics.Append(diags...)
	validateBundleColorIDs(&resp.Diagnostics, colorIDs)
}

// ValidateConfig refuses at plan time a color ID the client would refuse at apply time.
func (r *stateBundleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	colorIDs, diags := bundleColorIDsFromConfig(ctx, req.Config, func(value *stateBundleValueModel) types.String { return value.ColorID })
	resp.Diagnostics.Append(diags...)
	validateBundleColorIDs(&resp.Diagnostics, colorIDs)
}

// bundleColorIDsFromConfig returns the color of each value in the configuration, in order. values is read as a
// list, not into a slice of models: a list that is not known yet, for instance one built from another
// resource, cannot be decoded into a slice, and would fail the plan. Such a list, and an element that is not
// known, yield no color to check.
func bundleColorIDsFromConfig[T any](ctx context.Context, config tfsdk.Config, colorID func(*T) types.String) ([]types.String, diag.Diagnostics) {
	var values types.List
	diags := config.GetAttribute(ctx, path.Root("values"), &values)
	if diags.HasError() || values.IsNull() || values.IsUnknown() {
		return nil, diags
	}

	colorIDs := make([]types.String, 0, len(values.Elements()))
	for _, element := range values.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			colorIDs = append(colorIDs, types.StringNull())
			continue
		}

		var value T
		diags.Append(object.As(ctx, &value, basetypes.ObjectAsOptions{})...)
		colorIDs = append(colorIDs, colorID(&value))
	}

	return colorIDs, diags
}

// validateBundleColorIDs adds an error for every configured color ID that is not in the palette. colorIDs is
// the color of each value of the bundle, in order.
func validateBundleColorIDs(diagnostics *diag.Diagnostics, colorIDs []types.String) {
	for i, colorID := range colorIDs {
		if colorID.IsNull() || colorID.IsUnknown() {
			continue
		}

		if err := youtrack.ValidateFieldStyleID(colorID.ValueString()); err != nil {
			diagnostics.AddAttributeError(
				path.Root("values").AtListIndex(i).AtName("color_id"),
				errInvalidBundleValueColorID,
				err.Error(),
			)
		}
	}
}

// colorFromPlan is the style sent for a value. An unset color is left out when the value is created, and keeps
// the current one when the value is replaced, so that a value is only written when something differs.
func colorFromPlan(planned types.String, current *youtrack.FieldStyle) *youtrack.FieldStyle {
	if planned.IsUnknown() || planned.IsNull() {
		return colorOnly(current)
	}

	return &youtrack.FieldStyle{ID: planned.ValueString()}
}

// colorOnly keeps the ID of a style read from YouTrack and drops its background, foreground and type, which are
// derived from the ID: this is what a style built from the plan carries, so the two compare equal.
func colorOnly(style *youtrack.FieldStyle) *youtrack.FieldStyle {
	if style == nil || style.ID == "" {
		return nil
	}

	return &youtrack.FieldStyle{ID: style.ID}
}

func colorIDFromAPI(style *youtrack.FieldStyle) types.String {
	if style == nil || style.ID == "" {
		return types.StringNull()
	}

	return types.StringValue(style.ID)
}
