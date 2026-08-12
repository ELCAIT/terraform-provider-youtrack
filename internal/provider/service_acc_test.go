package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const accServiceResourceAddr = "youtrack_service.test"

func testAccServiceConfig(name, description string, trusted bool, redirectURI, flowsBlock string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_service" "test" {
  name              = %q
  application_name  = "tf-acc-service-app"
  home_url          = "https://tf-acc-service.example.com"
  description       = %q
  trusted           = %t
  redirect_uris     = [%q]
%s
}
`, name, description, trusted, redirectURI, flowsBlock)
}

func TestAccServiceResource(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("tf-acc-service-%d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfig(name, "initial description", false, "https://tf-acc-service.example.com/callback", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accServiceResourceAddr, "name", name),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "application_name", "tf-acc-service-app"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "description", "initial description"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "trusted", "false"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "consent_required", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "redirect_uris.0", "https://tf-acc-service.example.com/callback"),
					// Hub's default flow flags when a service is created without specifying them.
					resource.TestCheckResourceAttr(accServiceResourceAddr, "client_credentials_flow_enabled", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "auth_code_flow_enabled", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "pkce_required", "false"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "implicit_flow_enabled", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "resource_owner_flow_enabled", "true"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "id"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "key"),
					resource.TestCheckResourceAttrSet(accServiceResourceAddr, "secret"),
				),
			},
			{
				Config: testAccServiceConfig(name, "updated description", true, "https://tf-acc-service.example.com/callback2", `
  client_credentials_flow_enabled = false
  auth_code_flow_enabled          = true
  pkce_required                   = true
  implicit_flow_enabled           = false
  resource_owner_flow_enabled     = false
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accServiceResourceAddr, "description", "updated description"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "trusted", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "redirect_uris.0", "https://tf-acc-service.example.com/callback2"),
					// application_name is create-only; must survive the update unchanged.
					resource.TestCheckResourceAttr(accServiceResourceAddr, "application_name", "tf-acc-service-app"),
					// restrict the service to PKCE-protected authorization code flow only.
					resource.TestCheckResourceAttr(accServiceResourceAddr, "client_credentials_flow_enabled", "false"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "auth_code_flow_enabled", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "pkce_required", "true"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "implicit_flow_enabled", "false"),
					resource.TestCheckResourceAttr(accServiceResourceAddr, "resource_owner_flow_enabled", "false"),
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
