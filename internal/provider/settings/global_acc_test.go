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
	accGlobalSettingsResource = "youtrack_global_settings.test"
)

func testAccGlobalSettingsConfig() string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_global_settings" "test" {}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
	)
}

func TestAccGlobalSettings(t *testing.T) {
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

	original, err := client.GetGlobalSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original global settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateGlobalSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original global settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalSettingsConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accGlobalSettingsResource, "id"),
					resource.TestCheckResourceAttrSet(accGlobalSettingsResource, "last_updated"),
				),
			},
			{
				ResourceName:      accGlobalSettingsResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "global",
				// license is sensitive and not returned by the API, skip it
				ImportStateVerifyIgnore: []string{"license", "last_updated"},
			},
		},
	})
}
