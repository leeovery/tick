# Task tick-core-3-4: Blocked query, tick blocked & cancel-unblocks-dependents

## Task Summary

Implement the inverse of the `ready` query: a `BlockedQuery` that returns open tasks that cannot be worked because they have unclosed blockers OR open children. Register a `tick blocked` subcommand as an alias for `list --blocked`. Verify end-to-end that cancelling a blocker unblocks its dependents.

### Acceptance Criteria (from plan)

1. Returns open tasks with unclosed blockers or open children
2. Excludes tasks where all blockers closed
3. Only `open` in output
4. Cancel causes dependent to unblock
5. Multiple dependents unblock simultaneously
6. Partial unblock works correctly (two blockers, one cancelled, still blocked)
7. Deterministic ordering (priority ASC, created ASC)
8. `tick blocked` outputs aligned columns
9. Empty result prints `No tasks found.`, exit 0
10. `--quiet` outputs IDs only
11. Reuses ready query logic

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Returns open tasks with unclosed blockers or open children | PASS - SQL uses EXISTS subqueries for both conditions | PASS - SQL uses IN subqueries for both conditions | PASS - SQL uses EXISTS subqueries for both conditions |
| Excludes tasks where all blockers closed | PASS - tested with done blocker | PASS - tested with both done and cancelled blockers separately | PASS - tested with both done and cancelled blockers separately |
| Only `open` in output | PASS - SQL filters `t.status = 'open'`; tested for in_progress/done/cancelled exclusion | PASS - SQL filters `status = 'open'`; tested separately for in_progress, done, cancelled | PASS - SQL filters `t.status = 'open'`; tested separately for in_progress, done, cancelled |
| Cancel causes dependent to unblock | PASS - test verifies dependent moves to ready | PASS - test verifies dependent moves to ready AND leaves blocked list | PASS - test verifies dependent moves to ready AND leaves blocked list |
| Multiple dependents unblock simultaneously | PASS - tests two dependents unblocked | PASS - tests two dependents unblocked | PASS - tests two dependents unblocked |
| Partial unblock works correctly | PASS - cancels one of two blockers, verifies still blocked | PASS - cancels one of two blockers, verifies still blocked AND not in ready | PASS - cancels one of two blockers, verifies still blocked AND not in ready |
| Deterministic ordering | PARTIAL - no explicit ordering test | PASS - tests 4 tasks with varying priority and created timestamps | PASS - tests 4 tasks with varying priority and created timestamps |
| `tick blocked` outputs aligned columns | PARTIAL - no explicit column alignment test | PASS - verifies header fields and column position alignment | PASS - verifies header fields present |
| Empty result prints `No tasks found.`, exit 0 | PASS - tested | PASS - tested with both empty dir and with ready-only tasks | PASS - tested with both empty dir and with ready-only tasks |
| `--quiet` outputs IDs only | PASS - tests prefix is `tick-` | PASS - tests exact ID values and line count | PASS - tests exact ID values, no headers, no titles |
| Reuses ready query logic | PASS - uses `cmdListFiltered` shared function | PASS - uses `runList` with `--blocked` flag; SQL in same file as ReadySQL | PARTIAL - separate `runBlocked()` method duplicates output formatting from `runReady()` |

## Implementation Comparison

### Approach

All three versions implement the same core concept: a SQL query that selects open tasks with unclosed blockers OR open children, ordered by priority ASC then created ASC. The key differences lie in architectural integration.

**V1: Minimal delegation via shared helper**

V1 creates a standalone `blocked.go` file with a SQL constant and a one-line handler:

```go
// blocked.go (27 lines total)
const blockedQuery = `
SELECT t.id, t.title, t.status, t.priority
FROM tasks t
WHERE t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON d.blocked_by = blocker.id
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )
ORDER BY t.priority ASC, t.created ASC
`

func (a *App) cmdBlocked(workDir string, args []string) error {
	return a.cmdListFiltered(workDir, blockedQuery)
}
```

This delegates entirely to `cmdListFiltered` (defined in `ready.go`), which handles store open, query execution, empty message, quiet mode, and column formatting. The `blocked` case is added to `cli.go`'s switch statement. This is the most DRY approach -- the entire implementation is 27 lines.

