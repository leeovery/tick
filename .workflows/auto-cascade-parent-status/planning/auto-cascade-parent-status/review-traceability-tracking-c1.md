---
status: complete
created: 2026-03-05
cycle: 1
phase: Traceability Review
topic: Auto-Cascade Parent Status
---

# Review Tracking: Auto-Cascade Parent Status - Traceability

## Findings

### 1. ValidateReopen method not in spec API surface; Rule 9 ancestor walk extends spec

**Type**: Hallucinated content
**Spec Reference**: Architecture: StateMachine > API Surface; Rule 9
**Plan Reference**: Phase 1 / auto-cascade-parent-status-1-5 (tick-dc1dbf)
**Change Type**: update-task

**Details**:
The spec's API surface defines exactly 5 methods on StateMachine: Transition, ValidateAddChild, ValidateAddDep, Cascades, and ApplyWithCascades. Task auto-cascade-parent-status-1-5 introduces a 6th method `ValidateReopen()` that is not in the spec.

Additionally, the spec's Rule 9 says "Cannot reopen a child under a `cancelled` parent" -- referring to the direct parent. Task auto-cascade-parent-status-1-5 walks the full ancestor chain (grandparent, great-grandparent, etc.), returning an error if any ancestor is cancelled. The spec does not require ancestor-chain walking for Rule 9.

The Rule 9 check should be integrated into Transition() (which is in the spec API) and should check only the direct parent, matching the spec text.

**Current**:
```
Problem: Currently StateMachine.Transition() only validates the standard transition table (Rule 1). Rule 9 requires an additional guard: reopening a task whose parent (or any ancestor) is cancelled must be blocked with an error directing the user to reopen the parent first.

Solution: Add a ValidateReopen(tasks []Task, t *Task) error helper method to StateMachine that walks the ancestor chain and returns an error if any ancestor is cancelled. The CLI caller invokes this before Transition() when action is "reopen".

Outcome: Attempting to reopen a task that has a cancelled parent or ancestor returns an error. The guard walks the full ancestor chain to catch deeply nested cases.

Do:
- Add func (sm *StateMachine) ValidateReopen(tasks []Task, t *Task) error to state_machine.go
- Build a map of taskID -> *Task from the tasks slice for efficient lookup
- Walk the ancestor chain: starting from t.Parent, look up each parent, check if Status == StatusCancelled, if so return error; otherwise continue to that parent's parent
- Error message: fmt.Errorf("cannot reopen task under cancelled parent, reopen parent first")
- Stop walking when Parent is empty (root task) or task not found
- Add tests in state_machine_test.go

Acceptance Criteria:
- [ ] sm.ValidateReopen() exists with signature (tasks []Task, t *Task) error
- [ ] Task with cancelled direct parent returns error: "cannot reopen task under cancelled parent, reopen parent first"
- [ ] Task with cancelled grandparent (parent is open but grandparent is cancelled) returns error
- [ ] Task with no parent returns nil
- [ ] Task with open/in_progress/done parent returns nil
- [ ] Deeply nested chain (3+ levels) with cancelled ancestor at top returns error
- [ ] go test ./internal/task/... passes

Tests:
- "it blocks reopen under cancelled direct parent"
- "it blocks reopen under cancelled grandparent"
- "it blocks reopen under cancelled great-grandparent"
- "it allows reopen with no parent"
- "it allows reopen under open parent"
- "it allows reopen under done parent"
- "it allows reopen under in_progress parent"

Edge Cases:
- Deeply nested ancestor chain with cancelled grandparent but open direct parent still triggers error
- Task whose parent ID references a non-existent task -- stop walking, return nil (defensive)

Spec Reference: .workflows/specification/auto-cascade-parent-status/specification.md -- Rule 9
```

