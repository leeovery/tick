TASK: GitHub Actions Release Workflow

ACCEPTANCE CRITERIA:
- `.github/workflows/release.yml` exists and is valid YAML
- Workflow triggers only on tag pushes matching the semver pattern `v[0-9]+.[0-9]+.[0-9]+`
- No branch push triggers or pull request triggers are configured
- Workflow has `permissions: contents: write`
- Checkout step uses `fetch-depth: 0` for full git history
- Go is set up using `go-version-file: 'go.mod'`
- goreleaser is invoked with `release --clean` and `GITHUB_TOKEN` is passed as an environment variable
- Workflow runs on `ubuntu-latest`

STATUS: Complete

SPEC CONTEXT: The specification requires GitHub Releases with pre-built binaries for four platforms (darwin-amd64, darwin-arm64, linux-amd64, linux-arm64). goreleaser produces archives named `tick_X.Y.Z_{os}_{arch}.tar.gz`. The install script and Homebrew formula both depend on release assets being available at predictable GitHub Release URLs. The version is derived from the git tag.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/.github/workflows/release.yml` (lines 1-58)
- Notes:
  - The workflow file exists and is valid YAML.
  - The trigger uses two patterns: `v[0-9]*.[0-9]*.[0-9]*` (positive) and `!v*-*` (negative/exclusion). This differs from the plan's suggested `v[0-9]+.[0-9]+.[0-9]+`, but the deviation is correct. In GitHub Actions glob syntax, `+` is a literal character (not a regex quantifier), so the plan's pattern would literally match `v1+.2+.3+` -- clearly wrong. The implementation's approach of `[0-9]*` (digit followed by anything) plus `!v*-*` (exclude pre-release suffixes) is the correct idiom for GitHub Actions.
  - No branch push triggers or pull_request triggers configured.
  - `permissions: contents: write` is set (line 9-10).
  - Checkout uses `fetch-depth: '0'` (line 19). Note: value is a string `'0'` rather than integer `0`, but YAML parsers for GitHub Actions treat both identically -- this is fine.
  - Go setup uses `go-version-file: go.mod` (line 24).
  - goreleaser is invoked with `args: release --clean` and `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` (lines 27-31).
  - Job runs on `ubuntu-latest` (line 14).
  - Additional steps beyond the plan scope: "Extract checksums" (lines 33-42) and "Dispatch to homebrew-tools" (lines 44-58). These were added by later tasks (Phase 2 Homebrew integration). This is not scope creep for this task -- they are separate, additive steps that do not interfere with the core release workflow.

TESTS:
- Status: Adequate
- Coverage:
  - All 11 tests from the plan's test list are implemented in `/Users/leeovery/Code/tick/scripts/release_test.go`.
  - Tag matching: v1.0.0 (match), v0.1.0 (match), v12.34.56 (match), v1.0.0-beta (no match), v1.0.0-rc.1 (no match), latest (no match), 1.0.0 without v prefix (no match).
  - Branch/PR triggers: verified no branch triggers, verified no pull_request trigger.
  - Structural checks: fetch-depth 0, GITHUB_TOKEN, permissions write, go-version-file go.mod, ubuntu-latest, release --clean args.
  - The `matchesGitHubActionsPattern` function (lines 71-152) faithfully implements GitHub Actions' glob matching semantics including `*`, `**`, `?`, and character classes `[0-9]`. The `assertTagMatches` function (lines 319-348) correctly handles both positive and negative (!) patterns, matching GitHub Actions' actual evaluation order.
  - Additional tests for Homebrew dispatch steps (TestReleaseWorkflowHomebrewDispatch, lines 267-313) cover the steps added by later tasks -- appropriate and not over-testing for this task.
- Notes:
  - Minor gap: No test for build metadata exclusion (`v1.0.0+build.123`). The plan's edge cases mention this should not match. The current patterns would actually match it (`[0-9]*` matches `0+build`, and `!v*-*` does not exclude it since there is no `-`). In practice, build metadata semver tags are extremely rare in Go projects, so this is a low-risk gap. Non-blocking.
  - Tests use `t.Helper()` correctly on helper functions and follow project conventions (stdlib testing only, subtests via `t.Run()`).

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on helpers. YAML parsing uses a typed struct rather than raw map access, which is clean and maintainable.
- SOLID principles: Good. The `workflow`/`job`/`step` struct types have clear single responsibility. Helper functions `findStepByUses` and `findStepByName` are focused. The glob matcher is self-contained.
- Complexity: Acceptable. The `matchPattern` function (lines 75-152) has moderate cyclomatic complexity due to the recursive glob matching, but this is inherent to the problem domain and the code is well-structured with clear cases.
- Modern idioms: Yes. Clean Go, appropriate use of `strings.Contains`, no unnecessary allocations.
- Readability: Good. Struct field tags make YAML mapping explicit. Helper functions have doc comments. The glob matcher has inline comments explaining each branch. The `assertTagMatches` function documents the positive/negative pattern semantics.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The tag pattern `v[0-9]*.[0-9]*.[0-9]*` with `!v*-*` does not exclude build metadata tags like `v1.0.0+build.123`. The plan's edge cases mention this should be excluded. Adding `!v*+*` as an additional negative pattern would close this gap. Very low practical risk.
- Consider adding a test case for `v1.0.0+build.123` to document the expected behavior for build metadata tags, even if the decision is to accept the current behavior.
