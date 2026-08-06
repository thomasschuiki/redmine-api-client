# Plan: Rework `redmine_openapi_generator` to emit a consumable OpenAPI spec

> **Status:** Phases 0–5 done. Phase 2 remaining per-resource query filters and
> open decision 4 (split request/response models) are the only outstanding items.

## Why

The plugin's spec output is currently unusable for code generation. It models the
*database*, not the *API contract*:

1. **No response schemas** — every 2xx is `{ description: Success }` with no body
   (`spec_builder.rb:159`). A client for a read-heavy CLI cannot be generated.
2. **Schemas are ActiveRecord columns** (`introspector.rb:94`) — request bodies leak
   DB internals (`hashed_password`, `salt`, `twofa_totp_key`, `lft`, `rgt`,
   `lock_version`) and can never express nested objects, wrappers, or custom-field
   values.
3. **Path/operation errors** — all path params typed `integer` (projects/wikis use
   string identifiers), legacy noise routes (`/issues/changes()`, `/issues/{id}/quoted()`,
   `/issues/bulk_edit()`), duplicate `operationId`s (e.g. `newIssue` twice).
4. **No contract tests** — route snapshots were removed (`e3224ab`); CI only greps for
   `openapi:`/`paths:` (`release-plugin.yml:67`), so none of this is caught.

The payoff of fixing it: the OpenAPI spec becomes the single source of truth and
`internal/models` output becomes the actual types used by `internal/client`,
replacing the hand-written request/response structs. (The line references in
the bullets above describe the pre-rework code.)

## Design

Replace "introspect DB columns" with "emit an explicit per-resource contract".
Redmine's API is stable, so a declarative contract map in the plugin is the
deterministic source of truth; live API snapshots (per Redmine version) validate it
for drift. Ground truth for the contract = Redmine's actual API + the working
hand-written shapes already in `internal/client/*.go`.

### Target spec shape

- **Request bodies**: wrapper objects with explicit fields, e.g.
  `{"issue": {project_id, subject, tracker_id, status_id, priority_id, ...}}`
- **Single responses**: wrapped, e.g. `{"issue": {id, subject, ..., project: {id,name}, custom_fields: [...]}}`
- **List responses**: envelopes, e.g. `{"issues": [...], "total_count": N, "offset": N, "limit": N}`
- **Shared components**: `IDName {id,name}`, `CustomFieldValue {id,name,value}`,
  `Journal`, `JournalDetail`, `Relation`, `Attachment`, `Watcher`, `Error`
- **No DB-internal fields anywhere**

## Phases

### Phase 0 — Define the contract (source of truth) — ✅ DONE

- Enumerate the resources to cover, starting with those the CLI uses
  (`internal/client/*.go`): issues, projects, users, time entries, wiki pages,
  versions, trackers, statuses, priorities, categories, relations.
  Expand later to journals, attachments, news, queries, groups, memberships.
- **Watchers are deliberately excluded** — the API takes a flat `{"user_id": N}`
  body with no response schema, so no contract fits.
- For each resource, specify in the plugin a Ruby data structure (in a new file,
  e.g. `lib/redmine_openapi_generator/contract.rb`):
  - request fields (+ required set, excluded DB-internal fields)
  - single-response fields + nested `$ref`s
  - list envelope on/off
- Define shared component schemas once (see "Target spec shape").
- Acceptance: every shape above derives from the working `internal/client` types.

Implemented: `contract.rb` — 11 resources in `RESOURCES` (Issue, Project, User,
TimeEntry, WikiPage, Version, Tracker, IssueStatus, IssuePriority, IssueCategory,
Relation + their request/single/list schemas) plus 8 SHARED components (IDName,
IssueRef, Journal, JournalDetail, CustomField, CustomFieldValue, Attachment,
Error); Issue also defines an `'update'` block emitting
`IssueUpdate`/`IssueUpdateRequest`. 51 schemas total (43 resource + 8 shared).
All required sets match the client's request structs.

### Phase 1 — Rework schema generation (core change) — ✅ DONE

Files: `introspector.rb`, `spec_builder.rb`, new `contract.rb`.

- Remove DB column collection for schemas (`collect_models`); keep route/controller
  discovery.
- `SpecBuilder#schemas` builds component schemas from `contract.rb`, not columns.
- Emit `IDName`-style nested refs, `CustomFieldValue`, envelope schemas.
- Keep the AR→OpenAPI scalar mapping but apply it to contract fields.
- Request schemas get `required` lists; `custom_fields` gets a proper value-assignment
  shape (not the metadata shape the DB `custom_fields` table implies).
- Acceptance: no `hashed_password`/`salt`/`lft`/`rgt`/`lock_version` anywhere in the
  spec; every resource has request + single + list schemas.

### Phase 2 — Fix paths and operations — ✅ DONE (except per-resource query filters)

File: `spec_builder.rb`.

- Path params: `{id}` stays integer for issues/users/etc.; projects, wiki, versions
  use string `{identifier}` / `{title}`.
