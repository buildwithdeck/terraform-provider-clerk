resource "clerk_organization_role" "editor" {
  name        = "Editor"
  key         = "org:editor"
  description = "Can edit content within the organization"
  permissions = [clerk_organization_permission.create_posts.id]
}
