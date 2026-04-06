package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccApplicationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPlatform(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationDataSourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.clerk_application.test",
						tfjsonpath.New("instances"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func testAccApplicationDataSourceConfig() string {
	return `
resource "clerk_application" "test" {
  name              = "tf-acc-test-ds-app"
  environment_types = ["development", "production"]
}

data "clerk_application" "test" {
  id = clerk_application.test.id
}
`
}
