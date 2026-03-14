TASK: Update call sites and rename evaluateRule3

ACCEPTANCE CRITERIA:
- RunTransition in internal/cli/transition.go calls sm.ApplyUserTransition
- validateAndReopenParent in internal/cli/helpers.go calls sm.ApplySystemTransition
- autoCompleteParentIfTerminal in internal/cli/update.go calls sm.ApplySystemTransition
- evaluateRule3 is renamed to autoCompleteParentIfTerminal with updated doc comment
- The single call site of the renamed function (line 377 in update.go) references autoCompleteParentIfTerminal
- No references to ApplyWithCascades remain in the cli package
- No references to evaluateRule3 remain anywhere in the codebase
- go test ./... passes with zero failures

STATUS: Complete

SPEC CONTEXT: The spec identifies three call sites that invoke ApplyWithCascades: RunTransition (user-initiated, should use ApplyUserTransition with auto=false), validateAndReopenParent (system-initiated Rule 6, should use ApplySystemTransition with auto=true), and evaluateRule3 (system-initiated Rule 3, should use ApplySystemTransition with auto=true). evaluateRule3 should be renamed to autoCompleteParentIfTerminal.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/transition.go:37 -- sm.ApplyUserTransition(tasks, &tasks[i], command)
  - internal/cli/helpers.go:124 -- sm.ApplySystemTransition(tasks, &tasks[i], "reopen")
  - internal/cli/update.go:154 -- sm.ApplySystemTransition(tasks, &tasks[parentIdx], action)
  - internal/cli/update.go:130-134 -- autoCompleteParentIfTerminal definition with doc comment
  - internal/cli/update.go:380 -- single call site references autoCompleteParentIfTerminal
- Notes:
  - All three call sites correctly use the appropriate wrapper (ApplyUserTransition for user commands, ApplySystemTransition for system side effects)
  - No references to ApplyWithCascades exist in the cli package (confirmed via grep)
  - No references to evaluateRule3 exist in Go source files (only in workflow documentation/tracking files which is expected)
  - The doc comment on autoCompleteParentIfTerminal accurately describes its behavior: "checks if the original parent's remaining children are all terminal after a child was reparented away. If so, it triggers auto-completion via ApplySystemTransition"
  - The call site is at line 380 (acceptance criteria says "line 377" which was approximate -- line 377 is blank/assignment, 380 is the actual call). This is a trivial line number shift from intervening code, not a concern.
  - The create.go call site at line 226 also correctly uses validateAndReopenParent which internally calls ApplySystemTransition

TESTS:
- Status: Adequate
- Coverage:
  - Existing transition tests in internal/cli/transition_test.go exercise RunTransition paths (which now use ApplyUserTransition)
  - Existing helpers_test.go tests (lines 419-503) exercise validateAndReopenParent with multiple scenarios (open parent, in-progress parent, cancelled parent rejection, done parent reopen, case-insensitive matching, nonexistent parent)
  - Existing update tests in internal/cli/update_test.go cover reparent scenarios that exercise autoCompleteParentIfTerminal
  - Unit tests in internal/task/apply_cascades_test.go exercise both ApplyUserTransition (18 subtests) and ApplySystemTransition (auto=true verification)
- Notes: The task explicitly states "Existing transition, create, and update tests continue to pass without modification" -- this is a refactoring task, not a new-feature task, so no new tests are expected. The existing test coverage is appropriate since the behavioral contract is unchanged; only the internal routing (which wrapper to call) changed.

CODE QUALITY:
- Project conventions: Followed. Uses StateMachine struct pattern, error wrapping, pointer-to-slice-element mutation within Mutate closures, doc comments on exported/unexported functions.
- SOLID principles: Good. The rename from evaluateRule3 (implementation-detail name) to autoCompleteParentIfTerminal (intent-revealing name) improves readability. The split into ApplyUserTransition/ApplySystemTransition follows open/closed principle -- new transition modes can be added without changing the cascade engine.
- Complexity: Low. Each wrapper is a one-line delegation to applyWithCascades.
- Modern idioms: Yes. Clean Go patterns throughout.
- Readability: Good. The doc comment on autoCompleteParentIfTerminal clearly explains when and why ApplySystemTransition is used. The function name is self-documenting.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The acceptance criteria mentions "line 377 in update.go" for the call site but the actual call is at line 380. This is a minor spec drift due to code changes in other tasks (the parent not-found guard added in task 2-1 shifted lines). Not a code issue.
