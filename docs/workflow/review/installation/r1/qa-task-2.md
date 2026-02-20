TASK: goreleaser Configuration

ACCEPTANCE CRITERIA:
- [x] `.goreleaser.yaml` exists at the repository root
- [x] Configuration targets exactly four platform combinations: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- [x] Build points to `./cmd/tick/` as the main package with binary name `tick`
- [x] Archive `name_template` is explicitly set to `tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
- [x] Archive format is explicitly set to `tar.gz` (not relying on defaults)
- [x] `goreleaser check` passes without errors (manual verification step)
- [x] Snapshot release produces four `.tar.gz` archives matching the naming pattern (manual verification step)
- [x] Each archive contains only the `tick` binary (no extra files) (`files: [none*]` config)
- [x] `dist/` is listed in `.gitignore`

STATUS: Complete

SPEC CONTEXT: The specification defines release asset naming as `tick_X.Y.Z_{os}_{arch}.tar.gz` for four platforms (darwin-amd64, darwin-arm64, linux-amd64, linux-arm64). Each archive contains only the `tick` binary. This naming is load-bearing infrastructure -- the install script constructs download URLs from it and the Homebrew formula references it. The goreleaser convention is `{binary}_{version}_{os}_{arch}.tar.gz`.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/.goreleaser.yaml (33 lines)
- Notes:
  - `version: 2` at the top correctly targets goreleaser v2 API
  - `project_name: tick` set explicitly
  - Single build entry with `id: tick`, `main: ./cmd/tick/`, `binary: tick`
  - `env: [CGO_ENABLED=0]` for static binaries as specified
  - `goos: [darwin, linux]`, `goarch: [amd64, arm64]` -- exactly 4 platforms
  - `ldflags: [-s -w]` for stripped binaries as planned
  - Archive uses `formats: [tar.gz]` (v2 syntax, equivalent to the planned `format: tar.gz`)
  - `name_template: "tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}"` matches spec exactly
  - `files: [none*]` ensures archives contain only the binary
  - `changelog: disable: true` and `release: draft: false` as planned
  - `dist/` is listed in `/Users/leeovery/Code/tick/.gitignore` line 4
  - The `name_template` hardcodes `tick_` rather than using `{{ .ProjectName }}_` -- this is correct per the edge case requirement to avoid reliance on goreleaser defaults

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/scripts/naming_contract_test.go` (`TestAssetNamingContract`): Parses `.goreleaser.yaml`, substitutes template variables, and asserts output matches the spec naming convention for all 4 platform combinations. Also cross-checks that the install script produces identical filenames -- this is an excellent contract test.
  - The test covers darwin/arm64, darwin/amd64, linux/amd64, linux/arm64 with different version strings (1.2.3, 0.5.0, 10.20.30) to verify template expansion works correctly.
  - Tests 1-4 from the task (goreleaser check, snapshot build, snapshot release, archive contents) are manual verification steps requiring goreleaser to be installed. These are appropriately not automated as Go unit tests. The configuration is simple enough that YAML parsing + template verification provides sufficient automated coverage.
  - The `extractGoreleaserFilename` helper also validates that the `formats` field is present and non-empty, and that the `archives` section exists -- providing structural validation of the YAML.
- Notes: Test balance is good. The contract test is focused, verifies the critical requirement (naming convention alignment), and would fail if either the goreleaser config or install script changed their naming pattern. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on helper functions, `t.TempDir()` for isolation where applicable.
- SOLID principles: Good. The goreleaser YAML is a declarative configuration file with a single clear purpose. The contract test has good separation of concerns (separate helper functions for goreleaser vs install script filename extraction).
- Complexity: Low. The `.goreleaser.yaml` is 33 lines of straightforward declarative configuration.
- Modern idioms: Yes. Uses goreleaser v2 format (`version: 2`, `formats` instead of `format`). Test uses table-driven pattern per project convention.
- Readability: Good. The YAML is well-structured with clear sections. Test helper functions are well-named and documented.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The task plan mentioned `release: draft: false` to "ensure" the default. While included and correct, goreleaser's default is already `draft: false`. This is a harmless explicit declaration that adds clarity.
