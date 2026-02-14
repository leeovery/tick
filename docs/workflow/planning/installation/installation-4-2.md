---
id: installation-4-2
phase: 4
status: completed
created: 2026-02-14
---

# Extract loadScript helper in install_test.go

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
