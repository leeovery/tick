---
status: in-progress
created: 2026-03-04
cycle: 1
phase: Gap Analysis
topic: auto-cascade-parent-status
---

# Review Tracking: auto-cascade-parent-status - Gap Analysis

## Findings

### 1. Transition table in Rule 1 omits valid transitions present in codebase

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 1

**Details**:
Rule 1 lists `open → in_progress`, `in_progress → done`, `in_progress → cancelled`, `done/cancelled → open`. The existing codebase also allows `open → done` and `open → cancelled` (direct done/cancel without starting). The spec either intends to remove these transitions or omitted them. This matters because Rule 4 (downward cascade) applies terminal status to `open` children — if `open → done` is not valid, those cascades would fail validation. The current code handles this because `open` is a valid source for `done` and `cancel`.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 2. StateMachine.Transition mutation semantics unclear

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Architecture: StateMachine > API Surface

**Details**:
The existing `Transition()` function mutates the task in-place (Status, Updated, Closed fields). The spec says `Cascades()` is "pure" and does not mutate, but says nothing about whether `sm.Transition()` still mutates in-place. Additionally, `CascadeChange` contains `*Task` (pointer) while `Cascades()` accepts `[]Task` (by value). It's ambiguous whether `CascadeChange.Task` points into the caller's task slice or is a copy. An implementer needs to know: does `ApplyWithCascades` mutate the passed-in tasks, or return intent for the caller to apply?

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 3. Rule 6 trigger scope not fully defined — create vs reparent

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 6

**Details**:
Rule 6 says "adding a non-terminal child to a done parent triggers parent reopen." This could be triggered by: (a) creating a new task with parent set to the done task, (b) reparenting an existing non-terminal task to the done task. The reparenting note later says "no cascade reversal occurs on the original parent" when reparenting away, but doesn't explicitly confirm that reparenting TO a done parent triggers Rule 6. An implementer would need to know which code paths invoke `ValidateAddChild` and whether Rule 6 applies to both creation and reparenting.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 4. Store integration with cascade loop unspecified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Architecture: StateMachine

**Details**:
The spec defines the StateMachine API and its queue-based cascade processing but does not describe how it integrates with the Store layer. Currently `Store.Mutate` handles single-task persistence with file locking. A cascade may modify multiple tasks in a single logical operation. The spec should clarify: does the caller invoke `ApplyWithCascades`, receive all changes, then persist them in a single `Store.Mutate` call? Or does each cascade step persist independently? This affects atomicity — if the process crashes mid-cascade, partial cascades could leave inconsistent state. Given JSONL does full rewrites, a single atomic write is likely intended, but this should be stated.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 5. JSON formatter output for cascades not addressed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: CLI Display

**Details**:
The CLI Display section covers Pretty and Toon formats but omits JSON format. The Formatter interface has `FormatTransition` and the spec mentions three formatter implementations. An implementer would need to know the JSON output structure for cascade results — particularly whether cascaded changes appear as a nested array, flat list, or separate objects.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 6. task_transitions SQL schema not defined

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Transition History

**Details**:
The spec says the SQLite cache gains a `task_transitions` junction table "same pattern as `task_notes`" but does not define columns or indexes. The JSONL fields (`from`, `to`, `at`, `auto`) are described for the struct, but the SQL column names, types, and any indexes are left to the implementer. While "same pattern as task_notes" gives a hint, the `auto` boolean column and the `from`/`to` status columns have no direct analog in task_notes. Minor since an experienced implementer can infer this, but explicit schema removes ambiguity.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 7. Upward completion cascade (Rule 3) interaction with parentless leaf transition

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 3

**Details**:
Rule 3 triggers when "all children of a parent reach a terminal state." The spec doesn't clarify whether this check runs on every terminal transition or only when the transitioning task has a parent. This is likely obvious (only check the parent of the task that changed), but the recursive note — "triggers re-evaluation up the ancestor chain" — could be read as re-evaluating all parents in the system. Clarifying the trigger condition (evaluate only the direct parent of the changed task, then recurse upward) would remove any ambiguity.

**Proposed Addition**:

**Resolution**: Pending
**Notes**: This is minor — most implementers would get this right, but the recursive language could be tightened.
