# redmine_openapi_generator

A Redmine plugin that introspects a running Redmine instance and auto-generates an
OpenAPI 3.0.3 specification from its routes, API controllers, and ActiveRecord models.

## Why

Redmine ships with a REST API but no machine-readable spec. This plugin fills that
gap by running inside Rails and reading the source of truth directly: the routing
table, `accept_api_auth` declarations, and database schema. The output feeds into
the `redmine-spec` toolchain for validation, coverage checking, and Go model
generation.

## How It Works

The plugin has three data-collection phases:

1. **Routes** — iterates `Rails.application.routes.routes` to discover every
   HTTP verb, path template, controller, and action.

2. **API controllers** — finds all `ActionController::Base` descendants that call
   `accept_api_auth` and records which actions are API-exposed.

3. **Models** — maps a hardcoded set of well-known controllers to their
   ActiveRecord classes (`CONTROLLER_MODEL_MAP`) and reads column metadata
   (name, type, nullability, defaults) from the database schema.

The `SpecBuilder` then assembles a flat OpenAPI 3.0.3 document: paths with
parameters, operation summaries and IDs, request bodies referencing component
schemas, common error responses (401/403/412), and security schemes (Basic,
API key header, API key query, OAuth2).

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
      introspector.rb                      Data collection (routes, controllers, models)
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
- **requestBody** — JSON schema reference for POST/PUT/PATCH when a model map
  exists
- **responses** — 200/201/204 plus 401/403/412 common responses

### Schemas

Component schemas are generated from ActiveRecord column metadata for the mapped
models. Column types are translated from ActiveRecord to OpenAPI types:

| ActiveRecord | OpenAPI    |
|-------------|------------|
| integer     | integer    |
| string      | string     |
| text        | string     |
| boolean     | boolean    |
| float       | number     |
| datetime    | string     |
| date        | string     |
| json/jsonb  | object     |

Columns named `id`, `created_on`, `updated_on`, and `type` are excluded from
schemas.

### Security Schemes

| Name          | Type     | Description                           |
|---------------|----------|---------------------------------------|
| Basic         | http     | HTTP Basic Authentication             |
| ApiKey        | apiKey   | `X-Redmine-API-Key` header            |
| ApiKeyInQuery | apiKey   | `key` query parameter                 |
| OAuth2        | oauth2   | Authorization code flow               |

## Adding Models

To add a new controller/model mapping, edit `CONTROLLER_MODEL_MAP` in
`introspector.rb`:

```ruby
CONTROLLER_MODEL_MAP = {
  'issues'           => 'Issue',
  'projects'         => 'Project',
  'your_controller'  => 'YourModel',
  # ...
}.freeze
```

The key is the Redmine controller path (e.g., `your_controller` for
`YourController`, `admin/settings` for `Admin::SettingsController`). The value
must be a valid ActiveRecord model class name.

## Known Limitations

- **No enum/association inference** — schemas are flat column lists. Nested
  objects, required fields, enums, and association expansions are not generated.
- **Hardcoded model map** — adding new resource types requires editing
  `introspector.rb`. There is no runtime discovery of controller↔model
  associations.
- **No response schemas** — operations return generic 200/201/204 with no body
  schema. Redmine's API returns wrapper objects (`{ issue: { ... } }`) that are
  not modeled.
- **Path parameter types** — all `{id}` parameters are typed as `integer`. Some
  Redmine resources use string identifiers.
- **Plugin controller excluded** — `openapi_spec` is filtered out to avoid
  self-referential paths. Other plugins' controllers will appear in the output.
