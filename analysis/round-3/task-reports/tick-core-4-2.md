# Task 4-2: TOON formatter -- list, show, stats output

## Task Plan Summary

Implement a concrete `ToonFormatter` struct satisfying the `Formatter` interface. The TOON format is an agent-facing, token-optimized notation providing 30-60% savings over JSON. Key requirements:

- **FormatTaskList**: `tasks[N]{id,title,status,priority}:` header with indented data rows; zero tasks produce `tasks[0]{...}:` with no rows.
- **FormatTaskDetail**: Multi-section output with dynamic schema (omit `parent`/`closed` when null); `blocked_by` and `children` always present (even `[0]`); description omitted when empty, multiline rendered as indented lines.
- **FormatStats**: `stats` summary section + `by_priority` section (always 5 rows, priorities 0-4).
- **FormatTransition / FormatDepChange / FormatMessage**: Plain text passthrough.
- Escaping handled via `github.com/toon-format/toon-go`.
- 11 specified test cases covering all edge cases.

---

## V4 Implementation

### Architecture & Design

**Files modified:**
- `/internal/cli/format.go` -- Added `RelatedTask`, `TaskDetail`, and `StatsData` types; populated formerly-TODO struct fields.
- `/internal/cli/toon_formatter.go` -- New file, 229 lines.
- `/internal/cli/toon_formatter_test.go` -- New file, 420 lines (419 lines of tests).

**Formatter interface (V4 approach):**
V4's `Formatter` interface uses **primitive/decomposed parameters**:
```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatStats(w io.Writer, stats StatsData) error
FormatMessage(w io.Writer, msg string) error
```

Key characteristics:
1. **`quiet` flag pushed into the formatter**: `FormatTaskList` and `FormatDepChange` accept a `quiet bool` parameter, making each formatter responsible for quiet-mode behavior.
2. **Value types for data structs**: `TaskDetail` and `StatsData` passed by value (not pointer). `listRow` is unexported but has **exported fields** (`ID`, `Title`, `Status`, `Priority`).
3. **String decomposition for transitions/deps**: Instead of a struct, `FormatTransition` takes `(id, oldStatus, newStatus string)` and `FormatDepChange` takes `(taskID, blockedByID, action string, quiet bool)`.
4. **Builder pattern for sections**: The formatter uses private methods `buildTaskSection`, `buildRelatedSection`, `buildDescriptionSection`, `buildStatsSection`, `buildByPrioritySection` that return strings, then joins them with `\n` separators.
5. **Data types co-located in format.go**: `RelatedTask`, `TaskDetail`, and `StatsData` are exported types defined in `format.go`.

**Escaping approach (`toonEscapeValue`):**
```go
func toonEscapeValue(s string) string {
    doc := toon.NewObject(
        toon.Field{Key: "a", Value: []toon.Object{
            toon.NewObject(toon.Field{Key: "v", Value: s}),
        }},
    )
    result, err := toon.MarshalString(doc)
    if err != nil {
        return s
    }
    lines := strings.SplitN(result, "\n", 2)
    if len(lines) == 2 {
        return strings.TrimSpace(lines[1])
    }
    return s
}
```
This is a heavyweight approach: it marshals a complete TOON document with a single-element array just to extract one escaped value. It creates multiple allocations per field. Every field is passed through this function -- even fields that never contain special characters (like IDs, statuses).

**FormatTaskList approach:** Uses `toon.NewObject` + `toon.MarshalString` for list output. Builds a complete TOON document from Go objects programmatically (no struct tags).

**Error handling:**
V4 **discards write errors** from `fmt.Fprint`/`fmt.Fprintf` in nearly all methods. Only 3 error checks exist in the entire 229-line file, and those are for `toon.MarshalString` failures, not for writes to `w`. For example:
```go
func (f *ToonFormatter) FormatTransition(w io.Writer, id string, ...) error {
    fmt.Fprintf(w, "%s: %s -> %s\n", id, oldStatus, newStatus)
    return nil  // write error ignored
}
```

**`FormatMessage` returns error:**
```go
func (f *ToonFormatter) FormatMessage(w io.Writer, msg string) error {
    fmt.Fprintln(w, msg)
    return nil
}
```

