TASK: Refactor ApplyWithCascades into user/system wrappers with failing test

ACCEPTANCE CRITERIA:
- ApplyWithCascades is unexported as applyWithCascades with signature (sm StateMachine) applyWithCascades(tasks []Task, target *Task, action string, auto bool) (TransitionResult, []CascadeChange, error)
- ApplyUserTransition is exported with signature (sm StateMachine) ApplyUserTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- ApplySystemTransition is exported with signature (sm StateMachine) ApplySystemTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- All 18 existing subtests pass unchanged under ApplyUserTransition
- New test verifies ApplyUserTransition records Auto: false on primary target
- New test verifies ApplySystemTransition records Auto: true on primary target
- Cascade transitions still record Auto: true regardless of which wrapper is used
- go test ./internal/task/ passes with zero failures

STATUS: Complete

SPEC CONTEXT: ApplyWithCascades hardcoded Auto: false on the primary target's TransitionRecord, which was correct for user-initiated transitions but incorrect for system-initiated callers (validateAndReopenParent for Rule 6, evaluateRule3 for Rule 3 reparent). The fix is to parameterize the auto flag via an unexported applyWithCascades(auto bool) with two semantic wrappers: ApplyUserTransition (auto=false) and ApplySystemTransition (auto=true).

IMPLEMENTATION:
- Status: Implemented
- Location: internal/task/apply_cascades.go:1-115
  - ApplyUserTransition: lines 9-11
  - ApplySystemTransition: lines 17-19
  - applyWithCascades: lines 34-115
- Notes: All signatures match acceptance criteria exactly. The auto parameter is correctly used at line 59 for the primary target's TransitionRecord. Cascade transitions remain hardcoded Auto: true at line 101. No exported ApplyWithCascades remains anywhere in the task package. Doc comments on all three functions are accurate and complete. Call sites in internal/cli/ have already been updated to use the new wrappers (this is technically task 1-2 scope, but does not harm task 1-1).

TESTS:
- Status: Adequate
- Coverage:
  - 18 existing subtests migrated to ApplyUserTransition with no assertion changes (lines 41-446)
  - New test "it records auto=false on primary target for user transition" (lines 448-462) verifies primary Auto: false and cascade Auto: true
  - New test "it records auto=true on primary target for system transition" (lines 464-477) verifies primary Auto: true via ApplySystemTransition
  - Total: 20 subtests (18 migrated + 2 new)
  - assertTransition helper (lines 8-23) used for concise auto-flag assertions with t.Helper()
- Notes: Tests are focused and non-redundant. The new auto=false test includes a cascade assertion which also covers the "cascades still Auto: true regardless of wrapper" acceptance criterion. The auto=true test simulates the Rule 6 scenario (done parent reopen) which is the actual bug being fixed. The shared code path for cascades means testing cascade Auto: true under one wrapper implicitly covers the other, which is appropriate.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, "it does X" naming, t.Helper on assertTransition, StateMachine value receiver, error wrapping not needed here
- SOLID principles: Good -- ApplyUserTransition and ApplySystemTransition encode caller intent in their names (open/closed), applyWithCascades remains the single implementation (DRY), the auto parameter is properly encapsulated
- Complexity: Low -- wrappers are one-liners delegating to the core function, no additional branching
- Modern idioms: Yes -- idiomatic Go method wrappers
- Readability: Good -- doc comments clearly explain the user vs system distinction and what Auto means in each context
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The system transition test (line 464) does not explicitly verify that cascades produced by ApplySystemTransition also record Auto: true. This is adequately covered by the shared code path and the existing test at line 291, but a comment in the system transition test noting this shared coverage could help future readers understand why it is omitted.
