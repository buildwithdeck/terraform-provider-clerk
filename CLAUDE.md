# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for [Clerk](https://clerk.com) identity management, built with the HashiCorp Terraform Plugin Framework (not SDK v2). Registry address: `registry.terraform.io/buildwithdeck/clerk`.

## Architecture

**Entry point**: `main.go` → registers provider via `providerserver.Serve` at `registry.terraform.io/buildwithdeck/clerk`.

**All provider code lives in `internal/provider/`**:
- `provider.go` — `ClerkProvider` struct: configures Clerk SDK globally via `clerkgo.SetKey()`, registers all resources
- `resource_*.go` — one file per resource, each implementing full CRUD + import
- `*_test.go` — acceptance tests using `terraform-plugin-testing`

## Resource Implementation Pattern

Each resource follows this structure:

1. **Struct**: `type FooResource struct { configured bool }` + a `FooResourceModel` with `tfsdk` tags
2. **Constructor**: `NewFooResource() resource.Resource`
3. **Interfaces**: `resource.Resource`, `resource.ResourceWithConfigure`, `resource.ResourceWithImportState`
4. **Configure**: validates provider data (API key string), sets `configured = true`
5. **Schema**: uses `PlanModifiers` (`UseStateForUnknown`, `RequiresReplace`), `Sensitive: true` for secrets
6. **CRUD**: Create/Read/Update/Delete call Clerk SDK, map response via `mapFooResponseToModel()` helper
7. **ImportState**: uses `resource.ImportStatePassthroughID` with `path.Root("id")`

Key conventions:
- Optional fields: check `!plan.Field.IsNull() && !plan.Field.IsUnknown()` before including in API params
- JSON metadata fields: stored as strings, normalized via `normalizeJSON()` helper
- Timestamps: Clerk SDK returns milliseconds → converted to RFC3339 for Terraform state
- Write-only fields (e.g., `signing_key`): never populated from API reads, state preserved from plan
- Singleton resources (e.g., `application_settings`): use fixed ID, no ImportState, Delete just disables

## Dependencies

- Go 1.24.0+, Terraform >= 1.0
- `github.com/clerk/clerk-sdk-go/v2` — Clerk API client
- `github.com/hashicorp/terraform-plugin-framework` — provider framework
- `github.com/hashicorp/terraform-plugin-testing` — test harness

## Adding a New Resource

1. Create `internal/provider/resource_<name>.go` with struct, model, CRUD methods, and response mapper
2. Register constructor in `provider.go` `Resources()` method
3. Create `internal/provider/resource_<name>_test.go` with acceptance tests
4. Add example in `examples/resources/clerk_<name>/resource.tf`
5. Run `make generate` to produce docs

## Code of Conduct
Found in  `/conventions`
Contains Patterns and Practices about what claude needs to know for Branch Naming, Commits and Pull Requests at Deck.
