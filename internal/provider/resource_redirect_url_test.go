package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRedirectURLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccRedirectURLConfig("https://example.com/callback"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_redirect_url.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/callback"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_redirect_url.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRedirectURLConfig(url string) string {
	return fmt.Sprintf(`
resource "clerk_redirect_url" "test" {
  url = %[1]q
}
`, url)
}
