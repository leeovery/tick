# Phase 4: Output Formats

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 4-1: Formatter abstraction & TTY-based format selection | V2 | Moderate over V3, large over V1 | V1 fails 2 acceptance criteria (no CLI integration, no verbose helper). V2 integrates into existing types and CLI. V3 returns `string` instead of `io.Writer`, a structural divergence that cascades through every subsequent task. |
| 4-2: TOON formatter | V2 | Narrow over V3, moderate over V1 | V2 uses toon-go properly for lists and has exact-match tests. V1 rolls its own escape function despite importing toon-go. V3's toon-go prefix manipulation is fragile. |
| 4-3: Human-readable formatter | V2 | Moderate over V1/V3 | V2 is the only version with correct right-aligned stats (`%2d`), exact string-match tests for all outputs, and no spec violations. V3 has an `Updated == Created` omission bug. |
| 4-4: JSON formatter | V2 | Large over V1/V3 | V2 is the only version with correct nested stats (`by_status` + `workflow` + `by_priority`). V1 uses flat structure. V3 merges workflow into by_status. |
| 4-5: Integrate formatters into all commands | V2 | Moderate over V1, large over V3 | V2 resolves formatter once in `Run()`, has graceful fallback, 832 LOC integration tests. V3 creates a new formatter per command (violates spec). |
| 4-6: Verbose output & edge case hardening | V2 | Narrow over V1, moderate over V3 | V2 uses a proper Go `Logger` interface for store injection. V3 has no storage-layer abstraction and logs aspirationally from CLI layer. |

**V2 wins all 6 tasks.** This is a clean sweep, though the margin varies from narrow (4-2, 4-6) to large (4-4, 4-5).

## Cross-Task Architecture Analysis

### The Formatter Interface: A Phase-Defining Decision

The single most consequential decision in Phase 4 is the Formatter interface signature defined in task 4-1. It determines the shape of every formatter implementation (4-2, 4-3, 4-4), every integration point (4-5), and even how verbose logging interacts with output (4-6). Each version made a fundamentally different choice here, and the effects are visible in every subsequent task.

**V1: `io.Writer` + `error` + value types**
```go
type Formatter interface {
    FormatTaskList(w io.Writer, tasks []TaskListItem) error
    FormatTaskDetail(w io.Writer, detail TaskDetail) error
    FormatTransition(w io.Writer, data TransitionData) error
    FormatDepChange(w io.Writer, data DepChangeData) error
    FormatStats(w io.Writer, data StatsData) error
    FormatMessage(w io.Writer, message string) error
}
```

**V2: `io.Writer` + `error` + mixed types (existing + `interface{}`)**
```go
type Formatter interface {
    FormatTaskList(w io.Writer, tasks []TaskRow) error
    FormatTaskDetail(w io.Writer, data *showData) error
    FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
    FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
    FormatStats(w io.Writer, stats interface{}) error
    FormatMessage(w io.Writer, message string) error
}
```

**V3: Returns `string`, no `io.Writer`, no `error`**
```go
type Formatter interface {
    FormatTaskList(data *TaskListData) string
    FormatTaskDetail(data *TaskDetailData) string
    FormatTransition(taskID, oldStatus, newStatus string) string
    FormatDepChange(action, taskID, blockedByID string) string
    FormatStats(data *StatsData) string
    FormatMessage(msg string) string
}
```

### How Interface Design Cascades Through All Implementations

**1. Error handling divergence (visible in 4-2, 4-3, 4-4)**

V1 and V2 return `error` from every method. In practice, V1 inconsistently ignores `fmt.Fprintf` errors in some methods (e.g., `FormatTaskList` always returns `nil`) while propagating them in others (`FormatTransition`). V2 is more consistent but still ignores write errors in helper functions like `writeRelatedSection`. The promise of error propagation is only partially fulfilled.

V3 sidesteps the issue entirely by returning `string`. This makes error handling V3's caller's problem -- but in task 4-5, the callers (`fmt.Fprint(a.Stdout, formatter.FormatTaskList(data))`) silently discard the write error from `fmt.Fprint`. The result: V3 has worse error handling than V1/V2 at the integration layer, despite cleaner signatures.

**2. The `interface{}` tax in V2 (visible in 4-2, 4-3, 4-4)**

V2's `FormatStats(w io.Writer, stats interface{})` forces every formatter to begin with a type assertion:
```go
// Repeated in toon_formatter.go, pretty_formatter.go, json_formatter.go
sd, ok := stats.(*StatsData)
if !ok {
    return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
}
```

This boilerplate appears three times across the phase. V2 also defines a separate `StatsData` struct in `toon_formatter.go` because the interface doesn't prescribe the concrete type. This is V2's single worst design choice, and it is the only consistent weakness across all three formatter tasks.

