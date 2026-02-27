---
topic: installation
cycle: 3
total_proposed: 1
---
# Analysis Tasks: Installation (Cycle 3)

## Task 1: Extract TMPDIR extraction helper in install_test.go
status: approved
severity: medium
sources: duplication

**Problem**: Two tests in `scripts/install_test.go` ("cleans up temp directory on failure" at lines 602-612 and "cleans up temp directory on success" at lines 643-653) contain an identical 8-line block that parses script output to find the `TICK_TMPDIR=` line, strips the prefix, and fatals if not found. This is verbatim duplication within the same file.

**Solution**: Extract a local helper function `extractTmpDir(t *testing.T, output string) string` within `install_test.go`. The helper calls `t.Helper()`, splits output by newline, searches for the `TICK_TMPDIR=` prefix, returns the path, and calls `t.Fatal` if not found.

**Outcome**: Each call site reduces from 8 lines to 1 line. The TMPDIR extraction logic has a single definition, making it easier to modify if the output format changes.

**Do**:
1. In `scripts/install_test.go`, add a function:
   ```go
   func extractTmpDir(t *testing.T, output string) string {
       t.Helper()
       for _, line := range strings.Split(output, "\n") {
           if strings.HasPrefix(line, "TICK_TMPDIR=") {
               return strings.TrimPrefix(line, "TICK_TMPDIR=")
           }
       }
       t.Fatal("could not find TICK_TMPDIR in output")
       return ""
   }
   ```
2. Replace the 8-line block at lines 602-612 with: `tmpDirPath := extractTmpDir(t, out)`
3. Replace the 8-line block at lines 643-653 with: `tmpDirPath := extractTmpDir(t, out)`

**Acceptance Criteria**:
- No duplicated TMPDIR extraction logic remains in `install_test.go`
- The `extractTmpDir` helper is used in both temp directory cleanup tests
- Helper calls `t.Helper()` for correct test failure line reporting

**Tests**:
- All existing tests in `scripts/install_test.go` pass unchanged (`go test ./scripts/...`)
- No new tests needed -- this is a refactor of test code, validated by existing test execution
