AGENT: duplication
FINDINGS:
- FINDING: findRepoRoot duplicated across 3 test files (plus 1 inline variant)
  SEVERITY: high
  FILES: cmd/tick/build_test.go:56-72, homebrew-tap/formula_test.go:11-27, scripts/install_test.go:13-29, .github/workflows/release_test.go:45-59
  DESCRIPTION: The findRepoRoot function (walks up from cwd to find go.mod) is copied verbatim in 3 test files and inlined as equivalent logic in a 4th. Each is 15-17 lines with identical signature, loop structure, error messages, and return semantics. This was clearly written independently by separate task executors. Any future change to root-finding logic (e.g. looking for a different marker file) would need to be applied in 4 places.
  RECOMMENDATION: Extract to a shared internal/testutil package (e.g. internal/testutil/reporoot.go) exporting a FindRepoRoot(t *testing.T) string function. All 4 test files import and call it. The release_test.go loadWorkflow function would call testutil.FindRepoRoot(t) instead of inlining the loop.

- FINDING: Repeated step-search iteration in release_test.go
  SEVERITY: medium
  FILES: .github/workflows/release_test.go:210-222, .github/workflows/release_test.go:226-238, .github/workflows/release_test.go:248-260, .github/workflows/release_test.go:272-285
  DESCRIPTION: Four test cases iterate the same nested loop (for jobs, for steps, check strings.Contains(s.Uses, X)) with minor field variations. Each block is ~10 lines doing the same traversal with different match criteria. This is within a single file so it is near-duplicate logic rather than cross-file, but it was likely produced by the same task executor repeating the pattern.
  RECOMMENDATION: Extract a findStepByUses(w workflow, usesSubstring string) (step, bool) helper within release_test.go. Each test calls the helper and checks the relevant field (With["fetch-depth"], Env["GITHUB_TOKEN"], etc.) in 2-3 lines instead of 10.

SUMMARY: One high-severity finding: findRepoRoot is identically implemented in 3 test files plus inlined in a 4th, a clear Rule of Three violation warranting extraction to a shared test utility. One medium-severity finding: repeated step-search loops within release_test.go that could be consolidated via a local helper.
