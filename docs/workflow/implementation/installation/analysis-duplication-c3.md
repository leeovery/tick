AGENT: duplication
FINDINGS:
- FINDING: Duplicated TMPDIR extraction block in install_test.go
  SEVERITY: medium
  FILES: scripts/install_test.go:602-612, scripts/install_test.go:643-653
  DESCRIPTION: Two tests ("cleans up temp directory on failure" and "cleans up temp directory on success") contain an identical 8-line block that parses script output to extract the TICK_TMPDIR value. The block declares tmpDirPath, iterates output lines looking for the "TICK_TMPDIR=" prefix, strips the prefix, and fatals if not found. Both blocks are character-for-character identical. This is within a single file but produced by the same task executor repeating the pattern.
  RECOMMENDATION: Extract an extractTmpDir(t *testing.T, output string) string helper within install_test.go. Each call site reduces from 8 lines to 1. The helper calls t.Helper(), splits output by newline, searches for the prefix, and fatals if missing.
SUMMARY: One medium-severity finding: an 8-line TMPDIR extraction block is duplicated verbatim in two adjacent tests in install_test.go and should be extracted to a local helper.
