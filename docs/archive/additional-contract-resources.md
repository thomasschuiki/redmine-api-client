# Plan: Additional contract resources

> **Status:** Done. All six resources committed and smoke-tested against a live
> Redmine 6.1.3 container; every commit passes strict validation (final spec:
> 37 paths / 84 ops / 76 schemas). Work was committed incrementally per resource
> (contract change → regenerate → strict-validate → regenerate snapshots → bump
> snapshot counts → commit).

## Current state (facts)

- `internal/models/models.go` — one generated file, 50 types
  (libopenapi `golang.WithOptionalFieldsAsPointers(true)`, `date-time` → `time.Time`).
- New resources are emitted only when their model is in `Contract::RESOURCES` (spec_builder.rb
  `contract_resource?`) and their controller is in `Introspector::CONTROLLER_MODEL_MAP`.
- Verified route/controller surface (from Redmine 6.1.3 source):

| Resource      | Controller      | API actions                          | Routes (REST)                                   |
|---------------|-----------------|--------------------------------------|-------------------------------------------------|
| Journals      | journals        | update only                          | none (HTML edit/update) — **excluded**          |
| Attachments   | attachments     | show, download, thumbnail, upload, update, destroy | GET/PUT/PATCH/DELETE `/attachments/{id}`, POST `/uploads` |
| News          | news            | index, show, create, update, destroy | `/news`, `/news/{id}`, `/projects/{id}/news`    |
| Memberships   | members         | index, show, create, update, destroy | `/projects/{id}/memberships`, `/memberships/{id}` |
| Groups        | groups          | index, show, create, update, destroy, add_users, remove_user | `/groups`, `/groups/{id}` |
| Queries       | queries         | index only                           | `/queries` (show excluded)                      |
| Custom fields | custom_fields   | index only                           | `/custom_fields`                                |

`Introspector::CONTROLLER_MODEL_MAP` already has all of these (as `Member`, `Group`,
`Query`, `CustomField`, `Attachment`, `News`) — but `members` currently maps to `Member`;
rename to `Membership` for contract naming consistency.

---

## Task 2 — Additional contract resources

> **Note:** Task 1 (split generated models into request/response files) was
> dropped. Go imports are package-level, not file-level, so splitting a single
> generated file within one package changes nothing for callers, and the
> post-processing step would add fragile surface for no functional benefit. The
> hand-written alias layer in `internal/client` already exposes exactly the
> types the client needs.

### Ordering (each lands as its own commit)

1. **Attachments** — highest CLI value; `POST /uploads` excluded from the spec (decision 5).
2. **Memberships** — simple, admin-relevant.
3. **News** — read + create.
4. **Groups** — read + create/update.
5. **Queries** (read-only) and **Custom fields** (read-only) — trivially small.

Journals: **excluded** — not a real REST resource (only `accept_api_auth :update`), already
covered via `include=journals` on issues.

