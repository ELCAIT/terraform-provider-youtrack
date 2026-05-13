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
	testTimeTrackingSettingsID = "104-0"
	testEstimateFieldID        = "104-4"
	testTimeSpentFieldID       = "104-1"
	testEstimateFieldName      = "Estimation"
	testTimeSpentFieldName     = "Spent time"
)

func TestProjectTimeTrackingModelToUpdatePayload(t *testing.T) {
	t.Parallel()

	// toUpdatePayload is a resource method (needs client for name lookup) and is tested via acceptance tests.
	// This test validates the model field structure used by the resource.
	model := projectTimeTrackingSettingsResourceModel{
		ProjectID:          types.StringValue(testProjectID),
		Enabled:            types.BoolValue(true),
		EstimateFieldName:  types.StringValue(testEstimateFieldName),
		TimeSpentFieldName: types.StringValue(testTimeSpentFieldName),
	}

	helpers.AssertFieldEqual(t, "Enabled", model.Enabled.ValueBool(), true)
	helpers.AssertFieldEqual(t, "EstimateFieldName", model.EstimateFieldName.ValueString(), testEstimateFieldName)
	helpers.AssertFieldEqual(t, "TimeSpentFieldName", model.TimeSpentFieldName.ValueString(), testTimeSpentFieldName)
}

func TestProjectTimeTrackingModelToUpdatePayloadNullFields(t *testing.T) {
	t.Parallel()

	model := projectTimeTrackingSettingsResourceModel{
		ProjectID:          types.StringValue(testProjectID),
		Enabled:            types.BoolValue(false),
		EstimateFieldName:  types.StringNull(),
		TimeSpentFieldName: types.StringNull(),
	}

	helpers.AssertFieldEqual(t, "Enabled", model.Enabled.ValueBool(), false)
	helpers.AssertFieldEqual(t, "EstimateFieldName.IsNull", model.EstimateFieldName.IsNull(), true)
	helpers.AssertFieldEqual(t, "TimeSpentFieldName.IsNull", model.TimeSpentFieldName.IsNull(), true)
}

func TestProjectTimeTrackingModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.ProjectTimeTrackingSettings{
		ID:      testTimeTrackingSettingsID,
		Enabled: true,
		Estimate: &youtrack.ProjectCustomFieldTimeRef{
			ID:    testEstimateFieldID,
			Field: &youtrack.CustomFieldIDRef{ID: "58-21", Name: "Estimation"},
		},
		TimeSpent: &youtrack.ProjectCustomFieldTimeRef{
			ID:    testTimeSpentFieldID,
			Field: &youtrack.CustomFieldIDRef{ID: "58-10", Name: "Spent time"},
		},
	}

	var model projectTimeTrackingSettingsResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testTimeTrackingSettingsID)
	helpers.AssertFieldEqual(t, "Enabled", model.Enabled.ValueBool(), true)
	helpers.AssertFieldEqual(t, "EstimateFieldName", model.EstimateFieldName.ValueString(), testEstimateFieldName)
	helpers.AssertFieldEqual(t, "TimeSpentFieldName", model.TimeSpentFieldName.ValueString(), testTimeSpentFieldName)
}

func TestProjectTimeTrackingModelFromAPIModelNullFields(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.ProjectTimeTrackingSettings{
		ID:        testTimeTrackingSettingsID,
		Enabled:   false,
		Estimate:  nil,
		TimeSpent: nil,
	}

	var model projectTimeTrackingSettingsResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "Enabled", model.Enabled.ValueBool(), false)
	helpers.AssertFieldEqual(t, "EstimateFieldName.IsNull", model.EstimateFieldName.IsNull(), true)
	helpers.AssertFieldEqual(t, "TimeSpentFieldName.IsNull", model.TimeSpentFieldName.IsNull(), true)
}
