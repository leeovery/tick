---
status: in-progress
created: 2026-03-27
cycle: 2
phase: Plan Integrity Review
topic: Dep Tree Visualization
---

# Review Tracking: Dep Tree Visualization - Integrity

## Findings

### 1. Duplicate acceptance criterion in Task 2-2 (JSONFormatter FormatDepTree)

**Severity**: Minor
**Plan Reference**: Phase 2 / Task dep-tree-visualization-2-2
**Category**: Acceptance Criteria Quality
**Change Type**: update-task

**Details**:
The acceptance criteria for Task 2-2 contain an exact duplicate line: "Empty `roots` in full graph mode renders as `[]` not `null`" appears at both position 4 and position 7 in the criteria list. This is a copy-paste artifact. The duplicate should be removed to avoid confusion about whether a different criterion was intended in that slot.

**Current**:
```
**Acceptance Criteria**:
- [ ] Full graph mode outputs valid JSON with `mode`, `roots`, `chains`, `longest`, `blocked` keys
- [ ] Focused mode outputs valid JSON with `mode`, `target`, `blocked_by`, `blocks` keys
- [ ] All keys use snake_case (no camelCase)
- [ ] Empty `roots` in full graph mode renders as `[]` not `null`
- [ ] Asymmetric focused view omits the empty direction key from JSON output entirely (key absent, not `[]`)
- [ ] Leaf node `children` renders as `[]` not `null` (children is always present on nodes)
- [ ] Empty `roots` in full graph mode renders as `[]` not `null`
- [ ] Diamond dependencies appear as duplicate nodes in the nested tree structure
- [ ] Output uses 2-space indentation via `marshalIndentJSON`
- [ ] All output is valid JSON parseable by `json.Unmarshal`
- [ ] All existing tests pass (`go test ./...`)
```

**Proposed**:
```
**Acceptance Criteria**:
- [ ] Full graph mode outputs valid JSON with `mode`, `roots`, `chains`, `longest`, `blocked` keys
- [ ] Focused mode outputs valid JSON with `mode`, `target`, `blocked_by`, `blocks` keys
- [ ] All keys use snake_case (no camelCase)
- [ ] Empty `roots` in full graph mode renders as `[]` not `null`
- [ ] Asymmetric focused view omits the empty direction key from JSON output entirely (key absent, not `[]`)
- [ ] Leaf node `children` renders as `[]` not `null` (children is always present on nodes)
- [ ] Diamond dependencies appear as duplicate nodes in the nested tree structure
- [ ] Output uses 2-space indentation via `marshalIndentJSON`
- [ ] All output is valid JSON parseable by `json.Unmarshal`
- [ ] All existing tests pass (`go test ./...`)
```

**Resolution**: Pending
**Notes**:
