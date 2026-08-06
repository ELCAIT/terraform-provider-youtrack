---
name: youtrack-client-usage
description: Use whenever implementing or reviewing a Create/Read/Update/Delete/data-source body in this provider — i.e. whenever the code needs to talk to YouTrack or Hub. Explains the hard rule that this provider must call github.com/elcait/youtrack-api-client exclusively (never raw HTTP, never reimplemented REST logic), how to use the YouTrack REST API / Hub REST API references to figure out which client method is actually correct, what the client does and doesn't support today, and what to do when a needed client method doesn't exist yet. Pair with the terraform-provider-conventions skill for the surrounding Go/framework shape.
---

# Calling YouTrack/Hub from this provider

## The hard rule

`internal/provider/**` (and `internal/provider/settings/**`) may talk to
YouTrack/Hub **only** through `*youtrack.Client` methods from
`github.com/elcait/youtrack-api-client`. That means:

- No `net/http` calls anywhere in this repo. No constructing URLs, setting
  headers, or parsing JSON responses by hand in `internal/provider`.
- No reimplementing something the client already does (pagination, `fields=`
  query building, `$type` discriminators, retry/fallback chains) — call the
  client method instead, even if it feels like less code to inline it.
- If the operation you need genuinely has no client method: **stop and say
  so** rather than working around it with a direct HTTP call. The fix belongs
  upstream, in `youtrack-api-client` (checked out locally at
  `/home/flos/sources/github/elcait/youtrack-api-client`, canonical repo
  `github.com/ELCAIT/youtrack-api-client`) — that repo has its own
  `youtrack-api-integration` and `youtrack-go-conventions` skills for adding a
  new method correctly. Bump this repo's `go.mod` requirement once the new
  client version ships. Don't add a local shim to avoid that round-trip.

This is a deliberate architecture boundary, not just tidiness: it's what lets
the provider stay a thin Terraform-shaped layer, and what keeps YouTrack/Hub
quirks (successor-on-delete, `$type`, fallback endpoints, async settle
delays) fixed in exactly one place instead of leaking into provider code.

## Why REST API knowledge is still required

The client's Go signatures don't fully hide the underlying REST semantics —
you still need to know what the actual endpoint does to pick the *right*
method and use it correctly:

- **YouTrack REST API reference**: https://www.jetbrains.com/help/youtrack/devportal/rest-api-reference.html
  — issues, projects, custom fields, bundles, issue link types, workflows,
  articles, project-scoped admin resources.
- **Hub REST API reference**: https://www.jetbrains.com/help/youtrack/devportal/hub-rest-api-reference.html
  — users, groups, roles, permissions, auth modules, global settings — the
  identity/authorization service YouTrack is built on.

Heuristic: an *issue-tracker concept* (project, issue, field, bundle,
workflow) is YouTrack REST; a *who-can-log-in-and-what-can-they-do concept*
(user, group, role, permission, auth module, license) is Hub REST, even when
YouTrack also exposes it under its own `api/...` path as a proxy. `*Client`
in the Go library hides this behind one struct — the split only shows up in
which base path a method's doc comment or backing `const` implies, not in
the method's package or receiver.

Concretely, REST knowledge is what tells you things like:
- `UpdateOAuth2AuthModule`/other Hub updates use **POST**, not PUT — matters
  if you're ever tempted to hand-roll a request instead of calling the method.
- Two client methods can look similar but hit different servers with
  different guarantees (a YouTrack proxy path vs. a direct Hub path) — using
  the wrong one can behave differently across YouTrack/Hub version
  combinations. See "Hub vs YouTrack duality" below.
- An `*UpdatePayload`/`*UpsertPayload` struct's pointer fields (`*bool`,
  `*string`) exist specifically to distinguish "leave unchanged" from
  "clear to zero value" — building the payload with the wrong field set
  (or from a non-pointer helper) silently changes behavior versus what the
  Terraform user asked for.

## What the client currently supports (v1.1.5) — check before assuming

| Area | API | Representative methods |
|---|---|---|
| Projects | YouTrack | `CreateProject`, `GetProject`, `UpdateProject`, `DeleteProject`, `GetProjectCustomFields`, `AddProjectCustomField`, `UpdateProjectCustomField`, `RemoveProjectCustomField`, `GetProjectTimeTrackingSettings`/`UpdateProjectTimeTrackingSettings` |
| Custom fields (global) | YouTrack | `GetCustomFieldByID`, `GetCustomFieldByName`, `CreateCustomField`, `UpdateCustomField`, `DeleteCustomField` |
| Bundles | YouTrack | `GetEnumBundleByID`/`ByName`, `CreateEnumBundle`, `UpdateEnumBundle`, `DeleteEnumBundle`; equivalent `*StateBundle*` set + `UpdateStateBundleValue` |
| Issue link types | YouTrack | `GetAllIssueLinkTypes`, `GetIssueLinkTypeByID`, `CreateIssueLinkType`, `UpdateIssueLinkType`, `DeleteIssueLinkType` |
| Global settings | YouTrack | Symmetric `Get*Settings`/`Update*Settings` per domain: Appearance, Global, Locale, MailServer, Rest, System, Backup, GlobalTimeTracking, WorkTime; `ListWorkItemTypes`/`CreateWorkItemType`/`UpdateWorkItemType`/`DeleteWorkItemType` |
| Roles & permissions | Hub | `GetAllPermissions`, `GetPermissionGraph`, `GetYoutrackRoleById`, `CreateYoutrackRole`, `UpdateYoutrackRole`, `DeleteYoutrackRole`, `GetAllAssignedRoles`, `GetAssignedRoleById`, `CreateAssignedRole`, `UpdateAssignedRole`, `DeleteAssignedRole` |
| Auth modules | Hub | `CreateOAuth2AuthModule`, `GetOAuth2AuthModuleByID`, `UpdateOAuth2AuthModule`, `DeleteOAuth2AuthModule` — **OAuth2 only**, no SAML/LDAP |
| Users & groups | Hub + YouTrack proxy | `ListUsers`/`ListGroups` (paginated), `GetUserByLogin`, `GetUserGroupByName`, `CreateGroup`/`GetGroupByID`/`UpdateGroup`/`GetAllUsersGroup`/`DeleteGroup(groupID, successorID)`, `CreateUser`/`UpdateUser`/`BanUser`/`DeleteUser`, `AddUserToGroup`/`RemoveUserFromGroup` |

