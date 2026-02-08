# Task tick-core-5-1: tick stats Command

## Task Summary

Implement `tick stats` -- aggregate counts by status, priority, and workflow state (ready/blocked). Output via the Formatter interface in all three formats (TOON, Pretty, JSON).

**Requirements:**
- `StatsQuery` in the query/storage layer: query SQLite for counts grouped by status, priority, and workflow state
- Stats struct: `Total`, `ByStatus` (open/in_progress/done/cancelled), `Workflow` (ready/blocked), `ByPriority` (P0-P4, always 5 entries)
- Ready/blocked counts reuse the ready query logic from Phase 3
- CLI handler calls `StatsQuery`, passes result to `Formatter.FormatStats()`
- TOON: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` + `by_priority[5]{priority,count}:`
- Pretty: Three groups (Total, Status breakdown, Workflow, Priority with P0-P4 labels), right-aligned numbers
- JSON: Nested object with `total`, `by_status`, `workflow`, `by_priority`
- `--quiet`: suppress output entirely

**Required Tests:**
1. "it counts tasks by status correctly"
2. "it counts ready and blocked tasks correctly"
3. "it includes all 5 priority levels even at zero"
4. "it returns all zeros for empty project"
5. "it formats stats in TOON format"
6. "it formats stats in Pretty format with right-aligned numbers"
7. "it formats stats in JSON format with nested structure"
8. "it suppresses output with --quiet"

**Acceptance Criteria:**
1. StatsQuery returns correct counts by status, priority, workflow
2. All 5 priority levels always present (P0-P4)
3. Ready/blocked counts match Phase 3 query semantics
4. Empty project returns all zeros with full structure
5. TOON format matches spec example
6. Pretty format matches spec example with right-aligned numbers
7. JSON format nested with correct keys
8. `--quiet` suppresses all output

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| StatsQuery returns correct counts by status, priority, workflow | PASS -- queries status/priority/workflow via GROUP BY and reused WHERE fragments | PASS -- same approach with extracted helper functions |
| All 5 priority levels always present (P0-P4) | PASS -- `[5]int` array guarantees 5 entries; tested in "it includes all 5 priority levels even at zero" | PASS -- identical `[5]int` array; tested in `TestStats_IncludesAll5PriorityLevelsEvenAtZero` |
| Ready/blocked counts match Phase 3 query semantics | PASS -- reuses `readyWhere` and `blockedWhere` constants from `list.go` | PASS -- reuses `readyConditionsFor()` from `ready.go`; blocked computed as `Open - readyCount` |
| Empty project returns all zeros with full structure | PASS -- tested in "it returns all zeros for empty project" | PASS -- tested in `TestStats_ReturnsAllZerosForEmptyProject` |
| TOON format matches spec example | PASS -- tested with exact string matching of headers and data rows | PASS -- tested with exact string matching of headers and data rows |
| Pretty format matches spec with right-aligned numbers | PASS -- hardcoded `%2d` format widths; tested for presence of labels and right-aligned `" 3"` | PASS -- dynamically computed column widths using `%-*s %*d` format; tested for presence of all labels |
| JSON format nested with correct keys | PASS -- tested all top-level keys and nested keys; validates values | PASS -- uses typed `jsonStats` struct for unmarshaling; validates values and raw key presence |
| `--quiet` suppresses all output | PASS -- quiet check after DB query; tested output is empty | PASS -- quiet check before DB query (early return); tested output is empty |

## Implementation Comparison

### Approach

Both versions create a `stats.go` file with a `runStats` method on `*App`, query SQLite for status counts, priority counts, and workflow (ready/blocked) counts, then delegate rendering to the formatter.

**Command Registration:**

V2 registers in `app.go` with no args passed:
```go
case "stats":
    return a.runStats()
```

V4 registers in `cli.go` passing `subArgs`:
```go
case "stats":
    if err := a.runStats(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```
V4's pattern includes explicit error writing and exit code returns, consistent with V4's overall architecture where `Run()` returns `int`. V2's `Run()` returns `error`.

**Quiet Handling (genuinely different):**

V2 checks quiet *after* performing all database queries (line 98):
```go
if a.config.Quiet {
    return nil
}
return a.formatter.FormatStats(a.stdout, &stats)
```

V4 checks quiet *before* any database work (line 12):
```go
if a.Quiet {
    return nil
}
```
V4's early return is genuinely better -- it avoids opening the store and running 3 queries when the result will be discarded. This is a minor efficiency win but demonstrates better design thinking.

**Code Organization (genuinely different):**

V2 implements all query logic inline within the `store.Query` callback as a single monolithic block (103 lines total). Status query, priority query, ready count, and blocked count are all sequential blocks within one closure.

V4 extracts three named helper functions:
```go
func queryStatusCounts(db *sql.DB, stats *StatsData) error { ... }
func queryPriorityCounts(db *sql.DB, stats *StatsData) error { ... }
func queryWorkflowCounts(db *sql.DB, stats *StatsData) error { ... }
```
Called from a clean orchestrator:
```go
err = s.Query(func(db *sql.DB) error {
    if err := queryStatusCounts(db, &stats); err != nil {
        return err
    }
    if err := queryPriorityCounts(db, &stats); err != nil {
        return err
    }
    if err := queryWorkflowCounts(db, &stats); err != nil {
        return err
    }
    return nil
})
```
V4's decomposition is genuinely better for readability, testability, and maintenance. Each function has a clear doc comment explaining its responsibility.

**Blocked Count Computation (genuinely different):**

V2 runs a *separate SQL query* using `blockedWhere` (a dedicated WHERE fragment with `IN` subqueries):
```go
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + blockedWhere).Scan(&blockedCount)
```
Where `blockedWhere` is:
```go
const blockedWhere = `status = 'open'
  AND (
    id IN (SELECT d.task_id FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE t.status NOT IN ('done', 'cancelled'))
    OR id IN (SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress'))
  )`
