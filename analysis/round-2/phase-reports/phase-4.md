# Phase 4: Output Formats

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 4-1 Formatter Abstraction | V4 | Moderate | iota enum, `DetectTTY(io.Writer)`, single `ResolveFormat` call site |
| 4-2 TOON Formatter | V2 | Moderate | Exact spec fidelity, strict test assertions, error propagation |
| 4-3 Pretty Formatter | V4 | Moderate | Dynamic alignment everywhere, more test cases, defensive truncation |
| 4-4 JSON Formatter | V4 | Slim | Concrete `StatsData` type, cleaner domain types |
| 4-5 Integration | V4 | Moderate | Error-to-stderr testing, formatter type assertions, existing test updates |
| 4-6 Verbose | V2 | Slim | More tests (10 vs 7), mutation verbose coverage, unit tests for logger |

## Cross-Task Architecture Analysis

### The Formatter Interface: A Phase-Wide Fault Line

The most significant cross-task pattern is that V2 and V4 chose fundamentally different Formatter interface signatures in task 4-1, and every subsequent task (4-2 through 4-5) inherits those consequences. This is visible only at the phase level.

**V2 interface (in `formatter.go`):**
```go
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatStats(w io.Writer, stats interface{}) error
```

**V4 interface (in `format.go`):**
```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatStats(w io.Writer, stats StatsData) error
```

Three design decisions ripple across every formatter implementation and every command handler:

1. **`quiet bool` in interface vs command handler**: V4 pushes `quiet` into `FormatTaskList` and `FormatDepChange`, but handles it in the command handler for transitions, init, create, update, and show. This creates an inconsistency that manifests differently in each task. In `dep.go`, V4 simply calls `a.Formatter.FormatDepChange(a.Stdout, taskID, blockedByID, "added", a.Quiet)` and the formatter silently returns nil when quiet is true. But in `transition.go`, V4 checks `if a.Quiet { return nil }` before calling the formatter. V2 is consistent: quiet is always checked in the handler, formatters are never invoked in quiet mode. This is a cross-task architectural inconsistency that no individual task report can surface.

2. **`interface{}` vs `StatsData` for FormatStats**: V2's `interface{}` forces a type assertion in every formatter's `FormatStats` (toon, pretty, json -- three copies of `sd, ok := stats.(*StatsData)`). V4's concrete `StatsData` eliminates this entirely. The `interface{}` decision also creates a testable error path (wrong type passed) that V2 tests in every formatter, adding 3 test cases that only exist because of a weak interface design.

3. **`task.Status` vs `string` in FormatTransition**: V2 uses the domain type `task.Status`, which means all three formatters import `internal/task`. V4 uses plain strings and converts at the single call site in `transition.go`: `string(result.OldStatus)`. This means V4's formatter package has zero coupling to the domain layer, while V2's has coupling in every formatter file.

### Data Flow: How Show Data Reaches Formatters

The path from database to formatter output spans tasks 4-1 (interface), 4-2/4-3/4-4 (formatters), and 4-5 (integration). The two versions diverge significantly in how they supply data to `FormatTaskDetail`:

**V2: Shared query function reused across show, create, update**
```go
// show.go -- queryShowData is a package-level function
func queryShowData(store *storage.Store, lookupID string) (*showData, error) {
    // SQL query joins tasks + dependencies + children
    // Returns *showData with BlockedBy, Children, ParentTitle populated
}

// create.go -- reuses queryShowData after mutation
data, err := queryShowData(store, createdTask.ID)
if err != nil {
    fmt.Fprintln(a.stdout, createdTask.ID) // fallback
    return nil
}
return a.formatter.FormatTaskDetail(a.stdout, data)
```

