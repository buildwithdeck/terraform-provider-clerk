resource "clerk_instance_config" "example" {
  application_id = clerk_application.example.id
  instance_id    = clerk_application.example.instances[0].instance_id

  config = jsonencode({
    url_based_session_syncing = "1"
  })
}
