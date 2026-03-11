# QA: Task 5 — Remove dead silent-skip logic from command parsers

## STATUS: Complete
## FINDINGS_COUNT: 0 blocking issues

## Implementation

The acceptance criteria require that no `strings.HasPrefix(arg, "-")` skip logic remains in five functions: `parseCreateArgs`, `parseUpdateArgs`, `parseDepArgs`, `RunNoteAdd`, and `parseRemoveArgs`.

Verified via grep across all five target files -- zero occurrences of `strings.HasPrefix` in any of them:

- **`parseCreateArgs`** (`internal/cli/create.go:31-101`): Default case at line 93-98 assigns first positional arg as title. No flag filtering.
- **`parseUpdateArgs`** (`internal/cli/update.go:38-119`): Default case at line 111-116 assigns first positional arg as task ID. No flag filtering.
- **`parseDepArgs`** (`internal/cli/dep.go:37-48`): Copies all args into positional slice directly. No flag filtering.
- **`RunNoteAdd`** (`internal/cli/note.go:39-87`): Copies all args into positional slice directly. No flag filtering.
- **`parseRemoveArgs`** (`internal/cli/remove.go:20-39`): Only checks for `--force`/`-f`; everything else is treated as positional ID. No generic flag skipping.

The only remaining `strings.HasPrefix(arg, "-")` calls in the cli package are in the new central validation system:
- `internal/cli/flags.go:120` -- Part of `ValidateFlags()`, correctly skipping non-flag positional args during validation
- `internal/cli/app.go:348` -- Part of `parseArgs()`, correctly rejecting unknown flags before subcommand identification

These are new validation code, not the old silent-skip pattern. Implementation matches acceptance criteria exactly.

### Edge cases from task description

1. **Parser with known flags alongside skip removal (create/update/remove)**: All three parsers retain their known flag handling (e.g., `--priority` in create, `--title` in update, `--force`/`-f` in remove) while the old catch-all skip has been removed. Central validation in `ValidateFlags()` now rejects unknown flags before these parsers are called.

2. **Positional-only parsers (dep/note add) that use skip for extraction**: Both `parseDepArgs` and `RunNoteAdd` now treat all args as positional. This is safe because central validation has already rejected any unknown flags before dispatch.

## Test Adequacy

### Under-testing concerns

None. The cleanup is verified through multiple layers of testing:

1. **`TestParseRemoveArgs`** (`internal/cli/remove_test.go:1715-1867`): 9 subtests covering `parseRemoveArgs` behavior -- single ID, multiple IDs, deduplication, `--force`/`-f` extraction, ordering. No tests for old skip behavior remain.

2. **`TestUnknownFlagRejection`** (`internal/cli/unknown_flag_test.go:15-133`): Comprehensive end-to-end regression tests through `App.Run()` covering no-flag commands (9 commands), commands-with-flags (7 commands), and two-level commands (4 commands).

3. **`TestUnknownShortFlagRejection`** (`internal/cli/unknown_flag_test.go:137-168`): Short flag `-x` rejection across show, list, dep add.

4. **`TestBugReportScenario`** (`internal/cli/unknown_flag_test.go:172-196`): The original `dep add --blocks` bug report scenario tested with actual tasks.

5. **`TestFlagValidationAllCommands`** (`internal/cli/flag_validation_test.go:9-161`): Unit tests against `ValidateFlags()` for all commands, verifying both acceptance and rejection.

6. **`TestGlobalFlagsNotRejected`** (`internal/cli/unknown_flag_test.go:270-327`): Global flags pass through without triggering rejection.

Tests would fail if the skip logic were re-introduced (since central validation would no longer be reached, or parsers would silently consume unknown flags).

### Over-testing concerns

There is some overlap between `TestFlagValidationAllCommands` (unit-level) and `TestUnknownFlagRejection` (integration-level), but this is reasonable layering -- the unit tests verify `ValidateFlags()` in isolation while the integration tests verify the full dispatch path through `App.Run()`. This is not redundant; they test different things.

## Code Quality

- **Project conventions**: Followed. stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers. Handler signatures match the established pattern.
- **SOLID principles**: Good. Flag knowledge lives with the command (Single Responsibility), validation is centralized (DRY), parsers only handle their specific known flags.
- **Complexity**: Low. Each parser's default case is a simple positional assignment. `parseDepArgs` is now just 11 lines. `parseRemoveArgs` is a clean loop with switch on known flags.
- **Modern idioms**: Yes. Idiomatic Go switch/case, slice copy idiom (`append([]string{}, args...)`).
- **Readability**: Good. Each parser is straightforward -- known flags are handled explicitly, default case handles positional args. No ambiguity about what happens with unknown input.
- **Security**: No concerns.
- **Performance**: No concerns.

## Findings

No blocking or non-blocking issues found. The cleanup was executed cleanly -- all five parsers had their silent-skip logic removed without affecting known flag handling or positional argument extraction.
