package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	clerk "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/instancesettings"
)

// testAccEphemeralAppID holds the application ID of the ephemeral app created
// for acceptance tests. Domain tests reference this to create domains on the
// test application.
var testAccEphemeralAppID string

// TestMain manages ephemeral Clerk application lifecycle for acceptance tests.
// If CLERK_PLATFORM_API_KEY is set (and TF_ACC=1), it creates a temporary app,
// extracts the dev instance secret key, and sets CLERK_API_KEY automatically.
// Falls back to a static CLERK_API_KEY if no platform key is available.
func TestMain(m *testing.M) {
	// If not running acceptance tests, just run normally.
	if os.Getenv("TF_ACC") == "" {
		os.Exit(m.Run())
	}

	platformKey := os.Getenv("CLERK_PLATFORM_API_KEY")
	staticKey := os.Getenv("CLERK_API_KEY")

	// Fallback: no platform key, use static CLERK_API_KEY.
	if platformKey == "" {
		if staticKey == "" {
			log.Fatal("Either CLERK_PLATFORM_API_KEY or CLERK_API_KEY must be set for acceptance tests")
		}
		log.Println("TestMain: no CLERK_PLATFORM_API_KEY set, using static CLERK_API_KEY")
		os.Exit(m.Run())
	}

	// Create ephemeral application.
	client := NewPlatformClient(platformKey)
	ctx := context.Background()

	appName := fmt.Sprintf("tf-acc-%d", time.Now().Unix())
	log.Printf("TestMain: creating ephemeral app %q", appName)

	app, err := client.CreateApplication(ctx, &CreateApplicationParams{
		Name:             appName,
		EnvironmentTypes: []string{"development", "production"},
	})
	if err != nil {
		log.Fatalf("TestMain: failed to create ephemeral app: %v", err)
	}

	// Find dev instance secret key.
	var secretKey string
	for _, inst := range app.Instances {
		if inst.EnvironmentType == "development" && inst.SecretKey != "" {
			secretKey = inst.SecretKey
			break
		}
	}
	if secretKey == "" {
		// Clean up before failing.
		_, _ = client.DeleteApplication(ctx, app.ApplicationID)
		log.Fatal("TestMain: no development instance secret_key found in ephemeral app")
	}

	testAccEphemeralAppID = app.ApplicationID

	log.Printf("TestMain: ephemeral app %s created, waiting for instance readiness...", app.ApplicationID)

	// Wait for instance readiness by polling a lightweight API call.
	if err := waitForInstanceReady(secretKey, 60*time.Second); err != nil {
		_, _ = client.DeleteApplication(ctx, app.ApplicationID)
		log.Fatalf("TestMain: instance never became ready: %v", err)
	}

	log.Println("TestMain: instance is ready")

	// Enable organizations on the ephemeral dev instance via the Clerk Go SDK.
	log.Println("TestMain: enabling organizations on dev instance")
	clerk.SetKey(secretKey)
	_, err = instancesettings.UpdateOrganizationSettings(ctx, &instancesettings.UpdateOrganizationSettingsParams{
		Enabled: clerk.Bool(true),
	})
	if err != nil {
		log.Printf("TestMain: WARNING: failed to enable organizations: %v", err)
	}

	log.Println("TestMain: running tests")

	// Inject the ephemeral secret key.
	os.Setenv("CLERK_API_KEY", secretKey)

	// Run all tests.
	code := m.Run()

	// Teardown: delete ephemeral app.
	log.Printf("TestMain: deleting ephemeral app %s", app.ApplicationID)
	if _, err := client.DeleteApplication(ctx, app.ApplicationID); err != nil {
		log.Printf("TestMain: WARNING: failed to delete ephemeral app: %v", err)
	}

	// Restore original env var.
	if staticKey != "" {
		os.Setenv("CLERK_API_KEY", staticKey)
	} else {
		os.Unsetenv("CLERK_API_KEY")
	}

	os.Exit(code)
}

// waitForInstanceReady polls the Clerk API until the instance responds
// successfully or the timeout elapses.
func waitForInstanceReady(secretKey string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, "https://api.clerk.com/v1/jwt_templates", nil)
		if err != nil {
			return fmt.Errorf("creating readiness request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+secretKey)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("instance not ready after %s", timeout)
}
