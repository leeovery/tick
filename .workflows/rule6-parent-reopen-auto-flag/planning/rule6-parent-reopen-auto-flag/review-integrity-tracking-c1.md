---
status: in-progress
created: 2026-03-12
cycle: 1
phase: Plan Integrity Review
topic: Rule6 Parent Reopen Auto Flag
---

# Review Tracking: Rule6 Parent Reopen Auto Flag - Integrity

## Findings

### 1. Incorrect subtest count in Phase 1 acceptance criteria: 13 stated but 18 exist

**Severity**: Important
**Plan Reference**: Phase 1 acceptance criteria (planning.md)
**Category**: Acceptance Criteria Quality
**Change Type**: update-task

**Details**:
The Phase 1 acceptance criteria state "Existing 13 ApplyWithCascades subtests pass unchanged under ApplyUserTransition". The actual file `internal/task/apply_cascades_test.go` contains 18 subtests under `TestStateMachine_ApplyWithCascades`. An implementer using this as a verification checkpoint would be confused by the mismatch. The same incorrect count of "13" also appears in the specification, but the plan should be self-contained and accurate.

**Current**:
```
- [ ] Existing 13 `ApplyWithCascades` subtests pass unchanged under `ApplyUserTransition`
```

**Proposed**:
```
- [ ] Existing 18 `ApplyWithCascades` subtests pass unchanged under `ApplyUserTransition`
```

**Resolution**: Fixed
**Notes**: Updated planning.md acceptance criteria.

---

### 2. Incorrect subtest count throughout Task 1-1 description

**Severity**: Important
**Plan Reference**: Task 1-1 (tick-0930d3)
**Category**: Task Self-Containment
**Change Type**: update-task

**Details**:
Task 1-1's description references "13" existing subtests in the Solution, Outcome, Do (step 3 and step 5), Acceptance Criteria, and Tests sections. The actual count is 18. This creates false verification checkpoints that would confuse an implementer. All occurrences must be corrected, and the derived total in Do step 5 updated from "15 tests (13 migrated + 2 new)" to "20 tests (18 migrated + 2 new)".

**Current**:
```
**Solution**: Add an auto bool parameter to ApplyWithCascades and make it unexported (applyWithCascades). Expose two public wrappers: ApplyUserTransition (passes auto=false) and ApplySystemTransition (passes auto=true). Write tests first that verify the auto flag behavior of each wrapper, then implement the refactoring to make them pass. Migrate all 13 existing ApplyWithCascades tests to call ApplyUserTransition with no assertion changes.

**Outcome**: The task package exposes ApplyUserTransition and ApplySystemTransition as the only public entry points for cascading transitions. applyWithCascades is unexported. All 13 existing tests pass unchanged under ApplyUserTransition. Two new tests verify the auto flag distinction between the wrappers.

**Do**:
1. In internal/task/apply_cascades.go, write the new ApplyUserTransition and ApplySystemTransition methods on StateMachine that will wrap the existing logic. Start by adding stub methods that just call ApplyWithCascades — these will compile but the new test for ApplySystemTransition will fail because Auto is still hardcoded to false.
2. In internal/task/apply_cascades_test.go, add two new test cases inside a new TestStateMachine_ApplyUserTransition test function:
   - "it records auto=false on primary target for user transition" — calls ApplyUserTransition on a simple parent+child start scenario, asserts primary target's last TransitionRecord.Auto == false.
   - "it records auto=true on primary target for system transition" — calls ApplySystemTransition on a done parent being reopened, asserts primary target's last TransitionRecord.Auto == true.
3. In internal/task/apply_cascades_test.go, rename the test function TestStateMachine_ApplyWithCascades to TestStateMachine_ApplyUserTransition and change all 13 sm.ApplyWithCascades(...) calls to sm.ApplyUserTransition(...). No assertion changes needed.
4. Now implement the actual refactoring in internal/task/apply_cascades.go:
   - Rename ApplyWithCascades to applyWithCascades (unexported) and add auto bool as a fourth parameter.
   - Change line 43 (Auto: false) to Auto: auto so the primary target's TransitionRecord uses the parameterized value.
   - Add ApplyUserTransition(tasks []Task, target *Task, action string) that calls applyWithCascades(tasks, target, action, false).
   - Add ApplySystemTransition(tasks []Task, target *Task, action string) that calls applyWithCascades(tasks, target, action, true).
   - Update doc comments.
5. Run go test ./internal/task/ — all 15 tests (13 migrated + 2 new) should pass.

**Acceptance Criteria**:
- [ ] ApplyWithCascades is unexported as applyWithCascades with signature (sm StateMachine) applyWithCascades(tasks []Task, target *Task, action string, auto bool) (TransitionResult, []CascadeChange, error)
- [ ] ApplyUserTransition is exported with signature (sm StateMachine) ApplyUserTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- [ ] ApplySystemTransition is exported with signature (sm StateMachine) ApplySystemTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- [ ] All 13 existing subtests pass unchanged under ApplyUserTransition
- [ ] New test verifies ApplyUserTransition records Auto: false on primary target
- [ ] New test verifies ApplySystemTransition records Auto: true on primary target
- [ ] Cascade transitions still record Auto: true regardless of which wrapper is used
- [ ] go test ./internal/task/ passes with zero failures

**Tests**:
- "it records auto=false on primary target for user transition"
- "it records auto=true on primary target for system transition"
- All 13 existing subtests run under ApplyUserTransition with identical assertions
```