**V4: Lightweight conversion for create/update, inline query for show**
```go
// create.go -- pure conversion, no DB query
func taskToDetail(t *task.Task) TaskDetail {
    detail := TaskDetail{
        ID: t.ID, Title: t.Title, Status: string(t.Status),
        Priority: t.Priority, Description: t.Description, Parent: t.Parent,
        Created: t.Created.Format("2006-01-02T15:04:05Z"),
        Updated: t.Updated.Format("2006-01-02T15:04:05Z"),
    }
    // ... Closed handling
    return detail
}

// show.go -- full query inline, then manual conversion to TaskDetail
td := TaskDetail{ID: detail.ID, Title: detail.Title, ...}
for _, b := range blockedBy {
    td.BlockedBy = append(td.BlockedBy, RelatedTask{ID: b.ID, ...})
}
return a.Formatter.FormatTaskDetail(a.Stdout, td)
```

V2's approach means `create` and `show` produce byte-identical output (same query, same data structure). V4's approach means `create`/`update` output lacks blocked_by details, children, and parent titles -- it only has the bare task fields. This is architecturally cleaner (no extra DB query) but produces less rich output for mutations.

V4's `show.go` also has a DRY problem invisible at the task level: it defines local `taskDetail` and `relatedTask` structs (lines 31-48) that are near-identical to the exported `TaskDetail` and `RelatedTask` in `format.go`, then manually maps between them (lines 139-166). V2 avoids this because `showData`/`relatedTask` are defined once in `show.go` and used directly by the formatters.

### ready/blocked: DRY Divergence

V2 handles `ready` and `blocked` as aliases in the dispatcher:
```go
case "ready":
    return a.runList([]string{"--ready"})
case "blocked":
    return a.runList([]string{"--blocked"})
```

V4 creates dedicated `runReady` and `runBlocked` methods in separate files (`ready.go`, `blocked.go`) that duplicate the store-open, query, scan, close pattern from `runList`. Each is ~35 lines of duplicated boilerplate. The query logic is shared via `readyConditionsFor()` -- a well-designed SQL fragment function -- but the Go handler code is not. V2's approach is more DRY at the cost of synthetic argument parsing.

### Shared Helpers Across Formatters

**V2 cross-formatter sharing:**
- `marshalIndentTo(w io.Writer, v interface{})` -- package-level function in `json_formatter.go`, used by all 6 JSON format methods. Could theoretically be shared with other formatters but isn't.
- `escapeField(s string)` -- package-level function in `toon_formatter.go`, used by TOON detail and related sections. Has a fast-path optimization (`!strings.ContainsAny(s, ",\"\n\\")`) and a manual fallback.
- `writeRelatedSection(w io.Writer, name string, items []relatedTask)` -- package-level function in `toon_formatter.go`, used for both blocked_by and children.
- `truncateTitle(title string, maxLen int)` -- package-level function in `pretty_formatter.go`.

**V4 cross-formatter sharing:**
- `writeJSON` is a method on `*JSONFormatter`, not a package-level function, so it cannot be reused by other formatters.
- `toonEscapeValue(s string)` -- package-level function in `toon_formatter.go`, used across detail/related/stats sections. No fast-path optimization.
- `buildTaskSection`, `buildRelatedSection`, `buildDescriptionSection`, `buildStatsSection`, `buildByPrioritySection` -- all receiver methods on `*ToonFormatter`, composing via string joining.
- `truncateTitle(title string, maxLen int)` -- package-level function in `pretty_formatter.go`, with additional guard for `maxLen <= 3`.

V4's section-builder pattern in the TOON formatter (`buildTaskSection`, `buildRelatedSection`, etc.) provides better composability within that formatter. But making them receiver methods unnecessarily ties them to the struct. V2's package-level `writeRelatedSection` is more reusable.

### The StatsData Type: Defined in Different Places

V2 defines `StatsData` in `toon_formatter.go` (line 14-23) rather than in `formatter.go` with the interface. This is a cross-task code organization issue: the type is used by all three formatters and by `stats.go`, but lives in one formatter's file.

V4 defines `StatsData` in `format.go` (lines 45-54) alongside the `Formatter` interface and `TaskDetail` type. This is the correct location since it's part of the shared formatter contract.

### Error Propagation: A Systemic Pattern

Across all three formatters, V2 consistently captures and returns `fmt.Fprintf` write errors:
```go
// V2 pattern in FormatTransition, FormatDepChange, FormatMessage, FormatTaskList
_, err := fmt.Fprintf(w, "%s: %s -> %s\n", id, oldStatus, newStatus)
return err
```

