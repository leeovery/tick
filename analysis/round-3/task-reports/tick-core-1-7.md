# Task tick-core-1-7: tick list & tick show commands

## Task Summary

This task implements the two read commands that complete the walking skeleton: `tick list` (displays all tasks in aligned columns, ordered by priority ASC then created ASC) and `tick show <id>` (displays full details of a single task including blocked_by, children, parent, description). Both commands execute through the storage engine's read flow (shared lock, freshness check, SQLite query). Phase 1 scope means no filters on list -- filtering is Phase 3.

**Acceptance Criteria:**
1. `tick list` displays all tasks in aligned columns (ID, STATUS, PRI, TITLE)
2. `tick list` orders by priority ASC then created ASC
3. `tick list` prints `No tasks found.` when empty
4. `tick list --quiet` outputs only task IDs
5. `tick show <id>` displays full task details
6. `tick show` includes blocked_by with context (ID, title, status)
7. `tick show` includes children with context
8. `tick show` includes parent field when set
9. `tick show` omits empty optional sections
10. `tick show` errors when ID not found
11. `tick show` errors when no ID argument
12. Input IDs normalized to lowercase
13. Both commands use storage engine read flow
14. Exit code 0 on success, 1 on error

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| 1. List displays aligned columns (ID, STATUS, PRI, TITLE) | PASS -- Delegates to `a.Formatter.FormatTaskList()` which renders via `PrettyFormatter` with dynamic column widths; tested in `TestList_AllTasksWithAlignedColumns` (checks for TOON header format with `tasks[`, `id`, `status` fields) | PASS -- Delegates to `ctx.Fmt.FormatTaskList()` via `PrettyFormatter` with dynamic column widths; tested in `TestList/"it lists all tasks with aligned columns"` checking `ID`, `STATUS`, `PRI`, `TITLE` header |
| 2. List orders by priority ASC then created ASC | PASS -- SQL: `ORDER BY t.priority ASC, t.created ASC`; tested in `TestList_OrderByPriorityThenCreated` | PASS -- SQL: `ORDER BY priority ASC, created ASC`; tested in `"it lists tasks ordered by priority then created date"` |
| 3. List prints `No tasks found.` when empty | PARTIAL -- V4 test expects `tasks[0]` (TOON format), not the spec-prescribed `No tasks found.` -- the PrettyFormatter does print `No tasks found.` but test validates TOON output | PASS -- Test explicitly asserts `output != "No tasks found."` would fail, verifies exact string match `"No tasks found."` using `--pretty` flag |
| 4. `--quiet` outputs only task IDs | PASS -- Delegates to formatter's quiet mode; tested in `TestList_QuietFlag` | PASS -- Handles quiet directly in `runList()`: `fmt.Fprintln(ctx.Stdout, r.id)`; tested in `"it prints only task IDs with --quiet flag on list"` |
| 5. Show displays full task details | PASS -- Queries task, builds `TaskDetail` struct, passes to `a.Formatter.FormatTaskDetail()`; tested in `TestShow_FullTaskDetails` | PASS -- Queries task into `showData` struct, passes to `ctx.Fmt.FormatTaskDetail()`; tested in `"it shows full task details by ID"` with `--pretty` flag checking exact format `"ID:       tick-aaaaaa"` |
| 6. Show includes blocked_by with context | PASS -- SQL JOIN fetches blocker ID, title, status; tested in `TestShow_BlockedBySection` | PASS -- Same SQL JOIN approach; tested in `"it shows blocked_by section with ID, title, and status of each blocker"` |
| 7. Show includes children with context | PASS -- Queries `WHERE parent = ?`; tested in `TestShow_ChildrenSection` | PASS -- Same query; tested in `"it shows children section with ID, title, and status of each child"` |
| 8. Show includes parent field when set | PASS -- Queries parent title via separate SELECT; tested in `TestShow_ParentFieldWhenSet` | PASS -- Same approach; tested in `"it shows parent field with ID and title when parent is set"` |
| 9. Show omits empty optional sections | PARTIAL -- V4's TOON formatter always shows `blocked_by[0]` and `children[0]` even when empty; tests `TestShow_OmitsBlockedByWhenEmpty` and `TestShow_OmitsChildrenWhenEmpty` explicitly expect `blocked_by[0]` and `children[0]` (TOON shows count, doesn't omit). Description IS omitted when empty. | PASS -- PrettyFormatter omits `Blocked by:`, `Children:`, `Description:` when empty; tests verify absence of section headers via `strings.Contains` negation |
| 10. Show errors when ID not found | PASS -- `sql.ErrNoRows` check returns `"Task '%s' not found"` error; tested in `TestShow_ErrorNotFound` | PASS -- Uses `found` boolean flag; `"Task '%s' not found"` error; tested in `"it errors when task ID not found"` |
| 11. Show errors when no ID argument | PASS -- `len(args) == 0` check returns `"Task ID is required. Usage: tick show <id>"`; tested in `TestShow_ErrorNoIDArgument` | PASS -- `len(ctx.Args) == 0` check with same message; tested in `"it errors when no ID argument provided to show"` |
| 12. Input IDs normalized to lowercase | PASS -- `task.NormalizeID(strings.TrimSpace(args[0]))`; tested in `TestShow_NormalizesInputID` with `TICK-AAA111` | PASS -- `task.NormalizeID(ctx.Args[0])`; tested in `"it normalizes input ID to lowercase for show lookup"` with `TICK-AAAAAA` |
| 13. Both commands use storage engine read flow | PASS -- Both use `s.Query()` callback pattern; tested in `TestShow_UsesStorageEngineReadFlow` (double invocation) | PASS -- Both use `store.Query()` callback; tested in `"it executes through storage engine read flow"` for both list and show |
| 14. Exit code 0 on success, 1 on error | PASS -- All tests verify exit codes; error paths return non-nil error which cli.go converts to exit 1 | PASS -- All tests verify exit codes; error paths return non-nil error which cli.go converts to exit 1 |

