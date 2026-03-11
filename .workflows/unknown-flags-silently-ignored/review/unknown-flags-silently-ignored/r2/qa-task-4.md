TASK: Validate global flag pass-through and value-taking flag skipping (unknown-flags-silently-ignored-1-4)

ACCEPTANCE CRITERIA:
1. Every command with flags accepts all its valid flags through ValidateFlags
2. Every command rejects at least one unknown flag through ValidateFlags
3. ready rejects --ready and blocked rejects --blocked
4. Global flags (--quiet, -q, --verbose, -v, --toon, --pretty, --json, --help, -h) pass through on every command
5. Value-taking flags properly skip their value argument, including when value looks like a flag
6. The original bug report (dep add --blocks) is tested end-to-end and produces exact error message from spec

STATUS: Complete

SPEC CONTEXT: The specification requires all commands reject unknown flags, global flags pass through command-level validation, and value-taking flags correctly consume the next argument during validation. The spec defines 9 global flag variants and per-command flag inventories. The error format for unknown flags is: `Error: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.` Two-level commands use the parent in the help reference.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/flags.go:20-30 (globalFlagSet), :33-90 (commandFlags registry), :92-95 (ready/blocked derivation via copyFlagsExcept), :115-146 (ValidateFlags function)
- Notes: ValidateFlags iterates args, skips non-flag tokens, accepts global flags via globalFlagSet lookup, validates against per-command flag defs, and increments the index past value arguments for TakesValue flags. The ready/blocked flag sets are derived from list via copyFlagsExcept, removing both --ready and --blocked from each (slightly stricter than spec's "same as list minus --ready" / "same as list minus --blocked", but semantically sensible since --blocked on ready and --ready on blocked would be contradictory). No drift from planned behavior.

TESTS:
- Status: Adequate
- Coverage:
  - AC1 (accepts valid flags): TestFlagValidationAllCommands at flag_validation_test.go:108-115 -- table-driven over 7 commands with flags, passing all valid flags
  - AC2 (rejects unknown): TestFlagValidationAllCommands at flag_validation_test.go:123-133 (commands with flags) and :142-160 (13 no-flag commands)
  - AC3 (ready/blocked exclusion): TestReadyRejectsReady at :163-174 and TestBlockedRejectsBlocked at :176-187
  - AC4 (global flags pass-through): TestGlobalFlagsAcceptedOnAnyCommand at :189-203 -- exhaustive cross-product of 9 global flags x 20 commands
  - AC4 (global + command flags mixed): TestGlobalFlagsMixedWithCommandFlags at :205-229 -- tests on create, list, update
  - AC5 (value skipping): TestValueTakingFlagSkipping at :231-308 -- 7 subtests covering flag-like values, consecutive value-taking flags, known/global flags consumed as values, all create/update value-taking flags, boolean flags not consuming next arg
  - AC6 (bug report e2e): TestBugReportScenario at unknown_flag_test.go:172-196 -- full App.Run dispatch with actual tasks, exact error message match
  - Flag count validation: flag_validation_test.go:116-121 verifies commandFlags entry sizes match expected counts
  - Drift detection: TestCommandFlagsMatchHelp at :310-375 cross-checks commandFlags against help registry in both directions
- All 9 expected test names from the task definition are present
- Tests would fail if the feature broke (ValidateFlags returning nil when it should error, or erroring on global flags)
- No over-testing: previous r1 duplicate TestRemoveAcceptsShortFlag has been removed per Phase 4 consolidation

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, "it does X" naming, table-driven patterns, t.TempDir for isolation, fmt.Errorf error wrapping
- SOLID principles: Good. ValidateFlags has single responsibility (arg validation against a flag set). CommandFlags type parameter enables testability without coupling to the global registry. FlagDef is a minimal value type
- Complexity: Low. ValidateFlags is a single linear loop with O(1) map lookups and clear branching (non-flag skip, numeric skip, global skip, command flag check, value skip)
- Modern idioms: Yes. Map-based set membership, struct metadata, functional derivation via copyFlagsExcept
- Readability: Good. Comments explain non-obvious logic (numeric detection, global flag pass-through). Function/variable names are descriptive
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The ready/blocked flag sets exclude both --ready AND --blocked from each, which is slightly stricter than the spec wording ("same as list minus --ready" / "same as list minus --blocked"). This is semantically correct (--blocked on a ready-filtered view is contradictory) but worth noting the spec divergence. The flag counts in tests (6 each) confirm this is intentional.
