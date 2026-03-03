---
topic: auto-cascade-parent-status
status: in-progress
work_type: feature
date: 2026-03-03
---

# Discussion: Auto-Cascade Parent Status

## Context

When a child task transitions to `in_progress`, its parent (and ancestors) remain `open` until manually started. This feels wrong — if you're working on a subtask, the parent is implicitly in progress. The question is whether Tick should automatically cascade status changes upward through the ancestor chain, and what the implications are.

The primary pain point: parent containers sit as "open" and never get closed out, especially when AI agents (Claude) are working through subtasks via the workflow system. Parents get neglected.

### References

- [Hierarchy & Dependency Model](hierarchy-dependency-model.md)
- [Data Schema Design](data-schema-design.md)

## Questions

- [x] Should starting a child auto-start its ancestors?
- [x] Should cascade behavior be unconditional or configurable?
- [ ] Should completing/cancelling a parent cascade downward to children?
      - Done cascading: close all open children?
      - Cancel cascading: cancel all children?
- [ ] What happens on undo/reopen after a cascade?
      - Reopen a child that triggered upward cascade
      - Reopen a parent that cascaded downward
- [ ] Edge cases with multiple children and partial completion
      - Some siblings done, some open — parent status?
      - All children done — auto-complete parent?

---

*Each question above gets its own section below. Check off as concluded.*

---

## Should starting a child auto-start its ancestors?

### Context

The fundamental question: when a child transitions to `in_progress`, should the parent automatically follow? This touches on what "in progress" means for a parent task.

### Options Considered

**Option A: Parent as independent work item** — parent status is independently meaningful. "In progress" means *you* are working on it directly. No auto-cascade.

**Option B: Parent as container** — parent is an organizational grouping. Its status reflects aggregate child state. Auto-cascade makes sense because working on a subtask means working on the parent.

### Journey

Started by questioning whether parents are containers or work items. The user's mental model is clear: parents with children are containers/phases. The parent is a phase title. If work was independent, it should be a sibling with dependencies, not a child.

Explored what Linear does for comparison. Linear does NOT auto-start parents when children start — it's a gap users have built Zapier workarounds for. Linear only auto-closes parents when all children complete (opt-in, added Sept 2024). Their philosophy: automate low-judgment bookkeeping, keep high-judgment transitions manual.

However, Tick's existing design already treats parents as containers — `ReadyNoOpenChildren` prevents parents from being ready while children are open, and parent-child is explicitly grouping, not dependency. Auto-cascading is consistent with this existing philosophy.

Key insight: the real pain isn't "I can't see that a parent is in progress" — it's that parent containers sit around forever not getting closed. The upward start cascade is about visibility/correctness; the real fix is auto-closing parents when all children finish.

### Decision

**Yes, auto-cascade upward.** When a child transitions to `in_progress`, walk the ancestor chain and set any `open` ancestors to `in_progress`. This is consistent with Tick's existing parent-as-container model. Recursive — applies to grandparents etc.

---

## Should cascade behavior be unconditional or configurable?

### Context

Whether to add a configuration system to make cascade behavior opt-in/opt-out.

### Options Considered

**Option A: Configurable (Linear-style)** — per-team/per-project settings. Requires config file infrastructure.

**Option B: Unconditional** — always on, opinionated default.

### Journey

Considered the config route. Tick currently has zero configuration — no config file, no dotfile settings. Adding one is significant architectural cost: file format, loading, validation, defaults, documentation. It's permanent API surface.

Linear needed config because they serve thousands of teams. Tick is a personal CLI with one user currently. Opinionated defaults are a feature. The parent-as-container model is already baked into Tick's ready/blocked rules.

Decided to ship unconditional. If real user feedback demands config later, we'll have concrete requirements instead of guessing.

### Decision

**Unconditional.** No configuration. Two behaviors ship as default:
1. First child started → parent moves to `in_progress` (if parent is `open`)
2. All children reach terminal state → parent moves to `done`

If users complain, add config then with real feedback.

---

## Should completing/cancelling a parent cascade downward to children?
