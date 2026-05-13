package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accIssueLinkTypeResource = "youtrack_issue_link_type.test"
)

func testAccIssueLinkTypeConfig(name, sourceToTarget, targetToSource string, directed bool, aggregation bool) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_issue_link_type" "test" {
  name             = %q
  source_to_target = %q
  target_to_source = %q
  directed         = %t
  aggregation      = %t
}
`, name, sourceToTarget, targetToSource, directed, aggregation)
}

func TestAccIssueLinkType(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TF Acc Link Type %d", suffix)
	nameUpdated := fmt.Sprintf("TF Acc Link Type Updated %d", suffix)
	sourceToTarget := fmt.Sprintf("tf acc outward %d", suffix)
	targetToSource := fmt.Sprintf("tf acc inward %d", suffix)
	sourceToTargetUpdated := fmt.Sprintf("tf acc outward updated %d", suffix)
	targetToSourceUpdated := fmt.Sprintf("tf acc inward updated %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccIssueLinkTypeConfig(name, sourceToTarget, targetToSource, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "name", name),
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "source_to_target", sourceToTarget),
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "target_to_source", targetToSource),
					resource.TestCheckResourceAttrSet(accIssueLinkTypeResource, "id"),
				),
			},
			{
				Config: testAccIssueLinkTypeConfig(nameUpdated, sourceToTargetUpdated, targetToSourceUpdated, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "name", nameUpdated),
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "source_to_target", sourceToTargetUpdated),
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "target_to_source", targetToSourceUpdated),
					resource.TestCheckResourceAttr(accIssueLinkTypeResource, "directed", "true"),
				),
			},
			{
				ResourceName:      accIssueLinkTypeResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
