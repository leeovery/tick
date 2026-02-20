TASK: Extract buildKnownIDs helper to eliminate 3-file duplication

ACCEPTANCE CRITERIA:
- buildKnownIDs helper exists and is used by all three check files
- No inline knownIDs map construction remains in any check file
- All existing tests pass without modification

STATUS: Complete

SPEC CONTEXT: This is a pure refactoring task from analysis cycle 3 (Phase 6). The knownIDs map construction (make(map[string]struct{}, len(tasks)) + range loop) was duplicated across orphaned_parent.go, orphaned_dependency.go, and dependency_cycle.go. The task extracts this into a shared helper in helpers.go. No behavioral change expected.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/helpers.go:3-10
- The `buildKnownIDs(tasks []TaskRelationshipData) map[string]struct{}` function is defined exactly as specified
- Called from /Users/leeovery/Code/tick/internal/doctor/orphaned_parent.go:23
- Called from /Users/leeovery/Code/tick/internal/doctor/orphaned_dependency.go:23
- Called from /Users/leeovery/Code/tick/internal/doctor/dependency_cycle.go:26
- No inline knownIDs map construction remains in any check file. Confirmed via grep: the only `make(map[string]struct{})` calls in check files are in helpers.go:5 (the helper itself) and dependency_cycle.go:54 (the `seen` map for cycle deduplication, which is unrelated to knownIDs)
- Notes: Implementation matches the plan exactly. The helper is co-located with the existing `fileNotFoundResult` helper in helpers.go, which is the natural home.

TESTS:
- Status: Adequate
- Coverage: No new tests were added, which is correct for a pure refactor with no behavioral change. All three existing test suites (TestOrphanedParentCheck, TestOrphanedDependencyCheck, TestDependencyCycleCheck) thoroughly exercise the knownIDs lookup behavior through their normal test cases (valid refs, orphaned refs, empty files, etc.)
- Notes: The task explicitly states "no new tests needed -- this is a pure refactor with no behaviour change." This is appropriate. The existing tests cover the behavior that relies on buildKnownIDs: orphaned parent detection (11 subtests), orphaned dependency detection (17 subtests), and dependency cycle detection (20 subtests). If buildKnownIDs were broken, multiple tests across all three suites would fail.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, unexported helper (package-private), Go doc comment on the function, consistent with the existing fileNotFoundResult helper pattern in the same file.
- SOLID principles: Good. Single responsibility -- the helper does one thing (builds a set of known IDs). DRY is the primary motivation and is fully achieved.
- Complexity: Low. The function is 5 lines of straightforward map construction.
- Modern idioms: Yes. Idiomatic Go: pre-sized map with capacity hint, struct{} for set values.
- Readability: Good. Function name clearly communicates intent. The doc comment explains what it returns. Callers read as `knownIDs := buildKnownIDs(tasks)` which is more expressive than the inlined 3-line construction.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. This is a clean, well-scoped refactor that achieves its goal precisely.
