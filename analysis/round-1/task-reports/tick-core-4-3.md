# Task tick-core-4-3: Human-readable formatter -- list, show, stats output

## Task Summary

Implement `PrettyFormatter`, a concrete implementation of the `Formatter` interface that produces human-readable, column-aligned terminal output. No borders, colors, or icons -- plain `fmt.Print` output.

**Required methods:**
- **FormatTaskList**: Column-aligned table with header, dynamic widths. Empty list produces `"No tasks found."` with no headers. Long titles truncated with `...`.
- **FormatTaskDetail**: Key-value with aligned labels. Sections: base fields, Blocked by (indented), Children (indented), Description (indented block). Omit empty sections.
- **FormatStats**: Three groups (total, status breakdown, workflow counts, priority with P0-P4 labels). Right-align numbers. All rows present even at zero.
- **FormatTransition / FormatDepChange / FormatMessage**: Plain text passthrough.

**Acceptance criteria:**
1. Implements full Formatter interface
2. List matches spec format -- aligned columns with header
3. Empty list produces `"No tasks found."`
4. Show matches spec -- aligned labels, omitted empty sections
5. Stats three groups with right-aligned numbers
6. Priority P0-P4 labels always present
7. Long titles truncated in list
8. All output matches spec examples

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Implements full Formatter interface | PASS -- implements all 6 methods on `PrettyFormatter`, uses same types as interface (io.Writer + error return) | PASS -- implements all 6 methods; additionally has compile-time `var _ Formatter = &PrettyFormatter{}` check | PASS -- implements all 6 methods; uses return-string signature instead of io.Writer (different interface shape) |
| List: aligned columns with header | PASS -- dynamic widths for ID/STATUS, 2-space gutter, PRI fixed 4-char | PASS -- dynamic widths with explicit 3-space/2-space gutters, separate header/row format strings | PASS -- dynamic widths for ID/STATUS/PRI, 2-space gutter between all columns |
| Empty list: "No tasks found." | PASS | PASS -- also handles `nil` data pointer | PASS -- also handles `nil` data pointer |
| Show: aligned labels, omit empty sections | PASS -- hardcoded 10-char alignment; omits BlockedBy/Children/Description/Parent/Closed when empty | PASS -- same 10-char alignment; omits same sections; Parent uses string ID + optional title | PASS -- uses `%-10s` format (labelWidth=10); **BUG**: omits Updated when `Updated == Created`, which is NOT in the spec |
| Stats: three groups, right-aligned numbers | PARTIAL -- uses `%d` format (no width specifier), numbers are NOT right-aligned | PASS -- uses `%2d` format for right-alignment | PARTIAL -- uses individual width specifiers per line (`%8d`, `%9d`, `%2d`, etc.) attempting right-alignment but numbers don't align to a consistent column |
| Priority P0-P4 labels always present | PASS -- iterates fixed `[5]string` array of labels | PASS -- explicitly prints each P0-P4 line | PASS -- iterates `priorityLabels` map with `for i := 0; i <= 4` loop |
| Long titles truncated in list | PASS -- `maxTitleWidth = 60` | PASS -- `maxListTitleLen = 60` | PASS -- `maxTitleLength = 50` (different threshold from V1/V2) |
| Output matches spec examples | PARTIAL -- no exact-match tests; stats numbers not right-aligned | PASS -- exact string comparison tests for list, show, and stats | PARTIAL -- tests use `strings.Contains` checks only, not exact match; stats right-alignment uses inconsistent column widths |

## Implementation Comparison

### Approach

**Interface signatures differ fundamentally across all three versions** because each version works against a different `Formatter` interface definition from its base commit.

**V1** uses the simplest interface: methods take `(io.Writer, ValueType) error`:
```go
// V1 interface (from format.go)
FormatTaskList(w io.Writer, tasks []TaskListItem) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, data TransitionData) error
FormatDepChange(w io.Writer, data DepChangeData) error
FormatStats(w io.Writer, data StatsData) error
```
All data types are value types, passed by value. `TaskDetail` has `Parent *RelatedTask` (pointer to struct). `StatsData.ByPriority` is `[5]int`. `TransitionData` and `DepChangeData` are dedicated structs.

