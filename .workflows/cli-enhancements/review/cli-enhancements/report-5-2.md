TASK: cli-enhancements-6-3 -- ParseRefs should delegate to ValidateRefs

ACCEPTANCE CRITERIA:
- ParseRefs delegates validation to ValidateRefs instead of reimplementing it
- All existing ParseRefs tests pass without modification

STATUS: Complete

SPEC CONTEXT: External references validation is defined in the spec with rules: non-empty, no commas, no whitespace, max 200 chars, max 10 per task, silent dedup. The validation logic lives in ValidateRef (single ref) and ValidateRefs (slice). ParseRefs is the comma-separated input parser used by CLI flags. The analysis cycle 2 identified that ParseRefs was reimplementing the validation loop and count check that ValidateRefs already provides, with identical error format strings. The fix was to have ParseRefs delegate to ValidateRefs.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/refs.go:65-78
- Notes: ParseRefs now has a clean 3-step flow: (1) check empty input (line 66-68), (2) split and deduplicate (lines 70-71), (3) delegate to ValidateRefs (lines 73-75). This matches the exact code prescribed in the analysis task. The function dropped from the original ~19 lines to ~13 lines (including comment). The cycle-3 duplication analysis confirms: "The cycle-2 fixes (queryStringColumn/queryRelatedTasks helpers, ParseRefs delegation) addressed the significant patterns." One minor note: DeduplicateRefs is called twice in the pipeline (once in ParseRefs line 71, once inside ValidateRefs line 48), but this was explicitly acknowledged in the analysis as "harmless no-op on already-deduped input" and is acceptable.

TESTS:
- Status: Adequate
- Coverage: TestParseRefs at /Users/leeovery/Code/tick/internal/task/refs_test.go:123-179 has 5 subtests covering: comma splitting, whitespace trimming, deduplication, ref-with-whitespace rejection, empty input rejection. These are the original tests, unmodified, which is exactly what the acceptance criteria requires. ValidateRefs has its own comprehensive test suite (TestValidateRefs, 5 subtests at lines 75-121) covering dedup, count limits, per-ref validation, and empty list handling.
- Notes: The refactoring is purely structural (no behavioral change), so existing tests providing the same coverage is correct. No new tests needed. Not over-tested -- each test verifies a distinct behavior. Not under-tested -- the delegation path is fully exercised through the existing tests since ParseRefs now calls ValidateRefs which has its own tests.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming, error wrapping with fmt.Errorf, functional composition.
- SOLID principles: Good. This refactoring improves adherence to DRY and SRP -- validation logic now lives in one place (ValidateRefs) rather than being duplicated across ParseRefs and ValidateRefs.
- Complexity: Low. ParseRefs is a linear 3-step function with no branching beyond the two error checks.
- Modern idioms: Yes. Clean Go composition pattern.
- Readability: Good. The function comment accurately describes what it does. The code is self-documenting.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The double DeduplicateRefs call (once in ParseRefs, once inside ValidateRefs) is a minor inefficiency but was explicitly accepted in the analysis as harmless. If performance of ref parsing ever mattered, ValidateRefs could accept an already-deduped flag or a separate validateRefsNoDup variant, but this is not worth the complexity for the current use case.
