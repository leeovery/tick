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

**Proposed Addition**: Status transition table

**Resolution**: Approved
**Notes**: Added transition table after Task Statuses section. Allows skipping start (can done an open task directly).

---

### 3. tick init Behavior Unclear

**Category**: Insufficient Detail (Important)
**Affects**: CLI Commands section

**Details**:
`tick init` is listed as "Initialize .tick/ directory in current project" but:
- What if `.tick/` already exists? Error? Silent success? Overwrite?
- What files are created? Just the directory, or also tasks.jsonl?
- What output does it produce on success?

**Proposed Addition**: Init Command section with behavior details

**Resolution**: Approved
**Notes**: Added after Command Reference table.

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

**Proposed Addition**: Mutation Command Output section

**Resolution**: Approved
**Notes**: Added after Rebuild Command section. Covers create, update, start/done/cancel/reopen, dep add/rm with --quiet behavior.

---

### 5. tick reopen Target Status Unclear

**Category**: Ambiguity (Minor)
**Affects**: CLI Commands section

**Details**:
`tick reopen` "reopens a closed task" but doesn't specify what status it sets:
- Does it set status to `open`?
- Does it set status to `in_progress` (resume where you left off)?
- Does it clear the `closed` timestamp?

**Proposed Addition**: Note that reopen clears closed timestamp

**Resolution**: Approved
**Notes**: Added note after Status Transitions table. Also added --blocks option to create/update during this discussion (inverse of --blocked-by).

---

### 6. No Update Command for Task Fields

**Category**: Gap (Critical)
**Affects**: CLI Commands section

**Details**:
There is NO command to modify task fields after creation. The only modifications possible are:
- Status changes (`start`, `done`, `cancel`, `reopen`)
- Dependencies (`dep add/rm`)

There's no way to change `title`, `description`, `priority`, or `parent` after a task is created. This is a significant gap - users would need to cancel and recreate tasks to fix typos or adjust priorities.

**Proposed Addition**: Add `tick update` command

**Resolution**: Approved
**Notes**: Discovered during #4 discussion. Added to Command Reference and full Update Command section with options.
