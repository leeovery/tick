# QA: Task 7 -- Consolidate overlapping flag validation test coverage

## STATUS: Incomplete
## FINDINGS_COUNT: 1 blocking issue

## Implementation

Task unknown-flags-silently-ignored-3-1 ("Consolidate overlapping flag validation test coverage") requires reducing redundant test coverage across the flag validation test files. The task was a Phase 3 analysis response to overlaps identified during earlier review cycles -- specifically qa-task-4 findings #1 and #2.

**No consolidation has been performed.** All previously identified overlaps remain, and additional overlaps exist. The four test files in scope (`flag_validation_test.go`, `flags_test.go`, `unknown_flag_test.go`, `cli_test.go`) are unchanged from when the overlaps were first flagged.

## Test Adequacy

### Under-testing concerns

None. There is more than sufficient test coverage for flag validation. The problem is the opposite.

### Over-testing concerns

There is significant overlap across three test files. The following redundancies were identified:

**1. Exact duplicate: remove -f short flag**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:76-81` -- `TestValidateFlags/"it accepts -f on remove"` (unit)
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:310-324` -- `TestRemoveAcceptsShortFlag` (unit)
- These test the exact same thing with the exact same args at the same level (both call `ValidateFlags` directly). Pure duplication.

**2. Near-duplicate: global flags interspersed with create command args**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:110-115` -- `TestValidateFlags/"it handles global flags interspersed with command args"` tests `--verbose`, `--json` among `create` args
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:206-209` -- `TestGlobalFlagsMixedWithCommandFlags/"global flags mixed with command flags on create"` tests `--json`, `--verbose` among `create` args
- Both unit-level, nearly identical arg combinations, testing the same behavior.

**3. Redundant: unknown flag on list**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:24-33` -- `TestValidateFlags/"it returns error for unknown flag"` tests `--unknown` on `list`
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:123-133` -- `TestFlagValidationAllCommands` rejects `--unknown-flag` on `list` (among others)
- Both unit-level calls to `ValidateFlags` testing same behavior on `list`.

**4. Redundant: value-taking flag skipping**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:46-63` -- `TestValidateFlags` has two subtests for value-taking skip and boolean non-skip
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:231-308` -- `TestValueTakingFlagSkipping` has 8 subtests comprehensively covering the same concept plus more
- The two tests in `flags_test.go` are strict subsets of what `flag_validation_test.go` covers more thoroughly.

**5. Redundant: two-level command help hint**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:83-108` -- `TestValidateFlags/"it uses parent command in help hint for two-level commands"` (unit, all 4 two-level cmds)
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:143-160` -- `TestFlagValidationAllCommands` no-flag commands section covers dep add, dep remove, note add, note remove rejection
- `/Users/leeovery/Code/tick/internal/cli/unknown_flag_test.go:97-132` -- Two-level section of `TestUnknownFlagRejection` (integration, all 4 two-level cmds)
- Three separate tests verifying two-level command error formatting. The unit-level test in `flags_test.go` and the one in `flag_validation_test.go` both call `ValidateFlags` directly. Consolidation to one unit test + one integration test would suffice.

**6. Redundant: global flags across all commands**
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:189-203` -- `TestGlobalFlagsAcceptedOnAnyCommand` exhaustive 9 flags x 20 commands cross-product (unit, 180 subtests)
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:205-229` -- `TestGlobalFlagsMixedWithCommandFlags` (unit, 3 specific commands)
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:110-115` -- `TestValidateFlags/"it handles global flags interspersed..."` (unit, 1 command)
- `/Users/leeovery/Code/tick/internal/cli/unknown_flag_test.go:270-327` -- `TestGlobalFlagsNotRejected` (integration, 3 flags x 4 commands)
- The exhaustive cross-product in `flag_validation_test.go:189-203` makes the mixed-flag tests in both `flags_test.go` and `flag_validation_test.go:205-229` redundant at the unit level. The integration test in `unknown_flag_test.go` tests a different layer but could be smaller since the unit test is exhaustive.

**7. Redundant: short unknown flags**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:65-74` -- `TestValidateFlags/"it rejects short unknown flags"` on `list` (unit)
- `/Users/leeovery/Code/tick/internal/cli/unknown_flag_test.go:137-168` -- `TestUnknownShortFlagRejection` on show/list/dep add (integration)
- The unit test on `list` is already subsumed by the integration test (which also covers `list`).

**8. Overlapping completeness checks**
- `/Users/leeovery/Code/tick/internal/cli/flags_test.go:118-134` -- `TestCommandFlagsCoversAllCommands` verifies all expected commands exist in `commandFlags`
- `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:9-161` -- `TestFlagValidationAllCommands` implicitly verifies the same by iterating all commands with flag counts
- Both serve the same purpose of ensuring `commandFlags` is complete.

## Code Quality

**Project conventions**: Tests follow the project's `t.Run()` / `testing` stdlib conventions. No issues there.

**DRY**: This is the primary quality issue. The same assertions are written multiple times across three files with slightly different names. Flag validation testing should be organized with a clear ownership model:
- `flags_test.go` should own unit tests for `ValidateFlags()` core behavior
- `flag_validation_test.go` should own per-command flag metadata correctness and drift detection
- `unknown_flag_test.go` should own integration tests through `App.Run()`

Currently the boundaries are blurred with all three files containing unit-level `ValidateFlags()` calls testing overlapping scenarios.

**Readability**: The test organization is confusing. A developer looking at the test suite would need to read all three files to understand which tests cover what, with significant redundancy making it hard to identify the authoritative test for any given behavior.

**Complexity**: Low per-test, but the total test surface is unnecessarily large for what it covers.

## Findings

### Blocking

1. **Task not implemented** -- The task "Consolidate overlapping flag validation test coverage" has not been performed. All overlaps identified in the previous review cycle (qa-task-4 non-blocking findings #1 and #2) remain present. Additionally, at least 6 more overlap patterns exist across the three test files. No test was removed, moved, or consolidated. The task is incomplete.

   Files affected:
   - `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go`
   - `/Users/leeovery/Code/tick/internal/cli/flags_test.go`
   - `/Users/leeovery/Code/tick/internal/cli/unknown_flag_test.go`

### Non-blocking

1. **Suggested consolidation approach**: Reduce `flags_test.go` to core `ValidateFlags` unit tests only (no-flags baseline, unknown rejection, value-taking skip, boolean non-skip, short flag rejection, two-level help hint format, global flag passthrough -- one test each). Move all per-command coverage (flag counts, per-command acceptance/rejection, ready-rejects-ready, blocked-rejects-blocked, remove short flag, exhaustive global flag cross-product, comprehensive value-taking tests) into `flag_validation_test.go`. Keep `unknown_flag_test.go` as pure integration tests through `App.Run()` but trim to a representative sample since unit-level coverage is exhaustive. Remove exact duplicates (TestRemoveAcceptsShortFlag, the global-flags-on-create overlap).

2. **Test file naming**: `flags_test.go` vs `flag_validation_test.go` is a confusing distinction. Consider renaming to make the scope of each file clearer (e.g., `validate_flags_test.go` for unit tests of the `ValidateFlags` function, `command_flags_test.go` for per-command metadata and drift detection tests).
