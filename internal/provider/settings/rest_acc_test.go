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
	accRestSettingsResource = "youtrack_rest_settings.test"
	accRestServerURL        = "http://localhost:3000"
)

func testAccRestSettingsConfig(allowAllOrigins bool, origins ...string) string {
	allowAll := "false"
	if allowAllOrigins {
		allowAll = "true"
	}

	originsStr := ""
	for _, o := range origins {
		originsStr += fmt.Sprintf("%q, ", o)
	}
	if len(originsStr) > 2 {
		originsStr = originsStr[:len(originsStr)-2]
	}

	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_rest_settings" "test" {
  allow_all_origins = %s
  allowed_origins   = [%s]
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		allowAll,
		originsStr,
	)
}

func TestAccRestSettings(t *testing.T) {
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

	original, err := client.GetRestSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original REST settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateRestSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original REST settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRestSettingsConfig(false, accRestServerURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accRestSettingsResource, "allow_all_origins", "false"),
					resource.TestCheckResourceAttr(accRestSettingsResource, "allowed_origins.#", "1"),
					resource.TestCheckResourceAttr(accRestSettingsResource, "allowed_origins.0", accRestServerURL),
					resource.TestCheckResourceAttrSet(accRestSettingsResource, "last_updated"),
				),
			},
			{
				Config: testAccRestSettingsConfig(false, accRestServerURL, "http://localhost:8080"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accRestSettingsResource, "allowed_origins.#", "2"),
				),
			},
			{
				ResourceName:            accRestSettingsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