**Extra feature:** V4's `TaskDetail` includes a `ParentTitle` field and the task section schema includes `parent_title` when parent is present. This goes beyond the task plan spec, which does not mention `parent_title`.

### Code Quality

- **Idiomatic Go**: Generally clean, well-documented. Each exported method has a godoc comment.
- **Error handling weakness**: Systematic failure to check write errors. The `golang-pro` skill mandates "Handle all errors explicitly (no naked returns)" -- while there are no naked returns, silently discarding `fmt.Fprintf` return values violates the spirit of explicit error handling.
- **Escaping overhead**: `toonEscapeValue` is called for every single field value including IDs and statuses that provably never need escaping. This is wasteful.
- **`quiet` leaking into interface**: Passing `quiet bool` into `FormatTaskList` and `FormatDepChange` is a design smell. Quiet mode is a presentation concern that should be handled by the caller, not pushed into every formatter implementation. Each formatter must now implement quiet-mode logic identically.
- **Builder pattern**: The `buildXxxSection` methods return intermediate strings that are joined. This creates unnecessary string allocations compared to writing directly to the writer.

### Test Coverage

V4 provides **420 lines of tests** covering all 11 specified test cases from the plan:

1. "it formats list with correct header count and schema" -- checks exact output
2. "it formats zero tasks as empty section" -- nil slice input
3. "it escapes commas in titles" -- **table-driven subtest** with 3 sub-cases (list, detail title, related task title)
4. "it formats show with all sections" -- uses `strings.Contains` checks
5. "it omits parent/closed from schema when null"
6. "it renders blocked_by/children with count 0 when empty"
7. "it omits description section when empty"
8. "it renders multiline description as indented lines"
9. "it formats stats with all counts" -- uses `strings.Contains`
10. "it formats by_priority with 5 rows including zeros" -- **manually counts rows by line-scanning**
11. "it formats transition/dep as plain text" -- exact string match

**Test quality observations:**
- The escaping test (item 3) is truly table-driven with subtests -- good skill compliance.
- Tests for "show with all sections" and "stats with all counts" use `strings.Contains` instead of exact match. This is a **weaker assertion pattern** -- it can pass even if the output has extra/wrong content as long as the checked substring exists.
- The by_priority test counts rows by scanning lines for prefixes like `"0,"`, `"1,"` etc. This is brittle -- it would false-match if those prefixes appeared anywhere in the stats section (which they do: `"  47,12,3,28,4,8,4"` contains `"4,"`).
- V4 tests the `ParentTitle` feature which is not in the spec.
- No test for unknown dep action behavior (V4 silently does nothing for unknown actions).
- `FormatMessage` test uses exact match -- good.
- Interface compile check present.

### Spec Compliance

| Requirement | Status | Notes |
|---|---|---|
| Implements full Formatter interface | PASS | Compile-time check in test |
| List output matches spec TOON format | PASS | Exact match verified |
| Show multi-section with dynamic schema | PASS | parent/closed conditional |
| blocked_by/children always present | PASS | Even with count 0 |
| Stats produces summary + 5-row by_priority | PASS | Verified |
| Escaping via toon-go | PASS | Uses toon-go library |
| All output matches spec examples | PARTIAL | Added `parent_title` not in spec |

V4 adds `parent_title` to the task section schema, which is **beyond the spec**. The spec says `parent` is conditionally included, but makes no mention of `parent_title`. This is a scope creep issue.

### golang-pro Skill Compliance

| Rule | Status | Notes |
|---|---|---|
| Handle all errors explicitly | FAIL | Write errors from fmt.Fprint systematically discarded |
| Write table-driven tests with subtests | PARTIAL | Only escaping test is table-driven; others are individual |
| Document all exported functions/types | PASS | All exported items have godoc |
| Propagate errors with fmt.Errorf("%w") | PASS | Used in toon marshal error |
| No panic for error handling | PASS | No panics |
| No ignored errors without justification | FAIL | 9+ fmt.Fprint calls with discarded return values |

---

## V5 Implementation

### Architecture & Design

**Files modified:**
- `/internal/cli/toon_formatter.go` -- New file, 225 lines.
- `/internal/cli/toon_formatter_test.go` -- New file, 393 lines.
- `/internal/cli/format.go` -- Formatter interface, shared helpers `formatTransitionText`, `formatDepChangeText`, `formatMessageText`.

**Formatter interface (V5 approach):**
```go
FormatTaskList(w io.Writer, rows []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, data *TransitionData) error
FormatDepChange(w io.Writer, data *DepChangeData) error
FormatStats(w io.Writer, data *StatsData) error
FormatMessage(w io.Writer, msg string)               // no error return
```

Key characteristics:
1. **Struct parameters for all methods**: `TransitionData`, `DepChangeData`, `StatsData` are dedicated struct types. This bundles related data cleanly and makes the interface extensible without signature changes.
2. **Pointer receivers for data**: `*showData`, `*StatsData`, `*DepChangeData`, `*TransitionData` are all pointers, avoiding copy overhead for larger structs.
3. **`quiet` handled at caller level**: The `FormatTaskList` signature has no `quiet` param. Instead, the `list.go` command handler checks `ctx.Quiet` and short-circuits before calling the formatter. This means formatters don't need to know about quiet mode.
4. **Shared helper functions in format.go**: `formatTransitionText`, `formatDepChangeText`, `formatMessageText` are package-level functions that `ToonFormatter` (and presumably `PrettyFormatter`) delegate to. This eliminates code duplication across formatters.
5. **Separate type layers**: V5 uses unexported `listRow` (with unexported fields) internally in the command handler, then converts to exported `TaskRow` before passing to the formatter. Similarly, `showData` and `relatedTask` are unexported types used by the formatter within the same package.
6. **`FormatMessage` returns no error**: Since it is a simple `fmt.Fprintln`, V5 made the pragmatic choice of not returning an error from this method. This simplifies call sites.
7. **Data types co-located with formatter**: `TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` are defined in `toon_formatter.go`. While this co-location works when there is one formatter, it could create import issues if other formatters need these types. However, since all formatters are in the same package, this is fine.

**Escaping approach (`escapeField`):**
```go
func escapeField(s string) string {
    if !strings.ContainsAny(s, ",\"\n\\:[]{}") {
        return s
    }
    type wrapper struct {
        V string `toon:"v"`
    }
    out, err := toon.MarshalString(wrapper{V: s})
    if err != nil {
        return s
    }
    return strings.TrimPrefix(out, "v: ")
}
```
This is significantly better than V4's approach:
- **Fast-path**: Fields without special characters return immediately with zero allocations.
- **Struct-tag marshaling**: Uses `toon:"v"` struct tag for a cleaner API than manually building `toon.Object`.
- **Simpler extraction**: `strings.TrimPrefix(out, "v: ")` is simpler and more robust than splitting on newlines.
- **Only applied to title fields**: `escapeField` is only called on `d.title` and `rt.title`, not on IDs, statuses, timestamps, etc.

**FormatTaskList approach:** Uses struct-tagged `toonListRow` and `toonListWrapper` types with `toon.MarshalString`. This leverages the toon-go library's struct marshaling rather than manual object construction. Also passes `toon.WithIndent(2)` for formatting control.

**Error handling:**
V5 **checks every write error**. Every `fmt.Fprintf`/`fmt.Fprintln` call captures the error and returns it immediately if non-nil. There are 37 error-check instances in the file. Example:
```go
if _, err := fmt.Fprintf(w, "task{%s}:\n", strings.Join(schema, ",")); err != nil {
    return err
}
if _, err := fmt.Fprintf(w, "  %s\n", strings.Join(values, ",")); err != nil {
    return err
}
```

**Direct writes to `io.Writer`:** Unlike V4's builder pattern, V5 writes directly to the `io.Writer` in `FormatTaskDetail` and `FormatStats`, avoiding intermediate string building. This is more memory-efficient for large outputs.

### Code Quality

- **Idiomatic Go**: Very clean. Follows standard patterns. Good use of struct tags for toon-go integration.
- **Error handling excellence**: Every write error is checked and propagated. This is thorough and correct.
- **Efficient escaping**: The fast-path check in `escapeField` avoids unnecessary work. Only title fields are escaped.
- **DRY via shared helpers**: `formatTransitionText`, `formatDepChangeText`, `formatMessageText` are shared across formatters, reducing duplication.
- **Section separation**: Blank lines between sections are written explicitly with `fmt.Fprintln(w)`, making the output format clear from reading the code.
- **Description trimming**: V5 uses `strings.TrimSpace(d.description) != ""` to check for empty descriptions, which handles whitespace-only descriptions. V4 uses `detail.Description != ""` which would render a whitespace-only description.

### Test Coverage

V5 provides **393 lines of tests** covering all 11 specified test cases plus one extra:

1. "it formats list with correct header count and schema" -- **exact full output match**
2. "it formats zero tasks as empty section" -- exact match
3. "it formats show with all sections" -- **exact full output match** (key difference from V4)
4. "it omits parent/closed from schema when null" -- **exact full output match**
5. "it renders blocked_by/children with count 0 when empty" -- `strings.Contains` checks
6. "it omits description section when empty" -- negative `strings.Contains`
7. "it renders multiline description as indented lines" -- `strings.Contains`
8. "it escapes commas in titles" -- **exact full output match** (single test, not table-driven)
9. "it formats stats with all counts" -- **exact full output match**
10. "it formats by_priority with 5 rows including zeros" -- **exact full output match**
11. "it formats transition as plain text" / "dep add" / "dep removed" -- exact matches
12. **EXTRA**: "it includes closed in schema when present" -- not in the plan's test list but validates a spec requirement
13. "it formats message as plain text" -- exact match

**Test quality observations:**
- V5 tests heavily favor **exact full output matching** (`buf.String() != expected`). This is the strongest assertion pattern -- any deviation in output causes failure.
- The escaping test is **not table-driven** -- it tests only one case (comma in list title). V4's escaping test covers 3 scenarios (list, detail title, related task title).
- V5 adds an extra test for `closed` in schema -- good additional coverage.
- V5 tests `FormatMessage` without checking error return (because the method doesn't return one).
- Tests use `*showData` and `[]relatedTask` directly (unexported types) -- only possible because tests are in the same package.

### Spec Compliance

| Requirement | Status | Notes |
|---|---|---|
| Implements full Formatter interface | PASS | Compile-time check in test |
| List output matches spec TOON format | PASS | Exact match verified |
| Show multi-section with dynamic schema | PASS | parent/closed conditional |
| blocked_by/children always present | PASS | Even with count 0 |
| Stats produces summary + 5-row by_priority | PASS | Verified |
| Escaping via toon-go | PASS | Uses toon-go library |
| All output matches spec examples | PASS | No extra fields beyond spec |

V5 is fully spec-compliant with no scope creep.

### golang-pro Skill Compliance

| Rule | Status | Notes |
|---|---|---|
| Handle all errors explicitly | PASS | Every write error checked and propagated |
| Write table-driven tests with subtests | PARTIAL | Not truly table-driven; individual subtests |
| Document all exported functions/types | PASS | All exported items have godoc |
| Propagate errors with fmt.Errorf("%w") | PASS | Used in marshal error |
| No panic for error handling | PASS | No panics |
| No ignored errors without justification | PASS | All errors handled |

---

## Comparative Analysis

### Where V4 is Better

1. **Table-driven escaping test**: V4's "it escapes commas in titles" test is genuinely table-driven with 3 sub-cases (list, detail title, related task title). This is more thorough than V5's single-case escaping test and better aligns with the golang-pro skill's table-driven test requirement.

2. **Slightly more test coverage for escaping edge cases**: By testing comma escaping in detail titles and related task titles (not just list titles), V4 provides broader confidence that escaping works across all output paths.

3. **`quiet` mode tested implicitly**: Since V4 passes `quiet` through the formatter, the formatter itself can be tested for quiet behavior (though no such test exists). V5 moves this concern entirely out of the formatter, which is architecturally better but means the quiet path cannot be tested through the Formatter interface alone.

### Where V5 is Better

1. **Error handling (major)**: V5 checks every `fmt.Fprint`/`fmt.Fprintf` return value. V4 silently discards all write errors. This is the single largest quality difference. The golang-pro skill explicitly requires "Handle all errors explicitly." V5 has 37 error checks; V4 has 3 (none for writes). This matters in practice when writing to network-backed writers or when the writer is closed mid-operation.

2. **Interface design**: V5's Formatter interface is significantly cleaner:
   - Struct parameters (`*TransitionData`, `*DepChangeData`) are extensible without breaking the interface. Adding a field to the struct is non-breaking; adding a parameter to V4's decomposed signatures would break every implementation.
   - No `quiet bool` parameter leak -- formatters only handle formatting, not presentation logic.
   - `FormatMessage` honestly returns no error rather than always returning nil.

3. **Escaping efficiency**: V5's `escapeField` has a fast-path for strings without special characters and only escapes title fields. V4's `toonEscapeValue` marshals a full TOON document for every field, including IDs and timestamps that never need escaping.

4. **Shared helpers for DRY**: V5 extracts `formatTransitionText`, `formatDepChangeText`, `formatMessageText` into `format.go` so all formatters (Toon, Pretty, JSON) can reuse them. V4 has no such sharing.

5. **Direct writing vs string building**: V5 writes directly to `io.Writer` in `FormatTaskDetail` and `FormatStats`. V4 builds intermediate strings in `buildXxxSection` methods, joins them, then writes once. V5's approach uses less memory and is more idiomatic for io.Writer-based APIs.

6. **Exact output assertions in tests**: V5 tests primarily use exact full output comparison (`buf.String() != expected`). V4 tests for "show with all sections" and "stats" use `strings.Contains`, which can miss issues with extra/malformed output.

7. **Struct-tag-based toon-go usage**: V5 uses `toon:"id"` struct tags for list marshaling, which is the idiomatic way to use the toon-go library. V4 manually constructs `toon.Object` with `toon.Field` values, which is more verbose and error-prone.

8. **Spec compliance**: V5 strictly implements the spec. V4 adds `parent_title` which is not specified, constituting scope creep.

9. **Description edge case**: V5 uses `strings.TrimSpace(d.description) != ""` to check for empty descriptions, correctly handling whitespace-only strings. V4 uses `detail.Description != ""` which would pass through whitespace-only descriptions.

10. **Extra test case**: V5 includes "it includes closed in schema when present" which validates a spec requirement not explicitly in the plan's test list but still important.

### Differences That Are Neutral

1. **Data type location**: V4 puts `RelatedTask`, `TaskDetail`, `StatsData` in `format.go`. V5 puts `TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` in `toon_formatter.go`. Both work since all code is in the same package.

2. **Exported vs unexported detail types**: V4 uses exported `TaskDetail` with exported fields; V5 uses unexported `showData` with unexported fields, plus an exported `TaskRow`. Both are valid given all code is in `package cli`.

3. **Line count**: V4 formatter is 229 lines, V5 is 225 lines. V4 tests are 420 lines, V5 are 393 lines. Negligible difference.

4. **Sections joined vs written inline**: V4 joins section strings with `\n`. V5 writes `fmt.Fprintln(w)` for blank lines between sections. Both produce the same output.

5. **`toon.WithIndent(2)` in V5 list**: V5 passes `toon.WithIndent(2)` to `MarshalString` for list output. V4 relies on toon-go's default indentation. This is a library usage detail with no functional difference if the default is also 2 spaces.

---

## Verdict

**Winner: V5**

V5 is the clearly superior implementation. The deciding factors are:

1. **Error handling is not optional.** V4 systematically ignores write errors across every method. The golang-pro skill explicitly mandates "Handle all errors explicitly" and "No ignored errors without justification." V5 checks all 37 write-error sites. This alone would be sufficient to choose V5.

2. **Interface design is fundamentally better.** V5's use of struct parameters (`*TransitionData`, `*DepChangeData`, `*StatsData`) produces a more extensible, cleaner API. V4's decomposed parameters (`id, oldStatus, newStatus string`) lock the interface shape to the current set of fields. V5's removal of `quiet` from the formatter interface is a correct separation of concerns.

3. **Escaping is measurably more efficient.** V5's `escapeField` has a fast-path that avoids any toon-go call for fields without special characters, and only escapes title fields. V4's `toonEscapeValue` marshals a full TOON document for every single field value, wasting allocations.

4. **Test assertions are stronger.** V5's exact full output matching catches any deviation. V4's `strings.Contains` for show and stats output can miss structural issues.

5. **Spec compliance is stricter.** V5 implements exactly what the spec requires. V4 adds `parent_title` which is scope creep.

V4's only advantage is the table-driven escaping test, which is a minor testing style point. The fundamental architecture, error handling, interface design, and spec fidelity all favor V5.
