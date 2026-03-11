# QA: Task 6 -- Comprehensive unknown-flag regression tests across all commands

## STATUS: Complete
## FINDINGS_COUNT: 0 blocking issues

## Implementation

This task called for comprehensive regression tests verifying unknown flags are rejected across all commands. The implementation is spread across three test files:

1. **`internal/cli/unknown_flag_test.go`** -- Integration-level tests through `App.Run()`:
   - `TestUnknownFlagRejection`: Long-flag rejection for 9 no-flag commands, 7 commands-with-flags, and 4 two-level commands
   - `TestUnknownShortFlagRejection`: Short flag (`-x`) rejection for 3 representative commands (show, list, dep add)
   - `TestBugReportScenario`: Original bug report with real tasks (`dep add --blocks`)
   - `TestCommandsWithFlagsAcceptKnownFlags`: Positive tests for `list --status` and `create --priority`
   - `TestFlagValidationExcludedCommands`: `version` and `help` bypass validation
   - `TestGlobalFlagsNotRejected`: Global flags on 4 representative commands across dispatch paths

2. **`internal/cli/flag_validation_test.go`** -- Unit-level tests on `ValidateFlags()`:
   - `TestFlagValidationAllCommands`: All 7 commands-with-flags accept valid flags, reject unknown; all 13 no-flag commands reject any flag and have zero flags registered
   - `TestReadyRejectsReady` / `TestBlockedRejectsBlocked`: Edge case where `--ready` is unknown on `ready` and `--blocked` is unknown on `blocked`
   - `TestGlobalFlagsAcceptedOnAnyCommand`: All 9 global flags tested on all 20 commands (180 permutations)
   - `TestGlobalFlagsMixedWithCommandFlags`: Interleaved global + command flags on create, list, update
   - `TestValueTakingFlagSkipping`: 7 subtests covering value consumption, boolean-then-unknown, fake values
   - `TestRemoveAcceptsShortFlag`: `-f` and `--force` both accepted on `remove`
   - `TestCommandFlagsMatchHelp`: Drift detection between `commandFlags` registry and help system

3. **`internal/cli/flags_test.go`** -- Core `ValidateFlags` function unit tests:
   - `TestValidateFlags`: 8 subtests covering no flags, known flags, unknown long flags, unknown short flags, bug repro, value skipping, boolean non-skipping, two-level help hints, global flag interleaving
   - `TestCommandFlagsCoversAllCommands`: Registry completeness check for all 21 commands
   - `TestHelpCommand`: Unit test for `helpCommand()` helper
   - `TestFlagValidationWiring`: Integration test for unknown flag before subcommand

### Acceptance criteria mapping:

| Criterion | Covered |
|-----------|---------|
| Short flags (`-x`) rejected across all commands | Yes -- unit level in `flags_test.go:65`, integration level for 3 representative commands in `unknown_flag_test.go:137`. The `ValidateFlags` function handles short flags identically to long flags (same code path), so testing a representative sample at integration level is sufficient. |
| No-flag commands (13 total) reject any flag | Yes -- all 13 tested at unit level in `flag_validation_test.go:136-160`, all 13 tested at integration level in `unknown_flag_test.go` (9 in `noFlagCommands` + 4 in `twoLevelCommands`) |
| Two-level commands use fully-qualified name in error | Yes -- `unknown_flag_test.go:98-132` and `flags_test.go:83-108` |
| Commands with accepted flags still reject unknown ones | Yes -- 7 commands tested in `unknown_flag_test.go:55-95` and `flag_validation_test.go:108-133` |

## Test Adequacy

### Under-testing concerns

None significant. The short-flag integration test (`TestUnknownShortFlagRejection`) only covers 3 commands (show, list, dep add), not all 20+. However, this is not a real gap because:
- The `ValidateFlags` function has a single code path for short flags (line 120-122 of `flags.go` checks `strings.HasPrefix(arg, "-")` which covers both `-x` and `--unknown`)
- The unit-level test in `flags_test.go:65` proves `-x` is rejected
- The 3 integration tests cover representative types: no-flag command, command-with-flags, two-level command
- Testing all 20+ commands with `-x` at integration level would be redundant given the single code path

### Over-testing concerns

There is moderate overlap between the three test files:

1. **`flags_test.go:35` and `unknown_flag_test.go:172`** both test the `dep add --blocks` bug report scenario -- one at unit level, one at integration level. This is acceptable since they test different layers.

2. **`flags_test.go:83` and `unknown_flag_test.go:110`** both verify two-level command error formatting. Again, unit vs integration level makes this reasonable.

3. **`flag_validation_test.go:123` and `flags_test.go:24`** both test rejection of unknown flags on commands -- `flag_validation_test.go` is more comprehensive (all commands) while `flags_test.go` is more focused (one command). Slight redundancy but not problematic.

4. The `TestCommandFlagsMatchHelp` test in `flag_validation_test.go:326` is a drift-detection test that is arguably Phase 3 scope (task `unknown-flags-silently-ignored-3-3` "Add drift-detection test between commandFlags and help registry") rather than Phase 2 regression tests. However, including it here does no harm.

Overall test count is reasonable for the surface area being covered. No bloat.

## Code Quality

- **Project conventions**: Follows project patterns -- stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers, "it does X" naming convention in subtests.
- **SOLID principles**: Good. Tests are focused on behavior, not implementation details. Each test function has a clear single purpose.
- **Complexity**: Low. Table-driven tests with clear iteration. No complex setup or teardown.
- **Modern idioms**: Yes. Uses table-driven tests, subtests, and proper Go test patterns.
- **Readability**: Good. Test names clearly describe what is being verified. Comments explain non-obvious test rationale (e.g., why `setupTick` is needed for some commands but not others).
- **DRY**: The `App` creation pattern is repeated in each test, but this is standard Go test practice -- extracting it into a helper would reduce clarity.

## Findings

No blocking issues found.

### Non-blocking notes

1. **`unknown_flag_test.go:75`**: The `setupTick` field in the `commandsWithFlags` table is always `false` for every entry. The field could be removed to simplify the struct, or a comment could explain it was kept for future tests that need project setup. Very minor.

2. **`flag_validation_test.go:326-391` (`TestCommandFlagsMatchHelp`)**: This drift-detection test overlaps with Phase 3 task `unknown-flags-silently-ignored-3-3`. If Phase 3 creates its own version, one should be removed to avoid duplication. Not a problem for Phase 2 acceptance.

3. **Short flag integration coverage**: While 3 representative commands is sufficient, adding a no-flag command like `init` to `TestUnknownShortFlagRejection` would make the test types more symmetric (currently covers: command-with-flags, no-flag-command, two-level-command for long flags but only the same 3 types for short flags via show/list/dep-add which are no-flag/with-flags/two-level respectively -- actually this IS symmetric, so no issue).
