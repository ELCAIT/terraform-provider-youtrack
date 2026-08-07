package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const accServiceResourceAddr = "youtrack_service.test"

func testAccServiceConfig(name, description string, trusted bool, redirectURI string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_service" "test" {
  name              = %q
  application_name  = "tf-acc-service-app"
  home_url          = "https://tf-acc-service.example.com"
  description       = %q
  trusted           = %t
  redirect_uris     = [%q]
}
`, name, description, trusted, redirectURI)
}

func TestAccServiceResource(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("tf-acc-service-%d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfig(name, "initial description", false, "https://tf-acc-service.example.com/callback"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accServiceResourceAddr, "name", name),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "application_name", "tf-acc-service-app"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "description", "initial description"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "trusted", "false"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "consent_required", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "redirect_uris.0", "https://tf-acc-service.example.com/callback"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "id"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "key"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "secret"),
				),
			},
			{
				Config: testAccServiceConfig(name, "updated description", true, "https://tf-acc-service.example.com/callback2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accServiceResourceAddr, "description", "updated description"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "trusted", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "redirect_uris.0", "https://tf-acc-service.example.com/callback2"),
					// application_name is create-only; must survive the update unchanged.
					resource.TestCheckResourceAttr(accServiceResourceAddr, "application_name", "tf-acc-service-app"),
				),
			},
			{
				ResourceName:            accServiceResourceAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       importStateID(accServiceResourceAddr),
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}
