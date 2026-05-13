package settings_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	youtrack "github.com/elcait/youtrack-api-client/client"
)

const (
	accSystemSettingsResource   = "youtrack_system_settings.test"
	accSystemSettingsAdminEmail = "admin@example.com"
)

func testAccSystemSettingsConfig(adminEmail string, maxExport int64) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_system_settings" "test" {
  administrator_email = "%s"
  max_export_items    = %d
  base_url            = "%s"
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		adminEmail,
		maxExport,
		os.Getenv(envYouTrackURL),
	)
}

func TestAccSystemSettings(t *testing.T) {
	if os.Getenv("TF_ACC") != accTestEnabledValue {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}
	if os.Getenv(envYouTrackURL) == "" || os.Getenv(envYouTrackToken) == "" {
		t.Skip("set YOUTRACK_URL and YOUTRACK_TOKEN to run acceptance tests")
	}

	client, err := youtrack.NewClient(os.Getenv(envYouTrackURL), os.Getenv(envYouTrackToken))
	if err != nil {
		t.Fatalf("failed to create YouTrack client: %v", err)
	}

	original, err := client.GetSystemSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original system settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateSystemSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original system settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemSettingsConfig(accSystemSettingsAdminEmail, 500),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accSystemSettingsResource, "administrator_email", accSystemSettingsAdminEmail),
					resource.TestCheckResourceAttr(accSystemSettingsResource, "max_export_items", "500"),
					resource.TestCheckResourceAttrSet(accSystemSettingsResource, "last_updated"),
				),
			},
			{
				Config: testAccSystemSettingsConfig(accSystemSettingsAdminEmail, 1000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accSystemSettingsResource, "max_export_items", "1000"),
				),
			},
			{
				ResourceName:            accSystemSettingsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
