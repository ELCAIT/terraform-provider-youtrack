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
	accAppearanceResource = "youtrack_appearance_settings.test"
	accDateFieldFormatID  = "youtrack.datefieldformat.iso8601"
	accTimeZoneID         = "Etc/UTC"
	accTimeZoneIDUpdated  = "Europe/Zurich"
)

func testAccAppearanceConfig(dateFormatID, timeZoneID string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_appearance_settings" "test" {
  date_format_id = "%s"
  time_zone_id   = "%s"
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		dateFormatID,
		timeZoneID,
	)
}

func TestAccAppearanceSettings(t *testing.T) {
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

	original, err := client.GetAppearanceSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original appearance settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateAppearanceSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original appearance settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppearanceConfig(accDateFieldFormatID, accTimeZoneID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppearanceResource, "date_format_id", accDateFieldFormatID),
					resource.TestCheckResourceAttr(accAppearanceResource, "time_zone_id", accTimeZoneID),
					resource.TestCheckResourceAttrSet(accAppearanceResource, "last_updated"),
				),
			},
			{
				Config: testAccAppearanceConfig(accDateFieldFormatID, accTimeZoneIDUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppearanceResource, "time_zone_id", accTimeZoneIDUpdated),
				),
			},
			{
				ResourceName:            accAppearanceResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