**V2: Flag-based integration into existing list command**

V2 takes a fundamentally different approach. Instead of a separate handler, `blocked` is wired as `runList([]string{"--blocked"})` in `app.go`:

```go
// app.go
case "blocked":
    return a.runList([]string{"--blocked"})
```

The `BlockedSQL` constant is co-located with `ReadySQL` in `list.go`, and a `listFlags` struct replaces the previous boolean:

```go
// list.go
type listFlags struct {
    ready   bool
    blocked bool
}

func parseListFlags(args []string) listFlags {
    var flags listFlags
    for _, arg := range args {
        switch arg {
        case "--ready":
            flags.ready = true
        case "--blocked":
            flags.blocked = true
        }
    }
    return flags
}
```

SQL selection uses `if/else if`:

```go
querySQL := listAllSQL
if flags.ready {
    querySQL = ReadySQL
} else if flags.blocked {
    querySQL = BlockedSQL
}
```

This means V2 also supports `tick list --blocked` as a first-class flag, which V1 does not.

**V3: Self-contained command with exported condition constant**

V3 creates a full `blocked.go` (118 lines) with an exported `BlockedCondition` SQL fragment constant, a `queryBlockedTasks()` function, and a complete `runBlocked()` method:

```go
// blocked.go
const BlockedCondition = `
    t.status = 'open'
    AND (
        EXISTS (...)
        OR EXISTS (...)
    )
`

func queryBlockedTasks(db *sql.DB) ([]taskRow, error) {
    query := `SELECT t.id, t.title, t.status, t.priority FROM tasks t WHERE ` + BlockedCondition + ` ORDER BY t.priority ASC, t.created ASC`
    // ... full query execution
}

func (a *App) runBlocked() int {
    // Full command handler: DiscoverTickDir, NewStore, Query, format output
}
```

This mirrors V3's `ready.go` pattern exactly (which also has `ReadyCondition` and `queryReadyTasks`). The trade-off is duplicated output formatting logic between `runReady()` and `runBlocked()` -- both contain identical column-printing code.

### SQL Strategy: NOT IN vs EXISTS

A notable semantic difference:

- **V1**: Uses `EXISTS` with correlated subqueries and `NOT IN ('done', 'cancelled')` for blocker status check
- **V2**: Uses `IN` with uncorrelated subqueries and `NOT IN ('done', 'cancelled')` for blocker status check
- **V3**: Uses `EXISTS` with correlated subqueries and `IN ('open', 'in_progress')` for blocker status check

V1 and V2 check `blocker.status NOT IN ('done', 'cancelled')` -- this means any status OTHER than done/cancelled blocks (including hypothetical future statuses). V3 checks `blocker.status IN ('open', 'in_progress')` -- this is a positive match that only considers known blocking statuses. Both approaches are functionally equivalent given the current 4-status model but have different forward-compatibility implications. V1/V2 are more conservative (unknown statuses block); V3 is more permissive (only known statuses block).

### Code Quality

**V1:**
- Excellent DRY: 27 implementation lines by reusing `cmdListFiltered`
- Unexported `blockedQuery` constant -- appropriate for package-internal use
- No new types or functions beyond the SQL constant and one-line handler
- SQL uses `EXISTS` (correlated subquery) -- generally preferred for "existence" checks
- `cmdBlocked` accepts `args` parameter but ignores it -- minor inconsistency but matches `cmdReady`'s signature

**V2:**
- Good DRY: blocked integrates into the existing `runList` function
- Exported `BlockedSQL` constant alongside `ReadySQL` -- good co-location
- Introduces `listFlags` struct to replace the boolean `ready` flag -- proper refactoring
- `if/else if` chain means `--ready` and `--blocked` are mutually exclusive (ready wins) -- not explicitly documented
- SQL uses `IN` (uncorrelated subqueries) -- functionally equivalent but arguably less performant on large datasets since it materializes the subquery result
- Also exposes `tick list --blocked` as a first-class feature

