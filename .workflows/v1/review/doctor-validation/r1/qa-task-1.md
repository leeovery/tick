TASK: Check Interface & Diagnostic Runner

ACCEPTANCE CRITERIA:
- Check interface defined with Run method returning []CheckResult
- CheckResult struct has Name, Passed, Severity, Details, Suggestion fields
- Severity type with SeverityError and SeverityWarning constants
- DiagnosticRunner registers checks and runs all of them via RunAll
- DiagnosticReport collects all results with HasErrors(), ErrorCount(), WarningCount() accessors
- Zero registered checks produces empty report (not an error)
- Runner never short-circuits -- all checks run regardless of prior failures
- Single check returning multiple results produces multiple entries in report
- Tests written and passing for all edge cases (zero, all-pass, all-fail, mixed, multi-result)

STATUS: Complete

SPEC CONTEXT: The specification requires doctor to "run all checks" (design principle #4) -- it completes all validations before reporting, never stops early. The runner is the enforcement point for this guarantee. Doctor performs errors (exit code 1) and warnings (exit code 0 if only warnings). The Check interface must support returning multiple results because "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details."

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/doctor.go:1-107
- Notes: All acceptance criteria met. The implementation is clean and complete:
  - Check interface (line 41) with `Run(ctx context.Context, tickDir string) []CheckResult` -- note the `tickDir` parameter was added explicitly per Phase 4 task 4-2, which refined the original plan's `ctx`-based approach. This is an improvement, not drift.
  - CheckResult struct (lines 23-34) with all five required fields: Name, Passed, Severity, Details, Suggestion.
  - Severity type (lines 10-17) with SeverityError and SeverityWarning constants.
  - DiagnosticRunner (lines 84-107) with Register() and RunAll() methods. RunAll iterates all checks without short-circuiting.
  - DiagnosticReport (lines 45-80) with HasErrors(), ErrorCount(), WarningCount() accessors.
  - Zero checks returns empty DiagnosticReport (RunAll with empty slice returns DiagnosticReport{Results: nil}).
  - RunAll appends all results from all checks via `...` spread, preserving registration order and supporting multi-result checks.

TESTS:
- Status: Adequate
- Coverage: All 14 test cases specified in the task plan are implemented in /Users/leeovery/Code/tick/internal/doctor/doctor_test.go:1-284. Specifically:
  - "it returns empty report when zero checks are registered" (line 50) -- verifies 0 results, HasErrors false, ErrorCount 0, WarningCount 0
  - "it runs a single passing check and returns one result with Passed true" (line 68)
  - "it runs a single failing check and returns one result with Passed false" (line 84) -- also verifies Details and Suggestion non-empty
  - "it runs all checks when all pass -- report has no errors" (line 103)
  - "it runs all checks when all fail -- report collects all failures" (line 118)
  - "it runs all checks with mixed pass/fail -- report contains both" (line 136)
  - "it does not short-circuit -- failing check does not prevent subsequent checks from running" (line 164) -- uses `called` flag on stub
  - "it preserves registration order in results" (line 186)
  - "it collects multiple results from a single check" (line 204)
  - "HasErrors returns true when any error-severity result has Passed false" (line 221)
  - "HasErrors returns false when only warnings exist" (line 233)
  - "ErrorCount counts only error-severity failures" (line 245)
  - "WarningCount counts only warning-severity failures" (line 259)
  - "HasErrors returns false for empty report" (line 272)
- Edge cases covered: zero checks, all-pass, all-fail, mixed pass/fail, multi-result from single check, warnings-only, empty report.
- Tests use stub checks (stubCheck with `called` flag) which is the right level of abstraction -- not over-mocked, just enough to verify runner behavior.
- Tests would fail if the feature broke (e.g., short-circuit logic added, results not collected, severity filtering wrong).

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, error wrapping style consistent. No testify. Package comment present. Exported functions/types documented.
- SOLID principles: Good. Check interface is minimal (single method). DiagnosticRunner has single responsibility (register and run). DiagnosticReport has single responsibility (collect and query). Open/closed -- new checks implement the Check interface without modifying runner.
- Complexity: Low. RunAll is a simple loop with append. Report accessors are straightforward iterations. No branching complexity.
- Modern idioms: Yes. Uses `range count` (Go 1.22+) in test helper newMultiResultCheck (line 38). context.Context parameter on Check interface. Functional constructor NewDiagnosticRunner.
- Readability: Good. Clear naming, well-documented types, straightforward control flow. Package doc comment explains purpose.
- Issues: None found.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `tickDir` parameter on `Check.Run` was originally planned to be carried via context but was later changed to an explicit parameter in Phase 4 (task 4-2). This is a design improvement and the task plan was updated accordingly. No concern here.
- Passing checks default to `SeverityError` in the `newPassingCheck` helper (line 22). This is fine since severity only matters for failing checks, but it could be slightly more explicit/intentional. Very minor.
