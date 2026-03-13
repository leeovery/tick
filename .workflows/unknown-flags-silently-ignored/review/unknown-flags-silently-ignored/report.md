# Implementation Review: Unknown Flags Silently Ignored

**Plan**: unknown-flags-silently-ignored
**QA Verdict**: Approve

## Summary

All 10 tasks across 4 phases are implemented and tested. The bugfix achieves its core objective: every command now rejects unknown flags with clear, spec-compliant error messages. Central validation via `ValidateFlags` eliminates the silent-skip anti-pattern. Dead code removed, prerequisite normalizations applied, comprehensive regression tests in place. Zero blocking issues found. Two minor non-blocking observations remain from test consolidation (one strict-subset test overlap, one dispatch test in a unit test file). All tests pass (`go test ./...` green).

## QA Verification

### Specification Compliance

Implementation aligns with the specification across all requirements:

- Error format matches spec: `unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.`
- All 20 commands covered in `commandFlags` registry
- Global flags pass through without rejection
- Two-level commands use fully-qualified name in error, parent in help reference
- Pre-subcommand unknown flags rejected with simpler format (no command name)
- `version` and `help` excluded from validation
- Both dispatch paths (doctor/migrate and main switch) validate flags
- `dep rm` renamed to `dep remove`, `--from=value` syntax removed
- Dead `strings.HasPrefix(arg, "-")` skip logic removed from all 5 parsers

One approved deviation: ready/blocked flag sets exclude both `--ready` and `--blocked` (6 flags each vs spec's 7). This was reviewed in r1 and accepted as semantically more correct since the excluded flags would be contradictory on those commands.

### Plan Completion

- [x] Phase 1 acceptance criteria met (all 4 tasks complete)
- [x] Phase 2 acceptance criteria met (all 2 tasks complete)
- [x] Phase 3 acceptance criteria met (task 3-1 correctly superseded by 4-1; tasks 3-2 and 3-3 complete)
- [x] Phase 4 acceptance criteria met (task 4-1 complete with minor residual)
- [x] All 10 tasks completed
- [x] No scope creep — no unplanned files or features

### Code Quality

No issues found. Implementation follows project conventions:
- stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` isolation
- `ValidateFlags` is a pure function with injected flags map (testable, SRP)
- `copyFlagsExcept` prevents drift between ready/blocked and list flag sets
- `helpCommand` helper cleanly handles two-level command name extraction
- `qualifyCommand` correctly determines fully-qualified command for validation
- Numeric value guard prevents `-1` (negative priority) from triggering false flag rejection

### Test Quality

Tests adequately verify requirements. Three-file ownership model is clear:
- `flags_test.go` — core `ValidateFlags` unit tests
- `flag_validation_test.go` — per-command metadata correctness, drift detection, comprehensive value-taking coverage
- `unknown_flag_test.go` — integration tests through `App.Run()`

All spec testing requirements covered: unknown flag rejection, global flag passthrough, `dep add --blocks` bug report, short flag rejection, pre-subcommand rejection.

### Required Changes

None.

## Recommendations

1. **Minor test overlap** — `flags_test.go:17-21` ("it returns nil for args with known command flags") tests `ValidateFlags("create", ...)` with 2 flags, which is a strict subset of `flag_validation_test.go:109-113` testing all 8 create flags. Consider removing the subset test to fully satisfy the consolidation acceptance criterion, or accept it as intentional documentary smoke test.

2. **Dispatch test in unit file** — `TestFlagValidationWiring` in `flags_test.go:59-77` tests pre-subcommand unknown flag rejection through `App.Run()`. Per the established ownership model, dispatch tests belong in `unknown_flag_test.go`. Consider relocating.

3. **Version/help exclusion tests** — The exclusion tests verify the commands succeed but don't pass unknown flags like `--bogus` to prove flags are truly unvalidated. Strengthening with unknown flag args would make the tests more definitive.

4. **Drift detection direction** — `TestCommandFlagsMatchHelp` iterates `commandFlags` to find help mismatches but doesn't iterate the help registry in the reverse direction. A command added to help without `commandFlags` wouldn't be caught. Consider adding reverse iteration.
