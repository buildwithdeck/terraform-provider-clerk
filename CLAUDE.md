# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for [Clerk](https://clerk.com) identity management, built with the HashiCorp Terraform Plugin Framework (not SDK v2). Registry address: `registry.terraform.io/buildwithdeck/clerk`.

## Build & Test Commands

```bash
make build          # Build the provider
make install        # Build and install to $GOPATH/bin
make test           # Run unit tests
make testacc        # Run acceptance tests (requires CLERK_API_KEY and/or CLERK_PLATFORM_API_KEY)
make lint           # Run golangci-lint
make fmt            # Format code with gofmt
make generate       # Generate docs (go generate ./...)
```

Run a single test:
```bash
TF_ACC=1 go test -v -count=1 -run TestAccFooResource ./internal/provider/
```

Acceptance tests create real Clerk resources — both `CLERK_API_KEY` (instance secret key, `sk_test_...`) and `CLERK_PLATFORM_API_KEY` (platform key, `ak_...`) are needed for the full suite.

## Architecture

**Entry point**: `main.go` → registers provider via `providerserver.Serve` at `registry.terraform.io/buildwithdeck/clerk`.

**All provider code lives in `internal/provider/`**:
- `provider.go` — `ClerkProvider` struct: configures both API clients, passes `ProviderData` (containing both keys) to resources
- `platform_client.go` — `PlatformClient` struct: minimal HTTP client for the Platform API beta (`api.clerk.com/v1`)
- `resource_*.go` — one file per resource, each implementing full CRUD + import
- `*_test.go` — acceptance tests using `terraform-plugin-testing`

### Dual-API Architecture

The provider talks to two distinct Clerk APIs:

| API | Key env var | Client | Resources |
|-----|------------|--------|-----------|
| **Instance API** | `CLERK_API_KEY` | `clerk-sdk-go/v2` (global `clerkgo.SetKey()`) | `jwt_template`, `organization`, `application_settings`, `allowlist_identifier`, `blocklist_identifier`, `redirect_url`, `instance_domain` |
| **Platform API** (beta) | `CLERK_PLATFORM_API_KEY` | Custom `PlatformClient` (raw HTTP) | `application`, `domain`, `instance_config` |

Instance API resources use `clerkgo.SetKey()` globally + the SDK's typed request structs. Platform API resources use `PlatformClient` with manual JSON marshaling/HTTP calls.

In `Configure()`, Instance API resources extract `ProviderData.APIKey` and call `clerkgo.SetKey()`. Platform API resources extract `ProviderData.PlatformAPIKey` and create a `PlatformClient`.

## Resource Implementation Pattern

Each resource follows this structure:

1. **Struct**: `type FooResource struct { configured bool }` (Instance API) or `type FooResource struct { client *PlatformClient }` (Platform API) + a `FooResourceModel` with `tfsdk` tags
2. **Constructor**: `NewFooResource() resource.Resource`
3. **Interfaces**: `resource.Resource`, `resource.ResourceWithConfigure`, `resource.ResourceWithImportState`
4. **Configure**: validates provider data, sets up client access
5. **Schema**: uses `PlanModifiers` (`UseStateForUnknown`, `RequiresReplace`), `Sensitive: true` for secrets
6. **CRUD**: Create/Read/Update/Delete call Clerk API, map response via `mapFooResponseToModel()` helper
7. **ImportState**: uses `resource.ImportStatePassthroughID` with `path.Root("id")`

Key conventions:
- Optional fields: check `!plan.Field.IsNull() && !plan.Field.IsUnknown()` before including in API params
- JSON metadata fields: stored as strings, normalized via `normalizeJSON()` helper
- Timestamps: Clerk SDK returns milliseconds → converted to RFC3339 for Terraform state
- Write-only fields (e.g., `signing_key`): never populated from API reads, state preserved from plan
- Singleton resources (e.g., `application_settings`): use fixed ID, no ImportState, Delete just disables

## Dependencies

- Go 1.24.0+, Terraform >= 1.0
- `github.com/clerk/clerk-sdk-go/v2` — Clerk Instance API client
- `github.com/hashicorp/terraform-plugin-framework` — provider framework
- `github.com/hashicorp/terraform-plugin-testing` — test harness

## Adding a New Resource

1. Create `internal/provider/resource_<name>.go` with struct, model, CRUD methods, and response mapper
2. Register constructor in `provider.go` `Resources()` method
3. Create `internal/provider/resource_<name>_test.go` with acceptance tests
4. Add example in `examples/resources/clerk_<name>/resource.tf`
5. Run `make generate` to produce docs

## Local Development Setup

Add a dev override to `~/.terraformrc` to test locally built provider:
```hcl
provider_installation {
  dev_overrides {
    "buildwithdeck/clerk" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

Then run `make install` and use `terraform plan/apply` without `terraform init`.

## CI

- PR builds run acceptance tests via GitHub Actions (`.github/workflows/test.yml`)
- Tests use `CLERK_PLATFORM_API_KEY` from repository secrets; the workflow creates ephemeral Clerk resources

## Deck Engineering Conventions

Conventions and quality workflows are available via the `deck-patterns-and-practices` MCP server. Before starting work, fetch the latest guidance:

- **Branching, commits, PRs**: call `get_conventions` (format: `<type>/<ticket-id>_<short-description>`, e.g. `feature/PER-120_add-instance-domain`)
- **Security policy**: call `get_security_policy`
- **Code quality gate**: call `get_quality_workflow` — after implementing changes, run `aikido_full_scan` on every modified source file and fix all Critical/High issues before marking work complete