V4 systematically ignores write errors across all three formatters:
```go
// V4 pattern everywhere
fmt.Fprintf(w, "%s: %s -> %s\n", id, oldStatus, newStatus)
return nil
```

This is not a one-off difference -- it is a consistent design philosophy applied across tasks 4-2, 4-3, and 4-4. V2 treats the writer as potentially fallible (broken pipe, full disk). V4 assumes writes always succeed. V2's approach is more correct, though in practice write errors to stdout buffers are rare.

However, even V2 is inconsistent -- `FormatTaskDetail` and `FormatStats` in the pretty formatter discard `fmt.Fprintf` errors, while `FormatTaskList` captures them. This partial propagation is arguably worse than V4's consistent ignoring.

## Code Quality Patterns

### Naming Conventions

| Concept | V2 | V4 | Better |
|---------|-----|-----|--------|
| Format enum type | `OutputFormat string` | `Format int` | V4 (iota idiomatic) |
| Format constant | `FormatTOON` | `FormatToon` | V4 (Toon is not an acronym) |
| App entry point file | `app.go` | `cli.go` | V4 (more descriptive) |
| Formatter file | `formatter.go` | `format.go` | Tie |
| List row type | `TaskRow` (exported) + `listRow` (internal) | `listRow` (internal only) | V4 (no redundant exported type) |
| Detail type | `*showData` (unexported, pointer) | `TaskDetail` (exported, value) | V4 (semantically named, value type) |
| Related task type | `relatedTask` (unexported) | `RelatedTask` (exported) | V4 (exported for formatter contract) |
| Stats type | `StatsData` in `toon_formatter.go` | `StatsData` in `format.go` | V4 (correct file location) |
| Marshal helper | `marshalIndentTo` (package function) | `writeJSON` (receiver method) | V2 (more reusable) |
| Verbose logger | `a.verbose` | `a.vlog` | V2 (more readable) |
| Store helper | `a.newStore()` | `a.openStore()` | V4 (semantically accurate) |
| App field visibility | Mostly unexported | All exported | V2 (better encapsulation) |

### App Struct Encapsulation

V2 keeps App fields private:
```go
type App struct {
    config    Config
    FormatCfg FormatConfig     // one exported field
    formatter Formatter        // unexported
    verbose   *VerboseLogger   // unexported
    workDir   string           // unexported
    stdout    io.Writer        // unexported
    stderr    io.Writer        // unexported
}
```

V4 exports everything:
```go
type App struct {
    Stdout       io.Writer
    Stderr       io.Writer
    Dir          string
    Quiet        bool
    Verbose      bool
    OutputFormat Format
    IsTTY        bool
    Formatter    Formatter
    vlog         *VerboseLogger // one unexported field
}
```

V2's approach is better Go practice -- internal state should be unexported unless consumers need it. V4's exported fields make testing slightly easier (direct struct literal construction) but break encapsulation. This affects every command handler across the phase.

### Compile-Time Interface Checks

V2 consistently includes compile-time interface checks in implementation files:
```go
// pretty_formatter.go:19
var _ Formatter = &PrettyFormatter{}
// json_formatter.go:16
var _ Formatter = &JSONFormatter{}
```

V4 relies on test-time checks only. V2's approach catches interface violations at compile time without running tests -- standard Go idiom.

## Test Coverage Analysis

### Aggregate Counts

| Metric | V2 | V4 |
|--------|-----|-----|
| Phase 4 test LOC (format-related) | 2,874 | 2,915 |
| formatter/format base tests | 280 | 153 |
| TOON formatter tests | 396 | 420 |
| Pretty formatter tests | 439 | 644 |
| JSON formatter tests | 668 | 701 |
| Integration tests | 832 | 774 |
| Verbose tests | 259 | 223 |
| Phase 4 impl LOC (format-related) | 1,032 | 1,089 |
| Test:Impl ratio | 2.8:1 | 2.7:1 |

### Assertion Philosophy

V2 favors exact string matching for format output:
```go
// V2 toon_formatter_test.go
want := "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n  5,3,1,1,0,2,1\n\nby_priority..."
if got != want { t.Errorf("...") }
```

