---
id: migration-1-3
phase: 1
status: completed
created: 2026-01-31
---

# Migration Engine - Iterate & Insert

## Goal

The migration system has types and a contract (migration-1-1) and will have a beads provider to produce normalized tasks (migration-1-2), but nothing orchestrates the flow from provider to tick's data store. The migration engine receives a `Provider`, calls `Tasks()` to get normalized `MigratedTask` values, validates each one, generates a tick ID, inserts each into tick's data store via tick-core's storage engine, and collects a `Result` per task. Without this engine, there is no bridge between source data and tick persistence.

## Implementation

- Create an `Engine` struct in `internal/migrate/` that holds a reference to tick-core's storage interface (the `Store` or an interface wrapping its `Mutate` method) so the engine can insert tasks into tick's data store.
- Define a `TaskCreator` interface (or similar) that the engine depends on, abstracting the tick-core `Store.Mutate` interaction:
  ```go
  type TaskCreator interface {
      CreateTask(t MigratedTask) (string, error) // returns generated tick ID, or error
  }
  ```
  This keeps the migrate package decoupled from tick-core internals. The real implementation wraps tick-core's storage engine — it generates a tick ID, builds a tick-core `Task` struct from the `MigratedTask` fields (applying defaults: empty status becomes `open`, zero priority becomes `2`, zero Created becomes `time.Now()`, zero Updated becomes Created value), and persists via `Mutate`. A mock implementation is used in tests.
- Implement `Engine.Run(provider Provider) ([]Result, error)`:
  1. Call `provider.Tasks()` to get all `[]MigratedTask`
  2. If `provider.Tasks()` returns an error, return it immediately (source is unreadable — no tasks to process)
  3. Iterate over each `MigratedTask`:
     a. Call `task.Validate()` — if validation fails, record a `Result{Title: task.Title, Success: false, Err: validationErr}` and continue to the next task
     b. Call `TaskCreator.CreateTask(task)` — if insertion fails, return the error immediately (Phase 1 behavior: fail fast on insertion error; Phase 2 will add continue-on-error)
     c. On success, record `Result{Title: task.Title, Success: true, Err: nil}`
  4. Return the collected `[]Result` slice and `nil` error
- For tasks with empty Title that fail validation, use a fallback like `"(untitled)"` in the Result.Title so the result is always displayable.
- The engine does NOT handle output formatting (that is migration-1-4) or CLI wiring (migration-1-5). It returns `[]Result` for the caller to present.
- The engine does NOT handle `--dry-run` or `--pending-only` — those are Phase 2 concerns.

## Tests

- `"it returns a successful Result for each valid task inserted"`
- `"it calls Validate on each MigratedTask before insertion"`
- `"it skips tasks that fail validation and records failure Result with error"`
- `"it returns error immediately when provider.Tasks() fails"`
- `"it returns empty Results slice when provider returns zero tasks"`
- `"it returns error immediately when TaskCreator.CreateTask fails"` (Phase 1 fail-fast)
- `"it processes all tasks in order and returns Results in same order"`
- `"it applies defaults via TaskCreator — empty status becomes open, zero priority becomes 2"`
- `"it records Result with fallback title when task has empty title and fails validation"`
- `"it continues past validation failures but stops on insertion failures"`

## Edge Cases

- **Empty provider (zero tasks)**: `provider.Tasks()` returns an empty slice and nil error. Engine returns an empty `[]Result` and nil error. This is a valid state (the source had no tasks), not an error.
- **Insertion failure**: In Phase 1, any `TaskCreator.CreateTask` error causes `Engine.Run` to return immediately with the error and the partial `[]Result` collected so far (all successful results up to the failure, plus the failed one). Phase 2 will change this to continue-on-error with failure aggregation.

## Acceptance Criteria

- [ ] `Engine` struct exists in `internal/migrate/` with a `Run(Provider) ([]Result, error)` method
- [ ] `TaskCreator` interface abstracts tick-core task creation
- [ ] Each `MigratedTask` is validated before insertion; validation failures produce a failure `Result` without stopping the engine
- [ ] Provider error (from `Tasks()`) is returned immediately
- [ ] Insertion error (from `CreateTask`) is returned immediately with partial results (Phase 1 fail-fast)
- [ ] Zero tasks from provider returns empty `[]Result` and nil error
- [ ] Results are collected in provider order
- [ ] Engine is testable with mock `Provider` and mock `TaskCreator` (no real tick-core dependency in tests)

## Context

The specification defines the architecture as Provider -> Normalize -> Insert, where the core receives normalized data and inserts into tick's data store. The engine is the "Insert" step. It is provider-agnostic — it receives `[]MigratedTask` (the normalized format) and creates tick entries.

Error handling strategy from the spec: "Continue on error, report failures at end. No rollback — successfully imported tasks remain even if others fail." Phase 1 implements a simpler version: validation failures are skipped (continue), but insertion failures are fatal (fail fast). Phase 2 upgrades to full continue-on-error for insertion failures as well.

Tick-core's `Store.Mutate` is the write path — it handles locking, JSONL atomic write, and SQLite cache update. The `TaskCreator` abstraction wraps this so the migration engine doesn't couple directly to tick-core's `Store` type. The real `TaskCreator` implementation generates tick IDs via tick-core's `GenerateID` function (with collision retry against existing tasks) and builds a full `Task` struct from the `MigratedTask` fields.

Field defaults applied during insertion: empty status -> `open`, zero priority -> `2`, zero Created -> `time.Now()`, zero Updated -> Created value, zero Closed -> nil/zero. Title is validated as non-empty by `MigratedTask.Validate()`.

Specification reference: `docs/workflow/specification/migration.md`
