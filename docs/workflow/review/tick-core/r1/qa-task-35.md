TASK: Remove dead VerboseLog function (tick-core-7-4)

ACCEPTANCE CRITERIA:
- VerboseLog function no longer exists in format.go
- No test references to VerboseLog remain
- All existing tests pass
- No production code is affected

STATUS: Complete

SPEC CONTEXT: The spec defines --verbose flag behavior (debug detail to stderr). The implementation uses a VerboseLogger struct (verbose.go) with nil-receiver no-op pattern. The standalone VerboseLog function was a pre-VerboseLogger remnant identified during cycle 2 analysis as dead code.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/format.go (184 lines, no VerboseLog present)
- Notes: The VerboseLog standalone function does not exist anywhere in .go files. A comprehensive grep for `VerboseLog[^g]` (matching VerboseLog but not VerboseLogger) returns zero results in Go source files. The function and its io import have been cleanly removed. The VerboseLogger struct remains in /Users/leeovery/Code/tick/internal/cli/verbose.go (lines 13-28) as the sole verbose logging mechanism, which is the correct outcome.

TESTS:
- Status: Adequate
- Coverage: The associated test function (TestVerboseToStderrOnly, which tested the dead VerboseLog function) has been removed. No test references to VerboseLog remain. The VerboseLogger struct retains comprehensive test coverage in /Users/leeovery/Code/tick/internal/cli/verbose_test.go (218 lines, 7 subtests covering: message writing, nil-receiver no-op, stderr-only output, quiet+verbose interaction, format flag compatibility, piped output, and prefix verification).
- Notes: This task explicitly called for no new tests -- only removal of dead test code and verification via grep. The remaining verbose_test.go coverage is thorough and tests the replacement (VerboseLogger) adequately.

CODE QUALITY:
- Project conventions: N/A (deletion task, no new code)
- SOLID principles: Good -- removal of dead code improves single responsibility by consolidating verbose logging into one mechanism (VerboseLogger)
- Complexity: Low -- pure deletion
- Modern idioms: Yes -- the remaining VerboseLogger uses idiomatic Go nil-receiver pattern
- Readability: Good -- format.go is cleaner without the dead function
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The plan index (docs/workflow/planning/tick-core.md line 218) still shows this task as "pending" status, but the task file itself (tick-core-7-4.md) correctly shows "completed". The plan index should be updated for consistency.
