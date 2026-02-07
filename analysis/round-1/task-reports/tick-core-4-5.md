# Task tick-core-4-5: Integrate Formatters into All Commands

## Task Summary

Replace all hardcoded output from Phases 1-3 with the resolved Formatter. Every command produces format-aware output driven by TTY detection and flag overrides. `--quiet` overrides format entirely.

**Implementation requirements:**
- Wire `FormatConfig` + `Formatter` into every handler via CLI dispatcher
- create/update: `FormatTaskDetail` (same as show). `--quiet`: ID only.
- start/done/cancel/reopen: `FormatTransition`. `--quiet`: nothing.
- dep add/rm: `FormatDepChange`. `--quiet`: nothing.
- list (with filters): `FormatTaskList`. Empty handled per format. `--quiet`: IDs only.
- show: `FormatTaskDetail`. `--quiet`: ID only.
- init/rebuild: `FormatMessage`. `--quiet`: nothing.
- Format resolved once in dispatcher, not per-command. Errors remain plain text to stderr.

**Acceptance Criteria:**
1. All commands output via Formatter
2. --quiet overrides per spec (ID for mutations, nothing for transitions/deps/messages)
3. Empty list correct per format
4. TTY auto-detection end-to-end
5. Flag overrides work for all commands
6. Errors remain plain text stderr
7. Format resolved once in dispatcher

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| All commands output via Formatter | PASS - create, update, show, list, ready, blocked, transition, dep, init all use `a.fmtr.Format*()` | PASS - all commands use `a.formatter.Format*()` | PARTIAL - most commands use `a.formatConfig.Formatter()` but calls `Formatter()` per command, creating new instances each time |
| --quiet overrides per spec | PASS - create/update: ID only; transitions: nothing; dep: nothing; list: IDs; init: nothing | PASS - all quiet behaviors verified: create/update ID, transitions nothing, dep nothing, list IDs, init nothing, show ID | PASS - same quiet behaviors: create/update ID, transitions nothing, dep nothing, list IDs, init nothing, show ID |
| Empty list correct per format | PASS - test verifies TOON `tasks[0]`, Pretty `No tasks found.`, JSON `[]` | PASS - test verifies all three empty formats with detailed assertions | PASS - test verifies all three empty formats |
| TTY auto-detection end-to-end | PASS - `NewApp` checks `stdout.(*os.File)` + `DetectTTY(f)`, stores `isTTY` on App | PASS - `FormatConfig` resolved via `a.formatConfig()` which uses stdout type | PASS - `FormatConfig` resolved with TTY detection |
| Flag overrides work for all commands | PASS - tested --toon, --pretty, --json overrides | PASS - tested --toon, --pretty, --json overrides per command | PASS - tested --toon, --pretty, --json overrides per command |
| Errors remain plain text stderr | PASS - implicit (errors use `fmt.Fprintf(a.stderr, ...)`) | PASS - implicit in error handling paths | PASS - implicit in error handling paths |
| Format resolved once in dispatcher | PASS - `ResolveFormat` + `newFormatter` called once in `Run()` before switch dispatch | PASS - `a.formatter = newFormatter(a.FormatCfg.Format)` called once in `Run()` | FAIL - `a.formatConfig.Formatter()` called per command handler, creating a new formatter instance each time |

## Implementation Comparison

### Approach

All three versions share the same fundamental strategy: remove hardcoded `fmt.Fprintf` output from command handlers and replace with calls through the `Formatter` interface. They differ significantly in where the formatter is instantiated and how data flows to it.

**V1: Formatter stored on App, resolved once in `Run()`**

V1 adds two fields to the `App` struct in `cli.go`:

```go
type App struct {
    stdout io.Writer
    stderr io.Writer
    opts   GlobalOpts
    isTTY  bool
    fmtr   Formatter
}
```

TTY detection happens in `NewApp`:
```go
func NewApp(stdout, stderr io.Writer) *App {
    isTTY := false
    if f, ok := stdout.(*os.File); ok {
        isTTY = DetectTTY(f)
    }
    return &App{stdout: stdout, stderr: stderr, isTTY: isTTY}
}
```

Format resolution happens once in `Run()`:
```go
format, err := ResolveFormat(a.opts.Toon, a.opts.Pretty, a.opts.JSON, a.isTTY)
if err != nil {
    fmt.Fprintf(a.stderr, "Error: %s\n", err)
    return 1
}
a.fmtr = newFormatter(format)
```

