# Task tick-core-3-5: tick list filter flags -- --ready, --blocked, --status, --priority

## Task Summary

Wire ready/blocked queries into `tick list` as flags, plus add `--status` and `--priority` filters. The task requires:

- Four flags on `list`: `--ready` (bool), `--blocked` (bool), `--status <s>`, `--priority <p>`
- `--ready`/`--blocked` mutually exclusive -- error if both
- `--status` validates: open, in_progress, done, cancelled
- `--priority` validates: 0-4
- Filters combine as AND
- Contradictory combos (e.g., `--status done --ready`) -- empty result, no error
- No filters = all tasks (backward compatible)
- Reuses ReadyQuery/BlockedQuery from 3-3/3-4
- Output: aligned columns, `--quiet` IDs only

### Acceptance Criteria

1. `list --ready` = same as `tick ready`
2. `list --blocked` = same as `tick blocked`
3. `--status` filters by exact match
4. `--priority` filters by exact match
5. Filters AND-combined
6. `--ready` + `--blocked` -- error
7. Invalid values -- error with valid options
8. No matches -- `No tasks found.`, exit 0
9. `--quiet` outputs filtered IDs
10. Backward compatible (no filters = all)
11. Reuses query functions

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| 1. `list --ready` = same as `tick ready` | PASS. Wraps ReadyQuery constant as subquery: `SELECT ... FROM (ReadyQuery) AS ready WHERE 1=1`. Test confirms ready tasks appear, blocked/done excluded. | PASS. Calls `ReadyConditions()` to get SQL WHERE clauses and appends them as AND conditions in single query. Test confirms ready tasks appear, blocked excluded. |
| 2. `list --blocked` = same as `tick blocked` | PASS. Wraps BlockedQuery constant as subquery: `SELECT ... FROM (BlockedQuery) AS blocked WHERE 1=1`. Test confirms blocked tasks appear, unblocked excluded. | PASS. Calls `BlockedConditions()` to get SQL WHERE clauses and appends them inline. Test confirms blocked tasks appear, unblocked excluded. |
| 3. `--status` filters by exact match | PASS. Table-driven test covers all 4 status values in a single subtest with `for _, tt := range tests`. | PASS. Four individual subtests each testing one status value (`--status open`, `--status in_progress`, `--status done`, `--status cancelled`). |
| 4. `--priority` filters by exact match | PASS. Tests with priority 2, verifies 1 and 3 excluded. | PASS. Tests with priority 2, verifies 1 and 3 excluded. |
| 5. Filters AND-combined | PASS. Tests `--ready --priority 1` and `--status open --priority 1`. Both verify AND semantics. | PASS. Tests `--ready --priority 1` and `--status open --priority 1`. Both verify AND semantics. |
| 6. `--ready` + `--blocked` -- error | PASS. Test asserts exit code 1 and error message contains "ready" and "blocked". | PASS. Test asserts exit code 1 and error message contains exact "mutually exclusive" text. |
| 7. Invalid values -- error with valid options | PASS. Tests invalid status (checks for "open" and "done" in error), invalid priority 9 (checks "0" and "4"), non-numeric priority. | PASS. Tests invalid status (checks "invalid status" and all 4 valid options), invalid priority 5 (checks "invalid priority" and "0-4"), non-numeric priority (checks "invalid priority"). |
| 8. No matches -- `No tasks found.`, exit 0 | PASS. Test with done-only tasks, `--status open`, expects "No tasks found." and exit 0. | PASS. Identical scenario, exact string match `"No tasks found.\n"`. |
| 9. `--quiet` outputs filtered IDs | PASS. `--quiet` with `--priority 1` produces single ID line. | PASS. `--quiet` with `--status open` produces two ID lines. |
| 10. Backward compatible (no filters = all) | PASS. Test with open, in_progress, done tasks -- all appear. | PASS. Test with open, done, in_progress tasks -- all appear. |
| 11. Reuses query functions | PASS. Wraps `ReadyQuery` and `BlockedQuery` string constants as subqueries. | PARTIAL. Does not reuse the `ReadyQuery`/`BlockedQuery` constants. Instead uses `ReadyConditions()`/`BlockedConditions()` helper functions that return slices of WHERE clause fragments. This is a different form of reuse -- the conditions are shared, but the full query is not. The V6 codebase already has these helpers from a prior refactoring, so this is valid reuse of the underlying logic even though it doesn't reference the original constants. |

