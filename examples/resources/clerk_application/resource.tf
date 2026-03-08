resource "clerk_application" "example" {
  name = "my-application"
}

# With all optional create-only fields
resource "clerk_application" "full_example" {
  name              = "my-full-application"
  domain            = "example.com"
  environment_types = ["development", "production"]
  template          = "next"
}
