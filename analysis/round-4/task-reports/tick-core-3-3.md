# Task tick-core-3-3: Ready query & tick ready command

## Task Summary

The core value of Tick -- "what should I work on next?" A task is ready when it is `open`, all blockers are closed, and it has no open children. `tick ready` is an alias for `tick list --ready`. The task requires building both a reusable query function and the CLI command.

**Acceptance Criteria:**
1. Returns tasks matching all three conditions (open, unblocked, no open children)
2. Open/in_progress blockers exclude task
3. Open/in_progress children exclude task
4. Cancelled blockers unblock
5. Only `open` status returned
6. Deep nesting handled correctly
7. Deterministic ordering (priority ASC, created ASC)
8. `tick ready` outputs aligned columns
9. Empty -> `No tasks found.`, exit 0
10. `--quiet` outputs IDs only
11. Query function reusable by blocked query and list filters

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Returns tasks matching all three conditions | PASS - SQL query uses three WHERE clauses: `status = 'open'`, `NOT EXISTS` for blocker check, `NOT EXISTS` for children check | PASS - Identical SQL query logic |
| Open/in_progress blockers exclude task | PASS - SQL checks `blocker.status NOT IN ('done', 'cancelled')` | PASS - Identical SQL logic |
| Open/in_progress children exclude task | PASS - SQL checks `child.status IN ('open', 'in_progress')` | PASS - Identical SQL logic |
| Cancelled blockers unblock | PASS - Cancelled is in the `('done', 'cancelled')` allowed set | PASS - Identical |
| Only `open` status returned | PASS - `WHERE t.status = 'open'` | PASS - Identical |
| Deep nesting handled correctly | PASS - The `NOT EXISTS` children check only looks at direct children; combined with `status = 'open'`, parents with open children are excluded, leaving only leaves ready | PASS - Identical approach |
| Deterministic ordering | PASS - `ORDER BY t.priority ASC, t.created ASC` | PASS - Identical |
| `tick ready` outputs aligned columns | PASS - Delegates to `printListTable` which uses `fmt.Fprintf` with column widths | PASS - Inline `fmt.Fprintf` with `%-12s%-13s%-5s%s` format |
| Empty -> `No tasks found.`, exit 0 | PASS - Checks `len(rows) == 0`, prints message, returns nil | PASS - Identical logic |
| `--quiet` outputs IDs only | PASS - Loops over rows printing `r.id` when `ctx.Quiet` is set | PASS - Identical logic using `quiet` parameter |
| Query function reusable by blocked query and list filters | PASS - Exports `ReadyQuery` as a `const`, reused in the worktree's `list.go` via `buildReadyFilterQuery` | FAIL - Uses unexported `readySQL` const; `RunReady` is a standalone function with its own query execution, not composable with list filters |

## Implementation Comparison

### Approach

**V5** takes a delegation approach. At commit time, `runReady` was a standalone function (83 LOC) that executed the query directly. However, it exported the SQL as `ReadyQuery` (a public constant), which the codebase later leverages in `list.go` via `buildReadyFilterQuery`. By the worktree's final state, `runReady` was refactored to a 3-line function:

```go
// V5 worktree final state (ready.go line 31-33)
func runReady(ctx *Context) error {
    ctx.Args = append([]string{"--ready"}, ctx.Args...)
    return runList(ctx)
}
```

This means V5's `tick ready` truly became an alias for `tick list --ready`, exactly as the spec states. The `readyWhereClause` was also extracted as a shared constant for reuse by stats queries.

At commit time, V5's approach was a self-contained `runReady` that used `printListTable` for formatting:

```go
// V5 at commit time (ready.go line 36-83)
func runReady(ctx *Context) error {
    // ... store setup ...
    err = store.Query(func(db *sql.DB) error {
        sqlRows, err := db.Query(ReadyQuery)
        // ... scanning ...
    })
    // ...
    printListTable(ctx.Stdout, rows)
    return nil
}
```

**V6** takes a self-contained approach. `RunReady` is an exported function (100 LOC) that manages its own store, query execution, and output formatting entirely independently:

```go
// V6 (ready.go line 40-100)
func RunReady(dir string, quiet bool, stdout io.Writer) error {
    tickDir, err := DiscoverTickDir(dir)
    // ... store setup ...
    // ... query execution ...
    // Print header (same format as list)
    fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")
    for _, r := range rows {
        fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
    }
    return nil
}
```

V6's `handleReady` in `app.go` calls `RunReady` directly:

```go
// V6 (app.go line 117-122)
func (a *App) handleReady(flags globalFlags) error {
    dir, err := a.Getwd()
    if err != nil {
        return fmt.Errorf("could not determine working directory: %w", err)
    }
    return RunReady(dir, flags.quiet, a.Stdout)
}
```

**Key difference:** V5 leverages existing list infrastructure and exports the query for reuse. V6 duplicates formatting logic and keeps the query unexported/non-reusable.

### Code Quality

**SQL Query Definition:**

