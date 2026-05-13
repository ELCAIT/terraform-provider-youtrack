---
applyTo: "**/*.go"
---

- The should be compatible opentofu

# Terraform Provider Development Rules

## Forbidden Patterns
- Do not use fmt.Println or log.Printf
- Do not ignore diagnostics
- Do not hardcode API defaults
- Do not assume resource existence without checking
- Do not mutate plan values directly

## Framework
- Use Terraform Plugin Framework, not the legacy SDK
- Follow resource, data source, and provider separation strictly
- Use types from framework (types.String, types.Int64, etc.)
- Always check for null and unknown values explicitly
- Use req.Plan.Get and resp.State.Set correctly
- Do not assume values are known during planning

## Architecture Rules
- Terraform logic must not contain raw API logic
- API client must be abstracted behind a service layer
- No direct HTTP calls inside resource methods

## Schema Design
- Always define explicit schema with clear types
- Use Required, Optional, Computed correctly:
  - Required: user must provide
  - Optional: user may provide
  - Computed: set by API
- Avoid ambiguous combinations (e.g., Optional + Computed unless necessary)
- Add descriptions for all attributes

## Provider usage
- Generated provider code must be compatible with:
  - terraform init
  - terraform validate
  - terraform plan
  - terraform apply
  - terraform import

## Resource Implementation
- Implement all required methods:
  - Metadata
  - Schema
  - Create
  - Read
  - Update
  - Delete

- Ensure Create sets a stable ID
- Read must fully sync Terraform state with remote API
- Delete must handle already-deleted resources gracefully

## Plan & State Handling
- Never introduce non-deterministic values in plan
- Use plan modifiers when needed
- Preserve unknown values correctly
- Do not overwrite user-defined values unintentionally
- Never generate values in Create that were not present in the plan
- Ensure all Computed fields are predictable or explicitly unknown during plan
- Do not introduce diffs between plan and apply
- Use plan modifiers to stabilize values when required

## State Management Rules
- Read must always fully reflect remote system state
- Never leave stale values in state
- Ensure all attributes round-trip correctly
- Normalize API responses before storing in state

## Diagnostics
- Never panic
- Always return diagnostics.Diagnostics for errors
- Provide meaningful, user-facing error messages

## API Interaction
- Isolate API client logic from Terraform logic
- Do not call APIs in Schema methods
- Handle retries and transient errors properly

## Idempotency
- Ensure all operations are idempotent
- Re-running Apply should not cause unintended changes

## Imports
- Implement ImportState when applicable

## Testing
- Always write unit tests for internal/provider code, even if not explicitly requested
- Always write acceptance tests for provider code, even if not explicitly requested
- Prefer acceptance tests using Terraform testing framework
- Mock external APIs when unit testing
- Always generate acceptance tests for new resources
- Tests must cover:
  - create
  - update
  - import
  - destroy
- Use randomized names to avoid collisions
- Ensure tests can run in parallel safely

## Logging
- Use framework logging utilities
- Do not use fmt.Println for debugging

## Performance
- Avoid unnecessary API calls in Read
- Cache when safe, but never break correctness

## Security
- Never log secrets
- Mark sensitive fields appropriately in schema

## Critical Behavior Rules
- Never assume API defaults — always reflect them in state
- Avoid drift: Read must detect and reconcile all changes
- Do not suppress diffs unless explicitly justified
- Ensure all attributes round-trip correctly (plan → apply → state)

## Code quality
- Ensure code passes golangci-lint
- Follow Terraform best practices compatible with tflint and tfsec
