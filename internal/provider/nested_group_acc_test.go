package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accNestedGroupResource = "youtrack_nested_group.test"
)

func testAccNestedGroupConfig(name, description string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_nested_group" "test" {
  name        = %q
  description = %q
}
`, name, description)
}

func testAccNestedGroupConfigWithUser(name, description, userLogin string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_nested_group" "test" {
  name             = %q
  description      = %q
  own_user_logins  = [%q]
}
`, name, description, userLogin)
}

func TestAccNestedGroup(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("tf-acc-group-%d", suffix)
	description := "Acceptance test group"
	descriptionUpdated := "Acceptance test group (updated)"

	userLogin := os.Getenv(envAccUser)

	steps := []resource.TestStep{
		{
			Config: testAccNestedGroupConfig(name, description),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(accNestedGroupResource, "name", name),
				resource.TestCheckResourceAttr(accNestedGroupResource, "description", description),
				resource.TestCheckResourceAttrSet(accNestedGroupResource, "id"),
			),
		},
		{
			Config: testAccNestedGroupConfig(name, descriptionUpdated),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(accNestedGroupResource, "description", descriptionUpdated),
			),
		},
		{
			ResourceName:      accNestedGroupResource,
			ImportState:       true,
			ImportStateVerify: true,
		},
	}

	if userLogin != "" {
		steps = append(steps, resource.TestStep{
			Config: testAccNestedGroupConfigWithUser(name, descriptionUpdated, userLogin),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(accNestedGroupResource, "own_user_logins.#", "1"),
			),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps:                    steps,
	})
}
