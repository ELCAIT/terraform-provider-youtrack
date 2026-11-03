// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const accAppGlobalConfigResource = "youtrack_app_global_config.test"

func testAccAppGlobalConfigConfig(appName string, enabled bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_app_global_config" "test" {
  app_name = %q
  enabled  = %t
}
`, appName, enabled)
}

// testAccDisableApp disables the app outside of Terraform, so the next step
// proves the drift is detected and the app enabled again.
func testAccDisableApp(t *testing.T) func(*terraform.State) error {
	t.Helper()

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[accAppGlobalConfigResource]
		if !ok {
			return fmt.Errorf("resource %q not found", accAppGlobalConfigResource)
		}

		client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
		if err != nil {
			return err
		}

		disabled := false
		_, err = client.SetAppGlobalConfig(context.Background(), rs.Primary.Attributes["app_id"], youtrack.AppGlobalConfigUpdate{Enabled: &disabled})
		return err
	}
}

// TestAccAppGlobalConfig leaves the app enabled: destroying the resource keeps
// the configuration it wrote. It does not set settings, which depend on the app.
func TestAccAppGlobalConfig(t *testing.T) {
	skipUnlessAcc(t)

	appName := accProjectAppName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accAppGlobalConfigResource, "id"),
					resource.TestCheckResourceAttrSet(accAppGlobalConfigResource, "app_id"),
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "app_name", appName),
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
				),
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, false),
				Check:  resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "false"),
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
					testAccDisableApp(t),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check:  resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
			},
			{
				ResourceName: accAppGlobalConfigResource,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[accAppGlobalConfigResource].Primary.Attributes["app_name"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "app_name",
			},
		},
	})
}
