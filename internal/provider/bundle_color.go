package provider

import (
	"context"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	bundleValueColorIDDescription = "ID of the color style of the value, a number from 0 to 35. " +
		"The palette is fixed by YouTrack and a custom color cannot be set. " +
		"Defaults to \"0\", the default style, which shows the value with no color: " +
		"removing `color_id` from the configuration resets the color of the value, " +
		"and a color changed in YouTrack is reverted on the next apply."
	errInvalidBundleValueColorID = "Invalid color ID"
)

// bundleColorIDAttribute is the color of a bundle value. A value whose configuration leaves it out gets the
// default style, so the color is always known at plan time and a color set outside of Terraform is reverted.
func bundleColorIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(youtrack.DefaultFieldStyleID),
		Description: bundleValueColorIDDescription,
	}
}

// ValidateConfig refuses at plan time a color ID the client would refuse at apply time.
func (r *enumBundleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateBundleColorConfig(ctx, req, resp, func(value *enumBundleValueModel) types.String { return value.ColorID })
}

// ValidateConfig refuses at plan time a color ID the client would refuse at apply time.
func (r *stateBundleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateBundleColorConfig(ctx, req, resp, func(value *stateBundleValueModel) types.String { return value.ColorID })
}

// validateBundleColorConfig is the ValidateConfig of a bundle whose values are decoded as T, and colorID
// returns the color of such a value.
func validateBundleColorConfig[T any](ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse, colorID func(*T) types.String) {
	colorIDs, diags := bundleColorIDsFromConfig(ctx, req.Config, colorID)
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

// colorForNewValue is the style sent for a value that is added. A color that is not known is left out, and the
// value gets the default style.
func colorForNewValue(planned types.String) *youtrack.FieldStyle {
	if planned.IsUnknown() || planned.IsNull() {
		return nil
	}

	return &youtrack.FieldStyle{ID: planned.ValueString()}
}

// colorChange is the style sent when an existing value is replaced: only a planned color that differs from the
// current one is sent. Otherwise it is nil, and YouTrack keeps the color the value has: the current style is
// never sent back, since the client refuses one outside the palette it knows, for instance one read from a
// YouTrack with a larger palette.
func colorChange(planned types.String, current *youtrack.FieldStyle) *youtrack.FieldStyle {
	if planned.IsUnknown() || planned.IsNull() || planned.ValueString() == colorIDFromAPI(current).ValueString() {
		return nil
	}

	return &youtrack.FieldStyle{ID: planned.ValueString()}
}

// colorIDFromAPI is the color ID of a style read from YouTrack. A value with no style has the default one.
func colorIDFromAPI(style *youtrack.FieldStyle) types.String {
	if style == nil || style.ID == "" {
		return types.StringValue(youtrack.DefaultFieldStyleID)
	}

	return types.StringValue(style.ID)
}
