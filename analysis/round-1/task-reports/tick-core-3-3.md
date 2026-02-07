# Task tick-core-3-3: Ready query & tick ready command

## Task Summary

The core "what should I work on next?" feature. A task is **ready** when:

1. Status is `open`
2. All blockers are `done` or `cancelled`
3. No children with status `open` or `in_progress`

Requirements:
- `ReadyQuery` -- pure/SQL query returning matching tasks
- Order: priority ASC, created ASC (deterministic)
- Register `ready` subcommand -- alias for `list --ready`
- Output: aligned columns like `tick list`
- Empty: `No tasks found.`, exit 0
- `--quiet`: IDs only
- Query function reusable by blocked query and list filters

### Acceptance Criteria (from plan)

1. Returns tasks matching all three conditions
2. Open/in_progress blockers exclude task
3. Open/in_progress children exclude task
4. Cancelled blockers unblock
5. Only `open` status returned
6. Deep nesting handled correctly
7. Deterministic ordering
8. `tick ready` outputs aligned columns
9. Empty -> `No tasks found.`, exit 0
10. `--quiet` outputs IDs only
11. Query function reusable by blocked query and list filters

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Returns tasks matching all 3 conditions | PASS -- SQL with 3 WHERE clauses | PASS -- SQL with 3 WHERE clauses | PASS -- SQL with 3 WHERE clauses |
| Open/in_progress blockers exclude task | PASS -- `NOT IN ('done','cancelled')` check | PASS -- `NOT IN ('done','cancelled')` check | PASS -- `IN ('open','in_progress')` check |
| Open/in_progress children exclude task | PASS -- `IN ('open','in_progress')` child check | PASS -- `IN ('open','in_progress')` child check | PASS -- `IN ('open','in_progress')` child check |
| Cancelled blockers unblock | PASS -- tested with done+cancelled mix | PASS -- tested with done+cancelled mix | PASS -- tested done and cancelled separately |
| Only `open` status returned | PASS -- `WHERE t.status = 'open'` | PASS -- `WHERE status = 'open'` | PASS -- `WHERE t.status = 'open'` |
| Deep nesting handled correctly | FAIL -- no deep nesting test | PASS -- 3-level grandparent test | PASS -- 3-level root/mid/leaf test |
| Deterministic ordering | PASS -- `ORDER BY priority ASC, created ASC` | PASS -- `ORDER BY priority ASC, created ASC` | PASS -- `ORDER BY t.priority ASC, t.created ASC` |
| `tick ready` outputs aligned columns | PASS -- `%-12s %-12s %-4s %s` format | PASS -- reuses list.go formatting | PASS -- `%-12s %-12s %-4s %s` format |
| Empty -> `No tasks found.`, exit 0 | PASS -- tested | PASS -- tested | PASS -- tested |
| `--quiet` outputs IDs only | PASS -- tested | PASS -- tested | PASS -- tested |
| Query function reusable | PARTIAL -- `cmdListFiltered` takes SQL string, somewhat reusable | PASS -- `ReadySQL` exported const, `list --ready` flag | PASS -- `ReadyCondition` exported const + `queryReadyTasks` exported func |

## Implementation Comparison

### Approach

#### V1: New file `ready.go` with `cmdListFiltered` helper

V1 creates a dedicated `ready.go` file containing both the SQL query and a generic `cmdListFiltered` method. The `ready` command is a thin wrapper:

```go
// ready.go line 29-31
func (a *App) cmdReady(workDir string, args []string) error {
    return a.cmdListFiltered(workDir, readyQuery)
}
```

The `cmdListFiltered` function (lines 34-93) is a standalone method that opens the store, runs an arbitrary SQL query, and handles output formatting. The SQL is a `const` string:

```go
const readyQuery = `
SELECT t.id, t.title, t.status, t.priority
FROM tasks t
WHERE t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )
ORDER BY t.priority ASC, t.created ASC
`
```

Registration is in `cli.go`:
```go
case "ready":
    err = a.cmdReady(workDir, cmdArgs)
```

Key design decisions:
- Unexported `readyQuery` constant -- not reusable by other packages
- `cmdListFiltered` accepts arbitrary SQL, making it theoretically reusable but tightly coupled to the 4-column output format
- Duplicates store-opening logic already present in `cmdList`
- Uses `a.opts.Quiet` for quiet flag (different field name than other versions)

#### V2: Extends `list.go` with `--ready` flag, no new file

