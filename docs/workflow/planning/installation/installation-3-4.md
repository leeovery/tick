---
id: installation-3-4
phase: 3
status: completed
created: 2026-02-14
---

# Add cross-component asset naming contract test

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
