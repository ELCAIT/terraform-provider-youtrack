package settings_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	ytprovider "github.com/elcait/youtrack-provider/internal/provider"
	ytsettings "github.com/elcait/youtrack-provider/internal/provider/settings"

	youtrack "github.com/elcait/youtrack-api-client/client"
)

const (
	accTestEnabledValue = "1"
	providerName        = "youtrack"
	resourceAddress     = "youtrack_database_backup_settings.test"
	importStateID       = "global"
	envYouTrackURL      = "YOUTRACK_URL"
	envYouTrackToken    = "YOUTRACK_TOKEN"
	envNotifyLogin      = "YOUTRACK_NOTIFY_LOGIN"
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		providerName: providerserver.NewProtocol6WithError(ytprovider.New("test")()),
	}
}

func testAccCheckDatabaseBackupSettingsDestroy(_ *terraform.State) error {
	// Singleton settings resource has no dedicated remote delete semantics.
	return nil
}

func testAccDatabaseBackupSettingsConfig(location string, filesToKeep int, cronExpression, archiveFormat string, enabled bool, login string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_database_backup_settings" "test" {
  location        = "%s"
  files_to_keep   = %d
  cron_expression = "%s"
  archive_format  = "%s"
  enabled         = %t
  notified_users  = ["%s"]
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		location,
		filesToKeep,
		cronExpression,
		archiveFormat,
		enabled,
		login,
	)
}

func TestAccDatabaseBackupSettings(t *testing.T) {
	if os.Getenv("TF_ACC") != accTestEnabledValue {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}
	if os.Getenv(envYouTrackURL) == "" || os.Getenv(envYouTrackToken) == "" {
		t.Skip("set YOUTRACK_URL and YOUTRACK_TOKEN to run acceptance tests")
	}
	if os.Getenv(envNotifyLogin) == "" {
		t.Skip("set YOUTRACK_NOTIFY_LOGIN to an existing YouTrack user login")
	}

	notifyLogin := os.Getenv(envNotifyLogin)

	// Read the current settings before the test mutates them so we can restore them afterwards.
	client, err := youtrack.NewClient(os.Getenv(envYouTrackURL), os.Getenv(envYouTrackToken))
	if err != nil {
		t.Fatalf("failed to create YouTrack client for backup: %v", err)
	}

	original, err := client.GetBackupSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original backup settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateBackupSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original backup settings after test: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseBackupSettingsConfig(
					ytsettings.TestBackupPath,
					ytsettings.FilesToKeep,
					ytsettings.TestCronExpressionOne,
					ytsettings.TestBackupFormatZIP,
					true,
					notifyLogin,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceAddress, "location", ytsettings.TestBackupPath),
					resource.TestCheckResourceAttr(resourceAddress, "files_to_keep", fmt.Sprintf("%d", ytsettings.FilesToKeep)),
					resource.TestCheckResourceAttr(resourceAddress, "archive_format", ytsettings.TestBackupFormatZIP),
					resource.TestCheckResourceAttr(resourceAddress, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceAddress, "notified_users.0", notifyLogin),
				),
			},
			{
				Config: testAccDatabaseBackupSettingsConfig(
					ytsettings.TestBackupPath,
					5,
					"0 15 3 * * ?",
					ytsettings.TestBackupFormatZIP,
					true,
					notifyLogin,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceAddress, "location", ytsettings.TestBackupPath),
					resource.TestCheckResourceAttr(resourceAddress, "files_to_keep", "5"),
				),
			},
			{
				ResourceName:      resourceAddress,
				ImportState:       true,
				ImportStateId:     importStateID,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"last_updated",
				},
			},
		},
	})
}
