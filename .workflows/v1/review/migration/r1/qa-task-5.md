TASK: CLI Command - tick migrate --from

ACCEPTANCE CRITERIA:
- `tick migrate --from beads` executes the full pipeline: provider -> engine -> output
- `--from` flag is required; omission produces a usage error on stderr with exit code 1
- Unknown provider name produces an error on stderr with exit code 1
- `StoreTaskCreator` correctly creates tick tasks from `MigratedTask` values via tick-core
- Defaults are applied during task creation: empty status -> open, zero priority -> 2, zero timestamps -> sensible defaults
- Each imported task is printed via `Presenter.WriteResult` as it is processed
- Summary line is printed via `Presenter.WriteSummary` after all tasks
- Command exits 0 on success (including when some tasks fail validation)
- Command exits 1 on provider failure or insertion failure
- Provider registry resolves "beads" to a `BeadsProvider` using the current working directory
- End-to-end integration test passes with test fixtures

STATUS: Complete

SPEC CONTEXT: The spec defines `tick migrate --from <provider> [--dry-run] [--pending-only]`. Phase 1 requires only `--from`. Architecture is Provider -> Normalize -> Insert where the inserter is provider-agnostic. Output format shows per-task checkmarks/crosses with a summary line. Unknown provider errors should list available providers (enhanced listing deferred to Phase 2, but implementation already includes it via `UnknownProviderError`). Error strategy is continue-on-error with failure reporting at the end.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/migrate.go` (lines 1-131): CLI command handler, flag parsing, provider registry, RunMigrate orchestration
  - `/Users/leeovery/Code/tick/internal/cli/app.go` (lines 33-35): migrate subcommand registration in App.Run dispatcher
  - `/Users/leeovery/Code/tick/internal/migrate/store_creator.go` (lines 1-97): StoreTaskCreator implementation
  - `/Users/leeovery/Code/tick/internal/migrate/errors.go` (lines 1-36): UnknownProviderError with Available providers listing
  - `/Users/leeovery/Code/tick/internal/migrate/engine.go` (lines 1-89): Engine.Run with validation, continue-on-error
  - `/Users/leeovery/Code/tick/internal/migrate/presenter.go` (lines 1-81): Present function (header, per-task, summary, failures)
- Notes:
  - The `migrate` subcommand is registered early in `App.Run` (line 33-35), bypassing the format/formatter machinery. This matches the pattern used by `doctor` and is appropriate since migration has its own output format.
  - `parseMigrateArgs` (line 47-69) supports both `--from value` and `--from=value` syntax. It also parses `--dry-run` and `--pending-only` flags (Phase 2 features), which is fine since the handler already passes them through.
  - `newMigrateProvider` (line 19-29) uses a switch statement as the registry. Returns `*migrate.UnknownProviderError` with available providers list for unknown names.
  - `handleMigrate` (line 73-103) has a redundant empty-string check on line 80-83 — `parseMigrateArgs` already returns an error when `flags.from == ""` on line 65-67. However, this is a defense-in-depth pattern for the `--from=` (equals with empty value) edge case, which would pass the `from == ""` check in `parseMigrateArgs` but also be caught by `parseMigrateArgs` line 65. Actually examining closely: `--from=` would set `flags.from = ""` via TrimPrefix, and then line 65-67 in parseMigrateArgs catches it. So the check on line 80-83 in handleMigrate is truly redundant but harmless.
  - `RunMigrate` (line 108-130) correctly wires: creator selection (dry-run vs store), engine creation, provider execution, and presentation via `migrate.Present()`. Results are presented even on partial failure (line 127 runs regardless of runErr).
  - `StoreTaskCreator.CreateTask` uses `store.Mutate` to ensure atomic writes with collision-safe ID generation. Defaults are correctly applied: empty status -> `task.StatusOpen`, nil priority -> 2, zero Created -> `time.Now().UTC()`, zero Updated -> Created, zero Closed -> nil.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/migrate_test.go`: CLI-level tests covering:
    - Missing --from flag (exit 1, stderr contains --from)
    - Empty --from value (exit 1, stderr contains --from)
    - Unknown provider (exit 1, stderr contains UnknownProviderError format with available providers)
    - --from beads resolves provider and runs pipeline (exit 0, header + task in output)
    - Exit 0 when some tasks fail validation
    - Provider read failure (no .beads dir -> exit 1)
    - End-to-end integration with fixture data (verifies output AND persisted tasks including status/priority/description/closed mapping)
    - Zero tasks (header + summary, exit 0)
    - Tick not initialized (exit 1)
    - Failures section shown for invalid entries, omitted when all succeed
  - `/Users/leeovery/Code/tick/internal/cli/migrate_test.go` also covers dry-run and pending-only (Phase 2 features tested in same file)
  - `/Users/leeovery/Code/tick/internal/migrate/store_creator_test.go`: Unit tests for StoreTaskCreator:
    - Full field mapping
    - Default status open
    - Default priority 2 when nil
    - Default Created as time.Now when zero
    - Default Updated as Created when zero
    - Tick ID generation (tick- prefix, uniqueness)
    - Store write failure propagation
    - Nil Closed when zero
    - Preserves explicit priority 0 (distinguishes nil from zero)
  - `/Users/leeovery/Code/tick/internal/cli/migrate_test.go` line 16-86: Registry tests (beads resolution, unknown provider error, available providers sorted)
  - `/Users/leeovery/Code/tick/internal/migrate/errors_test.go`: UnknownProviderError format tests
