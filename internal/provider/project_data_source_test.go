package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"
)

func TestProjectDataSourceFromAPIModel(t *testing.T) {
	t.Parallel()

	var model projectDataSourceModel
	model.fromAPIModel(&youtrack.Project{
		ID:        "0-0",
		Name:      "Demo project",
		ShortName: "DEMO",
		Leader:    &youtrack.UserRef{ID: "2-1", Login: "admin"},
		Template:  true,
	})

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), "0-0")
	helpers.AssertFieldEqual(t, "ShortName", model.ShortName.ValueString(), "DEMO")
	helpers.AssertFieldEqual(t, "LeaderLogin", model.LeaderLogin.ValueString(), "admin")
	helpers.AssertFieldEqual(t, "Description null", model.Description.IsNull(), true)
	helpers.AssertFieldEqual(t, "Template", model.Template.ValueBool(), true)

	model.fromAPIModel(&youtrack.Project{ID: "0-1", ShortName: "NOLEAD"})
	helpers.AssertFieldEqual(t, "LeaderLogin null", model.LeaderLogin.IsNull(), true)
}
