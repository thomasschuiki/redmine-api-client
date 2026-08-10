# AI-Agent CLI Readiness Checklist

Use this to check our CLI (`redmine` tooling) against the principles from
[`ai-agent-cli-principles.md`](./ai-agent-cli-principles.md).

Status: `[x]` done · `[~]` partial · `[ ]` not done. Add notes per item.

## Structured Output

- [ ] `--json` flag (or `--output json`) on all commands that return data
- [ ] JSON goes to **stdout** only; progress/warnings/spinners/errors go to **stderr**
- [ ] JSON is flat, not deeply nested, with stable field names
- [ ] Consistent types across commands (IDs as numbers/strings, time as seconds or ISO 8601)
- [ ] Streaming output uses JSON Lines (one object per line)
- [ ] `--json` flag is documented in `--help` output

## Exit Codes

- [ ] `0` success, `1` general failure, `2` usage error
- [ ] `3` not found, `4` permission denied, `5` conflict (if applicable)
- [ ] Exit code is non-zero on failure (never `0` on failure)
- [ ] Exit-code contract is documented in `--help` / README
- [ ] Structured error (`error` type + message) emitted alongside non-zero exit

## Idempotency

- [ ] Create/apply commands are idempotent (`ensure`/`apply`/`sync`, or `--if-not-exists`)
- [ ] Re-running a succeeded command succeeds again
- [ ] Conflicts are detectable via a distinct exit code (e.g. `5`)
- [ ] No hidden state required for re-runs

## Self-Documentation

- [ ] `--help` includes description, usage, all flags with defaults
- [ ] Required vs optional args/flags are clear
- [ ] `--help` includes realistic examples
- [ ] Root `--help` lists all subcommands; subcommand `--help` gives detail
- [ ] `--version` present

## Composability

- [ ] `--quiet` / `-q` bare output (one value per line, no headers)
- [ ] Stdin input explicitly supported and documented (e.g. `-f -`)
- [ ] Batch/bulk operations (e.g. selector flags) instead of one call per item
- [ ] Output pipes cleanly into `jq`, `xargs`, `while read`

## Destructive Operations

- [ ] `--dry-run` on destructive commands, emitting a structured JSON diff
- [ ] `--yes` / `--force` bypasses all confirmation prompts
- [ ] Prompts auto-skip or fail with a clear message when stdin is not a TTY

## Errors

- [ ] Errors have parseable types/codes (not just human prose)
- [ ] Failing input is echoed back in the error
- [ ] Suggestions / next steps included where useful
- [ ] Transient vs permanent errors distinguishable (for retry logic)

## Grammar & Consistency

- [ ] `noun verb` hierarchy (`issue create`, `issue comment add`)
- [ ] Naming consistent across commands (no mixed `create-user` / `delete_user`)
- [ ] Same conventions apply to new commands added later

---

## Scoring

Count `[x]` items → ratio vs total. Target: **all `[x]`**. Items left `[ ]` or `[~]`
should each carry a note and an owner in a tracking issue.

Last checked: _(date)_
