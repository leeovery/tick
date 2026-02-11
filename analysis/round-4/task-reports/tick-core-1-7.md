# tick-core-1-7: tick list & tick show commands

## Task Summary

Implement `tick list` and `tick show <id>` read commands. `tick list` queries all tasks from SQLite ordered by priority ASC then created ASC and outputs aligned columns. `tick show <id>` displays full task detail including blocked_by, children, parent, description, and closed fields, omitting empty optional sections. Both use the storage engine read flow (shared lock, freshness check).

## Acceptance Criteria Compliance

| # | Criterion | V5 | V6 | Notes |
|---|-----------|----|----|-------|
| 1 | `tick list` displays all tasks in aligned columns (ID, STATUS, PRI, TITLE) | PASS | PASS | Both print header + data rows with fixed-width columns |
| 2 | `tick list` orders by priority ASC then created ASC | PASS | PASS | Both use `ORDER BY priority ASC, created ASC` |
| 3 | `tick list` prints `No tasks found.` when empty | PASS | PASS | Both check `len(rows) == 0` and emit message |
| 4 | `tick list --quiet` outputs only task IDs | PASS | PASS | Both iterate rows printing IDs only |
| 5 | `tick show <id>` displays full task details | PASS | PASS | Both render key-value output |
| 6 | `tick show` includes blocked_by with context (ID, title, status) | PASS | PASS | Both JOIN dependencies + tasks |
| 7 | `tick show` includes children with context | PASS | PASS | Both query `WHERE parent = ?` |
| 8 | `tick show` includes parent field when set | PASS | PASS | Both look up parent title |
| 9 | `tick show` omits empty optional sections | PASS | PASS | Conditional rendering for all optional sections |
| 10 | `tick show` errors when ID not found | PASS | PASS | V5: post-query `found` flag; V6: inline `sql.ErrNoRows` returns error |
| 11 | `tick show` errors when no ID argument | PASS | PASS | Both check `len(args) == 0` |
| 12 | Input IDs normalized to lowercase | PASS | PASS | Both call `task.NormalizeID()` |
| 13 | Both commands use storage engine read flow | PASS | PASS | Both call `store.Query()` |
| 14 | Exit code 0 on success, 1 on error | PASS | PASS | Error returns propagate to CLI dispatcher |

## Implementation Comparison

### Approach

#### CLI Architecture & Command Registration

**V5** uses a command map in `cli.go` mapping names to handler functions receiving a `*Context` struct:

```go
// V5: internal/cli/cli.go lines 97-112
var commands = map[string]func(*Context) error{
    "init":    runInit,
    "create":  runCreate,
    "list":    runList,
    "show":    runShow,
    ...
}
```

Handlers are unexported (`runList`, `runShow`) and receive a `*Context` with `WorkDir`, `Stdout`, `Stderr`, `Quiet`, `Verbose`, `Format`, `Fmt`, `Args`.

**V6** uses an `App` struct with a switch/case dispatcher in `app.go`:

```go
// V6: internal/cli/app.go lines 57-83
switch subcmd {
case "init":
    err = a.handleInit(fc, fmtr, subArgs)
case "create":
    err = a.handleCreate(fc, fmtr, subArgs)
case "list":
    err = a.handleList(fc, fmtr, subArgs)
case "show":
    err = a.handleShow(fc, fmtr, subArgs)
```

Handlers are methods on `*App` that delegate to exported functions (`RunList`, `RunShow`) with explicit parameter passing.

**Assessment**: V5's map-based dispatch is more extensible and idiomatic Go. V6's switch/case approach is straightforward but requires modifying the switch for each new command.

#### list.go -- Core Logic

**V5** (at commit time, `533ee60:internal/cli/list.go`, 80 lines):

```go
// V5: 533ee60:internal/cli/list.go line 23-79
func runList(ctx *Context) error {
    tickDir, err := DiscoverTickDir(ctx.WorkDir)
    ...
    store, err := engine.NewStore(tickDir)
    ...
    err = store.Query(func(db *sql.DB) error {
        sqlRows, err := db.Query(
            `SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`,
        )
        ...
    })
    ...
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
}
```

Output via a dedicated `printListTable` function using `%-12s %-12s %-4s %s\n` format spec. Column widths match spec exactly: ID (12), STATUS (12), PRI (4), TITLE (remainder).

**V6** (at commit time, `72b9e9e:internal/cli/list.go`, 77 lines):

```go
// V6: 72b9e9e:internal/cli/list.go line 13-77
func RunList(dir string, quiet bool, stdout io.Writer) error {
    ...
    err = store.Query(func(db *sql.DB) error {
        sqlRows, err := db.Query(
            `SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`,
        )
        ...
    })
    ...
    // Print header
    fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")
    // Print rows
    for _, r := range rows {
        fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
    }
```

Uses `%-12s%-13s%-5s%s\n` -- note STATUS column is 13 chars wide (not 12) and PRI is 5 chars wide (not 4). No space separator between columns (uses padding only). This **deviates from spec** which says "Column widths: ID (12), STATUS (12), PRI (4), TITLE (remainder)".

**Assessment**: V5 matches the spec's column widths exactly. V6 uses slightly different widths (STATUS=13, PRI=5), producing a header `"ID          STATUS       PRI  TITLE"` that looks similar but has different padding structure. The V6 test hardcodes the wider format, so tests pass, but it does not match the spec's stated `(12), (12), (4)`.

#### list.go -- `listRow` type placement

**V5** defines `listRow` as a package-level type (line 14-19).
**V6** defines `listRow` as a type local to the function body (line 28-33 inside `RunList`).

**Assessment**: V6's local type is slightly better scoped since it is only needed inside the function. However, V5's package-level type is fine for a small CLI package and matches Go convention for named structs.

#### show.go -- Core Logic

**V5** (`533ee60:internal/cli/show.go`, 191 lines):

```go
// V5: show.go lines 36-143
func runShow(ctx *Context) error {
    if len(ctx.Args) == 0 {
        return fmt.Errorf("Task ID is required. Usage: tick show <id>")
    }
    id := task.NormalizeID(ctx.Args[0])
    ...
    var data showData
    var found bool

    err = store.Query(func(db *sql.DB) error {
        var description, parent, closed sql.NullString
        err := db.QueryRow(...).Scan(...)
        if err == sql.ErrNoRows {
            return nil  // not found but not an error -- checked after
        }
        found = true
        ...
    })
    if !found {
        return fmt.Errorf("Task '%s' not found", id)
    }
    ...
    printShowDetails(ctx.Stdout, data)
```

Key design choice: `sql.ErrNoRows` is **not** treated as an error inside the Query callback. Instead, a `found` boolean is checked after. This separates "not found" from actual query errors cleanly.

Error message format: `"Task ID is required..."` and `"Task '%s' not found"` (uppercase T).

Uses `sql.NullString` for nullable columns.

**V6** (`72b9e9e:internal/cli/show.go`, 187 lines):

```go
// V6: show.go lines 37-138
func RunShow(dir string, quiet bool, args []string, stdout io.Writer) error {
    if len(args) == 0 {
        return fmt.Errorf("task ID is required. Usage: tick show <id>")
    }
    id := task.NormalizeID(args[0])
    ...
    err = store.Query(func(db *sql.DB) error {
        var descPtr, parentPtr, closedPtr *string
        err := db.QueryRow(...).Scan(...)
        if err == sql.ErrNoRows {
            return fmt.Errorf("task '%s' not found", id)
        }
        ...
    })
```

Key design choice: `sql.ErrNoRows` immediately returns an error inside the Query callback. This means the "not found" error propagates through the store's Query method.

Error message format: `"task ID is required..."` and `"task '%s' not found"` (lowercase t). The spec says `"Error: Task 'tick-xyz' not found"` and `"Error: Task ID is required..."` with uppercase T. The `Error:` prefix is added by the CLI dispatcher, but V6's lowercase `task` means the final output reads `"Error: task 'tick-xyz' not found"` rather than the spec's `"Error: Task 'tick-xyz' not found"`.

Uses `*string` pointers for nullable columns instead of `sql.NullString`.

**Assessment**: V5's error messages match the spec's casing exactly. V6 uses lowercase which does not match the spec samples. V6's `*string` approach is slightly more concise than `sql.NullString`. V5's `found` flag pattern is more defensive (separates not-found from query errors in the callback contract), while V6's inline error is more direct.

#### show.go -- Parent Format

**V5**:
```go
// V5: show.go line 162-166
if d.parent != "" {
    if d.parentTitle != "" {
        fmt.Fprintf(w, "Parent:   %s  %s\n", d.parent, d.parentTitle)
    } else {
        fmt.Fprintf(w, "Parent:   %s\n", d.parent)
    }
}
```
Parent line: `Parent:   tick-aaaaaa  Parent task` (two spaces between ID and title).

