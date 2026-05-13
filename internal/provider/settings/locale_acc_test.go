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
	accLocaleSettingsResource = "youtrack_locale_settings.test"
)

func testAccLocaleConfig(localeID, locale, language string, community bool, name string) string {
	communityStr := "false"
	if community {
		communityStr = "true"
	}

	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_locale_settings" "test" {
  id        = "%s"
  locale    = "%s"
  language  = "%s"
  community = %s
  name      = "%s"
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		localeID,
		locale,
		language,
		communityStr,
		name,
	)
}

func TestAccLocaleSettings(t *testing.T) {
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

	original, err := client.GetLocaleSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original locale settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateLocaleSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original locale settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLocaleConfig("en_US", "en_US", "en", false, "English"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accLocaleSettingsResource, "id", "en_US"),
					resource.TestCheckResourceAttr(accLocaleSettingsResource, "locale", "en_US"),
					resource.TestCheckResourceAttr(accLocaleSettingsResource, "language", "en"),
					resource.TestCheckResourceAttr(accLocaleSettingsResource, "community", "false"),
					resource.TestCheckResourceAttrSet(accLocaleSettingsResource, "last_updated"),
				),
			},
			{
				ResourceName:            accLocaleSettingsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
