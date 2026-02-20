TASK: Linux Install Script (installation-1-4)

ACCEPTANCE CRITERIA:
- [ ] `scripts/install.sh` exists and is executable
- [ ] Script detects OS via `uname -s` and rejects non-Linux platforms with a clear error (exit 1)
- [ ] Script detects architecture via `uname -m` and maps x86_64->amd64, aarch64->arm64, arm64->arm64
- [ ] Script exits with a clear error (exit 1) for unsupported architectures
- [ ] Script resolves the latest release version from the GitHub API without requiring jq
- [ ] Script constructs the download URL following `tick_{version}_{os}_{arch}.tar.gz` convention (version without leading v)
- [ ] Script installs to `/usr/local/bin` when writable, falls back to `~/.local/bin` otherwise
- [ ] Script creates `~/.local/bin` via `mkdir -p` if it does not exist during fallback
- [ ] Script overwrites any existing `tick` binary without prompting or version checking
- [ ] Script prints a PATH warning when installing to `~/.local/bin` and it is not in `$PATH`
- [ ] Script cleans up temporary files on both success and failure (trap)
- [ ] Script uses `set -euo pipefail` for strict error handling
- [ ] Script works when piped via `curl -fsSL ... | bash` (no interactive prompts, no TTY assumptions)

STATUS: Complete

SPEC CONTEXT:
The specification defines `scripts/install.sh` as the primary documented installation method, invoked via `curl -fsSL ... | bash`. On Linux it must: detect platform via uname, download the latest release binary from GitHub, install to `/usr/local/bin` (preferred) or `~/.local/bin` (fallback), overwrite existing binaries without version checking, and fail cleanly on download errors with no fallback to `go install` or source builds. Architecture mapping: x86_64->amd64, aarch64->arm64, arm64->arm64. macOS delegates to Homebrew (Phase 2 scope, but implemented in the same script).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/scripts/install.sh:1-194
- Notes:
  All acceptance criteria are met:
  1. File exists and is executable (verified by tests).
  2. `set -euo pipefail` on line 2.
  3. OS detection via `detect_os()` (lines 12-22): maps Linux->linux, Darwin->darwin, rejects all others with clear error including the OS name.
  4. Architecture detection via `detect_arch()` (lines 24-35): maps x86_64->amd64, aarch64->arm64, arm64->arm64, rejects others with clear error.
  5. Version resolution via `resolve_version()` (lines 37-49): uses `curl -fsSL` + `grep`/`sed` -- no jq dependency. GITHUB_API is configurable via env var for testing.
  6. URL construction via `construct_url()` (lines 51-57): follows `tick_{version_no_v}_{os}_{arch}.tar.gz` convention exactly, stripping leading `v` from tag.
  7. Install location via `select_install_dir()` (lines 71-96): checks primary dir writability, falls back, creates fallback with `mkdir -p`, warns about PATH if fallback not in PATH.
  8. Temp directory cleanup via `trap 'rm -rf "${TMPDIR_INSTALL}"' EXIT` on line 156.
  9. Binary install via `cp` + `chmod +x` (lines 186-187) -- overwrites any existing binary.
  10. Script wrapped in `main()` function for safe `curl | bash` piping.
  11. No interactive commands (no `read`, `select`, or TTY assumptions).
  12. Testability: env var hooks (`TICK_TEST_UNAME_S`, `TICK_TEST_UNAME_M`, `TICK_TEST_VERSION`, `TICK_TEST_TARBALL`, `TICK_TEST_MODE`, `TICK_INSTALL_DIR`, `TICK_FALLBACK_DIR`, `TICK_TEST_PATH`, `TICK_TEST_ECHO_TMPDIR`) enable comprehensive Go-driven testing without real network calls.

  Minor observation: The plan task originally scoped this as "Linux path only" with macOS rejection, but the implementation also handles the macOS/Homebrew path (Phase 2 work). This is acceptable -- the Phase 2 macOS tasks built on top of this file. The Linux path is complete and correct as specified.

