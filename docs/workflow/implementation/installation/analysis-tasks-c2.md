---
topic: installation
cycle: 2
total_proposed: 2
---
# Analysis Tasks: Installation (Cycle 2)

## Task 1: Move release workflow tests to a go-test-discoverable location
status: pending
severity: high
sources: architecture

**Problem**: `release_test.go` lives in `.github/workflows/` which Go's `./...` pattern skips (dot-prefixed directories are excluded by `go list ./...`). The 14 test cases validating the release workflow — tag patterns, permissions, goreleaser config, checkout depth, and an 80-line glob matcher — are never executed by `go test ./...`. They silently rot unless someone explicitly runs `go test ./.github/workflows/`.

**Solution**: Move `release_test.go` from `.github/workflows/` into the `scripts/` package (which already holds install_test.go and is discoverable by `go test ./...`). The test only needs `os.ReadFile` access to `.github/workflows/release.yml`, which it already resolves via `testutil.FindRepoRoot` — no location dependency.

**Outcome**: `go test ./...` discovers and runs all release workflow tests alongside other script/config tests. No silent rot.

**Do**:
1. Move `.github/workflows/release_test.go` to `scripts/release_test.go`
2. Change the package declaration from `package workflows_test` to `package scripts_test`
3. Update the YAML file path resolution: replace any hardcoded `.github/workflows/` relative paths with `filepath.Join(testutil.FindRepoRoot(t), ".github", "workflows", "release.yml")`
4. Remove any helper types or functions from `.github/workflows/` that were only used by the test (ensure no orphaned files remain)
5. Run `go test ./scripts/...` to verify all moved tests pass
6. Run `go test ./...` and confirm the release workflow tests appear in the output
7. Verify `.github/workflows/` contains only YAML workflow files, no Go files

**Acceptance Criteria**:
- No `.go` files exist in `.github/workflows/`
- `go list ./...` includes the package containing release workflow tests
- All 14 release workflow test cases pass when run via `go test ./...`
- The test still reads and validates `.github/workflows/release.yml` correctly

**Tests**:
- All existing release workflow tests pass in their new location
- `go test ./...` output includes release workflow test cases (verify with `-v` flag)

## Task 2: Extract loadScript helper in install_test.go
status: pending
severity: medium
sources: duplication

**Problem**: Eight tests in `scripts/install_test.go` independently read install.sh contents using the same 4-line sequence: `path := scriptPath(t); data, err := os.ReadFile(path); if err != nil { t.Fatalf(...) }; content := string(data)`. This appears in TestInstallScript (shebang, set -euo checks), TestTrapCleanup, and TestErrorHandlingHardening. The pattern violates the Rule of Three at 8 occurrences.

**Solution**: Extract a `loadScript(t *testing.T) string` helper in `install_test.go` (following the same pattern as `loadFormula` and `loadWorkflow` in their respective test files). All 8 call sites reduce to `content := loadScript(t)`.

**Outcome**: Single function for loading install.sh contents in tests. Adding new content-inspection tests requires one line instead of four.

**Do**:
1. Add a `loadScript` function in `scripts/install_test.go`:
   ```go
   func loadScript(t *testing.T) string {
       t.Helper()
       data, err := os.ReadFile(scriptPath(t))
       if err != nil {
           t.Fatalf("cannot read install.sh: %v", err)
       }
       return string(data)
   }
   ```
2. Replace all 8 occurrences of the read-file-to-string pattern with `content := loadScript(t)`
3. Run `go test ./scripts/...` to verify no regressions

**Acceptance Criteria**:
- A single `loadScript` helper exists in install_test.go
- No inline os.ReadFile calls for install.sh remain in install_test.go
- All existing tests pass unchanged

**Tests**:
- All existing tests in scripts/install_test.go pass after the refactor
