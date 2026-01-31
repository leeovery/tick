---
id: migration-1-1
phase: 1
status: pending
created: 2026-01-31
---

# Provider Contract & Migration Types

## Goal

The migration system needs a shared vocabulary before any provider or engine can be built. Without a normalized task type and a provider interface, there is no contract for beads (or future providers) to implement and no type for the engine to consume. This task defines the Go types and interface that all subsequent migration tasks depend on.

## Implementation

- Create a `migrate` package (e.g., `internal/migrate/`) to house all migration-related code.
- Define a `MigratedTask` struct representing a normalized task ready for insertion into tick. Fields mirror tick-core's `Task` schema but only the fields a provider can supply:
  - `Title` (string, required) — the only required field
  - `Status` (string, optional) — must map to tick statuses: `open`, `in_progress`, `done`, `cancelled`; defaults to `open` if empty
  - `Priority` (int, optional) — 0-4 range; defaults to `2` if not set (use pointer or sentinel to distinguish "not provided" from "zero")
  - `Description` (string, optional) — markdown text, no length limit
  - `Created` (time.Time, optional) — original creation timestamp; defaults to current time if zero
  - `Updated` (time.Time, optional) — original update timestamp; defaults to `Created` if zero
  - `Closed` (time.Time, optional) — closure timestamp; zero value means not closed
- Define a `Provider` interface with a single method:
  ```go
  type Provider interface {
      Name() string
      Tasks() ([]MigratedTask, error)
  }
  ```
  - `Name()` returns the provider identifier (e.g., `"beads"`) used in output
  - `Tasks()` returns all normalized tasks from the source, or an error if the source cannot be read
- Define a `Result` struct for per-task import outcomes:
  ```go
  type Result struct {
      Title   string
      Success bool
      Err     error  // nil on success
  }
  ```
- Add a `Validate` method for `MigratedTask` that checks:
  - Title is non-empty after trimming whitespace
  - Status, if provided, is one of the four valid tick statuses
  - Priority, if provided, is in 0-4 range
- Do NOT define insertion logic, CLI wiring, or output formatting — those belong to later tasks.

## Tests

- `"MigratedTask with only title is valid"`
- `"MigratedTask with all fields populated is valid"`
- `"MigratedTask with empty title is invalid"`
- `"MigratedTask with whitespace-only title is invalid"`
- `"MigratedTask with invalid status is rejected"`
- `"MigratedTask with valid status values are accepted"` (all four)
- `"MigratedTask with empty status is valid (defaults applied later)"`
- `"MigratedTask with priority out of range is rejected"` (-1 and 5)
- `"MigratedTask with priority in range is accepted"` (0 and 4 boundaries)
- `"Provider interface is implementable by a mock"` (compile-time check)

## Edge Cases

No runtime edge cases — foundational types-and-contracts task. Validation edge cases covered in tests.

## Acceptance Criteria

- [ ] `MigratedTask` struct exists with fields matching tick's task schema
- [ ] `Provider` interface exists with `Name()` and `Tasks()` methods
- [ ] `Result` struct exists with `Title`, `Success`, and `Err` fields
- [ ] Validation rejects empty/whitespace-only titles
- [ ] Validation rejects invalid status values while accepting all four valid ones plus empty
- [ ] Validation rejects priority outside 0-4 range
- [ ] A mock provider satisfies the `Provider` interface (proven by test)
- [ ] Types are self-contained within the migrate package

## Context

Spec mandates plugin/strategy: Provider → Normalize → Insert. Migration excludes `id` (generated on insert), `blocked_by` (different dependency model), and `parent` (hierarchy not migrated). Title is the only required field; everything else optional with defaults. Beads fields with no tick equivalent (issue_type, close_reason, created_by, dependencies) are discarded by the provider.

Specification reference: `docs/workflow/specification/migration.md`
