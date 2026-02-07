# Task tick-core-5-1: tick stats command

## Task Summary

Implement `tick stats` -- aggregate counts by status, priority, and workflow state (ready/blocked). Outputs via the Formatter interface in all three formats (TOON, Pretty, JSON).

**Requirements:**
- `StatsQuery` in the query/storage layer: query SQLite for counts grouped by status, priority, and workflow state
- Stats struct: `Total`, `ByStatus` (open/in_progress/done/cancelled), `Workflow` (ready/blocked), `ByPriority` (P0-P4 -- always 5 entries)
- Ready/blocked counts reuse the ready query logic from Phase 3 (open + unblocked + no open children = ready; open + not ready = blocked)
- CLI handler calls `StatsQuery`, passes result to `Formatter.FormatStats()`
- TOON: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` + `by_priority[5]{priority,count}:`
- Pretty: Three groups (Total, Status breakdown, Workflow, Priority with P0-P4 labels), right-aligned numbers
- JSON: Nested object with `total`, `by_status`, `workflow`, `by_priority`
- `--quiet`: suppress output entirely (stats has no mutation ID to return)

**Acceptance Criteria:**
1. StatsQuery returns correct counts by status, priority, workflow
2. All 5 priority levels always present (P0-P4)
3. Ready/blocked counts match Phase 3 query semantics
4. Empty project returns all zeros with full structure
5. TOON format matches spec example
6. Pretty format matches spec example with right-aligned numbers
7. JSON format nested with correct keys
8. --quiet suppresses all output

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. StatsQuery returns correct counts by status, priority, workflow | PASS -- queries status via GROUP BY, priority via GROUP BY, ready/blocked via subqueries | PASS -- identical query strategy, reuses `readyWhere`/`blockedWhere` constants from list.go | PASS -- identical query strategy, reuses `ReadyCondition`/`BlockedCondition` from ready.go/blocked.go |
| 2. All 5 priority levels always present (P0-P4) | PASS -- `[5]int` array guarantees 5 entries; formatter iterates 0-4 | PASS -- `[5]int` array guarantees 5 entries; formatter iterates 0-4 | PASS -- `[]PriorityCount` slice initialized with `make([]PriorityCount, 5)` and pre-filled 0-4 |
| 3. Ready/blocked counts match Phase 3 query semantics | PARTIAL -- inline SQL mirrors Phase 3 logic but does NOT reuse the actual constants; prone to drift | PASS -- directly references `readyWhere` and `blockedWhere` string constants from list.go | PASS -- directly references `ReadyCondition` and `BlockedCondition` from ready.go and blocked.go |
| 4. Empty project returns all zeros with full structure | PASS -- tested, `[5]int` zero-initialized | PASS -- tested, `[5]int` zero-initialized | PASS -- tested, slice pre-filled with zeros |
| 5. TOON format matches spec example | PASS -- `stats{total,...}:` header + `by_priority[5]{priority,count}:` | PASS -- identical format | PASS -- identical format |
| 6. Pretty format with right-aligned numbers | PASS -- fixed-width format strings | PASS -- `%2d` format strings with consistent alignment | PASS -- dynamic `%*d` alignment based on label length |
| 7. JSON format nested with correct keys | PARTIAL -- flat structure (no `by_status` or `workflow` nesting; all at top level) | PASS -- `by_status`, `workflow`, `by_priority` properly nested | PARTIAL -- `by_status` contains ready/blocked (spec says separate `workflow` key); no `workflow` key |
| 8. --quiet suppresses all output | PASS -- checks `a.opts.Quiet`, returns nil early | PASS -- checks `a.config.Quiet`, returns nil early | PASS -- checks `!a.formatConfig.Quiet`, skips fmt.Fprint |

## Implementation Comparison

### Approach

All three versions follow the same high-level pattern: discover tick directory, open store, run queries inside `store.Query()`, pass results to formatter. The key differences are in query reuse, data types, formatter signatures, and JSON structure.

**Command Registration:**

V1 registers in `cli.go` as `case "stats": err = a.cmdStats(workDir)` -- the handler takes `workDir` as a parameter and returns `error`. The App's `Run()` method handles error-to-exit-code conversion.

V2 registers in `app.go` as `case "stats": return a.runStats()` -- the handler takes no arguments (uses `a.workDir`) and returns `error`. The App's `Run()` method returns `error` directly.

V3 registers in `cli.go` as `case "stats": return a.runStats()` -- the handler takes no arguments (uses `a.Cwd`) and returns `int` (exit code). Error handling is done inside the handler.

**Query Implementation -- Ready/Blocked Reuse (Critical Difference):**

V1 (`stats.go` lines 72-91) writes the ready query **inline** as a raw SQL literal:
```go
err = db.QueryRow(`
    SELECT COUNT(*) FROM tasks t
    WHERE t.status = 'open'
      AND NOT EXISTS (
        SELECT 1 FROM dependencies d
        JOIN tasks blocker ON d.blocked_by = blocker.id
        WHERE d.task_id = t.id
          AND blocker.status NOT IN ('done', 'cancelled')
      )
      AND NOT EXISTS (
        SELECT 1 FROM tasks child
        WHERE child.parent = t.id
          AND child.status IN ('open', 'in_progress')
      )
`).Scan(&data.Ready)
// ...
data.Blocked = data.Open - data.Ready
```
This duplicates the Phase 3 logic and computes blocked as `open - ready` rather than using the actual blocked query.

V2 (`stats.go` lines 76-91) reuses the **unexported** `readyWhere` and `blockedWhere` constants from `list.go`:
```go
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + readyWhere).Scan(&readyCount)
// ...
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + blockedWhere).Scan(&blockedCount)
```
Both constants are unexported (lowercase), keeping them package-internal but properly shared.

V3 (`stats.go` lines 127-139) reuses the **exported** `ReadyCondition` and `BlockedCondition` constants from `ready.go` and `blocked.go`:
```go
readyQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + ReadyCondition
err = db.QueryRow(readyQuery).Scan(&data.Ready)
// ...
blockedQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + BlockedCondition
err = db.QueryRow(blockedQuery).Scan(&data.Blocked)
```
These are exported constants (PascalCase), with each defined in its own dedicated file (`ready.go`, `blocked.go`). V3 uses table alias `t` in the query, matching the conditions which assume `t` alias.

**Data Model:**

V1 and V2 both use `[5]int` for `ByPriority`:
```go
type StatsData struct {
    // ...
    ByPriority [5]int // index 0-4
}
```

V3 uses a richer `[]PriorityCount` slice:
```go
type StatsData struct {
    // ...
    ByPriority []PriorityCount
}
type PriorityCount struct {
    Priority int
    Count    int
}
```
V3's approach carries the priority label alongside the count, making the data self-describing. V1/V2 rely on array index convention.

**Formatter Interface Signature:**

V1: `FormatStats(w io.Writer, data StatsData) error` -- concrete value type, writes to `io.Writer`, returns error.

V2: `FormatStats(w io.Writer, stats interface{}) error` -- `interface{}` parameter requires type assertion inside every formatter. Writes to `io.Writer`, returns error.

V3: `FormatStats(data *StatsData) string` -- pointer parameter, returns string instead of writing to writer. Caller handles writing.

V2's `interface{}` approach loses type safety at the interface boundary:
```go
func (f *ToonFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
    // ...
}
```

**JSON Structure (Spec Compliance):**

The spec requires: `total`, `by_status`, `workflow`, `by_priority`.

V1 produces a **flat** JSON with all fields at the top level:
```go
obj := struct {
    Total      int `json:"total"`
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    // ... no by_status or workflow nesting
}
```
This does NOT match the spec's `by_status` and `workflow` nesting.

V2 produces the **correct** nested structure:
```go
type jsonStatsData struct {
    Total      int              `json:"total"`
    ByStatus   jsonByStatus     `json:"by_status"`
    Workflow   jsonWorkflow     `json:"workflow"`
    ByPriority []jsonPriorityEntry `json:"by_priority"`
}
```

V3 nests ready/blocked **inside** `by_status` instead of a separate `workflow` key:
```go
type jsonStats struct {
    Total      int              `json:"total"`
    ByStatus   jsonStatusCounts `json:"by_status"`
    ByPriority []jsonPriorityCount `json:"by_priority"`
}
type jsonStatusCounts struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    Done       int `json:"done"`
    Cancelled  int `json:"cancelled"`
    Ready      int `json:"ready"`   // <-- merged into by_status
    Blocked    int `json:"blocked"` // <-- merged into by_status
}
```
This deviates from the spec which calls for a separate `workflow` key.

**Blocked Count Computation:**

V1 computes blocked as `data.Blocked = data.Open - data.Ready` (arithmetic derivation).
V2 and V3 both execute a separate blocked query using the blocked condition.

V1's approach is mathematically equivalent only if the ready condition is exactly the complement of blocked among open tasks. If the blocked and ready conditions diverge in the future, V1 would silently produce incorrect results.

**Stats Handler Architecture:**

V3 uniquely extracts the query logic into a standalone function `queryStats(db *sql.DB) (*StatsData, error)` that is called from within the store.Query closure:
```go
err = store.Query(func(db *sql.DB) error {
    var queryErr error
    stats, queryErr = queryStats(db)
    return queryErr
})
```
V1 and V2 put all query logic inline inside the `store.Query` closure.

### Code Quality

**Error Handling:**

V1 wraps errors with context: `fmt.Errorf("querying status counts: %w", err)`.
V2 wraps similarly: `fmt.Errorf("failed to query status counts: %w", err)`.
V3 returns bare errors from `queryStats`: `return nil, err` for most queries. Less informative for debugging.

**Naming Conventions:**

V1: `cmdStats` (matches pattern `cmdXxx` for command handlers).
V2: `runStats` (matches pattern `runXxx`).
V3: `runStats` with extracted `queryStats` function.

V3's separation of `queryStats` from `runStats` is the cleanest separation of concerns. The query function is independently testable (though no unit tests of it exist).

**Type Safety:**

V2's `FormatStats(w io.Writer, stats interface{})` loses compile-time type checking. If someone passes the wrong type, it fails at runtime with `FormatStats: expected *StatsData, got X`. V1 and V3 both use concrete types (`StatsData` and `*StatsData` respectively), catching misuse at compile time.

**DRY Principle:**

V2 and V3 reuse the existing ready/blocked SQL fragments, adhering to DRY. V1 duplicates the ready query SQL inline, violating DRY and risking drift.

**Verbose Logging:**

V3 includes verbose logging (`a.WriteVerbose("store open %s", tickDir)`, `"lock acquire shared"`, `"cache freshness check"`, `"lock release"`). V1 wires verbose logging through `store.SetLogger(a.verbose.Log)`. V2 defers to `a.verbose` logger on the store.

### Test Quality

**V1 Test Functions (140 LOC, 8 subtests):**

1. `TestStatsCommand/counts tasks by status correctly` -- creates 4 tasks (open, in_progress, done, cancelled) via CLI commands, checks TOON output contains `stats{` and `4,`
2. `TestStatsCommand/counts ready and blocked tasks correctly` -- creates blocker + blocked task with `--blocked-by`, checks TOON output contains `stats{`
3. `TestStatsCommand/includes all 5 priority levels even at zero` -- single task at priority 2, checks `by_priority[5]` present
4. `TestStatsCommand/returns all zeros for empty project` -- empty project, checks `0,0,0,0,0,0,0` and `by_priority[5]`
5. `TestStatsCommand/formats stats in JSON format` -- checks `"total"` and `"by_priority"` present in JSON
6. `TestStatsCommand/formats stats in Pretty format` -- checks `Total:` label present
7. `TestStatsCommand/suppresses output with --quiet` -- uses `NewApp` directly, checks empty buffer
8. `TestStatsCommand/ready count matches ready query semantics` -- parent with child, checks JSON for `"ready": 1` and `"blocked": 1`

**V2 Test Functions (425 LOC, 8 subtests):**

1. `TestStats/it counts tasks by status correctly` -- 7 tasks via JSONL setup (2 open, 1 in_progress, 3 done, 1 cancelled), parses JSON, asserts exact counts for total=7, open=2, in_progress=1, done=3, cancelled=1
2. `TestStats/it counts ready and blocked tasks correctly` -- 7 tasks with blockers, parents, children, in_progress, done tasks; parses JSON; asserts ready=3, blocked=2
3. `TestStats/it includes all 5 priority levels even at zero` -- 2 tasks at priority 2, parses JSON, iterates all 5 entries checking expected map `{0:0, 1:0, 2:2, 3:0, 4:0}`
4. `TestStats/it returns all zeros for empty project` -- empty dir, parses JSON, checks total=0, all status=0, ready=0, blocked=0, all 5 priorities=0
5. `TestStats/it formats stats in TOON format` -- 2 tasks, checks exact header string, exact data line `2,1,0,1,0,1,0`, all 5 priority lines `0,0`, `1,1`, `2,1`, `3,0`, `4,0`
6. `TestStats/it formats stats in Pretty format with right-aligned numbers` -- 3 tasks, checks Total/Status/Workflow/Priority headers, P0-P4 labels with descriptors, right-alignment of "Total:" line containing ` 3`
7. `TestStats/it formats stats in JSON format with nested structure` -- 2 tasks, parses JSON, verifies all top-level keys (`total`, `by_status`, `workflow`, `by_priority`), all `by_status` subkeys, all `workflow` subkeys, `by_priority` structure with priority/count per entry, actual values
8. `TestStats/it suppresses output with --quiet` -- checks empty string output

**V3 Test Functions (388 LOC, 8 subtests):**

1. `TestStatsCommand/it counts tasks by status correctly` -- 7 tasks via setupTaskFull, parses JSON into typed struct, asserts total=7, open=2, in_progress=1, done=3, cancelled=1
2. `TestStatsCommand/it counts ready and blocked tasks correctly` -- 5 tasks with blockers/parent/child/done, parses JSON, asserts total=5, ready=2, blocked=2
3. `TestStatsCommand/it includes all 5 priority levels even at zero` -- 2 tasks at priorities 1 and 3, parses JSON, asserts 5 entries with map `{0:0, 1:1, 2:0, 3:1, 4:0}`
4. `TestStatsCommand/it returns all zeros for empty project` -- empty dir, parses JSON into fully typed struct, checks every field=0, all 5 priorities=0
5. `TestStatsCommand/it formats stats in TOON format` -- 2 tasks, checks exact header, exact data line, all 5 priority lines
6. `TestStatsCommand/it formats stats in Pretty format with right-aligned numbers` -- 2 tasks, checks all section headers (Total/Status/Workflow/Priority), all status labels (Open/In Progress/Done/Cancelled), workflow labels (Ready/Blocked), P0-P4 labels with descriptors
7. `TestStatsCommand/it formats stats in JSON format with nested structure` -- 1 task, checks string contains, parses JSON into typed struct, verifies total=1, open=1, ready=1, 5 priority entries, P0 count=1
8. `TestStatsCommand/it suppresses output with --quiet` -- checks empty string output

**Test Coverage Comparison:**

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Multiple statuses with exact counts | Weak (checks `4,` in string) | Strong (parses JSON, asserts each count) | Strong (parses JSON, typed struct) |
| Ready/blocked with blockers + parent/child | 2 tasks, loose string check | 7 tasks, exact count assertions (3 ready, 2 blocked) | 5 tasks, exact count assertions (2 ready, 2 blocked) |
| All priorities present at zero | Checks `by_priority[5]` string | Parses JSON, checks each of 5 entries | Parses JSON, checks each of 5 entries |
| Empty project all zeros | Checks TOON `0,0,0,0,0,0,0` string | Parses JSON, checks every field | Parses JSON, typed struct, every field |
| TOON exact format | Checks `stats{` present only | Checks exact header AND data line AND all 5 priority lines | Checks exact header AND data line AND all 5 priority lines |
| Pretty right-aligned | Checks `Total:` present only | Checks all headers, labels, P0-P4, right-alignment ` 3` | Checks all headers, labels, P0-P4 |
| JSON nested structure | Checks `"total"` and `"by_priority"` strings | Full key validation: top-level, nested `by_status`, `workflow`, `by_priority[].priority/count` | String checks + typed parse, fewer structural assertions |
| --quiet | Direct App instantiation | Direct App setup | Direct App setup |
| Phase 3 ready semantics | Parent+child test case | Complex multi-task scenario | Blocker+parent+child scenario |

**Test Gaps by Version:**

V1:
- Status count assertions are weak (checks `4,` substring, not individual counts)
- Ready/blocked test does not assert specific counts
- TOON format test only checks header presence, not data content
- Pretty format test only checks `Total:` label, not other groups
- JSON test does not verify `by_status`/`workflow` nesting (and indeed, V1's JSON is flat)
- No assertion of exact priority distribution across multiple levels

V2:
- Most thorough overall
- JSON test is the only one that validates the `workflow` key exists as a separate nested object
- Pretty test checks right-alignment pattern

V3:
- JSON test asserts fewer structural keys than V2
- JSON structure puts ready/blocked inside `by_status` rather than `workflow`, and tests validate this incorrect structure
- Pretty test checks more labels than V1 but does not check right-alignment pattern

**Test Setup Approaches:**

V1 uses **CLI-driven setup**: `initTickDir()`, `createTask()`, `runCmd()` to create tasks through the actual CLI, making tests true integration tests but slower and more brittle.

V2 uses **JSONL fixture injection**: `taskJSONL()` builds raw JSON lines, `setupTickDirWithContent()` writes directly to `tasks.jsonl`. Faster, more deterministic, allows precise control over task state.

V3 uses **structured fixture injection**: `setupTickDir()` + `setupTaskFull()` with explicit parameters for all task fields. Similar to V2 but uses a higher-level helper that accepts typed parameters.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 | 5 | 6 |
| Lines added | 247 | 533 | 529 |
| Impl LOC (stats.go) | 105 | 103 | 121 |
| Test LOC (stats_test.go) | 140 | 425 | 388 |
| Test functions | 8 | 8 | 8 |

## Verdict

**V2 is the best implementation.** Here is the specific evidence:

1. **Spec compliance (JSON structure):** V2 is the only version that correctly produces the JSON structure specified in the plan: `by_status` for status counts and a separate `workflow` key for ready/blocked. V1 uses a flat structure (all fields at top level), and V3 merges ready/blocked into `by_status` with no `workflow` key. The plan explicitly states: "JSON: Nested object with `total`, `by_status`, `workflow`, `by_priority`."

2. **Query reuse:** V2 reuses `readyWhere` and `blockedWhere` constants from `list.go`, satisfying the spec requirement to "reuse the ready query logic from Phase 3." V3 also achieves this via `ReadyCondition`/`BlockedCondition`. V1 duplicates the SQL inline, violating DRY and risking drift.

3. **Test quality:** V2 has the most thorough tests at 425 LOC. It validates exact counts for status (7 tasks, 4 statuses), comprehensive ready/blocked scenarios (7 tasks with various relationships yielding ready=3, blocked=2), TOON exact format including all 5 priority lines, Pretty format with right-alignment checks, and full JSON structural validation including the `workflow` nested key. V1's tests are notably weak with substring-only assertions.

4. **Type safety trade-off:** V2's `interface{}` formatter parameter is its weakest point -- it loses compile-time safety and requires runtime type assertions in every formatter. V1 and V3 are better here. However, this is a pre-existing interface design choice, not specific to this task's implementation.

5. **Architecture:** V3's extraction of `queryStats` as a standalone function is architecturally cleaner than V2's inline approach, but V3's JSON structure deviation from spec and its slightly less thorough test assertions make it second-best overall.

**Ranking: V2 > V3 > V1.** V2 wins on spec compliance (JSON nesting) and test thoroughness. V3 is second for its clean architecture and proper query reuse. V1 is last due to flat JSON, inline SQL duplication, and weak test assertions.
