package provider

import (
	"context"
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const colorTestStyleID = "20"

func TestEnumBundleValueColorToAPIModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		model enumBundleValueModel
		want  *youtrack.FieldStyle
	}{
		{name: "configured", model: enumBundleValueModel{ColorID: types.StringValue(colorTestStyleID)}, want: &youtrack.FieldStyle{ID: colorTestStyleID}},
		{name: "unset", model: enumBundleValueModel{ColorID: types.StringNull()}},
		{name: "unknown", model: enumBundleValueModel{ColorID: types.StringUnknown()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			element := tc.model.toAPIModel()
			if !reflect.DeepEqual(element.Color, tc.want) {
				t.Fatalf("unexpected color: got %+v, want %+v", element.Color, tc.want)
			}
		})
	}
}

func TestEnumBundleValueColorFromAPIModel(t *testing.T) {
	t.Parallel()

	var model enumBundleResourceModel
	model.fromAPIModel(&youtrack.EnumBundle{Values: []youtrack.EnumBundleElement{
		{ID: "67-1", Name: "Red", Ordinal: 1, Color: &youtrack.FieldStyle{ID: colorTestStyleID, Background: "#DB3B4B", Foreground: "#fff"}},
		{ID: "67-2", Name: "Plain", Ordinal: 2},
	}})

	helpers.AssertFieldEqual(t, "ColoredID", model.Values[0].ColorID.ValueString(), colorTestStyleID)
	helpers.AssertFieldEqual(t, "PlainIsNull", model.Values[1].ColorID.IsNull(), true)
}

// TestEnumValueColorDoesNotTriggerAWrite checks the comparison the reconciler makes: a value whose color is
// unchanged, or unset in the plan, must equal the current one, or every apply rewrites every value.
func TestEnumValueColorDoesNotTriggerAWrite(t *testing.T) {
	t.Parallel()

	current := youtrack.EnumBundleElement{
		ID: "67-1", Name: "Red", Ordinal: 1,
		Color: &youtrack.FieldStyle{ID: colorTestStyleID, Background: "#DB3B4B", Foreground: "#fff", Type: "FieldStyle"},
	}

	cases := []struct {
		name      string
		colorID   types.String
		wantEqual bool
	}{
		{name: "same color", colorID: types.StringValue(colorTestStyleID), wantEqual: true},
		{name: "unset keeps the current color", colorID: types.StringNull(), wantEqual: true},
		{name: "unknown keeps the current color", colorID: types.StringUnknown(), wantEqual: true},
		{name: "other color", colorID: types.StringValue("21")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := enumBundleValueModel{Name: types.StringValue("Red"), ColorID: tc.colorID, Archived: types.BoolValue(false)}
			got := reflect.DeepEqual(model.toValueUpdate(current, current.Ordinal), enumValueAsUpdate(current))
			if got != tc.wantEqual {
				t.Fatalf("unexpected comparison: got equal %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

func TestStateBundleValueColor(t *testing.T) {
	t.Parallel()

	element := (&stateBundleValueModel{Name: types.StringValue("Open"), ColorID: types.StringValue(colorTestStyleID)}).toAPIModel()
	if !reflect.DeepEqual(element.Color, &youtrack.FieldStyle{ID: colorTestStyleID}) {
		t.Fatalf("unexpected color: got %+v", element.Color)
	}

	var model stateBundleResourceModel
	model.fromAPIModel(&youtrack.StateBundle{Values: []youtrack.StateBundleElement{
		{ID: "68-1", Name: "Open", Ordinal: 1, Color: &youtrack.FieldStyle{ID: colorTestStyleID, Background: "#DB3B4B"}},
		{ID: "68-2", Name: "Plain", Ordinal: 2},
	}})
	helpers.AssertFieldEqual(t, "ColoredID", model.Values[0].ColorID.ValueString(), colorTestStyleID)
	helpers.AssertFieldEqual(t, "PlainIsNull", model.Values[1].ColorID.IsNull(), true)
}

// TestStateValueColorDoesNotTriggerAWrite is TestEnumValueColorDoesNotTriggerAWrite for state values.
func TestStateValueColorDoesNotTriggerAWrite(t *testing.T) {
	t.Parallel()

	current := youtrack.StateBundleElement{
		ID: "68-1", Name: "Open", Ordinal: 1,
		Color: &youtrack.FieldStyle{ID: colorTestStyleID, Background: "#DB3B4B", Foreground: "#fff", Type: "FieldStyle"},
	}

	cases := []struct {
		name      string
		colorID   types.String
		wantEqual bool
	}{
		{name: "same color", colorID: types.StringValue(colorTestStyleID), wantEqual: true},
		{name: "unset keeps the current color", colorID: types.StringNull(), wantEqual: true},
		{name: "other color", colorID: types.StringValue("21")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := stateBundleValueModel{Name: types.StringValue("Open"), ColorID: tc.colorID, IsResolved: types.BoolValue(false), Archived: types.BoolValue(false)}
			got := reflect.DeepEqual(model.toValueUpdate(current, current.Ordinal), stateValueAsUpdate(current))
			if got != tc.wantEqual {
				t.Fatalf("unexpected comparison: got equal %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

func TestValidateBundleColorIDs(t *testing.T) {
	t.Parallel()

	var diagnostics diag.Diagnostics
	validateBundleColorIDs(&diagnostics, []types.String{
		types.StringValue("35"), types.StringNull(), types.StringUnknown(), types.StringValue("36"), types.StringValue("red"),
	})

	helpers.AssertFieldEqual(t, "ErrorCount", diagnostics.ErrorsCount(), 2)
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

func TestBundleValidateConfigWithUnknownValues(t *testing.T) {
	t.Parallel()

	unknownList := func(valuesType tftypes.Type) tftypes.Value { return tftypes.NewValue(valuesType, tftypes.UnknownValue) }

	cases := []struct {
		name     string
		resource validatingResource
	}{
		{name: "enum", resource: &enumBundleResource{}},
		{name: "state", resource: &stateBundleResource{}},
	}

	for _, tc := range cases {
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

	var r validatingResource = &enumBundleResource{}
	config := bundleConfigWithValues(t, r, func(valuesType tftypes.Type) tftypes.Value {
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
		element["name"] = tftypes.NewValue(tftypes.String, "Critical")
		element["color_id"] = tftypes.NewValue(tftypes.String, "36")

		return tftypes.NewValue(valuesType, []tftypes.Value{tftypes.NewValue(elementType, element)})
	})

	var resp resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

	helpers.AssertFieldEqual(t, "ErrorCount", resp.Diagnostics.ErrorsCount(), 1)
}
