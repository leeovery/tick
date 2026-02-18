---
id: migration-2-1
phase: 2
status: completed
created: 2026-01-31
---

# Engine Continue-on-Error

## Goal

Phase 1's `Engine.Run` fails fast when `TaskCreator.CreateTask` returns an error — it returns immediately with partial results and the error. The specification requires "continue on error, report failures at end" for all task-level failures. This task changes the engine so that insertion failures are handled the same way validation failures already are: record a failure `Result` and continue to the next task. After this change, `Engine.Run` never returns an error for individual task failures. The only case where `Run` returns a non-nil error is when the provider itself fails (`provider.Tasks()` error), because that means no tasks are available to process at all.

## Implementation

- Modify `Engine.Run` in `internal/migrate/engine.go` (or wherever the engine lives) to change the `CreateTask` error handling path:
  - **Before (Phase 1)**: If `TaskCreator.CreateTask(task)` returns an error, return immediately with `(partialResults, err)`
  - **After**: If `TaskCreator.CreateTask(task)` returns an error, append `Result{Title: task.Title, Success: false, Err: insertionErr}` to the results slice and continue to the next task
- The validation failure path stays unchanged — it already records a failure `Result` and continues.
- The `provider.Tasks()` error path stays unchanged — it still returns the error immediately (provider-level failure, not a per-task failure).
- After the loop completes, return `(allResults, nil)`. The error return is now always nil when the provider succeeds, regardless of how many individual tasks failed validation or insertion.
- Ensure the `Result.Err` for insertion failures wraps or preserves the original error from `CreateTask` so the presenter (migration-2-2) can display a meaningful failure reason.
- Update or remove the Phase 1 test `"it returns error immediately when TaskCreator.CreateTask fails"` — this behavior no longer exists. Replace it with tests that verify continue-on-error behavior.

## Tests

- `"it continues processing after CreateTask fails and records failure Result"`
- `"it returns nil error when all tasks fail insertion"` — all tasks produce failure Results, but Run returns `(results, nil)` not an error
- `"it returns nil error with mixed validation and insertion failures"` — some tasks fail validation, some fail insertion, some succeed; all captured in Results, no error returned
- `"failure Result from insertion contains the original CreateTask error"`
- `"results slice contains entries for all tasks in provider order regardless of success or failure"`
- `"provider.Tasks() error still returns immediately with error"` — this behavior is unchanged
- `"successful tasks are persisted even when later tasks fail insertion"` — verify CreateTask is called for valid tasks that appear after a failed insertion
- `"all tasks fail insertion returns results with all failures and nil error"` — edge case: every single task fails CreateTask
- `"mixed failures: validation failure then insertion failure then success produces three Results in order"` — verifies ordering is preserved across different failure types

## Edge Cases

**All tasks fail insertion**: Every call to `CreateTask` returns an error. `Engine.Run` returns a `[]Result` where every entry has `Success: false` and a non-nil `Err`. The returned error is nil — this is not a provider failure, just a series of task failures. The caller (presenter) reports each failure individually.

**Mixed validation and insertion failures**: Some tasks fail `Validate()`, some pass validation but fail `CreateTask`, some succeed entirely. All three outcomes are captured as `Result` entries in provider order. The engine does not distinguish between validation and insertion failures in terms of flow control — both record a failure Result and continue. The `Result.Err` will differ (validation error vs insertion error), which the presenter uses to display the reason.

## Acceptance Criteria

- [ ] `Engine.Run` continues processing remaining tasks when `CreateTask` returns an error
- [ ] Each insertion failure produces a `Result{Success: false, Err: <original error>}` in the results slice
- [ ] `Engine.Run` returns `nil` error when provider succeeds, even if all tasks fail insertion
- [ ] `Engine.Run` still returns an error immediately when `provider.Tasks()` fails (unchanged)
- [ ] Results slice contains one entry per task from the provider, in order, regardless of success/failure
- [ ] Phase 1 fail-fast test is replaced with continue-on-error tests
- [ ] All tests written and passing

## Context

The specification is explicit about error handling strategy:

> **Strategy**: Continue on error, report failures at end.
> When a task fails to import: 1. Log the failure with reason 2. Continue processing remaining tasks 3. Report summary at end
> No rollback — successfully imported tasks remain even if others fail.

Phase 1 (migration-1-3) implemented a simpler version where validation failures continue but insertion failures fail fast, with a note that Phase 2 would upgrade to full continue-on-error. This task completes that upgrade.

The key semantic change: `Engine.Run`'s error return now exclusively signals provider-level failures (cannot read source data). All task-level outcomes — success, validation failure, insertion failure — are captured in the `[]Result` slice. This simplifies the caller: check the error for provider problems, then iterate results for per-task reporting.

Specification reference: `docs/workflow/specification/migration.md`
