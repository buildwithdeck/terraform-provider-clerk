package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccApplicationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPlatform(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccApplicationConfig("tf-acc-test-app"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-app"),
					),
				},
			},
			// Import — create-only fields will be null after import
			{
				ResourceName:            "clerk_application.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "domain", "proxy_path", "environment_types", "template"},
			},
			// Update name
			{
				Config: testAccApplicationConfig("tf-acc-test-app-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-app-updated"),
					),
				},
			},
		},
	})
}

func testAccApplicationConfig(name string) string {
	return fmt.Sprintf(`
resource "clerk_application" "test" {
  name = %[1]q
}
`, name)
}
