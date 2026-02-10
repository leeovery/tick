# Task 4-3: Human-readable formatter -- list, show, stats output

## Task Plan Summary

Implement `PrettyFormatter`, a concrete implementation of the `Formatter` interface for human-readable terminal output. Requirements:

- **FormatTaskList**: Column-aligned table with header, dynamic column widths, empty list produces `No tasks found.` (no headers), long titles truncated with `...`.
- **FormatTaskDetail**: Key-value pairs with aligned labels. Sections: base fields, Blocked by (indented), Children (indented), Description (indented block). Empty sections omitted entirely. Titles never truncated.
- **FormatStats**: Three groups (total, status/workflow breakdown, priority P0-P4 labels). Right-aligned numbers. All rows present even at zero.
- **FormatTransition/FormatDepChange/FormatMessage**: Plain text passthrough.
- Style: Minimalist and clean. No borders, colors, or icons.
- 10 specified test cases covering alignment, variable-width data, empty results, all sections, omitted sections, stats groups, zero counts, P0-P4 labels, title truncation in list, no truncation in show.

---

## V4 Implementation

### Architecture & Design

V4 uses an `App` struct pattern where the `Formatter` is stored on the `App` and all command handlers are methods on `App`. The Formatter interface uses **decomposed parameters** -- each method receives its data as individual arguments or value types:

```go
// V4 Formatter interface (format.go)
type Formatter interface {
    FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
    FormatTaskDetail(w io.Writer, detail TaskDetail) error
    FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
    FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
    FormatStats(w io.Writer, stats StatsData) error
    FormatMessage(w io.Writer, msg string) error
}
```

Key design choices:

1. **Data types are exported structs**: `TaskDetail`, `RelatedTask`, `StatsData` are all exported with uppercase fields, defined in `format.go`. The `listRow` type is unexported with uppercase fields (`ID`, `Status`, `Priority`, `Title`), defined in `list.go`.

2. **`quiet` is a Formatter concern**: The `quiet` flag is passed into `FormatTaskList` and `FormatDepChange`, making the formatter responsible for quiet-mode behavior. This mixes output format logic with command flow control.

3. **Value receivers for data types**: `FormatTaskDetail` receives `TaskDetail` by value, `FormatStats` receives `StatsData` by value. No pointers.

4. **Dynamic label alignment in `FormatTaskDetail`**: V4 computes the label width dynamically based on which fields are present:
   ```go
   labelWidth := len("Priority") // 8 chars - longest base label
   if detail.Parent != "" && len("Parent") > labelWidth {
       labelWidth = len("Parent")
   }
   ```
   However, this logic is flawed: `len("Parent")` is 6, which is never greater than 8 (`len("Priority")`), so the Parent check is dead code. The Closed check (`len("Closed")` = 6) is similarly dead. The label width is always 8 ("Priority").

5. **FormatMessage returns error**: All V4 Formatter methods return `error`, including `FormatMessage`. The `PrettyFormatter.FormatMessage` always returns `nil`.

6. **`maxListTitleLen` constant**: Title truncation max is defined as a package-level `const maxListTitleLen = 50`.

7. **Stats number alignment uses global max**: V4's `FormatStats` computes `maxNumWidth` across ALL values (including priority values), then uses that single width for everything:
   ```go
   allValues := []int{stats.Total, stats.Open, stats.InProgress, ...}
   for _, v := range stats.ByPriority {
       allValues = append(allValues, v)
   }
   maxNumWidth := 1
   for _, v := range allValues { ... }
   ```
   This means priority numbers and status numbers share the same column width. The minimum width is 1.

8. **PRI column is left-aligned**: The format string for list output left-aligns all columns including PRI:
   ```go
   fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", idWidth, statusWidth, priWidth)
   ```
   Priority values are converted to string via `strconv.Itoa(r.Priority)` and left-aligned.

### Code Quality

