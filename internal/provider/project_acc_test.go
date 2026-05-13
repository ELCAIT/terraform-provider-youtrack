// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accProjectResource  = "youtrack_project.test"
	envAccProjectLeader = "YOUTRACK_TEST_PROJECT_LEADER_LOGIN"
)

func testProjectLeaderLogin(t *testing.T) string {
	t.Helper()

	login := os.Getenv(envAccProjectLeader)
	if login == "" {
		t.Skipf("set %s to run project acceptance tests", envAccProjectLeader)
	}

	return login
}

func testAccProjectConfig(name, shortName, description, leaderLogin string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "test" {
  name         = %q
  short_name   = %q
  description  = %q
  leader_login = %q
}
`, name, shortName, description, leaderLogin)
}

func testAccProjectConfigArchived(name, shortName, leaderLogin string, archived bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "test" {
  name         = %q
  short_name   = %q
  leader_login = %q
  archived     = %t
}
`, name, shortName, leaderLogin, archived)
}

func TestAccProject(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TFAccProject%d", suffix)
	nameUpdated := fmt.Sprintf("TFAccProjectUpd%d", suffix)
	shortName := fmt.Sprintf("TAP%d", suffix%10000)
	description := "Acceptance test project"
	descriptionUpdated := "Updated acceptance test project"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(name, shortName, description, leaderLogin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectResource, "name", name),
					resource.TestCheckResourceAttr(accProjectResource, "short_name", shortName),
					resource.TestCheckResourceAttr(accProjectResource, "description", description),
					resource.TestCheckResourceAttr(accProjectResource, "leader_login", leaderLogin),
					resource.TestCheckResourceAttrSet(accProjectResource, "id"),
					resource.TestCheckResourceAttr(accProjectResource, "archived", "false"),
				),
			},
			{
				Config: testAccProjectConfig(nameUpdated, shortName, descriptionUpdated, leaderLogin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectResource, "name", nameUpdated),
					resource.TestCheckResourceAttr(accProjectResource, "description", descriptionUpdated),
				),
			},
			{
				Config: testAccProjectConfigArchived(nameUpdated, shortName, leaderLogin, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectResource, "archived", "true"),
				),
			},
			{
				ResourceName:      accProjectResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
