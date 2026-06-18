package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accEnumBundleResource = "youtrack_enum_bundle.test"
)

func testAccEnumBundleConfig(name, v1, v2 string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_enum_bundle" "test" {
  name = %q
	values = [
		{ name = %q },
		{ name = %q },
	]
}
`, name, v1, v2)
}

func TestAccEnumBundle(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TF Acc Enum Bundle %d", suffix)
	nameUpdated := fmt.Sprintf("TF Acc Enum Bundle Updated %d", suffix)
	value1 := fmt.Sprintf("Enum A %d", suffix)
	value2 := fmt.Sprintf("Enum B %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccEnumBundleConfig(name, value1, value2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accEnumBundleResource, "name", name),
					resource.TestCheckResourceAttr(accEnumBundleResource, "values.#", "2"),
					resource.TestCheckResourceAttrSet(accEnumBundleResource, "id"),
				),
			},
			{
				Config: testAccEnumBundleConfig(nameUpdated, value1, value2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accEnumBundleResource, "name", nameUpdated),
					resource.TestCheckResourceAttr(accEnumBundleResource, "values.#", "2"),
				),
			},
			{
				ResourceName:            accEnumBundleResource,
				ImportState:             true,
				ImportStateIdFunc:       importStateID(accEnumBundleResource),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values"},
			},
		},
	})
}
