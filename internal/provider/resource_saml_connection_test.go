package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSAMLConnectionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccSAMLConnectionConfig("Test SAML Connection", "tf-acc-saml-test.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test SAML Connection"),
					),
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact("tf-acc-saml-test.example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("provider_type"),
						knownvalue.StringExact("saml_custom"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_saml_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config:             testAccSAMLConnectionConfig("Updated SAML Connection", "tf-acc-saml-test.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated SAML Connection"),
					),
				},
			},
		},
	})
}

func testAccSAMLConnectionConfig(name, domain string) string {
	return fmt.Sprintf(`
resource "clerk_saml_connection" "test" {
  name     = %[1]q
  domain   = %[2]q
  provider_type = "saml_custom"
}
`, name, domain)
}
