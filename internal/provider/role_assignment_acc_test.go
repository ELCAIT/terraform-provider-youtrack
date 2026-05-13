package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accRoleAssignmentResource = "youtrack_role_assignment.test"
)

func testAccRoleAssignmentConfig(roleName, permission, holderLogin string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_role" "assignment_test" {
  name        = %q
  permissions = [%q]
}

resource "youtrack_role_assignment" "test" {
  role_id      = youtrack_role.assignment_test.id
  holder_login = %q
  holder_type  = "user"
}
`, roleName, permission, holderLogin)
}

func TestAccRoleAssignment(t *testing.T) {
	skipUnlessAcc(t)

	userLogin := os.Getenv(envAccUser)
	if userLogin == "" {
		t.Skip("set YOUTRACK_TEST_USER to run role assignment acceptance tests")
	}

	permission := os.Getenv(envAccPermission)
	if permission == "" {
		t.Skip("set YOUTRACK_TEST_PERMISSION (e.g. 'Read Issue') to run role assignment acceptance tests")
	}

	suffix := time.Now().UnixMilli()
	roleName := fmt.Sprintf("TF Acc RA Role %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleAssignmentConfig(roleName, permission, userLogin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accRoleAssignmentResource, "role_id"),
					resource.TestCheckResourceAttr(accRoleAssignmentResource, "holder_login", userLogin),
					resource.TestCheckResourceAttr(accRoleAssignmentResource, "holder_type", "user"),
					resource.TestCheckResourceAttrSet(accRoleAssignmentResource, "id"),
					resource.TestCheckResourceAttrSet(accRoleAssignmentResource, "holder_id"),
				),
			},
			{
				ResourceName:      accRoleAssignmentResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