**Proposed**:
```
**Solution**: Add an auto bool parameter to ApplyWithCascades and make it unexported (applyWithCascades). Expose two public wrappers: ApplyUserTransition (passes auto=false) and ApplySystemTransition (passes auto=true). Write tests first that verify the auto flag behavior of each wrapper, then implement the refactoring to make them pass. Migrate all 18 existing ApplyWithCascades tests to call ApplyUserTransition with no assertion changes.

**Outcome**: The task package exposes ApplyUserTransition and ApplySystemTransition as the only public entry points for cascading transitions. applyWithCascades is unexported. All 18 existing tests pass unchanged under ApplyUserTransition. Two new tests verify the auto flag distinction between the wrappers.

**Do**:
1. In internal/task/apply_cascades.go, write the new ApplyUserTransition and ApplySystemTransition methods on StateMachine that will wrap the existing logic. Start by adding stub methods that just call ApplyWithCascades — these will compile but the new test for ApplySystemTransition will fail because Auto is still hardcoded to false.
2. In internal/task/apply_cascades_test.go, add two new test cases inside a new TestStateMachine_ApplyUserTransition test function:
   - "it records auto=false on primary target for user transition" — calls ApplyUserTransition on a simple parent+child start scenario, asserts primary target's last TransitionRecord.Auto == false.
   - "it records auto=true on primary target for system transition" — calls ApplySystemTransition on a done parent being reopened, asserts primary target's last TransitionRecord.Auto == true.
3. In internal/task/apply_cascades_test.go, rename the test function TestStateMachine_ApplyWithCascades to TestStateMachine_ApplyUserTransition and change all 18 sm.ApplyWithCascades(...) calls to sm.ApplyUserTransition(...). No assertion changes needed.
4. Now implement the actual refactoring in internal/task/apply_cascades.go:
   - Rename ApplyWithCascades to applyWithCascades (unexported) and add auto bool as a fourth parameter.
   - Change line 43 (Auto: false) to Auto: auto so the primary target's TransitionRecord uses the parameterized value.
   - Add ApplyUserTransition(tasks []Task, target *Task, action string) that calls applyWithCascades(tasks, target, action, false).
   - Add ApplySystemTransition(tasks []Task, target *Task, action string) that calls applyWithCascades(tasks, target, action, true).
   - Update doc comments.
5. Run go test ./internal/task/ — all 20 tests (18 migrated + 2 new) should pass.

**Acceptance Criteria**:
- [ ] ApplyWithCascades is unexported as applyWithCascades with signature (sm StateMachine) applyWithCascades(tasks []Task, target *Task, action string, auto bool) (TransitionResult, []CascadeChange, error)
- [ ] ApplyUserTransition is exported with signature (sm StateMachine) ApplyUserTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- [ ] ApplySystemTransition is exported with signature (sm StateMachine) ApplySystemTransition(tasks []Task, target *Task, action string) (TransitionResult, []CascadeChange, error)
- [ ] All 18 existing subtests pass unchanged under ApplyUserTransition
- [ ] New test verifies ApplyUserTransition records Auto: false on primary target
- [ ] New test verifies ApplySystemTransition records Auto: true on primary target
- [ ] Cascade transitions still record Auto: true regardless of which wrapper is used
- [ ] go test ./internal/task/ passes with zero failures

**Tests**:
- "it records auto=false on primary target for user transition"
- "it records auto=true on primary target for system transition"
- All 18 existing subtests run under ApplyUserTransition with identical assertions
```

**Resolution**: Pending
**Notes**: Every occurrence of "13" changed to "18", and "15 tests (13 migrated + 2 new)" changed to "20 tests (18 migrated + 2 new)".

---
