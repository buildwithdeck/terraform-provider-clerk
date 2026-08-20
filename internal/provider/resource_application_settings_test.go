package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// The refresh between steps exercises the live GET
// /instance/organization_settings (issue #29: Read must not write).
// force_organization_selection is deliberately not set here — it is a beta
// API field and may be rejected on instances without the feature; it is
// covered by the unit test instead.
func TestAccApplicationSettingsResource(t *testing.T) {
	// Destroying this singleton disables organizations instance-wide, which
	// breaks every later TestAccOrganization*/TestAccRoleSet* test on the
	// shared ephemeral instance. Re-enable them once the test is done.
	// Gated on TF_ACC: resource.Test skips inside itself, but t.Cleanup
	// would still run.
	if os.Getenv("TF_ACC") != "" {
		t.Cleanup(func() {
			client := instancesettings.NewClient(&clerk.ClientConfig{
				BackendConfig: clerk.BackendConfig{
					Key: clerk.String(os.Getenv("CLERK_API_KEY")),
				},
			})
			if _, err := client.UpdateOrganizationSettings(context.Background(), &instancesettings.UpdateOrganizationSettingsParams{
				Enabled: clerk.Bool(true),
			}); err != nil {
				t.Errorf("failed to re-enable organizations after test: %v", err)
			}
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccApplicationSettingsConfig(3, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("enable_organizations"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("max_allowed_memberships"),
						knownvalue.Int64Exact(3),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("admin_delete_enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("creator_role"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update
			{
				Config: testAccApplicationSettingsConfig(5, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("max_allowed_memberships"),
						knownvalue.Int64Exact(5),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_settings.test",
						tfjsonpath.New("admin_delete_enabled"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccApplicationSettingsConfig(maxMemberships int, adminDelete bool) string {
	return fmt.Sprintf(`
resource "clerk_application_settings" "test" {
  enable_organizations    = true
  max_allowed_memberships = %d
  admin_delete_enabled    = %t
}
`, maxMemberships, adminDelete)
}
