---
scope: unknown-flags-silently-ignored
cycle: 1
source: review
total_proposed: 1
gate_mode: gated
---
# Review Tasks: Unknown Flags Silently Ignored (Cycle 1)

## Task 1: Consolidate overlapping flag validation test coverage
status: pending
severity: high
sources: qa-task-7, qa-task-2, qa-task-3, qa-task-4, qa-task-6

**Problem**: Task 3-1 was not implemented. Significant test duplication exists across three files: `flags_test.go`, `flag_validation_test.go`, and `unknown_flag_test.go`. There is 1 exact duplicate (`TestRemoveAcceptsShortFlag`), 7+ near-duplicate patterns testing the same behavior at the same abstraction level, and unclear ownership boundaries between files. This was identified in the original analysis cycle and a task was created to address it, but no consolidation was performed.

**Solution**: Establish clear file ownership and remove redundant tests:
- `flags_test.go` owns core `ValidateFlags` unit tests (function-level behavior: no-flags baseline, unknown rejection, value-taking skip, boolean non-skip, short flag rejection, two-level help hint format, global flag passthrough, negative number handling)
- `flag_validation_test.go` owns per-command flag metadata correctness (flag counts, per-command acceptance/rejection, ready/blocked cross-exclusion, exhaustive global flag cross-product, comprehensive value-taking coverage, drift detection)
- `unknown_flag_test.go` owns integration tests through `App.Run()` (end-to-end unknown flag rejection, bug report scenario, excluded commands, global flag pass-through)

**Outcome**: Each test file has a clear, non-overlapping scope. No exact duplicates remain. Unit-level tests that are strict subsets of more comprehensive tests in the same file category are removed. All flag validation behavior remains tested at the appropriate level.

**Do**:
1. Remove exact duplicate `TestRemoveAcceptsShortFlag` from `flag_validation_test.go:310-324` (already covered by `flags_test.go:76-81`)
2. Remove `flags_test.go:110-115` global-flags-interspersed subtest (subsumed by `flag_validation_test.go:189-203` exhaustive cross-product and `flag_validation_test.go:205-229` mixed flags tests)
3. Remove `flags_test.go:46-63` value-taking skip and boolean non-skip subtests (strict subsets of `flag_validation_test.go:231-308` comprehensive value-taking coverage)
4. Remove `flags_test.go:24-33` unknown flag on list subtest (subsumed by `flag_validation_test.go:123-133` which tests all commands)
5. Remove `flags_test.go:65-74` short unknown flag subtest (subsumed by `unknown_flag_test.go:137-168` integration test which covers list plus more)
6. Review `flags_test.go:83-108` two-level command subtests -- keep if they test `ValidateFlags` return value format details not covered elsewhere, otherwise remove (covered by `flag_validation_test.go:143-160` and `unknown_flag_test.go:97-132`)
7. Remove `flags_test.go:118-134` `TestCommandFlagsCoversAllCommands` if its assertions are a strict subset of `flag_validation_test.go:9-161` `TestFlagValidationAllCommands` (which already checks all commands exist with correct flag counts)
8. Run `go test ./internal/cli/` to confirm all remaining tests pass
9. Run `go test ./...` for full suite verification
10. Verify no test coverage gaps by checking that each key behavior (unknown flag rejection, value-taking skip, global flag passthrough, two-level error format, short flag handling, drift detection, per-command completeness) has at least one test at the appropriate level

**Acceptance Criteria**:
- Zero exact duplicate tests across the three files
- No unit-level `ValidateFlags` call in `flags_test.go` that is a strict subset of a more comprehensive test in `flag_validation_test.go`
- No unit-level `ValidateFlags` call in `unknown_flag_test.go` (that file is integration-only via `App.Run()`)
- `go test ./internal/cli/` passes with no failures
- `go test ./...` passes with no failures
- Each of these behaviors has at least one test: unknown flag rejection, value-taking flag skip, boolean flag non-skip, short flag rejection, global flag passthrough, two-level command error format, per-command flag completeness, drift detection

**Tests**:
- Run `go test ./internal/cli/ -v` and verify all tests pass
- Run `go test ./... -count=1` and verify full suite passes with no regressions