V2 takes a fundamentally different approach: it modifies the existing `list.go` file to add a `--ready` flag, and wires `tick ready` as an alias that passes `["--ready"]` to `runList`:

```go
// app.go
case "list":
    return a.runList(cmdArgs)
case "ready":
    return a.runList([]string{"--ready"})
```

The SQL is an exported constant in `list.go`:
```go
const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )
ORDER BY priority ASC, created ASC`
```

Flag parsing is a simple loop:
```go
func parseListFlags(args []string) (ready bool) {
    for _, arg := range args {
        if arg == "--ready" {
            ready = true
        }
    }
    return
}
```

Key design decisions:
- **No new file** -- extends existing `list.go`
- `ReadySQL` exported -- reusable by other code
- `tick ready` is truly an alias for `list --ready`
- Column order difference: `id, status, priority, title` (status before priority) vs V1/V3 `id, title, status, priority`
- Uses `NOT IN` subquery pattern instead of `NOT EXISTS` -- semantically equivalent but potentially less efficient on large datasets
- `runList` signature changed from `()` to `(args []string)` -- modifies existing API
- Also adds `parseListFlags` helper and `listAllSQL` constant to decouple the default query

#### V3: New file `ready.go` with exported `ReadyCondition` and `queryReadyTasks`

V3 creates `ready.go` with a maximally reusable design. The SQL WHERE clause is exported as a fragment:

```go
const ReadyCondition = `
    t.status = 'open'
    AND NOT EXISTS (
        SELECT 1 FROM dependencies d
        JOIN tasks blocker ON d.blocked_by = blocker.id
        WHERE d.task_id = t.id
          AND blocker.status IN ('open', 'in_progress')
    )
    AND NOT EXISTS (
        SELECT 1 FROM tasks child
        WHERE child.parent = t.id
          AND child.status IN ('open', 'in_progress')
    )
`
```

A separate `queryReadyTasks` function assembles the full query:
```go
func queryReadyTasks(db *sql.DB) ([]taskRow, error) {
    query := `
        SELECT t.id, t.title, t.status, t.priority
        FROM tasks t
        WHERE ` + ReadyCondition + `
        ORDER BY t.priority ASC, t.created ASC
    `
    // ...
}
```

And a package-level `taskRow` struct:
```go
type taskRow struct {
    ID       string
    Title    string
    Status   string
    Priority int
}
```

Registration in `cli.go`:
```go
case "ready":
    return a.runReady()