## Implementation Comparison

### Approach

**CLI Architecture -- Method vs Function (continuation from 1-6):**

V4 uses methods on `*App`:
```go
// V4: internal/cli/list.go line 119
func (a *App) runList(args []string) error {
```
```go
// V4: internal/cli/show.go line 13
func (a *App) runShow(args []string) error {
```
Registered via `switch/case` in `cli.go` (adding 12 lines).

V5 uses standalone functions taking `*Context`:
```go
// V5: internal/cli/list.go line 219
func runList(ctx *Context) error {
```
```go
// V5: internal/cli/show.go line 36
func runShow(ctx *Context) error {
```
Registered via the command map (adding 2 lines):
```go
"list":   runList,
"show":   runShow,
```

V5's command map pattern continues to show its benefit -- adding two commands requires 2 lines vs V4's 12 lines of boilerplate switch/case blocks. This is a clear V5 architectural advantage.

**list.go -- Scope Handling:**

This is a significant divergence. The task spec explicitly says "No arguments, no filters in Phase 1 (filters are Phase 3)." Yet both versions implement Phase 3 filters in this commit:

V4 implements `parseListFlags()` (lines 31-78) with `--ready`, `--blocked`, `--status`, `--priority` flags, the `listFlags` struct, the `validStatuses` list, `buildListQuery()` (lines 83-113) with dynamic SQL query building. The core list logic is 161 lines.

V5 implements `parseListFlags()` (lines 36-83) with `--ready`, `--blocked`, `--status`, `--priority`, and additionally `--parent` flag, the `listFilters` struct, `isValidStatus()` helper, `buildListQuery()` (lines 98-106) delegating to `buildReadyFilterQuery()`, `buildBlockedFilterQuery()`, `buildSimpleFilterQuery()`, plus `appendDescendantFilter()`, `queryDescendantIDs()` (recursive CTE for `--parent`), and `parentTaskExists()`. The core list logic is 297 lines.

V5's list.go is almost twice the size because it implements the `--parent` filter with recursive CTE descent -- a feature that goes beyond even Phase 3 spec into a forward-looking parent scoping capability. This is extra scope beyond the task.

**list.go -- Query Building:**

V4's `buildListQuery()` constructs SQL inline:
```go
// V4: internal/cli/list.go lines 83-113
func buildListQuery(f listFlags) (string, []interface{}) {
    if f.ready && !f.hasPri && f.status == "" {
        return readyQuery, nil
    }
    if f.blocked && !f.hasPri && f.status == "" {
        return blockedQuery, nil
    }
    query := "SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE 1=1"
    ...
    if f.ready {
        query += " AND t.status = 'open' AND" + readyConditionsFor("t")
    }
```

