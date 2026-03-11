# QA: Task 3 — Wire validation into parseArgs and both dispatch paths

## STATUS: Complete
## FINDINGS_COUNT: 0

## Implementation

All four acceptance criteria for this task are correctly implemented:

**1. Both dispatch paths validate flags before invoking handlers**
- Doctor path: `/Users/leeovery/Code/tick/internal/cli/app.go:57-61` — calls `ValidateFlags("doctor", subArgs, commandFlags)` before `handleDoctor()`
- Migrate path: `/Users/leeovery/Code/tick/internal/cli/app.go:63-68` — calls `ValidateFlags("migrate", subArgs, commandFlags)` before `handleMigrate()`
- Main switch path: `/Users/leeovery/Code/tick/internal/cli/app.go:99-104` — calls `ValidateFlags(qualifiedCmd, restArgs, commandFlags)` before the dispatch switch

**2. version and help excluded from flag validation**
- `/Users/leeovery/Code/tick/internal/cli/app.go:42-45` — `version` returns before any validation
- `/Users/leeovery/Code/tick/internal/cli/app.go:48-53` — `help` and `--help`/`-h` return before any validation
- Both are structurally excluded (early return), not by conditional skip — clean approach

**3. Unknown flags before subcommand produce correct error**
- `/Users/leeovery/Code/tick/internal/cli/app.go:347-349` — `parseArgs()` returns error with format: `unknown flag "{flag}". Run 'tick help' for usage.`
- The error is surfaced via the new 4th return value from `parseArgs()`, handled at lines 31-34
- Format matches spec (no `for "command"` since no command identified yet)

**4. Global flags not rejected by command-level validation**
- `/Users/leeovery/Code/tick/internal/cli/flags.go:129-131` — `globalFlagSet` lookup in `ValidateFlags()` skips global flags

**qualifyCommand helper** (`app.go:366-380`): Correctly determines fully-qualified command name for two-level commands (dep/note) by peeking at first positional arg. Returns remaining args after sub-subcommand extraction. Falls through gracefully for unknown sub-subcommands (lets the handler produce its own error).

No drift from the plan detected.

## Test Adequacy

### Under-testing concerns

- No direct unit test for `qualifyCommand()`. The function is simple (14 lines, clear logic) and is thoroughly tested indirectly through `App.Run()` integration tests for all two-level commands (`dep add`, `dep remove`, `note add`, `note remove`). This is acceptable but a unit test would serve as documentation of the edge cases (empty subArgs, unknown sub-subcommand).

### Over-testing concerns

There is some overlap between test files:
- `flag_validation_test.go` tests ValidateFlags for each command directly
- `unknown_flag_test.go` tests the same commands through `App.Run()`
- `flags_test.go` also tests ValidateFlags directly including the dep add bug repro

However, these serve different purposes: unit tests for the validator function vs integration tests for the full dispatch wiring. The overlap is intentional and reasonable given that this task specifically wires validation into dispatch paths — both levels of testing are warranted.

The test for `TestFlagValidationExcludedCommands` only verifies `version` and `help` succeed without error; it does not pass unknown flags to prove they are truly bypassed. However, since these commands return before validation runs (structural exclusion), the test adequately proves the exclusion works.

## Code Quality

- **Project conventions**: Followed. Uses `fmt.Errorf` for error wrapping, `t.Run()` subtests, `t.TempDir()` for isolation, stdlib testing only.
- **SOLID principles**: Good. `ValidateFlags` has a single responsibility. `qualifyCommand` cleanly separates command qualification from validation. Both are exported and testable independently.
- **DRY**: Good. Validation logic written once in `ValidateFlags`, called from three places with the same pattern. The doctor/migrate path duplicates the `if err != nil { fmt.Fprintf; return 1 }` pattern (lines 57-61 and 63-68), but this is the standard error handling pattern used throughout `app.go` — not worth abstracting.
- **Complexity**: Low. `qualifyCommand` has simple switch logic. `parseArgs` error return is a minimal change (one additional check in the existing loop).
- **Modern idioms**: Yes. Multiple return values for `parseArgs` error handling. Map lookups for flag validation.
- **Readability**: Good. Clear comments on `qualifyCommand` and `parseArgs`. The dispatch flow in `Run()` reads top-to-bottom: parse -> version -> help -> doctor/migrate (with validation) -> format resolution -> main switch (with validation).
- **Security**: N/A (CLI flag parsing, no user data exposure).
- **Performance**: Fine. Map lookups are O(1). No unnecessary allocations.

## Findings

No blocking issues found.

## Non-blocking notes

- `qualifyCommand` has no direct unit tests. Consider adding a table-driven test covering: single-level command passthrough, two-level "dep add"/"dep remove"/"note add"/"note remove", empty subArgs for "dep", and unknown sub-subcommand fallthrough. This would document the function's contract explicitly.
- The `TestFlagValidationExcludedCommands` tests for `version` and `help` could be strengthened by passing `--unknown` as a subArg (e.g., `[]string{"tick", "version", "--unknown"}`) to prove flags are truly not validated, not just that the commands succeed.
