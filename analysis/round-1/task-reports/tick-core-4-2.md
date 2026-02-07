# Task tick-core-4-2: TOON formatter -- list, show, stats output

## Task Summary

Implement a concrete `ToonFormatter` that satisfies the `Formatter` interface, producing TOON (Token-Oriented Object Notation) output optimized for AI agent consumption with 30-60% token savings over JSON.

**Required methods:**
- **FormatTaskList**: `tasks[N]{id,title,status,priority}:` header + indented data rows. Zero tasks produce `tasks[0]{...}:` with no rows.
- **FormatTaskDetail**: Multi-section output. Dynamic schema (omit `parent`/`closed` when null). `blocked_by`/`children` always present (even `[0]`). Description omitted when empty; multiline rendered as indented lines.
- **FormatStats**: `stats` summary section + `by_priority` section (always 5 rows, priorities 0-4).
- **FormatTransition / FormatDepChange / FormatMessage**: Plain text passthrough.
- **Escaping**: Commas in titles delegated to `toon-go` library.

**Acceptance Criteria:**
1. Implements full Formatter interface
2. List output matches spec TOON format exactly
3. Show output multi-section with dynamic schema
4. blocked_by/children always present, description conditional
5. Stats produces summary + 5-row by_priority
6. Escaping handled by toon-go
7. All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Implements full Formatter interface | PASS -- `ToonFormatter` has all 6 methods matching V1's `Formatter` interface signatures (`io.Writer`, error return) | PASS -- all 6 methods matching V2's `Formatter` interface (`io.Writer`, error return, uses `task.Status` for transition). Explicit compile-time check via `var _ Formatter = &ToonFormatter{}` | PASS -- all 6 methods matching V3's `Formatter` interface (string return, no `io.Writer`). Compile-time check via `var _ Formatter = &ToonFormatter{}` |
| List output matches spec TOON format exactly | PASS -- produces `tasks[2]{id,title,status,priority}:\n  tick-a1b2,...` matching spec | PASS -- uses `toon.MarshalString` for non-empty lists; produces identical output to spec. Empty list handled separately with manual header. | PARTIAL -- uses `toon.MarshalString(rows, toon.WithIndent(2))` then prepends "tasks" prefix. This depends on toon-go producing `[N]{...}:` format which may or may not match spec exactly. Has manual fallback. |
| Show output multi-section with dynamic schema | PASS -- builds schema/values slices dynamically, adds parent/closed conditionally | PASS -- identical approach: `schema`/`values` slices, conditional append for parent/closed | PASS -- uses `buildTaskSchema`/`buildTaskRow` helper methods for cleaner separation |
| blocked_by/children always present, description conditional | PASS -- always writes `blocked_by[N]` and `children[N]`; description gated on `d.Description != ""` | PASS -- same logic with `writeRelatedSection` helper | PASS -- same logic, inline |
| Stats produces summary + 5-row by_priority | PASS -- hardcoded `[5]int` array, loop 0..4 | PASS -- `[5]int` array, loop 0..4. But `FormatStats` takes `interface{}` and type-asserts to `*StatsData` (V2 defines its own `StatsData` in this file) | PASS -- `[]PriorityCount` slice (not `[5]int`), iterates over `data.ByPriority`. Count header uses `len(data.ByPriority)` rather than hardcoded 5 |
| Escaping handled by toon-go | PARTIAL -- implements its own `toonEscape` function (manual quote-wrapping), does NOT use `toon-go` library despite importing it in `go.mod` | PARTIAL -- uses `toon.MarshalString` for `FormatTaskList` (non-empty case), but `escapeField` uses toon-go as a roundabout wrapper that marshals then parses output. Has manual fallback. | PARTIAL -- `escapeValue` calls `toon.MarshalString(s)` on a raw string, trims trailing newline. Has manual fallback. `FormatTaskList` also tries `toon.MarshalString(rows)` but with dubious prefix manipulation. |
| All output matches spec examples | PASS -- output format matches spec examples exactly based on test assertions | PASS -- full exact-match tests against spec examples | PASS -- dedicated `TestToonFormatterSpecExamples` with 3 subtests verifying exact spec format |

