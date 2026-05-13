// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testProjectID               = "0-6"
	testProjectName             = "TF Test Project"
	testProjectShortName        = "TTP"
	testProjectDescription      = "A test project"
	testProjectLeaderIDValue    = "1-1"
	testProjectLeaderLoginValue = "admin"
	testFromEmail               = "from@example.com"
	testReplyToEmail            = "reply@example.com"
)

func TestProjectModelToCreatePayload(t *testing.T) {
	t.Parallel()

	model := projectResourceModel{
		Name:        types.StringValue(testProjectName),
		ShortName:   types.StringValue(testProjectShortName),
		Description: types.StringValue(testProjectDescription),
		LeaderLogin: types.StringValue(testProjectLeaderLoginValue),
		Template:    types.BoolValue(false),
	}

	payload := model.toCreatePayload(testProjectLeaderIDValue)
	helpers.AssertFieldEqual(t, "Name", payload.Name, testProjectName)
	helpers.AssertFieldEqual(t, "ShortName", payload.ShortName, testProjectShortName)
	helpers.AssertFieldEqual(t, "Description", payload.Description, testProjectDescription)
	helpers.AssertFieldEqual(t, "Leader.ID", payload.Leader.ID, testProjectLeaderIDValue)
}

func TestProjectModelToCreatePayloadNullOptionals(t *testing.T) {
	t.Parallel()

	model := projectResourceModel{
		Name:        types.StringValue(testProjectName),
		ShortName:   types.StringValue(testProjectShortName),
		Description: types.StringNull(),
		LeaderLogin: types.StringValue(testProjectLeaderLoginValue),
		Template:    types.BoolNull(),
	}

	payload := model.toCreatePayload(testProjectLeaderIDValue)
	helpers.AssertFieldEqual(t, "Description", payload.Description, "")
	helpers.AssertFieldEqual(t, "Template is nil", payload.Template == nil, true)
}

func TestProjectModelToUpdatePayload(t *testing.T) {
	t.Parallel()

	model := projectResourceModel{
		ID:           types.StringValue(testProjectID),
		Name:         types.StringValue(testProjectName),
		Description:  types.StringValue(testProjectDescription),
		LeaderLogin:  types.StringValue(testProjectLeaderLoginValue),
		Archived:     types.BoolValue(true),
		FromEmail:    types.StringValue(testFromEmail),
		ReplyToEmail: types.StringValue(testReplyToEmail),
	}

	payload := model.toUpdatePayload(testProjectLeaderIDValue)
	helpers.AssertFieldEqual(t, "Name", payload.Name, testProjectName)
	helpers.AssertFieldEqual(t, "Description", payload.Description, testProjectDescription)
	helpers.AssertFieldEqual(t, "Leader.ID", payload.Leader.ID, testProjectLeaderIDValue)
	helpers.AssertFieldEqual(t, "Archived", *payload.Archived, true)
	helpers.AssertFieldEqual(t, "FromEmail", payload.FromEmail, testFromEmail)
	helpers.AssertFieldEqual(t, "ReplyToEmail", payload.ReplyToEmail, testReplyToEmail)
}

func TestProjectModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.Project{
		ID:           testProjectID,
		Name:         testProjectName,
		ShortName:    testProjectShortName,
		Description:  testProjectDescription,
		Leader:       &youtrack.UserRef{ID: testProjectLeaderIDValue, Login: testProjectLeaderLoginValue, Name: "Admin User"},
		Archived:     false,
		Template:     false,
		FromEmail:    testFromEmail,
		ReplyToEmail: testReplyToEmail,
	}

	var model projectResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testProjectID)
	helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), testProjectName)
	helpers.AssertFieldEqual(t, "ShortName", model.ShortName.ValueString(), testProjectShortName)
	helpers.AssertFieldEqual(t, "Description", model.Description.ValueString(), testProjectDescription)
	helpers.AssertFieldEqual(t, "LeaderLogin", model.LeaderLogin.ValueString(), testProjectLeaderLoginValue)
	helpers.AssertFieldEqual(t, "Archived", model.Archived.ValueBool(), false)
	helpers.AssertFieldEqual(t, "Template", model.Template.ValueBool(), false)
	helpers.AssertFieldEqual(t, "FromEmail", model.FromEmail.ValueString(), testFromEmail)
	helpers.AssertFieldEqual(t, "ReplyToEmail", model.ReplyToEmail.ValueString(), testReplyToEmail)
}

func TestProjectModelFromAPIModelNullableFieldsAreNull(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.Project{
		ID:        testProjectID,
		Name:      testProjectName,
		ShortName: testProjectShortName,
	}

	var model projectResourceModel
	model.fromAPIModel(apiModel)

	helpers.AssertFieldEqual(t, "Description.IsNull", model.Description.IsNull(), true)
	helpers.AssertFieldEqual(t, "LeaderLogin.IsNull", model.LeaderLogin.IsNull(), true)
	helpers.AssertFieldEqual(t, "FromEmail.IsNull", model.FromEmail.IsNull(), true)
	helpers.AssertFieldEqual(t, "ReplyToEmail.IsNull", model.ReplyToEmail.IsNull(), true)
}
