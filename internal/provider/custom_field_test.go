package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
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
			"default_value_names": types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("Critical"),
				types.StringValue("Major"),
			}),
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
	helpers.AssertFieldEqual(t, "FieldDefaults.DefaultValues length", len(apiModel.FieldDefaults.DefaultValues), 2)
	helpers.AssertFieldEqual(t, "FieldDefaults.DefaultValues[0].Name", apiModel.FieldDefaults.DefaultValues[0].Name, "Critical")
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
			DefaultValues:  []youtrack.ProjectCustomFieldValueRef{{ID: "66-11", Name: "Critical"}, {ID: "66-12", Name: "Major"}},
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
	helpers.AssertFieldEqual(t, "EmptyFieldText", defaults.EmptyFieldText.ValueString(), "Not set")
	helpers.AssertFieldEqual(t, "BundleID", defaults.BundleID.ValueString(), testBundleID)
	helpers.AssertFieldEqual(t, "BundleName", defaults.BundleName.ValueString(), testBundleName)
	defaultValueNames, ok := helpers.ListToStringSlice(context.Background(), defaults.DefaultValueNames)
	helpers.AssertFieldEqual(t, "DefaultValueNames decode", ok, true)
	helpers.AssertFieldEqual(t, "DefaultValueNames length", len(defaultValueNames), 2)
	helpers.AssertFieldEqual(t, "DefaultValueNames[0]", defaultValueNames[0], "Critical")
}

func TestCustomFieldModelFromAPIModelEmptyFieldTextPreserved(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.CustomField{
		ID:            testCustomFieldID,
		Name:          testCustomFieldName,
		FieldType:     youtrack.FieldType{ID: testFieldTypeID},
		FieldDefaults: &youtrack.CustomFieldDefaults{CanBeEmpty: true, EmptyFieldText: "", IsPublic: true},
	}

	var model customFieldResourceModel
	model.fromAPIModel(apiModel)

	var defaults customFieldDefaultsModel
	if diags := model.FieldDefaults.As(context.Background(), &defaults, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("failed to decode field_defaults object: %v", diags)
	}

	helpers.AssertFieldEqual(t, "EmptyFieldText.IsNull", defaults.EmptyFieldText.IsNull(), false)
	helpers.AssertFieldEqual(t, "EmptyFieldText", defaults.EmptyFieldText.ValueString(), "")
	helpers.AssertFieldEqual(t, "DefaultValueNames.IsNull", defaults.DefaultValueNames.IsNull(), true)
}

func TestResolveCustomFieldDefaultValuesWithBundleName(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/api/admin/customFieldSettings/bundles/enum":
			_, _ = w.Write([]byte(`[{"id":"66-1","name":"Priorities"}]`))
		case "/api/admin/customFieldSettings/bundles/enum/66-1":
			_, _ = w.Write([]byte(`{"id":"66-1","name":"Priorities","values":[{"id":"66-11","name":"Critical"},{"id":"66-12","name":"Major"}]}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resource := &customFieldResource{client: client}
	plan := customFieldResourceModel{
		FieldTypeID: types.StringValue(testFieldTypeID),
		FieldDefaults: types.ObjectValueMust(customFieldDefaultsAttrTypes(), map[string]attr.Value{
			"can_be_empty":     types.BoolValue(true),
			"empty_field_text": types.StringValue("Not set"),
			"is_public":        types.BoolValue(false),
			"bundle_id":        types.StringNull(),
			"bundle_name":      types.StringValue(testBundleName),
			"default_value_names": types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("Major"),
			}),
		}),
	}

	apiModel := plan.toAPIModel()
	if err := resource.resolveCustomFieldDefaultValues(context.Background(), &apiModel, plan); err != nil {
		t.Fatalf("resolveCustomFieldDefaultValues returned error: %v", err)
	}

	if apiModel.FieldDefaults == nil || apiModel.FieldDefaults.Bundle == nil {
		t.Fatal("expected resolved bundle on field defaults")
	}

	helpers.AssertFieldEqual(t, "Bundle.ID", apiModel.FieldDefaults.Bundle.ID, testBundleID)
	helpers.AssertFieldEqual(t, "Bundle.Type", apiModel.FieldDefaults.Bundle.Type, bundleTypeEnum)
	helpers.AssertFieldEqual(t, "DefaultValues length", len(apiModel.FieldDefaults.DefaultValues), 1)
	helpers.AssertFieldEqual(t, "DefaultValues[0].ID", apiModel.FieldDefaults.DefaultValues[0].ID, "66-12")
	helpers.AssertFieldEqual(t, "DefaultValues[0].Name", apiModel.FieldDefaults.DefaultValues[0].Name, "Major")
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
