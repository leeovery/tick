# Task tick-core-1-7: tick list & tick show commands

## Task Summary

This task implements the two read commands that complete the Phase 1 walking skeleton. `tick list` displays all tasks in an aligned column table (ID, STATUS, PRI, TITLE) ordered by priority ASC then created ASC, with `--quiet` outputting only IDs. `tick show <id>` displays full task details in key-value format including optional sections for blocked_by, children, parent, description, and closed -- omitting sections with no data. Both commands must use the storage engine's read flow (shared lock, freshness check, SQLite query).

### Acceptance Criteria (from plan)

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Aligned columns (ID, STATUS, PRI, TITLE) | PARTIAL -- columns implemented with spaces between format directives (`"%-12s %-12s %-4s %s\n"`), but **commands not registered in CLI dispatcher switch statement**, so `tick list` would return "Unknown command" at runtime | PASS -- columns implemented (`"%-12s%-12s%-4s%s\n"`) but **no spaces between column values**, making output less readable when ID fills 12 chars | PASS -- columns implemented with spaces between format directives (`"%-12s %-12s %-4s %s\n"`), commands registered in dispatcher |
| 2. Order by priority ASC, created ASC | PASS -- SQL: `ORDER BY priority ASC, created ASC` | PASS -- identical SQL | PASS -- identical SQL |
| 3. `No tasks found.` when empty | PASS -- `fmt.Fprintln(a.stdout, "No tasks found.")` | PASS -- identical | PASS -- identical |
| 4. `--quiet` outputs only IDs | PASS -- iterates tasks, prints only IDs | PASS -- identical logic | PASS -- identical logic |
| 5. Full task details | PARTIAL -- implementation present but unreachable via CLI dispatcher | PASS -- all fields rendered | PASS -- all fields rendered |
| 6. blocked_by with context | PASS -- joins dependencies + tasks tables, shows ID, title, status | PASS -- identical join query | PASS -- identical join query |
| 7. Children with context | PASS -- queries `tasks WHERE parent=?` | PASS -- identical | PASS -- identical |
| 8. Parent field when set | PASS -- queries parent task, shows ID + title + status | PARTIAL -- shows ID + title but **no status** in parent display (spec example shows `ID  title (status)` for blocked_by/children but spec only says "show ID and title" for parent) | PARTIAL -- shows ID + title in parens: `Parent: tick-parent (Parent task)` instead of `Parent: tick-parent  Parent task` -- differs from spec's implied format |
| 9. Omits empty optional sections | PASS -- conditional checks on blockers, children, description, parent, closed | PASS -- identical logic | PASS -- identical logic |
| 10. Errors when ID not found | PASS -- returns `Task 'tick-xyz' not found` | PASS -- returns error from inside Query callback | PASS -- prints error to stderr with exit code 1 |
| 11. Errors when no ID argument | PASS -- checks `len(args) == 0` | PASS -- checks `len(args) == 0` | PASS -- checks `len(args) < 3` (full args include "tick" and "show") |
| 12. ID normalization to lowercase | PASS -- `task.NormalizeID(args[0])` | PASS -- identical | PASS -- `task.NormalizeID(args[2])` |
| 13. Storage engine read flow | PASS -- uses `store.Query()` | PASS -- uses `store.Query()` | PASS -- uses `store.Query()` |
| 14. Exit codes 0/1 | FAIL -- **commands unreachable through dispatcher** so errors would be "Unknown command" (exit 1) rather than proper error handling | PASS -- errors return from `runShow`/`runList` through `app.Run` error handler | PASS -- returns int exit codes directly |

## Implementation Comparison

### Approach

**File Organization:**

- **V1** places both `cmdList` and `cmdShow` in a single file `internal/cli/list.go` (243 lines), and all tests in `internal/cli/list_show_test.go` (269 lines). It does NOT modify `cli.go` to register the commands.
- **V2** separates `runList` into `internal/cli/list.go` (74 lines) and `runShow` into `internal/cli/show.go` (189 lines), with separate test files `list_test.go` (187 lines) and `show_test.go` (368 lines). It modifies `app.go` to register commands.
- **V3** separates `runList` into `internal/cli/list.go` (80 lines) and `runShow` into `internal/cli/show.go` (191 lines), with combined test file `list_show_test.go` (593 lines) plus a new shared `test_helpers_test.go` (164 lines). It modifies `cli.go` and also refactors `create_test.go` to extract shared helpers.

**Critical defect in V1 -- missing dispatcher registration:**

V1 added help text entries for list and show:
```go
fmt.Fprintln(a.stdout, "  list      List tasks")
fmt.Fprintln(a.stdout, "  show      Show task details")
```
But the `switch subcmd` block in `cli.go` still only has `case "init"` and `case "create"`. The methods `cmdList` and `cmdShow` exist but are dead code from the CLI's perspective. Running `tick list` would produce `Error: Unknown command 'list'`.

V2 correctly registers both in `app.go`:
```go
case "list":
    return a.runList()
case "show":
    return a.runShow(cmdArgs)
```

V3 correctly registers both in `cli.go`:
```go
case "list":
    return a.runList(args)
case "show":
    return a.runShow(args)
```

**Return type conventions:**

- **V1**: Methods return `error`. The `Run` wrapper prints `"Error: "` prefix and returns exit code 1.
- **V2**: Methods return `error`. Same wrapper pattern.
- **V3**: Methods return `int` (exit code). Each method handles its own error formatting directly to `a.Stderr`.

V3's approach means error formatting is duplicated in every command method. V1/V2's approach centralizes error formatting in the `Run` method. V2's approach is more idiomatic Go for CLI apps -- return errors upward.

**Argument handling:**

- **V1**: `cmdShow(workDir string, args []string)` receives pre-parsed args. Checks `len(args) == 0`.
- **V2**: `runShow(args []string)` receives pre-parsed args. Checks `len(args) == 0`. Uses `a.workDir` from App struct.
- **V3**: `runShow(args []string)` receives full args including `"tick"` and `"show"`. Checks `len(args) < 3` and uses `args[2]`.

V3's raw-args approach is more fragile -- the function must know its position in the argument chain. V2's pre-parsed args are cleaner.

**Null field handling in SQL:**

- **V1**: Uses `sql.NullString` for description, parent, closed. Checks `.Valid` before using.
- **V2**: Uses `sql.NullString` identically.
- **V3**: Uses `COALESCE(description, '')` in SQL, scans directly into `string`. Simpler Go code but pushes logic into SQL.

V3's COALESCE approach is more concise:
```go
// V3
row := db.QueryRow(`SELECT id, title, status, priority, COALESCE(description, ''), COALESCE(parent, ''), created, updated, COALESCE(closed, '') FROM tasks WHERE id = ?`, taskID)
err := row.Scan(&t.ID, &t.Title, &t.Status, &t.Priority, &t.Description, &t.Parent, &t.Created, &t.Updated, &t.Closed)
```
vs V1/V2:
```go
// V1/V2
var description, parent, closed sql.NullString
err := db.QueryRow("SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id=?", taskID,
).Scan(&td.ID, &td.Title, &td.Status, &td.Priority, &description, &parent, &created, &updated, &closed)
// ... then 3 separate .Valid checks
```

**"Not found" error pattern:**

- **V1**: Sets `found = true` inside query callback, checks `!found` after query. Returns error.
- **V2**: Returns `fmt.Errorf("Task '%s' not found", lookupID)` from inside the Query callback. This mixes business logic errors with storage-level errors.
- **V3**: Sets `found = true` inside query callback, checks `!found` after query. Prints error directly.

V1 and V3 are cleaner: they separate "no rows" (not an error) from actual database errors. V2 returns a business error through the storage layer's error channel, which could cause issues if the store layer wraps errors.

**Parent display format:**

- **V1**: `Parent:   tick-parent  Parent task (open)` -- includes parent's status
- **V2**: `Parent:   tick-parent  Parent task` -- ID and title only, two spaces between
- **V3**: `Parent:   tick-parent (Parent task)` -- title in parentheses after ID

The spec says: "Include `Parent:` only when set (show ID and title)". V2 matches the spec exactly. V1 adds extra status. V3 uses a different formatting convention (parens).

**Column formatting:**

- **V1/V3**: `"%-12s %-12s %-4s %s\n"` -- explicit space between columns
- **V2**: `"%-12s%-12s%-4s%s\n"` -- no explicit space, relies solely on padding

With a 12-char ID like `tick-a1b2c3`, V2's output would have no visual gap between columns. V1/V3's extra spaces ensure at least 1 character of separation.

**Description multiline handling:**

- **V1**: Splits on `\n` and indents each line: `for _, line := range strings.Split(td.Description, "\n") { fmt.Fprintf(a.stdout, "  %s\n", line) }`
- **V2**: Single print: `fmt.Fprintf(a.stdout, "  %s\n", d.Description)` -- does NOT handle multiline
- **V3**: Same as V1, splits and indents each line

V2's single-line output would render multiline descriptions incorrectly -- the second line would not be indented.

### Code Quality

**Naming:**

- V1 uses `cmdList`/`cmdShow` (consistent with existing `cmdInit`/`cmdCreate` pattern in its codebase).
- V2 uses `runList`/`runShow` (consistent with its existing `runInit`/`runCreate`).
- V3 uses `runList`/`runShow` (same as V2).
- All three follow their respective codebase conventions.

**Type definitions:**

- V1 defines `taskRow`, `depInfo`, `taskDetail` as local types inside functions. No reuse.
- V2 defines `showData` and `relatedTask` as package-level types in `show.go`. Better for documentation and reuse.
- V3 defines `taskDetails` and `relatedTask` as local types inside `runShow`. Similar to V1 but inside a different scope.

V2's package-level types are the most idiomatic Go approach -- they're documented via godoc and can be referenced from tests.

**Error messages:**

- V1: `fmt.Errorf("opening store: %w", err)`, `fmt.Errorf("querying tasks: %w", err)` -- wraps with context
- V2: `fmt.Errorf("failed to query tasks: %w", err)`, `fmt.Errorf("failed to scan task row: %w", err)` -- wraps with "failed to" prefix
- V3: Bare `return err` from query callback -- no wrapping

V1 and V2 provide better error context. V3's bare returns make debugging harder. V2's explicit row iteration error checking with `depRows.Err()` and `childRows.Err()` is the most thorough.

**Dead code:**

V1 has `_ = args` at the end of `cmdList`:
```go
_ = args
return nil
```
This is a code smell -- the `args` parameter is accepted but unused.

**Struct organization in V2:**

V2 separates `printShowOutput` into its own method:
```go
func (a *App) printShowOutput(d *showData) {
```
This separation of data gathering from rendering is cleaner than V1/V3 where rendering is inline in the main function.

### Test Quality

**V1 test functions (17 total, 0 matching spec names exactly):**

TestListCommand:
1. `lists all tasks with aligned columns`
2. `lists tasks ordered by priority then created`
3. `prints 'No tasks found.' when empty`
4. `prints only task IDs with --quiet flag`

TestShowCommand:
5. `shows full task details by ID`
6. `shows blocked_by section with context`
7. `shows children section`
8. `shows description when present`
9. `omits blocked_by section when no dependencies`
10. `omits children section when no children`
11. `omits description section when empty`
12. `shows parent with ID and title when set`
13. `omits parent field when not set`
14. `errors when task ID not found`
15. `errors when no ID argument provided`
16. `normalizes input ID to lowercase`
17. `outputs only task ID with --quiet flag`

**V1 MISSING tests (from spec):**
- "it shows closed timestamp when task is done or cancelled"
- "it omits closed field when task is open or in_progress"
- "it executes through storage engine read flow (shared lock, freshness check)"

**V2 test functions (20 total, all matching spec names exactly with "it" prefix):**

TestListCommand (5):
1. `it lists all tasks with aligned columns`
2. `it lists tasks ordered by priority then created date`
3. `it prints 'No tasks found.' when no tasks exist`
4. `it prints only task IDs with --quiet flag on list`
5. `it executes through storage engine read flow (shared lock, freshness check)`

TestShowCommand (15):
6. `it shows full task details by ID`
7. `it shows blocked_by section with ID, title, and status of each blocker`
8. `it shows children section with ID, title, and status of each child`
9. `it shows description section when description is present`
10. `it omits blocked_by section when task has no dependencies`
11. `it omits children section when task has no children`
12. `it omits description section when description is empty`
13. `it shows parent field with ID and title when parent is set`
14. `it omits parent field when parent is null`
15. `it shows closed timestamp when task is done or cancelled`
16. `it omits closed field when task is open or in_progress`
17. `it errors when task ID not found`
18. `it errors when no ID argument provided to show`
19. `it normalizes input ID to lowercase for show lookup`
20. `it outputs only task ID with --quiet flag on show`

**V2 MISSING tests:**
- No show-specific storage engine read flow test (only in list_test)

**V3 test functions (21 total, all matching spec names exactly with "it" prefix):**

TestListCommand (5):
1. `it lists all tasks with aligned columns`
2. `it lists tasks ordered by priority then created date`
3. `it prints 'No tasks found.' when no tasks exist`
4. `it prints only task IDs with --quiet flag on list`
5. `it executes through storage engine read flow (shared lock, freshness check)`

TestShowCommand (16):
6. `it shows full task details by ID`
7. `it shows blocked_by section with ID, title, and status of each blocker`
8. `it shows children section with ID, title, and status of each child`
9. `it shows description section when description is present`
10. `it omits blocked_by section when task has no dependencies`
11. `it omits children section when task has no children`
12. `it omits description section when description is empty`
13. `it shows parent field with ID and title when parent is set`
14. `it omits parent field when parent is null`
15. `it shows closed timestamp when task is done or cancelled`
16. `it omits closed field when task is open or in_progress`
17. `it errors when task ID not found`
18. `it errors when no ID argument provided to show`
19. `it normalizes input ID to lowercase for show lookup`
20. `it outputs only task ID with --quiet flag on show`
21. `it executes through storage engine read flow (shared lock, freshness check)` (for show)

**V3 MISSING tests:** None -- covers all 19 spec tests plus 2 storage engine read flow tests (list + show).

**Test setup patterns:**

- **V1**: Uses `initTickDir()` which calls `tick init`, then `createTask()` which calls `tick create`. Tests exercise the full CLI pipeline. This is integration-test style but couples tests to `create` working correctly.
- **V2**: Uses `setupTickDirWithContent(content)` and `setupInitializedTickDir()` which write JSONL directly. Tests are decoupled from create command.
- **V3**: Uses `setupTickDir()`, `setupTask()`, `setupTaskWithPriority()`, `setupTaskFull()` which write JSONL directly. Extracted into `test_helpers_test.go`. Most flexible setup functions. The `setupTaskFull` helper accepts all fields including `blockedBy []string`, `closed string`, etc.

V3's test helpers are the most complete and reusable. V1's approach of using `tick create` is fragile -- if create has bugs, list/show tests break. V2 and V3's direct JSONL manipulation is better for isolated testing.

**Column alignment verification:**

- **V1**: No explicit column alignment check.
- **V2**: Explicitly checks column positions match between header and data rows:
  ```go
  headerStatusPos := strings.Index(header, "STATUS")
  row1StatusPos := strings.Index(lines[1], "done")
  row2StatusPos := strings.Index(lines[2], "in_progress")
  if headerStatusPos != row1StatusPos || headerStatusPos != row2StatusPos {
  ```
- **V3**: No explicit column alignment check.

V2 is the only version that actually verifies column alignment programmatically.

**Children test thoroughness:**

- **V1**: Tests with 1 child
- **V2**: Tests with 2 children (verifies both appear)
- **V3**: Tests with 2 children (verifies both appear)

**Priority ordering test:**

- **V1**: Tests 3 tasks with priorities 3, 0, 2. Verifies first and last only.
- **V2**: Tests 3 tasks with priorities 2, 1, 1 (two same-priority). Verifies all 3 positions including same-priority tiebreak by created.
- **V3**: Tests 4 tasks with priorities 4, 0, 2, 2 (two same-priority). Verifies all 4 positions.

V3's ordering test is the most thorough with 4 tasks covering the tiebreak case.

**Error prefix verification:**

- **V1**: Checks `stderr` contains "not found" and "Task ID is required" -- does not verify `"Error: "` prefix
- **V2**: Checks `err.Error()` contains messages -- prefix added by wrapper, not tested
- **V3**: Checks stderr string starts with `"Error: "` prefix explicitly: `if !strings.HasPrefix(errOutput, "Error: ")`

V3 is the only version that verifies the error prefix convention.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 7 (5 code, 2 docs) | 10 (6 code, 2 docs, 1 config, 1 refactored) |
| Lines added | 512 | 830 | 1056 |
| Lines deleted | 0 | 4 | 91 |
| Impl LOC | 243 (1 file) | 263 (2 files: 74+189) | 271 (2 files: 80+191) |
| Test LOC | 269 (1 file) | 555 (2 files: 187+368) | 757 (2 files: 593+164) |
| Test functions | 17 | 20 | 21 |

## Verdict

**V2 is the best implementation**, with V3 as a close second and V1 as clearly the weakest.

**V1 has a critical defect**: commands are not registered in the CLI dispatcher switch statement. Despite implementing `cmdList` and `cmdShow` correctly, they are dead code from the CLI perspective. Running `tick list` or `tick show` would produce `"Error: Unknown command 'list'"`. V1 also misses 3 spec tests (closed field tests and storage engine read flow test). These omissions disqualify it.

**V2 vs V3:**

V2 wins on:
- **Cleaner architecture**: `runShow` returns `error` (not `int`), letting the central `Run` method handle formatting. This is standard Go CLI practice.
- **Better separation of concerns**: `printShowOutput` is extracted as a separate method. `showData` and `relatedTask` are package-level types.
- **Pre-parsed arguments**: `runShow(args)` receives only the relevant args, not the full command line.
- **Column alignment verification**: Only V2 programmatically verifies that columns actually align.
- **Idiomatic error handling**: Wraps errors with descriptive context (`"failed to query task: %w"`) plus checks `depRows.Err()` after iteration.

V3 wins on:
- **Test completeness**: 21 tests vs V2's 20 -- V3 adds a storage engine read flow test for both list AND show.
- **Test infrastructure**: Extracts shared helpers into `test_helpers_test.go`, refactors `create_test.go` to use them. This benefits future tasks.
- **Error prefix verification**: Explicitly tests `"Error: "` prefix convention.
- **Multiline description handling**: Splits on newlines like V1 (V2 does not).
- **Column spacing**: Adds explicit space separators between columns (V2 does not).
- **Priority ordering test**: 4 tasks vs V2's 3.

V2's column formatting issue (no spaces between format directives) is a minor visual defect. V2's single-line description output is a functional defect for multiline descriptions. However, V2's superior architecture, cleaner error handling, and better code organization make it the strongest overall implementation. The description multiline issue is a small bug that's easy to fix (add `strings.Split` loop), while V3's architectural choices (returning `int` from every method, handling raw args) create more pervasive code quality issues.

**Final ranking**: V2 > V3 > V1
