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

func TestAccApplicationTransferResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckPlatform(t)
			if testAccEphemeralAppID == "" {
				t.Fatal("testAccEphemeralAppID must be set (created by TestMain)")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccApplicationTransferConfig(testAccEphemeralAppID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application_transfer.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("pending"),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_transfer.test",
						tfjsonpath.New("application_id"),
						knownvalue.StringExact(testAccEphemeralAppID),
					),
					statecheck.ExpectKnownValue(
						"clerk_application_transfer.test",
						tfjsonpath.New("code"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import using "appID/transferID" format
			{
				ResourceName:      "clerk_application_transfer.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccApplicationTransferImportStateIDFunc,
			},
		},
	})
}

func testAccApplicationTransferImportStateIDFunc(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["clerk_application_transfer.test"]
	if !ok {
		return "", fmt.Errorf("resource clerk_application_transfer.test not found")
	}
	return rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["id"], nil
}

func testAccApplicationTransferConfig(appID string) string {
	return fmt.Sprintf(`
resource "clerk_application_transfer" "test" {
  application_id = %[1]q
}
`, appID)
}
