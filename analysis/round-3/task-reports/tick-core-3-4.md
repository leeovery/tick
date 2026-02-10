# Task 3-4: Blocked query, tick blocked & cancel-unblocks-dependents

## Task Plan Summary

This task implements the inverse of `tick ready`: the `tick blocked` command, which lists open tasks that cannot be worked. A task is blocked if it is `open` AND fails ready conditions (has at least one unclosed blocker OR has open/in_progress children). The command should be an alias for `list --blocked`. The plan also requires verifying that cancelling a blocker unblocks its dependents end-to-end.

Key requirements:
- `BlockedQuery` = open tasks minus ready tasks (reuse `ReadyQuery` logic)
- Order: priority ASC, created ASC
- Register `blocked` subcommand
- Cancelled blockers count as closed, unblocking dependents
- Only `open` tasks in output
- 12 specified tests covering query correctness, empty state, ordering, output formatting, quiet mode, and cancel-unblocks scenarios

## V4 Implementation

### Architecture & Design

V4 takes a **set-subtraction approach** to blocked query definition. It extracts `readyConditions` as a shared `const` string from `ready.go` and composes `blockedQuery` as:

```go
var blockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND t.id NOT IN (
    SELECT t.id FROM tasks t
    WHERE t.status = 'open'
      AND` + readyConditions + `
  )
ORDER BY t.priority ASC, t.created ASC
`
```

This directly implements the spec's definition: "blocked = open AND NOT ready". The `readyConditions` constant is defined in `ready.go` and hardcodes the table alias `t`:

```go
const readyConditions = `
  NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )`
```

The `runBlocked` function is a method on `*App` (the V4 app struct pattern) and performs its own store opening, query execution, row scanning, and rendering via `renderListOutput`. At commit time, the rendering was done by `renderListOutput` (a shared function in `list.go`). In later commits, this was refactored to use `a.Formatter.FormatTaskList`.

The `blocked` subcommand is registered via a `case "blocked":` in the `switch` statement in `cli.go`.

**SQL approach note**: The `NOT IN` subquery uses `SELECT t.id FROM tasks t` with the same alias `t` in both outer and inner queries. This works in SQLite because the inner `t` shadows the outer `t`, but it is a potential confusion point and would be clearer with distinct aliases.

### Code Quality

- **Error wrapping**: Uses `fmt.Errorf("failed to query blocked tasks: %w", err)` -- proper error propagation with `%w`.
- **Resource cleanup**: `defer sqlRows.Close()` and `defer s.Close()` present.
- **Naming**: `blockedQuery` is a package-level `var` (not `const`) because it uses string concatenation with `readyConditions`. This is appropriate.
- **Comments**: Thorough doc comments on `blockedQuery` and `runBlocked`, explaining the derivation from ready conditions.
- **Exported fields**: `listRow` uses exported fields (`ID`, `Status`, `Priority`, `Title`), which is appropriate for a struct used across formatting functions within the same package.
- **Code size**: `blocked.go` is 62 lines (at HEAD; 64 at commit time). Clean and focused.

### Test Coverage

V4 has **14 subtests** across 493 lines (at commit time). The test names map to the plan's 12 required tests, with two extras:

| Plan Test | V4 Test |
|-----------|---------|
| "it returns open task blocked by open/in_progress dep" | Split into TWO separate tests: `TestBlocked_ReturnsOpenTaskBlockedByOpenDep` and `TestBlocked_ReturnsOpenTaskBlockedByInProgressDep` |
| "it returns parent with open/in_progress children" | Split into TWO: `TestBlocked_ReturnsParentWithOpenChildren` and `TestBlocked_ReturnsParentWithInProgressChildren` |
| "it excludes task when all blockers done/cancelled" | `TestBlocked_ExcludesTaskWhenAllBlockersDoneOrCancelled` |
| "it excludes in_progress/done/cancelled from output" | `TestBlocked_ExcludesNonOpenStatuses` |
| "it returns empty when no blocked tasks" | `TestBlocked_EmptyWhenNoBlockedTasks` |
| "it orders by priority ASC then created ASC" | `TestBlocked_OrderByPriorityThenCreated` |
| "it outputs aligned columns via tick blocked" | `TestBlocked_AlignedColumnsOutput` |
| "it prints 'No tasks found.' when empty" | `TestBlocked_NoTasksFoundMessage` |
| "it outputs IDs only with --quiet" | `TestBlocked_QuietOutputsIDsOnly` |
| "cancel unblocks single dependent" | `TestBlocked_CancelUnblocksSingleDependent` |
| "cancel unblocks multiple dependents" | `TestBlocked_CancelUnblocksMultipleDependents` |
| "cancel does not unblock dependent still blocked by another" | `TestBlocked_CancelDoesNotUnblockDependentStillBlockedByAnother` |

**Test structure**: Each test is a standalone `func TestBlocked_Xxx(t *testing.T)` with a single `t.Run` subtest inside. This is somewhat redundant -- the top-level function name already identifies the test, and the inner `t.Run` name repeats it. However, it is consistent with the project's pattern.

**Test setup**: Uses `setupInitializedDirWithTasks(t, tasks)` and manually constructs `task.Task` structs with explicit field values including `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)` and manual `time.Add` offsets for creation ordering.

**Cancel-unblocks tests**: These are multi-step integration tests that:
1. Create tasks with blocking relationships
2. Verify the dependent appears in `tick blocked` output
3. Run `tick cancel` on the blocker
4. Verify the dependent appears in `tick ready` (and not in `tick blocked`)

The "cancel unblocks single dependent" test has 4 steps (blocked check, cancel, ready check, blocked recheck). This is thorough.

**Assertion style**: Uses `strings.Contains` for output checking and exact string comparison for "No tasks found." message. The ordering test uses line-by-line `strings.Contains` checks, which is appropriate.

### Spec Compliance

- **blocked = open AND NOT ready**: Implemented via `NOT IN` subquery referencing `readyConditions`. Fully compliant.
- **Reuses ready query logic**: Yes, `readyConditions` is extracted and shared. Change to ready conditions automatically propagates to blocked.
- **Only open in output**: The outer `WHERE t.status = 'open'` ensures this.
- **Cancel unblocks dependent**: Tested end-to-end with 3 cancel scenarios.
- **Deterministic ordering**: `ORDER BY t.priority ASC, t.created ASC` present.
- **Aligned columns**: `renderListOutput` uses `fmt.Fprintf` with `%-12s` formatting.
- **Empty state**: "No tasks found." message, exit 0.
- **--quiet**: IDs only output.
- **Register blocked subcommand**: Added to `cli.go` switch.

All acceptance criteria are met.

### golang-pro Skill Compliance

- **Error handling**: All errors handled explicitly with `%w` wrapping. Compliant.
- **Table-driven tests**: NOT used. Each test is a standalone function. The plan test names could have been organized as table-driven subtests under logical groupings (e.g., query tests, command tests, cancel tests). This is a **minor deviation** from the skill's "Write table-driven tests with subtests" requirement.
- **Exported function documentation**: `blockedQuery` is unexported (package-level `var`), but has a thorough doc comment. `runBlocked` is unexported method, documented.
- **No panics for error handling**: Compliant.
- **No ignored errors**: Compliant.

## V5 Implementation

### Architecture & Design

V5 takes a **direct positive expression** approach. Instead of defining blocked as "NOT IN ready set", it directly writes the blocked conditions as their own SQL:

```go
const BlockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON blocker.id = d.blocked_by
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )
ORDER BY t.priority ASC, t.created ASC
`
```

