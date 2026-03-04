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

### Cascade Rules

Eleven rules govern all transition, validation, and cascade behavior. Rules 1, 10, and 11 are existing logic to be migrated; the rest are new.

#### Upward Cascades

**Rule 1: Transition validation** (existing)
Standard transition table — `open → in_progress`, `in_progress → done`, `in_progress → cancelled`, `done/cancelled → open` (reopen). Invalid transitions return an error.

**Rule 2: Upward start cascade**
When a child transitions to `in_progress`, walk the ancestor chain and set any `open` ancestors to `in_progress`. Recursive — applies to grandparents and beyond.

**Rule 3: Upward completion cascade**
When all children of a parent reach a terminal state (`done` or `cancelled`), the parent automatically transitions:
- If at least one child is `done` → parent goes to `done`
- If all children are `cancelled` → parent goes to `cancelled`

Recursive — triggers re-evaluation up the ancestor chain.

#### Downward Cascades

**Rule 4: Downward done/cancel cascade**
When a parent is marked `done` or `cancelled`, non-terminal children (`open`, `in_progress`) copy the parent's terminal status. Children already `done` or `cancelled` are left untouched. Recursive — applies to grandchildren and beyond.

#### Reopen Behavior

**Rule 5: Auto-done undo**
When a child is reopened under a `done` parent, the parent reopens to `open`. The parent's `done` status was derived from all children being terminal — that premise is now broken.

**Rule 6: New child added to done parent**
Adding a non-terminal child to a `done` parent triggers parent reopen to `open`. Same principle as Rule 5.

**Rule 9: Block reopen under cancelled parent**
Cannot reopen a child under a `cancelled` parent. Error: "cannot reopen task under cancelled parent, reopen parent first."

**Rule 5 note:** No reverse cascade on reopen otherwise — reopening a child does not revert a started parent; reopening a parent does not reopen cancelled children.

#### Validation Rules

**Rule 7: Block adding child to cancelled parent**
Cannot add a child to a `cancelled` parent. Error: "cannot add child to cancelled task, reopen it first."

**Rule 8: Block adding dependency on cancelled task**
Cannot add a dependency on a `cancelled` task. Error: "cannot add dependency on cancelled task, reopen it first."

**Rule 10: Cycle detection** (existing)
No circular dependencies allowed. Migrated into StateMachine.

**Rule 11: Child-blocked-by-parent rejection** (existing)
A child cannot have its own parent as a dependency. Deadlock prevention. Migrated into StateMachine.

### Transition History

Each task gains a `transitions` array field, following the same pattern as `notes`. Each entry records:

- `from` — previous status
- `to` — new status
- `at` — timestamp
- `auto` — boolean, `true` if the transition was triggered by a cascade (not a direct user action)

Stored in JSONL as part of the Task struct. The SQLite cache gains a `task_transitions` junction table (same pattern as `task_notes`). Cache schema version must be incremented.

Growth is bounded — tasks don't transition many times, and JSONL already does full rewrites on every mutation.

---

## Working Notes

[In-progress discussion captured here]
