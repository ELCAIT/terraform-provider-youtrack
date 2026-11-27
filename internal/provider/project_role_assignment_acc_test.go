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
	accProjectRoleAssignmentUser  = "youtrack_project_role_assignment.user"
	accProjectRoleAssignmentGroup = "youtrack_project_role_assignment.group"
)

// testAccProjectRoleAssignmentConfig renders the resources, with the user assignment holding the role
// resource named userRole, which is either "test" or "other".
func testAccProjectRoleAssignmentConfig(suffix int64, shortName, leaderLogin, permission, userRole string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "test" {
  name         = "TFAccPRA%[1]d"
  short_name   = %[2]q
  leader_login = %[3]q
}

resource "youtrack_role" "test" {
  name        = "TF Acc PRA Role %[1]d"
  permissions = [%[4]q]
}

resource "youtrack_role" "other" {
  name        = "TF Acc PRA Other Role %[1]d"
  permissions = [%[4]q]
}

resource "youtrack_nested_group" "test" {
  name = "tf-acc-pra-group-%[1]d"
}

resource "youtrack_project_role_assignment" "user" {
  role_id      = youtrack_role.%[5]s.id
  project_id   = youtrack_project.test.id
  holder_login = %[3]q
  holder_type  = "user"
}

resource "youtrack_project_role_assignment" "group" {
  role_id      = youtrack_role.test.id
  project_id   = youtrack_project.test.id
  holder_login = youtrack_nested_group.test.name
  holder_type  = "group"
}
`, suffix, shortName, leaderLogin, permission, userRole)
}

func TestAccProjectRoleAssignment(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	permission := os.Getenv(envAccPermission)
	if permission == "" {
		t.Skip("set YOUTRACK_TEST_PERMISSION (e.g. 'Read Issue') to run project role assignment acceptance tests")
	}

	suffix := time.Now().UnixMilli()
	shortName := fmt.Sprintf("TPR%d", suffix%10000)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRoleAssignmentConfig(suffix, shortName, leaderLogin, permission, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectRoleAssignmentUser, "id"),
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentUser, "role_id", "youtrack_role.test", "id"),
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentUser, "project_id", "youtrack_project.test", "id"),
					resource.TestCheckResourceAttr(accProjectRoleAssignmentUser, "holder_login", leaderLogin),
					resource.TestCheckResourceAttr(accProjectRoleAssignmentUser, "holder_type", "user"),
					resource.TestCheckResourceAttrSet(accProjectRoleAssignmentGroup, "id"),
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentGroup, "project_id", "youtrack_project.test", "id"),
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentGroup, "holder_login", "youtrack_nested_group.test", "name"),
					resource.TestCheckResourceAttr(accProjectRoleAssignmentGroup, "holder_type", "group"),
				),
			},
			{
				ResourceName:      accProjectRoleAssignmentUser,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      accProjectRoleAssignmentGroup,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Every attribute forces a replacement, so changing the role replaces the assignment.
				Config: testAccProjectRoleAssignmentConfig(suffix, shortName, leaderLogin, permission, "other"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentUser, "role_id", "youtrack_role.other", "id"),
					resource.TestCheckResourceAttrPair(accProjectRoleAssignmentGroup, "role_id", "youtrack_role.test", "id"),
				),
			},
		},
	})
}
