TASK: Pending-Only Filter (migration-2-4)

ACCEPTANCE CRITERIA:
- filterPending function exists in internal/migrate/ and removes tasks with status "done" or "cancelled"
- filterPending retains tasks with status "open", "in_progress", or "" (empty)
- Engine accepts an Options struct with a PendingOnly field
- Engine.Run applies filterPending after provider.Tasks() and before the processing loop when PendingOnly is true
- Engine.Run does NOT filter when PendingOnly is false (default behavior unchanged)
- --pending-only flag is registered on the migrate command with default false
- CLI passes the --pending-only value to the engine via Options
- All tasks completed + --pending-only results in zero imported, zero failed (not an error)
- No completed tasks + --pending-only results in all tasks processed (filter is no-op)
- Mixed statuses + --pending-only results in only non-completed tasks processed
- Empty status is NOT filtered out (it represents a pending task)
- --pending-only composes with --dry-run without special-case logic
- Existing engine tests still pass with zero-value Options
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification defines `--pending-only` as an optional boolean flag on `tick migrate` that imports only non-completed tasks. Completed statuses in tick's model are "done" and "cancelled". The beads provider maps "closed" to "done", so --pending-only filters those out. The flag composes orthogonally with --dry-run (filtering stage vs persistence stage).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:10-13 (completedStatuses map)
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:16-18 (Options struct)
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:23-31 (filterPending function)
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:50-52 (NewEngine accepts Options)
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:65-67 (filter applied in Engine.Run)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:43 (pendingOnly in migrateFlags)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:63 (--pending-only parsing)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:97 (pendingOnly passed to RunMigrate)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:108 (RunMigrate accepts pendingOnly param)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:121 (Options{PendingOnly: pendingOnly} passed to NewEngine)
- Notes:
  - filterPending uses a map lookup (`completedStatuses`) for O(1) status checking, which is clean and extensible.
  - Uses `task.Status` type constants (StatusDone, StatusCancelled) rather than raw strings, consistent with migration-3-4 refactoring.
  - Filter is positioned exactly as specified: after provider.Tasks(), before the validation/insertion loop.
  - Default PendingOnly is false (Go zero value), so no filtering unless explicitly enabled.
  - Composition with --dry-run is fully orthogonal -- no special-case logic. Both flags are independent parameters to RunMigrate (line 97, 108).
  - All acceptance criteria are met.

TESTS:
- Status: Adequate
- Coverage:
  - TestFilterPending (engine_test.go:36-166): 9 subtests covering removes done, removes cancelled, retains open, retains in_progress, retains empty status, all completed (empty result), none completed (no-op), mixed statuses, preserves order. Matches all 9 plan-specified filterPending tests exactly.
  - TestEnginePendingOnly (engine_test.go:743-897): 5 subtests covering engine with PendingOnly true filters completed tasks, PendingOnly false does not filter, all completed returns empty results and nil error, no completed tasks returns all results, PendingOnly true still validates remaining tasks. Matches all 5 plan-specified engine integration tests.
  - TestMigratePendingOnly (migrate_test.go:460-535): 3 CLI integration subtests covering --pending-only defaults to false, --pending-only is accepted by migrate command, --pending-only combined with --dry-run filters then previews without writing. Matches all 3 plan-specified CLI tests.
  - Existing TestEngineRun tests (engine_test.go:168-741): All use Options{} (zero-value), confirming backward compatibility.
  - Total: 17 test cases covering this task's functionality.
- Notes:
  - Tests are well-structured with clear names matching the plan's test specifications.
  - The mixed statuses test (engine_test.go:127-145) verifies both count and title ordering, which also covers the "preserves task order" edge case.
  - The "still validates remaining tasks" test (engine_test.go:858-896) covers the important scenario where a pending task with empty title fails validation after filtering.
  - CLI integration tests use real beads fixtures and verify end-to-end behavior including persistence verification.
  - No over-testing observed -- each test covers a distinct scenario or edge case.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf, functional option-style constructor.
- SOLID principles: Good. filterPending has single responsibility (filtering). Engine remains open for extension (new Options fields) without modifying existing behavior (PendingOnly: false is default). Filter is provider-agnostic (placed in engine, not provider).
- Complexity: Low. filterPending is a simple loop with map lookup. Engine.Run has clear linear flow: fetch -> filter -> iterate.
- Modern idioms: Yes. Uses map[task.Status]bool for set membership, pre-allocated slice with make([]T, 0, len(tasks)), typed constants from task package.
- Readability: Good. completedStatuses map at package level is self-documenting. filterPending function and its doc comment clearly describe behavior. Engine.Run filter application is 3 lines and obvious.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The completedStatuses map and validStatuses map (in migrate.go) both define status categories. If more status-related filtering is added in the future, these could be consolidated, but the current separation is appropriate since they serve different purposes (filtering vs validation).