## Implementation Comparison

### Approach

All three versions solve the same problem but differ significantly in their Formatter interface signatures, file naming, type usage, and toon-go integration strategy.

#### Formatter Interface Signatures

**V1** uses the interface from commit `9e294b6`:
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
Methods accept `io.Writer` and return `error`. Data types use exported structs (`TaskListItem`, `TaskDetail`, `TransitionData`, `DepChangeData`, `StatsData`) defined in the same file. `Parent` is `*RelatedTask` (pointer, nil when absent). `StatsData.ByPriority` is `[5]int`.

**V2** uses the interface from commit `e10ae58`:
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
Notably: `FormatTaskDetail` takes `*showData` (unexported type from `show.go`). `FormatTransition` takes separate string args with `task.Status` type. `FormatStats` takes `interface{}` requiring runtime type assertion. `FormatDepChange` takes separate strings. V2 defines its own `StatsData` struct in `toon_formatter.go` because the interface uses `interface{}`.

**V3** uses the interface from commit `977a3c2`:
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
Returns `string` instead of writing to `io.Writer`. No `error` return. Uses dedicated data wrapper types (`TaskListData`, `TaskDetailData`, `StatsData`). `StatsData.ByPriority` is `[]PriorityCount` (slice of structs, not fixed array).

#### File Naming

- **V1**: `format_toon.go` / `format_toon_test.go` -- follows pattern from `format.go`
- **V2**: `toon_formatter.go` / `toon_formatter_test.go` -- follows pattern from `formatter.go`
- **V3**: `toon_formatter.go` / `toon_formatter_test.go` -- same as V2

#### FormatTaskList Implementation

**V1** (lines 14-20 of `format_toon.go`) -- Direct `fmt.Fprintf`:
```go
func (f *ToonFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
    fmt.Fprintf(w, "tasks[%d]{id,title,status,priority}:\n", len(tasks))
    for _, t := range tasks {
        fmt.Fprintf(w, "  %s,%s,%s,%d\n", t.ID, toonEscape(t.Title), t.Status, t.Priority)
    }
    return nil
}
```
Simple, direct. No special handling for zero tasks -- the loop simply doesn't execute and the header is emitted. Error return from `fmt.Fprintf` is silently ignored.

**V2** (lines 47-62 of `toon_formatter.go`) -- Uses toon-go for non-empty, manual for empty:
```go
func (f *ToonFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
    if len(tasks) == 0 {
        _, err := fmt.Fprint(w, "tasks[0]{id,title,status,priority}:\n")
        return err
    }
    rows := make([]toonListRow, len(tasks))
    for i, t := range tasks {
        rows[i] = toonListRow{ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority}
    }
    out, err := toon.MarshalString(toonListWrapper{Tasks: rows})
    if err != nil {
        return fmt.Errorf("toon marshal failed: %w", err)
    }
    _, err = fmt.Fprint(w, out+"\n")
    return err
}
```
Converts to internal `toonListRow` structs with toon tags, wraps in `toonListWrapper`, marshals via `toon.MarshalString`. The output of toon-go is used directly. Defines two helper types (`toonListRow`, `toonListWrapper`) specifically for this marshaling.

**V3** (lines 31-63 of `toon_formatter.go`) -- Returns string, tries toon-go with prefix manipulation:
```go
func (f *ToonFormatter) FormatTaskList(data *TaskListData) string {
    if data == nil || len(data.Tasks) == 0 {
        return "tasks[0]{id,title,status,priority}:\n"
    }
    rows := make([]toonTaskRow, len(data.Tasks))
    for i, task := range data.Tasks {
        rows[i] = toonTaskRow{...}
    }
    result, err := toon.MarshalString(rows, toon.WithIndent(2))
    if err != nil {
        return f.formatTaskListManual(data)
    }
    result = "tasks" + result
    if !strings.HasSuffix(result, "\n") {
        result += "\n"
    }
    return result
}
```
Attempts `toon.MarshalString` on a slice (not a named wrapper), then prepends "tasks" prefix assuming toon-go outputs `[N]{...}:`. Has a full `formatTaskListManual` fallback method. Also handles `nil` data. Null safety is better but the toon-go prefix manipulation is fragile.

#### FormatTaskDetail Implementation

All three versions use fundamentally the same algorithm: build a dynamic schema by conditionally appending "parent" and "closed" fields, then output each section.

**V1** (lines 23-60) -- Inline, uses `*RelatedTask` pointer for parent:
```go
if d.Parent != nil {
    fields = append(fields, "parent")
    values = append(values, d.Parent.ID)
}
```
Parent is checked via nil pointer, which is semantically clean.

**V2** (lines 68-112) -- Uses `*showData` with string `Parent`:
```go
if data.Parent != "" {
    schema = append(schema, "parent")
    values = append(values, data.Parent)
}
```
Extracts `writeRelatedSection` as a reusable helper function:
```go
func writeRelatedSection(w io.Writer, name string, items []relatedTask) {
    fmt.Fprintf(w, "%s[%d]{id,title,status}:\n", name, len(items))
    for _, item := range items {
        fmt.Fprintf(w, "  %s,%s,%s\n", item.ID, escapeField(item.Title), item.Status)
    }
}
```

**V3** (lines 75-111) -- Uses `*TaskDetailData`, extracts `buildTaskSchema` and `buildTaskRow` helpers:
```go
func (f *ToonFormatter) buildTaskSchema(data *TaskDetailData) string {
    fields := []string{"id", "title", "status", "priority"}
    if data.Parent != "" {
        fields = append(fields, "parent")
    }
    fields = append(fields, "created", "updated")
    if data.Closed != "" {
        fields = append(fields, "closed")
    }
    return strings.Join(fields, ",")
}
```
Better separation of concerns with private methods on the struct.

#### FormatStats Implementation

**V1** (lines 82-95) -- Direct, uses `StatsData` value type with `[5]int`:
```go
func (f *ToonFormatter) FormatStats(w io.Writer, data StatsData) error {
    fmt.Fprint(w, "stats{total,open,in_progress,done,cancelled,ready,blocked}:\n")
    fmt.Fprintf(w, "  %d,%d,%d,%d,%d,%d,%d\n", ...)
    fmt.Fprint(w, "\nby_priority[5]{priority,count}:\n")
    for i := 0; i < 5; i++ {
        fmt.Fprintf(w, "  %d,%d\n", i, data.ByPriority[i])
    }
    return nil
}
```

**V2** (lines 136-159) -- Takes `interface{}`, type-asserts:
```go
func (f *ToonFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
    ...
}
```
V2 must define its own `StatsData` struct (lines 13-23) because the interface uses `interface{}`. This is a consequence of the V2 interface design forcing a runtime type assertion.

**V3** (lines 167-187) -- Uses `*StatsData` with `[]PriorityCount`:
```go
sb.WriteString(fmt.Sprintf("\nby_priority[%d]{priority,count}:\n", len(data.ByPriority)))
for _, pc := range data.ByPriority {
    sb.WriteString(fmt.Sprintf("  %d,%d\n", pc.Priority, pc.Count))
}
```
The `by_priority` count is dynamic (`len(data.ByPriority)`) rather than hardcoded `5`. This means the count in the header always matches the actual number of rows, but it does not guarantee exactly 5 rows -- that responsibility falls on the caller.

#### Escaping Strategy

