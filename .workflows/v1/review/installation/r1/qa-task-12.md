TASK: Add cross-component asset naming contract test

ACCEPTANCE CRITERIA:
- A test exists that validates the asset naming convention is consistent across goreleaser, install script, and Homebrew formula
- The test reads from the actual source files (not hardcoded expectations)
- The test fails clearly if any source's naming pattern diverges

STATUS: Issues Found

SPEC CONTEXT: The specification defines the release asset naming convention as `tick_X.Y.Z_{os}_{arch}.tar.gz` (spec section "Release Asset Naming"). Three components must agree on this convention: goreleaser (produces assets), install script (downloads assets), and Homebrew formula (references assets). The analysis finding (analysis-architecture-c1.md) identified this as a medium-severity cross-component seam risk.

IMPLEMENTATION:
- Status: Partial
- Location: /Users/leeovery/Code/tick/scripts/naming_contract_test.go:1-136
- Notes: The test validates two of the three sources specified in the acceptance criteria. It checks goreleaser (.goreleaser.yaml name_template + formats) and install script (scripts/install.sh construct_url pattern). The Homebrew formula is NOT validated. The task explicitly requires "consistent across goreleaser, install script, and Homebrew formula" and lists three extraction steps (goreleaser, install script, Homebrew formula). The Homebrew formula now lives in a separate repository (leeovery/homebrew-tools) after a migration from the in-repo homebrew-tap/ directory, so it cannot be checked locally. This is a reasonable constraint but represents a deviation from the acceptance criteria. The release workflow does dispatch asset naming information (version, checksums) to homebrew-tools, but the URL pattern in the Homebrew formula is not contract-tested.

TESTS:
- Status: Adequate (for what is covered)
- Coverage: Table-driven test (TestAssetNamingContract) with 4 subtests covering all platform combinations: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64. Each subtest validates goreleaser output, install script output, and cross-checks them against each other and against an expected value. The extractGoreleaserFilename helper properly parses YAML and substitutes goreleaser template variables. The extractInstallScriptFilename helper uses regex to parse the construct_url function from the actual bash script.
- Notes: Three-way assertion per subtest (goreleaser vs want, install script vs want, goreleaser vs install script) is thorough. The "want" value acts as a specification anchor. Tests use both spaced and unspaced template variable forms for goreleaser (lines 52-57), which adds resilience.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run() subtests, t.Helper() on helpers, testutil.FindRepoRoot shared utility, table-driven test pattern.
- SOLID principles: Good. extractGoreleaserFilename and extractInstallScriptFilename have single responsibilities. The test function cleanly separates extraction from assertion.
- Complexity: Low. Each extraction function is straightforward: read file, parse, substitute, return.
- Modern idioms: Yes. Proper use of yaml.v3, regexp, t.Helper(), table-driven tests.
- Readability: Good. Clear function names, well-commented intent (lines 49, 85), struct type is minimal and focused.
- Issues: None significant.

BLOCKING ISSUES:
- The acceptance criteria explicitly require validating three sources: goreleaser, install script, AND Homebrew formula. Only two are validated. The Homebrew formula lives in a separate repository (leeovery/homebrew-tools) and is not contract-tested. While this is a practical limitation (external repo), it means the stated acceptance criterion "consistent across goreleaser, install script, and Homebrew formula" is not fully met. The original risk (silent 404 on naming drift) still exists for the Homebrew formula side. Options to resolve: (a) accept this as a known limitation and update the acceptance criteria to reflect two-source validation only, (b) add a note/comment in the test file documenting that Homebrew formula validation requires the external repo, or (c) add a CI step in homebrew-tools that validates naming against this repo's convention.

NON-BLOCKING NOTES:
- The goreleaserConfig struct (line 15) and yaml dependency are also used in release_test.go (which defines its own workflow struct). Both files import yaml.v3. There is no duplication issue since they parse different YAML structures.
- The regex on line 77 for extracting the construct_url pattern is tightly coupled to the current script structure. If construct_url's echo statement format changes (e.g., uses printf instead), the regex will silently fail to match. This is acceptable for a contract test but worth noting.
- Consider adding a comment at the top of the test file explaining that the Homebrew formula (third component in the original analysis finding) is not validated here because it lives in an external repository.