- **254 lines** of implementation code.
- All exported functions/types are documented.
- `truncateTitle` is a clean standalone helper, handles edge case `maxLen <= 3` by returning `"..."`.
- Error return values from `fmt.Fprintf` are consistently ignored (no `_, err :=` pattern). This is acceptable for `io.Writer` writes in a CLI context but not ideal.
- Variable naming is clear: `idWidth`, `statusWidth`, `priWidth`, `maxNumWidth`.
- The `ParentTitle` field exists on `TaskDetail` but the diff's `FormatTaskDetail` implementation only uses `detail.Parent`, not `detail.ParentTitle`. Looking at the full source, V4 actually does handle `ParentTitle` in the final version:
  ```go
  if detail.Parent != "" {
      parentVal := detail.Parent
      if detail.ParentTitle != "" {
          parentVal = detail.Parent + " " + detail.ParentTitle
      }
      fmt.Fprintf(w, fmtStr, "Parent:", parentVal)
  }
  ```
  This differs from the diff which showed no `ParentTitle` handling -- the full source file is the authoritative version.

### Test Coverage

V4 has **644 lines** of test code across 17 test functions organized into test groups:

1. **TestPrettyFormatter_ImplementsFormatter** -- compile-time interface check.
2. **TestPrettyFormatter_FormatTaskList** (5 subtests):
   - "it formats list with aligned columns" -- checks header + 2 data rows, verifies `strings.Contains` for column names and data values.
   - "it aligns with variable-width data" -- checks `strings.Index` column positions align.
   - "it shows 'No tasks found.' for empty list" -- checks empty output, verifies no headers.
   - "it truncates long titles in list" -- 100-char title, checks `...` presence and absence of full title.
   - "it outputs only IDs in quiet mode" -- checks quiet mode outputs IDs only.
3. **TestPrettyFormatter_FormatTaskDetail** (4 subtests):
   - "it formats show with all sections" -- checks fields/sections via `strings.Contains`.
   - "it omits empty sections in show" -- checks `!strings.Contains` for empty sections.
   - "it does not truncate in show" -- 100-char title preserved.
   - "it aligns labels in show output" -- calculates value start positions to verify alignment.
4. **TestPrettyFormatter_FormatStats** (4 subtests):
   - "it formats stats with all groups, right-aligned" -- checks group headers present.
   - "it shows zero counts in stats" -- all labels present with zeros.
   - "it renders P0-P4 priority labels" -- checks P0-P4 descriptors.
   - "it right-aligns numbers in stats" -- compares line ending positions.
5. **TestPrettyFormatter_FormatTransition** (1 subtest): exact string comparison.
6. **TestPrettyFormatter_FormatDepChange** (3 subtests): added, removed, quiet mode.
7. **TestPrettyFormatter_FormatMessage** (1 subtest): exact string comparison.

**Test quality assessment**:
- Tests use `strings.Contains` extensively rather than exact output comparison, which makes them more resilient to spacing changes but less precise. A test could pass even if the output has extra unexpected content.
- The alignment test for list output (`strings.Index`) is well-designed -- it verifies column positions match between header and data rows.
- The stats right-alignment test compares `len(strings.TrimRight(line, " \t"))` which correctly checks that numbers end at the same column.
- The label alignment test for show output is thorough, computing value start positions programmatically.
- Tests use descriptive names matching the plan's test list.

### Spec Compliance

| Requirement | Met? | Notes |
|---|---|---|
| Implements full Formatter interface | YES | Compile-time check in tests |
| List aligned columns with header | YES | Dynamic widths computed |
| Empty list -> "No tasks found." | YES | No headers shown |
| Show aligned labels, omitted empty sections | YES | Dynamic label width, conditional sections |
| Stats three groups, right-aligned numbers | YES | All groups present, right-aligned |
| Priority P0-P4 labels always present | YES | Even at zero counts |
| Long titles truncated in list | YES | maxListTitleLen = 50 |
| Titles not truncated in show | YES | No truncation applied |
| PRI column alignment | PARTIAL | Left-aligned rather than right-aligned, which is unusual for numeric columns |

### golang-pro Skill Compliance

