---
topic: auto-cascade-parent-status
status: in-progress
type: feature
work_type: feature
date: 2026-03-04
review_cycle: 0
finding_gate_mode: gated
sources:
  - name: auto-cascade-parent-status
    status: pending
---

# Specification: Auto-Cascade Parent Status

## Specification

### Core Concept

Tick's parent-child model treats parents as **containers** — organizational groupings whose status reflects aggregate child state. This is already baked into Tick's ready/blocked rules (`ReadyNoOpenChildren` prevents parents from being ready while children are open).

Auto-cascade extends this model: task status changes propagate automatically through the ancestor/descendant chain. Behavior is **unconditional** — no configuration system, no opt-in/opt-out. Two driving principles:

1. **Cancelled is a hard stop** — cannot add children, dependencies, or reopen children under a cancelled task. Requires explicit reopen first.
2. **Done is soft and revisitable** — adding a child to a done parent triggers automatic reopen.

Dependencies remain **advisory** — they affect queries (ready/blocked) not transitions. Cascades follow the same principle: dependency status does not gate state changes.

---

## Working Notes

[In-progress discussion captured here]
