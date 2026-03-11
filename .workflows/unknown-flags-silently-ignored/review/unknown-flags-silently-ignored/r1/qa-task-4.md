# QA: Task 4 — Validate global flag pass-through and value-taking flag skipping

## STATUS: Complete
## FINDINGS_COUNT: 0 blocking issues

## Implementation

This task requires two things:

1. **Global flags not rejected by command-level validation**: Implemented in `/Users/leeovery/Code/tick/internal/cli/flags.go:129-131`. The `ValidateFlags` function checks `globalFlagSet[arg]` before checking per-command flags, and returns `continue` (passes through) for any global flag. The `globalFlagSet` map at lines 20-30 contains all 9 global flag variants: `--quiet`, `-q`, `--verbose`, `-v`, `--toon`, `--pretty`, `--json`, `--help`, `-h`.

2. **Value-taking flags skip the next argument**: Implemented in `/Users/leeovery/Code/tick/internal/cli/flags.go:139-142`. When a flag is found in `cmdFlags` and its `FlagDef.TakesValue` is true, the loop index `i` is incremented to skip the next argument. This correctly handles cases where a flag value might look like another flag (e.g., `--description --not-a-flag`).

Both acceptance criteria items are fully met:
- Global flags (--quiet, --verbose, --toon, --pretty, --json, --help/-h/-q/-v) are not rejected
- Value-taking flags correctly skip the value argument during validation

### Edge cases from task definition

- **Global flags interspersed with command args**: Handled. Global flags appearing anywhere in the args list are accepted because `ValidateFlags` iterates all args and checks `globalFlagSet` for each flag-like arg.
- **--ready/--blocked on ready/blocked commands**: The `init()` function at lines 92-95 derives `ready` and `blocked` flag sets from `list` by explicitly excluding `--ready`/`--blocked` (and vice versa). `TestReadyRejectsReady` and `TestBlockedRejectsBlocked` in `flag_validation_test.go` verify this behavior.

## Test Adequacy

### Under-testing concerns

None. Coverage is thorough across multiple test files:

- **`flags_test.go:110-115`**: `TestValidateFlags/"it handles global flags interspersed with command args"` -- verifies `--verbose` and `--json` interspersed among `create` args.
- **`flag_validation_test.go:189-203`**: `TestGlobalFlagsAcceptedOnAnyCommand` -- exhaustive cross-product of all 9 global flags x all 20 commands. This is the strongest test: if any command incorrectly rejects a global flag, this catches it.
- **`flag_validation_test.go:205-229`**: `TestGlobalFlagsMixedWithCommandFlags` -- tests global flags mixed with command-specific flags on `create`, `list`, and `update`.
- **`flag_validation_test.go:231-308`**: `TestValueTakingFlagSkipping` -- comprehensive coverage of value skipping: flag-like values consumed as values, consecutive value-taking flags, boolean flags not consuming next arg, all value-taking flags on `create` and `update`.
- **`flag_validation_test.go:163-187`**: `TestReadyRejectsReady` and `TestBlockedRejectsBlocked` -- edge case coverage for the derived flag sets.
- **`flags_test.go:46-63`**: Unit tests for value-taking skip and boolean non-skip in `ValidateFlags`.

### Over-testing concerns

Minor overlap exists between `TestValidateFlags/"it handles global flags interspersed with command args"` (flags_test.go:110) and `TestGlobalFlagsMixedWithCommandFlags/"global flags mixed with command flags on create"` (flag_validation_test.go:206). Both test global flags + create command flags together. This is a minor issue -- the tests are in different files and have slightly different focuses (the former is in the core unit test file, the latter is in the broader validation test file). Not blocking.

`TestRemoveAcceptsShortFlag` (flag_validation_test.go:310-324) duplicates `TestValidateFlags/"it accepts -f on remove"` (flags_test.go:76-81). Same assertion, same args. Non-blocking but worth consolidating.

## Code Quality

**Project conventions**: Followed. Uses stdlib `testing` only, `t.Run()` subtests, helper naming matches "it does X" pattern, error wrapping with `fmt.Errorf`. No testify.

**SOLID principles**: Good. `ValidateFlags` has a single responsibility (checking args against a flag set). The `CommandFlags` type parameter makes it testable and decoupled from the global registry. `FlagDef` struct is minimal and focused.

**DRY**: Good use of `copyFlagsExcept` to derive `ready`/`blocked` flag sets from `list` rather than duplicating them. The `globalFlagSet` map is defined once and shared between `ValidateFlags` and `parseArgs`/`applyGlobalFlag` (though note `applyGlobalFlag` uses a switch rather than the map -- this is fine since they serve different purposes: one sets struct fields, the other is a set membership check).

**Complexity**: Low. `ValidateFlags` is a single linear loop with clear branching. No nesting beyond one level.

**Modern idioms**: Appropriate Go patterns. Map-based set membership, struct-based flag metadata, functional parameter passing.

**Readability**: Clear. Comments explain the "why" (e.g., numeric value detection at line 124-127, global flag skip at line 129). Function and variable names are descriptive.

**Security**: N/A for CLI flag parsing.

**Performance**: Fine. Linear scan of args with map lookups (O(1) amortized). No concerns.

## Findings

### Non-blocking

1. **Minor test duplication**: `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:310-324` (`TestRemoveAcceptsShortFlag`) duplicates `/Users/leeovery/Code/tick/internal/cli/flags_test.go:76-81` (`TestValidateFlags/"it accepts -f on remove"`). Both test the exact same scenario with the same args. Consider consolidating. (Severity: non-blocking)

2. **Minor test overlap**: `/Users/leeovery/Code/tick/internal/cli/flags_test.go:110-115` and `/Users/leeovery/Code/tick/internal/cli/flag_validation_test.go:206-209` both test global flags mixed with command flags on `create`. The overlap is slight since they use different arg combinations, but worth noting. (Severity: non-blocking)
