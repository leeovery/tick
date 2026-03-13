TASK: Consolidate overlapping flag validation test coverage (unknown-flags-silently-ignored-4-1, tick-b15fda)

ACCEPTANCE CRITERIA:
1. Zero exact duplicate tests across the three files (flags_test.go, flag_validation_test.go, unknown_flag_test.go)
2. No unit-level ValidateFlags call in flags_test.go that is a strict subset of a more comprehensive test in flag_validation_test.go
3. No unit-level ValidateFlags call in unknown_flag_test.go (that file is integration-only via App.Run())
4. go test ./internal/cli/ passes with no failures
5. go test ./... passes with no failures
6. Each of these behaviors has at least one test: unknown flag rejection, value-taking flag skip, boolean flag non-skip, short flag rejection, global flag passthrough, two-level command error format, per-command flag completeness, drift detection

STATUS: Issues Found

SPEC CONTEXT: The specification requires all commands reject unknown flags with a specific error format. This task is a Phase 4 remediation addressing the r1 review finding that Phase 3 Task 3-1 (test consolidation) was never implemented. The r1 review identified 1 exact duplicate (TestRemoveAcceptsShortFlag) and 7+ near-duplicate/redundant patterns across flags_test.go, flag_validation_test.go, and unknown_flag_test.go. The consolidation goal is to establish clear ownership: flags_test.go owns core ValidateFlags unit tests, flag_validation_test.go owns per-command metadata/drift tests, unknown_flag_test.go owns integration tests through App.Run().

IMPLEMENTATION:
- Status: Partially Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/flags_test.go (77 lines, 6 tests)
  - /Users/leeovery/Code/tick/internal/cli/flag_validation_test.go (376 lines, 9 test functions)
  - /Users/leeovery/Code/tick/internal/cli/unknown_flag_test.go (328 lines, 7 test functions)
- Notes:
  Significant consolidation was performed. The following r1-identified redundancies were removed:
  - TestRemoveAcceptsShortFlag exact duplicate: REMOVED from flag_validation_test.go
  - "it handles global flags interspersed with command args": REMOVED from flags_test.go
  - "it rejects short unknown flags": REMOVED from flags_test.go
  - "it uses parent command in help hint for two-level commands": REMOVED from flags_test.go
  - "it returns error for unknown flag" on list: REMOVED from flags_test.go
  - TestCommandFlagsCoversAllCommands: REMOVED from flags_test.go (subsumed by TestFlagValidationAllCommands)

  However, one subset violation remains (see Criterion 2 analysis below).

TESTS:
- Status: Adequate (minor over-testing remains)
- Coverage:
  All 8 required behaviors have test coverage:
  1. Unknown flag rejection: TestFlagValidationAllCommands (unit), TestUnknownFlagRejection (integration)
  2. Value-taking flag skip: TestValueTakingFlagSkipping (unit, 8 subtests)
  3. Boolean flag non-skip: TestValueTakingFlagSkipping/"boolean flags on update do not consume next arg" (unit)
  4. Short flag rejection: TestUnknownShortFlagRejection (integration, 3 commands)
  5. Global flag passthrough: TestGlobalFlagsAcceptedOnAnyCommand (unit, 9x20 cross-product), TestGlobalFlagsNotRejected (integration)
  6. Two-level command error format: TestUnknownFlagRejection two-level section (integration, 4 commands), TestHelpCommand (unit)
  7. Per-command flag completeness: TestFlagValidationAllCommands with flag counts
  8. Drift detection: TestCommandFlagsMatchHelp
- Notes:
  Criterion 3 satisfied: unknown_flag_test.go contains zero direct ValidateFlags calls -- all tests go through App.Run().

  Criterion 1 (zero exact duplicates): Satisfied. No test across the three files performs the exact same call with the exact same args at the exact same level of abstraction.

  Criterion 2 (no strict subsets in flags_test.go): One borderline violation.
  flags_test.go:17-21 ("it returns nil for args with known command flags") calls ValidateFlags("create", ["My Task", "--priority", "3", "--description", "desc"], commandFlags).
  flag_validation_test.go:109-113 ("it accepts all valid flags for create") calls ValidateFlags("create", [all 8 flag args], commandFlags).
  The flags_test.go test uses 2 flags; the flag_validation_test.go test uses all 8 flags. Both are unit-level ValidateFlags calls testing the same behavior (known flags accepted on create). The former is a strict subset of the latter.

  This is a minor issue. The test has documentary value as a basic happy-path test in the core function's test file, and is fast/small. But it strictly violates the stated acceptance criterion.

CODE QUALITY:
- Project conventions: Followed. stdlib testing, t.Run() subtests, t.TempDir() isolation, t.Helper() where appropriate. No testify or external test libs.
- SOLID principles: Good. Clear file ownership boundaries established post-consolidation: flags_test.go = core ValidateFlags unit tests, flag_validation_test.go = per-command metadata/drift/comprehensive coverage, unknown_flag_test.go = integration via App.Run().
- Complexity: Low. Each test is straightforward with clear assertions.
- Modern idioms: Yes. Table-driven tests used appropriately in flag_validation_test.go and unknown_flag_test.go.
- Readability: Good. Test names follow "it does X" convention. File organization is clearer after consolidation.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- flags_test.go:17-21 ("it returns nil for args with known command flags") is a strict subset of flag_validation_test.go:109-113 ("it accepts all valid flags for create"). This technically violates acceptance criterion 2. Consider removing it or differentiating it (e.g., testing a different command not covered in the comprehensive suite). However, this is a minor issue since the test is small, fast, and serves as a basic happy-path smoke test in the core function's test file.
- The dep add --blocks bug-repro test exists at both unit level (flags_test.go:24-33) and integration level (unknown_flag_test.go:172-196). These test different layers, so this is appropriate, but worth noting they verify the same scenario.
