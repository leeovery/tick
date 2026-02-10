# Task tick-core-3-2: tick dep add & tick dep rm commands

## Task Summary

This task wires up the CLI for managing task dependencies after creation. Two sub-subcommands under `tick dep`:

- `tick dep add <task_id> <blocked_by_id>` -- looks up both IDs, checks self-ref, checks duplicate, calls `ValidateDependency` (from tick-core-3-1 for cycle and child-blocked-by-parent detection), adds to `blocked_by`, updates timestamp. Output: `Dependency added: {task_id} blocked by {blocked_by_id}`
- `tick dep rm <task_id> <blocked_by_id>` -- looks up task_id, checks blocked_by_id in array, removes from `blocked_by`, updates timestamp. Output: `Dependency removed: {task_id} no longer blocked by {blocked_by_id}`

Key edge cases: `rm` does NOT validate `blocked_by_id` exists as a task (supports removing stale refs). Duplicate dep on add returns error with no mutation. `--quiet` suppresses output. IDs normalized to lowercase.

### Acceptance Criteria (from plan)

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

| Criterion | V4 | V5 |
|-----------|-----|-----|
| `dep add` adds dependency and outputs confirmation | PASS -- `runDepAdd` appends to `BlockedBy` slice at line 87, outputs `"Dependency added: %s blocked by %s"` at line 96; tested in `TestDepAdd_AddsDependencyBetweenTwoExistingTasks` and `TestDepAdd_OutputsConfirmation` | PASS -- `runDepAdd` appends to `BlockedBy` at line 86, outputs same format at line 93; tested in `TestDepAdd/"it adds a dependency..."` and `"it outputs confirmation on success"` |
| `dep rm` removes dependency and outputs confirmation | PASS -- `runDepRm` uses slice append to remove at lines 149-152, outputs `"Dependency removed: %s no longer blocked by %s"` at line 160; tested in `TestDepRm_RemovesExistingDependency` and `TestDepRm_OutputsConfirmation` | PASS -- `runDepRm` removes at lines 143-146, outputs same format at line 153; tested in `TestDepRm/"it removes an existing dependency"` and `"it outputs confirmation on success"` |
| Non-existent IDs return error | PASS -- `task_id` not found: `fmt.Errorf("Task '%s' not found", taskID)` at line 63; `blocked_by_id` not found: same message at line 72; tested in `TestDepAdd_ErrorTaskIDNotFound`, `TestDepRm_ErrorTaskIDNotFound`, `TestDepAdd_ErrorBlockedByIDNotFound` | PASS -- `task_id` not found: `fmt.Errorf("Task '%s' not found", taskID)` at line 63; `blocked_by_id` not found: same at line 68; tested in `TestDepAdd/"it errors when task_id not found"`, `TestDepRm/"it errors when task_id not found"`, `TestDepAdd/"it errors when blocked_by_id not found"` |
| Duplicate/missing dep return error | PASS -- duplicate: `fmt.Errorf("Task '%s' is already blocked by '%s'", ...)` at line 78; rm missing: `fmt.Errorf("Task '%s' is not blocked by '%s'", ...)` at line 144; tested in `TestDepAdd_ErrorDuplicateDependency` and `TestDepRm_ErrorDependencyNotFound` | PASS -- duplicate: same message at line 74; rm missing: same at line 139; tested in `TestDepAdd/"it errors on duplicate dependency"` and `TestDepRm/"it errors when dependency not found in blocked_by"` |
| Self-ref, cycle, child-blocked-by-parent return error | PASS -- self-ref checked before store access at line 43 (`"Cannot add dependency - creates cycle: %s \u2192 %s"`); cycle and child-blocked-by-parent delegated to `task.ValidateDependency` at line 82; tested in `TestDepAdd_ErrorSelfReference`, `TestDepAdd_ErrorCycle`, `TestDepAdd_ErrorChildBlockedByParent` | PASS -- self-ref delegated to `task.ValidateDependency` at line 79 (no separate check); cycle and child-blocked-by-parent also via `ValidateDependency`; tested in `TestDepAdd/"it errors on self-reference"`, `"it errors when add creates cycle"`, `"it errors when add creates child-blocked-by-parent"` |
| IDs normalized to lowercase | PASS -- `task.NormalizeID(strings.TrimSpace(args[0]))` at lines 39-40; tested in `TestDep_NormalizesIDsToLowercase` (both add and rm subtests) | PASS -- `task.NormalizeID(args[0])` at lines 41-42; tested in `TestDepAdd/"it normalizes IDs to lowercase"` and `TestDepRm/"it normalizes IDs to lowercase"` |
| `--quiet` suppresses output | PASS -- `if !a.Quiet` guard at line 95; tested in `TestDep_QuietSuppressesOutput` (both add and rm subtests) | PASS -- `if !ctx.Quiet` guard at line 92; tested in `TestDepAdd/"it suppresses output with --quiet"` and `TestDepRm/"it suppresses output with --quiet"` |
| `updated` timestamp refreshed | PASS -- `tasks[taskIdx].Updated = time.Now().UTC().Truncate(time.Second)` at line 88 (add) and line 155 (rm); tested in `TestDepAdd_UpdatesTimestamp` and `TestDepRm_UpdatesTimestamp` which verify `taskA.Updated.After(now)` | PASS -- same timestamp update at lines 87 (add) and 148 (rm); tested in `TestDepAdd/"it updates task's updated timestamp"` and `TestDepRm/"it updates task's updated timestamp"` with `time.Sleep(1100ms)` to ensure measurable difference |
| Persisted through storage engine | PASS -- mutation inside `s.Mutate()` callback; tested in `TestDep_PersistsViaAtomicWrite` which reads raw JSONL, parses JSON, and checks cache.db existence | PASS -- mutation inside `store.Mutate()` callback; tested in `TestDepAdd/"it persists via atomic write"` and `TestDepRm/"it persists via atomic write"` which re-read from file via `readTasksFromFile` |

