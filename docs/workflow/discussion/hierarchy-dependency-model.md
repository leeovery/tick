# Discussion: Hierarchy & Dependency Model

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick uses two distinct concepts for task relationships:
- **Hierarchy** (`parent` field): "I am part of this larger thing" - decomposition/organization
- **Dependencies** (`blocked_by` array): "I can't start until these are done" - sequencing

The data-schema-design discussion established that hierarchy is "organizational, not workflow constraint" and deferred the question of what `tick ready` actually returns as "implementation detail." This discussion resolves that implementation detail.

**User's question**: How do tasks depend on each other, what shows up when you run `tick ready`, and how does it all tie together?

### References

- [Research: exploration.md](../research/exploration.md) (lines 324-358) - Hierarchy/dependency model
- [Data Schema Design](data-schema-design.md) - Schema decisions, Q3 on hierarchy
- [CLI Command Structure & UX](cli-command-structure-ux.md) - `tick ready` as alias for `list --ready`

### What's Already Decided

From prior discussions:
- `blocked_by` array in JSONL, normalized to `dependencies` table in SQLite
- Hierarchy via `parent` field (no separate `type` field)
- Both `done` and `cancelled` satisfy dependencies
- Explicit completion - parents don't auto-complete when children finish

## Questions

- [x] What exactly does `tick ready` return?
      - **Decision**: Leaf tasks only (no open children, not blocked)
- [x] What parent context appears in `tick show`?
      - **Decision**: Parent ID + title (not full description)
- [ ] Should agents work on parent tasks directly, or only leaf tasks?
      - Addressed by Q1: agents work leaves; parents appear when children done
- [ ] How do hierarchy and dependencies interact?
      - Can a task depend on its own parent/child?
      - Can a parent be blocked by something its child isn't blocked by?
- [ ] What about tasks with no children and no blockers - always ready?
- [ ] Edge cases: cycles, orphans, deep nesting

---

## What exactly does `tick ready` return?

### Context

This is the core question. An agent runs `tick ready` to get its next task. What should appear in that list?

**Scenario from data-schema-design**:
```
Epic: Auth System (tick-a1b2)
├── Setup Sanctum (tick-c3d4) - DONE
├── Login endpoint (tick-e5f6, blocked_by=[tick-c3d4])
│     ├── Validation (tick-g7h8)
│     └── Rate limiting (tick-i9j0)
└── Logout endpoint (tick-k1l2, blocked_by=[tick-e5f6])
```

Setup Sanctum is done. Login endpoint is unblocked. But Login endpoint has children.

### Options Considered

**Option A: Return the unblocked parent (Login endpoint)**
- Query is simple: status=open, no incomplete blockers
- Agent sees Login endpoint, can `tick show` to see children
- Con: Agent must do extra lookup to find actual work units

**Option B: Return only leaf tasks (Validation, Rate limiting)**
- Tasks with open children are implicitly "not ready"
- Agent gets work items directly, no extra lookups
- Con: Query more complex, must check for open children

**Option C: Return all (parent + children)**
- Full visibility
- Con: Cluttered, agent might start parent when children need work first

### Journey

Started by examining what "hierarchy is organizational" (from data-schema-design) actually means in practice.

**Initial tension identified**: Prior discussion said "parent readiness determined only by explicit `blocked_by`, not by having open children" but also "agent works through subtasks." These seemed contradictory - if subtasks are just organizational, why would agent "work through" them? And if agent works through them, shouldn't they appear in `tick ready` instead of parent?

**Two coherent models emerged**:
- **Model A (Subtasks as documentation)**: Parent appears in ready, subtasks are notes/context, agent works on parent
- **Model B (Subtasks as work units)**: Only leaves appear in ready, parent is container, agent works through subtasks

User initially leaned toward Model B but raised concern: "If you create a task with description and acceptance criteria, then later add subtasks, does parent suddenly become unworkable?"