| Rule | Met? | Notes |
|---|---|---|
| Table-driven tests with subtests | PARTIAL | Uses subtests (`t.Run`) but not table-driven format. Each test is a separate function body, not iterating over test cases. |
| Document all exported functions/types | YES | All documented |
| Handle all errors explicitly | PARTIAL | `fmt.Fprintf` return values ignored throughout |
| Propagate errors with fmt.Errorf("%w") | N/A | No error wrapping needed in this file |
| No panic for normal error handling | YES | No panics |
| No ignored errors without justification | PARTIAL | `fmt.Fprintf` errors silently ignored |

---

## V5 Implementation

### Architecture & Design

V5 uses a `Context` struct pattern where the `Formatter` is stored on `Context.Fmt`. The Formatter interface uses **typed pointer parameters** for complex data, but the diff showed `interface{}` parameters while the final source uses concrete types:

```go
// V5 Formatter interface (format.go) -- final version
type Formatter interface {
    FormatTaskList(w io.Writer, rows []TaskRow) error
    FormatTaskDetail(w io.Writer, data *showData) error
    FormatTransition(w io.Writer, data *TransitionData) error
    FormatDepChange(w io.Writer, data *DepChangeData) error
    FormatStats(w io.Writer, data *StatsData) error
    FormatMessage(w io.Writer, msg string)
}
```

Key design choices:

1. **Data types split across files**: `TaskRow`, `StatsData`, `TransitionData`, `DepChangeData` are defined in `toon_formatter.go` (the first formatter implemented). `showData` and `relatedTask` are unexported types with lowercase fields defined in `show.go`.

2. **`quiet` is NOT a Formatter concern**: The `quiet` flag is handled by the command handler (e.g., `runList`, `runDepAdd`), which prints IDs directly and returns before calling the formatter. This is cleaner separation of concerns.

3. **Pointer receivers for struct data**: `FormatTaskDetail` receives `*showData`, `FormatStats` receives `*StatsData`, `FormatTransition` receives `*TransitionData`, `FormatDepChange` receives `*DepChangeData`. This avoids copying large structs.

4. **Fixed label width in `FormatTaskDetail`**: V5 uses a hardcoded 10-character label column:
   ```go
   fmt.Fprintf(w, "%-10s%s\n", "ID:", d.id)
   fmt.Fprintf(w, "%-10s%s\n", "Title:", d.title)
   ```
   This is simpler than V4's dynamic calculation and produces consistent output regardless of which optional fields are present. The longest label "Priority:" is 9 characters, plus 1 space = 10 chars total.

5. **FormatMessage returns no error**: `FormatMessage(w io.Writer, msg string)` has no error return in V5. This is pragmatic -- a simple `fmt.Fprintln` is extremely unlikely to fail in practice, and not all callers need to handle that error.

6. **Shared helper functions for Toon/Pretty**: V5 defines `formatTransitionText`, `formatDepChangeText`, and `formatMessageText` in `format.go` as shared helpers:
   ```go
   func (f *PrettyFormatter) FormatTransition(w io.Writer, data *TransitionData) error {
       return formatTransitionText(w, data)
   }
   ```
   This avoids code duplication between ToonFormatter and PrettyFormatter for identical behavior.

7. **PRI column is right-aligned**: V5 uses separate format strings for header and data rows:
   ```go
   fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%%ds  %%s\n", idW, statusW, priW)     // header: PRI as string
   priFmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%%dd  %%s\n", idW, statusW, priW)   // data: PRI as right-aligned int
   ```
   This right-aligns priority numbers, which is the correct behavior for numeric columns.

8. **`numWidth` helper with minimum of 3**: V5 extracts number width calculation into a reusable function with a minimum width of 3:
   ```go
   func numWidth(nums []int) int {
       w := 3
       for _, n := range nums { ... }
       return w
   }
   ```
   The minimum of 3 ensures consistent minimum spacing. This also separates summary/priority number widths (computed independently).

