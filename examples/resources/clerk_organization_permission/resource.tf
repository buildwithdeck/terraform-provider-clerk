resource "clerk_organization_permission" "example" {
  name        = "Create posts"
  key         = "org:posts:create"
  description = "Allows creating posts within the organization"
}