- Filter legacy/HTML route noise so only true REST API resources are emitted.
- Deduplicate `operationId`s (fix `operation_id`, e.g. `newIssue` vs `createIssue`).
- Correct status codes and attach body schemas: 200 list/single, 201 create, 204
  delete; add 404 (not found) and 422 (validation) with `Error` schema.
- Pagination query params on all list ops; add per-resource query filters
  (status_id, sort, etc. for issues). — **REMAINING: per-resource filters are
  not emitted yet**; only `offset`/`limit` pagination params exist.
- Acceptance: `redmine-spec validate` reports correct op counts; no duplicate
  operationIds; every op has a schema'd response.

Implemented: route filter = REST_ACTIONS + contract resource + path-param
requirement for object ops (kills `/issues`, `/time_entries/destroy`, form
routes); `normalize_path` strips `(.:format)`/`.:format`; `path_param_type`
(integer `*_id`, string identifier/title/other); refined `operationId`s
(`getCurrentUser`, `getProjectRoadmap`) + numeric-suffix dedupe; 200/201/204 by
verb, 404/422 error responses with `Error` schema.

### Phase 3 — Strengthen `redmine-spec` tooling — ✅ DONE

Files: `internal/spec/validate.go`, `internal/cmd/app.go`.

- Add quality gates to `Validate` (or a new `--strict` mode):
  - every 2xx response has a body schema
  - no banned DB-internal fields
  - no duplicate operationIds
  - every schema is referenced (no dead components)
- Update the `validate` command help/output to report gate results.
- Acceptance: the reworked spec passes strict validation; the old one fails on
  clearly-named gates (response-schema missing, DB-internal fields).

Implemented gates (via `gopkg.in/yaml.v3` content checks): `$ref` pointer
integrity, banned-field scan (`hashed_password`, `salt`, `lft`, `rgt`,
`lock_version`, `type`, `password_digest`, `auth_source_id`), required-schema
presence, operationId uniqueness, and requestBody + 200/201 schema presence
(delete 204 exempt). `spec validate` prints Paths/Operations/Schemas/… and
fails on ContentErrors. **`--strict` flag added** (`internal/cmd/app.go`);
`ValidateStrict` runs `checkDeadComponents` (any component schema unreferenced
by a `$ref`). Flagging the pre-rework spec: `Attachment` + `IssueUpdateRequest`;
wiring PUT/PATCH to `{Name}UpdateRequest` + removing the dead `Attachment`
entry made the regenerated spec strict-clean. Verified: old spec fails (exit 1),
reworked spec passes (29 paths / 69 ops / 50 schemas).

### Phase 4 — Regenerate and adopt generated types in the client (payoff) — ✅ DONE

- Regenerate `docs/openapi/openapi.yaml` via `scripts/gen-openapi-spec.sh`. — ✅
  committed at `docs/openapi/openapi.yaml` (2789 lines, 6.1.3).
- Regenerate `internal/models/models.go` (`redmine-spec generate-models`). — ✅
  51 types, replaces the 75 DB-derived types. Reproducible: regenerating from
  the committed spec yields byte-identical output.
- Adopt in `internal/client`:
  1. Request bodies first (low risk): embed generated request schemas in
     `IssueCreateRequest`, `IssueUpdateRequest`, `ProjectCreateRequest`,
     `TimeEntryCreateRequest`, `RelationCreateRequest`, watcher bodies.
     — ✅ done for the five bodies + `CustomFieldValue` (aliased to `internal/models`);
     call sites in `internal/redminecmd` updated. **Watcher bodies left as-is** —
     no schema exists for them.
  2. Response types: replace hand-written structs with generated ones for covered
     resources; keep only transport/pagination/error helpers hand-written.
     — ✅ All covered responses aliased to `internal/models` (Issue/IssueList,
     Project/ProjectList, User/UserList, TimeEntry/TimeEntryList, WikiPage/
     WikiPageList, Version/VersionList, Tracker, IssueStatus, IssuePriority,
     IssueCategory, Relation + their list envelopes, IDName, Journal, CustomField).
     Kept hand-written: `Client`, `ListOpts`, `RelationType`+consts+`ParseRelationType`,
     `RequestError`/`APIError`/`DecodeError`, and helpers. Kept the generator's
     pointer-heavy config (`WithOptionalFieldsAsPointers(true)`); nil-safe derefs
     added where display code reads `IsDefault`/`IsClosed`/`Description`/`Notes`/`TotalCount`.
- Run `go build ./...`, `go vet ./...`, `go test ./...`; CLI smoke test.
  — ✅ build/vet/test green. ✅ CLI smoke test against a live Redmine 6.1.3
  (docker: `redmine:latest` + `mysql:8.0`): project create/list, issue
  create/get/update/delete/list (with `--contains`/`--top`/`--output json`),
  time-entry create/list/delete, tracker/status/priority list, version/wiki/
  category list, user current, issue grep. Two pre-existing bugs surfaced and
  fixed: (a) non-2xx responses (404/422/403) were silently decoded as success —
  `decodeResponse` now returns `APIError` for non-2xx (also fixes
  `ResolveProject` 404-suggestion); (b) `--project` was never wired into
  `project_id` for issue create and time-entry create — now resolved via
  `ResolveProject`. Wire shapes verified (issue, relation, time-entry, project,
  update, custom-field single/multi).
