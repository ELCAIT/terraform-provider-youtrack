package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testStateBundleName = "Workflow States"

func TestStateBundleModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := stateBundleResourceModel{
		Name: types.StringValue(testStateBundleName),
		Values: []stateBundleValueModel{
			{
				Name:       types.StringValue("Open"),
				IsResolved: types.BoolValue(false),
				Archived:   types.BoolValue(false),
			},
			{
				Name:       types.StringValue("Done"),
				IsResolved: types.BoolValue(true),
				Archived:   types.BoolValue(false),
			},
		},
	}

	apiModel := model.toAPIModel()
	helpers.AssertFieldEqual(t, "Name", apiModel.Name, testStateBundleName)
	helpers.AssertFieldEqual(t, "ValuesLength", len(apiModel.Values), 2)
	helpers.AssertFieldEqual(t, "FirstValueName", apiModel.Values[0].Name, "Open")
	helpers.AssertFieldEqual(t, "SecondValueResolved", apiModel.Values[1].IsResolved, true)
}

func TestStateBundleModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.StateBundle{
		ID:           "68-9",
		Name:         testStateBundleName,
		IsUpdateable: true,
		Values: []youtrack.StateBundleElement{
			{ID: "69-1", Name: "Open", IsResolved: false, Ordinal: 0},
			{ID: "69-2", Name: "Done", IsResolved: true, Ordinal: 1},
		},
	}

	var model stateBundleResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), "68-9")
	helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), testStateBundleName)
	helpers.AssertFieldEqual(t, "IsUpdateable", model.IsUpdateable.ValueBool(), true)
	helpers.AssertFieldEqual(t, "ValuesLength", len(model.Values), 2)
	helpers.AssertFieldEqual(t, "SecondValueOrdinal", model.Values[1].Ordinal.ValueInt64(), int64(1))
}