V5 uses decomposed builder functions:
```go
// V5: internal/cli/list.go lines 98-106
func buildListQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
    if f.ready {
        return buildReadyFilterQuery(f, descendantIDs)
    }
    if f.blocked {
        return buildBlockedFilterQuery(f, descendantIDs)
    }
    return buildSimpleFilterQuery(f, descendantIDs)
}
```

V5 wraps the ready/blocked queries as subqueries:
```go
// V5: internal/cli/list.go lines 121-137
func buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string) (string, []interface{}) {
    q := `SELECT id, status, priority, title FROM (` + innerQuery + `) AS ` + alias + ` WHERE 1=1`
    ...
}
```

V5's approach is cleaner for composability -- wrapping existing queries as subqueries preserves their ordering and allows layering additional WHERE clauses. V4's approach of appending ready conditions inline is more direct but less composable.

**list.go -- Quiet Mode Handling:**

V4 delegates quiet mode to the formatter:
```go
// V4: internal/cli/list.go line 160
return a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)
```
The `FormatTaskList` signature includes `quiet bool`.

V5 handles quiet mode in `runList()` directly before calling the formatter:
```go
// V5: internal/cli/list.go lines 279-284
if ctx.Quiet {
    for _, r := range rows {
        fmt.Fprintln(ctx.Stdout, r.id)
    }
    return nil
}
```
The `FormatTaskList` signature takes only `(w io.Writer, rows []TaskRow) error`.

V5's approach is cleaner -- quiet mode is a CLI concern, not a formatting concern. The formatter only deals with formatted output. This is a better separation of responsibilities.

**show.go -- Data Types:**

V4 defines local structs inside `runShow()` and then copies them to exported `TaskDetail`/`RelatedTask` structs from `format.go`:
```go
// V4: internal/cli/show.go lines 31-48
type taskDetail struct {
    ID          string
    ...
}
type relatedTask struct {
    ID     string
    Title  string
    Status string
}
```
Then manually copies into `TaskDetail`:
```go
// V4: internal/cli/show.go lines 139-166
td := TaskDetail{
    ID:          detail.ID,
    ...
}
for _, b := range blockedBy {
    td.BlockedBy = append(td.BlockedBy, RelatedTask{...})
}
```

V5 defines package-level structs with unexported fields:
```go
// V5: internal/cli/show.go lines 12-32
type relatedTask struct {
    id     string
    title  string
    status string
}
type showData struct {
    id          string
    ...
    blockedBy   []relatedTask
    children    []relatedTask
}
```
The formatter takes `*showData` directly -- no copying needed:
```go
// V5: internal/cli/show.go line 143
return ctx.Fmt.FormatTaskDetail(ctx.Stdout, &data)
```

V5's approach is more efficient -- a single struct traverses from SQL scan to formatter output. V4's double-struct approach (local types -> exported types) creates unnecessary copying. However, V4's exported types (`TaskDetail`, `RelatedTask`) are more flexible for use by other packages.

**show.go -- Not-Found Error Placement:**

V4 returns the error inside the Query callback:
```go
// V4: internal/cli/show.go lines 61-63
if err == sql.ErrNoRows {
    return fmt.Errorf("Task '%s' not found", id)
}
```

V5 sets a `found` boolean and checks after Query:
```go
// V5: internal/cli/show.go line 65
if err == sql.ErrNoRows {
    return nil
}
...
found = true
```
```go
// V5: internal/cli/show.go lines 134-136
if !found {
    return fmt.Errorf("Task '%s' not found", id)
}
```

V5's approach separates the "database returned no rows" case from the "database error" case more cleanly. V4's approach is more concise. Both correctly handle the not-found case. V5's pattern is slightly more defensive because it ensures the query callback itself always succeeds, and the domain logic check happens in the calling scope.

**show.go -- Quiet Mode Handling:**

V4 handles quiet mode inside `runShow()` before calling the formatter:
```go
// V4: internal/cli/show.go lines 133-136
if a.Quiet {
    fmt.Fprintln(a.Stdout, detail.ID)
    return nil
}
```

V5 does the same:
```go
// V5: internal/cli/show.go lines 138-141
if ctx.Quiet {
    fmt.Fprintln(ctx.Stdout, data.id)
    return nil
}
```

Identical approach for show's quiet mode.

