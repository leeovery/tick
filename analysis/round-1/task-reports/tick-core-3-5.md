# Task tick-core-3-5: tick list filter flags -- --ready, --blocked, --status, --priority

## Task Summary

Wire ready/blocked queries into `tick list` as flags, plus add `--status` and `--priority` filters to complete the list command to full spec.

**Requirements:**
- Four flags on `list`: `--ready` (bool), `--blocked` (bool), `--status <s>`, `--priority <p>`
- `--ready`/`--blocked` mutually exclusive -- error if both
- `--status` validates: open, in_progress, done, cancelled
- `--priority` validates: 0-4
- Filters combine as AND
- Contradictory combos (e.g., `--status done --ready`) produce empty result, no error
- No filters = all tasks (backward compatible)
- Reuses ReadyQuery/BlockedQuery from 3-3/3-4
- Output: aligned columns, `--quiet` IDs only

**Acceptance Criteria:**
1. `list --ready` = same as `tick ready`
2. `list --blocked` = same as `tick blocked`
3. `--status` filters by exact match
4. `--priority` filters by exact match
5. Filters AND-combined
6. `--ready` + `--blocked` produces error
7. Invalid values produce error with valid options
8. No matches produces `No tasks found.`, exit 0
9. `--quiet` outputs filtered IDs
10. Backward compatible (no filters = all)
11. Reuses query functions

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| `list --ready` = same as `tick ready` | PARTIAL -- uses same `readyQuery` SQL and calls `cmdListFiltered`, but no test comparing output equality | PASS -- uses `readyWhere` fragment in `buildListQuery`; no explicit equality test but same WHERE clause by construction | PASS -- uses `ReadyCondition` in `queryReadyTasksWithFilters`; has explicit test `"it matches tick ready output"` comparing outputs |
| `list --blocked` = same as `tick blocked` | PARTIAL -- uses same `blockedQuery` SQL via `cmdListFiltered`; no equality test | PASS -- uses `blockedWhere` fragment; same construction guarantee; no explicit equality test | PASS -- uses `BlockedCondition`; has explicit test `"it matches tick blocked output"` comparing outputs |
| `--status` filters by exact match | PASS -- validated via `validStatuses` map, SQL uses `t.status = '<value>'` | PASS -- validated via `validStatuses` map, SQL uses parameterized `status = ?` | PASS -- validated in `validateListFlags`, SQL uses parameterized `t.status = ?` |
| `--priority` filters by exact match | PASS -- validated via `task.ValidatePriority`, SQL uses `t.priority = <value>` | PASS -- validated inline (0-4 range), SQL uses parameterized `priority = ?` | PASS -- validated in `validateListFlags`, SQL uses parameterized `t.priority = ?` |
| Filters AND-combined | PASS -- `applyListFilters` joins conditions with `AND` | PASS -- `buildListQuery` joins `where` slice with `AND` | PASS -- `queryListTasks` / `queryReadyTasksWithFilters` / `queryBlockedTasksWithFilters` append `AND` conditions |
| `--ready` + `--blocked` produces error | PASS -- checked after flag parsing; error contains "mutually exclusive" | PASS -- checked in `parseListFlags`; error contains "--ready" and "--blocked" | PASS -- checked in `validateListFlags`; error contains "mutually exclusive" |
| Invalid values produce error with valid options | PASS -- invalid status: `"invalid status '%s'. Valid: open, in_progress, done, cancelled"`; invalid priority delegated to `task.ValidatePriority` | PASS -- invalid status: `"invalid status '%s' -- valid values: open, in_progress, done, cancelled"`; invalid priority: `"invalid priority %d -- valid range: 0-4"` | PASS -- invalid status: `"invalid status '%s': must be one of open, in_progress, done, cancelled"`; invalid priority: `"invalid priority %d: must be between 0 and 4"` |
| No matches: `No tasks found.`, exit 0 | PASS -- tested; relies on `cmdListFiltered` which prints the message | PASS -- tested; `runList` checks `len(rows) == 0` | PASS -- tested; `runList` checks `len(tasks) == 0` |
| `--quiet` outputs filtered IDs | PASS -- tested (basic check: lines start with "tick-") | PASS -- tested (exact ID match + exact line count verification) | PASS -- tested (exact ID match + no headers + no titles + line count) |
| Backward compatible (no filters = all) | PASS -- tested | PASS -- tested (pre-existing + new test) | PASS -- tested |
| Reuses query functions | PASS -- reuses `readyQuery` and `blockedQuery` string constants directly | PASS -- refactored `ReadySQL`/`BlockedSQL` to compose from `readyWhere`/`blockedWhere` fragments; builds queries from fragments | PASS -- uses `ReadyCondition`/`BlockedCondition` in dedicated `queryReadyTasksWithFilters`/`queryBlockedTasksWithFilters` functions |

## Implementation Comparison

### Approach

**V1: Monolithic inline parsing + string surgery on SQL**

V1 takes the simplest path: flag parsing is inline in `cmdList`, using local variables rather than a struct:

```go
var (
    filterReady    bool
    filterBlocked  bool
    filterStatus   string
    filterPriority = -1 // sentinel: no filter
)
```

The core approach delegates to `cmdListFiltered` (a pre-existing shared function from 3-3/3-4 used by `cmdReady` and `cmdBlocked`). V1 selects the base SQL query string (`readyQuery`, `blockedQuery`, or a bare `SELECT`), then passes it to `applyListFilters` which performs string manipulation -- finding the `ORDER BY` position via `strings.LastIndex` and injecting `AND` conditions:

```go
orderIdx := strings.LastIndex(baseQuery, "ORDER BY")
if orderIdx > 0 {
    return baseQuery[:orderIdx] + "AND " + extra + "\n" + baseQuery[orderIdx:]
}
```

The SQL values are interpolated directly using `fmt.Sprintf("t.status = '%s'", status)` -- string interpolation of user input into SQL, not parameterized queries. Priority validation is delegated to `task.ValidatePriority(p)` from the task package.

**V2: Structured flag parsing + composable WHERE fragments + parameterized queries**

V2 refactors the entire SQL structure. The pre-existing `ReadySQL` and `BlockedSQL` constants are decomposed into `readyWhere` and `blockedWhere` WHERE clause fragments, with the full queries reconstructed via constant concatenation:

```go
const readyWhere = `status = 'open'
  AND id NOT IN (...)`

const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE ` + readyWhere + `
ORDER BY priority ASC, created ASC`
```

Flag parsing is extracted into a `parseListFlags` function returning a `listFlags` struct with a `hasPri bool` sentinel for distinguishing "priority 0" from "no priority filter". Validation happens within `parseListFlags`. Query building is in a pure function `buildListQuery(flags) (string, []interface{})` that constructs WHERE clauses from fragments and uses `?` parameterized placeholders:

```go
if flags.status != "" {
    where = append(where, "status = ?")
    queryArgs = append(queryArgs, flags.status)
}
```

The `runList` function then calls `db.Query(querySQL, queryArgs...)` with the parameters.

**V3: Separate parse/validate + dedicated query functions per filter mode + parameterized queries**

V3 separates flag parsing (`parseListFlags`) from validation (`validateListFlags`) into two distinct functions. The parsing function returns three values including remaining args: `(listFlags, []string, error)`. The flags struct uses `*int` for priority (nil = no filter) rather than a boolean sentinel:

```go
type listFlags struct {
    ready    bool
    blocked  bool
    status   string // empty = no filter
    priority *int   // nil = no filter
}
```

V3 has three separate query functions: `queryListTasks`, `queryReadyTasksWithFilters`, and `queryBlockedTasksWithFilters`. Each builds its own SQL with `ReadyCondition`/`BlockedCondition` string constants and appends `AND` conditions with parameterized `?` placeholders. The `runList` function returns `int` (exit code) rather than `error`, writing errors to `a.Stderr` directly. This means V3's `list.go` handles its own output formatting rather than delegating to a shared function.

### Code Quality

**SQL Injection Safety**

V1 has a SQL injection vulnerability. User-supplied status is interpolated directly:
```go
conditions = append(conditions, fmt.Sprintf("t.status = '%s'", status))
```
While the status is validated against `validStatuses` before reaching this point, the pattern is dangerous. If validation were bypassed or additional fields added without validation, injection would occur.

V2 and V3 both use parameterized queries (`?` placeholders with `queryArgs`), which is the correct Go/SQL practice.

**DRY / Code Reuse**

V1 reuses `readyQuery`/`blockedQuery` string constants and `cmdListFiltered` -- the most direct reuse, with zero duplication of SQL between `ready`/`blocked`/`list` commands.

V2 refactors the SQL into composable fragments (`readyWhere`/`blockedWhere`) that serve both the standalone `ReadySQL`/`BlockedSQL` constants and the filter builder. The `buildListQuery` function wraps fragments in parentheses when combining:
```go
where = append(where, "("+readyWhere+")")
```
This is architecturally clean but required modifying the pre-existing SQL constants.

V3 has the most code duplication. `queryReadyTasksWithFilters` and `queryBlockedTasksWithFilters` are nearly identical 30-line functions, differing only in which condition constant they use. The row-scanning loop is repeated three times (once in each query function):
```go
var tasks []taskRow
for rows.Next() {
    var t taskRow
    if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
        return nil, err
    }
    tasks = append(tasks, t)
}
```

**Type Safety for Priority Sentinel**

V1 uses `filterPriority = -1` as a sentinel for "no filter". This works but is a C-style pattern; -1 is a valid int.

V2 uses a separate `hasPri bool` field alongside `priority int`. Slightly verbose but explicit.

V3 uses `*int` (pointer to int). Idiomatic Go for optional values. `nil` vs `&p` is clear and type-safe:
```go
if flags.priority != nil {
    conditions = append(conditions, "t.priority = ?")
    args = append(args, *flags.priority)
}
```

**Error Handling Style**

V1 returns `error` from `cmdList`, matching the pre-existing pattern (`cmdReady`, `cmdBlocked`). The caller formats the error.

V2 returns `error` from `runList`, also matching its codebase's convention.

V3 returns `int` from `runList`, handling error formatting internally with `fmt.Fprintf(a.Stderr, "Error: %s\n", err)`. This means V3's function does its own stderr output rather than letting the caller decide, which gives more control at the cost of testability -- V3's tests must check `stderr.String()` rather than `err.Error()`.

**Separation of Concerns**

V1: Parsing and validation are interleaved inline in `cmdList`. No separation.

V2: Parsing and validation are combined in `parseListFlags`. Query building is in `buildListQuery`. Two-layer separation.

V3: Parsing in `parseListFlags`, validation in `validateListFlags`, querying split across three functions. Three-layer separation but with code duplication.

**Naming**

V1: `filterReady`, `filterBlocked`, `filterStatus`, `filterPriority` -- local variables with `filter` prefix, clear purpose.

V2: `listFlags` struct with `ready`, `blocked`, `status`, `priority`, `hasPri` fields. Clean, concise. `readyWhere`/`blockedWhere` are clear fragment names. `buildListQuery` is descriptive.

V3: `listFlags` struct identical naming to V2 (minus `hasPri`). `ReadyCondition`/`BlockedCondition` are exported constants (capitalized). `queryReadyTasksWithFilters`/`queryBlockedTasksWithFilters` are verbose but descriptive.

### Test Quality

**V1 Test Functions** (file: `internal/cli/list_filter_test.go`, 213 LOC):

1. `TestListFilters/filters to ready tasks with --ready` -- Creates tasks via `createTask`/`extractID` CLI helpers, checks ready task appears, blocked task absent
2. `TestListFilters/filters to blocked tasks with --blocked` -- Same setup, checks blocked appears, ready absent
3. `TestListFilters/filters by --status for all 4 values` -- Single test checking all 4 status values (open, in_progress, done, cancelled) by transitioning tasks via CLI commands
4. `TestListFilters/filters by --priority` -- Creates tasks with different priorities, checks filtering
5. `TestListFilters/combines --ready with --priority` -- AND combination test
6. `TestListFilters/combines --status with --priority` -- AND combination test with status transition
7. `TestListFilters/errors when --ready and --blocked both set` -- Error case, checks exit code 1 and "mutually exclusive" message
8. `TestListFilters/errors for invalid status value` -- Checks exit code 1 and "invalid status" in stderr
9. `TestListFilters/errors for invalid priority value` -- Uses priority 9, checks exit code 1
10. `TestListFilters/returns 'No tasks found.' when no matches` -- Filter for done when only open exists
11. `TestListFilters/outputs IDs only with --quiet after filtering` -- Checks lines start with "tick-"
12. `TestListFilters/returns all tasks with no filters` -- Two tasks, checks both appear

V1 uses integration-style tests via `initTickDir`/`createTask`/`runCmd` helpers that run the actual CLI binary. Tests exercise real end-to-end behavior but are slower and less isolated.

**V2 Test Functions** (file: `internal/cli/list_test.go`, 674 LOC, including pre-existing tests):

Pre-existing tests (not from this task):
- `TestListCommand/it lists all tasks with aligned columns`
- `TestListCommand/it lists tasks ordered by priority then created date`
- `TestListCommand/it prints 'No tasks found.' when no tasks exist`
- `TestListCommand/it prints only task IDs with --quiet flag on list`
- `TestListCommand/it executes through storage engine read flow (shared lock, freshness check)`

New tests added for this task in `TestListFilterFlags`:
1. `TestListFilterFlags/it filters to ready tasks with --ready` -- Uses shared `mixedContent` fixture; verifies 3 ready tasks appear, blocked/in_progress/done/cancelled absent
2. `TestListFilterFlags/it filters to blocked tasks with --blocked` -- Verifies blocked task appears, ready/in_progress absent
3. `TestListFilterFlags/it filters by --status open` -- Verifies open tasks appear, non-open absent (checks all 5 non-matching task types)
4. `TestListFilterFlags/it filters by --status in_progress` -- Dedicated sub-test per status
5. `TestListFilterFlags/it filters by --status done` -- Dedicated sub-test per status
6. `TestListFilterFlags/it filters by --status cancelled` -- Dedicated sub-test per status
7. `TestListFilterFlags/it filters by --priority` -- Priority 1 filter, checks positive and negative matches
8. `TestListFilterFlags/it combines --ready with --priority` -- Verifies AND semantics
9. `TestListFilterFlags/it combines --status with --priority` -- Verifies AND semantics (open + priority 2)
10. `TestListFilterFlags/it errors when --ready and --blocked both set` -- Checks `err` not nil, mentions both flags
11. `TestListFilterFlags/it errors for invalid status value` -- Checks error message mentions "invalid" and lists valid values
12. `TestListFilterFlags/it errors for invalid priority value` -- Checks error mentions range 0-4
13. `TestListFilterFlags/it errors for non-numeric priority value` -- Tests "high" as priority input
14. `TestListFilterFlags/it returns 'No tasks found.' when no matches` -- Exact string comparison
15. `TestListFilterFlags/it returns empty for contradictory filters without error` -- `--status done --ready` returns empty, no error
16. `TestListFilterFlags/it outputs IDs only with --quiet after filtering` -- Verifies exact IDs, exact line count (2), exact ordering
17. `TestListFilterFlags/it maintains deterministic ordering` -- Runs query twice, compares output equality
18. `TestListCommand/it returns all tasks with no filters` -- Added to pre-existing test suite

V2 uses unit-style tests via `NewApp()` with `setupTickDirWithContent`/`taskJSONL` fixtures -- constructs JSONL content and pre-populates the tick directory. Uses shared `mixedContent` factory function for consistent test data. Tests status values in dedicated sub-tests (4 separate tests vs V1's single combined test).

**V3 Test Functions** (file: `internal/cli/list_test.go`, 611 LOC, all new):

1. `TestListFilters/it filters to ready tasks with --ready` -- Uses `setupTaskFull` to create tasks with specific IDs
2. `TestListFilters/it filters to blocked tasks with --blocked` -- Ready/blocked pair
3. `TestListFilters/it filters by --status open` -- Separate test per status value
4. `TestListFilters/it filters by --status in_progress`
5. `TestListFilters/it filters by --status done`
6. `TestListFilters/it filters by --status cancelled`
7. `TestListFilters/it filters by --priority` -- 3 tasks with different priorities (0, 2, 4)
8. `TestListFilters/it combines --ready with --priority` -- 3 tasks: ready-p1, ready-p2, blocked-p1
9. `TestListFilters/it combines --status with --priority` -- 3 tasks: done-p1, done-p2, open-p1
10. `TestListFilters/it errors when --ready and --blocked both set` -- Checks exit code, stderr mentions both flags, "mutually exclusive"
11. `TestListFilters/it errors for invalid status value` -- Checks all 4 valid statuses listed in error
12. `TestListFilters/it errors for invalid priority value` -- Uses priority 5 (V2 uses 9), checks 0 and 4 mentioned
13. `TestListFilters/it errors for non-numeric priority value` -- Tests "high"
14. `TestListFilters/it returns 'No tasks found.' when no matches` -- Filter done, only open exists
15. `TestListFilters/it returns empty result for contradictory filters (--status done --ready)` -- Explicit contradictory test
16. `TestListFilters/it outputs IDs only with --quiet after filtering` -- Checks exact IDs, no headers, no titles, line count
17. `TestListFilters/it returns all tasks with no filters` -- 4 different-status tasks
18. `TestListFilters/it maintains deterministic ordering` -- Verifies exact line-by-line ordering (priority 0, 2, 2, 4 with created-date tiebreaker)
19. `TestListFilters/it matches tick ready output` -- **UNIQUE**: directly compares `list --ready` output to `tick ready` output
20. `TestListFilters/it matches tick blocked output` -- **UNIQUE**: directly compares `list --blocked` output to `tick blocked` output

V3 uses `setupTaskFull` with explicit IDs and full field control. Uses `bytes.Buffer` for stdout/stderr and constructs `App` directly. Each test creates fresh data rather than sharing a factory.

**Test Coverage Diff:**

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| --ready filter | YES | YES | YES |
| --blocked filter | YES | YES | YES |
| --status open | YES (combined) | YES (dedicated) | YES (dedicated) |
| --status in_progress | YES (combined) | YES (dedicated) | YES (dedicated) |
| --status done | YES (combined) | YES (dedicated) | YES (dedicated) |
| --status cancelled | YES (combined) | YES (dedicated) | YES (dedicated) |
| --priority filter | YES | YES | YES |
| --ready + --priority AND | YES | YES | YES |
| --status + --priority AND | YES | YES | YES |
| --ready + --blocked mutual exclusion | YES | YES | YES |
| Invalid status error | YES | YES | YES |
| Invalid priority (out of range) | YES | YES | YES |
| Non-numeric priority | NO | YES | YES |
| No matches: "No tasks found." | YES | YES | YES |
| Contradictory filters (empty, no error) | NO | YES | YES |
| --quiet with filtering | YES (basic) | YES (exact IDs/count) | YES (exact IDs/count/no headers) |
| No filters = all tasks | YES | YES | YES |
| Deterministic ordering | NO | YES (output equality) | YES (exact line-by-line position) |
| `list --ready` == `tick ready` | NO | NO | YES |
| `list --blocked` == `tick blocked` | NO | NO | YES |
| Pre-existing list tests preserved | N/A (separate file) | YES (same file) | N/A (new file) |

**Unique to V1:** None. V1 has the fewest tests.

**Unique to V2 (vs V1):** Non-numeric priority test, contradictory filters test, deterministic ordering test.

**Unique to V3 (vs V1 and V2):** `list --ready` == `tick ready` output equality test, `list --blocked` == `tick blocked` output equality test. Also V3's deterministic ordering test verifies exact row positions (line-by-line) while V2 only checks two-run equality.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4 | 5 |
| Lines added | 293 | 609 | 868 |
| Lines removed | 44 | 31 | 25 |
| Impl LOC (list.go total) | 279 | 225 | 295 |
| Test LOC | 213 | 674 (487 new + 187 pre-existing) | 611 |
| Test functions (new) | 12 | 18 (13 filter + 5 pre-existing modified) | 20 |
| Test file | list_filter_test.go (new) | list_test.go (extended) | list_test.go (new) |

## Verdict

**V2 is the best implementation.**

**V2 wins on architecture.** The `readyWhere`/`blockedWhere` fragment decomposition is the most elegant solution. It allows the standalone `ReadySQL`/`BlockedSQL` to be composed via `const` concatenation (`ReadySQL = "...WHERE " + readyWhere + " ORDER BY..."`), while the `buildListQuery` pure function reuses the same fragments with parameterized queries. This is zero-duplication: the SQL logic exists in exactly one place. V1 does string surgery on complete queries (fragile), and V3 duplicates the row-scanning loop three times across `queryListTasks`, `queryReadyTasksWithFilters`, and `queryBlockedTasksWithFilters`.

**V2 wins on SQL safety.** V2 uses parameterized queries throughout. V1 uses `fmt.Sprintf("t.status = '%s'", status)` -- a SQL injection anti-pattern even if pre-validated. V3 also uses parameterized queries, tying with V2 here.

**V2 wins on code economy.** V2's `list.go` is 225 LOC vs V3's 295 LOC, while achieving the same functionality. The single `buildListQuery` function replaces V3's three separate query functions. V1's 279 LOC includes the `cmdShow` function, so the filter logic is actually quite compact, but the string surgery approach (`applyListFilters`) is brittle.

**V3 wins on test quality but V2 is close.** V3 has two tests V2 lacks (`list --ready` == `tick ready` output equality, `list --blocked` == `tick blocked` output equality), which directly verify the acceptance criterion. V3's deterministic ordering test also verifies exact row positions rather than just consistency. However, V2 has more rigorous `--quiet` testing (exact ID values in exact order) and the same edge case coverage otherwise. The margin is narrow.

**V1 is the weakest.** It lacks the contradictory filters test, deterministic ordering test, non-numeric priority test, and output equality tests. Its SQL interpolation is a code quality issue. Its test helper approach (integration via CLI binary) means tests are end-to-end but harder to maintain and slower.

**Ranking: V2 > V3 > V1.**

V2 achieves the best balance: composable SQL fragments, parameterized queries, compact code, strong test coverage, and the cleanest architecture. V3 has superior test coverage (especially the output equality tests) and better type safety (`*int` for priority), but the code duplication in its three query functions is a notable weakness. V1 is functional but has SQL safety concerns and the thinnest test suite.
