# Task 5-1: tick stats command

## Task Plan Summary

Implement `tick stats` -- aggregate counts by status, priority, and workflow state (ready/blocked). The command queries SQLite for counts grouped by status, priority, and workflow state. The `StatsData` struct holds `Total`, `ByStatus` (open/in_progress/done/cancelled), `Workflow` (ready/blocked), and `ByPriority` (P0-P4, always 5 entries). Ready/blocked counts must reuse Phase 3 ready query logic. Output via the Formatter interface in all three formats: TOON (two tabular sections), Pretty (three groups with right-aligned numbers), and JSON (nested object). `--quiet` suppresses output entirely. Eight named tests are specified, plus five edge cases including empty project, all-same-status, zero-count priority levels, and ready/blocked semantic correctness.

## V4 Implementation

### Architecture & Design

V4 follows the method-on-struct pattern used throughout its codebase. The command handler is `(a *App) runStats(args []string) error` in `/private/tmp/tick-analysis-worktrees/v4/internal/cli/stats.go`. It wires into the dispatcher via a `case "stats":` in the `switch` block in `cli.go`.

The implementation is cleanly separated into three query helper functions:
- `queryStatusCounts(db, *StatsData)` -- inline SQL `"SELECT status, COUNT(*) FROM tasks GROUP BY status"`
- `queryPriorityCounts(db, *StatsData)` -- inline SQL `"SELECT priority, COUNT(*) FROM tasks GROUP BY priority"`
- `queryWorkflowCounts(db, *StatsData)` -- builds the ready count query by concatenating `readyConditionsFor("t")` from `ready.go`, then derives `Blocked = Open - readyCount`

The `StatsData` struct is defined in `format.go` (lines 44-57) alongside the `Formatter` interface, with the signature `FormatStats(w io.Writer, stats StatsData) error` -- passing `StatsData` by value.

Key design decisions:
1. **Ready/blocked reuse**: V4 calls `readyConditionsFor("t")` (a function returning a SQL fragment string) to build the ready count query. Blocked is computed arithmetically: `Blocked = Open - readyCount`. This is elegant -- only one query is needed for the workflow section, and the blocked count is derived rather than independently queried.
2. **SQL queries are inline strings** in the query functions, not exported constants.
3. **Value semantics for StatsData**: The `Formatter.FormatStats` method takes `StatsData` by value, not pointer. This is consistent with V4's other formatter methods.

### Code Quality

The code is clean and idiomatic Go. All errors are wrapped with `fmt.Errorf("%w", err)` using descriptive prefixes like `"failed to query status counts: %w"`. The `rows.Err()` check after iteration is correctly included in both `queryStatusCounts` and `queryPriorityCounts`. The `defer rows.Close()` is placed immediately after successful query execution.

The function at 119 lines is concise. The three helper functions each have clear single responsibilities. Comments are present on all exported and unexported functions.

One style note: V4's error messages use the "failed to X" prefix pattern (e.g., `"failed to query status counts: %w"`), which is a common Go convention.

The `queryWorkflowCounts` function is particularly well-designed: by deriving `Blocked = Open - readyCount`, it avoids duplicating the blocked query logic and guarantees that `Ready + Blocked = Open` is always a tautology. This is mathematically sound and eliminates any possibility of ready/blocked counts being inconsistent.

### Test Coverage

V4 provides 8 test functions matching all 8 specified test names exactly:
1. `TestStats_CountsTasksByStatusCorrectly` (7 tasks across all 4 statuses)
2. `TestStats_CountsReadyAndBlockedCorrectly` (5 tasks: 2 ready leaves, 1 blocked-by-dep, 1 parent with open child, 1 ready child)
3. `TestStats_IncludesAll5PriorityLevelsEvenAtZero` (2 tasks at priority 2 only)
4. `TestStats_ReturnsAllZerosForEmptyProject` (thorough zero-checks on every field)
5. `TestStats_FormatsToonFormat` (checks headers, data row values, priority rows)
6. `TestStats_FormatsPrettyFormatWithRightAlignedNumbers` (checks all section labels and P0-P4 labels)
7. `TestStats_FormatsJSONFormatWithNestedStructure` (deserializes and verifies nested keys + values)
8. `TestStats_SuppressesOutputWithQuiet` (verifies empty output)

Total: 393 lines of tests.

**Test quality details:**

- Status test uses 7 diverse tasks (2 open, 1 in_progress, 3 done, 1 cancelled) with explicit assertions on each count.
- Ready/blocked test is thorough: tests a blocker that is itself ready (tick-rdy111 blocks tick-blk111 but has no blockers itself), a parent with an open child, and a child that is ready despite being a child. Comments explain expected counts clearly.
- TOON test verifies exact data values: `"  2,1,0,1,0,1,0"` and checks all 5 priority rows with `"  0,0\n"`, `"  1,1\n"`, `"  2,1\n"`.
- JSON test uses a typed `jsonStats` struct for deserialization, providing compile-time type safety on the expected JSON structure.
- Empty project test exhaustively checks every field including all 5 priority entries.

