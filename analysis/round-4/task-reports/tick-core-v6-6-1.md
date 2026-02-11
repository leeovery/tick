# Task tick-core-6-1: Add Dependency Validation to Create and Update --blocked-by/--blocks (V6 Only -- Analysis Phase 6)

## Note
This is an analysis refinement task that only exists in V6. Standalone quality assessment, not a comparison.

## Task Summary

**Problem**: `tick dep add` correctly validates dependencies via `task.ValidateDependency()` for cycle detection and child-blocked-by-parent checks. However, `tick create --blocked-by` only calls `validateRefs()` (existence + self-reference), skipping cycle and parent checks. `tick create --blocks` and `tick update --blocks` append to targets' `blocked_by` arrays with no dependency validation at all. This allows invalid dependency graphs to be persisted.

**Solution**: Call `task.ValidateDependencies()` / `task.ValidateDependency()` in `RunCreate` for both `--blocked-by` and `--blocks`, and in `RunUpdate` for `--blocks`, after building the full task list including modifications.

**Acceptance Criteria**:
1. `tick create --blocked-by <parent-id>` on a child task returns child-blocked-by-parent error
2. `tick create --blocked-by` that would create a cycle returns cycle detection error
3. `tick create --blocks <child-id>` on a parent task returns child-blocked-by-parent error
4. `tick update --blocks <child-id>` on a parent task returns child-blocked-by-parent error
5. `tick create --blocks` that would create a cycle returns cycle detection error
6. No invalid dependency graphs can be persisted through any write path

## V6 Implementation

### Architecture & Design

The implementation adds validation calls at precisely the right points in the mutation callbacks of both `RunCreate` and `RunUpdate`. The key design decision is **validate after mutate, before persist** -- the validation runs against the in-memory task list that already includes the proposed changes (`applyBlocks` has already modified `BlockedBy` arrays, and the new task has already been appended), so the cycle detection DFS and parent-check see the graph as it would exist after persistence. Since `store.Mutate` only persists when the callback returns `(tasks, nil)`, returning an error rolls back all in-memory changes.

In `create.go` (lines 153-163):
```go
// Validate dependencies (cycle detection + child-blocked-by-parent) against full task list.
if len(opts.blockedBy) > 0 {
    if err := task.ValidateDependencies(tasks, id, opts.blockedBy); err != nil {
        return nil, err
    }
}
for _, blockID := range opts.blocks {
    if err := task.ValidateDependency(tasks, blockID, id); err != nil {
        return nil, err
    }
}
```

In `update.go` (lines 187-192):
```go
// Validate dependencies (cycle detection + child-blocked-by-parent) against full task list.
for _, blockID := range opts.blocks {
    if err := task.ValidateDependency(tasks, blockID, opts.id); err != nil {
        return nil, err
    }
}
```

The argument ordering to `ValidateDependency` is critical and correct:
- For `--blocked-by`: `ValidateDependencies(tasks, id, opts.blockedBy)` -- the new task (`id`) is blocked by the given IDs.
- For `--blocks`: `ValidateDependency(tasks, blockID, id)` -- the target (`blockID`) is now blocked by the source (`id`). The arguments match the function signature `(tasks, taskID, newBlockedByID)`.

### Code Quality

**Idiomatic Go**: The code follows Go conventions cleanly -- early returns on error, minimal nesting, clear variable naming.

**DRY**: The implementation reuses the existing `task.ValidateDependency()` and `task.ValidateDependencies()` functions rather than duplicating validation logic. This is the core value of the task -- ensuring all write paths share the same validation code path as `tick dep add`.

**Error handling**: All errors are propagated explicitly via `return nil, err`, which causes the `Mutate` callback to abort without persisting. No errors are silently ignored.

**Guard clause**: The `if len(opts.blockedBy) > 0` guard around `ValidateDependencies` is a minor optimization avoiding an empty loop, which is consistent with the existing guard on `applyBlocks` (line 147: `if len(opts.blocks) > 0`).

**Ordering correctness**: In `create.go`, the sequence is:
1. `applyBlocks` (line 148) -- modifies target tasks' BlockedBy
2. `append(tasks, newTask)` (line 151) -- adds new task to list
3. Validation (lines 153-163) -- checks the full, modified graph

This is correct because `detectCycle` in `dependency.go` builds its adjacency map from the current state of `tasks[i].BlockedBy`, which already includes the `applyBlocks` modifications. If validation were run before `applyBlocks`, cycles introduced by `--blocks` would not be detected.

In `update.go`, the same pattern holds: `applyBlocks` at line 185, then validation at line 188.

**Comment quality**: Each validation block has a clear one-line comment explaining intent.

### Test Coverage

**Test functions added (create_test.go):**

1. `"it rejects --blocked-by that would create child-blocked-by-parent dependency"` -- Creates parent task, attempts to create child with `--parent` and `--blocked-by` pointing to the same parent. Asserts exit code 1, error mentions "parent", and no new task persisted.

2. `"it rejects --blocked-by that would create a cycle"` -- Creates taskA, then attempts `create "Task C" --blocked-by tick-aaa111 --blocks tick-aaa111`, which would create cycle C->A->C. Asserts exit code 1, error mentions "cycle", no new task persisted.

3. `"it rejects --blocks that would create child-blocked-by-parent dependency"` -- **Skipped** with `t.Skip()` and a clear explanation: architecturally impossible since the new task gets a random ID that no existing child references as parent. This is a thoughtful acknowledgment of the design constraint.

4. `"it rejects --blocks that would create a cycle"` -- Creates taskA (blocked by taskB) and taskB, then `create "Task C" --blocked-by tick-aaa111 --blocks tick-bbb222`, which would create C->A->B->C. Asserts cycle error, no new task persisted.