**V6**:
```go
// V6: show.go line 157-161
if d.parentID != "" {
    if d.parentTitle != "" {
        fmt.Fprintf(w, "Parent:   %s (%s)\n", d.parentID, d.parentTitle)
    } else {
        fmt.Fprintf(w, "Parent:   %s\n", d.parentID)
    }
}
```
Parent line: `Parent:   tick-aaaaaa (Parent task)` (parenthesized title).

The spec example shows `Parent: tick-aaaaaa Parent task` but is ambiguous on the exact separator format. V6's parenthesized format is more consistent with the blocked_by/children format `tick-aaa  Title (status)`.

#### show.go -- Field Ordering

**V5** output order: ID, Title, Status, Priority, Created, Updated, [Closed], [Parent], [Blocked by], [Children], [Description]

**V6** output order: ID, Title, Status, Priority, [Parent], Created, Updated, [Closed], [Blocked by], [Children], [Description]

The spec example shows: ID, Title, Status, Priority, Created, Updated, [Blocked by], [Children], [Description]. Parent is not shown in the spec example's exact position. V6 places Parent before Created which is a valid choice but differs from V5's post-Updated placement.

#### show.go -- Dependency Ordering

**V5** does not use `ORDER BY` on dependency and children queries:
```go
// V5: show.go line 91
`SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON t.id = d.blocked_by WHERE d.task_id = ?`
```

**V6** uses `ORDER BY t.id` on both queries:
```go
// V6: show.go line 99
`SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id = ? ORDER BY t.id`
```

**Assessment**: V6's ORDER BY ensures deterministic output, which is better practice.

#### Description Empty Check

**V5**:
```go
// V5: show.go line 177
if strings.TrimSpace(d.description) != "" {
```

**V6**:
```go
// V6: show.go line 179 (at commit time)
if d.description != "" {
```

V5 trims whitespace before checking, so a description of `"   "` would be omitted. V6 only checks for empty string, so whitespace-only descriptions would render. V5 is more defensive.

#### Storage Package Naming

**V5** imports `github.com/leeovery/tick/internal/engine`.
**V6** imports `github.com/leeovery/tick/internal/storage`.

Different package names for the same concept. This is a project-level naming difference, not specific to this task.

### Code Quality

| Aspect | V5 | V6 |
|--------|----|----|
| Lines of code (commit) | list.go: 80, show.go: 191 | list.go: 77, show.go: 187 |
| Exported functions | No (uses Context pattern) | Yes (RunList, RunShow) |
| Error wrapping | `fmt.Errorf("querying tasks: %w", err)` | `fmt.Errorf("failed to query tasks: %w", err)` |
| Nullable handling | `sql.NullString` | `*string` pointer scan |
| Column widths | Matches spec (12, 12, 4) | Wider than spec (12, 13, 5) |
| Error casing | Uppercase `Task` matching spec | Lowercase `task` deviating from spec |
| Deterministic output | No ORDER BY on related queries | ORDER BY on related queries |
| Description guard | `strings.TrimSpace` check | Simple empty check |
| Separation of concerns | `printListTable`, `printShowDetails` helpers | Inline printing (list), `printShowOutput` helper (show) |

### Test Quality

#### V5 Tests

**File**: `internal/cli/list_test.go` (170 lines, 5 subtests)
**File**: `internal/cli/show_test.go` (340 lines, 15 subtests)

List tests (inside `TestList`):
1. `"it lists all tasks with aligned columns"` -- creates 2 tasks, checks header columns and first data row content (lines 13-59)
2. `"it lists tasks ordered by priority then created date"` -- 3 tasks with varying priority/created, verifies ordering (lines 61-103)
3. `"it prints 'No tasks found.' when no tasks exist"` -- empty project, checks exact output (lines 105-119)
4. `"it prints only task IDs with --quiet flag on list"` -- 2 tasks, verifies IDs only, no spaces (lines 121-150)
5. `"it executes through storage engine read flow (shared lock, freshness check)"` -- verifies JSONL-to-cache rebuild via Query (lines 152-169)

Show tests (inside `TestShow`):
1. `"it shows full task details by ID"` -- checks ID, Title, Status, Priority, Created, Updated lines (lines 13-45)
2. `"it shows blocked_by section with ID, title, and status of each blocker"` -- creates blocker+blocked, verifies section (lines 47-76)
3. `"it shows children section with ID, title, and status of each child"` -- parent+child, verifies section (lines 78-105)
4. `"it shows description section when description is present"` -- verifies Description section (lines 107-126)
5. `"it omits blocked_by section when task has no dependencies"` -- verifies no "Blocked by:" (lines 128-143)
6. `"it omits children section when task has no children"` -- verifies no "Children:" (lines 145-160)
7. `"it omits description section when description is empty"` -- verifies no "Description:" (lines 162-176)
8. `"it shows parent field with ID and title when parent is set"` -- creates parent+child, shows child, verifies Parent field (lines 178-202)
9. `"it omits parent field when parent is null"` -- verifies no "Parent:" (lines 204-219)
10. `"it shows closed timestamp when task is done or cancelled"` -- sets Closed time, verifies output (lines 221-242)
11. `"it omits closed field when task is open or in_progress"` -- verifies no "Closed:" (lines 244-259)
12. `"it errors when task ID not found"` -- verifies exit code 1, error message (lines 261-273)
13. `"it errors when no ID argument provided to show"` -- verifies exit code 1, usage hint (lines 275-287)
14. `"it normalizes input ID to lowercase for show lookup"` -- passes "TICK-AAAAAA", verifies found (lines 289-304)
15. `"it outputs only task ID with --quiet flag on show"` -- verifies only ID in output (lines 306-320)
16. `"it executes through storage engine read flow (shared lock, freshness check)"` -- verifies JSONL rebuild (lines 322-339)

**Total V5 tests: 21 subtests (5 list + 16 show)**

Note: spec requires 19 tests. V5 has 21 (16 show + 5 list). The spec's `"it executes through storage engine read flow"` test is split into separate list and show variants.

#### V6 Tests

**File**: `internal/cli/list_show_test.go` (456 lines in commit version)

Helper functions:
- `runList(t, dir, args...)` -- constructs `App` struct with injected writers and Getwd, runs `tick list` (lines 14-26)
- `runShow(t, dir, args...)` -- same pattern for `tick show` (lines 29-42)

List tests (inside `TestList`):
1. `"it lists all tasks with aligned columns"` -- 2 tasks, checks exact header and row strings (lines 45-78)
2. `"it lists tasks ordered by priority then created date"` -- 4 tasks, verifies ordering across 2 priority levels (lines 80-114)
3. `"it prints 'No tasks found.' when no tasks exist"` -- checks exact output including newline (lines 116-128)
4. `"it prints only task IDs with --quiet flag on list"` -- exact output comparison (lines 130-148)
5. `"it executes through storage engine read flow (shared lock, freshness check)"` -- runs list twice, compares outputs (lines 150-169)

Show tests (inside `TestShow`):
1. `"it shows full task details by ID"` -- exact full output comparison (lines 173-196)
2. `"it shows blocked_by section with ID, title, and status of each blocker"` -- exact substring matches (lines 198-217)
3. `"it shows children section with ID, title, and status of each child"` -- exact substring matches (lines 219-238)
4. `"it shows description section when description is present"` -- exact substring matches (lines 240-258)
5. `"it omits blocked_by section when task has no dependencies"` -- absence check (lines 260-275)
6. `"it omits children section when task has no children"` -- absence check (lines 277-292)
7. `"it omits description section when description is empty"` -- absence check (lines 294-309)
8. `"it shows parent field with ID and title when parent is set"` -- exact format `"Parent:   tick-parent (Auth System)\n"` (lines 311-327)
9. `"it omits parent field when parent is null"` -- absence check (lines 329-344)
10. `"it shows closed timestamp when task is done or cancelled"` -- exact timestamp match (lines 346-362)
11. `"it omits closed field when task is open or in_progress"` -- absence check (lines 364-378)
12. `"it errors when task ID not found"` -- exit code 1, lowercase error message check (lines 380-390)
13. `"it errors when no ID argument provided to show"` -- exit code 1, lowercase usage hint (lines 392-403)
14. `"it normalizes input ID to lowercase for show lookup"` -- uppercase input (lines 405-420)
15. `"it outputs only task ID with --quiet flag on show"` -- exact output comparison (lines 422-438)
16. `"it executes through storage engine read flow (shared lock, freshness check)"` -- runs twice, compares (lines 440-459)

**Total V6 tests: 21 subtests (5 list + 16 show)**

Both versions cover the same 19 spec tests (the storage engine read flow test is duplicated for both list and show, adding 2 extra, minus the single spec line = net 21).

#### Test Comparison

| Aspect | V5 | V6 |
|--------|----|----|
| Test files | 2 files (list_test.go, show_test.go) | 1 combined file (list_show_test.go) |
| Helper pattern | Uses `Run()` directly with `bytes.Buffer` | Uses `runList()`/`runShow()` wrapper helpers with `App{}` struct |
| Assertion style | `strings.Contains` checks | Mix of exact string equality and `strings.Contains` |
| Task construction | `task.NewTask()` with field mutations | Struct literals with all fields explicit |
| Format flag | Uses `--pretty` flag in most tests | Uses `IsTTY: true` to trigger pretty formatting |
| Precision | Loose (Contains-based) | Strict (exact output comparison where possible) |

V6's tests are more precise -- they use exact string comparisons for full output blocks (e.g., `"it shows full task details by ID"` compares the entire stdout). V5 uses `strings.Contains` for individual fields, which is more lenient but less thorough.

V6 also includes an additional test at the end of the file `"queryShowData populates RelatedTask fields for blockers and children"` (lines 461-515 in current worktree), which unit-tests the `queryShowData` function directly. This test was added in a later commit, not in the task-1-7 commit itself.

### Skill Compliance

| Rule | V5 | V6 |
|------|----|----|
| Use gofmt/golangci-lint on all code | YES -- standard formatting | YES -- standard formatting |
| Handle all errors explicitly | YES -- all errors checked | YES -- all errors checked |
| Write table-driven tests with subtests | PARTIAL -- uses subtests but not table-driven for most | PARTIAL -- same approach |
| Document all exported functions/types/packages | N/A -- no exported symbols in commit | YES -- `RunList`, `RunShow` have doc comments |
| Propagate errors with `fmt.Errorf("%w", err)` | YES -- `%w` wrapping throughout | YES -- `%w` wrapping throughout |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic for normal error handling | PASS | PASS |

### Spec-vs-Convention Conflicts

1. **Column widths**: Spec states "(12), (12), (4)". V5 matches exactly; V6 uses (12, 13, 5), deviating from spec.

2. **Error message casing**: Spec samples show `"Error: Task ID is required..."` and `"Error: Task 'tick-xyz' not found"`. V5 produces `"Error: Task ID is required..."` and `"Error: Task 'tick-xyz' not found"` (exact match). V6 produces `"Error: task ID is required..."` and `"Error: task 'tick-xyz' not found"` (lowercase `t`).

3. **Parent field format**: Spec example shows `Parent: tick-aaaaaa` with just the ID, but says "show ID and title". V5 shows `Parent:   tick-aaaaaa  Parent task`. V6 shows `Parent:   tick-aaaaaa (Auth System)`. V6's parenthesized format is arguably more readable and consistent with blocked_by/children format.

4. **Description empty check**: Spec says "Omit sections with no data". V5 uses `strings.TrimSpace` which is more thorough. V6 uses simple empty check which might show whitespace-only descriptions.

5. **show output field ordering**: Spec sample has Created/Updated immediately after Priority. V5 follows this. V6 inserts Parent between Priority and Created, which reorders the spec's sample layout.

## Diff Stats

| Metric | V5 | V6 |
|--------|----|----|
| Files changed | 7 | 6 |
| Lines added | 789 | 749 |
| Lines removed | 4 | 5 |
| list.go (lines at commit) | 80 | 77 |
| show.go (lines at commit) | 191 | 187 |
| list_test.go / list_show_test.go (lines at commit) | 170 (list) + 340 (show) = 510 | 456 (combined) |
| New files | list.go, list_test.go, show.go, show_test.go | list.go, list_show_test.go, show.go |
| Modified files | cli.go (+2 lines) | app.go (+22 lines) |

## Verdict

**V5 is the more spec-faithful implementation.**

V5 matches the task plan's column widths exactly (12, 12, 4), uses the correct error message casing (`"Task"` with uppercase T), and follows the spec's field ordering in show output. Its use of `strings.TrimSpace` for the description guard and the `found` boolean pattern for separating "not found" from query errors are more defensive. The code is clean, concise, and minimal for a Phase 1 implementation.

V6 is a strong implementation with better architectural choices for long-term maintenance: exported functions with explicit parameters (more testable), deterministic `ORDER BY` on related queries, exact-match test assertions, and clean helper functions. However, it deviates from the spec in column widths (STATUS=13, PRI=5 vs specified 12, 4), uses lowercase error messages where the spec requires uppercase, and reorders the show output fields.

Both implementations fully satisfy all 14 acceptance criteria and cover all 19 specified tests (plus 2 extras). The choice comes down to spec fidelity (V5) vs architectural quality (V6). For a task where the spec is explicit about formatting details, V5's literal compliance gives it the edge.
