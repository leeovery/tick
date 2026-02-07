# Task tick-core-1-6: tick create command

## Task Summary

This task implements the `tick create` command -- the first mutation command in the CLI. It takes a title (required) and optional flags (`--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`), generates a unique ID, validates inputs, persists the task via the storage engine's `Mutate` flow, and outputs the created task details.

### Acceptance Criteria (from plan)

1. `tick create "<title>"` creates task with correct defaults (status: open, priority: 2)
2. Generated ID follows `tick-{6 hex}` format, unique among existing tasks
3. All optional flags work: `--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`
4. `--blocks` correctly updates referenced tasks' `blocked_by` arrays
5. Missing or empty title returns error to stderr with exit code 1
6. Invalid priority returns error with exit code 1
7. Non-existent IDs in references return error with exit code 1
8. Task persisted via atomic write through storage engine
9. SQLite cache updated as part of write flow
10. Output shows task details on success
11. `--quiet` outputs only task ID
12. Input IDs normalized to lowercase
13. Timestamps set to current UTC ISO 8601

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Defaults (open, priority 2) | PASS - `task.NewTask(id, title, priority)` with sentinel -1 mapping to default | PASS - `task.NewTask(trimmedTitle, opts, existsFn)` with `*int` nil = default 2 | PASS - `task.DefaultPriority()` for default, direct struct build |
| 2. ID format tick-{6 hex} | PASS - `task.GenerateID(existsFn)` with bool collision check | PASS - `task.NewTask` calls `task.GenerateID` internally via existsFn | PASS - `task.GenerateID` returns (string, error) with `(bool, error)` existsFn |
| 3. All optional flags | PASS - All 5 flags parsed | PASS - All 5 flags parsed | PASS - All 5 flags parsed |
| 4. --blocks updates target tasks | PASS - Iterates `taskMap`, appends ID, updates timestamp | PASS - Iterates `modified` slice, matches by normalized ID, appends ID | PASS - Iterates `tasks` slice, appends ID, updates timestamp |
| 5. Missing/empty title error | PASS - Checks positional args empty + `ValidateTitle` | PASS - `titleProvided` bool + explicit "Title cannot be empty" | PASS - Two-stage: empty string check then `TrimTitle` + empty check |
| 6. Invalid priority error | PASS - `task.ValidatePriority(p)` before store | PASS - `task.ValidatePriority(*flags.priority)` inside Mutate | PASS - `task.ValidatePriority(flags.Priority)` before store |
| 7. Non-existent ID error | PASS - taskMap lookup for blocked-by, blocks, parent | PASS - existingIDs map lookup for all three | PASS - idSet map lookup for all three |
| 8. Atomic write via storage engine | PASS - `store.Mutate()` | PASS - `store.Mutate()` | PASS - `store.Mutate()` |
| 9. SQLite cache updated | PASS - Implicit via `store.Mutate` | PASS - Implicit via `store.Mutate` | PASS - Implicit via `store.Mutate` |
| 10. Output task details | PASS - `printTaskBasic` | PASS - `printTaskDetails` | PASS - `printTaskDetails` |
| 11. --quiet outputs ID only | PASS - `fmt.Fprintln(a.stdout, createdTask.ID)` | PASS - `fmt.Fprintln(a.stdout, createdTask.ID)` | PASS - `fmt.Fprintln(a.Stdout, createdTask.ID)` |
| 12. IDs normalized lowercase | PASS - `task.NormalizeID()` on parse | PASS - `task.NormalizeID()` on parse | PASS - `normalizeIDs()` helper + `task.NormalizeID()` after parse |
| 13. Timestamps UTC ISO 8601 | PASS - `task.NewTask` sets timestamps internally | PASS - `task.NewTask` sets timestamps internally | PASS - `task.DefaultTimestamps()` explicit call |

## Implementation Comparison

### Approach

All three versions follow the same high-level architecture: parse flags from args, validate title, open storage, execute a `store.Mutate()` callback that validates references / generates ID / builds the task / applies `--blocks` side effects, then output. The differences are in *how* they structure the parsing, validation ordering, and App integration.

#### CLI Integration (command dispatch)

**V1** modifies `cli.go`'s `Run` method to add `case "create"` and refactors `parseGlobalFlags` significantly. The original code parsed global flags and subcommand in a single loop; V1 rewrites this to use a `foundSubcmd` boolean so that after the subcommand is identified, all subsequent args become `remaining`:

```go
// V1 cli.go parseGlobalFlags rewrite
foundSubcmd := false
for i := 0; i < len(args); i++ {
    arg := args[i]
    if !foundSubcmd {
        switch arg {
        case "--quiet", "-q":
            a.opts.Quiet = true
            continue
        // ... other global flags
        }
    }
    if !foundSubcmd && !strings.HasPrefix(arg, "-") {
        subcmd = arg
        foundSubcmd = true
    } else if foundSubcmd {
        remaining = append(remaining, arg)
    }
}
```

V1 also removes `_ = cmdArgs` placeholder from the original init-only code. V1 passes `(workDir, cmdArgs)` to `cmdCreate`.

**V2** modifies `app.go`'s `parseGlobalFlags` to return 3 values `(string, []string, error)` instead of 2 `(string, error)`, slicing `args[i+1:]` when the subcommand is found:

```go
// V2 app.go parseGlobalFlags change
default:
    // First non-flag argument is the subcommand; rest are command args
    return arg, args[i+1:], nil
```

V2 also adds `create` to the usage help string. V2 passes `cmdArgs` only to `runCreate(cmdArgs)` -- the workDir is read from `a.workDir`.

**V3** makes the smallest change to `cli.go` -- just adds the `case "create"` and passes the full `args` slice:

```go
// V3 cli.go
case "create":
    return a.runCreate(args)
```

V3 also adds `create` to the help text. V3 passes the *entire* `args` slice (including "tick" and "create" at indices 0 and 1), and `parseCreateArgs` skips from index 2.

#### Command Entry Point Signature

- **V1**: `func (a *App) cmdCreate(workDir string, args []string) error` -- receives workDir as param, returns error
- **V2**: `func (a *App) runCreate(args []string) error` -- uses `a.workDir`, returns error
- **V3**: `func (a *App) runCreate(args []string) int` -- returns exit code directly

This is a significant design difference. V1 and V2 return `error`, relying on the caller (`Run`) to print to stderr and return exit code 1. V3 returns `int` and handles all error formatting internally via `fmt.Fprintf(a.Stderr, "Error: %s\n", err)` at each error point. V3's approach means every error path must include the `fmt.Fprintf` + `return 1` boilerplate, which is more verbose but gives finer control.

#### Flag Parsing Structure

**V1** does inline parsing within `cmdCreate` -- no separate type or function. Flags are local variables:

```go
// V1 create.go
var title, description, parent string
var blockedBy, blocks []string
priority := -1 // sentinel: use default
var positional []string
for i := 0; i < len(args); i++ {
    switch args[i] {
    case "--priority":
        // ...
```

**V2** defines a `createFlags` struct and a separate `parseCreateArgs` function (exported-name pattern but actually unexported):

```go
// V2 create.go
type createFlags struct {
    title         string
    titleProvided bool
    priority      *int
    description   string
    blockedBy     []string
    blocks        []string
    parent        string
}

func parseCreateArgs(args []string) (*createFlags, error) {
```

V2 uses `*int` for priority (nil = not provided) and a `titleProvided` boolean. V2 also first collects non-flag args (breaking on `--` prefix), then parses flags in a second loop. This means flags must come after the title. V2 returns an error for unknown flags (`return nil, fmt.Errorf("unknown flag %q for create command", arg)`).

**V3** defines an exported `CreateFlags` struct and a separate `parseCreateArgs` function:

```go
// V3 create.go
type CreateFlags struct {
    Priority    int
    Description string
    BlockedBy   []string
    Blocks      []string
    Parent      string
}

func parseCreateArgs(args []string) (CreateFlags, string, error) {
```

V3 starts parsing from index 2 (skipping "tick" and "create"), initializes Priority from `task.DefaultPriority()`, and returns the title as a separate value. V3 silently skips unknown flags (`default: i++`) rather than erroring. V3 also has a dedicated `normalizeIDs` helper function.

#### ID Normalization Timing

- **V1**: Normalizes IDs *during* flag parsing (`task.NormalizeID(id)` inline in each flag handler)
- **V2**: Same as V1 -- normalizes inline during parsing
- **V3**: Normalizes *after* parsing, in `runCreate`:
  ```go
  flags.BlockedBy = normalizeIDs(flags.BlockedBy)
  flags.Blocks = normalizeIDs(flags.Blocks)
  flags.Parent = task.NormalizeID(flags.Parent)
  ```

#### Validation Ordering

