# redmine_openapi_generator

A Redmine plugin that auto-generates an OpenAPI 3.0.3 specification for a running
Redmine instance's REST API. Payload shapes come from a curated contract
(`contract.rb`) rather than database-column introspection, which keeps the spec
free of persistence internals and gives every operation real request/response
schemas.

## Why

Redmine ships with a REST API but no machine-readable spec. This plugin fills
that gap by running inside Rails and reading the source of truth directly: the
routing table and the `accept_api_auth` declarations. The curated contract
provides the JSON shapes. The output feeds into the `redmine-spec` toolchain for
strict validation, snapshot tests, and Go model generation.

## How It Works

The plugin has two data-collection phases plus a static contract:

1. **Routes** — iterates `Rails.application.routes.routes` to discover every
   HTTP verb, path template, controller, and action.

2. **API controllers** — finds all `ActionController::Base` descendants that call
   `accept_api_auth` and records which actions are API-exposed.

3. **Contract** — `contract.rb` is the source of truth for payload shapes. It
   defines component schemas (`Issue`, `IssueRequest`, `IDName`, …) as OpenAPI
   3.0 schema objects. `SpecBuilder` emits schemas for the resources the contract
   covers and omits the rest.

`SpecBuilder` then assembles a flat OpenAPI 3.0.3 document: paths with
parameters, operation summaries and IDs, request bodies referencing component
schemas, response schemas, common error responses (401/403/412), and security
schemes (Basic, API key header, API key query, OAuth2).

## Plugin Structure

```
redmine_openapi_generator/
  init.rb                                  Plugin registration
  generate_spec.rb                         Standalone entry point for Docker usage
  config/
    routes.rb                              Mounts GET /openapi/spec
  app/
    controllers/
      openapi_spec_controller.rb           HTTP endpoint (accept_api_auth protected)
  lib/
    redmine_openapi_generator/
      introspector.rb                      Data collection (routes, controllers)
      contract.rb                          Curated component schemas (source of truth)
      spec_builder.rb                      OpenAPI assembly
    tasks/
      openapi.rake                         Rake task for in-process generation
```

## Usage

### Via Docker (primary workflow)

The `gen-openapi-spec.sh` script handles everything automatically: it starts a
MySQL container, runs Redmine migrations, injects the plugin, runs the
introspector, and copies the output out.

```bash
./scripts/gen-openapi-spec.sh [redmine_version] [output_path]
```

```bash
# Generate spec for the latest Redmine, write to docs/openapi/openapi.yaml
./scripts/gen-openapi-spec.sh latest

# Generate for a specific version
./scripts/gen-openapi-spec.sh 6.1.3 /tmp/redmine-api.yaml
```

Generate one snapshot per supported version into `testdata/openapi/`:

```bash
./scripts/gen-spec-snapshots.sh
# or a subset:
VERSIONS="6.1.3" ./scripts/gen-spec-snapshots.sh
```

Commit the regenerated snapshots. The Go snapshot tests
(`internal/spec/snapshot_test.go`) and CI enforce that committed snapshots stay
strict-valid and match what the plugin actually emits.

Requirements: Docker. MySQL is started and cleaned up automatically.

### Via Rake task

If the plugin is installed in a running Redmine instance:

```bash
RAILS_ENV=production bundle exec rake redmine:openapi:generate
```

Environment variables:

| Variable      | Default   | Description                           |
|---------------|-----------|---------------------------------------|
| `OUTPUT_DIR`  | `public`  | Directory to write the spec into      |
| `FORMAT`      | `yaml`    | `yaml` or `json`                      |

Output goes to `public/openapi.yaml` (or `openapi.json`).

### Via HTTP endpoint

When installed, the plugin exposes `GET /openapi/spec` which returns the spec as
JSON or YAML. The endpoint requires the `view_openapi_spec` permission (grants
to administrators by default).

```
GET /openapi/spec.json
GET /openapi/spec.yaml
```

### Standalone script

`generate_spec.rb` can be run directly inside a Rails environment:

```bash
RAILS_ENV=test OUTPUT_PATH=/tmp/openapi.yaml \
  ruby plugin/redmine_openapi_generator/generate_spec.rb
```

## What Gets Generated

### Paths

Every route with an HTTP verb is included, mapped to the controller action that
handles it. Paths are normalized from Redmine's Rails format to OpenAPI:

- `:id` → `{id}`
- `(.:format)` / `.{format}` → stripped (not a real API parameter)
- Namespaced controllers (e.g., `admin/issues`) get flattened path segments

### Operations

Each operation gets:

- **tags** — derived from the controller name (`issues` → "Issues",
  `issue_relations` → "Issue Relations")
- **summary** — human-readable action description ("List issue", "Get version",
  "Create news")
- **operationId** — camelCase ID (`getIssues`, `createIssue`, `deleteVersion`)
- **parameters** — path params from `{id}`, plus `offset`/`limit` for index
  actions
- **requestBody** — references a contract schema:
  - POST uses `{Name}Create` when present, else `{Name}Request`
  - PUT/PATCH uses `{Name}UpdateRequest` when present, else `{Name}Create`,
    else no request body
- **responses** — schemas for 200/201/204 plus 401/403/412 common responses

### Schemas

Component schemas come entirely from the contract. Each resource exposes:

- `{Name}` — the wrapped entity (`Issue`)
- `{Name}Single` — `{ issue: Issue }`
- `{Name}List` — `{ issues: [Issue], total_count, offset, limit }`
- `{Name}Request` / `{Name}Create` / `{Name}Update` / `{Name}UpdateRequest` —
  request payloads for create/update operations

Shared components (`IDName`, `Journal`, `Relation`, …) live under `SHARED`.

### Security Schemes

| Name          | Type     | Description                           |
|---------------|----------|---------------------------------------|
| Basic         | http     | HTTP Basic Authentication             |
| ApiKey        | apiKey   | `X-Redmine-API-Key` header            |
| ApiKeyInQuery | apiKey   | `key` query parameter                 |
| OAuth2        | oauth2   | Authorization code flow               |

## Adding or Changing Schemas

Edit `contract.rb`. Add the resource's component schemas under `RESOURCES` (or
shared pieces under `SHARED`), then run strict validation to make sure nothing
is left unreferenced:

```bash
./scripts/gen-openapi-spec.sh 6.1.3 /tmp/openapi.yaml
go run ./cmd/redmine-spec spec validate --strict --spec /tmp/openapi.yaml
```

Then regenerate and commit snapshots, models, and the bundled spec:

```bash
./scripts/gen-spec-snapshots.sh
cp testdata/openapi/redmine-6.1.3.yaml docs/openapi/openapi.yaml
go run ./cmd/redmine-spec generate-models --spec docs/openapi/openapi.yaml --out internal/models
```

## Known Limitations

- **Curated contract** — schemas are hand-maintained in `contract.rb`, not
  inferred from Redmine internals. New resource types require editing the
  contract; there is no automatic schema discovery.
- **Contract surface** — only resources the contract covers appear in the spec.
  Redmine endpoints without a contract entry are omitted.
- **Version metadata only** — the contract surface is identical across
  supported Redmine versions (5.1/6.0/6.1); snapshots differ only in the
  `info.version` string.
- **Path parameter types** — all `{id}` parameters are typed as `integer`. Some
  Redmine resources use string identifiers.
- **Plugin controller excluded** — `openapi_spec` is filtered out to avoid
  self-referential paths. Other plugins' controllers will appear in the output.
