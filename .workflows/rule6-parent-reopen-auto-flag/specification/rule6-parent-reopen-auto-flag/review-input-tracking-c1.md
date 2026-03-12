---
status: in-progress
created: 2026-03-12
cycle: 1
phase: Input Review
topic: rule6-parent-reopen-auto-flag
---

# Review Tracking: rule6-parent-reopen-auto-flag - Input Review

## Findings

### 1. Wrapper function signatures use incorrect parameter type

**Source**: Specification "Fix" section, compared against `internal/task/apply_cascades.go` line 18
**Category**: Enhancement to existing topic
**Affects**: Fix section

**Details**:
The specification defines the wrapper signatures as `ApplyUserTransition(tasks []*Task, target *Task, action string)` and `ApplySystemTransition(tasks []*Task, target *Task, action string)` using `[]*Task` (pointer slice). The actual codebase uses `[]Task` (value slice) for `ApplyWithCascades`: `func (sm StateMachine) ApplyWithCascades(tasks []Task, target *Task, action string)`. All three callers pass `[]Task`. The wrappers must match the existing convention.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Updated spec signatures from []*Task to []Task to match codebase convention.

---

### 2. Doc comment on ApplyWithCascades needs updating

**Source**: Investigation "Contributing Factors" section (line 86): "The function's doc comment explicitly states 'The primary task receives a TransitionRecord with Auto: false'"
**Category**: Enhancement to existing topic
**Affects**: Fix section

**Details**:
The investigation explicitly notes that `ApplyWithCascades` has a doc comment stating "The primary task receives a TransitionRecord with Auto: false" (confirmed at lines 5-8 of `apply_cascades.go`). When the function becomes the unexported `applyWithCascades` with an `auto bool` parameter, this doc comment becomes inaccurate. The specification should mention updating the doc comment to reflect the parameterized behavior, and adding appropriate doc comments to the two new exported wrappers.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:
