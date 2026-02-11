# Phase 4: Output Formats

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 4-1 Formatter abstraction & TTY selection | V6 | Moderate | V6: spec-faithful naming (`Format` not `OutputFormat`), compile-time interface check, 32 vs 18 tests. V5: better `ResolveFormat` signature (plain bools vs coupled `globalFlags`). |
| 4-2 TOON formatter | V6 | Moderate | V6: typed params from day one, generic `encodeToonSection[T]` helpers, fuller toon-go delegation. V5: spec-literal unquoted timestamps, explicit error propagation. |
| 4-3 Pretty formatter | V6 | Significant | V6: typed interface eliminated `interface{}` anti-pattern, simpler string-return API. V5: more reusable `truncateTitle(title, maxWidth)` with `maxWidth <= 3` edge case. |
| 4-4 JSON formatter | V6 | Significant | V6: compile-time type safety, `toJSONRelated()` DRY helper, nil-slice edge case tests. V5: proper `%w` error wrapping in `writeJSON`. |
| 4-5 Integrate formatters into all commands | V6 | Modest | V6: enriched create/update output via store re-query, consistently table-driven tests. V5: proper write error propagation, cleaner handler signatures via Context. |
| 4-6 Verbose output & edge case hardening | V5 | Modest | V5: 17 tests across 3 layers (unit + store + CLI) vs V6's 8 in 1 file. V6: better architecture (callback decoupling, nil-receiver pattern). |

**Phase score: V6 wins 5 of 6 tasks.** V5's sole win (4-6) is driven by test depth, not architecture.

## Cross-Task Architecture Analysis

### The Formatter Interface Decision Cascades Through Every Task

The single most impactful decision in Phase 4 was the Formatter interface design in task 4-1. That choice -- made once -- determined the shape of every subsequent task:

**V5's `io.Writer` + `error` interface:**
```go
FormatTaskList(w io.Writer, rows []TaskRow) error
```

**V6's `string` return interface:**
```go
FormatTaskList(tasks []task.Task) string
```

This is not visible from any single task report. The cascade:

1. **Task 4-2 (TOON)**: V5 must check every `fmt.Fprintf` return value -- 15+ error checks in `FormatTaskDetail` alone. V6 uses `strings.Builder` internally where writes cannot fail, eliminating the entire error-handling layer.

2. **Task 4-3 (Pretty)**: V5's `FormatTaskDetail` (lines 66-114) calls `fmt.Fprintf(w, ...)` 12 times and checks ZERO of those errors -- inconsistent with its own interface contract. V6's `strings.Builder` approach makes this inconsistency impossible.

3. **Task 4-4 (JSON)**: V5 needs a `writeJSON` helper that wraps `json.MarshalIndent` + `w.Write`, plus a `//nolint:errcheck` on `FormatMessage`. V6 needs a simpler `marshalIndentJSON` that just returns a string.

4. **Task 4-5 (Integration)**: V5 callers write `ctx.Fmt.FormatTaskDetail(ctx.Stdout, data)` -- the formatter directly handles output. V6 callers write `fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))` -- the caller handles output. V6's pattern is more testable (no buffer setup) but silently discards `Fprintln` write errors.

5. **Task 4-6 (Verbose)**: Both versions log to stderr independently of the formatter. The interface choice has no impact here, explaining why V5 wins this task on different grounds (test depth).

**Verdict on the cascade**: V6's string-return interface produces cleaner implementations in tasks 4-2 through 4-5 at the cost of losing write error propagation. V5's `io.Writer` interface is more Go-idiomatic for I/O operations but leads to inconsistent error handling in practice (PrettyFormatter ignores write errors). The `io.Writer` design promise is undermined by its own implementation.

### The `interface{}` Debt Visible Only at Phase Level

V5's commit-time Formatter interface used `interface{}` parameters (visible in task 4-2, 4-3, and 4-4 reports). Individual task reports correctly note this as a design weakness. What they miss: this required a **separate refactoring task (T6-7)** later to fix, meaning V5's Phase 4 generated technical debt that consumed additional development effort. V6 never incurred this debt -- its typed interface was correct from task 4-1.

The debt chain:
- Task 4-1: V5 defines `interface{}` params
- Task 4-2: V5 implements runtime type assertions (`data.([]TaskRow)`)
- Task 4-3: Same pattern, different types
- Task 4-4: Same pattern, different types
- Task T6-7 (later): V5 refactors to typed params

V6 paid zero refactoring cost. This is the strongest cross-task argument for V6's approach.

### Shared Helper Evolution: Functions vs Embedding

Both versions extract shared logic for `FormatTransition`, `FormatDepChange`, and `FormatMessage`. The evolution path differs:

**V5** uses package-level helper functions in `format.go`:
```go
func formatTransitionText(w io.Writer, data *TransitionData) error { ... }
func formatDepChangeText(w io.Writer, data *DepChangeData) error { ... }
func formatMessageText(w io.Writer, msg string) { ... }
```
Each concrete formatter delegates to these explicitly:
```go
func (f *ToonFormatter) FormatTransition(w io.Writer, data *TransitionData) error {
    return formatTransitionText(w, data)
}
```

**V6** uses `baseFormatter` struct embedding in `format.go`:
```go
type baseFormatter struct{}
func (b *baseFormatter) FormatTransition(id, oldStatus, newStatus string) string { ... }
func (b *baseFormatter) FormatDepChange(action, taskID, depID string) string { ... }
```
Concrete formatters embed it:
```go
type ToonFormatter struct { baseFormatter }
type PrettyFormatter struct { baseFormatter }
```

The cross-task pattern: V6's `baseFormatter` is defined once (4-1), used in 4-2 and 4-3, and tested independently in `base_formatter_test.go`. V6 also includes `TestAllFormattersProduceConsistentTransitionOutput` -- a **cross-formatter consistency test** that verifies ToonFormatter and PrettyFormatter produce identical transition output. V5 has no equivalent cross-formatter test.

V6's embedding is more idiomatic Go composition. V5's function delegation is simpler but requires each formatter to write trivial one-line wrapper methods (3 per formatter = 6 total across Toon and Pretty).

### Data Type Layering: Domain vs Presentation

A subtle but important cross-task pattern is how data flows from domain to formatter:

**V5 architecture:**
```
task.Task --> showData (unexported, CLI-layer) --> Formatter methods
task.Task --> TaskRow (exported, CLI-layer) --> FormatTaskList
```
V5 maps domain types to presentation types before the formatter sees them. `showData` has pre-formatted string fields (`created string`, not `time.Time`). This means formatters receive ready-to-display strings.

**V6 architecture:**
```
task.Task --> TaskDetail (wrapping task.Task) --> Formatter methods
task.Task --> directly to FormatTaskList --> FormatTaskList
```
V6 passes `task.Task` directly to `FormatTaskList` and wraps it minimally in `TaskDetail` for show. Formatters call `task.FormatTimestamp(t.Created)` themselves.

**Trade-off**: V5's pre-mapping means formatters are simpler (no domain knowledge needed), but it creates more intermediate types and a mapping layer that can have bugs. V6's direct domain type usage means formatters depend on `internal/task`, but eliminates the mapping layer entirely. For a CLI tool where the formatter and domain are in the same binary, V6's approach is pragmatic.

The proof is in `FormatTaskList` -- V5 requires `TaskRow` as an intermediate type with its own field definitions. V6 passes `[]task.Task` directly. The `TaskRow` type exists purely as a presentation DTO that duplicates fields already present on `task.Task`.

### Wiring Pattern: Context Struct vs Parameter Passing

**V5**: Formatter stored on `Context` struct, handlers access via `ctx.Fmt`:
```go
func runCreate(ctx *Context) error {
    // ...
    return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(createdTask))
}
```

**V6**: Formatter passed as explicit parameter to every handler:
```go
func (a *App) handleCreate(fc FormatConfig, fmtr Formatter, subArgs []string) error {
    // ...
    return RunCreate(dir, fc, fmtr, subArgs, a.Stdout)
}
```

V5's pattern is cleaner for handler signatures (one parameter), but couples the formatter to the Context object. V6's pattern makes dependencies explicit (good for testing, good for documentation) but creates verbose signatures. At the phase level, V6's explicit wiring flows through 11 handler methods -- a noticeable amount of boilerplate.

However, V6's explicit wiring enables `outputMutationResult` (task 4-5) -- a shared helper that takes `(store, id, fc, fmtr, stdout)` and consolidates create/update output logic. V5 cannot easily extract such a helper because the Context bundles too many concerns.

## Code Quality Patterns

### Error Handling Consistency

Across the entire phase, V5 and V6 show opposite error handling weaknesses:

**V5's inconsistency**: The `io.Writer` + `error` interface promises error propagation, but `PrettyFormatter.FormatTaskDetail` (lines 67-84) and `PrettyFormatter.FormatStats` (lines 142-179) ignore `fmt.Fprintf` return values for 20+ writes. `JSONFormatter.FormatMessage` uses `//nolint:errcheck`. The interface promises more than the implementation delivers.

**V6's consistent simplification**: By returning `string`, V6 eliminates the possibility of inconsistent error handling within formatters. The trade-off is that `marshalIndentJSON` silently returns `"null"` on marshal failure (which cannot realistically occur with controlled struct types), and `fmt.Fprintln` write errors are silently discarded at the call site.

Neither approach is perfect, but V6's is more honest -- it does not promise error handling it fails to deliver.

### LOC Efficiency

| Component | V5 | V6 |
|-----------|-----|-----|
| format.go (core abstraction) | 119 | 183 |
| toon_formatter.go | 225 | 202 |
| pretty_formatter.go | 215 | 167 |
| json_formatter.go | 207 | 210 |
| verbose.go/verbose.go | 37 | 39 |
| **Total impl** | **803** | **801** |
| **Total tests** | **2781** | **2774** |

Nearly identical total LOC. V6's `format.go` is 64 lines larger because it co-locates data types (`TaskDetail`, `Stats`, `RelatedTask`), the `baseFormatter`, the `StubFormatter`, and the `NewFormatter` factory -- all in one file. V5 scatters data types across formatter files (`TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` in `toon_formatter.go`).

V6's co-location is better: opening `format.go` gives a complete picture of the formatter system's public contract. V5 requires reading multiple files to understand the full type surface.

### Compile-Time Safety

V6 includes `var _ Formatter = (*XFormatter)(nil)` for every concrete formatter at the package level:
- `format.go:153`: `var _ Formatter = (*StubFormatter)(nil)`
- `toon_formatter.go:19`: `var _ Formatter = (*ToonFormatter)(nil)`
- `pretty_formatter.go:19`: `var _ Formatter = (*PrettyFormatter)(nil)`
- `json_formatter.go:14`: `var _ Formatter = (*JSONFormatter)(nil)`

V5 has these checks only in test files, not in source. This means V5 could have a compile-time interface violation that is only caught by running tests, not by a simple `go build`. V6's approach catches violations earlier in the development cycle.

### Generics Usage

V6 is the only version using Go generics (Go 1.18+):
```go
func encodeToonSection[T any](name string, rows []T) string { ... }
func encodeToonSingleObject[T any](name string, value T) string { ... }
```

These helpers eliminate duplicated TOON encoding logic across list, stats, and related-task sections. V5 has no generic functions -- it uses manual string building or type-specific code for each TOON section type. V6's generics usage is a legitimate improvement that reduces code and demonstrates awareness of modern Go features.

## Test Coverage Analysis

### Aggregate Test Metrics

| Metric | V5 | V6 |
|--------|-----|-----|
| Total test LOC (phase 4 files) | 2,781 | 2,774 |
| Test files | 8 | 7 |
| Test layers | 3 (unit, store, CLI) | 2 (unit, CLI) |
| Cross-formatter consistency test | No | Yes (`TestAllFormattersProduceConsistentTransitionOutput`) |
| Nil-slice edge cases | Sporadic | Systematic (list, blocked_by, children) |
| Store-level verbose tests | 5 dedicated tests | 0 |
| Table-driven pattern usage | Occasional | Consistent |

### V5's Testing Strength: Layered Verbose Coverage