**show.go -- Extra Utility in V5:**

V5 includes a `taskToShowData()` function (lines 150-168) that converts a `task.Task` to `*showData` for formatter output. This utility is not used by `runShow()` itself but enables other commands (like `create` and `update`) to reuse the show formatter for output. This is a forward-looking design decision.

### Code Quality

**Error Handling:**

Both versions handle errors explicitly and return them up the call chain. Neither ignores errors.

V4 wraps database errors with `fmt.Errorf("failed to ...: %w", err)`:
```go
// V4: internal/cli/show.go line 63
return fmt.Errorf("failed to query task: %w", err)
// V4: internal/cli/show.go line 89
return fmt.Errorf("failed to query dependencies: %w", err)
// V4: internal/cli/list.go line 143
return fmt.Errorf("failed to query tasks: %w", err)
```

V5 uses shorter wrapping:
```go
// V5: internal/cli/show.go line 69
return fmt.Errorf("querying task: %w", err)
// V5: internal/cli/show.go line 97
return fmt.Errorf("querying dependencies: %w", err)
// V5: internal/cli/list.go line 270
return fmt.Errorf("querying tasks: %w", err)
```

Both properly use `%w` for error wrapping. V5's messages are more concise, following the Go convention of short lowercase error messages. V4's "failed to" prefix is slightly more verbose but equally correct.

**Field Visibility:**

V4 uses exported fields in `listRow`:
```go
// V4: internal/cli/list.go lines 10-15
type listRow struct {
    ID       string
    Status   string
    Priority int
    Title    string
}
```

V5 uses unexported fields:
```go
// V5: internal/cli/list.go lines 14-19
type listRow struct {
    id       string
    status   string
    priority int
    title    string
}
```

V5's unexported fields are more Go-idiomatic for package-internal types. V4's exported fields are necessary because the `listRow` is passed to the formatter interface via `FormatTaskList(w io.Writer, rows []listRow, quiet bool)` in the same package, but still, since the type itself is unexported, exported fields are unnecessary. V5 solves this by converting `listRow` to an exported `TaskRow` struct before passing to the formatter.

**Naming:**

V4: `listRow`, `listFlags`, `parseListFlags`, `buildListQuery`, `runList`, `runShow`, `taskDetail`, `relatedTask` -- all clear.
V5: `listRow`, `listFilters`, `parseListFlags`, `buildListQuery`, `buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildWrappedFilterQuery`, `buildSimpleFilterQuery`, `appendDescendantFilter`, `queryDescendantIDs`, `parentTaskExists`, `runList`, `runShow`, `showData`, `relatedTask`, `taskToShowData` -- more granular decomposition.

V5 has significantly more functions (16 vs 6 in list.go). This is both a strength (each function does one thing) and a weakness (more indirection to follow).

**Documentation:**

Both versions document all functions with godoc-style comments. V5's comments are more descriptive:
```go
// V5: internal/cli/list.go lines 216-218
// runList implements the "tick list" command. It parses filter flags (--ready,
// --blocked, --status, --priority, --parent), builds the appropriate SQL query,
// and displays results in aligned columns.
```
vs:
```go
// V4: internal/cli/list.go lines 115-118
// runList implements the `tick list` command.
// It queries tasks from SQLite with optional filters (--ready, --blocked,
// --status, --priority), ordered by priority ASC then created ASC,
// and displays them as aligned columns.
```

Both are adequate. V5's is slightly more detailed.

### Test Quality

**V4 List Test Functions (12 top-level functions, each with a single t.Run, 618 lines):**

