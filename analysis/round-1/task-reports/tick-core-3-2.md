# Task tick-core-3-2: tick dep add & tick dep rm commands

## Task Summary

Wire up CLI commands for managing task dependencies post-creation. Two subcommands under `tick dep`:

- `tick dep add <task_id> <blocked_by_id>` -- Parse two positional IDs, validate both tasks exist, check self-reference, check duplicate dependency, call `ValidateDependency` (from tick-core-3-1) for cycle and child-blocked-by-parent checks, mutate via storage engine. Output: `Dependency added: {task_id} blocked by {blocked_by_id}`
- `tick dep rm <task_id> <blocked_by_id>` -- Look up task_id, check blocked_by_id is in the `blocked_by` array (does NOT validate blocked_by_id exists as a task -- supports removing stale refs), remove from array, update timestamp. Output: `Dependency removed: {task_id} no longer blocked by {blocked_by_id}`
- `--quiet` suppresses output. Errors to stderr, exit 1.
- IDs normalized to lowercase.

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| `dep add` adds dependency and outputs confirmation | PASS -- adds to `blocked_by`, prints "Dependency added: ..." | PASS -- adds to `blocked_by`, prints "Dependency added: ..." | PASS -- adds to `blocked_by`, prints "Dependency added: ..." |
| `dep rm` removes dependency and outputs confirmation | PASS -- removes from `blocked_by`, prints "Dependency removed: ..." | PASS -- removes from `blocked_by`, prints "Dependency removed: ..." | PASS -- removes from `blocked_by`, prints "Dependency removed: ..." |
| Non-existent IDs return error | PASS -- tests task_id not found on add, blocked_by_id not found on add | PASS -- tests task_id not found on add/rm, blocked_by_id not found on add | PASS -- tests task_id not found on add/rm, blocked_by_id not found on add |
| Duplicate/missing dep return error | PASS -- tests duplicate add and missing dep rm | PASS -- tests duplicate add and missing dep rm | PASS -- tests duplicate add and missing dep rm |
| Self-ref, cycle, child-blocked-by-parent return error | PASS -- delegates to ValidateDependency | PASS -- delegates to ValidateDependency | PASS -- delegates to ValidateDependency |
| IDs normalized to lowercase | PASS -- uses task.NormalizeID on both args | PASS -- uses task.NormalizeID on both args, also normalizes when comparing in Mutate | PASS -- uses task.NormalizeID on both args, also normalizes when comparing in Mutate |
| `--quiet` suppresses output | PASS -- checks `a.opts.Quiet` | PASS -- checks `a.config.Quiet` | PASS -- checks `a.flags.Quiet` |
| `updated` timestamp refreshed | PASS -- sets `time.Now().UTC().Truncate(time.Second)` | PASS -- sets `time.Now().UTC().Truncate(time.Second)` | PASS -- uses `task.DefaultTimestamps()` |
| Persisted through storage engine | PASS -- uses `store.Mutate()` | PASS -- uses `store.Mutate()`, explicit persistence tests | PASS -- uses `store.Mutate()`, explicit persistence tests |

## Implementation Comparison

### Approach

All three versions follow the same high-level architecture: a `dep` dispatcher function routing to `add`/`rm` subhandlers, each opening the store, calling `store.Mutate()` with a closure that validates and mutates, then printing output if not quiet. The differences lie in app structure integration, argument handling, error propagation, and normalization strategy.

#### App Structure & Routing

**V1** uses a two-argument method signature `cmdDep(workDir string, args []string)` where `workDir` is passed explicitly. The router in `cli.go` has already stripped the program name and global flags, so `args` contains `["add", "tick-xxx", "tick-yyy"]`:

```go
// V1: internal/cli/cli.go
case "dep":
    err = a.cmdDep(workDir, cmdArgs)
```

```go
// V1: internal/cli/dep.go
func (a *App) cmdDep(workDir string, args []string) error {
    subcmd := args[0]
    switch subcmd {
    case "add":
        return a.cmdDepAdd(workDir, args[1:])
```

**V2** uses a no-workDir signature `runDep(args []string)` because the App struct stores `a.workDir` internally. Args are already stripped of program name and global flags:

```go
// V2: internal/cli/app.go
case "dep":
    return a.runDep(cmdArgs)
```

```go
// V2: internal/cli/dep.go
func (a *App) runDep(args []string) error {
    subcmd := args[0]
    cmdArgs := args[1:]
    switch subcmd {
    case "add":
        return a.runDepAdd(cmdArgs)
```

