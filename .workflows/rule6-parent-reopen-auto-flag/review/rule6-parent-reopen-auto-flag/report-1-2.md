TASK: Update call sites and rename evaluateRule3

ACCEPTANCE CRITERIA:
- RunTransition in internal/cli/transition.go calls sm.ApplyUserTransition
- validateAndReopenParent in internal/cli/helpers.go calls sm.ApplySystemTransition
- autoCompleteParentIfTerminal in internal/cli/update.go calls sm.ApplySystemTransition
- evaluateRule3 is renamed to autoCompleteParentIfTerminal with updated doc comment
- The single call site of the renamed function (line 380 in update.go) references autoCompleteParentIfTerminal
- No references to ApplyWithCascades remain in the cli package
- No references to evaluateRule3 remain anywhere in the codebase (Go source)
- go test ./... passes with zero failures

STATUS: Complete

SPEC CONTEXT: The spec defines three call sites that invoke ApplyWithCascades: RunTransition (user-initiated), validateAndReopenParent (system-initiated Rule 6), and evaluateRule3 (system-initiated Rule 3). Task 1-1 created ApplyUserTransition and ApplySystemTransition wrappers. This task updates all call sites to use the correct wrapper and renames evaluateRule3 to autoCompleteParentIfTerminal.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/transition.go:37 -- ApplyUserTransition
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:124 -- ApplySystemTransition (in validateAndReopenParent)
  - /Users/leeovery/Code/tick/internal/cli/update.go:130-168 -- autoCompleteParentIfTerminal function definition with updated doc comment
  - /Users/leeovery/Code/tick/internal/cli/update.go:154 -- ApplySystemTransition call within autoCompleteParentIfTerminal
  - /Users/leeovery/Code/tick/internal/cli/update.go:380 -- single call site of autoCompleteParentIfTerminal
- Notes:
  - Zero references to ApplyWithCascades in any .go file across the entire codebase (confirmed via ripgrep)
  - Zero references to evaluateRule3 in any .go file (only in .workflows/ docs and .tick/tasks.jsonl)
  - create.go:226 correctly uses validateAndReopenParent (shared helper), which internally calls ApplySystemTransition

TESTS:
- Status: Adequate
- Coverage:
  - "it transitions task to in_progress via tick start" (transition_test.go:31) -- exercises RunTransition -> ApplyUserTransition path
  - "it reopens done parent when reparenting to it" (update_test.go:991) -- exercises validateAndReopenParent -> ApplySystemTransition path, verifies parent status changes to open
  - "it triggers Rule 3 on original parent when reparenting away" (update_test.go:1031) -- exercises autoCompleteParentIfTerminal -> ApplySystemTransition path, verifies parent auto-completes to done
  - All three tests verify persisted state (reading JSONL) and stdout output
- Notes: Tests are focused and not over-tested. Each covers a distinct call site path with meaningful assertions on both persisted state and CLI output.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, error wrapping with %w, doc comments on exported/helper functions.
- SOLID principles: Good. autoCompleteParentIfTerminal has a single responsibility (evaluate and apply Rule 3 auto-completion). The user/system split cleanly separates concerns.
- Complexity: Low. autoCompleteParentIfTerminal is straightforward: evaluate completion, find parent, apply transition.
- Modern idioms: Yes. Idiomatic Go patterns throughout.
- Readability: Good. The rename from evaluateRule3 to autoCompleteParentIfTerminal is self-documenting. Doc comment at update.go:130-133 accurately describes behavior.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
