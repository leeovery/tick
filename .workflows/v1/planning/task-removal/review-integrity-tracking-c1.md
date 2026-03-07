---
status: complete
created: 2026-02-19
cycle: 1
phase: Plan Integrity Review
topic: Task Removal
---

# Review Tracking: Task Removal - Integrity

## Findings

### 1. task-removal-1-2 missing dependency on task-removal-1-1

**Severity**: Critical
**Plan Reference**: Phase 1 / task-removal-1-2 (tick-64566b)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 1-2 (RunRemove handler) calls `fmtr.FormatRemoval(result)` and constructs `RemovalResult` and `RemovedTask` structs, all of which are defined and implemented by task 1-1. Without task 1-1 complete, task 1-2 cannot compile. The dependency must be explicit.

**Current**:
blocked_by: (none)

**Proposed**:
blocked_by: tick-7314b0

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.

---

### 2. task-removal-1-3 missing dependency on task-removal-1-2

**Severity**: Critical
**Plan Reference**: Phase 1 / task-removal-1-3 (tick-0607a0)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 1-3 (error cases) corrects the no-args error message in `parseRemoveArgs` and adds App.Run-level integration tests for error paths in the remove command. All of this code is created by task 1-2. Without 1-2 complete, there is no `parseRemoveArgs`, no `RunRemove`, and no `handleRemove` to test. The dependency must be explicit.

**Current**:
blocked_by: (none)

**Proposed**:
blocked_by: tick-64566b

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.

---

### 3. task-removal-2-2 missing dependency on task-removal-2-1

**Severity**: Critical
**Plan Reference**: Phase 2 / task-removal-2-2 (tick-0c56d2)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 2-2 (confirmation prompt) adds a `stderr io.Writer` parameter to RunRemove, references `a.Stdin` threaded by 2-1, and builds on the fact that `--force` omission no longer errors (changed by 2-1). Its Do step 1 changes the RunRemove signature that 2-1 already modified. Its Do step 6 updates test call sites that 2-1 already updated. Without 2-1 complete, the Stdin field does not exist and the --force error is still in place. The dependency must be explicit.

**Current**:
blocked_by: (none)

**Proposed**:
blocked_by: tick-8bc489

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.

---

### 4. task-removal-3-2 missing dependency on task-removal-3-1

**Severity**: Critical
**Plan Reference**: Phase 3 / task-removal-3-2 (tick-37bab0)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 3-2 (all-or-nothing validation) explicitly states "After task-removal-3-1, parseRemoveArgs returns a []string of deduplicated IDs, but RunRemove still uses only ids[0]." It extends RunRemove to use the full `ids` slice returned by the refactored `parseRemoveArgs`. Without 3-1 complete, parseRemoveArgs still returns a single string. The dependency must be explicit.

**Current**:
blocked_by: (none)

**Proposed**:
blocked_by: tick-2a1fa5

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.

---

### 5. task-removal-3-4 missing dependencies on task-removal-3-2 and task-removal-3-3

**Severity**: Critical
**Plan Reference**: Phase 3 / task-removal-3-4 (tick-9e0c27)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 3-4 (integrate cascade into removal flow) explicitly references "After tasks 3-2 and 3-3" in its Problem statement. It calls `collectDescendants` (defined by 3-3) and relies on the bulk validation/removeSet logic (built by 3-2). Currently only blocked by tick-0c56d2 (2-2). It needs all three: 2-2, 3-2, and 3-3.

**Current**:
blocked_by: tick-0c56d2

**Proposed**:
blocked_by: tick-0c56d2, tick-37bab0, tick-5b74ec

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.

---

### 6. task-removal-3-5 missing dependency on task-removal-3-4

**Severity**: Critical
**Plan Reference**: Phase 3 / task-removal-3-5 (tick-0424d3)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Task 3-5 (bulk+cascade interaction tests) explicitly states "After tasks 3-1 through 3-4, RunRemove supports bulk IDs, all-or-nothing validation, cascade descendant collection, and an expanded confirmation prompt." Every test in this task exercises functionality wired together by task 3-4. Without 3-4 complete, cascade is not integrated into the removal flow and these tests would fail. The dependency must be explicit.

**Current**:
blocked_by: (none)

**Proposed**:
blocked_by: tick-9e0c27

**Resolution**: Skipped
**Notes**: Intra-phase tasks execute sequentially by natural order. tick ready returns tasks in order — explicit blocked_by is not needed for sequential intra-phase dependencies.
