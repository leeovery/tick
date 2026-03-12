---
status: in-progress
created: 2026-03-12
cycle: 1
phase: Traceability Review
topic: Rule6 Parent Reopen Auto Flag
---

# Review Tracking: Rule6 Parent Reopen Auto Flag - Traceability

## Findings

### 1. Task 1-3 contains a third integration test and edge case not in the specification

**Type**: Hallucinated content
**Spec Reference**: Testing section -- spec lists exactly 4 tests (2 unit, 2 integration); no multi-level cascade test
**Plan Reference**: Phase 1 / Task rule6-parent-reopen-auto-flag-1-3 (tick-d6e894) -- third subtest and Edge Cases section
**Change Type**: update-task

**Details**:
The specification defines exactly two integration tests: (1) `create --parent <done-parent>` produces `auto=true` on parent reopen, and (2) `update --parent` reparent triggers auto-completion with `auto=true`. Task 1-3 adds a third integration test ("it records auto=true through multiple cascade levels on reparent Rule 3") and an Edge Cases section describing multi-level cascade propagation. This test exercises the cascade engine which the spec explicitly states is "unchanged" and already correctly records `auto=true` on cascades. The multi-level scenario is not in the specification.

The plan table row for Task 1-3 also lists "reparent triggers Rule 3 cascading through multiple levels" in the Edge Cases column, which should be "none".

**Current**:
Plan table row:
```
| rule6-parent-reopen-auto-flag-1-3 | Integration tests for auto flag in JSONL | reparent triggers Rule 3 cascading through multiple levels | authored | tick-d6e894 |
```

Task description (full):
```
**Problem**: After Tasks 1 and 2, the auto flag is correctly set at the domain level, but there are no end-to-end tests confirming the flag persists correctly in JSONL (the source of truth). The existing CLI tests verify status changes and output, but none inspect TransitionRecord.Auto in the persisted JSONL data.

**Solution**: Add integration tests in internal/cli/create_test.go and internal/cli/update_test.go that exercise the full CLI path (create --parent <done-parent> and update --parent reparent), then read the JSONL file and inspect the TransitionRecord.Auto field on the affected parent tasks.

**Outcome**: Two integration tests verify the auto flag flows through the full stack (CLI -> Store.Mutate -> JSONL persistence -> readback). A third test exercises the edge case of reparenting triggering Rule 3 cascading through multiple hierarchy levels.

**Do**:
1. In internal/cli/create_test.go, add a new subtest inside TestCreate:
   - "it records auto=true on parent reopen when creating child under done parent":
     - Set up a done parent task (tick-ppp111, status done, with Closed timestamp).
     - Run runCreate(t, dir, "New child", "--parent", "tick-ppp111").
     - Assert exit code 0.
     - Call readPersistedTasks(t, tickDir) to read back from JSONL.
     - Find the parent task by ID. Assert it has status open (reopened via Rule 6).
     - Assert len(parent.Transitions) == 1.
     - Assert parent.Transitions[0].From == task.StatusDone.
     - Assert parent.Transitions[0].To == task.StatusOpen.
     - Assert parent.Transitions[0].Auto == true — this is the core assertion proving the bug is fixed.
2. In internal/cli/update_test.go, add a new subtest inside TestUpdate:
   - "it records auto=true on original parent auto-completion when reparenting away":
     - Set up: original parent tick-aaa111 (in_progress), child1 tick-bbb222 (done, with Closed), child2 tick-ccc333 (open, parent=tick-aaa111), and a new parent tick-ddd444 (open).
     - Run runUpdate(t, dir, "tick-ccc333", "--parent", "tick-ddd444") to reparent child2 away.
     - Assert exit code 0.
     - Call readPersistedTasks(t, tickDir).
     - Find original parent tick-aaa111. Assert status is done (Rule 3 auto-completion — only done child remains).
     - Assert len(origParent.Transitions) == 1.
     - Assert origParent.Transitions[0].From == task.StatusInProgress.
     - Assert origParent.Transitions[0].To == task.StatusDone.
     - Assert origParent.Transitions[0].Auto == true — core assertion proving Rule 3 reparent records auto correctly.
3. In internal/cli/update_test.go, add a second subtest for the edge case:
   - "it records auto=true through multiple cascade levels on reparent Rule 3":
     - Set up a 3-level hierarchy: grandparent tick-ggg111 (in_progress, no parent), parent tick-ppp111 (in_progress, parent=tick-ggg111), child1 tick-ccc111 (done, Closed, parent=tick-ppp111), child2 tick-ccc222 (open, parent=tick-ppp111), external tick-eee111 (open, no parent).
     - Grandparent has one child (parent). Parent has two children (child1 done, child2 open).
     - Run runUpdate(t, dir, "tick-ccc222", "--parent", "tick-eee111") to reparent child2 away from parent.
     - Assert exit code 0.
     - Read JSONL via readPersistedTasks(t, tickDir).
     - Find parent tick-ppp111. Assert status done (Rule 3: only child1 done remains).
     - Assert parent.Transitions[0].Auto == true — system-initiated auto-completion.
     - Find grandparent tick-ggg111. Assert status done (Rule 3 cascaded: parent is now terminal, grandparent's only child is terminal).
     - Assert grandparent.Transitions[0].Auto == true — cascade from parent completion propagated upward.
4. Run go test ./internal/cli/ -run "auto=true" to verify the new tests pass, then go test ./... for full regression.

**Acceptance Criteria**:
- [ ] Integration test confirms create --parent <done-parent> produces auto=true on parent reopen transition in JSONL
- [ ] Integration test confirms update --parent reparent triggers auto-completion with auto=true in JSONL
- [ ] Integration test confirms multi-level cascade from reparent Rule 3 records auto=true at every level in JSONL
- [ ] All tests read JSONL directly via readPersistedTasks (not CLI output) to verify the source of truth
- [ ] go test ./... passes with zero failures

**Tests**:
- "it records auto=true on parent reopen when creating child under done parent" — verifies Rule 6 end-to-end: create --parent <done-parent> -> parent JSONL has TransitionRecord{From: done, To: open, Auto: true}
- "it records auto=true on original parent auto-completion when reparenting away" — verifies Rule 3 via reparent end-to-end: update --parent <new-parent> -> original parent JSONL has TransitionRecord{From: in_progress, To: done, Auto: true}
- "it records auto=true through multiple cascade levels on reparent Rule 3" — verifies Rule 3 cascading through grandparent+parent: both get Auto: true in their JSONL TransitionRecords

**Edge Cases**:
- Reparent triggers Rule 3 cascading through multiple levels: when reparenting a child away causes the parent to auto-complete, and the parent was the only child of a grandparent, the grandparent should also auto-complete. Both the parent and grandparent transitions should have Auto: true in JSONL. This is tested in the third subtest.

**Spec Reference**: .workflows/rule6-parent-reopen-auto-flag/specification/rule6-parent-reopen-auto-flag/specification.md — Testing section
```

