AGENT: standards
FINDINGS:
- FINDING: Failure detail section not rendered in CLI output path
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/cli/migrate.go:130-133
  DESCRIPTION: The spec defines a "Failure detail" section that should appear after the summary when any failures occur. The format is specified as:
    ```
    Failures:
    - Task "foo": Missing required field
    - Task "bar": Invalid date format
    ```
    The `Present()` function in presenter.go correctly calls `WriteFailures()` after `WriteSummary()`. However, `RunMigrate` in migrate.go does NOT use `Present()`. It calls `WriteHeader`, `WriteResult` (in a loop), and `WriteSummary` individually -- but never calls `WriteFailures`. This means when tasks fail during a real migration, the user sees the per-task cross marks and the summary count, but never sees the detailed failure listing that the spec requires.
  RECOMMENDATION: In `RunMigrate`, add `migrate.WriteFailures(stdout, results)` after the `migrate.WriteSummary(stdout, results)` call. Alternatively, replace the individual calls with a single call to `migrate.Present(stdout, provider.Name(), dryRun, results)` which already composes all four output sections correctly.

- FINDING: Beads provider silently drops invalid entries instead of reporting them as failures
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/migrate/beads/beads.go:90-105
  DESCRIPTION: The spec's error handling strategy says: "When a task fails to import: 1. Log the failure with reason 2. Continue processing remaining tasks 3. Report summary at end." The beads provider's `Tasks()` method silently skips malformed JSON lines, entries with empty titles, and entries that fail validation (lines 90-105). These skipped entries never reach the engine and thus never appear in the output. A user with malformed beads data would see fewer tasks than expected with no explanation. The spec expects failures to be visible in the output (cross mark line + failure detail section), but the provider swallows them before the engine can report them.
  RECOMMENDATION: The provider should return all parseable tasks (including those with empty titles or invalid data) and let the engine's validation report them as failures. Alternatively, the provider could return a separate list of parse errors alongside the valid tasks. The key requirement is that the user sees these failures in the output. At minimum, malformed JSON skip is acceptable (the provider can't construct a MigratedTask at all), but entries with empty titles or invalid field values should be passed through to the engine for proper reporting.

- FINDING: Unknown beads status values silently map to empty string
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/migrate/beads/beads.go:128
  DESCRIPTION: When a beads issue has a status value not in the statusMap (e.g., "wontfix", "blocked", "review"), `mapToMigratedTask` silently maps it to an empty string. This empty string then passes validation (empty status is valid per MigratedTask.Validate), and the StoreTaskCreator defaults it to "open". The user gets no indication that a status value was unrecognizable and defaulted. The spec says "Missing data uses sensible defaults or is left empty" which could justify this, but a beads issue with status "wontfix" is not missing data -- it has data that doesn't map. The user should at least be able to see this happened.
  RECOMMENDATION: Consider logging or reporting when a known-but-unmappable status is encountered. At minimum, this is acceptable per spec's "sensible defaults" guidance, but awareness of data loss during migration is important. A comment documenting this design decision would help.

SUMMARY: Two high-severity findings: the failure detail section specified in the output format is never rendered in the actual CLI path (RunMigrate skips WriteFailures), and the beads provider silently drops invalid entries rather than surfacing them as failures per the spec's error handling strategy. One medium finding about silent status mapping.