**Proposed**:
```
Problem: Currently StateMachine.Transition() only validates the standard transition table (Rule 1). Rule 9 requires an additional guard: reopening a task whose direct parent is cancelled must be blocked with an error directing the user to reopen the parent first.

Solution: Extend StateMachine.Transition() to accept an optional tasks slice context so it can check the direct parent's status when action is "reopen". If the task has a parent and that parent is cancelled, return an error before applying the transition. This keeps Rule 9 inside the Transition method consistent with the spec API surface (no separate ValidateReopen method).

Outcome: Attempting to reopen a task whose direct parent is cancelled returns an error. No new public method is added to StateMachine.

Do:
- Add a TransitionContext struct or add a tasks []Task parameter to Transition: func (sm *StateMachine) Transition(t *Task, action string, tasks []Task) (TransitionResult, error)
- When action == "reopen" and t.Parent is non-empty, look up the parent in tasks; if parent Status == StatusCancelled, return fmt.Errorf("cannot reopen task under cancelled parent, reopen parent first")
- Only check the direct parent, not the full ancestor chain (spec Rule 9 says "cancelled parent")
- Existing callers that don't need the parent check pass nil or empty slice for tasks
- Add tests in state_machine_test.go

Acceptance Criteria:
- [ ] Transition() with action "reopen" checks direct parent status
- [ ] Task with cancelled direct parent returns error: "cannot reopen task under cancelled parent, reopen parent first"
- [ ] Task with no parent returns nil (proceeds normally)
- [ ] Task with open/in_progress/done direct parent proceeds normally
- [ ] No separate ValidateReopen method on StateMachine (consistent with spec API surface)
- [ ] go test ./internal/task/... passes

Tests:
- "it blocks reopen under cancelled direct parent"
- "it allows reopen with no parent"
- "it allows reopen under open parent"
- "it allows reopen under done parent"
- "it allows reopen under in_progress parent"

Edge Cases:
- Task whose parent ID references a non-existent task -- skip check, proceed with reopen (defensive)

Spec Reference: .workflows/specification/auto-cascade-parent-status/specification.md -- Rule 9
```

**Resolution**: Fixed
**Notes**: Rule 9 check integrated into Transition() with direct-parent-only check. Ancestor walk unnecessary because Rule 4 cascade-cancels children transitively.

---

### 2. CLI callers reference ValidateReopen which should not exist per Finding 1

**Type**: Hallucinated content
**Spec Reference**: Architecture: StateMachine > API Surface
**Plan Reference**: Phase 1 / auto-cascade-parent-status-1-6 (tick-f998b0)
**Change Type**: update-task

**Details**:
Task auto-cascade-parent-status-1-6 references `sm.ValidateReopen()` in the Do steps and acceptance criteria. If Finding 1 is accepted (Rule 9 integrated into Transition), the CLI caller no longer needs a separate ValidateReopen call.

**Current**:
```
Do:
- In internal/cli/transition.go (RunTransition): instantiate var sm task.StateMachine; replace task.Transition(&tasks[i], command) with sm.Transition(&tasks[i], command); before the sm.Transition call, if command == "reopen", call sm.ValidateReopen(tasks, &tasks[i]) and return the error if non-nil
- In internal/cli/dep.go: replace task.ValidateDependency calls with sm.ValidateAddDep
- In internal/cli/create.go: replace task.ValidateDependencies/ValidateDependency calls with sm.ValidateAddDep in a loop; add sm.ValidateAddChild(parentTask) call where the parent is resolved
- In internal/cli/update.go: replace task.ValidateDependency calls with sm.ValidateAddDep; add sm.ValidateAddChild(parentTask) call where reparenting is handled
- Run go test ./... to confirm no regressions across all packages

Acceptance Criteria:
- [ ] RunTransition uses sm.Transition() instead of task.Transition()
- [ ] RunTransition calls sm.ValidateReopen() before transition when command is "reopen"
- [ ] Dependency commands use sm.ValidateAddDep() instead of task.ValidateDependency()
- [ ] Create command calls sm.ValidateAddChild() when a parent is specified
- [ ] Update command calls sm.ValidateAddChild() when reparenting to a new parent
- [ ] All existing CLI tests pass (go test ./internal/cli/...)
- [ ] All existing task tests pass (go test ./internal/task/...)
- [ ] Full test suite passes (go test ./...)

Tests:
- "it blocks creating a task with cancelled parent"
- "it blocks reparenting to cancelled parent"
- "it blocks reopening task under cancelled parent"
- "it blocks adding dependency on cancelled task"
- Existing transition tests continue to pass
- Existing dependency tests continue to pass
```

