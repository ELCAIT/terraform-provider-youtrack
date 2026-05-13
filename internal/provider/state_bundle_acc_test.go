package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accStateBundleResource = "youtrack_state_bundle.test"
)

func testAccStateBundleConfig(name, openName, doneName string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_state_bundle" "test" {
  name = %q
	values = [
		{ name = %q, is_resolved = false },
		{ name = %q, is_resolved = true },
	]
}
`, name, openName, doneName)
}

func TestAccStateBundle(t *testing.T) {
	skipUnlessAcc(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TF Acc State Bundle %d", suffix)
	nameUpdated := fmt.Sprintf("TF Acc State Bundle Updated %d", suffix)
	openName := fmt.Sprintf("Open %d", suffix)
	doneName := fmt.Sprintf("Done %d", suffix)
	openNameUpdated := fmt.Sprintf("Open Updated %d", suffix)
	doneNameUpdated := fmt.Sprintf("Done Updated %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccStateBundleConfig(name, openName, doneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accStateBundleResource, "name", name),
					resource.TestCheckResourceAttr(accStateBundleResource, "values.#", "2"),
					resource.TestCheckResourceAttrSet(accStateBundleResource, "id"),
				),
			},
			{
				Config: testAccStateBundleConfig(nameUpdated, openNameUpdated, doneNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accStateBundleResource, "name", nameUpdated),
					resource.TestCheckResourceAttr(accStateBundleResource, "values.#", "2"),
				),
			},
			{
				ResourceName:            accStateBundleResource,
				ImportState:             true,
				ImportStateIdFunc:       importStateID(accStateBundleResource),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values"},
			},
		},
	})
}
