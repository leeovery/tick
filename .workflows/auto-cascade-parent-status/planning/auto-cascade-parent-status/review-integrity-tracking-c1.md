---
status: complete
created: 2026-03-06
cycle: 1
phase: Plan Integrity Review
topic: Auto-Cascade Parent Status
---

# Review Tracking: Auto-Cascade Parent Status - Integrity

## Findings

### 1. ApplyWithCascades Transition calls missing tasks parameter

**Severity**: Important
**Plan Reference**: Phase 2 / auto-cascade-parent-status-2-7 (tick-d5cbbc)
**Category**: Task Self-Containment
**Change Type**: update-task

**Details**:
Task auto-cascade-parent-status-1-5 changes the `Transition()` signature to accept a `tasks []Task` parameter for Rule 9 parent checking. Task auto-cascade-parent-status-2-7 (ApplyWithCascades) calls `sm.Transition()` in both Step 1 (primary transition) and Step 5 (cascade loop) but omits the `tasks` parameter in both places. This would cause a compilation error. An implementer would need to cross-reference auto-cascade-parent-status-1-5 to discover the correct signature, violating self-containment.

**Current**:
```
Do:
- Create internal/task/apply_cascades.go with ApplyWithCascades method
- Signature: func (sm *StateMachine) ApplyWithCascades(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- Step 1: Call sm.Transition(target, action). On error, return immediately.
- Step 2: Append Transition{From: result.OldStatus, To: result.NewStatus, At: target.Updated, Auto: false} to target.Transitions
- Step 3: Call sm.Cascades(tasks, target, action) for initial cascade list
- Step 4: Initialize queue with initial cascades, seen-map with target.ID pre-seeded, results slice
- Step 5: Loop while queue non-empty: dequeue, check seen, mark seen, call Transition, append history entry with Auto: true, append to results, call Cascades for further cascades, enqueue new
- Step 6: Return primary result, results slice, nil
```

**Proposed**:
```
Do:
- Create internal/task/apply_cascades.go with ApplyWithCascades method
- Signature: func (sm *StateMachine) ApplyWithCascades(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- Step 1: Call sm.Transition(target, action, tasks). On error, return immediately.
- Step 2: Append Transition{From: result.OldStatus, To: result.NewStatus, At: target.Updated, Auto: false} to target.Transitions
- Step 3: Call sm.Cascades(tasks, target, action) for initial cascade list
- Step 4: Initialize queue with initial cascades, seen-map with target.ID pre-seeded, results slice
- Step 5: Loop while queue non-empty: dequeue, check seen, mark seen, call sm.Transition(change.Task, change.Action, tasks), append history entry with Auto: true, append to results, call Cascades for further cascades, enqueue new
- Step 6: Return primary result, results slice, nil
```

**Resolution**: Fixed
**Notes**: Updated both Transition() calls in auto-cascade-parent-status-2-7 to pass tasks parameter.

---