Commands then use `a.fmtr` directly. V1 also creates a shared helper `queryAndFormatTaskDetail` for show/create/update to share the same query+format path:
```go
func (a *App) queryAndFormatTaskDetail(store *storage.Store, taskID string) error {
    // ...query task detail...
    return a.fmtr.FormatTaskDetail(a.stdout, detail)
}
```

**V2: Formatter stored on App, resolved once in `Run()`**

V2 adds a `formatter` field to the `App` struct in `app.go`:
```go
type App struct {
    config    Config
    FormatCfg FormatConfig
    formatter Formatter
    // ...
}
```

Resolution is done once in `Run()`:
```go
a.FormatCfg = a.formatConfig()
a.formatter = newFormatter(a.FormatCfg.Format)
```

The `newFormatter` function is added at the bottom of `app.go`:
```go
func newFormatter(format OutputFormat) Formatter {
    switch format {
    case FormatPretty:
        return &PrettyFormatter{}
    case FormatJSON:
        return &JSONFormatter{}
    default:
        return &ToonFormatter{}
    }
}
```

V2 also extracts `queryShowData` as a reusable function in `show.go`:
```go
func queryShowData(store *storage.Store, lookupID string) (*showData, error) {
```

Create and update use this function to query full show data before passing it to the formatter. V2 adds a fallback pattern in create/update where if the query fails, it prints just the ID:
```go
data, err := queryShowData(store, createdTask.ID)
if err != nil {
    fmt.Fprintln(a.stdout, createdTask.ID)
    return nil
}
return a.formatter.FormatTaskDetail(a.stdout, data)
```

**V3: Formatter created per command via `FormatConfig.Formatter()` method**

V3 takes a different approach. Instead of storing the formatter on `App`, it adds a `Formatter()` method to `FormatConfig` in `format.go`:
```go
func (c FormatConfig) Formatter() Formatter {
    switch c.Format {
    case FormatJSON:
        return &JSONFormatter{}
    case FormatPretty:
        return &PrettyFormatter{}
    default:
        return &ToonFormatter{}
    }
}
```

Each command handler calls `a.formatConfig.Formatter()` independently:
```go
// In show.go:
formatter := a.formatConfig.Formatter()
fmt.Fprint(a.Stdout, formatter.FormatTaskDetail(data))

// In list.go:
formatter := a.formatConfig.Formatter()
fmt.Fprint(a.Stdout, formatter.FormatTaskList(data))

// In transition.go:
formatter := a.formatConfig.Formatter()
fmt.Fprint(a.Stdout, formatter.FormatTransition(...))
```

This means a new formatter is instantiated per command invocation, violating the spec's "format resolved once in dispatcher" requirement.

V3 also uses different Formatter interface semantics -- the formatters return strings rather than writing to an `io.Writer`, and the caller does `fmt.Fprint(a.Stdout, formatter.FormatTaskDetail(data))`. V1 and V2 pass `io.Writer` directly to the formatter methods.

### Data Structure Differences

The three versions use different DTOs to pass data to formatters:

**V1** uses types like `TaskDetail`, `RelatedTask`, `TaskListItem`, `TransitionData`, `DepChangeData`:
```go
detail := TaskDetail{
    ID: td.ID, Title: td.Title, Status: td.Status, Priority: td.Priority,
    Description: td.Description, Created: td.Created, Updated: td.Updated, Closed: td.Closed,
}
detail.BlockedBy = make([]RelatedTask, len(blockers))
detail.Children = make([]RelatedTask, len(children))
```

**V2** uses `showData` (existing struct) directly via `queryShowData`:
```go
data, err := queryShowData(store, createdTask.ID)
return a.formatter.FormatTaskDetail(a.stdout, data)
```

**V3** uses `TaskDetailData`, `RelatedTaskData`, `TaskListData`, `TaskRowData`:
```go
data := &TaskDetailData{
    ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority,
    Description: t.Description, Parent: t.Parent, ParentTitle: t.ParentTitle,
    Created: t.Created, Updated: t.Updated, Closed: t.Closed,
    BlockedBy: make([]RelatedTaskData, len(blockedBy)),
    Children:  make([]RelatedTaskData, len(children)),
}
```

### Code Quality

**V1: Clean separation, DRY helpers**

V1 is the most concise implementation at 463 insertions / 238 deletions. It achieves DRYness through `queryAndFormatTaskDetail` and `buildAndFormatTaskDetail` -- two helper methods that share the show query path for create, update, and show commands. The formatter is referenced via `a.fmtr` which is clean and idiomatic. The `newFormatter` function in `format.go` is placed logically next to the format resolution logic.

