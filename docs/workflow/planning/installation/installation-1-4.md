---
id: installation-1-4
phase: 1
status: pending
created: 2026-01-31
---

# Linux Install Script

## Goal

The release pipeline (tasks 1-1 through 1-3) produces release assets on GitHub, but there is no automated way for users to download and install the binary. This task creates `scripts/install.sh` — the primary documented installation method — implementing the Linux path: detect platform, download the correct release asset, and install the binary to an appropriate location. This is essential for ephemeral environments like Claude Code for Web and CI/CD pipelines where fast, idempotent installation via `curl | bash` is the primary use case.

## Implementation

1. **Create `scripts/install.sh`** with a bash shebang (`#!/usr/bin/env bash`) and enable strict mode (`set -euo pipefail`).

2. **Define constants** at the top of the script:
   - `REPO="leeovery/tick"` — the GitHub owner/repo.
   - `BINARY_NAME="tick"` — the name of the binary.
   - `GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"` — endpoint to resolve the latest release tag.

3. **Detect the operating system** using `uname -s`:
   - Map `Linux` to `linux`.
   - For any other value (including `Darwin`), exit with an error message. macOS behavior is out of scope for this task (Phase 2). Use a clear message like: `"Error: This installer does not support $(uname -s). macOS users should install via Homebrew."` Exit code 1.

4. **Detect the architecture** using `uname -m` and map to asset architecture:
   - `x86_64` maps to `amd64`.
   - `aarch64` maps to `arm64`.
   - `arm64` maps to `arm64`.
   - Any other value: exit with error message `"Error: Unsupported architecture: $(uname -m). Supported: x86_64, aarch64, arm64."` Exit code 1.

5. **Resolve the latest version** by querying the GitHub releases API:
   - Use `curl -fsSL` to fetch the latest release JSON from `$GITHUB_API`.
   - Extract the tag name (version) using `grep` and `sed` (or similar POSIX-compatible text processing) to avoid requiring `jq` as a dependency. The pattern to extract: `"tag_name": "vX.Y.Z"` — strip to just the version string (e.g., `v1.2.3`).
   - If the curl call fails or the version cannot be extracted, exit with error message `"Error: Failed to determine latest version."` Exit code 1.

6. **Construct the download URL** following the goreleaser naming convention:
   - Format: `https://github.com/${REPO}/releases/download/${VERSION}/tick_${VERSION_WITHOUT_V}_${OS}_${ARCH}.tar.gz`
   - Strip the leading `v` from the version for the filename portion (e.g., tag `v1.2.3` produces `tick_1.2.3_linux_amd64.tar.gz`).

7. **Download and extract the binary**:
   - Create a temporary directory using `mktemp -d`.
   - `curl -fsSL` the archive into the temp directory.
   - `tar -xzf` to extract the `tick` binary.
   - Set up a `trap` to clean up the temp directory on exit (both success and failure).
   - If download fails, exit with error message `"Error: Failed to download tick ${VERSION} for ${OS}/${ARCH}."` Exit code 1.

8. **Determine the install location** with fallback logic:
   - First choice: `/usr/local/bin` — test writability with `[ -w /usr/local/bin ]` (no sudo).
   - Fallback: `~/.local/bin` — if `/usr/local/bin` is not writable.
   - If `~/.local/bin` does not exist, create it with `mkdir -p ~/.local/bin`.
   - Print which install directory is being used so the user knows.

9. **Install the binary**:
   - Copy (or move) the extracted `tick` binary to the chosen install directory using `install -m 755` (or `cp` + `chmod +x`).
   - This overwrites any existing `tick` binary at that path (idempotent — no version checking).

10. **Print PATH warning if needed**:
    - If the install location is `~/.local/bin`, check whether it is in the user's `$PATH`.
    - If not in PATH, print a warning: `"Warning: ~/.local/bin is not in your PATH. Add it with: export PATH=\"\$HOME/.local/bin:\$PATH\""`.

11. **Print success message**:
    - Print the installed version and location, e.g., `"tick ${VERSION} installed to ${INSTALL_DIR}/tick"`.

12. **Make the script executable**: Ensure `scripts/install.sh` has the execute permission bit set (`chmod +x`).

## Tests

