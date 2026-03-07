---
id: migration-2-4
phase: 2
status: completed
created: 2026-01-31
---

# Pending-Only Filter

## Goal

When migrating from another tool, users often only care about active work — completed tasks are historical noise they do not want cluttering their fresh tick installation. Without a `--pending-only` flag, users must import everything and manually delete completed tasks afterward. This task adds a filtering stage to the engine that removes completed tasks (`Status` of `"done"` or `"cancelled"`) from the processing pipeline when the option is enabled. The filtering lives in the engine (between `provider.Tasks()` and the processing loop) so it is provider-agnostic — any current or future provider benefits from it. The CLI passes the option through when the user sets `--pending-only`.

## Implementation

- Add an `Options` struct (or extend one if it already exists) in `internal/migrate/` to carry engine configuration:
  ```go
  type Options struct {
      PendingOnly bool
  }
  ```
  Modify `Engine` to accept `Options` — either via its constructor (`NewEngine(creator TaskCreator, opts Options)`) or by adding an `Options` field. Choose the approach that is consistent with how the engine is already constructed (migration-1-3 / migration-1-5 established the pattern).

- Add a `filterPending` function (unexported) in `internal/migrate/`:
  ```go
  func filterPending(tasks []MigratedTask) []MigratedTask {
      var out []MigratedTask
      for _, t := range tasks {
          if t.Status != "done" && t.Status != "cancelled" {
              out = append(out, t)
          }
      }
      return out
  }
  ```
  This retains tasks with status `"open"`, `"in_progress"`, or `""` (empty — which defaults to `"open"` later). Only `"done"` and `"cancelled"` are considered completed and filtered out.

- Modify `Engine.Run` to apply the filter when `Options.PendingOnly` is true, immediately after receiving tasks from `provider.Tasks()` and before the validation/insertion loop:
  ```go
  tasks, err := provider.Tasks()
  if err != nil {
      return nil, err
  }
  if e.opts.PendingOnly {
      tasks = filterPending(tasks)
  }
  // existing validation + insertion loop over tasks
  ```
  When `PendingOnly` is false (the default), no filtering occurs — the full task list passes through unchanged.

- Add the `--pending-only` boolean flag to the `migrate` CLI command, following the same registration pattern as `--dry-run` and `--from`. Default is `false`.

- In the `migrate` command handler, pass the flag value into the engine's `Options`:
  ```go
  opts := migrate.Options{
      PendingOnly: pendingOnly,
  }
  engine := migrate.NewEngine(creator, opts)
  ```
  Ensure the `DryRunTaskCreator` path (from migration-2-3) still works — `--dry-run` and `--pending-only` are orthogonal. When both are set, filtering removes completed tasks first, then the no-op creator previews the remaining tasks. No special combination logic is needed.

- Update existing engine tests to pass a zero-value `Options{}` (or `Options{PendingOnly: false}`) so they continue to work unchanged. The `PendingOnly: false` default means no behavior change for any existing code paths.

## Tests

- `"filterPending removes tasks with status done"`
- `"filterPending removes tasks with status cancelled"`
- `"filterPending retains tasks with status open"`
- `"filterPending retains tasks with status in_progress"`
- `"filterPending retains tasks with empty status"` — empty status defaults to open later; must not be filtered
- `"filterPending returns empty slice when all tasks are completed"` — edge case: input has only done/cancelled tasks
- `"filterPending returns all tasks when none are completed"` — edge case: filter is a no-op
- `"filterPending with mixed statuses returns only non-completed tasks"` — open, in_progress, done, cancelled input produces only open and in_progress output
- `"filterPending preserves task order"`
- `"engine with PendingOnly true filters completed tasks before processing"`
- `"engine with PendingOnly false does not filter any tasks"`
- `"engine with PendingOnly true and all tasks completed returns empty Results and nil error"`
- `"engine with PendingOnly true and no completed tasks returns all Results"` — filter is a no-op
- `"engine with PendingOnly true still validates remaining tasks"` — a pending task with empty title still fails validation after filtering
- `"--pending-only flag defaults to false"`
- `"--pending-only flag is accepted by the migrate command"`
- `"--pending-only combined with --dry-run filters then previews without writing"`

## Edge Cases

**All tasks completed (zero remaining)**: Every task from the provider has status `"done"` or `"cancelled"`. After filtering, the task slice is empty. The engine processes zero tasks, returns an empty `[]Result` and nil error. The presenter prints the header and summary with `0 imported, 0 failed`. This is not an error — the source simply had no pending work.

**No completed tasks (filter is no-op)**: Every task from the provider has status `"open"`, `"in_progress"`, or `""`. The filter returns all tasks unchanged. The engine processes them all. Behavior is identical to running without `--pending-only`. This verifies the filter does not accidentally drop non-completed tasks.

**Mixed statuses**: The provider returns a mix of `"open"`, `"in_progress"`, `"done"`, and `"cancelled"` tasks. Only `"open"` and `"in_progress"` tasks pass through the filter. The results reflect only the filtered set. Task order is preserved — the nth non-completed task appears in its original relative position.

**Empty status after beads mapping**: The beads provider maps unknown/empty beads statuses to `""` (empty string) on `MigratedTask`. The `filterPending` function must NOT treat empty status as completed. Empty status later defaults to `"open"` during insertion, so it represents a pending task and must be retained.

**Composition with --dry-run**: Both flags set. Filtering removes completed tasks first (engine level), then `DryRunTaskCreator` previews the remaining tasks without persisting. The header shows `[dry-run]`. The filtered tasks each show `✓`. No special-case logic — the flags operate at different pipeline stages.

## Acceptance Criteria

- [ ] `filterPending` function exists in `internal/migrate/` and removes tasks with status `"done"` or `"cancelled"`
- [ ] `filterPending` retains tasks with status `"open"`, `"in_progress"`, or `""` (empty)
- [ ] `Engine` accepts an `Options` struct (or equivalent) with a `PendingOnly` field
- [ ] `Engine.Run` applies `filterPending` after `provider.Tasks()` and before the processing loop when `PendingOnly` is true
- [ ] `Engine.Run` does NOT filter when `PendingOnly` is false (default behavior unchanged)
- [ ] `--pending-only` flag is registered on the `migrate` command with default `false`
- [ ] CLI passes the `--pending-only` value to the engine via `Options`
- [ ] All tasks completed + `--pending-only` results in zero imported, zero failed (not an error)
- [ ] No completed tasks + `--pending-only` results in all tasks processed (filter is no-op)
- [ ] Mixed statuses + `--pending-only` results in only non-completed tasks processed
- [ ] Empty status is NOT filtered out (it represents a pending task)
- [ ] `--pending-only` composes with `--dry-run` without special-case logic
- [ ] Existing engine tests still pass with zero-value `Options`
- [ ] All tests written and passing

## Context

The specification defines `--pending-only` as an optional flag that imports only non-completed tasks:

```
tick migrate --from beads --pending-only  # Import only active work
```

The spec says "Import only non-completed tasks (default: import all)". Completed statuses in tick's model are `"done"` and `"cancelled"` (from migration-1-1's `MigratedTask.Status` valid values: `"open"`, `"in_progress"`, `"done"`, `"cancelled"`). The beads provider (migration-1-2) maps beads statuses as: `pending` -> `"open"`, `in_progress` -> `"in_progress"`, `closed` -> `"done"`. So after beads mapping, `--pending-only` filters out tasks with `"done"` status (originally `"closed"` in beads). No beads status maps to `"cancelled"`, but the filter handles it for correctness with future providers.

The filtering lives in the engine rather than the provider because it is provider-agnostic behavior — the concept of "pending vs completed" is defined by tick's status model, not the source system. Placing it in the engine (between `provider.Tasks()` and the processing loop) means every provider benefits from it automatically.

The `--pending-only` flag composes with `--dry-run` (migration-2-3) as orthogonal pipeline stages: `--pending-only` controls which tasks enter the pipeline (filtering), `--dry-run` controls what happens to tasks in the pipeline (persistence vs preview). This orthogonal design was established in migration-2-3's context.

Specification reference: `docs/workflow/specification/migration.md`
