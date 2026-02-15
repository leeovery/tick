TASK: Move release workflow tests to a go-test-discoverable location

ACCEPTANCE CRITERIA:
- No `.go` files exist in `.github/workflows/`
- `go list ./...` includes the package containing release workflow tests
- All 14 release workflow test cases pass when run via `go test ./...`
- The test still reads and validates `.github/workflows/release.yml` correctly

STATUS: Complete

SPEC CONTEXT: The release workflow tests validate the GitHub Actions release pipeline configuration (tag patterns, permissions, goreleaser config, checkout depth, Homebrew dispatch). These tests ensure the distribution infrastructure described in the installation specification remains correct. The tests were originally in `.github/workflows/release_test.go` which `go test ./...` skips because Go excludes dot-prefixed directories.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/scripts/release_test.go` (full file, 349 lines)
- Notes:
  - Package declaration correctly changed to `package scripts_test` (line 1)
  - YAML path resolution uses `filepath.Join(testutil.FindRepoRoot(t), ".github", "workflows", "release.yml")` via the `loadWorkflow` helper (lines 48-49) -- location-independent
  - No `.go` files remain in `.github/workflows/` (only `release.yml` remains)
  - No orphaned helper types or functions -- all types (`workflow`, `job`, `step`) and helpers (`loadWorkflow`, `matchesGitHubActionsPattern`, `matchPattern`, `findStepByUses`, `findStepByName`, `assertTagMatches`) are defined and used within the test file
  - No references to old `workflows_test` package in source code (only in planning docs)
  - 17 test cases total across `TestReleaseWorkflow` (15 subtests) and `TestReleaseWorkflowHomebrewDispatch` (2 subtests) -- exceeds the 14 mentioned in the task description, which is expected since additional Homebrew dispatch tests were added in later phases

TESTS:
- Status: Adequate
- Coverage: All existing release workflow tests are present in the new location. The test file validates tag patterns (valid semver, pre-release rejection, non-version rejection, no-v-prefix rejection), workflow triggers (no branch push, no pull request), permissions (contents write), checkout configuration (fetch-depth 0), Go setup (go-version-file), goreleaser configuration (GITHUB_TOKEN, release --clean args), runner (ubuntu-latest), and Homebrew dispatch (checksum extraction, repository_dispatch payload).
- Notes: The tests read the actual `release.yml` file and parse it, so they would fail immediately if the workflow file were modified incorrectly. The `matchesGitHubActionsPattern` function implements GitHub Actions glob semantics for accurate tag pattern validation.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing` only (no testify), `t.Run()` subtests, `t.Helper()` on all helpers. Error wrapping follows project patterns.
- SOLID principles: Good. `loadWorkflow` has single responsibility (parse YAML). `findStepByUses`/`findStepByName` are focused search helpers. `assertTagMatches` encapsulates tag pattern matching logic with proper include/exclude semantics.
- Complexity: Acceptable. The `matchPattern` function (lines 75-152) is the most complex piece with recursive backtracking for `*` and `**` patterns, but this is inherent to the glob matching problem and is well-structured with clear cases.
- Modern idioms: Yes. Uses `filepath.Join` for path construction, proper Go test patterns.
- Readability: Good. Functions are well-named and documented with comments. The test names are descriptive and follow the "what it tests" pattern. Helper types have doc comments explaining their purpose.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `findStepByName` helper (lines 256-265) mirrors `findStepByUses` closely. A generic `findStep` with a predicate function could DRY these up, but the current duplication is minimal (10 lines each) and both are clear -- not worth abstracting.
