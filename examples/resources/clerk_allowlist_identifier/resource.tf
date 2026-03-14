# Allow a specific email address
resource "clerk_allowlist_identifier" "specific_email" {
  identifier = "alice@example.com"
  notify     = true
}

# Allow all emails from a domain
resource "clerk_allowlist_identifier" "domain_wildcard" {
  identifier = "*@example.com"
}