## Implementation Comparison

### Approach

**V4: Method on `App` struct with manual sub-subcommand dispatch**

V4 implements `runDep` as a method on `*App` (line 14):

```go
func (a *App) runDep(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("Subcommand required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
    }
    subcommand := args[0]
    subArgs := args[1:]
    switch subcommand {
    case "add":
        return a.runDepAdd(subArgs)
    case "rm":
        return a.runDepRm(subArgs)
    default:
        return fmt.Errorf("Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subcommand)
    }
}
```

CLI registration is via a `case "dep":` block in `cli.go`'s switch statement (6 lines added):

```go
case "dep":
    if err := a.runDep(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

`runDepAdd` and `runDepRm` are separate methods on `*App` that each independently open the store, run `s.Mutate()`, and handle output.

Key V4 design choices:
- **Self-reference check is done before store access** (line 43): `if taskID == blockedByID { return fmt.Errorf("Cannot add dependency - creates cycle: %s \u2192 %s", taskID, taskID) }`. This is an optimization that avoids opening the store for a trivially invalid request.
- **ID normalization includes `strings.TrimSpace`**: `task.NormalizeID(strings.TrimSpace(args[0]))` at lines 39-40.
- **Linear scan for task lookup** in the Mutate callback: separate loops for finding `taskIdx` and checking `blockedByExists`.

**V5: Free functions with Context, command map dispatch**

V5 implements `runDep` as a free function taking `*Context` (line 10):

```go
func runDep(ctx *Context) error {
    if len(ctx.Args) == 0 {
        return fmt.Errorf("dep requires a subcommand: add, rm")
    }
    subcmd := ctx.Args[0]
    args := ctx.Args[1:]
    switch subcmd {
    case "add":
        return runDepAdd(ctx, args)
    case "rm":
        return runDepRm(ctx, args)
    default:
        return fmt.Errorf("unknown dep subcommand '%s'. Available: add, rm", subcmd)
    }
}
```

CLI registration is a single line in the `commands` map:

```go
"dep":    runDep,
```

`runDepAdd` and `runDepRm` are free functions that take `(ctx *Context, args []string)`.

Key V5 design choices:
- **No separate self-reference check** -- delegated entirely to `task.ValidateDependency` at line 79. This is simpler code but means the store is opened even for self-reference attempts.
- **Map-based task lookup in `runDepAdd`** (lines 55-58): builds a `map[string]int` for O(1) lookups:

```go
existing := make(map[string]int, len(tasks))
for i, t := range tasks {
    existing[t.ID] = i
}
taskIdx, found := existing[taskID]
```

- **Extra normalization in duplicate check** (line 73): `task.NormalizeID(dep) == blockedByID`. This handles cases where the stored `blocked_by` entry might have inconsistent casing.
- **Extra normalization in rm lookup** (line 134): `task.NormalizeID(dep) == blockedByID`. Same defensive normalization.

**Structural comparison of the Mutate callbacks:**

V4's `runDepAdd` Mutate callback uses separate sequential loops:
1. Loop to find `taskIdx`
2. Loop to find `blockedByExists`
3. Loop to check duplicate
4. Call `ValidateDependency`
5. Append + update timestamp

V5's `runDepAdd` Mutate callback uses a map for steps 1-2:
1. Build `existing` map (single loop)
2. Map lookup for `taskIdx`
3. Map lookup for `blockedByID` existence
4. Loop to check duplicate (with extra `NormalizeID` call)
5. Call `ValidateDependency`
6. Append + update timestamp

V5's map approach is O(n) construction + O(1) lookups vs V4's O(n) per lookup. For the `dep rm` path, V5 uses the same linear scan as V4 (no map), which is an inconsistency within V5.

**Error message style differences:**

| Scenario | V4 | V5 |
|----------|-----|-----|
| No subcommand | `"Subcommand required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>"` | `"dep requires a subcommand: add, rm"` |
| Unknown subcommand | `"Unknown dep subcommand '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>"` | `"unknown dep subcommand '%s'. Available: add, rm"` |
| Too few args (add) | `"Two task IDs required. Usage: tick dep add <task_id> <blocked_by_id>"` | `"dep add requires two IDs: tick dep add <task_id> <blocked_by_id>"` |
| Too few args (rm) | `"Two task IDs required. Usage: tick dep rm <task_id> <blocked_by_id>"` | `"dep rm requires two IDs: tick dep rm <task_id> <blocked_by_id>"` |
| Self-reference | `"Cannot add dependency - creates cycle: %s \u2192 %s"` (custom message) | Delegated to `ValidateDependency` |

V5's error messages for unknown subcommand and too-few-args are lowercase-starting, which is more Go-idiomatic. V4's are capitalized, matching spec style. V5's usage hint messages include the full command invocation pattern (`tick dep add <task_id> <blocked_by_id>`), which is arguably more helpful.

### Code Quality

**Imports:**

V4 imports:
```go
import (
    "fmt"
    "strings"
    "time"
    "github.com/leeovery/tick/internal/store"
    "github.com/leeovery/tick/internal/task"
)
```

V5 imports:
```go
import (
    "fmt"
    "time"
    "github.com/leeovery/tick/internal/engine"
    "github.com/leeovery/tick/internal/task"
)
```

V4 imports `strings` for `strings.TrimSpace`. V5 does not need it. V4 uses `store` package, V5 uses `engine` -- different naming for the same storage abstraction.

**Store creation pattern:**

V4 creates the store with explicit `NewStore` call within each command handler:
```go
s, err := store.NewStore(tickDir)
if err != nil {
    return err
}
defer s.Close()
```

V5 uses the same pattern but with the `engine` package:
```go
store, err := engine.NewStore(tickDir)
if err != nil {
    return err
}
defer store.Close()
```

Both versions create and close the store independently in `runDepAdd` and `runDepRm`. Neither version lifts store management to the parent `runDep` function, resulting in duplicated store open/close code across the two sub-subcommands. This is a shared minor DRY concern.

**Documentation:**

V4 doc comments:
```go
// runDep implements the `tick dep` command with sub-subcommands `add` and `rm`.
// runDepAdd implements `tick dep add <task_id> <blocked_by_id>`.
// It validates both IDs exist, checks for self-ref, duplicate, cycle, and
// child-blocked-by-parent, then adds the dependency and persists.
// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
// It looks up task_id, checks blocked_by_id in array, removes it, and persists.
// Note: rm does NOT validate that blocked_by_id exists as a task -- only checks
// array membership (supports removing stale refs).
```

V5 doc comments:
```go
// runDep implements the "tick dep" command, which dispatches to sub-subcommands
// "add" and "rm" for managing task dependencies post-creation.
// runDepAdd implements "tick dep add <task_id> <blocked_by_id>". It looks up
// both tasks, validates the dependency (self-ref, duplicate, cycle,
// child-blocked-by-parent), adds the dependency, and persists via atomic write.
// runDepRm implements "tick dep rm <task_id> <blocked_by_id>". It looks up the
// task, verifies the dependency exists in blocked_by, removes it, and persists
// via atomic write. It does NOT validate that blocked_by_id exists as a task,
// supporting removal of stale references.
```

Both have thorough doc comments on all three functions. V5 explicitly mentions "atomic write" in both add and rm comments, matching the spec. V4 mentions the self-ref check explicitly as distinct from cycle detection. Both note the stale-ref behavior for `rm`.

**Defensive normalization in V5:**

V5 applies `task.NormalizeID` during the duplicate check and rm lookup:
```go
// In runDepAdd duplicate check:
if task.NormalizeID(dep) == blockedByID {
// In runDepRm lookup:
if task.NormalizeID(dep) == blockedByID {
```

V4 compares directly:
```go
// In runDepAdd duplicate check:
if dep == blockedByID {
// In runDepRm lookup:
if dep == blockedByID {
```

V5's approach is more defensive -- it handles cases where existing `blocked_by` entries might have inconsistent casing (e.g., if data was manually edited). V4 assumes stored entries are already normalized.

**Output handling:**

Both versions use identical output patterns:
```go
// V4
if !a.Quiet {
    fmt.Fprintf(a.Stdout, "Dependency added: %s blocked by %s\n", taskID, blockedByID)
}

// V5
if !ctx.Quiet {
    fmt.Fprintf(ctx.Stdout, "Dependency added: %s blocked by %s\n", taskID, blockedByID)
}
```

The output strings are identical between versions and match the spec exactly.

**Self-reference handling:**

V4 checks self-reference explicitly before opening the store (line 43-45):
```go
if taskID == blockedByID {
    return fmt.Errorf("Cannot add dependency - creates cycle: %s \u2192 %s", taskID, taskID)
}
```

V5 does not check self-reference at the CLI level. It delegates entirely to `task.ValidateDependency` inside the Mutate callback (line 79). This means V5 opens the store even for trivially invalid self-reference requests, but is simpler code with a single validation path. V4's approach is more efficient (avoids unnecessary I/O) but duplicates part of the cycle detection logic from `ValidateDependency`.

### Test Quality

#### V4 Test Functions (17 top-level functions, 21 subtests)

1. **`TestDepAdd_AddsDependencyBetweenTwoExistingTasks`** (1 subtest)
   - `"it adds a dependency between two existing tasks"` -- creates two tasks, runs `tick dep add tick-aaa111 tick-bbb222`, verifies `taskA.BlockedBy` is `["tick-bbb222"]`

2. **`TestDepRm_RemovesExistingDependency`** (1 subtest)
   - `"it removes an existing dependency"` -- creates task A with `BlockedBy: ["tick-bbb222"]`, runs `tick dep rm`, verifies `BlockedBy` is empty

3. **`TestDepAdd_OutputsConfirmation`** (1 subtest)
   - `"it outputs confirmation on success (add)"` -- verifies exact output: `"Dependency added: tick-aaa111 blocked by tick-bbb222"`

4. **`TestDepRm_OutputsConfirmation`** (1 subtest)
   - `"it outputs confirmation on success (rm)"` -- verifies exact output: `"Dependency removed: tick-aaa111 no longer blocked by tick-bbb222"`

5. **`TestDepAdd_UpdatesTimestamp`** (1 subtest)
   - `"it updates task's updated timestamp on dep add"` -- creates task with fixed timestamp, runs dep add, verifies `taskA.Updated.After(now)`

6. **`TestDepRm_UpdatesTimestamp`** (1 subtest)
   - `"it updates task's updated timestamp on dep rm"` -- same pattern for rm

7. **`TestDepAdd_ErrorTaskIDNotFound`** (1 subtest)
   - `"it errors when task_id not found (add)"` -- runs with non-existent `tick-nonexist`, checks exit code 1, stderr contains `"Error:"` and `"tick-nonexist"`

8. **`TestDepRm_ErrorTaskIDNotFound`** (1 subtest)
   - `"it errors when task_id not found (rm)"` -- same for rm

9. **`TestDepAdd_ErrorBlockedByIDNotFound`** (1 subtest)
   - `"it errors when blocked_by_id not found (add)"` -- task exists but blocked_by target doesn't, checks stderr contains `"tick-nonexist"`

10. **`TestDepAdd_ErrorDuplicateDependency`** (1 subtest)
    - `"it errors on duplicate dependency (add)"` -- task already blocked by target, attempts add again, verifies exit code 1 AND verifies no mutation (blocked_by still has exactly 1 entry)

11. **`TestDepRm_ErrorDependencyNotFound`** (1 subtest)
    - `"it errors when dependency not found (rm)"` -- task exists but has no dependency on target, verifies error

12. **`TestDepAdd_ErrorSelfReference`** (1 subtest)
    - `"it errors on self-reference (add)"` -- runs `dep add tick-aaa111 tick-aaa111`, verifies exit code 1 and error on stderr

13. **`TestDepAdd_ErrorCycle`** (1 subtest)
    - `"it errors when add creates cycle"` -- B blocked by A, tries to add A blocked by B, verifies exit code 1, stderr contains `"cycle"`

14. **`TestDepAdd_ErrorChildBlockedByParent`** (1 subtest)
    - `"it errors when add creates child-blocked-by-parent"` -- child task with `Parent: "tick-parent1"`, tries to add child blocked by parent, verifies exit code 1, stderr contains `"parent"`

15. **`TestDep_NormalizesIDsToLowercase`** (2 subtests)
    - `"it normalizes IDs to lowercase (add)"` -- runs with `TICK-AAA111` and `TICK-BBB222`, verifies blocked_by stored as lowercase, verifies output uses lowercase
    - `"it normalizes IDs to lowercase (rm)"` -- runs rm with uppercase IDs, verifies removal works

16. **`TestDep_QuietSuppressesOutput`** (2 subtests)
    - `"it suppresses output with --quiet (add)"` -- runs with `--quiet`, verifies stdout is empty
    - `"it suppresses output with --quiet (rm)"` -- same for rm

17. **`TestDep_ErrorFewerThanTwoIDs`** (4 subtests)
    - `"it errors when fewer than two IDs provided (add, zero args)"` -- `tick dep add` with no args, checks stderr for `"Error:"` and `"Usage:"`
    - `"it errors when fewer than two IDs provided (add, one arg)"` -- `tick dep add tick-aaa111`, same checks
    - `"it errors when fewer than two IDs provided (rm, zero args)"` -- same for rm
    - `"it errors when no subcommand provided to dep"` -- `tick dep` alone, checks error

18. **`TestDep_PersistsViaAtomicWrite`** (1 subtest)
    - `"it persists via atomic write (add)"` -- runs dep add, reads raw JSONL file, parses JSON, verifies `blocked_by` in persisted data, checks `cache.db` exists

**V4 test structure notes:**
- Each scenario is a separate top-level `Test*` function (17 functions for 21 subtests)
- Task setup uses manual struct construction: `task.Task{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}`
- Fixed timestamp: `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)` used consistently
- Helper: `setupInitializedDirWithTasks(t, existing)`, `readTasksFromDir(t, dir)`, `setupInitializedDir(t)`
- CLI invocation: `app.Run([]string{"tick", ...})` via `App` struct construction
- Timestamp tests verify `taskA.Updated.After(now)` without sleep -- relies on time passing between `now` construction and execution
- Persistence test reads raw JSONL and parses with `json.Unmarshal` into `map[string]interface{}` -- more thorough than re-reading through the application layer

#### V5 Test Functions (2 top-level functions, 22 subtests)

**`TestDepAdd`** (13 subtests):

1. `"it adds a dependency between two existing tasks"` -- creates tasks via `task.NewTask`, runs `dep add`, verifies `BlockedBy` contains `"tick-bbbbbb"`

2. `"it outputs confirmation on success"` -- verifies stdout contains `"Dependency added: tick-aaaaaa blocked by tick-bbbbbb"`

3. `"it updates task's updated timestamp"` -- captures `originalUpdated`, calls `time.Sleep(1100 * time.Millisecond)`, runs dep add, verifies `tk.Updated.After(originalUpdated)`

4. `"it errors when task_id not found"` -- runs with `tick-nonexist`, checks exit code 1, stderr contains `"not found"`

5. `"it errors when blocked_by_id not found"` -- task exists but blocked_by target doesn't, checks `"not found"` in stderr

6. `"it errors on duplicate dependency"` -- existing blocked_by, attempts re-add, checks `"already"` in stderr

7. `"it errors on self-reference"` -- runs `dep add tick-aaaaaa tick-aaaaaa`, checks `"cycle"` in stderr

8. `"it errors when add creates cycle"` -- 3-task cycle: A blocked by B, B blocked by C, tries C blocked by A. Checks `"cycle"` in stderr

9. `"it errors when add creates child-blocked-by-parent"` -- child with `Parent: "tick-bbbbbb"`, tries child blocked by parent. Checks `"parent"` in stderr

10. `"it normalizes IDs to lowercase"` -- runs with `TICK-AAAAAA` and `TICK-BBBBBB`, verifies `BlockedBy` stored lowercase

11. `"it suppresses output with --quiet"` -- `--quiet` flag, verifies stdout empty

12. `"it errors when fewer than two IDs provided"` -- **table-driven** with 2 cases: `{"no IDs", []string{"tick", "dep", "add"}}` and `{"one ID", []string{"tick", "dep", "add", "tick-aaaaaa"}}`. Checks `"requires two"` in stderr

13. `"it persists via atomic write"` -- runs dep add, re-reads via `readTasksFromFile`, verifies `BlockedBy`

**`TestDepRm`** (9 subtests):

1. `"it removes an existing dependency"` -- creates task with `BlockedBy`, removes it, verifies empty

2. `"it outputs confirmation on success"` -- verifies stdout contains `"Dependency removed: tick-aaaaaa no longer blocked by tick-bbbbbb"`

3. `"it updates task's updated timestamp"` -- with `time.Sleep(1100ms)`, verifies `Updated.After(originalUpdated)`

4. `"it errors when task_id not found"` -- checks `"not found"` in stderr

5. `"it errors when dependency not found in blocked_by"` -- task exists but dependency not in `BlockedBy`, checks `"not blocked by"` in stderr

6. `"it removes stale dependency without validating blocked_by_id exists"` -- **unique to V5** -- task has `BlockedBy: ["tick-deleted"]` (a non-existent task), runs rm, verifies removal succeeds and confirmation output is printed. This directly tests the edge case from the spec: "rm does not validate blocked_by_id exists as a task -- only checks array membership (supports removing stale refs)"

7. `"it normalizes IDs to lowercase"` -- uppercase input, verifies removal works

8. `"it suppresses output with --quiet"` -- verifies quiet flag works for rm

9. `"it errors when fewer than two IDs provided"` -- **table-driven** with 2 cases (no IDs, one ID). Checks `"requires two"` in stderr

10. `"it persists via atomic write"` -- task has `BlockedBy: ["tick-bbbbbb", "tick-cccccc"]`, removes `tick-bbbbbb`, verifies only `tick-cccccc` remains. This tests partial removal (preserving other deps), which V4 does not test.

**V5 test structure notes:**
- 2 top-level functions (`TestDepAdd`, `TestDepRm`) with subtests nested via `t.Run`
- Task setup uses constructor: `task.NewTask("tick-aaaaaa", "Task A")` then mutates fields
- `time.Sleep(1100 * time.Millisecond)` in timestamp tests to ensure measurable time difference -- more robust than V4's approach but makes tests slower
- Table-driven sub-subtests for the "fewer than two IDs" scenarios
- CLI invocation: `Run([]string{"tick", ...}, dir, &stdout, &stderr, false)` -- free function
- Persistence test re-reads through application layer (not raw file parsing)

#### Test Coverage Diff

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| Add dependency between two tasks | Yes | Yes |
| Remove existing dependency | Yes | Yes |
| Add output confirmation (exact match) | Yes (exact `==`) | Yes (`strings.Contains`) |
| Rm output confirmation (exact match) | Yes (exact `==`) | Yes (`strings.Contains`) |
| Updated timestamp refreshed (add) | Yes | Yes (with sleep) |
| Updated timestamp refreshed (rm) | Yes | Yes (with sleep) |
| task_id not found (add) | Yes | Yes |
| task_id not found (rm) | Yes | Yes |
| blocked_by_id not found (add) | Yes | Yes |
| Duplicate dependency (add) | Yes (also verifies no mutation) | Yes |
| Dependency not found (rm) | Yes | Yes |
| Self-reference (add) | Yes | Yes |
| Cycle detection (add) | Yes (2-task cycle) | Yes (3-task cycle) |
| Child-blocked-by-parent (add) | Yes | Yes |
| Normalize IDs lowercase (add) | Yes (checks stored value AND output) | Yes (checks stored value only) |
| Normalize IDs lowercase (rm) | Yes | Yes |
| Quiet suppresses output (add) | Yes | Yes |
| Quiet suppresses output (rm) | Yes | Yes |
| Missing args: add zero args | Yes | Yes (table-driven) |
| Missing args: add one arg | Yes | Yes (table-driven) |
| Missing args: rm zero args | Yes | Yes (table-driven) |
| Missing args: rm one arg | No | Yes (table-driven) |
| No subcommand to dep | Yes | No |
| Persists via atomic write (add) | Yes (raw JSONL + cache.db) | Yes (re-read) |
| Persists via atomic write (rm) | No | Yes (partial removal verified) |
| Stale ref removal (rm without task existing) | No | **Yes** |
| Duplicate check: no mutation on error | Yes (verifies count unchanged) | No |
| Error contains task ID in message | Yes (checks for `"tick-nonexist"`) | No (checks generic `"not found"`) |
| Rm persistence with partial deps | No | **Yes** (removes one of two deps) |
| Cycle test complexity | 2-task cycle (A->B, B->A) | 3-task cycle (A->B, B->C, C->A) |
| Output uses lowercase in normalize test | Yes | No |

**Notable V5-exclusive tests:**
- **Stale ref removal test** (`"it removes stale dependency without validating blocked_by_id exists"`) -- directly tests the spec's documented edge case. V4 does not test this at all.
- **Rm persistence test with partial removal** -- removes one dependency from a task with two, verifies the other remains. More realistic than V4's add-only persistence test.
- **3-task cycle** -- more complex than V4's 2-task direct cycle

**Notable V4-exclusive tests:**
- **No-mutation verification on duplicate** -- after duplicate add attempt, verifies `BlockedBy` still has exactly 1 entry
- **No subcommand to dep** -- tests `tick dep` alone (V5 omits this)
- **Error message contains task ID** -- V4 checks that the not-found error includes the specific task ID (`"tick-nonexist"`)
- **Raw JSONL parsing in persistence test** -- reads raw file, parses JSON, checks `cache.db` existence
- **Output uses lowercase in normalize test** -- verifies the confirmation message uses lowercase IDs

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Handle all errors explicitly (no naked returns) | PASS -- all errors from `DiscoverTickDir`, `store.NewStore`, `s.Mutate` are checked and returned | PASS -- all errors from `DiscoverTickDir`, `engine.NewStore`, `store.Mutate` are checked and returned |
| Write table-driven tests with subtests | PARTIAL -- uses `t.Run` subtests throughout but mostly individual top-level functions; `TestDep_ErrorFewerThanTwoIDs` has multiple subtests but not table-driven | PARTIAL -- uses `t.Run` subtests with 2 top-level functions; `"it errors when fewer than two IDs provided"` uses table-driven pattern for both add and rm |
| Document all exported functions, types, and packages | PASS -- all three functions (`runDep`, `runDepAdd`, `runDepRm`) have doc comments; package-level doc exists in `cli.go` | PASS -- all three functions documented; package doc exists |
| Propagate errors with fmt.Errorf("%w", err) | PARTIAL -- errors from `ValidateDependency` and store operations returned directly without wrapping; custom errors use `fmt.Errorf` without `%w` | PARTIAL -- same pattern; direct error returns without wrapping |
| No hardcoded configuration | PASS -- no magic values | PASS -- no magic values |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Avoid _ assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |
| Use functional options or env vars (no hardcoded config) | PASS -- store path discovered via `DiscoverTickDir` | PASS -- same |

### Spec-vs-Convention Conflicts

**1. Capitalized error messages**

- **Spec says:** Error messages like `"Task '{id}' not found"` and usage hints with capitalized starts.
- **Go convention:** Error strings should not be capitalized (per Go Code Review Comments).
- **V4 chose:** Capitalized: `"Subcommand required. Usage: ..."`, `"Unknown dep subcommand '%s'. Usage: ..."`, `"Two task IDs required. Usage: ..."`, `"Task '%s' not found"`, `"Task '%s' is already blocked by '%s'"`, `"Task '%s' is not blocked by '%s'"`.
- **V5 chose:** Mixed: lowercase for dep-level errors (`"dep requires a subcommand: add, rm"`, `"unknown dep subcommand '%s'. Available: add, rm"`, `"dep add requires two IDs: ..."`) but capitalized for data errors (`"Task '%s' not found"`, `"Task '%s' is already blocked by '%s'"`, `"Task '%s' is not blocked by '%s'"`).
- **Assessment:** V5's approach is more Go-idiomatic for the routing/argument errors (lowercase) while keeping user-facing data errors capitalized. V4 uniformly capitalizes all errors. Since these are CLI-facing messages printed to users (not library errors being composed), both approaches are defensible. V5's mixed approach is arguably more nuanced.

**2. Self-reference error message vs ValidateDependency**

- **Spec says:** Self-ref should return error; delegates to tick-core-3-1.
- **V4 chose:** Adds its own self-ref check with custom message `"Cannot add dependency - creates cycle: %s \u2192 %s"` before calling `ValidateDependency`.
- **V5 chose:** Relies entirely on `ValidateDependency` for self-ref detection.
- **Assessment:** The spec says "Cycle/child-blocked-by-parent delegated to tick-core-3-1". Since `ValidateDependency` (from tick-core-3-1) already handles self-reference as a cycle, V5's delegation is spec-compliant. V4's redundant check is harmless (saves I/O) but technically duplicates logic. Both are reasonable.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 5 (cli.go, dep.go, dep_test.go, 2 docs) | 5 (cli.go, dep.go, dep_test.go, 2 docs) |
| Lines added (total) | 821 | 673 |
| Impl LOC (dep.go) | 180 | 166 |
| Test LOC (dep_test.go) | 632 | 503 |
| cli.go lines changed | +6 | +1 |
| Top-level test functions | 17 | 2 |
| Total test subtests | 21 | 22 |

## Verdict

**V5 is the slightly better implementation.**

Both versions fully satisfy all 9 acceptance criteria with no gaps. The deciding factors are:

1. **Test coverage breadth (V5 advantage):** V5 has 22 subtests vs V4's 21, and critically includes the **stale ref removal test** (`"it removes stale dependency without validating blocked_by_id exists"`), which directly validates a documented edge case from the spec that V4 completely ignores. V5 also tests **rm persistence with partial dependency removal** (removing one of two deps), which is more realistic than V4's add-only persistence test. V5's 3-task cycle test is more thorough than V4's 2-task cycle.

2. **Defensive normalization (V5 advantage):** V5 applies `task.NormalizeID(dep)` during duplicate checks and rm lookups, handling potential inconsistencies in stored data. V4 assumes stored entries are already normalized. While unlikely in practice, V5's approach is more robust.

3. **Table-driven tests (V5 advantage):** V5 uses table-driven patterns for the "fewer than two IDs" tests in both add and rm, which is more aligned with the golang-pro skill's preference. V4 uses individual subtests.

4. **CLI integration (V5 advantage):** V5's command map registration is 1 line vs V4's 6 lines. V5's free-function pattern with `*Context` is consistent with the broader V5 architecture.

5. **Test assertion depth (V4 advantage):** V4 has some stronger individual assertions: verifying no mutation on duplicate error, checking error messages contain specific task IDs, verifying output uses lowercase in normalize tests, testing the no-subcommand case, and parsing raw JSONL for persistence verification. These are individually valuable but don't compensate for V5's broader coverage.

6. **Self-reference optimization (V4 minor advantage):** V4's early self-ref check avoids opening the store for trivially invalid requests. This is a minor efficiency win but adds code duplication with `ValidateDependency`.

7. **Test speed (V4 advantage):** V4's timestamp tests don't use `time.Sleep` while V5 uses `time.Sleep(1100ms)` per timestamp test, adding ~2.2 seconds of wall-clock time. V4's approach works because the fixed `now` in the past is always before `time.Now()`.

Overall, V5's coverage of the stale ref edge case (a spec-documented edge case that V4 entirely misses), the partial rm persistence test, the more complex cycle test, and the defensive normalization approach outweigh V4's advantages in individual assertion depth and test speed. The margin is narrow.
