package provider

import (
	"context"
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	colorTestStyleID      = "20"
	colorTestOtherStyleID = "21"
	// colorTestLargerPaletteID is a style of a YouTrack whose palette is larger than the one the client knows.
	colorTestLargerPaletteID = "36"
	colorTestDescription     = "Changed"
	colorTestValueName       = "Critical"
	colorTestValueID         = "67-1"
)

// bundleColorKind gives the color tests the same entry points for an enum and a state bundle.
type bundleColorKind struct {
	name string
	// newValueColor is the style sent when a value with the given color_id is added.
	newValueColor func(colorID types.String) *youtrack.FieldStyle
	// readColorID is the color_id read back from a value with the given style.
	readColorID func(style *youtrack.FieldStyle) types.String
	// update returns the style sent when a value with the current style is planned with the given color_id and
	// description, and whether the value gets written at all.
	update func(colorID, description types.String, current *youtrack.FieldStyle) (*youtrack.FieldStyle, bool)
}

func bundleColorKinds() []bundleColorKind {
	return []bundleColorKind{
		{
			name: "enum",
			newValueColor: func(colorID types.String) *youtrack.FieldStyle {
				return (&enumBundleValueModel{Name: types.StringValue(colorTestValueName), ColorID: colorID}).toAPIModel().Color
			},
			readColorID: func(style *youtrack.FieldStyle) types.String {
				var model enumBundleResourceModel
				model.fromAPIModel(&youtrack.EnumBundle{Values: []youtrack.EnumBundleElement{{ID: colorTestValueID, Name: colorTestValueName, Ordinal: 1, Color: style}}})
				return model.Values[0].ColorID
			},
			update: func(colorID, description types.String, current *youtrack.FieldStyle) (*youtrack.FieldStyle, bool) {
				existing := youtrack.EnumBundleElement{ID: colorTestValueID, Name: colorTestValueName, Ordinal: 1, Color: current}
				model := enumBundleValueModel{Name: types.StringValue(colorTestValueName), Description: description, LocalizedName: types.StringUnknown(), ColorID: colorID, Archived: types.BoolValue(false)}
				update := model.toValueUpdate(existing, existing.Ordinal)
				return update.Color, !reflect.DeepEqual(update, enumValueAsUpdate(existing))
			},
		},
		{
			name: "state",
			newValueColor: func(colorID types.String) *youtrack.FieldStyle {
				return (&stateBundleValueModel{Name: types.StringValue(colorTestValueName), ColorID: colorID}).toAPIModel().Color
			},
			readColorID: func(style *youtrack.FieldStyle) types.String {
				var model stateBundleResourceModel
				model.fromAPIModel(&youtrack.StateBundle{Values: []youtrack.StateBundleElement{{ID: colorTestValueID, Name: colorTestValueName, Ordinal: 1, Color: style}}})
				return model.Values[0].ColorID
			},
			update: func(colorID, description types.String, current *youtrack.FieldStyle) (*youtrack.FieldStyle, bool) {
				existing := youtrack.StateBundleElement{ID: colorTestValueID, Name: colorTestValueName, Ordinal: 1, Color: current}
				model := stateBundleValueModel{Name: types.StringValue(colorTestValueName), Description: description, LocalizedName: types.StringUnknown(), ColorID: colorID, IsResolved: types.BoolValue(false), Archived: types.BoolValue(false)}
				update := model.toValueUpdate(existing, existing.Ordinal)
				return update.Color, !reflect.DeepEqual(update, stateValueAsUpdate(existing))
			},
		},
	}
}

// readStyle is a style as YouTrack returns it, with the colors and the type derived from the ID.
func readStyle(id string) *youtrack.FieldStyle {
	return &youtrack.FieldStyle{ID: id, Background: "#DB3B4B", Foreground: "#fff", Type: "FieldStyle"}
}

func TestBundleValueColorOfNewValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		colorID types.String
		want    *youtrack.FieldStyle
	}{
		{name: "configured", colorID: types.StringValue(colorTestStyleID), want: &youtrack.FieldStyle{ID: colorTestStyleID}},
		{name: "default", colorID: types.StringValue(youtrack.DefaultFieldStyleID), want: &youtrack.FieldStyle{ID: youtrack.DefaultFieldStyleID}},
		{name: "unknown is left out", colorID: types.StringUnknown()},
		{name: "null is left out", colorID: types.StringNull()},
	}

	for _, kind := range bundleColorKinds() {
		for _, tc := range cases {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				if got := kind.newValueColor(tc.colorID); !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("unexpected color: got %+v, want %+v", got, tc.want)
				}
			})
		}
	}
}

func TestBundleValueColorFromAPIModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style *youtrack.FieldStyle
		want  string
	}{
		{name: "colored", style: readStyle(colorTestStyleID), want: colorTestStyleID},
		{name: "default style", style: readStyle(youtrack.DefaultFieldStyleID), want: youtrack.DefaultFieldStyleID},
		{name: "no style is the default one", want: youtrack.DefaultFieldStyleID},
		{name: "style without ID is the default one", style: &youtrack.FieldStyle{}, want: youtrack.DefaultFieldStyleID},
	}

	for _, kind := range bundleColorKinds() {
		for _, tc := range cases {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				helpers.AssertFieldEqual(t, "ColorID", kind.readColorID(tc.style), types.StringValue(tc.want))
			})
		}
	}
}

// TestBundleValueColorUpdate checks what the reconciler sends for the color of an existing value: nothing when
// the color is unchanged, so that a value is not rewritten on every apply, and never the current style, which
// the client refuses when it is outside the palette it knows.
func TestBundleValueColorUpdate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		current     *youtrack.FieldStyle
		colorID     types.String
		description types.String
		wantColor   *youtrack.FieldStyle
		wantWrite   bool
	}{
		{name: "same color", current: readStyle(colorTestStyleID), colorID: types.StringValue(colorTestStyleID)},
		{name: "unknown keeps the current color", current: readStyle(colorTestStyleID), colorID: types.StringUnknown()},
		{name: "default on a value with no style", colorID: types.StringValue(youtrack.DefaultFieldStyleID)},
		{
			name: "other color", current: readStyle(colorTestStyleID), colorID: types.StringValue(colorTestOtherStyleID),
			wantColor: &youtrack.FieldStyle{ID: colorTestOtherStyleID}, wantWrite: true,
		},
		{
			name: "default resets a color", current: readStyle(colorTestStyleID), colorID: types.StringValue(youtrack.DefaultFieldStyleID),
			wantColor: &youtrack.FieldStyle{ID: youtrack.DefaultFieldStyleID}, wantWrite: true,
		},
		{
			name: "color on a value with no style", colorID: types.StringValue(colorTestStyleID),
			wantColor: &youtrack.FieldStyle{ID: colorTestStyleID}, wantWrite: true,
		},
		{
			name: "other change does not send a style outside the palette", current: readStyle(colorTestLargerPaletteID),
			colorID: types.StringValue(colorTestLargerPaletteID), description: types.StringValue(colorTestDescription), wantWrite: true,
		},
		{
			name: "other change with an unknown color does not send the current style", current: readStyle(colorTestLargerPaletteID),
			colorID: types.StringUnknown(), description: types.StringValue(colorTestDescription), wantWrite: true,
		},
	}

	for _, kind := range bundleColorKinds() {
		for _, tc := range cases {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				description := tc.description
				if description.IsNull() {
					description = types.StringUnknown()
				}

				color, write := kind.update(tc.colorID, description, tc.current)
				if !reflect.DeepEqual(color, tc.wantColor) {
					t.Fatalf("unexpected color sent: got %+v, want %+v", color, tc.wantColor)
				}
				helpers.AssertFieldEqual(t, "Write", write, tc.wantWrite)
			})
		}
	}
}

func TestValidateBundleColorIDs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		colorID   types.String
		wantError bool
	}{
		{name: "default style", colorID: types.StringValue(youtrack.DefaultFieldStyleID)},
		{name: "last style", colorID: types.StringValue("35")},
		{name: "null", colorID: types.StringNull()},
		{name: "unknown", colorID: types.StringUnknown()},
		{name: "outside the palette", colorID: types.StringValue(colorTestLargerPaletteID), wantError: true},
		{name: "negative", colorID: types.StringValue("-1"), wantError: true},
		{name: "leading zero", colorID: types.StringValue("07"), wantError: true},
		{name: "not a number", colorID: types.StringValue("red"), wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var diagnostics diag.Diagnostics
			validateBundleColorIDs(&diagnostics, []types.String{types.StringValue(colorTestStyleID), tc.colorID})

			helpers.AssertFieldEqual(t, "HasError", diagnostics.HasError(), tc.wantError)
			if tc.wantError {
				assertColorErrorPath(t, diagnostics, 1)
			}
		})
	}
}

