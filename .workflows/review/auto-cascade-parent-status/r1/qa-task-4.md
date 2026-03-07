TASK: Add cancelled dependency guard to ValidateAddDep

ACCEPTANCE CRITERIA:
- ValidateAddDep() blocks adding dependency on cancelled task with correct error message (Rule 8)
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rule 8 states: "Cannot add a dependency on a cancelled task. Error: 'cannot add dependency on cancelled task, reopen it first.'" This is a validation-only rule in the StateMachine.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/state_machine.go:86-94
- Notes: Rule 8 guard is placed at the top of ValidateAddDep, before cycle detection and child-blocked-by-parent checks. Iterates tasks to find the blocker by normalized ID, checks if its status is StatusCancelled, returns the exact error message from the spec. Clean early-return pattern with break after finding the blocker. Correct placement order -- checking cancelled status before more expensive graph operations.

TESTS:
- Status: Adequate
- Coverage: Tests at /Users/leeovery/Code/tick/internal/task/state_machine_test.go:446-527 cover:
  - Blocks adding dependency on cancelled task with exact error message match (line 446)
  - Allows adding dependency on open task (line 463)
  - Allows adding dependency on in_progress task (line 475)
  - Allows adding dependency on done task (line 487)
  - Verifies cycle detection still works after cancelled check passes (line 499)
  - Child-blocked-by-parent with mixed-case IDs still works (line 514)
- Notes: Good coverage of the positive case, all non-cancelled statuses as negative cases, and interaction with existing validations. Tests are focused and non-redundant. Edge case note in plan says "none" which is accurate -- the guard is straightforward.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming, NormalizeID for case-insensitive comparison.
- SOLID principles: Good. Single responsibility -- the cancelled check is a discrete block within ValidateAddDep.
- Complexity: Low. Simple linear scan with early break.
- Modern idioms: Yes. Range over index, early return, error message matches spec exactly.
- Readability: Good. Comment references "Rule 8" directly, making the spec link clear.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
