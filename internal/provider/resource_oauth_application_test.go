package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOAuthApplicationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOAuthApplicationConfig("Test OAuth App", "https://example.com/oauth/callback", "profile email"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_oauth_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test OAuth App"),
					),
					statecheck.ExpectKnownValue(
						"clerk_oauth_application.test",
						tfjsonpath.New("callback_url"),
						knownvalue.StringExact("https://example.com/oauth/callback"),
					),
					statecheck.ExpectKnownValue(
						"clerk_oauth_application.test",
						tfjsonpath.New("scopes"),
						knownvalue.StringExact("profile email"),
					),
					statecheck.ExpectKnownValue(
						"clerk_oauth_application.test",
						tfjsonpath.New("public"),
						knownvalue.Bool(false),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_oauth_application.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			// Update name
			{
				Config: testAccOAuthApplicationConfig("Updated OAuth App", "https://example.com/oauth/callback", "profile email"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_oauth_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated OAuth App"),
					),
				},
			},
		},
	})
}

func testAccOAuthApplicationConfig(name, callbackURL, scopes string) string {
	return fmt.Sprintf(`
resource "clerk_oauth_application" "test" {
  name         = %[1]q
  callback_url = %[2]q
  scopes       = %[3]q
  public       = false
}
`, name, callbackURL, scopes)
}
