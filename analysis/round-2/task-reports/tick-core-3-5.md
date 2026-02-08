# Task tick-core-3-5: tick list Filter Flags

## Task Summary

Wire `--ready`, `--blocked`, `--status`, and `--priority` filter flags into `tick list`. Four flags on `list`: `--ready` (bool), `--blocked` (bool), `--status <s>`, `--priority <p>`. `--ready`/`--blocked` are mutually exclusive (error if both). `--status` validates: open, in_progress, done, cancelled. `--priority` validates: 0-4. Filters combine as AND. Contradictory combos (e.g., `--status done --ready`) yield empty result, no error. No filters = all tasks (backward compatible). Reuses ReadyQuery/BlockedQuery from tasks 3-3/3-4. Output: aligned columns, `--quiet` IDs only.

**Acceptance Criteria:**
1. `list --ready` = same as `tick ready`
2. `list --blocked` = same as `tick blocked`
3. `--status` filters by exact match
4. `--priority` filters by exact match
5. Filters AND-combined
6. `--ready` + `--blocked` produces error
7. Invalid values produce error with valid options listed
8. No matches produces "No tasks found.", exit 0
9. `--quiet` outputs filtered IDs
10. Backward compatible (no filters = all)
11. Reuses query functions

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| `list --ready` = same as `tick ready` | PASS -- uses same `readyWhere` const fragment | PASS -- `buildListQuery` returns `readyQuery` directly when only `--ready` is set; falls back to `readyConditionsFor("t")` inline when combined with other flags |
| `list --blocked` = same as `tick blocked` | PASS -- uses same `blockedWhere` const fragment | PASS -- `buildListQuery` returns `blockedQuery` directly when only `--blocked`; builds inline blocked logic via `readyConditionsFor` when combined |
| `--status` filters by exact match | PASS -- parameterized `status = ?` | PASS -- parameterized `t.status = ?` |
| `--priority` filters by exact match | PASS -- parameterized `priority = ?` | PASS -- parameterized `t.priority = ?` |
| Filters AND-combined | PASS -- WHERE clauses joined with AND | PASS -- conditions appended with AND |
| `--ready` + `--blocked` produces error | PASS -- `parseListFlags` returns error | PASS -- `parseListFlags` returns error |
| Invalid values produce error with valid options | PASS -- status error lists valid values; priority error mentions 0-4 | PASS -- status error lists valid values; priority error lists 0,1,2,3,4 |
| No matches produces "No tasks found.", exit 0 | PASS -- tested in "it returns 'No tasks found.' when no matches" | PASS -- tested, though V4 output format uses `tasks[0]` (TOON format) rather than literal "No tasks found." string; behavior delegated to formatter |
| `--quiet` outputs filtered IDs | PASS -- tested in "it outputs IDs only with --quiet after filtering" | PASS -- tested in `TestList_QuietAfterFiltering` |
| Backward compatible (no filters = all) | PASS -- tested in "it returns all tasks with no filters" | PASS -- tested in `TestList_AllTasksNoFilters` |
| Reuses query functions | PASS -- reuses `readyWhere`/`blockedWhere` const fragments extracted from original `ReadySQL`/`BlockedSQL` | PASS -- reuses `readyConditionsFor()` function and full `readyQuery`/`blockedQuery` vars |

## Implementation Comparison

### Approach

Both versions follow the same high-level design: parse flags, validate, build dynamic SQL, execute query, render output. The key structural differences lie in how they reuse the ready/blocked SQL logic and how they organize the code across files.

**V2: Inline SQL fragment constants in `list.go`**

V2 refactors the previously-existing `ReadySQL` and `BlockedSQL` constants by extracting their WHERE clause bodies into new package-level constants `readyWhere` and `blockedWhere`, then reconstructing the full queries via string concatenation:

```go
const readyWhere = `status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )`

const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE ` + readyWhere + `
ORDER BY priority ASC, created ASC`
```

The `buildListQuery` function assembles WHERE clauses into a slice and joins them:

```go
func buildListQuery(flags listFlags) (string, []interface{}) {
    var where []string
    var queryArgs []interface{}
    if flags.ready {
        where = append(where, "("+readyWhere+")")
    } else if flags.blocked {
        where = append(where, "("+blockedWhere+")")
    }
    if flags.status != "" {
        where = append(where, "status = ?")
        queryArgs = append(queryArgs, flags.status)
    }
    if flags.hasPri {
        where = append(where, "priority = ?")
        queryArgs = append(queryArgs, flags.priority)
    }
    // ...joins with AND...
}
```

All changes are contained within `list.go` alone. No modifications to `ready.go` or `blocked.go`.

**V4: Parameterized function `readyConditionsFor` in `ready.go`**

V4 converts the ready conditions from a const string into a function that accepts a table alias parameter:

```go
func readyConditionsFor(alias string) string {
    return `
  NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = ` + alias + `.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = ` + alias + `.id
      AND child.status IN ('open', 'in_progress')
  )`
}
```

This is called from `ready.go` (`readyConditionsFor("t")`), `blocked.go` (`readyConditionsFor("t")`), and `list.go` (`readyConditionsFor("t")` for ready, `readyConditionsFor("t2")` for blocked combined queries). This touches 3 files (ready.go, blocked.go, list.go) but creates a more flexible foundation.

V4's `buildListQuery` includes a fast-path optimization for the simple cases:

```go
func buildListQuery(f listFlags) (string, []interface{}) {
    if f.ready && !f.hasPri && f.status == "" {
        return readyQuery, nil
    }
    if f.blocked && !f.hasPri && f.status == "" {
        return blockedQuery, nil
    }
    // Build dynamic query...
    query := "SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE 1=1"
    // ...
}
```

This means `list --ready` (with no other flags) returns exactly the same query as `tick ready`, character-for-character. V2 always builds a dynamic query with `(readyWhere)` wrapped in parens, which is semantically equivalent but not the exact same SQL string.

**V4's blocked combined query is notably more complex:**

```go
if f.blocked {
    query += " AND t.status = 'open' AND t.id NOT IN (SELECT t2.id FROM tasks t2 WHERE t2.status = 'open' AND" +
        readyConditionsFor("t2") + ")"
}
```

This uses a different table alias (`t2`) to avoid SQL ambiguity in the subquery, which is a genuine correctness concern. V2's `blockedWhere` uses unqualified column names and a JOIN on `tasks t` inside a subquery that could theoretically conflict with the outer query's `tasks` table, though in SQLite this is resolved by scoping.

**SQL correctness difference:** V2's `readyWhere`/`blockedWhere` fragments use unqualified column references (`id`, `status`, `priority`) with `JOIN tasks t` inside subqueries. When embedded in `buildListQuery`, the outer SELECT also references `tasks` without an alias. V4 consistently uses the `t` alias on all outer references and `readyConditionsFor("t2")` to avoid ambiguity in nested subqueries. This is a genuine correctness advantage for V4.

**V4 also uses `NOT EXISTS` instead of `NOT IN`:**

V2's ready fragment:
```go
id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
)
```

V4's ready fragment:
```go
NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
)
```

`NOT EXISTS` is generally preferred over `NOT IN` due to NULL-safety and potential query optimizer advantages, though for this specific case (non-nullable IDs) both are equivalent.

### Code Quality

**Flag parsing:**

V2 silently ignores unknown flags:
```go
switch args[i] {
case "--ready":
    flags.ready = true
case "--blocked":
    flags.blocked = true
case "--status":
    // ...
case "--priority":
    // ...
}
```

V4 returns an error for unknown flags:
```go
default:
    return f, fmt.Errorf("unknown flag %q for list command", args[i])
