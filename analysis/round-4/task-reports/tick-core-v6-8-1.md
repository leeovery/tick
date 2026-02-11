# Task Report: tick-core-8-1 (V6 Only)

## Task Summary

**Task**: Prevent duplicate `blocked_by` entries in `applyBlocks`

**Problem**: The `applyBlocks` helper in `helpers.go` blindly appended `sourceID` to target tasks' `BlockedBy` slices without deduplication. When `tick update T1 --blocks T2` was called and T1 was already in T2's `blocked_by`, a duplicate entry was created. This was inconsistent with `dep add`, which explicitly rejects duplicates.

**Solution**: Add a linear scan for existing entries before appending in `applyBlocks`. Skip the append (and timestamp update) when `sourceID` is already present.

## V6 Implementation

### Architecture

The change is surgically scoped to a single function (`applyBlocks`) in `/private/tmp/tick-analysis-worktrees/v6/internal/cli/helpers.go`. The function is a shared helper used by both `create` and `update` commands for the `--blocks` flag, so the fix propagates to both code paths automatically.

The design choice to silently skip duplicates (rather than returning an error like `dep add` does) is appropriate for `applyBlocks` because:
- `--blocks` is a convenience flag that may be applied alongside other mutations; erroring would abort the entire update.
- `dep add` is a targeted command where the user explicitly intends to add a dependency, so an error is more informative.

### Code Quality

The implementation is clean and idiomatic:

```go
alreadyPresent := false
for _, dep := range tasks[i].BlockedBy {
    if dep == sourceID {
        alreadyPresent = true
        break
    }
}
if !alreadyPresent {
    tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)
    tasks[i].Updated = now
}
```

- Linear scan is appropriate given `BlockedBy` slices are small (typically single digits).
- Early `break` avoids unnecessary iteration.
- The `Updated` timestamp is correctly guarded behind the `!alreadyPresent` check, ensuring no spurious timestamp changes.
- The doc comment on `applyBlocks` was updated to note the skip behavior.
- No `slices.Contains` usage -- consistent with the rest of the codebase which uses manual loops for the same pattern (see `dep.go:97-101`).

One minor observation: a helper function like `containsString(slice, target)` could DRY up this pattern (it appears in `dep.go` as well), but that is out of scope for this task and the current approach is perfectly readable.

### Test Coverage

Three levels of testing were added:

1. **Unit test** (`helpers_test.go`, "it skips duplicate when sourceID already in BlockedBy"): Verifies that calling `applyBlocks` with a `sourceID` already in `BlockedBy` does not create a duplicate and does not change `Updated`. Assertions check slice length, value, and timestamp stability.

2. **Integration test** (`update_test.go`, "it does not duplicate blocked_by when --blocks with existing dependency"): Exercises the full `update` command pipeline -- sets up a project with a blocker already in the target's `blocked_by`, runs `tick update --blocks`, and verifies the persisted task has no duplicate.

3. **Pre-existing tests preserved**: The existing tests for normal append, timestamp setting, no-op with non-existent blockIDs, and multiple blockIDs are unchanged and continue to validate non-duplicate behavior.

All three acceptance criteria from the plan are covered:
- No duplicate entries in `BlockedBy` (unit + integration)
- Existing non-duplicate behavior unchanged (pre-existing tests)
- `Updated` only modified when a new dependency is added (unit test)

The plan also specified an integration test for `tick create` with an already-blocking scenario, which was not added. However, the `update` integration test adequately covers the persistence path since both `create` and `update` share the same `applyBlocks` helper.

Tests follow the golang-pro pattern: subtests with `t.Run`, descriptive names, explicit assertions with informative error messages.

### Spec Compliance

The implementation matches the plan exactly:
- Step 1: Modify `applyBlocks` in `helpers.go` -- done.
- Step 2: Add duplicate check before append -- done, with the correct guard on `Updated`.
- Step 3: Only append and set timestamp when new -- done.

The behavior aligns with the spec's `blocked_by` semantics (unique blockers) and is consistent with `dep add`'s duplicate handling, though the two paths handle it differently (silent skip vs. error return), which is the correct design choice for each context.

### golang-pro Compliance

| Requirement | Status |
|---|---|
| Handle all errors explicitly | N/A (no error paths in this change) |
| Document all exported functions | N/A (function is unexported) |
| Write table-driven tests with subtests | Subtests used; not table-driven but appropriate for scenario-based tests |
| Propagate errors with `fmt.Errorf("%w", err)` | N/A |
| No ignored errors | Compliant |
| No panic for normal error handling | Compliant |
| No hardcoded configuration | Compliant |

## Quality Assessment

### Strengths

- **Minimal, focused change**: Only 10 lines of logic added to a single function, with no ripple effects across the codebase.
- **Correct timestamp semantics**: The `Updated` guard is a subtle but important detail that the implementation handles correctly.
- **Strong test coverage**: Both unit and integration tests cover the exact behavior change, with assertions on all three dimensions (no duplicate, correct value, unchanged timestamp).
- **Consistent with existing patterns**: The linear scan matches the identical pattern used in `dep.go`, maintaining codebase consistency.
- **Updated documentation**: The function's doc comment was amended to describe the skip behavior.

### Weaknesses

- **Missing `create` integration test**: The plan called for an integration test via `tick create "A" --blocks T1` with an existing dependency. Only the `update` path was integration-tested. While the shared helper provides coverage, a `create` integration test would be more thorough.
- **No `slices.Contains` or shared helper**: The duplicate-check pattern (linear scan for string in slice) is now duplicated between `applyBlocks` and `dep add`. A shared utility would improve maintainability, though this is a codebase-wide concern beyond this task's scope.

### Overall Rating: **Excellent**

The implementation is precise, correct, well-tested, and consistent with existing codebase patterns. The single missing `create` integration test is a minor gap that does not materially affect confidence in correctness, given the shared helper provides full path coverage. The change is exactly what the task required -- nothing more, nothing less.
