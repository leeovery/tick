# QA: Task 1 -- Normalize dep rm to dep remove and remove --from=value syntax

## STATUS: Complete
## FINDINGS_COUNT: 0

## Implementation

### dep rm renamed to dep remove

- `internal/cli/dep.go:25-32`: The switch statement in `handleDep` only routes "add" and "remove" as valid sub-subcommands. "rm" falls through to the default case and returns `unknown dep sub-command 'rm'`.
- `internal/cli/dep.go:19`: Usage message correctly shows `<add|remove>`.
- `internal/cli/dep.go:128-194`: `RunDepRemove` function is properly named and implemented.
- `internal/cli/app.go:374-376`: `qualifyCommand` only recognizes "add" and "remove" as valid dep/note sub-subcommands, consistent with the rename.
- `internal/cli/flags.go:75`: `commandFlags` registry contains `"dep remove"` (not `"dep rm"`).

No residual "rm" handling exists in production code.

### --from=value syntax removed from parseMigrateArgs

- `internal/cli/migrate.go:47-67`: `parseMigrateArgs` only matches `args[i] == "--from"` (exact match). There is no `strings.HasPrefix(args[i], "--from=")` branch. Passing `"--from=beads"` will not match the `--from` case, causing `flags.from` to remain empty, which triggers the error at line 63-64.
- Additionally, the central validator (`ValidateFlags` in `flags.go:115-146`) runs before `parseMigrateArgs` (see `app.go:64-65`). `"--from=beads"` does not match the known flag `"--from"` in `commandFlags["migrate"]`, so it is rejected as an unknown flag before `parseMigrateArgs` is even called.

Both normalizations match the acceptance criteria exactly.

## Test Adequacy

### Under-testing concerns

None. The relevant tests are:

1. **dep rm rejection** (`internal/cli/dep_test.go:878-900`): `TestDepRmReturnsError` verifies that `dep rm` produces exit code 1 and stderr contains "unknown dep sub-command 'rm'". This directly tests the acceptance criterion.

2. **--from=value rejection** (`internal/cli/migrate_test.go:351-376`): `TestMigrateFromFlagSyntax` covers both the positive case (`--from value` works, exit 0) and the negative case (`--from=beads` fails with exit 1 and stderr mentioning "--from"). This covers the acceptance criterion and the edge case.

3. **dep remove works** (`internal/cli/dep_test.go:361-619`, `793-875`): Extensive existing tests for `dep remove` using `"remove"` (not "rm") cover the happy path, error cases, partial IDs, and persistence.

4. **Edge case: --from with empty value** (`internal/cli/migrate_test.go:116-126`): Test "migrate command with empty --from value returns error" covers the `--from ""` edge case mentioned in the task.

### Over-testing concerns

None. Tests are focused and not redundant. Each test verifies a distinct behavior.

## Code Quality

- **Project conventions**: Followed. Uses stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on helpers, error wrapping with `fmt.Errorf`.
- **SOLID principles**: Good. `handleDep` has single responsibility (routing). `parseMigrateArgs` has single responsibility (parsing). Validation is separated into `ValidateFlags`.
- **Complexity**: Low. Simple switch statements and exact string matching. No complex control flow.
- **Modern idioms**: Yes. Idiomatic Go switch/case, nil map lookups return zero value correctly.
- **Readability**: Good. Usage messages are clear, error messages are descriptive and consistent.
- **Security**: N/A for this task.
- **Performance**: N/A for this task.

## Findings

No blocking issues.

## Non-blocking Notes

- The error message when `--from=beads` is passed comes from the central validator ("unknown flag") rather than from `parseMigrateArgs` ("--from flag is required"), because validation runs first (`app.go:64-65`). This is fine and actually desirable -- it's consistent with how all commands handle unknown flags. The test correctly asserts on the common substring "--from" which appears in both error paths.
