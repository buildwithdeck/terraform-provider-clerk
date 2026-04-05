package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationPermissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationPermissionConfig("Test Permission", "org:test:create", "A test permission"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Permission"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact("org:test:create"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test permission"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccOrganizationPermissionConfig("Updated Permission", "org:test:create", "An updated permission"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Permission"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("An updated permission"),
					),
				},
			},
		},
	})
}

func testAccOrganizationPermissionConfig(name, key, description string) string {
	return fmt.Sprintf(`
resource "clerk_organization_permission" "test" {
  name        = %[1]q
  key         = %[2]q
  description = %[3]q
}
`, name, key, description)
}
