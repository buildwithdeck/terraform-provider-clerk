resource "clerk_saml_connection" "okta" {
  name           = "Okta SSO"
  domain         = "example.com"
  provider_type  = "saml_custom"
  idp_entity_id  = "http://www.okta.com/exk1234567890"
  idp_sso_url    = "https://example.okta.com/app/example/sso/saml"

  attribute_mapping {
    user_id       = "nameid"
    email_address = "email"
    first_name    = "firstName"
    last_name     = "lastName"
  }
}
