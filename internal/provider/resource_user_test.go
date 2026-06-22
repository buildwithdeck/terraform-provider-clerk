package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccUserConfig("tf-acc-test-user@example.com", "Seed"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_user.test",
						tfjsonpath.New("email_addresses").AtSliceIndex(0),
						knownvalue.StringExact("tf-acc-test-user@example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_user.test",
						tfjsonpath.New("first_name"),
						knownvalue.StringExact("Seed"),
					),
					statecheck.ExpectKnownValue(
						"clerk_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"clerk_user.test",
						tfjsonpath.New("primary_email_address_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import — write-only fields are never returned by the API.
			{
				ResourceName:      "clerk_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"password_digest",
					"password_hasher",
					"skip_password_requirement",
					"skip_password_checks",
					"skip_legal_checks",
					"totp_secret",
					"backup_codes",
					"legal_accepted_at",
				},
			},
			// Update first_name
			{
				Config: testAccUserConfig("tf-acc-test-user@example.com", "Updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_user.test",
						tfjsonpath.New("first_name"),
						knownvalue.StringExact("Updated"),
					),
				},
			},
		},
	})
}

func testAccUserConfig(email, firstName string) string {
	return fmt.Sprintf(`
resource "clerk_user" "test" {
  email_addresses = [%[1]q]
  password        = "tf-acc-test-password-123!"
  first_name      = %[2]q
  last_name       = "User"
  public_metadata = jsonencode({ role = "tester" })
}
`, email, firstName)
}
