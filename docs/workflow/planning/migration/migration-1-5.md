---
id: migration-1-5
phase: 1
status: completed
created: 2026-01-31
---

# CLI Command - tick migrate --from

## Goal

The migration system has types (migration-1-1), a beads provider (migration-1-2), an engine (migration-1-3), and output formatting (migration-1-4), but no CLI entry point to invoke any of it. Without this task, users cannot run `tick migrate --from beads` — the entire pipeline is unreachable. This task registers the `migrate` subcommand, parses the `--from` flag, resolves the named provider from a hardcoded registry, wires together the real `TaskCreator` (wrapping tick-core's store), the `Printer` (writing to stdout), and the `Engine`, then executes the end-to-end flow.

## Implementation

- Register a `migrate` subcommand in tick's CLI (follow the existing command registration pattern). The command accepts a single required flag `--from <provider>`.
- Create a provider registry in `internal/migrate/` — a simple `map[string]` factory or lookup function. For Phase 1, hardcode a single entry: `"beads"` mapping to a factory that creates a `BeadsProvider` using the current working directory as the base directory.
  ```go
  // e.g., internal/migrate/registry.go
  func NewProvider(name string) (Provider, error)
  ```
  If the name is not in the registry, return an error: `unknown provider "<name>"`. Phase 1 uses a minimal error message; Phase 2 will enhance this to list available providers.
- In the `migrate` command handler:
  1. Parse the `--from` flag value. If `--from` is missing or empty, print a usage error to stderr and exit with code 1.
  2. Look up the provider by name via the registry. If unknown, print the error to stderr and exit 1.
  3. Create the real `TaskCreator` implementation that wraps tick-core's `Store.Mutate` — this adapter generates a tick ID, builds a tick-core `Task` from `MigratedTask` fields (applying defaults), and persists it. This may live in `internal/migrate/` as `StoreTaskCreator` or similar.
  4. Create the `Engine` with the real `TaskCreator`.
  5. Create the `Presenter` with `os.Stdout` as the writer.
  6. Call `presenter.WriteHeader(provider.Name())` to print the header.
  7. Call `engine.Run(provider)` to get `[]Result`.
  8. If `engine.Run` returns an error, print the error to stderr and exit 1.
  9. For each `Result`, call `presenter.WriteResult(result)`.
  10. Call `presenter.WriteSummary(results)` to print the summary.
  11. Exit 0 on success (even if some tasks had validation failures).
- The `StoreTaskCreator` (real `TaskCreator` implementation):
  - Accepts tick-core's `Store` (or the relevant write interface) in its constructor.
  - `CreateTask(t MigratedTask) (string, error)`:
    1. Generate a tick ID via tick-core's `GenerateID` function (with collision retry).
    2. Apply defaults: empty status → `"open"`, zero priority → `2`, zero Created → `time.Now()`, zero Updated → Created, zero Closed → nil/zero.
    3. Build a tick-core `Task` struct from the `MigratedTask` fields + generated ID + applied defaults.
    4. Persist via tick-core's write path (e.g., `Store.Mutate`).
    5. Return the generated ID or error.
- Do NOT implement `--dry-run` or `--pending-only` flags — Phase 2 scope.
- Do NOT implement the enhanced "Available providers:" listing — Phase 2 scope.
- Ensure the command handles "not initialized" errors gracefully if tick-core returns one.

## Tests

- `"migrate command requires --from flag"`
- `"migrate command with --from beads resolves beads provider"`
- `"migrate command with unknown provider returns error"`
- `"migrate command with empty --from value returns error"`
- `"registry returns BeadsProvider for name beads"`
- `"registry returns error for unknown provider name"`
- `"StoreTaskCreator creates a tick task from MigratedTask with all fields"`
- `"StoreTaskCreator applies default status open when MigratedTask status is empty"`
- `"StoreTaskCreator applies default priority 2 when MigratedTask priority is zero/unset"`
- `"StoreTaskCreator applies default Created as time.Now when zero"`
- `"StoreTaskCreator applies default Updated as Created when zero"`
- `"StoreTaskCreator generates a tick ID for each created task"`
- `"StoreTaskCreator returns error when store write fails"`
- `"end-to-end: migrate --from beads reads tasks, inserts, and prints output"` (integration test with test fixtures)
- `"migrate command exits 0 when some tasks fail validation but others succeed"`
- `"migrate command exits 1 when provider cannot be read"`

## Edge Cases

**Missing --from flag**: Required flag. If omitted or provided with an empty value, print a usage error to stderr and exit with code 1.

**Unknown provider (minimal)**: Phase 1 returns a simple error like `unknown provider "xyz"`. The enhanced error listing available providers is deferred to Phase 2.

**Tick not initialized**: If the current directory does not have a `.tick/` directory, tick-core will return an error when the `StoreTaskCreator` tries to write. The command propagates this error to stderr and exits 1.

**Provider read failure**: If `provider.Tasks()` returns an error (e.g., missing `.beads` directory), `engine.Run` returns that error. The command prints it to stderr and exits 1.

**Zero tasks (not an error)**: If the provider returns zero tasks, the engine returns empty results. The command prints the header and summary (e.g., "Done: 0 imported, 0 failed") and exits 0.

## Acceptance Criteria

- [ ] `tick migrate --from beads` executes the full pipeline: provider → engine → output
- [ ] `--from` flag is required; omission produces a usage error on stderr with exit code 1
- [ ] Unknown provider name produces an error on stderr with exit code 1
- [ ] `StoreTaskCreator` correctly creates tick tasks from `MigratedTask` values via tick-core
- [ ] Defaults are applied during task creation: empty status → open, zero priority → 2, zero timestamps → sensible defaults
- [ ] Each imported task is printed via `Presenter.WriteResult` as it is processed
- [ ] Summary line is printed via `Presenter.WriteSummary` after all tasks
- [ ] Command exits 0 on success (including when some tasks fail validation)
- [ ] Command exits 1 on provider failure or insertion failure
- [ ] Provider registry resolves `"beads"` to a `BeadsProvider` using the current working directory
- [ ] End-to-end integration test passes with test fixtures

## Context

The specification defines the command as `tick migrate --from <provider> [--dry-run] [--pending-only]`. Phase 1 implements only `--from`; the optional flags are Phase 2. The spec states: "If `--from` specifies an unrecognized provider, exit immediately with an error listing available providers" — the full listing is Phase 2; Phase 1 uses a minimal unknown provider error.

The architecture is Provider → Normalize → Insert, where "the inserter is provider-agnostic — it receives normalized data and creates tick entries." The `StoreTaskCreator` is the real implementation of the `TaskCreator` interface from migration-1-3, bridging from the migration package to tick-core's `Store.Mutate` write path.

Tick-core uses a dual-storage model (JSONL source of truth + SQLite cache). The `StoreTaskCreator` interacts with tick-core's mutation flow: acquire lock → read JSONL → apply mutation → atomic rewrite → update SQLite → release lock. The `StoreTaskCreator` does not implement this flow directly — it delegates to tick-core's existing write operations.

Output format from the spec:
```
Importing from beads...
  ✓ Task: Implement login flow
  ✓ Task: Fix database connection
  ✗ Task: Broken entry (skipped: missing title)

Done: 2 imported, 1 failed
```

The `Presenter` (migration-1-4) handles all formatting. This task wires it to `os.Stdout` and calls its methods in the correct sequence.

Specification reference: `docs/workflow/specification/migration.md`
