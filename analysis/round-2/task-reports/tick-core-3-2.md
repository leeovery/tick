# Task tick-core-3-2: tick dep add & tick dep rm Commands

## Task Summary

Wire up CLI commands for managing task dependencies post-creation. `tick dep add <task_id> <blocked_by_id>` adds a dependency (meaning task_id is blocked by blocked_by_id), and `tick dep rm <task_id> <blocked_by_id>` removes one.

Requirements:
- Register `dep` with sub-subcommands `add` and `rm`
- Both: two positional IDs (task first, dependency second), normalize to lowercase
- **add**: look up both IDs, check self-ref, check duplicate, call `ValidateDependency`, add to `blocked_by`, update timestamp. Output: `Dependency added: {task_id} blocked by {blocked_by_id}`
- **rm**: look up task_id, check blocked_by_id in array (NOT as a task -- supports stale refs), remove from `blocked_by`, update timestamp. Output: `Dependency removed: {task_id} no longer blocked by {blocked_by_id}`
- `--quiet` suppresses output; errors to stderr, exit 1

### Acceptance Criteria

1. `dep add` adds dependency and outputs confirmation
2. `dep rm` removes dependency and outputs confirmation
3. Non-existent IDs return error
4. Duplicate/missing dep return error
5. Self-ref, cycle, child-blocked-by-parent return error
6. IDs normalized to lowercase
7. `--quiet` suppresses output
8. `updated` timestamp refreshed
9. Persisted through storage engine

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| `dep add` adds dependency and outputs confirmation | PASS -- test `it adds a dependency between two existing tasks` verifies blocked_by array; `it outputs confirmation on success (add)` verifies exact message | PASS -- `TestDepAdd_AddsDependencyBetweenTwoExistingTasks` verifies blocked_by array; `TestDepAdd_OutputsConfirmation` verifies exact message |
| `dep rm` removes dependency and outputs confirmation | PASS -- test `it removes an existing dependency` verifies empty blocked_by; `it outputs confirmation on success (rm)` verifies exact message | PASS -- `TestDepRm_RemovesExistingDependency` verifies empty blocked_by; `TestDepRm_OutputsConfirmation` verifies exact message |
| Non-existent IDs return error | PASS -- tests for task_id not found (add/rm) and blocked_by_id not found (add) | PASS -- separate test functions for each: `TestDepAdd_ErrorTaskIDNotFound`, `TestDepRm_ErrorTaskIDNotFound`, `TestDepAdd_ErrorBlockedByIDNotFound` |
| Duplicate/missing dep return error | PASS -- `it errors on duplicate dependency (add)` verifies error + no mutation; `it errors when dependency not found in blocked_by (rm)` | PASS -- `TestDepAdd_ErrorDuplicateDependency` verifies error + no mutation; `TestDepRm_ErrorDependencyNotFound` |
| Self-ref, cycle, child-blocked-by-parent return error | PASS -- separate tests for self-ref, cycle, and child-blocked-by-parent | PASS -- `TestDepAdd_ErrorSelfReference`, `TestDepAdd_ErrorCycle`, `TestDepAdd_ErrorChildBlockedByParent` |
| IDs normalized to lowercase | PASS -- `it normalizes IDs to lowercase (add)` and `(rm)` test uppercase input | PASS -- `TestDep_NormalizesIDsToLowercase` with subtests for add and rm |
| `--quiet` suppresses output | PASS -- `it suppresses output with --quiet (add)` and `(rm)` | PASS -- `TestDep_QuietSuppressesOutput` with subtests for add and rm |
| `updated` timestamp refreshed | PASS -- `it updates task's updated timestamp on add` and `on rm` | PASS -- `TestDepAdd_UpdatesTimestamp` and `TestDepRm_UpdatesTimestamp` |
| Persisted through storage engine | PASS -- `it persists via atomic write (add)` reads raw file; `(rm)` verifies blocked_by removed from file | PASS -- `TestDep_PersistsViaAtomicWrite` reads raw JSONL, parses JSON, and also checks cache.db existence |

## Implementation Comparison

### Approach