1. `TestList_AllTasksWithAlignedColumns` -- Creates 2 tasks (done, in_progress), verifies TOON format header contains `tasks[`, `id`, `status`, checks task ID and data presence
2. `TestList_OrderByPriorityThenCreated` -- 4 tasks (priorities 3, 1, 1, 2), verifies order: hi1111 (P1 older) -> hi2222 (P1 newer) -> med (P2) -> low (P3)
3. `TestList_NoTasksFound` -- Empty dir, checks for `tasks[0]` (TOON empty indicator)
4. `TestList_QuietFlag` -- 2 tasks, `--quiet`, verifies exactly 2 lines with just IDs
5. `TestList_StatusFilter` -- Table-driven with 4 statuses (open, in_progress, done, cancelled), verifies inclusion and exclusion
6. `TestList_PriorityFilter` -- 3 tasks (P0, P1, P2), `--priority 1`, verifies only P1 included
7. `TestList_CombineReadyWithPriority` -- 4 tasks, `--ready --priority 2`, verifies only ready P2 task included
8. `TestList_CombineStatusWithPriority` -- 3 tasks, `--status open --priority 1`, verifies only open P1 included
9. `TestList_ErrorReadyAndBlocked` -- `--ready --blocked`, verifies exit 1 and "mutually exclusive" error
10. `TestList_ErrorInvalidStatusPriority` -- Table-driven (invalid status, negative priority, too-high priority, non-numeric priority), verifies errors
11. `TestList_NoMatchesReturnsNoTasksFound` -- Open task filtered by `--priority 4`, verifies `tasks[0]`
12. `TestList_QuietAfterFiltering` -- 3 tasks, `--quiet --status open`, verifies only open task IDs
13. `TestList_AllTasksNoFilters` -- 3 tasks (open, in_progress, done), no filters, all present
14. `TestList_DeterministicOrdering` -- 3 tasks, runs twice, verifies stable order
15. `TestList_BlockedFlag` -- 3 tasks, `--blocked`, verifies only blocked task included
16. `TestList_CombineBlockedWithPriority` -- 4 tasks, `--blocked --priority 2`, verifies only blocked P2 task

**V4 Show Test Functions (14 top-level functions, each with a single t.Run, 510 lines):**

1. `TestShow_FullTaskDetails` -- In_progress task, verifies ID, title, status, timestamps
2. `TestShow_BlockedBySection` -- 2 tasks with dependency, verifies `blocked_by` section presence, blocker ID, title, status
3. `TestShow_ChildrenSection` -- 2 tasks with parent/child, verifies `children[1]` section, child ID, title
4. `TestShow_DescriptionSection` -- Task with description, verifies `description:` section and text
5. `TestShow_OmitsBlockedByWhenEmpty` -- Verifies `blocked_by[0]` present (TOON shows empty count)
6. `TestShow_OmitsChildrenWhenEmpty` -- Verifies `children[0]` present (TOON shows empty count)
7. `TestShow_OmitsDescriptionWhenEmpty` -- Verifies `description:` NOT present
8. `TestShow_ParentFieldWhenSet` -- 2 tasks, verifies `parent` field with ID and title
9. `TestShow_OmitsParentWhenNull` -- Verifies `,parent,` NOT present in schema
10. `TestShow_ClosedTimestampWhenDone` -- Done task with closed time, verifies `closed` and timestamp
11. `TestShow_OmitsClosedWhenOpen` -- Open task, verifies `,closed` NOT present
12. `TestShow_ErrorNotFound` -- Non-existent ID, verifies exit 1, `Error:`, `not found`
13. `TestShow_ErrorNoIDArgument` -- No args to show, verifies exit 1, `Task ID is required`, usage hint
14. `TestShow_NormalizesInputID` -- Uppercase `TICK-AAA111`, verifies found and output normalized
15. `TestShow_QuietFlag` -- `--quiet show`, verifies output is exactly the task ID
16. `TestShow_UsesStorageEngineReadFlow` -- Double invocation, verifies cache reuse works

**V5 List Test Functions (1 top-level function `TestList`, 17 subtests, 583 lines):**

