---
status: complete
created: 2026-03-10
cycle: 1
phase: Traceability Review
topic: Unknown Flags Silently Ignored
---

# Review Tracking: Unknown Flags Silently Ignored - Traceability

## Findings

### 1. Edge case and test for value-taking flag at end of args not in specification

**Type**: Hallucinated content
**Spec Reference**: N/A
**Plan Reference**: Phase 1 / tick-3abf54 (Validate global flag pass-through and value-taking flag skipping)
**Change Type**: update-task

**Details**:
Task tick-3abf54 includes an edge case, acceptance criterion, and test for the scenario where a value-taking flag appears at the end of args with no following value (e.g., `--priority` with nothing after it). The task specifies the validator should return nil and let the command parser handle the "requires a value" error. The specification does not address this scenario -- it only describes the case where a value-taking flag has a value to skip ("given `--priority 3 --unknown`, the validator must know `--priority` takes a value so `3` is skipped"). The behavior for a missing value is an implementation decision not discussed or validated in the specification.

This appears in three places in the task:
- Edge Cases: "Value-taking flag at end of args (missing value): validator should not panic. Command parser handles the 'requires a value' error."
- Acceptance Criteria: "Value-taking flag at end of args (missing value) does not crash the validator"
- Tests: `"value-taking flag at end of args does not panic"`

**Current**:
Edge Cases:
- --ready on tick ready / --blocked on tick blocked: not in those commands' flag sets per spec. Validator should reject them.
- Global flags interspersed with command args: tick create --json "My task" --priority 1 --verbose should work.
- Value-taking flag at end of args (missing value): validator should not panic. Command parser handles the "requires a value" error.
- Value that looks like a flag: --description --not-a-flag should treat --not-a-flag as value of --description, not as unknown flag.

Acceptance Criteria:
- Every command with flags accepts all its valid flags through ValidateFlags
- Every command rejects at least one unknown flag through ValidateFlags
- ready rejects --ready and blocked rejects --blocked
- Global flags (--quiet, -q, --verbose, -v, --toon, --pretty, --json, --help, -h) pass through on every command
- Value-taking flags properly skip their value argument, including when value looks like a flag
- Value-taking flag at end of args (missing value) does not crash the validator
- The original bug report (dep add --blocks) is tested end-to-end and produces exact error message from spec

Tests:
- "it returns nil for args with no flags"
- "it returns nil for args with known command flags"
- "it returns error for unknown flag"
- "it skips global flags without error"
- "it skips value after value-taking flag"
- "it does not skip value after boolean flag"
- "it rejects short unknown flags"
- "it accepts -f on remove"
- "it uses parent command in help hint for two-level commands"
- "it returns error for flag on command with no flags"
- "it handles global flags interspersed with command args"
- "global flags are accepted on any command"
- "global flags mixed with command flags"
- "value-taking flag at end of args does not panic"
- "value that looks like a flag is skipped"
- "consecutive value-taking flags work correctly"
- "it rejects dep add --blocks (original bug report)"
- TestFlagValidationAllCommands subtests per command: "accepts all valid flags" and "rejects unknown flag"
- ready rejects --ready
- blocked rejects --blocked

**Proposed**:
Edge Cases:
- --ready on tick ready / --blocked on tick blocked: not in those commands' flag sets per spec. Validator should reject them.
- Global flags interspersed with command args: tick create --json "My task" --priority 1 --verbose should work.
- Value that looks like a flag: --description --not-a-flag should treat --not-a-flag as value of --description, not as unknown flag.

Acceptance Criteria:
- Every command with flags accepts all its valid flags through ValidateFlags
- Every command rejects at least one unknown flag through ValidateFlags
- ready rejects --ready and blocked rejects --blocked
- Global flags (--quiet, -q, --verbose, -v, --toon, --pretty, --json, --help, -h) pass through on every command
- Value-taking flags properly skip their value argument, including when value looks like a flag
- The original bug report (dep add --blocks) is tested end-to-end and produces exact error message from spec

Tests:
- "it returns nil for args with no flags"
- "it returns nil for args with known command flags"
- "it returns error for unknown flag"
- "it skips global flags without error"
- "it skips value after value-taking flag"
- "it does not skip value after boolean flag"
- "it rejects short unknown flags"
- "it accepts -f on remove"
- "it uses parent command in help hint for two-level commands"
- "it returns error for flag on command with no flags"
- "it handles global flags interspersed with command args"
- "global flags are accepted on any command"
- "global flags mixed with command flags"
- "value that looks like a flag is skipped"
- "consecutive value-taking flags work correctly"
- "it rejects dep add --blocks (original bug report)"
- TestFlagValidationAllCommands subtests per command: "accepts all valid flags" and "rejects unknown flag"
- ready rejects --ready
- blocked rejects --blocked

**Resolution**: Fixed
**Notes**: Minor finding. The edge case is a reasonable defensive programming concern but cannot be traced to the specification. The implementer may naturally handle this anyway, but the plan should only contain spec-derived content. Removing three lines (one edge case, one acceptance criterion, one test) from the task.