V1's `extractID` helper in `create_test.go` was also updated to handle multiple output formats, using `strings.FieldsFunc` for more robust parsing:
```go
parts := strings.FieldsFunc(line, func(r rune) bool {
    return r == ' ' || r == ',' || r == ':' || r == '\t' || r == '"'
})
```

**V2: Clean separation, graceful fallback**

V2 adds the `newFormatter` function in `app.go` which is slightly less logical than V1's placement in `format.go`, but still clean. V2's unique contribution is the graceful fallback pattern in create/update:
```go
data, err := queryShowData(store, createdTask.ID)
if err != nil {
    fmt.Fprintln(a.stdout, createdTask.ID)
    return nil
}
```

This is defensive programming -- if the show query fails after creation, the user still gets the task ID. V1 does not have this fallback. V2 also cleanly extracts `queryShowData` as a package-level function (not a method on App), which is more testable.

**V3: Repeated formatter creation, string-return pattern**

V3 has a structural issue: `a.formatConfig.Formatter()` is called in every command handler, creating a new formatter each time. For example, `list.go`, `blocked.go`, `ready.go` each contain:
```go
formatter := a.formatConfig.Formatter()
fmt.Fprint(a.Stdout, formatter.FormatTaskList(data))
```

This is repetitive (6+ call sites) and violates the spec requirement that "format resolved once in dispatcher." The formatters are stateless so it doesn't cause bugs, but it's architecturally wrong and more verbose.

V3's formatters return strings instead of writing to `io.Writer`, which means the caller must wrap in `fmt.Fprint()`. This is less idiomatic for Go I/O -- V1 and V2 pass `io.Writer` to the formatter, which is more composable and avoids intermediate string allocation.

V3 also does not reuse a shared query function for create/update -- the `show.go` changes build the `TaskDetailData` struct inline in each command handler (`create.go`, `update.go`, `show.go`), leading to duplicated struct-building code.

### Error Handling

V1 uses `return a.fmtr.FormatTransition(a.stdout, TransitionData{...})` which returns the error from the formatter. V2 uses `return a.formatter.FormatTransition(a.stdout, id, oldStatus, newStatus)` similarly. V3 uses `fmt.Fprint(a.Stdout, formatter.FormatTransition(...))` which silently discards the write error (returns int, error from Fprint are ignored).

### Test Quality

**V1 Integration Tests (226 LOC, 1 top-level function, 15 subtests):**

File: `internal/cli/format_integration_test.go`

Test functions:
1. `"it defaults to TOON when piped (non-TTY)"` -- verifies `tasks[` prefix for non-TTY stdout
2. `"it respects --pretty override"` -- checks for column headers `ID`, `STATUS`
3. `"it respects --json override"` -- validates JSON with `json.Valid()`
4. `"it respects --toon override"` -- checks `tasks[` prefix
5. `"it formats create as full task detail in TOON"` -- checks `task{` presence
6. `"it formats create as JSON with --json"` -- validates JSON
7. `"it formats transition in each format"` -- tests TOON (arrow `\u2192`) and JSON (valid JSON) for start+done
8. `"it formats show in TOON"` -- checks `task{`
9. `"it applies --quiet override for create (ID only)"` -- verifies `tick-` prefix, len 11
10. `"it applies --quiet override for transition (nothing)"` -- checks empty stdout
11. `"it applies --quiet override for list (IDs only)"` -- checks each line starts with `tick-`
12. `"it handles empty list per format"` -- TOON: `tasks[0]`, Pretty: `No tasks found.`, JSON: `[]`
13. `"it formats dep confirmations in TOON"` -- checks `Dependency added`
14. `"it formats init message in TOON"` -- checks `Initialized tick`
15. `"it errors on conflicting format flags"` -- exit code 1, `only one` in stderr

V1 also updated existing test files:
- `blocked_test.go`: Updated empty list check from `No tasks found.` to `tasks[0]`
- `list_filter_test.go`: Same empty list update
- `list_show_test.go`: Extensive updates -- removed format-specific assertions (e.g., `ID` header), updated to TOON expectations (`tasks[0]`, `blocked_by[1]`, `children[1]`)
- `ready_test.go`: Updated empty list check
- `update_test.go`: Updated `ID:` check to `task{` check
- `create_test.go`: Updated `extractID` helper for multi-format parsing

**V2 Integration Tests (832 LOC, 1 top-level function, 21 unique subtests with table-driven sub-subtests):**

File: `internal/cli/format_integration_test.go`

Test functions:
1. `"it formats init as message in each format"` -- table-driven: toon (plain text), pretty (plain text), json (`{"message":"..."}`)
2. `"it formats create as full task detail in each format"` -- table-driven: toon (`task{`, `blocked_by[`, `children[`), pretty (`ID:`, `Title:`, `Created:`), json (keys: `id`, `title`, `blocked_by`)
3. `"it formats update as full task detail in each format"` -- table-driven: toon (`task{`), pretty (`ID:`), json (verifies `title` == `"Updated title"`)
4. `"it formats transitions in each format"` -- table-driven: toon/pretty (task ID + arrow), json (struct with `id`, `from`, `to` fields)
5. `"it formats dep confirmations in each format"` -- table-driven: toon/pretty (`Dependency added:`), json (`action`, `task_id`, `blocked_by` fields)
6. `"it formats list in each format"` -- table-driven: toon (`tasks[2]`), pretty (headers), json (array of 2)
7. `"it formats show in each format"` -- table-driven: toon (`task{`), pretty (`ID:       tick-aaa111`), json (`id` key)
8. `"it applies --quiet override for create (ID only)"` -- tick- prefix, len 11
9. `"it applies --quiet override for update (ID only)"` -- exact `tick-aaa111`
10. `"it applies --quiet override for transitions (nothing)"` -- empty stdout
11. `"it applies --quiet override for dep add (nothing)"` -- empty stdout
12. `"it applies --quiet override for init (nothing)"` -- empty stdout
13. `"it applies --quiet override for list (IDs only)"` -- exact `tick-aaa111`
14. `"it applies --quiet override for show (ID only)"` -- exact `tick-aaa111`
15. `"it applies --quiet even when --json is set (quiet wins)"` -- empty stdout for transition
16. `"it handles empty list per format"` -- table-driven: toon (`tasks[0]`), pretty (`No tasks found.`), json (`[]`)
17. `"it defaults to TOON when piped (non-TTY)"` -- `tasks[` in output
18. `"it respects --toon override"` -- `task{` for show
19. `"it respects --pretty override"` -- `ID:       tick-aaa111` for show
20. `"it respects --json override"` -- valid JSON for show
21. `"it resolves format once in dispatcher not per command"` -- `app.formatter != nil` after Run

V2 also updated existing test files:
- `blocked_test.go`: Added `--pretty` flag to 4 tests that check columnar output
- `list_test.go`: Added `--pretty` to 4 tests
- `ready_test.go`: Added `--pretty` to 6 tests
- `show_test.go`: Added `--pretty` to 10 tests

**V3 Integration Tests (637 LOC, 9 top-level functions, 28 unique subtests):**

File: `internal/cli/formatter_integration_test.go`

Test functions:
1. `TestFormatterIntegration_CreateUpdateFormatsTaskDetail` -- table-driven create (toon `task{`, pretty `ID:/Title:`, json `{`+`"id"`), plus separate `"update uses Pretty format with --pretty flag"`
2. `TestFormatterIntegration_TransitionsFormatted` -- table-driven: toon/pretty (ID + arrow), json (`"id"`+`"from"`+`"to"`)
3. `TestFormatterIntegration_DepConfirmationsFormatted` -- table-driven: toon/pretty (`Dependency added`), json (`"action"`+`"task_id"`)
4. `TestFormatterIntegration_ListShowFormatted` -- 4 subtests: list toon/pretty/json, show toon
5. `TestFormatterIntegration_InitRebuildFormatted` -- 2 subtests: init toon (`Initialized`), init json (`"message"`)
6. `TestFormatterIntegration_QuietOverride` -- 7 subtests: create (ID only), update (ID only), start (nothing), dep add (nothing), list (IDs only), init (nothing), show (ID only)
7. `TestFormatterIntegration_EmptyListHandling` -- 3 subtests: toon (`tasks[0]`), pretty (`No tasks found`), json (`[]`)
8. `TestFormatterIntegration_TTYAutoDetection` -- 1 subtest: non-TTY defaults to TOON
9. `TestFormatterIntegration_FormatOverrides` -- 3 subtests: --toon, --pretty, --json force format
10. `TestFormatterIntegration_QuietOverridesJson` -- 2 subtests: create (ID not JSON), transition (nothing)
11. `TestFormatterIntegration_ErrorsAlwaysPlainText` -- 1 subtest: error with --json is plain text stderr

V3 also updated existing test files:
- `blocked_test.go`: Added `--pretty` to 4 tests
- `create_test.go`: Added `--pretty` to 1 test
- `list_show_test.go`: Added `--pretty` to 17 tests
- `list_test.go`: Added `--pretty` to 2 tests
- `ready_test.go`: Added `--pretty` to 4 tests
- `update_test.go`: Added `--pretty` to 1 test

### Test Coverage Comparison

**Tests unique to V1:**
- Conflicting format flags error test (`"it errors on conflicting format flags"`)

**Tests unique to V2:**
- Quiet override for update specifically (`"it applies --quiet override for update (ID only)"`)
- Quiet override for show specifically (`"it applies --quiet override for show (ID only)"`)
- Quiet override for init specifically (`"it applies --quiet override for init (nothing)"`)
- Quiet override for dep add specifically (`"it applies --quiet override for dep add (nothing)"`)
- Format resolved once assertion (`"it resolves format once in dispatcher not per command"` -- checks `app.formatter != nil`)
- Update formatted in each format (TOON/Pretty/JSON)
- Init formatted in each format with structured validation (JSON message struct)

**Tests unique to V3:**
- Errors always plain text to stderr (`TestFormatterIntegration_ErrorsAlwaysPlainText`)
- Multiple top-level test functions (9 vs 1) for better Go test output organization
- Quiet + JSON on create specifically (ID not JSON-wrapped)

**Tests in all three versions:**
- Default TOON on non-TTY
- --toon/--pretty/--json overrides
- Create formatted in TOON/JSON
- Transition formatted in each format
- Dep confirmations formatted
- List formatted in each format
- Show formatted
- Quiet create (ID only)
- Quiet transition (nothing)
- Quiet list (IDs only)
- Empty list per format (TOON/Pretty/JSON)
- Init formatted

**Test gaps:**
- V1 does NOT test quiet for update, show, dep add, or init individually
- V1 does NOT test update formatted output
- V1 does NOT test errors remain plain text stderr
- V2 does NOT test conflicting format flags
- V2 does NOT test errors remain plain text stderr
- V3 does NOT test conflicting format flags
- No version tests `rebuild` command formatting (spec mentions it)
- No version tests `cancel` or `reopen` transitions specifically (only `start`/`done`)

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 15 | 13 | 17 |
| Lines added | 463 | 927 | 789 |
| Lines deleted | 238 | 124 | 117 |
| Net lines | +225 | +803 | +672 |
| Integration test LOC | 226 | 832 | 637 |
| Integration test subtests | 15 | 21 | 28 |
| Top-level test functions | 1 | 1 | 9 |
| Existing tests modified | 6 files | 4 files | 7 files |

## Verdict

**V2 is the best implementation.**

Evidence:

1. **Spec compliance (format resolved once):** V2 resolves the formatter exactly once in `Run()` with `a.formatter = newFormatter(a.FormatCfg.Format)`, matching the spec. V1 does the same. V3 fails this criterion by calling `a.formatConfig.Formatter()` in every command handler.

2. **Code quality:** V2 extracts `queryShowData` as a clean, reusable package-level function and adds a graceful fallback pattern for create/update. V1 is slightly cleaner in formatter placement (`format.go` vs `app.go`) but lacks the fallback. V3 has duplicated struct-building code across command handlers and discards write errors from `fmt.Fprint`.

3. **Test thoroughness:** V2 has the most comprehensive integration test suite at 832 LOC with 21 named subtests. It tests every command type in all three formats via table-driven tests, covers all 7 quiet override scenarios (create, update, show, list, transition, dep, init), tests the `--quiet + --json` edge case, and even includes a structural assertion that `app.formatter != nil` to verify single resolution. V1 is significantly less thorough at 226 LOC with 15 subtests, missing quiet tests for update/show/dep/init and update format tests. V3 is strong at 637 LOC with 28 subtests and uniquely tests errors-as-plain-text, but its 9 top-level functions are less cohesive than V2's single organized function.

4. **Formatter interface design:** V2's formatters write to `io.Writer` (like V1), which is more idiomatic Go than V3's string-return approach. Writing to `io.Writer` avoids intermediate allocations and naturally propagates write errors.

V1 is the most concise but sacrifices test coverage. V3 has good test coverage and uniquely tests the errors-remain-plain-text edge case, but fails the "resolved once" architectural requirement. V2 strikes the best balance of correctness, code quality, and test thoroughness.
