TASK: tick doctor Command Wiring

ACCEPTANCE CRITERIA:
- [ ] `tick doctor` subcommand registered in CLI dispatch
- [ ] Doctor discovers `.tick/` directory via existing helper
- [ ] Missing `.tick/` directory produces error to stderr with exit code 1
- [ ] DiagnosticRunner created with CacheStalenessCheck registered
- [ ] Formatted output written to stdout (check/cross markers, details, suggestions, summary)
- [ ] Exit code 0 when all checks pass, exit code 1 when any error found
- [ ] Doctor outputs human-readable text only (no TOON/JSON variants)
- [ ] Doctor does not modify `tasks.jsonl` -- verified by before/after comparison
- [ ] Doctor does not modify `cache.db` -- verified by before/after comparison
- [ ] Doctor does not create new files in `.tick/`
- [ ] Doctor does not trigger cache rebuild when cache is stale
- [ ] Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines `tick doctor` as a diagnostic-only command that reports problems and suggests remedies but never modifies data. Design principle #1: "Report, don't fix." Design principle #2: "Human-focused -- no structured output variants." Exit codes: 0 = all checks passed (warnings allowed), 1 = one or more errors found. Doctor outputs human-readable text only -- no TOON/JSON variants. The `.tick/` directory discovery pattern is established in tick-core: walk up from cwd looking for `.tick/`, error if not found.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/doctor.go:17-42` -- `RunDoctor` function creates runner, registers all 10 checks, runs them, formats output, returns exit code
  - `/Users/leeovery/Code/tick/internal/cli/doctor.go:44-61` -- `handleDoctor` method discovers `.tick/` dir, handles missing dir error, delegates to RunDoctor
  - `/Users/leeovery/Code/tick/internal/cli/app.go:30-31` -- `doctor` subcommand registered in CLI dispatch, placed before format/formatter resolution
- Notes:
  - All 12 acceptance criteria are met.
  - Doctor is correctly placed before format flag parsing (line 30-31 of app.go), ensuring it bypasses `--toon`/`--pretty`/`--json` machinery. This matches the spec requirement of human-readable text only.
  - `DiscoverTickDir` is reused from the existing helper in `discover.go`, consistent with other commands.
  - Missing `.tick/` produces "Error: Not a tick project (no .tick directory found)" to stderr and returns exit code 1, matching the spec error message.
  - DiagnosticRunner is created and all 10 checks are registered (CacheStalenessCheck, JsonlSyntaxCheck, IdFormatCheck, DuplicateIdCheck, OrphanedParentCheck, OrphanedDependencyCheck, SelfReferentialDepCheck, DependencyCycleCheck, ChildBlockedByParentCheck, ParentDoneWithOpenChildrenCheck).
  - `ScanJSONLines` is called once and injected via context to avoid re-reading for each check. This is a good optimization.
  - FormatReport writes to stdout; ExitCode computes exit based on error severity only (warnings don't affect exit code), matching spec.
  - Doctor does not acquire a write lock, does not call any Store methods, and does not trigger cache rebuild.

TESTS:
- Status: Adequate
- Coverage:
  - `TestDoctor` (12 subtests in `/Users/leeovery/Code/tick/internal/cli/doctor_test.go:86-267`):
    - Healthy store exits 0 with "OK" output
    - Stale cache exits 1 with "stale" in output
    - Formatted output contains check marks and "No issues found." summary
    - Missing `.tick/` directory exits 1
    - Missing `.tick/` prints "Not a tick project" to stderr, stdout empty
    - `tasks.jsonl` byte-level comparison before/after (read-only invariant)
    - `cache.db` byte-level comparison before/after (read-only invariant)
    - No new files created in `.tick/` directory
    - Stale cache does not trigger rebuild (cache.db unchanged)
    - Cache staleness check is registered and runs (output contains "Cache")
    - "No issues found." summary for healthy store
    - "1 issue found." summary for stale cache
  - `TestDoctorFourChecks` (12 subtests, lines 307-538): Integration tests for the 4-check registration phase. Tests mixed pass/fail, individual check failures, combined summary count, no short-circuit, empty file, read-only invariant with 4 checks.
  - `TestDoctorTenChecks` (14 subtests, lines 550-836): Integration tests for full 10-check registration. Tests all labels visible, all 10 check marks, individual relationship check failures, warning-only exits 0, mixed error+warning, cross-phase error combinations, summary counts, no short-circuit, empty file, read-only invariant with 10 checks.
  - All 12 test cases from the task's Tests section are present and mapped.
  - Edge cases covered: missing `.tick/` directory (precondition failure with no diagnostic output), read-only verification (byte comparison of both files + no new files), stale cache not triggering rebuild, empty project.
- Notes:
  - The `TestDoctorFourChecks` and `TestDoctorTenChecks` test functions were added by later phases (2-4 and 3-7 respectively) and extend the wiring tests. While they test more than just this task's scope, they confirm the wiring remains correct as more checks are added -- this is appropriate integration testing.
  - There is some overlap between `TestDoctor/"it shows No issues found summary when all checks pass"` and `TestDoctor/"it prints formatted output to stdout with check markers and summary line"`, but they test slightly different aspects (one focuses on markers, the other on summary text) so this is acceptable.
  - Test helpers (`setupDoctorProject`, `setupDoctorProjectStale`, `runDoctor`) use `t.Helper()` correctly.
  - Tests go through the full `App.Run()` dispatch path via `runDoctor`, testing the actual wiring rather than just `RunDoctor` in isolation. This is the right level for a wiring task.

CODE QUALITY:
- Project conventions: Followed
  - Uses `t.TempDir()` for isolation, stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on helpers
  - Error wrapping follows `fmt.Errorf("context: %w", err)` pattern
  - Handler signature differs from other commands (returns `int` instead of `error`) which is intentional since doctor controls its own exit code. This is consistent with `handleMigrate` which also returns `int`.
  - DI pattern matches the App struct field injection pattern
- SOLID principles: Good
  - Single responsibility: `RunDoctor` handles orchestration (create runner, register checks, run, format, exit code); `handleDoctor` handles CLI concerns (discover dir, error to stderr). Clean separation.
  - Open/closed: New checks can be added by simply calling `runner.Register()` -- no existing code modified.
  - Dependency inversion: `RunDoctor` accepts `io.Writer` for output, checks implement the `Check` interface.
- Complexity: Low
  - `RunDoctor` is a linear sequence of steps with no branching except the ScanJSONLines error path (which gracefully degrades -- checks will scan individually).
  - `handleDoctor` has two error checks and a delegation -- minimal complexity.
- Modern idioms: Yes
  - `context.WithValue` for passing pre-scanned data is idiomatic Go for request-scoped values.
  - Uses `context.Background()` as the root context, appropriate for CLI tooling.
- Readability: Good
  - Clear godoc comments on both exported `RunDoctor` and unexported `handleDoctor`.
  - The comment on `RunDoctor` lists all 10 checks, which serves as documentation but may become stale if checks change. Minor concern, not blocking.
  - The separation between `handleDoctor` (CLI layer) and `RunDoctor` (logic layer) makes the code easy to follow.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The godoc on `RunDoctor` (line 11-16) hardcodes the list of all 10 checks. If checks are added/removed in future, this comment could drift. Consider a more general description like "registers all diagnostic checks" without enumerating them. Very minor.
- The `TestDoctorFourChecks` and `TestDoctorTenChecks` were added by later tasks (2-4, 3-7) in the same file. This is fine for now but as the test file grows to 837 lines, consider whether future tests should be in a separate integration test file. Not a current issue, just noting the trend.
