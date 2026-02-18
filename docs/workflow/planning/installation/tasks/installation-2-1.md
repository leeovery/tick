---
id: installation-2-1
phase: 2
status: completed
created: 2026-01-31
---

# Homebrew Tap Repository and Formula

## Goal

macOS is a primary audience for tick, and the specification designates Homebrew as the preferred macOS installation method. Homebrew handles code signing automatically, which is why the install script delegates to it rather than downloading binaries directly on macOS. This task creates the Homebrew tap repository structure and formula file so that `brew tap leeovery/tick && brew install tick` downloads and installs the correct pre-built binary from GitHub Releases. Without this, there is no Homebrew-based installation path, and the macOS install script behavior (tasks 2-2 and 2-3) has nothing to delegate to.

## Implementation

1. **Create the tap repository directory structure** within this repository at `homebrew-tap/` (this will later be published as the `homebrew-tick` repository at `github.com/leeovery/homebrew-tick`, but for now the formula is authored and validated locally):

   - `homebrew-tap/Formula/tick.rb` — the formula file.
   - `homebrew-tap/README.md` — brief instructions explaining this is a Homebrew tap for tick.

2. **Write the formula file** at `homebrew-tap/Formula/tick.rb`:

   ```ruby
   class Tick < Formula
     desc "CLI tool for tick"
     homepage "https://github.com/leeovery/tick"
     version "VERSION"  # Placeholder — updated per release
     license "MIT"      # Adjust to actual license

     on_macos do
       if Hardware::CPU.arm?
         url "https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_arm64.tar.gz"
         sha256 "SHA256_DARWIN_ARM64"  # Placeholder — updated per release
       elsif Hardware::CPU.intel?
         url "https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_amd64.tar.gz"
         sha256 "SHA256_DARWIN_AMD64"  # Placeholder — updated per release
       end
     end

     def install
       bin.install "tick"
     end

     test do
       assert_match "tick", shell_output("#{bin}/tick")
     end
   end
   ```

   Key details in the formula:
   - **URL construction**: The download tag uses `v#{version}` (e.g., `v1.2.3`) because GitHub release tags have the `v` prefix. The archive filename uses `#{version}` without the `v` prefix (e.g., `tick_1.2.3_darwin_arm64.tar.gz`) because goreleaser's `{{ .Version }}` strips the `v`. This is the critical edge case — the version in the URL path segment keeps the `v`, but the version in the filename does not.
   - **Architecture handling**: Uses `on_macos` block with `Hardware::CPU.arm?` for Apple Silicon (arm64) and `Hardware::CPU.intel?` for Intel (amd64). This ensures the correct binary is downloaded for each architecture.
   - **SHA256 placeholders**: Each architecture has its own sha256 checksum. These are placeholders that must be updated for each release (either manually or via CI automation).
   - **Install method**: The archive contains only the `tick` binary (per goreleaser configuration in task 1-2), so `bin.install "tick"` is sufficient.
   - **Test block**: Runs the binary and asserts output contains "tick" — matches the minimal binary from task 1-1.

3. **Create a README** at `homebrew-tap/README.md` explaining usage.

4. **Validate the formula syntax** by running `brew style homebrew-tap/Formula/tick.rb` (if Homebrew is available locally) or by reviewing against Homebrew formula conventions:
   - Class name matches formula filename (capitalized): `Tick` for `tick.rb`.
   - `desc`, `homepage`, `version`, `url`, `sha256` are all present.
   - `on_macos` block correctly branches on CPU type.
   - `def install` uses `bin.install` for the binary.
   - `test` block is present.

5. **Document the release update process** — add a comment at the top of the formula file or a section in the README explaining that for each release, the following must be updated:
   - `version` — set to the new version string (without `v` prefix, e.g., `1.2.3`).
   - `sha256` — one for each architecture, computed from the release assets.

## Tests

- `"formula file exists at homebrew-tap/Formula/tick.rb"` — verify the file is present and contains a valid Ruby class definition
- `"formula class name is Tick"` — parse the formula and assert the class is named `Tick` and inherits from `Formula`
- `"formula URL for Apple Silicon uses darwin_arm64 asset"` — assert the arm64 branch URL matches `https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_arm64.tar.gz`
- `"formula URL for Intel uses darwin_amd64 asset"` — assert the amd64 branch URL matches `https://github.com/leeovery/tick/releases/download/v#{version}/tick_#{version}_darwin_amd64.tar.gz`
- `"formula URL tag path includes v prefix but filename does not"` — verify the download URL uses `/download/v#{version}/tick_#{version}_` pattern, confirming the tag has `v` and the filename does not
- `"formula handles both Intel and Apple Silicon via on_macos block"` — assert the formula contains `Hardware::CPU.arm?` and `Hardware::CPU.intel?` branches within an `on_macos` block
- `"formula installs tick binary to bin"` — assert the install method contains `bin.install "tick"`
- `"formula includes a test block"` — assert the formula has a `test do` block
- `"formula includes sha256 for each architecture"` — assert there are two distinct sha256 declarations, one in each architecture branch
- `"README includes tap and install commands"` — verify the README contains `brew tap leeovery/tick` and `brew install tick`

## Edge Cases

- **Formula must handle both Intel and Apple Silicon macOS**: The formula uses Homebrew's `on_macos` block with `Hardware::CPU.arm?` and `Hardware::CPU.intel?` to serve the correct architecture-specific binary. Without this branching, users on one architecture would receive the wrong binary. The goreleaser configuration (task 1-2) produces separate archives for `darwin_amd64` and `darwin_arm64`, and the formula must select between them.

- **Version in formula URL must strip leading `v`**: GitHub release tags follow the `vX.Y.Z` convention (e.g., `v1.2.3`), which goreleaser uses for the download path (`/download/v1.2.3/`). However, goreleaser's `{{ .Version }}` template variable strips the `v` prefix, producing archive filenames like `tick_1.2.3_darwin_arm64.tar.gz`. The formula must use `v#{version}` in the tag path segment and `#{version}` (no `v`) in the filename. If the formula's `version` field is set to `1.2.3` (without `v`), this works naturally. The `version` field must never include the `v` prefix.

## Acceptance Criteria

- [ ] `homebrew-tap/Formula/tick.rb` exists with a valid Homebrew formula
- [ ] Formula class is named `Tick` and inherits from `Formula`
- [ ] Formula serves `darwin_arm64` archive for Apple Silicon Macs (`Hardware::CPU.arm?`)
- [ ] Formula serves `darwin_amd64` archive for Intel Macs (`Hardware::CPU.intel?`)
- [ ] Download URLs follow the pattern `https://github.com/leeovery/tick/releases/download/v{version}/tick_{version}_darwin_{arch}.tar.gz` (tag has `v`, filename does not)
- [ ] Each architecture branch has its own `sha256` declaration
- [ ] `def install` places the `tick` binary into Homebrew's bin directory via `bin.install "tick"`
- [ ] Formula includes a `test do` block that verifies the binary runs
- [ ] `homebrew-tap/README.md` exists with tap and install instructions
- [ ] The `version` field uses the bare version number without a `v` prefix (e.g., `1.2.3` not `v1.2.3`)

## Context

The specification defines Homebrew as the preferred macOS installation method:

> **Homebrew (macOS)**:
> ```bash
> brew tap {owner}/tick
> brew install tick
> ```
> - Preferred method for macOS users
> - Handles code signing automatically
> - Manages updates via `brew upgrade tick`

The release asset naming convention established in task 1-2 (goreleaser configuration) produces archives as `tick_X.Y.Z_{os}_{arch}.tar.gz`. goreleaser's `{{ .Version }}` template strips the leading `v` from the git tag — so tag `v1.2.3` produces archives named `tick_1.2.3_...`. The formula URL must account for this.

The tap will be published as a separate GitHub repository (`leeovery/homebrew-tick`) following Homebrew's convention that `brew tap leeovery/tick` looks for a repository named `homebrew-tick`. For now, the formula is authored locally under `homebrew-tap/` for validation. The actual publishing to a separate repository is an operational concern outside the scope of this task.

Starting with a custom tap is the right approach for a new project. The formula syntax is identical to what homebrew-core requires, so migrating later (once the project meets homebrew-core's notability threshold of ~50+ GitHub stars) is trivial — submit the same formula as a PR to `homebrew/homebrew-core`.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
