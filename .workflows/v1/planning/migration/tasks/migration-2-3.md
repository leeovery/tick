---
id: migration-2-3
phase: 2
status: completed
created: 2026-01-31
---

# Dry-Run Mode

## Goal

Users want to preview what a migration would import before committing changes to tick's data store. Without a `--dry-run` flag, the only way to see what would be imported is to actually import it — and since migration is append-only with no undo, this is irreversible. This task adds a `--dry-run` flag to the `tick migrate` command that previews import results without writing anything. The approach uses a no-op `TaskCreator` implementation rather than an engine option, keeping the engine unaware of dry-run semantics. When `--dry-run` is set, the CLI wires in a `DryRunTaskCreator` instead of the real `StoreTaskCreator`. Since the no-op creator always succeeds, every valid task shows `✓` in the output. A `[dry-run]` indicator in the header line makes it clear that no data was written.

## Implementation

- Create a `DryRunTaskCreator` struct in `internal/migrate/` that implements the `TaskCreator` interface:
  ```go
  type DryRunTaskCreator struct{}

  func (d DryRunTaskCreator) CreateTask(t MigratedTask) (string, error) {
      return "", nil // no-op: no ID generated, no persistence
  }
  ```
  The returned ID is an empty string since no tick task is actually created. The error is always nil — dry-run insertion never fails. Validation still runs (the engine validates before calling `CreateTask`), so tasks with invalid data (e.g., missing title) will still show as failures.

- Add the `--dry-run` boolean flag to the `migrate` CLI command. Follow the same pattern used for `--from`. Default is `false`.

- Modify the `migrate` command handler to select the `TaskCreator` based on the `--dry-run` flag:
  ```go
  var creator migrate.TaskCreator
  if dryRun {
      creator = migrate.DryRunTaskCreator{}
  } else {
      creator = migrate.NewStoreTaskCreator(store)
  }
  ```
  The engine, provider, and presenter are created and wired identically in both paths. The only difference is which `TaskCreator` the engine receives.

- Modify the presenter's header output to indicate dry-run mode. When dry-run is active, the header should read:
  ```
  Importing from beads... [dry-run]
  ```
  This requires passing a `dryRun bool` parameter to the presenter's header method (or to the presenter constructor). The per-task lines and summary line remain unchanged — the `[dry-run]` indicator in the header is sufficient to communicate that nothing was written.

- The `--dry-run` flag composes naturally with `--pending-only` (migration-2-4). When both are set, the provider filters to pending tasks only, and the no-op creator previews what would be imported from that filtered set. No special handling is needed for the combination — the flags are orthogonal by design.

- Do NOT modify the engine. The engine receives a `TaskCreator` and calls it; it does not know or care whether the creator persists data or is a no-op. This is the cleanest separation of concerns.

## Tests

- `"DryRunTaskCreator.CreateTask returns empty string and nil error"`
- `"DryRunTaskCreator.CreateTask never returns an error regardless of input"`
- `"DryRunTaskCreator satisfies the TaskCreator interface"` (compile-time check)
- `"engine with DryRunTaskCreator produces successful Result for each valid task"`
- `"engine with DryRunTaskCreator still fails validation for tasks with empty title"`
- `"dry-run header prints 'Importing from <provider>... [dry-run]'"`
- `"dry-run with zero tasks prints header and summary with zero counts"`
- `"dry-run with multiple tasks shows all as successful (checkmark for each)"`
- `"dry-run combined with --pending-only filters tasks and previews only pending ones"`
- `"--dry-run flag defaults to false"`
- `"non-dry-run execution still uses StoreTaskCreator"` (regression)
- `"dry-run summary shows correct imported count matching number of valid tasks"`

## Edge Cases

**Dry-run with zero tasks**: The provider returns an empty `[]MigratedTask`. The engine produces an empty `[]Result`. The presenter prints the header with the `[dry-run]` indicator and the summary with `0 imported, 0 failed`. No per-task lines. This is not an error — it simply means the source had nothing to import.

**Dry-run combined with --pending-only**: Both flags are set. The provider (via `--pending-only` filtering from migration-2-4) returns only non-completed tasks. The `DryRunTaskCreator` processes those tasks without persisting. The output shows only the filtered tasks with `✓` marks. No special interaction logic is needed — the flags compose cleanly because `--pending-only` affects what tasks the provider returns, while `--dry-run` affects what happens when tasks are inserted. They operate at different stages of the pipeline.

**Dry-run with tasks that fail validation**: Tasks with invalid data (e.g., missing title) still fail validation in the engine because `Validate()` runs before `CreateTask()`. These appear as `✗` failures in the output even during dry-run. This is correct — it previews exactly what would happen during a real import, including which tasks would be rejected.

## Acceptance Criteria

- [ ] `DryRunTaskCreator` exists in `internal/migrate/` and implements `TaskCreator`
- [ ] `DryRunTaskCreator.CreateTask` always returns `("", nil)` — no persistence, no errors
- [ ] `--dry-run` flag is registered on the `migrate` command with default `false`
- [ ] When `--dry-run` is set, the CLI wires `DryRunTaskCreator` instead of `StoreTaskCreator`
- [ ] When `--dry-run` is not set, the CLI still wires `StoreTaskCreator` (unchanged behavior)
- [ ] Header output includes `[dry-run]` indicator when dry-run is active
- [ ] Header output does NOT include `[dry-run]` when dry-run is inactive (regression check)
- [ ] Engine is NOT modified — dry-run is entirely a CLI wiring concern
- [ ] Validation still runs during dry-run; invalid tasks produce failure Results
- [ ] Zero tasks in dry-run mode produces header with `[dry-run]` and summary with zero counts
- [ ] `--dry-run` and `--pending-only` compose correctly without special-case logic
- [ ] All tests written and passing

## Context

The specification defines `--dry-run` as an optional flag that previews what would be imported without writing:

```
tick migrate --from beads --dry-run    # Preview import
```

The specification shows the same output format for dry-run and real import. Since the no-op `TaskCreator` always succeeds, every valid task shows `✓` (no insertion errors can occur). The `[dry-run]` header indicator is an addition beyond the spec's output example — it prevents user confusion about whether data was actually written.

The no-op `TaskCreator` approach is preferred over an engine option because it keeps the engine ignorant of dry-run semantics. The engine's contract is: receive a `Provider` and a `TaskCreator`, validate tasks, call `CreateTask`, collect results. Whether `CreateTask` actually persists is an implementation detail of the `TaskCreator`, not the engine's concern. This follows the same dependency-injection pattern established in migration-1-3 and migration-1-5.

The `DryRunTaskCreator` returns an empty string for the task ID. Since the migration output format (migration-1-4) does not display task IDs — only titles and success/failure — this has no visible effect on the output.

The `--dry-run` flag composes with `--pending-only` because the two flags operate at different pipeline stages: `--pending-only` filters at the provider level (what tasks are fetched), `--dry-run` swaps at the insertion level (how tasks are persisted). No coupling between them.

Specification reference: `docs/workflow/specification/migration.md`