Both versions follow essentially the same structural approach: a `runDep` dispatcher function that routes to `runDepAdd` and `runDepRm` handler functions, each validating arguments, opening storage, calling `store.Mutate()`, and printing output. The differences are in architectural details related to how the broader CLI app is structured (which comes from prior tasks, not this one).

**CLI Routing Integration**

V2 adds the `dep` case to `app.go`'s `Run` method, which returns `error`:
```go
// V2 internal/cli/app.go
case "dep":
    return a.runDep(cmdArgs)
```

V4 adds it to `cli.go`'s `Run` method, which returns `int` (exit code), with explicit error-to-stderr writing:
```go
// V4 internal/cli/cli.go
case "dep":
    if err := a.runDep(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

This is not a task-specific difference -- it reflects the different App architectures from earlier tasks. V2's `App.Run` returns `error` (with error-to-exit-code conversion happening elsewhere), while V4's `App.Run` returns `int` directly and writes errors to stderr inline.

**Self-Reference Check Placement**

V4 performs the self-reference check *before* opening the store:
```go
// V4 dep.go, runDepAdd, lines 43-46
if taskID == blockedByID {
    return fmt.Errorf("Cannot add dependency - creates cycle: %s -> %s", taskID, taskID)
}
```

V2 lets the self-reference check happen inside `ValidateDependency` within the `Mutate` callback. This means V2 opens the store, acquires the lock, and enters the mutation before detecting a trivially invalid self-reference. V4's early-return is a minor optimization that avoids unnecessary I/O for obvious failures.

**ID Comparison in Mutate Callbacks**

V2 normalizes stored IDs during comparison inside the Mutate callback:
```go
// V2 dep.go, line 56
normalizedID := task.NormalizeID(tasks[i].ID)
if normalizedID == taskID {
```

V4 compares directly without normalizing stored IDs:
```go
// V4 dep.go, line 65
if tasks[i].ID == taskID {
```

V4 assumes IDs are already stored in normalized (lowercase) form, which is the expected invariant. V2 is more defensive by normalizing at comparison time, which protects against corrupted data but adds unnecessary overhead if the invariant holds.

**Duplicate Check in V2 Also Normalizes**

V2 normalizes the existing blocked_by entries during duplicate detection:
```go
// V2 dep.go, line 72
if task.NormalizeID(existing) == blockedByID {
```

V4 compares directly:
```go
// V4 dep.go, line 80
if dep == blockedByID {
```

Same pattern -- V2 is more defensively coded, V4 trusts stored data.

**Dependency Removal Algorithm**

V2 builds a new slice excluding the target:
```go
// V2 dep.go, lines 140-148
newBlockedBy := make([]string, 0, len(tasks[taskIdx].BlockedBy))
for _, dep := range tasks[taskIdx].BlockedBy {
    if task.NormalizeID(dep) == blockedByID {
        found = true
        continue
    }
    newBlockedBy = append(newBlockedBy, dep)
}
tasks[taskIdx].BlockedBy = newBlockedBy
```

V4 finds the index and uses slice append trick:
```go
// V4 dep.go, lines 149-157
depIdx := -1
for i, dep := range tasks[taskIdx].BlockedBy {
    if dep == blockedByID {
        depIdx = i
        break
    }
}
tasks[taskIdx].BlockedBy = append(
    tasks[taskIdx].BlockedBy[:depIdx],
    tasks[taskIdx].BlockedBy[depIdx+1:]...,
)
```

V2's filter approach is allocation-heavy (new slice every time) but handles the theoretical case where the same dependency appears multiple times (it would remove all instances). V4's index-based splice is more standard Go but modifies the underlying array in-place, which is fine since there should only be one matching entry.

**Input Trimming**

V4 applies `strings.TrimSpace` to args before normalizing:
```go
// V4 dep.go, lines 41-42
taskID := task.NormalizeID(strings.TrimSpace(args[0]))
blockedByID := task.NormalizeID(strings.TrimSpace(args[1]))
```

V2 does not trim whitespace:
```go
// V2 dep.go, lines 40-41
taskID := task.NormalizeID(args[0])
blockedByID := task.NormalizeID(args[1])
```

V4 is slightly more robust against edge cases where shell quoting introduces whitespace, though in practice CLI args rarely have leading/trailing whitespace.

**Error Unwrapping**

V2 calls `unwrapMutationError(err)` after `store.Mutate` returns:
```go
// V2 dep.go, line 88
return unwrapMutationError(err)
```

V4 returns the error directly:
```go
// V4 dep.go, line 91
return err
```

This difference reflects underlying storage engine differences from prior tasks. V2's storage apparently wraps mutation errors in a container type that needs unwrapping; V4's does not.

**App Field Access**

V2 accesses configuration through nested fields: `a.config.Quiet`, `a.stdout`, `a.workDir`.
V4 uses flat exported fields: `a.Quiet`, `a.Stdout`, `a.Dir`.

### Code Quality

**Naming Conventions**

Both versions use clear, descriptive function names (`runDep`, `runDepAdd`, `runDepRm`). V4 names its local variable `subcommand` (in `runDep`) while V2 uses `subcmd` -- V4 is slightly more readable. Both use `taskIdx` and `blockedByID` consistently.

**Error Messages**

Both produce well-structured error messages. V2's dispatcher error:
```go
// V2
"Usage: tick dep <add|rm> <task_id> <blocked_by_id>"
```
V4's dispatcher error:
```go
// V4
"Subcommand required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>"
```
V4 is slightly more informative by stating what's missing before the usage hint.

**Imports**

V2 imports 4 packages: `fmt`, `time`, `storage`, `task`.
V4 imports 5 packages: `fmt`, `strings`, `time`, `store`, `task` (extra `strings` for `TrimSpace`).

The different storage package names (`storage` vs `store`) reflect different naming choices from prior tasks.

**DRY Principle**

Both versions have moderate code duplication between `runDepAdd` and `runDepRm` (discovery, store opening, task lookup). Neither version extracts shared helper functions for common operations like "find task by ID in slice." This is equivalent between versions -- the duplication is acceptable given the small scope.

**Comment Quality**

V4 has more detailed doc comments on `runDepRm`:
```go
// V4
// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
// It looks up task_id, checks blocked_by_id in array, removes it, and persists.
// Note: rm does NOT validate that blocked_by_id exists as a task -- only checks
// array membership (supports removing stale refs).
```
V2's is briefer:
```go
// V2
// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
```

V4's comment explicitly documents the stale-ref edge case design decision.

### Test Quality

**V2 Test Functions (18 subtests across 3 top-level functions)**

`TestDepAddCommand` (12 subtests):
1. `it adds a dependency between two existing tasks` -- verifies blocked_by array content
2. `it outputs confirmation on success (add)` -- exact message match
3. `it updates task's updated timestamp on add` -- checks timestamp changed
4. `it errors when task_id not found (add)` -- error contains task ID
5. `it errors when blocked_by_id not found (add)` -- error contains blocked_by ID
6. `it errors on duplicate dependency (add)` -- error contains "already" + no mutation check
7. `it errors on self-reference (add)` -- error contains "cycle"
8. `it errors when add creates cycle` -- A blocked by B, adding B blocked by A
9. `it errors when add creates child-blocked-by-parent` -- error contains "parent"
10. `it normalizes IDs to lowercase (add)` -- uppercase input, verifies lowercase in blocked_by and output
11. `it suppresses output with --quiet (add)` -- empty stdout
12. `it errors when fewer than two IDs provided (add)` -- tests zero args AND one arg
13. `it persists via atomic write (add)` -- reads raw file, checks for dependency string

`TestDepRmCommand` (9 subtests):
1. `it removes an existing dependency` -- verifies empty blocked_by
2. `it outputs confirmation on success (rm)` -- exact message match
3. `it updates task's updated timestamp on rm` -- checks timestamp changed
4. `it errors when task_id not found (rm)` -- error contains task ID
5. `it errors when dependency not found in blocked_by (rm)` -- error contains blocked_by ID
6. `it does not validate blocked_by_id exists as a task on rm (supports stale refs)` -- **unique to V2**
7. `it normalizes IDs to lowercase (rm)` -- uppercase input, verifies removal and output
8. `it suppresses output with --quiet (rm)` -- empty stdout
9. `it errors when fewer than two IDs provided (rm)` -- tests zero args AND one arg
10. `it persists via atomic write (rm)` -- reads raw file, verifies blocked_by absent and no temp files

`TestDepSubcommandRouting` (2 subtests):
1. `it errors for dep with no subcommand` -- error contains "Usage"
2. `it errors for dep with unknown subcommand` -- error contains "unknown"

**V4 Test Functions (22 subtests across 14 top-level functions)**

`TestDepAdd_AddsDependencyBetweenTwoExistingTasks` (1 subtest):
1. `it adds a dependency between two existing tasks`

`TestDepRm_RemovesExistingDependency` (1 subtest):
1. `it removes an existing dependency`

`TestDepAdd_OutputsConfirmation` (1 subtest):
1. `it outputs confirmation on success (add)`

`TestDepRm_OutputsConfirmation` (1 subtest):
1. `it outputs confirmation on success (rm)`

`TestDepAdd_UpdatesTimestamp` (1 subtest):
1. `it updates task's updated timestamp on dep add`

`TestDepRm_UpdatesTimestamp` (1 subtest):
1. `it updates task's updated timestamp on dep rm`

`TestDepAdd_ErrorTaskIDNotFound` (1 subtest):
1. `it errors when task_id not found (add)`

`TestDepRm_ErrorTaskIDNotFound` (1 subtest):
1. `it errors when task_id not found (rm)`

`TestDepAdd_ErrorBlockedByIDNotFound` (1 subtest):
1. `it errors when blocked_by_id not found (add)`

`TestDepAdd_ErrorDuplicateDependency` (1 subtest):
1. `it errors on duplicate dependency (add)` -- includes no-mutation verification

`TestDepRm_ErrorDependencyNotFound` (1 subtest):
1. `it errors when dependency not found (rm)`

`TestDepAdd_ErrorSelfReference` (1 subtest):
1. `it errors on self-reference (add)`

`TestDepAdd_ErrorCycle` (1 subtest):
1. `it errors when add creates cycle`

`TestDepAdd_ErrorChildBlockedByParent` (1 subtest):
1. `it errors when add creates child-blocked-by-parent`

`TestDep_NormalizesIDsToLowercase` (2 subtests):
1. `it normalizes IDs to lowercase (add)`
2. `it normalizes IDs to lowercase (rm)`

`TestDep_QuietSuppressesOutput` (2 subtests):
1. `it suppresses output with --quiet (add)`
2. `it suppresses output with --quiet (rm)`

`TestDep_ErrorFewerThanTwoIDs` (4 subtests):
1. `it errors when fewer than two IDs provided (add, zero args)`
2. `it errors when fewer than two IDs provided (add, one arg)`
3. `it errors when fewer than two IDs provided (rm, zero args)`
4. `it errors when no subcommand provided to dep`

`TestDep_PersistsViaAtomicWrite` (1 subtest):
1. `it persists via atomic write (add)` -- parses raw JSONL as JSON, also checks cache.db

**Test Coverage Gaps**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Stale ref removal (rm for non-existent task) | TESTED -- dedicated subtest | MISSING -- no test for this spec requirement |
| Atomic write persistence (rm) | TESTED -- `it persists via atomic write (rm)` | MISSING -- only tests add persistence, not rm |
| Fewer than two IDs (rm, one arg) | TESTED -- in combined subtest | MISSING -- only tests rm with zero args, not one arg |
| Unknown dep subcommand | TESTED -- `it errors for dep with unknown subcommand` | MISSING -- not tested |

V2 has **23 total subtests** (some subtests test multiple cases like zero and one arg). V4 has **22 subtests** but is missing 4 edge cases that V2 covers.

**Test Setup Patterns**

V2 uses JSONL string helpers:
```go
// V2 dep_test.go, lines 12-21
func twoOpenTasksJSONL() string {
    return openTaskJSONL("tick-aaa111") + "\n" + openTaskJSONL("tick-bbb222") + "\n"
}
func openTaskWithBlockedByJSONL(id, blockedByID string) string {
    return `{"id":"` + id + `",...,"blocked_by":["` + blockedByID + `"],...}`
}
```

V4 uses typed task struct construction:
```go
// V4 dep_test.go
existing := []task.Task{
    {ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
    {ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
}
dir := setupInitializedDirWithTasks(t, existing)
```

V4's approach is **significantly better** -- it uses typed Go structs, which means the compiler catches field name errors, refactors propagate automatically, and the setup is self-documenting. V2's raw JSONL string construction is fragile -- a typo in a JSON field name would only manifest at runtime.

**Assertion Quality**

V2 uses raw JSONL parsing via `map[string]interface{}`:
```go
// V2 dep_test.go
tk := readTaskByID(t, dir, "tick-aaa111")
blockedBy, ok := tk["blocked_by"].([]interface{})
```

V4 deserializes into typed `task.Task` structs:
```go
// V4 dep_test.go
tasks := readTasksFromDir(t, dir)
var taskA *task.Task
for i := range tasks {
    if tasks[i].ID == "tick-aaa111" {
        taskA = &tasks[i]
    }
}
```

V4's typed assertions are more robust. V2's `map[string]interface{}` requires type assertions at every access which could silently fail. V4's timestamp comparison (`taskA.Updated.After(now)`) is more precise than V2's string comparison (`tk["updated"] == "2026-01-19T10:00:00Z"`).

**Error Assertion Pattern**

V2 checks `err != nil` and asserts on `err.Error()`:
```go
// V2
err := app.Run([]string{"tick", "dep", "add", ...})
if err == nil { t.Fatal(...) }
if !strings.Contains(err.Error(), "tick-nonexist") { ... }
```

V4 checks exit code and stderr content:
```go
// V4
code := app.Run([]string{"tick", "dep", "add", ...})
if code != 1 { t.Errorf(...) }
errMsg := stderr.String()
if !strings.Contains(errMsg, "Error:") { ... }
```

V4's pattern tests the full user-facing behavior (exit code + stderr output), which is more integration-level. V2's pattern tests the programmatic return value, which is more unit-test-oriented. V4 is better from a behavior-verification standpoint since it tests what the user actually sees.

**Test Organization**

V2 groups tests under 3 top-level functions (`TestDepAddCommand`, `TestDepRmCommand`, `TestDepSubcommandRouting`), with all related subtests nested. This is more compact and groups related tests together logically.

V4 uses 14 separate top-level test functions, most containing a single subtest. This makes individual test selection with `-run` easier but results in more boilerplate and less logical grouping.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 5 | 5 |
| Lines added | 729 | 821 |
| Impl LOC (dep.go) | 170 | 180 |
| Test LOC (dep_test.go) | 554 | 632 |
| Test functions (top-level) | 3 | 14 |
| Test subtests (total) | ~23 | 22 |
| Unique edge cases tested | 17 | 13 |

## Verdict

**V2 is the better implementation for this task**, though both versions are competent and pass all acceptance criteria.

V2 wins primarily on **test completeness**. It covers 4 edge cases that V4 misses: stale ref removal on `rm` (a spec requirement), atomic write persistence for `rm`, the "one arg only" variant for `dep rm`, and unknown dep subcommand handling. V2 also has `openTaskWithBlockedByJSONL` and `openTaskWithParentJSONL` test helpers defined locally, making the test file self-contained.

V4 has some genuine advantages: the typed `task.Task` struct construction in tests is safer than V2's raw JSONL strings, the early self-reference check before opening the store is a minor optimization, `strings.TrimSpace` on inputs adds robustness, and the `runDepRm` doc comment documenting the stale-ref design decision is good practice. V4's exit-code + stderr testing pattern also tests more of the user-facing behavior.

However, V4's stale-ref comment is ironic -- it documents the edge case in the code but fails to test it. V2 tests it. For a task where the spec explicitly calls out "rm does not validate blocked_by_id exists as a task -- only checks array membership (supports removing stale refs)" as an edge case, missing this test is a notable gap.

The defensive normalization in V2 (`task.NormalizeID(tasks[i].ID)` during comparison) versus V4's direct comparison is a reasonable tradeoff -- V2 is more defensive, V4 is more efficient. Neither is clearly wrong.

On balance, V2's superior test coverage of spec-mandated edge cases outweighs V4's advantages in test setup type-safety and minor code improvements.
