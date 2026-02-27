TASK: Make tickDir an explicit parameter on the Check interface

ACCEPTANCE CRITERIA:
- Check interface signature is `Run(ctx context.Context, tickDir string) []CheckResult`
- No check implementation contains `ctx.Value(TickDirKey)`
- TickDirKey is removed from the package (or clearly deprecated if Task 1 adds other context keys that reuse the key type)
- All existing tests pass
- DiagnosticRunner.RunAll accepts and forwards tickDir

STATUS: Complete

SPEC CONTEXT: The doctor command runs diagnostic checks against the .tick data directory. This task is a Phase 4 analysis/refactoring task that eliminates the `context.WithValue(TickDirKey)` anti-pattern by making `tickDir` an explicit typed parameter on the Check interface and DiagnosticRunner.RunAll. The goal is structural type safety -- no runtime type assertions for the primary configuration dependency.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/doctor/doctor.go:41` -- Check interface: `Run(ctx context.Context, tickDir string) []CheckResult`
  - `/Users/leeovery/Code/tick/internal/doctor/doctor.go:101` -- DiagnosticRunner.RunAll: `RunAll(ctx context.Context, tickDir string) DiagnosticReport`
  - `/Users/leeovery/Code/tick/internal/doctor/doctor.go:104` -- Forwards tickDir: `check.Run(ctx, tickDir)`
  - `/Users/leeovery/Code/tick/internal/doctor/cache_staleness.go:22` -- `Run(_ context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/jsonl_syntax.go:19` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/id_format.go:22` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/duplicate_id.go:25` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/orphaned_parent.go:17` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/orphaned_dependency.go:17` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/self_referential_dep.go:17` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/dependency_cycle.go:20` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/child_blocked_by_parent.go:21` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/doctor/parent_done_open_children.go:20` -- `Run(ctx context.Context, tickDir string)`
  - `/Users/leeovery/Code/tick/internal/cli/doctor.go:37` -- CLI passes tickDir: `runner.RunAll(ctx, tickDir)`
- Notes:
  - All 10 check implementations accept `tickDir string` as the second parameter
  - No check implementation contains `ctx.Value(TickDirKey)` -- confirmed via grep across entire codebase (only doc files reference it)
  - `TickDirKey` and `tickDirKeyType` are completely removed from Go source (only found in planning/implementation docs)
  - The context parameter is retained for future extensibility (cancellation, timeouts) and for `JSONLinesKey` (the shared JSONL scan optimization from task 4-1)
  - The remaining `ctx.Value` usage in `jsonl_reader.go:79` is for `JSONLinesKey`, not `TickDirKey` -- this is correct and expected
  - CacheStalenessCheck no longer has an empty-tickDir guard; with empty string, `os.ReadFile` fails naturally, producing a consistent error via the normal error path
  - The stubCheck in doctor_test.go also implements the updated interface: `Run(_ context.Context, _ string) []CheckResult`

TESTS:
- Status: Adequate
- Coverage:
  - `doctor_test.go` -- DiagnosticRunner tests pass tickDir via `runner.RunAll(context.Background(), "")` confirming the new signature
  - `cache_staleness_test.go:475-491` -- Explicit test for empty tickDir string producing consistent error (SeverityError, Name "Cache")
  - All 10 check test files call `check.Run(ctxWithTickDir(tickDir), tickDir)` using the updated signature
  - `ctxWithTickDir` helper (cache_staleness_test.go:75-77) has been refactored to simply return `context.Background()`, documenting that tickDir is now passed as an explicit parameter
  - CLI doctor tests (`doctor_test.go`) exercise the full pipeline through `App.Run` including the `runner.RunAll(ctx, tickDir)` path
  - The task's "micro acceptance" requirement "Verify that passing empty string to RunAll produces consistent error behavior across all checks" is covered by the cache staleness empty-tickDir test; other checks would similarly fail at file open with consistent fileNotFoundResult behavior
- Notes:
  - The `ctxWithTickDir` helper is now a no-op that returns `context.Background()`. It could be inlined as `context.Background()` throughout tests, but keeping it as-is avoids unnecessary churn and documents the refactoring history. This is a minor non-blocking observation.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf, DI via struct fields
- SOLID principles: Good -- Interface Segregation improved by this change (concrete dependency made explicit rather than hidden in context). Single Responsibility maintained.
- Complexity: Low -- The refactoring reduced complexity by removing runtime type assertions and inconsistent guard clauses
- Modern idioms: Yes -- Explicit parameters over context values for known typed data follows Go community best practice (context.Value is for request-scoped cross-cutting concerns, not for function parameters)
- Readability: Good -- The interface signature `Run(ctx context.Context, tickDir string) []CheckResult` clearly communicates the required inputs. No magic context keys to discover.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `ctxWithTickDir` helper in `cache_staleness_test.go:75-77` is now a no-op wrapper around `context.Background()`. While harmless, it could be replaced with direct `context.Background()` calls in a future cleanup pass to reduce indirection. The comment documenting why it was retained is helpful.