- Acceptance: build/vet/tests green with no hand-written duplicates for covered
  resources; CLI commands unchanged behavior. — ✅

### Phase 5 — CI, snapshots, release wiring — ✅ DONE

- Restore snapshot tests removed in `e3224ab`: snapshot the generated spec per
  Redmine version (5.1.12, 6.0.10, 6.1.3) and diff to catch drift.
- `release-plugin.yml`: replace the grep "validation" with running
  `redmine-spec validate --strict` on each matrix output; assert no DB-internal
  fields; regenerate + commit spec/model on spec-affecting plugin changes.
- `release-cli.yml`: regenerate models from the committed spec; fail build if
  committed `models.go` drifts.
- Update plugin `README.md` (replace "Known Limitations" with the new contract model).
- Acceptance: CI enforces the gates and regenerates artifacts on change.

Implemented: `scripts/gen-spec-snapshots.sh` (per-version generation loop) +
committed `testdata/openapi/redmine-{5.1.12,6.0.10,6.1.3}.yaml`. Generated all
three via the Docker harness — every version strict-valid, **identical contract
surface (29 paths / 69 ops / 50 schemas)**; snapshots differ only in
`info.version`. `redmine-6.1.3.yaml` is byte-identical to the bundled
`docs/openapi/openapi.yaml` (regeneration is deterministic). New
`internal/spec/snapshot_test.go` asserts strict validity + exact per-version
path/op/schema counts and that `info.version` carries the right Redmine version.
`release-plugin.yml`: test-plugin now strict-validates each matrix output and
diff-gates it against the committed snapshot; new `sync-generated` job (plugin
tags only) reruns the snapshot script, syncs `docs/openapi/openapi.yaml` +
`internal/models/models.go`, and commits if changed (GITHUB_TOKEN pushes do not
re-trigger workflows → no loops). `release-cli.yml`: strict-validates the
committed contract and regenerates models to prove no drift from `models.go`.
READMEs rewritten for the contract model (contract.rb source of truth,
PUT/PATCH → `{Name}UpdateRequest`, snapshot workflow).

## Files touched

- `plugin/redmine_openapi_generator/lib/redmine_openapi_generator/contract.rb` (new)
- `plugin/redmine_openapi_generator/lib/redmine_openapi_generator/spec_builder.rb`
- `plugin/redmine_openapi_generator/lib/redmine_openapi_generator/introspector.rb`
- `internal/spec/validate.go`, `internal/cmd/app.go`
- `docs/openapi/openapi.yaml`, `internal/models/models.go` (regenerated)
- `internal/client/*.go` (adopt generated types)
- `scripts/gen-openapi-spec.sh`, `.github/workflows/release-plugin.yml`,
  `.github/workflows/release-cli.yml`
- `testdata/` (new spec snapshots), plugin `README.md`

## Definition of done

- [x] Spec is a faithful API contract: wrappers, envelopes, nested refs, custom
      field values, no DB internals, correct paths/params/statuses, unique operationIds.
- [x] `redmine-spec validate --strict` passes with the gates from Phase 3.
      (old spec fails on dead components; reworked spec is strict-clean)
- [x] `redmine-spec generate-models` output is consumed by `internal/client` for
      covered resources; `go build/vet/test` green; CLI behaves identically.
      (all request + response types aliased; only transport/pagination/error
      helpers hand-written)
- [x] Spec is reproducible per Redmine version (snapshot tests in CI).
- [x] README reflects the new contract model.

## Open decisions

1. **Scope of resource coverage** — start CLI-scoped (Phase 0 list) or go for all 22
   mapped models? Recommended: CLI-scoped first, expand after adoption works.
   — ✅ **Resolved:** CLI-scoped; 11 contract resources (watchers excluded — no
   request/response schema exists for them).
2. **Declarative contract vs. live response inference** — plan assumes declarative
   contract + snapshot validation. If live inference (seed data + call API in the
   Docker harness) is preferred, it should be prototyped in Phase 1 as an alternative
   schema source. Recommended: declarative (stable API, deterministic output).
   — ✅ **Resolved:** declarative contract in `contract.rb`.
3. **Generator config** — keep `WithOptionalFieldsAsPointers(true)` (pointer-heavy
   output) or evaluate per-type request/response generation for ergonomics.
   — ✅ **Resolved:** keep pointer-heavy. Faithful presence semantics + `omitempty`
   keeps absent fields out of output (non-pointer `time.Time` would render
   `0001-01-01T00:00:00Z`); display ripple was small (~6 field reads, now
   nil-safe derefs). Verified in the live smoke test.
4. **Split request vs response models** — consider two generated files (request/
   response) so the client imports only what it needs.
   — **Open.**
