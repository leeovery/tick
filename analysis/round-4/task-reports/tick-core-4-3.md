# Task tick-core-4-3: Human-readable formatter -- list, show, stats output

## Task Summary

Implement a concrete `PrettyFormatter` that implements the `Formatter` interface for terminal-facing human-readable output. Requirements:

- **FormatTaskList**: Column-aligned table with header. Dynamic widths. Empty list produces `No tasks found.` (no headers). Long titles truncated with `...` in list only.
- **FormatTaskDetail**: Key-value with aligned labels. Sections: base fields, Blocked by (indented), Children (indented), Description (indented block). Omit empty sections.
- **FormatStats**: Three groups (total, status breakdown + workflow counts, priority with P0-P4 labels). Right-align numbers. All rows present even at zero.
- **FormatTransition/FormatDepChange/FormatMessage**: plain text passthrough.
- No borders, no colors, no icons. Aligned columns only.

Acceptance criteria:
1. Implements full Formatter interface
2. List matches spec format -- aligned columns with header
3. Empty list produces `No tasks found.`
4. Show matches spec -- aligned labels, omitted empty sections
5. Stats three groups with right-aligned numbers
6. Priority P0-P4 labels always present
7. Long titles truncated in list
8. All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS -- `var _ Formatter = &PrettyFormatter{}` in test + implements all 6 methods | PASS -- `var _ Formatter = (*PrettyFormatter)(nil)` compile-time check in source + test |
| List matches spec format -- aligned columns with header | PASS -- dynamic width calculation, header row, `%-Xs %-Xs %Xd %s` format | PASS -- dynamic width calculation, header row, `%-*s` format |
| Empty list produces `No tasks found.` | PASS -- tested with empty `[]TaskRow{}` | PASS -- tested with empty `[]task.Task{}` AND `nil` (extra edge case) |
| Show matches spec -- aligned labels, omitted empty sections | PASS -- 10-char label column, omits blockedBy/children/description when empty | PASS -- 10-char label column, omits sections when empty |
| Stats three groups with right-aligned numbers | PASS -- dynamic numWidth computation, three groups: Total, Status, Workflow, Priority | PASS -- fixed-width format specifiers, three groups: Total, Status, Workflow, Priority |
| Priority P0-P4 labels always present | PASS -- all 5 labels rendered unconditionally | PASS -- all 5 labels rendered unconditionally |
| Long titles truncated in list | PASS -- `truncateTitle()` at `maxTitleWidth=50` | PASS -- `truncateTitle()` at `maxListTitleLen=50` |
| All output matches spec examples | PASS -- exact string comparison in tests | PASS -- exact string comparison in tests |

## Implementation Comparison

### Approach

The two versions differ fundamentally in their **interface design philosophy** and **I/O pattern**.

**V5: `io.Writer` + `interface{}` pattern (at commit time)**

V5 was built against an older Formatter interface that used `interface{}` parameters and `io.Writer` + `error` returns:

```go
// V5 Formatter interface (at commit time)
type Formatter interface {
    FormatTaskList(w io.Writer, data interface{}) error
    FormatTaskDetail(w io.Writer, data interface{}) error
    FormatTransition(w io.Writer, data interface{}) error
    FormatDepChange(w io.Writer, data interface{}) error
    FormatStats(w io.Writer, data interface{}) error
    FormatMessage(w io.Writer, msg string)
}
```

Each method writes directly to `io.Writer`, requiring runtime type assertions:

```go
func (f *PrettyFormatter) FormatTaskList(w io.Writer, data interface{}) error {
    rows, ok := data.([]TaskRow)
    if !ok {
        return fmt.Errorf("FormatTaskList: expected []TaskRow, got %T", data)
    }
    // ... writes to w ...
}
```

Note: The V5 worktree's final source shows typed parameters (the interface was later refactored), but the **commit itself** used `interface{}`. The refactoring was done in a subsequent commit.

**V6: `string` return + typed parameters**

V6 was built against a fully typed Formatter interface with no `io.Writer` or `interface{}`:

```go
// V6 Formatter interface
type Formatter interface {
    FormatTaskList(tasks []task.Task) string
    FormatTaskDetail(detail TaskDetail) string
    FormatTransition(id string, oldStatus string, newStatus string) string
    FormatDepChange(action string, taskID string, depID string) string
    FormatStats(stats Stats) string
    FormatMessage(msg string) string
}
```

Methods return `string`, build output internally via `strings.Builder`:

```go
func (f *PrettyFormatter) FormatTaskList(tasks []task.Task) string {
    if len(tasks) == 0 {
        return "No tasks found."
    }
    var b strings.Builder
    // ... builds string ...
    return b.String()
}
```

**Key structural differences:**

1. **Data types**: V5 uses custom DTOs (`TaskRow`, `showData`, `relatedTask`, `StatsData`, `TransitionData`, `DepChangeData`) -- all CLI-layer types. V6 uses domain types directly (`task.Task`, `TaskDetail` wrapping `task.Task`, `Stats`). V6's approach couples the formatter to domain types but eliminates a mapping layer.

2. **Transition/DepChange**: V5 implements these inline; V6 uses `baseFormatter` struct embedding to share implementations with `ToonFormatter`:
   ```go
   // V6
   type PrettyFormatter struct {
       baseFormatter
   }
   ```
   In the commit, V6's `PrettyFormatter` did NOT embed `baseFormatter` -- it implemented these methods directly. The worktree's final source shows the embedding was added later.

3. **FormatStats alignment strategy**: V5 computes dynamic column widths using `numWidth()` helper that returns the width needed for the widest number (min 3). V6 uses hardcoded fixed-width format specifiers (`%8d`, `%10d`, `%2d`, etc.) with per-line manual alignment. V5's approach is more maintainable for varying data magnitudes; V6's approach is simpler but breaks if numbers exceed the hardcoded widths.

4. **Trailing newlines**: V5 appends `\n` after every line (including final line) since it writes via `fmt.Fprintf(w, ...\n)`. V6 returns strings WITHOUT trailing newlines, using `\n` as separators between lines. This is a convention difference -- the caller decides how to handle the output differently in each version.

5. **Parent display in show**: V5 uses `"Parent:   tick-e5f6  Auth epic"` (double-space separator). V6 uses `"Parent:   tick-e5f6 (Auth System)"` (parenthesized). Both are valid interpretations of the spec.

### Code Quality

**Go idioms:**

V6 is more idiomatic. It uses:
- `strings.Builder` internally and returns `string` -- cleaner API
- `var _ Formatter = (*PrettyFormatter)(nil)` compile-time check in source (not just tests)
- `baseFormatter` embedding for shared logic (DRY)
- Direct use of domain types (`task.Task`, `task.Status`) instead of string intermediaries

V5's `interface{}` at commit time is anti-idiomatic Go. The type assertions (`data.([]TaskRow)`) sacrifice compile-time safety for runtime checks. This was later refactored out, but the commit itself represents a design that Go best practices strongly discourage.

**Naming:**

V5 uses `maxTitleWidth` (const) and `truncateTitle(title, maxWidth)` with a second parameter -- more flexible. V6 uses `maxListTitleLen` (const) and `truncateTitle(title)` with no parameter -- simpler but less reusable.

```go
// V5
func truncateTitle(title string, maxWidth int) string {
    if len(title) <= maxWidth { return title }
    if maxWidth <= 3 { return strings.Repeat(".", maxWidth) }
    return title[:maxWidth-3] + "..."
}

// V6
func truncateTitle(title string) string {
    if len(title) <= maxListTitleLen { return title }
    return title[:maxListTitleLen-3] + "..."
}
```

V5's version also handles the edge case where `maxWidth <= 3`, returning dots only. V6 does not handle this edge case.

**Error handling:**

V5 returns `error` from all methods and propagates write errors:
```go
if _, err := fmt.Fprintf(w, fmtStr, "ID", "STATUS", "PRI", "TITLE"); err != nil {
    return err
}
```

V6 returns `string` with no error path -- `strings.Builder.Write` never fails, so this is justified. However, V5 also ignores errors from many `fmt.Fprintf` calls in `FormatTaskDetail` and `FormatStats` (no error check on individual writes to `io.Writer`), making its error handling inconsistent.

**DRY:**

