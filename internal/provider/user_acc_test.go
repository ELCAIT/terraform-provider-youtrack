package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accUserResource = "youtrack_user.test"
)

func testAccUserConfig(login, fullName, email string, banned bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_user" "test" {
  login     = %q
  full_name = %q
  email     = %q
  password  = "TfAcc-%s-Aa1!"
  banned    = %t
}
`, login, fullName, email, login, banned)
}

func TestAccUser(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	login := fmt.Sprintf("tf-acc-svc-%d", suffix)
	loginUpdated := login + "-renamed"
	email := login + "@example.com"
	emailUpdated := loginUpdated + "@example.com"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(login, "TF Acc Service Account", email, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accUserResource, "id"),
					resource.TestCheckResourceAttr(accUserResource, "login", login),
					resource.TestCheckResourceAttr(accUserResource, "full_name", "TF Acc Service Account"),
					resource.TestCheckResourceAttr(accUserResource, "email", email),
					resource.TestCheckResourceAttr(accUserResource, "banned", "false"),
				),
			},
			{
				Config: testAccUserConfig(loginUpdated, "TF Acc Service Account Updated", emailUpdated, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accUserResource, "login", loginUpdated),
					resource.TestCheckResourceAttr(accUserResource, "full_name", "TF Acc Service Account Updated"),
					resource.TestCheckResourceAttr(accUserResource, "email", emailUpdated),
					resource.TestCheckResourceAttr(accUserResource, "banned", "true"),
				),
			},
			{
				Config: testAccUserConfig(loginUpdated, "TF Acc Service Account Updated", emailUpdated, false),
				Check:  resource.TestCheckResourceAttr(accUserResource, "banned", "false"),
			},
			{
				ResourceName:            accUserResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}
