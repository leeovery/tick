TASK: Output Formatter & Exit Code Logic

ACCEPTANCE CRITERIA:
- Formatter writes checkmark {Name}: OK for each passing result
- Formatter writes X {Name}: {Details} for each failing result
- Suggestion line arrow {Suggestion} appears only when suggestion is non-empty
- Summary line uses correct grammar: "No issues found." / "1 issue found." / "{N} issues found."
- Summary counts all failures (errors + warnings) for display
- Exit code returns 0 when no error-severity failures exist
- Exit code returns 1 when any error-severity failure exists
- Warnings do not affect exit code (exit 0 when only warnings)
- Output order matches result order in DiagnosticReport
- Formatter writes to provided io.Writer (testable without stdout capture)
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines human-readable text output only (no TOON/JSON). Format uses checkmark for passing checks, X for failures with details and suggested action, summary count at end. Exit code 0 means all checks passed (warnings allowed), exit code 1 means errors found. Doctor lists each error individually.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/format.go:11-47
- Notes: FormatReport (lines 11-37) iterates results in order, writes checkmark/X lines with correct formatting, conditionally includes suggestion line, prints blank line separator before summary only when results exist, uses report.ErrorCount() + report.WarningCount() for issue count (counts both errors and warnings), and handles singular/plural grammar via switch. ExitCode (lines 42-47) delegates to report.HasErrors() returning 0 or 1. Both functions are pure (no side effects beyond io.Writer). Implementation matches all acceptance criteria and spec requirements exactly.

TESTS:
- Status: Adequate
- Coverage: All 17 tests from the plan are present as subtests in /Users/leeovery/Code/tick/internal/doctor/format_test.go. TestFormatReport has 12 subtests covering: zero results, single pass, multiple passes, single fail with suggestion, fail without suggestion, multiple fails from different checks, multiple fails from one check (3 orphaned refs), singular summary, plural summary, all-pass summary, errors+warnings in summary count, order preservation. TestExitCode has 5 subtests covering: empty report (exit 0), all passing (exit 0), warnings only (exit 0), error-severity failure (exit 1), mixed errors+warnings (exit 1).
- Notes: Tests are well-balanced. Each tests a distinct behavior. No redundant assertions. Uses bytes.Buffer for io.Writer testability. Exact string comparison ensures formatting is pixel-perfect. All edge cases from the task (zero issues, single issue, multiple issues from one check, suggestion present/absent, warnings-only) are covered.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, error wrapping not applicable here (pure formatting), io.Writer for DI.
- SOLID principles: Good. Single responsibility (FormatReport formats, ExitCode determines exit code). Open for extension (operates on DiagnosticReport interface). No coupling to concrete check types.
- Complexity: Low. FormatReport is a simple loop with 3 conditionals. ExitCode is a single delegation. Cyclomatic complexity is minimal.
- Modern idioms: Yes. Idiomatic Go: fmt.Fprintf to io.Writer, switch for grammar, range over slice.
- Readability: Good. Self-documenting function names and clear flow. Godoc comments on both exported functions explain behavior and semantics.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. Implementation is clean, tests are comprehensive and well-balanced, code quality is high.
