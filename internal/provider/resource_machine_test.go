package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMachineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccMachineConfig("Test Machine"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_machine.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Machine"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_machine.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret_key"},
			},
			// Update name
			{
				Config: testAccMachineConfig("Updated Machine"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_machine.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Machine"),
					),
				},
			},
		},
	})
}

func testAccMachineConfig(name string) string {
	return fmt.Sprintf(`
resource "clerk_machine" "test" {
  name = %[1]q
}
`, name)
}
