TASK: Migration Output - Per-Task & Summary (migration-1-4)

ACCEPTANCE CRITERIA:
- Presenter exists in internal/migrate/ and writes to an io.Writer
- Header line prints "Importing from <provider>..." using the provider name
- Each successful result prints "  checkmark Task: <title>" (two-space indent, checkmark, task prefix)
- Summary line prints "Done: N imported, 0 failed" with the correct count of successful results
- A blank line separates the last per-task line from the summary line
- Zero results produce header + blank line + summary with "0 imported, 0 failed"
- Long titles are printed in full without truncation
- All output goes to the provided io.Writer (not hardcoded to stdout)
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification defines the output format for migration: a header line ("Importing from beads..."), per-task checkmark lines with two-space indent, and a summary line ("Done: N imported, M failed"). Phase 1 focuses on success-path output. The spec also shows failure lines (cross mark) and a failure detail section, which are Phase 2 scope but the implementation forward-compatibly handles them.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/migrate/presenter.go (82 lines)
- Key functions:
  - WriteHeader (line 10): prints "Importing from <provider>..." with optional [dry-run] suffix
  - WriteResult (line 21): prints "  checkmark Task: <title>" for success, "  cross Task: <title> (skipped: <reason>)" for failure
  - WriteSummary (line 35): counts successes/failures, prints "\nDone: N imported, M failed\n"
  - WriteFailures (line 51): prints failure detail section (Phase 2 addition, works correctly)
  - Present (line 74): orchestrates all four functions in sequence
- All output uses fmt.Fprintf to the provided io.Writer
- No truncation of long titles -- they are printed verbatim
- Zero results handled correctly: header + blank line (from WriteSummary's leading \n) + summary
- The blank line separator between tasks and summary comes from WriteSummary prepending "\n" to the "Done:" line
- Notes: Implementation includes dryRun parameter and WriteFailures function which are Phase 2 scope (migration-2-2, migration-2-3). This is forward-compatible scope addition, not drift -- Phase 1 acceptance criteria are fully met.

TESTS:
- Status: Adequate
- Coverage: All 11 tests specified in the plan are present and accounted for:
  1. TestWriteHeader/"prints Importing from <provider>... with provider name" (line 11)
  2. TestWriteHeader/"provider name in header matches what provider.Name() returns" (line 22)
  3. TestWriteResult/"prints checkmark and task title for successful result" (line 57)
  4. TestWriteResult/"per-task lines are indented with two spaces" (line 69)
  5. TestWriteResult/"long titles are printed in full without truncation" (line 116)
  6. TestWriteSummary/"prints Done: N imported, 0 failed with correct count" (line 130)
  7. TestPresent/"renders full output: header, per-task lines, blank line, summary" (line 293)
  8. TestPresent/"with zero results prints header and summary with zero counts" (line 314)
  9. TestPresent/"with single result prints one task line and count of 1" (line 327)
  10. TestPresent/"with multiple results prints each task on its own line" (line 344)
  11. TestPresent/"summary line is separated from task lines by a blank line" (line 368)
- Edge cases covered:
  - Zero tasks imported: tested (line 314)
  - Long titles (250 chars): tested (line 116)
  - Single result: tested (line 327)
  - Multiple results: tested (line 344)
- Additional tests beyond Phase 1 plan scope (Phase 2 coverage):
  - WriteResult for failed results with cross mark (line 80)
  - WriteResult with empty title fallback (line 104)
  - WriteSummary with mixed success/failure counts (multiple subtests)
  - WriteFailures tests (line 228-289)
  - Present with failures, dry-run variants (lines 382-495)
  - WriteHeader dry-run tests (lines 34-53)
- Notes: Tests are well-structured using subtests. Some WriteSummary tests are somewhat redundant (e.g., "counts only successful results" vs "prints Done: 2 imported, 1 failed for mixed results" test nearly identical logic with similar inputs), but this is minor and the tests are individually fast. Tests correctly use bytes.Buffer for io.Writer injection, matching the plan's approach.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Run subtests, fmt.Errorf for errors. Package-level functions rather than methods on a struct -- appropriate for stateless formatting.
- SOLID principles: Good. Single responsibility (presenter only formats, does not call engine or provider). Open/closed (WriteResult handles both success and failure paths without modification needed). Functions accept io.Writer interface (dependency inversion).
- Complexity: Low. Each function is straightforward with minimal branching. WriteSummary iterates once over results to count. WriteFailures iterates twice (filter then print) but results slices are small.
- Modern idioms: Yes. Idiomatic Go: package-level functions, io.Writer interface, fmt.Fprintf, Unicode escapes for checkmark/cross.
- Readability: Good. Function names clearly describe intent. Doc comments on all exported functions. The code is self-documenting.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- WriteResult line 30 calls r.Err.Error() without a nil check. If a Result{Success: false, Err: nil} were ever constructed, this would panic. The Result type's documentation says Err is "nil on success" but does not explicitly guarantee non-nil on failure. A defensive nil check (e.g., fallback to "unknown error") would be safer. Low risk since the engine always sets Err on failure.
- WriteSummary (line 35-46) and WriteFailures (line 51-68) both iterate over results to classify success/failure. Could be a single pass, but results slices are small in practice and the current approach is more readable. Not worth changing.
- TestWriteSummary has some redundancy: subtests at lines 130, 146, 162, 179, 195, 211 test variations of the same counting logic. The first three subtests (all success, mixed, multiple failures) are sufficient to verify the counting; the remaining three do not add meaningful coverage. This is minor over-testing but not worth removing.