**V3** returns `int` exit codes directly (not `error`), and takes the raw `args` slice with program name still present. The `dep` dispatcher discovers the tick directory upfront, then passes `tickDir` to subhandlers:

```go
// V3: internal/cli/cli.go
case "dep":
    return a.runDep(args)
```

```go
// V3: internal/cli/dep.go
func (a *App) runDep(args []string) int {
    tickDir, err := DiscoverTickDir(a.Cwd)
    if err != nil { ... return 1 }
    if len(args) < 3 { ... return 1 }
    subcommand := args[2]
    switch subcommand {
    case "add":
        return a.runDepAdd(tickDir, args)
```

Key difference: V3 discovers `tickDir` once in `runDep` and passes it down, while V1 and V2 discover it independently in each subhandler. V3 also writes errors directly to stderr and returns int, while V1 and V2 return `error` and let the caller format it.

#### Argument Indexing

**V1 and V2** receive pre-sliced args, so they access `args[0]` and `args[1]` for task_id and blocked_by_id.

**V3** receives the full raw args, so it accesses `args[3]` and `args[4]`:
```go
// V3
taskID := task.NormalizeID(args[3])
blockedByID := task.NormalizeID(args[4])
```

#### ID Normalization Strategy

**V1** normalizes inputs upfront and then does direct comparisons against task IDs stored in the JSONL file (assumes stored IDs are already lowercase):

```go
// V1
taskID := task.NormalizeID(args[0])
blockedByID := task.NormalizeID(args[1])
// ...
t, ok := taskMap[taskID]  // direct map lookup
```

**V2 and V3** normalize inputs upfront AND normalize stored IDs during comparison. This is more defensive but technically redundant if the storage layer guarantees lowercase:

```go
// V2
for i := range tasks {
    normalizedID := task.NormalizeID(tasks[i].ID)
    if normalizedID == taskID {
        taskIdx = i
    }
}
```

```go
// V3
for _, existingBlocker := range targetTask.BlockedBy {
    if task.NormalizeID(existingBlocker) == blockedByID {
```

#### Task Lookup Pattern

**V1** builds a `map[string]*task.Task` upfront, enabling O(1) lookups for both task_id and blocked_by_id:

```go
// V1
taskMap := make(map[string]*task.Task, len(tasks))
for i := range tasks {
    taskMap[tasks[i].ID] = &tasks[i]
}
t, ok := taskMap[taskID]
```

**V2** uses index-based linear scan, storing `taskIdx` and a `blockedByExists` bool:

```go
// V2
taskIdx := -1
blockedByExists := false
for i := range tasks {
    normalizedID := task.NormalizeID(tasks[i].ID)
    if normalizedID == taskID { taskIdx = i }
    if normalizedID == blockedByID { blockedByExists = true }
}
```

**V3** also uses linear scan but splits lookup: a `map[string]bool` for existence checking, then a separate loop to find the pointer:

```go
// V3
idSet := make(map[string]bool)
for _, t := range tasks { idSet[t.ID] = true }
if !idSet[taskID] { return error }
// ... then separately:
var targetTask *task.Task
for i := range tasks {
    if tasks[i].ID == taskID { targetTask = &tasks[i]; break }
}
```

V1's approach is the most efficient (single map, one pass). V3 does two passes. V2 does a single pass but uses linear scan.

#### dep rm: Stale Ref Handling

All three correctly implement the spec requirement that `dep rm` does NOT validate `blocked_by_id` exists as a task. V1 and V3 simply skip the existence check. V2 adds an explicit comment:

```go
// V2
// Find and remove the dependency from blocked_by (by array membership, not task existence)
```

#### Empty blocked_by Handling (dep rm)

**V1** explicitly sets `blocked_by` to `nil` when empty (to omit it from JSON serialization):

```go
// V1
if len(newBlockedBy) == 0 {
    t.BlockedBy = nil
} else {
    t.BlockedBy = newBlockedBy
}
```

**V2 and V3** assign the empty slice directly without nil-ification:

```go
// V2 and V3
tasks[taskIdx].BlockedBy = newBlockedBy  // may be empty []string
```

V1's approach is cleaner for JSON serialization (`omitempty` will drop nil but keep `[]`).

#### Timestamp Update

**V1 and V2** use `time.Now().UTC().Truncate(time.Second)` directly:

```go
// V1 and V2
t.Updated = time.Now().UTC().Truncate(time.Second)
```

**V3** uses a helper function `task.DefaultTimestamps()`:

```go
// V3
_, updated := task.DefaultTimestamps()
targetTask.Updated = updated
```

