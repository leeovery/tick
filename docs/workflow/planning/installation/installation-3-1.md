---
id: installation-3-1
phase: 3
status: pending
created: 2026-02-14
---

# Extract shared findRepoRoot test utility

**Problem**: The findRepoRoot function (walks up from cwd to find go.mod) is copied verbatim in 3 test files and inlined as equivalent logic in a 4th. Files: cmd/tick/build_test.go:56-72, homebrew-tap/formula_test.go:11-27, scripts/install_test.go:13-29, .github/workflows/release_test.go:45-59. Any change to root-finding logic must be applied in 4 places.

**Solution**: Extract to a shared internal/testutil package exporting a FindRepoRoot function. Replace all 4 implementations with calls to the shared function.

**Outcome**: Single source of truth for repo root discovery in tests. Changes to root-finding logic require editing one file.

**Do**:
1. Create `internal/testutil/reporoot.go` with `func FindRepoRoot(t *testing.T) string` containing the walk-up-to-go.mod logic
2. In `cmd/tick/build_test.go`, remove the local findRepoRoot function and import `internal/testutil`; replace calls with `testutil.FindRepoRoot(t)`
3. In `homebrew-tap/formula_test.go`, remove the local findRepoRoot function and import `internal/testutil`; replace calls with `testutil.FindRepoRoot(t)`
4. In `scripts/install_test.go`, remove the local findRepoRoot function and import `internal/testutil`; replace calls with `testutil.FindRepoRoot(t)`
5. In `.github/workflows/release_test.go`, remove the inlined root-finding logic in loadWorkflow and replace with a call to `testutil.FindRepoRoot(t)`
6. Run all tests to verify no regressions

**Acceptance Criteria**:
- No findRepoRoot function or equivalent inlined logic exists outside internal/testutil
- All 4 test files import and use testutil.FindRepoRoot
- All existing tests pass without modification to test logic (only the utility source changes)

**Tests**:
- Existing tests in all 4 files continue to pass after extraction
- internal/testutil.FindRepoRoot returns a path containing go.mod when called from any subdirectory of the repo
