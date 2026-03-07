---
id: doctor-validation-4-2
phase: 4
status: approved
created: 2026-02-13
---

# Make tickDir an explicit parameter on the Check interface

**Problem**: The tick directory path is passed to checks via `context.WithValue` using `TickDirKey`. Every check starts with `tickDir, _ := ctx.Value(TickDirKey).(string)` -- a runtime type assertion that silently returns empty string on failure. This is the "untyped parameters when concrete types are known" anti-pattern. CacheStalenessCheck guards against empty tickDir but 9 other checks do not, meaning they would construct paths like `filepath.Join("", "tasks.jsonl")` which resolves to a relative path. This inconsistency means checks fail with different error messages (or silently read wrong files) when the context key is missing.

**Solution**: Change the Check interface from `Run(ctx context.Context) []CheckResult` to `Run(ctx context.Context, tickDir string) []CheckResult`. Update DiagnosticRunner.RunAll to accept tickDir and pass it to each check. Remove the TickDirKey context value pattern. Remove the empty-tickDir guard from CacheStalenessCheck since the runner now provides a validated string. Keep the context parameter for future extensibility (cancellation, timeouts).

**Outcome**: The primary configuration dependency is explicit and type-safe. No check needs a runtime type assertion for tickDir. The inconsistent empty-tickDir guarding problem is eliminated structurally. If tickDir is ever empty, it fails at a single point (the runner) rather than inconsistently across 10 checks.

**Do**:
1. Change the `Check` interface in `internal/doctor/doctor.go` from `Run(ctx context.Context) []CheckResult` to `Run(ctx context.Context, tickDir string) []CheckResult`
2. Update `DiagnosticRunner.RunAll` signature to accept `tickDir string` and pass it to each `check.Run(ctx, tickDir)` call
3. Update all 10 check implementations to accept `tickDir string` as the second parameter instead of extracting it from context
4. Remove `tickDirKeyType`, `TickDirKey` from `internal/doctor/doctor.go` (unless still needed for TaskRelationshipsKey from Task 1 -- if so, keep the type but remove TickDirKey specifically)
5. Update `RunDoctor` in `internal/cli/doctor.go` to pass `tickDir` to `runner.RunAll(ctx, tickDir)` instead of embedding it in context
6. Remove the empty-tickDir guard from CacheStalenessCheck (lines 26-34) since the caller is now responsible
7. Remove all `tickDir, _ := ctx.Value(TickDirKey).(string)` lines from all 10 checks
8. Update all test files that create contexts with TickDirKey to instead pass tickDir directly
9. Run all tests

**Acceptance Criteria**:
- Check interface signature is `Run(ctx context.Context, tickDir string) []CheckResult`
- No check implementation contains `ctx.Value(TickDirKey)`
- TickDirKey is removed from the package (or clearly deprecated if Task 1 adds other context keys that reuse the key type)
- All existing tests pass
- DiagnosticRunner.RunAll accepts and forwards tickDir

**Tests**:
- All existing doctor and CLI doctor tests pass
- Verify that passing empty string to RunAll produces consistent error behavior across all checks
