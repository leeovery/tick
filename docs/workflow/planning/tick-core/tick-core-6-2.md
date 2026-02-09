---
id: tick-core-6-2
phase: 6
status: pending
created: 2026-02-09
---

# Extract shared ready/blocked SQL WHERE clauses

**Problem**: `StatsReadyCountQuery` in `stats.go` duplicates the WHERE clause from `ReadyQuery` in `ready.go` (same three conditions: status=open, NOT EXISTS unclosed blockers, NOT EXISTS open children). `StatsBlockedCountQuery` duplicates the WHERE clause from `BlockedQuery` in `blocked.go`. Additionally, `buildReadyFilterQuery` and `buildBlockedFilterQuery` in `list.go` are near-identical functions (~15 lines each) that differ only in which inner query constant they wrap. If the ready/blocked logic changes, both the list query and the stats count query must be updated in sync.

**Solution**: Extract the shared WHERE clause into constants (e.g. `readyWhereClause`, `blockedWhereClause`), then compose both the list query and count query from them. Also extract a shared `buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string)` function that both filter query builders call.

**Outcome**: The ready and blocked query logic is defined in exactly one place. Changes to readiness/blocked criteria propagate automatically to both list and stats queries.

**Do**:
1. In the appropriate query file(s), define `readyWhereClause` and `blockedWhereClause` as string constants containing just the WHERE conditions.
2. Redefine `ReadyQuery` as `"SELECT ... FROM tasks t WHERE " + readyWhereClause + " ORDER BY ..."` and `StatsReadyCountQuery` as `"SELECT COUNT(*) FROM tasks t WHERE " + readyWhereClause`. Same pattern for blocked.
3. In `internal/cli/list.go`, extract a shared `buildWrappedFilterQuery(innerQuery string, f listFilters, descendantIDs []string) (string, []interface{})` that both `buildReadyFilterQuery` and `buildBlockedFilterQuery` delegate to.
4. Run existing tests to verify no query behavior changes.

**Acceptance Criteria**:
- ReadyQuery and StatsReadyCountQuery share the same WHERE clause constant
- BlockedQuery and StatsBlockedCountQuery share the same WHERE clause constant
- buildReadyFilterQuery and buildBlockedFilterQuery are collapsed into calls to a shared function
- All existing list, ready, blocked, and stats tests pass unchanged

**Tests**:
- All existing tests for list --ready, list --blocked, stats, ready, and blocked commands pass
- No new tests needed -- this is a pure refactor with existing coverage
