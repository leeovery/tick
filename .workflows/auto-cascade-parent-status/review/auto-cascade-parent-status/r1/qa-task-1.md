TASK: Create StateMachine struct with Transition method (acps-1-1)

ACCEPTANCE CRITERIA:
- StateMachine.Transition() passes all existing transition tests with identical behavior to task.Transition()
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: The spec defines StateMachine as a stateless struct in internal/task/ that consolidates all 11 transition/validation rules. Rule 1 covers the standard transition table: open -> in_progress, open -> done, open -> cancelled, in_progress -> done, in_progress -> cancelled, done/cancelled -> open (reopen). The Transition method mutates the task in-place (Status, Updated, Closed fields) and returns a TransitionResult.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/state_machine.go:11 (struct), :35-66 (Transition method)
- Notes: Implementation matches spec exactly. StateMachine is a stateless struct (no fields). The transitions map at :20-25 encodes all valid status transitions from Rule 1. The method handles unknown commands (line 37-39), invalid transitions (line 41-46), and correctly mutates Status, Updated, and Closed fields. The original `task.Transition()` in transition.go:20-23 is now a thin wrapper delegating to `sm.Transition()`, preserving backward compatibility.

TESTS:
- Status: Adequate
- Coverage:
  - All 7 valid transitions tested (TestStateMachine_Transition_ValidTransitions)
  - All 9 invalid transitions tested (TestStateMachine_Transition_InvalidTransitions)
  - Unknown command edge case tested (TestStateMachine_Transition_UnknownCommand)
  - No-modification-on-error edge case tested for both invalid transition and unknown command (TestStateMachine_Transition_NoModificationOnError)
  - Closed timestamp behavior tested: set on done, set on cancel, cleared on reopen (TestStateMachine_Transition_ClosedTimestamp)
  - Updated timestamp tested for all valid transitions (TestStateMachine_Transition_UpdatedTimestamp)
  - Original transition_test.go tests remain and exercise the wrapper function, confirming backward compat
- Notes: Both edge cases from the plan (unknown command, no-op on invalid transition) are explicitly tested. The existing transition tests in transition_test.go continue to pass via the wrapper, ensuring no regression. Tests verify both the TransitionResult return value and the in-place mutation of the task, which is thorough without being redundant.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, table-driven tests, error wrapping with fmt.Errorf, "it does X" naming in subtests.
- SOLID principles: Good. StateMachine has single responsibility (transition logic). The stateless struct pattern matches the spec's "method grouping" idiom. The data-driven transitions map separates transition rules from control flow.
- Complexity: Low. The Transition method has clear linear flow: lookup rule, check validity, apply mutation.
- Modern idioms: Yes. Value receiver on StateMachine (appropriate for stateless struct), pointer receiver for Task mutation, proper use of time.UTC().Truncate().
- Readability: Good. Well-documented with godoc comments explaining behavior, parameters, and mutation semantics.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The StateMachine tests and the original Transition wrapper tests are nearly identical in coverage. This is borderline over-tested but acceptable since both code paths should be verified and the wrapper is a public API that callers depend on.