## Implementation Comparison

### Approach

**V5: Subquery wrapping of existing query constants**

V5 takes the existing `ReadyQuery` and `BlockedQuery` string constants (full `SELECT ... FROM tasks t WHERE ... ORDER BY ...` statements) and wraps them as subqueries:

```go
// V5 list.go lines 110-136 (buildReadyFilterQuery)
func buildReadyFilterQuery(f listFilters) (string, []interface{}) {
    q := `SELECT id, status, priority, title FROM (` + ReadyQuery + `) AS ready WHERE 1=1`
    var params []interface{}
    if f.status != "" {
        q += ` AND status = ?`
        params = append(params, f.status)
    }
    if f.hasPri {
        q += ` AND priority = ?`
        params = append(params, f.priority)
    }
    return q, params
}
```

The approach has three separate builder functions: `buildReadyFilterQuery`, `buildBlockedFilterQuery`, and `buildSimpleFilterQuery`. Routing between them is via `buildListQuery`:

```go
// V5 list.go lines 95-106
func buildListQuery(f listFilters) (string, []interface{}) {
    if f.ready {
        return buildReadyFilterQuery(f)
    }
    if f.blocked {
        return buildBlockedFilterQuery(f)
    }
    return buildSimpleFilterQuery(f)
}
```

This produces nested SQL like: `SELECT id, status, priority, title FROM (SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE ... ORDER BY ...) AS ready WHERE 1=1 AND status = ? AND priority = ?`

**V6: Inline condition composition**

V6 uses `ReadyConditions()` and `BlockedConditions()` functions (from `query_helpers.go`) that return `[]string` slices of SQL WHERE fragments. It composes a single flat query:

```go
// V6 list.go lines 199-239 (buildListQuery)
func buildListQuery(f ListFilter, descendantIDs []string) (string, []interface{}) {
    var conditions []string
    var args []interface{}

    if f.Ready {
        conditions = append(conditions, ReadyConditions()...)
    }
    if f.Blocked {
        conditions = append(conditions, BlockedConditions()...)
    }
    if f.Status != "" {
        conditions = append(conditions, `t.status = ?`)
        args = append(args, f.Status)
    }
    if f.HasPriority {
        conditions = append(conditions, `t.priority = ?`)
        args = append(args, f.Priority)
    }
    // ... descendant filter ...
    query := `SELECT t.id, t.status, t.priority, t.title FROM tasks t`
    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }
    query += " ORDER BY t.priority ASC, t.created ASC"
    return query, args
}
```

This produces flat SQL: `SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE t.status = 'open' AND NOT EXISTS (...) AND NOT EXISTS (...) AND t.priority = ? ORDER BY ...`

**Assessment:** V6's approach is genuinely better. It produces simpler SQL (no nested subqueries), consolidates all query logic into a single function instead of three parallel builder functions, and leverages composable condition helpers. V5's subquery wrapping works but generates unnecessary SQL complexity and duplicates filter-appending logic across three functions.

**Flag parsing architecture:**

Both versions use nearly identical manual flag parsing loops. Key difference: V5 validates immediately inside the switch cases (e.g., `isValidStatus` inline), while V6 validates after the loop in a separate block:

```go
// V5 -- validates inline during parsing
case "--status":
    // ...
    f.status = args[i]
    if !isValidStatus(f.status) {
        return f, fmt.Errorf("invalid status %q - valid values: ...")
    }

// V6 -- validates after loop
if f.Status != "" {
    valid := map[string]bool{
        string(task.StatusOpen): true,
        // ...
    }
    if !valid[f.Status] {
        return f, fmt.Errorf("invalid status '%s': must be one of ...")
    }
}
```

V6's deferred validation is marginally better: it references `task.Status*` constants from the domain model rather than hardcoded strings, making it less likely to drift from the source of truth.