**V2** uses a refactored interface with typed status parameters and `interface{}` for stats:
```go
// V2 interface (from formatter.go)
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatStats(w io.Writer, stats interface{}) error
```
`FormatTaskDetail` receives `*showData` (pointer). `FormatTransition` takes `task.Status` typed parameters. `FormatStats` takes `interface{}` and does a runtime type assertion to `*StatsData`. `StatsData.ByPriority` is `[5]int`. Parent is a string ID + optional `ParentTitle` string.

**V3** uses a completely different pattern -- methods return `string` instead of writing to `io.Writer`:
```go
// V3 interface (from format.go)
FormatTaskList(data *TaskListData) string
FormatTaskDetail(data *TaskDetailData) string
FormatTransition(taskID, oldStatus, newStatus string) string
FormatDepChange(action, taskID, blockedByID string) string
FormatStats(data *StatsData) string
FormatMessage(msg string) string
```
All methods return `string`. Input data uses wrapper types (`*TaskListData` wrapping `[]TaskRowData`, `*TaskDetailData`). `StatsData.ByPriority` is `[]PriorityCount` (slice of structs with `Priority` and `Count` fields). No `io.Writer` anywhere.

---

**FormatTaskList column formatting:**

V1 builds a single format string for all rows:
```go
fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-4s %%s\n", idW, statusW)
fmt.Fprintf(w, fmtStr, t.ID, t.Status, fmt.Sprintf("%d", t.Priority), title)
```
Priority is formatted as a string with `fmt.Sprintf("%d", t.Priority)` into a `%-4s` slot, which means it is left-aligned with 4-char width.

V2 builds separate header and row format strings with explicit gutter calculations:
```go
idCol := idWidth + 3
statusCol := statusWidth + 2
priCol := 5 // "PRI" + 2 spaces
headerFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%ds%%s\n", idCol, statusCol, priCol)
rowFmt := fmt.Sprintf("%%-%ds%%-%ds%%-%dd%%s\n", idCol, statusCol, priCol)
```
Priority uses `%-5d` (integer formatting) in rows but `%-5s` (string) for the header. This is the most precise approach.

V3 uses `strings.Builder` and a single format pattern:
```go
sb.WriteString(fmt.Sprintf("%-*s  %-*s  %-*d  %s\n",
    idWidth, task.ID,
    statusWidth, task.Status,
    priWidth, task.Priority,
    title))
```
V3 also dynamically sizes the PRI column by scanning `strconv.Itoa(task.Priority)` lengths, which V1/V2 do not.

---

**FormatTaskDetail structure:**

V1 takes `TaskDetail` by value with `Parent *RelatedTask`, rendering parent as `ID  Title (Status)`:
```go
if d.Parent != nil {
    fmt.Fprintf(w, "Parent:   %s  %s (%s)\n", d.Parent.ID, d.Parent.Title, d.Parent.Status)
}
```

V2 takes `*showData` with Parent as a string ID and optional ParentTitle:
```go
if data.Parent != "" {
    if data.ParentTitle != "" {
        fmt.Fprintf(w, "Parent:   %s  %s\n", data.Parent, data.ParentTitle)
    } else {
        fmt.Fprintf(w, "Parent:   %s\n", data.Parent)
    }
}
```
Note V2 does NOT include parent status in parentheses.

V3 takes `*TaskDetailData` with Parent string, uses `%-10s` width formatting:
```go
sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "ID:", data.ID))
```
V3 has a subtle **bug**: it omits `Updated` when `Updated == Created`:
```go
if data.Updated != "" && data.Updated != data.Created {
    sb.WriteString(fmt.Sprintf("%-*s%s\n", labelWidth, "Updated:", data.Updated))
}
```
The spec says nothing about omitting Updated when it equals Created. V1 and V2 always show Updated.

V3 also changes the field ordering: Parent appears after Closed, whereas V1 places Parent before Created and V2 places Parent before Created/Updated.

---

**FormatStats right-alignment:**

V1 uses plain `%d` with no width specifier:
```go
fmt.Fprintf(w, "Total:       %d\n", data.Total)
fmt.Fprintf(w, "  Open:        %d\n", data.Open)
```
Numbers are NOT right-aligned -- single-digit and multi-digit numbers will not line up.

V2 uses `%2d` consistently for all stats values:
```go
fmt.Fprintf(w, "Total:       %2d\n", sd.Total)
fmt.Fprintf(w, "  Open:        %2d\n", sd.Open)
fmt.Fprintf(w, "  P0 (critical): %2d\n", sd.ByPriority[0])
```
Numbers right-align within a 2-char field. For values >= 100 this would overflow, but `%2d` still works (just widens).

V3 uses variable-width format specifiers per line:
```go
sb.WriteString(fmt.Sprintf("Total:%8d\n", data.Total))
sb.WriteString(fmt.Sprintf("  Open:%9d\n", data.Open))
sb.WriteString(fmt.Sprintf("  In Progress:%2d\n", data.InProgress))
sb.WriteString(fmt.Sprintf("  Done:%9d\n", data.Done))
```
The widths (`%8d`, `%9d`, `%2d`) are calculated to make numbers end at the same column position. For priority labels, V3 uses a computed width: `fmt.Sprintf("  %s:%*d\n", label, 17-len(label), count)`.

---

**FormatDepChange action string handling:**

V1 checks `data.Action == "added"` vs fallthrough for "removed":
```go
if data.Action == "added" {
    // ...
}
// else: removed
```

V2 uses a `switch` with a `default` case that handles unknown actions gracefully:
```go
switch action {
case "added":
    // ...
case "removed":
    // ...
default:
    _, err := fmt.Fprintf(w, "Dependency %s: %s %s %s\n", action, taskID, action, blockedByID)
    return err
}
```

V3 checks `action == "add"` (not "added"):
```go
if action == "add" {
    return fmt.Sprintf("Dependency added: %s blocked by %s\n", taskID, blockedByID)
}
return fmt.Sprintf("Dependency removed: %s no longer blocked by %s\n", taskID, blockedByID)
```
The action verb difference ("add" vs "added") suggests different calling conventions in V3's interface.

---

**FormatStats data types for priority:**

V1 and V2 use `[5]int` (fixed-size array, index = priority level).

V3 uses `[]PriorityCount` (slice of `{Priority int, Count int}` structs) and iterates with a linear scan:
```go
for i := 0; i <= 4; i++ {
    label := priorityLabels[i]
    count := 0
    for _, pc := range data.ByPriority {
        if pc.Priority == i {
            count = pc.Count
            break
        }
    }
}
```
This is more flexible (sparse priorities) but O(n*m) vs O(1) for the fixed array.

### Code Quality

**Go idioms:**

V1 is the most concise and idiomatic. It writes directly to `io.Writer` in each method with no intermediate buffering. The `truncateTitle` function is clean. However, it lacks a compile-time interface check.

V2 adds the compile-time interface assertion `var _ Formatter = &PrettyFormatter{}` (line 20), which is idiomatic Go. It uses `task.Status` typed parameters for `FormatTransition`, improving type safety. The `switch` in `FormatDepChange` with a default case is defensive. However, `FormatStats` takes `interface{}` and does a runtime type assertion, which is un-idiomatic and fragile:
```go
func (f *PrettyFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
```

V3 returns `string` instead of writing to `io.Writer`. This is a design choice that makes testing simpler (direct string comparison) but adds memory allocation for every format call and requires the caller to write the string. V3 uses `strings.Builder` for string construction, which is idiomatic. The `priorityLabels` map is a good approach for label lookups. V3 also handles `nil` data in `FormatTaskDetail` and `FormatStats`, returning empty string -- a defensive but potentially silent-failure approach.

**Naming:**

V1 uses `maxTitleWidth` (clear). V2 uses `maxListTitleLen` (more precise, specifying "list"). V3 uses `maxTitleLength` (generic). V2's name is best as it clarifies the scope.

V1 file is named `format_pretty.go`; V2 and V3 use `pretty_formatter.go`. The V2/V3 name is more conventional (matches `toon_formatter.go`).

**Error handling:**

V1 and V2 return errors from `fmt.Fprintf` in some methods but silently ignore them in others (e.g., `FormatTaskDetail` does `fmt.Fprintf(w, ...)` without capturing the error). V2 is slightly better -- it checks errors in `FormatTaskList` for both header and row writes.

V3 never returns errors (returns `string`), so error handling is delegated to the caller's `io.Writer.Write()`.

**DRY:**

All three versions have similar levels of repetition in stats formatting. V3's `priorityLabels` map and loop is the DRYest approach for priority labels. V1 uses a loop with an array of labels. V2 explicitly writes each priority line.

### Test Quality

