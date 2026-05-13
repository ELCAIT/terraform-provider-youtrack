// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	accProjectTimeTrackingResource = "youtrack_project_time_tracking_settings.test"
)

func testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin string, enabled bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "parent" {
  name         = %q
  short_name   = %q
  leader_login = %q
}

resource "youtrack_project_time_tracking_settings" "test" {
  project_id = youtrack_project.parent.id
  enabled    = %t
}
`, projectName, shortName, leaderLogin, enabled)
}

func TestAccProjectTimeTrackingSettings(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	suffix := time.Now().UnixMilli()
	projectName := fmt.Sprintf("TFAccPTT%d", suffix)
	shortName := fmt.Sprintf("PTT%d", suffix%10000)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectTimeTrackingResource, "id"),
					resource.TestCheckResourceAttr(accProjectTimeTrackingResource, "enabled", "true"),
					resource.TestCheckResourceAttrSet(accProjectTimeTrackingResource, "project_id"),
				),
			},
			{
				Config: testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectTimeTrackingResource, "enabled", "false"),
				),
			},
			{
				ResourceName:      accProjectTimeTrackingResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[accProjectTimeTrackingResource]
					if !ok {
						return "", fmt.Errorf("resource %q not found", accProjectTimeTrackingResource)
					}
					return rs.Primary.Attributes["project_id"], nil
				},
			},
		},
	})
}