Both versions define identical SQL queries. The SQL logic is the same in both:

```sql
-- Both versions (identical)
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )
ORDER BY t.priority ASC, t.created ASC
```

**Naming and Export:**

- V5: `ReadyQuery` (exported const) -- enables reuse across packages. Error messages use `"querying ready tasks: %w"` and `"scanning ready task row: %w"`.
- V6: `readySQL` (unexported const) -- limits reuse to the `cli` package only. Error messages use `"failed to query ready tasks: %w"` and `"failed to scan ready task row: %w"`.

V5's error message style (`"querying ready tasks"`) is more idiomatic Go (verbs as gerunds). V6 uses `"failed to ..."` which is slightly redundant since errors already imply failure.

**DRY Principle:**

V5 reuses the `listRow` type and `printListTable` function from `list.go`. V6 defines `listRow` as a local type inside `RunReady` and duplicates the formatting logic inline. V6's `listRow` is identical in structure to what the list command would use but is not shared:

```go
// V6 (ready.go line 56-61) - local type definition
type listRow struct {
    id       string
    status   string
    priority int
    title    string
}
```

**Function Signatures:**

- V5: `runReady(ctx *Context) error` -- uses the shared `Context` pattern, consistent with all other commands.
- V6: `RunReady(dir string, quiet bool, stdout io.Writer) error` -- exported, takes primitive parameters. This means the command does not have access to format selection (toon/pretty/json) or verbose logging.

**Error Handling:** Both versions properly wrap errors with `fmt.Errorf("%w", err)`, close rows with defer, and check `sqlRows.Err()` after iteration.

### Test Quality

**V5 Test Functions (12 subtests in 2 top-level functions):**

`TestReadyQuery` (9 subtests):
1. `"it returns open task with no blockers and no children"` -- basic happy path
2. `"it excludes task with open/in_progress blocker"` -- tests both open AND in_progress blockers in one test (4 tasks)
3. `"it includes task when all blockers done/cancelled"` -- tests both done AND cancelled blockers
4. `"it excludes parent with open/in_progress children"` -- tests both open AND in_progress children
5. `"it includes parent when all children closed"` -- done + cancelled children
6. `"it excludes in_progress/done/cancelled tasks"` -- all non-open statuses in one test
7. `"it handles deep nesting - only deepest incomplete ready"` -- 3-level hierarchy
8. `"it returns empty list when no tasks ready"` -- empty result
9. `"it orders by priority ASC then created ASC"` -- 3 tasks with varying priority/created

`TestReadyCommand` (3 subtests):
10. `"it outputs aligned columns via tick ready"` -- header + column formatting
11. `"it prints 'No tasks found.' when empty"` -- empty project
12. `"it outputs IDs only with --quiet"` -- quiet mode

**V6 Test Functions (17 subtests in 1 top-level function):**

`TestReady` (17 subtests):
1. `"it returns open task with no blockers and no children"`
2. `"it excludes task with open blocker"` -- open blocker ONLY
3. `"it excludes task with in_progress blocker"` -- in_progress blocker ONLY
4. `"it includes task when all blockers done"` -- done ONLY
5. `"it includes task when all blockers cancelled"` -- cancelled ONLY
6. `"it excludes parent with open children"` -- open children ONLY
7. `"it excludes parent with in_progress children"` -- in_progress children ONLY
8. `"it includes parent when all children closed"`
9. `"it excludes in_progress tasks"` -- separate test
10. `"it excludes done tasks"` -- separate test
11. `"it excludes cancelled tasks"` -- separate test
12. `"it handles deep nesting - only deepest incomplete ready"`
13. `"it returns empty list when no tasks ready"`
14. `"it orders by priority ASC then created ASC"` -- 4 tasks (more thorough)
15. `"it outputs aligned columns via tick ready"` -- exact string matching
16. `"it prints 'No tasks found.' when empty"`
17. `"it outputs IDs only with --quiet"` -- exact string matching

**Test Coverage Diff:**

V6 has 5 more subtests because it splits combined scenarios into individual tests:
- V5 `"excludes task with open/in_progress blocker"` -> V6 splits into 2 tests
- V5 `"includes task when all blockers done/cancelled"` -> V6 splits into 2 tests
- V5 `"excludes parent with open/in_progress children"` -> V6 splits into 2 tests
- V5 `"excludes in_progress/done/cancelled tasks"` -> V6 splits into 3 tests

V6 also adds an extra empty-state test: `"it prints 'No tasks found.' when empty"` appears as a separate test alongside `"it returns empty list when no tasks ready"` (redundant coverage).

**Test Precision:**

V6 tests use exact string comparison for output validation:
```go
// V6 - exact match
if lines[1] != "tick-aaa111 open         1    Setup Sanctum" {
    t.Errorf("row 1 = %q, want %q", lines[1], "tick-aaa111 open         1    Setup Sanctum")
}
```

V5 tests use `strings.Contains` for most checks:
```go
// V5 - substring match
if !strings.Contains(output, "tick-aaaaaa") {
    t.Errorf("expected output to contain tick-aaaaaa, got %q", output)
}
```

