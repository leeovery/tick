TASK: Dry-Run Mode (migration-2-3)

ACCEPTANCE CRITERIA:
- DryRunTaskCreator exists in internal/migrate/ and implements TaskCreator
- DryRunTaskCreator.CreateTask always returns ("", nil)
- --dry-run flag is registered on the migrate command with default false
- When --dry-run is set, CLI wires DryRunTaskCreator instead of StoreTaskCreator
- When --dry-run is not set, CLI still wires StoreTaskCreator (unchanged behavior)
- Header output includes [dry-run] indicator when dry-run is active
- Header output does NOT include [dry-run] when dry-run is inactive
- Engine is NOT modified -- dry-run is entirely a CLI wiring concern
- Validation still runs during dry-run; invalid tasks produce failure Results
- Zero tasks in dry-run mode produces header with [dry-run] and summary with zero counts
- --dry-run and --pending-only compose correctly without special-case logic
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification defines `tick migrate --from <provider> [--dry-run] [--pending-only]` where `--dry-run` is an optional flag that previews what would be imported without writing. The spec shows the same output format for dry-run and real import. The plan task specifies a no-op TaskCreator approach that keeps the engine unaware of dry-run semantics, using dependency injection to swap creators at the CLI wiring level.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/migrate/dry_run_creator.go (lines 1-14): DryRunTaskCreator struct with compile-time interface check and no-op CreateTask
  - /Users/leeovery/Code/tick/internal/cli/migrate.go (lines 40-44): migrateFlags struct includes dryRun bool field
  - /Users/leeovery/Code/tick/internal/cli/migrate.go (lines 59): --dry-run flag parsed in parseMigrateArgs
  - /Users/leeovery/Code/tick/internal/cli/migrate.go (lines 108-119): RunMigrate selects DryRunTaskCreator vs StoreTaskCreator based on dryRun parameter
  - /Users/leeovery/Code/tick/internal/migrate/presenter.go (lines 10-16): WriteHeader appends [dry-run] indicator when dryRun is true
  - /Users/leeovery/Code/tick/internal/migrate/presenter.go (line 74): Present function passes dryRun through to WriteHeader
  - /Users/leeovery/Code/tick/internal/migrate/engine.go: Confirmed NO dry-run awareness -- engine is unmodified
- Notes: Implementation matches the plan exactly. The no-op TaskCreator approach is clean, follows DI principles, and keeps the engine agnostic. The plan suggested value receiver `DryRunTaskCreator` but implementation uses pointer receiver `*DryRunTaskCreator` -- this is a negligible difference and arguably more idiomatic Go for interface implementations.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/migrate/dry_run_creator_test.go:
    - Compile-time interface check (line 9)
    - "CreateTask returns empty string and nil error" (line 12)
    - "CreateTask never returns an error regardless of input" (line 23) -- tests with varied MigratedTask payloads
    - "engine with DryRunTaskCreator produces successful Result for each valid task" (line 47)
    - "engine with DryRunTaskCreator still fails validation for tasks with empty title" (line 85)
  - /Users/leeovery/Code/tick/internal/migrate/presenter_test.go:
    - "dry-run header prints Importing from <provider>... [dry-run]" (line 34)
    - "non-dry-run header does not include [dry-run]" (line 45)
    - "dry-run with zero tasks prints header with [dry-run] and summary with zero counts" (line 448)
    - "dry-run with multiple tasks shows all as successful with checkmark" (line 461)
    - "dry-run summary shows correct imported count matching number of valid tasks" (line 482)
  - /Users/leeovery/Code/tick/internal/cli/migrate_test.go:
    - "--dry-run flag defaults to false" (line 352) -- verifies no [dry-run] in output AND verifies tasks ARE persisted
    - "non-dry-run execution still uses StoreTaskCreator" (line 372) -- regression test verifying persistence
    - "dry-run with zero tasks prints header with [dry-run] and summary with zero counts" (line 390)
    - "dry-run with multiple tasks shows all as successful" (line 407) -- also verifies 0 persisted tasks
    - "dry-run summary shows correct imported count matching number of valid tasks" (line 442)
    - "--pending-only combined with --dry-run filters then previews without writing" (line 503) -- tests flag composition
- Notes: All 12 planned tests from the task spec are covered. Tests verify behavior at multiple levels (unit, integration, CLI end-to-end). The CLI tests are particularly strong because they verify both output AND persistence side effects (checking that dry-run produces zero persisted tasks). Edge cases from the plan (zero tasks, --pending-only combination, validation failures during dry-run) are all covered.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.Helper on helpers, t.TempDir for isolation, fmt.Errorf error wrapping, DI via struct fields.
- SOLID principles: Good. DryRunTaskCreator follows Liskov substitution (clean swap for StoreTaskCreator). Open/closed principle respected -- engine was not modified. Single responsibility maintained -- DryRunTaskCreator does one thing (nothing). Interface segregation via the focused TaskCreator interface.
- Complexity: Low. DryRunTaskCreator is 3 lines of logic. The CLI wiring is a simple if/else branch.
- Modern idioms: Yes. Compile-time interface check `var _ TaskCreator = (*DryRunTaskCreator)(nil)` is idiomatic Go. Pointer receiver on interface implementation is conventional.
- Readability: Good. Code is self-documenting. Comments on DryRunTaskCreator clearly explain its purpose and why ID is empty. The RunMigrate function clearly shows the creator selection logic.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `parseMigrateArgs` function has no direct unit tests; flag parsing for --dry-run is only tested via CLI integration tests. A table-driven unit test for `parseMigrateArgs` covering all flag combinations would improve test granularity, but this is not blocking since the integration tests provide adequate behavioral coverage.
