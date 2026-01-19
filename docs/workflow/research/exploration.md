# Tick Research Exploration

## What We're Exploring

**Tick** - A minimal, deterministic task tracker for AI coding agents.

Core thesis: Existing tools (Beads, br, Backlog.md) are either too complex or lack deterministic querying. Tick aims for zero-friction git integration with JSONL as source of truth and SQLite as auto-rebuilding cache.

## Key Architecture Decisions (Already Made)

1. **JSONL as source of truth** - Committed, human-readable, git-friendly diffs
2. **SQLite as disposable cache** - Auto-rebuilds from JSONL on staleness
3. **Dual write on mutations** - No sync commands ever
4. **No hooks, no daemons** - Just a CLI
5. **Single-agent focus** - Multi-agent is explicitly out of scope

## Open Questions to Explore

1. **ID format** - Sequential (`TICK-001`) vs hash-based (`tick-abc123`)?
2. **Subtasks** - Flat with parent reference vs nested structure?
3. **Archive strategy** - Separate file for done tasks vs keep in main?
4. **Config file** - Worth the complexity?
5. **Language choice** - Rust vs Go (or others)?

## Research Sessions

### Session 1 - 2026-01-19

**Starting context**: Comprehensive design document provided with architecture, schemas, CLI commands, and implementation plan.

---

#### Primary User & Use Case

**Q: Who is the primary user?**

**A:** AI coding agents (Claude Code specifically), with human oversight. Not a general-purpose task tracker.

**Key insights:**

1. **Agent-first design** - Built for agents to consume, no UI planned. Humans can use it but it's optimized for agent workflows.

2. **Two-agent workflow pattern**:
   - **Planning agent**: Takes specification → creates tasks, phases, epics with full dependency graph
   - **Implementation agent**: Queries tick to find next task, gets context needed to execute

3. **Integration context**: Part of a broader "Claude workflow package" with multiple planning phases. Tick replaces existing output formats (beads, linings, markdown systems) that all have issues.

4. **Collaboration tool**: Enables user-agent collaboration on structured work. User provides oversight, agent executes against deterministic task queries.

5. **The real pain**: Current tools either require manual sync (easy to forget → data loss), are too complex (hooks, daemons), or lack deterministic querying (agents parsing markdown can miss things).

---

#### Integration Context: Claude Technical Workflows

**Q: How does tick fit into the existing workflow package?**

**A:** Tick becomes a new output format for the planning phase, replacing/complementing existing options.

**Current output formats** (from [claude-technical-workflows](https://github.com/leeovery/claude-technical-workflows)):

| Format | Storage | Query Method | Issues |
|--------|---------|--------------|--------|
| **Local Markdown** | Single `.md` file | Parse markdown | No deterministic "what's ready?" query |
| **Beads** | JSONL + SQLite + daemon | `bd ready` | Complex - daemon, sync, hooks |
| **Backlog.md** | Individual `.md` files | MCP or parse files | Requires MCP or markdown parsing |
| **Linear** | External service | API/MCP | External dependency |

**What Tick offers**:
- Deterministic `tick ready` query (like Beads)
- Simple JSONL storage (like Beads)
- No daemon, no sync, no hooks (unlike Beads)
- No markdown parsing required (unlike Local Markdown, Backlog.md)

**Workflow integration point**:
```
Specification → Planning Agent → tick create (tasks, deps, epics)
                                      ↓
Implementation Agent → tick ready → execute task → tick done
```

The planning skill would get a new `output-tick.md` adapter alongside the existing formats.

---

#### Decision: ID Format

**Q: Sequential (`TICK-001`) vs hash-based (`tick-abc123`)?**

**A:** Hash-based with customizable prefix.

**Decision**: `{prefix}-{hash}` format (e.g., `tick-a3f2b7`, `auth-c1d9e4`)

**Rationale**:
- No merge conflicts - IDs are globally unique without coordination
- Prefix is customizable at `tick init` time (default: `tick`)
- Still readable and referenceable in conversation

**Open implementation detail**: Hash length and source (random? content-based?). Short enough to type, long enough to avoid collisions. 6-8 chars seems reasonable for project-scale uniqueness.

---

#### Decision: Hierarchy vs Dependencies

**Q: Subtasks - flat with parent reference vs nested structure?**

**A:** Flat with `parent` field. Infinite depth possible. Separate from dependencies.

**Two distinct concepts**:

| Concept | Field | Meaning |
|---------|-------|---------|
| **Hierarchy** | `parent: string` | "I am part of this larger thing" (decomposition) |
| **Dependency** | `blocked_by: string[]` | "I can't start until these are done" (sequencing) |

**Example**:
```
Epic: Auth System (tick-a1b2)
├── Task: Setup Sanctum (tick-c3d4, parent=tick-a1b2)
├── Task: Login endpoint (tick-e5f6, parent=tick-a1b2, blocked_by=[tick-c3d4])
│     ├── Subtask: Validation (tick-g7h8, parent=tick-e5f6)
│     └── Subtask: Rate limiting (tick-i9j0, parent=tick-e5f6)
└── Task: Logout endpoint (tick-k1l2, parent=tick-a1b2, blocked_by=[tick-e5f6])
```

**Key decisions**:
1. **Infinite depth** - Tasks can have children, which can have children
2. **Explicit completion** - Parent doesn't auto-complete when children done. Manual `tick done`.
3. **Organization, not enforcement** - Hierarchy helps organize work, doesn't constrain it

**Queries are simple**:
```sql
-- Get children of a task
SELECT * FROM tasks WHERE parent = ?;

-- Get dependencies
SELECT blocked_by FROM dependencies WHERE task_id = ?;
```

---

*Research continues...*