- `"it detects Linux OS and maps to linux"` — on a Linux system (or by mocking `uname -s` to return `Linux`), verify the script identifies the OS as `linux`
- `"it maps x86_64 architecture to amd64"` — mock `uname -m` returning `x86_64` and verify the script resolves arch to `amd64`
- `"it maps aarch64 architecture to arm64"` — mock `uname -m` returning `aarch64` and verify the script resolves arch to `arm64`
- `"it maps arm64 architecture to arm64"` — mock `uname -m` returning `arm64` and verify the script resolves arch to `arm64`
- `"it exits with error for unsupported architecture"` — mock `uname -m` returning `i386` or `ppc64` and verify the script exits with code 1 and an error message naming the unsupported architecture
- `"it exits with error for non-Linux OS"` — mock `uname -s` returning `Darwin` or `FreeBSD` and verify the script exits with code 1 and an appropriate error message
- `"it constructs the correct download URL"` — for a given version, OS, and arch, verify the URL matches `https://github.com/leeovery/tick/releases/download/v1.2.3/tick_1.2.3_linux_amd64.tar.gz`
- `"it installs to /usr/local/bin when writable"` — when `/usr/local/bin` is writable, verify the binary is placed there
- `"it falls back to ~/.local/bin when /usr/local/bin is not writable"` — when `/usr/local/bin` is not writable, verify the binary is placed in `~/.local/bin`
- `"it creates ~/.local/bin if it does not exist"` — when falling back and `~/.local/bin` does not exist, verify the directory is created and the binary is placed there
- `"it overwrites an existing binary"` — place a dummy file at the install location named `tick`, run the script, and verify it is replaced with the new binary
- `"it warns when ~/.local/bin is not in PATH"` — when installing to `~/.local/bin` and it is not in `$PATH`, verify a warning is printed with instructions to add it
- `"it does not warn about PATH when installing to /usr/local/bin"` — when installing to `/usr/local/bin`, verify no PATH warning is emitted
- `"it cleans up temporary directory on success"` — verify the temp directory created during download no longer exists after the script completes
- `"it cleans up temporary directory on failure"` — simulate a download failure and verify the temp directory is still cleaned up
- `"it exits with error when version resolution fails"` — mock the GitHub API call to fail and verify exit code 1 with an error message

## Edge Cases

- **`/usr/local/bin` not writable triggers `~/.local/bin` fallback**: The script must check writability without attempting sudo. If `/usr/local/bin` is not writable (common in containers and restricted environments), it silently falls back to `~/.local/bin`.
- **`~/.local/bin` may not exist**: When falling back, the directory might not be present. The script must `mkdir -p ~/.local/bin` before attempting to install there. This is safe and idempotent.
- **Overwrite existing binary**: The spec mandates "overwrite by default" with no version checking. If `tick` already exists at the install location, it is replaced unconditionally. This supports the idempotent design principle — re-running the script always installs the latest.
- **Unsupported architecture**: If `uname -m` returns a value not in the mapping (`x86_64`, `aarch64`, `arm64`), the script must exit with a clear error naming the detected architecture rather than attempting a download that would fail with a confusing 404.

## Acceptance Criteria

- [ ] `scripts/install.sh` exists and is executable
- [ ] Script detects OS via `uname -s` and rejects non-Linux platforms with a clear error (exit 1)
- [ ] Script detects architecture via `uname -m` and maps `x86_64` to `amd64`, `aarch64` to `arm64`, `arm64` to `arm64`
- [ ] Script exits with a clear error (exit 1) for unsupported architectures
- [ ] Script resolves the latest release version from the GitHub API without requiring `jq`
- [ ] Script constructs the download URL following `tick_{version}_{os}_{arch}.tar.gz` convention (version without leading `v`)
- [ ] Script installs to `/usr/local/bin` when writable, falls back to `~/.local/bin` otherwise
- [ ] Script creates `~/.local/bin` via `mkdir -p` if it does not exist during fallback
- [ ] Script overwrites any existing `tick` binary without prompting or version checking
- [ ] Script prints a PATH warning when installing to `~/.local/bin` and it is not in `$PATH`
- [ ] Script cleans up temporary files on both success and failure (trap)
- [ ] Script uses `set -euo pipefail` for strict error handling
- [ ] Script works when piped via `curl -fsSL ... | bash` (no interactive prompts, no TTY assumptions)

## Context

The specification defines the install script as the primary documented installation method, with the invocation pattern `curl -fsSL https://raw.githubusercontent.com/leeovery/tick/main/scripts/install.sh | bash`. It must work in ephemeral environments (Claude Code for Web, CI/CD) where speed and idempotency are critical — the script may run at every session start.

The specification explicitly states:
- **Install location priority**: `/usr/local/bin` if writable (no sudo), then `~/.local/bin` as fallback (user-writable, XDG-compliant).
- **Overwrite by default**: "If user runs install script, they want latest version" — no version checking.
- **No fallbacks on download failure**: "If binary download fails, script fails. No `go install` or source build fallback."
- **Architecture mapping**: `x86_64` -> `amd64`, `aarch64` -> `arm64`, `arm64` -> `arm64`.

The goreleaser naming convention (from task installation-1-2) produces archives as `tick_X.Y.Z_{os}_{arch}.tar.gz`. The download URL construction in this script must match that convention exactly. The version tag from GitHub releases uses the `vX.Y.Z` format, but the archive filename uses `X.Y.Z` (no `v` prefix).

This task implements only the Linux path. macOS behavior (Homebrew delegation) is handled in Phase 2. The script should detect macOS and exit with a helpful message rather than silently failing.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