V5 tests verbose logging at three layers:
1. `engine/verbose_test.go` (59 lines) -- unit tests for `VerboseLogger.Log`/`Logf`
2. `engine/store_verbose_test.go` (167 lines) -- store-level integration testing verbose output during real `Mutate`/`Query` operations
3. `cli/verbose_test.go` (191 lines) -- CLI integration testing end-to-end verbose flow

V6 tests at only two layers:
1. `cli/verbose_test.go` (218 lines) -- unit tests for `VerboseLogger` + CLI integration tests
2. No store-level verbose tests

V5's `store_verbose_test.go` verifies that specific verbose messages appear during real file I/O operations (lock acquire, cache freshness check, atomic write). V6 relies entirely on CLI-level integration tests to indirectly verify the store callback path. If the `storeOpts` bridge function were broken in V6, no test would catch it directly.

### V6's Testing Strength: Edge Cases and Consistency

V6 systematically tests nil slices across all three formatters (TOON, Pretty, JSON) for `FormatTaskList(nil)`, `BlockedBy: nil`, and `Children: nil`. V5 does not test nil slices -- only empty slices. In Go, a nil slice marshals to `null` in JSON, while an empty slice marshals to `[]`. V6's nil-slice tests directly guard against this common Go gotcha.

V6's `base_formatter_test.go` includes a cross-formatter consistency test that verifies ToonFormatter and PrettyFormatter produce identical transition output. This is a phase-level concern that no V5 test addresses.

V6's integration tests (`format_integration_test.go`) use a consistent table-driven pattern: every "format X in each format" test uses `[]struct{name, flag, checkFunc}`. V5 mixes individual subtests with occasional table-driven tests, resulting in less uniform test structure.

### Missing Coverage in Both

Neither version tests:
- **TOON output round-trip parsing** (write TOON, read it back with `toon.Unmarshal`)
- **Formatter output determinism** (same input always produces same output)
- **Large dataset formatting** (hundreds of tasks in list view)
- **Unicode in task titles** (CJK characters, emoji, RTL text) affecting column alignment
- **Concurrent formatter usage** (formatters are stateless so this is theoretically safe, but untested)

## Phase Verdict

**V6 is the stronger implementation of Phase 4**, winning 5 of 6 tasks.

### Architecture (V6 wins)

V6's core architectural choices -- typed Formatter interface, string-return methods, `baseFormatter` embedding, direct domain type usage, `func(msg string)` callback for verbose decoupling -- are individually better and compound across tasks. The typed interface alone eliminates an entire category of runtime errors and prevented the need for a later refactoring task that V5 required.

### Code Quality (V6 wins narrowly)

V5 promises more (io.Writer error propagation) but delivers less (inconsistent error checking in PrettyFormatter). V6 promises less (string returns, no error propagation) but delivers consistently. V6's compile-time interface checks, generics usage, and co-located type definitions demonstrate higher Go craftsmanship.

V5's sole code quality advantage -- proper `fmt.Errorf("%w", err)` wrapping in `writeJSON` -- is real but undermined by the inconsistent error handling elsewhere in its own formatter implementations.

### Test Quality (Mixed -- V5 wins depth, V6 wins breadth)

V5 has superior layered testing for verbose output (3 layers vs 2, with dedicated store-level tests). V6 has superior edge case coverage (nil slices, cross-formatter consistency) and more uniform table-driven structure.

This is the one area where V5 holds a genuine advantage at the phase level. V6's missing store-level verbose tests create a real gap in confidence that the verbose callback bridge works correctly.

### Spec Compliance (V6 wins narrowly)

V6 matches spec naming (`Format` not `OutputFormat`). V5 matches spec `ResolveFormat` signature verbatim. V5 matches spec timestamp examples (unquoted). V6 follows toon-go library behavior (quoted). Both make reasonable judgment calls. V6's naming compliance is more visible; V5's signature compliance is more pedantic.

### Overall

V6 demonstrates a better-designed system where tasks compose cleanly. The typed interface, baseFormatter embedding, generic helpers, and domain type reuse create a coherent whole that is greater than its parts. V5 is a competent implementation with stronger testing discipline in the verbose subsystem, but its `interface{}` debt and inconsistent error handling undermine its io.Writer design aspirations.