**V3:**
- Lowest DRY: full `runBlocked()` (118 lines) duplicates output formatting from `runReady()`
- Exports `BlockedCondition` as a SQL fragment (WHERE clause without SELECT/ORDER BY) -- more composable than a full query
- Separate `queryBlockedTasks()` function -- testable in isolation
- Reuses `taskRow` struct from `ready.go` -- good type reuse
- Adds `blocked` to help text in `printUsage()` -- only version to do this
- Column formatting uses `%-12s %-12s %-4s %s\n` (space-separated) vs V2's `%-12s%-12s%-4s%s\n` (no separators) -- V3 produces more readable output

### Test Quality

#### V1 Test Functions (10 subtests in `TestBlockedCommand`)

1. `returns open task blocked by open dep` -- creates tasks via CLI, checks blocked appears
2. `returns open task blocked by in_progress dep` -- starts blocker, checks blocked appears
3. `returns parent with open children` -- creates parent+child, checks parent appears
4. `excludes task when all blockers done/cancelled` -- marks done, checks excluded
5. `excludes in_progress/done/cancelled from output` -- transitions tasks, checks excluded
6. `returns empty when no blocked tasks` -- creates free task, checks "No tasks found."
7. `outputs IDs only with --quiet` -- checks all lines start with "tick-"
8. `cancel unblocks single dependent` -- verifies blocked->cancel->ready transition
9. `cancel unblocks multiple dependents` -- two dependents unblocked
10. `cancel does not unblock dependent still blocked by another` -- partial unblock

**V1 test approach**: Integration-style. Uses `initTickDir()`, `createTask()`, `runCmd()` helpers that execute the actual CLI. Task IDs are dynamically generated (extracted from create output), making tests more realistic but less deterministic for ordering.

**Missing from V1**: No ordering test. No explicit column alignment test. No test for parent with in_progress children. No separate tests for done vs cancelled blocker exclusion.

#### V2 Test Functions (16 subtests across 3 top-level functions)

`TestBlockedQuery` (12 subtests):
1. `it returns open task blocked by open dep`
2. `it returns open task blocked by in_progress dep`
3. `it returns parent with open children`
4. `it returns parent with in_progress children`
5. `it excludes task when all blockers done or cancelled`
6. `it excludes in_progress tasks from output`
7. `it excludes done and cancelled from output`
8. `it returns empty when no blocked tasks`
9. `it orders by priority ASC then created ASC` -- 4 blocked tasks with varying priorities
10. `it outputs aligned columns via tick blocked` -- checks header, column positions
11. `it prints 'No tasks found.' when empty`
12. `it outputs IDs only with --quiet` -- checks exact IDs and count

`TestBlockedViaListFlag` (1 subtest):
13. `it works via list --blocked flag` -- verifies `tick list --blocked` works

`TestCancelUnblocksDependents` (3 subtests):
14. `cancel unblocks single dependent -- moves to ready` -- 4-step: blocked, cancel, ready, not-blocked
15. `cancel unblocks multiple dependents`
16. `cancel does not unblock dependent still blocked by another` -- also verifies not in ready

**V2 test approach**: Unit-style. Uses `taskJSONL()` and `setupTickDirWithContent()` to create JSONL content directly, then constructs `App` with injected `strings.Builder` for stdout. Creates a new `App` per step in cancel-unblocks tests. Test naming follows "it does X" convention.

**Unique to V2**: `TestBlockedViaListFlag` tests `tick list --blocked`. Column alignment test checks actual position indices.

#### V3 Test Functions (17 subtests across 3 top-level functions)

`TestBlockedQuery` (13 subtests):
1. `it returns open task blocked by open dep`
2. `it returns open task blocked by in_progress dep`
3. `it returns parent with open children`
4. `it returns parent with in_progress children`
5. `it excludes task when all blockers done`
6. `it excludes task when all blockers cancelled`
7. `it excludes in_progress from output`
8. `it excludes done from output`
9. `it excludes cancelled from output`
10. `it returns empty when no blocked tasks`
11. `it orders by priority ASC then created ASC`

`TestBlockedCommand` (3 subtests):
12. `it outputs aligned columns via tick blocked`
13. `it prints 'No tasks found.' when empty`
14. `it outputs IDs only with --quiet`

