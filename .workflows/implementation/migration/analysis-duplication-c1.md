AGENT: duplication
FINDINGS:
- FINDING: RunMigrate inlines presenter orchestration that Present already provides
  SEVERITY: high
  FILES: internal/cli/migrate.go:123-133, internal/migrate/presenter.go:74-81
  DESCRIPTION: RunMigrate manually calls WriteHeader, loops over results calling WriteResult, then calls WriteSummary. The Present function in presenter.go does this exact same sequence plus additionally calls WriteFailures. RunMigrate duplicates the orchestration but omits WriteFailures, meaning failure detail output is silently lost. This is the classic independent-executor duplication pattern — one executor built the presenter with a composed Present function, another executor built RunMigrate and manually wired the individual Write* calls without knowing Present existed.
  RECOMMENDATION: Replace lines 123-133 in RunMigrate with a single call to migrate.Present(stdout, provider.Name(), dryRun, results). This eliminates the duplicated loop and also restores the missing WriteFailures call.

- FINDING: Inconsistent empty-title fallback strings across engine and presenter
  SEVERITY: medium
  FILES: internal/migrate/engine.go:69, internal/migrate/presenter.go:27-29, internal/migrate/presenter.go:64-66
  DESCRIPTION: Three locations apply an empty-title fallback. Engine.Run uses "(untitled)" for validation failures. WriteResult and WriteFailures both use "(unknown)" for display. These serve the same purpose — providing a human-readable placeholder when a task has no title — but use different strings. This is near-duplicate logic that drifted because it was written independently.
  RECOMMENDATION: Consolidate to a single exported constant (e.g., FallbackTitle = "(untitled)") in migrate.go and reference it from engine.go, WriteResult, and WriteFailures. Pick one consistent value.

- FINDING: Duplicate beads fixture helpers in CLI and beads test packages
  SEVERITY: low
  FILES: internal/cli/migrate_test.go:486-495, internal/migrate/beads/beads_test.go:13-24
  DESCRIPTION: setupBeadsFixture (CLI tests) and setupBeadsDir (beads tests) both create a .beads/issues.jsonl file in a temp directory with identical logic — MkdirAll/Mkdir the .beads dir, WriteFile the content. Written independently by separate executors for their respective test suites.
  RECOMMENDATION: This is test-only duplication across separate packages. Since Go test helpers cannot be shared across packages without a testutil package, and the logic is only ~8 lines, this is acceptable to leave as-is. Flag only for awareness — extraction into a shared testutil package is not warranted at this scale.

SUMMARY: The highest-impact finding is RunMigrate duplicating the Present function's orchestration logic while missing WriteFailures. A medium-severity inconsistency exists in empty-title fallback strings across three files. Test helper duplication across packages is minor.