5. `"it allows valid dependencies through create --blocked-by and --blocks"` -- Happy path: creates taskA and taskB, then creates taskC blocked-by A and blocking B. Verifies 3 tasks persisted, newTask.BlockedBy contains A, and B's BlockedBy contains the new task ID.

**Test functions added (update_test.go):**

6. `"it rejects --blocks that would create child-blocked-by-parent dependency"` -- Parent and child exist, parent `--blocks` child. Asserts error, child's BlockedBy remains empty.

7. `"it rejects --blocks that would create a cycle"` -- taskA blocked by taskB. Update taskA `--blocks` taskB would create A->B->A cycle. Asserts cycle error, taskB's BlockedBy remains empty.

**Assessment**: The test coverage is thorough. Every specified test from the plan is present. The tests verify both the error conditions (exit code, error message content) and the persistence guarantees (no invalid state written to disk). The inline comments in the cycle tests explaining the graph topology are particularly helpful for maintainability. The `t.Skip` for the architecturally impossible case is well-reasoned.

**Minor gap**: There is no explicit test for `tick update --blocked-by` creating a cycle, but `update` does not support `--blocked-by` (it was added via `tick dep add`), so this is not a gap -- it is correctly out of scope.

### Spec Compliance

| Acceptance Criterion | Status | Evidence |
|---|---|---|
| `create --blocked-by <parent-id>` on child returns parent error | Met | Test: "it rejects --blocked-by that would create child-blocked-by-parent dependency" |
| `create --blocked-by` cycle returns cycle error | Met | Test: "it rejects --blocked-by that would create a cycle" |
| `create --blocks <child-id>` on parent returns parent error | Acknowledged impossible | Test skipped with architectural explanation; validation code present for defense-in-depth |
| `update --blocks <child-id>` on parent returns parent error | Met | Test: update_test.go "it rejects --blocks that would create child-blocked-by-parent dependency" |
| `create --blocks` cycle returns cycle error | Met | Test: "it rejects --blocks that would create a cycle" |
| No invalid graphs persisted through any write path | Met | All tests verify persisted state after rejected operations |

The `create --blocks <child-id>` criterion is technically unmet because the skip is correct -- in the current architecture, `--blocks` on `create` cannot trigger child-blocked-by-parent since the new task has a random ID no existing child references. The validation code is still present as defense-in-depth. This is a pragmatic and honest handling of an impossible-to-trigger condition.

### golang-pro Skill Compliance

**MUST DO checks:**
- Handle all errors explicitly (no naked returns): **Compliant** -- every error from `ValidateDependency`/`ValidateDependencies` is returned.
- Propagate errors with `fmt.Errorf("%w", err)`: The errors from `ValidateDependency` are returned directly (not re-wrapped). This is acceptable since the underlying functions already produce well-formatted error messages.
- Write table-driven tests with subtests: The tests use individual subtests rather than table-driven format. Given each test has unique setup logic (different task graphs), individual subtests are appropriate here.
- Document all exported functions: Not applicable -- no new exported symbols are introduced.

**MUST NOT DO checks:**
- Ignore errors: **Compliant** -- no `_` assignments.
- Use panic for normal error handling: **Compliant** -- errors returned, no panics.
- Create goroutines without lifecycle management: Not applicable.
- Skip context cancellation handling: Not applicable -- no blocking operations added.
- Use reflection: Not applicable.
- Hardcode configuration: Not applicable.

## Quality Assessment

### Strengths

1. **Correct validation ordering**: The validate-after-mutate-before-persist pattern ensures the cycle detection DFS sees the actual graph that would be persisted, catching all edge cases including combined `--blocked-by` + `--blocks` scenarios.

2. **Minimal, surgical changes**: Only 20 lines of production code added across two files. The implementation leverages existing `ValidateDependency`/`ValidateDependencies` functions exactly as intended, achieving the goal with zero code duplication.

3. **Thorough test scenarios**: Tests cover both error paths (cycle, parent) and the happy path. Each rejection test verifies both the error message and that no invalid state was persisted -- a defense-in-depth approach to test assertions.

4. **Honest handling of impossible cases**: The `t.Skip` with a clear architectural explanation for `create --blocks` child-blocked-by-parent is more valuable than a contrived test that could never trigger in production. The code still includes the validation call for defense-in-depth.

5. **Well-documented test reasoning**: The inline comments in cycle tests walk through the graph topology step by step, making the tests self-documenting and easy to maintain.

### Weaknesses

1. **Rollback of `applyBlocks` side effects on validation failure**: In `create.go`, `applyBlocks` modifies target tasks' `BlockedBy` arrays at line 148. If validation fails at line 155 or 160, the function returns `(nil, err)`, causing `Mutate` to discard the modified slice. This is correct for persistence (nothing is written to disk), but it means the in-memory task slice has been mutated before the error return. This is safe only because `Mutate` does not use the slice after a `nil` return. It would be slightly more defensive to validate *before* calling `applyBlocks`, though doing so would require building a temporary graph view. The current approach works correctly but relies on `Mutate`'s contract.

2. **No `update --blocked-by` validation**: The `update` command does not support `--blocked-by` as a flag, so this is not a bug. However, if `--blocked-by` is ever added to `update`, the validation gap would silently reappear. A brief comment noting this would be helpful for future maintainers.

### Overall Quality Rating

**Excellent** -- The implementation is minimal, correct, well-tested, and directly addresses every acceptance criterion. The 20 lines of production code reuse existing validation infrastructure with zero duplication. The test suite is thorough with 7 new test cases covering all specified scenarios plus the happy path. The one skipped test is properly justified. The validate-after-mutate ordering is correct and the architectural reasoning is sound.
