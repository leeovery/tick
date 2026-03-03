---
topic: auto-cascade-parent-status
status: in-progress
work_type: feature
date: 2026-03-03
---

# Discussion: Auto-Cascade Parent Status

## Context

When a child task transitions to `in_progress`, its parent (and ancestors) remain `open` until manually started. This feels wrong — if you're working on a subtask, the parent is implicitly in progress. The question is whether Tick should automatically cascade status changes upward through the ancestor chain, and what the implications are.

### References

- [Hierarchy & Dependency Model](hierarchy-dependency-model.md)
- [Data Schema Design](data-schema-design.md)

## Questions

- [ ] Should starting a child auto-start its ancestors?
      - Implicit vs explicit status transitions
      - What "in progress" means for a parent task
- [ ] Should completing/cancelling a parent cascade downward to children?
      - Done cascading: close all open children?
      - Cancel cascading: cancel all children?
- [ ] What happens on undo/reopen after a cascade?
      - Reopen a child that triggered upward cascade
      - Reopen a parent that cascaded downward
- [ ] Should cascade behavior be unconditional or configurable?
      - Always-on vs opt-in/opt-out
      - Per-task vs global setting
- [ ] Edge cases with multiple children and partial completion
      - Some siblings done, some open — parent status?
      - All children done — auto-complete parent?
