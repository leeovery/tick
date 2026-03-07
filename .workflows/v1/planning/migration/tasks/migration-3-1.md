---
id: migration-3-1
phase: 3
status: completed
created: 2026-02-15
---

# Replace manual presenter calls in RunMigrate with Present function

**Problem**: `RunMigrate` in `internal/cli/migrate.go:123-133` manually calls `WriteHeader`, loops over results calling `WriteResult`, then calls `WriteSummary` -- but never calls `WriteFailures`. The `Present` function in `internal/migrate/presenter.go:74-81` does this exact sequence plus `WriteFailures`. The CLI therefore silently drops the spec-mandated "Failures:" detail section from output when tasks fail. All three analysis agents independently identified this.

**Solution**: Replace the manual presenter orchestration in `RunMigrate` with a single call to `migrate.Present()`. This requires restructuring `RunMigrate` slightly since the header is currently printed before `engine.Run()` while `Present` prints it after. Move the `Present` call to after `engine.Run()` completes, passing all results.

**Outcome**: The CLI renders the complete output including the failure detail section. No duplicated presenter orchestration. The `Present` function becomes the single point of output composition.

**Do**:
1. In `internal/cli/migrate.go`, remove lines 123-133 (the manual `WriteHeader`, result loop, and `WriteSummary` calls).
2. After `engine.Run()` returns results, call `migrate.Present(stdout, provider.Name(), dryRun, results)`.
3. The function should still return `runErr` after calling `Present` so partial results are displayed even on provider error.
4. Update CLI integration tests that assert on migration output to verify the failure detail section appears when failures exist.

**Acceptance Criteria**:
- `RunMigrate` calls `migrate.Present` instead of individual `Write*` functions.
- When tasks fail, the "Failures:" section appears in CLI output after the summary line.
- Existing passing tests continue to pass.
- A test case with failing tasks asserts the failure detail block is present in output.

**Tests**:
- Test `RunMigrate` with a provider that returns tasks causing validation failures; assert output contains both the cross-mark lines and the "Failures:" detail section with per-task error messages.
- Test `RunMigrate` with all-successful tasks; assert no "Failures:" section appears.
