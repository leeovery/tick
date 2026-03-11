---
topic: unknown-flags-silently-ignored
cycle: 1
total_proposed: 3
---
# Analysis Tasks: unknown-flags-silently-ignored (Cycle 1)

## Task 1: Consolidate overlapping flag validation test coverage
status: approved
severity: high
sources: duplication

**Problem**: Three test files (flags_test.go, flag_validation_test.go, unknown_flag_test.go) written by separate task executors contain substantial overlapping test coverage. Specific overlaps: (1) TestFlagValidationEndToEnd in flag_validation_test.go and TestBugReportScenario in unknown_flag_test.go test the exact same dep add --blocks E2E scenario with identical setup, args, and assertions. (2) TestFlagValidationWiring in flag_validation_test.go and TestUnknownFlagRejection in unknown_flag_test.go both test unknown flag rejection through App.Run() for the same commands (dep add, doctor, migrate, list, create). (3) TestFlagValidationWiring tests global flag acceptance and known flag acceptance on create, duplicating TestGlobalFlagsNotRejected and TestCommandsWithFlagsAcceptKnownFlags in unknown_flag_test.go. (4) "it returns error for flag on command with no flags" in flags_test.go iterates the same 13 no-flag commands as TestFlagValidationAllCommands in flag_validation_test.go. (5) "it skips global flags without error" in flags_test.go is a strict subset of TestGlobalFlagsAcceptedOnAnyCommand in flag_validation_test.go.

**Solution**: Consolidate tests into a clear two-file structure: flags_test.go for ValidateFlags unit tests (direct function calls), unknown_flag_test.go for E2E dispatch tests through App.Run(). Remove all overlapping tests from flag_validation_test.go that are already covered by unknown_flag_test.go, and remove overlapping tests from flags_test.go that are already covered by flag_validation_test.go. Move any unique tests from flag_validation_test.go (version/help exclusion tests) into unknown_flag_test.go. Keep flag_validation_test.go focused exclusively on ValidateFlags unit tests that are not already in flags_test.go.

**Outcome**: Each test scenario exists in exactly one file. flags_test.go owns ValidateFlags unit tests. unknown_flag_test.go owns E2E dispatch tests. flag_validation_test.go contains only unique unit tests not covered elsewhere (or is removed entirely if all its unique tests are relocated). No test coverage is lost.

**Do**:
1. Read all three test files fully to map every subtest and what it covers (unit vs dispatch, which command, which scenario).
2. Identify the canonical location for each test scenario based on the rule: ValidateFlags unit tests in flags_test.go, App.Run() dispatch tests in unknown_flag_test.go.
3. Remove TestFlagValidationEndToEnd from flag_validation_test.go (exact duplicate of TestBugReportScenario in unknown_flag_test.go).
4. Remove the following subtests from TestFlagValidationWiring that are already covered by unknown_flag_test.go: "it rejects unknown flag on dep add via full dispatch", "it rejects unknown flag on doctor", "it rejects unknown flag on migrate", "it rejects unknown flag on list", "it rejects unknown flag on create", "it accepts known flags on create through dispatch".
5. Move "it does not validate flags for version command" and "it does not validate flags for help command" from TestFlagValidationWiring to unknown_flag_test.go (these are unique dispatch tests not covered elsewhere). If TestFlagValidationWiring is now empty, remove it.
6. Remove "it returns error for flag on command with no flags" subtest from flags_test.go (covered more comprehensively by the no-flag-commands section in TestFlagValidationAllCommands in flag_validation_test.go).
7. Remove "it skips global flags without error" subtest from flags_test.go (strict subset of TestGlobalFlagsAcceptedOnAnyCommand in flag_validation_test.go).
8. If flag_validation_test.go now contains only TestFlagValidationAllCommands and related tests that are purely unit-level ValidateFlags calls, consider whether they should merge into flags_test.go or remain separate. Keep them separate if the file is still substantial; merge if only a few tests remain.
9. Run `go test ./internal/cli/ -count=1` and verify all tests pass with zero failures.
10. Run `go test ./internal/cli/ -count=1 -v` and verify no test name appears to duplicate another's scenario.

**Acceptance Criteria**:
- No two test files contain tests that exercise the same scenario at the same layer (unit or dispatch)
- TestFlagValidationEndToEnd is removed; TestBugReportScenario remains as the canonical E2E bug-report regression test
- Version/help exclusion dispatch tests exist in exactly one file
- All existing test scenarios are still covered (no coverage loss)
- `go test ./internal/cli/ -count=1` passes

**Tests**:
- Run `go test ./internal/cli/ -count=1` -- all tests pass
- Grep for "dep add.*blocks" across test files -- appears in at most 2 places (one unit test in flags_test.go, one E2E test in unknown_flag_test.go)
- Grep for "version command" across test files -- appears in exactly one file
- Grep for "help command" across test files -- flag exclusion test appears in exactly one file (note: TestHelpCommand testing helpCommand() is a separate unit test and is fine)

## Task 2: Derive ready/blocked flag sets programmatically from list
status: approved
severity: medium
sources: duplication, architecture