- Notes:
  - All 16 plan-specified tests are covered or have equivalent coverage.
  - The end-to-end test (line 212-280) is thorough: verifies output format, persisted tasks, field mapping (status, priority, description, closed timestamp).
  - Test at line 50-61 ("NewProvider still returns BeadsProvider for name beads (regression)") is redundant with line 17-31 — it tests the exact same thing with slightly fewer assertions. This is minor over-testing but not blocking.
  - The dry-run and pending-only tests in this file go beyond Phase 1 scope but are testing Phase 2 features that were already implemented. Not a concern for this task.

CODE QUALITY:
- Project conventions: Followed
  - Handler follows the existing command pattern (handleX method on App, RunX function)
  - Uses `fmt.Fprintf(a.Stderr, "Error: %s\n", err)` pattern consistent with other commands
  - Tests use stdlib testing, t.Run subtests, t.TempDir, t.Helper — all per project conventions
  - Error wrapping with %w used in store_creator (via Mutate)
- SOLID principles: Good
  - Single responsibility: parseMigrateArgs, newMigrateProvider, handleMigrate, RunMigrate each have distinct roles
  - Dependency inversion: StoreTaskCreator depends on Mutator interface, not concrete Store
  - Open/closed: Provider registry uses switch (acceptable for Phase 1; plan acknowledges minimal approach)
  - Interface segregation: Mutator interface exposes only what StoreTaskCreator needs (single Mutate method)
- Complexity: Low
  - parseMigrateArgs is a simple linear scan
  - handleMigrate is a straightforward sequential pipeline
  - StoreTaskCreator.CreateTask has clear, linear default-application logic
- Modern idioms: Yes
  - Uses *int for optional Priority (nil vs zero distinction)
  - Functional mutation pattern via store.Mutate closure
- Readability: Good
  - Code is well-commented (exported functions have doc comments)
  - Variable names are clear (mf for migrateFlags, mt for MigratedTask)
  - Flow in handleMigrate/RunMigrate is easy to follow

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The redundant empty-from check at `/Users/leeovery/Code/tick/internal/cli/migrate.go:80-83` is unreachable because `parseMigrateArgs` already catches `from == ""` on line 65-67. Could be removed for clarity, but harmless as defense-in-depth.
- Test "NewProvider still returns BeadsProvider for name beads (regression)" at `/Users/leeovery/Code/tick/internal/cli/migrate_test.go:50-61` is redundant with the first test at line 17-31. Consider removing to reduce noise.
- The `providerNames` slice at `/Users/leeovery/Code/tick/internal/cli/migrate.go:15` must be kept manually in sync with the switch in `newMigrateProvider`. A comment documents this, but it is a maintenance risk as more providers are added. The plan notes this is Phase 1 minimal approach, and Phase 2 task migration-2-5 will enhance the registry.
