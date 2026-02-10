# Task 3-5: tick list filter flags -- --ready, --blocked, --status, --priority

## Task Plan Summary

Wire ready/blocked queries into `tick list` as flags, plus add `--status` and `--priority` filters to complete the list command to full spec. Key requirements:

- Four flags on `list`: `--ready` (bool), `--blocked` (bool), `--status <s>`, `--priority <p>`
- `--ready`/`--blocked` mutually exclusive (error if both set)
- `--status` validates: open, in_progress, done, cancelled
- `--priority` validates: 0-4
- Filters combine as AND
- Contradictory combos (e.g., `--status done --ready`) produce empty result, not error
- No filters = all tasks (backward compatible)
- Reuses ReadyQuery/BlockedQuery from tasks 3-3/3-4
- Output: aligned columns, `--quiet` IDs only
- 12 specified tests covering all filter combos, edge cases, validation, and ordering

---

## V4 Implementation

### Architecture & Design

V4 modifies three files: `list.go`, `ready.go`, and `blocked.go`.

**Flag parsing** is implemented via a hand-rolled `parseListFlags` function that iterates over `args []string` with a switch statement. It produces a `listFlags` struct:

```go
type listFlags struct {
    ready    bool
    blocked  bool
    status   string
    priority int
    hasPri   bool // distinguishes "not set" from "set to 0"
}
```

V4 uses `f.priority = -1` as an initial sentinel value, which is somewhat redundant given that `hasPri` already tracks the "was set" state. The sentinel is never actually checked elsewhere (all downstream code checks `f.hasPri`), making it dead logic.

**Query building** is handled by `buildListQuery(f listFlags) (string, []interface{})`. This function has an optimization path: when `--ready` alone or `--blocked` alone is used without additional filters, it returns the pre-existing `readyQuery` or `blockedQuery` string directly. When combined filters are needed, it builds a dynamic SQL query:

```go
query := "SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE 1=1"
```

For `--ready` combined, it appends `readyConditionsFor("t")` inline. For `--blocked` combined, it reconstructs the blocked logic inline:

```go
query += " AND t.status = 'open' AND t.id NOT IN (SELECT t2.id FROM tasks t2 WHERE t2.status = 'open' AND" +
    readyConditionsFor("t2") + ")"
```

This introduces a notable design change: it refactors the previously `const readyConditions` string into a parameterized function `readyConditionsFor(alias string)` that accepts a table alias. This was necessary because the combined blocked query requires a different alias (`t2`) to avoid SQL ambiguity. The change propagates to `ready.go` and `blocked.go`, updating all references from `readyConditions` to `readyConditionsFor("t")`.

**Integration with App pattern**: `runList` is a method on `*App` (`func (a *App) runList(args []string) error`), consistent with V4's method-based dispatch pattern. It delegates rendering to `a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)`.

### Code Quality

**Naming**: `listFlags` is clear and descriptive. Field names (`ready`, `blocked`, `status`, `priority`, `hasPri`) are concise and follow Go conventions.

**Error messages**: Include the valid values inline:
```go
fmt.Errorf("invalid status %q; valid values: open, in_progress, done, cancelled", f.status)
fmt.Errorf("invalid priority %q; valid values: 0, 1, 2, 3, 4", args[i])
```
Uses semicolons as separators.

**Error handling**: V4 combines `strconv.Atoi` error check with range check in a single condition:
```go
p, err := strconv.Atoi(args[i])
if err != nil || p < 0 || p > 4 {
    return f, fmt.Errorf("invalid priority %q; ...", args[i])
}
```
This loses specificity -- a non-numeric input gets the same "invalid priority" message as an out-of-range number. The error message uses `%q` which quotes the original input, partially mitigating this.

**SQL construction**: The `readyConditionsFor` function uses string concatenation to build SQL. While not parameterized (the alias is a code-level constant, not user input), it is a reasonable approach for table alias injection. The function doc comment explains the parameterization:

```go
// readyConditionsFor returns the WHERE clause conditions that define a "ready" task
// parameterized by table alias:
```

**Exported vs unexported**: `readyQuery`, `blockedQuery`, `readyConditionsFor`, `listFlags`, `buildListQuery` are all unexported, which is correct since they are internal to the `cli` package. The `listRow` struct has exported fields (`ID`, `Status`, `Priority`, `Title`), which is consistent with V4's formatter interface that accepts `[]listRow` directly.

**Comment quality**: All exported and key unexported functions have doc comments. The `buildListQuery` comment explains query reuse. The `readyConditionsFor` comment explains the three consumers.

### Test Coverage

V4 provides **14 test functions** covering the 12 required tests plus 2 extras:

| Required Test | V4 Implementation |
|---|---|
| "it filters to ready tasks with --ready" | `TestList_ReadyFlag` |
| "it filters to blocked tasks with --blocked" | `TestList_BlockedFlag` |
| "it filters by --status (all 4 values)" | `TestList_StatusFilter` (subtests for each status) |
| "it filters by --priority" | `TestList_PriorityFilter` |
| "it combines --ready with --priority" | `TestList_CombineReadyWithPriority` |
| "it combines --status with --priority" | `TestList_CombineStatusWithPriority` |
| "it errors when --ready and --blocked both set" | `TestList_ErrorReadyAndBlocked` |
| "it errors for invalid status/priority values" | `TestList_ErrorInvalidStatusPriority` (table-driven with 4 cases) |
| "it returns 'No tasks found.' when no matches" | `TestList_NoMatchesReturnsNoTasksFound` |
| "it outputs IDs only with --quiet after filtering" | `TestList_QuietAfterFiltering` |
| "it returns all tasks with no filters" | `TestList_AllTasksNoFilters` |
| "it maintains deterministic ordering" | `TestList_DeterministicOrdering` |
| Extra: blocked + priority combo | `TestList_CombineBlockedWithPriority` |

**Test structure**: Each test function is a top-level `TestList_*` function containing a single `t.Run()` subtest. The status filter test uses subtests for each status value. The invalid status/priority test uses a table-driven pattern with 4 cases (invalid status, negative priority, too-high priority, non-numeric priority).

**Test quality observations**:
- Tests construct tasks with explicit timestamps to control ordering.
- Tests verify both inclusion (expected results present) and exclusion (unexpected results absent).
- The deterministic ordering test runs twice to verify consistency.
- Tests use `setupInitializedDirWithTasks` helper, which creates real SQLite state.
- Tests check exit codes before examining output.

**Missing test**: The plan specifies "Contradictory combos (e.g., `--status done --ready`) -> empty result, no error". V4 does **not** have an explicit test for contradictory filters. While the behavior would work correctly (the SQL `AND t.status = 'open' AND t.status = 'done'` produces empty results), the plan-specified edge case is not tested.

### Spec Compliance

| Acceptance Criterion | V4 Status |
|---|---|
| `list --ready` = same as `tick ready` | PASS: Returns `readyQuery` when alone |
| `list --blocked` = same as `tick blocked` | PASS: Returns `blockedQuery` when alone |
| `--status` filters by exact match | PASS: Parameterized `AND t.status = ?` |
| `--priority` filters by exact match | PASS: Parameterized `AND t.priority = ?` |
| Filters AND-combined | PASS: All conditions appended with `AND` |
| `--ready` + `--blocked` -> error | PASS: "mutually exclusive" error |
| Invalid values -> error with valid options | PASS: Error messages list valid options |
| No matches -> `No tasks found.`, exit 0 | PASS: Formatter handles empty rows |
| `--quiet` outputs filtered IDs | PASS: Formatter accepts `quiet` parameter |
| Backward compatible (no filters = all) | PASS: Falls through to base query |
| Reuses query functions | PASS: `readyConditionsFor`, `readyQuery`, `blockedQuery` |
| Contradictory filters -> empty result, no error | PASS (implicit): SQL yields empty set |

