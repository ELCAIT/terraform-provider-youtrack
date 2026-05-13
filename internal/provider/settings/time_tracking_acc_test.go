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
	accTimeTrackingResource    = "youtrack_global_time_tracking_settings.test"
	accTimeTrackingTypeSetPath = "work_item_types.*"
	accTimeTrackingMinutesOne  = 450
	accTimeTrackingMinutesTwo  = 420
	accTimeTrackingTypeNameOne = "acc-test-development"
	accTimeTrackingTypeNameTwo = "acc-test-investigation"
)

func testAccTimeTrackingConfig(minutesADay int) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_global_time_tracking_settings" "test" {
	work_time_settings = {
    minutes_a_day = %d
    work_days     = [1, 2, 3, 4, 5]
	}
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		minutesADay,
	)
}

func testAccTimeTrackingWithWorkItemTypesConfig(minutesADay int, typeOneName, typeTwoName string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_global_time_tracking_settings" "test" {
  work_time_settings = {
    minutes_a_day = %d
    work_days     = [1, 2, 3, 4, 5]
  }
  work_item_types = [
    { name = "%s", auto_attached = true },
    { name = "%s", auto_attached = false },
  ]
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		minutesADay,
		typeOneName,
		typeTwoName,
	)
}

func testAccTimeTrackingWithSingleWorkItemTypeConfig(minutesADay int, typeName string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  base_url = "%s"
  token    = "%s"
}

resource "youtrack_global_time_tracking_settings" "test" {
  work_time_settings = {
    minutes_a_day = %d
    work_days     = [1, 2, 3, 4, 5]
  }
  work_item_types = [
    { name = "%s", auto_attached = true },
  ]
}
`,
		os.Getenv(envYouTrackURL),
		os.Getenv(envYouTrackToken),
		minutesADay,
		typeName,
	)
}

func TestAccGlobalTimeTrackingSettings(t *testing.T) {
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

	original, err := client.GetWorkTimeSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original work time settings: %v", err)
	}

	t.Cleanup(func() {
		if _, restoreErr := client.UpdateWorkTimeSettings(context.Background(), original); restoreErr != nil {
			t.Errorf("failed to restore original work time settings: %v", restoreErr)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckDatabaseBackupSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTimeTrackingConfig(accTimeTrackingMinutesOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accTimeTrackingResource, "work_time_settings.minutes_a_day", fmt.Sprintf("%d", accTimeTrackingMinutesOne)),
					resource.TestCheckResourceAttrSet(accTimeTrackingResource, "work_time_settings.id"),
					resource.TestCheckResourceAttrSet(accTimeTrackingResource, "last_updated"),
				),
			},
			{
				Config: testAccTimeTrackingConfig(accTimeTrackingMinutesTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accTimeTrackingResource, "work_time_settings.minutes_a_day", fmt.Sprintf("%d", accTimeTrackingMinutesTwo)),
				),
			},
			{
				ResourceName:            accTimeTrackingResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated", "work_item_types"},
			},
		},
	})
}

func registerWorkItemTypesCleanup(t *testing.T, client *youtrack.Client, originalWorkTime youtrack.WorkTimeSettings, originalTypes []youtrack.WorkItemType) {
	t.Helper()
	t.Cleanup(func() {
		if _, restoreErr := client.UpdateWorkTimeSettings(context.Background(), originalWorkTime); restoreErr != nil {
			t.Errorf("failed to restore original work time settings: %v", restoreErr)
		}
		currentTypes, listErr := client.ListWorkItemTypes(context.Background())
		if listErr != nil {
			t.Errorf("failed to list work item types during cleanup: %v", listErr)
			return
		}
		for _, ct := range currentTypes {
			if err := client.DeleteWorkItemType(context.Background(), ct.ID); err != nil {
				t.Errorf("failed to delete work item type %q during cleanup: %v", ct.Name, err)
			}
		}
		for _, wt := range originalTypes {
			if _, err := client.CreateWorkItemType(context.Background(), wt); err != nil {
				t.Errorf("failed to restore work item type %q during cleanup: %v", wt.Name, err)
			}
		}
	})
}

func deleteWorkItemTypesBefore(t *testing.T, client *youtrack.Client, types []youtrack.WorkItemType) {
	t.Helper()
	// Delete all pre-existing work item types so the test starts with a clean slate.
	// This prevents YouTrack's soft-delete ("X (being removed)") from polluting test state.
	for _, wt := range types {
		if err := client.DeleteWorkItemType(context.Background(), wt.ID); err != nil {
			t.Fatalf("failed to delete pre-existing work item type %q before test: %v", wt.Name, err)
		}
	}
}

func TestAccGlobalTimeTrackingSettingsWorkItemTypes(t *testing.T) {
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

	originalWorkTime, err := client.GetWorkTimeSettings(context.Background())
	if err != nil {
		t.Fatalf("failed to read original work time settings: %v", err)
	}

	originalTypes, err := client.ListWorkItemTypes(context.Background())
	if err != nil {
		t.Fatalf("failed to read original work item types: %v", err)
	}

	registerWorkItemTypesCleanup(t, client, originalWorkTime, originalTypes)
	deleteWorkItemTypesBefore(t, client, originalTypes)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccTimeTrackingWithWorkItemTypesConfig(accTimeTrackingMinutesOne, accTimeTrackingTypeNameOne, accTimeTrackingTypeNameTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accTimeTrackingResource, "work_item_types.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(accTimeTrackingResource, accTimeTrackingTypeSetPath, map[string]string{
						"name":          accTimeTrackingTypeNameOne,
						"auto_attached": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(accTimeTrackingResource, accTimeTrackingTypeSetPath, map[string]string{
						"name":          accTimeTrackingTypeNameTwo,
						"auto_attached": "false",
					}),
				),
			},
			{
				// Update: remove one type, keep the other
				Config: testAccTimeTrackingWithSingleWorkItemTypeConfig(accTimeTrackingMinutesOne, accTimeTrackingTypeNameOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accTimeTrackingResource, "work_item_types.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(accTimeTrackingResource, accTimeTrackingTypeSetPath, map[string]string{
						"name":          accTimeTrackingTypeNameOne,
						"auto_attached": "true",
					}),
				),
			},
			{
				ResourceName:            accTimeTrackingResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "global",
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
