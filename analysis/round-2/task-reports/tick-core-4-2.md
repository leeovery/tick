# Task tick-core-4-2: TOON Formatter

## Task Summary

Implement a concrete `ToonFormatter` for the `Formatter` interface that produces TOON (Token-Oriented Object Notation) output -- an agent-facing format providing 30-60% token savings over JSON. The implementation must cover:

- **FormatTaskList**: `tasks[N]{id,title,status,priority}:` + indented data rows. Zero tasks produce `tasks[0]{...}:`.
- **FormatTaskDetail**: Multi-section output. Dynamic schema (omit parent/closed when null). blocked_by/children always present (even `[0]`). Description omitted when empty, multiline as indented lines.
- **FormatStats**: stats summary + by_priority (always 5 rows, 0-4).
- **FormatTransition/FormatDepChange/FormatMessage**: plain text passthrough.
- Escaping handled via `github.com/toon-format/toon-go`.

### Acceptance Criteria

1. Implements full Formatter interface
2. List output matches spec TOON format exactly
3. Show output multi-section with dynamic schema
4. blocked_by/children always present, description conditional
5. Stats produces summary + 5-row by_priority
6. Escaping handled by toon-go
7. All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS -- compile-time check `var _ Formatter = &ToonFormatter{}` in test | PASS -- identical compile-time check in test |
| List output matches spec TOON format exactly | PASS -- exact string match test against spec example | PASS -- exact string match test against spec example |
| Show output multi-section with dynamic schema | PASS -- dynamic schema building for parent/closed | PASS -- dynamic schema building for parent/closed, also adds parent_title field |
| blocked_by/children always present, description conditional | PASS -- tested for [0] count and description omission | PASS -- tested for [0] count and description omission |
| Stats produces summary + 5-row by_priority | PASS -- exact string match test | PASS -- `strings.Contains` based checks (less strict) |
| Escaping handled by toon-go | PASS -- uses toon-go MarshalString for both list (via struct tags) and field escaping | PASS -- uses toon-go Object API for list and toonEscapeValue for fields |
| All output matches spec examples | PASS -- test assertions match spec examples exactly | PARTIAL -- show output includes `parent_title` field not in spec; stats tests use Contains not exact match |

## Implementation Comparison

### Approach

Both versions create a `ToonFormatter` struct implementing all 6 Formatter methods. The high-level architecture is similar, but they diverge significantly in three areas: (1) how they interact with the toon-go library, (2) the Formatter interface signatures, and (3) how they structure the code internally.

#### Formatter Interface Differences

V2 preserves the existing interface signatures unchanged. The Formatter interface uses:
- `FormatTaskList(w io.Writer, tasks []TaskRow) error` -- takes exported `TaskRow` type
- `FormatTaskDetail(w io.Writer, data *showData) error` -- takes pointer to unexported `showData`
- `FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error` -- uses `task.Status` type
- `FormatDepChange(w io.Writer, action, taskID, blockedByID string) error` -- simple string params
- `FormatStats(w io.Writer, stats interface{}) error` -- uses `interface{}` for stats data

V4 modifies the Formatter interface as part of this task, changing signatures:
- `FormatTaskList(w io.Writer, rows []listRow, quiet bool) error` -- takes unexported `listRow`, adds `quiet` param
- `FormatTaskDetail(w io.Writer, detail TaskDetail) error` -- takes exported `TaskDetail` by value (not pointer)
- `FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error` -- uses plain strings instead of `task.Status`
- `FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error` -- reordered params, adds `quiet`
- `FormatStats(w io.Writer, stats StatsData) error` -- concrete type instead of `interface{}`

V4 also populates the previously-stubbed `TaskDetail` and `StatsData` structs in `format.go`, and adds a new exported `RelatedTask` type. V2 defines `StatsData` directly in `toon_formatter.go` and reuses existing unexported `showData`/`relatedTask` types from `show.go`.

#### toon-go Library Usage

**V2** uses the toon-go library in two distinct ways:

1. **FormatTaskList**: Uses struct tag-based marshaling via `toon.MarshalString()` with annotated structs:
```go
type toonListRow struct {
    ID       string `toon:"id"`
    Title    string `toon:"title"`
    Status   string `toon:"status"`
    Priority int    `toon:"priority"`
}
type toonListWrapper struct {
    Tasks []toonListRow `toon:"tasks"`
}
```
This delegates the full tabular encoding (header + rows) to the library.

