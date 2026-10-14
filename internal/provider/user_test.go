package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testUserResourceID       = "1-42"
	testUserResourceLogin    = "svc-ci"
	testUserResourceFullName = "CI Service Account"
	testUserResourceEmail    = "svc-ci@example.com"
	testUserResourcePassword = "s3cret-Pa55!"
)

func testUserResourceModel() userResourceModel {
	return userResourceModel{
		ID:       types.StringValue(testUserResourceID),
		Login:    types.StringValue(testUserResourceLogin),
		FullName: types.StringValue(testUserResourceFullName),
		Email:    types.StringValue(testUserResourceEmail),
		Password: types.StringValue(testUserResourcePassword),
		Banned:   types.BoolValue(true),
	}
}

func TestUserModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := testUserResourceModel()
	apiModel := model.toAPIModel()

	helpers.AssertFieldEqual(t, "Login", apiModel.Login, testUserResourceLogin)
	helpers.AssertFieldEqual(t, "FullName", apiModel.FullName, testUserResourceFullName)
	helpers.AssertFieldEqual(t, "Email", apiModel.Email, testUserResourceEmail)
	helpers.AssertFieldEqual(t, "Password", apiModel.Password, testUserResourcePassword)
	helpers.AssertFieldEqual(t, "Banned", apiModel.Banned, true)
	helpers.AssertFieldEqual(t, "ID", apiModel.ID, "")
}

func TestUserModelToAPIModelOmitsUnsetOptionals(t *testing.T) {
	t.Parallel()

	model := testUserResourceModel()
	model.Email = types.StringUnknown()
	model.Password = types.StringNull()

	apiModel := model.toAPIModel()

	helpers.AssertFieldEqual(t, "Email", apiModel.Email, "")
	helpers.AssertFieldEqual(t, "Password", apiModel.Password, "")
}

func TestUserModelToUpdateAPIModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		priorPassword types.String
		wantPassword  string
	}{
		{name: "unchanged password is not resent", priorPassword: types.StringValue(testUserResourcePassword), wantPassword: ""},
		{name: "changed password is sent", priorPassword: types.StringValue("old-password"), wantPassword: testUserResourcePassword},
		{name: "newly set password is sent", priorPassword: types.StringNull(), wantPassword: testUserResourcePassword},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plan := testUserResourceModel()
			prior := testUserResourceModel()
			prior.Password = tc.priorPassword

			apiModel := plan.toUpdateAPIModel(prior)

			helpers.AssertFieldEqual(t, "Password", apiModel.Password, tc.wantPassword)
			helpers.AssertFieldEqual(t, "Banned", apiModel.Banned, false)
			helpers.AssertFieldEqual(t, "Login", apiModel.Login, testUserResourceLogin)
		})
	}
}

func TestUserModelFromAPIModel(t *testing.T) {
	t.Parallel()

	model := userResourceModel{Password: types.StringValue(testUserResourcePassword)}
	model.fromAPIModel(&youtrack.User{
		ID:       testUserResourceID,
		Login:    testUserResourceLogin,
		FullName: testUserResourceFullName,
		Banned:   true,
	})

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), testUserResourceID)
	helpers.AssertFieldEqual(t, "Login", model.Login.ValueString(), testUserResourceLogin)
	helpers.AssertFieldEqual(t, "FullName", model.FullName.ValueString(), testUserResourceFullName)
	helpers.AssertFieldEqual(t, "Email is null", model.Email.IsNull(), true)
	helpers.AssertFieldEqual(t, "Banned", model.Banned.ValueBool(), true)
	helpers.AssertFieldEqual(t, "Password preserved", model.Password.ValueString(), testUserResourcePassword)
}
