package provider

import (
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnumBundleModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := enumBundleResourceModel{
		Name: types.StringValue("Priority"),
		Values: []enumBundleValueModel{
			{Name: types.StringValue("Major"), Archived: types.BoolValue(false)},
			{Name: types.StringValue("Minor"), Archived: types.BoolValue(false)},
		},
	}

	apiModel := model.toAPIModel()
	helpers.AssertFieldEqual(t, "Name", apiModel.Name, "Priority")
	helpers.AssertFieldEqual(t, "ValuesLength", len(apiModel.Values), 2)
	helpers.AssertFieldEqual(t, "FirstValueName", apiModel.Values[0].Name, "Major")
}

func TestEnumBundleModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.EnumBundle{
		ID:           "66-12",
		Name:         "Priority",
		IsUpdateable: true,
		Values: []youtrack.EnumBundleElement{
			{ID: "67-1", Name: "Major", Ordinal: 0},
			{ID: "67-2", Name: "Minor", Ordinal: 1},
		},
	}

	var model enumBundleResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), "66-12")
	helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), "Priority")
	helpers.AssertFieldEqual(t, "IsUpdateable", model.IsUpdateable.ValueBool(), true)
	helpers.AssertFieldEqual(t, "ValuesLength", len(model.Values), 2)
	helpers.AssertFieldEqual(t, "SecondValueOrdinal", model.Values[1].Ordinal.ValueInt64(), int64(1))
}

func TestRemovedEnumBundleValues_NoRemovedValues(t *testing.T) {
	t.Parallel()

	stateValues := []enumBundleValueModel{
		{ID: types.StringValue("67-1"), Name: types.StringValue("Major")},
		{ID: types.StringValue("67-2"), Name: types.StringValue("Minor")},
	}
	planValues := []enumBundleValueModel{
		{ID: types.StringValue("67-1"), Name: types.StringValue("Major")},
		{ID: types.StringValue("67-2"), Name: types.StringValue("Minor")},
	}

	got := removedEnumBundleValues(stateValues, planValues)
	if len(got) != 0 {
		t.Fatalf("removedEnumBundleValues() = %v, want empty", got)
	}
}

func TestRemovedEnumBundleValues_DetectsRemovedValues(t *testing.T) {
	t.Parallel()

	stateValues := []enumBundleValueModel{
		{ID: types.StringValue("67-1"), Name: types.StringValue("Critical")},
		{ID: types.StringValue("67-2"), Name: types.StringValue("Major")},
		{ID: types.StringValue("67-3"), Name: types.StringValue("Minor")},
	}
	planValues := []enumBundleValueModel{
		{ID: types.StringValue("67-2"), Name: types.StringValue("Major")},
	}

	got := removedEnumBundleValues(stateValues, planValues)
	want := []string{"Critical", "Minor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("removedEnumBundleValues() = %v, want %v", got, want)
	}
}
