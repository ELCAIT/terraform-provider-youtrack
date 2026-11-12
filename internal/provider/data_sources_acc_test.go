package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

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

func TestAccProjectDataSource(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	suffix := time.Now().UnixMilli() % 1000000
	const addr = "data.youtrack_project.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "test" {
  name         = "tf-acc-project-ds-%[1]d"
  short_name   = "PDS%[1]d"
  description  = "looked up by a data source"
  leader_login = %[2]q
}

data "youtrack_project" "test" {
  short_name = youtrack_project.test.short_name
}
`, suffix, leaderLogin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(addr, "id", "youtrack_project.test", "id"),
					resource.TestCheckResourceAttr(addr, "name", fmt.Sprintf("tf-acc-project-ds-%d", suffix)),
					resource.TestCheckResourceAttr(addr, "description", "looked up by a data source"),
					resource.TestCheckResourceAttr(addr, "leader_login", leaderLogin),
					resource.TestCheckResourceAttr(addr, "archived", "false"),
				),
			},
		},
	})
}
