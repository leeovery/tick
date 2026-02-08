# Task tick-core-3-4: Blocked Query & tick blocked, Cancel-Unblocks-Dependents

## Task Summary

Implement the inverse of the ready query: `BlockedQuery` returns open tasks that cannot be worked because they have unclosed blockers OR open children. Register `tick blocked` as an alias for `list --blocked`. Verify that cancelling a blocker correctly unblocks dependents end-to-end. Only `open` tasks appear in output. Order: priority ASC, created ASC.

### Acceptance Criteria (from plan)

1. Returns open tasks with unclosed blockers or open children
2. Excludes tasks where all blockers closed
3. Only `open` in output
4. Cancel -> dependent unblocks
5. Multiple dependents unblock simultaneously
6. Partial unblock works correctly
7. Deterministic ordering
8. `tick blocked` outputs aligned columns
9. Empty -> `No tasks found.`, exit 0
10. `--quiet` IDs only
11. Reuses ready query logic

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Returns open tasks with unclosed blockers or open children | PASS -- `BlockedSQL` uses `id IN (SELECT d.task_id ... WHERE t.status NOT IN ('done','cancelled'))` and `id IN (SELECT parent ... WHERE status IN ('open','in_progress'))`. Tests for open blocker, in_progress blocker, open children, in_progress children all present. | PASS -- `blockedQuery` uses `t.id NOT IN (SELECT t.id FROM tasks t WHERE t.status = 'open' AND <readyConditions>)`. Same scenarios tested. |
| Excludes tasks where all blockers closed | PASS -- Test `"it excludes task when all blockers done or cancelled"` verifies done+cancelled blockers are not flagged. | PASS -- Same test name, same verification. |
| Only `open` in output | PASS -- Two tests: one for in_progress exclusion, one for done/cancelled exclusion. | PASS -- Single test `TestBlocked_ExcludesNonOpenStatuses` covers in_progress, done, and cancelled in one scenario. |
| Cancel -> dependent unblocks | PASS -- `TestCancelUnblocksDependents/"cancel unblocks single dependent"` runs blocked->cancel->ready->blocked sequence. | PASS -- `TestBlocked_CancelUnblocksSingleDependent` does same 4-step verification. |
| Multiple dependents unblock simultaneously | PASS -- `TestCancelUnblocksDependents/"cancel unblocks multiple dependents"` with two dependents. | PASS -- `TestBlocked_CancelUnblocksMultipleDependents` with two dependents, plus verifies both blocked before cancel. |
| Partial unblock works correctly | PASS -- `TestCancelUnblocksDependents/"cancel does not unblock dependent still blocked by another"` tests two blockers, cancels one, verifies still blocked and not ready. | PASS -- `TestBlocked_CancelDoesNotUnblockDependentStillBlockedByAnother` with same logic plus explicit pre-cancel blocked verification. |
| Deterministic ordering | PASS -- Test creates 4 blocked tasks with priorities 1,1,2,3 and verifies exact row ordering. | PASS -- Same test structure, same assertions. |
| `tick blocked` outputs aligned columns | PASS -- Checks header starts with "ID", contains "STATUS", "PRI", "TITLE", and checks column alignment positions. | PASS -- Checks same header fields. Also verifies first data row content (ID, status, title). Does not verify position alignment. |
| Empty -> `No tasks found.`, exit 0 | PASS -- Two tests: one with ready-only tasks, one with empty DB. V2 returns `error` so exit 0 verified via `err == nil`. | PASS -- Two tests: same scenarios. V4 returns `int` exit code, explicitly checks `code != 0`. |
| `--quiet` IDs only | PASS -- Verifies 2 IDs in order, no header. | PASS -- Same verification. |
| Reuses ready query logic | FAIL -- `BlockedSQL` is a separate `const` with manually duplicated inverse conditions. No shared logic with `ReadySQL`. | PASS -- Extracts `readyConditions` const shared between `readyQuery` and `blockedQuery`. The blocked query is literally `NOT IN (SELECT ... WHERE status='open' AND <readyConditions>)`. |

## Implementation Comparison

### Approach

**V2** adds all blocked-query logic into the existing `list.go` file. It defines a new `BlockedSQL` constant alongside the existing `ReadySQL`, extends the `listFlags` struct with a `blocked` field, and conditionally selects the query in `runList`. The `tick blocked` subcommand is wired in `app.go` as `return a.runList([]string{"--blocked"})`.

Key V2 code in `list.go`:
```go
const BlockedSQL = `SELECT id, status, priority, title FROM tasks
WHERE status = 'open'
  AND (
    id IN (
      SELECT d.task_id FROM dependencies d
      JOIN tasks t ON d.blocked_by = t.id
      WHERE t.status NOT IN ('done', 'cancelled')
    )
    OR id IN (
      SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
    )
  )
ORDER BY priority ASC, created ASC`
```

The blocked query in V2 is a manually-written inverse. The `ReadySQL` uses `id NOT IN (...)` patterns, while `BlockedSQL` uses `id IN (...)` with the same subqueries. This is logically equivalent to "open AND NOT ready" but the two SQL constants are independently maintained -- there is no shared fragment. If the ready conditions ever change, `BlockedSQL` must be updated separately.

