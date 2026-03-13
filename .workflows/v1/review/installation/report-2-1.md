TASK: Homebrew Tap Repository and Formula

ACCEPTANCE CRITERIA:
- homebrew-tap/Formula/tick.rb exists with a valid Homebrew formula
- Formula class is named Tick and inherits from Formula
- Formula serves darwin_arm64 archive for Apple Silicon Macs (Hardware::CPU.arm?)
- Formula serves darwin_amd64 archive for Intel Macs (Hardware::CPU.intel?)
- Download URLs follow the pattern https://github.com/leeovery/tick/releases/download/v{version}/tick_{version}_darwin_{arch}.tar.gz (tag has v, filename does not)
- Each architecture branch has its own sha256 declaration
- def install places the tick binary into Homebrew's bin directory via bin.install "tick"
- Formula includes a test do block that verifies the binary runs
- homebrew-tap/README.md exists with tap and install instructions
- The version field uses the bare version number without a v prefix (e.g., 1.2.3 not v1.2.3)

STATUS: Issues Found

SPEC CONTEXT: The specification designates Homebrew as the preferred macOS installation method. Users should be able to run `brew tap {owner}/tick && brew install tick` (later migrated to `brew install leeovery/tools/tick`). Homebrew handles code signing automatically, which is why the macOS install script delegates to it rather than downloading binaries directly. Release assets follow the naming convention `tick_X.Y.Z_{os}_{arch}.tar.gz`.

IMPLEMENTATION:
- Status: Drifted (removed from this repo, migrated externally)
- Location: Formula no longer exists in this repository. The `homebrew-tap/` directory that originally contained `Formula/tick.rb` and `README.md` was removed during the migration to `homebrew-tools` (commit c4a2a84 "Migrate Homebrew TAP from homebrew-tick to homebrew-tools").
- Notes:
  - The plan originally specified `homebrew-tap/Formula/tick.rb` would live in this repo for local validation, with the understanding it would later be published to a separate GitHub repository.
  - The project has since evolved beyond the plan: instead of a separate `homebrew-tick` repo, the formula now lives in a shared `leeovery/homebrew-tools` repository (a multi-tool tap).
  - The release workflow at `.github/workflows/release.yml` (lines 33-58) properly integrates with this external formula via `repository_dispatch`: it extracts SHA256 checksums for both darwin architectures from goreleaser output and dispatches an `update-formula` event to `leeovery/homebrew-tools` with the version, sha256_arm64, and sha256_amd64 payload.
  - The install script at `/Users/leeovery/Code/tick/scripts/install.sh` (line 66) correctly references `leeovery/tools/tick` (the `homebrew-tools` tap path), not `leeovery/tick` (the old `homebrew-tick` path). This is consistent with the migration.
  - The actual formula content cannot be verified from this repository since it lives externally. The acceptance criteria that reference `homebrew-tap/Formula/tick.rb` existing in this repo are no longer met by design -- the formula was intentionally moved to an external repository.

  Assessment of acceptance criteria against current state:
  1. "homebrew-tap/Formula/tick.rb exists" -- NOT MET in this repo. Formula lives externally in homebrew-tools.
  2. "Formula class is named Tick and inherits from Formula" -- CANNOT VERIFY from this repo.
  3. "Formula serves darwin_arm64" -- PARTIALLY VERIFIED: release workflow extracts and dispatches sha256_arm64 checksum, confirming the external formula is expected to use it.
  4. "Formula serves darwin_amd64" -- PARTIALLY VERIFIED: release workflow extracts and dispatches sha256_amd64 checksum.
  5. "Download URLs follow v{version}/tick_{version} pattern" -- PARTIALLY VERIFIED: release workflow computes VERSION="${GITHUB_REF_NAME#v}" (strips v prefix) and constructs checksums using `tick_${VERSION}_darwin_{arch}.tar.gz`, which is consistent with the expected URL pattern.
  6. "Each architecture branch has its own sha256" -- PARTIALLY VERIFIED: two separate sha256 values are dispatched.
  7. "bin.install tick" -- CANNOT VERIFY from this repo.
  8. "test do block exists" -- CANNOT VERIFY from this repo.
  9. "homebrew-tap/README.md exists" -- NOT MET in this repo. Was intentionally removed during migration.
  10. "version field uses bare version without v prefix" -- PARTIALLY VERIFIED: VERSION="${GITHUB_REF_NAME#v}" in the release workflow strips the v prefix before dispatching.