**3. String return shapes testing patterns (visible in 4-2, 4-3, 4-4, 4-5)**

V3's `string` return simplifies test setup (no `bytes.Buffer` needed):
```go
// V3 test pattern
result := formatter.FormatTaskList(data)
if !strings.Contains(result, "tasks[2]") { ... }

// V1/V2 test pattern
var buf bytes.Buffer
err := formatter.FormatTaskList(&buf, tasks)
if err != nil { t.Fatal(err) }
if !strings.Contains(buf.String(), "tasks[2]") { ... }
```

V3 tests are 2-3 lines shorter per assertion. This partly explains V3's higher test counts (49 in 4-1, 20 in 4-2, 17 in 4-3, 21 in 4-4) -- the lower ceremony per test encourages more tests. However, quantity does not equal quality: V3's tests rely heavily on `strings.Contains` partial matching, while V2 uses exact string comparison in key tests.

**4. Formatter instantiation pattern (visible in 4-5)**

V1 and V2 both have a `newFormatter(format)` factory that returns a concrete `Formatter` based on the resolved format. Both store the result on the `App` struct and resolve once in `Run()`. This is architecturally correct per the spec.

V3 puts the factory on `FormatConfig` itself:
```go
func (c FormatConfig) Formatter() Formatter {
    switch c.Format {
    case FormatJSON: return &JSONFormatter{}
    case FormatPretty: return &PrettyFormatter{}
    default: return &ToonFormatter{}
    }
}
```

This is called per-command: `formatter := a.formatConfig.Formatter()` appears in `list.go`, `show.go`, `create.go`, `blocked.go`, `ready.go`, `transition.go`, `dep.go`, and `update.go`. The formatters are stateless so this doesn't cause bugs, but it violates the spec requirement that format is resolved once in the dispatcher, and it is a missed opportunity for a single assignment like V1/V2.

### Shared Patterns Across Pretty/Toon/JSON Formatters

**All three versions share these structural patterns across their formatters:**

1. **Empty-to-sentinel mapping**: Every formatter maps `nil`/empty lists to a format-appropriate sentinel (`tasks[0]{...}:` for TOON, `"No tasks found."` for Pretty, `[]` for JSON). This logic is independently implemented in each formatter with no shared helper.

2. **Dynamic schema for detail output**: All versions build a dynamic field list for task detail, conditionally including `parent` and `closed`. The algorithm is identical -- build a `[]string` of field names, conditionally `append`:
```go
// Pattern shared across all 3 versions, all 3 formatters
if parent != "" {
    fields = append(fields, "parent")
}
fields = append(fields, "created", "updated")
if closed != "" {
    fields = append(fields, "closed")
}
```

3. **Priority label table**: All versions maintain a mapping of priority integers to human labels (P0=critical through P4=low) for pretty and stats output. V1 uses a `[5]string` array, V2 hardcodes each line, V3 uses a `map[int]string`. None share this mapping across formatters.

4. **JSON struct mirroring**: All three versions define parallel "json-specific" struct types (`jsonTaskDetail`, `jsonTaskRow`, etc.) that mirror the formatter input types but with `json:"snake_case"` tags. This duplication is unavoidable in Go's JSON serialization model but means each version maintains two parallel type hierarchies.

### Data Type Strategy: Fresh vs. Reuse

V1 and V3 define **fresh data types** for the formatter interface (`TaskListItem`, `TaskDetail`, `RelatedTask` for V1; `TaskListData`, `TaskDetailData`, `RelatedTaskData` for V3). These types exist solely for the formatting layer and must be populated from the domain types in task 4-5.

V2 **reuses existing types** where possible (`*showData` from `show.go`, `task.Status` from the domain). This reduces the type surface area but creates tighter coupling -- the `showData` type from Phase 1 is now part of the public formatter interface, meaning changes to show output require coordinated changes with the formatter.

In task 4-5 (integration), this choice has a measurable impact:
- V1 builds fresh `TaskDetail` structs in a shared helper `queryAndFormatTaskDetail`
- V2 reuses the existing `queryShowData` function and passes `*showData` directly to formatters
- V3 builds fresh `TaskDetailData` structs inline in each command handler (most verbose, least DRY)

V2's reuse approach requires fewer lines of conversion code in 4-5 but the `interface{}` parameter on `FormatStats` undermines the type-reuse benefit.

### Format Resolution Flow

The format resolution pipeline spans tasks 4-1 and 4-5:

```
TTY Detection -> Flag Parsing -> ResolveFormat -> newFormatter -> Dispatch
```

