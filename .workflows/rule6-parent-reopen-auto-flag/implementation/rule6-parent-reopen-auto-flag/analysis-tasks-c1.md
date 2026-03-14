---
topic: rule6-parent-reopen-auto-flag
cycle: 1
total_proposed: 2
---
# Analysis Tasks: rule6-parent-reopen-auto-flag (Cycle 1)

## Task 1: Add parent not-found guard in autoCompleteParentIfTerminal
status: pending
severity: medium
sources: architecture

**Problem**: In `internal/cli/update.go`, `autoCompleteParentIfTerminal` declares `var parentIdx int` (zero-value 0) and searches for the parent in a loop. If the loop exits without finding the parent, `parentIdx` remains 0 and the function proceeds to call `ApplySystemTransition` on `tasks[0]` -- a completely unrelated task. This is currently unreachable because `EvaluateParentCompletion` returns `shouldComplete=false` when the parent doesn't exist, but correctness depends on upstream behavior never changing.

**Solution**: Initialize `parentIdx` to -1 as a sentinel value and add an explicit not-found guard after the loop. Return nil when the parent is not found, making the function self-contained and safe regardless of upstream changes.

**Outcome**: `autoCompleteParentIfTerminal` is defensively correct -- if the parent task is not found in the slice for any reason, the function returns nil instead of silently mutating an unrelated task.

**Do**:
1. In `internal/cli/update.go`, in the `autoCompleteParentIfTerminal` function, change `var parentIdx int` to `parentIdx := -1`
2. After the for-loop that searches for the parent (the `for i := range tasks` block), add: `if parentIdx < 0 { return nil }`
3. Run `go test ./internal/cli/ -run TestUpdate` to confirm no regressions

**Acceptance Criteria**:
- `parentIdx` is initialized to -1, not zero-value 0
- An explicit guard `if parentIdx < 0 { return nil }` exists between the search loop and the `ApplySystemTransition` call
- All existing tests pass

**Tests**:
- Existing `TestUpdate` tests continue to pass, confirming no behavioral change for the reachable code path

## Task 2: Extract assertTransition test helper in apply_cascades_test.go
status: pending
severity: medium
sources: duplication

**Problem**: The same 4-assertion block (check Transitions length, check From, check To, check Auto) is repeated 10+ times across subtests in `internal/task/apply_cascades_test.go`. Each instance is 8-10 lines of identical structure with only the expected values and task reference varying.

**Solution**: Extract a test helper function `assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool)` in `apply_cascades_test.go` that performs all four assertions. Replace each repeated block with a single call to this helper.

**Outcome**: Each 8-10 line assertion block is reduced to a single line while preserving the same assertion coverage and error messages. The test file becomes significantly shorter and easier to maintain.

**Do**:
1. In `internal/task/apply_cascades_test.go`, add a helper function at the top of the file (after imports):
   - Signature: `func assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool)`
   - Call `t.Helper()` as the first line
   - Assert `len(task.Transitions) > index` with a fatal if not (replacing the length check)
   - Assert `task.Transitions[index].From == from`
   - Assert `task.Transitions[index].To == to`
   - Assert `task.Transitions[index].Auto == auto`
2. Replace each repeated 4-assertion block throughout the file with a call to `assertTransition`
3. Run `go test ./internal/task/ -run TestApply` to confirm all subtests still pass

**Acceptance Criteria**:
- An `assertTransition` helper exists in `apply_cascades_test.go` with `t.Helper()` call
- All previous inline assertion blocks are replaced with calls to the helper
- No assertion coverage is lost -- same fields are checked with same expected values
- All existing tests pass

**Tests**:
- All existing `TestApplyUserTransition` and `TestApplySystemTransition` subtests pass with identical results
