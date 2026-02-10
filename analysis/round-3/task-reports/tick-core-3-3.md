# Task 3-3: Ready query & tick ready command

## Task Plan Summary

This task implements the core value proposition of Tick -- answering "what should I work on next?" A task is "ready" when it meets three conditions: (1) status is `open`, (2) all blockers are closed (done or cancelled), and (3) no children have status `open` or `in_progress`. The task requires:

- A `ReadyQuery` -- a pure/SQL query implementing the three conditions
- Ordering: priority ASC, created ASC (deterministic)
- A `ready` subcommand registered as an alias for `list --ready`
- Output: aligned columns like `tick list`
- Empty result: "No tasks found.", exit 0
- `--quiet`: IDs only
- The query must be reusable by blocked query and list filters

The plan specifies 12 tests covering every combination of blockers, children, statuses, deep nesting, ordering, empty results, aligned columns, "No tasks found." message, and quiet mode.

---

## V4 Implementation

### Architecture & Design

V4 creates a new file `internal/cli/ready.go` containing the SQL query and handler, and modifies `internal/cli/list.go` to extract shared rendering logic.

**SQL Query (commit 37e69a6)**:
The `readyQuery` is declared as a `const` string:

```go
const readyQuery = `
SELECT t.id, t.status, t.priority, t.title
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

**Command handler**: The `runReady` method on `*App` follows the same pattern as the existing `runList` -- discovers tick dir, opens store, runs query, renders output. Importantly, V4 at the commit point delegates output rendering to `renderListOutput`, a shared function extracted from `list.go`.

**Shared rendering**: V4 extracts `listRow` struct and `renderListOutput` function from `list.go` into a shared helper:

```go
func renderListOutput(rows []listRow, stdout io.Writer, quiet bool) error {
    if len(rows) == 0 {
        fmt.Fprintln(stdout, "No tasks found.")
        return nil
    }
    if quiet {
        for _, r := range rows {
            fmt.Fprintln(stdout, r.ID)
        }
        return nil
    }
    fmt.Fprintf(stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
    for _, r := range rows {
        fmt.Fprintf(stdout, "%-12s %-12s %-4d %s\n", r.ID, r.Status, r.Priority, r.Title)
    }
    return nil
}
```

Both `runList` and `runReady` call `renderListOutput`, which fulfills the spec's requirement that the ready command shares rendering with list.

**Dispatch**: The `cli.go` switch statement adds a `"ready"` case alongside other commands.

**Reusability**: At the commit point, the query is a standalone `const`. In the current HEAD (post later tasks), V4 evolved to extract `readyConditionsFor(alias string) string` which returns the WHERE clause conditions parameterized by table alias. This makes the ready logic truly reusable: `blockedQuery` and `buildListQuery` both call `readyConditionsFor("t")`. This is an important design decision for the "reusable by blocked query and list filters" requirement.

**Store package**: V4 imports `github.com/leeovery/tick/internal/store`.

### Code Quality

- **Naming**: `readyQuery` (unexported const) is appropriate since it's only used within the package. The `renderListOutput` function name is descriptive. Field names on `listRow` are exported (`ID`, `Status`, etc.).
- **Error handling**: All errors are explicitly handled and wrapped with context via `fmt.Errorf("failed to query ready tasks: %w", err)`. The prefix "failed to" is consistent throughout.
- **Resource management**: `defer s.Close()` and `defer sqlRows.Close()` are correctly placed.
- **Separation**: The rendering logic is properly extracted, eliminating duplication between list and ready.
- **Method receiver**: Uses `(a *App)` consistent with V4's struct-method pattern.

### Test Coverage

V4 provides 12 tests in individual top-level `TestReady_*` functions (490 lines), each containing a single subtest. The tests cover:

| # | Test | Coverage |
|---|------|----------|
| 1 | `TestReady_OpenTaskNoBlockersNoChildren` | Basic happy path |
| 2 | `TestReady_ExcludesTaskWithOpenBlocker` | Open blocker exclusion |
| 3 | `TestReady_ExcludesTaskWithInProgressBlocker` | In-progress blocker exclusion |
| 4 | `TestReady_IncludesTaskWhenAllBlockersDoneOrCancelled` | Both done + cancelled blockers |
| 5 | `TestReady_ExcludesParentWithOpenChildren` | Parent exclusion |
| 6 | `TestReady_ExcludesParentWithInProgressChildren` | IP child exclusion |
| 7 | `TestReady_IncludesParentWhenAllChildrenClosed` | All children closed |
| 8 | `TestReady_ExcludesNonOpenStatuses` | All 4 statuses tested |
| 9 | `TestReady_DeepNesting` | 3-level hierarchy |
| 10 | `TestReady_EmptyList` | No ready tasks |
| 11 | `TestReady_OrderByPriorityThenCreated` | 4 tasks, 3 priorities |
| 12 | `TestReady_AlignedColumnsOutput` | Header + data format |
| 13 | `TestReady_NoTasksFoundMessage` | Empty project |
| 14 | `TestReady_QuietFlag` | IDs only output |

All 12 plan tests are covered, plus V4 adds a 13th (NoTasksFoundMessage with empty project) and 14th separate test.

**Test structure**: Each test function creates tasks with explicit timestamps using `time.Date(2026, 1, 19, ...)`, sets up an initialized directory, runs the CLI via `app.Run([]string{"tick", "ready"})`, and asserts on stdout output.

**Assertions**: Tests use `strings.Contains` / `!strings.Contains` for inclusion/exclusion checks. The ordering test verifies line positions explicitly. The empty list test checks for the exact string "No tasks found." via `strings.TrimSpace`.

**Weakness**: V4 tests construct `task.Task` structs directly with field literals (e.g., `{ID: "tick-aaa111", Title: "Simple task", Status: task.StatusOpen, ...}`) rather than using a constructor. This is verbose but provides clarity.

### Spec Compliance

| Requirement | Met? | Notes |
|-------------|------|-------|
| ReadyQuery with 3 conditions | Yes | SQL uses `WHERE status='open'`, NOT EXISTS for blockers, NOT EXISTS for children |
| Order: priority ASC, created ASC | Yes | `ORDER BY t.priority ASC, t.created ASC` |
| Register `ready` subcommand | Yes | Added to `cli.go` switch |
| Alias for `list --ready` | Partial | At commit point, it's a standalone handler sharing `renderListOutput`, not literally delegating to list. Later evolved to be truly aliased via list. |
| Output: aligned columns like tick list | Yes | Uses shared `renderListOutput` |
| Empty: "No tasks found.", exit 0 | Yes | Handled in `renderListOutput` |
| `--quiet`: IDs only | Yes | Handled in `renderListOutput` |
| Query reusable by blocked/list | Partial | At commit point, query is const. Later evolved to `readyConditionsFor()` helper making it reusable. |

### golang-pro Skill Compliance

| Rule | Compliance | Notes |
|------|------------|-------|
| Handle all errors explicitly | Yes | Every error is checked |
| Write table-driven tests with subtests | Partial | Tests use subtests but are not table-driven; each is a separate function |
| Document all exported functions/types | N/A | `readyQuery` is unexported; `renderListOutput` is not exported |
| Propagate errors with `fmt.Errorf("%w", err)` | Yes | `fmt.Errorf("failed to query ready tasks: %w", err)` |
| No panic for error handling | Yes | No panics |
| No ignored errors | Yes | All errors handled |

---

## V5 Implementation

### Architecture & Design

V5 creates `internal/cli/ready.go` with the SQL query as an exported `const ReadyQuery` and a standalone `runReady` handler function.

**SQL Query (commit 6fbacef)**:
```go
const ReadyQuery = `
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
`
```

The query is functionally identical to V4 (the only difference is the order of `blocker.id = d.blocked_by` vs `d.blocked_by = blocker.id`, which is semantically equivalent).

