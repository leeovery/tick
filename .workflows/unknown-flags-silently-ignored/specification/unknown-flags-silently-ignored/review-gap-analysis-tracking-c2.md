---
status: in-progress
created: 2026-03-10
cycle: 2
phase: Gap Analysis
topic: unknown-flags-silently-ignored
---

# Review Tracking: unknown-flags-silently-ignored - Gap Analysis

## Findings

### 1. Two-Level Commands section still references `dep add/rm`

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Design > Two-Level Commands

**Details**:
The Two-Level Commands section (line 49) says `dep add/rm` but the Normalize Dep Subcommand section specifies renaming `dep rm` to `dep remove`, and the Command Flag Inventory table already uses `dep remove`. An implementer reading the Two-Level Commands section in isolation would see the old name.

**Proposed Addition**:
Change `dep add/rm` to `dep add/remove` in the Two-Level Commands section.

**Resolution**: Pending
**Notes**:

---

### 2. Error format template inconsistent with example for two-level commands

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Error Behavior

**Details**:
The error message template uses `{command}` in two positions:
```
Error: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.
```

The example for `dep add` shows different values in each position:
```
Error: unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.
```

The first `{command}` is `dep add` (fully-qualified) but the second is `dep` (parent only). This makes sense because `tick help dep add` is not a valid invocation (the help system registers `dep` as a single command), but the template implies the same value in both positions. An implementer would need to decide what logic to use for the help reference in two-level commands without clear guidance from the template.

**Proposed Addition**:
Either adjust the template to use distinct placeholders (e.g., `{command}` and `{help-command}`), or add a note below the template explaining that for two-level commands the help reference uses the parent command name since `tick help` only accepts top-level command names.

**Resolution**: Pending
**Notes**:
