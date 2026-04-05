package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRoleSetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccRoleSetConfig("Test Role Set", "role_set:test_role_set", "A test role set"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_role_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Role Set"),
					),
					statecheck.ExpectKnownValue(
						"clerk_role_set.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact("role_set:test_role_set"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_role_set.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"default_role_key", "creator_role_key"},
			},
			// Update
			{
				Config: testAccRoleSetConfig("Updated Role Set", "role_set:test_role_set", "An updated role set"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_role_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Role Set"),
					),
				},
			},
		},
	})
}

func TestAccRoleSetWithRolesResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create role set with a role
			{
				Config: testAccRoleSetWithRolesConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_role_set.test_with_roles",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Role Set With Roles"),
					),
				},
			},
		},
	})
}

func testAccRoleSetConfig(name, key, description string) string {
	return fmt.Sprintf(`
resource "clerk_organization_role" "default_for_set" {
  name = "Default role for set test"
  key  = "org:default_for_set"
}

resource "clerk_organization_role" "creator_for_set" {
  name = "Creator role for set test"
  key  = "org:creator_for_set"
}

resource "clerk_role_set" "test" {
  name             = %[1]q
  key              = %[2]q
  description      = %[3]q
  default_role_key = clerk_organization_role.default_for_set.key
  creator_role_key = clerk_organization_role.creator_for_set.key
  roles            = [clerk_organization_role.default_for_set.key, clerk_organization_role.creator_for_set.key]
}
`, name, key, description)
}

func testAccRoleSetWithRolesConfig() string {
	return `
resource "clerk_organization_role" "default_for_roles_set" {
  name = "Default role for roles set test"
  key  = "org:default_for_roles_set"
}

resource "clerk_organization_role" "creator_for_roles_set" {
  name = "Creator role for roles set test"
  key  = "org:creator_for_roles_set"
}

resource "clerk_organization_role" "test_for_role_set" {
  name = "Role for role set test"
  key  = "org:test_for_role_set"
}

resource "clerk_role_set" "test_with_roles" {
  name             = "Role Set With Roles"
  key              = "role_set:test_role_set_with_roles"
  default_role_key = clerk_organization_role.default_for_roles_set.key
  creator_role_key = clerk_organization_role.creator_for_roles_set.key
  roles            = [clerk_organization_role.default_for_roles_set.key, clerk_organization_role.creator_for_roles_set.key, clerk_organization_role.test_for_role_set.key]
}
`
}
