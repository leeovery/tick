TASK: Add ValidateAddChild for cancelled parent guard

ACCEPTANCE CRITERIA:
- ValidateAddChild() blocks adding child to cancelled parent with correct error message (Rule 7)
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rule 7 states "Cannot add a child to a cancelled parent. Error: cannot add child to cancelled task, reopen it first." The spec also clarifies ValidateAddChild is pure validation only -- Rule 6 (done parent reopen) is the caller's responsibility.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/state_machine.go:68-76
- Notes: Clean, minimal implementation. Checks parent.Status == StatusCancelled and returns the exact error message from the spec. Returns nil for all other statuses. Method receiver is value type (StateMachine is stateless struct), consistent with other methods. Comment documents Rule 7 and caller responsibility for Rule 6.

TESTS:
- Status: Adequate
- Coverage: Four subtests cover: cancelled parent (error with exact message check), open parent (no error), in_progress parent (no error), done parent (no error). This covers every possible status value.
- Notes: Tests are well-structured, focused, and not redundant. Each subtest covers a distinct status. The cancelled case verifies both error presence and exact message text. Plan notes "edge cases: none" which is accurate -- the function is a single status check.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, "it does X" naming, error wrapping style matches project patterns.
- SOLID principles: Good. Single responsibility -- pure validation, no side effects. Clean separation from Rule 6 caller logic.
- Complexity: Low. Single if-check, cyclomatic complexity of 2.
- Modern idioms: Yes. Idiomatic Go -- simple guard clause with early return.
- Readability: Good. Self-documenting with clear doc comment.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
