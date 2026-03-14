package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAllowlistIdentifierResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccAllowlistIdentifierConfig("test@example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_allowlist_identifier.test",
						tfjsonpath.New("identifier"),
						knownvalue.StringExact("test@example.com"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_allowlist_identifier.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"notify"},
			},
		},
	})
}

func testAccAllowlistIdentifierConfig(identifier string) string {
	return `
resource "clerk_allowlist_identifier" "test" {
  identifier = "` + identifier + `"
  notify     = false
}
`
}