**Unknown flag handling:** V5 has an explicit `default:` case that returns an error for unknown flags. V6's switch has no `default:` case, silently ignoring unknown flags. V5's approach is more robust for catching user typos.

**Dispatch integration:** V5 passes `ctx.Args` to `parseListFlags` inside `runList`. V6 parses flags in `handleList` within `app.go` before calling `RunList`, achieving better separation of concerns (parsing vs. execution).

### Code Quality

**Naming:**

V5 uses unexported types: `listFilters` (struct), `listRow` (struct), `parseListFlags`, `buildListQuery`, etc. All appropriately scoped to the package.

V6 exports the filter type: `ListFilter` with exported fields (`Ready`, `Blocked`, `Status`, `Priority`, `HasPriority`). This is necessary because `parseListFlags` returns it to `app.go`'s `handleList`, which then passes it to `RunList` -- a public function. The export is justified by the architecture.

V5's `hasPri` vs V6's `HasPriority` -- V6's name is more descriptive.

**Error handling:**

Both versions handle all errors explicitly. Error messages differ in style:

```go
// V5
fmt.Errorf("invalid status %q - valid values: open, in_progress, done, cancelled", f.status)
fmt.Errorf("invalid priority %d - valid values: 0, 1, 2, 3, 4", p)

// V6
fmt.Errorf("invalid status '%s': must be one of open, in_progress, done, cancelled", f.Status)
fmt.Errorf("invalid priority '%d': must be 0-4", f.Priority)
```

V5 uses `%q` (quoted with Go syntax), V6 uses `'%s'` (single-quoted). Both are acceptable. V5 enumerates all priority values (0, 1, 2, 3, 4); V6 uses range notation (0-4). Both convey the information.

**Error wrapping:** V5 uses `fmt.Errorf("querying tasks: %w", err)` while V6 uses `fmt.Errorf("failed to query tasks: %w", err)`. Both comply with the skill requirement to propagate errors with `%w`.

**DRY:**

V5 has duplicated filter-appending code across `buildReadyFilterQuery`, `buildBlockedFilterQuery`, and `buildSimpleFilterQuery` -- each has nearly identical `if f.status != ""` and `if f.hasPri` blocks. This is a DRY violation.

V6 consolidates all filter logic into a single `buildListQuery` function. No duplication.

**Type safety:**

V6 references `task.StatusOpen`, `task.StatusInProgress`, etc. from the domain model for validation:

```go
// V6 list.go lines 66-71
valid := map[string]bool{
    string(task.StatusOpen):       true,
    string(task.StatusInProgress): true,
    string(task.StatusDone):       true,
    string(task.StatusCancelled):  true,
}
```

V5 uses a hardcoded string slice:

```go
// V5 list.go line 32
var validStatuses = []string{"open", "in_progress", "done", "cancelled"}
```

V6's approach is more type-safe and less likely to drift from the domain model.

**Documentation:** Both versions document all exported and significant unexported functions. V5 has doc comments on `buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildSimpleFilterQuery`, `isValidStatus`, `parseListFlags`, etc. V6 documents `parseListFlags`, `RunList`, `buildListQuery`, `queryDescendantIDs`. Both are compliant with the skill requirement.

### Test Quality

**V5 Test Functions** (in `list_test.go`, within `TestList`):

1. `it filters to ready tasks with --ready` -- ready/blocked/done task setup, verifies inclusion/exclusion
2. `it filters to blocked tasks with --blocked` -- blocker/blocked/ready setup
3. `it filters by --status (all 4 values)` -- table-driven subtest with 4 entries (open, in_progress, done, cancelled)
4. `it filters by --priority` -- three priorities, filters to priority 2
5. `it combines --ready with --priority` -- AND combination of ready + priority 1
6. `it combines --status with --priority` -- AND combination of status open + priority 1
7. `it errors when --ready and --blocked both set` -- mutual exclusion error
8. `it errors for invalid status value` -- status "invalid" produces error
9. `it errors for invalid priority value` -- priority 9 produces error
10. `it errors for non-numeric priority value` -- priority "abc" produces error
11. `it returns 'No tasks found.' when no matches` -- done-only tasks, filter for open
12. `it outputs IDs only with --quiet after filtering` -- quiet + priority filter
13. `it returns all tasks with no filters` -- backward compatibility
14. `it maintains deterministic ordering` -- priority + created-date ordering
15. `it handles contradictory filters with empty result not error` -- --status done --ready

