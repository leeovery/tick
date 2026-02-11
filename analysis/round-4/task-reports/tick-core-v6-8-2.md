# tick-core-8-2: Extract Post-Mutation Output Helper

## Task Summary

The task called for extracting duplicated post-mutation output logic from `create.go` and `update.go` into a shared helper function in `helpers.go`. Both commands contained structurally identical blocks that check quiet mode (print ID only), or else query the task, convert it to a detail struct, format it, and print it. The goal was DRY consolidation so future output-format changes only require editing one location.

## V6 Implementation

### Architecture

The implementation follows the plan precisely. A new `outputMutationResult` function was added to `internal/cli/helpers.go`, which is the correct home for shared CLI helpers. The function signature matches the plan exactly:

```go
func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error
```

Both `RunCreate` (line 173 in `create.go`) and `RunUpdate` (line 201 in `update.go`) now terminate with a single call to this helper. The inline output blocks (16-17 lines each) were cleanly removed from both callers.

The helper is unexported (lowercase), which is appropriate since it is only used within the `cli` package. It sits at the top of `helpers.go`, before the other helpers, and follows the same structural pattern.

### Code Quality

- **Idiomatic Go**: The function takes `io.Writer` for output (not `*os.File`), follows standard error-return patterns, and uses early return for the quiet-mode fast path.
- **Imports**: `fmt` and `io` were added to `helpers.go`. Both `create.go` and `update.go` still legitimately use `fmt` for `fmt.Errorf` calls, so no unused import was left behind.
- **Documentation**: The exported-style doc comment on `outputMutationResult` is thorough, describing both the quiet and non-quiet code paths. Technically it is unexported so a doc comment is not required by the golang-pro skill, but it is good practice and the existing helpers in the file follow the same convention.
- **Error handling**: Errors from `queryShowData` are propagated directly. No errors are silently swallowed.
- **No behavior change**: The logic is a direct mechanical extraction -- the quiet check, `queryShowData`, `showDataToTaskDetail`, and `FormatTaskDetail` sequence is preserved exactly.

### Test Coverage

Three new test cases were added in `TestOutputMutationResult` within `helpers_test.go` (lines 174-261):

1. **Quiet mode** -- verifies output is `"tick-aaa111\n"` and nothing more.
2. **Non-quiet mode** -- verifies output contains the task ID, title, and formatted field labels (`ID:`, `Status:`).
3. **Non-existent task ID** -- verifies the function returns an error containing `"not found"`.

The tests use real stores via `setupTickProjectWithTasks` / `setupTickProject`, `strings.Builder` as the writer, and `PrettyFormatter` as the formatter. This is an integration-level test of the helper, which is appropriate given that the function's purpose is to orchestrate store queries and formatting.

The plan's acceptance criteria state that existing tests for `RunCreate` and `RunUpdate` should continue to pass unchanged. No existing test was modified, confirming behavioral equivalence.

### Spec Compliance

Every item in the plan's "Do" list is satisfied:

| Plan Requirement | Status |
|---|---|
| Add `outputMutationResult` to `helpers.go` | Done, exact signature |
| Replace inline output in `create.go` | Done (lines 173-188 replaced with single call) |
| Replace inline output in `update.go` | Done (lines 201-215 replaced with single call) |
| Add `fmt`, `io` imports to `helpers.go` | Done |

Acceptance criteria:

| Criterion | Status |
|---|---|
| `create.go` and `update.go` no longer contain inline post-mutation output | Met |
| Both commands produce identical output | Met (mechanical extraction) |
| Helper is single source of truth | Met |
| Existing tests pass unchanged | Met (no test modifications) |

### golang-pro Compliance

| Constraint | Assessment |
|---|---|
| All errors handled explicitly | Yes -- `queryShowData` error checked and returned |
| Errors propagated with context | The function returns raw errors from `queryShowData`; no additional wrapping. Acceptable here since the caller context is obvious |
| Exported types/functions documented | N/A (unexported function, but documented anyway) |
| Table-driven tests with subtests | The new tests use subtests (`t.Run`) but are not table-driven. Given the three cases have different setup/assertion logic, individual subtests are more readable than a table |
| No ignored errors | Correct |
| No panic for error handling | Correct |
| `io.Writer` for output abstraction | Yes |

## Quality Assessment

### Strengths

- **Textbook DRY refactoring**: The extraction is clean and mechanical with zero behavioral changes. The diff is symmetric -- the same block removed from two files, consolidated into one.
- **Good test coverage**: Three test cases covering the happy paths (quiet, non-quiet) and the error path (missing task). Tests exercise real storage, not mocks, giving high confidence.
- **Correct placement**: The helper lives in `helpers.go` alongside other shared CLI utilities, consistent with the existing file's purpose.
- **Minimal blast radius**: Only the files that needed changing were touched. Imports were correctly updated (added to `helpers.go`, retained in callers where still used).

### Weaknesses

- **No edge case for quiet mode with non-existent ID**: In quiet mode, `outputMutationResult` prints the ID and returns nil without verifying the task exists. This is technically the pre-existing behavior (the original inline code did the same), so the refactoring is correct, but it could be considered a latent issue worth noting.
- **Minor**: The non-quiet test asserts on substring presence (`"ID:"`, `"Status:"`) rather than exact output format. This is pragmatic but loosely coupled to the formatter output.

### Overall Rating

**Excellent**

This is a clean, well-scoped refactoring that achieves exactly what the plan specified. The code is idiomatic, the tests are thorough, and the change carries zero risk of behavioral regression. The implementation matches the plan's function signature and placement precisely, and the resulting code is measurably simpler in both `create.go` and `update.go`.