2. **escapeField()**: For manual field escaping in detail/related sections, marshals a single-field struct and extracts the value:
```go
type wrapper struct {
    V string `toon:"v"`
}
out, err := toon.MarshalString(wrapper{V: s})
return strings.TrimPrefix(out, "v: ")
```
This produces `v: <value>` and strips the prefix. Has a fast-path check: `if !strings.ContainsAny(s, ",\"\n\\")` to skip marshaling for clean values.

**V4** uses the toon-go Object API throughout:

1. **FormatTaskList**: Builds `toon.Object` slices with `toon.NewObject(toon.Field{...})`:
```go
objects := make([]toon.Object, len(rows))
for i, r := range rows {
    objects[i] = toon.NewObject(
        toon.Field{Key: "id", Value: r.ID},
        toon.Field{Key: "title", Value: r.Title},
        ...
    )
}
doc := toon.NewObject(toon.Field{Key: "tasks", Value: objects})
result, err := toon.MarshalString(doc)
```

2. **toonEscapeValue()**: Marshals a single-element tabular array and extracts the value from the second line:
```go
doc := toon.NewObject(
    toon.Field{Key: "a", Value: []toon.Object{
        toon.NewObject(toon.Field{Key: "v", Value: s}),
    }},
)
result, err := toon.MarshalString(doc)
lines := strings.SplitN(result, "\n", 2)
return strings.TrimSpace(lines[1])
```
No fast-path optimization -- calls toon-go for every value regardless.

#### Code Structure

**V2** writes directly to the `io.Writer` in each method using `fmt.Fprintf`/`fmt.Fprintln`. The `FormatTaskDetail` method builds sections inline with sequential writes. A helper `writeRelatedSection()` is a package-level function.

**V4** uses a builder pattern: each section is constructed as a string by private methods (`buildTaskSection`, `buildRelatedSection`, `buildDescriptionSection`, `buildStatsSection`, `buildByPrioritySection`), then joined with `\n` separators and written in one `fmt.Fprint` call. All builder methods are receiver methods on `*ToonFormatter`.

V4's approach:
```go
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, detail TaskDetail) error {
    var sections []string
    sections = append(sections, f.buildTaskSection(detail))
    sections = append(sections, f.buildRelatedSection("blocked_by", detail.BlockedBy))
    sections = append(sections, f.buildRelatedSection("children", detail.Children))
    if detail.Description != "" {
        sections = append(sections, f.buildDescriptionSection(detail.Description))
    }
    fmt.Fprint(w, strings.Join(sections, "\n"))
    return nil
}
```

V2's approach:
```go
func (f *ToonFormatter) FormatTaskDetail(w io.Writer, data *showData) error {
    // Build dynamic schema inline
    fmt.Fprintf(w, "task{%s}:\n", strings.Join(schema, ","))
    fmt.Fprintf(w, "  %s\n", strings.Join(values, ","))
    fmt.Fprintln(w)
    writeRelatedSection(w, "blocked_by", data.BlockedBy)
    fmt.Fprintln(w)
    writeRelatedSection(w, "children", data.Children)
    // ...
}
```

#### Quiet Mode

V4 adds `quiet bool` parameters to `FormatTaskList` and `FormatDepChange`. When quiet is true, `FormatTaskList` outputs only IDs (one per line), and `FormatDepChange` outputs nothing. V2 has no quiet support in these methods -- quiet mode would need to be handled elsewhere.

#### Parent Title

V4's `TaskDetail` includes a `ParentTitle` field and renders it in the schema when parent is present:
```go
if detail.Parent != "" {
    fields = append(fields, "parent", "parent_title")
    values = append(values, toonEscapeValue(detail.Parent), toonEscapeValue(detail.ParentTitle))
}
```
This produces `task{id,title,status,priority,parent,parent_title,created,updated}:` which differs from the spec example (`task{id,title,status,priority,parent,created,updated}:`). V2 follows the spec exactly with just `parent`.

### Code Quality

#### Go Idioms and Naming

**V2** follows standard Go conventions. Uses unexported helper types (`toonListRow`, `toonListWrapper`) with struct tags. Package-level helper `writeRelatedSection` and `escapeField`. The `FormatStats` method accepts `interface{}` and does a type assertion:
```go
func (f *ToonFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
```
This is a code smell -- using `interface{}` loses compile-time type safety and requires a runtime assertion. However, V2 did not modify the existing Formatter interface, which had this signature from a prior task.

**V4** uses concrete types throughout. `FormatStats(w io.Writer, stats StatsData)` provides compile-time safety. Uses `strconv.Itoa` for integer conversion instead of V2's `fmt.Sprintf("%d", ...)`. V4's builder methods are receiver methods on `*ToonFormatter`, which is a matter of style but keeps related logic grouped.