**Missing test for contradictory filters**: While the behavior is correct, the explicit test from the plan is absent.

### golang-pro Skill Compliance

| MUST DO Rule | V4 Status |
|---|---|
| Use gofmt/golangci-lint | Not verifiable from diff; code formatting appears correct |
| context.Context on blocking operations | N/A: No blocking operations added in this task |
| Handle all errors explicitly | PASS: All errors returned with `fmt.Errorf` wrapping |
| Write table-driven tests with subtests | PARTIAL: Status filter and error validation use table-driven; others are standalone |
| Document all exported functions/types | PASS: `listRow`, `renderListOutput`, `readyConditionsFor` documented |
| Propagate errors with `%w` | PASS: `fmt.Errorf("failed to query tasks: %w", err)` |

| MUST NOT DO Rule | V4 Status |
|---|---|
| Ignore errors | PASS: No suppressed errors |
| Use panic for error handling | PASS: No panics |
| Hardcode configuration | PASS: Valid statuses defined as package-level `var` |

---

## V5 Implementation

### Architecture & Design

V5 modifies only `list.go` and `list_test.go`. The `ready.go` and `blocked.go` files were **not modified** by this commit. V5 already had the `ReadyQuery` and `BlockedQuery` as exported `const` strings, and the blocked query uses a direct positive-match approach (checking for existence of unclosed blockers OR open children) rather than the V4 approach of "NOT IN ready set."

**Flag parsing** uses an identical hand-rolled approach, producing a `listFilters` struct:

```go
type listFilters struct {
    ready    bool
    blocked  bool
    status   string
    priority int
    hasPri   bool // true when --priority was explicitly set
}
```

Key difference: V5 does **not** use a sentinel value for `priority` -- the zero value is fine since `hasPri` is the authoritative flag. This is slightly cleaner.

V5 also extracts a helper function `isValidStatus(s string) bool`:

```go
func isValidStatus(s string) bool {
    for _, v := range validStatuses {
        if s == v { return true }
    }
    return false
}
```

V4 inlines this logic within `parseListFlags`. The extracted function is more readable and testable in isolation, though V5 does not test it independently.

**Query building** is decomposed into multiple functions forming a clear hierarchy:

```
buildListQuery(f, descendantIDs)
  +-- buildReadyFilterQuery(f, descendantIDs)
  |     +-- buildWrappedFilterQuery(ReadyQuery, "ready", f, descendantIDs)
  +-- buildBlockedFilterQuery(f, descendantIDs)
  |     +-- buildWrappedFilterQuery(BlockedQuery, "blocked", f, descendantIDs)
  +-- buildSimpleFilterQuery(f, descendantIDs)
```

The key architectural difference is the **subquery wrapping** approach. Instead of inlining ready/blocked conditions into a single flat query, V5 wraps the existing `ReadyQuery` or `BlockedQuery` as a subquery:

```go
q := `SELECT id, status, priority, title FROM (` + ReadyQuery + `) AS ready WHERE 1=1`
```

This preserves the inner query's ordering (which is relied upon since the outer query adds no `ORDER BY`). The approach means V5 never needs to reconstruct ready/blocked SQL logic in `list.go` -- it treats `ReadyQuery` and `BlockedQuery` as opaque building blocks.

**Additional feature**: The V5 `listFilters` struct includes a `parent` field and `--parent` flag parsing, plus `queryDescendantIDs` and `parentTaskExists` helper functions with a recursive CTE query. These exist in the committed code but were likely added in a later task (3-6, parent scoping). However, they are present in the V5 snapshot and the `buildListQuery` function signature accepts `descendantIDs []string`, which adds complexity to this task's code even when the feature is unused. Importantly, the diff shows only the list filter flags being added in this commit -- the `--parent` support is included in the same file but was part of this same commit (the diff includes `--parent` parsing).

Wait -- re-examining the V5 diff more carefully: the diff only shows changes in `list.go` from the `f395505` commit. The full V5 `list.go` file includes `--parent` logic, but the diff does NOT include it, meaning `--parent` was already present before this commit or was added in a different commit. Let me verify.

