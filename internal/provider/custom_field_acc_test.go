package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accCustomFieldResource = "youtrack_custom_field.test"
)

func testAccCustomFieldConfig(fieldName, fieldTypeID string, autoAttached, displayed bool, alias string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_custom_field" "test" {
  name                       = %q
  field_type_id              = %q
  is_auto_attached           = %t
  is_displayed_in_issue_list = %t
  aliases                    = %q
	field_defaults = {
		can_be_empty     = true
		empty_field_text = "Not set"
		is_public        = true
	}
}
`, fieldName, fieldTypeID, autoAttached, displayed, alias)
}

func TestAccCustomField(t *testing.T) {
	skipUnlessAcc(t)

	fieldTypeID := os.Getenv(envAccCustomFieldTypeID)
	if fieldTypeID == "" {
		fieldTypeID = "enum[1]"
	}

	suffix := time.Now().UnixMilli()
	fieldName := fmt.Sprintf("TF Acc Field %d", suffix)
	fieldNameUpdated := fmt.Sprintf("TF Acc Field Updated %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccCustomFieldConfig(fieldName, fieldTypeID, false, true, "cf_alias_one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accCustomFieldResource, "name", fieldName),
					resource.TestCheckResourceAttr(accCustomFieldResource, "field_type_id", fieldTypeID),
					resource.TestCheckResourceAttr(accCustomFieldResource, "is_auto_attached", "false"),
					resource.TestCheckResourceAttr(accCustomFieldResource, "is_displayed_in_issue_list", "true"),
					resource.TestCheckResourceAttrSet(accCustomFieldResource, "id"),
				),
			},
			{
				Config: testAccCustomFieldConfig(fieldNameUpdated, fieldTypeID, true, false, "cf_alias_two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accCustomFieldResource, "name", fieldNameUpdated),
					resource.TestCheckResourceAttr(accCustomFieldResource, "is_auto_attached", "true"),
					resource.TestCheckResourceAttr(accCustomFieldResource, "is_displayed_in_issue_list", "false"),
					resource.TestCheckResourceAttr(accCustomFieldResource, "aliases", "cf_alias_two"),
				),
			},
			{
				ResourceName:            accCustomFieldResource,
				ImportState:             true,
				ImportStateIdFunc:       importStateID(accCustomFieldResource),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"has_running_job", "field_defaults.bundle_name"},
			},
		},
	})
}