**V1 Test Functions** (file: `internal/cli/format_pretty_test.go`, 228 lines):
1. `TestPrettyFormatterFormatTaskList`
   - `"it formats list with aligned columns"` -- checks line count and header contains ID/STATUS
   - `"it aligns with variable-width data"` -- checks in_progress appears in output
   - `"it shows 'No tasks found.' for empty list"` -- exact string match
   - `"it truncates long titles in list"` -- 80-char title, checks truncation and `...`
2. `TestPrettyFormatterFormatTaskDetail`
   - `"it formats show with all sections"` -- `strings.Contains` for ID, Blocked by, Children, Description
   - `"it omits empty sections in show"` -- checks absence of Blocked by, Children, Description, Parent
   - `"it does not truncate in show"` -- 80-char title, full title appears
3. `TestPrettyFormatterFormatStats`
   - `"it formats stats with all groups, right-aligned"` -- `strings.Contains` for Total, Status, Workflow, Priority
   - `"it shows zero counts in stats"` -- zero StatsData, checks Total present
   - `"it renders P0-P4 priority labels"` -- checks all 5 labels
4. `TestPrettyFormatterFormatTransition`
   - `"it formats transition as plain text"` -- exact string match

**V2 Test Functions** (file: `internal/cli/pretty_formatter_test.go`, 439 lines):
1. `TestPrettyFormatterImplementsInterface`
   - `"it implements the full Formatter interface"` -- compile-time check
2. `TestPrettyFormatterFormatTaskList`
   - `"it formats list with aligned columns"` -- **exact string match** of full output
   - `"it aligns with variable-width data"` -- verifies STATUS column position matches header across rows
   - `"it shows 'No tasks found.' for empty list"` -- exact string match
   - `"it truncates long titles in list"` -- 100-char title, checks `...` suffix and absence of full title
3. `TestPrettyFormatterFormatTaskDetail`
   - `"it formats show with all sections"` -- **exact string match** of full output
   - `"it omits empty sections in show"` -- checks absence of sections AND exact string match of expected output
   - `"it does not truncate in show"` -- 100-char title, full title appears
4. `TestPrettyFormatterFormatStats`
   - `"it formats stats with all groups, right-aligned"` -- **exact string match** of full output
   - `"it shows zero counts in stats"` -- all-zero data, checks all sections present + P0/P4 labels
   - `"it renders P0-P4 priority labels"` -- checks all 5 label strings
   - `"it returns error for non-StatsData input"` -- tests `interface{}` type assertion failure
5. `TestPrettyFormatterFormatTransitionAndDep`
   - `"it formats transition as plain text"` -- exact match, uses `task.StatusOpen`/`task.StatusInProgress`
   - `"it formats dep add as plain text"` -- exact match
   - `"it formats dep removed as plain text"` -- exact match
   - `"it formats message as plain text"` -- exact match

**V3 Test Functions** (file: `internal/cli/pretty_formatter_test.go`, 676 lines):
1. `TestPrettyFormatter`
   - `"it implements Formatter interface"` -- compile-time check
2. `TestPrettyFormatterFormatTaskList`
   - `"it formats list with aligned columns"` -- `strings.Contains` checks for header/data (no exact match)
   - `"it aligns with variable-width data"` -- checks TITLE column position alignment
   - `"it shows 'No tasks found.' for empty list"` -- exact string match
   - `"it truncates long titles in list"` -- 100-char title, checks `...` and absence of full title
3. `TestPrettyFormatterFormatTaskDetail`
   - `"it formats show with all sections"` -- `strings.Contains` checks for all labels, blocker data, children, description
   - `"it omits empty sections in show"` -- checks absence + checks basic fields still present
   - `"it does not truncate in show"` -- 100-char title, full title appears