Looking at the V5 diff carefully: it adds `listFilters`, `parseListFlags` (including `--parent`), `isValidStatus`, `buildListQuery`, `buildReadyFilterQuery`, `buildBlockedFilterQuery`, `buildSimpleFilterQuery`, and modifies `runList`. The `--parent` case IS in the diff (lines added include `case "--parent":` in parseListFlags). However, the `queryDescendantIDs`, `parentTaskExists`, and `appendDescendantFilter` functions are NOT in the diff, suggesting they existed before this commit.

Actually, looking more carefully at the diff structure: the diff shows the `runList` function being modified. The full file has the descendant/parent functions, but those were already present. The diff adds the filter parsing and query building that threads `descendantIDs` through. This means the `--parent` flag was partially pre-existing or V5 integrated parent scoping into this same task. Either way, the `buildListQuery` signature in V5 accepts `descendantIDs` as a parameter.

**Integration with Context pattern**: `runList` is a standalone function `func runList(ctx *Context) error`, consistent with V5's function-based dispatch via a command map. The function handles `ctx.Quiet` inline by looping and printing IDs, and for non-quiet output converts `listRow` to `TaskRow` before passing to `ctx.Fmt.FormatTaskList(ctx.Stdout, taskRows)`.

**Delegation pattern for ready/blocked commands**: V5's `runReady` and `runBlocked` simply prepend `--ready`/`--blocked` to `ctx.Args` and delegate to `runList`:

```go
func runReady(ctx *Context) error {
    ctx.Args = append([]string{"--ready"}, ctx.Args...)
    return runList(ctx)
}
```

This is elegant -- it means `tick ready` and `tick list --ready` use exactly the same code path. V4 has separate `runReady`/`runBlocked` implementations with duplicate query execution, row scanning, and formatting code.

### Code Quality