```

V4's approach is more robust -- it catches typos like `--statue` or `--priortiy` rather than silently ignoring them. This is genuinely better.

**Status validation:**

V2 uses a `map[string]bool`:
```go
var validStatuses = map[string]bool{
    "open": true, "in_progress": true, "done": true, "cancelled": true,
}
```

V4 uses a `[]string` with a linear scan:
```go
var validStatuses = []string{"open", "in_progress", "done", "cancelled"}
valid := false
for _, s := range validStatuses {
    if f.status == s { valid = true; break }
}
```

V2's map lookup is O(1), V4's slice iteration is O(n) but n=4 so irrelevant. V2 is more idiomatic for set-membership checks.

**Priority validation timing:**

V2 separates parsing from validation -- it parses the number first, stores it, then validates range in a separate step after the loop:
```go
p, err := strconv.Atoi(args[i])
if err != nil {
    return flags, fmt.Errorf("--priority must be a number (0-4)")
}
flags.priority = p
flags.hasPri = true
// ...later...
if flags.hasPri && (flags.priority < 0 || flags.priority > 4) {
    return flags, fmt.Errorf("invalid priority %d — valid range: 0-4", flags.priority)
}
```

V4 validates immediately during parsing:
```go
p, err := strconv.Atoi(args[i])
if err != nil || p < 0 || p > 4 {
    return f, fmt.Errorf("invalid priority %q; valid values: 0, 1, 2, 3, 4", args[i])
}
```

V4's approach is more concise and produces a single error message regardless of failure reason. V2 gives different error messages for "not a number" vs "out of range," which is more informative.

**Priority sentinel value:**

V2 uses `hasPri bool` with default `priority = 0`.
V4 uses `hasPri bool` AND sets `f.priority = -1` as sentinel:
```go
f.priority = -1 // sentinel: not set
```

V4's sentinel is redundant given the `hasPri` bool already distinguishes set/unset. V2 is cleaner here.

**Error message style:**

V2: `"invalid status '%s' — valid values: open, in_progress, done, cancelled"`
V4: `"invalid status %q; valid values: open, in_progress, done, cancelled"`

V4 uses `%q` (quoted) which is slightly more Go-idiomatic for user-facing values. Both are fine.

**`listRow` struct location:**

V2 defines `listRow` as a local type inside `runList`:
```go
func (a *App) runList(args []string) error {
    // ...
    type listRow struct {
        ID       string
        Status   string
        Priority int
        Title    string
    }
```

V4 defines `listRow` as a package-level type in `list.go` and reuses it in `ready.go` and `blocked.go`:
```go
type listRow struct {
    ID       string
    Status   string
    Priority int
    Title    string
}
```

V4's approach is better -- it eliminates duplication since `runReady` and `runBlocked` also need this struct.

**Rendering delegation:**

V2 has inline quiet-mode logic in `runList`:
```go
if a.config.Quiet {
    if len(rows) == 0 { return nil }
    for _, r := range rows { fmt.Fprintln(a.stdout, r.ID) }
    return nil
}
taskRows := make([]TaskRow, len(rows))
for i, r := range rows {
    taskRows[i] = TaskRow{ID: r.ID, Status: r.Status, Priority: r.Priority, Title: r.Title}
}
return a.formatter.FormatTaskList(a.stdout, taskRows)
```

V4 delegates everything to the formatter:
```go
return a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)
```

V4 is cleaner -- the formatter handles both quiet and normal modes. V2 duplicates the quiet logic and requires a `listRow` -> `TaskRow` conversion that V4 avoids entirely.

**Files changed:**

V2 modified only `list.go` and `list_test.go`. It left `ReadySQL` and `BlockedSQL` as exported constants and added new unexported `readyWhere`/`blockedWhere` fragments alongside them.

V4 modified `list.go`, `list_test.go`, `ready.go`, and `blocked.go`. It converted the `readyConditions` const to a `readyConditionsFor(alias)` function and updated all callers.

### Test Quality

**V2 Test Functions (in `list_test.go`):**

Top-level functions: `TestListCommand` (6 subtests), `TestListFilterFlags` (15 subtests)

Pre-existing tests in `TestListCommand`:
1. `"it lists all tasks with aligned columns"` -- verifies header and data alignment
2. `"it lists tasks ordered by priority then created date"` -- verifies sort order
3. `"it prints 'No tasks found.' when no tasks exist"` -- empty state
4. `"it prints only task IDs with --quiet flag on list"` -- quiet mode
5. `"it returns all tasks with no filters"` -- **NEW** -- backward compatibility with 4 different statuses
6. `"it executes through storage engine read flow"` -- verifies cache.db creation

New tests in `TestListFilterFlags`:
7. `"it filters to ready tasks with --ready"` -- 7 assertions (3 included, 4 excluded)
8. `"it filters to blocked tasks with --blocked"` -- 3 assertions
9. `"it filters by --status open"` -- 5 assertions
10. `"it filters by --status in_progress"` -- 2 assertions
11. `"it filters by --status done"` -- 2 assertions
12. `"it filters by --status cancelled"` -- 2 assertions
13. `"it filters by --priority"` -- 4 assertions
14. `"it combines --ready with --priority"` -- 3 assertions
15. `"it combines --status with --priority"` -- 4 assertions
16. `"it errors when --ready and --blocked both set"` -- checks error contains both flag names
17. `"it errors for invalid status value"` -- checks error message mentions invalid and valid values
18. `"it errors for invalid priority value"` -- checks 0-4 range in error
19. `"it errors for non-numeric priority value"` -- just checks error is non-nil
20. `"it returns 'No tasks found.' when no matches"` -- exact string match
21. `"it returns empty for contradictory filters without error"` -- `--status done --ready`
22. `"it outputs IDs only with --quiet after filtering"` -- exact line-by-line match
23. `"it maintains deterministic ordering"` -- runs twice, compares output

V2 uses a shared `mixedContent()` helper function that creates a consistent dataset (3 ready, 1 blocked, 1 in-progress, 1 done, 1 cancelled) reused across most filter tests. Tests use individual subtests within `TestListFilterFlags`. Status filter tests are 4 separate subtests (one per status value). Error validation tests are individual subtests.

**V4 Test Functions (in `list_test.go`):**

Top-level functions: 17 separate `func Test*` functions

Pre-existing tests:
1. `TestList_AllTasksWithAlignedColumns` -- verifies TOON format header and task data
2. `TestList_OrderByPriorityThenCreated` -- sort order with 4 tasks
3. `TestList_NoTasksFound` -- empty state, checks for `tasks[0]`
4. `TestList_QuietFlag` -- quiet mode IDs only

New tests:
5. `TestList_StatusFilter` -- **table-driven** with 4 subtests (one per status), single shared task list
6. `TestList_PriorityFilter` -- 3 tasks with different priorities
7. `TestList_CombineReadyWithPriority` -- 4 tasks with blocker/blocked/ready mix
8. `TestList_CombineStatusWithPriority` -- 3 tasks including done
9. `TestList_ErrorReadyAndBlocked` -- checks exit code 1 and "mutually exclusive" in stderr
10. `TestList_ErrorInvalidStatusPriority` -- **table-driven** with 4 subtests (invalid status, negative priority, too-high priority, non-numeric)
11. `TestList_NoMatchesReturnsNoTasksFound` -- checks `tasks[0]` format
12. `TestList_QuietAfterFiltering` -- filtered quiet output
13. `TestList_AllTasksNoFilters` -- 3 tasks with different statuses
14. `TestList_DeterministicOrdering` -- runs twice, verifies exact row order
15. `TestList_BlockedFlag` -- 3 tasks with one blocked
16. `TestList_CombineBlockedWithPriority` -- 4 tasks, combines --blocked --priority
17. `TestList_ReadyFlag` -- 3 tasks with one blocked

**Test Structure Comparison:**

V2 groups all filter tests under one `TestListFilterFlags` parent with subtests. V4 uses separate top-level test functions per concern (e.g., `TestList_StatusFilter`, `TestList_PriorityFilter`). V4's approach gives better isolation and clearer test names in output but is more verbose.

V4 uses **table-driven tests** for status filter (4 subtests in one table) and error validation (4 cases in one table). V2 uses individual subtests for each status value and each error case.

**Test Data Setup:**

V2 uses `taskJSONL()` helper to create raw JSONL content and `setupTickDirWithContent()`. V4 uses `task.Task` structs and `setupInitializedDirWithTasks()`, which is type-safe and uses the domain model.

**Edge Cases Unique to Each Version:**

- V2 tests `"it returns empty for contradictory filters without error"` (`--status done --ready`) -- **V4 does NOT test this**
- V2 tests `"it errors for non-numeric priority value"` as a separate subtest -- V4 includes this as one row in the table-driven test
- V4 tests `TestList_CombineBlockedWithPriority` (4 tasks, two blocked at different priorities) -- **V2 does NOT test this combination**
- V4's `TestList_DeterministicOrdering` verifies **exact row positions** (line 1 = P1, line 2 = P2 older, line 3 = P2 newer); V2 only compares that two runs produce identical output without checking positions

**Test Gap Analysis:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Contradictory filters (--status done --ready) | Tested | NOT tested |
| --blocked combined with --priority | NOT tested | Tested |
| Non-numeric priority error | Tested (separate subtest) | Tested (table row) |
| Unknown flag error | NOT tested (silently ignored) | NOT tested (but code handles it) |
| --status without value | NOT tested | NOT tested |
| --priority without value | NOT tested | NOT tested |
| Storage engine flow | Tested (cache.db check) | NOT tested in this file |

**Assertion Style:**

V2 checks `err == nil` / `err != nil` and uses `err.Error()` for message checking.
V4 checks `code != 0` / `code != 1` (exit codes) and uses `stderr.String()` for error messages.

V4's approach aligns better with how a CLI actually reports errors (exit code + stderr) vs V2 treating them as Go errors. V4's pattern is more realistic for integration-style tests.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 2 (.go) | 4 (.go) |
| Lines added | 604 | 561 |
| Lines removed | 28 | 13 |
| Impl LOC (list.go) | 226 | 161 |
| Impl LOC (total, incl. ready.go/blocked.go changes) | 226 | 161 + 79 + 62 = 302 (but only ~121 lines new in list.go; ready.go/blocked.go already existed) |
| Test LOC | 674 | 618 |
| Top-level test functions | 2 | 17 |
| Total test subtests (new for this task) | 16 | 17 |

## Verdict

**V4 is the better implementation**, by a moderate margin. The key differentiators:

1. **SQL correctness**: V4's `readyConditionsFor(alias)` function properly parameterizes table aliases, avoiding potential ambiguity in nested subqueries. V2's unqualified column references in `readyWhere`/`blockedWhere` fragments work in SQLite's scoping model but are fragile if queries become more complex.

2. **Unknown flag handling**: V4 errors on unknown flags (`default: return f, fmt.Errorf("unknown flag %q...")`), while V2 silently ignores them. This is a meaningful UX difference.

3. **Fast-path optimization**: V4's `buildListQuery` returns the exact `readyQuery`/`blockedQuery` strings when no additional filters are set, ensuring `list --ready` produces byte-identical SQL to `tick ready`. V2 always wraps fragments in parens and builds dynamically.

4. **Code organization**: V4's `readyConditionsFor` function in `ready.go` centralizes the ready-condition logic in one place and makes it reusable with different aliases. V2's `readyWhere` const in `list.go` duplicates the concept (now the ready logic lives in both `list.go` and wherever the original `ReadySQL` lived).

5. **Shared types**: V4's package-level `listRow` struct is shared across `list.go`, `ready.go`, and `blocked.go`, eliminating duplication. V2 keeps it as a local type in `runList`.

6. **Test quality**: V4 uses table-driven tests for status filters and error validation, which is more idiomatic Go. V4 also tests `--blocked --priority` combination which V2 misses. However, V2 tests the contradictory filter edge case (`--status done --ready`) which V4 omits -- a notable gap in V4.

7. **Formatter delegation**: V4 delegates all output rendering (including quiet mode) to `a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)`, while V2 has inline quiet-mode handling in `runList`. V4 is cleaner and more maintainable.

V2's advantages are minor: map-based status validation is slightly more idiomatic, separate error messages for non-numeric vs out-of-range priority are more informative, and the contradictory filter test is a coverage edge V4 lacks. These do not outweigh V4's structural and correctness advantages.
