// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const accAppGlobalConfigResource = "youtrack_app_global_config.test"

func testAccAppGlobalConfigConfig(appName string, enabled bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_app_global_config" "test" {
  app_name = %q
  enabled  = %t
}
`, appName, enabled)
}

// testAccDisableApp disables the app outside of Terraform, so the next step
// proves the drift is detected and the app enabled again.
func testAccDisableApp(t *testing.T) func(*terraform.State) error {
	t.Helper()

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[accAppGlobalConfigResource]
		if !ok {
			return fmt.Errorf("resource %q not found", accAppGlobalConfigResource)
		}

		client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
		if err != nil {
			return err
		}

		disabled := false
		_, err = client.SetAppGlobalConfig(context.Background(), rs.Primary.Attributes["id"], youtrack.AppGlobalConfigUpdate{Enabled: &disabled})
		return err
	}
}

// TestAccAppGlobalConfig leaves the app enabled: destroying the resource keeps
// the configuration it wrote. It does not set settings, which depend on the app.
func TestAccAppGlobalConfig(t *testing.T) {
	skipUnlessAcc(t)

	appName := accProjectAppName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accAppGlobalConfigResource, "id"),
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "app_name", appName),
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
				),
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, false),
				Check:  resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "false"),
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
					testAccDisableApp(t),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAppGlobalConfigConfig(appName, true),
				Check:  resource.TestCheckResourceAttr(accAppGlobalConfigResource, "enabled", "true"),
			},
			{
				ResourceName: accAppGlobalConfigResource,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[accAppGlobalConfigResource].Primary.Attributes["app_name"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "app_name",
			},
		},
	})
}

func testAccAppGlobalConfigGroupsConfig(appName, groupName string, withGroup bool) string {
	groups := "[]"
	if withGroup {
		groups = "[data.youtrack_group.test.id]"
	}

	return providerBlock() + fmt.Sprintf(`
data "youtrack_group" "test" {
  name = %q
}

resource "youtrack_app_global_config" "test" {
  app_name            = %q
  enabled             = true
  permitted_group_ids = %s
}
`, groupName, appName, groups)
}

// testAccRestoreVisibility puts back who may use the app once the test is
// done: destroying the resource leaves the visibility it wrote.
func testAccRestoreVisibility(t *testing.T, client *youtrack.Client, appID string) {
	t.Helper()

	before, err := client.GetAppGlobalConfig(context.Background(), appID)
	if err != nil {
		t.Fatalf("could not read the app configuration: %v", err)
	}

	var groups, users []string
	for _, g := range before.VisibilitySettings.PermittedGroups {
		groups = append(groups, g.ID)
	}
	for _, u := range before.VisibilitySettings.PermittedUsers {
		users = append(users, u.ID)
	}

	t.Cleanup(func() {
		if _, err := client.SetAppVisibility(context.Background(), appID, groups, users); err != nil {
			t.Errorf("could not restore the app visibility: %v", err)
		}
	})
}

// testAccPermitUser allows the user to use the app outside of Terraform,
// keeping the groups Terraform wrote.
func testAccPermitUser(client *youtrack.Client, appID, userID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		groupID := s.RootModule().Resources[accAppGlobalConfigResource].Primary.Attributes["permitted_group_ids.0"]
		_, err := client.SetAppVisibility(context.Background(), appID, []string{groupID}, []string{userID})
		return err
	}
}

func testAccCheckUserPermitted(client *youtrack.Client, appID, userID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		config, err := client.GetAppGlobalConfig(context.Background(), appID)
		if err != nil {
			return err
		}
		for _, u := range config.VisibilitySettings.PermittedUsers {
			if u.ID == userID {
				return nil
			}
		}
		return fmt.Errorf("user %s is no longer permitted: %+v", userID, config.VisibilitySettings)
	}
}

// TestAccAppGlobalConfigKeepsPermittedUsers proves that writing the permitted
// groups writes back the permitted users, which Terraform does not manage.
func TestAccAppGlobalConfigKeepsPermittedUsers(t *testing.T) {
	skipUnlessAcc(t)

	groupName, login := os.Getenv(envAccGroupName), os.Getenv(envAccUser)
	if groupName == "" || login == "" {
		t.Skip("set YOUTRACK_TEST_GROUP_NAME and YOUTRACK_TEST_USER to run this test")
	}

	client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
	if err != nil {
		t.Fatal(err)
	}
	appName := accProjectAppName()
	app, err := client.GetAppByName(context.Background(), appName)
	if err != nil {
		t.Fatalf("could not find app %q: %v", appName, err)
	}
	user, err := client.GetUserByLogin(context.Background(), login)
	if err != nil {
		t.Fatalf("could not find user %q: %v", login, err)
	}
	testAccRestoreVisibility(t, client, app.ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAppGlobalConfigGroupsConfig(appName, groupName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "permitted_group_ids.#", "1"),
					testAccPermitUser(client, app.ID, user.Id),
				),
			},
			{
				Config: testAccAppGlobalConfigGroupsConfig(appName, groupName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accAppGlobalConfigResource, "permitted_group_ids.#", "0"),
					testAccCheckUserPermitted(client, app.ID, user.Id),
				),
			},
		},
	})
}
