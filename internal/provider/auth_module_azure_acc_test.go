package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccAzureConfig(name, clientID, clientSecret, tenant string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_auth_module_azure" "test" {
  name          = %q
  client_id     = %q
  client_secret = %q
  tenant        = %q
  is_default    = false
}
`, name, clientID, clientSecret, tenant)
}

const accAzureModuleAddr = "youtrack_auth_module_azure.test"

func TestAccAzureAuthModule(t *testing.T) {
	skipUnlessAcc(t)

	if os.Getenv(envAzureClientID) == "" || os.Getenv(envAzureClientSecret) == "" || os.Getenv(envAzureTenant) == "" {
		t.Skip("set YOUTRACK_AZURE_CLIENT_ID, YOUTRACK_AZURE_CLIENT_SECRET, YOUTRACK_AZURE_TENANT to run Azure acceptance tests")
	}

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("tf-acc-azure-%d", suffix)
	nameUpdated := fmt.Sprintf("tf-acc-azure-updated-%d", suffix)

	clientID := os.Getenv(envAzureClientID)
	clientSecret := os.Getenv(envAzureClientSecret)
	tenant := os.Getenv(envAzureTenant)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAzureConfig(name, clientID, clientSecret, tenant),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAzureModuleAddr, "name", name),
					resource.TestCheckResourceAttr(accAzureModuleAddr, "client_id", clientID),
					resource.TestCheckResourceAttr(accAzureModuleAddr, "tenant", tenant),
					resource.TestCheckResourceAttr(accAzureModuleAddr, "is_default", "false"),
					resource.TestCheckResourceAttrSet(accAzureModuleAddr, "id"),
					resource.TestCheckResourceAttrSet(accAzureModuleAddr, "server_url"),
				),
			},
			{
				Config: testAccAzureConfig(nameUpdated, clientID, clientSecret, tenant),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAzureModuleAddr, "name", nameUpdated),
				),
			},
			{
				ResourceName:            accAzureModuleAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}
