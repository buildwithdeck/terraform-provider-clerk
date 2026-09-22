package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationRoleConfig("Test Role", "org:test_role", "A test role"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Role"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact("org:test_role"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test role"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccOrganizationRoleConfig("Updated Role", "org:test_role", "An updated role"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Role"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("An updated role"),
					),
				},
			},
		},
	})
}

func TestAccOrganizationRoleWithPermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create role with permission
			{
				Config: testAccOrganizationRoleWithPermissionsConfig("Role With Perms", "org:test_role_perms", "org:test:perm_for_role"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test_with_perms",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Role With Perms"),
					),
				},
			},
		},
	})
}

func testAccOrganizationRoleConfig(name, key, description string) string {
	return fmt.Sprintf(`
resource "clerk_organization_role" "test" {
  name        = %[1]q
  key         = %[2]q
  description = %[3]q
}
`, name, key, description)
}

func testAccOrganizationRoleWithPermissionsConfig(roleName, roleKey, permKey string) string {
	return fmt.Sprintf(`
resource "clerk_organization_permission" "test_for_role" {
  name = "Perm for role test"
  key  = %[3]q
}

resource "clerk_organization_role" "test_with_perms" {
  name        = %[1]q
  key         = %[2]q
  permissions = [clerk_organization_permission.test_for_role.id]
}
`, roleName, roleKey, permKey)
}

// TestAccOrganizationRolePermissionsOrderInsensitive verifies that a role's
// permissions are order-insensitive: creating a role with three permissions and
// then re-applying the same set in a different order must produce an empty plan.
// With an ordered List this reordering caused a perpetual diff (and tainted the
// role on apply); with a Set it is a no-op.
func TestAccOrganizationRolePermissionsOrderInsensitive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with permissions in order a, b, c.
			{
				Config: testAccOrganizationRolePermsOrderConfig("a", "b", "c"),
			},
			// Same three permissions, reordered — must be a no-op.
			{
				Config:             testAccOrganizationRolePermsOrderConfig("c", "a", "b"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccOrganizationRolePermsOrderConfig(first, second, third string) string {
	ref := map[string]string{
		"a": "clerk_organization_permission.perm_a.id",
		"b": "clerk_organization_permission.perm_b.id",
		"c": "clerk_organization_permission.perm_c.id",
	}
	// The permissions are chained with depends_on so Terraform creates them
	// sequentially: Clerk's API returns HTTP 500 (internal_clerk_error,
	// "Unable to create organization permission") when several permissions are
	// created concurrently, which is what Terraform would do by default for
	// three resources with no dependencies between them.
	return fmt.Sprintf(`
resource "clerk_organization_permission" "perm_a" {
  name = "Order Test A"
  key  = "org:order_test:a"
}

resource "clerk_organization_permission" "perm_b" {
  name = "Order Test B"
  key  = "org:order_test:b"

  depends_on = [clerk_organization_permission.perm_a]
}

resource "clerk_organization_permission" "perm_c" {
  name = "Order Test C"
  key  = "org:order_test:c"

  depends_on = [clerk_organization_permission.perm_b]
}

resource "clerk_organization_role" "order_test" {
  name        = "Order Test Role"
  key         = "org:test_role_order"
  permissions = [%[1]s, %[2]s, %[3]s]
}
`, ref[first], ref[second], ref[third])
}
