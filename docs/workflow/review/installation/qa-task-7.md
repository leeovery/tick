TASK: macOS Install Script: No Homebrew Error Path (installation-2-3)

ACCEPTANCE CRITERIA:
- On macOS (Darwin) without `brew` in PATH, the script exits with code 1
- The error message includes `Please install via Homebrew:` and `brew tap leeovery/tick && brew install tick`
- The error message is printed to stderr
- No binary download is attempted (no GitHub API call, no temp directory, no file operations)
- The script does not fall through to the Linux download path
- The script does not prompt for user input or assume a TTY

STATUS: Complete

SPEC CONTEXT:
The specification defines that on macOS without Homebrew, the install script should exit with the message:
```
Please install via Homebrew:
brew tap {owner}/tick && brew install tick
```
The rationale is that macOS does not handle direct binary downloads to avoid code signing complexity -- Homebrew handles signing automatically. The design principle of "simple: no complex fallback chains" reinforces this.

IMPLEMENTATION:
- Status: Implemented (with minor intentional drift from plan wording)
- Location: /Users/leeovery/Code/tick/scripts/install.sh:59-64
- Notes:
  The `install_macos()` function (lines 59-68) checks `command -v brew` and, when brew is not found, prints the error message to stderr and returns 1. The implementation uses `brew install leeovery/tools/tick` rather than the plan's `brew tap leeovery/tick && brew install tick`. This is intentional -- the project migrated from the `homebrew-tick` tap to `homebrew-tools` (see git commit c4a2a84 "Migrate Homebrew TAP from homebrew-tick to homebrew-tools"), and the shorthand `brew install leeovery/tools/tick` auto-taps. This is reflected consistently across CLAUDE.md, README.md, the release workflow, and all tests.

  The flow for macOS without brew:
  1. `detect_os()` returns "darwin" (line 15)
  2. Main flow dispatches to `install_macos()` (lines 146-149)
  3. `command -v brew` fails (line 60)
  4. Two stderr lines printed: "Please install via Homebrew:" and "brew install leeovery/tools/tick" (lines 61-62)
  5. Returns 1 (line 63), which propagates to exit 1
  6. No download, temp dir, or other operations occur

  The test mode dispatch also supports direct `install_macos` invocation (lines 127-129), enabling isolated testing of this function.

TESTS:
- Status: Adequate
- Coverage: All six planned test cases are implemented in `TestMacOSNoBrewError` (/Users/leeovery/Code/tick/scripts/install_test.go:734-818)
  1. "it exits with code 1 on macOS when brew is not found" (line 743) -- verifies exit code is exactly 1 via ExitError.ExitCode()
  2. "it prints the Homebrew install instructions on macOS without brew" (line 757) -- checks for `brew install leeovery/tools/tick` in output
  3. "it prints the Please install via Homebrew message" (line 767) -- checks for exact "Please install via Homebrew:" text
  4. "it does not attempt a binary download on macOS without brew" (line 777) -- checks absence of "Downloading" and "curl" in output
  5. "it does not create a temporary directory on macOS without brew" (line 790) -- checks absence of "TICK_TMPDIR=" and "mktemp" in output
  6. "it outputs the error message to stderr not stdout" (line 803) -- uses `runScriptSeparateOutputs` to capture stdout/stderr independently; verifies message appears on stderr and not stdout

- Notes:
  Tests are well-structured. The `noBrew` environment map is shared across all subtests (DRY). The `runScriptSeparateOutputs` helper (lines 719-732) is purpose-built for the stderr/stdout separation test without over-engineering. Each test verifies one specific behavior. The tests would fail if the feature broke (e.g., if someone removed the `return 1`, test 1 would fail; if the message changed, tests 2-3 would fail; if stderr was switched to stdout, test 6 would fail).

  No over-testing concerns: the six tests each verify a distinct aspect of the error path. No redundant assertions.

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on helpers. No testify. Test names are descriptive sentences.
- SOLID principles: Good. The `install_macos()` function has a single responsibility (macOS installation logic including the error path). The function is cleanly separated from OS detection and the main install flow.
- Complexity: Low. The no-brew error path is 4 lines of shell code: if-check, two echo statements, return. Straightforward.
- Modern idioms: Yes. Uses `command -v` (POSIX) instead of `which`. Uses `>&2` for stderr. Uses `return 1` within a function (not `exit 1`) which is correct since the function is called within a `set -e` script.
- Readability: Good. The error message is clear and actionable. The code flow is obvious. Test names describe expected behavior.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The plan's acceptance criteria text references `brew tap leeovery/tick && brew install tick`, but the implementation uses `brew install leeovery/tools/tick`. This is an intentional project-wide migration (commit c4a2a84) and is consistently reflected across all project files. The plan task text was not retroactively updated to match. This is cosmetic drift in documentation, not a functional issue.
