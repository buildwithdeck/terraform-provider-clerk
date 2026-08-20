package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read must never write. The old implementation issued a PATCH during
// refresh, mutating the instance on every terraform plan (issue #29).
func TestApplicationSettingsReadIsGetOnly(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method != http.MethodGet {
			t.Errorf("Read issued a mutating request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "organization_settings",
			"enabled": true,
			"max_allowed_memberships": 5,
			"max_allowed_roles": 10,
			"max_allowed_permissions": 20,
			"creator_role": "org:admin",
			"admin_delete_enabled": true,
			"domains_enabled": false,
			"domains_enrollment_modes": [],
			"domains_default_role": "org:member"
		}`))
	}))
	defer server.Close()

	r := &ApplicationSettingsResource{
		client: instancesettings.NewClient(&clerk.ClientConfig{
			BackendConfig: clerk.BackendConfig{
				Key: clerk.String("sk_test_x"),
				URL: clerk.String(server.URL),
			},
		}),
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(ctx, &ApplicationSettingsResourceModel{
		ID:                         types.StringValue("application_settings"),
		EnableOrganizations:        types.BoolValue(true),
		MaxAllowedMemberships:      types.Int64Value(5),
		MaxAllowedRoles:            types.Int64Value(10),
		MaxAllowedPermissions:      types.Int64Value(20),
		CreatorRole:                types.StringValue("org:admin"),
		AdminDeleteEnabled:         types.BoolValue(true),
		DomainsEnabled:             types.BoolValue(false),
		DomainsDefaultRole:         types.StringValue("org:member"),
		ForceOrganizationSelection: types.BoolValue(false),
	})
	if diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	if len(requests) != 1 || requests[0] != "GET /instance/organization_settings" {
		t.Fatalf("expected exactly one GET /instance/organization_settings, got %v", requests)
	}

	// The write-only field must survive a refresh untouched.
	var got ApplicationSettingsResourceModel
	if diags := readResp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading result state: %v", diags)
	}
	if got.ForceOrganizationSelection.ValueBool() != false || got.ForceOrganizationSelection.IsNull() {
		t.Fatalf("force_organization_selection changed on Read: %v", got.ForceOrganizationSelection)
	}
}
