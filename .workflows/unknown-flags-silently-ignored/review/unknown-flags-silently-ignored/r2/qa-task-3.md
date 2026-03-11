TASK: Wire validation into parseArgs and both dispatch paths

ACCEPTANCE CRITERIA:
- parseArgs returns error for unknown flags before the subcommand
- Pre-subcommand error format: unknown flag "{flag}". Run 'tick help' for usage.
- Post-subcommand unknown flags produce: unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.
- version and help commands excluded from flag validation
- Both dispatch paths (main switch and doctor/migrate) validate flags
- Two-level commands validate against correct sub-subcommand flag set
- The dep add --blocks bug report scenario is tested and rejected
- All existing tests pass (parseArgs call sites updated)

STATUS: Complete

SPEC CONTEXT: The spec requires all commands to reject unrecognised flags with a clear error message. Two dispatch paths exist: doctor/migrate (dispatched before format resolution) and the main switch (after format resolution). Both must validate flags. version and help are excluded. Two-level commands (dep add/remove, note add/remove) use the fully-qualified command name in the error but the parent command in the help reference. Pre-subcommand unknown flags use a simpler error format (no command name since none identified yet).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - parseArgs error return: internal/cli/app.go:337-358 (4th return value, error check at line 348-349)
  - parseArgs call site handling: internal/cli/app.go:30-34
  - version exclusion: internal/cli/app.go:42-45 (early return before validation)
  - help exclusion: internal/cli/app.go:48-53 (early return before validation)
  - Doctor dispatch validation: internal/cli/app.go:57-61
  - Migrate dispatch validation: internal/cli/app.go:63-68
  - Main switch validation: internal/cli/app.go:99-104
  - qualifyCommand helper: internal/cli/app.go:366-380
- Notes: All acceptance criteria are met. The implementation follows the spec's design: parseArgs gains an error return for pre-subcommand unknown flags, qualifyCommand determines the fully-qualified command name for two-level commands, ValidateFlags is called from three locations (doctor path, migrate path, main switch path). version and help are structurally excluded via early returns. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - Pre-subcommand unknown flag: flags_test.go:60-77 ("it rejects unknown flag before subcommand") -- tests App.Run with "--bogus" before "list", verifies exit code 1 and correct error format without command name
  - version exclusion: unknown_flag_test.go:233-248 ("it does not validate flags for version command")
  - help exclusion: unknown_flag_test.go:250-266 ("it does not validate flags for help command")
  - Doctor dispatch path: unknown_flag_test.go:31 ("it rejects --unknown on doctor") via noFlagCommands table
  - Migrate dispatch path: unknown_flag_test.go:67 ("it rejects --priority on migrate") via commandsWithFlags table
  - List rejection: unknown_flag_test.go:65 ("it rejects --force on list")
  - Create rejection: unknown_flag_test.go:63 ("it rejects --force on create")
  - Known flags on create: unknown_flag_test.go:215-227 ("it accepts --priority 1 on create")
  - Bug report scenario (dep add --blocks): unknown_flag_test.go:172-196 -- full dispatch with actual tasks in store, verifies the exact spec error message
  - Two-level commands (dep add, dep remove, note add, note remove): unknown_flag_test.go:97-132 -- verifies fully-qualified name in error and parent in help reference
  - Short flags: unknown_flag_test.go:137-168 -- tests -x rejection on show, list, dep add
- Notes: All required tests from the task specification are present. Tests exercise both unit-level (ValidateFlags directly) and integration-level (App.Run dispatch) paths. The version/help exclusion tests verify the commands succeed without errors but do not pass unknown flags (e.g., --unknown) to prove flags are truly not validated. This is a minor gap noted in r1 as well, but acceptable since exclusion is structural (early return) rather than conditional.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, t.TempDir for isolation, fmt.Errorf error wrapping, error surfaced via Stderr with "Error:" prefix
- SOLID principles: Good. ValidateFlags has single responsibility (validate args against flag set). qualifyCommand has single responsibility (determine fully-qualified command name). Both are exported and independently testable. The dispatch flow in Run() follows open/closed -- new commands only need a commandFlags entry and a switch case.
- Complexity: Low. qualifyCommand is 14 lines with a simple switch. parseArgs change is one additional conditional (3 lines). The three ValidateFlags call sites follow the same pattern.
- Modern idioms: Yes. Multiple return values for parseArgs error, map lookups for flag validation, nil map safety in Go (ValidateFlags works even if command key is missing from commandFlags since nil map lookups return zero values safely).
- Readability: Good. The dispatch flow in Run() reads clearly top-to-bottom: parse -> version -> help -> doctor/migrate (with validation) -> format resolution -> qualify + validate -> dispatch. Comments on qualifyCommand explain the fallthrough behavior for unknown sub-subcommands.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The version/help exclusion tests could be strengthened by passing unknown flags (e.g., `[]string{"tick", "version", "--unknown"}`) to prove flags are truly not validated, not just that the base commands succeed. This was noted in r1 and remains valid but non-blocking since the exclusion is structural.
- qualifyCommand still has no direct unit test. While thoroughly covered via integration tests, a small table-driven unit test would document edge cases (empty subArgs, unknown sub-subcommand fallthrough) explicitly.
