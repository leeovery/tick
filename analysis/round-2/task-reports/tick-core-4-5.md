# Task tick-core-4-5: Integrate Formatters into All Commands

## Task Summary

Replace all hardcoded output from Phases 1-3 with the resolved Formatter. Every command produces format-aware output driven by TTY detection and flag overrides. `--quiet` overrides format entirely. Specifically:

- Wire `FormatConfig` + `Formatter` into every handler via CLI dispatcher
- **create/update**: `FormatTaskDetail` (same as show). `--quiet`: ID only.
- **start/done/cancel/reopen**: `FormatTransition`. `--quiet`: nothing.
- **dep add/rm**: `FormatDepChange`. `--quiet`: nothing.
- **list** (with filters): `FormatTaskList`. Empty handled per format. `--quiet`: IDs only.
- **show**: `FormatTaskDetail`. `--quiet`: ID only.
- **init/rebuild**: `FormatMessage`. `--quiet`: nothing.
- Format resolved once in dispatcher, not per-command. Errors remain plain text to stderr.

### Acceptance Criteria (from plan)

1. All commands output via Formatter
2. `--quiet` overrides per spec (ID for mutations, nothing for transitions/deps/messages)
3. Empty list correct per format
4. TTY auto-detection end-to-end
5. Flag overrides work for all commands
6. Errors remain plain text stderr
7. Format resolved once in dispatcher

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| All commands output via Formatter | PASS -- create, update, show, list, init, transitions, dep add/rm all routed through `a.formatter.Format*` | PASS -- create, update, show, list, init, transitions, dep add/rm, ready, blocked all routed through `a.Formatter.Format*` |
| --quiet overrides per spec | PASS -- create/update: ID only; transitions/deps/init: nothing; list: IDs only; show: ID only | PASS -- Same behavior, verified by tests |
| Empty list correct per format | PASS -- TOON tasks[0], Pretty "No tasks found.", JSON `[]` tested | PASS -- TOON tasks[0], Pretty "No tasks found.", JSON `[]` tested |
| TTY auto-detection end-to-end | PASS -- Tests confirm non-TTY defaults to TOON | PASS -- Tests confirm non-TTY defaults to TOON, explicit formatter type assertion |
| Flag overrides work for all commands | PASS -- --toon, --pretty, --json tested via show command | PASS -- --toon, --pretty, --json tested via init (flag value only) |
| Errors remain plain text stderr | PARTIAL -- No explicit error-format test in integration suite | PASS -- `TestFormatIntegration_ErrorsRemainPlainTextStderr` explicitly tests all 3 formats produce plain text stderr |
| Format resolved once in dispatcher | PASS -- `newFormatter()` called once in `App.Run()`, verified by test checking `app.formatter != nil` | PASS -- `resolveFormatter()` called once in `App.Run()`, verified by 4 tests checking `app.Formatter` type |

## Implementation Comparison

### Approach

Both versions follow the same fundamental strategy: remove hardcoded `fmt.Fprintf` output from command handlers and replace with calls to the `Formatter` interface methods. The format is resolved once in the dispatcher (`App.Run`), and `--quiet` checks remain in individual command handlers (before calling the formatter).

**V2 approach -- centralized query for create/update:**

V2 extracts `queryShowData()` as a package-level function in `show.go`, reused by `create.go` and `update.go`. After creating/updating a task, V2 opens a SQL query to fetch the full `showData` (including blocked_by relations, children, parent title) and passes the rich `*showData` to the formatter:

```go
// create.go (V2)
data, err := queryShowData(store, createdTask.ID)
if err != nil {
    // Fallback: if query fails, just print the ID
    fmt.Fprintln(a.stdout, createdTask.ID)
    return nil
}
return a.formatter.FormatTaskDetail(a.stdout, data)
```

The `Formatter.FormatTaskDetail` takes `*showData` directly, which means formatters have access to the full relational data (blocked_by tasks with titles and statuses, children, parent title). This is the same data structure used by `show`, so create/update output is byte-for-byte identical to show output.

**V4 approach -- lightweight conversion for create/update:**

