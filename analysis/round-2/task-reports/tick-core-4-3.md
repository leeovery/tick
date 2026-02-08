# Task tick-core-4-3: Human-Readable Formatter

## Task Summary

Implement `PrettyFormatter` — a concrete implementation of the `Formatter` interface that produces human-readable, column-aligned terminal output. No borders, colors, or icons.

Required methods:
- **FormatTaskList**: Column-aligned table with header. Dynamic widths. Empty list produces `No tasks found.` with no headers.
- **FormatTaskDetail**: Key-value with aligned labels. Sections: base fields, Blocked by (indented), Children (indented), Description (indented block). Omit empty sections.
- **FormatStats**: Three groups (total, status/workflow, priority with P0-P4 labels). Right-align numbers. All rows present even at zero.
- **FormatTransition / FormatDepChange / FormatMessage**: Plain text passthrough.
- Long titles truncated with `...` in list; full in show.

Required tests:
1. Formats list with aligned columns
2. Aligns with variable-width data
3. Shows `No tasks found.` for empty list
4. Formats show with all sections
5. Omits empty sections in show
6. Formats stats with all groups, right-aligned
7. Shows zero counts in stats
8. Renders P0-P4 priority labels
9. Truncates long titles in list
10. Does not truncate in show

Acceptance criteria:
- Implements full Formatter interface
- List matches spec format — aligned columns with header
- Empty list -> `No tasks found.`
- Show matches spec — aligned labels, omitted empty sections
- Stats three groups with right-aligned numbers
- Priority P0-P4 labels always present
- Long titles truncated in list
- All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Implements full Formatter interface | PASS — compile-time `var _ Formatter = &PrettyFormatter{}` check present | PASS — test asserts `var _ Formatter = &PrettyFormatter{}` (no compile-time var but tested) |
| List matches spec format — aligned columns with header | PASS — dynamic column widths, header + data rows with format strings | PASS — dynamic column widths, header + data rows with format strings |
| Empty list -> `No tasks found.` | PASS — checked at top of FormatTaskList | PASS — checked after quiet mode branch |
| Show matches spec — aligned labels, omitted empty sections | PASS — hardcoded 10-char label width, omits empty BlockedBy/Children/Description | PASS — dynamic label width based on longest label, omits empty sections including Parent/Closed |
| Stats three groups with right-aligned numbers | PASS — hardcoded `%2d` format for right-alignment | PASS — dynamic `%*d` width computed from max value across all fields |
| Priority P0-P4 labels always present | PASS — always prints all 5 priority lines | PASS — always prints all 5 priority lines |
| Long titles truncated in list | PASS — `maxListTitleLen = 60`, truncates with `...` | PASS — `maxListTitleLen = 50`, truncates with `...` |
| All output matches spec examples | PASS — column-aligned, no borders/colors | PASS — column-aligned, no borders/colors |

## Implementation Comparison

### Approach

Both versions create a `PrettyFormatter` struct with no fields that implements all six `Formatter` interface methods. The core structural differences stem from the two versions having **different Formatter interface signatures** defined in their predecessor tasks (4-1 format resolution), which forces different parameter types throughout.

**Interface signature differences (from prior tasks, not this one):**

V2 (`formatter.go`):
```go
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatStats(w io.Writer, stats interface{}) error
```

V4 (`format.go`):
```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatStats(w io.Writer, stats StatsData) error
```

Key interface differences affecting this task:
1. V4's `FormatTaskList` accepts a `quiet bool` parameter — V2 does not.
2. V4's `FormatTaskDetail` takes `TaskDetail` (exported, value) — V2 takes `*showData` (unexported, pointer).
3. V4's `FormatStats` takes `StatsData` (concrete type) — V2 takes `interface{}` and must type-assert.
4. V4's `FormatDepChange` takes a `quiet bool` parameter — V2 does not.
5. V4's `FormatTransition` uses `string` for status — V2 uses `task.Status`.

#### FormatTaskList

Both compute dynamic column widths by iterating over data to find the widest ID and STATUS values.

V2 uses separate gutter constants:
```go
idCol := idWidth + 3
statusCol := statusWidth + 2
priCol := 5 // "PRI" + 2 spaces
```
Then uses **two different format strings** — one for the header (string PRI) and one for data rows (integer PRI):
```go
headerFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%ds%%s\n", idCol, statusCol, priCol)
rowFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%dd%%s\n", idCol, statusCol, priCol)
```

V4 uses a **uniform 2-space gutter** between all columns and a **single format string** for both header and data rows, converting priority to string via `strconv.Itoa`:
```go
fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", idWidth, statusWidth, priWidth)
```
V4 also dynamically computes PRI column width (checking if any priority has more than 1 digit), while V2 hardcodes it at 5.