1. `"it lists all tasks with aligned columns"` -- 2 tasks, `--pretty`, checks header starts with `ID`, contains `STATUS`, `PRI`, `TITLE`, checks data presence
2. `"it lists tasks ordered by priority then created date"` -- 3 tasks (P3 early, P1, P3 late), verifies order
3. `"it prints 'No tasks found.' when no tasks exist"` -- Empty, `--pretty`, exact match `"No tasks found."`
4. `"it prints only task IDs with --quiet flag on list"` -- 2 tasks, `--quiet`, verifies 2 lines, each a task ID with no spaces
5. `"it executes through storage engine read flow (shared lock, freshness check)"` -- Single task, verifies output contains ID
6. `"it filters to ready tasks with --ready"` -- 4 tasks (ready, blocker, blocked, done), verifies correct inclusion/exclusion
7. `"it filters to blocked tasks with --blocked"` -- 3 tasks, verifies only blocked task appears
8. `"it filters by --status (all 4 values)"` -- Table-driven with 4 statuses, verifies inclusion/exclusion (reuses same dir)
9. `"it filters by --priority"` -- 3 tasks, `--priority 2`, verifies only P2 task
10. `"it combines --ready with --priority"` -- 4 tasks, `--ready --priority 1`, verifies only ready P1 task
11. `"it combines --status with --priority"` -- 3 tasks, `--status open --priority 1`, verifies only open P1
12. `"it errors when --ready and --blocked both set"` -- Verifies exit 1, error mentions ready and blocked
13. `"it errors for invalid status value"` -- `--status invalid`, verifies error lists valid values
14. `"it errors for invalid priority value"` -- `--priority 9`, verifies error mentions 0-4
15. `"it errors for non-numeric priority value"` -- `--priority abc`, verifies exit 1
16. `"it returns 'No tasks found.' when no matches"` -- Done task, `--pretty --status open`, exact match
17. `"it outputs IDs only with --quiet after filtering"` -- 2 tasks, `--quiet --priority 1`, verifies single ID
18. `"it returns all tasks with no filters"` -- 3 tasks, all present
19. `"it maintains deterministic ordering"` -- 3 tasks, `--pretty --status open`, verifies order
20. `"it handles contradictory filters with empty result not error"` -- `--status done --ready`, verifies exit 0 and `No tasks found.`

**V5 Show Test Functions (1 top-level function `TestShow`, 16 subtests, 340 lines):**

1. `"it shows full task details by ID"` -- `--pretty show`, checks exact format `"ID:       tick-aaaaaa"`, `"Title:    Login endpoint"`, `"Status:   in_progress"`, `"Priority: 1"`, `"Created:"`, `"Updated:"`
2. `"it shows blocked_by section with ID, title, and status of each blocker"` -- `--pretty`, checks `"Blocked by:"`, blocker ID, title, `"(done)"`
3. `"it shows children section with ID, title, and status of each child"` -- `--pretty`, checks `"Children:"`, child ID, title, `"(open)"`
4. `"it shows description section when description is present"` -- `--pretty`, checks `"Description:"` and text
5. `"it omits blocked_by section when task has no dependencies"` -- `--pretty`, verifies `"Blocked by:"` NOT present
6. `"it omits children section when task has no children"` -- `--pretty`, verifies `"Children:"` NOT present
7. `"it omits description section when description is empty"` -- `--pretty`, verifies `"Description:"` NOT present
8. `"it shows parent field with ID and title when parent is set"` -- `--pretty`, checks `"Parent:"`, parent ID, parent title
9. `"it omits parent field when parent is null"` -- `--pretty`, verifies `"Parent:"` NOT present
10. `"it shows closed timestamp when task is done or cancelled"` -- `--pretty`, checks `"Closed:"` and `"2026-01-19T14:30:00Z"`
11. `"it omits closed field when task is open or in_progress"` -- `--pretty`, verifies `"Closed:"` NOT present
12. `"it errors when task ID not found"` -- Verifies exit 1, exact message `"Task 'tick-nonexist' not found"`
13. `"it errors when no ID argument provided to show"` -- Verifies exit 1, `"Task ID is required"`
14. `"it normalizes input ID to lowercase for show lookup"` -- `TICK-AAAAAA`, verifies found
15. `"it outputs only task ID with --quiet flag on show"` -- `--quiet`, exact match
16. `"it executes through storage engine read flow (shared lock, freshness check)"` -- Single invocation, verifies output

**Test Gap Analysis:**

| Test Area | V4 | V5 |
|-----------|-----|-----|
| Omitting blocked_by/children when empty (spec compliance) | FAIL -- Tests expect `blocked_by[0]` and `children[0]` to be present (TOON format shows empty sections with count) instead of omitting them per spec | PASS -- Tests verify `"Blocked by:"` and `"Children:"` are NOT present when empty |
| Exact output format validation | Weak -- Tests check for presence of tokens in TOON format (`tasks[`, `id`), not exact aligned columns | Strong -- Tests check exact formatted output `"ID:       tick-aaaaaa"` matching spec format |
| No tasks found exact message | Weak -- Checks for `tasks[0]` not `No tasks found.` | Strong -- Exact string match against spec |
| Contradictory filter handling | Not tested | Tested -- `--status done --ready` returns empty result not error |
| Double invocation for freshness | Tested in show (2 calls) | Tested in list (single call after JSONL write) |
| Combined blocked + priority filter | Tested in `TestList_CombineBlockedWithPriority` | Not tested (V5 tests ready+priority but not blocked+priority) |
| Non-numeric priority error | Tested in `TestList_ErrorInvalidStatusPriority` table | Tested as separate subtest |
| `--pretty` flag format compliance | Not tested -- defaults to TOON | Extensively tested with `--pretty` flag |