V4 introduces a `taskToDetail()` helper in `create.go` that converts a `task.Task` struct to a `TaskDetail` value type without making any additional database queries:

```go
// create.go (V4)
func taskToDetail(t *task.Task) TaskDetail {
    detail := TaskDetail{
        ID:          t.ID,
        Title:       t.Title,
        Status:      string(t.Status),
        Priority:    t.Priority,
        Description: t.Description,
        Parent:      t.Parent,
        Created:     t.Created.Format("2006-01-02T15:04:05Z"),
        Updated:     t.Updated.Format("2006-01-02T15:04:05Z"),
    }
    if t.Closed != nil {
        detail.Closed = t.Closed.Format("2006-01-02T15:04:05Z")
    }
    return detail
}
```

The `Formatter.FormatTaskDetail` takes a `TaskDetail` value type. This means create/update output will NOT include blocked_by context (related task titles/statuses), children details, or parent title -- only the basic task fields. The comment explicitly acknowledges this: "blocked_by and children are empty since create/update don't have the full DB context for related task details."

**Key architectural difference -- the Formatter interface:**

V2's `Formatter` interface uses domain types directly:

```go
// V2 formatter.go
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatTaskList(w io.Writer, tasks []TaskRow) error
```

V4's `Formatter` interface uses the shared `TaskDetail` struct and passes `quiet` into some formatters:

```go
// V4 format.go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
```

V4's `FormatTaskList` and `FormatDepChange` accept a `quiet bool` parameter, pushing the quiet-handling responsibility into the formatter. V2 keeps quiet handling in the command handlers and never calls the formatter at all when quiet is active.

**V2's quiet handling pattern:**

```go
// dep.go (V2)
if !a.config.Quiet {
    return a.formatter.FormatDepChange(a.stdout, "added", taskID, blockedByID)
}
return nil
```

**V4's quiet handling pattern:**

```go
// dep.go (V4)
return a.Formatter.FormatDepChange(a.Stdout, taskID, blockedByID, "added", a.Quiet)
```

V4 delegates quiet handling to the formatter for dep and list, but still handles it in the command handler for transitions, init, create, update, and show. This creates an inconsistency -- some commands check quiet before calling the formatter, others pass quiet to the formatter.

**Handling of `blocked` and `ready` commands:**

V2 aliases `ready` and `blocked` to `runList` with synthetic args in the dispatcher:
```go
case "ready":
    return a.runList([]string{"--ready"})
case "blocked":
    return a.runList([]string{"--blocked"})
```

V4 has dedicated `runReady` and `runBlocked` methods that call `a.Formatter.FormatTaskList()` directly. Both approaches work; V2 is more DRY, V4 gives each command its own entry point.

**`listRow` vs `TaskRow` type usage in formatters:**

V2 defines `listRow` as a local type in `list.go` and also has `TaskRow` in `formatter.go`. In the `runList` handler, V2 converts `listRow` to `TaskRow` before passing to the formatter:

```go
// list.go (V2)
taskRows := make([]TaskRow, len(rows))
for i, r := range rows {
    taskRows[i] = TaskRow{
        ID:       r.ID,
        Status:   r.Status,
        Priority: r.Priority,
        Title:    r.Title,
    }
}
return a.formatter.FormatTaskList(a.stdout, taskRows)
```

V4 uses `listRow` directly in the Formatter interface (`FormatTaskList(w io.Writer, rows []listRow, quiet bool)`), avoiding the conversion step. This is slightly more efficient but couples the formatter interface to an internal type.

### Code Quality

**Exported vs unexported fields on App:**

V2 keeps most App fields unexported (`config`, `formatter`, `workDir`, `stdout`, `stderr`) with the exception of `FormatCfg`. V4 exports everything (`Stdout`, `Stderr`, `Dir`, `Quiet`, `Verbose`, `OutputFormat`, `IsTTY`, `FormatCfg`, `Formatter`). V2's approach is better Go practice -- exported fields should only be exposed when needed by external packages.

**Error handling in create/update:**

