# Task Report: tick-core-7-2 (V6 Only)

## Task Summary

**Task**: Add relationship context to create command output
**Commit**: `687ef3a` (V6)
**Problem**: `tick create` with `--blocked-by`, `--blocks`, or `--parent` flags produced output missing relationship context. The code constructed a `TaskDetail{Task: createdTask}` with empty `BlockedBy`/`Children` slices, unlike `tick show` and `tick update` which queried SQLite for full relationship data.
**Solution**: Replace manual `TaskDetail` construction with a `queryShowData` + `showDataToTaskDetail` call after mutation, making create output identical to show output.

## V6 Implementation

### Architecture

The change is minimal and well-targeted. In `internal/cli/create.go`, the post-mutation output block (lines 179-194 at commit time) was rewritten from:

```go
if fc.Quiet {
    fmt.Fprintln(stdout, createdTask.ID)
} else {
    detail := TaskDetail{Task: createdTask}
    fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
}
```

to:

```go
if fc.Quiet {
    fmt.Fprintln(stdout, createdTask.ID)
    return nil
}

data, err := queryShowData(store, createdTask.ID)
if err != nil {
    return err
}

detail := showDataToTaskDetail(data)
fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
```

This reuses the existing `queryShowData` (defined in `show.go`) which performs SQL JOINs to fetch blocker titles/statuses, parent title, and children. The `showDataToTaskDetail` conversion function is also reused. The approach is identical to what `RunUpdate` already used, ensuring consistent output across all mutation commands.

A subsequent commit (`53c8fbc`, tick-core-8-2) later extracted this pattern into a shared `outputMutationResult` helper in `helpers.go`, which both `RunCreate` and `RunUpdate` now call. This is a natural DRY refactoring that validates the correctness of the pattern introduced here.

### Code Quality

**Strengths**:
- The diff is +15/-4 lines for source code -- minimal, surgical change.
- Early return pattern for quiet mode is cleaner than the if/else it replaced.
- Error from `queryShowData` is properly handled and propagated.
- Reuses existing well-tested functions rather than duplicating logic.
- Comments are updated to explain the intent ("Full output: query the task with relationships like `tick show`").

**Minor observations**:
- The `queryShowData` call performs a second database read after the `Mutate` call. Since `Mutate` writes to both JSONL and SQLite, the data is immediately available. This is an extra query but keeps the code simple and consistent with `show`.
- No error wrapping with `%w` on the `queryShowData` error return -- but this is consistent with the pattern used everywhere else in the codebase (the wrapping is done inside `queryShowData` itself).

### Test Coverage

Five new test cases added (138 lines), covering all acceptance criteria:

| Test | What it verifies |
|------|-----------------|
| `it shows blocker title and status in output when created with --blocked-by` | Checks output contains "Blocked by:", blocker title, blocker ID, and "open" status |
| `it shows parent title in output when created with --parent` | Checks output contains "Parent:", parent ID, parent title |
| `it shows relationship context when created with --blocks` | Verifies `--blocks` creates the task and output includes the new task ID and title |
| `it outputs only task ID with --quiet flag after create with relationships` | Confirms `--quiet` outputs only the ID even when relationships exist |
| `it produces correct output without relationships (empty blocked_by/children)` | Verifies standard fields present and relationship sections absent when none exist |

**Test quality**:
- Tests use the existing `runCreate` helper which exercises the full App.Run path (integration-level).
- Tests set up realistic preconditions with `setupTickProjectWithTasks`.
- Assertions are targeted -- checking for specific strings rather than exact output format, which is appropriately resilient to formatting changes.
- The `--blocks` test reads persisted tasks to find the generated ID (since IDs are random), showing awareness of the system's behavior.
- The `--quiet` with relationships test is important -- it validates the early return path is still correct.

**Gaps**:
- The `--blocks` test does not verify that the *target* task's relationship context appears in the output (it only checks the new task's ID and title). The `--blocks` flag creates a reverse relationship on the target, not the new task itself, so the new task's output would not show blocking relationships. This is correct behavior but the test name ("shows relationship context") is slightly misleading.
- No test for combined flags (e.g., `--blocked-by` + `--parent` simultaneously).
- No test for `queryShowData` returning an error (e.g., if the database were somehow corrupted between mutate and query). This is an extreme edge case.

### Spec Compliance

The task plan states the spec (line 631) requires create output to be "full task details (same format as tick show), TTY-aware." The implementation achieves this by using the exact same code path (`queryShowData` -> `showDataToTaskDetail` -> `FormatTaskDetail`) that `RunShow` uses. All acceptance criteria from the plan are met:

1. `tick create "test" --blocked-by tick-abc` output includes blocker's title and status -- verified by test.
2. `tick create "test" --parent tick-abc` output includes parent's title -- verified by test.
3. Create output matches show output for the same task -- achieved structurally by using the same query/format path.
4. `--quiet` mode still outputs only the task ID -- verified by test.
5. All existing create tests pass -- the change is backward-compatible.

### golang-pro Compliance

| Rule | Status | Notes |
|------|--------|-------|
| Handle all errors explicitly | Pass | `queryShowData` error checked and returned |
| Propagate errors with `fmt.Errorf("%w", err)` | N/A | No new error wrapping needed; errors originate from `queryShowData` which wraps internally |
| Document exported functions | Pass | No new exported functions introduced |
| Table-driven tests with subtests | Partial | Tests use subtests (`t.Run`) but are not table-driven. Each test case is a standalone function due to differing setup and assertions. This is reasonable given the integration nature. |
| No ignored errors | Pass | No `_` assignments |
| No panic for error handling | Pass | |
| No goroutines without lifecycle | N/A | No concurrency introduced |

## Quality Assessment

### Strengths

1. **Minimal, focused change** -- 15 lines of production code achieve the goal cleanly by reusing existing infrastructure.
2. **Strong test coverage** -- 5 new tests covering all acceptance criteria, including edge cases (quiet mode, no relationships).
3. **Consistent architecture** -- Uses the same query/format path as `show` and `update`, which was later formalized into the shared `outputMutationResult` helper.
4. **Correct error handling** -- The error from the post-mutation query is properly checked and propagated.
5. **Clean early return** -- The quiet-mode early return simplifies control flow compared to the previous if/else.

### Weaknesses

1. **Extra database round-trip** -- After `Mutate` completes, a second SQL query fetches the same data that was just written. This is architecturally clean (separation of write and read paths) but involves redundant I/O. For a CLI tool, the performance impact is negligible.
2. **`--blocks` test is shallow** -- It verifies the task was created and appears in output but does not validate that any relationship-specific context appears (because `--blocks` affects the *target* task, not the created task). The test name suggests more than it actually checks.
3. **No combined-flag test** -- A test creating a task with both `--parent` and `--blocked-by` would increase confidence.

### Overall Rating

**Strong** -- This is a clean, well-scoped bug fix that correctly addresses a real gap where create output diverged from show output. The implementation reuses existing infrastructure, the tests cover all stated acceptance criteria, and the code follows established patterns. The weaknesses are minor (an extra query that is architecturally justified, and slightly shallow coverage of one edge case). The change was later naturally absorbed into a shared helper, confirming its design was sound.
