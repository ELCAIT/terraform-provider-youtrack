package provider

import (
	"context"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	testCustomFieldID            = "58-999"
	testCustomFieldName          = "Severity"
	testCustomFieldLocalizedName = "Localized Severity"
	testCustomFieldAliases       = "severity,sevr"
	testFieldTypeID              = "enum[1]"
	testBundleID                 = "66-1"
	testBundleName               = "Priorities"
	testFieldPresentation        = "enum[1]"
)

func TestCustomFieldModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := customFieldResourceModel{
		Name:                   types.StringValue(testCustomFieldName),
		FieldTypeID:            types.StringValue(testFieldTypeID),
		LocalizedName:          types.StringValue(testCustomFieldLocalizedName),
		Aliases:                types.StringValue(testCustomFieldAliases),
		IsAutoAttached:         types.BoolValue(true),
		IsDisplayedInIssueList: types.BoolValue(true),
		FieldDefaults: types.ObjectValueMust(customFieldDefaultsAttrTypes(), map[string]attr.Value{
			"can_be_empty":     types.BoolValue(true),
			"empty_field_text": types.StringValue("Not set"),
			"is_public":        types.BoolValue(false),
			"bundle_id":        types.StringValue(testBundleID),
			"bundle_name":      types.StringNull(),
		}),
	}

	apiModel := model.toAPIModel()

	helpers.AssertFieldEqual(t, "Name", apiModel.Name, testCustomFieldName)
	helpers.AssertFieldEqual(t, "FieldType.ID", apiModel.FieldType.ID, testFieldTypeID)
	helpers.AssertFieldEqual(t, "LocalizedName", *apiModel.LocalizedName, testCustomFieldLocalizedName)
	helpers.AssertFieldEqual(t, "Aliases", *apiModel.Aliases, testCustomFieldAliases)
	helpers.AssertFieldEqual(t, "IsAutoAttached", *apiModel.IsAutoAttached, true)
	helpers.AssertFieldEqual(t, "IsDisplayedInIssueList", *apiModel.IsDisplayedInIssueList, true)
	helpers.AssertFieldEqual(t, "FieldDefaults.Bundle.ID", apiModel.FieldDefaults.Bundle.ID, testBundleID)
	helpers.AssertFieldEqual(t, "FieldDefaults.Bundle.Type", apiModel.FieldDefaults.Bundle.Type, bundleTypeEnum)
	helpers.AssertFieldEqual(t, "FieldDefaults.Type", apiModel.FieldDefaults.Type, defaultsTypeEnum)
}

func TestCustomFieldModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.CustomField{
		ID:                     testCustomFieldID,
		Name:                   testCustomFieldName,
		LocalizedName:          testCustomFieldLocalizedName,
		Aliases:                testCustomFieldAliases,
		FieldType:              youtrack.FieldType{ID: testFieldTypeID, Presentation: testFieldPresentation},
		IsAutoAttached:         true,
		IsDisplayedInIssueList: true,
		Ordinal:                42,
		IsUpdateable:           true,
		HasRunningJob:          false,
		FieldDefaults: &youtrack.CustomFieldDefaults{
			CanBeEmpty:     true,
			EmptyFieldText: "Not set",
			IsPublic:       false,
			Bundle:         &youtrack.BundleRef{ID: testBundleID, Name: testBundleName},
		},
	}

	var model customFieldResourceModel
	model.fromAPIModel(apiModel)

	var defaults customFieldDefaultsModel
	if diags := model.FieldDefaults.As(context.Background(), &defaults, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("failed to decode field_defaults object: %v", diags)
	}

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testCustomFieldID)
	helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), testCustomFieldName)
	helpers.AssertFieldEqual(t, "FieldTypeID", model.FieldTypeID.ValueString(), testFieldTypeID)
	helpers.AssertFieldEqual(t, "FieldTypePresentation", model.FieldTypePresentation.ValueString(), testFieldPresentation)
	helpers.AssertFieldEqual(t, "BundleID", defaults.BundleID.ValueString(), testBundleID)
	helpers.AssertFieldEqual(t, "BundleName", defaults.BundleName.ValueString(), testBundleName)
}

