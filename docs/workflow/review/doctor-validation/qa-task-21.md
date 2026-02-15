TASK: Use DiagnosticReport methods for issue count in FormatReport

ACCEPTANCE CRITERIA:
- FormatReport does not maintain its own counter during iteration
- issueCount is derived from report.ErrorCount() + report.WarningCount()
- All existing format tests pass without modification

STATUS: Complete

SPEC CONTEXT: The spec requires a summary count at the end of doctor output showing total issues. ErrorCount() and WarningCount() are canonical methods on DiagnosticReport that count non-passing results by severity. This refactoring eliminates an independent counting loop in FormatReport, making the issue count a single source of truth.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/format.go:27
- Notes: The local `issueCount := 0` declaration and `issueCount++` increment inside the display loop have been removed. Line 27 now reads `issueCount := report.ErrorCount() + report.WarningCount()`, computed after the display loop and before the switch statement. The rest of FormatReport is unchanged. No independent counting logic remains in the function.

TESTS:
- Status: Adequate
- Coverage: Existing tests in /Users/leeovery/Code/tick/internal/doctor/format_test.go cover all relevant scenarios: zero issues, single error, multiple errors, mixed errors and warnings (lines 174-189 and 139-155), order preservation with mixed severities (lines 191-208). All tests include Severity fields on CheckResult structs, so they exercise the ErrorCount()/WarningCount() code paths. ExitCode tests are unaffected (lines 211-271).
- Notes: No test modification was needed since the refactoring is behavior-preserving. The tests verify output strings which include the issue count summary line, so any regression in the counting logic would cause test failures.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, idiomatic Go patterns.
- SOLID principles: Good. Single source of truth for issue counting (DRY improvement). FormatReport delegates counting responsibility to DiagnosticReport methods rather than reimplementing the logic.
- Complexity: Low. The function is simpler after the refactoring -- no counter variable tracked through the loop.
- Modern idioms: Yes. Clean delegation to receiver methods.
- Readability: Good. The intent of `report.ErrorCount() + report.WarningCount()` is immediately clear and self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
