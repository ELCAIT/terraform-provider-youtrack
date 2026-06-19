// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	accProjectTimeTrackingResource = "youtrack_project_time_tracking_settings.test"
)

func testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin string, enabled bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_project" "parent" {
  name         = %q
  short_name   = %q
  leader_login = %q
}

resource "youtrack_project_time_tracking_settings" "test" {
  project_id = youtrack_project.parent.id
  enabled    = %t
}
`, projectName, shortName, leaderLogin, enabled)
}

func TestAccProjectTimeTrackingSettings(t *testing.T) {
	skipUnlessAcc(t)

	client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
	if err != nil {
		t.Fatalf("failed to create YouTrack client: %v", err)
	}

	originalTypes, err := client.ListWorkItemTypes(context.Background())
	if err != nil {
		t.Fatalf("failed to read original work item types: %v", err)
	}

	registerProjectTimeTrackingWorkItemTypesCleanup(t, client, originalTypes)
	deleteProjectTimeTrackingWorkItemTypesBefore(t, client, originalTypes)

	leaderLogin := testProjectLeaderLogin(t)
	suffix := time.Now().UnixMilli()
	projectName := fmt.Sprintf("TFAccPTT%d", suffix)
	shortName := fmt.Sprintf("PTT%d", suffix%10000)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectTimeTrackingResource, "id"),
					resource.TestCheckResourceAttr(accProjectTimeTrackingResource, "enabled", "true"),
					resource.TestCheckResourceAttrSet(accProjectTimeTrackingResource, "project_id"),
				),
			},
			{
				Config: testAccProjectTimeTrackingConfig(projectName, shortName, leaderLogin, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectTimeTrackingResource, "enabled", "false"),
				),
			},
			{
				ResourceName:      accProjectTimeTrackingResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[accProjectTimeTrackingResource]
					if !ok {
						return "", fmt.Errorf("resource %q not found", accProjectTimeTrackingResource)
					}
					return rs.Primary.Attributes["project_id"], nil
				},
			},
		},
	})
}

func registerProjectTimeTrackingWorkItemTypesCleanup(t *testing.T, client *youtrack.Client, originalTypes []youtrack.WorkItemType) {
	t.Helper()
	t.Cleanup(func() {
		currentTypes, listErr := client.ListWorkItemTypes(context.Background())
		if listErr != nil {
			t.Errorf("failed to list work item types during cleanup: %v", listErr)
			return
		}

		for _, currentType := range currentTypes {
			if err := client.DeleteWorkItemType(context.Background(), currentType.ID); err != nil {
				t.Errorf("failed to delete work item type %q during cleanup: %v", currentType.Name, err)
			}
		}

		for _, originalType := range originalTypes {
			if _, err := client.CreateWorkItemType(context.Background(), originalType); err != nil {
				t.Errorf("failed to restore work item type %q during cleanup: %v", originalType.Name, err)
			}
		}
	})
}

func deleteProjectTimeTrackingWorkItemTypesBefore(t *testing.T, client *youtrack.Client, types []youtrack.WorkItemType) {
	t.Helper()
	for _, workItemType := range types {
		if err := client.DeleteWorkItemType(context.Background(), workItemType.ID); err != nil {
			t.Fatalf("failed to delete pre-existing work item type %q before test: %v", workItemType.Name, err)
		}
	}
}
