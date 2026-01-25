---
status: in-progress
created: 2026-01-25
phase: Gap Analysis
topic: tick-core
---

# Review Tracking: tick-core - Gap Analysis

## Findings

### 1. TOON escaping rules not specified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Output Formats section
**Priority**: Critical

**Details**:
The TOON format uses commas as delimiters. The spec shows:
```
tasks[2]{id,title,status,priority}:
  tick-a1b2,Setup Sanctum,done,1
```

What happens if a title contains a comma? E.g., "Fix bug, deploy changes"
- Does it become: `tick-a1b2,Fix bug, deploy changes,open,1` (parsing breaks)
- Or is there escaping: `tick-a1b2,"Fix bug, deploy changes",open,1`

This is critical for agents to parse output correctly.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 2. "Blocked" query definition missing

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: CLI Commands section
**Priority**: Important

**Details**:
The spec says `tick blocked` is an alias for `list --blocked`. The "ready" query is clearly defined (open + unblocked + no open children).

But what makes a task "blocked"? Is it:
- (a) Any task with incomplete `blocked_by` dependencies?
- (b) Any task with open children (even if no blocked_by)?
- (c) Both (a) and (b)?

A task could be "not ready" for different reasons. The implementer needs to know exactly what "blocked" means.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 3. Empty state outputs not specified

**Source**: Specification analysis
**Category**: Insufficient Detail
**Affects**: Output Formats section
**Priority**: Important

**Details**:
What does output look like when:
- `tick list` with no tasks
- `tick ready` with nothing ready
- `tick blocked` with nothing blocked

For TOON: `tasks[0]{id,title,status,priority}:` with no rows?
For human: Just headers with no data? Or a message like "No tasks found"?

Agents need to know what "empty" looks like to handle it correctly.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 4. Lock timeout/failure behavior not specified

**Source**: Specification analysis
**Category**: Insufficient Detail
**Affects**: Synchronization section
**Priority**: Important

**Details**:
File locking is well-specified (exclusive for writes, shared for reads). But:
- What if the lock can't be acquired (e.g., another process holds it)?
- Is there a timeout? How long to wait?
- What error message is shown?

This affects reliability in concurrent environments.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 5. Timestamp timezone not specified

**Source**: Specification analysis
**Category**: Ambiguity
**Affects**: Task Schema section
**Priority**: Important

**Details**:
The spec says "ISO 8601 timestamp" for created/updated/closed. Examples show UTC (`2026-01-19T10:00:00Z`).

But is UTC required? Or can implementations use local time with offset (`2026-01-19T10:00:00-05:00`)?

This affects:
- Sorting/filtering by date
- Display formatting
- Consistency across machines in different timezones

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 6. `tick doctor` checks not fully enumerated

**Source**: Specification analysis
**Category**: Insufficient Detail
**Affects**: CLI Commands section
**Priority**: Minor

**Details**:
The spec mentions two doctor checks:
- "tick-child references non-existent parent tick-deleted"
- "tick-epic is done but has open children"

But doctor is a diagnostic command that should validate data integrity. What other checks should it perform?
- Orphaned dependencies (blocked_by references non-existent task)?
- Invalid status values?
- Invalid priority values?
- Duplicate IDs?
- Cycle detection?

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---

### 7. Title/description validation rules not specified

**Source**: Specification analysis
**Category**: Insufficient Detail
**Affects**: Task Schema section
**Priority**: Minor

**Details**:
The spec says title is "required" and "non-empty string". But:
- Is there a max length? (What if someone passes a 10MB title?)
- Can title contain newlines?
- Can description contain any character?

Practical limits help implementers and prevent abuse.

**Proposed Addition**:
(pending discussion)

**Resolution**: Pending
**Notes**:

---
