# QA: Task 2 — Reproduce bug and build flag metadata with central validator

## STATUS: Complete
## FINDINGS_COUNT: 0 blocking issues

## Implementation

The task requires: (1) reproducing the bug where `dep add --blocks` is silently ignored, (2) building a flag metadata registry (`FlagDef` with `TakesValue`), and (3) implementing a central `ValidateFlags` function.

All acceptance criteria are met:

- **Unknown flags produce correct error format**: `ValidateFlags` at `/Users/leeovery/Code/tick/internal/cli/flags.go:115-146` generates `unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.` -- matches spec exactly.
- **Two-level command error format**: `helpCommand()` at `flags.go:151-156` extracts the parent for help references (e.g., "dep add" error uses `tick help dep`), while the error message uses the fully-qualified name ("dep add").
- **Value-taking flags skip next arg**: `flags.go:140-142` increments `i` when `def.TakesValue` is true, correctly skipping the value position.
- **The `dep add --blocks` bug scenario is tested and rejected**: `flags_test.go:35-44` explicitly tests this.
- **Short aliases (-f)**: `commandFlags` at `flags.go:80` includes `-f` for `remove`; validator treats short flags identically to long ones.
- **Global flags accepted everywhere**: `globalFlagSet` at `flags.go:20-30` checked before per-command lookup, so global flags pass through.
- **Negative numbers not treated as flags**: `flags.go:125-127` skips args like "-1" where the second character is a digit.
- **Flag metadata per command**: `commandFlags` at `flags.go:33-90` is a comprehensive registry covering all 21 commands (including ready/blocked derived via `init()` at `flags.go:92-95`).
- **Both dispatch paths covered**: `app.go:57-68` validates doctor/migrate before their handlers; `app.go:100-104` validates all other commands via `qualifyCommand` + `ValidateFlags`.
- **version and help excluded**: `app.go:43-53` returns before reaching validation.

### Implementation locations:
- `FlagDef`, `CommandFlags`, `globalFlagSet`, `commandFlags`, `ValidateFlags`, `helpCommand`, `copyFlagsExcept`: `/Users/leeovery/Code/tick/internal/cli/flags.go`
- `qualifyCommand`: `/Users/leeovery/Code/tick/internal/cli/app.go:366-380`
- Wiring in `Run()`: `/Users/leeovery/Code/tick/internal/cli/app.go:57-68` and `app.go:99-104`
- `parseArgs` pre-subcommand unknown flag rejection: `/Users/leeovery/Code/tick/internal/cli/app.go:348-349`

## Test Adequacy

### Under-testing concerns

None. The test coverage is thorough and well-structured:

- **Bug repro**: `flags_test.go:35-44` tests `dep add --blocks` specifically.
- **Value-taking skipping**: `flags_test.go:46-52` and extensive coverage in `flag_validation_test.go:231-308` (covers value that looks like a flag, value that looks like a known flag, value that looks like a global flag, all value-taking flags on create, all on update, boolean flags not consuming next arg).
- **Short flags**: `flags_test.go:65-74` rejects `-x`; `flags_test.go:76-81` accepts `-f` on remove.
- **Two-level commands**: `flags_test.go:83-108` tests all four two-level commands (dep add, dep remove, note add, note remove).
- **Global flags**: `flag_validation_test.go:189-203` tests all 9 global flags across all 20 commands (180 combinations).
- **Global flags mixed with command flags**: `flag_validation_test.go:205-229` tests three representative commands.
- **All commands covered**: `flag_validation_test.go:9-161` tests every command with flags (valid args accepted, unknown rejected, correct flag counts) and every no-flag command.
- **ready/blocked cross-exclusion**: `flag_validation_test.go:163-187` verifies `ready` rejects `--ready` and `blocked` rejects `--blocked`.
- **Drift detection**: `flag_validation_test.go:326-391` cross-checks `commandFlags` against help registry entries bidirectionally.
- **Pre-subcommand rejection**: `flags_test.go:152-170` tests via `App.Run`.
- **Registry completeness**: `flags_test.go:118-134` verifies all expected commands exist and count matches.

### Over-testing concerns

There is some overlap between `flags_test.go` and `flag_validation_test.go` -- for example, both test global flag acceptance on commands, short flag `-f` on remove, and unknown flag rejection. However, this appears intentional: `flags_test.go` contains unit-level tests for `ValidateFlags` covering core logic paths, while `flag_validation_test.go` contains systematic coverage across all commands. The overlap is minor (a few cases) and not burdensome. The drift-detection test in `flag_validation_test.go:326-391` (Task 3-3 scope) is included here but is valuable and non-redundant.

## Code Quality

- **Project conventions**: Follows project patterns -- stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` where appropriate. Error wrapping uses `fmt.Errorf` with `%q` for flag/command quoting.
- **SOLID principles**: Good. `ValidateFlags` has a single responsibility (validation). It accepts `CommandFlags` as a parameter rather than using the global directly, enabling testability. `FlagDef` is a clean value type. `helpCommand` is a small pure function.
- **DRY**: `copyFlagsExcept` in `flags.go:98-107` derives `ready`/`blocked` flags from `list` programmatically, avoiding duplication. Global flags checked in one place (`globalFlagSet`).
- **Complexity**: Low. `ValidateFlags` is a single loop with clear branching. No deep nesting. `qualifyCommand` is a simple switch.
- **Modern idioms**: Uses `map` lookups for O(1) flag checking. No unnecessary allocations.
- **Readability**: Code is well-documented with doc comments on all exported types and functions. Intent is clear throughout. The numeric value check at `flags.go:125-127` has a comment explaining the edge case.
- **Security**: N/A for CLI flag parsing.
- **Performance**: No concerns. Map lookups are efficient. No unnecessary iterations.

## Findings

No blocking issues.

### Non-blocking notes:

1. **Minor overlap**: `TestValidateFlags` in `flags_test.go` and `TestRemoveAcceptsShortFlag`/`TestGlobalFlagsMixedWithCommandFlags` in `flag_validation_test.go` cover overlapping scenarios. Per the plan, Task 3-1 ("Consolidate overlapping flag validation test coverage") is specifically designated to address this, so this is expected at this stage.

2. **`qualifyCommand` lacks unit tests**: The function at `app.go:366-380` is tested indirectly through the wiring test and integration-level tests, but has no direct unit test verifying its return values for each case (single-level command, dep add, dep remove, note add, note remove, unknown sub-subcommand, empty subArgs). This is non-blocking since the function is simple and well-covered indirectly.
