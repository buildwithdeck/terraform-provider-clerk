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

func TestAccOrganizationMembershipResource(t *testing.T) {
	if testAccTestUserID == "" {
		t.Skip("testAccTestUserID not set, skipping membership test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationMembershipConfig(testAccTestUserID, "org:admin"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_membership.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("org:admin"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_membership.test",
						tfjsonpath.New("user_id"),
						knownvalue.StringExact(testAccTestUserID),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization_membership.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["clerk_organization_membership.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["organization_id"] + "/" + rs.Primary.Attributes["user_id"], nil
				},
			},
			// Update role
			{
				Config: testAccOrganizationMembershipConfig(testAccTestUserID, "org:member"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_membership.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("org:member"),
					),
				},
			},
		},
	})
}

func testAccOrganizationMembershipConfig(userID, role string) string {
	return fmt.Sprintf(`
resource "clerk_organization" "test_for_membership" {
  name = "tf-acc-test-org-membership"
}

resource "clerk_organization_membership" "test" {
  organization_id = clerk_organization.test_for_membership.id
  user_id         = %[1]q
  role            = %[2]q
}
`, userID, role)
}