V4 favors `strings.Contains` checks:
```go
// V4 toon_formatter_test.go
if !strings.Contains(got, "stats{total,open,in_progress,done,cancelled,ready,blocked}:") {
    t.Errorf("missing stats header")
}
```

V2's approach catches any formatting regression (extra whitespace, wrong newlines). V4's approach is more resilient to minor changes but could miss formatting drift. This pattern repeats across tasks 4-2, 4-3, and 4-4 -- it is a systematic testing philosophy difference.

### Cross-Task Test Gap Pattern

A pattern emerges when looking across tasks: both versions have complementary blind spots.

**V2 systematically tests across all 3 formats** in integration (table-driven), but has no error-to-stderr validation.

**V4 has explicit error-to-stderr tests** across all formats, but has systematic gaps in Pretty format coverage (missing for update, show, init, dep).

Neither version tests the full matrix of `format x command x quiet` combinations, but V2 comes closer with its table-driven approach. V4's individual subtests are more descriptive but cover fewer cells of the matrix.

### Untested Quiet Paths

V4 adds `quiet bool` to `FormatTaskList` and `FormatDepChange` but does not test these code paths in the formatter unit tests for any of the three formatters. The quiet paths are only tested in the integration test. V2 avoids this problem by handling quiet outside the formatter entirely.

## Phase Verdict

**V4 is the better implementation overall**, though the margin is narrow and V2 has genuine advantages in specific areas.

**V4 wins on architecture and type design:**
- The `Format int` iota enum is more idiomatic than `OutputFormat string`
- `DetectTTY(io.Writer)` is a cleaner API than V2's two-function approach with injected stat callbacks
- Concrete `StatsData` and `TaskDetail` types eliminate runtime type assertions
- `ResolveFormat` is called at a single site (not duplicated)
- No domain-layer coupling in formatters (no `task.Status` imports)
- Dynamic alignment in PrettyFormatter handles arbitrary value sizes (V2's `%2d` breaks at 100+)
- `formatName()` utility provides clean enum-to-string conversion for verbose logging
- Defensive `truncateTitle` guard against `maxLen <= 3`
- V4 decomposed stats queries into named helper functions (`queryStatusCounts`, `queryPriorityCounts`, `queryWorkflowCounts`)
- V4 computes `Blocked = Open - Ready` rather than running a separate blocked query, which is both simpler and mathematically guaranteed consistent

**V2 wins on consistency and correctness:**
- Uniform quiet handling: always in command handler, never in formatter -- no inconsistency
- Compile-time interface checks in implementation files (Go idiom)
- App fields properly encapsulated (unexported)
- Consistent error propagation from `fmt.Fprintf` in formatters
- Exact string matching in tests catches formatting regressions V4's `Contains` checks miss
- `queryShowData()` reused across show/create/update ensures identical output
- More comprehensive test matrix (table-driven across all 3 formats for every command)
- `FormatDepChange` handles unknown actions with a default case (V4 silently drops them)
- `StatsData` pointer semantics (`*StatsData`) in V2's `FormatStats` is slightly more canonical for JSON nullable semantics on parent/closed
- Mutation verbose testing (create command) in task 4-6

**The deciding factors:**

V4's type safety improvements (eliminating `interface{}` from the Formatter contract) and its idiomatic Go patterns (`iota` enum, `io.Writer` for TTY detection) represent structural improvements that benefit the codebase long-term. The dynamic alignment in PrettyFormatter is a real correctness issue -- V2's hardcoded `%2d` will produce misaligned output when task counts exceed 99.

V2's quiet handling consistency is the strongest counter-argument. V4's split approach (some commands check quiet, others pass it to the formatter) creates a maintenance burden where developers must know which pattern to follow when adding new commands. V2's clear rule ("quiet is always checked before calling the formatter") is simpler.

On balance, V4's type system and idiomatic improvements outweigh V2's consistency advantages. The ideal implementation would combine V4's type design with V2's quiet handling strategy, compile-time interface checks, encapsulated App fields, and exact-match test assertions.