**Proposed**:
```
Do:
- In internal/cli/transition.go (RunTransition): instantiate var sm task.StateMachine; replace task.Transition(&tasks[i], command) with sm.Transition(&tasks[i], command, tasks) passing the full tasks slice so Transition can perform Rule 9 check internally
- In internal/cli/dep.go: replace task.ValidateDependency calls with sm.ValidateAddDep
- In internal/cli/create.go: replace task.ValidateDependencies/ValidateDependency calls with sm.ValidateAddDep in a loop; add sm.ValidateAddChild(parentTask) call where the parent is resolved
- In internal/cli/update.go: replace task.ValidateDependency calls with sm.ValidateAddDep; add sm.ValidateAddChild(parentTask) call where reparenting is handled
- Run go test ./... to confirm no regressions across all packages

Acceptance Criteria:
- [ ] RunTransition uses sm.Transition() instead of task.Transition()
- [ ] RunTransition passes tasks slice to sm.Transition() for Rule 9 parent check
- [ ] Dependency commands use sm.ValidateAddDep() instead of task.ValidateDependency()
- [ ] Create command calls sm.ValidateAddChild() when a parent is specified
- [ ] Update command calls sm.ValidateAddChild() when reparenting to a new parent
- [ ] All existing CLI tests pass (go test ./internal/cli/...)
- [ ] All existing task tests pass (go test ./internal/task/...)
- [ ] Full test suite passes (go test ./...)

Tests:
- "it blocks creating a task with cancelled parent"
- "it blocks reparenting to cancelled parent"
- "it blocks reopening task under cancelled parent"
- "it blocks adding dependency on cancelled task"
- Existing transition tests continue to pass
- Existing dependency tests continue to pass
```

**Resolution**: Fixed
**Notes**: ValidateReopen references removed. CLI now passes tasks slice to Transition() for Rule 9 check.

---

### 3. Unchanged terminal children collection not covered by any task

**Type**: Incomplete coverage
**Spec Reference**: CLI Display section -- "Both formats show the same information -- the primary transition plus all cascaded changes and unchanged terminal children."
**Plan Reference**: Phase 3 / auto-cascade-parent-status-3-3 (tick-a24919)
**Change Type**: add-to-task

**Details**:
The spec explicitly requires showing unchanged terminal children in cascade output. Task auto-cascade-parent-status-3-1 defines `UnchangedEntry` in CascadeResult, and auto-cascade-parent-status-3-2 renders them. But no task describes how unchanged terminal children are collected. ApplyWithCascades (auto-cascade-parent-status-2-7) returns only `(TransitionResult, []CascadeChange, error)` per the spec API -- unchanged children are not in that return type. Task auto-cascade-parent-status-3-3 says "build CascadeResult" but does not describe the logic for finding unchanged terminal children (children of the primary task that are already terminal and were not cascaded). An implementer would need to go back to the specification to understand this requirement.

**Current**:
```
Do:
- In RunTransition, create task.StateMachine{}, call sm.ApplyWithCascades inside store.Mutate
- If len(cascades) == 0: use FormatTransition (no visual change)
- If len(cascades) > 0: build CascadeResult, call FormatCascadeTransition
- Quiet mode: print only task ID
- Return modified tasks slice from Mutate
```

**Proposed**:
```
Do:
- In RunTransition, create task.StateMachine{}, call sm.ApplyWithCascades inside store.Mutate
- If len(cascades) == 0: use FormatTransition (no visual change)
- If len(cascades) > 0: build CascadeResult by populating Cascaded from []CascadeChange and Unchanged by scanning tasks for children of the primary task that are in terminal states (done/cancelled) and were not included in the cascade changes (already terminal before the cascade)
- Quiet mode: print only task ID
- Return modified tasks slice from Mutate
```

**Resolution**: Fixed
**Notes**: Added explicit collection logic to auto-cascade-parent-status-3-3 Do steps and acceptance criteria.
