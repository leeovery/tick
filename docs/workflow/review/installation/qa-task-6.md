TASK: macOS Install Script: Homebrew Delegation (installation-2-2)

ACCEPTANCE CRITERIA:
- scripts/install.sh detects macOS via uname -s returning Darwin and enters Homebrew delegation path
- When brew is available on macOS, the script runs brew tap leeovery/tick followed by brew install tick
- A failure in brew tap prevents brew install from running and causes the script to exit non-zero
- A failure in brew install causes the script to exit non-zero
- On successful Homebrew install, the script prints a success message and exits 0
- The macOS path does not execute any Linux-specific logic
- Re-running the script when tick is already installed via Homebrew completes without error (idempotent)
- brew command output (stdout/stderr) is visible to the user -- not suppressed
- Script continues to work correctly when piped via curl -fsSL ... | bash

STATUS: Complete

SPEC CONTEXT:
The specification defines macOS install script behavior as: if brew command exists, run "brew tap {owner}/tick && brew install tick" and exit successfully. macOS does not handle direct binary downloads to avoid code signing complexity. The design principles require simplicity, idempotency, and speed.

IMPLEMENTATION:
- Status: Implemented (with intentional drift from original task wording)
- Location: /Users/leeovery/Code/tick/scripts/install.sh:59-69 (install_macos function), lines 146-149 (main flow macOS branch)
- Notes:
  - The task specified two separate commands: `brew tap leeovery/tick && brew install tick`. The implementation uses the single-command form `brew install leeovery/tools/tick` (line 66), which implicitly taps and installs. This is functionally equivalent and actually superior -- it avoids the tap-then-install race and is idiomatic Homebrew usage.
  - The tap name changed from `leeovery/tick` to `leeovery/tools` (confirmed by git commit c4a2a84 "Migrate Homebrew TAP from homebrew-tick to homebrew-tools"). This is an intentional post-task migration, not a bug. The implementation is current with the actual tap name.
  - OS detection via `detect_os()` (lines 12-22) correctly maps `Darwin` to `darwin` using `TICK_TEST_UNAME_S` env var for testability.
  - The main flow (line 146-149) branches early on darwin, calls `install_macos`, and exits 0 -- no Linux logic executes.
  - Brew availability check uses `command -v brew &> /dev/null` (line 60), which is portable and correct.
  - `set -euo pipefail` (line 2) ensures any brew failure propagates as non-zero exit.
  - Brew output is not redirected or suppressed -- visible to user.
  - Script wraps body in `main() {}` with `main "$@"` at end (lines 4, 191-193) for safe `curl | bash` piping.

TESTS:
- Status: Adequate
- Coverage:
  - TestOSDetection: "detects macOS via uname -s returning Darwin" (line 97-108) -- verifies detect_os returns "darwin" for Darwin input
  - TestMacOSInstall (lines 644-715):
    - "it runs brew install leeovery/tools/tick when brew is available on macOS" -- uses createFakeBrew helper, checks brew log for correct command
    - "it exits 0 on successful Homebrew install" -- verifies exit code
    - "it prints a success message after Homebrew install" -- checks for "success" and "homebrew" in output
    - "it propagates exit code when brew install fails" -- uses "install-fail" fake brew behavior, verifies non-zero exit
    - "it does not run Linux download logic on macOS" -- verifies no "Downloading" in output
    - "it does not suppress brew output" -- verifies brew command output visible in script output
    - "it handles tick already installed via Homebrew (idempotent)" -- uses "already-installed" fake brew, verifies exit 0
  - Test infrastructure: createFakeBrew (lines 426-471) and runScriptWithFakeBrew (lines 474-489) are well-designed helpers that create fake brew scripts with configurable behavior and log invocations
  - Edge cases covered: install failure propagation, idempotent re-install, brew output passthrough, no Linux logic on macOS
  - Missing planned test: "propagates exit code when brew tap fails" -- not applicable since implementation uses single-command approach (brew install leeovery/tools/tick) rather than separate tap + install. This is acceptable.
- Notes: Tests are focused and not over-tested. Each test verifies a distinct acceptance criterion. The fake brew approach is a clean testing pattern for shell scripts tested from Go.

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib testing only, t.Run subtests, t.Helper on helpers, t.TempDir for isolation -- all consistent with CLAUDE.md patterns.
- SOLID principles: Good. install_macos is a single-responsibility function. The test mode dispatch (lines 100-138) cleanly separates concerns for unit-testing individual functions.
- Complexity: Low. The install_macos function is 9 lines with a single branch (brew available vs not). The main flow macOS branch is 3 lines.
- Modern idioms: Yes. Uses `command -v` for command detection (POSIX-portable), `&>` for redirection, set -euo pipefail for strict mode.
- Readability: Good. Function names are descriptive (install_macos, detect_os). The test names clearly describe what they verify. The createFakeBrew helper documents its behavior parameter.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The task description specifies `brew tap leeovery/tick && brew install tick` (two commands), but the implementation uses the combined `brew install leeovery/tools/tick` form. This is a positive drift -- the single-command approach is more idiomatic, simpler, and handles tap implicitly. The tap name also reflects the post-task migration to `homebrew-tools`. No action needed; just noting the divergence from the original task wording.
- The no-brew error message (line 62) also uses the combined form `brew install leeovery/tools/tick` rather than the two-step `brew tap ... && brew install tick` from the spec. This is consistent within the script and provides a simpler instruction to users.
