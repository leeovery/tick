TASK: Consolidate overlapping flag validation test coverage (3-1 / tick-2ec1bc, superseded by 4-1 / tick-b15fda)

ACCEPTANCE CRITERIA:
1. No two test files contain tests that exercise the same scenario at the same layer (unit or dispatch)
2. TestFlagValidationEndToEnd is removed; TestBugReportScenario remains as the canonical E2E bug-report regression test
3. Version/help exclusion dispatch tests exist in exactly one file
4. All existing test scenarios are still covered (no coverage loss)
5. go test ./internal/cli/ -count=1 passes

STATUS: Issues Found

SPEC CONTEXT: The specification requires all commands reject unknown flags with clear error messages. Testing section requires: each command rejects unknown flags, global flags not rejected, dep add --blocks bug report covered, short flags rejected, pre-subcommand unknown flags rejected. The consolidation task (3-1) was created during Phase 3 analysis to address over-testing identified in the r1 review. Task 3-1 was not implemented; task 4-1 (Phase 4 review remediation) was created to address the same scope and is the commit that performed the actual work (commit c9cbcdc).

IMPLEMENTATION:
- Status: Mostly Implemented (via task 4-1)
- Location: internal/cli/flags_test.go, internal/cli/flag_validation_test.go, internal/cli/unknown_flag_test.go
- Notes: Task 3-1 was correctly skipped in favor of task 4-1 which performed the consolidation. The major consolidation items from r1 review have all been addressed: TestFlagValidationEndToEnd removed, TestRemoveAcceptsShortFlag removed, TestCommandFlagsCoversAllCommands removed, global-flags-interspersed subtest removed, value-taking/boolean subtests removed from flags_test.go, unknown-flag-on-list subtest removed, short-unknown-flag subtest removed, two-level command subtests removed from flags_test.go. File ownership model is mostly clear: flags_test.go has core ValidateFlags unit tests, flag_validation_test.go has per-command metadata and comprehensive validation, unknown_flag_test.go has integration tests via App.Run().

TESTS:
- Status: Adequate with minor residual overlap
- Coverage: All key behaviors remain tested: unknown flag rejection (unit + integration), value-taking flag skipping, boolean flag non-skip, short flag rejection, global flag passthrough, two-level command error format, per-command flag completeness, drift detection (TestCommandFlagsMatchHelp), bug report scenario, version/help exclusion, pre-subcommand unknown flag.
- Notes: Two minor issues remain (see non-blocking notes). The massive over-testing identified in r1 (8 overlap patterns) has been reduced substantially.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run() subtests, "it does X" naming, t.TempDir() for isolation
- SOLID principles: Good -- clear separation of concerns between the three test files
- Complexity: Low -- test structure is straightforward table-driven patterns
- Modern idioms: Yes
- Readability: Good -- file ownership is now largely clear. flags_test.go is core unit tests, flag_validation_test.go is per-command metadata correctness, unknown_flag_test.go is integration
- Issues: See non-blocking notes

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
1. **Residual unit-layer overlap in flags_test.go:17-21**: The subtest "it returns nil for args with known command flags" calls ValidateFlags("create", ["My Task", "--priority", "3", "--description", "desc"]) which is a strict subset of flag_validation_test.go:109-114 which tests all 8 valid flags on create. Both are unit-level tests exercising the same scenario (known flags accepted on create). Per AC1, no two files should exercise the same scenario at the same layer. This is a minor violation -- removing this subtest from flags_test.go would resolve it with no coverage loss.
2. **Dispatch test in wrong file**: TestFlagValidationWiring (flags_test.go:59-77) tests unknown flag before subcommand via App.Run(). Per the established ownership model, dispatch-level tests belong in unknown_flag_test.go. This test is the only coverage for the pre-subcommand error path, so it should be moved rather than removed. This does not technically violate AC1 since no equivalent test exists in unknown_flag_test.go, but it violates the file ownership model established by the review-tasks-c1 specification.
