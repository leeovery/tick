---
id: tick-core-6-1
phase: 6
status: pending
created: 2026-02-09
---

# Add dependency validation to create --blocked-by and --blocks paths

**Problem**: The `create` command's `--blocked-by` path calls `ValidateBlockedBy` (self-reference only) and `validateIDsExist`, but never calls `task.ValidateDependency` or `task.ValidateDependencies` which perform cycle detection and the child-blocked-by-parent check. Similarly, `--blocks` in both `create` and `update` adds the task ID to target tasks' `blocked_by` arrays without any dependency validation. The `dep add` command correctly validates via `task.ValidateDependency` but the same validation was not applied to these equivalent code paths. A user could create a task with `--blocked-by` that forms a cycle or violates the child-blocked-by-parent rule, and it would be silently persisted.

**Solution**: In `create.go`, after validating IDs exist and before building the new task, call `task.ValidateDependencies(tasks, id, blockedBy)` for the `--blocked-by` list. For each `blockID` in the `--blocks` list (in both `create.go` and `update.go`), call `task.ValidateDependency(tasks, blockID, id)` to validate the reverse dependency. This mirrors the validation already performed in `dep.go:runDepAdd`.

**Outcome**: All dependency constraints (cycle detection, child-blocked-by-parent rule) are enforced consistently whether dependencies are added via `create --blocked-by`, `create --blocks`, `update --blocks`, or `dep add`.

**Do**:
1. In `internal/cli/create.go`, inside the Mutate closure, after the `validateIDsExist` calls and before building `newTask`: if `len(blockedBy) > 0`, call `task.ValidateDependencies(tasks, id, blockedBy)` and return error on failure.
2. In `internal/cli/create.go`, inside the Mutate closure, before the `--blocks` loop: for each `blockID` in `blocks`, call `task.ValidateDependency(tasks, blockID, id)` and return error on failure.
3. In `internal/cli/update.go`, inside the Mutate closure, in the `opts.blocks != nil` validation section (around line 95-101): for each `blockID` in `blocks`, call `task.ValidateDependency(tasks, blockID, id)` and return error on failure.
4. Add tests covering: (a) create --blocked-by that would form a cycle is rejected, (b) create --blocked-by where child is blocked by parent is rejected, (c) create --blocks that would form a cycle is rejected, (d) update --blocks that would form a cycle is rejected, (e) valid dependencies still succeed.

**Acceptance Criteria**:
- `tick create "X" --blocked-by <id>` rejects cycles with the same error message as `dep add`
- `tick create "X" --blocked-by <parent>` rejects child-blocked-by-parent with the same error message as `dep add`
- `tick create "X" --blocks <id>` rejects cycles when the reverse dependency would create one
- `tick update <id> --blocks <id2>` rejects cycles when the reverse dependency would create one
- All existing tests continue to pass

**Tests**:
- Test create with --blocked-by forming a direct cycle (A blocks B, create C --blocked-by A,C where C blocks A)
- Test create with --blocked-by where the new task is a child blocked by its parent
- Test create with --blocks forming a cycle
- Test update with --blocks forming a cycle
- Test that valid --blocked-by and --blocks dependencies are still accepted
