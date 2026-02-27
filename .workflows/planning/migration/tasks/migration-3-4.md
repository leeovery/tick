---
id: migration-3-4
phase: 3
status: completed
created: 2026-02-15
---

# Use task.Status type and constants instead of raw status strings

**Problem**: The `task` package defines `type Status string` with constants (`StatusOpen`, `StatusDone`, `StatusCancelled`, `StatusInProgress`). The migrate package re-declares the same values as raw string literals in three independent maps: `validStatuses` in `migrate.go:11`, `completedStatuses` in `engine.go:6`, and `statusMap` in `beads/beads.go:18`. If a status value changes in the `task` package, these maps would silently produce wrong behavior. The `MigratedTask.Status` field is `string` rather than `task.Status`, so the type system cannot catch mismatches. In `store_creator.go:52`, a raw cast `task.Status(mt.Status)` converts the untyped string. Additionally, unknown beads status values (e.g., "wontfix") silently map to an empty string with no user visibility.

**Solution**: Change `MigratedTask.Status` to `task.Status` and replace the string-literal maps with references to `task` package constants. This makes the type system enforce correctness and eliminates the raw cast in `store_creator.go`.

**Outcome**: Status values are type-safe throughout the migration pipeline. Any drift between the task package and migration package becomes a compile error.

**Do**:
1. In `internal/migrate/migrate.go`, change `MigratedTask.Status` from `string` to `task.Status`. Add import for `task` package.
2. Replace `validStatuses` map with references to task constants: `task.StatusOpen`, `task.StatusInProgress`, `task.StatusDone`, `task.StatusCancelled`.
3. In `internal/migrate/engine.go`, replace `completedStatuses` map entries with `task.StatusDone` and `task.StatusCancelled`. The map key type becomes `task.Status`.
4. In `internal/migrate/beads/beads.go`, change `statusMap` values from `string` to `task.Status`: `task.StatusOpen`, `task.StatusInProgress`, `task.StatusDone`.
5. In `internal/migrate/store_creator.go:52`, remove the `task.Status(mt.Status)` cast since `mt.Status` is now already `task.Status`. The empty-string check (`mt.Status == ""`) still works since `task.Status` is an underlying string.
6. Update all tests that construct `MigratedTask` with string status values to use `task.Status` constants.

**Acceptance Criteria**:
- `MigratedTask.Status` is `task.Status` not `string`.
- No raw status string literals remain in `migrate.go`, `engine.go`, or `beads.go` maps.
- No `task.Status()` cast in `store_creator.go`.
- All tests compile and pass.

**Tests**:
- Existing unit tests for `Validate`, `Engine.Run`, `StoreTaskCreator.CreateTask`, and beads provider should pass after updating status values to use constants.
- Verify that providing an invalid status to `MigratedTask.Validate()` still returns an error.
