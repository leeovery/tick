---
status: in-progress
created: 2026-03-10
cycle: 1
phase: Plan Integrity Review
topic: Unknown Flags Silently Ignored
---

# Review Tracking: Unknown Flags Silently Ignored - Integrity

## Findings

### 1. Missing Outcome and Tests fields across all tasks

**Severity**: Important
**Plan Reference**: All tasks (tick-928bf7, tick-adbf78, tick-8879b7, tick-3abf54, tick-f1dae6, tick-f52ed8)
**Category**: Task Template Compliance
**Change Type**: add-to-task

**Details**:
The canonical task template in task-design.md requires six fields: Problem, Solution, Outcome, Do, Acceptance Criteria, and Tests. All six tasks are missing the "Outcome" field (what success looks like) and the "Tests" field (named test cases). Test descriptions exist within the "Do" sections of some tasks but are not broken out as a separate Tests section. The Outcome field serves a distinct purpose from Acceptance Criteria -- it defines the high-level success state in one sentence, while AC lists specific verifiable conditions. The Tests field explicitly names test cases an implementer should write, separate from implementation steps.

**Current**:
All six tasks have the following structure:
```
Problem: ...
Solution: ...
Do: ...
Acceptance Criteria: ...
Edge Cases: ...
Spec Reference: ...
```

**Proposed**:
Add Outcome and Tests fields to each task. Below are the additions for each task (to be inserted after Solution and after Acceptance Criteria, respectively):

**tick-928bf7** (Normalize dep rm to dep remove and remove --from=value syntax):
```
Outcome: dep rm is renamed to dep remove consistently throughout the codebase, --from=value equals-sign syntax is removed from parseMigrateArgs, and all tests are updated accordingly.
```
```
Tests:
- "it removes dependency via dep remove"
- "it returns unknown sub-command error for dep rm"
- "it accepts --from value (space-separated) on migrate"
- "it rejects --from=value (equals-sign) on migrate"
- "it shows <add|remove> in dep help text"
```

**tick-adbf78** (Reproduce bug and build flag metadata with central validator):
```
Outcome: A ValidateFlags function and commandFlags registry exist that can detect unknown flags on any command, with correct handling of boolean vs value-taking flags, global flag pass-through, and two-level command error formatting.
```
```
Tests:
- "it returns nil for args with no flags"
- "it returns nil for args with known command flags"
- "it returns error for unknown flag"
- "it returns error for unknown flag on dep add (bug repro)"
- "it skips global flags without error"
- "it skips value after value-taking flag"
- "it does not skip value after boolean flag"
- "it rejects short unknown flags"
- "it accepts -f on remove"
- "it uses parent command in help hint for two-level commands"
- "it returns error for flag on command with no flags"
- "it handles global flags interspersed with command args"
```

**tick-8879b7** (Wire validation into parseArgs and both dispatch paths):
```
Outcome: Every command (except version and help) validates flags through the central validator before handler invocation, covering both the main dispatch switch and the doctor/migrate dispatch path.
```
```
Tests:
- "it rejects unknown flag on dep add via full dispatch"
- "it rejects unknown flag before subcommand"
- "it does not validate flags for version command"
- "it does not validate flags for help command"
- "it rejects unknown flag on doctor"
- "it rejects unknown flag on migrate"
- "it rejects unknown flag on list"
- "it rejects unknown flag on create"
- "it accepts known flags on create through dispatch"
```

**tick-3abf54** (Validate global flag pass-through and value-taking flag skipping):
```
Outcome: Comprehensive test coverage confirms that global flags pass through on all commands, value-taking flags correctly skip their argument, ready rejects --ready, blocked rejects --blocked, and the original bug report scenario is verified end-to-end.
```
```
Tests:
- "it accepts all valid flags for each command" (table-driven per command)
- "it rejects unknown flag for each command" (table-driven per command)
- "ready rejects --ready"
- "blocked rejects --blocked"
- "global flags are accepted on any command"
- "global flags mixed with command flags"
- "value that looks like a flag is skipped"
- "consecutive value-taking flags work correctly"
- "it rejects dep add --blocks (original bug report)"
```

**tick-f1dae6** (Remove dead silent-skip logic from command parsers):
```
Outcome: No strings.HasPrefix(arg, "-") skip-and-continue logic remains in any command parser, and all existing tests pass unchanged.
```
```
Tests:
- "all existing create tests pass after skip removal"
- "all existing update tests pass after skip removal"
- "all existing dep tests pass after skip removal"
- "all existing note tests pass after skip removal"
- "all existing remove tests pass after skip removal"
```

**tick-f52ed8** (Comprehensive unknown-flag regression tests across all commands):
```
Outcome: A systematic regression test suite proves every command rejects unknown flags (short and long) with the correct error format after dead code removal.
```
```
Tests:
- "it rejects --unknown on each no-flag command" (init, show, start, done, cancel, reopen, stats, doctor, rebuild)
- "it rejects an invalid flag on each command with flags" (create, update, list, remove, migrate, ready, blocked)
- "it rejects --unknown on two-level commands with fully-qualified name in error" (dep add, dep remove, note add, note remove)
- "it rejects short flag -x on show, list, dep add"
- "it rejects dep add --blocks (bug report scenario)"
- "it accepts known flags on commands that have them"
- "it does not reject global flags on any command"
```

