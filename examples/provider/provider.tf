# Instance API key — required for clerk_jwt_template, clerk_organization,
# and clerk_application_settings resources.
# Set via CLERK_API_KEY environment variable or the api_key attribute.

# Platform API key — required for clerk_application, clerk_domain, and
# clerk_instance_config resources. The Platform API is a beta feature.
# Set via CLERK_PLATFORM_API_KEY environment variable or the platform_api_key attribute.

provider "clerk" {
  # api_key          = "sk_test_..."  # or set CLERK_API_KEY env var
  # platform_api_key = "ak_..."      # or set CLERK_PLATFORM_API_KEY env var
}
