# Task 4-5: Integrate formatters into all commands

## Task Plan Summary

Replace all hardcoded output from Phases 1-3 with the resolved Formatter. Every command produces format-aware output driven by TTY detection and flag overrides. `--quiet` overrides format entirely. The formatter is resolved once in the dispatcher, not per-command. Specific requirements:

- **create/update**: `FormatTaskDetail` (same as show). `--quiet`: ID only.
- **start/done/cancel/reopen**: `FormatTransition`. `--quiet`: nothing.
- **dep add/rm**: `FormatDepChange`. `--quiet`: nothing.
- **list** (with filters): `FormatTaskList`. Empty handled per format. `--quiet`: IDs only.
- **show**: `FormatTaskDetail`. `--quiet`: ID only.
- **init/rebuild**: `FormatMessage`. `--quiet`: nothing.
- Errors remain plain text to stderr regardless of format.

Nine test categories are specified, covering each command type in each format, quiet overrides, empty list per format, TTY auto-detection defaults, and flag overrides.

---

## V4 Implementation

### Architecture & Design

V4 uses a method-receiver `App` struct pattern. The `App` struct gains a `Formatter` field (type `Formatter` interface) set once in `App.Run()`:

```go
// cli.go
type App struct {
    // ...
    Formatter Formatter
}

func (a *App) Run(args []string) int {
    // ...
    a.Formatter = resolveFormatter(a.OutputFormat)
    // ...
}
```

The `resolveFormatter` function is added to `format.go` and maps `Format` to concrete types:

```go
func resolveFormatter(f Format) Formatter {
    switch f {
    case FormatPretty:
        return &PrettyFormatter{}
    case FormatJSON:
        return &JSONFormatter{}
    default:
        return &ToonFormatter{}
    }
}
```

**Formatter interface** uses primitive parameters for several methods:

```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatMessage(w io.Writer, msg string) error
```

Notable: `FormatTaskList` and `FormatDepChange` accept `quiet bool` directly, pushing quiet-handling responsibility into the formatter. `FormatTransition` takes bare strings rather than a struct.

**Command wiring** is straightforward. Each command handler (e.g., `runCreate`, `runList`, `runTransition`) replaces its hardcoded `fmt.Fprintf` calls with `a.Formatter.FormatXxx(...)`. For example, `create.go` removes the old `printTaskDetails` method and adds a `taskToDetail` converter function:

```go
func taskToDetail(t *task.Task) TaskDetail {
    detail := TaskDetail{
        ID: t.ID, Title: t.Title, Status: string(t.Status),
        Priority: t.Priority, Description: t.Description,
        Parent: t.Parent,
        Created: t.Created.Format("2006-01-02T15:04:05Z"),
        Updated: t.Updated.Format("2006-01-02T15:04:05Z"),
    }
    if t.Closed != nil {
        detail.Closed = t.Closed.Format("2006-01-02T15:04:05Z")
    }
    return detail
}
```

This uses a **value type** (`TaskDetail`) with exported fields, and the `show.go` handler manually builds the same `TaskDetail` struct from its query results, including `ParentTitle` support.

V4 also added `ParentTitle` to `TaskDetail`, `RelatedTask`, the JSON formatter struct, the TOON formatter, and the Pretty formatter -- all in this single commit. The TOON formatter was updated to emit `parent_title` as a separate field in the schema.

The old `renderListOutput` helper is removed from `list.go` since formatters now handle it.

### Code Quality

- **Error handling**: `FormatMessage` returns `error`, which is consistent with Go conventions. The `init.go` handler properly returns the error from `a.Formatter.FormatMessage(...)`.
- **Naming**: `taskToDetail` is clear. `resolveFormatter` is descriptive. Method receiver on `App` is consistently `a`.
- **Dead code removal**: Old `printTaskDetails` method and `renderListOutput` function are cleanly removed.
- **Formatter interface leak**: `quiet bool` is embedded in `FormatTaskList` and `FormatDepChange` interface methods. This mixes presentation concerns (quiet suppression) with formatting concerns. The formatter should not need to know about quiet -- that is a command-level decision. The `list.go` delegates quiet to the formatter: `a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)`, but `transition.go` handles quiet at the call site: `if a.Quiet { return nil }`. This inconsistency is a design flaw.
- **Data type reuse**: `listRow` (unexported) is used directly in the `Formatter` interface, coupling the internal list data structure to the public formatting contract. This means any change to `listRow` fields propagates to all formatters.

### Test Coverage

V4 creates `format_integration_test.go` with **774 lines** containing comprehensive integration tests:

1. `TestFormatIntegration_CreateFormatsAsTaskDetail` -- TOON, Pretty, JSON
2. `TestFormatIntegration_UpdateFormatsAsTaskDetail` -- TOON, JSON (table-driven)
3. `TestFormatIntegration_TransitionsInEachFormat` -- TOON/Pretty (plain text) and JSON (structured)
4. `TestFormatIntegration_DepConfirmationsInEachFormat` -- dep add TOON, dep add JSON, dep rm JSON
5. `TestFormatIntegration_ListShowInEachFormat` -- list TOON/Pretty/JSON, show TOON/JSON
6. `TestFormatIntegration_InitRebuildInEachFormat` -- init TOON, init JSON
7. `TestFormatIntegration_QuietOverridePerCommandType` -- create, transition, dep, init, list, show, update (7 subtests)
8. `TestFormatIntegration_QuietPlusJsonQuietWins` -- transition and create with --quiet + --json
9. `TestFormatIntegration_EmptyListPerFormat` -- TOON zero-count, Pretty message, JSON empty array
10. `TestFormatIntegration_DefaultsToonWhenPiped` -- non-TTY buffer
11. `TestFormatIntegration_FlagOverrides` -- --toon, --pretty, --json
12. `TestFormatIntegration_ErrorsRemainPlainTextStderr` -- all three formats
13. `TestFormatIntegration_FormatterResolvedOnceInDispatcher` -- Formatter field set, type assertions

Test structure uses individual `t.Run` subtests. The `UpdateFormatsAsTaskDetail` test uses a table-driven approach with a slice of struct test cases. The quiet tests are thorough, covering all command types individually.

**Existing test updates**: V4 updates `blocked_test.go`, `list_test.go`, `ready_test.go`, `show_test.go`, and `update_test.go` to match the new TOON default format instead of the old hardcoded Pretty-like output. For example, empty list tests change from `"No tasks found."` to `strings.Contains(output, "tasks[0]")`.

### Spec Compliance

| Requirement | Status |
|---|---|
| All commands output via Formatter | YES -- create, update, show, list, ready, blocked, start/done/cancel/reopen, dep add/rm, init, stats, rebuild all wired |
| --quiet overrides per spec | YES -- ID for mutations, nothing for transitions/deps/messages |
| Empty list correct per format | YES -- TOON zero-count, Pretty "No tasks found.", JSON `[]` |
| TTY auto-detection end-to-end | YES -- tested with bytes.Buffer (non-TTY) |
| Flag overrides work | YES -- --toon, --pretty, --json all tested |
| Errors remain plain text stderr | YES -- tested across all three formats |
| Format resolved once in dispatcher | YES -- `a.Formatter = resolveFormatter(a.OutputFormat)` in `Run()` |

### golang-pro Skill Compliance

| Rule | Status |
|---|---|
| Handle all errors explicitly | PARTIAL -- `FormatMessage` returns `error` (good), but `FormatTransition` and some `fmt.Fprintf` return values are unchecked in formatters |
| Write table-driven tests | PARTIAL -- `UpdateFormatsAsTaskDetail` uses table-driven, but most tests are individual subtests |
| Document all exported functions | YES -- `resolveFormatter`, `TaskDetail`, `RelatedTask`, `FormatConfig` all have doc comments |
| Propagate errors with fmt.Errorf("%w") | N/A for this task (no new error wrapping introduced) |
| No panic for error handling | YES |
| No hardcoded config | YES -- format resolved from flags |

---

## V5 Implementation

### Architecture & Design

V5 uses a `Context` struct pattern (not method-receiver on `App`). The `Fmt` field is added to `Context`:

```go
type Context struct {
    WorkDir string
    Stdout  io.Writer
    Stderr  io.Writer
    Quiet   bool
    Verbose bool
    Format  OutputFormat
    Fmt     Formatter // resolved once in dispatcher from Format
    Args    []string
}
```

The formatter is resolved in `parseArgs`:

```go
ctx.Format = format
ctx.Fmt = newFormatter(format)
```

**Formatter interface** uses typed struct pointers for complex methods:

```go
FormatTaskList(w io.Writer, rows []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, data *TransitionData) error
FormatDepChange(w io.Writer, data *DepChangeData) error
FormatStats(w io.Writer, data *StatsData) error
FormatMessage(w io.Writer, msg string)  // no error return
```

Key differences from V4:
1. **`FormatMessage` returns no error** -- it is `func(...) ` not `func(...) error`. This is simpler but technically swallows any write error.
2. **`FormatTaskList` takes `[]TaskRow` (exported type)** -- not `[]listRow` (unexported). `TaskRow` is a clean data transfer object defined in `toon_formatter.go`.
3. **`FormatTaskDetail` takes `*showData` (unexported pointer)** -- uses the same internal `showData` struct from `show.go` directly. This couples the formatter interface to show's internal data model.
4. **`FormatTransition` and `FormatDepChange` take typed structs** (`*TransitionData`, `*DepChangeData`) instead of bare primitive parameters.
5. **No `quiet` parameter on formatter methods** -- quiet handling is done at the call site in every command handler.

**Typed data structs** are defined in `toon_formatter.go`:

```go
type TaskRow struct {
    ID, Title, Status string
    Priority          int
}

type TransitionData struct {
    ID, OldStatus, NewStatus string
}

type DepChangeData struct {
    Action, TaskID, BlockedByID string
}
```

**Shared formatting functions** are defined in `format.go`:

```go
func formatTransitionText(w io.Writer, data *TransitionData) error { ... }
func formatDepChangeText(w io.Writer, data *DepChangeData) error { ... }
func formatMessageText(w io.Writer, msg string) { ... }
```

These are called by both `ToonFormatter` and `PrettyFormatter`, avoiding duplication.

**Command wiring**: Each handler accesses `ctx.Fmt` and calls the appropriate method. For example, `transition.go` uses a closure pattern:

```go
func runTransition(command string) func(*Context) error {
    return func(ctx *Context) error {
        // ...
        if !ctx.Quiet {
            return ctx.Fmt.FormatTransition(ctx.Stdout, &TransitionData{
                ID:        id,
                OldStatus: string(result.OldStatus),
                NewStatus: string(result.NewStatus),
            })
        }
        return nil
    }
}
```

The `list.go` handler converts `listRow` to `TaskRow` before passing to the formatter:

```go
taskRows := make([]TaskRow, len(rows))
for i, r := range rows {
    taskRows[i] = TaskRow{
        ID: r.id, Title: r.title, Status: r.status, Priority: r.priority,
    }
}
return ctx.Fmt.FormatTaskList(ctx.Stdout, taskRows)
```

A `taskToShowData` helper converts `task.Task` to `*showData` for create/update output.

### Code Quality

- **Error handling**: `FormatMessage` does NOT return error, which violates Go conventions for I/O operations. In `init.go`, the call is `ctx.Fmt.FormatMessage(...)` with no error check possible. If the write fails, the error is silently lost. The `//nolint:errcheck` comment in `JSONFormatter.FormatMessage` confirms the choice was deliberate but still questionable.
- **Naming**: `ctx.Fmt` is terse but readable in context. `newFormatter` is clear. `taskToShowData` is descriptive.
- **Separation of concerns**: Quiet handling is consistently at the call site -- the formatter never sees quiet. This is a cleaner separation than V4.
- **Data coupling issue**: `FormatTaskDetail` takes `*showData`, which is an unexported type with unexported fields. This means the formatter implementations must live in the same package, and the interface signature uses a concrete internal type. This is weaker than V4's exported `TaskDetail` struct.
- **Shared text functions**: `formatTransitionText`, `formatDepChangeText`, `formatMessageText` in `format.go` eliminate duplication between Toon and Pretty formatters for plain-text operations. This is cleaner than V4 where each formatter has its own copy.
- **Conversion layer**: The `listRow -> TaskRow` conversion in `list.go` adds a few lines of mapping code but provides cleaner decoupling between internal query results and the formatter interface.

### Test Coverage

V5 creates `formatter_integration_test.go` with **707 lines** containing comprehensive integration tests:

1. Create formatted as task detail in toon, pretty, json (3 subtests)
2. Update formatted as task detail in each format (table-driven with 3 entries)
3. Transitions in toon (plain text) and json (structured) (2 subtests)
4. Dep add/rm in toon and json (3 subtests)
5. List in toon, pretty, json (3 subtests)
6. Show in toon, json (2 subtests)
7. Init/rebuild in toon, json (2 subtests)
8. Quiet override for create, update, show, transitions, dep, list, init (7 subtests)
9. Empty list in toon (zero count), pretty (message), json (empty array) (3 subtests)
10. TTY auto-detection: defaults to toon when non-TTY, pretty when TTY (2 subtests)
11. Flag overrides: --toon overrides TTY, --pretty overrides non-TTY, --json (3 subtests)
12. Errors remain plain text stderr (table-driven, 3 subtests)
13. Formatter resolved in Context: JSONFormatter, ToonFormatter (non-TTY), PrettyFormatter (TTY) (3 subtests)

All tests in V5 use the `Run()` function (the top-level entry point), making them true integration tests. The last group also tests `parseArgs` directly for formatter type assertions.

**Existing test updates**: V5 modifies `blocked_test.go`, `list_test.go`, `ready_test.go`, `show_test.go`, `update_test.go`, and `parent_scope_test.go` to add `--pretty` flags to commands that previously relied on the old hardcoded Pretty-like format. This is a different strategy from V4 -- V5 pins existing tests to `--pretty` to preserve exact output expectations, while V4 updates assertions to match TOON format.

### Spec Compliance

| Requirement | Status |
|---|---|
| All commands output via Formatter | YES -- all commands wired via `ctx.Fmt` |
| --quiet overrides per spec | YES -- ID for mutations, nothing for transitions/deps/messages |
| Empty list correct per format | YES -- TOON zero-count, Pretty "No tasks found.", JSON `[]` |
| TTY auto-detection end-to-end | YES -- tested with `isTTY` parameter |
| Flag overrides work | YES -- all three formats tested |
| Errors remain plain text stderr | YES -- tested across all formats |
| Format resolved once in dispatcher | YES -- `ctx.Fmt = newFormatter(format)` in `parseArgs()` |

### golang-pro Skill Compliance

| Rule | Status |
|---|---|
| Handle all errors explicitly | NO -- `FormatMessage` returns no error, suppressing write failures. `//nolint:errcheck` in JSONFormatter |
| Write table-driven tests | PARTIAL -- update test uses table-driven; errors test iterates formats in loop; most are individual subtests |
| Document all exported functions | YES -- `Formatter`, `FormatConfig`, `TaskRow`, `TransitionData`, `DepChangeData`, `StatsData`, `newFormatter`, `ResolveFormat` |
| Propagate errors with fmt.Errorf("%w") | N/A |
| No panic for error handling | YES |
| No hardcoded config | YES |

---

## Comparative Analysis

### Where V4 is Better

1. **`FormatMessage` returns `error`**: V4's `FormatMessage(w io.Writer, msg string) error` follows Go conventions for I/O operations. V5's void return silently swallows write errors and uses `//nolint:errcheck`, which violates the golang-pro MUST DO rule "Handle all errors explicitly."

2. **Exported data types in Formatter interface**: V4 uses `TaskDetail` (exported struct with exported fields) and `RelatedTask` (exported) in the `Formatter` interface. This makes the interface usable from other packages and is more idiomatic Go. V5 uses `*showData` (unexported type with unexported fields) for `FormatTaskDetail`, which couples the formatter interface to the `cli` package's internals.

3. **More comprehensive test file**: V4's integration test file is 774 lines versus V5's 707 lines. V4 includes Pretty format testing for `create` and `show` that V5 omits from the dedicated integration test (V5 only has toon and json for show in the integration test, relying on existing tests updated with `--pretty` for Pretty coverage).

4. **Existing test migration strategy**: V4 updates test assertions to match the new TOON default format (e.g., changing `"No tasks found."` to `tasks[0]` checks). This means the existing tests verify the actual default behavior. V5 adds `--pretty` flags to existing tests to preserve their original assertions, which means those tests no longer verify the default non-TTY behavior.

### Where V5 is Better

1. **Typed data structs for formatter methods**: V5 uses `*TransitionData`, `*DepChangeData` structs instead of V4's bare primitives (`id string, oldStatus string, newStatus string`). This is more maintainable -- adding a field to a transition output only requires changing the struct, not every formatter method signature.