- **V1**: Title validation before store open, priority validation during parse, reference validation inside Mutate, self-reference validation inside Mutate after ID generation
- **V2**: Title validation (presence + trim + empty) before store open, priority validation inside Mutate, reference validation inside Mutate
- **V3**: Title validation (empty + trim + empty + `ValidateTitle`) before store open, priority validation before store open, reference validation inside Mutate

V2 is the only version that defers priority validation into the Mutate callback. V3 is the only version that validates priority *before* opening the store, which is slightly more efficient for invalid input.

#### Task Construction

**V1** uses `task.NewTask(id, title, priority)` then mutates fields:

```go
newTask := task.NewTask(id, title, priority)
newTask.Description = description
newTask.Parent = parent
if len(blockedBy) > 0 {
    newTask.BlockedBy = blockedBy
}
```

**V2** uses `task.NewTask(trimmedTitle, opts, existsFn)` with a `TaskOptions` struct, where `NewTask` handles ID generation internally:

```go
opts := &task.TaskOptions{
    Priority:    flags.priority,
    Description: flags.description,
    BlockedBy:   flags.blockedBy,
    Parent:      flags.parent,
}
newTask, err := task.NewTask(trimmedTitle, opts, existsFn)
```

**V3** builds the `task.Task` struct directly:

```go
newTask := task.Task{
    ID:          newID,
    Title:       title,
    Status:      task.StatusOpen,
    Priority:    flags.Priority,
    Description: flags.Description,
    BlockedBy:   flags.BlockedBy,
    Parent:      flags.Parent,
    Created:     created,
    Updated:     updated,
}
```

V2's approach has the best encapsulation -- the `task` package controls ID generation and defaults. V1 is a middle ground. V3 is the most explicit but leaks construction details into the CLI layer.

#### --blocks Implementation

All three correctly iterate target tasks and append the new task's ID:

**V1** uses the `taskMap` (map of pointers to original slice elements):
```go
for _, blockID := range blocks {
    t := taskMap[blockID]
    t.BlockedBy = append(t.BlockedBy, id)
    t.Updated = now
}
```

**V2** iterates the `modified` slice (after appending the new task):
```go
for i := range modified {
    normalizedID := task.NormalizeID(modified[i].ID)
    for _, blocksID := range flags.blocks {
        if normalizedID == blocksID {
            modified[i].BlockedBy = append(modified[i].BlockedBy, newTask.ID)
            modified[i].Updated = now
            break
        }
    }
}
```

**V3** iterates the original `tasks` slice (before appending):
```go
for i := range tasks {
    for _, blocksID := range flags.Blocks {
        if tasks[i].ID == blocksID {
            tasks[i].BlockedBy = append(tasks[i].BlockedBy, newTask.ID)
            tasks[i].Updated = updated
        }
    }
}
```

V1's approach is O(b) where b = len(blocks) since it uses map lookups. V2 and V3 are O(n*b) since they iterate all tasks for each blocks ID. V2 additionally normalizes IDs during iteration (redundant since they should already be normalized). V3 does not `break` out of the inner loop, which is a minor inefficiency but harmless since IDs are unique.

#### Output Formatting

All three produce similar key-value output. Differences:

- **V1** (`printTaskBasic`): Prints Created only. Does not print Updated.
- **V2** (`printTaskDetails`): Takes `*task.Task` (pointer). Prints both Created and Updated. Has its own `formatTime` helper.
- **V3** (`printTaskDetails`): Takes `task.Task` (value). Prints both Created and Updated. Uses `t.Created` directly (string field).

V1's omission of the Updated field in output is a minor gap. V2 has a dedicated `formatTime` function. V3 relies on the task's timestamp already being a formatted string.

### Code Quality

#### Go Idioms & Naming

**V1**: Uses `cmdCreate` naming (prefix `cmd`). Uses `printTaskBasic`. Private fields, no separate types for flags. Method `a.opts.Quiet` for quiet check. Uses `task.FormatTimestamp` from the task package.

**V2**: Uses `runCreate` naming (prefix `run`). Defines `createFlags` struct (unexported, good). Has `titleProvided bool` for explicit presence tracking. Uses `a.config.Quiet`. Uses pointer receiver for task output (`*task.Task`). Most idiomatic Go naming overall.

**V3**: Uses `runCreate` naming. Defines `CreateFlags` struct (exported -- unusual for an internal package struct used only in one file). Uses `a.flags.Quiet`. Returns `int` instead of `error` from the command handler, which is less idiomatic Go (errors are preferred over status codes in internal APIs). However, it matches the `Run` method's return type.

