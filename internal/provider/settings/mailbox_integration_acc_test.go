package settings_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	youtrack "github.com/elcait/youtrack-api-client/client"
)

const accMailboxIntegrationSettingsResource = "youtrack_mailbox_integration_settings.test"

func testAccMailboxIntegrationSettingsConfig(cronExpression string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = %q
  token    = %q
}

resource "youtrack_mailbox_integration_settings" "test" {
  cron_expression = %q
}
`, os.Getenv(envYouTrackURL), os.Getenv(envYouTrackToken), cronExpression)
}

func TestAccMailboxIntegrationSettings(t *testing.T) {
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

	original, err := client.GetMailboxCronSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original mailbox integration settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateMailboxCronSettings(context.Background(), original.CronExpression); restoreErr != nil {
			t.Errorf("failed to restore original mailbox integration settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccMailboxIntegrationSettingsConfig("0 0/5 * * * ?"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accMailboxIntegrationSettingsResource, "id", importStateID),
					resource.TestCheckResourceAttr(accMailboxIntegrationSettingsResource, "cron_expression", "0 0/5 * * * ?"),
				),
			},
			{
				Config: testAccMailboxIntegrationSettingsConfig("0 0/10 * * * ?"),
				Check:  resource.TestCheckResourceAttr(accMailboxIntegrationSettingsResource, "cron_expression", "0 0/10 * * * ?"),
			},
			{
				ResourceName:      accMailboxIntegrationSettingsResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     importStateID,
			},
		},
	})
}