V1 and V2 execute this pipeline in `Run()` before the command switch. V3 splits it: `NewFormatConfig` runs early (TTY + flags), but `Formatter()` runs per-command. This means V3's pipeline has a gap in the middle where the format is resolved but the formatter is not yet created.

### Verbose Integration Architecture (4-6)

The verbose system intersects with formatters at exactly one point: logging the resolved format. All versions log something like `format=toon (auto-detected)` or `format resolved: pretty`. But the architecture differs:

- V1/V2 inject a logger into the storage layer, so verbose output accurately reflects internal operations (lock acquired, cache checked, etc.)
- V3 wraps storage calls with verbose logging at the CLI layer, producing messages like `"lock acquire exclusive"` *before* the lock is actually acquired

This is architecturally significant: V1/V2's approach is correct by construction, while V3's approach is correct only if the storage operations succeed. If a lock acquisition fails, V3 still logs "lock acquire exclusive."

## Code Quality Patterns

### Interface Consistency

**V2 is the most internally consistent** despite its `interface{}` wart. It consistently:
- Writes to `io.Writer` in all formatters
- Returns `error` from all methods
- Uses compile-time `var _ Formatter = &ToonFormatter{}` checks in all formatter files
- Places `newFormatter` factory in a single location
- Uses `marshalIndentTo` as a shared JSON helper

**V1 is clean but disconnected from the CLI.** Its format types are self-contained but create a parallel type universe that requires conversion code.

