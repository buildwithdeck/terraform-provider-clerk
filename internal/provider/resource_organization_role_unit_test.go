package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The role's "permissions" attribute must be a Set, not a List.
//
// Clerk returns a role's permissions in an unspecified order, so modeling them
// as an ordered List makes Terraform report spurious reorder diffs on every plan
// and mark the role tainted after apply ("provider produced inconsistent result").
// A Set is order-insensitive and avoids that churn. This is a regression guard:
// if the attribute is ever changed back to a ListAttribute, this test fails.
func TestOrganizationRolePermissionsAttributeIsSet(t *testing.T) {
	r, ok := NewOrganizationRoleResource().(*OrganizationRoleResource)
	if !ok {
		t.Fatal("NewOrganizationRoleResource did not return *OrganizationRoleResource")
	}

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["permissions"]
	if !ok {
		t.Fatal(`schema is missing the "permissions" attribute`)
	}

	if _, isSet := attr.(schema.SetAttribute); !isSet {
		t.Fatalf("permissions must be schema.SetAttribute (order-insensitive) to avoid reorder churn; got %T", attr)
	}
}
