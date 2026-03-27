resource "clerk_organization_domain" "example" {
  organization_id = clerk_organization.example.id
  name            = "example.com"
  enrollment_mode = "automatic_invitation"
  verified        = true
}