```

V4 computes blocked as the *complement* of ready within open tasks:
```go
stats.Blocked = stats.Open - readyCount
```
V4's approach is genuinely better: it avoids a redundant database query, is mathematically equivalent (blocked = open - ready, since every open task is either ready or blocked), and is simpler to reason about. It also avoids maintaining a separate `blockedWhere` constant that must be kept in sync with `readyWhere`.

**Total Count Computation:**

V2 sums the four status counts in Go:
```go
stats.Total = stats.Open + stats.InProgress + stats.Done + stats.Cancelled
```

V4 accumulates the total during the status row scan:
```go
for rows.Next() {
    // ...
    stats.Total += count
    switch status { ... }
}
```
Different but equivalent. V4 is slightly more resilient to unknown status values (they'd be counted in total), though neither handles that case explicitly.

**Ready Query Reuse:**

V2 uses `readyWhere` (a `const string` in `list.go`) without table alias:
```go
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + readyWhere).Scan(&readyCount)
```

V4 uses `readyConditionsFor("t")` (a function in `ready.go`) with a table alias parameter:
```go
readyCountQuery := `SELECT COUNT(*) FROM tasks t WHERE t.status = 'open' AND` + readyConditionsFor("t")
```
V4's function-based approach is more flexible (parameterized alias), but both correctly reuse Phase 3 ready logic.

**Formatter Interface Type Safety (genuinely different):**

V2 uses `interface{}` parameter:
```go
FormatStats(w io.Writer, stats interface{}) error
```
Every formatter must type-assert:
```go
func (f *JSONFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
```

V4 uses concrete `StatsData` type:
```go
FormatStats(w io.Writer, stats StatsData) error
```
No type assertion needed:
```go
func (f *JSONFormatter) FormatStats(w io.Writer, stats StatsData) error {
    byPriority := make([]jsonPriorityEntry, 5)
```
V4 is genuinely better: compile-time type safety, no runtime assertion overhead, and the value-type parameter (`StatsData` not `*StatsData`) is appropriate for a small struct.

**Pretty Formatter Right-Alignment:**

V2 uses hardcoded format widths:
```go
fmt.Fprintf(w, "Total:       %2d\n", sd.Total)
fmt.Fprintf(w, "  Open:        %2d\n", sd.Open)
fmt.Fprintf(w, "  P0 (critical): %2d\n", sd.ByPriority[0])
```

V4 dynamically computes column widths based on the largest number and longest label:
```go
maxNumWidth := 1
for _, v := range allValues {
    w := len(strconv.Itoa(v))
    if w > maxNumWidth { maxNumWidth = w }
}
fmt.Fprintf(w, "%-*s  %*d\n", totalLabelWidth, "Total:", maxNumWidth, stats.Total)
```
V4's approach is genuinely better: it scales correctly when numbers exceed 2 digits (e.g., 100+ tasks). V2's `%2d` would break alignment for 3+ digit numbers.

**TOON Formatter:**

V2 uses inline `fmt.Fprintf` calls:
```go
fmt.Fprintln(w, "stats{total,open,in_progress,done,cancelled,ready,blocked}:")
fmt.Fprintf(w, "  %d,%d,%d,%d,%d,%d,%d\n", sd.Total, sd.Open, sd.InProgress, sd.Done, sd.Cancelled, sd.Ready, sd.Blocked)
```

V4 extracts helper methods `buildStatsSection` and `buildByPrioritySection`, joining sections with newlines:
```go
func (f *ToonFormatter) FormatStats(w io.Writer, stats StatsData) error {
    var sections []string
    sections = append(sections, f.buildStatsSection(stats))
    sections = append(sections, f.buildByPrioritySection(stats.ByPriority))
    fmt.Fprint(w, strings.Join(sections, "\n"))
    return nil
}
```
V4's helper decomposition is consistent with its other formatter methods. Different but roughly equivalent in this case.

### Code Quality

**Go Idioms:**

V2 uses unexported fields (`a.config.Quiet`, `a.workDir`, `a.formatter`, `a.stdout`, `a.newStore`). V4 uses exported fields (`a.Quiet`, `a.Dir`, `a.Formatter`, `a.Stdout`, `a.openStore`). Both patterns are valid; V4's exported fields make testing simpler (direct struct construction without setters).

**Error Handling:**

Both properly check `rows.Err()` after iterating. Both wrap SQL errors with `fmt.Errorf("failed to ...: %w", err)`. V4 explicitly checks `rows.Err()` via `return rows.Err()` at the end of each helper (clean idiom). V2 checks inline with `if err := rows.Err(); err != nil { return err }`.

**DRY:**

V2 has the blocked query logic duplicated (once in `blockedWhere` const, once in the SQL). V4 computes blocked as `Open - Ready`, eliminating the need for a blocked query entirely.

**Naming:**

V2: `store` for the store variable, `readyCount`/`blockedCount` as local vars.
V4: `s` for the store variable (shorter, less descriptive), `readyCount` as local var.

### Test Quality

**V2 Test Functions (all inside `TestStats`):**

1. `TestStats/"it counts tasks by status correctly"` -- 7 tasks across all 4 statuses, checks total=7, open=2, in_progress=1, done=3, cancelled=1 via JSON output
2. `TestStats/"it counts ready and blocked tasks correctly"` -- 7 tasks with blockers, parents, children; checks ready=3, blocked=2. Comments explain why each task is ready/blocked
3. `TestStats/"it includes all 5 priority levels even at zero"` -- 2 tasks both P2; checks all 5 entries present, P2=2, others=0
4. `TestStats/"it returns all zeros for empty project"` -- empty project; checks total=0, all statuses=0, workflow=0, 5 priority entries all 0
5. `TestStats/"it formats stats in TOON format"` -- 2 tasks; checks header strings, data row, by_priority header, all 5 priority lines
6. `TestStats/"it formats stats in Pretty format with right-aligned numbers"` -- 3 tasks; checks all section headers, all P0-P4 labels, right-alignment of number 3
7. `TestStats/"it formats stats in JSON format with nested structure"` -- 2 tasks; checks all top-level and nested JSON keys exist, verifies by_priority structure (priority values 0-4, priority/count keys), verifies actual values
8. `TestStats/"it suppresses output with --quiet"` -- 2 tasks; checks output is empty string

**V4 Test Functions (separate top-level functions):**

1. `TestStats_CountsTasksByStatusCorrectly/"it counts tasks by status correctly"` -- 7 tasks across all 4 statuses, checks total=7, open=2, in_progress=1, done=3, cancelled=1 via typed `jsonStats` struct
2. `TestStats_CountsReadyAndBlockedCorrectly/"it counts ready and blocked tasks correctly"` -- 5 tasks (no done/cancelled like V2 has); checks ready=3, blocked=2. Comments explain each task's state
3. `TestStats_IncludesAll5PriorityLevelsEvenAtZero/"it includes all 5 priority levels even at zero"` -- 2 tasks both P2; checks 5 entries, verifies priority values 0-4, expected counts [0,0,2,0,0]
4. `TestStats_ReturnsAllZerosForEmptyProject/"it returns all zeros for empty project"` -- empty project; checks total=0, all statuses=0, workflow=0, 5 priorities all 0
5. `TestStats_FormatsToonFormat/"it formats stats in TOON format"` -- 2 tasks; checks header, data row, by_priority header, 3 of 5 priority rows (P0, P1, P2 only)
6. `TestStats_FormatsPrettyFormatWithRightAlignedNumbers/"it formats stats in Pretty format with right-aligned numbers"` -- 4 tasks; checks all section headers, all sub-labels including "Open:", "In Progress:", "Done:", "Cancelled:", "Ready:", "Blocked:", and all P0-P4 labels
7. `TestStats_FormatsJSONFormatWithNestedStructure/"it formats stats in JSON format with nested structure"` -- 3 tasks (includes cancelled); validates typed struct fields, also verifies raw JSON string contains expected keys
8. `TestStats_SuppressesOutputWithQuiet/"it suppresses output with --quiet"` -- 1 task; checks output is empty string

**Test Setup Approach:**

V2 uses `taskJSONL()` helper to create JSONL strings manually, then `setupTickDirWithContent()` to write them to a file. This is a string-based approach that constructs raw data.

V4 uses `[]task.Task` struct literals passed to `setupInitializedDirWithTasks()`. This is a type-safe approach that works with the domain model directly.

V4's approach is genuinely better: compile-time checked, self-documenting field names, and less error-prone than string construction.

**JSON Assertion Approach:**

V2 uses `map[string]interface{}` and manual type assertions with `float64` casts:
```go
var result map[string]interface{}
json.Unmarshal([]byte(stdout.String()), &result)
total := int(result["total"].(float64))
```

V4 uses a typed `jsonStats` struct:
```go
var result jsonStats
json.Unmarshal(stdout.Bytes(), &result)
if result.Total != 7 { ... }
```
V4's approach is genuinely better: cleaner, no manual type assertions, compile-time verified field access.

**Test Coverage Gaps:**

- V2 TOON test checks all 5 priority rows; V4 only checks 3 (P0, P1, P2). V2 is more thorough here.
- V2 Pretty test checks for right-alignment with `strings.Contains(line, " 3")`; V4 checks labels but does not verify number alignment. V2 is more thorough here.
- V4 Pretty test checks more labels (individual "Open:", "In Progress:", "Done:", "Cancelled:", "Ready:", "Blocked:" labels); V2 only checks section headers. V4 is more thorough here.
- V4 JSON test uses 3 tasks (including cancelled); V2 uses 2. V4 has slightly broader coverage.
- V4 JSON test verifies raw JSON key presence in addition to typed struct; V2 verifies keys via `map[string]interface{}`. Roughly equivalent.
- V2 quiet test uses 2 tasks; V4 uses 1. No meaningful difference.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 5 (app.go, stats.go, stats_test.go, 2 docs) | 5 (cli.go, stats.go, stats_test.go, 2 docs) |
| Lines added | 533 | 521 |
| Impl LOC (stats.go) | 103 | 119 |
| Test LOC (stats_test.go) | 425 | 393 |
| Test functions | 8 (subtests in 1 top-level) | 8 (subtests in 8 top-level) |

## Verdict

**V4 is the better implementation.** The evidence:

1. **Type safety**: V4's `FormatStats(w io.Writer, stats StatsData)` signature provides compile-time safety. V2's `interface{}` parameter requires runtime type assertions in every formatter, adding boilerplate and runtime failure risk.

2. **Blocked computation**: V4's `Blocked = Open - Ready` is mathematically elegant, avoids a redundant database query, and eliminates the need to maintain a separate `blockedWhere` constant. V2 runs 4 queries; V4 runs 3.

3. **Code decomposition**: V4 extracts `queryStatusCounts`, `queryPriorityCounts`, and `queryWorkflowCounts` as named functions with doc comments. V2 has all logic in a single monolithic closure.

4. **Quiet optimization**: V4 checks `--quiet` before opening the database. V2 runs all queries first, then discards the result.

5. **Pretty formatting**: V4 dynamically computes column widths for right-alignment, scaling correctly for any number magnitude. V2 hardcodes `%2d` which breaks for 100+ tasks.

6. **Test setup**: V4 uses type-safe `[]task.Task` struct literals. V2 uses string-based `taskJSONL()` helper.

V2 has a minor advantage in TOON test thoroughness (all 5 priority rows checked vs V4's 3) and in the Pretty test's explicit right-alignment verification. However, these are small test coverage differences that do not outweigh V4's architectural advantages.
