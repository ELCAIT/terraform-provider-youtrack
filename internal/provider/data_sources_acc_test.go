package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accUserDataSourceAddr = "data.youtrack_user.test"
)

func testAccUserDataSourceConfig(login string) string {
	return providerBlock() + fmt.Sprintf(`
data "youtrack_user" "test" {
  login = %q
}
`, login)
}

func TestAccUserDataSource(t *testing.T) {
	skipUnlessAcc(t)

	login := os.Getenv(envAccUser)
	if login == "" {
		t.Skip("set YOUTRACK_TEST_USER to run user data source acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig(login),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accUserDataSourceAddr, "login", login),
					resource.TestCheckResourceAttrSet(accUserDataSourceAddr, "id"),
					resource.TestCheckResourceAttrSet(accUserDataSourceAddr, "full_name"),
				),
			},
		},
	})
}

func testAccGroupDataSourceConfig(name string) string {
	return providerBlock() + fmt.Sprintf(`
data "youtrack_group" "test" {
  name = %q
}
`, name)
}

func TestAccGroupDataSource(t *testing.T) {
	skipUnlessAcc(t)

	groupName := os.Getenv(envAccGroupName)
	if groupName == "" {
		t.Skip("set YOUTRACK_TEST_GROUP_NAME to run group data source acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.youtrack_group.test", "name", groupName),
					resource.TestCheckResourceAttrSet("data.youtrack_group.test", "id"),
				),
			},
		},
	})
}
