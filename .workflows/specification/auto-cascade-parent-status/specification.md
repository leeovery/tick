---
topic: auto-cascade-parent-status
status: in-progress
type: feature
work_type: feature
date: 2026-03-04
review_cycle: 1
finding_gate_mode: gated
sources:
  - name: auto-cascade-parent-status
    status: incorporated
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
Standard transition table — `open → in_progress`, `open → done`, `open → cancelled`, `in_progress → done`, `in_progress → cancelled`, `done/cancelled → open` (reopen). Invalid transitions return an error.

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

Dependency state on target children does not gate cascade transitions — a child with unresolved dependencies is still cascaded. Consistent with the advisory dependency principle.

#### Reopen Behavior

**Rule 5: Auto-done undo**
When a child is reopened under a `done` parent, the parent reopens to `open`. The parent's `done` status was derived from all children being terminal — that premise is now broken.

**Rule 6: New child added to done parent**
Adding a non-terminal child to a `done` parent triggers parent reopen to `open`. Same principle as Rule 5.

**Rule 9: Block reopen under cancelled parent**
Cannot reopen a child under a `cancelled` parent. Error: "cannot reopen task under cancelled parent, reopen parent first."

**Rule 5 note:** No reverse cascade on reopen otherwise — reopening a child does not revert a started parent; reopening a parent does not reopen cancelled children.

**Reparenting note:** When a child is reparented away (moved to a different parent or made parentless), no cascade reversal occurs on the original parent. The parent's state is its own.

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

### CLI Display

Both formats show the same information — the primary transition plus all cascaded changes and unchanged terminal children. Only presentation differs.

#### Pretty Format

Uses tree formatting with box-drawing characters.

**Downward cascade (cancel):**
```
tick-parent1: in_progress → cancelled

Cascaded:
├─ tick-child1 "Login": in_progress → cancelled
├─ tick-child2 "Signup": open → cancelled
│  ├─ tick-grand1 "Form": open → cancelled
│  └─ tick-grand2 "Validation": open → cancelled
└─ tick-child3 "Logout": done (unchanged)
```

**Upward cascade (start):**
```
tick-child1: open → in_progress

Cascaded:
├─ tick-parent1 "Auth phase": open → in_progress
└─ tick-grand1 "Sprint 3": open → in_progress
```

#### Toon Format

Flat lines with `(auto)` and `(unchanged)` markers for machine parsing.

**Downward cascade (cancel):**
```
tick-parent1: in_progress → cancelled
tick-child1: in_progress → cancelled (auto)
tick-child2: open → cancelled (auto)
tick-grand1: open → cancelled (auto)
tick-grand2: open → cancelled (auto)
tick-child3: done (unchanged)
```

Both formats show unchanged terminal children so the user can see what was *not* affected by the cascade.

### Architecture: StateMachine

A `StateMachine` struct in `internal/task/` consolidates all 11 rules — transition validation, cascade logic, and mutation validation — into a single architectural unit.

#### Design Properties

- **Stateless struct** — no fields, no constructor needed. Method grouping only (standard Go idiom).
- **No external libraries** — stdlib-only, consistent with Tick's dependency philosophy.
- **Pure cascade computation** — `Cascades()` computes what should change without mutating. Returns a list. Separation enables easy testing: assert on returned list without inspecting task mutations.
- **Queue-based cascade processing** — instead of recursive calls, uses a work queue:
  1. Apply primary transition
  2. Compute cascades → add to queue
  3. Pop next cascade from queue, apply it
  4. Check if that cascade triggers more → add to queue
  5. Track processed tasks in a `seen` map to deduplicate
  6. Loop until queue is empty
- **Termination guarantee** — cascades only move tasks toward terminal states or reopen under specific conditions. Parent-child is a DAG (acyclic). Queue always drains.
- **Mutation model** — `Transition()` mutates the target task in-place (Status, Updated, Closed fields), consistent with the existing implementation. `Cascades()` is pure — it reads task state but does not mutate; it returns `[]CascadeChange` describing what should change. `ApplyWithCascades()` orchestrates both: it calls `Transition()` on the target (mutating it), then runs the cascade queue, calling `Transition()` on each cascaded task (mutating them). The caller receives back all changes and persists them atomically.

#### API Surface

```go
type StateMachine struct{}

type CascadeChange struct {
    Task      *Task
    Action    string
    OldStatus Status
    NewStatus Status
}

// Core transition — absorbs existing transition.go
func (sm *StateMachine) Transition(t *Task, action string) (TransitionResult, error)

// Validation — absorbs dependency.go + new rules
func (sm *StateMachine) ValidateAddChild(parent *Task) error
func (sm *StateMachine) ValidateAddDep(tasks []Task, taskID, blockerID string) error

// Cascade computation — pure, does NOT mutate
func (sm *StateMachine) Cascades(tasks []Task, changed *Task, action string) []CascadeChange

// Combined apply + cascade loop — main entry point for callers
func (sm *StateMachine) ApplyWithCascades(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
```

#### Migration

Existing logic from `transition.go` and `dependency.go` migrates into the StateMachine. `task.Transition()` becomes `sm.Transition()`, `task.ValidateDependency()` becomes `sm.ValidateAddDep()`. Old functions become thin wrappers or get deleted. Callers update accordingly.

### Dependencies

Prerequisites that must exist before implementation can begin:

**None.** This feature builds on existing infrastructure — the Task struct, JSONL storage, SQLite cache, transition table, dependency validation, and CLI formatters all exist. The StateMachine consolidates and extends existing code; it does not depend on any unbuilt systems.

The existing `transition.go` and `dependency.go` logic is migrated into the new StateMachine — these are refactoring targets, not external dependencies.

---

## Working Notes

[In-progress discussion captured here]