#### Error Handling

**V1**: Returns `error` from `cmdCreate`. Error messages use `fmt.Errorf("task '%s' not found", depID)`. Clean single-point error formatting in the caller.

**V2**: Returns `error` from `runCreate`. Error messages include context: `fmt.Errorf("task %q not found (referenced in --blocked-by)", id)`. Best error messages of the three -- the user knows *which flag* caused the error.

**V3**: Returns `int` with inline error formatting. Error messages are `fmt.Errorf("task '%s' not found", id)` (same as V1 but without flag context). The inline pattern requires 2 lines per error point:
```go
fmt.Fprintf(a.Stderr, "Error: %s\n", err)
return 1
```
This is repeated 8 times in `runCreate`, which is verbose.

#### DRY

**V1**: Most compact implementation. No helper types or functions beyond `printTaskBasic` and `extractID` in tests.

**V2**: Separates `parseCreateArgs` and `formatTime` as standalone functions. The `createFlags` struct cleanly separates parsing from execution.

**V3**: Separates `parseCreateArgs`, `normalizeIDs`, and `printTaskDetails`. The `normalizeIDs` helper is good DRY practice. However, the repeated anonymous struct type in tests (7 times!) is a significant DRY violation:
```go
var newTask *struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    // ... 8 more fields
}
```
This struct is defined inline in 7 different test cases instead of once.

#### Type Safety

**V2** has the strongest type safety: `*int` for optional priority (nil vs zero is distinguishable), `titleProvided bool` for explicit presence. V1 uses `-1` sentinel for priority. V3 uses `task.DefaultPriority()` which defaults to 2, making it impossible to distinguish "user didn't specify" from "user specified 2" (though this is harmless in practice since 2 is the default).

### Test Quality

#### V1 Test Functions (16 tests)

1. `creates a task with only a title` - Checks output contains ID and title
2. `sets status to open on creation` - Checks output contains "open"
3. `sets default priority to 2` - Reads JSONL, checks `"priority":2`
4. `sets priority from --priority flag` - Creates with priority 0, reads JSONL
5. `rejects priority outside 0-4 range` - Tests priority 5 only
6. `sets description from --description flag` - Reads JSONL for description
7. `sets blocked_by from --blocked-by flag` - Creates dependency chain, reads JSONL
8. `sets blocked_by from --blocked-by with multiple IDs` - Comma-separated IDs
9. `updates target tasks' blocked_by when --blocks is used` - Verifies target modified
10. `sets parent from --parent flag` - Reads JSONL for parent field
11. `errors when title is missing` - No args, checks exit 1 + "Title is required"
12. `errors when title is empty string` - Empty string, checks exit 1
13. `errors when --blocked-by references non-existent task` - Checks "not found"
14. `errors when --parent references non-existent task` - Checks "not found"
15. `persists task to tasks.jsonl` - Reads raw file
16. `outputs only task ID with --quiet flag` - Verifies length=11, prefix "tick-"
17. `normalizes input IDs to lowercase` - Uses uppercased ID
18. `trims whitespace from title` - Checks output and JSONL

Helper: `extractID` parses output to find tick-XXXXXX pattern.

**Missing from V1**: No test for `--blocks` with non-existent task. No test for "all optional fields specified". No test for "generates unique ID" standalone. No test for whitespace-only title. No test for negative priority. No test for timestamps. No test that verifies no partial mutation on error. No "outputs full task details on success" test.

#### V2 Test Functions (20 tests)

