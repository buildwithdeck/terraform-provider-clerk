package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckPlatform(t)
			if testAccEphemeralAppID == "" {
				t.Fatal("testAccEphemeralAppID must be set")
			}
			if testAccEphemeralInstanceID == "" {
				t.Fatal("testAccEphemeralInstanceID must be set")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig(testAccEphemeralAppID, testAccEphemeralInstanceID),
				// Just verify it runs without error and total_count is set
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.clerk_users.test",
						tfjsonpath.New("total_count"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccUsersDataSourceConfig(appID, instanceID string) string {
	return fmt.Sprintf(`
data "clerk_users" "test" {
  application_id = %[1]q
  instance_id    = %[2]q
}
`, appID, instanceID)
}
