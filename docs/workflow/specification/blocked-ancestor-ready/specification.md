---
topic: blocked-ancestor-ready
status: in-progress
type: feature
date: 2026-02-20
review_cycle: 0
finding_gate_mode: gated
sources:
  - name: blocked-ancestor-ready
    status: pending
---

# Specification: Blocked Ancestor Ready

## Problem

Children of blocked parents incorrectly appear as "ready" in `tick ready` and `tick list --ready`. The current ready check uses 3 conditions (open status, no own unclosed blockers, no open children) but never checks whether the task's parent — or any ancestor — is itself dependency-blocked.

### Example

```
Phase 1 (open)
Phase 2 (open, blocked_by: Phase 1)
  └─ subtask-A (open)    ← shows as "ready" despite Phase 2 being blocked
  └─ subtask-B (open)    ← same
```

The subtasks pass all ready checks because they personally have no blockers — only their parent does.

### Current Implementation

`internal/cli/query_helpers.go` defines:
- `ReadyConditions()` — 3 conditions: open status, no unclosed blockers, no open children
- `BlockedConditions()` — De Morgan inverse: open status AND (has unclosed blocker OR has open children)
- Both use simple EXISTS/NOT EXISTS subqueries against `dependencies` and `tasks` tables
- No ancestor traversal anywhere

### Affected Code Paths

| File | Usage |
|------|-------|
| `internal/cli/query_helpers.go` | `ReadyConditions()`, `BlockedConditions()`, `ReadyWhereClause()` |
| `internal/cli/list.go` | Uses conditions for `--ready` and `--blocked` filters |
| `internal/cli/stats.go` | Uses `ReadyWhereClause()` for ready count |
