---
status: in-progress
created: 2026-03-04
cycle: 2
phase: Gap Analysis
topic: auto-cascade-parent-status
---

# Review Tracking: auto-cascade-parent-status - Gap Analysis

## Findings

### 1. Rule 6 reopen does not fit ValidateAddChild API

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 6, Architecture: StateMachine > API Surface

**Details**:
Rule 7 blocks adding a child to a cancelled parent (returns error). Rule 6 triggers a reopen when adding a child to a done parent. Both involve "adding a child to a parent in terminal state," but Rule 7 is pure validation (error or nil) while Rule 6 is a mutation (reopen the parent). The API surface shows `ValidateAddChild(parent *Task) error` which suggests validation-only semantics. An implementer would not know where Rule 6's reopen side-effect lives. Options include: (a) ValidateAddChild mutates the parent for done case and only errors for cancelled, (b) a separate method handles the done-parent reopen, (c) the caller checks parent status and calls Transition separately. The spec should clarify which code path is responsible for the Rule 6 reopen.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added clarification that ValidateAddChild is pure validation; caller handles Rule 6 reopen.

---

### 2. Formatter interface changes for cascade output not specified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: CLI Display, Architecture: StateMachine

**Details**:
The spec defines detailed cascade output for Pretty, Toon, and JSON formats, but the existing `FormatTransition(id, oldStatus, newStatus string)` method can only render a single transition line. The cascade display needs to render a tree (Pretty), a flat list with markers (Toon), or a structured object with arrays (JSON). The spec does not define what new Formatter method(s) are needed, what their signatures are, or what data structure they receive. An implementer would need to design this interface themselves. At minimum the spec should state whether a new `FormatCascadeResult` method (or similar) is added to the Formatter interface, and what input struct it takes (likely wrapping TransitionResult + []CascadeChange + unchanged children).

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added Formatter Interface subsection to CLI Display.

---

### 3. Reparent-away does not clarify Rule 3 re-evaluation

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 3, Reopen Behavior > Reparenting note

**Details**:
The reparenting note says "no cascade reversal occurs on the original parent" when a child is reparented away. But reparenting away changes the set of children. If a parent has two children (one done, one open) and the open child is reparented away, the parent now has only one child which is done. Rule 3 says "when all children of a parent reach a terminal state" the parent auto-completes. Reparenting away isn't a child "reaching terminal state," but it does change whether the "all terminal" condition is met. The spec should clarify whether reparent-away triggers a Rule 3 re-evaluation on the original parent, or whether the "no cascade reversal" note means no cascades of any kind fire on the original parent during reparenting.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 4. Terminology: "action" vs "command" inconsistency

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Architecture: StateMachine > API Surface

**Details**:
The existing codebase uses "command" (transitionTable keys: "start", "done", "cancel", "reopen"). The spec's API surface uses "action" as the parameter name (`Transition(t *Task, action string)`, `CascadeChange.Action`). This is a minor naming inconsistency but an implementer needs to know: is "action" the same set of strings as "command" ("start", "done", "cancel", "reopen")? Or does it use status names ("in_progress", "done", "cancelled", "open")? CascadeChange.Action is particularly ambiguous since cascades don't map to user commands -- a cascade that sets a child to "cancelled" was not a user "cancel" command. The spec should clarify the vocabulary.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:
