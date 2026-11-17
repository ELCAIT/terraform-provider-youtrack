package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	accColorFirstValue  = "values.0.color_id"
	accColorSecondValue = "values.1.color_id"
	accColorDefault     = "0"
	accColorSet         = "20"
	accColorChanged     = "21"
	accColorOutOfBand   = "5"
	accFirstValueName   = "First"
	accSecondValueName  = "Second"
)

// recolorFunc sets the color of a value of a bundle directly in YouTrack, as a change made outside of Terraform.
type recolorFunc func(ctx context.Context, client *youtrack.Client, bundleName, valueName, colorID string) error

func recolorEnumValue(ctx context.Context, client *youtrack.Client, bundleName, valueName, colorID string) error {
	bundle, err := client.GetEnumBundleByName(ctx, bundleName)
	if err != nil {
		return err
	}
	for _, value := range bundle.Values {
		if strings.EqualFold(value.Name, valueName) {
			update := enumValueAsUpdate(value)
			update.Color = &youtrack.FieldStyle{ID: colorID}
			_, err = client.ReplaceEnumBundleValue(ctx, bundle.ID, value.ID, update)
			return err
		}
	}
	return fmt.Errorf("value %q not found in enum bundle %q", valueName, bundleName)
}

func recolorStateValue(ctx context.Context, client *youtrack.Client, bundleName, valueName, colorID string) error {
	bundle, err := client.GetStateBundleByName(ctx, bundleName)
	if err != nil {
		return err
	}
	for _, value := range bundle.Values {
		if strings.EqualFold(value.Name, valueName) {
			update := stateValueAsUpdate(value)
			update.Color = &youtrack.FieldStyle{ID: colorID}
			_, err = client.ReplaceStateBundleValue(ctx, bundle.ID, value.ID, update)
			return err
		}
	}
	return fmt.Errorf("value %q not found in state bundle %q", valueName, bundleName)
}

// testAccRecolorOutOfBand returns a PreConfig that colors the value valueName in YouTrack, failing the test if
// it cannot.
func testAccRecolorOutOfBand(t *testing.T, recolor recolorFunc, bundleName, valueName, colorID string) func() {
	return func() {
		client, err := youtrack.NewClient(os.Getenv(envAccURL), os.Getenv(envAccToken))
		if err != nil {
			t.Fatalf("could not create the client: %v", err)
		}
		if err := recolor(context.Background(), client, bundleName, valueName, colorID); err != nil {
			t.Fatalf("could not recolor %q outside of Terraform: %v", valueName, err)
		}
	}
}

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
    { name = %[4]q, %[3]s },
    { name = %[5]q },
  ]
}
`, kind, name, color, accFirstValueName, accSecondValueName)
}

// testAccColoredBundleSteps covers the color of the values of a bundle: set on create (a value without one gets
// the default style), changed, changed outside of Terraform (which the next apply reverts, on a value that sets
// it and on one that does not), removed from the configuration (which resets it to the default style), set
// again, set explicitly to the default style, and imported with its colors verified. After every step the
// framework checks that a new plan is empty, so none of these leaves a permanent diff.
func testAccColoredBundleSteps(t *testing.T, kind, name string, recolor recolorFunc) []resource.TestStep {
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
			PreConfig: func() {
				testAccRecolorOutOfBand(t, recolor, name, accFirstValueName, accColorOutOfBand)()
				testAccRecolorOutOfBand(t, recolor, name, accSecondValueName, accColorOutOfBand)()
			},
			Config: testAccColoredBundleConfig(kind, name, accColorChanged),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(address, accColorFirstValue, accColorChanged),
				resource.TestCheckResourceAttr(address, accColorSecondValue, accColorDefault),
			),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, ""),
			Check:  resource.TestCheckResourceAttr(address, accColorFirstValue, accColorDefault),
		},
		{
			Config: testAccColoredBundleConfig(kind, name, accColorChanged),
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
		Steps:                    testAccColoredBundleSteps(t, "enum", fmt.Sprintf("TF Acc Enum Color %d", time.Now().UnixMilli()), recolorEnumValue),
	})
}

func TestAccStateBundleColor(t *testing.T) {
	skipUnlessAcc(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps:                    testAccColoredBundleSteps(t, "state", fmt.Sprintf("TF Acc State Color %d", time.Now().UnixMilli()), recolorStateValue),
	})
}