9. **Stats alignment uses independent group widths**: V5's `FormatStats` uses a hardcoded `summaryLabelW = 12` (matching "In Progress:" as the widest label) and computes `priLabelW` dynamically for the priority group. Numbers are independently sized per group:
   ```go
   summaryNumW := numWidth(summaryNums)
   priNumW := numWidth(d.ByPriority[:])
   ```

10. **FormatDepChange handles unknown action**: V5 returns an error for unknown action values:
    ```go
    default:
        return fmt.Errorf("FormatDepChange: unknown action %q", d.Action)
    ```
    V4 silently ignores unknown actions.

11. **Description empty check uses TrimSpace**: V5 checks `strings.TrimSpace(d.description) != ""` to handle whitespace-only descriptions as empty. V4 checks `detail.Description != ""` which would render a whitespace-only description.

### Code Quality

- **251 lines** of implementation code (slightly shorter than V4's 254).
- All exported functions/types are documented.
- `truncateTitle` handles `maxWidth <= 3` differently: `strings.Repeat(".", maxWidth)` vs V4's `"..."`. For `maxWidth` of 1 or 2, V5 produces "." or ".." while V4 produces "...". V5's approach is more correct since it respects the width constraint.
- Error return values from `fmt.Fprintf` are checked in `FormatTaskList`:
  ```go
  if _, err := fmt.Fprintf(w, fmtStr, "ID", "STATUS", "PRI", "TITLE"); err != nil {
      return err
  }
  ```
  But NOT checked in `FormatTaskDetail` or `FormatStats`. This is inconsistent.
- The shared helper pattern (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) eliminates code duplication between ToonFormatter and PrettyFormatter.
- Using unexported `showData` with lowercase fields (`d.id`, `d.title`) tightly couples the formatter to the CLI package. However, since `PrettyFormatter` is in the same package, this is acceptable.

### Test Coverage

V5 has **406 lines** of test code across 5 test groups:

1. **TestPrettyFormatterImplementsInterface** -- compile-time interface check.
2. **TestPrettyFormatterFormatTaskList** (4 subtests):
   - "it formats list with aligned columns" -- **exact string comparison** against expected output.
   - "it aligns with variable-width data" -- **exact string comparison**.
   - "it shows 'No tasks found.' for empty list" -- **exact string comparison**.
   - "it truncates long titles in list" -- checks truncated string present, full title absent.
3. **TestPrettyFormatterFormatTaskDetail** (3 subtests):
   - "it formats show with all sections" -- **exact string comparison** against multi-line expected.
   - "it omits empty sections in show" -- **exact string comparison** + `strings.Contains` negative checks.
   - "it does not truncate in show" -- checks full title present.
4. **TestPrettyFormatterFormatStats** (3 subtests):
   - "it formats stats with all groups, right-aligned" -- **exact string comparison** against full expected output.
   - "it shows zero counts in stats" -- `strings.Contains` checks for all groups.
   - "it renders P0-P4 priority labels" -- `strings.Contains` for label strings.
5. **TestPrettyFormatterTransitionAndDep** (4 subtests):
   - "it formats transition as plain text" -- exact string comparison.
   - "it formats dep add as plain text" -- exact string comparison.
   - "it formats dep removed as plain text" -- exact string comparison.
   - "it formats message as plain text" -- exact string comparison.

**Test quality assessment**:
- **Exact string comparison** is the dominant pattern. This is more rigorous than V4's `strings.Contains` approach. Any change to formatting (extra space, wrong alignment) will immediately cause test failure. Example:
  ```go
  expected := "ID         STATUS       PRI  TITLE\n" +
      "tick-a1b2  done           1  Setup Sanctum\n" +
      "tick-c3d4  in_progress    1  Login endpoint\n"
  if buf.String() != expected {
      t.Errorf("output =\n%s\nwant =\n%s", buf.String(), expected)
  }
  ```
- The stats test with exact comparison is particularly valuable -- it verifies the exact spacing, right-alignment, and grouping.
- Missing test from plan: No explicit "it outputs only IDs in quiet mode" test. However, in V5 quiet mode is handled at the command level, not in the formatter, so this test is appropriately absent from PrettyFormatter tests.
- Tests are shorter (406 vs 644 lines) but achieve equal or better coverage through exact comparisons.

### Spec Compliance

| Requirement | Met? | Notes |
|---|---|---|
| Implements full Formatter interface | YES | Compile-time check in tests |
| List aligned columns with header | YES | Dynamic widths, exact output verified |
| Empty list -> "No tasks found." | YES | No headers shown |
| Show aligned labels, omitted empty sections | YES | Fixed 10-char label column, conditional sections |
| Stats three groups, right-aligned numbers | YES | Independent group alignment, exact output verified |
| Priority P0-P4 labels always present | YES | Even at zero counts |
| Long titles truncated in list | YES | maxTitleWidth = 50 |
| Titles not truncated in show | YES | No truncation applied |
| PRI column alignment | YES | Right-aligned numeric column |

### golang-pro Skill Compliance

| Rule | Met? | Notes |
|---|---|---|
| Table-driven tests with subtests | PARTIAL | Uses subtests (`t.Run`) but not table-driven format |
| Document all exported functions/types | YES | All documented |
| Handle all errors explicitly | PARTIAL | `fmt.Fprintf` errors checked in FormatTaskList but not in FormatTaskDetail/FormatStats |
| Propagate errors with fmt.Errorf("%w") | YES | `formatDepChangeText` uses `fmt.Errorf` for unknown action |
| No panic for normal error handling | YES | No panics |
| No ignored errors without justification | PARTIAL | Inconsistent error checking across methods |

---

## Comparative Analysis

### Where V4 is Better

1. **More exhaustive test count**: V4 has 17 subtests vs V5's 14. V4 includes:
   - An explicit quiet-mode test for `FormatTaskList` (though V5 correctly omits this since quiet is not a formatter concern in V5).
   - A dedicated label-alignment test for show output that programmatically verifies value start positions.
   - A dedicated right-alignment test for stats that compares line ending positions.

2. **Dynamic label width calculation in show**: V4 computes `labelWidth` based on which optional fields are present. While the current logic has dead code (Parent/Closed can never exceed Priority width), the approach is more adaptive to future label changes. V5's hardcoded `%-10s` would need manual updating if a longer label were added.

3. **`FormatMessage` returns `error`**: V4's `FormatMessage` returning `error` is more consistent with the rest of the interface. V5's void return breaks the pattern where all format methods return errors.

### Where V5 is Better

1. **Cleaner interface design -- quiet separated from formatting**: V5 handles `quiet` mode at the command level, not inside the formatter. This is architecturally superior:
   - V4: `FormatTaskList(w, rows, quiet)` -- the formatter decides what quiet means.
   - V5: The command handler checks `ctx.Quiet`, prints IDs, and never calls the formatter.
   - This means V5 formatters are purely about rendering, not flow control.

2. **Exact string comparison in tests**: V5's tests compare against complete expected output strings. This catches subtle bugs like wrong spacing, missing newlines, or wrong alignment that V4's `strings.Contains` tests would miss. Example -- V5 verifies the exact stats output:
   ```go
   expected := "Total:       47\n" +
       "\n" +
       "Status:\n" +
       "  Open:        12\n" +
       "  In Progress:  3\n" + ...
   ```
   V4 only checks that labels are present, not that the numbers are correctly right-aligned in the output.

3. **Right-aligned PRI column**: V5 right-aligns the PRI numeric column using `%d` formatting:
   ```go
   priFmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%%dd  %%s\n", idW, statusW, priW)
   ```
   V4 left-aligns PRI as a string via `%%-%ds` + `strconv.Itoa()`. Right-alignment is the conventional formatting for numeric columns in tabular output.

4. **Shared text helpers reduce duplication**: V5 defines `formatTransitionText`, `formatDepChangeText`, and `formatMessageText` as shared helpers used by both ToonFormatter and PrettyFormatter. V4 duplicates the identical logic in both formatters.

5. **Typed data parameters**: V5 uses concrete pointer types (`*TransitionData`, `*DepChangeData`, `*StatsData`) instead of V4's decomposed parameters (`id string, oldStatus string, newStatus string`). Typed parameters:
   - Are self-documenting (the struct field names describe the data).
   - Are easier to extend (adding a field doesn't change the method signature).
   - Avoid argument ordering mistakes.

6. **Independent number width per stats group**: V5 computes `summaryNumW` and `priNumW` independently:
   ```go
   summaryNumW := numWidth(summaryNums)
   priNumW := numWidth(d.ByPriority[:])
   ```
   V4 uses a single `maxNumWidth` across all groups. V5's approach produces tighter formatting when priority and status counts have very different magnitudes.

7. **`numWidth` minimum of 3**: V5's `numWidth` returns at least 3, ensuring consistent minimum spacing. V4's `maxNumWidth` starts at 1, which could produce cramped output for single-digit values.

8. **Error handling for unknown action**: V5's `FormatDepChange` returns `fmt.Errorf("FormatDepChange: unknown action %q", d.Action)` for unknown actions. V4 silently ignores unknown actions. Explicit error handling is more robust.

9. **Whitespace-only description handling**: V5 checks `strings.TrimSpace(d.description) != ""`, correctly treating whitespace-only descriptions as empty. V4 checks `detail.Description != ""`, which would render a description section containing only whitespace.

10. **`truncateTitle` correctness for small maxWidth**: V5's `strings.Repeat(".", maxWidth)` correctly produces "." for maxWidth=1 and ".." for maxWidth=2. V4's `"..."` would produce a 3-character string even when maxWidth is 1 or 2, violating the width constraint.

### Differences That Are Neutral

1. **Exported vs unexported data types**: V4 uses exported `TaskDetail`/`RelatedTask` with uppercase fields. V5 uses unexported `showData`/`relatedTask` with lowercase fields. Both are used only within the `cli` package, so visibility doesn't matter for this task.

2. **Value vs pointer receivers for stats/detail data**: V4 passes `StatsData` by value, V5 passes `*StatsData` by pointer. For these small structs, the performance difference is negligible.

3. **Constant naming**: `maxListTitleLen` (V4) vs `maxTitleWidth` (V5). Both are clear.

4. **Test line count**: V4 has 644 lines vs V5's 406 lines. V5's tests are shorter because exact string comparison replaces multi-line assertion logic. Fewer lines of test code achieving equal or better coverage is generally positive, but could be considered neutral.

5. **Stats label width computation**: V4 dynamically computes `maxStatusLabel` by iterating labels. V5 hardcodes `summaryLabelW = 12` since "In Progress:" (12 chars) is always the widest. Both produce correct output; V5's approach is simpler but less adaptive.

---

## Verdict

**Winner: V5**

V5 is the stronger implementation for the following decisive reasons:

1. **Architectural superiority**: Separating quiet-mode handling from the Formatter interface is a clear improvement. Formatters should format; command handlers should control flow. V5's Formatter interface is cleaner and more focused.

2. **Test rigor**: V5's exact string comparison tests are categorically more precise than V4's `strings.Contains` assertions. A V5 test will catch any formatting regression -- wrong spacing, misalignment, extra whitespace -- while V4's tests could pass with subtly broken output. The stats exact-match test alone is worth more than V4's four stats subtests combined.

3. **Correct PRI alignment**: Right-aligning numeric columns is a fundamental formatting convention. V5 gets this right; V4 does not.

4. **Code reuse**: V5's shared text helpers (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) eliminate duplication between Toon and Pretty formatters. V4 duplicates this logic.

5. **Interface extensibility**: V5's typed struct parameters (`*TransitionData`, `*StatsData`) are more extensible than V4's decomposed parameters. Adding a field to `TransitionData` doesn't change any method signature.

6. **Edge case correctness**: V5 handles whitespace-only descriptions, unknown dep actions, and small truncation widths more correctly.

V4's only meaningful advantage is its slightly more exhaustive test suite (17 vs 14 subtests), but V5's tests achieve superior coverage through exact comparison, making this a net negative for V4. The label-alignment test in V4 is clever but unnecessary when you have exact output comparison.
