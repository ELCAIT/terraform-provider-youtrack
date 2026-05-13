// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

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
	}

	bundleRef := &youtrack.BundleRef{ID: testProjectCustomFieldBundleID, Type: bundleTypeEnum}
	payload := model.toUpsertPayload(testProjectCustomFieldFieldID, bundleRef)
	helpers.AssertFieldEqual(t, "Field.ID", payload.Field.ID, testProjectCustomFieldFieldID)
	helpers.AssertFieldEqual(t, "Type", payload.Type, testProjectCustomFieldType)
	helpers.AssertFieldEqual(t, "Bundle.ID", payload.Bundle.ID, testProjectCustomFieldBundleID)
	helpers.AssertFieldEqual(t, "CanBeEmpty", *payload.CanBeEmpty, canBeEmpty)
	helpers.AssertFieldEqual(t, "EmptyFieldText", payload.EmptyFieldText, "No value")
	helpers.AssertFieldEqual(t, "IsPublic", *payload.IsPublic, isPublic)
}

func TestProjectCustomFieldModelToUpsertPayloadNullOptionals(t *testing.T) {
	t.Parallel()

	model := projectCustomFieldResourceModel{
		ProjectID: types.StringValue(testProjectID),
		FieldName: types.StringValue(testProjectCustomFieldFieldName),
		FieldType: types.StringValue(testProjectCustomFieldType), BundleName: types.StringNull(), CanBeEmpty: types.BoolNull(),
		EmptyFieldText: types.StringNull(),
		IsPublic:       types.BoolNull(),
	}

	payload := model.toUpsertPayload(testProjectCustomFieldFieldID, nil)
	helpers.AssertFieldEqual(t, "CanBeEmpty is nil", payload.CanBeEmpty == nil, true)
	helpers.AssertFieldEqual(t, "Bundle is nil", payload.Bundle == nil, true)
	helpers.AssertFieldEqual(t, "EmptyFieldText", payload.EmptyFieldText, "")
	helpers.AssertFieldEqual(t, "IsPublic is nil", payload.IsPublic == nil, true)
}

func TestProjectCustomFieldModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.ProjectCustomField{
		ID:             testProjectCustomFieldID,
		Field:          &youtrack.CustomFieldIDRef{ID: testProjectCustomFieldFieldID, Name: "Priority"},
		Bundle:         &youtrack.BundleRef{ID: testProjectCustomFieldBundleID, Name: testProjectCustomFieldBundleName},
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
}