**Explored hybrid**: What if both parent and subtasks appear, but sorted with subtasks first? This creates implicit ordering via sort rather than blocking.

**Challenge raised**: Traced concrete scenario:
```
Parent: Implement login (tick-p1)
Subtask: Add validation (tick-s1, parent=tick-p1)
Subtask: Add rate limiting (tick-s2, parent=tick-p1)
```

If all three appear in `tick ready` with subtasks sorted first:
1. Agent works tick-s1 (validation), done
2. Agent works tick-s2 (rate limiting), done
3. Agent picks tick-p1... what's left?

If subtasks must be done before parent work makes sense, that's a real dependency - should use `blocked_by`. The sorting trick was masking an actual dependency relationship.

**Key realization**: If subtasks need to complete before parent can be worked, that IS a dependency - use the dependency system. Hierarchy should remain purely organizational.

**Resolution**: If a task has subtasks, the parent becomes a container. The subtasks ARE the work. Parent only becomes "ready" when children are all closed. This is the leaf-only model (Option B) but with clear reasoning.

**Concern about parent context**: Would epic's description inform subtasks? In Linear/JIRA, humans typically work the task in front of them without checking parent. Only look at parent if task is unclear or ambiguous. Agent can do the same - `tick show` reveals parent exists, agent can look if needed.

**Task quality principle**: Tasks SHOULD be self-contained. Parent context is a safety net, not a crutch. This puts burden on task writing quality, not the tool.

### Decision

**Option B: Leaf-only `tick ready`**

A task appears in `tick ready` only if ALL of these are true:
1. Status is `open`
2. All `blocked_by` tasks are closed (`done` or `cancelled`)
3. **Has no open children** (tasks with status `open` or `in_progress` where `parent` = this task)

**The rules**:
- No children → leaf task → appears in `tick ready` when unblocked
- Has open children → container → does NOT appear in `tick ready`
- All children closed → parent becomes leaf → appears in `tick ready`
- Explicit `blocked_by` → always respected regardless of hierarchy

**Example**:
```
Epic: Auth System (tick-a1b2) - has 3 children, WON'T appear
├── Setup Sanctum (tick-c3d4) - DONE, won't appear
├── Login endpoint (tick-e5f6) - READY (leaf, unblocked) ✓
└── Logout endpoint (tick-k1l2, blocked_by=[tick-e5f6]) - blocked, won't appear
```

Agent sees only: `tick-e5f6`

When Login and Logout are done, Auth System appears (now a "leaf" with no open children). Agent can verify epic is complete, mark done.

**Rationale**:
- Deterministic: agent always gets actual work units, no ambiguity
- Simple mental model: children ARE the work, parent is container
- Matches human behavior: work the task in front of you
- Dependencies remain explicit: if order matters, use `blocked_by`

---

## Parent Context in `tick show`

### Context

When `tick show <id>` displays a task that has a parent, how much parent information should be included?

### Options Considered

**Option A: Just parent ID**
```
parent: tick-a1b2
```
- Minimal, agent must do extra lookup to understand context

**Option B: Parent ID + title**
```
parent: tick-a1b2 "Auth System"
```
- Gives enough signal for "do I need more context?"
- Agent can `tick show tick-a1b2` if needed

**Option C: Parent ID + title + description**
- Full context inline
- Bloats every task view
- Often unnecessary if task is well-written

### Decision

**Option B: Parent ID + title**

The title provides enough context to determine if deeper investigation is needed. Agent can choose to `tick show <parent_id>` for full description if the current task is unclear.

**Rationale**:
- Mirrors human workflow: work the task, check parent only if unclear
- Puts burden on task quality (tasks should be self-contained)
- Parent context is safety net, not automatic inclusion
- Keeps task output focused on the task itself

---

*Additional question sections to be added as we explore*

---

## Summary

### Key Insights
*[To be captured]*

### Current State
- Exploring the core question: what does `tick ready` return?

### Next Steps
- [ ] Resolve the `tick ready` behavior question
- [ ] Work through edge cases