**V1** -- Implements its own `toonEscape` function (lines 103-114):
```go
func toonEscape(s string) string {
    needsQuote := strings.ContainsAny(s, ",\n\r") ||
        (len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t'))
    if !needsQuote {
        return s
    }
    escaped := strings.ReplaceAll(s, `"`, `""`)
    return `"` + escaped + `"`
}
```
Does NOT actually use toon-go for escaping despite importing it. Uses TOON doubling convention (`""`) for embedded quotes.

**V2** -- `escapeField` uses toon-go indirectly (lines 177-192):
```go
func escapeField(s string) string {
    if !strings.ContainsAny(s, ",\"\n\\") {
        return s
    }
    type wrapper struct {
        V string `toon:"v"`
    }
    out, err := toon.MarshalString(wrapper{V: s})
    if err != nil {
        return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
    }
    return strings.TrimPrefix(out, "v: ")
}
```
Creates a throwaway struct, marshals it, then parses the result to extract the escaped value. This is a roundabout use of toon-go. The fallback uses backslash escaping (`\"`) which differs from TOON's doubling convention.

**V3** -- `escapeValue` uses toon-go directly on a string (lines 201-216):
```go
func escapeValue(s string) string {
    escaped, err := toon.MarshalString(s)
    if err != nil {
        if strings.Contains(s, ",") {
            return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
        }
        return s
    }
    escaped = strings.TrimSuffix(escaped, "\n")
    return escaped
}
```
Marshals a bare string directly. The fallback also uses backslash escaping. The behavior depends on what `toon.MarshalString` does with a raw `string` input.

#### FormatDepChange Action Values

- **V1**: Checks `data.Action == "added"` (past tense)
- **V2**: Checks `action` as "added" / "removed" (past tense), with a default case for unknown actions
- **V3**: Checks `action == "add"` (present tense)

This is a subtle but important difference that reflects the different interface designs across the three worktrees.

### Code Quality

#### Go Idioms and Naming

**V1**: File named `format_toon.go`. Uses `toonEscape` (unexported, file-scoped helper). Clean, minimal struct with no extra types beyond what's needed. The `ToonFormatter` struct has no fields, which is idiomatic for stateless formatters.

**V2**: File named `toon_formatter.go`. Introduces 3 helper types (`toonListRow`, `toonListWrapper`, `StatsData`) and 2 helper functions (`writeRelatedSection`, `escapeField`). The `StatsData` definition is duplicated from what would normally be in the interface file -- this is a consequence of the `interface{}` parameter in `FormatStats`. Good use of `writeRelatedSection` helper for DRY.

**V3**: File named `toon_formatter.go`. Introduces 1 helper type (`toonTaskRow`) and private methods (`buildTaskSchema`, `buildTaskRow`, `formatTaskListManual`). Best separation of concerns with the schema/row builders as methods on the formatter.

#### Error Handling

**V1**: Consistently returns `nil` from `FormatTaskList`, `FormatTaskDetail`, and `FormatStats`, silently ignoring `fmt.Fprintf` errors. Only `FormatTransition`, `FormatDepChange`, and `FormatMessage` return write errors. This is inconsistent.

**V2**: Better error propagation overall. `FormatTaskList` returns marshaling errors. `FormatStats` returns type assertion errors. `writeRelatedSection` does not return errors (same issue as V1 for those writes). `escapeField` has a fallback on error rather than propagating.

**V3**: Returns `string`, so there are no write errors to handle. Nil checks on data params (`if data == nil`). Error from `toon.MarshalString` triggers fallback rather than propagation. Clean approach -- errors are avoided by design.

#### Type Safety

**V1**: All methods use concrete, exported types. Fully type-safe at compile time.

**V2**: `FormatStats(w io.Writer, stats interface{})` requires a runtime type assertion:
```go
sd, ok := stats.(*StatsData)
if !ok {
    return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
}
```
This is the weakest type safety of the three. A wrong type passed at call site is only caught at runtime.

**V3**: All methods use concrete pointer types. Fully type-safe at compile time.

### Test Quality

#### V1 Test Functions (5 top-level, 14 subtests)