1. `it creates a task with only a title (defaults applied)` - Parses JSONL, checks title/status/priority
2. `it creates a task with all optional fields specified` - Pre-populates 2 tasks, creates with all flags
3. `it generates a unique ID for the created task` - Checks prefix + length
4. `it sets status to open on creation` - Reads JSONL status
5. `it sets default priority to 2 when not specified` - Reads JSONL priority
6. `it sets priority from --priority flag` - Priority 0
7. `it rejects priority outside 0-4 range` - Tests priority 5 AND negative -1
8. `it sets description from --description flag` - Reads JSONL
9. `it sets blocked_by from --blocked-by flag (single ID)` - Pre-populated task, reads JSONL
10. `it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)` - 2 blockers
11. `it updates target tasks' blocked_by when --blocks is used` - Checks target + updated timestamp
12. `it sets parent from --parent flag` - Reads JSONL
13. `it errors when title is missing` - Checks "Title is required"
14. `it errors when title is empty string` - Checks "Title cannot be empty"
15. `it errors when title is whitespace only` - Checks "Title cannot be empty"
16. `it errors when --blocked-by references non-existent task` - Checks error message + no partial mutation
17. `it errors when --blocks references non-existent task` - Checks error message + no partial mutation
18. `it errors when --parent references non-existent task` - Checks error message + no partial mutation
19. `it persists the task to tasks.jsonl via atomic write` - Reads back and checks title + timestamps exist
20. `it outputs full task details on success` - Checks output for ID/title/status
21. `it outputs only task ID with --quiet flag` - Checks prefix + length=11
22. `it normalizes input IDs to lowercase` - Uses "TICK-AAA111"
23. `it trims whitespace from title` - Reads JSONL

Helpers: `setupInitializedTickDir`, `setupTickDirWithContent`, `readTasksJSONL` (uses `json.Unmarshal` into `map[string]interface{}`).

**V2 unique tests**: "all optional fields specified", "generates unique ID", "whitespace-only title", "--blocks non-existent task", "outputs full task details", negative priority (-1). V2 also verifies "no partial mutation" on all three reference error tests.

#### V3 Test Functions (22 tests)

1. `it creates a task with only a title (defaults applied)` - Checks title/status/priority
2. `it creates a task with all optional fields specified` - Uses setupTask, checks all fields
3. `it generates a unique ID for the created task` - Checks prefix + length
4. `it sets status to open on creation` - Checks status
5. `it sets default priority to 2 when not specified` - Checks priority
6. `it sets priority from --priority flag` - Priority 0
7. `it rejects priority outside 0-4 range` - Priority 5, checks no task created
8. `it sets description from --description flag` - Checks description
9. `it sets blocked_by from --blocked-by flag (single ID)` - Pre-populated task
10. `it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)` - 3 blockers
11. `it updates target tasks' blocked_by when --blocks is used` - Checks target + updated timestamp
12. `it sets parent from --parent flag` - Pre-populated task
13. `it errors when title is missing` - Checks "Title is required"
14. `it errors when title is empty string` - Checks "Title"
15. `it errors when title is whitespace only` - Checks "Title cannot be empty"
16. `it errors when --blocked-by references non-existent task` - Checks "not found" + no partial mutation
17. `it errors when --blocks references non-existent task` - Checks "not found" + no partial mutation
18. `it errors when --parent references non-existent task` - Checks "not found" + no partial mutation
19. `it persists the task to tasks.jsonl via atomic write` - Raw file read
20. `it outputs full task details on success` - Checks output for ID:/Title:/Status: labels
21. `it outputs only task ID with --quiet flag` - Checks prefix + no Title:/Status: in output
22. `it normalizes input IDs to lowercase` - Uses "TICK-BLOCKER"
23. `it trims whitespace from title` - Checks title
24. `it sets timestamps to current UTC ISO 8601` - Checks Created==Updated, contains T and Z

Helpers: `setupTickDir`, `readTasksFromDir` (uses `storage.ReadJSONL` -- actual storage package), `setupTask` (writes raw JSONL).

**V3 unique tests**: "timestamps to current UTC ISO 8601" (unique to V3). V3 also checks `code != 1` (exit code) directly since its `runCreate` returns `int`. V3 verifies "no task created" on invalid priority (unique). V3 tests 3 comma-separated IDs for blocked-by (others test 2). V3's `--quiet` test also verifies absence of "Title:" and "Status:" in output (others just check length/prefix).

#### Test Infrastructure Comparison

- **V1**: Uses `initTickDir` which runs `tick init` via `app.Run`. Tests interact through the full CLI pipeline. `createTask` helper returns stdout/stderr/code. Parses JSONL with raw `os.ReadFile` + string matching.
- **V2**: Uses `setupInitializedTickDir` (manual `.tick/` dir creation) and `setupTickDirWithContent` (pre-populated JSONL). `readTasksJSONL` parses to `map[string]interface{}` -- flexible but loses type safety. Tests call `app.Run()` directly (no helper for create).
- **V3**: Uses `setupTickDir` (manual dir creation) and `setupTask` (appends raw JSONL lines). `readTasksFromDir` calls `storage.ReadJSONL` -- the real storage reader -- giving strongest integration testing. Uses actual `task.Task` struct types. But repeats the anonymous struct 7 times in test cases.