4. `TestPrettyFormatterFormatStats`
   - `"it formats stats with all groups, right-aligned"` -- `strings.Contains` for labels and values
   - `"it shows zero counts in stats"` -- checks labels present + verifies InProgress shows 0
   - `"it renders P0-P4 priority labels"` -- checks all 5 labels (**BUG**: uses `&&` instead of `||` -- `!strings.Contains(result, "P0") && !strings.Contains(result, "critical")` passes even if "P0" is present but "critical" isn't)
5. `TestPrettyFormatterFormatTransition`
   - `"it formats transition as plain text"` -- exact string match
6. `TestPrettyFormatterFormatDepChange`
   - `"it formats dep add as plain text"` -- exact string match (uses "add" action)
   - `"it formats dep rm as plain text"` -- exact string match (uses "remove" action)
7. `TestPrettyFormatterFormatMessage`
   - `"it formats message as plain text"` -- exact string match
8. `TestPrettyFormatterSpecExamples`
   - `"it matches spec list example format"` -- checks header prefix, verifies column positions exist
   - `"it matches spec show example format"` -- `strings.Contains` checks for labels and sections
   - `"it matches spec stats example format"` -- `strings.Contains` checks for all section headers and priority labels

**Test coverage differences:**

| Edge case / test | V1 | V2 | V3 |
|------------------|-----|-----|-----|
| Exact output string match (list) | No | Yes | No |
| Exact output string match (show) | No | Yes | No |
| Exact output string match (stats) | No | Yes | No |
| Column position alignment verification | No | Yes (STATUS col) | Yes (TITLE col) |
| Interface compile-time check test | No | Yes | Yes |
| FormatDepChange add | No | Yes | Yes |
| FormatDepChange remove | No | Yes | Yes |
| FormatMessage | No | Yes | Yes |
| FormatStats type assertion error | No | Yes | N/A (no interface{}) |
| Spec example conformance tests | No | No | Yes (3 tests) |
| Parent omission when nil/empty | Yes | No (not tested) | No (not tested) |
| Closed field omission | No | No | No |
| nil data pointer handling | No | No | No (despite impl) |
| Zero-count value verification | No | Yes (section presence) | Yes (InProgress=0 check) |

**Unique to V1:** Tests parent omission explicitly (`"should omit Parent when nil"`).

**Unique to V2:** Exact string match tests for list, show, and stats. Tests FormatStats type assertion error. Most rigorous output verification.

**Unique to V3:** Dedicated spec example tests (`TestPrettyFormatterSpecExamples`) with 3 sub-tests. Tests FormatMessage separately. However, spec example tests use `strings.Contains` rather than exact match, reducing their value.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4 | 5 |
| Lines added | 370 | 633 | 901 |
| Impl LOC | 142 | 191 | 206 |
| Test LOC | 228 | 439 | 676 |
| Test functions (sub-tests) | 10 | 15 | 17 |

V1 is the most compact. V3 is the largest, partly due to `TestPrettyFormatterSpecExamples` and the return-string pattern requiring more test boilerplate.

## Verdict

**V2 is the best implementation.**

**Evidence:**

1. **Strictest test verification.** V2 is the only version with exact string comparison tests for all three major outputs (list, show, stats). This catches alignment/spacing regressions that `strings.Contains` tests in V1 and V3 would miss. For example, V2's list test:
   ```go
   want := "ID          STATUS       PRI  TITLE\n" +
       "tick-a1b2   done         1    Setup Sanctum\n" +
       "tick-c3d4   in_progress  1    Login endpoint\n"
   if buf.String() != want {
   ```

2. **Correct right-alignment in stats.** V2 uses consistent `%2d` formatting across all stats lines, producing clean right-aligned numbers. V1 uses `%d` (no alignment at all). V3 uses per-line width specifiers that attempt alignment but with inconsistent approaches.

3. **Compile-time interface check.** `var _ Formatter = &PrettyFormatter{}` prevents the implementation from drifting out of sync with the interface.

4. **Robust FormatDepChange.** The `switch` with `default` case handles unexpected action strings instead of silently falling through.

5. **Type-safe FormatTransition.** Uses `task.Status` typed parameters instead of raw strings.

6. **No spec violations.** V3 has a bug where it omits `Updated` when `Updated == Created`, which is not called for by the spec. V1 has no right-alignment in stats. V2 has no known spec violations.

**V2 weaknesses:** `FormatStats` taking `interface{}` is un-idiomatic -- this is a constraint from V2's `Formatter` interface definition, not a V2-specific choice. It correctly handles this with a type assertion and error return.

**V3 notable positives:** The return-string pattern simplifies testing (no buffer setup), the `priorityLabels` map is DRY, and the `PriorityCount` struct type is more flexible than `[5]int`. However, the `Updated == Created` bug and weaker test assertions outweigh these.

**V1 notable positives:** Most concise at 142 impl LOC. Clean and simple. But lacks exact-match tests, right-alignment in stats, and coverage of dep/message formatting.
