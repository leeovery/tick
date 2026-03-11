TASK: Comprehensive unknown-flag regression tests across all commands

ACCEPTANCE CRITERIA:
- Every command in Command Flag Inventory has at least one test confirming unknown-flag rejection
- Short flags (-x) tested and rejected across simple and two-level commands
- Long flags (--unknown) tested and rejected across all command types
- Two-level commands show fully-qualified name in error ("dep add") but parent in help ref (tick help dep)
- dep add --blocks bug report test (from Phase 1) still passes after cleanup
- Commands with accepted flags tested with a flag outside their set
- Global flags confirmed to not trigger rejection
- Error format: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.

STATUS: Complete

SPEC CONTEXT: The spec requires all 20 commands to reject unrecognised flags with the format `unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.` Two-level commands use fully-qualified name in error but parent in help reference. Global flags must pass through without triggering rejection.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/unknown_flag_test.go (all 328 lines)
- Notes: This is a test-only task. The tests exercise the validation implemented in internal/cli/flags.go:ValidateFlags(). The test file contains 6 top-level test functions covering rejection, short flags, bug report scenario, known-flag acceptance, excluded commands, and global flag passthrough. Supplementary unit-level coverage exists in internal/cli/flag_validation_test.go and internal/cli/flags_test.go.

TESTS:
- Status: Adequate
- Coverage:
  - All 20 commands from the spec covered: 9 no-flag commands in noFlagCommands table (line 17), 7 commands-with-flags (line 55), 4 two-level commands (line 98)
  - Short flag rejection: 3 representative commands (show, list, dep add) covering all command types (line 137)
  - Bug report scenario: dep add --blocks with real tasks (line 172)
  - Known flag acceptance: list --status and create --priority (line 200)
  - Excluded commands: version and help bypass validation (line 232)
  - Global flag passthrough: --verbose, --json, --quiet on 4 commands across dispatch paths (line 270)
  - Error format verified in every rejection test via strings.Contains
  - Two-level commands verify both fully-qualified name (wantCmdRef) and parent help ref (wantHelpRef) separately (lines 123-129)
- Notes: Short flag tests cover 3 commands rather than all 20, which is acceptable since ValidateFlags has a single code path for all flag-like arguments (strings.HasPrefix(arg, "-")). Unit-level tests in flag_validation_test.go provide exhaustive per-command coverage.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run() subtests, t.TempDir() for isolation, "it does X" naming convention.
- SOLID principles: Good. Each test function has a single clear purpose. Table-driven tests separate data from logic.
- Complexity: Low. Simple table-driven iteration with clear assertions.
- Modern idioms: Yes. Standard Go table-driven test patterns with subtests.
- Readability: Good. Comments explain test rationale (e.g., line 1 doc comment, line 293-294 explaining why init needs clean dir).
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `setupTick` field in the `commandsWithFlags` struct (line 62) is always `false` for every entry. It could be removed to simplify the struct, or a comment could explain why it exists. This was noted in r1 and remains unchanged -- very minor.
