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

# With logo and favicon
resource "clerk_application" "branded_example" {
  name    = "my-branded-application"
  logo    = filebase64("logo.png")
  favicon = filebase64("favicon.ico")
}