> **Decisions (resolved 2026-08-10):**
> 1. **Empty `render_api_ok` responses** — measured against a live Redmine 6.1.3:
>    news create/update/destroy, members update/destroy, attachments update/destroy,
>    groups update/destroy reply **204 No Content with an empty body** (not "200 with
>    no body" as originally assumed). Create endpoints that DO render a body:
>    memberships → 201, groups → 201, attachments create (via upload) → 201.
>    Implemented via a per-resource contract `'render_ok' => %w[...]` list:
>    News (`create update`), Membership (`update`), Group (`update`),
>    Attachment (`update`). The builder emits **204 with no content** for render_ok
>    actions and DELETE; the CLI methods on those ops pass `target=nil` to
>    `decodeResponse`, and `news create`/`group update` print a confirmation line.
>    Pinned by `TestRenderOkWritesPinEmptyResponses`.
> 2. **Attachment download/thumbnail** — keep skipping them (not REST_ACTIONS); download is
>    served via `getAttachment`'s `content_url`. **Recorded** in the plugin README's
>    "Known Limitations".
> 3. **Scope** — full pass: contract + models + snapshots **and** CLI commands per resource.
> 4. **Show-op query params** — add per-resource `include`/`builtin` params on show ops
>    (`getGroup`, `getNews2`) via the `SHOW_QUERY_PARAMS` map in `spec_builder.rb`.
> 5. **`POST /uploads` excluded from the spec** — modeling it needs a binary-body special
>    case plus a non-REST action allowlist in `spec_builder.rb`, for little CLI benefit
>    (the client can raw-POST the file itself). **Recorded** in the plugin README's
>    "Known Limitations"; `redmine upload` stays, hand-written (no spec schema).
> 6. **News show keeps auto `getNews2`** — no `getNewsItem` refinement; numbered suffixes
>    are already the norm (`updateUser2`, `getWikiPage3`).
> 7. **Wire `accept_api_auth` filtering** — the introspector already collects
>    `accept_api_actions` (introspector.rb) but the builder never uses them.
>    `contract_resource?` now requires the action be in the controller's API-exposed
>    list. This makes queries/custom fields truly index-only **and** removes the
>    spurious create/update/delete ops the spec previously emitted for trackers,
>    issue_statuses and enumerations (all `accept_api_auth :index` only). Snapshot
>    counts dropped accordingly.

### Shared steps per resource
1. Add `RESOURCES` entry in `contract.rb` (schemas, `path_key`, `create`/`update` blocks).
2. Wire `Introspector::CONTROLLER_MODEL_MAP` (rename `members` → `Membership`).
3. Land the `accept_api_auth` filter first (decision 7); regenerate 6.1.3 → strict-validate
   → fix issues (dead components).
4. Regenerate all snapshots + `docs/openapi/openapi.yaml`; **bump the ops count** in
   `internal/spec/snapshot_test.go`; extend filter/resource assertions.
5. Add CLI commands in `internal/client` + `internal/redminecmd`; smoke-test.
6. Update READMEs (main + plugin: supported resources, new commands, excluded endpoints in
   "Known Limitations"). Commit.

### Builder changes (land with the first resource that needs them)
- **`accept_api_auth` filtering** (land first, decision 7): one condition in
  `contract_resource?` — `return false unless @introspector.api_controllers[route[:controller]].include?(route[:action])`.
  The introspector already records the per-controller API-exposed action list; it is
  currently dead data. This is what makes the read-only resources (queries, custom fields)
  actually emit index-only, and it cleans up the pre-existing leaks (trackers, issue_statuses,
  enumerations currently emit create/update/delete ops despite `accept_api_auth :index`).
- **Show-op query params**: `build_parameters` only emits params for index/list ops today.
  Extend it with a per-resource map (e.g. `API_PARAMS` keyed by controller+action) so show
  ops can carry `include` (`getGroup`, `getNews`) and index ops can carry `builtin`
  (`getGroups`).
- **No upload special case, no news refinement, no show-route exclusion** — see decisions
  5 and 6; the spurious `/attachments/{id}/{filename}` route is simply emitted as
  `getAttachment2` (consistent with the existing `updateUser2`/`getWikiPage2` practice).

### 1. Attachments

Schemas: `Attachment` (re-add — it was removed in Phase 3 as dead; now referenced),
`AttachmentSingle` (`{ attachment: Attachment }`), `AttachmentUpdate`/`AttachmentUpdateRequest`
(`{ attachment: { filename, description } }`). **No `Upload` schema** (decision 5).

`Attachment` shape (from `render_api_attachment_attributes`):
`{ id, filename, filesize, content_type, description, content_url, thumbnail_url?, author: IDName, created_on }`.
`content_url` is the authenticated download URL (`/attachments/download/{id}/{filename}`).

Operations:
- `getAttachment` (GET `/attachments/{id}` → `AttachmentSingle`)
- `getAttachment2` (GET `/attachments/{id}/{filename}` — the spurious `named_attachment`
  show route; same handler, emitted as-is, decision 6)
- `updateAttachment`/`updateAttachment2` (PUT/PATCH `/attachments/{id}` → **204 No Content**,
  render_ok, decision 1; body `AttachmentUpdateRequest` — matches
  `@attachment.safe_attributes = params[:attachment]`)
- `deleteAttachment` (DELETE `/attachments/{id}` → 204)

PUT and PATCH both emit an op, so updates appear as `*2` pairs — same as today's
`updateIssue2`/`updateVersion2`.

Excluded from the spec (recorded in plugin README "Known Limitations"):
- `POST /uploads` — upload requires a binary body + non-REST action; **decision 5**. The CLI
  implements it as a hand-written raw octet-stream POST (no spec schema) and returns the
  `{ upload: { id, token } }` token for use in issue create.
- `download`/`thumbnail` — not REST_ACTIONS; download is served via `getAttachment`'s
  `content_url` (decision 2).

Notes:
- QUERY_FILTERS: none (no list op).
- CLI: `redmine attachment get <id> [--output FILE]` (stream `content_url`),
  `redmine attachment update <id> --description D`, `redmine attachment delete <id>`,
  `redmine upload --file PATH [--filename F]` (raw POST, token output).

### 2. Memberships

Schemas: `Membership`, `MembershipSingle`, `MembershipList`, `MembershipRequest`,
`MembershipCreate` (`{ user_id, user_ids, role_ids }`), `MembershipUpdate`/`UpdateRequest`
(`{ role_ids }`).

`Membership` shape (verified from `members/show.api.rsb`): `{ id, project: IDName,
roles: [IDName], user?: IDName, group?: IDName }` — the principal key is **dynamic**
(`user` or `group`, via `principal.class.name.underscore`), so model both as optional.

Operations: `getProjectMemberships` (GET `/projects/{project_id}/memberships`),
`createProjectMembership` (POST same), `getMembership` (GET `/memberships/{id}`),
`updateMembership`/`updateMembership2` (PUT/PATCH `/memberships/{id}`),
`deleteMembership` (DELETE same).

Notes:
- Rename `CONTROLLER_MODEL_MAP['members']` → `'Membership'`.
- `project_id` is a path param on the nested routes (no query `project_id`).
- Index/list envelope **confirmed**: `api.array :memberships, api_meta(...)` → `total_count`/`offset`/`limit`.
- Create returns 201 `MembershipSingle`; update/destroy are `render_api_ok` → **204 No Content** (decision 1).
- CLI: `redmine membership list --project P`, `redmine membership add --project P --user U --roles 1,2`,
  `redmine membership remove <ID>`.

### 3. News

Schemas: `News`, `NewsSingle`, `NewsList`, `NewsRequest`, `NewsCreate`,
`NewsUpdate`/`NewsUpdateRequest`.

`News` shape (verified from `news/index.api.rsb` + `show.api.rsb`):
`{ id, project?: IDName, author?: IDName, title, summary?, description, created_on }`.

Operations: `getNews` (GET `/news`), `getNews2` (GET `/news/{id}` — show, auto-disambiguated,
decision 6), `createNews` (POST `/news`), `getProjectNews` (GET `/projects/{project_id}/news`),
`createProjectNews` (POST same), `updateNews`/`updateNews2` (PUT/PATCH `/news/{id}`),
`deleteNews` (DELETE `/news/{id}`).

Notes/risks:
- **operationId collision**: `News` is a mass noun, so `resource_noun` yields `News` for both
  index and show → `getNews` collides. `refined_operation_id` has no branch for it, so the
  fallback `disambiguate` yields **`getNews2`** — accepted (decision 6). Not a validation
  problem: the IDs are unique, and `checkDuplicateOperationIDs` passes.
- **`NewsRequest` needs `project_id`** — `find_project_by_project_id :only => :create` 404s
  `POST /news` without it. **Verified empirically against a live server**: the param must be
  **top-level** (`params[:project_id]`, the nested `news.project_id` 404s). Implemented as a
  generic contract `'top_level'` mechanism: `components()` merges the listed keys into the
  request schema next to the wrapper (`NewsCreate = { news, project_id }`), and the CLI
  `news create` writes `project_id` at the top level with `--project` required.
- Create/update/destroy are `render_api_ok` → **204 No Content** (decision 1).
- Index envelope: `api.array :news, api_meta(...)` → envelope true.
- Show supports `include=attachments|comments` → carried by `SHOW_QUERY_PARAMS`.
- CLI: `redmine news list [--project P]`, `redmine news get <ID>`, `redmine news create --project P --title T --description D`.
- **Actual op names** (measured): `getNews` (index), `getNews2` (show), `createNews`,
  `getProjectNews`, `createProjectNews`, `updateNews`/`updateNews2`, `deleteNews`.

### 4. Groups

Schemas: `Group`, `GroupSingle`, `GroupList`, `GroupRequest`, `GroupCreate`
(`{ name, user_ids }`), `GroupUpdate`/`GroupUpdateRequest` (`{ name, user_ids }`).

`Group` shape (verified from `groups/show.api.rsb`):
`{ id, name, builtin?, custom_fields: [CustomField], users?: [IDName], memberships?: [...] }`
— `users` only present with `include=users`, so model it optional and add the include param
to `getGroup`.

Operations: `getGroups` (GET `/groups`), `createGroup`, `getGroup`,
`updateGroup`/`updateGroup2` (PUT/PATCH), `deleteGroup`.

Notes:
- **Index has no envelope** (`api.array :groups` without `api_meta`) → `GroupList` is a plain
  array; index supports `builtin=1` (add as a query param).
- Create returns 201 `GroupSingle`; update/destroy are `render_api_ok` → **204 No Content** (decision 1).
- `add_users`/`remove_user` (POST/DELETE `/groups/{id}/users`, controller `groups` with
  `accept_api_auth`) — **deferred**; they'd need custom operationId refinement
  (`verb_for` returns the raw action → `add_usersGroup`) and no CLI command needs them yet.
- CLI: `redmine group list`, `redmine group get <ID> [--with-users]`, `redmine group create --name N`,
  `redmine group update <ID> --name N`, `redmine group delete <ID>`.

### 5. Queries (read-only)

Schemas: `Query`, `QueryList`.

`Query` shape (verified from `queries/index.api.rsb`): **only** `{ id, name, is_public,
project_id }` are actually emitted — no `filters`/`columns`/`sort_criteria`/`group_by`/
`visibility`. Keep the schema minimal: `id`, `name`, `project_id` (nullable), `is_public`.

Operations: `getQueries` (GET `/queries`) only. Routes for create/update/destroy exist
(`resources :queries, :except => [:show]`) but the controller is `accept_api_auth :index`
only — the **`accept_api_auth` filter (decision 7) is what keeps this to a single op**;
without it the builder would emit `createQuery`/`updateQuery`/`deleteQuery` with no request
schemas.

Notes:
- Plain users see only their own + public queries; admins see all.
- Index envelope: `api.array :queries, api_meta(...)` → envelope true.
- CLI: `redmine query list`; wire `--query ID` into `issue list` (already a valid filter).

### 6. Custom fields (read-only)

Schemas: `CustomFieldList` (`{ custom_fields: [CustomFieldDefinition] }`) — **do not reuse
SHARED `CustomField`**: that is the *value* shape (`id, name, value, multiple`) used inside
issue/project responses. The index returns field *definitions* — add `CustomFieldDefinition`
with `{ id, name, description, customized_type, field_format, is_required, is_filter,
searchable, multiple, visible, editable, default_value, possible_values, regexp, min_length,
max_length, trackers?, roles? }`.

Operations: `getCustomFieldDefinitions` (GET `/custom_fields`) only. Same situation as queries:
the controller is `accept_api_auth :index` only but
`resources :custom_fields, :except => :show` has create/update/destroy routes — the
**`accept_api_auth` filter (decision 7)** keeps this read-only.

Notes:
- **OperationId naming deviation**: the custom-fields op is `getCustomFieldDefinitions`
  and the schema is `CustomFieldDefinitionList` (not `getCustomFields`/`CustomFieldList`
  as originally written). The introspector model name for `custom_fields` is
  `CustomFieldDefinition`, and op names derive from the model name — the deviation keeps
  names consistent with the SHARED `CustomField` *value* shape. `getCustomFields` never
  existed.
- **No envelope** — `api.array :custom_fields` without `api_meta`; response is a plain array.
- High value for `issue create --custom-field ID=VALUE` ergonomics (discover IDs/names).
- CLI: `redmine custom-field list`.

### Expected impact
- Contract components: 50 → ~79. New types: Attachment (re-add), AttachmentSingle,
  AttachmentUpdate/UpdateRequest, Membership×7, News×7, Group×7, Query/QueryList,
  CustomFieldDefinition/CustomFieldList. **No `Upload`** (decision 5).
- Operations: 69 → ~84. Additions: attachments +5 (get, getAttachment2, update+update2,
  delete), memberships +6, news +8, groups +6, queries +1, custom fields +1 (= +27; PUT and
  PATCH each emit an op, hence the `*2` update pairs). The `accept_api_auth` filter
  (decision 7) then **removes ~12 existing ops** — the spurious create/update/delete ops
  for trackers, issue_statuses and enumerations (e.g. `createTracker`, `deleteIssueStatus`)
  — so the exact count comes from regeneration.
- `requiredSchemas` in `validate.go` grows (add `Attachment`, `AttachmentSingle`,
  `AttachmentUpdate`, `AttachmentUpdateRequest`, `Membership*`, `News*`, `Group*`, `Query`,
  `QueryList`, `CustomFieldDefinition`, `CustomFieldList`) to guard against silent drops.
- Gates to watch: `checkDeadComponents` (every new schema must be referenced — re-adding
  `Attachment` makes it live again; runs in strict mode only). `checkDuplicateOperationIDs`
  and `checkPayloadsHaveSchemas` run in **all** modes and already pass — no new blockers.

---

## Definition of done

- [x] `redmine-spec spec validate --strict` green after every regeneration.
- [x] All new resources strict-valid across 5.1.12 / 6.0.10 / 6.1.3 snapshots (regenerated,
      final counts: **37 paths / 84 ops / 76 schemas**; filters + new params asserted).
- [x] No dead components; no `info.version` mismatch.
- [x] `accept_api_auth` filter landed; tracker/issue_statuses/enumerations no longer leak
      create/update/delete ops.
- [x] CLI commands landed and smoke-tested for all six resources (attachment get/update/delete,
      upload as raw POST, membership list/add/remove, news list/get/create,
      group list/get/create/update/delete, query list, custom-field list, issue list --query).
- [x] Plugin README "Known Limitations" records the excluded endpoints (`POST /uploads`,
      attachment download/thumbnail).
- [x] READMEs list the new resources/commands; plan.md decision 4 marked resolved.
