TASK: Extract assertTransition test helper in apply_cascades_test.go

ACCEPTANCE CRITERIA:
- An assertTransition helper exists in apply_cascades_test.go with t.Helper() call
- All previous inline assertion blocks are replaced with calls to the helper
- No assertion coverage is lost -- same fields are checked with same expected values
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: The analysis identified 10+ repeated 4-assertion blocks (check Transitions length, check From, check To, check Auto) across subtests in apply_cascades_test.go. Each instance was 8-10 lines of identical structure. The task was to extract these into a shared helper to reduce duplication.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/task/apply_cascades_test.go:8-23
- Notes: The helper function `assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool)` is defined at the top of the file after imports, exactly matching the prescribed signature. It calls `t.Helper()` as the first line. It performs a bounds check with `t.Fatalf` (correctly fatal to prevent index-out-of-range panics), then checks From, To, and Auto fields with `t.Errorf`. The helper is called 12 times across the test file, covering all subtests that verify transition records. No inline assertion blocks remain -- the only direct `.Transitions[` accesses outside the helper are: (1) line 74 checking `Transitions[0].At.IsZero()` (timestamp check, correctly left inline since the helper doesn't cover At), and (2) lines 238 and 329 checking `len(tasks[...].Transitions) != 0` in error-case tests (correctly left inline since they verify zero transitions, not transition contents).

TESTS:
- Status: Adequate
- Coverage: The helper is exercised by 12 call sites across 7 subtests. All existing TestApplyUserTransition and TestApplySystemTransition subtests use the helper where transition assertions are needed. Error-case tests that verify no transitions were recorded remain appropriately inline.
- Notes: No test coverage was lost. The same fields (From, To, Auto) are checked with the same expected values. The bounds check on Transitions length is preserved via the Fatalf guard.

CODE QUALITY:
- Project conventions: Followed. Uses `t.Helper()` as required by project CLAUDE.md. Uses stdlib testing only (no testify). Follows existing helper patterns seen across the codebase.
- SOLID principles: Good. Single responsibility -- the helper does one thing (assert a transition record).
- Complexity: Low. Straight-line assertion code with clear error messages.
- Modern idioms: Yes. Idiomatic Go test helper pattern.
- Readability: Good. Self-documenting function name, clear parameter names, formatted error messages include the index for disambiguation when multiple transitions exist on a task.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