V4 uses `strconv.Itoa`:
```go
values := []string{
    toonEscapeValue(detail.ID),
    toonEscapeValue(detail.Title),
    toonEscapeValue(detail.Status),
    strconv.Itoa(detail.Priority),
}
```

V2 uses `fmt.Sprintf`:
```go
values := []string{
    data.ID,
    escapeField(data.Title),
    data.Status,
    fmt.Sprintf("%d", data.Priority),
}
```

`strconv.Itoa` is marginally more efficient (avoids format string parsing) but functionally identical.

#### Error Handling

**V2** propagates write errors from `fmt.Fprintf` in `FormatTransition`, `FormatDepChange`, `FormatMessage`, and `FormatTaskList`:
```go
func (f *ToonFormatter) FormatTransition(w io.Writer, id string, ...) error {
    _, err := fmt.Fprintf(w, "%s: %s -> %s\n", id, oldStatus, newStatus)
    return err
}
```

**V4** ignores write errors from `fmt.Fprintf` throughout, always returning nil:
```go
func (f *ToonFormatter) FormatTransition(w io.Writer, id string, ...) error {
    fmt.Fprintf(w, "%s: %s -> %s\n", id, oldStatus, newStatus)
    return nil
}
```
This is a minor deficiency in V4 -- write errors (e.g., broken pipe) are silently swallowed.

#### Dependency Declaration

**V2** declares toon-go as `// indirect` in go.mod despite importing it directly. This is technically incorrect -- a direct import should be a direct dependency.

**V4** correctly declares toon-go as a direct dependency (no `// indirect` comment).

#### Type Coupling

**V2** imports `github.com/leeovery/tick/internal/task` for `task.Status` in `FormatTransition`. V4 uses plain `string` parameters, avoiding the coupling. This is arguably better for the formatter layer since it's purely a presentation concern and doesn't need domain types.

### Test Quality

#### V2 Test Functions

Top-level functions (4) with subtests (13 total):

1. **TestToonFormatterFormatTaskList** (3 subtests):
   - `"it formats list with correct header count and schema"` -- 2 tasks, exact string match
   - `"it formats zero tasks as empty section"` -- empty slice, exact string match
   - `"it escapes commas in titles"` -- single task with comma in title, exact string match

2. **TestToonFormatterFormatTaskDetail** (6 subtests):
   - `"it formats show with all sections"` -- full detail with parent, blockers, children, multiline description; exact string match
   - `"it omits parent/closed from schema when null"` -- no parent/closed; exact string match
   - `"it renders blocked_by/children with count 0 when empty"` -- Contains-based check for `blocked_by[0]` and `children[0]`
   - `"it omits description section when empty"` -- negative Contains check
   - `"it renders multiline description as indented lines"` -- Contains check for indented description block
   - `"it includes closed in schema when present"` -- Contains check for `closed` in schema

3. **TestToonFormatterFormatStats** (2 subtests):
   - `"it formats stats with all counts"` -- full stats data, exact string match
   - `"it formats by_priority with 5 rows including zeros"` -- zeros in priority, exact string match

4. **TestToonFormatterFormatTransitionAndDep** (4 subtests):
   - `"it formats transition as plain text"` -- exact string match with arrow character
   - `"it formats dep add as plain text"` -- exact string match
   - `"it formats dep removed as plain text"` -- exact string match
   - `"it formats message as plain text"` -- exact string match

5. **TestToonFormatterImplementsInterface** (1 subtest):
   - `"it implements the full Formatter interface"` -- compile-time check

**Total: 5 top-level functions, 16 subtests.**

#### V4 Test Functions

Top-level functions (5) with subtests (14 total):

1. **TestToonFormatter_ImplementsFormatter** (1 subtest):
   - `"it implements the full Formatter interface"` -- compile-time check

2. **TestToonFormatter_FormatTaskList** (3 subtests):
   - `"it formats list with correct header count and schema"` -- 2 tasks, exact string match
   - `"it formats zero tasks as empty section"` -- nil slice (not empty slice), exact string match
   - `"it escapes commas in titles"` -- **table-driven** with 3 sub-cases (in list, in task detail title, in related task title); checks for quote character presence only (not exact output match)

3. **TestToonFormatter_FormatTaskDetail** (5 subtests):
   - `"it formats show with all sections"` -- full detail including parent, parent_title, closed, blockers, children, description; Contains-based checks (not exact match)
   - `"it omits parent/closed from schema when null"` -- Contains-based negative/positive checks
   - `"it renders blocked_by/children with count 0 when empty"` -- Contains checks
   - `"it omits description section when empty"` -- negative Contains check
   - `"it renders multiline description as indented lines"` -- Contains check

4. **TestToonFormatter_FormatStats** (2 subtests):
   - `"it formats stats with all counts"` -- Contains-based checks (header and data row, not full match)
   - `"it formats by_priority with 5 rows including zeros"` -- line-counting logic to verify 5 rows; Contains check for zero counts

5. **TestToonFormatter_FormatTransitionAndDep** (4 subtests):
   - `"it formats transition as plain text"` -- exact string match
   - `"it formats dep change as plain text"` -- exact string match
   - `"it formats dep removal as plain text"` -- exact string match
   - `"it formats message as plain text"` -- exact string match

**Total: 5 top-level functions, 15 subtests (but "escapes commas" has 3 nested sub-cases, so 17 effective test cases).**

#### Test Coverage Differences

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Closed field included in schema | YES (dedicated test) | YES (tested in "all sections" test) |
| Exact full output matching for list | YES | YES |
| Exact full output matching for show | YES (full string) | NO (Contains-based only) |
| Exact full output matching for stats | YES (full string) | NO (Contains-based only) |
| Escaping in detail title | NO | YES (table-driven sub-case) |
| Escaping in related task title | NO | YES (table-driven sub-case) |
| Quiet mode list (IDs only) | N/A (no quiet param) | NOT TESTED |
| Quiet mode dep change | N/A (no quiet param) | NOT TESTED |
| FormatStats type assertion error | NOT TESTED (interface{} path) | N/A (concrete type) |
| nil vs empty slice for list | Tests empty slice `[]TaskRow{}` | Tests nil slice |
| parent_title field | N/A (not in V2) | YES (in "all sections" test) |

V2's stats and show tests use **exact string matching** -- any change to output format would break the test, providing stronger regression protection. V4 uses **Contains-based** assertions for stats and show, which are more lenient but could miss formatting regressions (e.g., extra whitespace, wrong ordering of sections).

V4 tests escaping in more contexts (detail title, related task title) via a well-structured table-driven test. However, its assertion (`strings.Contains(got, "\"")`) only checks that a quote exists somewhere in output, not that the specific value was correctly escaped.

V4 adds quiet parameters but does **not** test either quiet code path.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 6 | 7 |
| Lines added (total) | 597 | 684 |
| Impl LOC (toon_formatter.go) | 192 | 229 |
| Impl LOC (format.go changes) | 0 | +26 net |
| Test LOC | 396 | 419 |
| Top-level test functions | 5 | 5 |
| Test subtests | 16 | 15 (17 effective with nested) |

## Verdict

**V2 is the stronger implementation of this specific task.**

Key reasons:

1. **Spec fidelity**: V2's test output matches the specification examples exactly. V4 introduces a `parent_title` field in the show schema that does not appear in the spec (`task{id,title,status,priority,parent,parent_title,created,updated,closed}:` vs the spec's `task{id,title,status,priority,parent,created,updated}:`). While `parent_title` may be useful, it deviates from the acceptance criterion "All output matches spec examples."

2. **Test rigor**: V2 uses exact string matching for stats and show output, providing strict regression detection. V4 uses `strings.Contains` for these critical outputs, which could miss formatting issues. V2's stats test compares the full output byte-for-byte; V4's only checks that certain substrings exist.

3. **Error propagation**: V2 correctly captures and returns write errors from `fmt.Fprintf`. V4 discards them by not checking return values, which is a minor but real deficiency.

4. **Minimal scope**: V2 limits its changes to the new file plus go.mod/go.sum. V4 modifies the shared `format.go` file (changing the Formatter interface), which is a broader scope change that goes beyond the task requirement of implementing a TOON formatter.

V4 has advantages in:

- **Type safety**: Using `StatsData` directly rather than `interface{}` is genuinely better, though this was a pre-existing interface constraint V2 chose not to change.
- **Forward-looking design**: Adding `quiet` parameters and `parent_title` anticipates future needs. The section-builder pattern provides cleaner separation of concerns.
- **Broader escaping tests**: Testing escaping in detail titles and related task titles covers more surface area.
- **Correct go.mod**: Declaring toon-go as a direct dependency (not indirect) is technically correct.
- **Reduced coupling**: Using `string` instead of `task.Status` in `FormatTransition` is better for a presentation layer.

However, for this specific task -- implementing a TOON formatter that matches the spec -- V2 delivers higher fidelity to the specification with stricter test validation, while V4 takes liberties with the interface and output format that, while potentially useful, are not required by the task and introduce spec deviation.