```

Key design decisions:
- `ReadyCondition` is a SQL fragment (WHERE clause only), not a complete query -- maximally composable
- `queryReadyTasks` is a standalone function (not a method), takes `*sql.DB` directly -- reusable outside the App context
- `taskRow` at package level -- reusable by future list filters
- `runReady` returns `int` (exit code) directly rather than `error`
- Also updates `docs/workflow/implementation/tick-core-context.md` with integration notes
- Adds `ready` to the help/usage output

### SQL Approach Differences

| Aspect | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Export level | unexported `readyQuery` | exported `ReadySQL` | exported `ReadyCondition` |
| SQL scope | Complete query | Complete query | WHERE clause fragment |
| Blocker check | `NOT IN ('done','cancelled')` | `NOT IN ('done','cancelled')` | `IN ('open','in_progress')` |
| Subquery style | `NOT EXISTS` | `NOT IN` | `NOT EXISTS` |
| Table alias | `t` | no alias | `t` |

The blocker check logic differs subtly:
- V1/V2: `blocker.status NOT IN ('done', 'cancelled')` -- excludes if blocker is anything other than done/cancelled. If a new status were added, it would default to blocking.
- V3: `blocker.status IN ('open', 'in_progress')` -- excludes only if blocker is open or in_progress. If a new status were added, it would default to non-blocking.

Both are correct for the current status set (`open`, `in_progress`, `done`, `cancelled`), but V1/V2's approach is more defensive (new statuses block by default), while V3's is more permissive.

### Code Quality

**Error handling patterns:**

V1 returns `error` from `cmdReady` and wraps errors:
```go
return fmt.Errorf("querying tasks: %w", err)
return fmt.Errorf("scanning task: %w", err)
```

V2 returns `error` from `runList` and wraps:
```go
return fmt.Errorf("failed to query tasks: %w", err)
```

V3 returns `int` (exit code) from `runReady` and prints to stderr:
```go
fmt.Fprintf(a.Stderr, "Error: %s\n", err)
return 1
```

V3's pattern is a notable departure. Returning int directly avoids the need for error-to-exit-code conversion elsewhere, but it means errors are formatted in `runReady` rather than centrally. The `queryReadyTasks` function properly returns errors for reuse scenarios.

**DRY / Code Reuse:**

- V1: Creates `cmdListFiltered` -- a new generic filtered-list method. However, the existing `cmdList` likely has similar logic that isn't shared. The method duplicates store-opening, query execution, and formatting.
- V2: Best DRY -- extends the existing `runList` to accept args, reuses all formatting and store logic. Zero code duplication.
- V3: Creates `queryReadyTasks` and `taskRow` at package level for future reuse, but duplicates the store-open/format pattern from what presumably exists in list.go.

**Naming conventions:**

- V1: `cmdReady`, `cmdListFiltered`, `readyQuery`, `taskRow` (local struct) -- consistent with `cmd` prefix pattern
- V2: `ReadySQL`, `parseListFlags`, `listAllSQL` -- exported SQL, clean naming
- V3: `ReadyCondition`, `queryReadyTasks`, `runReady`, `taskRow` (package-level struct) -- exported condition, `run` prefix for command handlers, `query` prefix for data access

**Type safety:**

All three versions define a `taskRow` struct with the same fields. V3 makes it package-level, V1 makes it local to `cmdListFiltered`, V2 makes it local to `runList`.

### Test Quality

#### V1 Test Functions (9 tests, all in `TestReadyCommand`)

1. `returns open task with no blockers and no children` -- creates task via CLI, checks it appears
2. `excludes task with open blocker` -- creates blocker+blocked pair, verifies only blocker shows
3. `excludes task with in_progress blocker` -- starts blocker, verifies blocked task excluded
4. `includes task when all blockers done/cancelled` -- marks one done, one cancelled, verifies dependent ready
5. `excludes parent with open children` -- creates parent+child, verifies only child appears
6. `includes parent when all children closed` -- marks child done, verifies parent appears
7. `excludes in_progress/done/cancelled tasks` -- tests all three non-open statuses in one test
8. `returns empty list when no tasks ready` -- all tasks blocked/started, verifies "No tasks found."
9. `orders by priority ASC then created ASC` -- 3 tasks with priorities 0,2,3, checks order
10. `outputs IDs only with --quiet` -- verifies ID-only output

**Missing from V1:**
- No deep nesting test (grandparent -> parent -> child)
- No aligned columns test (checks content but not column alignment)
- No separate "excludes parent with in_progress children" test
- No separate "excludes done tasks" and "excludes cancelled tasks" tests (combined into one)

**Testing approach:** V1 uses CLI-level helpers (`initTickDir`, `createTask`, `runCmd`, `extractID`) that create tasks via the actual CLI. This is more integration-style testing.

#### V2 Test Functions (14 tests in `TestReadyQuery` + 1 in `TestReadyViaListFlag`)

`TestReadyQuery`:
1. `it returns open task with no blockers and no children`
2. `it excludes task with open blocker`
3. `it excludes task with in_progress blocker`
4. `it includes task when all blockers done or cancelled`
5. `it excludes parent with open children`
6. `it excludes parent with in_progress children`
7. `it includes parent when all children closed`
8. `it excludes in_progress tasks`
9. `it excludes done tasks`
10. `it excludes cancelled tasks`
11. `it handles deep nesting -- only deepest incomplete ready`
12. `it returns empty list when no tasks ready`
13. `it orders by priority ASC then created ASC`
14. `it outputs aligned columns via tick ready`
15. `it prints 'No tasks found.' when empty`
16. `it outputs IDs only with --quiet`

`TestReadyViaListFlag`:
17. `it works via list --ready flag as well`

**Testing approach:** V2 uses `setupTickDirWithContent` and `taskJSONL` helpers to construct JSONL data directly, then creates `NewApp()` instances with custom stdout. Also includes a custom `itoa` helper. Tests the `list --ready` alias separately. The aligned columns test is thorough, checking header presence, column position alignment (`strings.Index`), and data content.

**Unique to V2:**
- `list --ready` alias test
- Column alignment position verification
- Separate in_progress/done/cancelled exclusion tests
- Parent with in_progress children test (separate from open children)
- Custom `taskJSONL` and `itoa` test helpers for data setup

#### V3 Test Functions (15 tests in `TestReadyQuery` + 3 in `TestReadyCommand`)

`TestReadyQuery`:
1. `it returns open task with no blockers and no children`
2. `it excludes task with open blocker`
3. `it excludes task with in_progress blocker`
4. `it includes task when all blockers done`
5. `it includes task when all blockers cancelled`
6. `it excludes parent with open children`
7. `it excludes parent with in_progress children`
8. `it includes parent when all children closed`
9. `it excludes in_progress tasks`
10. `it excludes done tasks`
11. `it excludes cancelled tasks`
12. `it handles deep nesting - only deepest incomplete ready`
13. `it returns empty list when no tasks ready`
14. `it orders by priority ASC then created ASC`

`TestReadyCommand`:
15. `it outputs aligned columns via tick ready`
16. `it prints 'No tasks found.' when empty`
17. `it outputs IDs only with --quiet`

**Testing approach:** V3 uses `setupTickDir` and `setupTaskFull` helpers that take explicit parameters (id, title, status, priority, description, parent, blockedBy, created, updated, closed). Uses `bytes.Buffer` for stdout/stderr capture. Separates query logic tests (`TestReadyQuery`) from command output tests (`TestReadyCommand`).

**Unique to V3:**
- Separate tests for "done blocker unblocks" and "cancelled blocker unblocks" (V2 combines into one)
- Clear separation between query tests and command output tests
- Uses `setupTaskFull` with explicit closed timestamp for done/cancelled tasks

#### Test Coverage Diff

| Test Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Open task with no blockers/children | Yes | Yes | Yes |
| Excludes task with open blocker | Yes | Yes | Yes |
| Excludes task with in_progress blocker | Yes | Yes | Yes |
| Includes when blockers done+cancelled (combined) | Yes | Yes | No (split) |
| Includes when blocker done (separate) | No | No | Yes |
| Includes when blocker cancelled (separate) | No | No | Yes |
| Excludes parent with open children | Yes | Yes | Yes |
| Excludes parent with in_progress children | No | Yes | Yes |
| Includes parent when all children closed | Yes | Yes | Yes |
| Excludes in_progress tasks | Yes (combined) | Yes | Yes |
| Excludes done tasks | Yes (combined) | Yes | Yes |
| Excludes cancelled tasks | Yes (combined) | Yes | Yes |
| Deep nesting (3+ levels) | No | Yes | Yes |
| Empty list | Yes | Yes | Yes |
| Priority+created ordering | Yes | Yes | Yes |
| Aligned columns | No | Yes | Yes |
| Empty -> "No tasks found." | Implicit | Yes | Yes |
| --quiet IDs only | Yes | Yes | Yes |
| `list --ready` alias | No | Yes | No |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 | 5 | 6 |
| Lines added | 274 | 554 | 620 |
| Impl LOC | 93 (ready.go) + 2 (cli.go) = 95 | 43 (list.go changes) + 4 (app.go) = 47 | 123 (ready.go) + 3 (cli.go) = 126 |
| Test LOC | 179 | 510 | 477 |
| Test functions | 10 | 17 | 17 |

## Verdict

**V2 is the best implementation** for the following reasons:

1. **Best adherence to the spec**: The plan says `tick ready` is an "alias for `list --ready`". V2 is the only version that literally implements this -- `ready` case calls `runList([]string{"--ready"})`, and `list --ready` works independently. V1 and V3 create standalone `ready` commands that do not support `list --ready`.

2. **Best code reuse / DRY**: V2 modifies the existing `list.go` rather than creating a new file with duplicated store-opening and formatting logic. The `ReadySQL` constant is exported for reuse. No code duplication.

3. **Most thorough testing**: V2 has 17 test functions covering every edge case including deep nesting, column alignment verification (with position checking), parent with in_progress children, and the `list --ready` alias. Only V3 matches in test count but lacks the `list --ready` alias test.

4. **Exported SQL for reuse**: `ReadySQL` is exported and can be reused by blocked query and list filters as the acceptance criteria require.

**V3 is a close second**: It has the most composable SQL design (`ReadyCondition` as a WHERE fragment), a cleanly separated `queryReadyTasks` function, and good test coverage. However, it does not implement `list --ready` (only `tick ready`), and it duplicates store/formatting logic that already exists in `list.go`. The `ReadyCondition` fragment approach is arguably more future-proof than V2's full-query export, but the lack of `list --ready` support is a spec gap.

**V1 is the weakest**: It has 30% fewer tests (missing deep nesting, column alignment, parent with in_progress children), uses an unexported SQL constant (not reusable), does not support `list --ready`, and the `cmdListFiltered` helper duplicates logic from the existing list command. The test approach using CLI helpers is more integration-like but results in less granular coverage.