V6 uses `baseFormatter` embedding for `FormatTransition` and `FormatDepChange` (though at commit time it didn't -- it had inline implementations). V5 uses shared functions `formatTransitionText()` and `formatDepChangeText()` in `format.go`, called by both Pretty and Toon formatters. Both achieve DRY; V5's approach is function-based composition, V6's is struct embedding.

**Type safety:**

V6 is strongly typed throughout -- `[]task.Task`, `TaskDetail`, `Stats`. V5 at commit time used `interface{}` requiring runtime type assertions, which is fundamentally less safe.

### Test Quality

**V5 Test Functions (12 subtests across 4 top-level functions):**

1. `TestPrettyFormatterImplementsInterface`
   - `"it implements the full Formatter interface"` -- compile-time check
2. `TestPrettyFormatterFormatTaskList`
   - `"it formats list with aligned columns"` -- exact string match, 2 tasks
   - `"it aligns with variable-width data"` -- exact string match, different IDs/statuses
   - `"it shows 'No tasks found.' for empty list"` -- empty `[]TaskRow{}`
   - `"it truncates long titles in list"` -- checks truncated substring present, full absent
3. `TestPrettyFormatterFormatTaskDetail`
   - `"it formats show with all sections"` -- full output with blockers, children, description, parent
   - `"it omits empty sections in show"` -- no blockers/children/description, checks absence
   - `"it does not truncate in show"` -- long title preserved in show output
4. `TestPrettyFormatterFormatStats`
   - `"it formats stats with all groups, right-aligned"` -- exact string match, realistic data
   - `"it shows zero counts in stats"` -- `strings.Contains` checks for section headers
   - `"it renders P0-P4 priority labels"` -- loop checking all 5 labels present
5. `TestPrettyFormatterTransitionAndDep`
   - `"it formats transition as plain text"` -- exact match
   - `"it formats dep add as plain text"` -- exact match
   - `"it formats dep removed as plain text"` -- exact match
   - `"it formats message as plain text"` -- exact match

**V6 Test Functions (14 subtests under 1 top-level function):**

1. `TestPrettyFormatter` (single parent, all subtests nested)
   - Compile-time interface check (inline, not a named subtest)
   - `"it formats list with aligned columns"` -- exact string match, 2 tasks with `task.Task`
   - `"it aligns with variable-width data"` -- positional alignment checks (column positions), not exact match
   - `"it shows No tasks found for empty list"` -- empty `[]task.Task{}`
   - `"it shows No tasks found for nil list"` -- **nil** slice (extra edge case not in V5)
   - `"it formats show with all sections"` -- exact string match with `TaskDetail`
   - `"it omits empty sections in show"` -- exact match + absence checks
   - `"it includes closed timestamp when present in show"` -- **extra edge case** not in V5
   - `"it formats stats with all groups right-aligned"` -- exact string match
   - `"it shows zero counts in stats"` -- exact string match (stronger than V5's `Contains` checks)
   - `"it renders P0-P4 priority labels"` -- loop checking all 5 labels
   - `"it truncates long titles in list"` -- suffix check + length check on title content
   - `"it does not truncate in show"` -- full title present
   - `"it formats transition as plain text"` -- exact match
   - `"it formats dep change as plain text"` -- both add and remove in single subtest
   - `"it formats message as plain text passthrough"` -- exact match

**Test coverage diff:**

| Edge case | V5 | V6 |
|-----------|-----|-----|
| Aligned list columns | Exact string match | Exact string match |
| Variable-width alignment | Exact string match | Positional alignment (more robust) |
| Empty list | `[]TaskRow{}` | `[]task.Task{}` + `nil` (extra) |
| Nil list | Not tested | Tested |
| All sections in show | Tested | Tested |
| Omit empty sections | Tested | Tested |
| Closed timestamp | Not tested | Tested |
| Stats with data | Exact match | Exact match |
| Stats with zeros | Contains checks only | Exact string match (stronger) |
| P0-P4 labels | Loop check | Loop check |
| Long title truncation | Substring + absence | Suffix + length check |
| No truncation in show | Tested | Tested |
| Transition | Tested | Tested |
| Dep add | Tested separately | Tested in combined subtest |
| Dep remove | Tested separately | Tested in combined subtest |
| Message | Tested | Tested |

V6 has two additional edge cases: nil list and closed timestamp. V6's zero-count stats test is stronger (exact match vs Contains). V5's dep change tests are separated (add vs remove as separate subtests), which is slightly better for test isolation.

**Assertion style:**

V5 uses `bytes.Buffer` with `buf.String()` comparison (since methods write to `io.Writer`). V6 compares returned strings directly. Both use `t.Fatalf` for setup failures and `t.Errorf` for assertion failures. Neither uses a third-party assertion library.

V5 tests are organized across 4 top-level test functions grouped by method. V6 nests everything under a single `TestPrettyFormatter`. V5's structure is more granular for `go test -run` filtering.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint | PASS -- standard formatting observed | PASS -- standard formatting observed |
| Handle all errors explicitly | PARTIAL -- FormatTaskDetail/FormatStats ignore fmt.Fprintf errors to io.Writer | PASS -- returns string, no error path needed; `strings.Builder` writes never fail |
| Write table-driven tests with subtests | PARTIAL -- uses subtests but not table-driven format; each test is an individual subtest | PARTIAL -- uses subtests but not table-driven format; each test is an individual subtest |
| Document all exported functions, types, and packages | PASS -- all exported items have doc comments | PASS -- all exported items have doc comments |
| Propagate errors with fmt.Errorf("%w", err) | PARTIAL -- uses `fmt.Errorf` for type assertion failures but without `%w` (no wrapped error) | N/A -- no error returns |
| Do not ignore errors (avoid _ assignment) | PARTIAL -- `fmt.Fprintf` return values ignored in FormatTaskDetail (lines 67-84) and FormatStats (lines 142-179) | PASS -- no ignored errors; `strings.Builder` writes cannot fail |
| Do not use panic for normal error handling | PASS | PASS |
| Do not use reflection without justification | V5 commit uses `interface{}` which effectively requires runtime type checking (a form of runtime reflection) | PASS -- fully typed |
| Do not hardcode configuration | V5: PASS -- `maxTitleWidth = 50` const | V6: PASS -- `maxListTitleLen = 50` const |

### Spec-vs-Convention Conflicts

**1. `interface{}` vs typed parameters (V5 only)**

- **Spec says**: Implement the Formatter interface (as defined by earlier tasks).
- **Language convention**: Go strongly favors typed interfaces; `interface{}` is discouraged as a parameter type when concrete types are known.
- **V5 chose**: Implemented against the existing `interface{}` Formatter interface, faithfully matching the pre-existing contract.
- **V6 chose**: Implemented against a typed Formatter interface that was refactored before this task.
- **Assessment**: V5 had no choice -- it was working against the interface as it existed. The issue is architectural (the interface was poorly designed), not an implementation failing. V6 benefited from a better interface.

**2. `io.Writer` + `error` vs `string` return**

- **Spec says**: Formatter methods format output for display.
- **Language convention**: For formatters that build complete strings (not streaming), returning `string` is simpler and eliminates error paths that cannot meaningfully fail with `strings.Builder`.
- **V5 chose**: `io.Writer` + `error` pattern (matches its Formatter interface).
- **V6 chose**: `string` return pattern (matches its Formatter interface).
- **Assessment**: V6's interface is the better design for this use case. Formatters here build complete output strings, not streaming output. The `io.Writer` pattern adds complexity with no benefit.

**3. Transition arrow character**

- **V5**: Uses Unicode right arrow `\u2192` ("open -> in_progress" displayed as "open -> in_progress").
- **V6 at commit time**: Uses ASCII `->` ("open -> in_progress").
- **Assessment**: Minor cosmetic difference. The spec says "plain text passthrough." V6's ASCII arrow is more portable; V5's Unicode arrow is more visually polished. Both are reasonable.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 4 (2 impl + planning + tracking) | 4 (2 impl + planning + tracking) |
| Lines added | 657 (impl files only) | 542 (impl files only) |
| Impl LOC | 251 (at commit) / 215 (final worktree) | 178 (at commit, same as worktree at that point) |
| Test LOC | 406 | 364 |
| Test functions | 4 top-level, 15 subtests | 1 top-level, 14 subtests |

## Verdict

**V6 is the stronger implementation**, primarily due to its superior interface design and type safety.

Key advantages of V6:
1. **Type safety**: Fully typed parameters eliminate runtime type assertion failures. V5's `interface{}` pattern at commit time is a significant Go anti-pattern.
2. **Simpler API surface**: Returning `string` instead of `(io.Writer, error)` is the correct pattern for formatters that build complete output. It eliminates error-handling complexity that adds no value (V5 inconsistently checks write errors anyway).
3. **Additional test coverage**: V6 tests nil list handling and closed timestamp rendering -- two edge cases V5 misses. V6's zero-count stats test uses exact string matching instead of weaker `Contains` checks.
4. **DRY via embedding**: V6's `baseFormatter` embedding (in later worktree state) and direct inline implementations (at commit time) are cleaner than V5's shared function approach.
5. **Domain type reuse**: V6 uses `task.Task` directly instead of mapping to `TaskRow`, eliminating an intermediate type and reducing the surface for mapping bugs.

V5 has one minor advantage: its `truncateTitle` function handles the `maxWidth <= 3` edge case, and its test structure with multiple top-level functions provides better `go test -run` granularity. These are small points that do not outweigh V6's architectural advantages.

The core differentiator is that V6 was built against a properly designed Formatter interface, which cascades into cleaner implementation, simpler error handling, and stronger type safety throughout.
