# tick-core-7-5: Consolidate duplicate relatedTask struct into RelatedTask

## Task Summary

Refactoring task to eliminate a redundant unexported `relatedTask` struct in `show.go` that was structurally identical to the exported `RelatedTask` in `format.go`. The task removes the duplicate type, updates `queryShowData` to scan directly into `RelatedTask` fields, and eliminates the field-by-field conversion loops in `showDataToTaskDetail`.

**Commit:** `630779b` -- `impl(tick-core): Ttick-core-7-5 -- consolidate duplicate relatedTask struct into RelatedTask`

## V6 Implementation

### Architecture

The change is surgically scoped. Two files are touched for implementation:

- **`internal/cli/show.go`**: The `showData` struct fields `blockedBy` and `children` change from `[]relatedTask` to `[]RelatedTask`. The unexported `relatedTask` struct (with fields `id`, `title`, `status`) is deleted entirely. `queryShowData` now scans SQL rows directly into `RelatedTask` exported fields (`&r.ID`, `&r.Title`, `&r.Status`). The `showDataToTaskDetail` function drops both conversion loops (~18 lines) and directly assigns `d.blockedBy` and `d.children` to the `TaskDetail` output struct.
- **`internal/cli/list_show_test.go`**: A new test subtest validates that `queryShowData` populates `RelatedTask` fields correctly for both children and blockers.

The `RelatedTask` type definition in `format.go` (lines 86-91) is unchanged -- it was already the canonical exported type. The refactor simply makes `show.go` use it directly instead of maintaining its own shadow copy.

### Code Quality

The resulting `show.go` is clean and minimal at 169 lines. Key observations:

- **No intermediate type**: Data flows directly from SQL scan into the exported `RelatedTask` type with no mapping step. This is the right approach -- the previous code was performing an identity transformation between two structurally identical types.
- **Error handling preserved**: All `Scan` error checks, `depRows.Err()`, and `childRows.Err()` checks remain intact. Error wrapping with `%w` is consistent.
- **No behavioral change**: The function signatures, SQL queries, and output formatting are all unchanged. This is a pure internal simplification.
- **`showDataToTaskDetail` simplification**: The function drops from ~30 lines to ~15 lines. The remaining code handles `task.Task` construction and time parsing only -- the `RelatedTask` slices are passed through directly.

Minor note: the commit message has a typo (`Ttick-core-7-5` instead of `tick-core-7-5`), but this is cosmetic.

### Test Coverage

The new test (`queryShowData populates RelatedTask fields for blockers and children`) directly validates the refactored code path:

- Sets up 4 tasks: a parent with a child, and a blocker/blocked pair.
- Opens the store directly and calls `queryShowData` (unit-level, not through the CLI runner).
- Asserts all three exported fields (`ID`, `Title`, `Status`) on both the child `RelatedTask` and the blocker `RelatedTask`.
- Uses `t.Fatalf` for setup/length assertions and `t.Errorf` for field-level checks -- correct testing pattern.

The existing integration-level tests (testing show output strings for blockers, children, etc.) provide additional regression coverage. All 14 existing `TestShow` subtests remain unchanged, confirming no behavioral regression.

The test style is table-driven-adjacent (explicit setup/assert per scenario) and uses subtests with descriptive names -- consistent with the codebase style and golang-pro guidelines.

### Spec Compliance

All five acceptance criteria from the task plan are met:

1. The unexported `relatedTask` struct no longer exists in `show.go` -- **done** (struct definition and comment removed).
2. `queryShowData` populates `RelatedTask` directly -- **done** (scans into `&r.ID`, `&r.Title`, `&r.Status`).
3. `showDataToTaskDetail` no longer has field-by-field conversion loops -- **done** (direct assignment of slices).
4. All existing show and format tests pass unchanged -- **done** (existing tests untouched).
5. New test verifies `queryShowData` correctly populates `RelatedTask` fields -- **done**.

### golang-pro Compliance

- **Error handling**: All errors explicitly handled, no naked returns, error wrapping with `%w`.
- **Exported type documentation**: `RelatedTask` in `format.go` already has a doc comment; no new exported types added.
- **Testing**: Subtests with descriptive names, proper use of `t.Helper()` in test helpers, `t.Fatalf` for fatal preconditions, `t.Errorf` for assertions.
- **No reflection, no panic for control flow, no ignored errors**: Clean.
- **Code organization**: The change keeps the single responsibility clear -- `show.go` handles querying, `format.go` owns the type definitions.

## Quality Assessment

### Strengths

- **Precise scope**: The change does exactly one thing -- eliminates a duplicate type and its associated boilerplate. No scope creep.
- **Net negative line count**: -35 lines added, +68 lines (56 of which are test). The implementation is 33 lines shorter while adding test coverage.
- **Direct test of the refactored function**: Rather than only testing through the CLI runner, the new test calls `queryShowData` directly, which is the function that changed. This is the correct level of test granularity for a refactor.
- **Zero behavioral change**: SQL queries, output format, and public API are all unchanged. Only internal wiring simplified.

### Weaknesses

- **Minor commit message typo**: `Ttick-core-7-5` has a double `T`. Cosmetic only.
- **No negative/edge-case test added**: The new test covers the happy path (1 child, 1 blocker). A case with zero or multiple related tasks would slightly improve coverage, though existing integration tests already cover the zero case.

### Overall Rating: Excellent

A textbook refactoring task: clearly scoped, precisely executed, properly tested, net reduction in code complexity. The elimination of the duplicate type removes a maintenance hazard (two types that had to be kept in sync) and simplifies the data flow from SQL to formatted output.
