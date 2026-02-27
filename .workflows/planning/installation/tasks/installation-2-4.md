---
id: installation-2-4
phase: 2
status: completed
created: 2026-01-31
---

# Install Script Error Handling Hardening

## Goal

The install script built in Phase 1 (task 1-4) implements the Linux happy path and basic error exits, and Phase 2 tasks 2-2/2-3 add macOS behavior. However, the script does not yet handle several real-world failure modes that arise in production usage: curl piping with server errors, partial/corrupt downloads, and OS values that are neither Linux nor Darwin. This task hardens `scripts/install.sh` so that every failure path produces a clear, actionable error message and a non-zero exit code — critical because the script's primary invocation pattern is `curl -fsSL ... | bash` in ephemeral environments where silent failures waste developer time.

## Implementation

1. **Wrap script body in a `main()` function** with `main "$@"` as the last line. This ensures bash must receive the complete file before execution begins. If the download is truncated, `main` is never called and nothing executes.

2. **Verify `set -euo pipefail`** is present at the top (from task 1-4). `set -e` exits on error, `set -u` catches unset variables, `set -o pipefail` propagates failures through pipes.

3. **Verify all `curl` calls use `-f` flag** for HTTP error detection. The `-f` flag causes curl to return a non-zero exit code on HTTP errors rather than saving the error page as content. Check both:
   - The GitHub API call to resolve the latest version.
   - The archive download call.
   - After each curl call, print a specific error message: for version resolution: `"Error: Failed to determine latest version. Check network connectivity and that GitHub is accessible."` For download: `"Error: Failed to download tick ${VERSION} for ${OS}/${ARCH}. The release asset may not exist."`.

4. **Validate the downloaded archive before extraction**:
   - Check the file exists and has non-zero size: `[ -s "${TMP_DIR}/${ARCHIVE}" ]`.
   - If empty or missing, exit with: `"Error: Downloaded archive is empty or missing. This may indicate a network issue or that the release asset does not exist for this platform."` Exit code 1.

5. **Catch tar extraction failures** — wrap the `tar -xzf` call so extraction failure is caught and reported: `"Error: Failed to extract archive. The download may be corrupt."` Exit code 1.

6. **Validate the extracted binary exists** after tar extraction:
   - Check `[ -f "${TMP_DIR}/${BINARY_NAME}" ]`.
   - If not found, exit with: `"Error: Binary '${BINARY_NAME}' not found in archive. The release asset may have an unexpected structure."` Exit code 1.

7. **Add explicit catch-all for unsupported OS**. After the `Linux` and `Darwin` branches, add a catch-all that exits with: `"Error: Unsupported operating system: $(uname -s). This installer supports Linux and macOS only."` Exit code 1. This covers FreeBSD, NetBSD, SunOS, CYGWIN, MINGW, etc.

8. **Verify the `trap` cleanup is comprehensive**. Confirm the trap covers `EXIT` so cleanup happens on success, failure, and interruption:
   ```bash
   trap 'rm -rf "${TMP_DIR}"' EXIT
   ```

9. **Ensure no interactive prompts or TTY assumptions**. Verify the script does not use `read`, `select`, or any command that requires a TTY. When piped via `curl | bash`, stdin is the pipe, not the terminal.

## Tests