Total: 15 test functions (including 4 status subtests within #3).

**V6 Test Functions** (in `list_filter_test.go`, within `TestListFilter`):

1. `it filters to ready tasks with --ready` -- blocker/dependent setup
2. `it filters to blocked tasks with --blocked` -- blocker/dependent setup
3. `it filters by --status open` -- open/done/ip setup
4. `it filters by --status in_progress` -- open/ip setup
5. `it filters by --status done` -- open/done setup
6. `it filters by --status cancelled` -- open/cancelled setup
7. `it filters by --priority` -- three priorities, filters to 2
8. `it combines --ready with --priority` -- ready + priority 1
9. `it combines --status with --priority` -- status open + priority 1
10. `it errors when --ready and --blocked both set` -- mutual exclusion
11. `it errors for invalid status value` -- "invalid" status, checks for all valid options
12. `it errors for invalid priority value` -- priority 5 (not 9), checks for "0-4"
13. `it errors for non-numeric priority value` -- "abc", checks for "invalid priority"
14. `it returns 'No tasks found.' when no matches` -- done-only, filter open
15. `it outputs IDs only with --quiet after filtering` -- quiet + status open (tests 2 IDs)
16. `it returns all tasks with no filters` -- backward compatibility
17. `it maintains deterministic ordering` -- 4 tasks, priority + created ordering
18. `contradictory filters return empty result no error` -- --status done --ready

Total: 18 test functions.

**Test Coverage Differences:**

- V6 tests each `--status` value in a separate subtest (4 tests) vs V5's table-driven approach (1 test with 4 subtests). Functionally equivalent coverage but V6 is more verbose.
- V6's `--quiet` test filters by `--status open` and checks two IDs (more comprehensive than V5's single-ID `--priority 1` test).
- V6's ordering test uses 4 tasks instead of 3, exercising both priority tiers more thoroughly.
- V6's invalid priority test uses 5 (just above valid range) vs V5's 9 -- both are valid but V6 is a better boundary test.
- V6's non-numeric priority test additionally checks the error message contains "invalid priority", whereas V5 only checks exit code.
- V6 uses a shared test helper `runList` from `list_show_test.go` that constructs an `App` directly, while V5 uses the top-level `Run` function. V6's approach tests through the real dispatch path (including `handleList` in `app.go`).

**Assertion Quality:**

V5 uses `strings.Contains` with `t.Errorf` for most checks. V6 also uses `strings.Contains` but with terser error messages (e.g., `t.Error("ready task should appear with --ready flag")` vs V5's `t.Errorf("expected ready task tick-aaaaaa in output, got %q", output)`). V5's assertions include the actual output in failure messages which aids debugging; V6's are shorter but less diagnostic.

V6's ordering test uses `strings.HasPrefix` for line matching, which is stricter than V5's `strings.Contains` for the same test.

**Table-driven tests:**

V5 uses table-driven tests for `--status` filtering (the only case with clear repetition). V6 uses individual subtests for each status value. The skill requires "table-driven tests with subtests" -- V5 is slightly more compliant here, though the difference is minor since V6's individual tests are still subtests of `TestListFilter`.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS. Code appears properly formatted. | PASS. Code appears properly formatted. |
| Handle all errors explicitly (no naked returns) | PASS. All errors checked and returned. | PASS. All errors checked and returned. |
| Write table-driven tests with subtests | PARTIAL. Uses table-driven for --status test only. Other tests are individual subtests. | PARTIAL. No table-driven tests. All individual subtests within `TestListFilter`. |
| Document all exported functions, types, and packages | PASS. All significant functions documented. `listFilters` unexported but documented. | PASS. `ListFilter`, `RunList`, `parseListFlags`, `buildListQuery`, `queryDescendantIDs` all documented. |
| Propagate errors with fmt.Errorf("%w", err) | PASS. `fmt.Errorf("querying tasks: %w", err)`, `fmt.Errorf("scanning task row: %w", err)`. | PASS. `fmt.Errorf("failed to query tasks: %w", err)`, `fmt.Errorf("failed to scan task row: %w", err)`, `fmt.Errorf("failed to check parent task: %w", err)`. |
| Do not ignore errors (_ assignment) | PASS. No ignored errors. | PASS. No ignored errors. |
| Do not use panic for normal error handling | PASS. No panics. | PASS. No panics. |
| Do not hardcode configuration | PASS for this task scope. | PASS for this task scope. |

### Spec-vs-Convention Conflicts

**1. Reuse of query functions (AC #11)**

The spec says: "Reuses ReadyQuery/BlockedQuery from 3-3/3-4."

V5 literally reuses the `ReadyQuery` and `BlockedQuery` string constants by wrapping them as subqueries. This is verbatim spec compliance but produces nested SQL that is less efficient and harder to debug.

V6 uses `ReadyConditions()` and `BlockedConditions()` helper functions instead. These were introduced in the V6 codebase's prior refactoring of query_helpers.go. They return the same logical conditions but as composable fragments rather than complete queries. The V6 codebase does not have `ReadyQuery`/`BlockedQuery` constants in the same form -- it has `ReadyConditions()`/`BlockedConditions()` functions which serve the same purpose.

Assessment: V6's approach is a reasonable judgment call. It reuses the underlying query logic (the conditions) while producing cleaner SQL. The spec's intent (don't re-implement the ready/blocked logic) is satisfied by both.

**2. Table-driven tests (skill constraint)**

The skill requires "table-driven tests with subtests." The spec lists 12 individual test names, not table-driven scenarios.

V5 uses table-driven tests for the `--status` test (4 entries) but individual subtests elsewhere. V6 uses individual subtests throughout.

Assessment: Neither version fully embraces table-driven tests. V5 is slightly more compliant. However, the test scenarios differ enough in setup (number of tasks, field combinations) that forcing them into tables would reduce readability without meaningful benefit. Both are reasonable judgment calls.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 2 (list.go, list_test.go) | 3 (app.go, list.go, list_filter_test.go) |
| Lines added | 562 | 547 |
| Impl LOC (diff) | 149 | 144 (137 list.go + 7 app.go) |
| Test LOC (diff) | 413 | 403 |
| Test functions | 15 (incl. 4 table-driven subtests) | 18 |

## Verdict

**V6 is the stronger implementation**, though the margin is moderate.

**V6's advantages:**

1. **Cleaner SQL generation.** V6 produces flat queries with inline WHERE conditions instead of nested subqueries. This is genuinely better -- simpler SQL, easier to debug, likely more efficient for the query planner.

2. **Better DRY.** V6 consolidates all query-building into a single `buildListQuery` function. V5 duplicates filter-appending logic across three separate builder functions (`buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildSimpleFilterQuery`).

3. **Stronger type safety.** V6 validates status values against `task.Status*` constants from the domain model rather than hardcoded strings. This prevents drift if status values change.

4. **Better separation of concerns.** V6 parses flags in `handleList` (app.go) and passes the parsed struct to `RunList`, cleanly separating CLI parsing from business logic. V5 parses inside `runList` itself.

5. **More thorough test assertions.** V6's non-numeric priority test checks the error message content (not just exit code), and the ordering test uses 4 tasks for more thorough coverage. V6's invalid priority test (value 5) is a better boundary test than V5's (value 9).

**V5's advantages:**

1. **Unknown flag detection.** V5's `default:` case in the flag parser catches unknown flags with `fmt.Errorf("unknown list flag %q", args[i])`. V6 silently ignores unknown flags, which could hide user mistakes.

2. **Table-driven test for status filtering.** V5 uses a proper table-driven subtest for the 4 status values, which is slightly more aligned with the Go skill constraint.

3. **More diagnostic test failures.** V5's assertions include the actual output in `t.Errorf("expected %s in output, got %q", ...)` while V6 uses terser `t.Error("description")` without showing the actual value.

The unknown-flag detection in V5 is a genuine quality advantage, but V6's structural improvements (SQL composition, DRY, type safety, separation of concerns) represent more impactful engineering decisions that affect maintainability and correctness. V6 wins overall.