V4 additionally handles `quiet` mode (outputs only IDs, no header):
```go
if quiet {
    for _, r := range rows {
        fmt.Fprintln(w, r.ID)
    }
    return nil
}
```

**Truncation threshold**: V2 uses `maxListTitleLen = 60`, V4 uses `maxListTitleLen = 50`.

V4's `truncateTitle` has an edge-case guard for `maxLen <= 3`:
```go
if maxLen <= 3 {
    return "..."
}
```
V2 lacks this guard, which could panic with a negative slice index if `maxLen < 3` (unlikely but a real defensive coding difference).

#### FormatTaskDetail

V2 uses **hardcoded 10-character label alignment** via literal spacing:
```go
fmt.Fprintf(w, "ID:       %s\n", data.ID)
fmt.Fprintf(w, "Title:    %s\n", data.Title)
fmt.Fprintf(w, "Status:   %s\n", data.Status)
fmt.Fprintf(w, "Priority: %d\n", data.Priority)
```

V4 uses **dynamic label alignment** computed from the longest label:
```go
labelWidth := len("Priority") // 8 chars — longest base label
// ... checks Parent, Closed lengths
fmtStr := fmt.Sprintf("%%-%ds  %%s\n", labelWidth+1)
fmt.Fprintf(w, fmtStr, "ID:", detail.ID)
```

V2 includes a `ParentTitle` field that shows the parent task's title alongside its ID:
```go
if data.ParentTitle != "" {
    fmt.Fprintf(w, "Parent:   %s  %s\n", data.Parent, data.ParentTitle)
}
```
V4 only shows the Parent ID, since `TaskDetail` has no `ParentTitle` field.

V2 converts priority with `%d` format verb directly. V4 converts via `strconv.Itoa(detail.Priority)` to pass to a `%s` format string (since all fields use the same format string).

Both versions omit empty BlockedBy, Children, and Description sections identically. V4 also omits Parent and Closed when empty.

#### FormatStats

V2 uses **hardcoded `%2d` format** for right-alignment (assumes max 2-digit numbers):
```go
fmt.Fprintf(w, "Total:       %2d\n", sd.Total)
fmt.Fprintf(w, "  Open:        %2d\n", sd.Open)
```

V4 uses **fully dynamic alignment** — computes `maxNumWidth` from the widest number and `maxStatusLabel` / `maxPriLabel` from the longest label in each section:
```go
maxNumWidth := 1
for _, v := range allValues {
    w := len(strconv.Itoa(v))
    if w > maxNumWidth { maxNumWidth = w }
}
fmt.Fprintf(w, "%-*s  %*d\n", totalLabelWidth, "Total:", maxNumWidth, stats.Total)
```

V2's hardcoded `%2d` will break alignment if counts exceed 99. V4's dynamic approach handles arbitrary sizes.

V2 requires a type assertion from `interface{}`:
```go
sd, ok := stats.(*StatsData)
if !ok {
    return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
}
```
V4 receives `StatsData` directly (concrete type), no assertion needed.

V4 also uses label arrays and loops for status/workflow/priority sections:
```go
statusLabels := []string{"Open:", "In Progress:", "Done:", "Cancelled:"}
for i, label := range statusLabels {
    fmt.Fprintf(w, "  %-*s  %*d\n", maxStatusLabel, label, maxNumWidth, statusValues[i])
}
```
V2 writes each line individually with hardcoded format strings.

#### FormatTransition

Both produce identical output: `"{id}: {old} -> {new}\n"` with Unicode arrow. V2 uses `task.Status` type; V4 uses plain strings. Both check/return errors identically.

#### FormatDepChange

Both handle "added" and "removed" actions with the same output messages. V2 has a fallback `default` case for unknown actions. V4 does not — unknown actions silently produce no output. V4 adds quiet mode support that suppresses all output.

#### FormatMessage

Both are identical — `fmt.Fprintln(w, message)`.

### Code Quality

**Compile-time interface check:**
V2 has an explicit compile-time check (line 19):
```go
var _ Formatter = &PrettyFormatter{}
```
V4 relies on the test assertion only. V2's approach is the more idiomatic Go pattern — it catches interface violations at compile time, not just at test time.

**Type safety:**
V4's `FormatStats` signature uses concrete `StatsData` (no type assertion needed), which is genuinely better type safety. V2 must assert from `interface{}`, introducing a runtime failure mode (tested, but still a code smell). This is an inherited interface design difference, not a choice made in this task.

