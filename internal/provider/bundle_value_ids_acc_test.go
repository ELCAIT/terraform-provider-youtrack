package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// bundleValueIDs records the ID of each value of a bundle resource by name, so
// a later step can check that an update kept them.
type bundleValueIDs map[string]string

func (ids bundleValueIDs) values(s *terraform.State, resourceName string) (map[string]string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceName)
	}

	byName := map[string]string{}
	for i := 0; ; i++ {
		name, ok := rs.Primary.Attributes[fmt.Sprintf("values.%d.name", i)]
		if !ok {
			return byName, nil
		}
		byName[name] = rs.Primary.Attributes[fmt.Sprintf("values.%d.id", i)]
	}
}

func (ids bundleValueIDs) record(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		current, err := ids.values(s, resourceName)
		for name, id := range current {
			ids[name] = id
		}
		return err
	}
}

// unchanged fails when any recorded value now has a different ID: YouTrack
// clears a recreated value on every issue that used it.
func (ids bundleValueIDs) unchanged(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		current, err := ids.values(s, resourceName)
		if err != nil {
			return err
		}
		for name, id := range ids {
			if got, ok := current[name]; ok && got != id {
				return fmt.Errorf("value %q was recreated: id %s became %s", name, id, got)
			}
		}
		return nil
	}
}

func testAccBundleValueIDsConfig(resourceType, name, extra string, reordered bool) string {
	values := `
    { name = "Alpha", description = "first" },
    { name = "Beta" },`
	if reordered {
		values = `
    { name = "Beta" },
    { name = "Alpha", description = "changed", archived = true },
    { name = "Gamma" },`
	}

	return providerBlock() + fmt.Sprintf(`
resource %q "test" {
  name   = %q
  values = [%s
  ]
}
%s`, resourceType, name, values, extra)
}

// TestAccBundleValuesKeepTheirIDs is the regression test for bundle updates
// recreating every value: renaming the bundle, editing, archiving and
// reordering values and adding a new one must leave existing values' IDs alone.
func TestAccBundleValuesKeepTheirIDs(t *testing.T) {
	skipUnlessAcc(t)

	for _, resourceType := range []string{"youtrack_enum_bundle", "youtrack_state_bundle", "youtrack_owned_bundle"} {
		t.Run(resourceType, func(t *testing.T) {
			resourceName := resourceType + ".test"
			name := fmt.Sprintf("TF Acc IDs %s %d", resourceType, time.Now().UnixMilli())
			ids := bundleValueIDs{}

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: testAccBundleValueIDsConfig(resourceType, name, "", false),
						Check:  ids.record(resourceName),
					},
					{
						Config: testAccBundleValueIDsConfig(resourceType, name+" renamed", "", true),
						Check: resource.ComposeAggregateTestCheckFunc(
							ids.unchanged(resourceName),
							resource.TestCheckResourceAttr(resourceName, "values.#", "3"),
							resource.TestCheckResourceAttr(resourceName, "values.0.name", "Beta"),
							resource.TestCheckResourceAttr(resourceName, "values.1.name", "Alpha"),
							resource.TestCheckResourceAttr(resourceName, "values.1.description", "changed"),
							resource.TestCheckResourceAttr(resourceName, "values.1.archived", "true"),
							resource.TestCheckResourceAttr(resourceName, "values.2.name", "Gamma"),
						),
					},
					{
						ResourceName:      resourceName,
						ImportState:       true,
						ImportStateIdFunc: importStateID(resourceName),
						ImportStateVerify: true,
					},
				},
			})
		})
	}
}