**Weakness:** Each test function wraps a single `t.Run` subtest, which is slightly redundant (the outer function name already describes the case). This is not idiomatic table-driven testing, but since each test scenario requires different setup, individual functions are acceptable.

### Spec Compliance

Fully compliant:
- StatsQuery returns correct counts by status, priority, workflow -- verified by tests.
- All 5 priority levels always present (P0-P4) -- `[5]int` array guarantees this + test verifies.
- Ready/blocked counts reuse Phase 3 query semantics via `readyConditionsFor("t")`.
- Empty project returns all zeros with full structure -- test verifies comprehensively.
- TOON format matches spec: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` and `by_priority[5]{priority,count}:`.
- Pretty format has three visual groups (Total, Status, Workflow, Priority) with P0-P4 labels.
- JSON format produces nested object with `total`, `by_status`, `workflow`, `by_priority` keys.
- `--quiet` suppresses all output (early return before any DB work).

### golang-pro Skill Compliance

| Rule | Status | Detail |
|------|--------|--------|
| Handle all errors explicitly | PASS | All error returns checked; error wrapping with `%w` |
| Write table-driven tests with subtests | PARTIAL | Uses subtests but each function has only one `t.Run`, not table-driven |
| Document all exported functions/types | PASS | `StatsData` and `Formatter` interface documented in format.go |
| Propagate errors with fmt.Errorf("%w") | PASS | All error wrapping uses `%w` |
| No ignored errors | PASS | No `_` assignments |
| No panic for normal error handling | PASS | No panics |

## V5 Implementation

### Architecture & Design

V5 uses a function-based command dispatch pattern. The handler is `runStats(ctx *Context) error` in `/private/tmp/tick-analysis-worktrees/v5/internal/cli/stats.go`. It wires into the dispatcher via the `commands` map: `"stats": runStats`. This is V5's standard pattern where `*Context` replaces V4's `*App`.

The implementation has:
- Four exported SQL constants: `StatsQuery`, `StatsPriorityQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery`
- Two helper functions: `queryStatusCounts`, `queryPriorityCounts`
- Ready/blocked counting done inline in `runStats` using `db.QueryRow` with the exported const queries

Key design decisions:
1. **Ready/blocked reuse via shared const clauses**: V5 defines `readyWhereClause` in `ready.go` and `blockedWhereClause` in `blocked.go` as `const` strings. The stats queries compose these via Go const concatenation: `StatsReadyCountQuery = "SELECT COUNT(*) FROM tasks t WHERE " + readyWhereClause`. This leverages compile-time string concatenation.
2. **Separate ready and blocked queries**: Unlike V4's arithmetic derivation, V5 runs two independent SQL queries -- one for ready count, one for blocked count. Both are exported constants.
3. **Pointer semantics for StatsData**: The `Formatter.FormatStats` method takes `*StatsData` (pointer), consistent with V5's other formatter methods.
4. **Exported SQL constants**: All four query constants are exported (`StatsQuery`, `StatsPriorityQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery`). This could enable testing the SQL independently, though no such tests exist.
5. **Store creation via engine**: V5 uses `engine.NewStore(tickDir, ctx.storeOpts()...)` with functional options, more composable than V4's `a.openStore(tickDir)` pattern.

### Code Quality

Clean, idiomatic code. Errors are wrapped with `fmt.Errorf` using the "verb-ing noun" style: `"querying status counts: %w"`, `"scanning status count: %w"`, `"counting ready tasks: %w"`. This follows the Go convention of lower-case error messages without "failed to" prefix. V5's error wrapping style is slightly more idiomatic per Go conventions (error strings should not be capitalized and should not start with "failed to").

At 140 lines (21 more than V4), the extra length comes from the four exported const declarations. The query constants are well-documented with doc comments explaining what each query returns.

The `StatsData` struct is defined in `toon_formatter.go` (line 21) rather than `format.go`. This is a somewhat surprising location -- the struct is format-agnostic data but lives in a format-specific file. V4's placement in `format.go` alongside the `Formatter` interface is more logical.

### Test Coverage

V5 provides all 8 required tests as subtests under a single `TestStats` function:
1. `"it counts tasks by status correctly"`
2. `"it counts ready and blocked tasks correctly"`
3. `"it includes all 5 priority levels even at zero"`
4. `"it returns all zeros for empty project"`
5. `"it formats stats in TOON format"`
6. `"it formats stats in Pretty format with right-aligned numbers"`
7. `"it formats stats in JSON format with nested structure"`
8. `"it suppresses output with --quiet"`

Total: 301 lines of tests (92 fewer than V4).

**Test quality details:**

- **Status test** uses `task.NewTask()` constructor (cleaner task creation) and validates via TOON format string parsing rather than JSON deserialization. This tests TOON output correctness simultaneously with status counting, but the assertions are less precise -- it uses `strings.HasPrefix(statsLine, "5,2,1,1,1,")` which only checks partial values (doesn't verify ready/blocked counts explicitly).
- **Ready/blocked test** creates the same 5-task topology as V4 (2 ready leaves, 1 blocked-by-dep, 1 parent, 1 child). Uses `json.Unmarshal` into `map[string]interface{}` instead of a typed struct, requiring `float64` type assertions. This is less type-safe but still works.
- **Priority test** also uses `map[string]interface{}` for JSON parsing, with explicit `float64` comparisons.
- **Empty project test** comprehensively checks all fields via untyped JSON parsing.
- **TOON format test** only checks headers are present, not data values. Much less thorough than V4's TOON test.
- **Pretty format test** checks section labels and P0 label only (not P1-P4). Less thorough than V4.
- **JSON format test** only checks key existence, not values. Less thorough than V4.
- **Quiet test** is equivalent.

**Weakness:** V5's format-specific tests (TOON, Pretty, JSON) are significantly less thorough than V4's. The TOON test doesn't verify actual data values. The Pretty test only checks for P0 but not P1-P4. The JSON test checks key existence only. V4 verifies actual values, all priority labels, and data row content.

**V5 advantage:** Uses `task.NewTask()` for task construction, which is cleaner and uses the factory's defaults. V4 constructs tasks with struct literals, requiring explicit `Created`, `Updated`, `Priority` fields.

### Spec Compliance

Fully compliant:
- StatsQuery returns correct counts by status, priority, workflow.
- All 5 priority levels always present via `[5]int` array.
- Ready/blocked counts reuse Phase 3 query semantics via `readyWhereClause`/`blockedWhereClause` shared constants.
- Empty project returns all zeros with full structure.
- TOON format matches spec (verified by reading FormatStats implementation).
- Pretty format has three visual groups with P0-P4 labels.
- JSON format produces nested object with correct keys.
- `--quiet` suppresses all output.

### golang-pro Skill Compliance

| Rule | Status | Detail |
|------|--------|--------|
| Handle all errors explicitly | PASS | All error returns checked; wrapping with `%w` |
| Write table-driven tests with subtests | PARTIAL | Uses subtests under single parent, not table-driven |
| Document all exported functions/types | PASS | All exported consts and functions documented |
| Propagate errors with fmt.Errorf("%w") | PASS | Consistent error wrapping |
| No ignored errors | PASS | No `_` assignments |
| No panic for normal error handling | PASS | No panics |

## Comparative Analysis

### Where V4 is Better

1. **Blocked count derivation is more robust**: V4 computes `Blocked = Open - readyCount`. This is mathematically guaranteed to produce `Ready + Blocked = Open`. V5 runs two independent queries (`StatsReadyCountQuery` and `StatsBlockedCountQuery`), which means if the ready and blocked WHERE clauses ever diverge, the counts could become inconsistent (Ready + Blocked != Open). V4's approach is a safer invariant.

2. **Significantly more thorough format tests**: V4's TOON test verifies exact data row values (`"  2,1,0,1,0,1,0"`), checks all 5 priority rows with specific values (`"  0,0\n"`, `"  1,1\n"`, `"  2,1\n"`). V4's Pretty test checks all P0-P4 labels explicitly. V4's JSON test uses a typed `jsonStats` struct for deserialization and verifies actual numeric values plus raw JSON key names. V5's format tests only check headers/labels exist, not actual values. This is a material difference in test coverage.

3. **StatsData placement**: V4 places `StatsData` in `format.go` alongside the `Formatter` interface definition. This is the logical home -- `StatsData` is a data transfer object consumed by formatters, so co-locating it with the interface that consumes it makes sense. V5 puts it in `toon_formatter.go`, which is the wrong location for a format-agnostic struct.

4. **Type-safe JSON test deserialization**: V4 deserializes JSON into `jsonStats` struct, providing compile-time type safety. V5 uses `map[string]interface{}` with `float64` assertions, which is fragile and fails to catch structural changes at compile time.

5. **More comprehensive status test**: V4's status counting test uses 7 tasks (2 open, 1 in_progress, 3 done, 1 cancelled) with diverse priorities, while V5 uses 5 tasks (2 open, 1 in_progress, 1 done, 1 cancelled). V4's test exercises multiple tasks per status which better validates the GROUP BY aggregation.

### Where V5 is Better

1. **Query reuse via const composition**: V5 defines `readyWhereClause` and `blockedWhereClause` as `const` strings shared across ready.go, blocked.go, and stats.go. The shared clauses are composed at compile time. V4 uses a `readyConditionsFor(alias)` function for the ready side, but the blocked query in `blocked.go` uses `NOT IN (SELECT ... readyConditionsFor("t"))` -- a subquery inversion. V5's explicit `blockedWhereClause` with direct EXISTS/OR conditions is more readable and potentially more performant (avoids NOT IN subquery).

2. **Exported SQL constants**: V5 exports its query constants (`StatsQuery`, `StatsPriorityQuery`, etc.), making them available for independent testing or inspection. V4 uses inline strings, which are less discoverable.

3. **Cleaner test task construction**: V5 uses `task.NewTask("tick-aaaaaa", "Open one")` which fills in default values (status=open, priority=2, timestamps=now). V4 uses struct literals requiring explicit `Created`, `Updated`, `Priority` fields on every task, which is more verbose and error-prone.

4. **Function-based dispatch pattern**: V5's `commands` map with `"stats": runStats` is more extensible and eliminates the growing switch statement in V4's `cli.go`. Adding new commands in V5 is a one-line map entry rather than a new case block.

5. **Consistent pointer semantics**: V5's `FormatStats(w io.Writer, data *StatsData)` uses pointer semantics consistently with all other formatter methods. V4's `FormatStats(w io.Writer, stats StatsData)` passes by value, which is inconsistent if other V4 methods use pointers (though in V4, value semantics appear to be the norm for formatter data).

6. **More idiomatic error messages**: V5 uses `"querying status counts: %w"` (lowercase, gerund form) versus V4's `"failed to query status counts: %w"`. The Go convention per the Go blog and standard library is lowercase error strings without "failed to" prefixes.

7. **Better Pretty formatter alignment logic**: V5 pre-computes format strings once (`summaryFmt`, `indentFmt`, `priFmt`) and applies them cleanly. V4 uses `fmt.Fprintf(w, "  %-*s  %*d\n", maxStatusLabel, label, maxNumWidth, value)` with computed widths, which works but is harder to read with the `%-*s` and `%*d` width specifiers repeated everywhere.

### Differences That Are Neutral

1. **StatsData value vs pointer**: V4 passes `StatsData` by value, V5 by pointer. For a small struct (~7 ints + [5]int = 12 ints), value semantics are fine. Pointer avoids copying but the difference is negligible. Both are acceptable.

2. **Single TestStats parent vs individual test functions**: V5 groups all 8 tests under `TestStats(t *testing.T)` with subtests. V4 uses 8 separate top-level test functions. Both approaches are valid. V5's grouping makes it slightly easier to run all stats tests with `-run TestStats`. V4's individual functions give each test its own name at the top level.

3. **Quiet handling**: Both implementations check `Quiet` at the top of `runStats` and return `nil` immediately. Identical behavior.

4. **File length**: V4 stats.go is 119 lines, V5 is 140 lines. The difference is primarily the 4 exported const declarations in V5. Neither is problematic.

## Verdict

**Winner: V4**

V4 wins primarily on **test quality** and **architectural soundness of the blocked count derivation**.

The most significant advantage is V4's blocked count computation: `Blocked = Open - readyCount`. This is a mathematical invariant that guarantees `Ready + Blocked = Open` regardless of query implementation details. V5 runs two independent queries, which works correctly today but creates a maintenance risk -- if `readyWhereClause` and `blockedWhereClause` ever become inconsistent (e.g., a developer updates one but not the other), the ready and blocked counts could fail to sum to the open count. V4's approach eliminates this entire class of bugs by construction.

The second decisive factor is test thoroughness. V4's format-specific tests verify actual output values (exact TOON data rows, all 5 P0-P4 labels, JSON values and key names), while V5's format tests only check that headers/labels exist. If a bug introduced wrong values but preserved the output structure, V5's tests would pass while V4's would catch it. For the status counting test, V4 uses 7 tasks with varied distributions while V5 uses 5. V4's typed JSON deserialization via `jsonStats` struct catches structural changes at compile time.

V5 has genuine advantages in query composition (const-based clause sharing), task construction in tests (NewTask factory), and dispatch architecture (map vs switch). However, these are design conveniences that don't affect correctness. V4's advantages directly impact defect detection capability (tests) and correctness guarantees (blocked derivation).

The StatsData placement issue (V5 putting it in `toon_formatter.go` instead of `format.go`) is a minor organizational flaw in V5 that further tips the balance toward V4.
