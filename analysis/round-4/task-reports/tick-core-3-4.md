# Task tick-core-3-4: Blocked query, tick blocked & cancel-unblocks-dependents

## Task Summary

Implement the inverse of the ready query: a `BlockedQuery` that returns open tasks failing ready conditions (have unclosed blockers OR have open children). Register a `tick blocked` subcommand as an alias for `list --blocked`. Verify that cancelling a blocker unblocks its dependents end-to-end.

### Acceptance Criteria (from plan)

1. Returns open tasks with unclosed blockers or open children
2. Excludes tasks where all blockers closed
3. Only `open` in output
4. Cancel -> dependent unblocks
5. Multiple dependents unblock simultaneously
6. Partial unblock works correctly
7. Deterministic ordering (priority ASC, created ASC)
8. `tick blocked` outputs aligned columns
9. Empty -> `No tasks found.`, exit 0
10. `--quiet` IDs only
11. Reuses ready query logic

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Returns open tasks with unclosed blockers or open children | PASS -- `BlockedQuery` WHERE clause uses EXISTS for unclosed blockers and open children; tested in "it returns open task blocked by open/in_progress dep" and "it returns parent with open/in_progress children" | PASS -- `blockedSQL` has identical WHERE clause; tested in separate subtests for open dep, in_progress dep, open children, in_progress children |
| Excludes tasks where all blockers closed | PASS -- tested in "it excludes task when all blockers done/cancelled" | PASS -- tested in "it excludes task when all blockers done or cancelled" |
| Only `open` in output | PASS -- SQL has `t.status = 'open'`; tested in "it excludes in_progress/done/cancelled from output" | PASS -- same SQL filter; tested in 3 separate subtests (in_progress, done, cancelled) |
| Cancel -> dependent unblocks | PASS -- "cancel unblocks single dependent - moves to ready" runs blocked, cancel, ready, blocked in sequence | PASS -- "cancel unblocks single dependent moves to ready" same workflow |
| Multiple dependents unblock simultaneously | PASS -- "cancel unblocks multiple dependents" | PASS -- "cancel unblocks multiple dependents" |
| Partial unblock works correctly | PASS -- "cancel does not unblock dependent still blocked by another" | PASS -- "cancel does not unblock dependent still blocked by another" plus additional static "partial unblock: two blockers one cancelled still blocked" test |
| Deterministic ordering | PASS -- `ORDER BY t.priority ASC, t.created ASC`; tested in "it orders by priority ASC then created ASC" | PASS -- identical ORDER BY; tested with 4 blocked tasks (2 priorities x 2 creation times) |
| `tick blocked` outputs aligned columns | PASS -- tested in "it outputs aligned columns via tick blocked" checking header/row content | PASS -- tested with exact string matching of header and rows |
| Empty -> `No tasks found.`, exit 0 | PASS -- tested in "it returns empty when no blocked tasks" and "it prints 'No tasks found.' when empty" | PASS -- tested in "it returns empty when no blocked tasks", "it returns empty when no tasks exist", and "it prints 'No tasks found.' when empty" |
| `--quiet` IDs only | PASS -- tested in "it outputs IDs only with --quiet" | PASS -- tested in "it outputs IDs only with --quiet" with exact expected output |
| Reuses ready query logic | PARTIAL -- `BlockedQuery` is a standalone SQL constant that is the logical inverse of `ReadyQuery`, but does not structurally reuse `ReadyQuery`'s WHERE clause. However, `runBlocked` delegates to `runList` via `ctx.Args = append([]string{"--blocked"}, ctx.Args...)`, reusing list infrastructure. | PARTIAL -- `blockedSQL` is also a standalone SQL constant. `handleBlocked` calls `RunBlocked` directly, which does NOT delegate to `RunList`. No structural SQL reuse from ready. |

## Implementation Comparison

### Approach

**V5** takes a thin-wrapper approach. The `blocked.go` file defines two things:

1. A `BlockedQuery` SQL constant (exported, used by list.go's `buildBlockedFilterQuery`)
2. A `runBlocked` function that delegates entirely to `runList`:

```go
// V5 blocked.go (at commit c66ead3)
func runBlocked(ctx *Context) error {
    tickDir, err := DiscoverTickDir(ctx.WorkDir)
    if err != nil {
        return err
    }
    store, err := engine.NewStore(tickDir)
    if err != nil {
        return err
    }
    defer store.Close()
    var rows []listRow
    err = store.Query(func(db *sql.DB) error {
        sqlRows, err := db.Query(BlockedQuery)
        ...
    })
    ...
    printListTable(ctx.Stdout, rows)
    return nil
}
```

V5's `runBlocked` is a self-contained function that opens the store, runs `BlockedQuery` directly, handles empty/quiet cases, and delegates table formatting to the shared `printListTable` helper from `list.go`. Registration is via the command map in `cli.go`:

```go
// V5 cli.go
var commands = map[string]func(*Context) error{
    ...
    "blocked": runBlocked,
}
```

**V6** takes a nearly identical standalone approach. The `blocked.go` file defines:

1. A `blockedSQL` constant (unexported, lowercase)
2. A `RunBlocked` exported function:

```go
// V6 blocked.go (at commit 4a3317b)
func RunBlocked(dir string, quiet bool, stdout io.Writer) error {
    tickDir, err := DiscoverTickDir(dir)
    if err != nil {
        return err
    }
    store, err := storage.NewStore(tickDir)
    ...
    err = store.Query(func(db *sql.DB) error {
        sqlRows, err := db.Query(blockedSQL)
        ...
    })
    ...
    // Print header (same format as list/ready)
    fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")
    for _, r := range rows {
        fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
    }
    return nil
}
```

V6 registration is in `app.go`:

```go
// V6 app.go
case "blocked":
    err = a.handleBlocked(flags)

func (a *App) handleBlocked(flags globalFlags) error {
    dir, err := a.Getwd()
    if err != nil {
        return fmt.Errorf("could not determine working directory: %w", err)
    }
    return RunBlocked(dir, flags.quiet, a.Stdout)
}
```

**Key structural differences:**

1. **SQL constant naming**: V5 uses exported `BlockedQuery`; V6 uses unexported `blockedSQL`. V6's choice is more appropriate since the constant is only used within the package.

2. **Formatting reuse**: V5 reuses `printListTable` from `list.go`. V6 duplicates the `fmt.Fprintf` formatting inline, creating a maintenance liability since the same column widths and format strings appear in `ready.go`, `blocked.go`, and potentially `list.go`.

3. **Architecture pattern**: V5 follows the same `Context`-based handler pattern used throughout the codebase. V6 uses an `App` struct with method receivers and function-based dispatch, passing primitive parameters (`dir string, quiet bool, stdout io.Writer`) instead of a context struct.

4. **listRow scope**: V5 uses a package-level `listRow` type. V6 defines `listRow` as a local type inside `RunBlocked`, duplicating the definition from `RunReady`.

5. **Store import**: V5 imports `engine.NewStore`; V6 imports `storage.NewStore` -- reflecting different package naming conventions between versions.

### Code Quality

**Error handling**: Both versions handle errors explicitly with `%w` wrapping. V5 uses terser error prefixes (`"querying blocked tasks: %w"`), while V6 uses `"failed to query blocked tasks: %w"`. Both are idiomatic.

**DRY principle**: V5 is significantly DRYer. It reuses `printListTable` for formatting and `listRow` as a shared type. V6 duplicates the table formatting logic (the `fmt.Fprintf` header and row printing) and the `listRow` type definition across `RunReady` and `RunBlocked`.

V5 formatting delegation:
```go
// V5 -- reuses shared helper
printListTable(ctx.Stdout, rows)
```

V6 duplicated formatting:
```go
// V6 blocked.go -- inline copy of formatting from ready.go
fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")
for _, r := range rows {
    fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
}
```

**Function signatures**: V5 uses `func runBlocked(ctx *Context) error` -- consistent with all other command handlers. V6 uses `func RunBlocked(dir string, quiet bool, stdout io.Writer) error` -- exported and parameter-based rather than context-based.

**Documentation**: Both versions document all exported symbols. V5's `BlockedQuery` doc comment explains the inverse relationship to `ReadyQuery`. V6's `blockedSQL` doc comment likewise explains the inverse of `readySQL`. Both are adequate.

**SQL query**: The SQL is functionally identical between versions -- same WHERE clause, same EXISTS subqueries, same ORDER BY.

### Test Quality

**V5 Test Functions:**

1. `TestBlockedQuery` (6 subtests):
   - "it returns open task blocked by open/in_progress dep" -- tests both open and in_progress blockers in one test, verifies blockers themselves do not appear
   - "it returns parent with open/in_progress children" -- tests both open and in_progress children, verifies children do not appear
   - "it excludes task when all blockers done/cancelled" -- done + cancelled blockers
   - "it excludes in_progress/done/cancelled from output" -- in_progress task with blocker, plus done/cancelled tasks
   - "it returns empty when no blocked tasks" -- ready task only
   - "it orders by priority ASC then created ASC" -- 3 tasks with different priorities and creation times

2. `TestBlockedCommand` (3 subtests):
   - "it outputs aligned columns via tick blocked" -- checks header columns and row content
   - "it prints 'No tasks found.' when empty" -- empty project
   - "it outputs IDs only with --quiet" -- verifies IDs only, no spaces

3. `TestCancelUnblocksDependents` (3 subtests):
   - "cancel unblocks single dependent - moves to ready" -- 4-step: check blocked, cancel, check ready, check not blocked
   - "cancel unblocks multiple dependents" -- cancel shared blocker, check both ready
   - "cancel does not unblock dependent still blocked by another" -- partial cancel, check still blocked, check not ready

**V5 total: 3 test functions, 12 subtests**

**V6 Test Functions:**

1. `TestBlocked` (15 subtests):
   - "it returns open task blocked by open dep" -- separate from in_progress
   - "it returns open task blocked by in_progress dep" -- separate test
   - "it returns parent with open children" -- separate from in_progress
   - "it returns parent with in_progress children" -- separate test
   - "it excludes task when all blockers done or cancelled"
   - "it excludes in_progress tasks from output" -- standalone
   - "it excludes done tasks from output" -- standalone
   - "it excludes cancelled tasks from output" -- standalone
   - "it returns empty when no blocked tasks" -- with ready task
   - "it returns empty when no tasks exist" -- empty project (extra test V5 lacks)
   - "it orders by priority ASC then created ASC" -- 4 tasks with 2 priorities
   - "it outputs aligned columns via tick blocked" -- exact string matching
   - "it prints 'No tasks found.' when empty"
   - "it outputs IDs only with --quiet" -- exact expected output
   - "partial unblock: two blockers one cancelled still blocked" -- static partial unblock (extra test V5 lacks)

2. `TestCancelUnblocksDependents` (3 subtests):
   - "cancel unblocks single dependent moves to ready"
   - "cancel unblocks multiple dependents"
   - "cancel does not unblock dependent still blocked by another"

**V6 total: 2 test functions, 18 subtests**

**Test coverage diff:**

V6 has more granular subtests that split combined V5 tests into individual cases. Notable V6 extras:
- "it returns empty when no tasks exist" -- tests completely empty project (V5 only tests with a ready task present)
- "partial unblock: two blockers one cancelled still blocked" -- tests partial unblock as a static scenario without cancel action (V5 only tests partial unblock through the cancel workflow)
- Separate tests for in_progress, done, cancelled exclusion (V5 combines these into one test)
- Separate tests for open vs in_progress blocker deps (V5 combines both in one test)

V5 extras:
- V5's "cancel unblocks single dependent" test is more thorough: it verifies the task is blocked BEFORE cancel, then checks BOTH ready AND not-blocked after cancel (4 steps). V6 does the same.
- No test gaps unique to V5.

**Assertion style:**

V5 uses `strings.Contains` for most checks, which is loose. For the quiet test, V5 checks `strings.HasPrefix(line, "tick-")` and `!strings.Contains(line, " ")` instead of exact matching.

V6 uses exact string matching for key outputs:
```go
// V6 exact matching
expected := "tick-aaa111\ntick-bbb222\n"
if stdout != expected {
    t.Errorf("stdout = %q, want %q", stdout, expected)
}
```

V6 also uses exact header/row matching for alignment tests:
```go
// V6
if header != "ID          STATUS       PRI  TITLE" {
    t.Errorf("header = %q, want %q", header, ...)
}
```

V5 uses substring checks for header:
```go
// V5 -- looser assertion
if !strings.HasPrefix(lines[0], "ID") { ... }
if !strings.Contains(lines[0], "STATUS") { ... }
```

**Test helpers:**

V6 defines a `runBlocked` test helper that encapsulates App creation:
```go
func runBlocked(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
    t.Helper()
    var stdoutBuf, stderrBuf bytes.Buffer
    app := &App{...}
    ...
}
```

V5 calls the global `Run` function directly in each test, leading to more boilerplate per test.

**Task construction:**

V5 uses `task.NewTask()` with subsequent field mutations. V6 uses struct literals:
```go
// V5
blocker := task.NewTask("tick-aaaaaa", "Open blocker")
t1 := task.NewTask("tick-bbbbbb", "Blocked by open")
t1.BlockedBy = []string{"tick-aaaaaa"}

// V6
tasks := []task.Task{
    {ID: "tick-blk111", Title: "Blocker open", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
    {ID: "tick-dep111", Title: "Depends on open", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-blk111"}, ...},
}
```

V6's struct literals are more explicit about all field values, reducing reliance on `NewTask` defaults.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code follows standard formatting | PASS -- code follows standard formatting |
| Handle all errors explicitly (no naked returns) | PASS -- all errors checked, wrapped with %w | PASS -- all errors checked, wrapped with %w |
| Write table-driven tests with subtests | PARTIAL -- uses subtests (`t.Run`) but not table-driven format; each subtest sets up its own scenario | PARTIAL -- uses subtests but not table-driven format; each subtest is standalone |
| Document all exported functions, types, and packages | PASS -- `BlockedQuery` and `runBlocked` documented | PASS -- `blockedSQL` (unexported, still documented), `RunBlocked` documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- e.g., `fmt.Errorf("querying blocked tasks: %w", err)` | PASS -- e.g., `fmt.Errorf("failed to query blocked tasks: %w", err)` |
| Do not ignore errors | PASS -- no `_` assignments | PASS -- no `_` assignments |
| Do not use panic for normal error handling | PASS | PASS |
| Do not hardcode configuration | PASS -- no hardcoded config; SQL is a constant but that's appropriate for query definition | PASS -- same |

**Note on table-driven tests:** The skill requires "table-driven tests with subtests." Neither version uses the classic Go table-driven pattern (`tests := []struct{...}{...}; for _, tt := range tests { t.Run(...) }`). Both use individual `t.Run` subtests. Given that each test scenario requires unique setup (different task graphs), table-driven tests would be awkward and less readable here. Both versions make a reasonable judgment call to use individual subtests instead.

### Spec-vs-Convention Conflicts

**1. "Reuses ready query logic" vs. self-contained SQL**

- **Spec says:** "Simplest: blocked = open minus ready (reuse ReadyQuery)" and acceptance criterion "Reuses ready query logic."
- **Convention:** The blocked query is the De Morgan inverse of the ready query. Structurally reusing ready logic would require either (a) computing the set difference at runtime (query all open, subtract ready), or (b) negating the ready WHERE clause fragments. Both add complexity for dubious benefit, since the SQL is more readable as a standalone statement.
- **V5:** Defines a standalone `BlockedQuery` constant. The `runBlocked` function does its own store/query/format work but reuses `printListTable`. Does NOT structurally derive from `ReadyQuery`.
- **V6:** Defines a standalone `blockedSQL` constant. `RunBlocked` is fully self-contained. Does NOT derive from `readySQL`.
- **Assessment:** Both versions make the same reasonable judgment call. The SQL inverse is straightforward and the duplication is acceptable. Neither version literally "reuses ready query logic" at the SQL level. V5 gets slightly closer by reusing `printListTable` for output formatting.

**2. "tick blocked = alias for list --blocked" vs. standalone command**

- **Spec says:** "`tick blocked` = alias for `list --blocked`"
- **V5:** `runBlocked` is a standalone function that does NOT delegate to `runList`. It opens its own store and runs `BlockedQuery` directly. However, the `list.go` file does have `buildBlockedFilterQuery` that wraps `BlockedQuery` for `list --blocked`, so `list --blocked` and `tick blocked` use the same SQL constant.
- **V6:** `RunBlocked` is also standalone, does NOT delegate to `RunList`. The `handleBlocked` method in `app.go` calls `RunBlocked` directly.
- **Assessment:** Both deviate from the spec's "alias" language. In both versions, `tick blocked` and `tick list --blocked` would produce the same results because they use the same SQL, but the code paths are different. This is a minor deviation that avoids coupling the blocked command to list flag parsing.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 3 (blocked.go, blocked_test.go, cli.go) | 3 (app.go, blocked.go, blocked_test.go) |
| Lines added | 518 (+518, -13 realignment) | 575 (+575, -2) |
| Impl LOC | 99 (blocked.go) + 11 (cli.go change) = 110 | 114 (blocked.go) + 11 (app.go) = 125 |
| Test LOC | 431 | 461 |
| Test functions | 3 (12 subtests) | 2 (18 subtests) |

## Verdict

**V5 is the better implementation**, though the margin is moderate.

**V5 advantages:**
1. **DRYer code** -- reuses `printListTable` from `list.go` for output formatting instead of duplicating `fmt.Fprintf` format strings. This is the most significant difference, as V6's inline formatting creates a maintenance liability (format changes must be synchronized across `ready.go`, `blocked.go`, and `list.go`).
2. **Shared type** -- uses package-level `listRow` instead of redefining it locally, reducing duplication.
3. **Leaner implementation** -- 99 lines vs 114 lines for the same functionality, with less boilerplate.

**V6 advantages:**
1. **More granular tests** -- 18 subtests vs 12, splitting combined scenarios into individual cases (e.g., separate tests for in_progress, done, cancelled exclusion). This improves test failure diagnostics.
2. **Exact string assertions** -- uses exact matching for quiet output and column alignment, catching regressions that V5's substring checks would miss.
3. **Extra test cases** -- "it returns empty when no tasks exist" and "partial unblock: two blockers one cancelled still blocked" test scenarios V5 lacks.
4. **Unexported SQL constant** (`blockedSQL` vs exported `BlockedQuery`) -- better encapsulation since the constant is only used within the package.

**Why V5 wins:** The DRY advantage is the deciding factor. V6's duplicated formatting code is a genuine quality issue that could lead to inconsistencies. V6's test quality advantages are real but secondary -- the extra granularity provides marginal diagnostic benefit, and V5's tests cover all the same functional scenarios (just grouped differently). Both versions satisfy all acceptance criteria.