The dispatch in `app.go`:
```go
case "blocked":
    return a.runList([]string{"--blocked"})
```

**V4** takes a fundamentally different architectural approach. It creates a dedicated `blocked.go` file with its own `runBlocked` method, and refactors `ready.go` to extract a shared `readyConditions` constant. The blocked query is derived from the ready conditions via set subtraction.

Key V4 code in `ready.go`:
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

var readyQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND` + readyConditions + `
ORDER BY t.priority ASC, t.created ASC
`
```

Key V4 code in `blocked.go`:
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

V4's `runBlocked` is a standalone method that handles its own store open/close and calls the shared `renderListOutput` function. The dispatch in `cli.go`:
```go
case "blocked":
    if err := a.runBlocked(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

**Key architectural difference**: V4 genuinely reuses the ready logic via `readyConditions`, fulfilling the acceptance criterion "Reuses ready query logic." V2 duplicates the logic manually. This is a genuinely better approach in V4, not merely a different one.

**SQL approach difference**: V2 uses `IN` subqueries (correlated subselects returning task_ids), while V4's ready conditions use `NOT EXISTS` (existence checks). The `NOT EXISTS` pattern in V4 is more idiomatic SQL for this type of check and can be more efficient on large datasets, though for typical tick usage the difference is negligible.

**Routing difference**: V2 routes `tick blocked` through `runList` via flag injection (`[]string{"--blocked"}`). V4 routes it to a standalone `runBlocked` method. V2's approach means there is no `list --blocked` separate from `tick blocked` since both go through the same codepath, while V4 similarly doesn't implement a `--blocked` flag on `list` (it goes directly to `runBlocked`). V2 actually also adds `--blocked` as a parseable flag in `parseListFlags`, meaning `tick list --blocked` works. V4 does NOT support `tick list --blocked`.

### Code Quality

**DRY principle**: V4 is significantly better. V2 has two independently maintained SQL constants (`ReadySQL` and `BlockedSQL`) that must stay in sync. V4 has one shared `readyConditions` constant composed into both queries.

**File organization**: V4 is better. V2 stuffs everything into `list.go`, making it a grab-bag of list, ready, and blocked logic. V4 separates concerns: `ready.go` owns ready conditions and the ready query, `blocked.go` owns the blocked query, `list.go` owns the generic list rendering.

**Naming**: Both are clear. V2 uses exported `BlockedSQL` and `ReadySQL` constants. V4 uses unexported `blockedQuery` and `readyQuery` variables. V4's use of `var` (instead of `const`) is necessitated by Go's inability to concatenate string constants at init time with `+` in `const` blocks -- this is the correct pattern.

**Error handling**: Both wrap SQL errors appropriately. V2:
```go
return fmt.Errorf("failed to query tasks: %w", err)
```
V4:
```go
return fmt.Errorf("failed to query blocked tasks: %w", err)
```
V4's error message is more specific ("blocked tasks" vs generic "tasks").

**Type safety**: V2 defines `listRow` as a local type inside `runList`. V4 defines `listRow` at package level in `list.go` and reuses it in `ready.go` and `blocked.go`. V4's approach is better for code sharing.

**`list --blocked` support**: V2 supports `tick list --blocked` as a first-class flag. V4 does not -- `tick blocked` goes directly to `runBlocked`. The task spec says `tick blocked = list --blocked`, and V2 adheres to this more literally. However, V4's `tick blocked` command works correctly for the user-facing behavior.

### Test Quality

**V2 test functions (3 top-level, 16 subtests)**:

Top-level:
1. `TestBlockedQuery` -- 12 subtests covering the blocked query behavior
2. `TestBlockedViaListFlag` -- 1 subtest for `list --blocked`
3. `TestCancelUnblocksDependents` -- 3 subtests for cancel-unblocks behavior

Subtests in `TestBlockedQuery`:
- `"it returns open task blocked by open dep"` -- open blocker keeps task blocked, blocker itself excluded
- `"it returns open task blocked by in_progress dep"` -- in_progress blocker keeps task blocked
- `"it returns parent with open children"` -- parent blocked by open child, child excluded
- `"it returns parent with in_progress children"` -- parent blocked by in_progress child
- `"it excludes task when all blockers done or cancelled"` -- done+cancelled blockers don't block
- `"it excludes in_progress tasks from output"` -- non-open status excluded
- `"it excludes done and cancelled from output"` -- non-open status excluded (separate test)
- `"it returns empty when no blocked tasks"` -- ready-only tasks yield "No tasks found."
- `"it orders by priority ASC then created ASC"` -- 4 tasks sorted correctly
- `"it outputs aligned columns via tick blocked"` -- header + column alignment check
- `"it prints 'No tasks found.' when empty"` -- empty DB yields message
- `"it outputs IDs only with --quiet"` -- quiet mode, 2 IDs in order

Subtests in `TestBlockedViaListFlag`:
- `"it works via list --blocked flag"` -- end-to-end via `list --blocked`

Subtests in `TestCancelUnblocksDependents`:
- `"cancel unblocks single dependent -- moves to ready"` -- 4-step: blocked->cancel->ready->not-blocked
- `"cancel unblocks multiple dependents"` -- 2 dependents freed by single cancel
- `"cancel does not unblock dependent still blocked by another"` -- partial unblock, 3-step

**V4 test functions (14 top-level, 14 subtests)**:

1. `TestBlocked_ReturnsOpenTaskBlockedByOpenDep` -- open blocker, blocker excluded
2. `TestBlocked_ReturnsOpenTaskBlockedByInProgressDep` -- in_progress blocker
3. `TestBlocked_ReturnsParentWithOpenChildren` -- parent blocked, child excluded
4. `TestBlocked_ReturnsParentWithInProgressChildren` -- parent blocked by in_progress child
5. `TestBlocked_ExcludesTaskWhenAllBlockersDoneOrCancelled` -- done+cancelled blockers
6. `TestBlocked_ExcludesNonOpenStatuses` -- in_progress, done, cancelled all in one test
7. `TestBlocked_EmptyWhenNoBlockedTasks` -- ready-only tasks
8. `TestBlocked_OrderByPriorityThenCreated` -- 4-task ordering
9. `TestBlocked_AlignedColumnsOutput` -- header + row content
10. `TestBlocked_NoTasksFoundMessage` -- empty DB
11. `TestBlocked_QuietOutputsIDsOnly` -- quiet mode
12. `TestBlocked_CancelUnblocksSingleDependent` -- 4-step cancel-unblock with pre-verification
13. `TestBlocked_CancelUnblocksMultipleDependents` -- 2 dependents, pre-verification of blocked state
14. `TestBlocked_CancelDoesNotUnblockDependentStillBlockedByAnother` -- 4-step partial unblock with pre-verification

**Test structure difference**: V2 uses grouped subtests under 3 top-level functions; V4 uses 14 independent top-level test functions each containing exactly one `t.Run` subtest. V4's approach is more verbose but gives clearer test isolation and naming in `go test -v` output.

**Test setup difference**: V2 uses `taskJSONL()` helper to build raw JSONL strings and `setupTickDirWithContent()`. V4 uses typed `task.Task` structs with `setupInitializedDirWithTasks()`, which provides type safety (compile-time checks on field names) and is more idiomatic Go.

**Unique V2 test**: `TestBlockedViaListFlag/"it works via list --blocked flag"` -- tests `tick list --blocked` as a distinct invocation path. V4 does not have this test because V4 does not implement `list --blocked`.

**Unique V4 approach**: V4's cancel-unblock tests all include a pre-verification step (checking the task is actually blocked before cancelling), making the tests more thorough. V2's `"cancel unblocks single dependent"` does include pre-verification, but the `"cancel unblocks multiple dependents"` test in V2 skips it (goes straight to cancel).

**Assertion quality**: V2's column alignment test checks actual character positions:
```go
headerStatusPos := strings.Index(header, "STATUS")
row1StatusPos := strings.Index(lines[1], "open")
if headerStatusPos != row1StatusPos ...
```
V4's alignment test checks header fields and row content but does NOT verify positional alignment. V2's assertion is stronger here.

**V2 coverage gap**: None -- V2 covers all spec'd tests plus an extra `list --blocked` test.

**V4 coverage gap**: No `list --blocked` test. V4 also combines the "excludes in_progress" and "excludes done/cancelled" into a single test (`TestBlocked_ExcludesNonOpenStatuses`), which is fine but provides slightly less granularity in failure diagnosis.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 3 (app.go, list.go, blocked_test.go) | 4 (cli.go, ready.go, blocked.go, blocked_test.go) |
| Lines added | 562 | 582 |
| Impl LOC (new/changed) | ~38 (36 in list.go + 2 in app.go) | ~70 (64 in blocked.go + ~6 in cli.go) |
| Impl LOC (ready.go refactor) | 0 | ~19 net (30 added - 11 removed) |
| Test LOC | 524 | 493 |
| Top-level test functions | 3 | 14 |
| Subtests | 16 | 14 |

## Verdict

**V4 is the better implementation**, primarily due to one critical difference: it genuinely reuses the ready query logic by extracting `readyConditions` as a shared constant, while V2 manually duplicates the inverse SQL conditions. This directly addresses acceptance criterion #11 ("Reuses ready query logic"), which V2 fails.

V4's approach of defining `blocked = open AND NOT IN ready_set` is mathematically correct and self-maintaining: any future change to what makes a task "ready" will automatically propagate to the blocked query. V2's manual inversion (`id IN` vs `id NOT IN`) requires synchronized updates to two separate SQL constants, which is error-prone.

V4 also demonstrates better code organization (separate `blocked.go` file), better type safety in tests (typed `task.Task` structs vs raw JSONL strings), and more specific error messages.

V2 has two advantages: (1) it supports `tick list --blocked` as a flag (matching the spec's statement that `tick blocked = list --blocked`) and tests it, and (2) its column alignment test is stronger, actually verifying character positions. However, these are minor compared to V4's architectural advantage of genuine logic reuse.
