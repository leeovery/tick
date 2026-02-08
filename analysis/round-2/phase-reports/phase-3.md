# Phase 3: Dependencies

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 3-1 Dependency validation | V4 | Moderate | Better decomposition (5 focused functions vs 2 monolithic); dedicated `dependency.go` file; more thorough tests (12 vs 11 subtests). V2 has stronger defensive normalization via `NormalizeID()` at every comparison. |
| 3-2 dep add/rm CLI | V2 | Moderate | V2 tests 4 edge cases V4 misses (stale ref removal, rm persistence, one-arg rm, unknown subcommand). V4 has better typed test setup and early self-ref optimization. |
| 3-3 Ready query | V2 | Moderate | V2 tests `"No tasks found."` correctly per spec; V4 checks TOON format `tasks[0]` instead. V2 tests `list --ready` alias. V4 has better SQL (`NOT EXISTS`) and reuse (`readyConditionsFor(alias)`). |
| 3-4 Blocked query | V4 | Strong | V4 genuinely reuses ready logic via shared `readyConditions`; V2 duplicates inverse SQL manually. V4 fulfills acceptance criterion #11 that V2 fails. |
| 3-5 List filter flags | V4 | Moderate | V4's `readyConditionsFor(alias)` avoids SQL ambiguity in nested queries; fast-path optimization returns exact ready/blocked SQL; unknown-flag error handling. V2 misses `--blocked --priority` test combo. |

## Cross-Task Architecture Analysis

### The Central Design Divergence: SQL Fragment Reuse Strategy

The most significant cross-task pattern is how ready/blocked/list SQL logic composes across tasks 3-3, 3-4, and 3-5. This is invisible at the individual task level but defines the phase's architectural quality.

**V2: Independent SQL constants with manual inversion**

V2 defines `readyWhere`, `blockedWhere`, `ReadySQL`, and `BlockedSQL` as independent string constants in a single `list.go` file:

```go
// list.go (V2)
const readyWhere = `status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )`

const blockedWhere = `status = 'open'
  AND (
    id IN (
      SELECT d.task_id FROM dependencies d
      JOIN tasks t ON d.blocked_by = t.id
      WHERE t.status NOT IN ('done', 'cancelled')
    )
    OR id IN (
      SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
    )
  )`
```

The `blockedWhere` is a manually-written logical negation of `readyWhere` -- `NOT IN` becomes `IN`, the two conditions switch from AND to OR. These must be maintained in lockstep. Any change to ready conditions requires a parallel change to blocked conditions.

The `buildListQuery` function then wraps these fragments in parentheses and joins them with AND:

```go
if flags.ready {
    where = append(where, "("+readyWhere+")")
} else if flags.blocked {
    where = append(where, "("+blockedWhere+")")
}
```

**V4: Single source of truth with parameterized reuse**

V4 defines a single `readyConditionsFor(alias)` function in `ready.go` that generates the ready conditions for any table alias:

```go
// ready.go (V4)
func readyConditionsFor(alias string) string {
    return `
  NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = ` + alias + `.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = ` + alias + `.id
      AND child.status IN ('open', 'in_progress')
  )`
}
```

The blocked query in `blocked.go` derives itself via set subtraction:

```go
// blocked.go (V4)
var blockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND t.id NOT IN (
    SELECT t.id FROM tasks t
    WHERE t.status = 'open'
      AND` + readyConditionsFor("t") + `
  )
ORDER BY t.priority ASC, t.created ASC`
```

And `buildListQuery` in `list.go` uses the alias parameter to avoid SQL ambiguity:

```go
// list.go (V4)
if f.blocked {
    query += " AND t.status = 'open' AND t.id NOT IN (SELECT t2.id FROM tasks t2 WHERE t2.status = 'open' AND" +
        readyConditionsFor("t2") + ")"
}
```

This is the defining architectural difference of Phase 3. V4's approach means:
1. Ready logic is defined once, changes propagate automatically
2. The `alias` parameter prevents SQL table-name collisions in nested contexts
3. `blockedQuery` is mathematically derived from ready conditions, not manually inverted

### Command Dispatch Pattern

V2 routes `ready` and `blocked` as aliases into `runList` by injecting synthetic flags:

```go
// app.go (V2)
case "ready":
    return a.runList([]string{"--ready"})
case "blocked":
    return a.runList([]string{"--blocked"})
```

V4 routes to dedicated handlers:

```go
// cli.go (V4)
case "ready":
    if err := a.runReady(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

V2's approach is more DRY -- `runList` handles all three code paths (list, ready, blocked). V4's approach creates near-identical `runReady` and `runBlocked` methods that differ only in the query variable and one error message string:

```diff
-func (a *App) runReady(args []string) error {
+func (a *App) runBlocked(args []string) error {
     // ...identical store opening boilerplate...
-    sqlRows, err := db.Query(readyQuery)
+    sqlRows, err := db.Query(blockedQuery)
         if err != nil {
-            return fmt.Errorf("failed to query ready tasks: %w", err)
+            return fmt.Errorf("failed to query blocked tasks: %w", err)
```

This is 30+ lines of boilerplate duplicated between two files. V2 avoids this entirely. However, V4's `buildListQuery` has a compensating advantage: its fast-path optimization returns the exact `readyQuery`/`blockedQuery` string when no additional flags are set, ensuring `tick list --ready` and `tick ready` execute byte-identical SQL.

### Formatter Interface Division

V2 separates `listRow` (local to `runList`) from `TaskRow` (in `formatter.go`), requiring a conversion step:

```go
// V2 list.go
taskRows := make([]TaskRow, len(rows))
for i, r := range rows {
    taskRows[i] = TaskRow{ID: r.ID, Status: r.Status, Priority: r.Priority, Title: r.Title}
}
return a.formatter.FormatTaskList(a.stdout, taskRows)
```

V4 defines `listRow` at package level and passes it directly to the formatter, which also accepts `quiet` as a parameter:

```go
// V4 list.go
return a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)
```

V4's design means `listRow` is shared across `list.go`, `ready.go`, and `blocked.go` without type conversion. It also pushes quiet-mode rendering into the formatter where it belongs, rather than having inline quiet-mode handling in the command function.

### Error Propagation Architecture

V2's `App.Run` returns `error`, and `main.go` adds the "Error: " prefix:

```go
// main.go (V2)
if err := app.Run(os.Args); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err)
    os.Exit(1)
}
```

V2 also requires `unwrapMutationError()` after every `store.Mutate()` call (used in `dep.go`, `create.go`, `update.go`, `transition.go` -- 5 call sites total):

```go
// V2 dep.go
err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) { ... })
if err != nil {
    return unwrapMutationError(err)
}
```

V4's `App.Run` returns `int` and handles errors inline via `writeError`:

```go
// cli.go (V4)
func (a *App) writeError(err error) {
    fmt.Fprintf(a.Stderr, "Error: %s\n", err.Error())
}
```

V4 does not need `unwrapMutationError` -- its storage layer apparently does not wrap mutation errors. This eliminates a cross-command boilerplate function that V2 must call at every mutation site.

**Note on task report inaccuracy**: The task 3-1 report states V2 is missing the "Error: " prefix while V4 has it. In reality, both produce the same user-facing output: both validation functions omit "Error:", and both add it at the top level (V2 in `main.go`, V4 in `writeError`). The functional difference is case: V2 uses `"Cannot add dependency"` while V4 uses `"cannot add dependency"` (lowercase). The spec shows `"Error: Cannot add dependency"` with uppercase C, so V2's casing is closer to the spec format after the "Error: " prefix is prepended.

### Validation Layer Architecture

V2 embeds dependency validation directly into `task.go` (the existing ~150-line model file), growing it to ~357 lines. The validation functions coexist with `NewTask`, `Transition`, `ValidateTitle`, `ValidatePriority`, etc.

V4 creates a dedicated `dependency.go` (135 lines) with 7 focused functions. This separation means:
- Dependency logic can be reviewed/modified without touching the core task model
- Function count per file stays manageable
- `dependency_test.go` tests dependency logic in isolation from task creation/transition tests

## Code Quality Patterns

### Case Normalization: Defensive vs Trust-Based

This is the most pervasive cross-task quality pattern. V2 applies `task.NormalizeID()` at every comparison point:

```go
// V2 dep.go -- normalizes stored IDs during lookup
normalizedID := task.NormalizeID(tasks[i].ID)
if normalizedID == taskID { ... }

// V2 dep.go -- normalizes blocked_by entries during duplicate check
if task.NormalizeID(existing) == blockedByID { ... }

// V2 task.go -- normalizes inside cycle detection
normalizedDep := NormalizeID(dep)
if normalizedDep == targetID { ... }
```

V4 normalizes only at the CLI entry point and trusts stored data is already lowercase:

```go
// V4 dep.go -- direct comparison, no normalization
if tasks[i].ID == taskID { ... }
if dep == blockedByID { ... }
```

This pattern holds across all Phase 3 tasks. V2 uses `NormalizeID()` 6 times in `ValidateDependency`, 4 times in `runDepAdd`, and 3 times in `runDepRm`. V4 uses it exactly twice -- both in `dep.go` at the CLI entry boundary.

V2's approach is safer against data corruption. V4's is cleaner if the invariant (IDs stored lowercase) is maintained by the storage layer. Neither is strictly wrong, but V2's defensive style trades performance for robustness.

### SQL Idiom: NOT IN vs NOT EXISTS

V2 consistently uses `NOT IN` subqueries across ready, blocked, and list:

```sql
-- V2 readyWhere
AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
)
```

V4 consistently uses `NOT EXISTS` with correlated subqueries:

```sql
-- V4 readyConditionsFor
AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
)
```

`NOT EXISTS` is the preferred SQL idiom: it handles NULL correctly, can short-circuit evaluation, and receives better query plan optimization. V4 also aliases the joined table as `blocker` instead of reusing `t`, preventing ambiguity with the outer table alias.

### Error Message Casing

V2 consistently uses uppercase first letter: `"Cannot add dependency"`, `"Task '%s' not found"`, `"Unknown dep subcommand"`.

V4 uses lowercase for validation errors (`"cannot add dependency"`) but uppercase for CLI errors (`"Task '%s' not found"`, `"Unknown dep subcommand"`). This inconsistency in V4 is a minor quality issue.

### Graph Traversal Algorithm Choice

V2 uses DFS with path-carrying stack frames. V4 uses BFS with a parent map and path reconstruction. Both are correct for cycle detection. The algorithmic tradeoff:

- V2's DFS carrying full paths: O(V * max_path_length) memory, but path is immediately available when cycle is found
- V4's BFS with parent map: O(V) memory for the parent map, plus O(path_length) for reconstruction; BFS finds the shortest cycle path

For a task tracker with typically small dependency graphs, neither matters for performance. V4's BFS finding the shortest cycle path produces slightly more helpful error messages.

## Test Coverage Analysis

### Aggregate Counts

| Metric | V2 | V4 |
|--------|-----|-----|
| Phase 3 impl files | 3 (task.go partial, dep.go, list.go) | 5 (dependency.go, dep.go, list.go, ready.go, blocked.go) |
| Phase 3 test files | 4 (task_test.go partial, dep_test.go, ready_test.go, blocked_test.go, list_test.go) | 5 (dependency_test.go, dep_test.go, ready_test.go, blocked_test.go, list_test.go) |
| Phase 3 impl LOC | ~395 (dep.go: 169, list.go: 226, task.go dep portion: ~100) | ~608 (dep.go: 171, list.go: 161, ready.go: 79, blocked.go: 62, dependency.go: 135) |
| Phase 3 test LOC | ~2,262 (dep_test: 554, ready_test: 510, blocked_test: 524, list_test: 674) | ~2,241 (dep_test: 632, ready_test: 494, blocked_test: 497, list_test: 618) |
| Total subtests (est.) | ~65 | ~62 |

### Test Helper Strategy (Cross-Task Pattern)

**V2** defines per-file JSONL string helpers that are not shared between test files:
- `dep_test.go`: `twoOpenTasksJSONL()`, `openTaskWithBlockedByJSONL()`, `openTaskWithParentJSONL()` (relies on `openTaskJSONL` from `transition_test.go` since they share the package)
- `ready_test.go`: `taskJSONL()` -- a general-purpose helper with 7 parameters
- `blocked_test.go`: Reuses `taskJSONL()` from `ready_test.go` (same package)
- All use `setupTickDirWithContent()` from `create_test.go`

**V4** uses a single centralized setup pattern across all test files:
- `create_test.go`: Defines `setupInitializedDir()`, `setupInitializedDirWithTasks()`, `readTasksFromDir()`
- Every phase 3 test file (`dep_test.go`, `ready_test.go`, `blocked_test.go`, `list_test.go`) uses `setupInitializedDirWithTasks()` with typed `task.Task` structs

V4's approach is significantly better for cross-task composition. The typed `task.Task` struct setup:
1. Gets compile-time field name validation (a typo like `Statsu` is caught at build time)
2. Uses typed constants (`task.StatusOpen`) instead of string literals (`"open"`)
3. Requires no conversion -- the same `task.Task` struct flows from test setup to storage to assertion
4. Is used identically across 5+ test files with no per-file helper variants

V2's raw JSONL approach means each test file creates its own JSONL builder with subtly different parameter sets and field orderings. A schema change (e.g., adding a required field) would require updating multiple helper functions across multiple test files.

### Edge Cases Tested By Only One Version

| Edge Case | Tested By | Why It Matters |
|-----------|-----------|----------------|
| Stale ref removal via `dep rm` | V2 only | Spec explicitly says rm does not validate blocked_by_id exists as a task |
| `dep rm` atomic write persistence | V2 only | Verifies mutation survives storage round-trip |
| `dep rm` with one arg (not two) | V2 only | Tests argument count validation for rm path |
| Unknown dep subcommand | V2 only | Tests error for `tick dep foo` |
| `tick list --ready` alias | V2 only | Verifies spec requirement that ready is alias for list --ready |
| `--blocked --priority` combined | V4 only | Tests filter combination V2 misses |
| Contradictory `--status done --ready` | V2 only | Verifies empty result without error for impossible combo |
| Batch validation happy path | V4 only | Tests `ValidateDependencies` with all valid deps |

### Testing Philosophy Divergence

V2 groups related tests under 2-3 top-level test functions with many subtests. V4 creates one top-level test function per scenario. This persists across all 5 tasks:

- V2 task 3-4: 3 top-level functions, 16 subtests
- V4 task 3-4: 14 top-level functions, 14 subtests

V4's approach provides better test isolation (each function gets its own setup/teardown) and clearer `go test -run` targeting, but creates more boilerplate. V2's grouping provides logical organization and shared setup.

## Phase Verdict

**V4 is the better Phase 3 implementation**, primarily due to a single architectural decision that cascades through three of the five tasks: the `readyConditionsFor(alias)` function that provides genuine single-source-of-truth reuse for ready/blocked/list SQL conditions.

**Architecture (V4 wins decisively):**
The ready-conditions reuse strategy is the phase's defining quality. V2's manual inversion of `readyWhere` into `blockedWhere` violates DRY and creates a maintenance hazard -- any change to what makes a task "ready" requires a parallel change to the blocked query. V4's `readyConditionsFor(alias)` function eliminates this: `blockedQuery` is derived automatically via set subtraction, and `buildListQuery` can safely embed ready conditions in nested subqueries using different table aliases. This directly fulfills acceptance criterion 3-4 #11 ("Reuses ready query logic"), which V2 fails.

**File organization (V4 wins):**
V4 separates concerns into `dependency.go`, `dep.go`, `ready.go`, `blocked.go`, and `list.go` -- each file owns one concept. V2 packs dependency validation into the already-large `task.go` and puts ready/blocked/list SQL into a single `list.go`. V4's separation makes each file independently reviewable and keeps function counts per file manageable.

**SQL quality (V4 wins):**
V4 consistently uses `NOT EXISTS` (NULL-safe, optimizer-friendly) over V2's `NOT IN`. V4's parameterized table aliases prevent ambiguity in nested SQL contexts. Both produce correct results for this workload, but V4's SQL is more robust under evolution.

**Test quality (V2 wins narrowly):**
V2 covers more edge cases overall (stale ref removal, rm persistence, contradictory filters, `list --ready` alias). V4 has better test infrastructure (typed `task.Task` structs, centralized `setupInitializedDirWithTasks` helper, exit-code-based assertions). The test infrastructure advantage matters more long-term, but V2's coverage of spec-mandated edge cases is notable.

**Defensive coding (V2 wins):**
V2's pervasive `NormalizeID()` at every comparison point is more robust against data corruption. V4 normalizes only at CLI entry points and trusts the storage layer. V2 also has the `unwrapMutationError` pattern that handles a storage-layer wrapping concern, while V4's cleaner storage layer makes this unnecessary.

**Cross-task composition (V4 wins):**
V4's `listRow` type is defined once at package level and shared across `list.go`, `ready.go`, and `blocked.go` -- no type conversion needed. V4's formatter interface accepts `quiet` as a parameter, keeping rendering logic in the formatter. V2 has inline quiet-mode handling in `runList` and a `listRow` to `TaskRow` conversion step.

**Net assessment:** V4's architectural advantages (genuine reuse, file separation, SQL quality, shared types) outweigh V2's edge-case test coverage. V2's test gaps can be filled incrementally; V4's architecture fundamentally prevents the class of maintenance bugs that V2's duplicated SQL introduces. The phase-level winner is V4.