**V4 Test Format Issue:**

V4's tests validate TOON format output (the default non-TTY format) rather than the spec-prescribed key-value format. For example:
```go
// V4: show_test.go line 178
if !strings.Contains(output, "blocked_by[0]") {
    t.Errorf("expected output to contain 'blocked_by[0]' when task has no deps, got %q", output)
}
```
The spec says "Omit sections with no data" but V4's TOON formatter always includes `blocked_by[0]` and `children[0]`. V4's tests pass because they test the TOON format, which by design shows empty arrays with count 0. This is a spec compliance issue -- the tests validate the wrong format.

V5's tests use `--pretty` flag to force the pretty formatter, which matches the spec's key-value output format:
```go
// V5: show_test.go line 64
if !strings.Contains(output, "Blocked by:") {
    t.Errorf("expected 'Blocked by:' section, got %q", output)
}
```

**Task Construction:**

V4 constructs tasks with struct literals:
```go
// V4: list_test.go line 15
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
}
```

V5 uses the `task.NewTask()` factory:
```go
// V5: list_test.go line 14
t1 := task.NewTask("tick-aaaaaa", "Setup Sanctum")
t1.Status = task.StatusDone
t1.Priority = 1
```

V5's factory approach is more resilient to changes in `task.Task` struct defaults -- if a new required field is added, `NewTask()` handles it. V4's struct literals would break. V5 is genuinely better here for maintainability.

**Test Infrastructure:**

V4 creates App instances and calls `app.Run()`:
```go
// V4: list_test.go line 22
app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
code := app.Run([]string{"tick", "list"})
```

V5 calls the package-level `Run()` function:
```go
// V5: list_test.go line 23
code := Run([]string{"tick", "--pretty", "list"}, dir, &stdout, &stderr, false)
```

V5's approach more closely mirrors actual CLI invocation. The `isTTY: false` parameter is explicit.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Error wrapping with `fmt.Errorf("%w", err)` | PASS -- All SQL errors wrapped: `"failed to query tasks: %w"`, `"failed to query dependencies: %w"`, etc. | PASS -- All SQL errors wrapped: `"querying task: %w"`, `"querying dependencies: %w"`, etc. |
| Table-driven tests with subtests | PARTIAL -- Uses separate top-level functions with single `t.Run()` each. One table-driven test (`TestList_ErrorInvalidStatusPriority` with 4 cases, `TestList_StatusFilter` with 4 statuses) | PARTIAL -- All subtests under single `TestList`/`TestShow` parent. One table-driven test (`"it filters by --status"` with 4 statuses, `"it errors for invalid..."` separately) |
| Explicit error handling (no ignored errors) | PASS -- All errors checked and returned | PASS -- All errors checked and returned |
| Exported function documentation | PASS -- All functions documented with godoc comments | PASS -- All functions documented with godoc comments |
| Context.Context for blocking operations | N/A -- Read commands don't have long-running blocking operations; store handles locking internally | N/A -- Same |
| No panic for error handling | PASS -- No panics | PASS -- No panics |
| No goroutines without lifecycle management | PASS -- No goroutines | PASS -- No goroutines |
| Run race detector on tests | Not verifiable from code alone -- requires `go test -race` | Not verifiable from code alone |

### Spec-vs-Convention Conflicts

**Section omission in TOON format:**

The spec says "Omit sections with no data (blocked_by, children, description, parent, closed)." V4's TOON formatter renders `blocked_by[0]` and `children[0]` for empty arrays -- a design choice where TOON format explicitly shows the count of items, including zero. This conflicts with the spec's omission requirement. V4's tests are written to match TOON behavior rather than spec behavior.

V5's tests use `--pretty` and verify actual omission of empty sections, directly matching the spec. This is not truly a spec-vs-convention conflict but rather V4 testing the wrong format for spec compliance verification. Both versions' PrettyFormatters correctly omit empty sections.

