package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccInstanceDomainResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccInstanceDomainConfig("tf-acc-test.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_instance_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test.example.com"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_instance_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccInstanceDomainConfig("tf-acc-test-updated.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_instance_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-updated.example.com"),
					),
				},
			},
		},
	})
}

func testAccInstanceDomainConfig(name string) string {
	return fmt.Sprintf(`
resource "clerk_instance_domain" "test" {
  name = %[1]q
}
`, name)
}
