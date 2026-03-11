TASK: Add drift-detection test between commandFlags and help registry

ACCEPTANCE CRITERIA:
- A test exists that fails if a flag is added to commandFlags but not to the corresponding help.go commandInfo.Flags (or vice versa)
- The test currently passes, confirming the two registries are in sync
- Two-level commands (dep add, dep remove, note add, note remove) are handled appropriately
- Short flag aliases (e.g., -f) are handled correctly in the comparison
- go test ./internal/cli/ -count=1 passes

STATUS: Complete

SPEC CONTEXT: The specification defines a central commandFlags registry in flags.go and a separate help registry in help.go, both listing per-command flags. These two sources must stay in sync. The drift-detection test ensures that adding a flag to one without the other causes a test failure, preventing silent divergence.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/flag_validation_test.go:310-375 (TestCommandFlagsMatchHelp)
- Notes: The test iterates all entries in commandFlags, skips two-level commands (those containing a space), then performs bidirectional comparison of long flags (--prefixed) between commandFlags and the help registry's flagInfo entries. The approach is sound and well-structured.

TESTS:
- Status: Adequate
- Coverage:
  - Bidirectional check: flags in commandFlags but missing from help (lines 347-358), and flags in help but missing from commandFlags (lines 361-373)
  - Two-level commands (dep add, dep remove, note add, note remove) correctly skipped since help groups them under parent commands (dep, note) which have no individual per-subcommand flags
  - Short flag aliases handled correctly: commandFlags entries like "-f" are filtered out (only "--" prefixed compared), and help entries like "--force, -f" are split and only long forms retained
  - Commands excluded from validation (version, help) correctly not present in commandFlags, so they are never iterated
  - Derived commands (ready, blocked) are in both registries and will be checked since they are single-level names
  - Test would fail if: (1) a new long flag added to commandFlags but not help, (2) a new long flag added to help but not commandFlags, (3) a command in commandFlags has no help entry (caught by findCommand nil check at line 323)
- Notes: One minor gap -- the test does not iterate the help registry (commands slice) to verify that every help entry has a corresponding commandFlags entry. If a new command were added to help.go but not to commandFlags, this test would not catch it. However, this gap is mitigated by the existing TestFlagValidationAllCommands test which explicitly lists all commands and checks their flag counts, plus the fact that ValidateFlags would panic/error at runtime for a command missing from commandFlags. This is a non-blocking observation.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming convention, no external test libraries.
- SOLID principles: Good. Single responsibility -- test does one thing (drift detection). The test is cleanly separated from other validation tests.
- Complexity: Low. Straightforward iteration with two map comparisons per command. No complex logic.
- Modern idioms: Yes. Uses map[string]bool for set operations, sort.Strings for deterministic error output.
- Readability: Good. Clear comments explaining the skip logic for two-level commands and short flags. Variable names (flagsFromRegistry, flagsFromHelp) clearly communicate intent.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The test only iterates commandFlags to find commands, not the help registry's commands slice. A command added to help.go but not commandFlags would not be caught by this test. This is a minor gap since other tests and runtime behavior would surface the issue, but adding a reverse check (iterating commands and verifying commandFlags entries exist, excluding "version" and "help") would make the drift detection fully bidirectional at the command level as well as the flag level.