| Top-level Function | Subtests |
|---|---|
| `TestToonFormatterFormatTaskList` | "it formats list with correct header count and schema", "it formats zero tasks as empty section", "it escapes commas in titles" |
| `TestToonFormatterFormatTaskDetail` | "it formats show with all sections", "it omits parent/closed from schema when null", "it renders blocked_by/children with count 0 when empty", "it omits description section when empty", "it renders multiline description as indented lines", "it includes closed in schema when present" |
| `TestToonFormatterFormatStats` | "it formats stats with all counts", "it formats by_priority with 5 rows including zeros" |
| `TestToonFormatterFormatTransition` | "it formats transition as plain text" |
| `TestToonFormatterFormatDepChange` | "it formats dep add as plain text", "it formats dep rm as plain text" |

**Assertion style**: Uses `strings.HasPrefix`, `strings.Contains` for partial matching. The "by_priority with 5 rows" test has a dead loop that checks nothing meaningful (lines 225-228):
```go
for i := 0; i < 5; i++ {
    if !strings.Contains(out, "  "+strings.TrimSpace(strings.Repeat(" ", 0))+string(rune('0'+i))+",") {
        // Check each priority row exists
    }
}
```
This loop body is empty -- the comment is the only thing inside the `if`. The actual assertions are separate checks for `"0,0"` and `"2,5"`.

**Missing**: No `FormatMessage` test. No compile-time interface check. No exact full-output matching (uses partial `Contains` checks).

#### V2 Test Functions (5 top-level, 16 subtests)

| Top-level Function | Subtests |
|---|---|
| `TestToonFormatterFormatTaskList` | "it formats list with correct header count and schema", "it formats zero tasks as empty section", "it escapes commas in titles" |
| `TestToonFormatterFormatTaskDetail` | "it formats show with all sections", "it omits parent/closed from schema when null", "it renders blocked_by/children with count 0 when empty", "it omits description section when empty", "it renders multiline description as indented lines", "it includes closed in schema when present" |
| `TestToonFormatterFormatStats` | "it formats stats with all counts", "it formats by_priority with 5 rows including zeros" |
| `TestToonFormatterFormatTransitionAndDep` | "it formats transition as plain text", "it formats dep add as plain text", "it formats dep removed as plain text", "it formats message as plain text" |
| `TestToonFormatterImplementsInterface` | "it implements the full Formatter interface" |

**Assertion style**: Uses exact string comparison (`buf.String() != want`) for most tests. The "show with all sections" test builds the full expected string and compares exactly. This is strictly stronger than V1's partial matching.

**Unique to V2**: `FormatMessage` test. Compile-time interface check (`var _ Formatter = &ToonFormatter{}`). Exact full-output matching for detail, stats, and list.

#### V3 Test Functions (8 top-level, 20 subtests)

| Top-level Function | Subtests |
|---|---|
| `TestToonFormatter` | "it implements Formatter interface" |
| `TestToonFormatterFormatTaskList` | "it formats list with correct header count and schema", "it formats zero tasks as empty section", "it escapes commas in titles" |
| `TestToonFormatterFormatTaskDetail` | "it formats show with all sections", "it omits parent/closed from schema when null", "it renders blocked_by/children with count 0 when empty", "it omits description section when empty", "it renders multiline description as indented lines", "it includes parent in schema when set", "it includes closed in schema when set" |
| `TestToonFormatterFormatStats` | "it formats stats with all counts", "it formats by_priority with 5 rows including zeros" |
| `TestToonFormatterFormatTransition` | "it formats transition/dep as plain text" |
| `TestToonFormatterFormatDepChange` | "it formats dep add as plain text", "it formats dep rm as plain text" |
| `TestToonFormatterFormatMessage` | "it formats message as plain text" |
| `TestToonFormatterSpecExamples` | "it matches spec list example format", "it matches spec show example format", "it matches spec stats example format" |

**Assertion style**: Mixed -- `HasPrefix`/`Contains` for basic tests, plus exact line-by-line verification in `TestToonFormatterSpecExamples`. The spec example tests do exact line matching after `strings.Split`:
```go
lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
if lines[0] != "tasks[2]{id,title,status,priority}:" {
    t.Errorf("header mismatch: %q", lines[0])
}
```

**Unique to V3**: Separate `TestToonFormatterSpecExamples` with 3 subtests that verify exact spec compliance for list, show, and stats examples. "it includes parent in schema when set" as a separate test (V1/V2 test parent presence only within the "all sections" test). The by_priority test explicitly counts priority lines:
```go
var priorityLines []string
inPriority := false
for _, line := range lines {
    if strings.Contains(line, "by_priority") { inPriority = true; continue }
    if inPriority && strings.HasPrefix(line, "  ") { priorityLines = append(priorityLines, line) }
}
if len(priorityLines) != 5 { ... }
```

#### Test Coverage Comparison

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| List with 2 tasks | Yes | Yes | Yes |
| Zero tasks empty section | Yes | Yes | Yes |
| Commas in titles | Yes | Yes | Yes |
| Show with all sections | Yes | Yes | Yes |
| Parent omitted when null | Yes | Yes | Yes |
| Closed omitted when null | Yes (within parent test) | Yes (within parent test) | Yes (within parent test) |
| Closed included when set | Yes | Yes | Yes |
| Parent included when set | Tested within "all sections" | Tested within "all sections" | Dedicated test |
| blocked_by/children count 0 | Yes | Yes | Yes |
| Description omitted when empty | Yes | Yes | Yes |
| Multiline description | Yes | Yes | Yes |
| Stats with all counts | Yes | Yes | Yes |
| by_priority 5 rows with zeros | Yes (weak assertion) | Yes (exact match) | Yes (line counting) |
| Transition plain text | Yes | Yes | Yes |
| Dep add plain text | Yes | Yes | Yes |
| Dep remove plain text | Yes | Yes | Yes |
| Message plain text | **NO** | Yes | Yes |
| Interface compile check | **NO** | Yes | Yes |
| Spec example exact match | **NO** | **NO** (but uses exact string comparison) | Yes (3 dedicated tests) |
| Nil data handling | **NO** | **NO** | Implicit (code has nil checks) |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 4 | 6 | 7 |
| Lines added | 405 | 597 | 847 |
| Impl LOC | 114 | 192 | 216 |
| Test LOC | 288 | 396 | 609 |
| Top-level test functions | 5 | 5 | 8 |
| Subtests | 14 | 16 | 20 |

## Verdict

**V2 is the best overall implementation**, with V3 having the best tests.

**V1** is the most compact and readable implementation (114 LOC) but has the weakest test coverage: it misses `FormatMessage` testing, has no compile-time interface check, relies on partial string matching rather than exact output comparison, and contains a dead test loop. Its custom `toonEscape` function does NOT actually use toon-go despite importing it, which directly violates acceptance criterion 6 ("Escaping handled by toon-go"). However, its escaping logic is correct and handles TOON's quote-doubling convention properly.

**V2** strikes the best balance between implementation quality and test rigor. It properly uses `toon.MarshalString` for the list formatter (satisfying the toon-go requirement), has exact string matching in tests, includes compile-time interface verification, and tests `FormatMessage`. The `writeRelatedSection` helper is good DRY practice. Its main weakness is the `interface{}` parameter on `FormatStats` requiring a runtime type assertion -- but this is a consequence of the underlying interface design from task 4-1, not this task's implementation.

**V3** has the most thorough test suite (609 LOC, 20 subtests), including dedicated spec-example verification tests that directly validate acceptance criterion 7. It has the cleanest separation of concerns with `buildTaskSchema`/`buildTaskRow` methods. However, its toon-go integration for `FormatTaskList` is the most fragile -- it relies on manipulating toon-go's output format by prepending "tasks" to the marshaled string, which assumes a specific toon-go output structure. Its `escapeValue` function's behavior depends on marshaling a bare `string` through toon-go, which is an undocumented usage pattern. The `by_priority` section using `len(data.ByPriority)` rather than hardcoded `5` is arguably more flexible but shifts the "always 5 rows" invariant to the caller.

**Recommendation**: Take V2's implementation as the base, adopt V3's `TestToonFormatterSpecExamples` test pattern, and ensure V1's missing `FormatMessage` test gap is covered (which V2 already does).
