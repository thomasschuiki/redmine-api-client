# Writing CLI Tools That AI Agents Actually Want to Use

Essence of the article by Ugo Enyioha: <https://dev.to/uenyioha/writing-cli-tools-that-ai-agents-actually-want-to-use-39no>

## Core Thesis

If an agent has shell access, a well-designed CLI often beats an MCP server. The author deleted three MCP servers in favor of direct CLI usage because:

- **Token cost** — MCP tool definitions (descriptions, JSON schemas) are loaded into the agent's system prompt and consume context before the tool is even used. A CLI call only consumes tokens when actually invoked (a simple file conversion switched from MCP to CLI cut token usage ~40%).
- **Zero indirection** — Agents already know how to read `--help`, parse output, and handle errors. Wrapping a CLI in an MCP server is pure overhead.
- **Reliability** — MCP servers crash, lose connections, and have startup latency. A stateless CLI binary is always available.

> Rule of thumb: *"If a human would use a CLI for it, the agent should too."*

**When MCP still wins:** 50+ tools behind one server, stateful sessions across calls, dynamic tool discovery at scale, multi-agent architectures needing protocol-level composition.

## The Eight Rules

### 1. Structured Output Is Not Optional
Support `--json` or `--output json`. Agents are good at parsing text, but not *reliable* at it — table wrapping or inconsistent casing causes silent failures.

- **JSON to stdout, everything else to stderr.** Progress, warnings, spinners → stderr. Stdout is your API contract.
- **Flat over nested:** `{"pod_name": "web-1", "pod_status": "Running"}` beats nested objects.
- **Consistent types:** numbers stay numbers, time as seconds or ISO 8601.
- **JSON Lines (JSONL)** — one JSON object per line — for streaming output.

### 2. Exit Codes Are the Agent's Control Flow
Agents check `$?` to decide next steps. A tool returning 0 on failure breaks every dependent workflow.

```
0 = success
1 = general failure
2 = usage error (bad arguments)
3 = resource not found
4 = permission denied
5 = conflict (resource already exists)
```

Pair exit codes with structured error output (error type in stdout JSON, human message on stderr).

### 3. Make Commands Idempotent
Agents retry; networks fail. A `create` that fails on the second run forces special-case retry logic. Prefer declarative verbs (`ensure`, `apply`, `sync`) over imperative (`create`, `delete`) — the `kubectl apply` model. If you can't be idempotent, make conflicts *detectable* via a distinct exit code (e.g. 5 = already exists).

### 4. Self-Documenting Beats External Docs
`--help` is a tool description, parameter spec, and usage guide combined. Good help includes: description, usage, all args/flags with defaults, clear required vs optional, realistic examples, and explicit documentation of the `--json` flag (agents can't use what they don't know exists). Use subcommand discovery: `myctl --help` lists subcommands; `myctl deploy --help` gives full detail.

### 5. Design for Composability
- `--quiet` / `-q` emits bare values (one per line, no headers/decoration) for piping into `xargs` / `while read`.
- **Stdin acceptance is explicit and documented** (`myctl apply -f -`).
- **Batch operations exist** (`myctl delete --selector app=old`) instead of many individual calls.
- Support both "filter in the CLI" and "pipe to jq".

### 6. Provide Dry-Run and Confirmation Bypass
- `--dry-run` produces **structured output** — a JSON diff of changes, not prose.
- `--yes` / `--no-confirm` / `--force` bypasses all prompts — an agent can't type "y".
- **Detect non-interactive terminals:** if stdin is not a TTY, skip prompts automatically or fail with a clear error telling the user to pass `--yes`.

### 7. Errors Should Be Actionable
On failure the agent must decide: retry, try something else, or give up.

- **Error codes/types in structured output:** `"image_not_found"` is parseable; `"Error occurred"` is not.
- **Include the failing input** so the agent can construct a fix.
- **Suggest next steps:** `"suggestion": "run myctl images list"`.
- **Separate transient from permanent errors** so the agent can build correct retry logic.

### 8. Use Consistent Noun-Verb Grammar
Use `noun verb` hierarchy (`myctl user create`, `myctl user group add`) — the `docker container ls` / `gh pr create` model. `myctl --help` shows resources (nouns), `myctl user --help` shows actions (verbs). Exploration becomes a deterministic tree search instead of a guessing game.

## Conventions Cheat Sheet

| Convention | Spec |
|---|---|
| Structured output | `--json` or `--output json`; JSONL for streaming |
| Stream separation | JSON → stdout; progress/warnings/spinners/errors → stderr |
| JSON shape | Flat fields over nesting; consistent field names & types; seconds or ISO 8601 for time |
| Exit codes | `0` success, `1` general failure, `2` usage error, `3` not found, `4` permission denied, `5` conflict — documented |
| Bare output | `--quiet` / `-q` (one value per line, no headers) |
| Idempotency | `ensure` / `apply` / `sync` verbs, or `--if-not-exists` flag |
| Destructive preview | `--dry-run` producing JSON diff output |
| Prompt bypass | `--yes` / `--no-confirm` / `--force` |
| Interactive detection | Fail or auto-skip prompts when stdin is not a TTY |
| Self-documentation | `--help` with required/optional flags, defaults, examples |
| Stdin | Document explicitly (e.g. `myctl apply -f -`) |
| Batch | Selector/bulk flags (e.g. `--selector app=old`) |
| Grammar | `noun verb` hierarchy (`user create`, `user delete`) |

## MCP vs CLI Decision Framework

| Factor | Choose CLI | Choose MCP |
|---|---|---|
| Number of operations | < 15 commands | 50+ tools |
| State between calls | Stateless | Stateful sessions needed |
| Agent has shell access | Yes | No (API-only) |
| Token budget matters | Yes | Less constrained |
| Existing CLI exists | Wrap or use directly | Build MCP server |
| Multi-agent system | Single agent | Protocol composition |
| Reliability requirement | High (no server to crash) | Acceptable server dependency |

> Closing insight: *"The best agent tooling often starts as an MCP server and migrates to a CLI once you understand the actual usage patterns — or starts as a CLI and stays there because it was good enough all along."*