**V3 has the most internal inconsistency:**
- Returns `string` from formatters but callers wrap in `fmt.Fprint` (two-step output)
- No compile-time interface check in production code (only tests)
- Factory method on `FormatConfig` instead of standalone function
- Verbose prefix changed from `[verbose]` to `verbose:` mid-phase (task 4-6 fixes task 4-1's choice)

### DRY Across Formatters

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| JSON serialization helper | `jsonWrite` (shared) | `marshalIndentTo` (shared) | `json.MarshalIndent` repeated 6 times |
| Related section output (TOON) | Inline | `writeRelatedSection` helper | Inline |
| Task detail query (4-5) | `queryAndFormatTaskDetail` shared | `queryShowData` shared | Inline per command |
| Store creation (4-6) | `openStore` helper | `newStore` helper | Direct `storage.NewStore` per command |
| Verbose logging in commands | Centralized in store | Centralized in store | Copy-pasted 4-6 lines per handler |

V2 is the DRYest across the phase. V3 is consistently the most repetitive, with identical patterns copy-pasted across multiple files.

### Error Handling

Three distinct philosophies:

1. **V1: Inconsistent propagation.** Some methods return `fmt.Fprintf` errors, others return `nil`. The intent to propagate is there but the execution is spotty.

2. **V2: Consistent propagation with wrapping.** Errors are wrapped with context (`"json marshal failed: %w"`, `"FormatStats: expected *StatsData, got %T"`). The `marshalIndentTo` helper propagates both marshal and write errors.

3. **V3: Swallow and fallback.** JSON errors return `"[]"` or `"{}"`. TOON errors trigger manual fallback formatters. No error ever reaches the caller. This is the simplest model but hides failures.

### Action Verb Inconsistency

A subtle cross-task pattern: the `FormatDepChange` action values differ across versions:
- V1: `"added"` / `"removed"` (past tense)
- V2: `"added"` / `"removed"` (past tense) with `default` case
- V3: `"add"` / `"remove"` (present tense)

This reflects different assumptions about who is responsible for the verb tense -- the caller or the formatter. V3's present-tense approach means the formatter constructs the message ("Dependency added:..." from action="add"), while V1/V2 pass the display-ready verb.

## Test Coverage Analysis

### Aggregate Test Counts

| Task | V1 Tests | V2 Tests | V3 Tests |
|------|----------|----------|----------|
| 4-1: Abstraction | 11 | 25 | 49 |
| 4-2: TOON | 14 | 16 | 20 |
| 4-3: Pretty | 10 | 15 | 17 |
| 4-4: JSON | 12 | 14+6 table | 21 |
| 4-5: Integration | 15 | 21 | 28 |
| 4-6: Verbose | 10 | 12 | 10 |
| **Phase Total** | **72** | **109** | **145** |

### Test LOC

| Task | V1 LOC | V2 LOC | V3 LOC |
|------|--------|--------|--------|
| 4-1 | 151 | 280 | 547 |
| 4-2 | 288 | 396 | 609 |
| 4-3 | 228 | 439 | 676 |
| 4-4 | 289 | 668 | 702 |
| 4-5 | 226 | 832 | 637 |
| 4-6 | 162 | 259 | 253 |
| **Phase Total** | **1,344** | **2,874** | **3,424** |

V3 writes the most tests by a significant margin (2.5x V1, 1.2x V2). However, **test quality favors V2**: V2 consistently uses exact string matching for output verification, while V3 relies on `strings.Contains` partial matches. V3's volume comes partly from the lower ceremony of string-return testing and from dedicated spec-example tests that are, ironically, weaker than V2's exact-match approach.

### Test Strategy Comparison

- **V2's strategy**: Fewer but stronger assertions. Exact output matching catches spacing, alignment, and ordering regressions. Table-driven tests for combinatorial coverage (e.g., JSON valid-across-all-methods).

- **V3's strategy**: More tests, partial matching. Dedicated `SpecExamples` tests in 4-2 and 4-3 verify spec compliance but use `strings.Contains` which can pass even with wrong output structure. Higher coverage of edge cases (nil inputs, orthogonality tests).

- **V1's strategy**: Minimal but targeted. Misses several important test categories (no compile-time checks, no exact output matches, gaps in FormatMessage coverage). Weakest by a clear margin.

### Patterns Across All Test Suites

All three versions test the same core scenarios:
- TTY pipe detection defaults
- Format flag overrides (all 3 flags)
- Conflicting flags produce errors
- Empty list output per format
- Task detail with all/minimal fields
- Stats with priority breakdown
- Quiet overrides per command type

V2 uniquely tests:
- Stats type assertion error (consequence of `interface{}`)
- `nil` input producing `[]` for JSON lists (tests the Go nil-slice gotcha)
- Format resolved once in dispatcher (structural assertion)
- All 6 JSON methods produce valid JSON (table-driven)

V3 uniquely tests:
- Special character round-trips in JSON (quotes, backslashes, newlines)
- StubFormatter all 6 methods individually
- Quiet and verbose orthogonal to format selection
- Errors always plain text to stderr

## Phase Verdict

**V2 is the clear winner of Phase 4, winning all 6 tasks.**

### Summary of V2 Strengths

1. **Correct spec compliance across all tasks.** V2 is the only version with properly nested JSON stats (`by_status` + `workflow` + `by_priority`), correct right-aligned pretty stats, proper toon-go usage for list marshaling, single-point formatter resolution, and accurate verbose instrumentation inside the storage layer.

2. **Best interface design with one exception.** The `io.Writer` + `error` pattern follows Go conventions, enables streaming, and naturally propagates errors. The compile-time interface checks are consistently present. The single weakness -- `FormatStats(w io.Writer, stats interface{})` -- is contained and compensated with runtime error handling.

3. **Strongest integration architecture.** V2 reuses existing types (`*showData`, `task.Status`), centralizes store creation, resolves the formatter once, and provides graceful fallback in create/update. This shows the deepest understanding of how the formatting system fits into the broader application.

4. **Best test quality.** At 2,874 test LOC, V2 strikes the right balance between coverage and rigor. Its exact string-match tests for all three formatters (4-2, 4-3, 4-4) are strictly stronger than V1's partial matching or V3's `strings.Contains` approach. The table-driven valid-JSON test across all 6 methods is exemplary.

5. **Best DRY discipline.** Shared helpers (`marshalIndentTo`, `writeRelatedSection`, `queryShowData`, `newStore`) reduce duplication across the phase. V3 is notably weak here, with 6 repeated `json.MarshalIndent` calls and copy-pasted verbose logging across 8+ files.

### Where V2 Falls Short

- The `interface{}` parameter on `FormatStats` is the single worst type-safety decision in the phase, forcing runtime assertions in 3 formatter files.
- V2's exported `App.FormatCfg` field leaks internal state.
- Test count (109) is lower than V3 (145), missing some edge cases V3 covers (special characters in JSON, stub method coverage).

### V3: Strong Testing, Weak Architecture

V3 writes the most tests (145 across the phase, 3,424 LOC) and catches edge cases the others miss. But its architectural choices undermine the overall design: string-returning formatters break Go streaming patterns, per-command formatter creation violates the spec, CLI-layer verbose logging is semantically inaccurate, and copy-paste across handlers creates maintenance burden. V3 consistently prioritizes local simplicity over system-level coherence.

### V1: Clean but Incomplete

V1 is the most concise and readable version. Its code is well-structured when present but frequently incomplete: task 4-1 fails 2 acceptance criteria, stats are flat instead of nested in 4-4, and test coverage is the weakest across every task. V1 treats each task as an isolated module rather than part of an integrated system.

### Final Ranking

1. **V2** -- Best architecture, best spec compliance, best test quality, wins all 6 tasks
2. **V3** -- Most thorough testing, but architectural choices cascade into spec violations and maintenance burden
3. **V1** -- Cleanest code style, but consistently incomplete in both implementation and testing