**Command handler**: At the commit point, `runReady` is a standalone function (free function, not a method) that takes `*Context`. It opens the store, runs the query, and renders output inline -- it does NOT delegate to `renderListOutput` or `printListTable` initially. Instead it duplicates the empty/quiet/table rendering logic inline:

```go
if len(rows) == 0 {
    fmt.Fprintln(ctx.Stdout, "No tasks found.")
    return nil
}
if ctx.Quiet {
    for _, r := range rows {
        fmt.Fprintln(ctx.Stdout, r.id)
    }
    return nil
}
printListTable(ctx.Stdout, rows)
return nil
```

This means at the commit point, the empty-list and quiet-mode logic is duplicated between `runReady` and `runList`. The `printListTable` function is shared for the table rendering only.

**Evolution**: In the current HEAD (after later tasks), V5 refactored `runReady` to delegate to `runList`:
```go
func runReady(ctx *Context) error {
    ctx.Args = append([]string{"--ready"}, ctx.Args...)
    return runList(ctx)
}
```
This is the cleanest possible alias implementation. Also, V5 extracted `readyWhereClause` as a shared `const` for reuse in `ReadyQuery` and `StatsReadyCountQuery`.

**Dispatch**: V5 uses a `commands` map for dispatch rather than a switch statement:
```go
var commands = map[string]func(*Context) error{
    ...
    "ready":  runReady,
}
```
This is more idiomatic and extensible.

