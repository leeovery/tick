---
id: tick-core-1-5
phase: 1
status: pending
created: 2026-01-30
---

# CLI framework & tick init

## Goal

Tasks 1-1 through 1-4 built the data layer (model, JSONL, SQLite cache, storage engine), but there is no way for a user or agent to invoke any of it. Tick needs a CLI entry point and its first command: `tick init`. This task sets up the CLI framework using Go's standard library (no third-party CLI framework in v1) with subcommand dispatch, global flags, error handling conventions, and implements `tick init` — the command that creates the `.tick/` directory with an empty `tasks.jsonl` file.

## Implementation

- Set up `main.go` as the CLI entry point with `os.Args` subcommand dispatch
- Implement subcommand routing: parse first non-flag argument as subcommand, dispatch to handler. Unknown subcommands return error to stderr with exit code 1.
- Define global flags:
  - `--quiet` / `-q`: suppress non-essential output
  - `--verbose` / `-v`: more detail for debugging
  - `--toon`: force TOON output format
  - `--pretty`: force human-readable output format
  - `--json`: force JSON output format
- Implement TTY detection on stdout using `os.Stdout.Stat()` to check for `ModeCharDevice`. Non-TTY defaults to TOON, TTY defaults to human-readable. Flags override. (Actual formatting is Phase 4 — this task wires up detection and flag plumbing only.)
- Implement error handling conventions:
  - All errors go to stderr
  - Error messages prefixed with `Error: `
  - Exit code 0 for success, 1 for any error
  - Handlers return errors, main exits
- Implement `tick init`:
  1. Determine `.tick/` path from cwd
  2. Check if `.tick/` already exists → error if yes
  3. Create `.tick/` directory (mode 0755)
  4. Create empty `tasks.jsonl` (mode 0644, 0 bytes)
  5. Do NOT create `cache.db`
  6. Print confirmation: `Initialized tick in <absolute-path>/.tick/`
  7. With `--quiet`: no output on success
- Implement `.tick/` directory discovery helper: walk up from cwd looking for `.tick/`. Error if not found: `Error: Not a tick project (no .tick directory found)`

## Tests

- `"it creates .tick/ directory in current working directory"`
- `"it creates empty tasks.jsonl inside .tick/"`
- `"it does not create cache.db at init time"`
- `"it prints confirmation with absolute path on success"`
- `"it prints nothing with --quiet flag on success"`
- `"it errors when .tick/ already exists"`
- `"it returns exit code 1 when .tick/ already exists"`
- `"it writes error messages to stderr, not stdout"`
- `"it discovers .tick/ directory by walking up from cwd"`
- `"it errors when no .tick/ directory found (not a tick project)"`
- `"it routes unknown subcommands to error"`
- `"it detects TTY vs non-TTY on stdout"`

## Edge Cases

- Already initialized: if `.tick/` exists, error with exit code 1. Do not modify. Even a corrupted `.tick/` (missing `tasks.jsonl`) counts as "already initialized".
- No parent directory / not writable: surface OS error as `Error: Could not create .tick/ directory: <os error>`.
- Directory discovery: walk up from cwd to filesystem root. Stop at first `.tick/` match. If none found, error.
- Unknown subcommand: `Error: Unknown command '<name>'. Run 'tick help' for usage.` with exit code 1.
- No subcommand: print basic usage (list of commands) with exit code 0.

## Acceptance Criteria

- [ ] `tick init` creates `.tick/` directory with empty `tasks.jsonl`
- [ ] `tick init` does not create `cache.db`
- [ ] `tick init` prints confirmation with absolute path
- [ ] `tick init` with `--quiet` produces no output on success
- [ ] `tick init` when `.tick/` exists returns error to stderr with exit code 1
- [ ] All errors written to stderr with `Error: ` prefix
- [ ] Exit code 0 for success, 1 for errors
- [ ] Global flags parsed: `--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`
- [ ] TTY detection on stdout selects default output format
- [ ] `.tick/` directory discovery walks up from cwd
- [ ] Unknown subcommands return error with exit code 1

## Context

The specification states: "`tick init` creates `.tick/` directory, empty `tasks.jsonl`, SQLite cache created on first operation (not at init)." Directory structure: `.tick/tasks.jsonl` (git tracked), `.tick/cache.db` (gitignored, not at init), `.tick/lock` (created by flock on first use).

Error handling: exit 0 = success, 1 = error. All errors to stderr. Verbosity/format flags wire up detection only — actual formatting is Phase 4.

The storage engine (tick-core-1-4) expects `.tick/` with `tasks.jsonl` to exist. `tick init` is the prerequisite.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
