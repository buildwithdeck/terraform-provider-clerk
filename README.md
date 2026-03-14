# Terraform Provider for Clerk

A Terraform provider for managing [Clerk](https://clerk.com) resources as infrastructure-as-code.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider)

## Using the Provider

```hcl
terraform {
  required_providers {
    clerk = {
      source = "buildwithdeck/clerk"
    }
  }
}

provider "clerk" {
  # Instance API key (required) — for managing resources on a specific Clerk instance
  # Set via CLERK_API_KEY environment variable or api_key attribute

  # Platform API key (optional) — for managing applications, domains, and instance config
  # Set via CLERK_PLATFORM_API_KEY environment variable or platform_api_key attribute
}
```

See the [documentation](https://registry.terraform.io/providers/buildwithdeck/clerk/latest/docs) for full resource reference.

## Supported Resources

This provider uses two Clerk APIs. Each resource requires the corresponding API key.

### Instance API (`CLERK_API_KEY`)

These resources manage settings on a specific Clerk instance (identified by its secret key):

- `clerk_jwt_template` — Manage JWT templates
- `clerk_organization` — Manage organizations
- `clerk_application_settings` — Manage instance-level application settings

### Platform API (`CLERK_PLATFORM_API_KEY`) — Beta

These resources use the [Platform API](https://clerk.com/docs) to manage applications and their sub-resources across your workspace. The Platform API is a **beta feature** — contact Clerk support or visit your dashboard to request access.

- `clerk_application` — Create and manage Clerk applications (supports logo/favicon upload)
- `clerk_domain` — Manage domains on a Clerk application
- `clerk_instance_config` — Manage instance configuration (auth settings, OAuth connections, etc.)

## Authentication

### Instance API key (required)

Set the `CLERK_API_KEY` environment variable to your Clerk **secret key** (starts with `sk_test_` or `sk_live_`):

```bash
export CLERK_API_KEY="sk_test_..."
```

Or configure it directly in the provider block:

```hcl
provider "clerk" {
  api_key = "sk_test_..."
}
```

### Platform API key (optional, beta)

If you need to manage `clerk_application`, `clerk_domain`, or `clerk_instance_config` resources, you also need a **Platform API key** (starts with `ak_`). This is a separate key from the instance secret key.

```bash
export CLERK_PLATFORM_API_KEY="ak_..."
```

Or configure it in the provider block:

```hcl
provider "clerk" {
  api_key          = "sk_test_..."
  platform_api_key = "ak_..."
}
```

> **Note:** The Platform API key must have the appropriate scopes enabled for each resource (e.g., `applications:manage`, `application_domains:manage`, `applications:manage` for instance config). Enable scopes in your Clerk dashboard under the Platform API key settings.

## Development

### Build

```bash
make build
```

### Run Acceptance Tests

Acceptance tests create real resources against the Clerk API. Both keys are required to run the full test suite.

```bash
export CLERK_API_KEY="sk_test_..."
export CLERK_PLATFORM_API_KEY="ak_..."
make testacc
```

### Local Installation

Add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "buildwithdeck/clerk" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

Then:

```bash
make install
```

### Generate Documentation

```bash
go generate ./...
```

## License

[Mozilla Public License v2.0](./LICENSE)
