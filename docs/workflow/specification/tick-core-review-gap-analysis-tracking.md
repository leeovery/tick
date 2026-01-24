---
status: in-progress
created: 2026-01-24
phase: Gap Analysis
topic: tick-core
---

# Review Tracking: tick-core - Gap Analysis

## Findings

### 1. JSONL Example Contradicts Text

**Category**: Contradiction
**Affects**: JSONL Format section

**Details**:
Line 177 shows a minimal task example:
```jsonl
{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
```

But wait - this example DOES have `updated`. Let me re-check... Actually this is fine, the example includes updated. No contradiction.

**Resolution**: Skipped
**Notes**: False alarm - example is correct.

---

### 2. Status Transitions Undefined

**Category**: Insufficient Detail (Important)
**Affects**: Task Statuses section, individual command behaviors

**Details**:
The spec says "Only valid transitions enforced by commands" but doesn't define what transitions are valid:
- Can you `tick done` a task that's `open` (never started)?
- Can you `tick start` a task that's `done`?
- What transitions does each command perform?

An implementer would have to guess.

**Proposed Addition**: Status transition table or per-command transition rules

**Resolution**: Pending
**Notes**:

---

### 3. tick init Behavior Unclear

**Category**: Insufficient Detail (Important)
**Affects**: CLI Commands section

**Details**:
`tick init` is listed as "Initialize .tick/ directory in current project" but:
- What if `.tick/` already exists? Error? Silent success? Overwrite?
- What files are created? Just the directory, or also tasks.jsonl?
- What output does it produce on success?

**Proposed Addition**: Describe init behavior including existing directory handling

**Resolution**: Pending
**Notes**:

---

### 4. Mutation Command Output Not Specified

**Category**: Insufficient Detail (Important)
**Affects**: CLI Commands section, Output Formats section

**Details**:
Output formats are specified for list, show, stats. But what about:
- `tick create` - Does it output the new task ID? Full task details?
- `tick start/done/cancel/reopen` - Confirmation message? Task status?
- `tick dep add/rm` - Confirmation? Updated dependency list?

An agent needs to know what to expect from these commands.

**Proposed Addition**: Output specification for mutation commands

**Resolution**: Pending
**Notes**:

---

### 5. tick reopen Target Status Unclear

**Category**: Ambiguity (Minor)
**Affects**: CLI Commands section

**Details**:
`tick reopen` "reopens a closed task" but doesn't specify what status it sets:
- Does it set status to `open`?
- Does it set status to `in_progress` (resume where you left off)?
- Does it clear the `closed` timestamp?

**Proposed Addition**: Clarify reopen behavior

**Resolution**: Pending
**Notes**:
