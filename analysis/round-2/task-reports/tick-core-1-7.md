# Task tick-core-1-7: tick list & tick show Commands

## Task Summary

This task implements two read commands: `tick list` (displays all tasks in an aligned column table) and `tick show <id>` (displays full details for a single task). Both commands execute through the storage engine's read flow (shared lock, freshness check, SQLite query).

**Key requirements:**
- `tick list`: no arguments/filters; query all tasks ordered by priority ASC then created ASC; display as aligned columns (ID 12, STATUS 12, PRI 4, TITLE remainder); empty result prints "No tasks found."; `--quiet` outputs only IDs.
- `tick show <id>`: requires positional `<id>` arg; normalize ID to lowercase; query task + blocked_by (ID, title, status) + children (ID, title, status); key-value output format; omit empty optional sections (blocked_by, children, description, parent, closed); `--quiet` outputs only the task ID.
- Error handling: missing ID arg shows usage hint; not-found shows error with normalized ID; errors to stderr with exit code 1.

**Acceptance Criteria (from plan):**
1. `tick list` displays all tasks in aligned columns (ID, STATUS, PRI, TITLE)
2. `tick list` orders by priority ASC then created ASC
3. `tick list` prints "No tasks found." when empty
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

| Criterion | V2 | V4 |
|-----------|-----|-----|
| `tick list` aligned columns (ID, STATUS, PRI, TITLE) | PASS -- uses `"%-12s%-12s%-4s%s\n"` format | PASS -- uses `"%-12s %-12s %-4s %s\n"` format with extra space separators |
| `tick list` orders by priority ASC then created ASC | PASS -- SQL: `ORDER BY priority ASC, created ASC` | PASS -- identical SQL |
| `tick list` prints "No tasks found." when empty | PASS -- `fmt.Fprintln(a.stdout, "No tasks found.")` | PASS -- `fmt.Fprintln(a.Stdout, "No tasks found.")` |
| `tick list --quiet` outputs only task IDs | PASS -- checks `a.config.Quiet`, prints only IDs | PASS -- checks `a.Quiet`, prints only IDs |
| `tick show <id>` displays full task details | PASS -- key-value format with all core fields | PASS -- identical key-value format |
| `tick show` includes blocked_by with context | PASS -- JOIN query on dependencies + tasks table | PASS -- identical JOIN query |
| `tick show` includes children with context | PASS -- queries tasks WHERE parent = id | PASS -- identical query |
| `tick show` includes parent field when set | PASS -- shows parent ID + title; falls back to ID-only if parent not found | PASS -- shows parent ID + title via `parentInfo *relatedTask` |
| `tick show` omits empty optional sections | PASS -- conditional output for blocked_by, children, description, parent, closed | PASS -- identical conditional logic |
| `tick show` errors when ID not found | PASS -- returns `fmt.Errorf("Task '%s' not found", lookupID)` | PASS -- returns `fmt.Errorf("Task '%s' not found", id)` |
| `tick show` errors when no ID argument | PASS -- returns error with usage hint | PASS -- identical error message |
| Input IDs normalized to lowercase | PASS -- `task.NormalizeID(args[0])` | PASS -- `task.NormalizeID(strings.TrimSpace(args[0]))` |
| Both commands use storage engine read flow | PASS -- calls `store.Query()` | PASS -- calls `s.Query()` |
| Exit code 0 on success, 1 on error | PASS -- errors returned from `Run()` handled by caller | PASS -- explicit `return 1` on error in `Run()` |

## Implementation Comparison

### Approach

Both versions follow a nearly identical high-level approach: register commands in the CLI dispatcher, implement `runList` and `runShow` as methods on `App`, use the storage engine's `Query()` callback pattern to execute SQL within a shared-lock context, and render output via `fmt.Fprintf`/`fmt.Fprintln`.

**CLI Dispatcher Integration**

V2 registers commands in `app.go` with direct return of errors:
```go
case "list":
    return a.runList()
case "show":
    return a.runShow(cmdArgs)
```

