package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationConfig("tf-acc-test-org"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-org"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccOrganizationConfig("tf-acc-test-org-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-org-updated"),
					),
				},
			},
		},
	})
}

func testAccOrganizationConfig(name string) string {
	return fmt.Sprintf(`
resource "clerk_organization" "test" {
  name = %[1]q
}
`, name)
}