This means the blocked query does NOT reuse the ready query's conditions at all. It is a standalone, independently written SQL query that happens to be the logical inverse. If the ready conditions change (e.g., a new blocking condition is added), `BlockedQuery` must be manually updated in parallel.

The `runBlocked` function is a package-level function taking `*Context` (V5's CLI context pattern). At commit time, it performs its own store opening, query execution, row scanning, and inline rendering:

```go
if len(rows) == 0 {
    fmt.Fprintln(ctx.Stdout, "No tasks found.")
    return nil
}
if ctx.Quiet {
    for _, r := range rows {
        fmt.Fprintln(ctx.Stdout, r.id)
    }
    return nil
}
printListTable(ctx.Stdout, rows)
```

The `blocked` subcommand is registered via the `commands` map in `cli.go`, which is more idiomatic than a switch statement:

```go
var commands = map[string]func(*Context) error{
    ...
    "blocked": runBlocked,
}
```

**Later evolution**: In subsequent commits (730feed for parent scoping, 41e0947 for shared SQL WHERE clauses), V5's `blocked.go` was refactored. The current HEAD version extracts `blockedWhereClause` as a shared `const` and delegates `runBlocked` to `runList` with `--blocked` prepended. However, this analysis focuses on commit c66ead3.

### Code Quality

- **Error wrapping**: Uses `fmt.Errorf("querying blocked tasks: %w", err)` -- slightly more concise prefix style than V4's "failed to query blocked tasks:". Both are proper `%w` wrapping.
- **Resource cleanup**: `defer sqlRows.Close()` and `defer store.Close()` present.
- **Naming**: `BlockedQuery` is exported (`const`). The V5 convention exports query constants, while V4 keeps them unexported. Exporting is less ideal for an internal package -- the query is only used within `cli` and doesn't need external visibility.
- **Comments**: Good doc comments on `BlockedQuery` and `runBlocked`.
- **Field names**: `listRow` uses unexported fields (`id`, `status`, `priority`, `title`), which is more idiomatic Go for an internal struct not shared across packages.
- **Code size**: `blocked.go` is 85 lines at commit time (vs V4's 64). The extra lines come from inline rendering logic that V4 delegates to `renderListOutput`.
- **Inline rendering**: The empty/quiet/table logic is inlined in `runBlocked` rather than extracted to a shared function. This creates code duplication with `runReady` and `runList`, all of which have the same rendering pattern.

### Test Coverage

V5 has **12 subtests** across 431 lines. Tests are organized under 3 top-level functions:

1. `TestBlockedQuery` (6 subtests): Query correctness tests
2. `TestBlockedCommand` (3 subtests): CLI output formatting tests
3. `TestCancelUnblocksDependents` (3 subtests): End-to-end cancel tests

| Plan Test | V5 Test |
|-----------|---------|
| "it returns open task blocked by open/in_progress dep" | Combined into ONE subtest under `TestBlockedQuery` |
| "it returns parent with open/in_progress children" | Combined into ONE subtest under `TestBlockedQuery` |
| "it excludes task when all blockers done/cancelled" | Subtest under `TestBlockedQuery` |
| "it excludes in_progress/done/cancelled from output" | Subtest under `TestBlockedQuery` |
| "it returns empty when no blocked tasks" | Subtest under `TestBlockedQuery` |
| "it orders by priority ASC then created ASC" | Subtest under `TestBlockedQuery` |
| "it outputs aligned columns via tick blocked" | Subtest under `TestBlockedCommand` |
| "it prints 'No tasks found.' when empty" | Subtest under `TestBlockedCommand` |
| "it outputs IDs only with --quiet" | Subtest under `TestBlockedCommand` |
| "cancel unblocks single dependent" | Subtest under `TestCancelUnblocksDependents` |
| "cancel unblocks multiple dependents" | Subtest under `TestCancelUnblocksDependents` |
| "cancel does not unblock dependent still blocked by another" | Subtest under `TestCancelUnblocksDependents` |

**Test structure**: V5 groups related tests under logical parent functions with `t.Run` subtests, which is better organization. The grouping is semantically meaningful (query logic vs command output vs cancel behavior).

**Test setup**: Uses `task.NewTask("tick-aaaaaa", "Open blocker")` constructor and `initTickProjectWithTasks(t, tasks)`. The `NewTask` constructor handles default field initialization (status, priority, created, updated), reducing boilerplate. Tasks are created with mutations:

```go
ipBlocker := task.NewTask("tick-cccccc", "IP blocker")
ipBlocker.Status = task.StatusInProgress
```

**Combined tests**: V5 combines the "blocked by open dep" and "blocked by in_progress dep" scenarios into a single subtest with 4 tasks (2 blocking pairs). Similarly, "parent with open children" and "parent with in_progress children" are combined. This tests both scenarios in one pass, verifying 4 assertions instead of 2. This is arguably better coverage density but slightly harder to isolate failures.

**Assertion style**: Same `strings.Contains` pattern as V4.

### Spec Compliance

- **blocked = open AND NOT ready**: Implemented as direct positive expression. Logically equivalent to the spec, but does NOT reuse ready query logic as the plan explicitly states: "Simplest: blocked = open minus ready (reuse `ReadyQuery`)" and "Reuses ready query logic" is an acceptance criterion.
- **Only open in output**: `WHERE t.status = 'open'` present. Compliant.
- **Cancel unblocks dependent**: Tested end-to-end with 3 scenarios. Compliant.
- **Deterministic ordering**: `ORDER BY t.priority ASC, t.created ASC`. Compliant.
- **Aligned columns**: `printListTable` uses same `%-12s` formatting. Compliant.
- **Empty state**: "No tasks found." with exit 0. Compliant.
- **--quiet**: IDs only. Compliant.
- **Register blocked subcommand**: Added to `commands` map. Compliant.
- **Reuses ready query logic**: **NOT COMPLIANT** at commit time. The plan's acceptance criterion "Reuses ready query logic" is not met. The blocked query is independently written SQL. (This was later fixed in commit 41e0947 which extracted shared WHERE clauses.)

### golang-pro Skill Compliance

- **Error handling**: All errors handled with `%w` wrapping. Compliant.
- **Table-driven tests**: NOT used. Tests use `t.Run` subtests within logical groups, but none are table-driven. Same deviation as V4.
- **Exported function documentation**: `BlockedQuery` is exported and documented. `runBlocked` is unexported and documented.
- **No panics**: Compliant.
- **No ignored errors**: Compliant.

## Comparative Analysis

### Where V4 is Better

1. **Spec compliance on reuse**: V4 directly reuses `readyConditions` from `ready.go` to compose the blocked query. The plan explicitly says "Simplest: blocked = open minus ready (reuse `ReadyQuery`)" and lists "Reuses ready query logic" as an acceptance criterion. V4 satisfies this; V5 does not (at commit time). This is the single most significant difference.

2. **DRY rendering**: V4 delegates rendering to `renderListOutput` (a shared function), avoiding code duplication between `runBlocked`, `runReady`, and `runList`. V5 inlines the empty/quiet/table logic in each command function, creating 3 copies of the same pattern. (V5 later refactored this away, but at commit time it was duplicated.)

3. **More granular test cases**: V4 splits "blocked by open dep" and "blocked by in_progress dep" into separate tests. This makes it easier to identify exactly which scenario fails. V5 combines them, so a failure in the combined test requires more investigation.

4. **Subtlety of NOT IN approach**: V4's `NOT IN (ready set)` approach is a direct translation of the specification ("blocked = open AND NOT ready"). It is easier to audit against the spec because the relationship is structural, not just a hope that the two independently-written queries are logical inverses.

### Where V5 is Better

1. **Test organization**: V5 groups tests into 3 logical parent functions (`TestBlockedQuery`, `TestBlockedCommand`, `TestCancelUnblocksDependents`). This is cleaner than V4's 12 standalone `TestBlocked_Xxx` functions. The grouping communicates intent: these are query tests, these are output tests, these are integration tests.

2. **Test setup brevity**: V5 uses `task.NewTask()` constructor and `initTickProjectWithTasks()`, reducing boilerplate. V4 manually constructs `task.Task{}` literals with all fields specified, which is verbose but explicit.

3. **Command registration**: V5 uses a `commands` map for dispatch, which is more maintainable and idiomatic than V4's large `switch` statement. Adding a new command in V5 is a single map entry; in V4 it requires a new `case` block.

4. **SQL clarity**: V5's blocked query uses direct positive conditions (`EXISTS ... unclosed blocker OR EXISTS ... open children`), which is arguably more readable than V4's double-negation (`NOT IN (... NOT EXISTS ... AND NOT EXISTS ...)`). The V4 query requires reasoning about "not in the set of things that do not have these conditions" which is harder to parse mentally.

5. **Unexported listRow fields**: V5 uses lowercase field names (`id`, `status`, `priority`, `title`) for the internal `listRow` struct. This is more idiomatic Go -- unexported fields for package-internal types.

6. **Combined test coverage density**: V5's combined "open/in_progress dep" test creates 4 tasks and validates all 4 in one subtest, testing the interaction between both blocking types simultaneously. This is slightly higher value per test.

### Differences That Are Neutral

1. **Error message prefix style**: V4 uses "failed to query blocked tasks:" while V5 uses "querying blocked tasks:". Both are acceptable Go error wrapping styles.

2. **Query declared as `var` vs `const`**: V4 uses `var blockedQuery` (because of string concatenation with `readyConditions`). V5 uses `const BlockedQuery` (standalone string). Both are correct for their approach.

3. **Exported vs unexported query name**: V4's `blockedQuery` is unexported; V5's `BlockedQuery` is exported. In an internal package this distinction is minimal, though V5's export is unnecessary since the query is only used within the `cli` package. (V5 later uses the exported query in list.go's `buildBlockedFilterQuery`, which wraps it in a subquery, so the export becomes justified in later commits.)

4. **Test line counts**: V4 has ~493 test lines (14 subtests); V5 has ~431 test lines (12 subtests). The difference is explained by V4's split of combined scenarios and slightly more verbose setup.

5. **App method vs free function**: V4's `(a *App) runBlocked(args []string)` vs V5's `runBlocked(ctx *Context)`. This is an architectural choice made at the project level, not specific to this task.

## Verdict

**Winner: V4**

The decisive factor is **spec compliance on the "reuse ready query logic" requirement**. The task plan explicitly states:

> "Simplest: blocked = open minus ready (reuse `ReadyQuery`)"

And the acceptance criteria include:

> "Reuses ready query logic"

V4 satisfies this by extracting `readyConditions` and composing `blockedQuery` as `open AND NOT IN (ready set)`. This is a direct, structural implementation of the specification. If the ready conditions change (e.g., a new blocking condition is added), V4's blocked query automatically picks up the change.

V5 at commit c66ead3 writes the blocked query as independent, standalone SQL. While logically equivalent, this means ready and blocked conditions can drift apart if one is modified without the other. V5 recognized this problem and fixed it in a later commit (41e0947), but at the time of this task's implementation, the acceptance criterion was unmet.

Beyond spec compliance, V4 also has better DRY adherence for rendering logic (shared `renderListOutput` vs V5's 3 inline copies) and more granular test isolation (separate open vs in_progress blocker tests).

V5 has genuine advantages in test organization, command dispatch architecture, and SQL readability, but these are secondary to the spec compliance gap. The fact that V5 needed a follow-up commit to fix the shared SQL problem confirms that V4's approach was architecturally sounder for this specific task.
