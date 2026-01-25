---
status: in-progress
created: 2026-01-25
phase: Input Review
topic: tick-core
---

# Review Tracking: tick-core - Input Review

## Findings

### 1. Short alias `tk` not documented

**Source**: cli-command-structure-ux.md, line 26
**Category**: Enhancement to existing topic
**Affects**: CLI Commands section

**Details**:
The CLI discussion mentions: "Short alias: `tk` works as alternative to `tick`"

This alias is not documented in the specification. If the alias is still intended, it should be added to the CLI section.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 2. Human-friendly error suggestions not specified

**Source**: cli-command-structure-ux.md, lines 216-221
**Category**: Enhancement to existing topic
**Affects**: Error Handling section

**Details**:
The CLI discussion shows human-friendly error output with suggestions:
```
Error: Task 'tick-xyz123' not found

Did you mean?
  tick-xyz124  Setup authentication
```

The spec only says "Plain text format (human-readable)" without specifying this suggestion feature. Should human errors include "Did you mean?" suggestions for typos/close matches?

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 3. Collision error message references archive (deferred feature)

**Source**: id-format-implementation.md, line 164
**Category**: Gap/Ambiguity
**Affects**: ID Generation section

**Details**:
The discussion suggests collision error message: "failed to generate unique ID after 5 attempts - consider archiving completed tasks"

But archive is explicitly a non-goal for v1. The spec correctly omits the archive reference but uses: "task list may be too large"

This is actually CORRECT in the spec - just noting the spec properly adapted the discussion to align with non-goals.

**Proposed Addition**:
None needed - spec is correct.

**Resolution**: Skipped
**Notes**: Spec correctly adapted error message to remove reference to deferred archive feature.

---

### 4. `update` command not explicitly discussed

**Source**: Specification analysis (not in discussions)
**Category**: Gap/Ambiguity
**Affects**: CLI Commands section

**Details**:
The spec includes a `tick update` command for modifying task fields (title, description, priority, parent). This command is not explicitly discussed in cli-command-structure-ux.md, which focuses on status transitions (start/done/cancel/reopen) and dependency management.

The update command is a logical necessity - there needs to be a way to modify task fields after creation. The spec's design seems reasonable.

**Proposed Addition**:
None needed - spec addition is appropriate.

**Resolution**: Skipped
**Notes**: Reasonable spec enhancement filling a gap in discussions.

---

### 5. Stats command output not explicitly discussed

**Source**: Specification analysis (not in discussions)
**Category**: Gap/Ambiguity
**Affects**: Output Formats section

**Details**:
The spec defines detailed stats output (TOON and human-readable formats) showing total, open, in_progress, done, cancelled, ready, blocked counts plus breakdown by priority.

None of the discussions explicitly define what `tick stats` should output. The spec's design seems reasonable but wasn't validated through discussion.

**Proposed Addition**:
None needed - spec design is reasonable.

**Resolution**: Skipped
**Notes**: Spec fills gap with reasonable design. May want user confirmation the stats fields are correct.

---

### 6. Tab completion mentioned but not in spec

**Source**: tui.md, line 196
**Category**: Enhancement to existing topic
**Affects**: CLI Commands section (or could be Notes)

**Details**:
tui.md mentions: "Tab completion via Cobra (shell-level, not app-level)"

This is not mentioned in the specification. Tab completion is typically an implementation detail handled by the CLI framework, so omission may be intentional.

**Proposed Addition**:
(pending discussion - may be implementation detail)

**Resolution**: Pending
**Notes**:

---

### 7. TOON error format conflict between discussions

**Source**: cli-command-structure-ux.md lines 206-215 vs toon-output-format.md lines 230-246
**Category**: Gap/Ambiguity (resolved conflict)
**Affects**: Error Handling section

**Details**:
CLI discussion initially proposed structured TOON errors for agents:
```
| Agent via pipe | Structured (TOON) with error code, message, context |
```

TOON discussion later decided plain text errors for all:
```
**Option A: Plain text errors to stderr**
```

The spec follows the TOON discussion decision (plain text). This is the correct resolution - the later, more focused discussion on output formats should take precedence.

**Proposed Addition**:
None needed - spec correctly resolved the conflict.

**Resolution**: Skipped
**Notes**: Spec follows the TOON discussion decision, which is appropriate.

---
