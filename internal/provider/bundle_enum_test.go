package provider

import (
	"errors"
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

func TestEnumBundleToAPIModelPreservingExisting(t *testing.T) {
	t.Parallel()

	model := enumBundleResourceModel{
		Name: types.StringValue("Priority"),
		Values: []enumBundleValueModel{
			{ID: types.StringValue("67-1"), Name: types.StringValue("Critical"), Archived: types.BoolValue(false)},
			{Name: types.StringValue("Trivial"), Archived: types.BoolValue(false)},
		},
	}

	current := &youtrack.EnumBundle{
		ID:   "66-12",
		Name: "Priority",
		Values: []youtrack.EnumBundleElement{
			{ID: "67-1", Name: "Major", Ordinal: 0},
			{ID: "67-2", Name: "Minor", Ordinal: 1},
		},
	}

	apiModel := model.toAPIModelPreservingExisting(current)
	helpers.AssertFieldEqual(t, "ValuesLength", len(apiModel.Values), 3)
	helpers.AssertFieldEqual(t, "FirstValueName", apiModel.Values[0].Name, "Critical")
	helpers.AssertFieldEqual(t, "SecondValueName", apiModel.Values[1].Name, "Minor")
	helpers.AssertFieldEqual(t, "ThirdValueName", apiModel.Values[2].Name, "Trivial")
}

func TestIsRequiredCustomFieldWorkflowError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "required custom field rule",
			err:  errors.New("unexpected status code 400: {\"error_rule_name\":\"@jetbrains/required-custom-fields-feature\"}"),
			want: true,
		},
		{
			name: "workflow field required",
			err:  errors.New("unexpected status code 400: {\"error\":\"Field required\",\"error_type\":\"workflow\"}"),
			want: true,
		},
		{
			name: "unrelated validation",
			err:  errors.New("unexpected status code 400: {\"error\":\"name must not be empty\"}"),
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isRequiredCustomFieldWorkflowError(tc.err)
			helpers.AssertFieldEqual(t, "Matched", got, tc.want)
		})
	}
}

func TestEnumBundleToAPIModelPreservingExistingMatchesByName(t *testing.T) {
	t.Parallel()

	model := enumBundleResourceModel{
		Name: types.StringValue("Priority"),
		Values: []enumBundleValueModel{
			{Name: types.StringValue("Major"), Archived: types.BoolValue(false)},
			{Name: types.StringValue("Minor"), Archived: types.BoolValue(true)},
		},
	}

	current := &youtrack.EnumBundle{
		ID:   "66-12",
		Name: "Priority",
		Values: []youtrack.EnumBundleElement{
			{ID: "67-1", Name: "Major", Ordinal: 0, Archived: false},
			{ID: "67-2", Name: "Minor", Ordinal: 1, Archived: false},
		},
	}

	apiModel := model.toAPIModelPreservingExisting(current)
	helpers.AssertFieldEqual(t, "ValuesLength", len(apiModel.Values), 2)
	helpers.AssertFieldEqual(t, "FirstValueName", apiModel.Values[0].Name, "Major")
	helpers.AssertFieldEqual(t, "FirstValueID", apiModel.Values[0].ID, "67-1")
	helpers.AssertFieldEqual(t, "SecondValueName", apiModel.Values[1].Name, "Minor")
	helpers.AssertFieldEqual(t, "SecondValueID", apiModel.Values[1].ID, "67-2")
	helpers.AssertFieldEqual(t, "SecondValueArchived", apiModel.Values[1].Archived, true)
}

func TestUnexpectedEnumValueNames(t *testing.T) {
	t.Parallel()

	plan := enumBundleResourceModel{
		Name: types.StringValue("Priority"),
		Values: []enumBundleValueModel{
			{Name: types.StringValue("Major")},
			{Name: types.StringValue("Minor")},
		},
	}

	updated := &youtrack.EnumBundle{
		Values: []youtrack.EnumBundleElement{
			{Name: "Major"},
			{Name: "Minor"},
			{Name: "Critical"},
		},
	}

	unexpected := unexpectedEnumValueNames(plan, updated)
	helpers.AssertFieldEqual(t, "UnexpectedLength", len(unexpected), 1)
	helpers.AssertFieldEqual(t, "UnexpectedName", unexpected[0], "Critical")
}