V3's approach is more DRY if other commands also need timestamps, ensuring consistency.

#### Error Propagation

**V1** returns `error` from all dep functions, letting the top-level `Run()` handle formatting to stderr:
```go
// V1 - Run() handles the error
if err != nil {
    fmt.Fprintf(a.stderr, "Error: %s\n", err)
    return 1
}
```

**V2** returns `error` but uses `unwrapMutationError(err)` for Mutate errors:
```go
// V2
if err != nil {
    return unwrapMutationError(err)
}
```

**V3** returns `int` exit codes and formats errors directly in each handler:
```go
// V3
if err != nil {
    fmt.Fprintf(a.Stderr, "Error: %s\n", err)
    return 1
}
```

#### Usage Messages

All three provide usage hints. Error messages differ slightly:

- V1: `"dep add requires two IDs. Usage: tick dep add <task_id> <blocked_by_id>"`
- V2: `"Two task IDs required. Usage: tick dep add <task_id> <blocked_by_id>"`
- V3: `"Error: Two IDs required. Usage: tick dep add <task_id> <blocked_by_id>"`

V3 also added `dep` to the help text:
```go
// V3 only
fmt.Fprintln(a.Stdout, "  dep     Manage dependencies (add/rm)")
```

### Code Quality

#### Error Message Casing

**V1** uses lowercase error messages consistently (`"task '%s' not found"`, `"dependency already exists: ..."`), which follows Go conventions.

**V2** uses title-case (`"Task '%s' not found"`, `"Task '%s' is already blocked by '%s'"`), which is non-idiomatic for Go error messages.

**V3** also uses lowercase (`"task '%s' not found"`, `"task '%s' is already blocked by '%s'"`).

V1 and V3 follow Go conventions; V2 does not.

#### Doc Comments

**V1** has no doc comments on any of the three functions.

**V2** has doc comments on all three functions:
```go
// runDep dispatches the dep subcommands: add and rm.
func (a *App) runDep(args []string) error {
// runDepAdd implements `tick dep add <task_id> <blocked_by_id>`.
func (a *App) runDepAdd(args []string) error {
// runDepRm implements `tick dep rm <task_id> <blocked_by_id>`.
func (a *App) runDepRm(args []string) error {
```

**V3** has doc comments on all three functions:
```go
// runDep executes the dep subcommand with its sub-subcommands (add, rm).
func (a *App) runDep(args []string) int {
// runDepAdd executes the dep add subcommand.
func (a *App) runDepAdd(tickDir string, args []string) int {
// runDepRm executes the dep rm subcommand.
func (a *App) runDepRm(tickDir string, args []string) int {
```

V2 and V3 are better documented.

#### Code Duplication

All three versions have significant duplication between `depAdd` and `depRm` (store open, task lookup, timestamp update, output). None extracted shared helpers. This is acceptable given the different validation logic in each path.

**V3** reduced one duplication by discovering `tickDir` in the parent dispatcher rather than in each subhandler.

#### Naming Conventions

- V1: `cmdDep`, `cmdDepAdd`, `cmdDepRm` -- prefix `cmd`
- V2: `runDep`, `runDepAdd`, `runDepRm` -- prefix `run`
- V3: `runDep`, `runDepAdd`, `runDepRm` -- prefix `run`

All are consistent within their respective codebases.

### Test Quality

#### V1 Test Functions (16 subtests in 1 top-level function `TestDepCommands`)

1. `adds a dependency between two existing tasks` -- creates two tasks via CLI, runs dep add, reads JSONL to check `blocked_by`
2. `removes an existing dependency` -- creates task with `--blocked-by`, removes it, checks JSONL
3. `outputs confirmation on add` -- checks stdout contains "added" and task ID
4. `outputs confirmation on rm` -- checks stdout contains "removed" and task ID
5. `updates task updated timestamp` -- compares JSONL before/after dep add
6. `errors when task_id not found on add` -- uses nonexistent task_id
7. `errors when blocked_by_id not found on add` -- uses nonexistent blocked_by_id
8. `errors on duplicate dependency` -- adds dependency that already exists via `--blocked-by`
9. `errors when dependency not found on rm` -- removes dependency that was never added
10. `errors on self-reference` -- adds task blocked by itself
11. `errors when add creates cycle` -- A blocked by B, tries B blocked by A
12. `errors when add creates child-blocked-by-parent` -- child blocked by its parent
13. `normalizes IDs to lowercase` -- uses `strings.ToUpper` on IDs
14. `suppresses output with --quiet` -- checks stdout is empty
15. `errors when fewer than two IDs provided` -- tests one ID and zero IDs
16. `errors when no subcommand provided` -- `tick dep` with no add/rm

