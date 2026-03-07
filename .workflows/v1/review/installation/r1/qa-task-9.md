TASK: Extract shared findRepoRoot test utility

ACCEPTANCE CRITERIA:
- No findRepoRoot function or equivalent inlined logic exists outside internal/testutil
- All 4 test files import and use testutil.FindRepoRoot
- All existing tests pass without modification to test logic (only the utility source changes)

STATUS: Complete

SPEC CONTEXT: This is a refactoring task from analysis cycle 1, not directly tied to a spec requirement. The task addresses DRY violations where the findRepoRoot helper was duplicated across 4 test files. The CLAUDE.md project conventions explicitly list `internal/testutil/` as the home for shared test helpers and note `FindRepoRoot` by name.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/testutil/reporoot.go:12-28
- Notes: Clean implementation. The function accepts `*testing.T`, calls `t.Helper()`, walks up from cwd looking for `go.mod`, and fatals if the filesystem root is reached without finding it. Follows all project conventions: uses `t.Helper()`, uses `t.Fatalf` for unrecoverable errors, uses `filepath.Dir` for parent traversal.

Callers verified:
- /Users/leeovery/Code/tick/cmd/tick/build_test.go:15 -- `testutil.FindRepoRoot(t)`
- /Users/leeovery/Code/tick/scripts/install_test.go:17 -- via `scriptPath` helper calling `testutil.FindRepoRoot(t)`
- /Users/leeovery/Code/tick/scripts/release_test.go:48 -- `testutil.FindRepoRoot(t)` in `loadWorkflow`
- /Users/leeovery/Code/tick/scripts/naming_contract_test.go:105 -- `testutil.FindRepoRoot(t)`

Note: The original task mentions `homebrew-tap/formula_test.go` as the 3rd file, but this file no longer exists because the Homebrew tap was migrated to a separate `homebrew-tools` repository (per plan phase 2 and git history). This is expected -- the file was removed as part of the tap migration, so the extraction task correctly does not reference it. The 4th file (`.github/workflows/release_test.go`) was relocated to `scripts/release_test.go` by a later task (installation-4-1), which also uses the shared utility.

No remaining `func findRepoRoot` definitions or inlined go.mod-walking logic exist anywhere in the codebase outside `internal/testutil/reporoot.go`.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/testutil/reporoot_test.go has 2 subtests:
    1. Verifies FindRepoRoot returns a path containing go.mod (line 12-17)
    2. Verifies consistent results on repeated calls (line 20-25)
  - All 4 consumer test files continue to exercise FindRepoRoot transitively through their existing tests
- Notes: Tests are appropriately scoped. The "returns a path containing go.mod" test directly verifies the core contract. The consistency test ensures deterministic behavior. No over-testing -- only 2 focused tests for the shared utility. The task's micro acceptance criterion "FindRepoRoot returns a path containing go.mod when called from any subdirectory" is partially covered (tested from the testutil package subdirectory, not from multiple subdirectories), but since every consumer test file implicitly tests from its own package directory, the cross-directory behavior is verified in aggregate.

CODE QUALITY:
- Project conventions: Followed. Uses `t.Helper()`, `t.Fatalf` for fatal errors, stdlib `testing` only, `t.Run()` subtests. Package is `testutil` (exported), test file uses `testutil_test` (black-box testing). Package comment on line 2 of reporoot.go.
- SOLID principles: Good. Single responsibility -- one function, one file, one purpose. No unnecessary abstractions.
- Complexity: Low. Simple loop with one branch condition and a termination check.
- Modern idioms: Yes. Idiomatic Go error handling, filepath operations, and test patterns.
- Readability: Good. Clear function name, doc comment explains behavior, implementation is straightforward to follow.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The task description references `homebrew-tap/formula_test.go` which no longer exists due to the tap migration. This is not an implementation issue but a minor drift between the task description and current repo state. The task's intent (single source of truth for repo root discovery) is fully achieved.