`TestCancelUnblocksDependents` (3 subtests):
15. `cancel unblocks single dependent - moves to ready`
16. `cancel unblocks multiple dependents`
17. `cancel does not unblock dependent still blocked by another`

**V3 test approach**: Unit-style, similar to V2. Uses `setupTickDir()` and `setupTaskFull()` helpers. Uses `bytes.Buffer` for stdout/stderr. Reuses single App instance across steps in cancel tests (with `stdout.Reset()`).

**Unique to V3**: Separate tests for done vs cancelled blocker exclusion (tests 5 and 6). Separate tests for excluding done, cancelled, and in_progress from output (tests 7, 8, 9). More granular decomposition of exclusion scenarios.

### Test Gaps Analysis

| Test Scenario | V1 | V2 | V3 |
|---------------|-----|-----|-----|
| Blocked by open dep | YES | YES | YES |
| Blocked by in_progress dep | YES | YES | YES |
| Parent with open children | YES | YES | YES |
| Parent with in_progress children | NO | YES | YES |
| Excludes when all blockers done | YES (combined) | YES (combined) | YES (separate) |
| Excludes when all blockers cancelled | YES (combined) | YES (combined) | YES (separate) |
| Excludes in_progress from output | YES (combined) | YES | YES |
| Excludes done from output | YES (combined) | YES (combined) | YES (separate) |
| Excludes cancelled from output | NO | YES (combined) | YES (separate) |
| No blocked tasks -> empty message | YES | YES (x2) | YES (x2) |
| Priority/created ordering | NO | YES | YES |
| Column alignment | NO | YES (position check) | YES (header only) |
| Quiet mode IDs only | YES (prefix check) | YES (exact values) | YES (exact values + no headers) |
| `list --blocked` flag | NO | YES | NO |
| Cancel single unblock | YES | YES | YES |
| Cancel multiple unblock | YES | YES | YES |
| Partial unblock | YES | YES | YES |
| Cancel: verify no longer in blocked | NO | YES | YES |
| Cancel partial: verify not in ready | NO | YES | YES |
| Blocker not in blocked output | NO | YES | YES |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 | 3 | 3 |
| Lines added | 204 | 562 | 651 |
| Impl LOC (blocked logic only) | 27 | ~43 (additions to list.go) | 118 |
| Test LOC | 175 | 524 | 530 |
| Test functions (t.Run) | 10 | 16 | 17 |
| Top-level test functions | 1 | 3 | 3 |

## Verdict

**V2 is the best implementation.**

**Rationale:**

1. **Architecture**: V2 is the only version that properly integrates blocked as a list filter (`tick list --blocked`), matching the specification's description that `tick blocked` = `list --blocked`. V1 and V3 treat it as a standalone command only. V2 also includes a test (`TestBlockedViaListFlag`) proving this pathway works.

2. **DRY principle**: V2 achieves full code reuse by routing through `runList`. V1 achieves similar reuse through `cmdListFiltered`, but V3 duplicates the full output-formatting logic (column headers, quiet mode, empty message) across `runReady()` and `runBlocked()`.

3. **SQL co-location**: V2 places `BlockedSQL` alongside `ReadySQL` in `list.go`, making the inverse relationship obvious. V1 and V3 separate them into different files.

4. **Test quality**: V2 has 16 tests covering all acceptance criteria including ordering, column alignment (with position index checking), quiet mode (with exact value assertions), and `list --blocked`. V1 has only 10 tests and is missing ordering, column alignment, and parent-with-in_progress-children tests. V3 has 17 tests with the most granular exclusion coverage but lacks the `list --blocked` test.

5. **Refactoring quality**: V2 properly refactors the `parseListFlags` function from a single boolean to a `listFlags` struct, anticipating future filter flags.

**V3 is a close second** -- it has the most thorough test decomposition (separate done vs cancelled tests) and introduces composable SQL fragments (`BlockedCondition`). However, its output formatting duplication and lack of `list --blocked` integration are drawbacks.

**V1 is third** -- it achieves the task with minimal code but has significant test coverage gaps (no ordering test, no column alignment test) and doesn't expose `list --blocked`.