TESTS:
- Status: Adequate
- Coverage:
  Test file: /Users/leeovery/Code/tick/scripts/install_test.go (1045 lines, ~50 test cases)
  Mapped to plan test requirements:
  - "it detects Linux OS and maps to linux" -- TestOSDetection/"accepts Linux OS" (line 123)
  - "it maps x86_64 architecture to amd64" -- TestArchDetection table (line 157)
  - "it maps aarch64 architecture to arm64" -- TestArchDetection table (line 158)
  - "it maps arm64 architecture to arm64" -- TestArchDetection table (line 159)
  - "it exits with error for unsupported architecture" -- TestArchDetection table i686 (line 160), ppc64le (line 161)
  - "it exits with error for non-Linux OS" -- TestOSDetection/"rejects unsupported OS" (line 110), "rejects FreeBSD" (line 136)
  - "it constructs the correct download URL" -- TestURLConstruction (line 190) with 3 sub-tests covering amd64, arm64, and v-prefix stripping
  - "it installs to /usr/local/bin when writable" -- TestInstallDirectorySelection/"installs to custom dir when writable" (line 261)
  - "it falls back to ~/.local/bin when /usr/local/bin is not writable" -- TestInstallDirectorySelection/"falls back when primary dir is not writable" (line 283)
  - "it creates ~/.local/bin if it does not exist" -- TestInstallDirectorySelection/"creates fallback dir via mkdir -p" (line 308)
  - "it overwrites an existing binary" -- TestFullInstallFlow/"overwrites existing binary without prompting" (line 544)
  - "it warns when ~/.local/bin is not in PATH" -- TestPATHWarning/"prints PATH warning when fallback dir not in PATH" (line 358)
  - "it does not warn about PATH when installing to /usr/local/bin" -- TestPATHWarning/"no PATH warning when primary dir is writable" (line 339)
  - "it cleans up temporary directory on success" -- TestFullInstallFlow/"cleans up temp directory on success" (line 612)
  - "it cleans up temporary directory on failure" -- TestFullInstallFlow/"cleans up temp directory on failure" (line 582)
  - "it exits with error when version resolution fails" -- TestVersionResolution/"fails with error when GitHub API is unreachable" (line 244)

  Additional tests beyond plan requirements (from Phase 2 and hardening tasks):
  - TestMacOSInstall (7 sub-tests) -- macOS/Homebrew delegation
  - TestMacOSNoBrewError (5 sub-tests) -- macOS without Homebrew
  - TestErrorHandlingHardening (9 sub-tests) -- main function wrapping, curl flags, corrupt archives, missing binaries in archive, interactive command checks

  Edge cases from spec all covered:
  - /usr/local/bin not writable -> fallback: covered
  - ~/.local/bin may not exist: covered (mkdir -p test)
  - Overwrite existing binary: covered
  - Unsupported architecture: covered (i686, ppc64le)

- Notes:
  Test quality is good. Tests use env var injection to test script functions in isolation (TICK_TEST_MODE dispatch), which is a pragmatic approach for testing bash scripts from Go. The full_install tests use fake tarballs to avoid network calls. No over-testing detected -- each test verifies a distinct behavior. The table-driven approach for arch detection is clean and idiomatic Go.

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers -- all per CLAUDE.md conventions.
- SOLID principles: Good. The bash script decomposes logic into focused functions (detect_os, detect_arch, resolve_version, construct_url, select_install_dir, install_macos). Each has single responsibility. Test helpers (scriptPath, loadScript, runScript, extractTmpDir, createFakeTarball, createFakeBrew) are well-factored.
- Complexity: Low. The script has straightforward control flow. Each function is a simple case/if chain. The test mode dispatch is clean. No deeply nested logic.
- Modern idioms: Yes. Bash uses `set -euo pipefail`, `local` variables, `trap` for cleanup, `mktemp -d`, parameter expansion (`${version#v}`), and a main() wrapper for curl-pipe safety. Go tests use table-driven subtests where appropriate.
- Readability: Good. Script functions are well-named and self-documenting. The TICK_TEST_* env var convention is consistent and clear. The test file has useful comments explaining helper behavior (e.g., createFakeBrew behavior parameter docs).
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `select_install_dir` function on line 75 checks `[ -d "${primary}" && -w "${primary}" ]`. If `/usr/local/bin` does not exist at all (rare but possible), it correctly falls back because `-d` fails. This is the right behavior but not explicitly tested -- the "not writable" test creates the directory as 0555. Could add a test where the primary directory does not exist at all, though this is a minor gap given the `-d` check naturally handles it.
- The `loadScript` helper reads the script into memory for content assertions. This is used by 5 tests. Given these are static content checks (shebang, set flags, trap presence, no interactive commands), they could be considered slightly fragile if the script format changes, but they verify meaningful properties and are appropriate.
- The `TestVersionResolution` test (line 243) uses `file:///nonexistent` to simulate API failure. The comment notes that curl's own failure under `set -e` may exit before the custom error message is reached. The test pragmatically only checks the exit code, which is sufficient.
