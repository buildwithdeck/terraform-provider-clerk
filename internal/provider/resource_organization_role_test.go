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
