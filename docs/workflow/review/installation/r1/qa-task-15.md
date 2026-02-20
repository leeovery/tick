TASK: Extract TMPDIR extraction helper in install_test.go

ACCEPTANCE CRITERIA:
- No duplicated TMPDIR extraction logic remains in install_test.go
- The extractTmpDir helper is used in both temp directory cleanup tests
- Helper calls t.Helper() for correct test failure line reporting

STATUS: Complete

SPEC CONTEXT: This is a test-code refactoring task from analysis cycle 3. The install script tests include two tests ("cleans up temp directory on failure" and "cleans up temp directory on success") that previously contained identical 8-line blocks to parse TICK_TMPDIR from script output. The task extracts this into a shared helper.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/scripts/install_test.go:44-55
- Notes: The `extractTmpDir` helper function is implemented exactly as specified in the plan. It calls `t.Helper()`, splits output by newline, searches for the `TICK_TMPDIR=` prefix, returns the path, and calls `t.Fatal` if not found. The implementation matches the plan's code snippet precisely.

Call sites:
- /Users/leeovery/Code/tick/scripts/install_test.go:604 -- `tmpDirPath := extractTmpDir(t, out)` in "cleans up temp directory on failure"
- /Users/leeovery/Code/tick/scripts/install_test.go:635 -- `tmpDirPath := extractTmpDir(t, out)` in "cleans up temp directory on success"

No duplicated TMPDIR extraction logic remains. The only other reference to `TICK_TMPDIR` in the file (line 795) is in `TestMacOSNoBrewError` where it checks that `TICK_TMPDIR=` does NOT appear in output -- a different concern entirely.

TESTS:
- Status: Adequate
- Coverage: This is a refactor of test code itself. No new tests are needed. The existing tests ("cleans up temp directory on failure" and "cleans up temp directory on success") exercise the helper. If the helper broke, these tests would fail.
- Notes: Validated by existing test execution as specified in the task.

CODE QUALITY:
- Project conventions: Followed. Uses `t.Helper()` per CLAUDE.md conventions. Uses stdlib testing only (no testify). File-local helper function consistent with other helpers in the same file (e.g., `loadScript`, `scriptPath`, `runScript`).
- SOLID principles: Good. Single responsibility -- the helper does one thing (extract TMPDIR from output).
- Complexity: Low. Simple loop with prefix check. Cyclomatic complexity is minimal.
- Modern idioms: Yes. Uses `strings.Split`, `strings.HasPrefix`, `strings.TrimPrefix` idiomatically.
- Readability: Good. Function name is descriptive, godoc comment explains purpose, the unreachable `return ""` after `t.Fatal` satisfies the compiler cleanly.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