**V1 test approach**: Integration-style, uses actual CLI `createTask` + `runCmd` helpers. Tests create real tasks by running the full CLI flow, then inspect JSONL files directly with `os.ReadFile`.

#### V2 Test Functions (25 subtests in 3 top-level functions)

`TestDepAddCommand` (13 subtests):
1. `it adds a dependency between two existing tasks` -- sets up JSONL directly, verifies via `readTaskByID`
2. `it outputs confirmation on success (add)` -- exact string match
3. `it updates task's updated timestamp on add` -- checks timestamp changed
4. `it errors when task_id not found (add)` -- checks error contains task ID
5. `it errors when blocked_by_id not found (add)` -- checks error contains blocked_by ID
6. `it errors on duplicate dependency (add)` -- also verifies no mutation occurred
7. `it errors on self-reference (add)` -- checks error contains "cycle"
8. `it errors when add creates cycle` -- A blocked by B, tries B blocked by A
9. `it errors when add creates child-blocked-by-parent` -- checks error contains "parent"
10. `it normalizes IDs to lowercase (add)` -- verifies stored ID is lowercase, output is lowercase
11. `it suppresses output with --quiet (add)` -- checks stdout is empty string
12. `it errors when fewer than two IDs provided (add)` -- tests zero and one ID
13. `it persists via atomic write (add)` -- reads file directly with `os.ReadFile`

`TestDepRmCommand` (10 subtests):
14. `it removes an existing dependency` -- verifies blocked_by is empty/omitted
15. `it outputs confirmation on success (rm)` -- exact string match
16. `it updates task's updated timestamp on rm` -- checks timestamp changed
17. `it errors when task_id not found (rm)` -- checks error message
18. `it errors when dependency not found in blocked_by (rm)` -- task has no deps
19. `it does not validate blocked_by_id exists as a task on rm (supports stale refs)` -- **unique test**: blocked_by references nonexistent task, rm succeeds
20. `it normalizes IDs to lowercase (rm)` -- verifies stored data and output
21. `it suppresses output with --quiet (rm)` -- checks stdout empty
22. `it errors when fewer than two IDs provided (rm)` -- tests zero and one ID
23. `it persists via atomic write (rm)` -- reads file, checks blocked_by absent

`TestDepSubcommandRouting` (2 subtests):
24. `it errors for dep with no subcommand` -- checks "Usage"
25. `it errors for dep with unknown subcommand` -- checks "unknown"

**V2 test approach**: Unit-style, directly constructs App with `NewApp()`, sets `workDir` and `stdout`, uses custom helpers (`setupTickDirWithContent`, `openTaskJSONL`, `openTaskWithBlockedByJSONL`, `readTaskByID`). Tests set up raw JSONL content directly, giving precise control. Separates add/rm/routing into three test groups.

#### V3 Test Functions (23 subtests in 1 top-level function `TestDepCommands`)

1. `it adds a dependency between two existing tasks` -- uses `setupTask`, `readTasksFromDir`
2. `it removes an existing dependency` -- uses `setupTaskFull` with blocked_by
3. `it outputs confirmation on success for add` -- exact string match
4. `it outputs confirmation on success for rm` -- exact string match
5. `it updates task updated timestamp on add` -- checks timestamp differs from original
6. `it updates task updated timestamp on rm` -- checks timestamp differs from original
7. `it errors when task_id not found on add` -- checks stderr contains ID and "not found"
8. `it errors when task_id not found on rm` -- checks stderr contains ID and "not found"
9. `it errors when blocked_by_id not found on add` -- checks stderr
10. `it errors on duplicate dependency on add` -- checks stderr contains "already" and "blocked by"
11. `it errors when dependency not found on rm` -- checks stderr contains "not blocked by"
12. `it errors on self-reference on add` -- checks stderr contains "cycle"
13. `it errors when add creates cycle` -- A blocked by B, tries B blocked by A
14. `it errors when add creates child-blocked-by-parent` -- checks "cannot be blocked by its parent"
15. `it normalizes IDs to lowercase` -- one test covering both add (verifies stored data + output)
16. `it suppresses output with --quiet on add` -- also verifies dependency was still added
17. `it suppresses output with --quiet on rm` -- also verifies dependency was still removed
18. `it errors when fewer than two IDs provided for add` -- tests zero and one ID
19. `it errors when fewer than two IDs provided for rm` -- tests zero and one ID
20. `it persists via atomic write` -- adds then removes via separate App instances
21. `it allows rm to remove stale refs without validating blocked_by_id exists` -- stale ref test
22. `it errors with unknown dep subcommand` -- checks "Unknown" and "dep"
23. `it errors with missing dep subcommand` -- checks "Usage:"