**Naming**: `listFilters` (vs V4's `listFlags`) -- both reasonable, `Filters` arguably more descriptive of purpose vs mechanism. Field names are identical.

**Error messages**: Use dashes as separators and are slightly more specific for non-numeric priority:
```go
fmt.Errorf("invalid status %q - valid values: open, in_progress, done, cancelled", f.status)
fmt.Errorf("invalid priority %q - must be a number 0-4", args[i])      // non-numeric
fmt.Errorf("invalid priority %d - valid values: 0, 1, 2, 3, 4", p)     // out of range
```

V5 separates the non-numeric and out-of-range priority cases into distinct error messages, which is more helpful for debugging. V4 combines them into one.

**Error handling**: V5 splits the priority validation into two checks:
```go
p, err := strconv.Atoi(args[i])
if err != nil {
    return f, fmt.Errorf("invalid priority %q - must be a number 0-4", args[i])
}
if p < 0 || p > 4 {
    return f, fmt.Errorf("invalid priority %d - valid values: 0, 1, 2, 3, 4", p)
}
```

This provides distinct error messages for "not a number" vs "out of range," which is better UX.

**SQL construction**: The subquery wrapping approach (`SELECT ... FROM (ReadyQuery) AS ready WHERE 1=1`) is cleaner because it never reconstructs ready/blocked SQL logic. However, it relies on SQLite preserving subquery ordering (which SQLite does when no conflicting ORDER BY is applied in the outer query). This is a minor portability concern but irrelevant for SQLite.

**Exported vs unexported**: `ReadyQuery` and `BlockedQuery` are exported `const` strings (capital R and B), while `listFilters`, `buildListQuery`, etc. are unexported. The `listRow` struct has unexported fields (`id`, `status`, `priority`, `title`), which is then converted to `TaskRow` (exported, capital fields) for the formatter. This extra conversion step adds a few lines but provides cleaner separation -- the internal representation (`listRow`) is distinct from the output contract (`TaskRow`).

**Quiet mode handling**: V5 handles quiet mode inline in `runList`:
```go
if ctx.Quiet {
    for _, r := range rows {
        fmt.Fprintln(ctx.Stdout, r.id)
    }
    return nil
}
```

V4 passes `quiet` through to the formatter. V5's approach bypasses the formatter entirely for quiet mode, which is simpler but means the formatter interface doesn't handle quiet -- the responsibility is split.

### Test Coverage

V5 provides all tests as subtests within a single `TestList` function, adding the new filter tests to the pre-existing list test function. This produces a cleaner test structure with a flat hierarchy.

| Required Test | V5 Implementation |
|---|---|
| "it filters to ready tasks with --ready" | Subtest within `TestList` |
| "it filters to blocked tasks with --blocked" | Subtest within `TestList` |
| "it filters by --status (all 4 values)" | Subtest with 4 status subtests |
| "it filters by --priority" | Subtest within `TestList` |
| "it combines --ready with --priority" | Subtest within `TestList` |
| "it combines --status with --priority" | Subtest within `TestList` |
| "it errors when --ready and --blocked both set" | Subtest within `TestList` |
| "it errors for invalid status/priority values" | Two separate subtests (status, priority) |
| "it errors for non-numeric priority" | Dedicated subtest |
| "it returns 'No tasks found.' when no matches" | Subtest within `TestList` |
| "it outputs IDs only with --quiet after filtering" | Subtest within `TestList` |
| "it returns all tasks with no filters" | Subtest within `TestList` |
| "it maintains deterministic ordering" | Subtest within `TestList` |
| **"it handles contradictory filters with empty result not error"** | **Explicit subtest** |

**Test count**: V5 has 15 subtests for the new filter functionality, including the contradictory filter edge case that V4 lacks.

**Test construction pattern**: V5 uses `task.NewTask(id, title)` to construct test tasks, then sets fields individually. V4 uses struct literal construction with all fields. V5's approach is slightly more readable when many default fields are acceptable:

```go
// V5:
blocker := task.NewTask("tick-aaaaaa", "Open blocker")
blocked := task.NewTask("tick-bbbbbb", "Blocked task")
blocked.BlockedBy = []string{"tick-aaaaaa"}

// V4:
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
    {ID: "tick-bbb222", Title: "Blocked task", Status: task.StatusOpen, Priority: 2, ...},
}
```

V5's pattern relies on `NewTask` setting sensible defaults (status=open, priority=2, timestamps). V4's pattern is more explicit but more verbose.

**Test invocation**: V5 uses a free function `Run(args, dir, stdout, stderr, isTTY)` while V4 uses `app.Run(args)` on an `App` struct. V5's is more functional; V4's is more OOP.

**Assertion for mutually exclusive error**: V5 checks that both words "ready" and "blocked" appear in stderr:
```go
if !strings.Contains(stderr.String(), "ready") || !strings.Contains(stderr.String(), "blocked") {
```
V4 checks for "mutually exclusive":
```go
if !strings.Contains(errMsg, "mutually exclusive") {
```
V4's assertion is more specific about the exact message. V5's is more robust to rewording but less precise.

**Invalid value error tests**: V5 splits into three separate subtests (invalid status, invalid priority, non-numeric priority). V4 uses a single table-driven test with 4 cases (invalid status, negative priority, too-high priority, non-numeric priority). V4's table-driven approach is more idiomatic for Go tests.

### Spec Compliance

| Acceptance Criterion | V5 Status |
|---|---|
| `list --ready` = same as `tick ready` | PASS: `runReady` delegates to `runList` with `--ready` |
| `list --blocked` = same as `tick blocked` | PASS: `runBlocked` delegates to `runList` with `--blocked` |
| `--status` filters by exact match | PASS: Parameterized SQL |
| `--priority` filters by exact match | PASS: Parameterized SQL |
| Filters AND-combined | PASS: All conditions appended with `AND` |
| `--ready` + `--blocked` -> error | PASS: "mutually exclusive" error |
| Invalid values -> error with valid options | PASS: Error messages list valid values |
| No matches -> `No tasks found.`, exit 0 | PASS: Formatter handles empty rows |
| `--quiet` outputs filtered IDs | PASS: Handled inline in `runList` |
| Backward compatible (no filters = all) | PASS: Falls through to base query |
| Reuses query functions | PASS: Wraps `ReadyQuery`/`BlockedQuery` as subqueries |
| Contradictory filters -> empty result, no error | PASS: Tested explicitly |

### golang-pro Skill Compliance

| MUST DO Rule | V5 Status |
|---|---|
| Use gofmt/golangci-lint | Not verifiable from diff; code formatting appears correct |
| context.Context on blocking operations | N/A: No blocking operations added |
| Handle all errors explicitly | PASS: All errors returned with wrapping |
| Write table-driven tests with subtests | PARTIAL: Status filter uses subtests; error tests are individual |
| Document all exported functions/types | PASS: All functions documented |
| Propagate errors with `%w` | PASS: `fmt.Errorf("querying tasks: %w", err)` |

| MUST NOT DO Rule | V5 Status |
|---|---|
| Ignore errors | PASS: No suppressed errors |
| Use panic for error handling | PASS: No panics |
| Hardcode configuration | PASS: Valid statuses defined as package-level `var` |

---

## Comparative Analysis

### Where V4 is Better

1. **Table-driven error validation tests**: V4's `TestList_ErrorInvalidStatusPriority` uses a single table-driven test with 4 cases, which is more idiomatic Go per the golang-pro skill. V5 splits this into 3 separate subtests, which is less DRY.

2. **Self-contained query building**: V4's `buildListQuery` is a single function that handles all cases. While V5's decomposition is arguably more maintainable, V4's approach is simpler to reason about in a single read-through.

3. **Formatter handles quiet mode**: V4 passes `quiet` to the formatter, keeping the rendering responsibility in a single place. V5 handles quiet mode inline in `runList`, splitting the rendering responsibility between the command handler and the formatter.

### Where V5 is Better

1. **Contradictory filter test**: V5 explicitly tests the edge case `--status done --ready` returning an empty result with exit 0. This is explicitly called out in the task plan and edge cases. V4 omits this test entirely.

2. **Delegation pattern for ready/blocked**: V5's `runReady` and `runBlocked` each delegate to `runList` by prepending `--ready`/`--blocked` to args:
   ```go
   func runReady(ctx *Context) error {
       ctx.Args = append([]string{"--ready"}, ctx.Args...)
       return runList(ctx)
   }
   ```
   This eliminates all code duplication between `tick ready`/`tick list --ready` and `tick blocked`/`tick list --blocked`. V4 retains entirely separate `runReady` and `runBlocked` implementations with duplicate query execution, row scanning, and formatting logic (~40 lines each of duplicated code). This means V4's `tick ready` and `tick list --ready` are technically separate code paths that could diverge.

3. **Distinct error messages for priority validation**: V5 provides separate error messages for non-numeric input vs out-of-range input:
   ```go
   "invalid priority %q - must be a number 0-4"    // non-numeric
   "invalid priority %d - valid values: 0, 1, 2, 3, 4"  // out of range
   ```
   V4 combines both into one generic message. V5's approach gives better user feedback.

4. **Subquery wrapping for ready/blocked filters**: V5 wraps `ReadyQuery`/`BlockedQuery` as subqueries rather than reconstructing their SQL logic. This means:
   - V5's `list.go` never needs to know the internal SQL structure of ready/blocked queries
   - Changes to `ReadyQuery`/`BlockedQuery` automatically propagate
   - No need for the `readyConditionsFor(alias)` refactoring that V4 required

   V4 had to refactor `readyConditions` from a `const` to a `func readyConditionsFor(alias string)` across three files to handle the alias conflict, and then inline-reconstruct the blocked query logic in `buildListQuery`. This is more fragile and creates tighter coupling.

5. **No unnecessary alias refactoring**: V5 keeps `readyWhereClause` and `blockedWhereClause` as simple `const` strings (hardcoded to alias `t`). V4 had to convert to a parameterized function, which is more complex and introduces string concatenation with variable aliases into SQL -- a pattern that could theoretically be misused.

6. **Cleaner `listFilters` struct**: V5 omits the `priority = -1` sentinel initialization. V4 sets it but never uses it (all logic checks `hasPri`), which is dead code.

7. **`isValidStatus` helper**: Extracted into its own function for clarity and potential reuse. V4 inlines the validation, which is less modular.

8. **Test structure as subtests**: All V5 filter tests are subtests of `TestList`, producing output like:
   ```
   TestList/it_filters_to_ready_tasks_with_--ready
   TestList/it_filters_to_blocked_tasks_with_--blocked
   ```
   This creates a cohesive test group. V4 uses separate top-level functions (`TestList_ReadyFlag`, `TestList_StatusFilter`, etc.), which scatter related tests across the file.

### Differences That Are Neutral

1. **Struct naming**: `listFlags` (V4) vs `listFilters` (V5) -- both are clear and descriptive.

2. **Error message punctuation**: V4 uses semicolons ("invalid status %q; valid values: ..."), V5 uses dashes ("invalid status %q - valid values: ..."). Neither is objectively better.

3. **Field export conventions**: V4's `listRow` has exported fields (`ID`, `Status`); V5's has unexported fields (`id`, `status`) and converts to exported `TaskRow` for the formatter. V5 adds a conversion step but achieves cleaner separation.

4. **Unknown flag error text**: V4: `"unknown flag %q for list command"`, V5: `"unknown list flag %q"`. Both are clear.

5. **Test ID patterns**: V4 uses `tick-aaa111`, `tick-bbb222`, etc. V5 uses `tick-aaaaaa`, `tick-bbbbbb`. Both create unique, recognizable IDs.

6. **`--parent` flag**: V5 includes `--parent` parsing in this commit, which was likely part of a broader plan. This adds scope beyond the task spec but doesn't break any existing behavior (it falls through to the `default` case in tests not using it). V4 does not include this.

---

## Verdict

**Winner: V5**

V5 is the stronger implementation for the following specific reasons:

1. **Superior spec compliance**: V5 includes an explicit test for the contradictory filter edge case (`--status done --ready` yields empty result, exit 0), which is explicitly called out in the task plan's edge cases section and acceptance criteria. V4 omits this test entirely. While V4's code handles it correctly at runtime, the missing test represents a gap in verification.

2. **Elimination of code duplication**: V5's delegation pattern (`runReady` prepending `--ready` and calling `runList`) removes ~80 lines of duplicated code across `ready.go` and `blocked.go`. More importantly, it guarantees that `tick ready` and `tick list --ready` are literally the same code path, making future maintenance safer. V4 has separate implementations that could silently diverge.

3. **Better encapsulation of query logic**: V5's subquery wrapping treats `ReadyQuery`/`BlockedQuery` as black boxes. V4 had to break open the ready query abstraction (converting `readyConditions` from a const to a function, modifying three files) and reconstruct blocked query logic inline in `buildListQuery`. V5's approach is more maintainable and less coupled.

4. **Better error specificity**: V5's distinct messages for non-numeric vs out-of-range priority values provide clearer user feedback.

5. **Cleaner function decomposition**: V5's `buildReadyFilterQuery`/`buildBlockedFilterQuery`/`buildSimpleFilterQuery` hierarchy with the shared `buildWrappedFilterQuery` helper is well-decomposed and follows single-responsibility principles.

V4's advantages (table-driven error tests, formatter-managed quiet mode) are real but relatively minor. The table-driven test pattern is indeed more idiomatic Go, and V5 could have structured its error tests that way. However, V4's structural shortcomings -- duplicated ready/blocked implementations, tighter query coupling, and the missing contradictory filter test -- represent more significant concerns for correctness and maintainability.
