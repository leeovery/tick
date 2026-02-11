# Task tick-core-3-6: Parent scoping -- --parent flag with recursive descendant CTE

## Task Summary

Add a `--parent <id>` flag to `tick list` (and by extension `tick ready` and `tick blocked`, since they are aliases). The flag uses a recursive CTE in SQLite to collect all descendant IDs of the specified parent, then applies existing query filters within that narrowed set. The parent task itself is excluded from results. Non-existent parent IDs return an error. Parent with no descendants returns empty result (exit 0). `--parent` composes with `--status`, `--priority`, `--ready`, and `--blocked` as AND conditions. Case-insensitive parent ID matching. `--quiet` outputs IDs only within the scoped set.

### Acceptance Criteria

1. `tick list --parent <id>` returns only descendants (recursive, all levels)
2. `tick ready --parent <id>` returns only ready tasks within descendant set
3. `tick blocked --parent <id>` returns only blocked tasks within descendant set
4. Parent task itself excluded from results
5. Non-existent parent ID returns error: `Error: Task '<id>' not found`
6. Parent with no descendants returns empty result (`No tasks found.`, exit 0)
7. Deep nesting (3+ levels) collects all descendants recursively
8. `--parent` composes with `--status` filter (AND)
9. `--parent` composes with `--priority` filter (AND)
10. `--parent` composes with `--ready` flag (AND)
11. `--parent` composes with `--blocked` flag (AND)
12. Case-insensitive parent ID matching
13. `--quiet` outputs IDs only within the scoped set

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| 1. `tick list --parent <id>` returns only descendants | PASS - CTE collects descendants, `appendDescendantFilter` restricts via `AND id IN (...)` | PASS - Identical CTE, descendant IDs injected into `buildListQuery` via `t.id IN (...)` |
| 2. `tick ready --parent <id>` scoped ready | PASS - `runReady` prepends `--ready` to args, delegates to `runList`; tested | PASS - `handleReady` in `app.go` prepends `--ready`, delegates to `RunList`; tested |
| 3. `tick blocked --parent <id>` scoped blocked | PASS - `runBlocked` prepends `--blocked` to args, delegates to `runList`; tested | PASS - `handleBlocked` in `app.go` prepends `--blocked`, delegates to `RunList`; tested |
| 4. Parent excluded from results | PASS - CTE starts from `WHERE parent = ?`, not the parent itself; tested | PASS - Same CTE logic; tested |
| 5. Non-existent parent returns error | PASS - `parentTaskExists` check, `fmt.Errorf("Task '%s' not found")` with capital T; tested | PASS - Inline `COUNT(*)` check, `fmt.Errorf("task '%s' not found")` with lowercase t; tested |
| 6. No descendants returns empty (exit 0) | PASS - Early return when `len(descendantIDs) == 0`; tested | PASS - `1 = 0` impossible condition returns empty; tested |
| 7. Deep nesting (3+ levels) | PASS - Recursive CTE handles arbitrary depth; tested with 4 levels | PASS - Same CTE; tested with 4 levels |
| 8. Composes with `--status` | PASS - tested | PASS - tested |
| 9. Composes with `--priority` | PASS - tested | PASS - tested |
| 10. Composes with `--ready` | PASS - tested | PASS - tested |
| 11. Composes with `--blocked` | PASS - tested | PASS - tested |
| 12. Case-insensitive parent ID | PASS - `task.NormalizeID` in `parseListFlags`; tested | PASS - Same; tested |
| 13. `--quiet` within scoped set | PASS - tested | PASS - tested |

## Implementation Comparison

### Approach

Both versions take fundamentally the same approach: (1) parse `--parent` flag with `task.NormalizeID`, (2) validate parent exists in DB, (3) collect descendant IDs via recursive CTE, (4) inject `id IN (...)` filter into the list query. The key structural differences lie in how ready/blocked delegation works and how the query builder handles the empty-descendants case.

**Ready/Blocked delegation:**

V5 refactors `runReady` and `runBlocked` to be thin wrappers that prepend their respective flag to `ctx.Args` and call `runList`:

```go
// V5 ready.go, line 31-33
func runReady(ctx *Context) error {
    ctx.Args = append([]string{"--ready"}, ctx.Args...)
    return runList(ctx)
}
```

V6 takes the same approach but at the `App` dispatcher level in `app.go`. It modifies `handleReady` and `handleBlocked` to accept `subArgs`, prepend the flag, parse filters, and call `RunList`:

```go
// V6 app.go, lines 142-151
func (a *App) handleReady(fc FormatConfig, fmtr Formatter, subArgs []string) error {
    dir, err := a.Getwd()
    if err != nil {
        return fmt.Errorf("could not determine working directory: %w", err)
    }
    filter, err := parseListFlags(append([]string{"--ready"}, subArgs...))
    if err != nil {
        return err
    }
    return RunList(dir, fc, fmtr, filter, a.Stdout)
}
```

V6 eliminates `ready.go` and `blocked.go` entirely (both reduced to `package cli` only -- 1 line each), consolidating logic in `app.go`. V5 keeps the SQL query constants (`ReadyQuery`, `BlockedQuery`) in their respective files and reduces the function bodies to 2-line delegators. V5's approach retains colocation of SQL with related code; V6 moves SQL conditions to `query_helpers.go` (`ReadyConditions()`, `BlockedConditions()` returning `[]string` slices).

**Empty-descendants handling:**

V5 uses an early return pattern:

```go
// V5 list.go, lines 253-256
if len(descendantIDs) == 0 {
    return nil
}
```

V6 uses an impossible SQL condition:

```go
// V6 list.go, lines 228-231
} else if f.Parent != "" {
    // Parent exists but has no descendants: use impossible condition.
    conditions = append(conditions, `1 = 0`)
}
```

Both are correct. V5's early return avoids an unnecessary query. V6's `1 = 0` is more uniform (the query always runs) but does execute a pointless SQL statement. In practice the difference is negligible.

**Parent existence check:**

V5 extracts a separate `parentTaskExists` function:

```go
// V5 list.go, lines 207-214
func parentTaskExists(db *sql.DB, id string) (bool, error) {
    var count int
    err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = ?`, id).Scan(&count)
    if err != nil {
        return false, fmt.Errorf("checking parent task: %w", err)
    }
    return count > 0, nil
}
```

V6 inlines the same check directly in `RunList`:

```go
// V6 list.go, lines 109-115
var exists int
err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", filter.Parent).Scan(&exists)
if err != nil {
    return fmt.Errorf("failed to check parent task: %w", err)
}
if exists == 0 {
    return fmt.Errorf("task '%s' not found", filter.Parent)
}
```

V5's approach is slightly more reusable and testable as a standalone function. Both are functionally equivalent.

**Error message casing:**

V5 returns `"Task '%s' not found"` (capital T). V6 returns `"task '%s' not found"` (lowercase t). The spec says: `Error: Task '<id>' not found`. V5 matches the spec verbatim. V6 deviates -- though the error is wrapped with `Error:` prefix at the App level, the inner message starts lowercase (Go convention for error strings per `go vet`).

**Query builder architecture:**

V5 uses separate builder functions per query type (`buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildSimpleFilterQuery`) plus a shared `buildWrappedFilterQuery` to DRY the ready/blocked wrapping:

```go
// V5 list.go, lines 121-136
func buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string) (string, []interface{}) {
    q := `SELECT id, status, priority, title FROM (` + innerQuery + `) AS ` + alias + ` WHERE 1=1`
    var params []interface{}
    q, params = appendDescendantFilter(q, params, descendantIDs)
    // ...
}
```

V6 uses a single `buildListQuery` that assembles conditions from `ReadyConditions()` / `BlockedConditions()` slices:

```go
// V6 list.go, lines 199-239
func buildListQuery(f ListFilter, descendantIDs []string) (string, []interface{}) {
    var conditions []string
    var args []interface{}
    if f.Ready {
        conditions = append(conditions, ReadyConditions()...)
    }
    if f.Blocked {
        conditions = append(conditions, BlockedConditions()...)
    }
    // ...status, priority, descendant filters appended...
    query := `SELECT t.id, t.status, t.priority, t.title FROM tasks t`
    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }
    query += " ORDER BY t.priority ASC, t.created ASC"
    return query, args
}
```

V6's approach is more unified -- a single query builder with composable conditions. V5's approach wraps prebuilt SQL subqueries. Both work correctly. V6's is arguably cleaner for composition since all filters are peers in a flat conditions slice. V5's wrapping approach preserves the original SQL queries as constants which can be easier to debug via copy-paste to a SQL tool.

### Code Quality

**Go idioms and naming:**

V5 uses unexported types (`listFilters`, `listRow`) with unexported fields -- idiomatic for package-internal types. V6 uses an exported `ListFilter` struct with exported fields (`Ready`, `Blocked`, `Status`, `Priority`, `HasPriority`, `Parent`), which is appropriate since V6's `RunList` is a public function called from `app.go`.

**Error handling:**

Both versions handle all errors explicitly. Both use `fmt.Errorf` with `%w` wrapping consistently.

V5 error messages use a mix of styles: `"querying descendants: %w"`, `"scanning descendant ID: %w"`, `"Task '%s' not found"` (capital T).

V6 uses a consistent `"failed to ..."` prefix: `"failed to query descendants: %w"`, `"failed to scan descendant ID: %w"`, `"task '%s' not found"` (lowercase t, Go convention).

V6's error message style is more Go-idiomatic (errors should not be capitalized per Go convention and `go vet`).

**DRY principle:**

V5 introduces `buildWrappedFilterQuery` to share the outer-SELECT wrapping logic between ready and blocked filter queries, avoiding duplicated status/priority/descendant filter code (lines 121-136). However, it still has `buildReadyFilterQuery` and `buildBlockedFilterQuery` as thin wrappers.

V6 achieves DRY more aggressively by having a single `buildListQuery` that composes all conditions in a flat list. The ready/blocked SQL conditions come from `ReadyConditions()` and `BlockedConditions()` in `query_helpers.go`.

**Type safety:**

V6 uses `task.Status` type for status validation via a map keyed on `task.StatusOpen`, `task.StatusInProgress`, etc. (list.go lines 66-74). V5 uses a string slice `validStatuses` with raw strings `"open"`, `"in_progress"` etc. V6's approach is more type-safe.

### Test Quality

**V5 test functions (all in `TestParentScope`):**

1. `"it returns all descendants of parent (direct children)"` -- 2 children of parent, 1 unrelated
2. `"it returns all descendants recursively (3+ levels deep)"` -- 4-level hierarchy
3. `"it excludes parent task itself from results"` -- uses `--pretty` flag, checks header skipping
4. `"it returns empty result when parent has no descendants"` -- single parent, expects "No tasks found."
5. `"it errors with Task not found for non-existent parent ID"` -- exit code 1, error message check
6. `"it returns only ready tasks within parent scope with tick ready --parent"` -- ready vs blocked children, outside ready
7. `"it returns only blocked tasks within parent scope with tick blocked --parent"` -- blocked vs ready, outside blocked
8. `"it combines --parent with --status filter"` -- done vs open children, outside done
9. `"it combines --parent with --priority filter"` -- P1 vs P3 children, outside P1
10. `"it combines --parent with --ready and --priority"` -- ready P1 vs ready P3 vs blocked P1
11. `"it combines --parent with --blocked and --status"` -- blocked + status open
12. `"it handles case-insensitive parent ID"` -- `TICK-AAAAAA` input
13. `"it excludes tasks outside the parent subtree"` -- 2 parents, each with 1 child, plus root task
14. `"it outputs IDs only with --quiet within scoped set"` -- checks 2 ID-only lines, no outside task
15. `"it returns No tasks found when descendants exist but none match filters"` -- status=done with only open child

Total: **15 test cases**.

**V6 test functions (all in `TestParentScope`):**

1. `"it returns all descendants of parent (direct children)"` -- 2 children, 1 unrelated
2. `"it returns all descendants recursively (3+ levels deep)"` -- 4-level hierarchy
3. `"it excludes parent task itself from results"` -- checks all lines for parent ID prefix
4. `"it returns empty result when parent has no descendants"` -- expects "No tasks found.\n"
5. `"it errors with task not found for non-existent parent ID"` -- exit code 1, lowercase error check
6. `"it returns only ready tasks within parent scope with tick ready --parent"` -- ready child, blocked child, outside
7. `"it returns only blocked tasks within parent scope with tick blocked --parent"` -- blocked child, ready child, outside
8. `"it combines --parent with --status filter"` -- open vs done children
9. `"it combines --parent with --priority filter"` -- P1 vs P3 children
10. `"it combines --parent with --ready and --priority"` -- ready P1 vs ready P3 vs blocked P1
11. `"it combines --parent with --blocked and --status"` -- blocked + status open
12. `"it handles case-insensitive parent ID"` -- `TICK-PARENT` input
13. `"it excludes tasks outside the parent subtree"` -- 2 parents, children, orphan
14. `"it outputs IDs only with --quiet within scoped set"` -- exact string match `"tick-ch1111\ntick-ch2222\n"`
15. `"it returns No tasks found when descendants exist but none match filters"` -- status=cancelled (neither open nor done)

Total: **15 test cases**.

**Edge case coverage comparison:**

Both versions test the same 15 scenarios. The test names match 1:1 with the spec's test list. Both cover all specified edge cases.

**Assertion style differences:**

V5 uses `bytes.Buffer` directly, calling `Run()` (the package-level CLI entry point). Tests invoke the full CLI pipeline:
```go
// V5 parent_scope_test.go, line 24
code := Run([]string{"tick", "list", "--parent", "tick-aaaaaa"}, dir, &stdout, &stderr, false)
```

V6 uses purpose-built test helpers (`runList`, `runReady`, `runBlocked`) that construct an `App` struct directly. This is a cleaner separation:
```go
// V6 parent_scope_test.go, line 23
stdout, stderr, exitCode := runList(t, dir, "--parent", "tick-parent")
```

V6's quiet test (line 319) uses exact string comparison: `expected := "tick-ch1111\ntick-ch2222\n"`. V5's quiet test (line 398) uses line count + prefix checks, which is more resilient to output changes but less precise.

V6 tests use deterministic timestamps (`time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)` defined once at top), while V5 uses `time.Now().UTC().Truncate(time.Second)` in some tests and `task.NewTask()` (which presumably sets its own timestamps) in most. V6's approach is more deterministic.

V6 constructs tasks as struct literals with all fields explicit, while V5 uses `task.NewTask()` constructor and then mutates fields. V6's approach makes all test data visible at the declaration site.

**Test gap analysis:** Neither version has gaps relative to the other -- both implement all 15 specified tests. V6's "no match" test uses `--status cancelled` instead of V5's `--status done`, which is a slightly better choice since it proves the filter works against a status no descendant has (V5's test also works because the single child is open, not done).

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS - code is well-formatted | PASS - code is well-formatted |
| Handle all errors explicitly (no naked returns) | PASS - all errors checked and wrapped | PASS - all errors checked and wrapped |
| Write table-driven tests with subtests | PARTIAL - subtests via `t.Run` within `TestParentScope`, but not table-driven (each is a standalone subtest) | PARTIAL - same structure, subtests via `t.Run`, not table-driven |
| Document all exported functions, types, and packages | PASS (V5) - `parentTaskExists`, `queryDescendantIDs`, `appendDescendantFilter` etc all documented; exported types `ReadyQuery`, `BlockedQuery` documented | PASS (V6) - `RunList`, `ListFilter`, `queryDescendantIDs` etc all documented; `ReadyConditions()`, `BlockedConditions()` documented |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS - consistent wrapping throughout | PASS - consistent wrapping throughout |
| Ignore errors (avoid _ assignment without justification) | PASS - no ignored errors | PASS - no ignored errors |
| Use panic for normal error handling | PASS - no panics | PASS - no panics |
| Hardcode configuration | PASS - no hardcoded config | PASS - no hardcoded config |

**Table-driven tests note:** Both versions use subtests (`t.Run`) within a single `TestParentScope` function, which satisfies the "subtests" requirement. However, neither uses the table-driven pattern (defining a slice of test cases and iterating). Given that each test case has significantly different setup (different task hierarchies, different flags, different assertions), individual subtests are a reasonable judgment call -- table-driven would add complexity without clarity.

### Spec-vs-Convention Conflicts

**Error message casing:**

- The spec says: `Error: Task '<id>' not found` (capital T in "Task").
- Go convention (enforced by `go vet`): error strings should not be capitalized.
- V5 chose: `fmt.Errorf("Task '%s' not found", filters.parent)` -- matches spec verbatim.
- V6 chose: `fmt.Errorf("task '%s' not found", filter.Parent)` -- follows Go convention.
- Assessment: The `Error:` prefix is added by the App layer (`fmt.Fprintf(a.Stderr, "Error: %s\n", err)`), so the user-facing output becomes `Error: task 'xxx' not found` (V6) vs `Error: Task 'xxx' not found` (V5). The spec's exact wording uses capital T, but Go convention strongly discourages this. V6's deviation is a reasonable judgment call. V5's verbatim compliance is also defensible since the error is user-facing CLI output, not a programmatic error string. This is a wash.

**Table-driven tests:**

- The skill says: "Write table-driven tests with subtests."
- The task spec lists 15 individual tests, each with unique setup.
- Both versions chose individual subtests over table-driven.
- Assessment: Reasonable. The tests have heterogeneous setup (varying task hierarchies, flag combinations, assertion shapes). Forcing them into a table would require complex setup functions and reduce readability.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 4 (blocked.go, list.go, ready.go, parent_scope_test.go) | 5 (app.go, blocked.go, list.go, ready.go, parent_scope_test.go) |
| Lines added (impl) | ~120 new lines in list.go, ~6 new in blocked/ready (delegators) | ~75 new lines in list.go, ~16 new in app.go (handler changes) |
| Impl LOC (post-change) | 297 (list.go) + 37 (blocked.go) + 34 (ready.go) = 368 | 240 (list.go) + 1 (blocked.go) + 1 (ready.go) + 258 (app.go) = 500 |
| Test LOC | 437 | 345 |
| Test functions | 15 (in 1 top-level TestParentScope) | 15 (in 1 top-level TestParentScope) |

Note: V6's app.go LOC (258) includes much code unrelated to this task -- only ~16 lines were changed in app.go for this task. V6's lower test LOC (345 vs 437) comes from using struct literal task construction (more compact than `NewTask()` + field mutation) and test helpers that return strings instead of `bytes.Buffer`.

## Verdict

Both versions implement this task correctly and completely. All 13 acceptance criteria pass for both. All 15 specified tests are present in both. The core algorithm (recursive CTE + ID IN filter) is identical.

**V6 is marginally better** for the following reasons:

1. **Unified query builder:** V6's single `buildListQuery` with composable `ReadyConditions()`/`BlockedConditions()` slices is cleaner than V5's wrapping approach with `buildWrappedFilterQuery`. The flat conditions list makes it obvious how all filters interact.

2. **Go-idiomatic error strings:** V6's lowercase `"task '%s' not found"` follows Go convention. While V5's capital T matches the spec literally, the `Error:` prefix is added by the app layer regardless, and Go's `go vet` would flag V5's capitalized error.

3. **More type-safe status validation:** V6 validates against `task.StatusOpen` etc. (typed constants) rather than raw strings.

4. **Deterministic test timestamps:** V6 uses `time.Date(...)` for all tests, avoiding `time.Now()` non-determinism.

5. **Compact test data:** V6's struct literal task construction is more readable and compact than V5's `NewTask()` + mutation pattern.

6. **Test helpers:** V6's `runList`/`runReady`/`runBlocked` helpers with `t.Helper()` produce cleaner test failure output.

V5's advantages are minor: the extracted `parentTaskExists` function is slightly more reusable, and the early-return for zero descendants avoids an unnecessary SQL query. V5 also retains SQL query constants (`ReadyQuery`, `BlockedQuery`) colocated with their respective files, which aids SQL debugging. These are differences of style rather than correctness.

Overall, V6 shows slightly more mature Go patterns but both implementations are production-quality.
