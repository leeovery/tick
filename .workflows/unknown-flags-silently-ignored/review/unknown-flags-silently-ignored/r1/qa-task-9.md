# QA: Task 9 -- Add drift-detection test between commandFlags and help registry

## STATUS: Complete
## FINDINGS_COUNT: 0 blocking issues

## Implementation

The task required adding a test that detects drift between the `commandFlags` registry in `flags.go` and the `commands` help registry in `help.go`. The implementation is `TestCommandFlagsMatchHelp` at `internal/cli/flag_validation_test.go:326-391`.

The test:
1. Iterates every entry in `commandFlags`.
2. Skips two-level commands (containing a space, e.g., "dep add") since help groups them under the parent command -- matches acceptance criteria.
3. Uses `findCommand()` to look up the corresponding `commandInfo` in the help registry. If not found, fails with an error (line 339).
4. Extracts long flag names (`--` prefixed) from `commandFlags`, ignoring short aliases like `-f` (lines 344-349).
5. Parses help's `flagInfo.Name` field (which may contain combined forms like `--force, -f`) by splitting on `", "` and keeping long forms (lines 353-361).
6. Asserts bidirectional sync:
   - Every long flag in `commandFlags` appears in help (lines 364-375).
   - Every long flag in help appears in `commandFlags` (lines 378-389).
7. Sorts missing flags for deterministic error output (lines 372, 385).

All acceptance criteria from the analysis task are met:
- A test exists that fails if a flag is added to `commandFlags` but not help (or vice versa).
- Two-level commands are skipped appropriately.
- Short flag aliases are handled correctly (excluded from commandFlags set, parsed out of help's combined format).
- The test is in `flag_validation_test.go`, the canonical location for ValidateFlags-adjacent unit tests after the Task 1 consolidation.

## Test Adequacy

### Under-testing concerns

None significant. The test covers all single-level commands in both directions. Two-level commands (dep add, dep remove, note add, note remove) are intentionally skipped because the help registry groups them under parent commands ("dep", "note") which have no individual flags. This is documented in the analysis task's acceptance criteria and is the correct approach given the help.go data model.

One minor observation: the test does not verify that the `version` and `help` commands in the help registry are absent from `commandFlags` (since they are excluded from validation). However, this is outside the scope of this task -- the test's purpose is to catch flag drift for validated commands, not to enforce the exclusion policy.

### Over-testing concerns

None. The test is focused and necessary. Each subtest checks one direction of the sync for one command. No redundant assertions.

## Code Quality

- **Project conventions**: Follows project patterns -- stdlib `testing` only, `t.Run()` subtests, "it does X" naming convention in subtest names. Uses `sort.Strings` for deterministic output.
- **SOLID principles**: Good. The test has a single responsibility (drift detection). It uses the same `findCommand()` and `commandFlags` that production code uses, meaning it tests real registries rather than mocks.
- **Complexity**: Low. Simple iteration with set comparison. Clear code paths.
- **Modern idioms**: Yes. Uses `map[string]bool` as sets, range iteration, `strings.Split`/`strings.HasPrefix` for parsing.
- **Readability**: Good. Comments explain the logic at each step (lines 343, 351, 363, 377). Variable names (`flagsFromRegistry`, `flagsFromHelp`) are self-documenting.
- **DRY**: Acceptable. The forward and reverse checks are structurally similar but differ in direction and error messages. Not worth abstracting.

## Findings

No blocking issues.

### Non-blocking notes

1. `flag_validation_test.go:327`: The test iterates `commandFlags` which is a map -- iteration order is non-deterministic. This does not affect correctness (each command is tested independently via `t.Run`) but means verbose test output order varies between runs. This is a standard Go idiom for map iteration in tests and not worth changing.

2. `flag_validation_test.go:336-339`: The error message for missing `findCommand` could be more specific about which registry is incomplete (help vs commandFlags). Current message is adequate for debugging but could be clearer. Non-blocking style preference.
