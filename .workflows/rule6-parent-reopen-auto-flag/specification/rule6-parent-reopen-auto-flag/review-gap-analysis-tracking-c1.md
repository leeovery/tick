---
status: in-progress
created: 2026-03-12
cycle: 1
phase: Gap Analysis
topic: rule6-parent-reopen-auto-flag
---

# Review Tracking: rule6-parent-reopen-auto-flag - Gap Analysis

## Findings

### 1. Integration test verification mechanism unspecified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Testing section

**Details**:
The spec calls for two integration tests that verify `auto=true` on transition records:
- `create --parent <done-parent>` produces `auto=true` on parent reopen
- `update --parent` reparent triggers auto-completion with `auto=true`

However, the current `show --json` output (`jsonTaskDetail`) does not include a `Transitions` field. There is no CLI-visible way to inspect the `auto` flag on transition records. The spec doesn't describe how integration tests should verify the `auto` value. Options include: reading JSONL directly, querying the SQLite cache's `task_transitions` table, or using the store API. An implementer would have to decide the approach.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 2. evaluateRule3 rename target not specified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Call Site Updates section

**Details**:
The spec states "`evaluateRule3` should be renamed to something descriptive (unexported, single call site)" but does not specify the target name. An implementer must invent a name, which is a minor design decision. A concrete suggestion (e.g., `autoCompleteOriginalParent` or `completeParentIfChildrenTerminal`) would eliminate ambiguity.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 3. Existing test update strategy unclear

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Testing section

**Details**:
The spec says "Update existing `ApplyWithCascades` unit tests for the new wrapper signatures." The existing test file (`apply_cascades_test.go`) has 13 subtests that all call `sm.ApplyWithCascades(...)` directly. After the refactoring, `ApplyWithCascades` becomes the unexported `applyWithCascades`. The spec doesn't clarify whether:
1. Existing tests should be updated to call `ApplyUserTransition` (since they test user-initiated semantics with `Auto: false`)
2. Some tests should be split to test both wrappers
3. Tests should continue testing the unexported function directly (which is possible since tests are in `package task`)

Since all existing tests assert `Auto: false` on the primary target (user-initiated behavior), the natural mapping is to rewrite them against `ApplyUserTransition`. But the spec leaves this implicit.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