**Problem**: The commandFlags registry in flags.go defines the "ready" entry (7 flags) and "blocked" entry (7 flags) by copy-pasting 6 of the 8 flags from the "list" entry, minus --ready and --blocked respectively. The specification explicitly defines the relationship as "ready = list flags minus --ready" and "blocked = list flags minus --blocked". The current implementation encodes this relationship by duplicating the literal flag definitions rather than expressing the derivation. If a new filter flag is added to list, both ready and blocked must be updated independently, creating a three-way sync risk.

**Solution**: Compute the ready and blocked flag sets programmatically from the list entry using an init() function. Copy list's flag map, then delete the excluded flag for each derived command.

**Outcome**: The mathematical relationship (ready = list - {--ready}, blocked = list - {--blocked}) is expressed in code and self-maintaining. Adding a filter flag to list automatically includes it in ready and blocked.

**Do**:
1. In flags.go, remove the literal "ready" and "blocked" entries from the commandFlags map literal.
2. Add an init() function that: (a) copies the list entry's map into a new map for "ready", deleting "--ready"; (b) copies the list entry's map into a new map for "blocked", deleting "--blocked"; (c) assigns both to commandFlags.
3. Write a small helper function (e.g., `copyFlagsExcept(source map[string]FlagDef, exclude string) map[string]FlagDef`) that creates a shallow copy of the flag map and deletes the excluded key.
4. Verify the existing tests TestFlagValidationAllCommands (which checks flag counts: ready=7, blocked=7) and TestReadyRejectsReady / TestBlockedRejectsBlocked still pass.
5. Run `go test ./internal/cli/ -count=1`.

**Acceptance Criteria**:
- The commandFlags map literal contains no "ready" or "blocked" entries
- An init() function derives ready and blocked from list's flags
- commandFlags["ready"] has exactly 7 entries (list's 8 minus --ready)
- commandFlags["blocked"] has exactly 7 entries (list's 8 minus --blocked)
- `go test ./internal/cli/ -count=1` passes

**Tests**:
- TestFlagValidationAllCommands verifies flag counts (ready=7, blocked=7) and that all valid flags are accepted
- TestReadyRejectsReady verifies --ready is rejected on ready command
- TestBlockedRejectsBlocked verifies --blocked is rejected on blocked command
- Run `go test ./internal/cli/ -count=1` -- all pass

## Task 3: Add drift-detection test between commandFlags and help registry
status: approved
severity: medium
sources: architecture

**Problem**: Two independent registries define flag information for the same commands: commandFlags in flags.go maps commands to their valid flags with type metadata, and commands in help.go lists flags with display metadata (name, arg hint, description). These must be kept in sync manually. Adding a flag to commandFlags without updating help.go means help text is incomplete. Adding a flag to help.go without updating commandFlags means the validator rejects a documented flag. There is no compile-time or test-time check ensuring the two registries agree.

**Solution**: Add a test that iterates commandFlags entries and verifies each non-global flag name appears in the corresponding commandInfo.Flags slice, and vice versa. This is a lightweight guard that catches drift without restructuring the registries.

**Outcome**: Any future flag addition that updates one registry but not the other is caught by a failing test, preventing silent drift between validation and documentation.

**Do**:
1. In flag_validation_test.go (or flags_test.go -- whichever is the canonical location for ValidateFlags unit tests after Task 1), add a new test function TestCommandFlagsMatchHelp.
2. For each entry in commandFlags, look up the corresponding commandInfo by name using findCommand(). Note: for two-level commands like "dep add", the help registry uses the parent name ("dep") which has no flags listed -- skip two-level commands in this check, or handle them by checking the parent's help entry. The help registry groups "dep" and "note" as single entries without per-subcommand flags, so skip commands containing a space.
3. For each single-level command with flags in commandFlags, extract the flag names from commandFlags[cmd] (excluding short flags like "-f" which are combined with long flags in help.go as "--force, -f").
4. Extract flag names from the commandInfo.Flags slice (note: help.go uses "--force, -f" format in the Name field -- split on ", " to get individual flag names).
5. Assert that the set of long flags in commandFlags matches the set of flag names in commandInfo.Flags. Allow the help entry to contain combined short forms (e.g., "--force, -f") where commandFlags has separate "--force" and "-f" entries.
6. Also check the reverse: flags in help.go should all appear in commandFlags.
7. Run `go test ./internal/cli/ -count=1` and verify the new test passes (confirming the registries are currently in sync).

**Acceptance Criteria**:
- A test exists that fails if a flag is added to commandFlags but not to the corresponding help.go commandInfo.Flags (or vice versa)
- The test currently passes, confirming the two registries are in sync
- Two-level commands (dep add, dep remove, note add, note remove) are handled appropriately (either skipped or matched against the parent command)
- Short flag aliases (e.g., -f) are handled correctly in the comparison
- `go test ./internal/cli/ -count=1` passes

**Tests**:
- TestCommandFlagsMatchHelp passes on the current codebase
- Manually verify by temporarily adding a flag to commandFlags["list"] without updating help.go -- the test should fail (then revert)
