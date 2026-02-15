TASK: Install Script Error Handling Hardening (installation-2-4)

ACCEPTANCE CRITERIA:
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

STATUS: Complete

SPEC CONTEXT: The specification defines the install script invocation pattern as `curl -fsSL ... | bash`. Design principles emphasize fail-fast with clear messages, no partial state, early error detection, and no fallbacks on download failure. Four supported platforms (darwin/linux x amd64/arm64); any OS not matching Linux or Darwin should be rejected.

IMPLEMENTATION:
- Status: Implemented with minor drift from Implementation steps
- Location: /Users/leeovery/Code/tick/scripts/install.sh
- Notes:
  - main() wrapper: Line 4 `main() {`, line 191 `}`, line 193 `main "$@"` -- correct
  - set -euo pipefail: Line 2 -- correct
  - curl -f flag: Both curl calls (line 43 for API, line 166 for download) use `-fsSL` -- correct
  - Archive size check: Lines 169-172 use `[ ! -s ... ]` -- correct
  - Tar extraction check: Lines 174-177 use `if ! tar xzf ...` -- correct
  - Binary existence check: Lines 179-182 use `[ ! -f ... ]` -- correct
  - Unsupported OS catch-all: Lines 14-21 in `detect_os()` with `*` case -- correct, includes OS name in message
  - Trap cleanup: Line 156 `trap 'rm -rf "${TMPDIR_INSTALL}"' EXIT` -- correct
  - No interactive commands: No `read` or `select` found -- correct
  - Minor drift 1: Error messages after curl failures rely on `set -e` to exit; no custom script-level messages for HTTP errors on the download curl call. The task Implementation step 3 specified distinct messages like "Failed to determine latest version..." and "Failed to download tick ${VERSION}..." after each curl. In practice, curl's own stderr output is visible, and `set -e` ensures non-zero exit, but the user does not see a script-authored error message for these paths.
  - Minor drift 2: Error messages are slightly shorter than specified in Implementation steps. For example, "Downloaded archive is empty or missing." omits "This may indicate a network issue..." (step 4). "Binary 'tick' not found in archive." omits "The release asset may have an unexpected structure." (step 6). Still actionable, just less detailed.
  - Minor drift 3: The version resolution error message (line 45) says "could not resolve latest version from GitHub API" rather than the specified "Failed to determine latest version. Check network connectivity and that GitHub is accessible." The message is still actionable but less prescriptive about troubleshooting.

TESTS:
- Status: Adequate
- Coverage:
  - TestErrorHandlingHardening (lines 820-1044) covers 14 of the 16 specified test cases:
    - main() function wrapper and last-line check: COVERED
    - curl -f flag for both API and download calls: COVERED (static analysis)
    - Empty archive error: COVERED (functional test with zero-byte file)
    - Corrupt archive error: COVERED (functional test with non-gzip content)
    - Binary not found in archive: COVERED (functional test with valid tar.gz missing tick)
    - FreeBSD, CYGWIN, unknown OS, and OS name in error: COVERED (4 test cases)
    - No read/select commands: COVERED (static analysis of script)
    - set -euo pipefail: COVERED (in TestInstallScript)
    - Trap cleanup: COVERED (basic in TestTrapCleanup, functional in TestFullInstallFlow lines 582-641)
  - "script prints specific error for failed version resolution": PARTIALLY covered -- TestVersionResolution (line 243) tests exit code but explicitly notes the custom error message may not be reached; does not verify message text
  - "script prints specific error for failed download": NOT covered -- no test exists for curl download failure producing a specific error message
- Notes:
  - The two missing/partial test cases align with the implementation drift: since the script does not produce custom messages for curl failures (relying on set -e), there is nothing to test. This is consistent but both the implementation and test are slightly under-specified relative to the task plan.
  - Tests are well-structured with clear test names, proper use of subtests, and the `runScript` helper pattern avoids duplication
  - No over-testing detected -- each test verifies a distinct error path
  - Edge cases from the task (FreeBSD, CYGWIN, partial download via main wrapper, curl pipe safety) are all covered

CODE QUALITY:
- Project conventions: N/A (bash script, not Go code; Go test conventions followed well -- stdlib testing, t.Run subtests, t.Helper, t.TempDir)
- SOLID principles: Good -- detect_os, detect_arch, resolve_version, construct_url, select_install_dir are well-separated single-responsibility functions. Test mode dispatch provides clean testability without polluting the main flow.
- Complexity: Low -- each function is short with clear control flow. The main install flow is linear with early-exit error checks.
- Modern idioms: Yes -- uses `${VAR:-default}` for test injection, `set -euo pipefail`, `trap ... EXIT`, `command -v` instead of `which`, `[[ ]]` instead of `[ ]` for conditionals
- Readability: Good -- clear section comments ("Testable functions", "Test mode dispatch", "Main install flow"), descriptive error messages, consistent formatting
- Security: No issues -- no eval, no unquoted variables, all variables properly quoted with `"${...}"`
- Performance: N/A (install script, not a hot path)
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Error messages for curl download failures could be more user-friendly. Currently, if the archive download curl call fails, the user sees only curl's stderr (e.g., "curl: (22) The requested URL returned error: 404 Not Found") with no script-level context. Adding an `|| { echo "Error: Failed to download..." >&2; exit 1; }` after the curl call on line 166 (and similarly wrapping the resolve_version curl on line 43) would match the task's Implementation steps and provide a better user experience. This is non-blocking because `set -e` ensures the script exits non-zero, and curl's `-f` flag stderr output is reasonably informative.
- Error messages are slightly shorter than specified in the Implementation steps (missing trailing explanatory sentences). The current messages are still actionable; adding the additional context like "This may indicate a network issue or that the release asset does not exist for this platform" would improve the user experience for less experienced users.
- Line 180 hardcodes `'tick'` instead of using `'${BINARY_NAME}'`. While the value is the same, using the variable would be more consistent and maintainable if the binary name ever changes.