- `"script body is wrapped in a main function"` — parse `scripts/install.sh` and verify the executable logic is inside a function (e.g., `main()`) with a `main "$@"` call as the last non-comment line
- `"main function call is the last line"` — verify `main "$@"` or `main` appears as the final statement, ensuring a truncated download results in no execution
- `"curl uses -f flag for version resolution API call"` — grep the script and verify the curl call to the GitHub API includes `-f` (or `-fsSL`)
- `"curl uses -f flag for archive download"` — grep the script and verify the curl call to download the archive includes `-f` (or `-fsSL`)
- `"script exits with error for empty downloaded archive"` — mock a download that produces a zero-byte file and verify the script exits with code 1 and an error message mentioning empty or missing
- `"script exits with error for corrupt archive"` — provide a file that is not a valid gzip archive and verify `tar` failure is caught, exit code 1, error message mentions corrupt
- `"script exits with error when binary not found in archive"` — provide a valid tar.gz that does not contain the `tick` binary and verify exit code 1 with error message
- `"script exits with error for FreeBSD"` — mock `uname -s` returning `FreeBSD` and verify exit code 1 with error message naming the unsupported OS
- `"script exits with error for CYGWIN"` — mock `uname -s` returning `CYGWIN_NT-10.0` and verify exit code 1 with error message
- `"script exits with error for unknown OS"` — mock `uname -s` returning an arbitrary string (e.g., `FooOS`) and verify exit code 1
- `"unsupported OS error message includes the detected OS name"` — verify the error output contains the actual `uname -s` value so the user knows what was detected
- `"script does not contain read or select commands"` — grep the script for `read ` and `select ` (excluding comments) and verify none are found
- `"script uses set -euo pipefail"` — verify the script contains `set -euo pipefail` near the top
- `"trap cleans up temp directory on EXIT"` — verify the script contains a trap on `EXIT` that removes the temp directory
- `"script prints specific error for failed version resolution"` — mock the GitHub API curl call to fail and verify the error message mentions version resolution and network/GitHub accessibility
- `"script prints specific error for failed download"` — mock the archive download curl call to fail and verify the error message includes the version, OS, and architecture

## Edge Cases

- **Script piped via curl with server error**: When invoked as `curl -fsSL URL | bash`, if the server returns an HTTP error, curl's `-f` flag causes curl to exit non-zero. The `main()` function wrapper ensures the script body is not executed until the entire file is received. If curl fails before the script is fully transmitted, `main` is never called and nothing executes.

- **Partial download**: If the network connection drops mid-transfer while piping the install script, bash receives a truncated file. Without the `main()` wrapper, bash could execute partial commands with unpredictable results. With the wrapper, the `main "$@"` call at the end is never received, so the function is defined but never invoked. For partial downloads of the release asset archive, the file size check and tar extraction validation catch the corruption.

- **OS value that is neither Linux nor Darwin (e.g., FreeBSD)**: The OS detection uses `uname -s` which can return `FreeBSD`, `NetBSD`, `OpenBSD`, `SunOS`, `CYGWIN_NT-10.0`, `MINGW64_NT-10.0`, etc. The explicit catch-all rejects any value not in the supported set with a clear error message naming the detected OS.

## Acceptance Criteria

- [ ] Script body is wrapped in a `main()` function with `main "$@"` as the last line
- [ ] All `curl` calls use the `-f` flag to fail on HTTP errors
- [ ] Script validates the downloaded archive is non-empty before extraction
- [ ] Script catches and reports tar extraction failures with a clear error message
- [ ] Script validates the expected binary exists after extraction
- [ ] Script exits with code 1 and a descriptive error for any OS other than Linux or Darwin
- [ ] Unsupported OS error message includes the actual detected OS name
- [ ] Script contains no interactive commands (`read`, `select`)
- [ ] Script uses `set -euo pipefail`
- [ ] Trap on `EXIT` ensures temp directory cleanup on all exit paths
- [ ] Each error path produces a distinct, actionable error message
- [ ] Script works correctly when invoked as `curl -fsSL ... | bash`

## Context

The specification defines the install script invocation pattern as `curl -fsSL https://raw.githubusercontent.com/{repo}/main/scripts/install.sh | bash`. The specification's design principles emphasize:

- **Simple**: Error handling should fail fast with clear messages, not retry or attempt alternatives.
- **Idempotent**: Error handling must not leave partial state (temp files, broken symlinks).
- **Fast**: Error detection should happen early.
- **No fallbacks on download failure**: "If binary download fails, script fails. No `go install` or source build fallback."

The `main()` function wrapper pattern is a well-established best practice for scripts distributed via `curl | bash`. It prevents partial execution if the download is interrupted.

The specification lists four supported platforms and explicitly states Windows is not in scope for v1. Any OS not matching Linux or Darwin should be rejected immediately.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