**Assessment:** V5 made the better choice by testing with `--pretty` to verify spec compliance. V4's TOON format behavior is a reasonable design for a machine-readable format (always showing the schema is useful for parsers), but the tests should have verified spec compliance separately.

**Error message format:**

The spec shows `Error: Task 'tick-xyz' not found` with a capital T. Both versions produce this exactly:
```go
// Both versions:
return fmt.Errorf("Task '%s' not found", id)
```

Go convention says error strings should not be capitalized, but since this is wrapped by the CLI's `writeError` / `fmt.Fprintf(stderr, "Error: %s\n", err)`, the capitalized error text becomes the message after `Error: `. Both versions make the same pragmatic choice to capitalize for user-facing output, which is a reasonable judgment call.

**"No tasks found." output:**

The spec says list should print `No tasks found.` when empty. V4's PrettyFormatter does this correctly, but V4's tests check the TOON format (`tasks[0]`). V5's tests check the spec-exact string. Both implementations are correct; V4's test coverage for this specific criterion is weaker.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 7 (2 docs, 5 code incl. cli.go) | 7 (2 docs, 5 code incl. cli.go) |
| Lines added (total) | 978 | 789 |
| Impl LOC (list.go) | 161 | 297 |
| Impl LOC (show.go) | 169 | 168 |
| Test LOC (list_test.go) | 618 (added in commit) | 583 (added in commit) |
| Test LOC (show_test.go) | 510 (added in commit) | 340 (added in commit) |
| Test functions (list) | 16 top-level functions (16 subtests) | 1 top-level function (20 subtests) |
| Test functions (show) | 16 top-level functions (16 subtests) | 1 top-level function (16 subtests) |
| cli.go lines changed | +12 | +2 |

V5's list.go is 136 lines longer due to the `--parent` filter with recursive CTE and decomposed query builders. V5's test files are 205 lines shorter total, primarily from the show_test.go difference (510 vs 340 lines). V5's more compact test structure (single top-level function, factory-based task creation) contributes to shorter tests.

## Verdict

**V5 wins on spec compliance, test accuracy, and architectural cleanliness.**

**V5 advantages:**

1. **Spec compliance in tests** -- V5 tests use `--pretty` flag to verify the spec-prescribed output format (aligned columns with header `ID STATUS PRI TITLE`, key-value detail format with `ID:       tick-aaaaaa`). V4 tests validate TOON format output which does not match the spec's described format. This is the most significant difference -- V5's tests actually verify the acceptance criteria as written.

2. **Section omission compliance** -- V5 correctly tests that empty sections (blocked_by, children, description, parent, closed) are omitted. V4 tests expect `blocked_by[0]` and `children[0]` to be present even when empty, which contradicts acceptance criterion #9 ("omits empty optional sections").

3. **Quiet mode separation** -- V5 handles quiet mode in the command function before calling the formatter, keeping formatting logic pure. V4 passes `quiet` through to the formatter interface, coupling a CLI concern with output formatting.

4. **Command map efficiency** -- V5 adds 2 lines to cli.go vs V4's 12 lines. This advantage compounds across tasks.

5. **Contradictory filter test** -- V5 tests `--status done --ready` (contradictory combination) returns empty result not error. V4 has no equivalent edge case.

6. **Factory-based test construction** -- V5 uses `task.NewTask()` which is more resilient to struct changes.

**V4 advantages:**

1. **More list filter combination tests** -- V4 tests `--blocked --priority` combination explicitly; V5 does not.

2. **Double-invocation storage test** -- V4's `TestShow_UsesStorageEngineReadFlow` makes two consecutive calls to verify cache freshness works. V5 makes a single call.

3. **Exported data types** -- V4's `TaskDetail` and `RelatedTask` exported types allow other packages to use the show data structures. V5's unexported `showData` is package-internal only.

**Neutral differences:**

- Both implement Phase 3 filters in a Phase 1 task (beyond task scope). V5 goes further with `--parent` and recursive CTE.
- V4 has more total test lines (1128 vs 923) but V5's tests are more accurate to the spec.
- Both properly wrap errors, document functions, and handle edge cases.

The deciding factor is that V5's tests verify what the spec actually requires, while V4's tests verify TOON format output that diverges from spec requirements (particularly section omission). Test accuracy -- testing the right thing -- outweighs test volume.
