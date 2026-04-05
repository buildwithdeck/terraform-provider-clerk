package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationDomainResource(t *testing.T) {
	t.Skip("Skipped: test Clerk instance does not have organization domains enabled")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationDomainConfig("tf-acc-test.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test.example.com"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_organization_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"verified"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["clerk_organization_domain.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["organization_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update enrollment mode
			{
				Config: testAccOrganizationDomainConfigWithEnrollment("tf-acc-test.example.com", "automatic_suggestion"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_domain.test",
						tfjsonpath.New("enrollment_mode"),
						knownvalue.StringExact("automatic_suggestion"),
					),
				},
			},
		},
	})
}

func testAccOrganizationDomainConfig(domainName string) string {
	return fmt.Sprintf(`
resource "clerk_organization" "test_for_domain" {
  name = "tf-acc-test-org-domain"
}

resource "clerk_organization_domain" "test" {
  organization_id = clerk_organization.test_for_domain.id
  name            = %[1]q
}
`, domainName)
}

func testAccOrganizationDomainConfigWithEnrollment(domainName, enrollmentMode string) string {
	return fmt.Sprintf(`
resource "clerk_organization" "test_for_domain" {
  name = "tf-acc-test-org-domain"
}

resource "clerk_organization_domain" "test" {
  organization_id = clerk_organization.test_for_domain.id
  name            = %[1]q
  enrollment_mode = %[2]q
}
`, domainName, enrollmentMode)
}
