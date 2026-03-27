resource "clerk_organization_membership" "example" {
  organization_id = clerk_organization.example.id
  user_id         = "user_abc123"
  role            = "org:admin"
}
