# Implementation Review: Unknown Flags Silently Ignored

**Plan**: unknown-flags-silently-ignored
**QA Verdict**: Request Changes

## Summary

The core bugfix is solid. All commands now reject unknown flags with correct error messages, both dispatch paths are covered, global flags pass through, value-taking flags skip their values, `dep rm` is normalized to `dep remove`, and `--from=value` syntax is removed. 8 of 9 plan tasks are fully implemented and well-tested. The one incomplete task is Phase 3 Task 3-1 (test consolidation) — the overlapping test coverage identified during analysis was never consolidated. All tests pass (green suite).

## QA Verification

### Specification Compliance

Implementation aligns with specification across all requirements:
- Error message format matches spec exactly: `Error: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.`
- Two-level commands use fully-qualified name in error, parent in help reference
- Global flags accepted on all commands
- Pre-subcommand unknown flags rejected
- `version` and `help` excluded from validation
- All 20+ commands covered

One justified deviation: ready/blocked flag sets exclude both `--ready` and `--blocked` (6 flags each) rather than just the self-referential one (spec implies 7). This is semantically more correct since the flags are mutually exclusive.

### Plan Completion

- [x] Phase 1 acceptance criteria met — all 4 tasks complete
- [x] Phase 2 acceptance criteria met — all 2 tasks complete
- [ ] Phase 3 acceptance criteria met — **Task 3-1 incomplete** (test consolidation not performed)
- [x] No scope creep — all changes are within plan scope

### Code Quality

No issues found. Implementation follows project conventions (stdlib testing, `t.Run()`, `fmt.Errorf` wrapping, DI via struct fields). `ValidateFlags` is clean with single responsibility. `copyFlagsExcept` eliminates flag set duplication. `qualifyCommand` cleanly separates command qualification from validation.

### Test Quality

Tests adequately verify all requirements at both unit and integration levels. The drift-detection test (`TestCommandFlagsMatchHelp`) is a valuable safeguard against future registry divergence. Coverage is thorough across all commands, flag types, and dispatch paths.

However, significant test overlap exists across three files (`flags_test.go`, `flag_validation_test.go`, `unknown_flag_test.go`) — this is the subject of the incomplete Task 3-1:
- 1 exact duplicate (`TestRemoveAcceptsShortFlag`)
- 7+ near-duplicate patterns testing the same behavior at the same abstraction level
- Unclear ownership boundaries between files

### Required Changes

1. **Complete Task 3-1**: Consolidate overlapping flag validation test coverage across `flags_test.go`, `flag_validation_test.go`, and `unknown_flag_test.go`. At minimum:
   - Remove the exact duplicate `TestRemoveAcceptsShortFlag` from `flag_validation_test.go` (already covered in `flags_test.go`)
   - Establish clear ownership: `flags_test.go` owns core `ValidateFlags` unit tests, `flag_validation_test.go` owns per-command metadata/drift tests, `unknown_flag_test.go` owns integration tests through `App.Run()`
   - Remove unit-level tests from one file that are strict subsets of more comprehensive tests in another (e.g., value-taking flag tests in `flags_test.go:46-63` are subsumed by `flag_validation_test.go:231-308`)

## Recommendations

- Consider adding a direct unit test for `qualifyCommand` (`app.go:366-380`) — currently only tested indirectly through integration tests
- `TestFlagValidationExcludedCommands` could be strengthened by passing `--unknown` to `version`/`help` to prove flags are truly not validated (not just that the commands succeed)
- The `setupTick` field in `unknown_flag_test.go:75` `commandsWithFlags` table is always `false` — could be removed or documented
