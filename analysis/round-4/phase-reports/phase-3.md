# Phase 3: Hierarchy & Dependencies

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 3-1 Dependency validation | V5 | Moderate | ID normalization via `NormalizeID` throughout (5 calls vs 0) -- correctness gap in V6; single-pass map construction; stronger batch test ordering |
| 3-2 dep add & dep rm | V6 | Slight | Shared `parseDepArgs` helper eliminates arg-parsing duplication; faster timestamp tests (no `time.Sleep`); extra test coverage (no-subcommand path, output normalization checks) |
| 3-3 Ready query & tick ready | V5 | Strong | Exported `ReadyQuery` satisfies reusability AC; reuses `listRow` and `printListTable` from list.go; evolves into 3-line delegator to `runList` |
| 3-4 Blocked query & cancel-unblocks | V5 | Moderate | DRYer via shared `printListTable`; V6 duplicates `fmt.Fprintf` formatting and `listRow` type across ready/blocked/list -- maintenance liability |
| 3-5 List filter flags | V6 | Moderate | Single `buildListQuery` with composable conditions vs V5's 3 parallel builder functions; type-safe status validation via `task.Status*` constants; cleaner SQL (flat vs nested subqueries) |
| 3-6 Parent scoping | V6 | Slight | Unified query builder with flat conditions; Go-idiomatic lowercase error strings; deterministic test timestamps via `time.Date`; struct literal task construction |

## Cross-Task Architecture Analysis

### The Divergent Evolution of Query Composition

The most consequential architectural difference in this phase -- visible only when tasks 3-3 through 3-6 are examined together -- is how each version evolves its query reuse strategy across four tasks.

**V5's trajectory: SQL constants + subquery wrapping**

V5 defines `ReadyQuery` and `BlockedQuery` as exported `const` strings containing complete `SELECT ... FROM ... WHERE ... ORDER BY` statements (`ready.go` lines 22-27, `blocked.go` lines 25-30). It also extracts the WHERE fragments as `readyWhereClause` and `blockedWhereClause` for use by the stats command. When task 3-5 (list filters) arrives, V5 wraps these complete queries as subqueries:

```go
// V5 list.go line 122 -- wrapping pattern
q := `SELECT id, status, priority, title FROM (` + ReadyQuery + `) AS ready WHERE 1=1`
```

This produces nested SQL. V5 then needs three separate builder functions (`buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildSimpleFilterQuery`) plus a shared `buildWrappedFilterQuery` to reduce duplication among the first two. By task 3-6, `appendDescendantFilter` is added as yet another helper, called from `buildWrappedFilterQuery` and `buildSimpleFilterQuery`.

The result is a 5-function query-building apparatus (lines 98-175 of `list.go`) where filter logic is applied in two different shapes: outer-SELECT wrapping for ready/blocked, and direct WHERE clauses for the simple case.

**V6's trajectory: Composable condition slices**

V6 creates `query_helpers.go` (65 lines) with `ReadyConditions()` and `BlockedConditions()` returning `[]string` slices of WHERE clause fragments. `buildListQuery` (list.go lines 199-239) is a single 41-line function that assembles all conditions -- ready, blocked, status, priority, descendant -- as peers in a flat `conditions` slice, joined with `AND`:

```go
// V6 list.go lines 203-204
if f.Ready {
    conditions = append(conditions, ReadyConditions()...)
}
```

This produces flat SQL: `SELECT ... FROM tasks t WHERE cond1 AND cond2 AND ...`. No nesting, no aliasing, no parallel builder functions.

**Assessment:** V6's approach is architecturally superior for composition. Adding a new filter in V5 requires modifying 3+ builder functions; in V6 it requires adding one `if` block in `buildListQuery`. The flat SQL is also easier to debug by pasting into a SQLite shell. However, V5's approach has one advantage: the complete query constants (`ReadyQuery`, `BlockedQuery`) are directly copy-pasteable to a SQL tool without assembly, and they explicitly document the full query as a readable unit.

### The Delegation vs. Duplication Divide

Both versions converge on the same final architecture for ready/blocked commands: delegating to the list command with a prepended flag. But V6 takes a detour through duplication that leaves structural debt.

**V5** (worktree final state): `ready.go` and `blocked.go` each contain SQL constants plus a 3-line function that prepends `--ready`/`--blocked` to args and calls `runList`. The SQL is colocated with the command name, and `runList` handles all query building, execution, and formatting.

**V6** (worktree final state): `ready.go` and `blocked.go` are each 1 line (`package cli`). The delegation happens in `app.go` (`handleReady`, `handleBlocked`), which prepend the flag and call `RunList`. The SQL conditions live in `query_helpers.go`.

However, at the commit points for tasks 3-3 and 3-4, V6 had standalone `RunReady` and `RunBlocked` functions that duplicated the entire query-execute-format pipeline, including a local `listRow` type definition and inline `fmt.Fprintf` formatting. This duplication was later eliminated by the task 3-6 refactoring. V5's `runReady`/`runBlocked` also started as standalone functions but were refactored earlier (during tasks 3-5/3-6), and V5 reused `printListTable` from the start, avoiding the formatting duplication that V6 accumulated.

The cross-task pattern: V5 invested in shared infrastructure early (exported query constants, shared `printListTable`), while V6 built standalone commands first and consolidated later. V5's approach avoids the intermediate duplication state entirely.

### Context vs. Explicit Parameters

V5 threads a `*Context` struct through all commands. V6 exports `Run*` functions with explicit parameter lists (`dir string, fc FormatConfig, fmtr Formatter, ...`).

Across this phase, the difference manifests concretely:

- **V5 dep.go** accesses `ctx.Quiet`, `ctx.Fmt`, `ctx.Stdout`, `ctx.WorkDir` -- 4 fields from one parameter. V6's `RunDepAdd` takes `dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer` -- 5 parameters.
- **V5 list.go** accesses `ctx.Args` for flag parsing inside `runList`. V6 parses flags in `handleList` (app.go) and passes the parsed `ListFilter` struct to `RunList` -- better separation of parsing from execution.
- **V5 ready/blocked delegation** mutates `ctx.Args` in place (`ctx.Args = append([]string{"--ready"}, ctx.Args...)`). V6 creates a new slice in `handleReady` (`parseListFlags(append([]string{"--ready"}, subArgs...))`). V5's mutation is a mild code smell; V6's is cleaner.

The tradeoff: V5's Context reduces function signatures but couples commands to a shared struct. V6's explicit parameters make each function independently testable (no Context construction needed) but create verbose signatures. For this phase, V6's approach pays off in test helpers -- `runReady(t, dir, args...)` is simpler than constructing a full `Run([]string{...}, dir, &stdout, &stderr, false)` invocation.

## Code Quality Patterns

### ID Normalization Consistency

The most significant cross-task quality pattern is ID normalization discipline. Tracking `NormalizeID` usage across all 6 tasks:

**V5:** Normalizes consistently in every layer where IDs are compared.
- Task 3-1: 5 `NormalizeID` calls in `dependency.go` -- all comparisons use normalized IDs
- Task 3-2: Normalizes input args (`task.NormalizeID(args[0])`) AND stored deps during duplicate/rm checks (`task.NormalizeID(dep) == blockedByID`)
- Tasks 3-5/3-6: Normalizes `--parent` value via `task.NormalizeID(args[i])`

**V6:** Normalizes inconsistently.
- Task 3-1: Zero `NormalizeID` calls in `dependency.go` -- raw string equality for all comparisons. This is a correctness gap: if `ValidateDependency` receives IDs with different casing, it will fail to detect cycles or parent relationships.
- Task 3-2: Normalizes input args via `parseDepArgs` but uses direct comparison (`dep == blockedByID`) for stored deps
- Tasks 3-5/3-6: Normalizes `--parent` value correctly

V6's task 3-1 gap is particularly concerning because `ValidateDependency` is called from `RunDepAdd` (task 3-2), which normalizes its input IDs before passing them. The stored task IDs are also normalized at write time. So in practice, V6 works -- but only because of normalization at the CLI layer. If `ValidateDependency` were called from a non-CLI context (e.g., a future API layer), the lack of normalization would be a latent bug. V5's defense-in-depth approach is more robust.

### Error Message Style

Across all 6 tasks, V5 and V6 maintain internally consistent but mutually different error styles:

| Pattern | V5 | V6 |
|---------|-----|-----|
| Error casing | `"Cannot add dependency"`, `"Task '%s' not found"` | `"cannot add dependency"`, `"task '%s' not found"` |
| Error prefix style | `"querying ready tasks: %w"` (gerund) | `"failed to query ready tasks: %w"` (past participle) |
| Quoting | `%q` (Go-style double quotes) | `'%s'` (single quotes) |
| User-facing tone | Matches spec casing verbatim | Follows Go convention (lowercase) |