**Not supported at all today: Issue CRUD** (no `CreateIssue`/`GetIssue`/etc.)
and non-OAuth2 auth modules. If a task implies either, that's a client gap,
not a provider bug — flag it per "The hard rule" above instead of attempting
a workaround.

Before relying on any signature, check this repo's pinned version in
`go.mod` (`github.com/elcait/youtrack-api-client vX.Y.Z`) against that
library's `CHANGELOG.md` — non-additive changes have shipped there without a
semver-major bump (an import-path rename, and `DeleteGroup` gaining a
required `successorID` parameter), so don't assume a method you remember from
an older version still has the same shape.

## Naming → REST verb (so you can predict what a method does before reading it)

- `CreateX(ctx, payload) (*X, error)` → POST
- `GetX`/`GetXByID`/`GetXByName` → GET single resource
- `ListX(ctx, top, skip int) ([]X, error)` → GET collection, OData-style
  `$top`/`$skip` pagination (pass `0, 0` for server defaults — not "return
  nothing"). `GetAllX(ctx) ([]X, error)` → GET collection, no pagination
  params (already fetches everything).
- `UpdateX(ctx, id, payload) (*X, error)` → **POST** for both YouTrack and
  Hub updates in this API family, not PUT.
- `DeleteX(ctx, id) error` → DELETE, already idempotent (404 treated as
  success internally for most delete methods) — your provider `Delete`
  should still check `youtrack.IsNotFoundError(err)` defensively (see
  `terraform-provider-conventions`), but you don't need to add your own
  idempotency layer on top.

## Error handling contract exposed by the client

- `youtrack.IsNotFoundError(err) bool` — generic 404 (`*HTTPError` with
  `StatusCode == 404`). Use this in `Read` to call `resp.State.RemoveResource`,
  and in `Delete` to treat "already gone" as success.
- `youtrack.IsEnumBundleNotFoundError` / `IsStateBundleNotFoundError` /
  `IsCustomFieldNotFoundError` — name/lookup misses that aren't a server 404
  (used by the `GetXByName` family). Don't conflate these with
  `IsNotFoundError`; a resource that looks up by name needs to check the
  matching name-specific predicate, not just the generic one.
- No typed 409/validation errors exist — anything else surfaces as
  `*HTTPError` with a raw `StatusCode`/`Body`. If you need to distinguish a
  conflict from a generic failure, inspect `HTTPError.StatusCode` yourself
  (`errors.As`) rather than assuming a helper exists for it.

## Hub vs YouTrack duality (for users/groups/roles/permissions)

There is one `*youtrack.Client`, not separate Hub/YouTrack clients — the
split is purely which base path a method's implementation targets
(`api/...` vs `hub/api/rest/...`), and for identity resources the client
sometimes tries several endpoint variants and falls through on 404/405
because YouTrack-version-to-Hub-proxy support varies (see `AddUserToGroup`,
`DeleteGroup` in the client source). Practical implications for the
provider:

- Don't assume two methods that "sound the same" hit the same server or
  guarantee the same behavior — read the doc comment/implementation if it
  matters for your resource's semantics (e.g. whether a change is visible
  immediately).
- `DeleteGroup` requires a `successorID` — Hub needs somewhere to put the
  group's members/ownership on delete. If your resource wraps group/user
  deletion, you must supply a sensible successor (don't hardcode a guess;
  look at how existing callers resolve one, e.g. via `GetUserByLogin`/
  `GetAllUsersGroup`) rather than leaving it to the caller by surprise.

## Workflow checklist for a new resource/attribute that needs an API call

1. Identify the owning REST resource via the two references above (fetch the
   specific resource's page, not the whole reference).
2. Search the local `youtrack-api-client` checkout (or its godoc) for an
   existing `Create/Get/Update/Delete/List` method covering it. Read the
   method's doc comment **and** the `const` block above it in the same file
   (path, `fields=` param) to confirm it actually returns/accepts what your
   Terraform attribute needs — a plausible-sounding method name is not proof
   it covers your case (e.g. it might be missing a field you need, forcing a
   client-side change anyway).
3. If it exists and is sufficient: call it directly from `Create`/`Read`/
   `Update`/`Delete` per `terraform-provider-conventions`. Do not add any
   HTTP logic locally.
4. If it's missing or insufficient (missing field, missing filter, wrong
   granularity): stop implementing the provider side. Report the gap — it
   needs to be added to `youtrack-api-client` first (see "The hard rule").
5. When mapping the response into Terraform state (`fromAPIModel`) or the
   request out of Terraform plan (`toAPIModel`), map field-by-field
   explicitly — REST/Hub JSON is camelCase (`shortName`, `ringId`,
   `fromEmail`) and won't match Terraform's snake_case attribute names
   one-to-one; don't assume a name match without checking the client struct's
   `json:` tags.
