// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testProjectCustomFieldID         = "82-8"
	testProjectCustomFieldFieldID    = "46-21"
	testProjectCustomFieldFieldName  = "Priority"
	testProjectCustomFieldBundleID   = "71-1"
	testProjectCustomFieldBundleName = "My Priorities"
	testProjectCustomFieldType       = "EnumProjectCustomField"
)

func TestProjectCustomFieldModelToUpsertPayload(t *testing.T) {
	t.Parallel()

	canBeEmpty := true
	isPublic := false

	model := projectCustomFieldResourceModel{
		ProjectID:      types.StringValue(testProjectID),
		FieldName:      types.StringValue(testProjectCustomFieldFieldName),
		FieldType:      types.StringValue(testProjectCustomFieldType),
		BundleName:     types.StringValue(testProjectCustomFieldBundleName),
		CanBeEmpty:     types.BoolValue(canBeEmpty),
		EmptyFieldText: types.StringValue("No value"),
		IsPublic:       types.BoolValue(isPublic),
		DefaultValueNames: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("High"),
			types.StringValue("Low"),
		}),
	}

	bundleRef := &youtrack.BundleRef{ID: testProjectCustomFieldBundleID, Type: bundleTypeEnum}
	defaultValues := []youtrack.ProjectCustomFieldValueRef{{ID: "80-1", Name: "High"}, {ID: "80-2", Name: "Low"}}
	payload := model.toUpsertPayload(testProjectCustomFieldFieldID, bundleRef, defaultValues)
	helpers.AssertFieldEqual(t, "Field.ID", payload.Field.ID, testProjectCustomFieldFieldID)
	helpers.AssertFieldEqual(t, "Type", payload.Type, testProjectCustomFieldType)
	helpers.AssertFieldEqual(t, "Bundle.ID", payload.Bundle.ID, testProjectCustomFieldBundleID)
	helpers.AssertFieldEqual(t, "CanBeEmpty", *payload.CanBeEmpty, canBeEmpty)
	helpers.AssertFieldEqual(t, "EmptyFieldText", payload.EmptyFieldText, "No value")
	helpers.AssertFieldEqual(t, "IsPublic", *payload.IsPublic, isPublic)
	helpers.AssertFieldEqual(t, "DefaultValues length", len(payload.DefaultValues), 2)
	helpers.AssertFieldEqual(t, "DefaultValues[0].Name", payload.DefaultValues[0].Name, "High")
}

func TestProjectCustomFieldModelToUpsertPayloadNullOptionals(t *testing.T) {
	t.Parallel()

	model := projectCustomFieldResourceModel{
		ProjectID:         types.StringValue(testProjectID),
		FieldName:         types.StringValue(testProjectCustomFieldFieldName),
		FieldType:         types.StringValue(testProjectCustomFieldType),
		BundleName:        types.StringNull(),
		CanBeEmpty:        types.BoolNull(),
		EmptyFieldText:    types.StringNull(),
		IsPublic:          types.BoolNull(),
		DefaultValueNames: types.ListNull(types.StringType),
	}

	payload := model.toUpsertPayload(testProjectCustomFieldFieldID, nil, nil)
	helpers.AssertFieldEqual(t, "CanBeEmpty is nil", payload.CanBeEmpty == nil, true)
	helpers.AssertFieldEqual(t, "Bundle is nil", payload.Bundle == nil, true)
	helpers.AssertFieldEqual(t, "EmptyFieldText", payload.EmptyFieldText, "")
	helpers.AssertFieldEqual(t, "IsPublic is nil", payload.IsPublic == nil, true)
	helpers.AssertFieldEqual(t, "DefaultValues is empty", len(payload.DefaultValues), 0)
}

func TestProjectCustomFieldModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.ProjectCustomField{
		ID:             testProjectCustomFieldID,
		Field:          &youtrack.CustomFieldIDRef{ID: testProjectCustomFieldFieldID, Name: "Priority"},
		Bundle:         &youtrack.BundleRef{ID: testProjectCustomFieldBundleID, Name: testProjectCustomFieldBundleName},
		DefaultValues:  []youtrack.ProjectCustomFieldValueRef{{ID: "80-1", Name: "High"}, {ID: "80-2", Name: "Low"}},
		CanBeEmpty:     true,
		EmptyFieldText: "No priority",
		IsPublic:       true,
		Type:           testProjectCustomFieldType,
	}

	var model projectCustomFieldResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testProjectCustomFieldID)
	helpers.AssertFieldEqual(t, "FieldName", model.FieldName.ValueString(), "Priority")
	helpers.AssertFieldEqual(t, "CanBeEmpty", model.CanBeEmpty.ValueBool(), true)
	helpers.AssertFieldEqual(t, "EmptyFieldText", model.EmptyFieldText.ValueString(), "No priority")
	helpers.AssertFieldEqual(t, "IsPublic", model.IsPublic.ValueBool(), true)
	helpers.AssertFieldEqual(t, "BundleName", model.BundleName.ValueString(), testProjectCustomFieldBundleName)
	helpers.AssertFieldEqual(t, "FieldType", model.FieldType.ValueString(), testProjectCustomFieldType)

	defaultValueNames, ok := helpers.ListToStringSlice(context.Background(), model.DefaultValueNames)
	helpers.AssertFieldEqual(t, "DefaultValueNames decode", ok, true)
	helpers.AssertFieldEqual(t, "DefaultValueNames length", len(defaultValueNames), 2)
	helpers.AssertFieldEqual(t, "DefaultValueNames[0]", defaultValueNames[0], "High")
}

func TestProjectCustomFieldModelFromAPIModelNullEmptyFieldText(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.ProjectCustomField{
		ID:         testProjectCustomFieldID,
		Field:      &youtrack.CustomFieldIDRef{ID: testProjectCustomFieldFieldID},
		CanBeEmpty: false,
		IsPublic:   false,
		Type:       testProjectCustomFieldType,
	}

	var model projectCustomFieldResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "EmptyFieldText.IsNull", model.EmptyFieldText.IsNull(), true)
	helpers.AssertFieldEqual(t, "CanBeEmpty", model.CanBeEmpty.ValueBool(), false)
	helpers.AssertFieldEqual(t, "IsPublic", model.IsPublic.ValueBool(), false)
	helpers.AssertFieldEqual(t, "DefaultValueNames.IsNull", model.DefaultValueNames.IsNull(), true)
}

func TestDeriveProjectCustomFieldTypeFromIssueType(t *testing.T) {
	t.Parallel()

	derived, err := deriveProjectCustomFieldType("EnumIssueCustomField")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	helpers.AssertFieldEqual(t, "derived type", derived, "EnumProjectCustomField")
}

func TestDeriveProjectCustomFieldTypeKeepsProjectType(t *testing.T) {
	t.Parallel()

	derived, err := deriveProjectCustomFieldType("PeriodProjectCustomField")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	helpers.AssertFieldEqual(t, "derived type", derived, "PeriodProjectCustomField")
}

func TestDeriveProjectCustomFieldTypeInvalidType(t *testing.T) {
	t.Parallel()

	_, err := deriveProjectCustomFieldType("")
	helpers.AssertFieldEqual(t, "error is returned", err != nil, true)
}

func TestDeriveProjectCustomFieldTypeFromFieldTypeIDEnum(t *testing.T) {
	t.Parallel()

	derived, err := deriveProjectCustomFieldTypeFromFieldTypeID("enum[1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	helpers.AssertFieldEqual(t, "derived type", derived, "EnumProjectCustomField")
}

func TestDeriveProjectCustomFieldTypeFromFieldTypeIDPeriod(t *testing.T) {
	t.Parallel()

	derived, err := deriveProjectCustomFieldTypeFromFieldTypeID("period")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	helpers.AssertFieldEqual(t, "derived type", derived, "PeriodProjectCustomField")
}

func TestDeriveProjectCustomFieldTypeFromFieldTypeIDSimple(t *testing.T) {
	t.Parallel()

	derived, err := deriveProjectCustomFieldTypeFromFieldTypeID("integer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	helpers.AssertFieldEqual(t, "derived type", derived, "SimpleProjectCustomField")
}

func TestDeriveProjectCustomFieldTypeFromFieldTypeIDInvalid(t *testing.T) {
	t.Parallel()

	_, err := deriveProjectCustomFieldTypeFromFieldTypeID("mystery[1]")
	helpers.AssertFieldEqual(t, "error is returned", err != nil, true)
}
