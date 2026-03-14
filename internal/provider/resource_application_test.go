package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccApplicationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPlatform(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccApplicationConfig("tf-acc-test-app"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-app"),
					),
				},
			},
			// Import — create-only fields will be null after import
			{
				ResourceName:            "clerk_application.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "domain", "proxy_path", "environment_types", "template", "logo", "favicon"},
			},
			// Update name
			{
				Config: testAccApplicationConfig("tf-acc-test-app-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-app-updated"),
					),
				},
			},
		},
	})
}

// Minimal valid 1x1 PNG image, base64-encoded.
const testLogoBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

// Minimal valid 1x1 PNG used as favicon for testing.
const testFaviconBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

func TestAccApplicationResource_LogoFavicon(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPlatform(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with logo
			{
				Config: testAccApplicationConfigWithImages("tf-acc-test-branded", testLogoBase64, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-branded"),
					),
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("logo"),
						knownvalue.StringExact(testLogoBase64),
					),
				},
			},
			// Update: add favicon, keep logo
			{
				Config: testAccApplicationConfigWithImages("tf-acc-test-branded", testLogoBase64, testFaviconBase64),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("favicon"),
						knownvalue.StringExact(testFaviconBase64),
					),
				},
			},
			// Update: remove logo (set to null)
			{
				Config: testAccApplicationConfigWithImages("tf-acc-test-branded", "", testFaviconBase64),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_application.test",
						tfjsonpath.New("favicon"),
						knownvalue.StringExact(testFaviconBase64),
					),
				},
			},
		},
	})
}

func testAccApplicationConfigWithImages(name, logo, favicon string) string {
	extras := ""
	if logo != "" {
		extras += fmt.Sprintf("\n  logo = %q", logo)
	}
	if favicon != "" {
		extras += fmt.Sprintf("\n  favicon = %q", favicon)
	}
	return fmt.Sprintf(`
resource "clerk_application" "test" {
  name = %[1]q%[2]s
}
`, name, extras)
}

func testAccApplicationConfig(name string) string {
	return fmt.Sprintf(`
resource "clerk_application" "test" {
  name = %[1]q
}
`, name)
}