TESTS:
- Status: Under-tested (formula-specific tests removed, no replacement)
- Coverage:
  - The plan specified 10 test cases for formula validation (class name, URL patterns, architecture handling, sha256 declarations, install method, test block, README content). None of these tests exist in the current codebase.
  - The analysis tasks (installation-3-1) referenced a `homebrew-tap/formula_test.go` that previously existed but has been removed along with the formula.
  - The naming contract test at `/Users/leeovery/Code/tick/scripts/naming_contract_test.go` originally planned to also verify the Homebrew formula URL pattern (task installation-3-4 step 1c), but the current implementation only validates goreleaser and install script naming -- it does NOT include the Homebrew formula as a third source. This is a gap from the original plan, though understandable since the formula is no longer in this repo.
  - The release workflow dispatch tests at `/Users/leeovery/Code/tick/scripts/release_test.go` (lines 267-313, `TestReleaseWorkflowHomebrewDispatch`) verify:
    - Checksum extraction step exists with correct ID, references checksums.txt, extracts darwin_arm64 and darwin_amd64 hashes
    - Dispatch step targets homebrew-tools repo, uses update-formula event type, includes tool=tick, sha256_arm64, sha256_amd64, and CICD_PAT
  - These dispatch tests provide good coverage of the integration point, but they do not verify the formula content itself.
- Notes:
  - The missing formula tests are a natural consequence of the migration -- you cannot test a file that no longer exists in the repo. However, there is no replacement mechanism to verify the external formula's correctness from this repo.
  - The install script tests at `/Users/leeovery/Code/tick/scripts/install_test.go` (TestMacOSInstall, lines 644-715) thoroughly test the `brew install leeovery/tools/tick` delegation, which validates the end-user integration path works.

CODE QUALITY:
- Project conventions: Followed. The release workflow YAML is clean and well-structured. The dispatch payload is minimal and correct.
- SOLID principles: Good. The release workflow has clear separation of concerns: build (goreleaser), checksum extraction, and dispatch to external formula repo.
- Complexity: Low. The checksum extraction step is a straightforward shell pipeline.
- Modern idioms: Yes. Uses GitHub Actions outputs, repository_dispatch pattern is standard.
- Readability: Good. Workflow steps are clearly named and sequential.
- Issues:
  - The naming contract test (`/Users/leeovery/Code/tick/scripts/naming_contract_test.go`) was originally designed to verify three sources (goreleaser, install script, Homebrew formula) but now only verifies two. The test still passes and is valuable, but the Homebrew formula leg is silently absent rather than explicitly documented as externally managed.

BLOCKING ISSUES:
- The `homebrew-tap/Formula/tick.rb` file and `homebrew-tap/README.md` do not exist in this repository. The formula was migrated to an external `homebrew-tools` repository. This means 2 of 10 acceptance criteria are structurally unmet (file existence) and 4 more cannot be verified from this repo (formula content details). This is a deliberate architectural decision (the migration), not a bug, but it means the task as originally scoped is no longer fully verifiable within this repository. Whether this is truly "blocking" depends on whether the external formula is considered in scope.

NON-BLOCKING NOTES:
- The naming contract test at `/Users/leeovery/Code/tick/scripts/naming_contract_test.go` should ideally document why the Homebrew formula leg was removed (a comment noting the formula moved to homebrew-tools and is verified there, or is no longer part of this repo's contract test scope).
- Consider adding a test that verifies the dispatch payload structure matches what the homebrew-tools repository expects (e.g., validate the JSON keys in the dispatch payload are correct).
- The install script's error message (line 62) says `brew install leeovery/tools/tick` but the original spec says `brew tap {owner}/tick && brew install tick`. The one-liner `brew install leeovery/tools/tick` is actually the more modern and simpler Homebrew idiom (it auto-taps), so this is an improvement over the spec, not a regression.
