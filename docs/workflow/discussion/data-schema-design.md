# Discussion: Data Schema Design

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick uses JSONL as the source of truth (committed to git) and SQLite as an ephemeral cache (gitignored, auto-rebuilds). The research phase proposed schemas for both, but several questions need resolution before specification.

**Key constraint**: The schema must support the CLI decisions already made:
- 4 statuses: `open`, `in_progress`, `done`, `cancelled`
- Dependencies via `blocked_by` array
- Hierarchy via `parent` field
- Output includes related entity context (not just IDs)

### References

- [Research: exploration.md](../research/exploration.md) (lines 62-138) - Proposed schemas
- [CLI Command Structure & UX](cli-command-structure-ux.md) - Status values, commands
- [TOON Output Format](toon-output-format.md) - Output structure decisions

### Proposed Schema (from research)

**Task fields**:
```
id: string              # tick-a1b2c3 (prefix + hash)
title: string
status: "open" | "in_progress" | "done"  # Now 4 values per CLI discussion
priority: 0 | 1 | 2 | 3 | 4
type: "epic" | "task" | "bug" | "spike"

# Optional
description?: string
blocked_by?: string[]
parent?: string
assignee?: string
labels?: string[]
created: string         # ISO 8601
updated?: string
closed?: string
notes?: string
```

## Questions

- [x] Are all proposed fields necessary? Should any be removed or added?
- [ ] What are the exact constraints and validation rules for each field?
- [ ] Is the SQLite schema optimal for expected queries?
- [ ] How should the `ready_tasks` view handle edge cases?
- [ ] Should `cancelled` tasks behave differently from `done` in queries?

---

## Q1: Field Selection - What's Necessary?

### Context

Research proposed 15 fields. Need to determine minimal viable schema without over-engineering.

### Fields Questioned

**`type` (epic|task|bug|spike)**
- Research assumed this distinguishes "big things" (epics) from "small things" (tasks)
- But hierarchy (`parent`) already provides this - a task with children is implicitly an "epic"
- Agent running `tick ready` works on leaf tasks; hierarchy provides context

**`assignee`**
- Research states "single-agent focus"
- If only one agent, who gets assigned? No value.

**`labels`**
- Flexible tagging adds complexity
- Priority already handles urgency; type was handling categorization
- Can add later if needed

**`notes` vs `description`**
- Two free-text fields with unclear distinction
- Research: `description` for task details, `notes` for "free-form context for agents"
- Overlap is significant - one field sufficient

**`updated`**
- Tracks last modification time
- Minimal overhead - already updating the record anyway

### Decision

**Remove**: `type`, `assignee`, `labels`, `notes`
**Keep**: `updated`, `description`

**Rationale**:
- **Type removed**: Hierarchy handles epic/task distinction. A task with children is effectively an epic. Agent instructions explain navigation. Keeps schema simpler.
- **Assignee removed**: Single-agent focus. No multi-agent coordination planned.
- **Labels removed**: YAGNI. Priority covers urgency. Can add later if needed.
- **Notes removed**: `description` is sufficient for free-form context. One field, not two.
- **Updated kept**: Useful for debugging/audit. Overhead negligible since we're mutating anyway.

### Resulting Schema

```
id: string              # tick-a1b2c3 (prefix + hash)
title: string
status: "open" | "in_progress" | "done" | "cancelled"
priority: 0 | 1 | 2 | 3 | 4
description?: string
blocked_by?: string[]
parent?: string
created: string         # ISO 8601
updated?: string
closed?: string
```

**10 fields** (down from 15). Minimal, focused.

---

## Q2: Hierarchy and `tick ready` Behavior

### Context

With `type` removed, hierarchy does more work. Need to clarify exactly how `tick ready` behaves with parent/child relationships.

### The Scenario

```
Epic: Auth System (tick-a1b2)
├── Setup Sanctum (tick-c3d4) - DONE
├── Login endpoint (tick-e5f6, blocked_by=[tick-c3d4])
│     ├── Validation (tick-g7h8)
│     └── Rate limiting (tick-i9j0)
└── Logout endpoint (tick-k1l2, blocked_by=[tick-e5f6])
```

Setup Sanctum is done. Login endpoint is unblocked. But Login endpoint has children.

**Question**: Should `tick ready` return:
- A) Login endpoint (the unblocked parent)
- B) Validation and Rate limiting (the leaf children)
- C) All three (parent + children)

### Options Considered

**Option A: Parent appears, agent decides**
- `tick ready` returns Login endpoint
- Agent sees it has children via `tick show`, works on them
- Pro: Simple query logic
- Con: Agent must do extra lookup to find actual work

**Option B: Only leaf tasks appear**
- Tasks with open children are implicitly "not ready"
- `tick ready` returns only Validation and Rate limiting
- Pro: Ready list = actual work items, no extra lookups
- Con: Query more complex (must check for children)

**Option C: Return all**
- Both parent and children appear
- Pro: Full visibility
- Con: Cluttered, agent may start parent when children need work first

---

