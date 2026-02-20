---
status: complete
created: 2026-02-20
cycle: 1
phase: Traceability Review
topic: Blocked Ancestor Ready
---

# Review Tracking: Blocked Ancestor Ready - Traceability

## Findings

### 1. Task 1-2 missing intermediate grouping task scenario for blocked results

**Type**: Incomplete coverage
**Spec Reference**: Test Scenarios table - "Intermediate grouping task" row, Expected: "Subtask is NOT ready, IS blocked"
**Plan Reference**: Phase 1 / Task 1-2 (tick-52f1cf) - Tests section
**Change Type**: add-to-task

**Details**:
The spec defines six test scenarios. The "Intermediate grouping task" scenario expects both "NOT ready" and "IS blocked". Task 1-1 covers the "NOT ready" side with a dedicated integration test, but Task 1-2 only tests child and grandchild appearing in blocked results -- it does not include a test for the intermediate grouping task scenario appearing in blocked output. While structurally similar to the grandchild test, the spec treats it as a distinct scenario worth verifying.

**Current**:
```
Tests:
- "it returns child of dependency-blocked parent in blocked" (integration)
- "it returns grandchild of dependency-blocked grandparent in blocked" (integration)
- "it excludes descendant from blocked when ancestor blocker resolved" (integration)
- "it maintains stats count consistency with blocked ancestors" (integration)
- "BlockedConditions includes ancestor blocker in OR clause" (unit)
```

**Proposed**:
```
Tests:
- "it returns child of dependency-blocked parent in blocked" (integration)
- "it returns grandchild of dependency-blocked grandparent in blocked" (integration)
- "it returns descendant behind intermediate grouping task under blocked ancestor in blocked" (integration)
- "it excludes descendant from blocked when ancestor blocker resolved" (integration)
- "it maintains stats count consistency with blocked ancestors" (integration)
- "BlockedConditions includes ancestor blocker in OR clause" (unit)
```

**Resolution**: Fixed
**Notes**: Applied automatically (auto mode).

---

### 2. Task 1-2 acceptance criterion not in spec: "Stats blocked count equals open - ready"

**Type**: Hallucinated content
**Spec Reference**: Test Scenarios table - "Stats count consistency" row, Expected: "ReadyWhereClause() counts match list --ready output"
**Plan Reference**: Phase 1 / Task 1-2 (tick-52f1cf) - Acceptance Criteria
**Change Type**: remove-from-task

**Details**:
The spec's stats consistency scenario only requires that "ReadyWhereClause() counts match list --ready output". Task 1-2 adds a separate acceptance criterion "Stats blocked count equals open - ready" which is not stated in the specification. While this formula is a mathematical consequence of the De Morgan relationship the spec describes, the spec does not assert it as a requirement. The existing acceptance criterion "Stats ready count matches list --ready count for mixed scenario" already covers the spec requirement.

**Current**:
```
- Stats ready count matches list --ready count for mixed scenario
- Stats blocked count equals open - ready
```

**Proposed**:
```
- Stats ready count matches list --ready count for mixed scenario
```

**Resolution**: Fixed
**Notes**: Applied automatically (auto mode).
