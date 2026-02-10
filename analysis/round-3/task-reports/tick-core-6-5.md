# Task 6-5: Remove doctor command from help text (V5 Only -- Phase 6 Refinement)

## Task Plan Summary

The `printUsage` function in `internal/cli/cli.go` advertised a `tick doctor` command ("Run diagnostics and validation") that was never implemented -- no entry existed in the `commands` map. Running `tick doctor` produced an "Unknown command" error, misleading users and AI agents. The task required removing the doctor line from help text, verifying `tick help` no longer mentions it, and confirming `tick doctor` still returns the expected unknown-command error.

## Note

This is a Phase 6 analysis refinement task that only exists in V5. It addresses help text accuracy found during post-implementation analysis. This is a standalone quality assessment, not a comparison.

## V5 Implementation

### Architecture & Design

The change is architecturally minimal and surgically correct. It targets exactly one function (`printUsage` at line 184 of `cli.go`) and removes a single `fmt.Fprintln` line that advertised the nonexistent `doctor` command.

After the change, the help text entries (lines 189-203 of `cli.go`) are in exact 1:1 correspondence with the `commands` map entries (lines 97-112 of `cli.go`). Both contain exactly 14 commands: `init`, `create`, `update`, `start`, `done`, `cancel`, `reopen`, `list`, `show`, `ready`, `blocked`, `dep`, `stats`, `rebuild`. No phantom entries remain.

The diff touches three files:
1. `internal/cli/cli.go` -- removes the doctor help line (1 line deletion)
2. `internal/cli/cli_test.go` -- adds two new test cases (29 lines added)
3. `docs/workflow/` -- updates tracking metadata (status: completed, task pointer advanced)

This is a textbook example of a minimal, focused change with no side effects.

### Code Quality

**Production code change** (cli.go line 202, removed):
```go
fmt.Fprintln(w, "  doctor    Run diagnostics and validation")
```

The removal preserves the existing formatting pattern. The surrounding lines maintain consistent column alignment with the two-space indent and padded command names. No blank lines were introduced or removed around the deletion point, keeping the visual structure intact.

**Test code** (cli_test.go lines 255-282):

Two test cases were added under the existing `TestSubcommandRouting` test function:

Test 1 -- "it does not advertise doctor command in help output" (lines 255-267):
```go
t.Run("it does not advertise doctor command in help output", func(t *testing.T) {
    dir := t.TempDir()
    var stdout, stderr bytes.Buffer
    code := Run([]string{"tick"}, dir, &stdout, &stderr, false)
    if code != 0 {
        t.Fatalf("expected exit code 0, got %d", code)
    }
    if strings.Contains(stdout.String(), "doctor") {
        t.Error("help output should not advertise the unimplemented doctor command")
    }
})
```

This test invokes `tick` with no subcommand (triggering `printUsage`), then asserts the stdout does not contain the substring "doctor". The assertion is broad enough to catch any re-introduction of the word "doctor" in help text, which is appropriate for a regression guard.

Test 2 -- "it returns unknown command error for doctor" (lines 269-282):
```go
t.Run("it returns unknown command error for doctor", func(t *testing.T) {
    dir := t.TempDir()
    var stdout, stderr bytes.Buffer
    code := Run([]string{"tick", "doctor"}, dir, &stdout, &stderr, false)
    if code != 1 {
        t.Errorf("expected exit code 1, got %d", code)
    }
    expected := "Error: Unknown command 'doctor'. Run 'tick help' for usage.\n"
    if stderr.String() != expected {
        t.Errorf("stderr = %q, want %q", stderr.String(), expected)
    }
})
```

This test confirms that `tick doctor` continues to produce the standard unknown-command error with exit code 1, verifying no accidental handler was registered.

Both tests follow the project's established patterns:
- Use `t.TempDir()` for isolation
- Use `bytes.Buffer` for stdout/stderr capture
- Use descriptive BDD-style test names with "it ..." prefix
- Check both exit code and output content
- Are placed logically within `TestSubcommandRouting`

### Test Coverage

The tests directly address both acceptance criteria from the task plan:

| Acceptance Criterion | Test | Status |
|---------------------|------|--------|
| `tick help` does not list the doctor command | "it does not advertise doctor command in help output" | Covered |
| `tick doctor` still returns "Unknown command" error | "it returns unknown command error for doctor" | Covered |
| No functional commands are affected | Implicit -- the only change is a deletion from help text; existing command tests remain untouched | Covered |

The task plan's "Tests" section specified:
1. "Verify `tick help` output does not contain 'doctor'" -- directly covered
2. "Verify `tick doctor` still returns 'Unknown command' error (unchanged behavior)" -- directly covered

### Spec Compliance

The task plan specified two concrete "Do" items:

1. **"In `internal/cli/cli.go`, remove the line `fmt.Fprintln(w, "  doctor    Run diagnostics and validation")` from `printUsage`."** -- Done exactly. The diff shows this exact line removed from `printUsage`.

2. **"Verify `tick help` output no longer mentions doctor."** -- Verified via the new test case that asserts `!strings.Contains(stdout.String(), "doctor")`.

All three acceptance criteria are satisfied:
- Help does not list doctor (verified by test and code inspection)
- No functional commands are affected (only help text changed; commands map untouched)
- Help output test was updated (two new test cases added as regression guards)

### golang-pro Skill Compliance

| Requirement | Status | Notes |
|------------|--------|-------|
| Handle all errors explicitly | N/A | No new error paths introduced |
| Write table-driven tests with subtests | Partial | Uses subtests via `t.Run()` but not table-driven format. Justified: only 2 cases with different setup/assertions, table-driven would add complexity for no benefit |
| Document all exported functions | N/A | No new exported symbols |
| Propagate errors with `fmt.Errorf("%w", err)` | N/A | No new error propagation |
| Use gofmt | Compliant | Code formatting is consistent with project style |
| No panic for error handling | Compliant | Uses `t.Fatalf` and `t.Errorf` appropriately |
| No ignored errors | Compliant | All return values checked |

The change is too small to trigger most golang-pro constraints. The constraints that do apply (formatting, error checking in tests, subtest usage) are all satisfied.

## Quality Assessment

### Strengths

1. **Surgical precision**: The production change is exactly one line removed. No collateral modifications, no reformatting, no unnecessary refactoring. This is the ideal diff for a help-text-only fix.

2. **Complete test coverage for the task scope**: Both acceptance criteria have dedicated regression tests. The tests will catch any future re-introduction of the doctor command in help text.

3. **Test placement and style consistency**: The new tests are placed in the correct test function (`TestSubcommandRouting`), follow the project's BDD naming convention ("it does not...", "it returns..."), and use the same buffer/exit-code assertion pattern as adjacent tests.

4. **Defensive test design**: Test 1 uses `strings.Contains(stdout.String(), "doctor")` rather than checking for the exact removed line, meaning it will catch any variant of doctor re-appearing in help output. Test 2 verifies the exact error message string, ensuring the unknown-command path remains stable.

5. **No behavioral changes**: The `commands` map is untouched. Only the cosmetic help text was modified, ensuring zero risk of functional regression.

### Weaknesses

1. **Minor: `tick help` itself is broken**: Running `tick help` triggers the unknown-command error because "help" is not in the `commands` map and only an empty subcommand triggers `printUsage`. The error message says `Run 'tick help' for usage` but that command itself fails. This pre-existed and is outside this task's scope, but the tests exercise `tick` (no subcommand) rather than `tick help`, which means the task plan's wording "Verify `tick help` output" is not literally tested. The tests use `Run([]string{"tick"}, ...)` which is the correct code path for triggering `printUsage`, so this is purely a semantic mismatch between the plan's wording and the actual mechanism, not a bug in the implementation.

2. **No test for help text completeness**: There is no test asserting that every entry in the `commands` map has a corresponding entry in `printUsage` (or vice versa). Such a test would prevent future phantom entries. This is outside the task scope but would be a valuable addition.

### Overall Quality Rating

**Excellent** -- The implementation is minimal, correct, well-tested, and follows all project conventions. The one-line production change does exactly what was specified. The two test cases provide complete regression coverage for the acceptance criteria. The code style is consistent with the surrounding codebase. There are no defects, no unnecessary changes, and no missing requirements. The only observations are pre-existing architectural issues (the `tick help` routing gap) that are clearly outside the scope of this task.
