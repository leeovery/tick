# Task tick-core-4-5: Integrate formatters into all commands

## Task Summary

Replace all hardcoded output from Phases 1-3 with the resolved Formatter. Every command produces format-aware output driven by TTY detection and flag overrides. `--quiet` overrides format entirely.

**Implementation requirements:**
- Wire `FormatConfig` + `Formatter` into every handler via CLI dispatcher
- **create/update**: `FormatTaskDetail` (same as show). `--quiet`: ID only.
- **start/done/cancel/reopen**: `FormatTransition`. `--quiet`: nothing.
- **dep add/rm**: `FormatDepChange`. `--quiet`: nothing.
- **list** (with filters): `FormatTaskList`. Empty handled per format. `--quiet`: IDs only.
- **show**: `FormatTaskDetail`. `--quiet`: ID only.
- **init/rebuild**: `FormatMessage`. `--quiet`: nothing.
- Format resolved once in dispatcher, not per-command. Errors remain plain text to stderr.

**Acceptance criteria:**
1. All commands output via Formatter
2. --quiet overrides per spec (ID for mutations, nothing for transitions/deps/messages)
3. Empty list correct per format
4. TTY auto-detection end-to-end
5. Flag overrides work for all commands
6. Errors remain plain text stderr
7. Format resolved once in dispatcher

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| All commands output via Formatter | PASS -- create, update, show, list, transition, dep, init all route through `ctx.Fmt.*` methods | PASS -- create, update, show, list, transition, dep, init all route through `fmtr.*` methods |
| --quiet overrides per spec | PASS -- create/update/show output ID only; transition/dep/init output nothing | PASS -- create/update/show output ID only; transition/dep/init output nothing |
| Empty list correct per format | PASS -- removed hardcoded "No tasks found." and delegates to formatter | PASS -- removed hardcoded "No tasks found." and delegates to formatter |
| TTY auto-detection end-to-end | PASS -- tested with isTTY=false (TOON) and isTTY=true (Pretty) | PASS -- tested with IsTTY=false (TOON) and IsTTY=true (Pretty) via App struct |
| Flag overrides work for all commands | PASS -- tested --toon, --pretty, --json overrides | PASS -- tested --toon, --pretty, --json overrides |
| Errors remain plain text stderr | PASS -- tested with all 3 format flags, error stays plain text | PASS -- tested with --json, error stays plain text |
| Format resolved once in dispatcher | PASS -- resolved in `parseArgs()` and stored in `ctx.Fmt` | PASS -- resolved in `App.Run()` and passed as `fmtr` parameter to handlers |

## Implementation Comparison

### Approach

**V5: Context-embedded Formatter (single-object pattern)**

V5 adds the Formatter as a field on the `Context` struct, resolving it once during `parseArgs()`:

```go
// cli.go lines 33-34
Fmt     Formatter // resolved once in dispatcher from Format
Args    []string  // remaining args after global flags and subcommand
```

```go
// cli.go lines 162-163
ctx.Format = format
ctx.Fmt = newFormatter(format)
```

Handlers access the formatter through `ctx.Fmt`, and quiet/format info through `ctx.Quiet`/`ctx.Format`. The dispatcher uses a command map:

```go
var commands = map[string]func(*Context) error{
    "init":    runInit,
    "create":  runCreate,
    ...
}
```

Each handler is a standalone function receiving `*Context`, keeping signatures minimal:

```go
func runCreate(ctx *Context) error { ... }
func runShow(ctx *Context) error { ... }
```

**V6: Parameter-passing pattern (explicit injection)**

V6 resolves the formatter in `App.Run()` and passes it as separate parameters to every handler:

```go
// app.go lines 30-41
fc, err := NewFormatConfig(flags, a.IsTTY)
...
fmtr := NewFormatter(fc.Format)
```

Handlers receive `FormatConfig` and `Formatter` as explicit arguments:

```go
func (a *App) handleCreate(fc FormatConfig, fmtr Formatter, subArgs []string) error {
    ...
    return RunCreate(dir, fc, fmtr, subArgs, a.Stdout)
}
```

The underlying `Run*` functions are package-level with explicit parameters:

```go
func RunCreate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error
func RunShow(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error
```

**Key Structural Difference: Formatter Interface Signatures**

This is the most significant architectural divergence:

V5 Formatter methods accept `io.Writer` and return `error`:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, rows []TaskRow) error
    FormatTaskDetail(w io.Writer, data *showData) error
    FormatTransition(w io.Writer, data *TransitionData) error
    FormatDepChange(w io.Writer, data *DepChangeData) error
    FormatStats(w io.Writer, data *StatsData) error
    FormatMessage(w io.Writer, msg string)
}
```

V6 Formatter methods return `string`:
```go
type Formatter interface {
    FormatTaskList(tasks []task.Task) string
    FormatTaskDetail(detail TaskDetail) string
    FormatTransition(id string, oldStatus string, newStatus string) string
    FormatDepChange(action string, taskID string, depID string) string
    FormatStats(stats Stats) string
    FormatMessage(msg string) string
}
```

V5's io.Writer-based approach gives formatters direct write control, handles errors from write operations, and avoids intermediate string allocations. V6's string-return approach is simpler to call (just wrap in `fmt.Fprintln`) but requires building the entire output string in memory before writing, and loses write error propagation.

**Data Type Differences**

V5 passes `*showData` (an internal struct with lowercase fields) directly to the formatter:
```go
return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(createdTask))
```

V6 uses a separate `TaskDetail` struct (exported, wrapping `task.Task`) and requires a conversion step (`showDataToTaskDetail`):
```go
detail := showDataToTaskDetail(data)
fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
```

V6 also introduces a `baseFormatter` struct for shared text-based transition/dep formatting:
```go
type baseFormatter struct{}
func (b *baseFormatter) FormatTransition(id string, oldStatus string, newStatus string) string {
    return fmt.Sprintf("%s: %s -> %s", id, oldStatus, newStatus)
}
```

V5 uses standalone helper functions (`formatTransitionText`, `formatDepChangeText`) for the same purpose.

**Create/Update Output Strategy**

V5 directly uses `taskToShowData()` to convert a `task.Task` to `*showData` and passes it to the formatter after mutation:
```go
return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(updatedTask))
```

V6 uses a shared `outputMutationResult()` helper that re-queries the store after mutation to get enriched data:
```go
func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
    if fc.Quiet {
        fmt.Fprintln(stdout, id)
        return nil
    }
    data, err := queryShowData(store, id)
    ...
    detail := showDataToTaskDetail(data)
    fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
    return nil
}
```

V6's approach is genuinely better here: by re-querying from the store, create/update output includes enriched data (parent titles, blocked-by task names) just like the show command. V5's `taskToShowData()` explicitly documents it does NOT enrich this data:

```go
// taskToShowData converts a task.Task to showData for formatter output.
// It populates basic fields but does not enrich blockedBy or children with
// context (titles/statuses) since those require DB queries.
```

### Code Quality

**Go Idioms and Naming**

V5 uses lowercase-initial function names for handlers (`runCreate`, `runShow`), making them package-private. This is idiomatic for internal dispatch. The `newFormatter()` factory is also unexported.

V6 exports the Run functions (`RunCreate`, `RunShow`, `RunTransition`) and the formatter factory (`NewFormatter`). This allows external testing but increases the public API surface unnecessarily for an internal package.

V5 naming: `newFormatter`, `formatTransitionText`, `formatDepChangeText`, `formatMessageText`
V6 naming: `NewFormatter`, `baseFormatter`, `showDataToTaskDetail`, `outputMutationResult`

Both follow Go conventions, but V6's exported names in an `internal/` package are somewhat unnecessary.

**Error Handling**

V5's io.Writer-based formatter properly propagates write errors:
```go
func (f *ToonFormatter) FormatTaskList(w io.Writer, rows []TaskRow) error {
    if len(rows) == 0 {
        _, err := fmt.Fprint(w, "tasks[0]{id,title,status,priority}:\n")
        return err
    }
```

V6's string-return approach silently discards write errors from `fmt.Fprintln`:
```go
fmt.Fprintln(stdout, fmtr.FormatTaskList(tasks))
```

This is a meaningful quality difference -- V5 properly handles write errors (which can occur with broken pipes, full disks, etc.), while V6 ignores them.

**DRY**

V5 uses standalone shared helper functions in `format.go`:
```go
func formatTransitionText(w io.Writer, data *TransitionData) error { ... }
func formatDepChangeText(w io.Writer, data *DepChangeData) error { ... }
func formatMessageText(w io.Writer, msg string) { ... }
```

V6 uses struct embedding with `baseFormatter`:
```go
type baseFormatter struct{}
func (b *baseFormatter) FormatTransition(id string, oldStatus string, newStatus string) string { ... }
func (b *baseFormatter) FormatDepChange(action string, taskID string, depID string) string { ... }
```

V6's approach is more idiomatic Go (composition via embedding). V5's approach is simpler but requires each formatter to explicitly call the helpers.

V6's `outputMutationResult` helper consolidates the create/update output logic, reducing duplication between the two commands.

**Type Safety**

V5 passes typed structs (`*TransitionData`, `*DepChangeData`) to formatter methods, providing clear contracts.

V6 passes individual primitive parameters (`id string, oldStatus string, newStatus string`), which is simpler but loses the documentation benefit of named struct fields.

For task lists, V5 uses a custom `TaskRow` struct while V6 reuses `task.Task`. V6's approach couples the formatter to the domain model, but is simpler since no conversion is needed at the list level.

### Test Quality

**V5 Test Functions (formatter_integration_test.go, 707 lines):**

Top-level function: `TestFormatterIntegration`

Subtests (37 t.Run calls):
1. "it formats create as full task detail in toon format"
2. "it formats create as full task detail in pretty format"
3. "it formats create as full task detail in json format"
4. "it formats update as full task detail in each format" (table-driven: --toon, --pretty, --json)
5. "it formats transitions in toon format (plain text)"
6. "it formats transitions in json format (structured)"
7. "it formats dep add confirmation in toon format"
8. "it formats dep add confirmation in json format"
9. "it formats dep rm confirmation in json format"
10. "it formats list in toon format"
11. "it formats list in pretty format"
12. "it formats list in json format"
13. "it formats show in toon format"
14. "it formats show in json format"
15. "it formats init message in toon format"
16. "it formats init message in json format"
17. "it applies --quiet override for create (ID only)"
18. "it applies --quiet override for update (ID only)"
19. "it applies --quiet override for show (ID only)"
20. "it applies --quiet override for transitions (nothing)"
21. "it applies --quiet override for dep (nothing)"
22. "it applies --quiet override for list (IDs only)"
23. "it applies --quiet override for init (nothing)"
24. "it handles empty list in toon format (zero count)"
25. "it handles empty list in pretty format (message)"
26. "it handles empty list in json format (empty array)"
27. "it defaults to toon when piped (non-TTY)"
28. "it defaults to pretty when TTY"
29. "it respects --toon override when TTY"
30. "it respects --pretty override when non-TTY"
31. "it respects --json override"
32. "errors remain plain text stderr regardless of format" (sub-runs per format flag)
33. "it resolves formatter in Context for handlers to use"
34. "it sets ToonFormatter when non-TTY default"
35. "it sets PrettyFormatter when TTY default"

Additional test modifications:
- blocked_test.go: 4 tests updated to add `--pretty` flag
- list_test.go: 6 tests updated to add `--pretty` flag
- parent_scope_test.go: 3 tests updated to add `--pretty` flag
- ready_test.go: 4 tests updated to add `--pretty` flag
- show_test.go: 11 tests updated to add `--pretty` flag
- update_test.go: 1 test updated to add `--pretty` flag

V5 approach: Tests use the `Run()` function directly, passing format flags in the args slice. Each format is tested individually with separate subtests. Update uses a small table-driven approach (3 formats).

**V6 Test Functions (format_integration_test.go, 749 lines):**

Top-level function: `TestFormatIntegration`

Subtests (30 t.Run calls, heavily table-driven):
1. "it formats create as full task detail in each format" (table: toon, pretty, json)
2. "it formats transitions in each format" (table: toon, pretty, json)
3. "it formats dep confirmations in each format" (table: toon, pretty, json)
4. "it formats list in each format" (table: toon, pretty, json)
5. "it formats show in each format" (table: toon, pretty, json)
6. "it formats init in each format" (table: toon, pretty, json)
7. "it applies --quiet override for each command type":
   - "create outputs ID only when quiet"
   - "transition outputs nothing when quiet"
   - "dep add outputs nothing when quiet"
   - "list outputs IDs only when quiet"
   - "show outputs ID only when quiet"
   - "init outputs nothing when quiet"
8. "it handles empty list per format" (table: toon, pretty, json)
9. "it defaults to TOON when piped, Pretty when TTY":
   - "non-TTY defaults to toon"
   - "TTY defaults to pretty"
10. "it respects --toon/--pretty/--json overrides":
    - "--toon overrides TTY default"
    - "--pretty overrides piped default"
    - "--json overrides piped default"
11. "quiet plus json: quiet wins, no JSON wrapping"
12. "errors remain plain text to stderr regardless of format"

Additional test modifications:
- blocked_test.go: Helper + 2 tests updated with `IsTTY: true`
- cli_test.go: 6 tests updated to pass `FormatConfig{}`/`&PrettyFormatter{}`
- create_test.go: Helper updated with `IsTTY: true`
- dep_test.go: Helper updated with `IsTTY: true`
- list_show_test.go: Helpers updated with `IsTTY: true` + column width adjustments
- ready_test.go: Helper + column width adjustments
- transition_test.go: Helper updated with `IsTTY: true` + arrow character change
- update_test.go: Helper updated with `IsTTY: true`

**Test Quality Comparison:**

V6 is more consistently table-driven: every "format X in each format" test uses the `[]struct{name, flag, checkFunc}` pattern. V5 mixes individual subtests (create toon/pretty/json as separate tests) with occasional table-driven tests (update).

V5 has 3 additional tests for verifying the Context.Fmt field is correctly set by type-assertion (`*JSONFormatter`, `*ToonFormatter`, `*PrettyFormatter`). V6 lacks this direct verification -- it only tests behavior indirectly through output.

V5 explicitly tests `--quiet + --json` for create (verifying no JSON wrapping). V6 tests `--quiet + --json` only for transitions.

V5 tests error formatting with all 3 format flags (sub-test per flag). V6 tests only with `--json`.

V5 tests dep rm in JSON format. V6 does not test dep rm separately (only dep add per format).

V5 is missing: pretty-format transition test, pretty-format show test, pretty-format init test. V6 covers all 3 formats for every command type through table-driven tests.

V6 is missing: direct formatter resolution verification, error formatting across all 3 formats, dep rm specific test.

Edge cases tested in both: empty list per format, quiet+json interaction, TTY auto-detection, flag overrides.

V6 tests use the `App` struct directly with `IsTTY` field, which is more realistic. V5 tests use the `Run()` function with an isTTY boolean parameter.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code follows standard formatting | PASS -- code follows standard formatting |
| Handle all errors explicitly (no naked returns) | PASS -- all errors propagated; io.Writer-based formatter returns errors | PARTIAL -- string-return formatters discard fmt.Fprintln write errors |
| Write table-driven tests with subtests | PARTIAL -- mostly individual subtests, only update test is table-driven | PASS -- consistently table-driven with `[]struct{name, flag, checkFunc}` |
| Document all exported functions, types, and packages | PASS -- all exported symbols documented | PASS -- all exported symbols documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- consistently used | PASS -- consistently used |
| Do not ignore errors without justification | PASS -- formatter errors propagated | PARTIAL -- `fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))` ignores write error; `time.Parse` errors silently ignored in `showDataToTaskDetail` |
| Do not hardcode configuration | PASS -- format resolved from flags/TTY | PASS -- format resolved from flags/TTY |

### Spec-vs-Convention Conflicts

**Formatter Interface Design: io.Writer vs string return**

- The spec says: "Wire FormatConfig + Formatter into every handler via CLI dispatcher"
- Go convention for formatters that write output: accept `io.Writer` and return `error` (like `json.Encoder.Encode`, `template.Execute`, etc.)
- V5 chose the io.Writer pattern, which is more idiomatic and propagates write errors
- V6 chose the string-return pattern, which is simpler to call but loses error propagation
- Assessment: V5's choice is the more conventional Go approach. V6's choice is a reasonable simplification but sacrifices error handling quality.

**Transition Arrow Character**

- The spec says: "Transitions plain text in TOON/Pretty, structured in JSON"
- V5 uses Unicode right arrow `\u2192` in transition text output (matching pre-existing behavior)
- V6 changes to ASCII `->` in transition text output
- Assessment: This is a minor stylistic difference. V6's change to `->` is slightly more portable but diverges from the pre-existing output format.

**Create/Update Data Enrichment**

- The spec says: "create/update: FormatTaskDetail (same as show)"
- V5 uses `taskToShowData()` which does NOT include enriched related task info (titles/statuses)
- V6 uses `outputMutationResult()` which re-queries the store for enriched data
- Assessment: V6 is more faithful to "same as show" since show includes enriched blocked-by/children data. V5's output will be missing related task titles/statuses.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 16 | 18 |
| Lines added | 803 | 921 |
| Lines removed | 121 | 155 |
| Impl LOC (new/changed) | ~96 (format.go + handler changes) | ~172 (format.go + app.go + helpers.go + handler changes) |
| Test LOC (new file) | 707 | 749 |
| Test functions (integration) | 37 subtests | 30 subtests (table-driven, ~same effective coverage) |

## Verdict

**V6 is the better implementation**, though by a modest margin. The key differentiators:

1. **Enriched mutation output (V6 wins)**: V6's `outputMutationResult()` re-queries the store after create/update, ensuring output matches `show` format exactly (with parent titles, blocker names). V5's `taskToShowData()` explicitly skips this enrichment, producing less-complete output for create/update. This is a spec compliance issue -- the spec says "create/update = show format."

2. **Table-driven tests (V6 wins)**: V6 consistently uses the `[]struct{name, flag, checkFunc}` table-driven pattern for all "format X in each format" tests, which is the explicit skill constraint. V5 mostly uses individual subtests.

3. **DRY via composition (V6 wins)**: V6's `baseFormatter` embedding and `outputMutationResult` helper reduce duplication more effectively than V5's standalone helper functions.

4. **Error propagation (V5 wins)**: V5's io.Writer-based Formatter interface properly propagates write errors, while V6's string-return interface silently discards them. V6 also silently ignores `time.Parse` errors in `showDataToTaskDetail`. This is a real quality gap per the skill constraint "Handle all errors explicitly."

5. **Parameter threading (V5 wins marginally)**: V5's Context-embedded Formatter keeps handler signatures clean (`func(ctx *Context) error`), while V6 threads `FormatConfig` and `Formatter` through every handler signature, adding boilerplate.

6. **Test coverage gaps**: V5 misses pretty-format tests for transition/show/init. V6 misses direct formatter resolution verification and multi-format error tests. Both are minor.

On balance, V6's spec compliance advantage (enriched create/update output) and better test structure outweigh V5's error handling advantage. However, V6 should address the silent error discarding in a production codebase.
