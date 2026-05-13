package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accRoleResource = "youtrack_role.test"
)

func testAccRoleConfig(name, permission string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_role" "test" {
  name        = %q
  permissions = [%q]
}
`, name, permission)
}

func TestAccRole(t *testing.T) {
	skipUnlessAcc(t)

	permission := os.Getenv(envAccPermission)
	if permission == "" {
		t.Skip("set YOUTRACK_TEST_PERMISSION (e.g. 'Read Issue') to run role acceptance tests")
	}

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TF Acc Role %d", suffix)
	nameUpdated := fmt.Sprintf("TF Acc Role Updated %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig(name, permission),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accRoleResource, "name", name),
					resource.TestCheckResourceAttr(accRoleResource, "permissions.0", permission),
					resource.TestCheckResourceAttrSet(accRoleResource, "id"),
				),
			},
			{
				Config: testAccRoleConfig(nameUpdated, permission),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accRoleResource, "name", nameUpdated),
				),
			},
			{
				ResourceName:            accRoleResource,
				ImportState:             true,
				ImportStateIdFunc:       importStateID(accRoleResource),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"permissions"},
			},
		},
	})
}
