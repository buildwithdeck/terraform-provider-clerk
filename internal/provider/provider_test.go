package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"clerk": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CLERK_API_KEY") == "" {
		t.Fatal("CLERK_API_KEY must be set for acceptance tests")
	}
}

func testAccPreCheckPlatform(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("CLERK_PLATFORM_API_KEY") == "" {
		t.Fatal("CLERK_PLATFORM_API_KEY must be set for acceptance tests")
	}
}