V4 registers in `cli.go` with explicit exit code handling:
```go
case "list":
    if err := a.runList(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
case "show":
    if err := a.runShow(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

This is an architectural difference inherited from earlier tasks -- V2's `App.Run()` returns `error`, while V4's returns `int` (exit code). V4's approach is more explicit about error-to-stderr routing but more verbose. V2 delegates error formatting to the caller.

**runList Signature**

V2: `func (a *App) runList() error` -- takes no args since list needs none.
V4: `func (a *App) runList(args []string) error` -- accepts args even though unused, for consistency with other commands and future extensibility.

V4's approach is slightly more forward-looking but technically introduces an unused parameter.

**Column Formatting**

V2 uses packed column format with no separator between columns:
```go
fmt.Fprintf(a.stdout, "%-12s%-12s%-4s%s\n", "ID", "STATUS", "PRI", "TITLE")
```

V4 adds explicit space separators between columns:
```go
fmt.Fprintf(a.Stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
```

The spec says "Column widths: ID (12), STATUS (12), PRI (4), TITLE (remainder)". V2 follows this precisely (the width includes any visual padding). V4 adds extra spaces on top of the 12-char width, making columns 13+ chars wide in practice. V2 is arguably more faithful to the spec. Both produce visually readable output.

**Show Data Structures**

V2 defines package-level types:
```go
type showData struct {
    ID          string
    Title       string
    Status      string
    Priority    int
    Description string
    Parent      string
    ParentTitle string
    Created     string
    Updated     string
    Closed      string
    BlockedBy   []relatedTask
    Children    []relatedTask
}

type relatedTask struct {
    ID     string
    Title  string
    Status string
}
```

V4 defines types locally inside `runShow`:
```go
type taskDetail struct {
    ID          string
    // ...
}

type relatedTask struct {
    ID     string
    Title  string
    Status string
}

var detail taskDetail
var blockedBy []relatedTask
var children []relatedTask
var parentInfo *relatedTask
```

V2 bundles everything (including `BlockedBy` and `Children` slices) into a single `showData` struct and has a separate `printShowOutput(d *showData)` method. V4 keeps `blockedBy`, `children`, and `parentInfo` as separate variables alongside `detail`.

V2's approach is cleaner for extraction and testability since all data flows through one struct. V4's local types reduce visibility scope, which is valid since these types are only used in this one function.

**Parent Handling**

V2 stores `ParentTitle` in the `showData` struct and queries only the parent title:
```go
err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", d.Parent).Scan(&parentTitle)
if err == nil {
    d.ParentTitle = parentTitle
}
```
V2 also handles the orphaned-parent edge case gracefully by showing ID-only if title lookup fails:
```go
if d.ParentTitle != "" {
    fmt.Fprintf(a.stdout, "Parent:   %s  %s\n", d.Parent, d.ParentTitle)
} else {
    fmt.Fprintf(a.stdout, "Parent:   %s\n", d.Parent)
}
```

V4 queries full parent info (id, title, status):
```go
var p relatedTask
err := db.QueryRow(
    "SELECT id, title, status FROM tasks WHERE id = ?",
    detail.Parent,
).Scan(&p.ID, &p.Title, &p.Status)
if err == nil {
    parentInfo = &p
}
```
V4 silently skips the parent if the lookup fails. If the parent task was deleted but still referenced, V4 shows nothing; V2 still shows the parent ID.

V2's approach is genuinely better here -- it degrades gracefully for orphaned parents instead of silently hiding the relationship.

**Description Rendering**

V2 renders description as a single indented line:
```go
fmt.Fprintf(a.stdout, "  %s\n", d.Description)
```

V4 splits description by newlines and indents each line:
```go
for _, line := range strings.Split(detail.Description, "\n") {
    fmt.Fprintf(a.Stdout, "  %s\n", line)
}
```

V4's approach is genuinely better for multi-line descriptions, preserving formatting. V2 would render multi-line descriptions incorrectly (only the first line indented).

**ID Normalization**

V2: `lookupID := task.NormalizeID(args[0])`
V4: `id := task.NormalizeID(strings.TrimSpace(args[0]))`

V4 adds `strings.TrimSpace()` before normalization, which is a minor defensive improvement against whitespace in arguments.

### Code Quality

**Naming Conventions**

V2 uses `a.stdout` (unexported field) and `a.config.Quiet` (config struct pattern), `a.workDir`:
```go
fmt.Fprintln(a.stdout, "No tasks found.")
if a.config.Quiet { ... }
```

V4 uses `a.Stdout` (exported field) and `a.Quiet` (flat fields), `a.Dir`:
```go
fmt.Fprintln(a.Stdout, "No tasks found.")
if a.Quiet { ... }
```

V2's unexported fields with a config struct is more idiomatic Go for internal packages -- it prevents external callers from reaching in. V4's exported fields are simpler but expose internals.

**Package Naming**

V2 uses `storage` package (`github.com/leeovery/tick/internal/storage`).
V4 uses `store` package (`github.com/leeovery/tick/internal/store`).

This is a project-wide naming difference from earlier tasks, not specific to this commit.

**Error Handling**

Both versions handle errors identically -- wrapping with context strings like `"failed to query tasks: %w"`, `"failed to scan task row: %w"`, etc. Both properly check `sql.ErrNoRows` for not-found. Both defer `Close()` on row iterators and check `.Err()` after iteration. Error handling quality is equivalent.

**DRY Principle**

V2 extracts rendering into a dedicated `printShowOutput` method:
```go
func (a *App) printShowOutput(d *showData) {
    fmt.Fprintf(a.stdout, "ID:       %s\n", d.ID)
    // ...
}
```

V4 renders inline within `runShow`. For this task's scope, both approaches are equivalent. V2's extraction makes future testing of rendering logic independently possible.

**Type Safety**

Both versions use `sql.NullString` correctly for nullable columns (description, parent, closed). Both convert to plain strings after validity check. Equivalent quality.

### Test Quality

**V2 Test Functions (list_test.go: 5 subtests)**

All subtests are nested under a single `TestListCommand`:
1. `"it lists all tasks with aligned columns"` -- verifies header contains ID/STATUS/PRI/TITLE, data rows contain expected values, checks column alignment by comparing index positions
2. `"it lists tasks ordered by priority then created date"` -- 3 tasks with different priorities, verifies ordering
3. `"it prints 'No tasks found.' when no tasks exist"` -- empty initialized dir
4. `"it prints only task IDs with --quiet flag on list"` -- verifies only IDs output, correct order
5. `"it executes through storage engine read flow (shared lock, freshness check)"` -- verifies cache.db exists after list operation

**V2 Test Functions (show_test.go: 15 subtests)**

All subtests under single `TestShowCommand`:
1. `"it shows full task details by ID"` -- checks all core fields present
2. `"it shows blocked_by section with ID, title, and status of each blocker"` -- verifies section presence and content
3. `"it shows children section with ID, title, and status of each child"`
4. `"it shows description section when description is present"`
5. `"it omits blocked_by section when task has no dependencies"` -- negative check
6. `"it omits children section when task has no children"` -- negative check
7. `"it omits description section when description is empty"` -- negative check
8. `"it shows parent field with ID and title when parent is set"`
9. `"it omits parent field when parent is null"` -- negative check
10. `"it shows closed timestamp when task is done or cancelled"`
11. `"it omits closed field when task is open or in_progress"` -- negative check
12. `"it errors when task ID not found"` -- checks error message content
13. `"it errors when no ID argument provided to show"` -- checks error + usage hint
14. `"it normalizes input ID to lowercase for show lookup"` -- passes uppercase ID
15. `"it outputs only task ID with --quiet flag on show"`

**V4 Test Functions (list_test.go: 4 top-level functions, 4 subtests)**

1. `TestList_AllTasksWithAlignedColumns` > `"it lists all tasks with aligned columns"` -- uses typed `task.Task` structs, checks header and row content
2. `TestList_OrderByPriorityThenCreated` > `"it lists tasks ordered by priority then created date"` -- 4 tasks (3 priority levels), more thorough than V2's 3 tasks
3. `TestList_NoTasksFound` > `"it prints 'No tasks found.' when no tasks exist"`
4. `TestList_QuietFlag` > `"it prints only task IDs with --quiet flag on list"`

**V4 Test Functions (show_test.go: 15 top-level functions, 15 subtests)**

1. `TestShow_FullTaskDetails` > `"it shows full task details by ID"` -- more granular assertions (checks each label and value separately)
2. `TestShow_BlockedBySection` > `"it shows blocked_by section with ID, title, and status of each blocker"`
3. `TestShow_ChildrenSection` > `"it shows children section with ID, title, and status of each child"`
4. `TestShow_DescriptionSection` > `"it shows description section when description is present"`
5. `TestShow_OmitsBlockedByWhenEmpty` > `"it omits blocked_by section when task has no dependencies"`
6. `TestShow_OmitsChildrenWhenEmpty` > `"it omits children section when task has no children"`
7. `TestShow_OmitsDescriptionWhenEmpty` > `"it omits description section when description is empty"`
8. `TestShow_ParentFieldWhenSet` > `"it shows parent field with ID and title when parent is set"`
9. `TestShow_OmitsParentWhenNull` > `"it omits parent field when parent is null"`
10. `TestShow_ClosedTimestampWhenDone` > `"it shows closed timestamp when task is done or cancelled"`
11. `TestShow_OmitsClosedWhenOpen` > `"it omits closed field when task is open or in_progress"`
12. `TestShow_ErrorNotFound` > `"it errors when task ID not found"`
13. `TestShow_ErrorNoIDArgument` > `"it errors when no ID argument provided to show"`
14. `TestShow_NormalizesInputID` > `"it normalizes input ID to lowercase for show lookup"`
15. `TestShow_QuietFlag` > `"it outputs only task ID with --quiet flag on show"`
16. `TestShow_UsesStorageEngineReadFlow` > `"it executes through storage engine read flow (shared lock, freshness check)"`

**Test Coverage Gaps**

V2 list tests include a storage engine read flow test; V4 list tests do NOT have this test. V4 has this test for show but not for list. The task plan specifies `"it executes through storage engine read flow (shared lock, freshness check)"` as a single test (not per-command), so V2 covers it in list, V4 covers it in show. Both are partial; neither explicitly tests the read flow for BOTH commands.

**Test Data Setup**

V2 uses raw JSONL strings for test data:
```go
content := `{"id":"tick-aaa111","title":"Setup Sanctum",...}`
dir := setupTickDirWithContent(t, content)
```

V4 uses typed `task.Task` structs:
```go
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
}
dir := setupInitializedDirWithTasks(t, tasks)
```

V4's approach is genuinely better -- it provides type safety, uses domain constants (e.g., `task.StatusDone` vs raw `"done"` string), and is less error-prone. V2's raw JSONL is fragile (typos in field names won't be caught at compile time).

**Test Organization**

V2 nests all subtests under a single top-level `TestListCommand` / `TestShowCommand`. V4 uses separate top-level test functions for each scenario (e.g., `TestShow_BlockedBySection`, `TestShow_OmitsParentWhenNull`). V4's approach allows running individual test cases more easily with `go test -run TestShow_BlockedBySection` and provides better test output. This is a meaningful organizational improvement.

**Error Verification in Tests**

V2 checks errors by inspecting `err` return value:
```go
err := app.Run([]string{"tick", "show", "tick-xyz123"})
if err == nil { t.Fatal("expected error...") }
if !strings.Contains(err.Error(), "tick-xyz123") { ... }
```

V4 checks exit code and stderr output:
```go
code := app.Run([]string{"tick", "show", "tick-xyz123"})
if code != 1 { t.Errorf("expected exit code 1, got %d", code) }
errMsg := stderr.String()
if !strings.Contains(errMsg, "Error:") { ... }
if !strings.Contains(errMsg, "tick-xyz123") { ... }
```

V4's approach is more thorough -- it verifies the exit code AND the error output format (including the `Error:` prefix). V2 only verifies the error value itself, not how it's presented to the user.

**Assertion Quality**

V2's column alignment test is more thorough:
```go
headerStatusPos := strings.Index(header, "STATUS")
row1StatusPos := strings.Index(lines[1], "done")
row2StatusPos := strings.Index(lines[2], "in_progress")
if headerStatusPos != row1StatusPos || headerStatusPos != row2StatusPos { ... }
```

V4 does not verify column alignment positions, only that the header contains the expected column names. V2 is genuinely better here -- it actually validates the alignment specification rather than just checking content presence.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 7 (app.go, list.go, list_test.go, show.go, show_test.go, 2 docs) | 7 (cli.go, list.go, list_test.go, show.go, show_test.go, 2 docs) |
| Lines added | 830 | 978 |
| Impl LOC (list.go) | 74 | 77 |
| Impl LOC (show.go) | 189 | 189 |
| Test LOC (list_test.go) | 187 | 175 |
| Test LOC (show_test.go) | 368 | 520 |
| Test functions (list) | 1 top-level, 5 subtests | 4 top-level, 4 subtests |
| Test functions (show) | 1 top-level, 15 subtests | 15 top-level, 15 subtests + 1 extra (storage flow) |

## Verdict

**V4 is the better implementation**, though the margin is modest and V2 has some individual strengths.

V4's advantages:
- **Type-safe test data**: Using `task.Task` structs with domain constants (`task.StatusDone`) instead of raw JSONL strings eliminates an entire class of test data bugs. This is the single most impactful difference.
- **Multi-line description handling**: V4 splits description text by newlines and indents each line, while V2 dumps the whole description as one line. For real-world use, V4's approach is functionally superior.
- **Test organization**: Separate top-level test functions allow finer-grained test execution and clearer output.
- **Error verification**: V4 tests verify both exit codes and stderr content (including the `Error:` prefix format), providing more end-to-end coverage.
- **Defensive TrimSpace**: V4 adds `strings.TrimSpace()` before ID normalization, a small but practical guard.
- **Storage engine flow test for show**: V4 includes a `TestShow_UsesStorageEngineReadFlow` test with a second call to verify cache reuse.

V2's advantages:
- **Column alignment verification**: V2's alignment test actually checks column positions line up, not just that content exists. V4 misses this.
- **Orphaned parent handling**: V2 gracefully degrades to showing just the parent ID if the parent task is missing; V4 silently hides the relationship.
- **Storage engine flow test for list**: V2 has this test for list; V4 has it only for show.
- **Extracted render method**: V2's `printShowOutput(d *showData)` method cleanly separates data gathering from rendering.
- **Spec-faithful column widths**: V2's format string matches the spec's "12, 12, 4" widths exactly without extra separators.

On balance, V4's type-safe test infrastructure, multi-line description handling, and more thorough error testing outweigh V2's advantages in alignment testing and orphaned-parent handling. Both implementations are solid and fully meet the acceptance criteria.
