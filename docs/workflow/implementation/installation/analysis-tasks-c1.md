---
topic: installation
cycle: 1
total_proposed: 4
---
# Analysis Tasks: Installation (Cycle 1)

## Task 1: Extract shared findRepoRoot test utility
status: pending
severity: high
sources: duplication

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

## Task 2: Extract step-search helper in release_test.go
status: pending
severity: medium
sources: duplication

**Problem**: Four test cases in .github/workflows/release_test.go (lines 210-285) repeat the same nested loop pattern (iterate jobs, iterate steps, check strings.Contains on Uses field) with minor field variations. Each block is ~10 lines of near-identical traversal code.

**Solution**: Extract a local findStepByUses helper within release_test.go that encapsulates the nested loop search pattern.

**Outcome**: Each test case calls the helper in 2-3 lines instead of repeating 10 lines of loop code. Future step-search tests follow the same concise pattern.

**Do**:
1. Add a `findStepByUses(w workflow, usesSubstring string) (step, bool)` helper function in release_test.go (unexported, file-local)
2. Replace each of the 4 nested loop blocks with a call to findStepByUses
3. Each test case checks the returned step's specific fields (With, Env, etc.)
4. Run release_test.go tests to verify no regressions

**Acceptance Criteria**:
- A single findStepByUses helper exists in release_test.go
- The 4 test cases that previously had inline loops now call the helper
- No change to what is being asserted, only how the step is located

**Tests**:
- All existing tests in .github/workflows/release_test.go pass unchanged

## Task 3: Document Homebrew tap repository requirement
status: pending
severity: medium
sources: standards

**Problem**: The spec says `brew tap {owner}/tick` which requires a GitHub repo at `github.com/leeovery/homebrew-tick`. The current formula lives inside the main tick repo under `homebrew-tap/`. The `brew tap leeovery/tick` command looks for `github.com/leeovery/homebrew-tick`, not `github.com/leeovery/tick/homebrew-tap/`, so tap discovery will fail at runtime. The install script (line 66) also runs `brew tap leeovery/tick`.

**Solution**: Add clear documentation in homebrew-tap/README.md explaining that the formula files must be copied or synced to a separate `homebrew-tick` repository for `brew tap` to work. This documents the deployment prerequisite without changing the development structure.

**Outcome**: Any developer or CI process knows that the formula directory contents must be published to a separate homebrew-tick repo. The gap between development layout and runtime expectation is explicitly documented.

**Do**:
1. Create or update `homebrew-tap/README.md` to explain that the formula lives here for development but must be synced to a separate `github.com/leeovery/homebrew-tick` repository for `brew tap leeovery/tick` to work
2. Document that the release workflow or a manual step is responsible for this sync
3. Note that `scripts/install.sh` line 66 depends on this external repo existing

**Acceptance Criteria**:
- homebrew-tap/README.md clearly states the formula must be published to a separate homebrew-tick repository
- The sync requirement is documented as a deployment prerequisite

**Tests**:
- No automated tests required; this is a documentation task

## Task 4: Add cross-component asset naming contract test
status: pending
severity: medium
sources: architecture

**Problem**: Three independent sources define the release asset filename convention: goreleaser name_template (.goreleaser.yaml:24), install script construct_url (scripts/install.sh:56), and Homebrew formula URL (homebrew-tap/Formula/tick.rb:9). These are independently maintained with no test asserting they produce identical filenames. If any side drifts (e.g., goreleaser adds a "v" prefix), installs silently break with 404 errors.

**Solution**: Add a single integration test that parses the goreleaser name_template, the install script's URL construction pattern, and the Homebrew formula URL pattern, then verifies all three produce the same filename for a given version/os/arch tuple.

**Outcome**: Any drift in asset naming convention between the three sources is caught at test time, preventing silent 404 failures in production installs.

**Do**:
1. Create a test file (e.g., `scripts/naming_contract_test.go` or `internal/integration/naming_test.go`) that:
   a. Reads `.goreleaser.yaml` and extracts the name_template
   b. Reads `scripts/install.sh` and extracts the URL construction pattern from construct_url
   c. Reads `homebrew-tap/Formula/tick.rb` and extracts the URL pattern
2. For a sample version/os/arch tuple (e.g., "1.2.3", "darwin", "arm64"), verify all three produce `tick_1.2.3_darwin_arm64.tar.gz`
3. Test should fail with a clear message identifying which source diverged

**Acceptance Criteria**:
- A test exists that validates the asset naming convention is consistent across goreleaser, install script, and Homebrew formula
- The test reads from the actual source files (not hardcoded expectations)
- The test fails clearly if any source's naming pattern diverges

**Tests**:
- The contract test itself passes with current implementation
- Intentionally modifying the goreleaser name_template causes the test to fail (manual verification during development)
