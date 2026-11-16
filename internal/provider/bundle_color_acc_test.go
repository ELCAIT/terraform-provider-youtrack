package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accColorFirstValue  = "values.0.color_id"
	accColorSecondValue = "values.1.color_id"
	accColorDefault     = "0"
	accColorSet         = "20"
	accColorChanged     = "21"
)

// testAccColoredBundleConfig is a bundle of the given kind with two values. firstColor is the color_id of the
// first one, left out when empty; the second value never sets it.
func testAccColoredBundleConfig(kind, name, firstColor string) string {
	color := ""
	if firstColor != "" {
		color = fmt.Sprintf("color_id = %q", firstColor)
	}

	return providerBlock() + fmt.Sprintf(`
resource "youtrack_%[1]s_bundle" "test" {
  name = %[2]q
  values = [
    { name = "First", %[3]s },
    { name = "Second" },
  ]
}
`, kind, name, color)
}

// testAccColoredBundleSteps covers the color of the values of a bundle: set on create, changed, left unset
// (which keeps the color), set back to the default style, and imported with its colors verified.
func testAccColoredBundleSteps(kind, name string) []resource.TestStep {
	address := fmt.Sprintf("youtrack_%s_bundle.test", kind)

	return []resource.TestStep{
		{
			Config: testAccColoredBundleConfig(kind, name, accColorSet),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(address, accColorFirstValue, accColorSet),
				resource.TestCheckResourceAttr(address, accColorSecondValue, accColorDefault),
			),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, accColorChanged),
			Check:  resource.TestCheckResourceAttr(address, accColorFirstValue, accColorChanged),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, ""),
			Check:  resource.TestCheckResourceAttr(address, accColorFirstValue, accColorChanged),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, accColorDefault),
			Check:  resource.TestCheckResourceAttr(address, accColorFirstValue, accColorDefault),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, accColorChanged),
			Check:  resource.TestCheckResourceAttr(address, accColorFirstValue, accColorChanged),
		},
		{
			ResourceName:      address,
			ImportState:       true,
			ImportStateIdFunc: importStateID(address),
			ImportStateVerify: true,
		},
	}
}

func TestAccEnumBundleColor(t *testing.T) {
	skipUnlessAcc(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps:                    testAccColoredBundleSteps("enum", fmt.Sprintf("TF Acc Enum Color %d", time.Now().UnixMilli())),
	})
}

func TestAccStateBundleColor(t *testing.T) {
	skipUnlessAcc(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps:                    testAccColoredBundleSteps("state", fmt.Sprintf("TF Acc State Color %d", time.Now().UnixMilli())),
	})
}
