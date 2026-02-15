TASK: Self-Referential Dependency Check (doctor-validation-3-3)

ACCEPTANCE CRITERIA:
- [x] SelfReferentialDepCheck implements the Check interface
- [x] Check reuses ParseTaskRelationships from task 3-1 (via getTaskRelationships)
- [x] Passing check returns CheckResult with Name "Self-referential dependencies" and Passed true
- [x] Each self-referential task produces its own failing CheckResult with the task ID in details
- [x] Details follow wording: "tick-{id} depends on itself"
- [x] Self-reference detected even when mixed with other valid dependencies in blocked_by
- [x] Multiple self-referential tasks each produce separate failing results
- [x] Tasks with empty or absent blocked_by are not flagged
- [x] Duplicate self-references in the same task's blocked_by produce one error (per-task, not per-entry)
- [x] Unparseable lines skipped by parser -- not flagged as self-referential
- [x] Missing tasks.jsonl returns error-severity failure with init suggestion
- [x] Suggestion is "Manual fix required" for self-referential dependency errors
- [x] All failures use SeverityError
- [x] Check is read-only -- never modifies tasks.jsonl
- [x] Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: Specification Error #7: "Self-referential dependencies -- Task depends on itself." Fix suggestion table maps this to "Manual fix required." Doctor lists each error individually. Doctor is diagnostic only and never modifies data.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/self_referential_dep.go:1-51
- Registration: /Users/leeovery/Code/tick/internal/cli/doctor.go:25
- Notes: Clean, concise implementation. The struct is stateless (no fields), implements Check interface via Run(ctx, tickDir). Uses getTaskRelationships (which wraps the shared JSONL reader from task 3-1/4-1) and fileNotFoundResult helper (from task 4-3). The break-on-first-match pattern in the inner loop correctly deduplicates multiple self-references per task (lines 28-29). No drift from plan.

TESTS:
- Status: Adequate
- Coverage: All 17 tests from the plan are present and map directly to the specified test list:
  - Passing: no self-refs, empty file, deps but none self-referential, empty blocked_by
  - Failing: task lists itself, self-ref among valid deps, multiple self-ref tasks, only self-reference (implicit in "lists itself" test)
  - Details/format: task ID in details, wording verification, Name consistency, SeverityError, suggestion text
  - Edge cases: duplicate self-refs deduplicated, unparseable lines skipped, missing file, no ID normalization, read-only verification
- Notes: Tests are well-structured using t.Run subtests. Table-driven tests used appropriately for Name and Severity checks across multiple scenarios. The assertReadOnly helper is reused from shared test utilities. No over-testing -- each test verifies a distinct behavior or acceptance criterion.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.TempDir for isolation, t.Helper on helpers, fmt.Errorf error wrapping pattern (though not needed here), Check interface compliance.
- SOLID principles: Good. Single responsibility (only detects self-references, cycle detection is in 3-4). Implements Check interface (LSP/ISP). Depends on abstractions (getTaskRelationships, fileNotFoundResult).
- Complexity: Low. Simple linear scan with inner loop. The break-on-first-match is a clean way to deduplicate without extra data structures.
- Modern idioms: Yes. Idiomatic Go -- no unnecessary allocations, nil slice for failures (append handles nil), clean control flow.
- Readability: Good. The implementation is 51 lines including comments and is self-documenting. The selfRef boolean + break pattern clearly communicates intent.
- Issues: None found.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
