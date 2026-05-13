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
	accNotificationSettingsResource = "youtrack_notification_settings.test"
)

func testAccMailConfig(enabled bool, protocol, host string, port int, anonymous bool, from string) string {
	enabledStr := "false"
	if enabled {
		enabledStr = "true"
	}

	anonymousStr := "false"
	if anonymous {
		anonymousStr = "true"
	}

	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_notification_settings" "test" {
  is_enabled    = %s
  mail_protocol = "%s"
  host          = "%s"
  port          = %d
  anonymous     = %s
  from          = "%s"
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		enabledStr,
		protocol,
		host,
		port,
		anonymousStr,
		from,
	)
}

func TestAccNotificationSettings(t *testing.T) {
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

	original, err := client.GetMailServer(context.Background())
	if err != nil {
		t.Fatalf("failed to read original mail server settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateMailServer(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original mail server settings: %v", restoreErr)
		}
	})

	// Use the server's existing values to avoid inconsistent-result errors from API normalization.
	// We only toggle is_enabled to verify the resource can perform an update.
	initialEnabled := original.IsEnabled
	updatedEnabled := !initialEnabled

	protocol := original.MailProtocol
	if protocol == "" {
		protocol = "SMTP"
	}
	host := original.Host
	if host == "" {
		host = "localhost"
	}
	port := original.Port
	if port == 0 {
		port = 25
	}
	from := original.From
	if from == "" {
		from = "noreply@example.com"
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMailConfig(initialEnabled, protocol, host, port, original.Anonymous, from),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accNotificationSettingsResource, "host", host),
					resource.TestCheckResourceAttr(accNotificationSettingsResource, "port", fmt.Sprintf("%d", port)),
					resource.TestCheckResourceAttrSet(accNotificationSettingsResource, "last_updated"),
				),
			},
			{
				Config: testAccMailConfig(updatedEnabled, protocol, host, port, original.Anonymous, from),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accNotificationSettingsResource, "is_enabled", fmt.Sprintf("%t", updatedEnabled)),
				),
			},
			{
				ResourceName:            accNotificationSettingsResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
