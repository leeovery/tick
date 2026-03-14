TASK: Extract assertTransition test helper in apply_cascades_test.go

ACCEPTANCE CRITERIA:
- An assertTransition helper exists in apply_cascades_test.go with t.Helper() call
- All previous inline assertion blocks are replaced with calls to the helper
- No assertion coverage is lost -- same fields are checked with same expected values
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: Analysis cycle 1 identified a repeated 4-assertion block (length, From, To, Auto) appearing 10+ times across subtests in apply_cascades_test.go. Each instance was 8-10 lines of identical structure with only expected values varying. Extraction into a helper reduces duplication while preserving assertion coverage.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/task/apply_cascades_test.go:8-23 (helper definition)
- Notes: Helper signature `func assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool)` matches the analysis spec exactly. Includes `t.Helper()` on line 9. Uses `t.Fatalf` for the length guard (fails fast if transition missing) and `t.Errorf` for field comparisons (reports all mismatches). 12 call sites across 7 subtests replace all previous inline assertion blocks. The one-off `At.IsZero()` check (line 74) is correctly left inline since it was not part of the repeated pattern. Error-path `len(Transitions) != 0` checks (lines 238, 329) are also correctly left inline as they assert absence of transitions, not a specific transition's fields.

TESTS:
- Status: Adequate
- Coverage: All 18 existing subtests under TestApplyUserTransition and the 1 TestApplySystemTransition subtest are preserved and use the helper where transition assertions are needed. The helper is invoked 12 times across the file, covering all cascade scenarios (upward start, downward cancel, chained completion, reopen chains, auto flag verification for both user and system transitions).
- Notes: No assertion coverage was lost. The same fields (From, To, Auto) are checked with identical expected values. The length guard is implicit in the helper via the `t.Fatalf` on insufficient transition count. Test adequacy is inherently validated by the fact that all existing tests pass -- the task is a pure refactoring with no behavioral change.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Helper() on helpers, t.Run() subtests, naming follows existing patterns.
- SOLID principles: Good. Single-responsibility helper with clear purpose.
- Complexity: Low. Helper is straightforward with no branching beyond the length guard.
- Modern idioms: Yes. Idiomatic Go test helper pattern.
- Readability: Good. Helper name clearly communicates intent. Parameters are well-ordered (t, task, index, expected values). Error messages include the transition index for debuggability.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
