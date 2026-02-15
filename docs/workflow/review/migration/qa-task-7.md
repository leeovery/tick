TASK: Presenter Failure Output (migration-2-2)

ACCEPTANCE CRITERIA:
- Failed results render as `  ✗ Task: <title> (skipped: <reason>)` with two-space indent
- Failed results with empty title use `(unknown)` as the title placeholder
- Summary line shows correct count of both imported and failed tasks
- Failures detail section prints after summary when failures exist, with format `- Task "<title>": <reason>`
- Failures detail section is completely absent (no blank line, no header) when there are zero failures
- Special characters in error reasons are printed verbatim
- Successful result rendering is unchanged (regression)
- All output goes to the provided `io.Writer`
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification (docs/workflow/specification/migration.md) defines the output format for migration, including per-task lines with checkmarks/cross marks, a summary line with imported/failed counts, and a "Failures:" detail section shown after the summary when failures exist. Error handling strategy is "continue on error, report failures at end." The spec output example shows both success and failure lines inline, plus a separate failure detail section.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/migrate/presenter.go (lines 21-81)
- Notes:
  - `WriteResult` (line 21-31): Correctly renders checkmark for success, cross mark with `(skipped: reason)` for failure. Uses `FallbackTitle` constant for empty titles.
  - `WriteSummary` (line 35-46): Counts successes and failures independently, produces `Done: N imported, M failed` format.
  - `WriteFailures` (line 51-69): Filters to failures only, prints `Failures:` header and `- Task "title": reason` lines. No-op when zero failures (returns without writing anything, not even a blank line).
  - `Present` (line 74-81): Composes WriteHeader, WriteResult (loop), WriteSummary, WriteFailures in correct sequence.
  - `FallbackTitle` constant defined in `/Users/leeovery/Code/tick/internal/migrate/migrate.go:21` as `"(untitled)"`.
  - All functions write to the provided `io.Writer` parameter.
  - Special characters are printed verbatim via `fmt.Fprintf` with `%s` verb for error reasons.
  - Integration: `RunMigrate` in `/Users/leeovery/Code/tick/internal/cli/migrate.go:127` calls `migrate.Present(stdout, provider.Name(), dryRun, results)` which composes all output sections.

  Plan drift note: The plan task specifies `(unknown)` as the empty-title fallback, but the implementation uses `(untitled)`. This was an intentional consolidation in migration-3-3 which changed the canonical fallback to `(untitled)` as more descriptive. All usage sites now reference the single `FallbackTitle` constant. This is an improvement, not a defect.

TESTS:
- Status: Adequate
- Coverage:
  - TestWriteResult: "prints cross mark and skip reason for failed result" (line 80-90), "prints checkmark for successful result" regression (lines 57-67, 92-102), "prints fallback title when failed result has empty title" (line 104-114)
  - TestWriteSummary: "counts failures correctly" (line 162-177), "2 imported, 1 failed for mixed" (line 179-193), "0 imported, 3 failed when all fail" (line 195-209), "3 imported, 0 failed when no failures" regression (line 211-225)
  - TestWriteFailures: "prints failure detail section" (line 229-245), "prints nothing when zero failures" (line 247-259), "uses fallback title for empty title" (line 261-274), "preserves special characters in failure reason" (line 276-289)
  - TestPresent: "renders full output with failures" (line 382-404), "omits failure detail section when all successful" (line 406-423), "with all failures shows zero imported and failure detail" (line 425-446)
  - All 14 test names from the plan are covered (with minor name variations).
  - Edge cases covered: empty title fallback, special characters (quotes, angle brackets, ampersand), zero failures (detail omitted), all failures, mixed results.
- Notes: Tests are well-structured with exact string comparisons that would fail if behavior changed. No over-testing detected -- each test verifies a distinct aspect. The WriteSummary section has some overlap between "counts only successful results" and "prints Done: 2 imported, 1 failed" but they test different input configurations so this is acceptable.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, writes to io.Writer (DI pattern), error wrapping with fmt.Errorf. No external test libraries.
- SOLID principles: Good. Single responsibility -- each Write* function handles one output section. Present composes them. FallbackTitle constant avoids magic strings.
- Complexity: Low. Each function is linear with simple conditionals. WriteFailures filters then iterates -- straightforward.
- Modern idioms: Yes. Idiomatic Go with fmt.Fprintf, %q for quoting titles in detail section, range loops.
- Readability: Good. Function names are descriptive. Comments document each exported function. The code is self-explanatory.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `%q` verb used in WriteFailures (line 67) for title quoting will escape internal quotes in titles (e.g., a title containing `"` would render as `\"` in the detail section). This is a reasonable defensive behavior but differs from WriteResult which uses `%s` for the title. The spec does not call out titles with embedded quotes so this is not a functional issue.
- WriteSummary tests at lines 130-144 and 211-225 both test the "3 imported, 0 failed" case with identical inputs and expected outputs. Minor redundancy but not worth flagging as over-testing since the test names emphasize different aspects (count accuracy vs. regression).
