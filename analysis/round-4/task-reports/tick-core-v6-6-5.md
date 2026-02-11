# Task tick-core-6-5: Extract shared helpers for --blocks application and ID parsing (V6 Only -- Analysis Phase 6)

## Note

This is an analysis refinement task that only exists in V6. Standalone quality assessment.

## Task Summary

The task addresses duplication between `create.go` and `update.go` in two areas: (1) the comma-separated ID parsing pattern (`strings.Split` -> trim -> normalize -> filter empty) that appeared three times across the two files, and (2) the `--blocks` application loop (iterate tasks, match by blockID, append sourceID to BlockedBy, set Updated) that was structurally identical in both files. The solution extracts `parseCommaSeparatedIDs` and `applyBlocks` as shared helper functions in a new `helpers.go` file.

## V6 Implementation

### Architecture & Design

The helpers are placed in `internal/cli/helpers.go`, which already contained `outputMutationResult` and `openStore` -- establishing it as the shared utility file for the `cli` package. This is the right location; the functions are package-private (unexported), keeping them internal to the CLI layer.

**parseCommaSeparatedIDs** takes a raw comma-separated string and returns a normalized slice of IDs. It delegates normalization to the existing `task.NormalizeID`, maintaining consistency with the rest of the codebase.

**applyBlocks** operates on a `[]task.Task` slice by index (using `range i`), which correctly mutates the slice elements in-place. It takes `sourceID`, `blockIDs`, and a `now time.Time` parameter, making it pure with respect to time (no hidden `time.Now()` calls). The final version also includes a dedup guard that prevents appending `sourceID` if it already exists in `BlockedBy`, which is a sensible improvement over the original inline logic.

Both callers (`create.go` line 148, `update.go` line 185) invoke the helpers cleanly. The refactoring preserves the existing control flow: blocks are applied, then dependency validation runs on the modified task list.

The O(n*m*k) complexity of `applyBlocks` (n tasks, m blockIDs, k existing BlockedBy entries for dedup) is acceptable given that task counts and dependency lists are small in practice.

### Code Quality

- Clean, minimal function signatures with no unnecessary abstraction
- Good doc comments on both exported helpers (`parseCommaSeparatedIDs`, `applyBlocks`)
- No dead code introduced; the inline patterns in create.go and update.go are fully removed
- The `strings.Split` / `task.NormalizeID` / `strings.TrimSpace` chain is correctly preserved
- The `var ids []string` (nil-initialized) pattern in `parseCommaSeparatedIDs` is idiomatic Go -- returns nil for empty input rather than an empty slice
- Remaining `NormalizeID(TrimSpace(...))` calls in create.go:65 and update.go:69 are for the `--parent` single-value flag, not comma-separated lists, so they correctly remain inline
- No unused imports introduced

One minor note: `applyBlocks` could use a set/map for the dedup check on larger `BlockedBy` slices, but a linear scan is appropriate here given the expected cardinality.

### Test Coverage

The test file `helpers_test.go` provides thorough coverage:

**parseCommaSeparatedIDs** (7 subtests via table-driven pattern):
- Single ID
- Multiple IDs
- Whitespace around IDs
- Empty string input
- Only commas and whitespace
- Normalizes to lowercase
- Filters empty segments from trailing comma

**applyBlocks** (5 subtests):
- Appends sourceID to matching task's BlockedBy
- Sets Updated timestamp on modified tasks
- No-op with non-existent blockIDs (including verifying Updated is unchanged)
- Skips duplicate when sourceID already in BlockedBy (verifying no duplicate appended and Updated unchanged)
- Handles multiple blockIDs targeting different tasks

The test file also includes tests for `outputMutationResult` (3 tests) and `openStore` (3 tests), which were existing helpers in the same file.

Tests use `t.Run` subtests correctly, provide clear failure messages with `%v`/`%q` formatting, and use explicit time values rather than `time.Now()` to avoid flakiness. The table-driven pattern for `parseCommaSeparatedIDs` follows golang-pro skill guidance.

### Spec Compliance

All acceptance criteria are met:

| Criterion | Status |
|-----------|--------|
| No inline comma-separated ID parsing loops in create.go or update.go | Met -- grep confirms zero `strings.Split.*","` matches in either file |
| No inline --blocks application loops in create.go or update.go | Met -- both use `applyBlocks(...)` |
| Both helpers called from both create and update | Met -- `parseCommaSeparatedIDs` called in create.go:53,59 and update.go:77; `applyBlocks` called in create.go:148 and update.go:185 |
| All existing create and update tests pass | Met (implied by task completion status) |

All specified test cases are covered:

| Test Case | Status |
|-----------|--------|
| parseCommaSeparatedIDs with single ID, multiple IDs, whitespace, empty strings | Covered |
| parseCommaSeparatedIDs normalizes to lowercase | Covered |
| applyBlocks correctly appends sourceID to matching tasks' BlockedBy | Covered |
| applyBlocks sets Updated timestamp on modified tasks | Covered |
| applyBlocks with non-existent blockIDs (no-op) | Covered |

Additionally, the implementation goes beyond the spec by adding dedup protection in `applyBlocks` (preventing duplicate entries in BlockedBy) with a corresponding test case.

### golang-pro Skill Compliance

| Requirement | Status |
|-------------|--------|
| Handle all errors explicitly | N/A -- pure functions, no errors to handle |
| Write table-driven tests with subtests | Met -- `parseCommaSeparatedIDs` uses table-driven; `applyBlocks` uses subtests |
| Document all exported functions | N/A -- functions are unexported, but still documented |
| Propagate errors with fmt.Errorf("%w", err) | N/A -- no errors produced |
| No panic for error handling | Met |
| No ignored errors | Met |
| No goroutines without lifecycle management | Met -- no concurrency |
| No hardcoded configuration | Met |

## Quality Assessment

### Strengths

- **Clean extraction**: The refactoring is textbook DRY -- the extracted logic exactly matches what was duplicated, with no over-abstraction or unnecessary generalization
- **Enhanced correctness**: The dedup guard in `applyBlocks` is a meaningful improvement that prevents data corruption from repeated `--blocks` applications
- **Thorough testing**: 12 test cases covering both helpers plus the pre-existing helpers, with good edge case coverage (empty input, whitespace-only, trailing commas, duplicate detection)
- **Minimal blast radius**: Only 4 files changed (plus tracking docs). The refactoring is purely mechanical with no behavioral changes to the calling code paths
- **Idiomatic Go**: Nil slice returns, range-by-index for mutation, `time.Time` parameter instead of internal `time.Now()`, unexported helpers in the right package

### Weaknesses

- **No error return from applyBlocks**: If a blockID doesn't match any task, the function silently does nothing. This is arguably correct behavior (validation happens elsewhere), but a return value indicating how many tasks were modified could aid debugging. This is a very minor point since validation is handled by the callers before/after the call.
- **Test assertion style**: The `applyBlocks` tests manually check slice lengths and individual elements rather than using a helper like `slices.Equal`. This is functional but slightly verbose. Standard library approach for Go 1.21+.

### Overall Quality Rating

Excellent

The task is a clean, well-scoped refactoring that eliminates real duplication, adds a correctness improvement (dedup), and delivers comprehensive test coverage. The code is idiomatic, well-documented, and precisely satisfies all acceptance criteria. No issues of substance.
