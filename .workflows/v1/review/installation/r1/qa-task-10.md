TASK: Extract step-search helper in release_test.go

ACCEPTANCE CRITERIA:
- A single findStepByUses helper exists in release_test.go
- The 4 test cases that previously had inline loops now call the helper
- No change to what is being asserted, only how the step is located

STATUS: Complete

SPEC CONTEXT: This task is an analysis/refactoring task from cycle 1 (Phase 3). It addresses code duplication in the release workflow test file, where four test cases repeated the same nested loop pattern to search for workflow steps by their Uses field. The release workflow tests validate the GitHub Actions release pipeline configuration (tag triggers, goreleaser setup, checkout depth, etc.).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/scripts/release_test.go:244-253
- Notes: The `findStepByUses(w workflow, usesSubstring string) (step, bool)` helper is implemented exactly as specified. It is unexported, file-local (only referenced in release_test.go), and follows the idiomatic Go `(value, bool)` return pattern. The function searches all jobs and steps for a Uses field containing the given substring, returning the first match. An additional companion helper `findStepByName` (lines 256-265) was also added following the same pattern for name-based step searches used in the Homebrew dispatch tests. This is a minor scope addition but consistent with the task's intent.

  Four test cases now use the helper:
  1. Line 198: `findStepByUses(w, "actions/checkout")` -- verifies fetch-depth 0
  2. Line 205: `findStepByUses(w, "goreleaser/goreleaser-action")` -- verifies GITHUB_TOKEN env
  3. Line 218: `findStepByUses(w, "actions/setup-go")` -- verifies go-version-file
  4. Line 234: `findStepByUses(w, "goreleaser/goreleaser-action")` -- verifies release --clean args

  The "workflow runs on ubuntu-latest" test (line 224) correctly retains its inline loop since it searches by RunsOn (a job-level property), not by step Uses field.

TESTS:
- Status: Adequate
- Coverage: This is a refactoring task -- existing tests are the verification. All 4 test cases that previously used inline loops now call findStepByUses. The assertions themselves are unchanged (checking With, Env map values). The tests would still fail if the feature broke (e.g., if the workflow file removed the checkout step or changed its configuration).
- Notes: No new tests were needed since this is a pure refactoring. The existing tests serve as the regression safety net. The task explicitly states "All existing tests in release_test.go pass unchanged" as the test criterion.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Run subtests, t.Helper on helpers. Follows the Go patterns described in CLAUDE.md.
- SOLID principles: Good. Single responsibility -- the helper does one thing (find a step by Uses substring). The extraction reduces coupling between test logic and traversal mechanics.
- Complexity: Low. The helper is a simple nested loop with early return. Cyclomatic complexity is minimal.
- Modern idioms: Yes. The (value, bool) return pattern mirrors Go's map lookup and type assertion conventions. Uses strings.Contains appropriately.
- Readability: Good. Helper has a clear doc comment explaining behavior and return semantics. Each caller is now 2-3 concise lines instead of 10 lines of loop code.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `findStepByName` helper (lines 256-265) was added beyond the task scope but is a reasonable extension following the same pattern. It is used by `TestReleaseWorkflowHomebrewDispatch` tests (lines 271, 290) and keeps the codebase consistent.
- The two helpers (`findStepByUses` and `findStepByName`) share identical structure differing only in which field they check. A generic approach could unify them, but for two simple functions this would be premature abstraction.