2. **Clean quiet separation**: V5 handles `--quiet` entirely at the call site in every command handler. The formatter interface has no knowledge of quiet. V4 leaks quiet into `FormatTaskList` and `FormatDepChange` signatures, creating inconsistency (some methods handle quiet internally, others externally).

3. **`TaskRow` exported type decouples list format from internal query**: V5 defines `TaskRow` as an explicit exported DTO and converts `listRow` to `TaskRow` before calling the formatter. V4 passes `[]listRow` (unexported) directly to the formatter interface, coupling the interface to the internal data model.

4. **Shared text formatting helpers**: V5 extracts `formatTransitionText`, `formatDepChangeText`, `formatMessageText` as shared functions in `format.go`. This eliminates duplication between Toon and Pretty formatters for identical plain-text rendering. V4 has each formatter implement the identical plain-text logic independently.

5. **TTY end-to-end testing**: V5 tests both `isTTY=true` (defaults to Pretty) and `isTTY=false` (defaults to TOON) in its integration tests. V4 only tests non-TTY (bytes.Buffer) in its integration test for the default case.

6. **`parseArgs` direct testing**: V5 tests `parseArgs` directly to verify formatter type assignment (JSONFormatter, ToonFormatter, PrettyFormatter), giving targeted coverage of the resolution logic separate from full command execution.

7. **Command dispatch via map**: V5 uses `var commands = map[string]func(*Context) error{...}` for dispatch, which is more extensible and avoids the large switch statement in V4's `Run()`. (This is pre-existing architecture, not introduced in this task, but it means V5's integration is cleaner.)

### Differences That Are Neutral

1. **`App` struct (V4) vs `Context` struct (V5)**: Both approaches are valid. V4 stores the Formatter on the long-lived App. V5 stores it on the per-invocation Context. Both achieve "resolved once in dispatcher."

2. **Test structure**: Both use subtests with descriptive names. Both cover all nine test categories from the spec. The difference in line count (774 vs 707) is mostly due to V5's slightly more concise test helper usage.

3. **`ParentTitle` support**: Both add `ParentTitle` to their task detail data structures and update TOON, Pretty, and JSON formatters. V4 adds it as `parent_title` in TOON schema; V5 handles it similarly through `showData.parentTitle`.

4. **Existing `printTaskDetails` / `printShowDetails` removal**: Both cleanly remove the old hardcoded output functions and replace them with formatter calls.

5. **`FormatConfig` struct**: Both define a `FormatConfig` struct, though it is not directly used in this task's changes (it was added in a prior task).

---

## Verdict

**V5 is the better implementation**, though the margin is narrow.

The decisive factors:

1. **Interface design**: V5's use of typed structs (`*TransitionData`, `*DepChangeData`, `*showData`, `[]TaskRow`) for formatter methods is fundamentally better API design than V4's mix of bare primitives. When the transition output needs a new field (e.g., task title), V5 changes the struct; V4 must update every formatter method signature plus every call site. This is the most architecturally significant difference.

2. **Clean quiet separation**: V5 consistently handles quiet at the command level, keeping formatters pure. V4 mixes the concern by passing `quiet bool` into some formatter methods but not others. This inconsistency makes V4's formatter interface harder to reason about and harder to extend.

3. **Shared text helpers**: V5's `formatTransitionText`, `formatDepChangeText`, `formatMessageText` eliminate real code duplication between Toon and Pretty formatters, following DRY principles without adding abstraction complexity.

4. **`TaskRow` decoupling**: V5's explicit conversion from `listRow` to `TaskRow` is a small cost in code but a significant gain in decoupling the formatter interface from internal data structures.

**V4's one significant advantage** -- `FormatMessage` returning `error` -- is a real golang-pro compliance issue in V5. Silently swallowing write errors is wrong. However, in practice, `FormatMessage` only runs for `init` and `rebuild` output, where a write failure to stdout would be caught by the OS-level exit, and the error would manifest as truncated output. This is a real but low-severity flaw.

V4's other advantage -- exported `TaskDetail` vs unexported `*showData` -- is noteworthy but less impactful since all formatters live in the same `cli` package regardless.

Overall, V5's superior interface design, consistent quiet handling, and DRY helper extraction outweigh V4's better error handling on `FormatMessage`. V5 produces a formatter interface that is easier to extend, easier to test in isolation, and more maintainable long-term.
