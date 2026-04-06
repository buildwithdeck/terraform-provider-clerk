# Look up an existing Clerk application by ID.
data "clerk_application" "example" {
  id = "app_abc123"
}

# Use the application's instance details to configure instance-level resources.
output "dev_instance_id" {
  value = [for inst in data.clerk_application.example.instances : inst.instance_id if inst.environment_type == "development"][0]
}
