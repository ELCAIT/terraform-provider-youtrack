---
name: terraform-provider-conventions
description: Use whenever adding, changing, or reviewing a resource, data source, or provider-level code under internal/provider (including the internal/provider/settings subpackage) in this repo. Covers OpenTofu/terraform-plugin-framework best practice, this repo's exact file layout and helper functions, schema/plan-modifier conventions, error-message naming, docs/examples generation, and testing requirements. Pair with the youtrack-client-usage skill for how to pick the correct youtrack-api-client call inside Create/Read/Update/Delete.
---

# Terraform provider conventions for terraform-provider-youtrack

This provider is built on `terraform-plugin-framework` (never the legacy
`terraform-plugin-sdk/v2` — `depguard` in `.golangci.yml` fails the build if
you import it) and must stay compatible with plain `terraform` **and**
OpenTofu (`tofu init/validate/plan/apply/import`). It talks to YouTrack/Hub
exclusively through `github.com/elcait/youtrack-api-client` — see the
`youtrack-client-usage` skill for that boundary and for choosing the right
client call. This skill is about everything around that call: schema,
CRUD shape, state handling, registration, docs, and tests.

## Architecture, in one page

- `internal/provider/provider.go`: `Configure` builds `*youtrack.Client` once
  from the `base_url`/`token` provider config (env fallback `TF_VAR_YOUTRACK_URL`
  / `TF_VAR_YOUTRACK_TOKEN`) and sets **both** `resp.DataSourceData` and
  `resp.ResourceData` to it. That's the only place the client is constructed.
- `Resources()` and `DataSources()` in `provider.go` return flat slices of
  `New*Resource`/`New*DataSource` constructor funcs. **Nothing is
  auto-discovered** — a new resource/data source is invisible to Terraform
  until you add its constructor to one of these two slices. Resources that
  live in the `settings` subpackage are referenced with a `settings.` prefix
  (e.g. `settings.NewSystemSettingsResource`).
- One file per resource: `x.go` (struct, model, `Metadata`/`Schema`/`Configure`/
  CRUD/`ImportState`) plus `x_helpers.go` (error-message constants,
  `toAPIModel`/`fromAPIModel`, business logic). Split further only when a
  resource is genuinely complex (`custom_field.go` + `custom_field_helpers.go`;
  `bundle_enum.go` + `bundle_schema_helpers.go` + `bundle_value_merge_helpers.go`).
  Follow this split for new resources instead of inventing a different layout.

## internal/helpers cheat sheet

`helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"` —
use these instead of hand-rolling the equivalent framework calls; they're what
every existing resource uses and what a reviewer will expect.

| Function | Use it for |
|---|---|
| `GetClientFromConfigure(req resource.ConfigureRequest, resp)` | Resource `Configure` — type-asserts `ProviderData` to `*youtrack.Client`, adds the standard diagnostics on mismatch. **Resource-only**; data sources don't have a matching helper (see below). |
| `GetPlanAndCheckError(ctx, req resource.CreateRequest, resp, &plan)` | Start of `Create` |
| `GetPlanAndCheckErrorUpdate(ctx, req resource.UpdateRequest, resp, &plan)` | Start of `Update` |
| `GetStateAndCheckError(ctx, req resource.ReadRequest, resp, &state)` | Start of `Read` |
| `GetStateAndCheckErrorDelete(ctx, req resource.DeleteRequest, resp, &state)` | Start of `Delete` |
| `SetStateAndCheckError(ctx, resp, state)` | End of Create/Read/Update/ImportState — type-switches on the response type, calls `.State.Set`, appends diagnostics |
| `HasResourceID(id types.String) bool` / `ValidateResourceID(id, &resp.Diagnostics, summary, detail) bool` | Guard Read/Update against a blank/null/unknown ID before calling the API |
| `ValidateEmailField(value, path.Root("field"), "description", &diagnostics)` | Config-time email validation (used from `ValidateConfig`) |
| `ListToStringSlice` / `SetToStringSlice` | Convert `types.List`/`types.Set` to `[]string` (Set variant dedups/trims/sorts) |
| `BoolFromOptional` / `OptionalBoolValue` | Convert `types.Bool` ↔ plain `bool` for optional fields |
| `StringFromOptional` (trims) / `OptionalStringValue` (raw) | Convert `types.String` → `string` for optional fields |
| `StringOrNull(s string) types.String` | API → state: collapse `""` to null. Use when the API's "unset" and "empty string" are the same thing. |
| `StringOrEmpty(s string) types.String` | API → state: keep `""` as a real value. Use when the API distinguishes null from empty string. |
| `HolderTypeOrDefault(value, fallback string) string` | Default a `$type`-style discriminator |
| `helpers.ErrCouldNotUpdateFmt` | `fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "role", err)` — shared Update error message format |
| `AssertFieldEqual(t, fieldName, got, want)` | **Test-only** — use in table-driven unit tests instead of a bare `if got != want` |