// assertColorErrorPath checks that the first error of diagnostics is on the color of the value at index.
func assertColorErrorPath(t *testing.T, diagnostics diag.Diagnostics, index int) {
	t.Helper()

	withPath, ok := diagnostics.Errors()[0].(diag.DiagnosticWithPath)
	if !ok {
		t.Fatal("the error has no attribute path")
	}
	helpers.AssertFieldEqual(t, "Path", withPath.Path().String(), path.Root("values").AtListIndex(index).AtName("color_id").String())
}

// validatingResource is a bundle resource that validates its configuration.
type validatingResource interface {
	resource.Resource
	resource.ResourceWithValidateConfig
}

// bundleConfigWithValues is a configuration of the given resource where only values is set.
func bundleConfigWithValues(t *testing.T, r validatingResource, values func(valuesType tftypes.Type) tftypes.Value) tfsdk.Config {
	t.Helper()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema type is not an object")
	}

	attributes := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}
	attributes["values"] = values(objectType.AttributeTypes["values"])

	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, attributes)}
}

// validatingBundles are the bundle resources whose configuration validation checks the color of their values.
func validatingBundles() []struct {
	name     string
	resource validatingResource
} {
	return []struct {
		name     string
		resource validatingResource
	}{
		{name: "enum", resource: &enumBundleResource{}},
		{name: "state", resource: &stateBundleResource{}},
	}
}

func TestBundleValidateConfigWithUnknownValues(t *testing.T) {
	t.Parallel()

	unknownList := func(valuesType tftypes.Type) tftypes.Value { return tftypes.NewValue(valuesType, tftypes.UnknownValue) }

	for _, tc := range validatingBundles() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := resource.ValidateConfigRequest{Config: bundleConfigWithValues(t, tc.resource, unknownList)}
			var resp resource.ValidateConfigResponse
			tc.resource.ValidateConfig(context.Background(), req, &resp)

			helpers.AssertFieldEqual(t, "ErrorCount", resp.Diagnostics.ErrorsCount(), 0)
		})
	}
}

func TestBundleValidateConfigRefusesUnknownPaletteColor(t *testing.T) {
	t.Parallel()

	for _, tc := range validatingBundles() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := bundleConfigWithValues(t, tc.resource, func(valuesType tftypes.Type) tftypes.Value {
				list, ok := valuesType.(tftypes.List)
				if !ok {
					t.Fatal("values is not a list")
				}
				elementType, ok := list.ElementType.(tftypes.Object)
				if !ok {
					t.Fatal("a value is not an object")
				}
				element := make(map[string]tftypes.Value, len(elementType.AttributeTypes))
				for name, attributeType := range elementType.AttributeTypes {
					element[name] = tftypes.NewValue(attributeType, nil)
				}
				element["name"] = tftypes.NewValue(tftypes.String, colorTestValueName)
				element["color_id"] = tftypes.NewValue(tftypes.String, colorTestLargerPaletteID)

				return tftypes.NewValue(valuesType, []tftypes.Value{tftypes.NewValue(elementType, element)})
			})

			var resp resource.ValidateConfigResponse
			tc.resource.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			helpers.AssertFieldEqual(t, "ErrorCount", resp.Diagnostics.ErrorsCount(), 1)
			assertColorErrorPath(t, resp.Diagnostics, 0)
		})
	}
}

// TestBundleColorIDSchema checks that both bundles default color_id to the default style.
func TestBundleColorIDSchema(t *testing.T) {
	t.Parallel()

	for _, tc := range validatingBundles() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp resource.SchemaResponse
			tc.resource.Schema(context.Background(), resource.SchemaRequest{}, &resp)

			values, ok := resp.Schema.Attributes["values"].(schema.ListNestedAttribute)
			if !ok {
				t.Fatal("values is not a list nested attribute")
			}
			colorID, ok := values.NestedObject.Attributes["color_id"].(schema.StringAttribute)
			if !ok {
				t.Fatal("color_id is not a string attribute")
			}

			var defaultResp defaults.StringResponse
			colorID.Default.DefaultString(context.Background(), defaults.StringRequest{}, &defaultResp)
			helpers.AssertFieldEqual(t, "Default", defaultResp.PlanValue, types.StringValue(youtrack.DefaultFieldStyleID))
		})
	}
}
