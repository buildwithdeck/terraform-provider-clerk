# The Clerk provider supports two usage modes. You can use either or both
# depending on whether you have access to the Platform API beta.
#
# Mode 1: Instance-only (most users)
# -----------------------------------
# Manage resources within a single Clerk instance using the Instance API.
# Only api_key is required. This covers organizations, JWT templates,
# application_settings, SAML connections, OAuth apps, and more.
#
#   provider "clerk" {
#     api_key = "sk_test_..."  # or set CLERK_API_KEY env var
#   }
#
# Mode 2: Platform + Instance (beta)
# -----------------------------------
# Manage applications and their instances via the Platform API, then
# configure each instance with the Instance API. Both keys are needed.
#
#   provider "clerk" {
#     api_key          = "sk_test_..."  # or set CLERK_API_KEY env var
#     platform_api_key = "ak_..."      # or set CLERK_PLATFORM_API_KEY env var
#   }
#
# Either key can be omitted — resources that require the missing key will
# produce a clear error, while all other resources work normally.

provider "clerk" {
  # api_key          = "sk_test_..."  # or set CLERK_API_KEY env var
  # platform_api_key = "ak_..."      # or set CLERK_PLATFORM_API_KEY env var
}
