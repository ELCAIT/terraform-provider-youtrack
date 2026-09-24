package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accOwnedBundleResource = "youtrack_owned_bundle.test"
	accOwnedFieldResource  = "youtrack_custom_field.owned"
)

func testOwnedBundleOwnerLogin(t *testing.T) string {
	t.Helper()

	login := os.Getenv(envAccUser)
	if login == "" {
		t.Skipf("set %s to run owned bundle acceptance tests", envAccUser)
	}

	return login
}

func testAccOwnedBundleCreateConfig(name, owner string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_owned_bundle" "test" {
  name = %[1]q

  values = [
    {
      name        = "Backend"
      description = "Server side"
      owner_login = %[2]q
    },
    {
      name = "Frontend"
    },
  ]
}
`, name, owner)
}

// testAccOwnedBundleUpdateConfig renames the bundle, moves the owner from one
// value to the other, archives a value, reorders them and adds a new one.
func testAccOwnedBundleUpdateConfig(name, owner string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_owned_bundle" "test" {
  name = %[1]q

  values = [
    {
      name        = "Frontend"
      owner_login = %[2]q
    },
    {
      name        = "Mobile"
      description = "Apps"
    },
    {
      name        = "Backend"
      description = "Server side"
      archived    = true
    },
  ]
}
`, name, owner)
}

// testAccOwnedFieldConfig drops a value and attaches the bundle to an
// ownedField[1] custom field by name, with a default value.
func testAccOwnedFieldConfig(name, owner, fieldName string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_owned_bundle" "test" {
  name = %[1]q

  values = [
    {
      name        = "Frontend"
      owner_login = %[2]q
    },
    {
      name        = "Mobile"
      description = "Apps"
    },
  ]
}

resource "youtrack_custom_field" "owned" {
  name          = %[3]q
  field_type_id = "ownedField[1]"

  field_defaults = {
    can_be_empty        = true
    bundle_name         = youtrack_owned_bundle.test.name
    default_value_names = ["Mobile"]
  }
}
`, name, owner, fieldName)
}

func TestAccOwnedBundle(t *testing.T) {
	skipUnlessAcc(t)
	owner := testOwnedBundleOwnerLogin(t)

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("TF Acc Owned Bundle %d", suffix)
	nameUpdated := fmt.Sprintf("TF Acc Owned Bundle Updated %d", suffix)
	fieldName := fmt.Sprintf("TF Acc Owned Field %d", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccOwnedBundleCreateConfig(name, owner),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accOwnedBundleResource, "id"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "name", name),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.#", "2"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.0.name", "Backend"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.0.owner_login", owner),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.0.description", "Server side"),
					resource.TestCheckNoResourceAttr(accOwnedBundleResource, "values.1.owner_login"),
				),
			},
			{
				Config: testAccOwnedBundleUpdateConfig(nameUpdated, owner),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accOwnedBundleResource, "name", nameUpdated),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.#", "3"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.0.name", "Frontend"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.0.owner_login", owner),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.1.name", "Mobile"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.2.name", "Backend"),
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.2.archived", "true"),
					resource.TestCheckNoResourceAttr(accOwnedBundleResource, "values.2.owner_login"),
				),
			},
			{
				ResourceName:      accOwnedBundleResource,
				ImportState:       true,
				ImportStateIdFunc: importStateID(accOwnedBundleResource),
				ImportStateVerify: true,
			},
			{
				Config: testAccOwnedFieldConfig(nameUpdated, owner, fieldName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accOwnedBundleResource, "values.#", "2"),
					resource.TestCheckResourceAttr(accOwnedFieldResource, "field_type_id", "ownedField[1]"),
					resource.TestCheckResourceAttrPair(accOwnedFieldResource, "field_defaults.bundle_id", accOwnedBundleResource, "id"),
					resource.TestCheckResourceAttr(accOwnedFieldResource, "field_defaults.default_value_names.#", "1"),
					resource.TestCheckResourceAttr(accOwnedFieldResource, "field_defaults.default_value_names.0", "Mobile"),
				),
			},
		},
	})
}

// testAccProjectOwnedFieldConfig attaches an ownedField[1] field to a project
// with a project-specific bundle and a default value from it.
func testAccProjectOwnedFieldConfig(suffix int64, owner, leaderLogin string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_owned_bundle" "global" {
  name   = "TF Acc Owned Global %[1]d"
  values = [{ name = "Shared" }]
}

resource "youtrack_owned_bundle" "project" {
  name = "TF Acc Owned Project %[1]d"
  values = [
    { name = "Backend", owner_login = %[2]q },
    { name = "Frontend" },
  ]
}

resource "youtrack_custom_field" "global" {
  name          = "TF Acc Owned Project Field %[1]d"
  field_type_id = "ownedField[1]"

  field_defaults = {
    can_be_empty = true
    bundle_name  = youtrack_owned_bundle.global.name
  }
}

resource "youtrack_project" "parent" {
  name         = "TFAccOwned%[1]d"
  short_name   = "OWN%[4]d"
  leader_login = %[3]q
}

resource "youtrack_project_custom_field" "test" {
  project_id          = youtrack_project.parent.id
  field_name          = youtrack_custom_field.global.name
  field_type          = "OwnedProjectCustomField"
  bundle_name         = youtrack_owned_bundle.project.name
  default_value_names = ["Backend"]
}
`, suffix, owner, leaderLogin, suffix%10000)
}

func TestAccProjectOwnedField(t *testing.T) {
	skipUnlessAcc(t)
	owner := testOwnedBundleOwnerLogin(t)
	leaderLogin := testProjectLeaderLogin(t)

	suffix := time.Now().UnixMilli()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectOwnedFieldConfig(suffix, owner, leaderLogin),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "field_type", "OwnedProjectCustomField"),
					resource.TestCheckResourceAttrPair(accProjectCustomFieldResource, "bundle_name", "youtrack_owned_bundle.project", "name"),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "default_value_names.#", "1"),
					resource.TestCheckResourceAttr(accProjectCustomFieldResource, "default_value_names.0", "Backend"),
				),
			},
		},
	})
}