**V3 test approach**: Uses `bytes.Buffer` for stdout/stderr directly, struct literal `&App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}`. Uses `setupTask` and `setupTaskFull` helpers for test data. Uses typed struct with `json` tags in `readTasksFromDir` for precise field access.

#### Test Coverage Gaps

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Stale ref removal (rm) | NOT TESTED | TESTED | TESTED |
| Persistence (add) | NOT TESTED | TESTED | TESTED (combined) |
| Persistence (rm) | NOT TESTED | TESTED | TESTED (combined) |
| Unknown dep subcommand | PARTIAL (checks "add"/"rm" in error) | TESTED | TESTED |
| task_id not found on rm | NOT TESTED | TESTED | TESTED |
| Normalize IDs on rm | NOT TESTED (only tests add) | TESTED (separate rm test) | NOT TESTED (only tests add in normalize test) |
| Timestamp update on rm | NOT TESTED (only tests add) | TESTED | TESTED |
| Quiet on rm | NOT TESTED (only tests add) | TESTED | TESTED |
| Fewer than two IDs on rm | NOT TESTED (only tests add) | TESTED | TESTED |
| Duplicate add no-mutation verification | NOT TESTED | TESTED (`len(blockedBy) != 1`) | NOT TESTED |
| Quiet still performs operation | NOT TESTED | NOT TESTED | TESTED (verifies dep added/removed even with quiet) |

V1 has the most significant test gaps: no persistence tests, no stale ref test, no rm-specific error tests (task_id not found on rm, quiet on rm, normalize on rm, fewer IDs on rm, timestamp on rm). V1 only tests these scenarios for `add`.

V2 is the most thorough: separate add/rm test groups ensure each subcommand's error paths are independently verified. The stale ref test and no-mutation verification on duplicate are unique strengths.

V3 is close to V2 in coverage but has one gap (normalize IDs on rm is not separately tested) and shares the quiet-still-performs-operation test that V2 lacks.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed (Go only) | 3 | 3 | 3 |
| Lines added (Go only) | 416 | 726 | 880 |
| Impl LOC (dep.go) | 152 | 170 | 185 |
| Test LOC (dep_test.go) | 262 | 554 | 692 |
| Test subtests | 16 | 25 | 23 |
| Top-level test functions | 1 | 3 | 1 |

## Verdict

**V2 is the best implementation.**

**Test quality (decisive factor):** V2 has the most comprehensive test coverage with 25 subtests organized into three logical groups (`TestDepAddCommand`, `TestDepRmCommand`, `TestDepSubcommandRouting`). It is the only version that tests every error path for both `add` AND `rm` independently. V1 has serious coverage gaps -- it only tests many error scenarios for `add` but not `rm` (task_id not found, quiet, normalize, fewer IDs, timestamp). V3 is close to V2 but lacks a separate normalize test for `rm` and uses a single monolithic test function.

**Unique V2 test strengths:** (a) The stale ref test explicitly verifies the spec's edge case that `rm` does not validate `blocked_by_id` as a task. (b) The duplicate-add no-mutation test verifies the `blocked_by` array is unchanged after a failed duplicate add, not just that an error occurred. (c) The persistence tests verify data survives to disk for both add and rm separately.

**Implementation quality:** V2's code is clean with good doc comments, though it uses non-idiomatic title-case error messages (`"Task '%s' not found"` instead of `"task '%s' not found"`). V1 has the best error message style (idiomatic lowercase) and the most efficient task lookup (map-based O(1) vs linear scan). V3 has the best structural design (tick dir discovery in parent, `DefaultTimestamps()` helper) but the most complex argument handling due to raw args.

**V1's weaknesses outweigh its strengths:** Despite having the cleanest implementation code (152 LOC, nil-ification of empty blocked_by, map-based lookups), V1's test suite has too many gaps to be the winner. Missing persistence tests and missing rm-specific error tests mean critical acceptance criteria are not verified for the rm subcommand.

**Recommendation:** Use V2's test suite structure and coverage as the baseline. Adopt V1's error message casing (lowercase) and map-based task lookup for efficiency. Consider V3's tick dir discovery pattern (discover once in parent).