V1's approach of running `tick init` is the most realistic integration test. V2 and V3 skip init and write directly, which is faster but tests less of the real flow. V3's use of `storage.ReadJSONL` in tests couples tests to the storage package but verifies real serialization.

#### Test Coverage Gap Analysis

| Edge Case from Plan | V1 | V2 | V3 |
|---------------------|-----|-----|-----|
| Title defaults applied | YES | YES | YES |
| All optional fields | NO | YES | YES |
| Unique ID generated | NO (implicit) | YES | YES |
| Status open | YES | YES | YES |
| Default priority 2 | YES | YES | YES |
| Priority from flag | YES | YES | YES |
| Priority outside range | Partial (5 only) | YES (5 and -1) | Partial (5 only) |
| Description flag | YES | YES | YES |
| blocked-by single | YES | YES | YES |
| blocked-by multiple | YES | YES | YES (3 IDs) |
| --blocks updates target | YES | YES | YES |
| Parent flag | YES | YES | YES |
| Missing title | YES | YES | YES |
| Empty title | YES | YES | YES |
| Whitespace-only title | NO | YES | YES |
| blocked-by non-existent | YES | YES | YES |
| blocks non-existent | NO | YES | YES |
| parent non-existent | YES | YES | YES |
| Persists to JSONL | YES | YES | YES |
| Full output on success | NO | YES | YES |
| Quiet flag | YES | YES | YES |
| Normalizes IDs | YES | YES | YES |
| Trims title | YES | YES | YES |
| Timestamps UTC ISO 8601 | NO | Partial (exists check) | YES |
| No partial mutation on error | NO | YES (3 tests) | YES (3 tests) |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 | 5 | 5 (3 code) |
| Lines added | 509 (+486 net) | 887 (+878 net) | 1111 (+1111 net) |
| Impl LOC (create.go) | 193 | 250 | 259 |
| Test LOC (create_test.go) | 288 | 623 | 849 |
| Test functions | 16 | 20 | 22 |
| cli.go/app.go changes | +36/-23 (refactor) | +18/-9 | +3/-0 |

Note: V3 line count includes a committed binary (`tick` at 7MB) and docs files. Code-only diff is 3 files, 1111 additions.

## Verdict

**V2 is the best implementation.**

**Evidence:**

1. **Code architecture**: V2 has the cleanest separation between flag parsing (`parseCreateArgs` returning a typed struct), validation, and execution. The `createFlags` struct with `*int` for optional priority and `titleProvided bool` is the most type-safe representation. V2's error messages include flag context (`"task %q not found (referenced in --blocked-by)"`), which is superior for UX.

2. **Test quality**: V2 covers 20 of the 22 plan-specified test cases (missing only the explicit timestamp format test). V2 is the only version that tests both positive and negative out-of-range priority. V2 tests verify no partial mutation on all three reference error cases. V2's `setupTickDirWithContent` helper enables clean test data setup without coupling to the full CLI pipeline.

3. **CLI integration**: V2's change to `parseGlobalFlags` returning `(string, []string, error)` is a clean, minimal refactor. V1's larger `parseGlobalFlags` rewrite introduces more risk. V3's approach of passing the full `args` slice is the simplest but forces `parseCreateArgs` to know about program name position.

4. **Error handling pattern**: V2 returns `error` from `runCreate` (idiomatic Go), letting the `Run` method handle stderr formatting. V3's approach of returning `int` with inline error formatting repeats `fmt.Fprintf(a.Stderr, ...)` + `return 1` eight times. V1 also returns `error` but has slightly less informative messages.

5. **Minor V2 disadvantage**: V2's `--blocks` implementation iterates the full `modified` slice with redundant `NormalizeID` calls (O(n*b) vs V1's O(b) with map lookup). This is a minor performance difference that would only matter with very large task lists.

**V3** is a close second with the most comprehensive test suite (22 tests, including unique timestamp validation) but is held back by the exported `CreateFlags` type in an internal package, the verbose inline error handling pattern, repeated anonymous struct definitions in tests (7 occurrences), and silently skipping unknown flags. V3 also commits a 7MB binary, which is a process issue.

**V1** is the weakest: fewest tests (16, missing 6 from the plan), no test for `--blocks` non-existent task, no whitespace-only title test, no timestamp test, and no partial-mutation-prevention verification. However, V1's implementation code is the most compact and its `--blocks` implementation via `taskMap` pointer is the most efficient.