V6's exact matching catches formatting regressions more precisely. V5's substring matching is more resilient to incidental formatting changes but less precise.

**Ordering Test:**

V6 uses 4 tasks (two pairs of priority) which provides better verification of both priority and created ordering. V5 uses 3 tasks (1 high priority, 2 low priority), which is sufficient but less thorough.

**Test Helpers:**

- V5 uses `Run()` function directly (integration-style through full CLI dispatch)
- V6 uses a `runReady` helper that creates an `App` struct directly (unit-test style, more isolated)

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS - Code is properly formatted | PASS - Code is properly formatted |
| Handle all errors explicitly (no naked returns) | PASS - All errors checked and returned | PASS - All errors checked and returned |
| Write table-driven tests with subtests | PARTIAL - Uses subtests via `t.Run` but not table-driven format; each subtest is a standalone scenario | PARTIAL - Same pattern: subtests via `t.Run` but not table-driven |
| Document all exported functions, types, and packages | PASS - `ReadyQuery` and `runReady` are documented | PASS - `RunReady`, `readySQL` documented with godoc comments |
| Propagate errors with fmt.Errorf("%w", err) | PASS - `fmt.Errorf("querying ready tasks: %w", err)` | PASS - `fmt.Errorf("failed to query ready tasks: %w", err)` |
| Ignore errors (avoid _ assignment) | PASS - No ignored errors | PASS - No ignored errors |
| Use panic for normal error handling | PASS - No panics | PASS - No panics |
| Hardcode configuration | PASS - No hardcoded config | PASS - No hardcoded config |

Both versions are PARTIAL on table-driven tests. The spec's test cases (12 scenarios) are individually complex enough that table-driven patterns would be awkward -- each test sets up a different combination of tasks, blockers, and parent relationships. Using individual subtests is a reasonable deviation.

### Spec-vs-Convention Conflicts

**Conflict 1: `tick ready` as alias vs standalone command**

- **Spec says:** "`tick ready` is an alias for `tick list --ready`"
- **Convention:** Code reuse (DRY), single responsibility
- **V5:** At commit time, duplicated the query execution logic. Later refactored to truly delegate: `ctx.Args = append([]string{"--ready"}, ctx.Args...); return runList(ctx)`. The exported `ReadyQuery` constant enables reuse.
- **V6:** Implements `tick ready` as a completely standalone command with its own formatting. Not a true alias -- it cannot benefit from list's filter flags or formatter selection.
- **Assessment:** V5's final approach (delegation) is clearly better. V6's standalone approach means `tick ready` and `tick list --ready` could drift apart in behavior. V5 at commit time was in the same position as V6 -- both duplicated logic -- but V5 exported the query for future reuse.

**Conflict 2: Reusable query function**

- **Spec says:** "Query function reusable by blocked query and list filters"
- **Convention:** Unexported by default (Go convention of minimal API surface)
- **V5:** Exported `ReadyQuery` as a public constant, enabling reuse. Later extracted `readyWhereClause` for even more granular reuse.
- **V6:** Used unexported `readySQL`, making it unreusable outside the ready command.
- **Assessment:** The spec explicitly requires reusability. V5 followed the spec. V6 followed Go convention of minimal exports but violated the spec requirement.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 5 (3 impl + 2 docs) | 5 (3 impl + 2 docs) |
| Lines added | 476 | 514 |
| Impl LOC | 83 (ready.go) + 1 (cli.go) = 84 | 100 (ready.go) + 11 (app.go) = 111 |
| Test LOC | 389 | 400 |
| Test functions | 2 top-level, 12 subtests | 1 top-level, 17 subtests |

## Verdict

**V5 is the better implementation.**

1. **Reusability (spec requirement):** V5 exports `ReadyQuery` as a public constant, directly satisfying the acceptance criterion "Query function reusable by blocked query and list filters." V6 uses unexported `readySQL`, failing this criterion.

2. **DRY:** V5 reuses `listRow` and `printListTable` from `list.go`. V6 duplicates both the type definition (as a local type inside the function) and the formatting logic. This duplication creates a maintenance risk where `list` and `ready` formatting could diverge.

3. **True alias semantics:** V5 later evolved `runReady` into a 3-line delegation to `runList`, making `tick ready` a genuine alias for `tick list --ready`. V6's standalone `RunReady` cannot compose with list filters or benefit from list's formatter selection.

4. **Architecture integration:** V5 uses the shared `Context` pattern consistent with all other commands, supporting format selection (toon/pretty/json) and verbose logging. V6's `RunReady(dir, quiet, stdout)` signature lacks format and verbose support, making it inconsistent with the rest of the CLI.

V6 has marginally more granular tests (17 subtests vs 12), which is a minor advantage. The split of combined test scenarios into individual subtests provides slightly better failure diagnostics. However, this does not outweigh V5's architectural superiority.

The implementation LOC difference (84 vs 111) further reflects V5's efficiency through code reuse.
