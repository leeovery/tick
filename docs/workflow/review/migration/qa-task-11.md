TASK: Replace manual presenter calls in RunMigrate with Present function

ACCEPTANCE CRITERIA:
- RunMigrate calls migrate.Present instead of individual Write* functions.
- When tasks fail, the "Failures:" section appears in CLI output after the summary line.
- Existing passing tests continue to pass.
- A test case with failing tasks asserts the failure detail block is present in output.

STATUS: Complete

SPEC CONTEXT: The specification (Output Format section) mandates a "Failures:" detail section shown after the summary when any tasks fail, listing each failed task with its reason. The original implementation manually called WriteHeader, looped WriteResult, and called WriteSummary -- but never called WriteFailures, silently dropping the spec-mandated failure detail section.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/migrate.go:127
- Notes: RunMigrate calls `migrate.Present(stdout, provider.Name(), dryRun, results)` as a single orchestration point. No individual Write* calls (WriteHeader, WriteResult, WriteSummary, WriteFailures) remain in migrate.go. The Present function at /Users/leeovery/Code/tick/internal/migrate/presenter.go:74-81 calls all four Write* functions in sequence, including WriteFailures which was previously missing from the CLI path. The function correctly calls Present after engine.Run() returns, and returns runErr afterward so partial results are displayed even on provider error (line 129). All fmt.Fprintf calls remaining in migrate.go are error messages to Stderr in handleMigrate, not output composition.

TESTS:
- Status: Adequate
- Coverage:
  - TestRunMigrateFailureDetail (migrate_test.go:547-617): Two subtests using a stubProvider to test RunMigrate directly.
    - "output includes Failures detail section when tasks fail validation" (line 548): Provider returns 3 tasks (1 valid, 1 empty title, 1 invalid priority). Asserts cross-mark lines, "Failures:\n" section, per-task error messages ("title is required", "priority must be"), and correct summary counts ("1 imported, 2 failed").
    - "output omits Failures section when all tasks succeed" (line 589): Provider returns 2 valid tasks. Asserts "Failures:" does NOT appear, and summary shows "2 imported, 0 failed".
  - Integration-level tests in TestMigrateCommand also cover this:
    - "migrate shows failures for invalid entries from beads provider" (line 315): End-to-end with real beads provider, asserts "Failures:" section, validation errors, and malformed entry sentinel.
    - "migrate output omits Failures section when all tasks succeed" (line 181): End-to-end asserting no "Failures:" section on success.
  - Presenter unit tests (presenter_test.go:292-496): TestPresent has 9 subtests covering Present function directly with failures, all-success, zero results, dry-run combinations, confirming full output composition including failure detail section.
- Notes: Test coverage is thorough without being redundant. The stubProvider-based tests isolate RunMigrate's behavior, the integration tests verify end-to-end with real beads provider, and the presenter unit tests verify Present's output composition. Each level tests a distinct concern.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.Helper on helpers, t.TempDir for isolation, error wrapping with %w, functional DI pattern matching the codebase.
- SOLID principles: Good. Present function is the single point of output composition (SRP). RunMigrate delegates output entirely to Present (DIP). No duplication of presenter orchestration logic.
- Complexity: Low. RunMigrate is a straightforward linear flow: create creator, create engine, run, present, return error.
- Modern idioms: Yes. Clean Go patterns throughout.
- Readability: Good. The comment on line 126 ("Present results regardless of error (partial results on failure)") clearly explains intent.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