**Error handling:**
V2 checks and returns errors from `fmt.Fprintf` in FormatTaskList (both header and rows) and FormatTransition/FormatDepChange/FormatMessage. V4 ignores the error returns from `fmt.Fprintf`/`fmt.Fprintln` throughout (e.g., `fmt.Fprintln(w, "No tasks found.")` — return value discarded). Since the `io.Writer` is typically a buffer or stdout, write errors are rare but V2 is more correct.

However, V2 is inconsistent — `FormatTaskDetail` and `FormatStats` also discard `fmt.Fprintf` errors.

**DRY:**
V4 is more DRY in FormatStats, using arrays and loops instead of individual lines. V2's approach is more readable/explicit but repetitive.

**Naming:**
V2 uses `maxListTitleLen` (exported-style constant, but lowercase so unexported — correct).
V4 uses the same name. Both are fine.

V2 parameter naming is slightly less clear: `FormatDepChange(w, action, taskID, blockedByID)` vs V4's `FormatDepChange(w, taskID, blockedByID, action, quiet)` — V4 groups the IDs together then adds action and quiet, which reads more naturally.

**Imports:**
V2 imports `github.com/leeovery/tick/internal/task` (for `task.Status`). V4 imports only stdlib (`strconv`, `strings`, `fmt`, `io`). V4's lack of internal dependencies is cleaner for this package.

### Test Quality

#### V2 Test Functions (439 lines)

| # | Function | Test Name | Edge Cases |
|---|----------|-----------|------------|
| 1 | `TestPrettyFormatterImplementsInterface` | "it implements the full Formatter interface" | Compile-time check in test |
| 2 | `TestPrettyFormatterFormatTaskList` | "it formats list with aligned columns" | Exact string comparison of 2-row output |
| 3 | | "it aligns with variable-width data" | Checks STATUS column position alignment across rows |
| 4 | | "it shows 'No tasks found.' for empty list" | Exact string match |
| 5 | | "it truncates long titles in list" | 100-char title, checks `...` suffix and absence of full title |
| 6 | `TestPrettyFormatterFormatTaskDetail` | "it formats show with all sections" | Exact string comparison with all fields + BlockedBy + Children + Description |
| 7 | | "it omits empty sections in show" | Checks absence of "Blocked by:", "Children:", "Description:" + exact string match |
| 8 | | "it does not truncate in show" | 100-char title, checks full title present |
| 9 | `TestPrettyFormatterFormatStats` | "it formats stats with all groups, right-aligned" | Exact string comparison of full stats output |
| 10 | | "it shows zero counts in stats" | All-zero stats, checks presence of all section headers and labels |
| 11 | | "it renders P0-P4 priority labels" | Checks all 5 labels present via substring |
| 12 | | "it returns error for non-StatsData input" | Passes string instead of *StatsData, checks error returned |
| 13 | `TestPrettyFormatterFormatTransitionAndDep` | "it formats transition as plain text" | Exact string match with Unicode arrow |
| 14 | | "it formats dep add as plain text" | Exact string match |
| 15 | | "it formats dep removed as plain text" | Exact string match |
| 16 | | "it formats message as plain text" | Exact string match |

Total: **16 test cases** across 4 top-level test functions.

#### V4 Test Functions (644 lines)

| # | Function | Test Name | Edge Cases |
|---|----------|-----------|------------|
| 1 | `TestPrettyFormatter_ImplementsFormatter` | "it implements the full Formatter interface" | Compile-time check in test |
| 2 | `TestPrettyFormatter_FormatTaskList` | "it formats list with aligned columns" | Checks line count, header columns, data content via substring |
| 3 | | "it aligns with variable-width data" | Checks TITLE column position alignment + STATUS column alignment |
| 4 | | "it shows 'No tasks found.' for empty list" | Checks text + absence of headers |
| 5 | | "it truncates long titles in list" | 100-char title, checks `...` and absence of full title |
| 6 | | "it outputs only IDs in quiet mode" | Quiet mode: checks 2 lines, ID-only output |
| 7 | `TestPrettyFormatter_FormatTaskDetail` | "it formats show with all sections" | Checks presence of all fields, values, sections, and content via substring |
| 8 | | "it omits empty sections in show" | Checks absence of Parent, Closed, Blocked by, Children, Description |
| 9 | | "it does not truncate in show" | 100-char title present |
| 10 | | "it aligns labels in show output" | Computes value start positions, checks all aligned |
| 11 | `TestPrettyFormatter_FormatStats` | "it formats stats with all groups, right-aligned" | Checks presence of all labels and group headers |
| 12 | | "it shows zero counts in stats" | All-zero stats, checks all labels and priority labels present |
| 13 | | "it renders P0-P4 priority labels" | Checks all 5 P0-P4 labels with descriptors |
| 14 | | "it right-aligns numbers in stats" | Finds Open/In Progress lines, checks trailing position matches |
| 15 | `TestPrettyFormatter_FormatTransition` | "it formats transition as plain text" | Exact string match |
| 16 | `TestPrettyFormatter_FormatDepChange` | "it formats dep added as plain text" | Exact string match |
| 17 | | "it formats dep removed as plain text" | Exact string match |
| 18 | | "it suppresses dep output in quiet mode" | Quiet mode: checks empty output |
| 19 | `TestPrettyFormatter_FormatMessage` | "it formats message as plain text" | Exact string match |

