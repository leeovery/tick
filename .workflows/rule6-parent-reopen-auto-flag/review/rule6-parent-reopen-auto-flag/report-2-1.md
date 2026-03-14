TASK: Add parent not-found guard in autoCompleteParentIfTerminal

ACCEPTANCE CRITERIA:
- parentIdx is initialized to -1, not zero-value 0
- An explicit guard if parentIdx < 0 { return nil } exists between the search loop and the ApplySystemTransition call
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: Analysis finding identified a latent bug where autoCompleteParentIfTerminal declared `var parentIdx int` (zero-value 0). If the parent search loop exited without finding the parent, the function would proceed to call ApplySystemTransition on tasks[0] -- an unrelated task. Currently unreachable because EvaluateParentCompletion returns shouldComplete=false when the parent doesn't exist, but correctness depended on upstream behavior never changing.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/update.go:142 (parentIdx := -1), internal/cli/update.go:149-151 (guard)
- Notes: Clean, minimal change. Sentinel value -1 is idiomatic Go. Guard returns nil exactly as specified. No other code was modified. The fix makes autoCompleteParentIfTerminal self-contained and defensively correct regardless of upstream changes.

TESTS:
- Status: Adequate
- Coverage: Existing TestUpdate tests in internal/cli/update_test.go cover all reachable code paths of autoCompleteParentIfTerminal, including Rule 3 auto-completion with done, cancelled, mixed results, no-trigger cases, clearing parent, and reparent combinations. The guard protects an unreachable path, so no new test is needed -- adding one would require mocking internal state inconsistencies.
- Notes: No under-testing or over-testing. The acceptance criteria correctly identifies that existing tests passing confirms no behavioral regression.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, error wrapping style, function naming conventions all consistent
- SOLID principles: Good -- the guard makes the function self-contained (single responsibility, no hidden dependency on upstream behavior)
- Complexity: Low -- two-line addition, no new branches in practice
- Modern idioms: Yes -- sentinel value with negative check is standard Go pattern
- Readability: Good -- the guard is immediately clear in intent
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
