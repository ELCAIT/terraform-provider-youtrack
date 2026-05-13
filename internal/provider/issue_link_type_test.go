package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testIssueLinkTypeID             = "80-9"
	testIssueLinkTypeName           = "Relates"
	testIssueLinkTypeSourceToTarget = "relates to"
	testIssueLinkTypeTargetToSource = "related to"
	testLocalizedName               = "Verweist"
)

func TestIssueLinkTypeModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := issueLinkTypeResourceModel{
		Name:                    types.StringValue(testIssueLinkTypeName),
		SourceToTarget:          types.StringValue(testIssueLinkTypeSourceToTarget),
		TargetToSource:          types.StringValue(testIssueLinkTypeTargetToSource),
		Directed:                types.BoolValue(true),
		Aggregation:             types.BoolValue(false),
		LocalizedName:           types.StringValue(testLocalizedName),
		LocalizedSourceToTarget: types.StringNull(),
		LocalizedTargetToSource: types.StringNull(),
	}

	apiModel := model.toAPIModel()
	helpers.AssertFieldEqual(t, "Name", apiModel.Name, testIssueLinkTypeName)
	helpers.AssertFieldEqual(t, "SourceToTarget", apiModel.SourceToTarget, testIssueLinkTypeSourceToTarget)
	helpers.AssertFieldEqual(t, "TargetToSource", apiModel.TargetToSource, testIssueLinkTypeTargetToSource)
	helpers.AssertFieldEqual(t, "Directed", apiModel.Directed, true)
	helpers.AssertFieldEqual(t, "Aggregation", apiModel.Aggregation, false)
	helpers.AssertFieldEqual(t, "LocalizedName", apiModel.LocalizedName, testLocalizedName)
}

func TestIssueLinkTypeModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.IssueLinkType{
		ID:                      testIssueLinkTypeID,
		Name:                    testIssueLinkTypeName,
		SourceToTarget:          testIssueLinkTypeSourceToTarget,
		TargetToSource:          testIssueLinkTypeTargetToSource,
		Directed:                true,
		Aggregation:             false,
		ReadOnly:                true,
		LocalizedName:           testLocalizedName,
		LocalizedSourceToTarget: "nach außen",
		LocalizedTargetToSource: "nach innen",
	}

	var model issueLinkTypeResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testIssueLinkTypeID)
	helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), testIssueLinkTypeName)
	helpers.AssertFieldEqual(t, "SourceToTarget", model.SourceToTarget.ValueString(), testIssueLinkTypeSourceToTarget)
	helpers.AssertFieldEqual(t, "TargetToSource", model.TargetToSource.ValueString(), testIssueLinkTypeTargetToSource)
	helpers.AssertFieldEqual(t, "Directed", model.Directed.ValueBool(), true)
	helpers.AssertFieldEqual(t, "Aggregation", model.Aggregation.ValueBool(), false)
	helpers.AssertFieldEqual(t, "ReadOnly", model.ReadOnly.ValueBool(), true)
	helpers.AssertFieldEqual(t, "LocalizedName", model.LocalizedName.ValueString(), testLocalizedName)
	helpers.AssertFieldEqual(t, "LocalizedSourceToTarget", model.LocalizedSourceToTarget.ValueString(), "nach außen")
	helpers.AssertFieldEqual(t, "LocalizedTargetToSource", model.LocalizedTargetToSource.ValueString(), "nach innen")
}

func TestIssueLinkTypeModelFromAPIModelEmptyLocalizedFieldsAreNull(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.IssueLinkType{
		ID:             testIssueLinkTypeID,
		Name:           testIssueLinkTypeName,
		SourceToTarget: testIssueLinkTypeSourceToTarget,
		TargetToSource: testIssueLinkTypeTargetToSource,
	}

	var model issueLinkTypeResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "LocalizedName.IsNull", model.LocalizedName.IsNull(), true)
	helpers.AssertFieldEqual(t, "LocalizedSourceToTarget.IsNull", model.LocalizedSourceToTarget.IsNull(), true)
	helpers.AssertFieldEqual(t, "LocalizedTargetToSource.IsNull", model.LocalizedTargetToSource.IsNull(), true)
}