Total: **19 test cases** across 6 top-level test functions.

#### Test Quality Comparison

**Assertion style:**
V2 uses exact string comparison for most tests (e.g., `if buf.String() != want`), which is more strict and catches any formatting regression. V4 primarily uses `strings.Contains` checks (presence-based assertions), which is more resilient to minor format changes but less precise — it could pass even if extra unwanted content appears.

Exception: V4's FormatTransition/FormatDepChange/FormatMessage tests use exact string matching.

**Stats right-alignment testing:**
V2 tests right-alignment via exact string comparison — the expected output literal contains the precise spacing. V4 has a dedicated test ("it right-aligns numbers in stats") that programmatically verifies column alignment by computing line-end positions. V4's approach is more explicit about testing the alignment property itself.

**Unique to V2:**
- Test for `FormatStats` with non-StatsData input (error path) — line 12 above. This tests the `interface{}` type assertion failure, which is specific to V2's interface design.

**Unique to V4:**
- Quiet mode for `FormatTaskList` — line 6 above.
- Label alignment verification in show — line 10 above (programmatically checks all values start at same column).
- Quiet mode for `FormatDepChange` — line 18 above.
- Explicit right-alignment test for stats — line 14 above.

**Test gaps:**
- Neither version tests `FormatTaskDetail` with Parent field populated (V2 tests it without parent; V4's "all sections" test includes Parent).
- V2 does not test the `Closed` field in show output.
- V4's "all sections" test includes both Parent and Closed, making it more thorough.
- Neither version tests multiline description rendering (both implement it but only test single-line descriptions).
- Neither version tests `FormatDepChange` with an unknown action.
- V2 does not test quiet mode (it is not in V2's interface).

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 4 | 4 |
| Lines added | 633 (630 in impl/test) | 901 (898 in impl/test) |
| Impl LOC | 191 | 254 |
| Test LOC | 439 | 644 |
| Test functions | 16 test cases across 4 top-level functions | 19 test cases across 6 top-level functions |

## Verdict

**V4 is the better implementation**, with caveats.

**V4 advantages:**

1. **Dynamic alignment everywhere.** V4 computes widths dynamically in FormatStats (label widths and number widths), FormatTaskDetail (label widths), and FormatTaskList (including PRI column). V2 hardcodes `%2d` for stats numbers (breaks at 100+) and fixed 10-char label width in show.

2. **More complete test coverage.** V4 has 19 test cases vs V2's 16, including dedicated right-alignment verification for stats, label alignment verification for show, and quiet mode tests. These are genuinely additional edge cases, not padding.

3. **Better type safety.** V4's `FormatStats(w, StatsData)` takes a concrete type. V2's `FormatStats(w, interface{})` requires a runtime type assertion with an error path — a code smell inherited from V2's interface design.

4. **Defensive truncation.** V4's `truncateTitle` guards against `maxLen <= 3`, preventing a potential panic.

5. **Quiet mode support.** V4 handles quiet mode in FormatTaskList and FormatDepChange, which is beyond the task spec but useful for the broader CLI.

6. **Fewer internal dependencies.** V4 imports only stdlib packages; V2 imports `internal/task` for `task.Status`.

**V2 advantages:**

1. **Compile-time interface check.** `var _ Formatter = &PrettyFormatter{}` is more idiomatic Go than relying on a test.

2. **Stricter test assertions.** V2's exact string comparisons for list, show, and stats output catch any formatting drift. V4's substring-based checks for show and stats are more tolerant but less precise.

3. **Better error propagation.** V2 returns errors from `fmt.Fprintf` in FormatTaskList and some passthrough methods. V4 silently discards write errors in most methods.

4. **Default case in FormatDepChange.** V2 handles unknown actions with a fallback message. V4 silently ignores unknown actions.

5. **Simpler code.** V2 is 191 LOC vs V4's 254 LOC, achieving the same core functionality more concisely.

**Net assessment:** V4's dynamic alignment and stronger test suite outweigh V2's advantages. The hardcoded `%2d` in V2's stats is a real limitation, and V4's additional test cases (right-alignment verification, label alignment, quiet mode) cover genuinely important behavior. V2's stricter assertion style and compile-time check are better practices that V4 should have adopted, but they represent minor quality differences rather than correctness issues.