V2 has a defensive fallback in create/update when `queryShowData` fails:

```go
// V2 create.go
data, err := queryShowData(store, createdTask.ID)
if err != nil {
    fmt.Fprintln(a.stdout, createdTask.ID)
    return nil
}
```

V4 has no such fallback because `taskToDetail()` is a pure conversion that cannot fail. V2's approach is more robust for the case where the database query after creation might fail, but V4 avoids that scenario entirely by not querying.

**Return value style:**

V2's `App.Run` returns `error`. V4's `App.Run` returns `int` (exit code) and handles errors internally via `writeError()`:

```go
// V4 cli.go
func (a *App) Run(args []string) int {
    ...
    case "create":
        if err := a.runCreate(subArgs); err != nil {
            a.writeError(err)
            return 1
        }
        return 0
    ...
}
```

This is a pre-existing architectural difference, not introduced by this task.

**Format type representation:**

V2 uses `OutputFormat` as a `string` type (`"toon"`, `"pretty"`, `"json"`). V4 uses `Format` as an `int` type (iota constants). The `int` approach (V4) is more idiomatic Go for enums -- it's more efficient and prevents invalid string values.

**Transition type signatures:**

V2 passes `task.Status` type to `FormatTransition`:
```go
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
```

V4 passes `string` type:
```go
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
```

V4 converts at the call site: `string(result.OldStatus)`, `string(result.NewStatus)`. V4's approach is slightly better because it decouples the formatter interface from the `task` package domain types.

**Removal of `renderListOutput`:**

V4 removes the `renderListOutput` helper function from `list.go` (which did hardcoded column formatting) and replaces it with `a.Formatter.FormatTaskList()`. V2 also removes the hardcoded `printTaskDetails` and `printShowOutput` methods. Both correctly eliminate the Phase 1-3 hardcoded output.

**V4 adds `ParentTitle` to format types:**

V4 adds `ParentTitle` to `TaskDetail`, `jsonTaskDetail`, and updates toon/pretty formatters to use it. V2 does the same through the `showData` struct (which already had `ParentTitle` from the show command). Both approaches correctly propagate parent title to all formatters.

### Test Quality

**V2 test structure:**

V2 uses a single top-level test function `TestFormatIntegration` with 29 subtests. Uses table-driven tests for format-variant testing (init/create/update/transitions/deps/list/show each tested across toon/pretty/json). Quiet override tests are individual subtests. One `--quiet + --json` edge case test. Empty list per format uses table-driven approach. One TTY auto-detection test. Three flag override tests. One dispatcher resolution test.

**V2 test functions and subtests (all under TestFormatIntegration):**
1. `it formats init as message in each format` (table: toon, pretty, json)
2. `it formats create as full task detail in each format` (table: toon, pretty, json)
3. `it formats update as full task detail in each format` (table: toon, pretty, json)
4. `it formats transitions in each format` (table: toon, pretty, json)
5. `it formats dep confirmations in each format` (table: toon, pretty, json)
6. `it formats list in each format` (table: toon, pretty, json)
7. `it formats show in each format` (table: toon, pretty, json)
8. `it applies --quiet override for create (ID only)`
9. `it applies --quiet override for update (ID only)`
10. `it applies --quiet override for transitions (nothing)`
11. `it applies --quiet override for dep add (nothing)`
12. `it applies --quiet override for init (nothing)`
13. `it applies --quiet override for list (IDs only)`
14. `it applies --quiet override for show (ID only)`
15. `it applies --quiet even when --json is set (quiet wins)`
16. `it handles empty list per format` (table: toon, pretty, json)
17. `it defaults to TOON when piped (non-TTY)`
18. `it respects --toon override`
19. `it respects --pretty override`
20. `it respects --json override`
21. `it resolves format once in dispatcher not per command`

**V4 test structure:**

V4 uses 13 top-level test functions with 39 subtests. Tests are organized by feature area (create, update, transitions, deps, list/show, init, quiet, quiet+json, empty list, TTY default, flag overrides, errors, dispatcher resolution). Most tests are individual subtests rather than table-driven, which leads to more boilerplate but more explicit test descriptions.

**V4 test functions and subtests:**
1. `TestFormatIntegration_CreateFormatsAsTaskDetail`
   - `it formats create as full task detail in TOON (default non-TTY)`
   - `it formats create as task detail in Pretty with --pretty`
   - `it formats create as task detail in JSON with --json`
2. `TestFormatIntegration_UpdateFormatsAsTaskDetail`
   - `it formats update as full task detail in each format` (table: toon, json -- missing pretty!)
3. `TestFormatIntegration_TransitionsInEachFormat`
   - `it formats transitions in TOON/Pretty as plain text`
   - `it formats transitions in JSON as structured object`
4. `TestFormatIntegration_DepConfirmationsInEachFormat`
   - `it formats dep add in TOON as plain text`
   - `it formats dep add in JSON as structured object`
   - `it formats dep rm in JSON as structured object`
5. `TestFormatIntegration_ListShowInEachFormat`
   - `it formats list in TOON (default non-TTY)`
   - `it formats list in Pretty with --pretty`
   - `it formats list in JSON with --json`
   - `it formats show in TOON (default non-TTY)`
   - `it formats show in JSON with --json`
6. `TestFormatIntegration_InitRebuildInEachFormat`
   - `it formats init as message in TOON`
   - `it formats init as message in JSON`
7. `TestFormatIntegration_QuietOverridePerCommandType`
   - `it outputs only ID for create with --quiet`
   - `it outputs nothing for transition with --quiet`
   - `it outputs nothing for dep with --quiet`
   - `it outputs nothing for init with --quiet`
   - `it outputs IDs only for list with --quiet`
   - `it outputs only ID for show with --quiet`
   - `it outputs only ID for update with --quiet`
8. `TestFormatIntegration_QuietPlusJsonQuietWins`
   - `it outputs nothing with --quiet + --json for transition (quiet wins)`
   - `it outputs only ID with --quiet + --json for create (quiet wins, no JSON wrapping)`
9. `TestFormatIntegration_EmptyListPerFormat`
   - `it outputs TOON zero-count for empty list (default non-TTY)`
   - `it outputs Pretty message for empty list`
   - `it outputs JSON empty array for empty list`
10. `TestFormatIntegration_DefaultsToonWhenPiped`
    - `it defaults to TOON when stdout is bytes.Buffer (non-TTY)`
11. `TestFormatIntegration_FlagOverrides`
    - `it respects --toon override`
    - `it respects --pretty override`
    - `it respects --json override`
12. `TestFormatIntegration_ErrorsRemainPlainTextStderr`
    - `it writes errors to stderr as plain text regardless of format` (table: --toon, --pretty, --json)
13. `TestFormatIntegration_FormatterResolvedOnceInDispatcher`
    - `it stores formatter on App after Run resolves format`
    - `it uses ToonFormatter for non-TTY`
    - `it uses PrettyFormatter with --pretty`
    - `it uses JSONFormatter with --json`

**Test coverage gaps:**

V2 is MISSING:
- Error output testing (no test that errors remain plain text on stderr regardless of format)
- `dep rm` format testing (only tests `dep add`)
- `--quiet` for update is tested but does not verify absence of format labels (only checks output = ID)

V4 is MISSING:
- Update in Pretty format (table only includes toon + json, skips pretty)
- Show in Pretty format (tests TOON and JSON, not Pretty)
- Init in Pretty format (tests TOON and JSON, not Pretty)
- `dep rm` in TOON format (only tests dep rm in JSON)
- `dep add` in Pretty format (only tests TOON and JSON)
- Transition in Pretty format (combined TOON/Pretty into one test but doesn't test Pretty flag explicitly)

V4 has BETTER coverage for:
- Error handling (explicit stderr plain text test across all formats)
- Formatter type assertions (verifies concrete formatter type matches expected)
- `--quiet + --json` for create (verifies no JSON wrapping)
- `dep rm` tested at all (V2 only tests `dep add`)

V2 has BETTER coverage for:
- Systematic format coverage (table-driven across all 3 formats for every command)
- Pretty format coverage (included in every table)

**Test setup differences:**

V2 uses `setupInitializedTickDir(t)`, `setupTickDirWithContent(t, content)`, and helper functions like `openTaskJSONL(id)`, `twoOpenTasksJSONL()` that create JSONL files on disk.

V4 uses `setupInitializedDir(t)`, `setupInitializedDirWithTasks(t, tasks)` that accept `[]task.Task` slices. V4's setup is slightly more type-safe because it uses Go structs rather than raw JSONL strings.

**Assertion quality:**

V2 uses `strings.Contains` checks and `json.Unmarshal` for JSON validation. V4 does the same. Both check for format-specific markers (e.g., `task{` for TOON, `ID:` for Pretty, valid JSON for JSON format). Neither version uses deep equality assertions.

V4's empty list test is more precise, checking exact string equality:
```go
if output != "tasks[0]{id,title,status,priority}:" {
    t.Errorf("expected TOON empty list format, got %q", output)
}
```

V2 uses looser contains check:
```go
if !strings.Contains(output, "tasks[0]") {
    t.Errorf("toon empty list should contain tasks[0] header, got:\n%s", output)
}
```

**Existing test updates:**

Both versions update existing tests from Phases 1-3 to account for format changes. V2 adds `--pretty` flags to existing tests that expected the old hardcoded format (e.g., `blocked_test.go`, `list_test.go`, `ready_test.go`, `show_test.go`). V4 does the same but also updates assertion expectations to be format-agnostic where the default format changed from Pretty-like to TOON (e.g., changing `"Blocked by:"` to `"blocked_by"`).

V4 touches 23 files vs V2's 15, partly because V4 updates more existing test files to account for TOON being the new default (V2 takes the shortcut of adding `--pretty` flags to force the old format in pre-existing tests). V4's approach of updating assertions to be TOON-aware is arguably more correct since those tests now verify the actual default behavior.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 15 | 23 |
| Lines added | 930 | 990 |
| Lines deleted | 126 | 248 |
| Integration test LOC | 832 | 774 |
| Top-level test functions | 1 | 13 |
| Total subtests | 29 | 39 |
| Existing tests modified | 4 files | 8 files |

## Verdict

**V4 is the better implementation**, though both are competent.

The primary differentiator is **error output testing**: V4's `TestFormatIntegration_ErrorsRemainPlainTextStderr` explicitly validates acceptance criterion #6 (errors remain plain text stderr) across all three format flags. V2 has no equivalent test, leaving this criterion verified only by code inspection.

V4's formatter type assertions in `TestFormatIntegration_FormatterResolvedOnceInDispatcher` are also stronger evidence for criterion #7 -- they verify the concrete type (`*ToonFormatter`, `*PrettyFormatter`, `*JSONFormatter`) rather than just checking `!= nil`.

V4 more thoroughly updates existing tests to work with the new TOON default format, rather than V2's approach of adding `--pretty` flags to force the old behavior. V4's approach means the existing tests actually validate the real default output format.

However, V4 has notable test gaps for Pretty format coverage (missing pretty tests for update, show, init, dep add, dep rm). V2's table-driven approach systematically covers all three formats for every command type, which is more thorough in that dimension.

V4's decision to pass `quiet` into `FormatDepChange` and `FormatTaskList` creates an inconsistency: some commands check quiet before calling the formatter, others delegate it. V2 is more consistent -- quiet is always handled in the command handler, and formatters are never called in quiet mode. This is cleaner separation of concerns.

V4's `taskToDetail()` approach for create/update is simpler and more efficient (no extra DB query) but produces less rich output (no blocked_by/children details). V2's `queryShowData()` approach ensures create/update output matches show output exactly, which better matches the spec's "create/update: FormatTaskDetail (same as show)."

On balance, V4 wins on error testing, formatter resolution testing, existing test correctness, and file scope thoroughness. V2 wins on format coverage systematicity and architectural consistency of quiet handling. The error testing gap is a more significant deficiency than the missing Pretty variants, making V4 the stronger implementation overall.
