TASK: Add parent not-found guard in autoCompleteParentIfTerminal

ACCEPTANCE CRITERIA:
- parentIdx is initialized to -1, not zero-value 0
- An explicit guard `if parentIdx < 0 { return nil }` exists between the search loop and the ApplySystemTransition call
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: Analysis finding identified a latent bug where `autoCompleteParentIfTerminal` declared `var parentIdx int` (zero-value 0). If the parent-search loop exited without finding the parent, the function would silently call `ApplySystemTransition` on `tasks[0]` -- a completely unrelated task. Currently unreachable because `EvaluateParentCompletion` returns `shouldComplete=false` when the parent doesn't exist, but correctness depended on upstream behavior never changing. The fix makes the function self-contained by using a sentinel value and explicit guard.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/update.go:142 (`parentIdx := -1`), lines 149-151 (`if parentIdx < 0 { return nil }`)
- Notes: Both changes are exactly as specified. The sentinel initialization is on line 142, the guard is on lines 149-151, and the `ApplySystemTransition` call follows on line 154. The guard is correctly positioned between the search loop (lines 143-148) and the transition call. No drift from what was planned.

TESTS:
- Status: Adequate
- Coverage: Existing TestUpdate tests exercise the `autoCompleteParentIfTerminal` code path through 7+ subtests covering Rule 3 scenarios (reparent triggers auto-completion to done, to cancelled, mix of done+cancelled, no trigger when non-terminal children remain, clearing parent, combined Rule 6 + Rule 3, auto=true on auto-completion). These tests all exercise the reachable code path where the parent IS found.
- Notes: No new test was required or expected for this task -- the guard protects an unreachable-but-dangerous path. Writing a unit test for it would require calling the unexported function directly with a fabricated scenario where `EvaluateParentCompletion` returns true but the parent doesn't exist in the slice, which would be testing an impossible state. The existing tests confirming no regression is the correct test strategy here.

CODE QUALITY:
- Project conventions: Followed. Uses idiomatic Go patterns (sentinel value -1, explicit guard, early return nil).
- SOLID principles: Good. The guard makes the function self-contained (no dependency on upstream caller behavior for correctness).
- Complexity: Low. Two lines added, no increase in cyclomatic complexity (the guard is a simple early return).
- Modern idioms: Yes. Short variable declaration `:=` for initialization, standard Go sentinel pattern.
- Readability: Good. The intent is immediately clear -- initialize to -1, search, bail if not found.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
