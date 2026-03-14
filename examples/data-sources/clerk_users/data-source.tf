data "clerk_users" "example" {
  application_id = clerk_application.example.id
  instance_id    = clerk_application.example.instances[0].instance_id
}
