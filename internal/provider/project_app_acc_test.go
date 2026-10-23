// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	accProjectAppResource = "youtrack_project_app.test"
	envAccProjectAppName  = "YOUTRACK_TEST_PROJECT_APP_NAME"

	// defaultAccProjectAppName is a workflow app bundled with every YouTrack instance.
	defaultAccProjectAppName = "@jetbrains/youtrack-workflow-due-date"
)

func accProjectAppName() string {
	if name := os.Getenv(envAccProjectAppName); name != "" {
		return name
	}

	return defaultAccProjectAppName
}

// testAccProjectAppConfig attaches the app to a template project, which is how
// projects created from the template inherit it. A nil enabled leaves the
// attribute unset so its default applies.
func testAccProjectAppConfig(projectName, shortName, leaderLogin, appName string, enabled *bool) string {
	enabledLine := ""
	if enabled != nil {
		enabledLine = fmt.Sprintf("enabled    = %t", *enabled)
	}

	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "parent" {
  name         = %q
  short_name   = %q
  leader_login = %q
  template     = true
}

resource "youtrack_project_app" "test" {
  project_id = youtrack_project.parent.id
  app_name   = %q
  %s
}
`, projectName, shortName, leaderLogin, appName, enabledLine)
}

// testAccDetachProjectApp detaches the app outside of Terraform, so the next
// step proves the drift is detected and the app attached again.
func testAccDetachProjectApp(t *testing.T) func(*terraform.State) error {
	t.Helper()

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[accProjectAppResource]
		if !ok {
			return fmt.Errorf("resource %q not found", accProjectAppResource)
		}

		client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
		if err != nil {
			return err
		}

		return client.DetachAppFromProject(context.Background(), rs.Primary.Attributes["app_id"], rs.Primary.Attributes["project_id"])
	}
}

func TestAccProjectApp(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	appName := accProjectAppName()

	suffix := time.Now().UnixMilli()
	projectName := fmt.Sprintf("TFAccPApp%d", suffix)
	shortName := fmt.Sprintf("PAPP%d", suffix%10000)

	disabled, enabled := false, true

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectAppConfig(projectName, shortName, leaderLogin, appName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectAppResource, "id"),
					resource.TestCheckResourceAttrSet(accProjectAppResource, "app_id"),
					resource.TestCheckResourceAttr(accProjectAppResource, "app_name", appName),
					resource.TestCheckResourceAttr(accProjectAppResource, "enabled", "true"),
				),
			},
			{
				Config: testAccProjectAppConfig(projectName, shortName, leaderLogin, appName, &disabled),
				Check:  resource.TestCheckResourceAttr(accProjectAppResource, "enabled", "false"),
			},
			{
				Config: testAccProjectAppConfig(projectName, shortName, leaderLogin, appName, &enabled),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectAppResource, "enabled", "true"),
					testAccDetachProjectApp(t),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccProjectAppConfig(projectName, shortName, leaderLogin, appName, &enabled),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectAppResource, "id"),
					resource.TestCheckResourceAttr(accProjectAppResource, "enabled", "true"),
				),
			},
			{
				ResourceName:      accProjectAppResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[accProjectAppResource]
					if !ok {
						return "", fmt.Errorf("resource %q not found", accProjectAppResource)
					}
					return rs.Primary.Attributes["project_id"] + "/" + rs.Primary.Attributes["app_name"], nil
				},
			},
		},
	})
}
