---
id: tick-core-6-5
phase: 6
status: completed
created: 2026-02-09
---

# Remove doctor command from help text

**Problem**: The help text in `printUsage` (cli.go line 203) advertises `tick doctor - Run diagnostics and validation` but the `commands` map has no "doctor" entry. Running `tick doctor` produces "Error: Unknown command 'doctor'". The spec lists `tick doctor` for specific diagnostics (orphaned children, parent-done-before-children), but the command is not implemented. This misleads users and agents that parse help output.

**Solution**: Remove the doctor entry from the help text. If doctor is implemented in a future cycle, the help text can be re-added at that time.

**Outcome**: The help text only advertises commands that are actually implemented.

**Do**:
1. In `internal/cli/cli.go`, remove the line `fmt.Fprintln(w, "  doctor    Run diagnostics and validation")` from `printUsage`.
2. Verify `tick help` output no longer mentions doctor.

**Acceptance Criteria**:
- `tick help` does not list the doctor command
- No functional commands are affected
- If a help output test exists, update it to remove the doctor line

**Tests**:
- Verify `tick help` output does not contain "doctor"
- Verify `tick doctor` still returns "Unknown command" error (unchanged behavior)
