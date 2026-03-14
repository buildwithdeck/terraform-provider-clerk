package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDomainResource(t *testing.T) {
	domainName := fmt.Sprintf("tf-acc-%d.example.com", time.Now().Unix())

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
				Config: testAccDomainConfig(testAccEphemeralAppID, domainName, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"clerk_domain.test",
						tfjsonpath.New("application_id"),
						knownvalue.StringExact(testAccEphemeralAppID),
					),
				},
			},
			// Import using "appID/domainID" format
			{
				ResourceName:            "clerk_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       testAccDomainImportStateIDFunc,
				ImportStateVerifyIgnore: []string{"proxy_path"},
			},
		},
	})
}

func testAccDomainImportStateIDFunc(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["clerk_domain.test"]
	if !ok {
		return "", fmt.Errorf("resource clerk_domain.test not found")
	}
	return rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["id"], nil
}

func testAccDomainConfig(appID, name, proxyPath string) string {
	proxyLine := ""
	if proxyPath != "" {
		proxyLine = fmt.Sprintf("\n  proxy_path = %q", proxyPath)
	}
	return fmt.Sprintf(`
resource "clerk_domain" "test" {
  application_id = %[1]q
  name           = %[2]q%[3]s
}
`, appID, name, proxyLine)
}
