resource "clerk_role_set" "default" {
  name        = "Default Role Set"
  key         = "default_roles"
  description = "Default set of roles assigned to new organization members"
  roles       = [clerk_organization_role.editor.key]
}
