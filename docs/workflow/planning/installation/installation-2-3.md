---
id: installation-2-3
phase: 2
status: pending
created: 2026-01-31
---

# macOS Install Script: No Homebrew Error Path

## Goal

Task 2-2 adds macOS detection and Homebrew delegation to `scripts/install.sh`. However, if the user is on macOS and does not have Homebrew installed, the script must not silently fail or attempt a direct binary download (which would bypass code signing). This task implements the explicit error path: when `uname -s` returns `Darwin` and the `brew` command is not found, the script exits with a clear, instructive error message directing the user to install via Homebrew.

## Implementation

1. **Locate the macOS branch** in `scripts/install.sh`. Task 2-2 adds a `Darwin` case to the OS detection logic. Within that branch, after the check for `command -v brew`, there should be an `else` path for when Homebrew is not found. This task implements that `else` path.

2. **Print the instructive error message** when `brew` is not available on macOS. The message must match the specification:
   ```
   Please install via Homebrew:
   brew tap leeovery/tick && brew install tick
   ```
   Output this to stderr (using `>&2`) so it is visible even when stdout is redirected or piped.

3. **Exit with code 1** immediately after printing the error message. The script must not continue to the Linux download path or attempt any other installation method.

4. **Ensure the message is the complete output** — the two-line message should be printed as-is. To maintain consistency with the rest of the script's error messaging style (established in task 1-4), prefixing with `"Error: "` on the first line is acceptable as long as the Homebrew command is clearly shown. The specification's exact wording takes precedence if there is ambiguity.

5. **Verify the flow** — on macOS without Homebrew, the script should:
   - Detect `Darwin` from `uname -s`
   - Check for `brew` command existence
   - Not find `brew`
   - Print the error message to stderr
   - Exit with code 1
   - Not attempt any download, temp directory creation, or binary installation

## Tests

- `"it exits with code 1 on macOS when brew is not found"` — mock `uname -s` to return `Darwin` and ensure `brew` is not on PATH; verify the script exits with code 1
- `"it prints the Homebrew install instructions on macOS without brew"` — mock `uname -s` to return `Darwin` without `brew` available; verify stderr output contains `brew tap leeovery/tick && brew install tick`
- `"it prints the 'Please install via Homebrew' message"` — verify stderr output contains the line `Please install via Homebrew:` (matching specification wording)
- `"it does not attempt a binary download on macOS without brew"` — mock `uname -s` to return `Darwin` without `brew`; verify no curl download calls are made (no temp directory created, no GitHub release API queried for binary download)
- `"it does not create a temporary directory on macOS without brew"` — verify that the temp directory creation and trap cleanup code paths are never reached when exiting at the no-Homebrew error
- `"it outputs the error message to stderr not stdout"` — capture stdout and stderr separately; verify the instructive message appears on stderr and stdout is empty

## Edge Cases

No additional edge cases identified for this task. The behavior is a straightforward error-and-exit path with no fallback logic or conditional branching beyond the Homebrew existence check already handled by task 2-2's flow structure.

## Acceptance Criteria

- [ ] On macOS (Darwin) without `brew` in PATH, the script exits with code 1
- [ ] The error message includes `Please install via Homebrew:` and `brew tap leeovery/tick && brew install tick`
- [ ] The error message is printed to stderr
- [ ] No binary download is attempted (no GitHub API call, no temp directory, no file operations)
- [ ] The script does not fall through to the Linux download path
- [ ] The script does not prompt for user input or assume a TTY

## Context

The specification explicitly defines the macOS no-Homebrew behavior:

> **No Homebrew**: Exit with message:
> ```
> Please install via Homebrew:
> brew tap {owner}/tick && brew install tick
> ```

The rationale for not supporting direct binary download on macOS is code signing complexity — Homebrew handles signing automatically. The specification's design principle of "simple: no complex fallback chains" reinforces this: rather than attempting to download a binary that may be blocked by macOS Gatekeeper, the script gives the user a clear path forward.

This task is intentionally small and focused. The branching structure (macOS detection, brew check) is established by task 2-2. This task fills in the error case within that structure. Separating it keeps task 2-2 focused on the happy path and keeps this error path independently testable.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
