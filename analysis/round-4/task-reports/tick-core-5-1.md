# Task tick-core-5-1: tick stats command

## Task Summary

Implement `tick stats` -- aggregate counts by status, priority, and workflow state (ready/blocked). Output via the Formatter interface in all three formats (TOON, Pretty, JSON).

**Required structures:**
- `StatsQuery` in the query/storage layer: query SQLite for counts grouped by status, priority, and workflow state
- Stats struct: `Total`, `ByStatus` (open/in_progress/done/cancelled), `Workflow` (ready/blocked), `ByPriority` (P0-P4 -- always 5 entries)
- Ready/blocked counts must reuse the ready query logic from Phase 3
- CLI handler calls `StatsQuery`, passes result to `Formatter.FormatStats()`
- TOON: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` + `by_priority[5]{priority,count}:`
- Pretty: Three groups (Total, Status breakdown, Workflow, Priority with P0-P4 labels), right-aligned numbers
- JSON: Nested object with `total`, `by_status`, `workflow`, `by_priority`
- `--quiet`: suppress output entirely

**Acceptance Criteria:**
1. StatsQuery returns correct counts by status, priority, workflow
2. All 5 priority levels always present (P0-P4)
3. Ready/blocked counts match Phase 3 query semantics
4. Empty project returns all zeros with full structure
5. TOON format matches spec example
6. Pretty format matches spec example with right-aligned numbers
7. JSON format nested with correct keys
8. --quiet suppresses all output

**Required Tests:**
- "it counts tasks by status correctly"
- "it counts ready and blocked tasks correctly"
- "it includes all 5 priority levels even at zero"
- "it returns all zeros for empty project"
- "it formats stats in TOON format"
- "it formats stats in Pretty format with right-aligned numbers"
- "it formats stats in JSON format with nested structure"
- "it suppresses output with --quiet"

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| StatsQuery returns correct counts by status, priority, workflow | PASS -- uses `StatsQuery`, `StatsPriorityQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery` as named SQL constants plus `queryStatusCounts()` and `queryPriorityCounts()` helpers | PASS -- inlines SQL strings directly inside `RunStats()` closure, queries same data |
| All 5 priority levels always present (P0-P4) | PASS -- `ByPriority [5]int` array initialized to zero; only indices 0-4 written to | PASS -- identical `ByPriority [5]int` array approach |
| Ready/blocked counts match Phase 3 query semantics | PASS -- `StatsReadyCountQuery` and `StatsBlockedCountQuery` reuse `readyWhereClause` and `blockedWhereClause` constants from `ready.go`/`blocked.go` via string concatenation | PASS -- ready query calls `ReadyWhereClause()` function; blocked computed arithmetically as `stats.Open - stats.Ready` |
| Empty project returns all zeros with full structure | PASS -- tested in "it returns all zeros for empty project" | PASS -- tested in "it returns all zeros for empty project" |
| TOON format matches spec example | PASS -- tested with header/value assertions | PASS -- tested with exact header + exact row value assertions |
| Pretty format matches spec example with right-aligned numbers | PASS -- tested with `strings.Contains` for group labels and P0 label | PASS -- tested with `strings.Contains` for labels AND exact right-aligned strings like `"Total:        2"`, `"P0 (critical):  0"`, `"P4 (backlog):   0"` |
| JSON format nested with correct keys | PASS -- tested for key existence | PASS -- tested for key existence AND value correctness |
| --quiet suppresses all output | PASS -- returns `nil` early when `ctx.Quiet` is true | PASS -- returns `nil` early when `fc.Quiet` is true |

## Implementation Comparison

### Approach

**V5** (`/private/tmp/tick-analysis-worktrees/v5/internal/cli/stats.go`, 140 lines):

V5 follows a decomposed pattern with four named SQL constants defined at package scope:

```go
const StatsQuery = `SELECT status, COUNT(*) as cnt FROM tasks GROUP BY status`
const StatsPriorityQuery = `SELECT priority, COUNT(*) as cnt FROM tasks GROUP BY priority`
const StatsReadyCountQuery = `SELECT COUNT(*) FROM tasks t WHERE ` + readyWhereClause
const StatsBlockedCountQuery = `SELECT COUNT(*) FROM tasks t WHERE ` + blockedWhereClause
```

The ready and blocked queries reuse `readyWhereClause` and `blockedWhereClause` -- Go `const` string fragments defined in `ready.go` (line 6) and `blocked.go` (line 6) respectively -- via compile-time string concatenation. This means the stats queries are guaranteed to be semantically identical to the `ReadyQuery` and `BlockedQuery` used by `tick ready` and `tick blocked`.

The `runStats` function (line 43) is a private method matching V5's command dispatch pattern (`map[string]func(*Context) error`). It delegates to two private helpers: `queryStatusCounts(db, &data)` (line 93) and `queryPriorityCounts(db, &data)` (line 123). Total is computed by accumulating counts from the status query rows (`data.Total += count` at line 106) rather than a separate `SELECT COUNT(*)`.

The function calls `ctx.Fmt.FormatStats(ctx.Stdout, &data)` (line 88), which matches V5's Formatter interface signature: `FormatStats(w io.Writer, data *StatsData) error`. The formatter writes directly to the writer.

**V6** (`/private/tmp/tick-analysis-worktrees/v6/internal/cli/stats.go`, 95 lines):

V6 uses an exported `RunStats(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error` function (line 11) matching V6's architecture where each command is an exported function called from `handleStats` in `app.go`. The SQL queries are inlined as string literals within the closure:

```go
db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.Total)
db.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
db.Query("SELECT priority, COUNT(*) FROM tasks GROUP BY priority")
```

For the ready count, V6 calls `ReadyWhereClause()` (a function in `query_helpers.go` line 63) which joins `ReadyConditions()` with `AND`:

```go
readyQuery := "\n\t\t\tSELECT COUNT(*) FROM tasks t\n\t\t\tWHERE " + ReadyWhereClause()
```

For blocked count, V6 uses arithmetic derivation:

```go
stats.Blocked = stats.Open - stats.Ready
```

This avoids a second SQL query entirely. It is correct because blocked = open AND NOT ready, so `blocked = open_count - ready_count`.

The function calls `fmt.Fprintln(stdout, fmtr.FormatStats(stats))` (line 93), which matches V6's Formatter interface signature: `FormatStats(stats Stats) string`. The formatter returns a string rather than writing directly.

### Code Quality

**Architecture Fit:**

V5 registers the command in `cli.go` line 110 via `"stats": runStats` in the `commands` map. The handler takes `*Context` and is unexported. This matches V5's established pattern.

V6 adds a `case "stats"` in the switch in `app.go` line 76 calling `a.handleStats(fc, fmtr)`, which is a thin method (lines 168-174 in `app.go`) that resolves the working directory and delegates to `RunStats()`. This matches V6's established pattern of exported `Run*` functions.

Both are idiomatic for their respective codebases.

**Query Strategy:**

V5 executes 4 SQL operations (status GROUP BY, priority GROUP BY, ready COUNT, blocked COUNT). Total is derived from summing status counts. No redundant query.

V6 executes 4 SQL operations (total COUNT, status GROUP BY, priority GROUP BY, ready COUNT). Blocked is derived arithmetically. The total COUNT is technically redundant since the same value could be derived by summing status counts (as V5 does), but the extra query is trivially cheap and more explicit.

**DRY for ready/blocked queries:**

V5 reuses `readyWhereClause` and `blockedWhereClause` as compile-time const concatenation. The stat queries (`StatsReadyCountQuery`, `StatsBlockedCountQuery`) are guaranteed to stay in sync with `ReadyQuery` and `BlockedQuery` because they share the same `const` fragment. This is the strongest form of DRY in Go for SQL fragments.

V6 reuses `ReadyWhereClause()` -- a function that returns a runtime string by joining `ReadyConditions()`. This is also DRY but uses a function call instead of compile-time const concatenation. For the blocked count, V6 sidesteps the need for a blocked SQL query entirely by computing `Open - Ready`. This is mathematically equivalent (and arguably more elegant), but it does NOT reuse the `BlockedConditions()` function from `query_helpers.go`, meaning if the definition of "blocked" ever diverged from "open AND NOT ready", the stats blocked count would silently become incorrect. The task plan says "Ready/blocked reuse Phase 3 query logic" -- V6's arithmetic derivation does not literally reuse the query, though it produces the same result under current semantics.

**Decomposition:**

V5 extracts `queryStatusCounts` and `queryPriorityCounts` as separate functions. This is cleaner for testing individual query logic in isolation, though no unit tests exercise them directly.

V6 keeps everything inline in a single function body. At 95 lines, `RunStats` is still readable, but the inline SQL strings and all four queries in one closure make the function denser.

**Error Handling:**

Both versions wrap all errors with `fmt.Errorf("%w", err)` as required by the skill file.

V5 error messages use the pattern `"counting ready tasks: %w"`, `"querying status counts: %w"`, `"scanning status count: %w"`.

V6 uses `"failed to query total count: %w"`, `"failed to query status counts: %w"`, `"failed to scan status count: %w"`, `"failed to iterate status counts: %w"`. V6 also explicitly checks `rows.Err()` after iteration loops (lines 54, 74), which V5 also does (lines 118, 139). Both are correct.

**Exported Types and Documentation:**

V5 type: `StatsData` (defined in `toon_formatter.go` line 21), contains `Total`, `Open`, `InProgress`, `Done`, `Cancelled`, `Ready`, `Blocked`, `ByPriority [5]int`. Documented with comment "StatsData holds all statistics data for formatting."

V6 type: `Stats` (defined in `format.go` line 103), contains the same fields. Documented with implicit naming convention.

Both satisfy the skill requirement to "Document all exported functions, types, and packages."

### Test Quality

Both versions have a single `TestStats` function containing 8 subtests matching the 8 required test names.

**Test Function Inventory:**

| Test Name | V5 | V6 |
|-----------|-----|-----|
| `it counts tasks by status correctly` | Lines 14-58 | Lines 32-71 |
| `it counts ready and blocked tasks correctly` | Lines 60-103 | Lines 74-110 |
| `it includes all 5 priority levels even at zero` | Lines 106-149 | Lines 113-145 |
| `it returns all zeros for empty project` | Lines 151-203 | Lines 148-190 |
| `it formats stats in TOON format` | Lines 206-223 | Lines 192-237 |
| `it formats stats in Pretty format with right-aligned numbers` | Lines 226-253 | Lines 240-275 |
| `it formats stats in JSON format with nested structure` | Lines 255-284 | Lines 278-326 |
| `it suppresses output with --quiet` | Lines 286-300 | Lines 328-342 |

**Test Harness:**

V5 tests use the global `Run()` function directly:
```go
code := Run([]string{"tick", "--toon", "stats"}, dir, &stdout, &stderr, false)
```

V6 defines a local `runStats` helper (lines 14-26) that constructs an `App` struct with injected `Getwd` and `IsTTY`:
```go
func runStats(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
    app := &App{Stdout: &stdoutBuf, Stderr: &stderrBuf, Getwd: func() (string, error) { return dir, nil }, IsTTY: false}
    fullArgs := append([]string{"tick", "stats"}, args...)
    code := app.Run(fullArgs)
    return stdoutBuf.String(), stderrBuf.String(), code
}
```

V6's approach is slightly more encapsulated and reduces boilerplate in each subtest.

**Test Setup:**

V5 constructs tasks using `task.NewTask()` and mutates fields:
```go
open1 := task.NewTask("tick-aaaaaa", "Open one")
ip := task.NewTask("tick-cccccc", "In progress")
ip.Status = task.StatusInProgress
```

V6 uses struct literals:
```go
{ID: "tick-aaa111", Title: "Open task 1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}
```

V6's approach is more explicit about all fields (including `Created`, `Updated`, `Priority`), making the test data more deterministic and visible. V5 relies on `NewTask()` defaults for fields like `Created` and `Priority`, which may or may not matter.

**Assertion Depth:**

*"it counts tasks by status correctly"*:
- V5: Creates 5 tasks (2 open, 1 IP, 1 done, 1 cancelled). Uses TOON format, checks header string, checks data line prefix `"5,2,1,1,1,"`. Does NOT individually verify each status count via structured parsing.
- V6: Creates 7 tasks (2 open, 1 IP, 3 done, 1 cancelled). Uses JSON format, parses JSON, asserts `total=7`, `open=2`, `in_progress=1`, `done=3`, `cancelled=1` individually. More thorough.

*"it counts ready and blocked tasks correctly"*:
- V5: 5 tasks (ready1, ready2 unblocked; blocked1 has dep; parent1 has open child; child1 is ready leaf). Expects ready=3, blocked=2. Parses JSON.
- V6: 6 tasks (adds an in_progress blocker and a done task to the scenario). Expects ready=2, blocked=2. Parses JSON. The scenarios differ: V5 expects child1 to be ready (3 total), V6 expects only tick-aaa111 and tick-ccc222 to be ready (2 total). This is because V6 has tick-aaa222 blocked by tick-bbb111 (which is in_progress), while V5 has blocked1 blocked by ready1 (which is open). Both correctly test the ready semantics.

*"it includes all 5 priority levels even at zero"*:
- V5: 1 task at priority 2. Checks all 5 entries exist, each has correct priority index and count.
- V6: 2 tasks at priority 2. Checks all 5 entries, expected counts `[0,0,2,0,0]`. V6 is slightly more thorough with the `expectedCounts` slice pattern.

*"it returns all zeros for empty project"*:
- Both are nearly identical: parse JSON, check total=0, all status counts=0, workflow ready/blocked=0, all 5 priority counts=0.

*"it formats stats in TOON format"*:
- V5: 1 task. Checks `strings.Contains` for header strings only.
- V6: 2 tasks (1 open, 1 done). Splits output into sections, verifies exact header strings, exact data row values (`"  2,1,0,1,0,1,0"`), exact priority rows (`"  0,0"`, `"  1,1"`, `"  2,1"`, etc.). Much more thorough -- verifies actual data values, not just structure.

*"it formats stats in Pretty format with right-aligned numbers"*:
- V5: Checks `strings.Contains` for "Total:", "Status:", "Workflow:", "Priority:", "P0 (critical):". Does NOT verify right-alignment.
- V6: Same `strings.Contains` checks plus exact right-aligned substring checks: `"Total:        2"`, `"P0 (critical):  0"`, `"P4 (backlog):   0"`. Actively tests right-alignment.

*"it formats stats in JSON format with nested structure"*:
- V5: Checks key existence only (`"total"`, `"by_status"`, `"workflow"`, `"by_priority"`).
- V6: Checks key existence AND verifies values: `total=2`, `by_status.open=1`, `by_status.done=1`, `workflow.ready=1`. More thorough.

*"it suppresses output with --quiet"*:
- Both identical in logic: run with --quiet, assert empty stdout.

**Test Gaps:**

V5 TOON format test only checks headers, not data values. If the formatter emitted wrong numbers but correct headers, the test would pass.

V5 Pretty format test does not verify right-alignment at all. If numbers were left-aligned, the test would pass.

V5 JSON format test only checks key existence, not correctness of values.

V6 has no significant test gaps relative to the spec requirements.

### Skill Compliance

| Skill Constraint | V5 | V6 |
|-----------------|-----|-----|
| Use gofmt | PASS (assumed -- standard Go project tooling) | PASS |
| Handle all errors explicitly | PASS -- all `err` values checked | PASS -- all `err` values checked |
| Write table-driven tests with subtests | PARTIAL -- uses subtests (`t.Run`) but not table-driven format. Each test is a separate anonymous function, not a loop over test cases. | PARTIAL -- same pattern. Uses subtests, not table-driven. |
| Document all exported functions, types, and packages | PASS -- `StatsQuery`, `StatsPriorityQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery`, `queryStatusCounts`, `queryPriorityCounts` all documented. However, `runStats` is unexported so not required. | PASS -- `RunStats` has a doc comment. |
| Propagate errors with fmt.Errorf("%w", err) | PASS | PASS |
| No panic for normal error handling | PASS | PASS |
| No ignored errors | PASS | PASS |

Note: Neither version uses table-driven test style in the strict sense (iterating `[]struct{name, input, want}`). However, the subtests cover distinct scenarios, which is a common and accepted Go testing pattern. The skill file says "Write table-driven tests with subtests" -- both versions use subtests but not table-driven iteration. This is a minor compliance gap for both.

### Spec-vs-Convention Conflicts

**Blocked Count: SQL vs Arithmetic**

The spec says "Ready/blocked counts reuse the ready query logic from Phase 3." V5 literally reuses both `readyWhereClause` and `blockedWhereClause` SQL fragments, satisfying the spec verbatim. V6 reuses `ReadyWhereClause()` for the ready count but computes blocked as `Open - Ready` arithmetically.

V6's arithmetic approach is arguably better engineering: it avoids a redundant SQL query, is simpler, and is mathematically equivalent. However, it does NOT reuse Phase 3 blocked query logic -- it derives the same result differently. If the semantics of "blocked" ever changed to include non-open statuses, V6's arithmetic would break silently. V5's approach is more defensive against future spec changes.

**Total Count: Derived vs Queried**

V5 derives total by summing status group counts. V6 queries `SELECT COUNT(*) FROM tasks` separately. Both are correct. V6's approach is more explicit but adds a trivially cheap extra query.

**Formatter Interface Signature**

V5's `FormatStats(w io.Writer, data *StatsData) error` writes directly to the writer and returns an error. V6's `FormatStats(stats Stats) string` returns a string, leaving the caller to write it. This is a pre-existing design difference between the codebases, not a decision made by this task.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed (Go only) | 3 (cli.go +1 line, stats.go new, stats_test.go new) | 3 (app.go +11 lines, stats.go new, stats_test.go new) |
| stats.go lines | 140 | 95 |
| stats_test.go lines | 301 | 343 |
| Total Go insertions | 466 | 469 |
| Named SQL constants | 4 (StatsQuery, StatsPriorityQuery, StatsReadyCountQuery, StatsBlockedCountQuery) | 0 (all inline) |
| Helper functions | 2 (queryStatusCounts, queryPriorityCounts) | 0 (all inline) |
| SQL queries executed | 4 (status, priority, ready, blocked) | 4 (total, status, priority, ready) |
| Test subtests | 8 | 8 |

## Verdict

Both versions fully satisfy all 8 acceptance criteria. The core logic is functionally equivalent -- both query status counts, priority counts, ready counts, and derive blocked counts, passing results through the Formatter interface.

**V6 is the stronger implementation overall**, for two reasons:

1. **Test thoroughness**: V6 tests are substantially more rigorous. The TOON format test verifies exact data row values (not just headers). The Pretty format test verifies actual right-alignment strings. The JSON test verifies values, not just key existence. The status counting test uses a larger, more varied dataset (7 tasks vs 5). V5 has notable test gaps: its TOON test would pass even if data values were wrong, its Pretty test does not actually verify right-alignment, and its JSON test does not verify values.

2. **Conciseness**: V6's `stats.go` is 95 lines vs V5's 140. The inline SQL and arithmetic blocked derivation reduce code without sacrificing clarity.

**V5 has one advantage**: its query reuse strategy is more robust. By using compile-time `const` concatenation of `readyWhereClause` and `blockedWhereClause`, V5 guarantees that stats ready/blocked semantics can never diverge from `tick ready`/`tick blocked`. V6's arithmetic derivation (`Blocked = Open - Ready`) is correct under current semantics but introduces a coupling assumption that is invisible at the code level. V5 also decomposes the query logic into named helper functions (`queryStatusCounts`, `queryPriorityCounts`), which is more maintainable if the query logic grows.

Neither version uses true table-driven tests, which is a minor skill compliance gap for both.
