AGENT: duplication
FINDINGS:
- FINDING: Overlapping test coverage between flag_validation_test.go and unknown_flag_test.go
  SEVERITY: high
  FILES: internal/cli/flag_validation_test.go:316-339, internal/cli/unknown_flag_test.go:172-196
  DESCRIPTION: TestFlagValidationEndToEnd in flag_validation_test.go and TestBugReportScenario in unknown_flag_test.go both test the exact same scenario: dep add --blocks is rejected through full App.Run() dispatch with actual tasks. Both create the same two tasks, call app.Run with identical args, assert exit code 1, and check for the same error substring. These were written by separate task executors (Ttick-3abf54 and Ttick-f52ed8) who each independently covered the original bug report.
  RECOMMENDATION: Remove TestFlagValidationEndToEnd from flag_validation_test.go. TestBugReportScenario in unknown_flag_test.go is the canonical location for the bug-report regression test, and it has a clearer name.

- FINDING: Overlapping dispatch-level unknown flag rejection tests across flag_validation_test.go and unknown_flag_test.go
  SEVERITY: high
  FILES: internal/cli/flag_validation_test.go:176-335, internal/cli/unknown_flag_test.go:15-289
  DESCRIPTION: TestFlagValidationWiring in flag_validation_test.go and TestUnknownFlagRejection in unknown_flag_test.go both test unknown flag rejection through App.Run() for the same commands. Specifically: (1) "dep add --blocks" rejection tested in both TestFlagValidationWiring and TestUnknownFlagRejection's two-level commands section. (2) Unknown flag before subcommand tested in both. (3) Doctor, migrate, list, create rejection tested in both. (4) Global flags not rejected tested in both TestFlagValidationWiring and TestGlobalFlagsNotRejected. (5) Known flags accepted on create tested in both TestFlagValidationWiring and TestCommandsWithFlagsAcceptKnownFlags. The two files overlap substantially in their end-to-end coverage.
  RECOMMENDATION: Consolidate into a single file. Keep unknown_flag_test.go as the canonical end-to-end regression suite (it uses table-driven patterns and covers all command types systematically). Move any unique tests from TestFlagValidationWiring that are not covered by unknown_flag_test.go (version/help exclusion tests) into unknown_flag_test.go, then remove the overlapping tests from flag_validation_test.go. Rename flag_validation_test.go to focus only on ValidateFlags unit tests (which do not overlap).

- FINDING: Overlapping no-flag command rejection between flags_test.go and flag_validation_test.go
  SEVERITY: medium
  FILES: internal/cli/flags_test.go:120-132, internal/cli/flag_validation_test.go:141-165
  DESCRIPTION: TestValidateFlags in flags_test.go has a subtest "it returns error for flag on command with no flags" that iterates the same 13 no-flag commands and asserts --anything is rejected. TestFlagValidationAllCommands in flag_validation_test.go has a section iterating the same 13 no-flag commands asserting --unknown is rejected. Both test ValidateFlags directly (not through App.Run), both iterate the same command list, both assert the error contains the flag name. Written by separate executors.
  RECOMMENDATION: Remove the "it returns error for flag on command with no flags" subtest from flags_test.go. The coverage in flag_validation_test.go is more comprehensive (it also verifies flag counts and tests commands-with-flags).

- FINDING: Overlapping global flag acceptance tests between flags_test.go and flag_validation_test.go
  SEVERITY: medium
  FILES: internal/cli/flags_test.go:46-54, internal/cli/flag_validation_test.go:194-208
  DESCRIPTION: TestValidateFlags in flags_test.go tests that all 9 global flags are accepted on "show" command. TestGlobalFlagsAcceptedOnAnyCommand in flag_validation_test.go tests all 9 global flags across all 20 commands. The flags_test.go test is a strict subset of the flag_validation_test.go test.
  RECOMMENDATION: Remove the "it skips global flags without error" subtest from flags_test.go since flag_validation_test.go provides strictly more coverage of the same behavior.

- FINDING: Overlapping dep add --blocks unit test between flags_test.go and flag_validation_test.go
  SEVERITY: low
  FILES: internal/cli/flags_test.go:35-44, internal/cli/flag_validation_test.go:316-339
  DESCRIPTION: Both files have a unit-level test for the dep add --blocks bug report scenario calling ValidateFlags directly. flags_test.go tests it as "it returns error for unknown flag on dep add (bug repro)" and flag_validation_test.go tests it as "it rejects dep add --blocks (original bug report)" via full dispatch. The unit-level test in flags_test.go is reasonable to keep since it tests the validator directly, but the duplication with the dispatch test is worth noting.
  RECOMMENDATION: Keep the unit test in flags_test.go (it tests ValidateFlags in isolation). The dispatch-level duplicate was already addressed in finding 1 above.

- FINDING: Near-duplicate flag definitions for ready/blocked in commandFlags registry
  SEVERITY: medium
  FILES: internal/cli/flags.go:59-91
  DESCRIPTION: The "ready" entry (7 flags) and "blocked" entry (7 flags) in commandFlags each repeat 6 identical flag definitions from "list" (8 flags). "ready" is list minus --ready, "blocked" is list minus --blocked. If a new filter flag is added to list, it must be manually added to both ready and blocked. This is a duplication-by-design that matches the spec ("ready" = list flags minus --ready, "blocked" = list flags minus --blocked), but the map literal form makes drift likely.
  RECOMMENDATION: Derive ready and blocked flag sets programmatically from list's flags using a helper function in an init() block, e.g. listFlagsExcept("--ready") and listFlagsExcept("--blocked"). This keeps the spec-defined relationship explicit and eliminates the three-way sync requirement.

SUMMARY: Three test files (flags_test.go, flag_validation_test.go, unknown_flag_test.go) were written by separate task executors and contain substantial overlapping coverage. The most impactful consolidation is merging dispatch-level tests into unknown_flag_test.go and keeping flags_test.go focused on ValidateFlags unit tests. The ready/blocked flag definitions in flags.go are a lower-priority extraction candidate.
