package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccInstanceConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckPlatform(t)
			if testAccEphemeralAppID == "" {
				t.Fatal("testAccEphemeralAppID must be set for instance config tests")
			}
			if testAccEphemeralInstanceID == "" {
				t.Fatal("testAccEphemeralInstanceID must be set for instance config tests")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with a simple config
			{
				Config: testAccInstanceConfigConfig(`{"auth_password":{"min_length":10}}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_instance_config.test",
						tfjsonpath.New("config"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"clerk_instance_config.test",
						tfjsonpath.New("config_version"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"clerk_instance_config.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update with modified config
			{
				Config: testAccInstanceConfigConfig(`{"auth_password":{"min_length":12}}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_instance_config.test",
						tfjsonpath.New("config"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"clerk_instance_config.test",
						tfjsonpath.New("config_version"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccInstanceConfigConfig(configJSON string) string {
	return fmt.Sprintf(`
resource "clerk_instance_config" "test" {
  application_id = %[1]q
  instance_id    = %[2]q
  config         = %[3]q
}
`, testAccEphemeralAppID, testAccEphemeralInstanceID, configJSON)
}
