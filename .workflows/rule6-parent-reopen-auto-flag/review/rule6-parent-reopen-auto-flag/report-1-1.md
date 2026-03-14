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

SPEC CONTEXT: ApplyWithCascades hardcodes Auto: false on the primary target's TransitionRecord. This is correct for user-initiated transitions but incorrect for system-initiated callers (validateAndReopenParent for Rule 6, evaluateRule3 for Rule 3). The fix adds an auto bool parameter to the now-unexported applyWithCascades, exposed via two semantic wrappers: ApplyUserTransition (auto=false) and ApplySystemTransition (auto=true).

IMPLEMENTATION:
- Status: Implemented
- Location: internal/task/apply_cascades.go:1-115
- Notes: All acceptance criteria met:
  - applyWithCascades is unexported at line 34 with correct signature including auto bool parameter
  - ApplyUserTransition exported at line 9, delegates with auto=false
  - ApplySystemTransition exported at line 17, delegates with auto=true
  - Line 59: primary target uses the auto parameter for TransitionRecord
  - Line 101: cascade transitions hardcode Auto: true regardless of wrapper
  - No remaining references to exported ApplyWithCascades in any .go file
  - Doc comments on all three functions accurately describe behavior
  - Call sites already updated: transition.go:37 uses ApplyUserTransition, helpers.go:124 and update.go:154 use ApplySystemTransition

TESTS:
- Status: Adequate
- Coverage:
  - 18 existing subtests migrated to ApplyUserTransition with no assertion changes
  - "it records auto=false on primary target for user transition" (line 448): verifies primary target gets Auto=false and cascade gets Auto=true via ApplyUserTransition
  - "it records auto=true on primary target for system transition" (line 464): verifies primary target gets Auto=true via ApplySystemTransition
  - "it records auto transitions on all cascaded tasks" (line 291): pre-existing test also validates auto=false on primary and auto=true on cascade
  - assertTransition helper (line 8) properly validates From, To, and Auto fields with t.Helper()
- Notes: The new auto=false test (line 448) and the existing "it records auto transitions on all cascaded tasks" test (line 291) overlap significantly -- both verify ApplyUserTransition sets auto=false on primary and auto=true on cascades. This is minor and not blocking since the newer test was explicitly required by the plan's acceptance criteria.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, t.Helper on helpers, error wrapping, method receiver pattern matches existing code
- SOLID principles: Good -- single responsibility is clean; the user/system split follows open/closed by allowing new entry points without modifying the core cascade engine
- Complexity: Low -- wrappers are trivial one-liners delegating to the core function
- Modern idioms: Yes -- uses standard Go patterns (unexported core + exported semantic wrappers)
- Readability: Good -- doc comments are accurate and clearly explain the auto flag semantics; wrapper naming makes intent obvious at call sites
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Minor test overlap: "it records auto transitions on all cascaded tasks" (line 291) and "it records auto=false on primary target for user transition" (line 448) test the same auto=false/true behavior on user transitions. Both were required by the plan so this is acceptable.
