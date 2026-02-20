---
status: in-progress
created: 2026-02-20
cycle: 1
phase: Plan Integrity Review
topic: Blocked Ancestor Ready
---

# Review Tracking: Blocked Ancestor Ready - Integrity

## Findings

### 1. Task 1-1 missing required Outcome field

**Severity**: Important
**Plan Reference**: Phase 1 / Task 1-1 (tick-fb9d84)
**Category**: Task Template Compliance
**Change Type**: add-to-task

**Details**:
The canonical task template requires an explicit "Outcome" field ("One sentence minimum -- what success looks like"). Task 1-1 has Problem, Solution, Do, Acceptance Criteria, Tests, Edge Cases, and Spec Reference but no Outcome. An implementer benefits from a clear statement of the verifiable end state separate from the granular acceptance criteria.

**Current**:
```
Problem: Children and deeper descendants of dependency-blocked ancestors incorrectly appear as "ready" because ReadyConditions() only checks the task's own blockers and children -- never walks up the parent chain.

Solution: Add ReadyNoBlockedAncestor() helper with recursive CTE in query_helpers.go, integrate as 4th condition in ReadyConditions(). All consumers automatically pick up the change.

Do:
```

**Proposed**:
```
Problem: Children and deeper descendants of dependency-blocked ancestors incorrectly appear as "ready" because ReadyConditions() only checks the task's own blockers and children -- never walks up the parent chain.

Solution: Add ReadyNoBlockedAncestor() helper with recursive CTE in query_helpers.go, integrate as 4th condition in ReadyConditions(). All consumers automatically pick up the change.

Outcome: ReadyConditions() includes ancestor-chain dependency checking so that no descendant of a dependency-blocked ancestor appears in ready results, with all existing ready consumers (list --ready, tick ready, stats) automatically inheriting the fix.

Do:
```

**Resolution**: Pending
**Notes**:

---

### 2. Task 1-2 missing required Outcome field

**Severity**: Important
**Plan Reference**: Phase 1 / Task 1-2 (tick-52f1cf)
**Category**: Task Template Compliance
**Change Type**: add-to-task

**Details**:
Same issue as Task 1-1. The canonical task template requires an explicit "Outcome" field. Task 1-2 has all other required fields but no Outcome.

**Current**:
```
Problem: Tasks blocked due to a dependency-blocked ancestor do not appear in tick list --blocked or tick blocked output. After Task 1 adds the ancestor check to ReadyConditions(), these tasks fall into a gap -- excluded from both ready and blocked results.

Solution: Add the EXISTS inverse of the ancestor CTE to the BlockedConditions() OR clause in query_helpers.go. This is the De Morgan complement of ReadyNoBlockedAncestor(): NOT EXISTS becomes EXISTS with the same recursive CTE.

Do:
```

**Proposed**:
```
Problem: Tasks blocked due to a dependency-blocked ancestor do not appear in tick list --blocked or tick blocked output. After Task 1 adds the ancestor check to ReadyConditions(), these tasks fall into a gap -- excluded from both ready and blocked results.

Solution: Add the EXISTS inverse of the ancestor CTE to the BlockedConditions() OR clause in query_helpers.go. This is the De Morgan complement of ReadyNoBlockedAncestor(): NOT EXISTS becomes EXISTS with the same recursive CTE.

Outcome: BlockedConditions() includes ancestor-blocker detection so that descendants of dependency-blocked ancestors appear in blocked results, closing the gap between ready and blocked filters with stats counts remaining consistent.

Do:
```

**Resolution**: Pending
**Notes**:
