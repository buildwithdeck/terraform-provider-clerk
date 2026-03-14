resource "clerk_domain" "example" {
  application_id = clerk_application.example.id
  name           = "app.example.com"
}
