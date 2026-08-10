# AI-Agent CLI Readiness Checklist — 2026-08-10

Baseline audit of the `redmine` CLI against
[`ai-agent-cli-principles.md`](./ai-agent-cli-principles.md).
Status legend: `[x]` done · `[~]` partial · `[ ]` not done.

Audit method: read `internal/redminecmd/*`, `internal/output/output.go`,
`internal/client/client.go`, `cmd/redmine/main.go`, and ran the built binary
against error/help/flag paths.

---

## Structured Output

- `[~]` **`--json` flag (or `--output json`) on all commands that return data**
  Global `--output yaml|json` exists (default `yaml`); works for data-returning
  commands. No `--json` shortcut. Mutation commands (`issue update/delete`,
  `group update/delete`, `attachment update/delete`, `news create`,
  `membership remove`, `relation remove`, `time-entry delete`, `project delete`)
  print plain text like `Issue 123 updated` and never emit JSON.
- `[~]` **JSON to stdout; progress/warnings/errors to stderr**
  Structured data → stdout; retry logs (`client.SetLogWriter(os.Stderr)`) and
  errors → stderr. But success messages from mutations go to stdout as plain
  text even with `--output json`, and `issue grep` / `status list` /
  `tracker list` / `priority list` / `category list` print human tables to
  stdout in default (yaml) mode.
- `[ ]` **JSON flat, not deeply nested**
  Output mirrors Redmine API shapes, which are deeply nested
  (`issues[]` → `tracker`/`status`/`author` objects). `--fields` selects
  top-level keys but does not flatten.
- `[x]` **Consistent types across commands**
  All data originates from the same Redmine API with stable names/types; times
  come through as API timestamps.
- `[ ]` **Streaming output uses JSON Lines**
  No JSONL anywhere. `issue list --all` paginates internally and emits one blob.
- `[~]` **`--json` flag documented in `--help`**
  `--output` is documented globally ("Output format (yaml, json)") and several
  subcommands say "Use --output json for machine-parseable output". Caveat:
  `issue get` help shows an example with `-o yaml`, but no `-o` alias exists
  for the global flag (verified: `-o` errors "flag provided but not defined").

## Exit Codes

- `[~]` **`0` success, `1` general failure, `2` usage error**
  Success → 0, runtime errors → 1. Unknown command → 3 (urfave default).
  Missing required flag → 1. No distinct `2` for usage errors.
- `[ ]` **`3` not found, `4` permission denied, `5` conflict**
  Not implemented. HTTP 404/403/409 all surface as exit 1.
- `[x]` **Exit code is non-zero on failure**
  Verified: errors never return 0.
- `[ ]` **Exit-code contract documented**
  README/help document no exit codes.
- `[ ]` **Structured error emitted alongside non-zero exit**
  Errors are plain text on stderr. (`client.APIError`/`RequestError` carry
  status/method/URL but are only rendered as text.)

## Idempotency

- `[ ]` **Create/apply commands idempotent**
  Imperative creates only (`issue create`, `project create`, `group create`,
  `news create`, `time-entry create`). Re-running fails on a server conflict;
  no `ensure`/`apply`/`--if-not-exists`.
- `[~]` **Re-running a succeeded command succeeds**
  True for reads and updates; not for creates.
- `[ ]` **Conflicts detectable via a distinct exit code**
  No (always 1).
- `[x]` **No hidden state required for re-runs**
  Stateless client; config via env/file only.

## Self-Documentation

- `[x]` **`--help` includes description, usage, all flags with defaults**
  Every command carries `Description`, `ArgsUsage`, and flags with `Usage`
  text; defaults rendered by urfave/cli v3.
- `[~]` **Required vs optional is clear**
  `Required: true` flags render clearly, but positional args (e.g. `issue get
  <id>`, `project get <identifier>`) are required only by convention and
  validated at runtime.
- `[x]` **`--help` includes realistic examples**
  Present on `issue get/create/update`, `issue relation add`, etc.
- `[x] **Root `--help` lists subcommands; subcommand `--help` gives detail**
  `redmine --help` lists 17 commands; `redmine issue --help` lists subcommands
  with per-command detail.
- `[ ]` **`--version` present**
  `redmine --version` → "flag provided but not defined: -version". The
  `version` subcommand is project versions, not the CLI version.

## Composability

- `[ ]` **`--quiet` / `-q` bare output (one value per line)**
  Not implemented.
- `[ ]` **Stdin input explicitly supported and documented**
  No command reads stdin (`os.Stdin` unused).
- `[~]` **Batch/bulk operations**
  Read side: `issue list --all` auto-paginates. No bulk delete/update or
  selectors. Repeatable `--watcher`/`--custom-field` are additive only.
- `[x]` **Output pipes cleanly into `jq`/`xargs`**
  `--output json` yields valid JSON on stdout; `--fields` trims size.

## Destructive Operations

- `[ ]` **`--dry-run` with structured JSON diff**
  Not implemented.
- `[~]` **`--yes` / `--force` bypasses prompts**
  No prompts exist (deletes are immediate and unconfirmed), so nothing ever
  blocks — but there is also no bypass flag to honor a human-review workflow.
- `[x]` **Non-interactive terminals are safe**
  No interactive prompts, so non-TTY use is never blocked (no TTY detection
  needed).

## Errors

- `[ ]` **Errors have parseable types/codes**
  Plain text only.
- `[~]` **Failing input echoed back**
  Client errors include method + URL; some messages echo the value
  ("invalid issue ID: …"). Not structured and not systematic.
- `[~]` **Suggestions / next steps included**
  Ad hoc only: `project get` suggests close matches; DNS errors suggest
  "check your Redmine URL".
- `[~]` **Transient vs permanent distinguishable**
  Client auto-retries 5xx/network with backoff and reports "after N retries"
  in the message, but the caller gets no machine-readable classification.

## Grammar & Consistency

- `[x]` **`noun verb` hierarchy**
  `issue create`, `issue relation add`, `group update`, `membership add`,
  `attachment delete`, etc. Follows the `docker container ls` model.
- `[~]` **Naming consistent across commands**
  `upload` is a bare-verb top-level command (exception to noun-verb). Flag
  naming otherwise consistent (`time-entry`, `custom-field`,
  `fixed-version-id` kebab-cased).
- `[x]` **Same conventions applied to new commands**
  Established patterns are consistently followed.

---

## Summary

| Status | Count |
|--------|-------|
| `[x]` done | 10 |
| `[~]` partial | 12 |
| `[ ]` not done | 11 |

**Strengths:** help/self-documentation, noun-verb grammar, consistent JSON via
`--output json`, stderr separation for logs, statelessness.

**Biggest gaps for agent use:** no `--json` shortcut (and default is YAML), no
documented/distinct exit codes, no idempotent create, no `--dry-run` /
`--quiet` / stdin, no structured error output, no `--version`.

## Suggested next improvements (highest impact first)

1. Global `--json` alias and switch default output to json (or keep yaml but
   document the alias).
2. Exit-code contract: `0`/`1`/`2` usage/`3` not-found/`4` permission/`5`
   conflict — map from `APIError.StatusCode`; document in README.
3. Structured error JSON on stdout for non-zero exits (`--json` mode).
4. Idempotent create (`--if-not-exists` or `ensure` verb) on create commands.
5. `--quiet` bare output and `--version` flag.
6. `--dry-run` + structured diff on delete/create commands.
7. Fix `issue get` help example (`-o yaml` → `--output yaml`).

Re-check date: _(next audit)_