Picking `StringOrNull` vs `StringOrEmpty` wrong is a real drift bug: it makes
`Read` disagree with what `Create` just set, producing a perpetual diff. Check
what the API actually returns for "not set" before choosing.

## Error-message naming

Per resource, define in `x_helpers.go`:

```go
const (
	errCreatingX   = "Error creating x"
	errReadingX    = "Error reading x"
	errUpdatingX   = "Error updating x"  // or use helpers.ErrCouldNotUpdateFmt at the call site
	errDeletingX   = "Error deleting x"
	errMissingXID  = "Missing x ID"
	errXIDRequired = "X ID is required to read the x"
)
```

Always `resp.Diagnostics.AddError(errCreatingX, fmt.Sprintf("Could not create x: %v", err))`
(or the `helpers.ErrCouldNotUpdateFmt` form for updates). Don't invent a
different phrasing per resource — diagnostics summaries are part of the
provider's user-facing surface and consistency matters more than cleverness.

## CRUD skeleton

```go
var (
	_ resource.Resource                = &xResource{}
	_ resource.ResourceWithConfigure   = &xResource{}
	_ resource.ResourceWithImportState = &xResource{}
	// _ resource.ResourceWithValidateConfig = &xResource{} // only if you need config-time validation
)

func NewXResource() resource.Resource { return &xResource{} }

type xResource struct{ client *youtrack.Client }

func (r *xResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *xResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan xResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}
	apiModel := plan.toAPIModel()
	created, err := r.client.CreateX(ctx, apiModel) // see youtrack-client-usage for picking this call
	if err != nil {
		resp.Diagnostics.AddError(errCreatingX, fmt.Sprintf("Could not create x: %v", err))
		return
	}
	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

func (r *xResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state xResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}
	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingXID, errXIDRequired) {
		return
	}
	apiModel, err := r.client.GetX(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingX, fmt.Sprintf("Could not read x: %v", err))
		return
	}
	state.fromAPIModel(apiModel)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *xResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan xResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}
	updated, err := r.client.UpdateX(ctx, plan.ID.ValueString(), plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingX, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "x", err))
		return
	}
	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

func (r *xResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state xResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}
	if !helpers.HasResourceID(state.ID) {
		return
	}
	if err := r.client.DeleteX(ctx, state.ID.ValueString()); err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingX, fmt.Sprintf("Could not delete x: %v", err))
	}
	// No resp.State.RemoveResource call needed — the framework clears state
	// automatically when Delete returns without error diagnostics.
}

func (r *xResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Notable deviations you'll find and *why*, so you don't "fix" them into
inconsistency the other way:

- If the API never returns a field back (e.g. an OAuth2 client secret), the
  resource preserves it explicitly after `fromAPIModel` — see
  `oauth2AuthModuleResource.Create`/`Update` re-setting `plan.ClientSecret`
  from the payload it sent, and `Read` preserving it from prior state.
- If a field can only be set at creation (immutable), don't just skip it in
  `Update` — use a `stringplanmodifier.RequiresReplace()`/`boolplanmodifier.RequiresReplace()`
  plan modifier so Terraform proposes a replace instead of silently ignoring
  a user's changed value (see `project.go`'s `short_name`, `template`).
- If Read needs a fallback lookup because IDs can drift (e.g. bundle IDs),
  see `bundle_enum.go`'s retry-by-name-then-`IsEnumBundleNotFoundError`
  pattern before assuming a bare `GetXByID` + `IsNotFoundError` is enough.
- If `Update` must forbid changing a field, compare `req.State` (previous)
  against `plan` explicitly and `AddAttributeError` — see `custom_field.go`
  rejecting a `field_type_id` change.

## Data sources are simpler and deliberately different

`group_data_source.go` / `data_source.go` show the pattern: **do not** use
the `internal/helpers` Get/Set wrappers (they're typed for `resource.*`
requests) — call `req.Config.Get(ctx, &state)` and
`resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)` directly. `Configure`
duplicates the nil-check/type-assert inline against `datasource.ConfigureRequest`
(reusing `helpers.ErrUnexpectedResourceConfigType`/`helpers.ErrUnexpectedConfigureType`
for message consistency, since there's no data-source equivalent of
`GetClientFromConfigure`). A data source has one required identifying input
attribute (login/name/...) and the rest `Computed`.

## Schema conventions

- `id`: `Computed: true` + `stringplanmodifier.UseStateForUnknown()` — every
  resource, no exceptions, to avoid a perpetual diff on a server-generated ID.
- Immutable attribute → `stringplanmodifier.RequiresReplace()` /
  `boolplanmodifier.RequiresReplace()`, not a comment saying "don't change this".
  Attribute the API defaults/returns but the user may also set →
  `Optional: true, Computed: true` + `stringplanmodifier.UseStateForUnknown()`
  (and `booldefault.StaticBool(false)` where a concrete default reads better
  than "whatever the API happens to return").
  **Caution:** `boolplanmodifier.UseStateForUnknown()` does NOT protect a
  `Default`-bearing bool attribute from clobbering out-of-band changes —
  `TransformDefaults` runs before plan modifiers and overwrites the planned
  value from the raw config alone (null config → default), so by the time
  `UseStateForUnknown` runs the value is already known and its `IsUnknown()`
  guard never fires. If an `Optional+Computed+Default` bool represents
  something that can change outside Terraform (e.g. a Hub admin toggling a
  security setting directly), add a custom `planmodifier.Bool` that checks
  `req.State.Raw.IsNull()` (skip on create, let `Default` apply) and
  `req.ConfigValue.IsNull()` (preserve `req.StateValue` on update) — see
  `preserveStateWhenUnconfiguredBoolModifier` in `service.go`.
- Every `schema.Schema` and every attribute needs a `Description` — this also
  feeds `tfplugindocs` generation (see below), so a missing description is a
  missing line in the published docs, not just a lint nit.
- `Sensitive: true` for genuinely secret values (provider `token`, OAuth2
  `client_secret`, license fields) — sparingly, not defensively.
- **No `terraform-plugin-framework-validators` schema validators exist in this
  codebase.** Don't introduce them for a new resource. Semantic/cross-field
  validation (non-empty string, non-empty list, mutual exclusivity) goes in an
  optional `ResourceWithValidateConfig.ValidateConfig` method using
  `resp.Diagnostics.AddAttributeError(path.Root("field"), summary, detail)` —
  see `role.go`, `settings/system.go`, `settings/mail.go`.
- Nested structure: `schema.SingleNestedAttribute` for a single nested object
  (`custom_field.go`'s `field_defaults`), `schema.ListNestedAttribute` for a
  repeated block (`bundle_enum.go`'s `values`).

## Registering a new resource or data source

Adding the Go file is not enough. You must also:

1. Add `NewXResource`/`NewXDataSource` to `Resources()`/`DataSources()` in
   `internal/provider/provider.go` (prefix with `settings.` if it lives in
   that subpackage).
2. Add `examples/resources/youtrack_x/resource.tf` (a realistic example) and
   `examples/resources/youtrack_x/import.sh` (a `terraform import` line —
   explain the ID format in a comment if it's not obvious, especially for
   singleton `settings` resources) — or the `data-sources/` equivalents for a
   data source.
3. Run `make generate` (runs `terraform fmt` on `examples/` then
   `tfplugindocs generate`, producing `docs/resources/x.md` /
   `docs/data-sources/x.md` from the schema `Description`s + the example
   files) and commit the result. CI's `generate` job fails the PR on any
   uncommitted diff here — this is not optional cleanup.

## Testing requirements

- **Unit tests** (`x_test.go`): pure Go, table-driven, no framework harness —
  test `toAPIModel`/`fromAPIModel` and any standalone business logic (e.g.
  `role_test.go`'s `reconcileRolePermissions`). Use `helpers.AssertFieldEqual(t, "field", got, want)`
  or `reflect.DeepEqual` for structs/slices.
- **Acceptance tests** (`x_acc_test.go`): use `terraform-plugin-testing`
  (never the legacy SDK — banned by `depguard`). Reuse the shared helpers in
  `acc_test_helpers_test.go`: `skipUnlessAcc(t)` (skips unless `TF_ACC=1` and
  `YOUTRACK_URL`/`YOUTRACK_TOKEN` are set), `testProviderFactories()`,
  `providerBlock()` (prepend to every test's HCL), `importStateID(resourceName)`
  for `ImportStateIdFunc`. `settings/` has its own equivalents since those
  resources are singletons and use `ImportStateId: "global"` instead.
  Randomize names with `time.Now().UnixMilli()` to avoid collisions across
  parallel runs; cover create, an update step, and an import step
  (`ImportStateVerify: true`, with `ImportStateVerifyIgnore` for fields the
  API never returns, like a write-only secret).
- CI (`go test ... ./...`) runs both kinds together — acceptance tests
  self-skip in CI because the env vars aren't set there, so `skipUnlessAcc`
  is load-bearing, not decorative; don't remove it from a new acc test file.

## CI/lint gates you must pass

- `gofmt -l .` empty, `go vet ./...` clean, `golangci-lint run` clean
  (enabled: `copyloopvar, depguard, durationcheck, errcheck, forcetypeassert,
  gosec, ineffassign, makezero, misspell, nilerr, predeclared, staticcheck,
  unconvert, unparam, unused`). `depguard` specifically bans any
  `terraform-plugin-sdk/v2` import — if you find yourself reaching for one,
  you're using the wrong package (`terraform-plugin-framework` /
  `terraform-plugin-testing` have the equivalent).
- `make generate` must produce no diff (docs/examples in sync).
- Pre-commit also runs `terraform_fmt`/`terraform_tflint` against `examples/`
  — a broken or non-idiomatic example `.tf` file fails that hook.
- `GNUmakefile` default target is `fmt lint install generate` — run `make`
  before considering a change done.

## Code quality: complexity and duplication (not mechanically gated — review for it deliberately)

`.github/copilot-instructions.md` and `.github/instructions/go.instructions.md`
require **cognitive complexity ≤ 15 per function** and **no duplicated code or
string literals** (extract repeated literals — error messages, attribute
names, format strings — to a constant; factor out repeated logic into a
helper instead of copy-pasting it into the next resource). `.golangci.yml`
does **not** enable `gocognit`/`cyclop`/`dupl`/`goconst` — `golangci-lint run`
passing does **not** mean these two rules are satisfied. Treat them as a
manual review checklist on every new/changed function, not something CI will
catch for you:

- Prefer early returns over nested conditionals; prefer a data-driven loop
  (slice of cases) over a long `if`/`else if` chain — this repo's fallback
  patterns (e.g. `bundle_enum.go`'s retry-by-name-then-not-found) are close to
  the ceiling already, so add a new case as a data entry, not a new branch.
  If a `Create`/`Update`/`Read` body is accrediting nested branching (e.g. one
  more special-case field, one more fallback), extract the branch into a
  named helper in the resource's `_helpers.go` file rather than growing the
  method in place.
- Before adding a new error-message string, schema attribute name, or
  path/format string, grep for it first — this repo already extracts these to
  named constants per resource (`errCreatingX`, `errReadingX`, ...); add to
  that block instead of inlining a literal a second time.
- Before writing a second `toAPIModel`/`fromAPIModel`-shaped conversion, a
  second "validate ID is non-blank" check, or a second reconciliation loop
  that looks like an existing one (e.g. `role.go`'s permission
  reconciliation), check whether `internal/helpers` already has it or whether
  it should move there — don't let the same shape of logic reappear
  per-resource when it's actually generic.

There is also `.github/instructions/terraform-provider.instructions.md`, a
generic (non-repo-specific) OpenTofu/framework best-practices checklist fed to
Copilot. It's a reasonable sanity check but doesn't know this repo's actual
helper names, file layout, or generation flow — treat *this* skill as the
concrete operationalization of it for this codebase.
