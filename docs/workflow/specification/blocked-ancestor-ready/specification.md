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

## Design Decisions

### Blocker Type: Dependency Blockers Only

A task can be "blocked" in two senses:
1. **Dependency-blocked**: has unclosed entries in the `dependencies` table
2. **Children-blocked**: has open/in-progress children (parent waiting on child work)

Only **dependency blockers** on ancestors propagate down to affect descendant readiness. The "has open children" state is structural — it's the normal state for any parent whose children are the work to be done. A parent with open children isn't externally blocked; it's just waiting for its own subtasks to complete.

If children-blocked propagated, leaf tasks would never be ready since their parent always has open children (the leaf task itself).

### Traversal Depth: Full Ancestor Chain

The ancestor check walks the **full ancestor chain** to the root, not just the immediate parent.

**Why not immediate parent only:** Intermediate grouping tasks create a gap. Example:
```
Phase 1 (open)
Phase 2 (open, blocked_by: Phase 1)
  └─ Group A (open, no own blockers)
      └─ subtask-X (open)
```
With immediate-parent-only, subtask-X checks Group A (not blocked), so subtask-X incorrectly appears ready despite Phase 2 being dependency-blocked.

**Why full chain is safe:** Recursive CTE is an established pattern in the codebase (`queryDescendantIDs()` in `list.go`). Ancestor chains are typically shallow (2-4 levels), so performance is a non-issue.