V5 is closer to the spec's written format. V6 is closer to Go community conventions (`go vet` recommends lowercase error strings). Since both versions prepend `"Error: "` at the App layer, V6's lowercase produces `"Error: task 'x' not found"` while V5 produces `"Error: Task 'x' not found"`. The spec writes `"Error: Task '<id>' not found"` -- V5 matches exactly.

Neither version is wrong, but V5's consistency with spec text makes verification easier during acceptance testing.

### Shared Infrastructure Investment

V5 builds shared infrastructure that compounds across tasks:

| Infrastructure | Created in | Reused in |
|---------------|-----------|----------|
| `ReadyQuery` (exported const) | 3-3 | 3-5 (`buildReadyFilterQuery`), stats |
| `BlockedQuery` (exported const) | 3-4 | 3-5 (`buildBlockedFilterQuery`), stats |
| `readyWhereClause` / `blockedWhereClause` | 3-3 / 3-4 | stats queries |
| `listRow` (package-level type) | 3-3 or earlier | 3-3, 3-4, 3-5, 3-6 |
| `printListTable` (shared formatter) | pre-3-3 | 3-3, 3-4 (before delegation refactor) |
| `appendDescendantFilter` | 3-6 | used by all 3 builder functions |
| `parentTaskExists` | 3-6 | used by `runList` |

V6 builds different shared infrastructure:

| Infrastructure | Created in | Reused in |
|---------------|-----------|----------|
| `ReadyConditions()` / `BlockedConditions()` | 3-3/3-4 (as `query_helpers.go`) | 3-5/3-6 (`buildListQuery`), stats |
| `ReadyWhereClause()` | query_helpers.go | stats |
| `openStore` (helper) | helpers.go (pre-phase 3) | All commands |
| `parseDepArgs` (shared arg parser) | 3-2 | `RunDepAdd`, `RunDepRm` |
| `parseCommaSeparatedIDs` | helpers.go | create command |
| `applyBlocks` | helpers.go | create command |
| `outputMutationResult` | helpers.go | create, update |

V6 invests more heavily in CLI-layer helpers (`openStore`, `outputMutationResult`, `applyBlocks`) that eliminate boilerplate across commands. V5 invests more in SQL-layer shared constants. Both approaches are valid; V6's CLI helpers are arguably more impactful for reducing per-command boilerplate, while V5's SQL constants provide better debugging ergonomics.

### DRY Violations

**V6's formatting duplication (tasks 3-3 and 3-4):** At commit time, `RunReady` and `RunBlocked` both contain:
```go
fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")
for _, r := range rows {
    fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
}
```
Plus a local `listRow` type definition. This is resolved by task 3-6 when both commands delegate to `RunList`, but the intermediate duplication is a quality liability.

**V5's filter duplication (task 3-5):** `buildReadyFilterQuery`, `buildBlockedFilterQuery`, and `buildSimpleFilterQuery` all contain nearly identical status/priority/descendant filter-appending code. V5 partially addresses this with `buildWrappedFilterQuery` and `appendDescendantFilter`, but the `buildSimpleFilterQuery` still has its own copy of the status/priority logic.

Both versions have DRY issues, but they occur in different areas. V6's duplication is more visible (identical code blocks across files) while V5's is more structural (parallel function shapes with shared logic).

## Test Coverage Analysis

### Test Infrastructure Divergence

V6 invests significantly more in per-command test helpers:

- `runDep(t, dir, args...)` in `dep_test.go`
- `runReady(t, dir, args...)` in `ready_test.go`
- `runBlocked(t, dir, args...)` in `blocked_test.go`
- `runList(t, dir, args...)` in `list_show_test.go`

Each helper constructs an `App` struct with injected `Stdout`, `Stderr`, and `Getwd`, calls `app.Run(fullArgs)`, and returns `(stdout, stderr, exitCode)`. This pattern eliminates 5-7 lines of boilerplate per test.

V5 calls the package-level `Run([]string{...}, dir, &stdout, &stderr, false)` directly, declaring `bytes.Buffer` variables in each test. More verbose but tests the full dispatch path including global flag parsing.

**Cross-task pattern:** V6's helpers enable terser test bodies but test through the `App` struct interface rather than the raw `Run` function. Both exercise the full CLI pipeline including flag parsing and error formatting.

### Timestamp Testing Strategy

A consistent cross-task difference:

**V5 (tasks 3-2, 3-4):** Uses `time.Sleep(1100 * time.Millisecond)` to ensure timestamp changes are observable. This adds ~1.1 seconds per timestamp test, compounding across the phase.

**V6 (tasks 3-2, 3-4, 3-6):** Sets initial timestamps to 1 hour in the past (`now.Add(-1 * time.Hour)`) and brackets the expected range with `before`/`after` captures around the mutation call. Zero sleep overhead.

Across the phase, V6's approach likely saves 3-5 seconds of test execution time -- meaningful for CI feedback loops.

### Assertion Stringency

**V5** uses `strings.Contains` for most output checks, with `t.Errorf("expected %s in output, got %q", ...)` providing diagnostic context on failure. Looser matching but better debugging.

**V6** uses exact string matching for key outputs (`stdout != expected`) and exact line matching for formatting tests. Catches more regressions (whitespace, alignment, ordering) but failure messages are sometimes terse (`t.Error("ready task should appear")` without showing actual output).

### Test Count by Task

| Task | V5 Tests | V6 Tests | Delta |
|------|----------|----------|-------|
| 3-1 | 11 | 11 | 0 |
| 3-2 | 22 | 23 | +1 V6 (no-subcommand) |
| 3-3 | 12 | 17 | +5 V6 (split combined scenarios) |
| 3-4 | 12 | 18 | +6 V6 (split + extras) |
| 3-5 | 15 | 18 | +3 V6 (separate status tests) |
| 3-6 | 15 | 15 | 0 |
| **Total** | **87** | **102** | **+15 V6** |

V6 has 17% more subtests. The additional tests come primarily from splitting combined scenarios (e.g., "excludes open/in_progress blocker" becoming two separate tests) and adding edge cases (empty project, partial unblock static scenarios). The marginal diagnostic value per extra test is moderate -- they catch the same category of bugs with better failure localization.

### Extra V6 Test Infrastructure

V6 includes `query_helpers_test.go` (56 lines) testing `ReadyConditions()`, `BlockedConditions()`, and `ReadyWhereClause()` at the unit level. V5 has no equivalent -- its SQL constants are tested only through integration tests. V6 also has `helpers_test.go` (311 lines) testing `parseCommaSeparatedIDs`, `applyBlocks`, `outputMutationResult`, and `openStore` -- general-purpose helpers that span multiple phases.

This additional test infrastructure demonstrates V6's stronger investment in unit-testing shared components independently, not just through integration tests.

## Phase Verdict

**V5 wins Phase 3 by a narrow margin (3.5 to 2.5 on the scorecard).**

The decisive factors:

1. **ID normalization discipline (V5).** V5's consistent use of `NormalizeID` across all 6 tasks -- especially in `dependency.go` (task 3-1) and stored-dep comparisons (task 3-2) -- provides defense-in-depth against case-sensitivity bugs. V6's zero-normalization approach in the validation layer is a latent correctness gap that only survives because the CLI layer normalizes first.

2. **Early DRY investment (V5).** V5's shared `listRow` type and `printListTable` function, plus exported SQL constants colocated with command files, avoid the intermediate duplication state that V6 passes through in tasks 3-3 and 3-4.

3. **Spec compliance on reusability (V5).** The spec explicitly requires "Query function reusable by blocked query and list filters." V5's exported `ReadyQuery`/`BlockedQuery` constants directly satisfy this. V6 uses unexported `readySQL`/`blockedSQL` at commit time for tasks 3-3/3-4, failing the reusability criterion until `query_helpers.go` is introduced.

**V6's strengths are real and should inform the final implementation:**

1. **Composable query architecture.** V6's `ReadyConditions()`/`BlockedConditions()` returning `[]string` slices and a single `buildListQuery` function is architecturally cleaner than V5's subquery-wrapping approach with 5 builder functions. The flat SQL is simpler and more maintainable.

2. **Type-safe validation.** V6's status validation against `task.Status*` constants is more robust than V5's hardcoded string slice.

3. **Test infrastructure.** V6's per-command test helpers, deterministic timestamps, unit-tested shared helpers, and 17% more subtests represent a more mature test suite.

4. **Separation of concerns.** V6's flag parsing in `handleList` (app.go) with a parsed `ListFilter` struct passed to `RunList` cleanly separates CLI parsing from business logic. V5 mixes parsing into `runList` itself.

**The ideal implementation** would combine V5's normalization discipline and early DRY patterns with V6's composable query architecture, type-safe validation, and test infrastructure.
