resource "clerk_oauth_application" "my_app" {
  name         = "My OAuth App"
  callback_url = "https://example.com/oauth/callback"
  scopes       = "profile email"
  public       = false
}
