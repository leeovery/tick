---
id: tick-core-7-1
phase: 7
status: pending
created: 2026-02-10
---

# Extract shared ready-query SQL conditions

**Problem**: The "ready" SQL WHERE conditions (status='open' AND NOT EXISTS unclosed blockers AND NOT EXISTS open children) are independently authored in list.go (buildListQuery, lines 211-224) and stats.go (RunStats, lines 86-99). The blocked query in list.go (lines 227-245) is the De Morgan inverse, making three total locations encoding ready/blocked semantics. If the definition of "ready" changes (e.g., adding a new condition), all locations must be updated in sync -- a drift risk flagged by both the duplication and architecture agents.

**Solution**: Extract the ready NOT EXISTS subquery clauses as named SQL constants or a helper function (e.g., `readyConditions() []string` or `const readyBlockerCondition` and `readyChildCondition`) in a shared location such as a new `query_helpers.go` file. Both `buildListQuery` and `RunStats` compose their queries from these shared fragments. The blocked conditions can be derived as the negation.

**Outcome**: The ready/blocked SQL definition exists in exactly one place. Changes to readiness semantics require updating a single location. Both list and stats queries stay in sync automatically.

**Do**:
1. Create `internal/cli/query_helpers.go` (or add to an existing shared file)
2. Define shared SQL condition fragments for the two NOT EXISTS subqueries that define "ready": (a) no unclosed blockers, (b) no open children
3. Refactor `buildListQuery` in `internal/cli/list.go` to use the shared conditions for both the `--ready` and `--blocked` filters
4. Refactor the ready count query in `internal/cli/stats.go` (RunStats) to use the same shared conditions
5. Ensure the blocked filter in list.go derives from the negation of the ready conditions rather than independently re-implementing them
6. Run all existing tests to verify no behavioral changes

**Acceptance Criteria**:
- The ready NOT EXISTS subqueries appear in exactly one location (the shared helper)
- buildListQuery uses the shared helper for both ready and blocked filters
- RunStats uses the shared helper for the ready count query
- All existing list and stats tests pass unchanged
- `tick ready`, `tick blocked`, and `tick stats` produce identical output to before

**Tests**:
- Test that the shared ready conditions produce correct SQL fragments
- Test that list --ready still returns correct results after refactor
- Test that list --blocked still returns correct results after refactor
- Test that stats ready/blocked counts remain accurate after refactor
