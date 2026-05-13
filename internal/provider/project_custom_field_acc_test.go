// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	accProjectCustomFieldResource   = "youtrack_project_custom_field.test"
	envAccCustomFieldNameForProject = "YOUTRACK_TEST_PROJECT_CUSTOM_FIELD_NAME"
	envAccCustomFieldTypeForProject = "YOUTRACK_TEST_PROJECT_CUSTOM_FIELD_TYPE"
)

func testCustomFieldNameForProject(t *testing.T) string {
	t.Helper()

	name := os.Getenv(envAccCustomFieldNameForProject)
	if name == "" {
		name = "TF Acc Project Field"
	}

	return name
}

func testCustomFieldTypeForProject(t *testing.T) string {
	t.Helper()

	fieldType := os.Getenv(envAccCustomFieldTypeForProject)
	if fieldType == "" {
		fieldType = "EnumProjectCustomField"
	}

	return fieldType
}

func testCustomFieldTypeIDForProject() string {
	fieldTypeID := os.Getenv(envAccCustomFieldTypeID)
	if fieldTypeID == "" {
		fieldTypeID = "enum[1]"
	}

	return fieldTypeID
}

func testAccProjectCustomFieldConfig(projectName, shortName, leaderLogin, fieldName, fieldType, fieldTypeID string, canBeEmpty, isPublic bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_custom_field" "global" {
  name                       = %q
  field_type_id              = %q
  is_auto_attached           = false
  is_displayed_in_issue_list = true
  aliases                    = "pcf_alias"
  field_defaults = {
    can_be_empty     = true
    empty_field_text = "Not set"
    is_public        = true
  }
}

resource "youtrack_project" "parent" {
  name         = %q
  short_name   = %q
  leader_login = %q
}

resource "youtrack_project_custom_field" "test" {
  project_id   = youtrack_project.parent.id
  field_name   = youtrack_custom_field.global.name
  field_type   = %q
  can_be_empty = %t
  is_public    = %t
}
`, fieldName, fieldTypeID, projectName, shortName, leaderLogin, fieldType, canBeEmpty, isPublic)
}

func TestAccProjectCustomField(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	fieldName := fmt.Sprintf("%s %d", testCustomFieldNameForProject(t), time.Now().UnixMilli())
	fieldType := testCustomFieldTypeForProject(t)
	fieldTypeID := testCustomFieldTypeIDForProject()

	suffix := time.Now().UnixMilli()
	projectName := fmt.Sprintf("TFAccPCF%d", suffix)
	shortName := fmt.Sprintf("PCF%d", suffix%10000)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectCustomFieldConfig(projectName, shortName, leaderLogin, fieldName, fieldType, fieldTypeID, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accProjectCustomFieldResource, "id"),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "field_name", fieldName),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "can_be_empty", "true"),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "is_public", "true"),
				),
			},
			{
				Config: testAccProjectCustomFieldConfig(projectName, shortName, leaderLogin, fieldName, fieldType, fieldTypeID, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "can_be_empty", "false"),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "is_public", "false"),
				),
			},
			{
				ResourceName:      accProjectCustomFieldResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[accProjectCustomFieldResource]
					if !ok {
						return "", fmt.Errorf("resource %q not found", accProjectCustomFieldResource)
					}
					projectID := rs.Primary.Attributes["project_id"]
					id := rs.Primary.Attributes["id"]
					return projectID + "/" + id, nil
				},
			},
		},
	})
}
