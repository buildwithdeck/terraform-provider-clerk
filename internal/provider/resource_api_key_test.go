package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAPIKeyResource(t *testing.T) {
	t.Skip("Skipped: test Clerk instance does not have API keys feature enabled")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccAPIKeyConfig("Test API Key", "Initial description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_api_key.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test API Key"),
					),
					statecheck.ExpectKnownValue(
						"clerk_api_key.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Initial description"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "seconds_until_expiration"},
			},
			// Update description
			{
				Config: testAccAPIKeyConfig("Test API Key", "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_api_key.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
		},
	})
}

func testAccAPIKeyConfig(name, description string) string {
	return fmt.Sprintf(`
resource "clerk_api_key" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}
