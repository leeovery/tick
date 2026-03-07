---
id: installation-2-2
phase: 2
status: completed
created: 2026-01-31
---

# macOS Install Script: Homebrew Delegation

## Goal

The install script created in Phase 1 (task 1-4) handles Linux but explicitly rejects macOS with a placeholder error message. The specification requires that on macOS, the install script delegates to Homebrew — running `brew tap leeovery/tick && brew install tick` — so that macOS users who run the `curl | bash` install command get tick installed via the Homebrew path automatically. This avoids code signing complexity (Homebrew handles signing) and keeps the install experience seamless. Task 2-1 established the Homebrew formula; this task wires the install script to use it.

## Implementation

1. **Modify `scripts/install.sh`** to add macOS detection in the OS case block. In task 1-4, the script detects OS via `uname -s` and rejects anything that is not `Linux`. Replace the `Darwin` rejection with a macOS handler.

2. **Add macOS handling** that runs when `uname -s` returns `Darwin`:

   - Check whether the `brew` command is available using `command -v brew >/dev/null 2>&1`.
   - If `brew` is found, proceed with Homebrew delegation (step 3).
   - If `brew` is not found, this task does NOT handle that path — task 2-3 will implement the no-Homebrew error message.

3. **Implement Homebrew delegation** when `brew` is available:

   - Run: `brew tap leeovery/tick`
   - If the tap command succeeds, run: `brew install tick`
   - Use `&&` chaining or explicit exit code checks so that a failure in either command propagates the exit code to the caller.
   - **Do not suppress stderr/stdout** from brew commands — the user should see Homebrew's normal output (download progress, install confirmation).
   - After successful install, print a success message (e.g., `"tick installed successfully via Homebrew."`) and exit 0.

4. **Handle idempotent re-install** (tick already installed via Homebrew):

   - `brew install tick` when tick is already installed prints a message like "Warning: tick is already installed" and exits 0. This is acceptable behavior — no special handling needed.
   - Stick with `brew install tick` (not `brew reinstall`) as the spec defines. Users who want to update use `brew upgrade tick` per the specification's update table.

5. **Ensure the macOS path skips all Linux-specific logic**: The detect-architecture, download-binary, and install-to-path logic from the Linux path must not execute when on macOS. The OS detection should branch early — either via a function call or an early return/exit after the Homebrew delegation completes.

## Tests

- `"it detects macOS via uname -s returning Darwin"` — mock `uname -s` to return `Darwin` and verify the script enters the macOS code path (does not attempt Linux binary download)
- `"it runs brew tap leeovery/tick when brew is available on macOS"` — mock `uname -s` as `Darwin`, mock `brew` as available, and verify `brew tap leeovery/tick` is executed
- `"it runs brew install tick after successful tap"` — mock `uname -s` as `Darwin`, mock `brew` as available, mock `brew tap` succeeding, and verify `brew install tick` is executed
- `"it exits 0 on successful Homebrew install"` — mock the full happy path (Darwin, brew available, tap succeeds, install succeeds) and verify exit code 0
- `"it prints a success message after Homebrew install"` — verify output contains a message indicating tick was installed via Homebrew
- `"it propagates exit code when brew tap fails"` — mock `brew tap leeovery/tick` returning exit code 1, verify the script exits with a non-zero code and does not attempt `brew install`
- `"it propagates exit code when brew install fails"` — mock `brew tap` succeeding but `brew install tick` returning exit code 1, verify the script exits with a non-zero code
- `"it does not run Linux download logic on macOS"` — mock `uname -s` as `Darwin` with brew available, verify no GitHub API call or binary download is attempted
- `"it handles tick already installed via Homebrew (idempotent)"` — mock `brew install tick` outputting an "already installed" warning and exiting 0, verify the script exits 0 without error
- `"it does not suppress brew output"` — mock brew commands and verify their stdout/stderr are passed through to the user (not redirected to /dev/null)

## Edge Cases

- **brew tap or brew install failure should propagate exit code**: If `brew tap leeovery/tick` fails (e.g., network error, invalid tap URL), the script must not swallow the error and must not proceed to `brew install`. The exit code from the failing brew command must propagate to the script's exit code. Similarly, if the tap succeeds but `brew install tick` fails (e.g., formula error, download failure), that exit code must propagate. This is critical when the script is run via `curl | bash` in CI — the calling process needs to detect failures.

- **tick already installed via Homebrew (idempotent re-install)**: The specification mandates idempotent behavior ("Safe to run multiple times"). When a user re-runs the install script on macOS and tick is already installed via Homebrew, `brew install tick` exits 0 with a warning message. This is the desired behavior — no special detection or `brew reinstall` logic is needed. The script should not treat an "already installed" message as an error.

## Acceptance Criteria

- [ ] `scripts/install.sh` detects macOS via `uname -s` returning `Darwin` and enters the Homebrew delegation path
- [ ] When `brew` is available on macOS, the script runs `brew tap leeovery/tick` followed by `brew install tick`
- [ ] A failure in `brew tap` prevents `brew install` from running and causes the script to exit non-zero
- [ ] A failure in `brew install` causes the script to exit non-zero
- [ ] On successful Homebrew install, the script prints a success message and exits 0
- [ ] The macOS path does not execute any Linux-specific logic (no architecture detection, no binary download, no install location logic)
- [ ] Re-running the script when tick is already installed via Homebrew completes without error (idempotent)
- [ ] brew command output (stdout/stderr) is visible to the user — not suppressed
- [ ] Script continues to work correctly when piped via `curl -fsSL ... | bash`

## Context

The specification defines macOS install script behavior as:

> ### macOS Behavior
>
> 1. **Check for Homebrew**: If `brew` command exists
>    - Run `brew tap {owner}/tick && brew install tick`
>    - Exit successfully
> 2. **No Homebrew**: Exit with message:
>    ```
>    Please install via Homebrew:
>    brew tap {owner}/tick && brew install tick
>    ```

This task implements step 1 only (Homebrew present). Task 2-3 implements step 2 (no Homebrew error path).

The specification's design principles emphasize:
- **Simple**: No complex fallback chains — on macOS, either Homebrew works or it does not.
- **Idempotent**: Safe to run multiple times.
- **Fast**: Important for ephemeral environments.

The `{owner}` placeholder is `leeovery` (matching the repository owner established in task 1-4's constants).

macOS does not handle direct binary downloads per the specification: "This avoids code signing complexity — Homebrew handles signing automatically."

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
