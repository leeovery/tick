TASK: Data Integrity Check Registration (doctor-validation-2-4)

ACCEPTANCE CRITERIA:
- [x] JsonlSyntaxCheck registered in the doctor command handler
- [x] IdFormatCheck registered in the doctor command handler
- [x] DuplicateIdCheck registered in the doctor command handler
- [x] All four checks (including existing CacheStalenessCheck) run in a single tick doctor invocation
- [x] Each check receives the correct .tick/ directory path
- [x] All four checks run regardless of individual results (no short-circuit)
- [x] Exit code 0 when all four checks pass
- [x] Exit code 1 when any check produces an error-severity failure
- [x] Summary count reflects total errors across all four checks
- [x] Output shows results for all four checks (passing and failing)
- [x] Doctor remains read-only with four checks registered (no data modification)
- [x] Tests written and passing for all edge cases (all pass, all new fail, mixed, empty file)

STATUS: Complete

SPEC CONTEXT:
The specification requires all checks to run in a single invocation (design principle #4: "Run all checks -- Doctor completes all validations before reporting, never stops early"). The output format uses checkmark/cross markers with a summary count. Exit code 0 when all pass, 1 when any error. Phase 2 specifically adds JSONL syntax, ID uniqueness, and ID format checks alongside the existing cache staleness check. This task is pure wiring -- registering the three new checks in the command handler.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/doctor.go:17-42
- Notes: All four Phase 2 checks are registered at lines 19-22 in the correct order (CacheStalenessCheck, JsonlSyntaxCheck, IdFormatCheck, DuplicateIdCheck). The tickDir is passed to `runner.RunAll(ctx, tickDir)` at line 37 and propagated to each check's `Run(ctx, tickDir)` method. The pre-scanned JSONL lines are loaded once at line 32-35 and passed via context, which the three new checks consume through `getJSONLines(ctx, tickDir)`. No new types or interfaces were created -- this is purely wiring, as specified. The implementation also includes 6 additional Phase 3 relationship checks (lines 23-28), which is expected since the file reflects the completed state including later phases. The Phase 2 registration is correct and not affected by the additional checks.

TESTS:
- Status: Adequate
- Coverage: The `TestDoctorFourChecks` test group at /Users/leeovery/Code/tick/internal/cli/doctor_test.go:307-539 covers all 15 test cases specified in the task plan:
  - "it registers all four checks" (line 308): verifies all 4 labels present in output
  - "it runs all four checks in a single tick doctor invocation" (line 320): counts 10 checkmarks (updated for all phases)
  - "it exits 0 when all four checks pass" (line 332)
  - "it exits 1 when only the JSONL syntax check fails" (line 342)
  - "it exits 1 when only the ID format check fails" (line 353)
  - "it exits 1 when only the duplicate ID check fails" (line 364)
  - "it exits 1 when all three new checks fail but cache check passes" (line 376)
  - "it exits 1 when all four checks fail" (line 393)
  - "it exits 1 when cache check fails but all three new checks pass" (line 408)
  - "it reports mixed results correctly" (line 418): checks specific checkmark/cross markers
  - "it displays results for all four checks in output" (line 440)
  - "it shows correct summary count reflecting errors from all checks combined" (line 453): expects "3 issues found."
  - "it runs all four checks even when the first check fails (no short-circuit)" (line 472)
  - "it handles empty tasks.jsonl" (line 487): verifies exit 0 and all labels present
  - "it does not modify tasks.jsonl or cache.db (read-only invariant)" (line 504)
- Edge cases from the plan are all covered:
  - All new checks pass alongside passing cache check: tested via "exits 0 when all four checks pass"
  - All new checks fail alongside passing cache check: tested via "exits 1 when all three new checks fail but cache check passes"
  - Mixed results across all four checks: tested via "reports mixed results correctly" and "shows correct summary count"
  - Empty tasks.jsonl: tested via "handles empty tasks.jsonl"
- Notes: The test at line 320 counts 10 checkmarks rather than 4, reflecting that all 10 checks are now registered. This is correct for the current state but means the test title ("all four checks") is slightly misleading. This is minor -- the test still validates that the Phase 2 checks run. Tests are end-to-end via `runDoctor` helper which exercises the full App dispatch path, making them reliable regression tests.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.TempDir for isolation, t.Helper on helpers. Error wrapping with fmt.Errorf. Matches CLAUDE.md patterns.
- SOLID principles: Good. Single responsibility -- the doctor handler only wires checks; each check is its own type. Open/closed -- new checks added by Register() without modifying runner internals. Dependency inversion -- checks implement the Check interface.
- Complexity: Low. The RunDoctor function is a straightforward sequence: create runner, register checks, scan JSONL, run all, format, return exit code. No branching complexity.
- Modern idioms: Yes. Context propagation for shared JSONL data. Pointer receivers for Check implementations. Functional separation of concerns (runner, formatter, exit code logic are separate functions).
- Readability: Good. The registration block at lines 19-28 is clean and self-documenting. Each line registers one check, making the set visible at a glance. The doc comment at lines 12-15 enumerates all 10 checks.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The test "it runs all four checks in a single tick doctor invocation" (line 320) counts 10 checkmarks and the comment says "should have 10 passing checks (4 original + 6 relationship/hierarchy)". The test name says "four checks" but verifies 10. This is a minor naming inconsistency from later phases extending the test expectation. Not blocking since the test is functionally correct.
- The test "it shows correct summary count reflecting errors from all checks combined" (line 453) uses content that triggers syntax, ID format, and duplicate errors. The expected count "3 issues found." assumes each check produces exactly 1 error from that content. This is a reasonable assumption verified by the individual check implementations, but coupling test expectations across checks makes the test slightly brittle to changes in individual check behavior. This is inherent to integration tests and acceptable.
