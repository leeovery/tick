# Phase 3: Dependencies

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 3-1 Dependency validation (cycle detection, child-blocked-by-parent) | V2 | Moderate over V3 | V2 matches spec error format exactly, uses idiomatic Go comma-ok return, normalizes IDs. V1 lacks normalization entirely. |
| 3-2 `dep add` / `dep rm` commands | V2 | Clear over V3 | V2 has 25 tests covering every error path for both add AND rm independently; V1 only tests rm error paths via add. |
| 3-3 Ready query & `tick ready` | V2 | Moderate over V3 | V2 is the only version implementing `tick ready` as a true alias for `list --ready` per spec; best DRY via extending existing `list.go`. |
| 3-4 Blocked query, `tick blocked`, cancel-unblocks | V2 | Moderate over V3 | V2 integrates blocked as `list --blocked` flag; V3 duplicates output formatting between `runReady()` and `runBlocked()`. |
| 3-5 List filter flags | V2 | Narrow over V3 | V2's composable WHERE fragments + parameterized queries + `buildListQuery` pure function. V3 has unique output-equality tests but triplicated row-scanning. |

## Cross-Task Architecture Analysis

### How the five tasks compose into a dependency system

Phase 3 builds a dependency pipeline: 3-1 provides validation logic, 3-2 wires it into CLI mutations, 3-3/3-4 provide query layers, and 3-5 integrates everything into the list command. The key architectural question is: **how well do these five pieces compose?**

#### V1: Shared helper function, unexported SQL constants

V1's integration strategy is a shared `cmdListFiltered(workDir string, query string)` method introduced in task 3-3. This single function is reused by:
- `cmdReady` (3-3) passing `readyQuery`
- `cmdBlocked` (3-4) passing `blockedQuery`
- `cmdList` (3-5) passing dynamically-modified SQL via `applyListFilters`

The validation pipeline flows: `cmdDepAdd` -> `task.ValidateDependency` (3-1's pure function) -> cycle/parent checks. However, the SQL constants (`readyQuery`, `blockedQuery`) are **unexported**, meaning they exist in the `cli` package only. Task 3-5 reuses them by passing the constant strings directly to `cmdListFiltered`, then performing string surgery via `applyListFilters`:

```go
// V1 applyListFilters -- string manipulation on SQL
orderIdx := strings.LastIndex(baseQuery, "ORDER BY")
if orderIdx > 0 {
    return baseQuery[:orderIdx] + "AND " + extra + "\n" + baseQuery[orderIdx:]
}
```

This is fragile. If the SQL constants change their formatting (e.g., removing `ORDER BY`, adding CTEs), the string surgery breaks silently. The integration works for the current codebase but does not scale.

Cross-task data flow: `dependency.go` (pure, no I/O) -> `dep.go` (CLI, calls `store.Mutate`) -> `ready.go`/`blocked.go` (SQL queries via `cmdListFiltered`) -> `list.go` (filter flags via string surgery on same SQL). The seams are at SQL string boundaries.

#### V2: Composable WHERE fragments, unified list routing

V2's architecture is the most tightly integrated. The central insight is that `ready`, `blocked`, and `list` are all **variants of the same query** differing only in WHERE clauses. V2 implements this literally:

1. Task 3-1 adds `ValidateDependency` to `task.go` (validation logic).
2. Task 3-2 adds `dep.go` calling `ValidateDependency`.
3. Task 3-3 introduces `ReadySQL` as an exported constant in `list.go`, and wires `tick ready` as `runList([]string{"--ready"})`.
4. Task 3-4 adds `BlockedSQL` alongside `ReadySQL` in the same file, with `tick blocked` as `runList([]string{"--blocked"})`.
5. Task 3-5 decomposes `ReadySQL`/`BlockedSQL` into `readyWhere`/`blockedWhere` fragments, reconstructing the full constants via concatenation, and adds `buildListQuery` which composes fragments with parameterized `?` placeholders.

The SQL decomposition in task 3-5 is architecturally significant:

```go
// V2: ReadySQL composed from fragment
const readyWhere = `status = 'open'
  AND id NOT IN (...)`

const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE ` + readyWhere + `
ORDER BY priority ASC, created ASC`
```

This means `ReadySQL` (used by `tick ready`) and `buildListQuery` with `--ready` flag (used by `tick list --ready`) share the **exact same SQL fragment** at compile time. Zero runtime duplication, zero string surgery.

The command routing is equally clean:
```go
// V2 app.go -- all three commands route through runList
case "list":    return a.runList(cmdArgs)
case "ready":   return a.runList([]string{"--ready"})
case "blocked": return a.runList([]string{"--blocked"})
```

This means `ready` and `blocked` are genuinely aliases, not parallel implementations. The flag parsing, SQL selection, output formatting, quiet mode, and empty-result handling are all shared.

#### V3: Exported condition constants, but duplicated execution paths

V3 exports SQL WHERE fragments as constants (`ReadyCondition`, `BlockedCondition`) at the package level, which is the most composable SQL design. However, the execution side is not unified:

- `runReady()` calls `queryReadyTasks(db)` -- its own query function
- `runBlocked()` calls `queryBlockedTasks(db)` -- its own query function
- `runList()` calls one of `queryListTasks()`, `queryReadyTasksWithFilters()`, or `queryBlockedTasksWithFilters()` -- three more query functions

This means V3 has **five separate query functions** for what is logically three SQL patterns (all, ready, blocked) with optional filter overlays. Each contains its own row-scanning loop:

```go
// V3: This pattern appears 5 times across ready.go, blocked.go, list.go
var tasks []taskRow
for rows.Next() {
    var t taskRow
    if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
        return nil, err
    }
    tasks = append(tasks, t)
}
```

Similarly, the output formatting (column headers, quiet mode, empty message) is duplicated across `runReady()`, `runBlocked()`, and `runList()`. Task 3-4's report explicitly flags this: "V3 duplicates the full output-formatting logic between `runReady()` and `runBlocked()`."

The SQL fragment design is strong (`ReadyCondition` as a composable WHERE clause is arguably more flexible than V2's approach), but V3 fails to capitalize on it with a unified execution layer.

### Cycle detection feeding into dep add flow

All three versions follow the same pipeline: `dep add` CLI handler -> existence checks -> duplicate check -> `task.ValidateDependency(tasks, taskID, blockedByID)` -> cycle detection + parent check -> mutate.

The key cross-task pattern is that `ValidateDependency` (task 3-1) is a **pure function** taking `[]Task` -- it needs the full task list loaded into memory. In `dep add` (task 3-2), this means the task list is loaded once inside the `store.Mutate` closure and passed both to `ValidateDependency` for graph validation and used directly for the mutation. This is efficient: one read, validate, mutate, write.

V1 builds a `map[string]*Task` for O(1) lookups in the dep handler AND uses raw slice comparison in `ValidateDependency`. V2 and V3 both use linear scans in the dep handler but normalize IDs during comparison. The validation function uses a separate `taskMap` internally in all three versions -- meaning the task map is built twice (once in `dep.go`, once in `dependency.go`/`task.go`). None of the three versions pass the pre-built map from the CLI handler into the validation function. This is a minor inefficiency visible only at the phase level.

### SQL fragment sharing across ready/blocked/list

This is where the three architectures diverge most visibly:

| Aspect | V1 | V2 | V3 |
|--------|-----|-----|-----|
| SQL storage | Unexported `const` strings per file | Exported `const` + unexported WHERE fragments in `list.go` | Exported `const` WHERE fragments per file |
| SQL sharing mechanism | Pass string to `cmdListFiltered`, modify with string surgery | Compose in `buildListQuery` via `readyWhere`/`blockedWhere` fragments | Use condition constants in dedicated query functions |
| Files containing SQL | `ready.go`, `blocked.go`, `list.go` (3 files) | `list.go` only (1 file) | `ready.go`, `blocked.go`, `list.go` (3 files) |
| Query execution paths | 1 (`cmdListFiltered`) | 1 (`runList` with `db.Query`) | 5 (`queryReadyTasks`, `queryBlockedTasks`, `queryListTasks`, `queryReadyTasksWithFilters`, `queryBlockedTasksWithFilters`) |
| Parameterized queries | No (string interpolation) | Yes (`?` placeholders) | Yes (`?` placeholders) |

V2 achieves single-file, single-execution-path, parameterized SQL. V1 achieves single-execution-path but with unsafe SQL interpolation. V3 has the best SQL design (composable fragments) but the worst execution design (5 paths).

## Code Quality Patterns

### Error handling consistency

**V1**: Returns `error` from all handlers. Caller formats to stderr. Self-reference error uses a unique format ("task cannot be blocked by itself") while other errors use "Cannot add dependency" prefix. Error messages use Go-idiomatic lowercase but with em-dashes instead of hyphens. **Inconsistent within the phase.**

**V2**: Returns `error` from all handlers. Uses `unwrapMutationError` for `Mutate` errors in dep commands. All dependency errors use consistent "Cannot add dependency - ..." prefix (matching spec). Error messages use title-case ("Task '%s' not found") which is non-idiomatic Go. **Consistent format, non-idiomatic casing.**

**V3**: Returns `int` exit codes from all handlers, formats errors to stderr directly. All dependency errors use consistent "cannot add dependency - ..." prefix (lowercase). Uses Go-idiomatic lowercase. **Consistent format, idiomatic casing, but non-standard return type.**

### ID normalization consistency

This is a cross-task pattern where V1 has a systematic weakness:

- **V1 task 3-1**: No `NormalizeID` calls in `ValidateDependency` or `detectCycle`. Raw string comparison.
- **V1 task 3-2**: Uses `NormalizeID` on CLI inputs but not on stored data during map lookups.
- **V1 tasks 3-3/3-4/3-5**: SQL handles case via SQLite's built-in behavior (IDs stored lowercase).

V2 and V3 consistently call `NormalizeID` at every comparison boundary: CLI input normalization, `ValidateDependency` normalization, dep handler normalization, and `BlockedBy` array traversal normalization. This defense-in-depth approach means even if one layer's normalization were removed, the others would catch it.

### SQL subquery strategy

A subtle but consistent divergence:
- **V1**: Uses `NOT EXISTS` with correlated subqueries (blocker check uses `NOT IN ('done','cancelled')`)
- **V2**: Uses `NOT IN` with uncorrelated subqueries (same status check)
- **V3**: Uses `NOT EXISTS` with correlated subqueries (blocker check uses `IN ('open','in_progress')`)

V3's positive-match `IN ('open','in_progress')` is the semantic outlier. For the blocked query (task 3-4), V3 uses the same `IN ('open','in_progress')` pattern. This means V3 treats unknown future statuses as non-blocking, while V1 and V2 treat them as blocking. This is an architectural decision that cuts across tasks 3-3 and 3-4 and would only be visible at the phase level.

### DRY across the phase

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| SQL query execution | 1 shared function (`cmdListFiltered`) | 1 shared function (`runList`) | 5 separate functions |
| Output formatting | 1 shared function | 1 shared function | 3 copies (ready, blocked, list) |
| Store opening | Shared via `cmdListFiltered` | Shared via `runList` | Repeated in `runReady`, `runBlocked`, `runList` |
| Row scanning | 1 implementation | 1 implementation | 5 implementations |
| Empty result message | 1 check in `cmdListFiltered` | 1 check in `runList` | 3 checks |
| Quiet mode | 1 check in `cmdListFiltered` | 1 check in `runList` | 3 checks |

V2 is clearly the DRYest. V1 is close but uses unsafe SQL practices. V3 has significant duplication that would compound as more filter modes were added.

## Test Coverage Analysis

### Aggregate test counts

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Task 3-1 test functions | 11 | 11 | 12 |
| Task 3-2 test functions | 16 | 25 | 23 |
| Task 3-3 test functions | 10 | 17 | 17 |
| Task 3-4 test functions | 10 | 16 | 17 |
| Task 3-5 test functions | 12 | 18 | 20 |
| **Phase total** | **59** | **87** | **89** |
| Task 3-1 test LOC | 163 | 181 | 248 |
| Task 3-2 test LOC | 262 | 554 | 692 |
| Task 3-3 test LOC | 179 | 510 | 477 |
| Task 3-4 test LOC | 175 | 524 | 530 |
| Task 3-5 test LOC | 213 | 674 | 611 |
| **Phase test LOC total** | **992** | **2443** | **2558** |

### Testing approach patterns

**V1** consistently uses integration-style tests: create tasks via the actual CLI, run commands, inspect JSONL files. This is realistic but slower and less isolated. V1 tests use `strings.Contains` assertions rather than exact string matching, which means format regressions could slip through.

**V2** consistently uses unit-style tests: construct `App` with injected stdout, set up JSONL content directly via helpers (`setupTickDirWithContent`, `taskJSONL`), inspect results via custom readers (`readTaskByID`). Uses exact string matching for error messages. Organizes tests into logical groups (e.g., `TestDepAddCommand`, `TestDepRmCommand`, `TestDepSubcommandRouting`).

**V3** consistently uses unit-style tests similar to V2: `bytes.Buffer` for stdout/stderr, `setupTaskFull` helper for test data, direct App construction. Uses exact string matching. Has unique cross-feature tests (`"it matches tick ready output"` comparing `list --ready` to `tick ready`).

### Cross-task test patterns

**V2's test helper ecosystem** is the most developed: `setupTickDirWithContent`, `taskJSONL`, `openTaskJSONL`, `openTaskWithBlockedByJSONL`, `readTaskByID`, `itoa`. These are reused across tasks 3-2 through 3-5, providing a consistent test vocabulary.

**V3's test helpers** are also consistent (`setupTickDir`, `setupTask`, `setupTaskFull`, `readTasksFromDir`) but are sometimes redefined within each test file rather than shared.

**V1's test helpers** (`initTickDir`, `createTask`, `runCmd`, `extractID`) are the most end-to-end but least reusable for testing query logic in isolation.

### Notable test gaps visible only at the phase level

1. **V1 never tests `list --ready` or `list --blocked`** as flags. Tasks 3-3 and 3-4 create standalone commands, and task 3-5 adds flags, but V1's tests for 3-3/3-4 do not verify that `list --ready` produces the same output as `tick ready`.

2. **V3 is the only version with output equality tests** (`list --ready` == `tick ready` and `list --blocked` == `tick blocked`). This is a phase-level integration test that verifies the cross-task contract between tasks 3-3/3-4 and 3-5.

3. **No version tests the full pipeline end-to-end**: create task with `--blocked-by`, verify it appears in `tick blocked`, cancel the blocker, verify it moves to `tick ready`, then verify `tick list --ready` shows it. V2's cancel-unblocks tests in task 3-4 come closest but do not exercise the list filter path.

## Phase Verdict

**V2 is the clear phase winner.**

V2 wins on architecture, DRY principle, SQL safety, and test quality -- the four dimensions that matter most at the phase level.

**Architecture**: V2 is the only version where `ready`, `blocked`, and `list` are unified through a single execution path (`runList`). The command routing (`case "ready": return a.runList([]string{"--ready"})`) means there is literally one code path for all three commands. V1 achieves partial unification via `cmdListFiltered` but undermines it with string surgery. V3 has the best SQL fragment design but the worst execution design (5 query functions, 3 output formatters).

**SQL safety**: V2 uses parameterized queries throughout. V3 also uses parameterized queries. V1 uses `fmt.Sprintf` to interpolate user input into SQL -- a security anti-pattern that would be flagged in any code review, even though the input is pre-validated.

**DRY**: V2 has one query execution path, one output formatter, one store-opening sequence, one row scanner. V3 has 5 query functions, 3 output formatters, 3 store-opening sequences, and 5 row scanners. At the phase level, V3's duplication is its most significant weakness.

**Test coverage**: V2 has 87 test functions (vs V1's 59 and V3's 89). V3 has the raw count lead, but V2's tests are better organized (separate test groups for add/rm/routing in 3-2) and cover more edge cases per task. V2 is the only version that tests `list --ready` and `list --blocked` as flags in tasks 3-3 and 3-4 respectively. V3 compensates with unique output equality tests in task 3-5.

**ID normalization**: V2 and V3 both normalize consistently; V1 does not normalize in the validation layer, creating a correctness risk for case-sensitive ID comparisons.

**Spec compliance**: V2's error messages match the specification format most closely across all five tasks. V1 uses em-dashes and inconsistent prefixes. V3 uses lowercase "cannot" and adds unsolicited explanation text.

**V2's only weakness**: It places dependency validation in `task.go` rather than a dedicated file (task 3-1), and uses title-case error messages in the dep commands (task 3-2) which is non-idiomatic Go.

**Final ranking: V2 > V3 > V1.**

V2 wins all five tasks individually and demonstrates the strongest cross-task integration. V3 has better SQL composability and test count but undermines itself with execution-path duplication. V1 is functional but has systematic weaknesses (no ID normalization, SQL injection patterns, weakest test suite) that compound across the phase.
