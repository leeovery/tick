---
id: tick-core-6-1
phase: 6
status: pending
created: 2026-02-10
---

# Add dependency validation to create and update --blocked-by/--blocks

**Problem**: The spec (line 403) requires validating dependencies at write time before persisting to JSONL. `tick dep add` correctly calls `task.ValidateDependency()` for cycle detection and child-blocked-by-parent checks. However, `tick create --blocked-by` only calls `validateRefs()` which checks existence and self-reference but NOT cycles or child-blocked-by-parent. `tick create --blocks` and `tick update --blocks` append to target tasks' blocked_by arrays with no dependency validation at all. This allows invalid dependency graphs (cycles, child-blocked-by-parent) to be persisted.

**Solution**: Call `task.ValidateDependency()` or `task.ValidateDependencies()` in `RunCreate` for both `--blocked-by` and `--blocks` targets after building the new task, and in `RunUpdate` for `--blocks` targets. The full task list (including the new/modified task) must be passed to enable proper graph analysis.

**Outcome**: All write paths that modify blocked_by arrays enforce the same validation rules as `tick dep add` -- cycle detection and child-blocked-by-parent rejection -- before persisting to JSONL.

**Do**:
1. In `internal/cli/create.go` `RunCreate`, after the new task is built and added to the task list, call `task.ValidateDependencies()` (or loop with `task.ValidateDependency()`) for each entry in `opts.blockedBy` against the full task list
2. In `internal/cli/create.go` `RunCreate`, for each `opts.blocks` target, after appending the new task ID to the target's blocked_by, call `task.ValidateDependency()` to validate the new dependency against the full task list
3. In `internal/cli/update.go` `RunUpdate`, for each `opts.blocks` target, after appending the task ID to the target's blocked_by, call `task.ValidateDependency()` to validate the new dependency against the full task list
4. If any validation fails, return the error before persisting -- the Mutate callback should return early with the validation error

**Acceptance Criteria**:
- `tick create --blocked-by <parent-id>` on a child task returns child-blocked-by-parent error
- `tick create --blocked-by` that would create a cycle returns cycle detection error
- `tick create --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick update --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick create --blocks` that would create a cycle returns cycle detection error
- No invalid dependency graphs can be persisted through any write path

**Tests**:
- Test create with --blocked-by that would create child-blocked-by-parent dependency is rejected
- Test create with --blocked-by that would create a cycle is rejected
- Test create with --blocks that would create child-blocked-by-parent dependency is rejected
- Test create with --blocks that would create a cycle is rejected
- Test update with --blocks that would create child-blocked-by-parent dependency is rejected
- Test update with --blocks that would create a cycle is rejected
- Test that valid dependencies through create --blocked-by and --blocks still work correctly