**Resolution**: Fixed
**Notes**: The test lists for tick-adbf78 and tick-8879b7 are extracted from their existing Do sections. For tick-f1dae6, the "tests" are verification that existing tests still pass -- this is appropriate for a cleanup task. For the others, test names are derived from the acceptance criteria.

---

### 2. Task 2-2 (tick-f52ed8) missing dependency on Phase 1 completion

**Severity**: Critical
**Plan Reference**: Phase 2 / tick-f52ed8
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 2-2 (tick-f52ed8, "Comprehensive unknown-flag regression tests") has no blocked_by dependencies. Its description states "Phase 1 wired central flag validation and Task 2-1 removed dead skip logic. Comprehensive regression tests are needed proving every command correctly rejects unknown flags." The tests run through App.Run() and assert that unknown flags are rejected -- this requires Phase 1's central validation to be wired in first.

Task 2-1 (tick-f1dae6) correctly depends on tick-3abf54 (the last Phase 1 task). But tick-f52ed8 has no dependency at all, meaning `tick ready` would surface it as ready before Phase 1 is complete. An implementer picking it up would write tests that fail because the validation isn't wired yet.

The fix is to add a dependency on tick-f1dae6 (Task 2-1), which transitively ensures Phase 1 is complete (via tick-f1dae6's dependency on tick-3abf54) and ensures the dead code removal is done before regression testing.

**Current**:
tick-f52ed8 has no blocked_by dependencies.

**Proposed**:
Add blocked_by dependency: tick-f52ed8 blocked by tick-f1dae6.

This ensures the regression test suite runs after both (a) Phase 1's central validation is wired in (transitively via tick-f1dae6 -> tick-3abf54) and (b) the dead skip logic is removed (tick-f1dae6 directly).

**Resolution**: Fixed
**Notes**: The dependency command would be: `tick dep add tick-f52ed8 tick-f1dae6`

---

### 3. Duplicate bug report end-to-end test across Task 1-4 and Task 2-2

**Severity**: Minor
**Plan Reference**: Phase 1 / tick-3abf54 and Phase 2 / tick-f52ed8
**Category**: Scope and Granularity
**Change Type**: update-task

**Details**:
Both tick-3abf54 (Task 1-4) and tick-f52ed8 (Task 2-2) include an end-to-end integration test for the original bug report scenario (`dep add --blocks`). Task 1-4 Do step 4 says: "End-to-end integration test for original bug report: 'it rejects dep add --blocks (original bug report)' -- full app.Run with setup project containing both tasks." Task 2-2 Do step 3 says: "Add TestBugReportScenario: tick dep add tick-aaa --blocks tick-bbb errors with..."

Writing the same test twice is redundant. Task 1-4 is the right place for this test since it verifies the fix works after Phase 1 wiring. Task 2-2 should reference the existing test rather than recreating it, or simply omit it since the test already exists from Task 1-4.

**Current**:
In tick-f52ed8 Do step 3:
```
3. Add TestBugReportScenario: tick dep add tick-aaa --blocks tick-bbb errors with unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.
```

In tick-f52ed8 Acceptance Criteria:
```
- dep add --blocks bug report explicitly tested
```

**Proposed**:
In tick-f52ed8 Do step 3:
```
3. Verify TestBugReportScenario (written in Task 1-4) still passes after dead code removal. No new test needed -- the existing end-to-end test from tick-3abf54 covers this.
```

In tick-f52ed8 Acceptance Criteria:
```
- dep add --blocks bug report test (from Phase 1) still passes after cleanup
```

**Resolution**: Pending
**Notes**: This is a minor concern. Having the test written twice wouldn't cause failures, just redundancy. The implementer could reasonably skip the duplicate, but explicit guidance is cleaner.

---

### 4. Task 1-3 parseArgs test update guidance references wrong test names

**Severity**: Minor
**Plan Reference**: Phase 1 / tick-8879b7
**Category**: Task Self-Containment
**Change Type**: update-task

**Details**:
Task 1-3 (tick-8879b7) Do step 2 says: "Update all parseArgs call sites in tests (internal/cli/cli_test.go): TestParseArgs and TestParseArgsGlobalFlagsAfterSubcommand." Looking at the actual codebase, the test at line 357 is a subtest within a broader test structure, and line 370 handles global flags after subcommand. The test names in the task description may not match the actual test function names exactly. The task should reference the file and line pattern to match rather than potentially wrong test names, or simply say "all call sites of parseArgs in cli_test.go."

**Current**:
In tick-8879b7 Do step 2:
```
2. Update all parseArgs call sites in tests (internal/cli/cli_test.go):
   - TestParseArgs and TestParseArgsGlobalFlagsAfterSubcommand: update flags, subcmd, _ := parseArgs(...) to flags, subcmd, _, err := parseArgs(...) and check err is nil.
```

**Proposed**:
In tick-8879b7 Do step 2:
```
2. Update all parseArgs call sites in tests (internal/cli/cli_test.go):
   - Find all occurrences of `parseArgs(` in cli_test.go (currently 4 call sites at lines 345, 357, 370, 386). Update each from `flags, subcmd, _ := parseArgs(...)` or `flags, subcmd, subArgs := parseArgs(...)` to include the new error return: `flags, subcmd, _, err := parseArgs(...)` / `flags, subcmd, subArgs, err := parseArgs(...)`. Assert err is nil in each case.
```

**Resolution**: Pending
**Notes**: The current guidance would still lead to the correct outcome since the implementer would search for parseArgs call sites. This is a polish improvement for clarity.
