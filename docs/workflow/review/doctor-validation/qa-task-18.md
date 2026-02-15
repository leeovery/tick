TASK: Extract fileNotFoundResult helper for repeated tasks.jsonl-not-found error

ACCEPTANCE CRITERIA:
- A single fileNotFoundResult function exists in the doctor package
- At least 8 of the 9 check files use this helper instead of constructing the literal inline
- The helper is unexported (lowercase) since it is internal to the doctor package
- All existing tests pass
- Unit test for fileNotFoundResult verifying it returns the expected CheckResult with correct fields

STATUS: Complete

SPEC CONTEXT: Doctor is a diagnostic-only command that runs all checks without short-circuiting. Nine error checks plus one warning check validate data store integrity. The "tasks.jsonl not found" case is a common early-return across nearly all checks, producing an error-severity result with a suggestion to run tick init.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/helpers.go:14-22
- Notes: Helper is unexported, returns []CheckResult with Name set from parameter, Passed=false, Severity=SeverityError, Details="tasks.jsonl not found", Suggestion="Run tick init or verify .tick directory". All 9 check files use the helper (exceeding the "at least 8" requirement): jsonl_syntax.go:22, duplicate_id.go:28, id_format.go:25, orphaned_parent.go:20, orphaned_dependency.go:20, self_referential_dep.go:20, dependency_cycle.go:23, child_blocked_by_parent.go:24, parent_done_open_children.go:23. CacheStalenessCheck correctly retains its custom message ("tasks.jsonl not found or unreadable: %v") at cache_staleness.go:33 and does not use this helper. No inline "tasks.jsonl not found" result literals remain in any production code file.

TESTS:
- Status: Adequate
- Coverage: /Users/leeovery/Code/tick/internal/doctor/helpers_test.go contains a table-driven test (TestFileNotFoundResult) with 3 subtests covering different check names. Each subtest verifies all 5 fields: Name, Passed, Severity, Details, Suggestion. Additionally, all 9 existing check-file test suites still include "missing tasks.jsonl" test cases that indirectly exercise the helper.
- Notes: Tests are focused and necessary. The 3 subtests confirm the parameterized Name field works correctly while verifying the static fields. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, table-driven pattern, no testify. Follows error wrapping and naming conventions from CLAUDE.md.
- SOLID principles: Good. Single responsibility -- helper does exactly one thing. The DRY violation of 9 identical result constructors is eliminated.
- Complexity: Low. Function is a single return statement constructing a struct literal.
- Modern idioms: Yes. Idiomatic Go helper function.
- Readability: Good. Clear function name, doc comment explains purpose and parameter usage.
- Issues: None. The file also contains buildKnownIDs (from task doctor-validation-6-1), which is a reasonable colocation of package-internal helpers.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