func TestBundleTypeForFieldTypeID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		fieldTypeID string
		wantType    string
		wantOK      bool
	}{
		{name: "enum", fieldTypeID: testFieldTypeID, wantType: bundleTypeEnum, wantOK: true},
		{name: "state", fieldTypeID: "state[1]", wantType: bundleTypeState, wantOK: true},
		{name: "owned", fieldTypeID: "ownedField[1]", wantType: bundleTypeOwned, wantOK: true},
		{name: "build", fieldTypeID: "build[1]", wantType: bundleTypeBuild, wantOK: true},
		{name: "version", fieldTypeID: "version[1]", wantType: bundleTypeVer, wantOK: true},
		{name: "unknown", fieldTypeID: "string", wantType: "", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotType, gotOK := bundleTypeForFieldTypeID(tc.fieldTypeID)

			helpers.AssertFieldEqual(t, "ok", gotOK, tc.wantOK)
			helpers.AssertFieldEqual(t, "type", gotType, tc.wantType)
		})
	}
}

func TestDefaultsTypeForFieldTypeID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		fieldTypeID string
		wantType    string
		wantOK      bool
	}{
		{name: "enum", fieldTypeID: testFieldTypeID, wantType: defaultsTypeEnum, wantOK: true},
		{name: "state", fieldTypeID: "state[1]", wantType: defaultsTypeState, wantOK: true},
		{name: "owned", fieldTypeID: "ownedField[1]", wantType: defaultsTypeOwned, wantOK: true},
		{name: "build", fieldTypeID: "build[1]", wantType: defaultsTypeBuild, wantOK: true},
		{name: "version", fieldTypeID: "version[1]", wantType: defaultsTypeVer, wantOK: true},
		{name: "unknown", fieldTypeID: "string", wantType: "", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotType, gotOK := defaultsTypeForFieldTypeID(tc.fieldTypeID)

			helpers.AssertFieldEqual(t, "ok", gotOK, tc.wantOK)
			helpers.AssertFieldEqual(t, "type", gotType, tc.wantType)
		})
	}
}

func TestCustomFieldRequestBundleHelpers(t *testing.T) {
	t.Parallel()

	withBundle := youtrack.CustomFieldUpsertRequest{
		FieldDefaults: &youtrack.CustomFieldDefaultsUpsertModel{
			Bundle: &youtrack.BundleRef{ID: testBundleID},
		},
	}

	withoutBundle := customFieldRequestWithoutBundle(withBundle)

	helpers.AssertFieldEqual(t, "has bundle before", customFieldRequestHasBundle(withBundle), true)
	helpers.AssertFieldEqual(t, "has bundle after", customFieldRequestHasBundle(withoutBundle), false)
	helpers.AssertFieldEqual(t, "original unchanged", customFieldRequestHasBundle(withBundle), true)
}

func TestCustomFieldTypeChangeRequested(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		currentType types.String
		plannedType types.String
		want        bool
	}{
		{
			name:        "same type",
			currentType: types.StringValue(testFieldTypeID),
			plannedType: types.StringValue(testFieldTypeID),
			want:        false,
		},
		{
			name:        "different type",
			currentType: types.StringValue(testFieldTypeID),
			plannedType: types.StringValue("string"),
			want:        true,
		},
		{
			name:        "current unknown",
			currentType: types.StringUnknown(),
			plannedType: types.StringValue("string"),
			want:        false,
		},
		{
			name:        "planned unknown",
			currentType: types.StringValue(testFieldTypeID),
			plannedType: types.StringUnknown(),
			want:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := customFieldTypeChangeRequested(tc.currentType, tc.plannedType)
			helpers.AssertFieldEqual(t, "type change requested", got, tc.want)
		})
	}
}