**Proposed**:
Plan table row:
```
| rule6-parent-reopen-auto-flag-1-3 | Integration tests for auto flag in JSONL | none | authored | tick-d6e894 |
```

Task description (full):
```
**Problem**: After Tasks 1 and 2, the auto flag is correctly set at the domain level, but there are no end-to-end tests confirming the flag persists correctly in JSONL (the source of truth). The existing CLI tests verify status changes and output, but none inspect TransitionRecord.Auto in the persisted JSONL data.

**Solution**: Add integration tests in internal/cli/create_test.go and internal/cli/update_test.go that exercise the full CLI path (create --parent <done-parent> and update --parent reparent), then read the JSONL file and inspect the TransitionRecord.Auto field on the affected parent tasks.

**Outcome**: Two integration tests verify the auto flag flows through the full stack (CLI -> Store.Mutate -> JSONL persistence -> readback).

**Do**:
1. In internal/cli/create_test.go, add a new subtest inside TestCreate:
   - "it records auto=true on parent reopen when creating child under done parent":
     - Set up a done parent task (tick-ppp111, status done, with Closed timestamp).
     - Run runCreate(t, dir, "New child", "--parent", "tick-ppp111").
     - Assert exit code 0.
     - Call readPersistedTasks(t, tickDir) to read back from JSONL.
     - Find the parent task by ID. Assert it has status open (reopened via Rule 6).
     - Assert len(parent.Transitions) == 1.
     - Assert parent.Transitions[0].From == task.StatusDone.
     - Assert parent.Transitions[0].To == task.StatusOpen.
     - Assert parent.Transitions[0].Auto == true -- this is the core assertion proving the bug is fixed.
2. In internal/cli/update_test.go, add a new subtest inside TestUpdate:
   - "it records auto=true on original parent auto-completion when reparenting away":
     - Set up: original parent tick-aaa111 (in_progress), child1 tick-bbb222 (done, with Closed), child2 tick-ccc333 (open, parent=tick-aaa111), and a new parent tick-ddd444 (open).
     - Run runUpdate(t, dir, "tick-ccc333", "--parent", "tick-ddd444") to reparent child2 away.
     - Assert exit code 0.
     - Call readPersistedTasks(t, tickDir).
     - Find original parent tick-aaa111. Assert status is done (Rule 3 auto-completion -- only done child remains).
     - Assert len(origParent.Transitions) == 1.
     - Assert origParent.Transitions[0].From == task.StatusInProgress.
     - Assert origParent.Transitions[0].To == task.StatusDone.
     - Assert origParent.Transitions[0].Auto == true -- core assertion proving Rule 3 reparent records auto correctly.
3. Run go test ./internal/cli/ -run "auto=true" to verify the new tests pass, then go test ./... for full regression.

**Acceptance Criteria**:
- [ ] Integration test confirms create --parent <done-parent> produces auto=true on parent reopen transition in JSONL
- [ ] Integration test confirms update --parent reparent triggers auto-completion with auto=true in JSONL
- [ ] All tests read JSONL directly via readPersistedTasks (not CLI output) to verify the source of truth
- [ ] go test ./... passes with zero failures

**Tests**:
- "it records auto=true on parent reopen when creating child under done parent" -- verifies Rule 6 end-to-end: create --parent <done-parent> -> parent JSONL has TransitionRecord{From: done, To: open, Auto: true}
- "it records auto=true on original parent auto-completion when reparenting away" -- verifies Rule 3 via reparent end-to-end: update --parent <new-parent> -> original parent JSONL has TransitionRecord{From: in_progress, To: done, Auto: true}

**Spec Reference**: .workflows/rule6-parent-reopen-auto-flag/specification/rule6-parent-reopen-auto-flag/specification.md -- Testing section
```

**Resolution**: Pending
**Notes**:

---