**Store package**: V5 imports `github.com/leeovery/tick/internal/engine` (different package name from V4's `store`).

**listRow fields**: V5 uses unexported fields (`id`, `status`, `priority`, `title`) on `listRow`, which is correct since the struct is only used within the `cli` package.

### Code Quality

- **Naming**: `ReadyQuery` is exported -- this is intentional for reuse by other packages or tests. The `listRow` struct fields are unexported (lowercase), which is slightly more idiomatic for package-internal types. The `runReady` function name matches the dispatch pattern.
- **Error handling**: All errors wrapped with context: `fmt.Errorf("querying ready tasks: %w", err)`. The prefix style ("querying" vs V4's "failed to query") is slightly more concise and follows Go convention of not starting error messages with capital letters or "failed to".
- **Resource management**: `defer store.Close()` and `defer sqlRows.Close()` correctly placed.
- **Duplication**: At the commit point, there is code duplication -- empty-list and quiet-mode handling are inlined in both `runReady` and `runList`. The `printListTable` function is shared but the surrounding rendering logic is not. This was later cleaned up.
- **Free function pattern**: V5 uses `func runReady(ctx *Context) error` (free function) vs V4's `func (a *App) runReady(args []string) error` (method). The free-function + Context pattern is more testable and composable.

### Test Coverage

V5 provides tests organized into two top-level functions: `TestReadyQuery` (9 subtests) and `TestReadyCommand` (3 subtests), totaling 12 subtests in 389 lines.

| # | Test | Coverage |
|---|------|----------|
| 1 | `TestReadyQuery / open task with no blockers and no children` | Basic happy path |
| 2 | `TestReadyQuery / excludes task with open/in_progress blocker` | Combines open AND in_progress blocker in ONE test |
| 3 | `TestReadyQuery / includes task when all blockers done/cancelled` | Both done + cancelled |
| 4 | `TestReadyQuery / excludes parent with open/in_progress children` | Combines open AND IP children in ONE test |
| 5 | `TestReadyQuery / includes parent when all children closed` | Done + cancelled children |
| 6 | `TestReadyQuery / excludes in_progress/done/cancelled tasks` | All non-open statuses |
| 7 | `TestReadyQuery / deep nesting` | 3-level hierarchy |
| 8 | `TestReadyQuery / empty list` | No ready tasks |
| 9 | `TestReadyQuery / orders by priority ASC then created ASC` | 3 tasks, 2 priorities |
| 10 | `TestReadyCommand / aligned columns via tick ready` | Header + data |
| 11 | `TestReadyCommand / 'No tasks found.' when empty` | Empty project |
| 12 | `TestReadyCommand / IDs only with --quiet` | Quiet mode |

All 12 plan tests are covered. V5 is slightly more efficient by combining related scenarios: test #2 covers both open and in_progress blockers in one subtest (4 tasks), and test #4 covers both open and IP children in one subtest (4 tasks).

**Test structure**: V5 uses `task.NewTask(id, title)` constructor which provides sensible defaults (status=open, priority=2, timestamps=now). This is cleaner and less verbose than V4's struct literals.

**Assertions**: Same `strings.Contains` pattern as V4. The ordering test uses 3 tasks (vs V4's 4), which is sufficient. The quiet test validates that each line has no spaces (ensuring only ID, no other columns).

**Weakness**: The ordering test uses only 2 distinct priorities (1 and 3) vs V4's 3 (1, 2, 3). Both are adequate but V4's is slightly more thorough.

### Spec Compliance

| Requirement | Met? | Notes |
|-------------|------|-------|
| ReadyQuery with 3 conditions | Yes | Identical SQL logic |
| Order: priority ASC, created ASC | Yes | Same ORDER BY clause |
| Register `ready` subcommand | Yes | Added to `commands` map |
| Alias for `list --ready` | Partial | At commit point, standalone handler. Later evolved to delegate to runList with `--ready` prepended. |
| Output: aligned columns like tick list | Yes | Uses shared `printListTable` at commit; aligned columns |
| Empty: "No tasks found.", exit 0 | Yes | Inlined in handler |
| `--quiet`: IDs only | Yes | Inlined in handler |
| Query reusable by blocked/list | Partial | At commit, exported `ReadyQuery` const. Later evolved to `readyWhereClause` shared const. |

### golang-pro Skill Compliance

| Rule | Compliance | Notes |
|------|------------|-------|
| Handle all errors explicitly | Yes | Every error checked |
| Write table-driven tests with subtests | Partial | Subtests used, grouped logically into TestReadyQuery and TestReadyCommand, but not truly table-driven |
| Document all exported functions/types | Yes | `ReadyQuery` const has GoDoc. `runReady` is unexported but documented. |
| Propagate errors with `fmt.Errorf("%w", err)` | Yes | `fmt.Errorf("querying ready tasks: %w", err)` |
| No panic for error handling | Yes | No panics |
| No ignored errors | Yes | All errors handled |

---

## Comparative Analysis

### Where V4 is Better

1. **Shared rendering from the start**: V4 at the commit point extracts `renderListOutput` as a shared function used by both `runList` and `runReady`. This eliminates duplication of empty-list, quiet-mode, and table-rendering logic from day one. V5 at the commit point has the empty/quiet logic duplicated inline in `runReady` and `runList`, relying on a later task to consolidate.

2. **Slightly more thorough ordering test**: V4's ordering test uses 4 tasks across 3 priority levels (1, 2, 3), while V5 uses 3 tasks across 2 priority levels (1, 3). V4's test is marginally more comprehensive in verifying the sort.

3. **Separate open/IP blocker tests**: V4 has `TestReady_ExcludesTaskWithOpenBlocker` and `TestReady_ExcludesTaskWithInProgressBlocker` as separate test functions, making failure diagnosis more precise. V5 combines them into one subtest.

4. **Test count naming matches plan**: V4's test function names map almost 1:1 to the plan's test list, making traceability straightforward.

### Where V5 is Better

1. **Exported query constant**: V5 exports `ReadyQuery` (capital R), making it directly accessible for reuse by other packages or test files. V4's `readyQuery` is unexported. While both are within the same package, V5's approach is more forward-thinking for reusability.

2. **Cleaner dispatch architecture**: V5 uses a `commands` map:
   ```go
   var commands = map[string]func(*Context) error{
       "ready": runReady,
   }
   ```
   vs V4's growing switch statement. The map-based approach is more idiomatic, extensible, and eliminates the repeated `if err := ...; err != nil { a.writeError(err); return 1 }` boilerplate for each command.

3. **Context-based design**: V5's `runReady(ctx *Context) error` free function pattern is cleaner than V4's `(a *App) runReady(args []string) error` method pattern. The `Context` struct encapsulates all invocation state, making handlers more testable and composable. The `Run()` function signature `Run(args, workDir, stdout, stderr, isTTY)` is more explicit than constructing an `App` struct.

4. **Unexported listRow fields**: V5's `listRow{id, status, priority, title}` uses unexported fields, which is more correct for a package-internal type. V4's exported fields (`ID`, `Status`) on an unexported type is a Go anti-pattern (though harmless).

5. **More concise error messages**: V5's `"querying ready tasks: %w"` follows Go convention better than V4's `"failed to query ready tasks: %w"`. Go errors should read naturally when chained: `"opening store: querying ready tasks: ..."` is more natural than `"opening store: failed to query ready tasks: ..."`.

6. **Test constructor usage**: V5 uses `task.NewTask("tick-aaaaaa", "Simple open task")` which provides sensible defaults, making tests more concise and less prone to errors from forgetting required fields. V4 constructs structs explicitly, which is more verbose.

7. **Combined test scenarios are more realistic**: V5's approach of testing both open and in_progress blockers in a single test with 4 tasks better reflects real-world scenarios where multiple blocker states coexist.

8. **Future-proof evolution**: V5's commit-point design, while having duplication, evolved into the cleanest possible alias pattern:
   ```go
   func runReady(ctx *Context) error {
       ctx.Args = append([]string{"--ready"}, ctx.Args...)
       return runList(ctx)
   }
   ```
   V4 also evolved well (via `readyConditionsFor`), but V5's final form is more elegant -- `runReady` is literally 3 lines.

9. **Quiet mode assertion quality**: V5's quiet test validates that output lines contain no spaces (`if strings.Contains(line, " ")`) ensuring only IDs are printed, which is a stronger assertion than V4's exact string match.

### Differences That Are Neutral

1. **SQL JOIN order**: V4 writes `JOIN tasks blocker ON d.blocked_by = blocker.id` vs V5's `JOIN tasks blocker ON blocker.id = d.blocked_by`. These are semantically identical.

2. **Store package naming**: V4 uses `internal/store` while V5 uses `internal/engine`. This is an architectural naming difference that has no bearing on task quality.

3. **Test line counts**: V4 has 490 lines of tests vs V5's 389 lines. V5's tests are more concise due to the `NewTask` constructor and combined scenarios, but the coverage is equivalent.

4. **Fixed vs dynamic column widths**: Both implementations at their commit points use fixed column widths (`%-12s %-12s %-4s`). In evolution, V5's pretty formatter uses dynamic column widths calculated from data, while V4 also evolves to a formatter interface. This is a post-commit difference.

---

## Verdict

**Winner: V5**

V5 wins by a moderate margin. Both implementations produce functionally correct code with equivalent SQL queries and thorough test coverage. The differentiators are:

1. **Architecture**: V5's `Context` + free function + map-based dispatch pattern is strictly superior to V4's `App` struct + method + switch-statement pattern. It is more testable, composable, and idiomatic Go.

2. **Field visibility**: V5 correctly uses unexported fields on package-internal types, while V4 exports fields on an unexported struct -- a minor Go anti-pattern.

3. **Error message style**: V5 follows Go convention more closely with lowercase, non-prefixed error messages.

4. **Test ergonomics**: V5's use of `task.NewTask()` constructor and the `Run()` function signature makes tests more concise and less error-prone.

5. **Evolution trajectory**: V5's commit-point code naturally evolved into the cleanest possible alias pattern (3-line delegation to `runList`), whereas V4's evolution, while solid, required a more complex `readyConditionsFor` helper.

V4's advantage in extracting shared rendering from the start (avoiding duplication at commit time) is real but minor -- it's a temporary difference that both implementations resolve. V4 also has slightly more granular test isolation (separate functions per scenario), which aids debugging but adds verbosity without adding coverage.

The net assessment is that V5 demonstrates better Go idioms, cleaner architecture, and more maintainable test patterns, making it the stronger implementation.
