resource "clerk_role_set" "default" {
  name             = "Default Role Set"
  key              = "role_set:default_roles"
  description      = "Default set of roles assigned to new organization members"
  default_role_key = clerk_organization_role.member.key
  creator_role_key = clerk_organization_role.admin.key
  roles            = [clerk_organization_role.member.key, clerk_organization_role.admin.key, clerk_organization_role.editor.key]
}
