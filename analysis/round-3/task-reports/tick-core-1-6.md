# Task tick-core-1-6: tick create command

## Task Summary

This task implements the `tick create` command -- the first mutation command. It takes a title (required) and optional flags (`--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`), generates a unique ID, validates inputs, persists via the storage engine's Mutate flow, and outputs the created task details.

**Acceptance Criteria:**
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

| Criterion | V4 | V5 |
|-----------|-----|-----|
| 1. Creates task with correct defaults (open, priority 2) | PASS -- `NewTask()` sets `StatusOpen` and `defaultPriority` (2); tested in `TestCreate_WithOnlyTitle` and `TestCreate_DefaultPriority` | PASS -- `NewTask()` sets `StatusOpen` and `DefaultPriority` (2); tested in `TestCreate/"it creates a task with only a title"` and `"it sets default priority"` |
| 2. ID follows `tick-{6 hex}` format, unique | PASS -- `GenerateID()` with collision check; tested in `TestCreate_GeneratesUniqueID` with regex `^tick-[0-9a-f]{6}$` and two-task uniqueness check | PASS -- `GenerateID()` with collision check; tested in `TestCreate/"it generates a unique ID"` with regex |
| 3. All optional flags work | PASS -- All flags parsed in `parseCreateArgs()`; tested individually in `TestCreate_PriorityFlag`, `TestCreate_DescriptionFlag`, `TestCreate_BlockedBySingleID`, `TestCreate_BlockedByMultipleIDs`, `TestCreate_ParentFlag` | PASS -- All flags parsed in `parseCreateArgs()`; tested individually as subtests |
| 4. `--blocks` updates referenced tasks' `blocked_by` | PASS -- Mutate callback iterates tasks, appends new ID to target's `BlockedBy`, updates timestamp; tested in `TestCreate_BlocksUpdatesTargetTasks` verifying both `BlockedBy` and `Updated` | PASS -- Mutate callback uses index lookup via `existing[blockID]` to update target; tested in `"it updates target tasks' blocked_by when --blocks is used"` |
| 5. Missing/empty title returns error exit 1 | PASS -- Tested in `TestCreate_ErrorMissingTitle`, `TestCreate_ErrorEmptyTitle`, `TestCreate_ErrorWhitespaceOnlyTitle` | PASS -- Tested in subtests for missing, empty string, and whitespace-only title |
| 6. Invalid priority returns error exit 1 | PASS -- Validated via `ValidatePriority()` in `NewTask()`; tested in `TestCreate_RejectsPriorityOutsideRange` (table-driven: -1, 5, 100, "abc") | PASS -- Validated via `ValidatePriority()` called explicitly in `runCreate()`; tested with table-driven (-1, 5, 99) |
| 7. Non-existent IDs return error exit 1 | PASS -- Existence checked in Mutate callback; tested in `TestCreate_ErrorBlockedByNonExistent`, `TestCreate_ErrorBlocksNonExistent`, `TestCreate_ErrorParentNonExistent` | PASS -- Existence checked via `validateIDsExist()` helper; tested in three subtests |
| 8. Persisted via atomic write | PASS -- Uses `s.Mutate()`; tested in `TestCreate_PersistsViaAtomicWrite` (reads raw JSONL, verifies valid JSON) | PASS -- Uses `store.Mutate()`; tested in `"it persists the task to tasks.jsonl via atomic write"` |
| 9. SQLite cache updated | PASS -- Tested in `TestCreate_PersistsViaAtomicWrite` (checks `cache.db` exists after create) | PARTIAL -- No explicit test for cache.db existence; relies on storage engine integration |
| 10. Output shows task details | PASS -- Tested in `TestCreate_OutputsTaskDetails` (checks for ID pattern, title, status "open") | PASS -- Tested in `"it outputs full task details on success"` (checks for title and ID presence) |
| 11. `--quiet` outputs only task ID | PASS -- Tested in `TestCreate_QuietFlag` with regex `^tick-[0-9a-f]{6}$` on trimmed output | PASS -- Tested in `"it outputs only task ID with --quiet flag"` with same regex |
| 12. Input IDs normalized to lowercase | PASS -- `parseCommaSeparatedIDs()` calls `task.NormalizeID()`; tested in `TestCreate_NormalizesInputIDs` using `TICK-AAA111` | PASS -- `normalizeIDs()` calls `task.NormalizeID()`; tested in `"it normalizes input IDs to lowercase"` using `TICK-AAAAAA` |
| 13. Timestamps set to UTC ISO 8601 | PASS -- Tested in `TestCreate_TimestampsSetToUTC` (checks range, equality of created/updated, UTC location) | PARTIAL -- No dedicated timestamp test; timestamps are set by `NewTask()` which uses `time.Now().UTC().Truncate(time.Second)` but not explicitly verified in tests |

## Implementation Comparison

### Approach

Both versions create a new file `internal/cli/create.go` and `internal/cli/create_test.go`, and register the `create` subcommand in `cli.go`. The fundamental approach is identical: parse args for title and flags, validate inputs, open the storage engine, execute a Mutate callback that builds a lookup, validates references, creates the task, handles `--blocks`, and returns the modified task list.

**CLI Architecture -- Method vs Function:**

V4 uses a method on `*App`:
```go
// V4: internal/cli/create.go line 15
func (a *App) runCreate(args []string) error {
```
Registered in `cli.go` via `switch/case`:
```go
// V4: internal/cli/cli.go
case "create":
    if err := a.runCreate(subArgs); err != nil {
```

V5 uses a standalone function taking a `*Context`:
```go
// V5: internal/cli/create.go line 17
func runCreate(ctx *Context) error {
```
Registered via a command map:
```go
// V5: internal/cli/cli.go
var commands = map[string]func(*Context) error{
    "init":   runInit,
    "create": runCreate,
}
```

V5's command map pattern is more extensible -- adding a new command is one map entry vs a new switch case. This is a genuinely better architectural decision that pays off across later tasks.

**Storage Engine Import Path:**

V4 imports `github.com/leeovery/tick/internal/store` while V5 imports `github.com/leeovery/tick/internal/engine`. Both are the same storage engine concept from task 1-4 with different package names. This is a purely cosmetic difference.

**Task Construction -- Factory vs Field Assignment:**

V4 delegates task construction entirely to `task.NewTask()`:
```go
// V4: internal/cli/create.go lines 63-77
opts := &task.TaskOptions{
    Description: flags.description,
    BlockedBy:   flags.blockedBy,
    Parent:      flags.parent,
}
if flags.priority != nil {
    opts.Priority = flags.priority
}

exists := func(id string) bool { return existingIDs[id] }
newTask, err := task.NewTask(title, opts, exists)
```
V4's `NewTask()` (in `task.go`) handles title validation, priority validation, ID generation, self-reference checks, and struct construction all in one call. This is a "fat constructor" approach.

V5 separates validation from construction:
```go
// V5: internal/cli/create.go lines 22-28
title, err = task.ValidateTitle(title)
if err != nil {
    return err
}
if err := task.ValidatePriority(opts.priority); err != nil {
    return err
}
```
Then inside the Mutate callback:
```go
// V5: internal/cli/create.go lines 63-68
id, err := task.GenerateID(func(candidate string) bool {
    _, found := existing[candidate]
    return found
})

// V5: internal/cli/create.go lines 84-91
newTask := task.NewTask(id, title)
newTask.Priority = opts.priority
newTask.Description = opts.description
if len(blockedBy) > 0 {
    newTask.BlockedBy = blockedBy
}
newTask.Parent = parent
```

V5's `NewTask(id, title)` is a slim constructor that only sets defaults; all validation happens before the call. This is a "thin constructor + explicit setup" approach. Both patterns are valid Go. V5's approach makes validation errors return earlier (before opening the store), which is marginally better for user experience. V4's approach centralizes more logic in the domain layer, which is better for consistency if `NewTask()` is called from multiple places.

**ID Lookup Map Type:**

V4 uses `map[string]bool`:
```go
// V4: internal/cli/create.go line 36
existingIDs := make(map[string]bool, len(tasks))
```

V5 uses `map[string]int` (storing index):
```go
// V5: internal/cli/create.go line 50
existing := make(map[string]int, len(tasks))
for i, t := range tasks {
    existing[t.ID] = i
}
```

V5's index-based map enables direct `tasks[idx]` access for `--blocks` updates, avoiding the inner loop:
```go
// V5: internal/cli/create.go lines 96-100
for _, blockID := range blocks {
    idx := existing[blockID]
    tasks[idx].BlockedBy = append(tasks[idx].BlockedBy, id)
    tasks[idx].Updated = now
}
```

vs V4's nested loop:
```go
// V4: internal/cli/create.go lines 82-90
for i := range tasks {
    for _, blockTarget := range flags.blocks {
        if tasks[i].ID == blockTarget {
            tasks[i].BlockedBy = append(tasks[i].BlockedBy, newTask.ID)
            tasks[i].Updated = now
        }
    }
}
```

V5's approach is O(b) where b is the number of blocks targets; V4's is O(n*b) where n is the total task count. V5 is genuinely better for performance and clarity here.

**Normalization Placement:**

V4 normalizes IDs inside `parseCommaSeparatedIDs()` and the `--parent` case:
```go
// V4: internal/cli/create.go line 166
flags.parent = task.NormalizeID(strings.TrimSpace(args[i]))
```
```go
// V4: internal/cli/create.go lines 181-189
func parseCommaSeparatedIDs(s string) []string {
    ...
    ids = append(ids, task.NormalizeID(trimmed))
    ...
}
```

V5 normalizes IDs as a separate step inside the Mutate callback via a dedicated `normalizeIDs()` function:
```go
// V5: internal/cli/create.go lines 70-72
blockedBy := normalizeIDs(opts.blockedBy)
blocks := normalizeIDs(opts.blocks)
parent := task.NormalizeID(opts.parent)
```

V5's approach is cleaner separation of concerns -- parsing is pure parsing, normalization is a distinct step. V4 mixes normalization into parsing. This is a minor design difference favoring V5.

**Unknown Flag Handling:**

V5 rejects unknown flags:
```go
// V5: internal/cli/create.go line 154
case strings.HasPrefix(arg, "-"):
    return "", opts, fmt.Errorf("unknown flag '%s'", arg)
```

V4 has no explicit unknown flag check -- unknown flags would be treated as the title argument. This is a V5 advantage for robustness.

**Flags Struct:**

V4 uses `*int` for priority (nil means "not specified"):
```go
// V4: internal/cli/create.go line 102
type createFlags struct {
    priority    *int
```

V5 uses `int` with a default value:
```go
// V5: internal/cli/create.go lines 112-113
type createOpts struct {
    priority    int
```
```go
// V5: internal/cli/create.go line 124
opts := createOpts{
    priority: task.DefaultPriority,
}
```

V5's approach is simpler -- no pointer indirection needed. The default is set at parse time rather than requiring nil-checking later. This is a marginal V5 advantage in clarity.

**Output Format:**

V4 uses a method `(a *App) printTaskDetails(t *task.Task)`:
```go
// V4: internal/cli/create.go lines 197-219
fmt.Fprintf(a.Stdout, "ID:          %s\n", t.ID)
fmt.Fprintf(a.Stdout, "Title:       %s\n", t.Title)
```

V5 uses a function `printTaskDetails(w io.Writer, t task.Task)`:
```go
// V5: internal/cli/create.go lines 231-254
fmt.Fprintf(w, "ID:       %s\n", t.ID)
fmt.Fprintf(w, "Title:    %s\n", t.Title)
```

V5 takes an `io.Writer` parameter, which is more testable and follows Go conventions for output functions. V5 also formats Description and BlockedBy in a more structured multi-line layout with section headers, while V4 puts them on single lines. Both are acceptable Phase 1 placeholder formats.

V5 also uses `task.FormatTimestamp()` for timestamp output, reusing a shared helper, while V4 hardcodes the format string `"2006-01-02T15:04:05Z"`. V5 is DRYer here.

**Self-Reference Validation:**

V4 delegates self-reference checks to `NewTask()` which calls `ValidateBlockedBy()` and `ValidateParent()` internally. V5 calls these explicitly in the Mutate callback:
```go
// V5: internal/cli/create.go lines 75-79
if err := task.ValidateBlockedBy(id, blockedBy); err != nil {
    return nil, err
}
if err := task.ValidateParent(id, parent); err != nil {
    return nil, err
}
```

Additionally, V5's `ValidateBlockedBy()` and `ValidateParent()` use case-insensitive comparison:
```go
// V5: task.go
func ValidateBlockedBy(taskID string, blockedBy []string) error {
    normalizedID := NormalizeID(taskID)
    for _, dep := range blockedBy {
        if NormalizeID(dep) == normalizedID {
```

V4's versions use direct string equality without normalization. Since IDs are already normalized by the time they reach validation in both versions, this difference is academic in practice.

### Code Quality

**Error Handling:**

Both versions handle errors explicitly and return them up the call chain. Neither ignores errors.

V4 uses `fmt.Errorf("task '%s' not found (referenced in --blocked-by)", id)` directly inline. V5 extracts this into a reusable `validateIDsExist()` helper:
```go
// V5: internal/cli/create.go lines 205-212
func validateIDsExist(existing map[string]int, ids []string, flagName string) error {
    for _, id := range ids {
        if _, found := existing[id]; !found {
            return fmt.Errorf("task '%s' not found (referenced in %s)", id, flagName)
        }
    }
    return nil
}
```

V5's extraction is DRYer -- the same pattern is used for `--blocked-by`, `--blocks`, and `--parent` validation. V4 repeats the lookup-and-error pattern three times.

**Naming:**

V4: `createFlags`, `parseCreateArgs`, `parseCommaSeparatedIDs`, `printTaskDetails` -- all clear.
V5: `createOpts`, `parseCreateArgs`, `splitCSV`, `normalizeIDs`, `validateIDsExist`, `printTaskDetails` -- all clear. V5 has more granular function extraction, with `splitCSV` and `normalizeIDs` as separate concerns.

**Documentation:**

Both versions document all exported and unexported functions with godoc-style comments. V5 is slightly more descriptive in its function-level comments.

**Type Safety:**

V4 uses `*task.Task` (pointer) for `createdTask`:
```go
// V4: internal/cli/create.go line 33
var createdTask *task.Task
```

V5 uses value type `task.Task`:
```go
// V5: internal/cli/create.go line 46
var createdTask task.Task
```

V5's value type avoids potential nil pointer issues if the mutation callback doesn't execute. Both work correctly in practice.

### Test Quality

**V4 Test Functions (17 functions, 877 lines):**

1. `TestCreate_WithOnlyTitle` -- Creates task with title only, verifies all defaults (status, priority, description, blocked_by, parent, closed)
2. `TestCreate_WithAllOptionalFields` -- Uses all flags together, verifies priority, description, blocked_by, parent
3. `TestCreate_GeneratesUniqueID` -- Creates two tasks, verifies ID format regex and uniqueness
4. `TestCreate_SetsStatusOpen` -- Verifies status is "open"
5. `TestCreate_DefaultPriority` -- Verifies priority defaults to 2
6. `TestCreate_PriorityFlag` -- Verifies `--priority 0` works
7. `TestCreate_RejectsPriorityOutsideRange` -- Table-driven: -1, 5, 100, "abc" (4 cases)
8. `TestCreate_DescriptionFlag` -- Verifies `--description` works
9. `TestCreate_BlockedBySingleID` -- Single ID in `--blocked-by`
10. `TestCreate_BlockedByMultipleIDs` -- Multiple comma-separated IDs in `--blocked-by`
11. `TestCreate_BlocksUpdatesTargetTasks` -- Verifies `--blocks` adds to target's blocked_by AND updates timestamp
12. `TestCreate_ParentFlag` -- Verifies `--parent` works
13. `TestCreate_ErrorMissingTitle` -- No title arg, checks exit code 1, stderr contains "Title is required"
14. `TestCreate_ErrorEmptyTitle` -- Empty string title
15. `TestCreate_ErrorWhitespaceOnlyTitle` -- Whitespace-only title
16. `TestCreate_ErrorBlockedByNonExistent` -- Non-existent blocked-by ID, verifies no tasks written
17. `TestCreate_ErrorBlocksNonExistent` -- Non-existent blocks ID, verifies no tasks written
18. `TestCreate_ErrorParentNonExistent` -- Non-existent parent ID
19. `TestCreate_PersistsViaAtomicWrite` -- Reads raw JSONL, verifies valid JSON, checks cache.db exists
20. `TestCreate_OutputsTaskDetails` -- Output contains ID, title, "open"
21. `TestCreate_QuietFlag` -- Output is exactly one ID matching regex
22. `TestCreate_NormalizesInputIDs` -- Uses `TICK-AAA111`, verifies stored as lowercase
23. `TestCreate_TrimsWhitespaceFromTitle` -- Verifies leading/trailing whitespace stripped
24. `TestCreate_TimestampsSetToUTC` -- Checks timestamps within time range, created==updated, UTC location

Each is a separate top-level `Test` function with a single inner `t.Run()`. This is technically not table-driven -- each test stands alone. The tests exercise the full CLI path through `app.Run()`.

**V5 Test Functions (1 top-level function, 21 subtests, 519 lines):**

All subtests live under `TestCreate(t *testing.T)`:

1. `"it creates a task with only a title (defaults applied)"` -- Title, status, priority checks
2. `"it creates a task with all optional fields specified"` -- All flags, checks title, priority, description, blocked_by, parent
3. `"it generates a unique ID for the created task"` -- Regex check
4. `"it sets status to open on creation"` -- Status check
5. `"it sets default priority to 2 when not specified"` -- Default priority
6. `"it sets priority from --priority flag"` -- Priority 0
7. `"it rejects priority outside 0-4 range"` -- Table-driven: -1, 5, 99 (3 cases)
8. `"it sets description from --description flag"` -- Description check
9. `"it sets blocked_by from --blocked-by flag (single ID)"` -- Single blocked-by
10. `"it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)"` -- Multiple blocked-by
11. `"it updates target tasks' blocked_by when --blocks is used"` -- Blocks updates target
12. `"it sets parent from --parent flag"` -- Parent check
13. `"it errors when title is missing"` -- Exit code 1, "Title is required"
14. `"it errors when title is empty string"` -- Exit code 1
15. `"it errors when title is whitespace only"` -- Exit code 1
16. `"it errors when --blocked-by references non-existent task"` -- Exit 1, no partial mutation
17. `"it errors when --blocks references non-existent task"` -- Exit 1, no partial mutation
18. `"it errors when --parent references non-existent task"` -- Exit 1, no partial mutation
19. `"it persists the task to tasks.jsonl via atomic write"` -- Raw file contains title
20. `"it outputs full task details on success"` -- Title and ID in output
21. `"it outputs only task ID with --quiet flag"` -- Regex match
22. `"it normalizes input IDs to lowercase"` -- TICK-AAAAAA -> tick-aaaaaa
23. `"it trims whitespace from title"` -- Whitespace trimmed

**Test Gap Analysis:**

| Test Area | V4 | V5 |
|-----------|-----|-----|
| Timestamp verification | Has `TestCreate_TimestampsSetToUTC` checking range, equality, and timezone | Missing -- no timestamp test |
| Cache.db existence | Tested in `TestCreate_PersistsViaAtomicWrite` | Not tested |
| Non-number priority ("abc") | Tested as 4th case in table | Not tested (only numeric out-of-range) |
| Blocks timestamp update | Verified `targetTask.Updated.After(now)` | Not verified -- only checks `BlockedBy` content |
| Output contains "open" | Tested explicitly | Not tested (only checks title and ID) |
| Closed field is nil | Checked in `TestCreate_WithOnlyTitle` | Not checked |
| Two-task uniqueness | Creates 2 tasks, verifies different IDs | Only creates 1, checks format |

V4 has broader test coverage with more edge cases and deeper assertions. V5 has fewer tests (519 vs 877 lines) and misses several verification points.

**Test Structure:**

V4 uses separate top-level test functions, each with a single `t.Run()`. This means each test name is a separate function in the test file. It is clean but slightly verbose (each function has its own setup boilerplate).

V5 uses a single `TestCreate` function with all subtests as `t.Run()` children. This is a more compact organization but all subtests share the top-level function name.

Neither version uses true table-driven testing (a slice of test cases with a loop) for the main test suite, though both use table-driven for the priority rejection test.

**Test Infrastructure:**

V4 uses direct App construction and utility functions `setupInitializedDir()`, `setupInitializedDirWithTasks()`, `readTasksFromDir()` (which use `task.WriteJSONL`/`task.ReadJSONL`).

V5 uses `Run()` function call and `initTickProject()` (which actually runs `tick init`), `initTickProjectWithTasks()`, `readTasksFromFile()` (which manually parses JSON). V5's `initTickProject()` is more of an integration test approach since it exercises the real init command.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Error wrapping with `fmt.Errorf("%w", err)` | PARTIAL -- Uses `fmt.Errorf` for new errors but no `%w` wrapping of received errors in create.go (errors from `task.NewTask()` and store are returned directly, not wrapped) | PARTIAL -- Same pattern; returns errors directly without wrapping in create.go |
| Table-driven tests with subtests | PARTIAL -- Uses table-driven for priority rejection (4 cases); other tests are individual functions with `t.Run()` | PARTIAL -- Uses table-driven for priority rejection (3 cases); other tests are subtests under one parent |
| Explicit error handling (no ignored errors) | PASS -- All errors checked and returned | PASS -- All errors checked and returned |
| Exported function documentation | PASS -- All functions (exported and unexported) documented | PASS -- All functions documented |
| Context.Context for blocking operations | N/A -- No long-running blocking operations in create command itself (store handles locking) | N/A -- Same |
| No panic for error handling | PASS -- No panics | PASS -- No panics |
| No goroutines without lifecycle management | PASS -- No goroutines | PASS -- No goroutines |

### Spec-vs-Convention Conflicts

**Error message capitalization:**

The spec says: `Error: Title is required. Usage: tick create "<title>" [options]`

Both versions produce error messages that start with a lowercase letter after `Error:` in some cases (following Go convention that error strings should not be capitalized). Specifically, V4's `task.ValidateTitle()` returns `"title is required and cannot be empty"` while the spec wants `"Title cannot be empty"`. V5 has the same pattern. Both versions correctly capitalize the "Title is required" message from `parseCreateArgs()`:
```go
// Both versions:
return "", nil, fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
```

This is a reasonable judgment call -- the user-facing error from `parseCreateArgs` matches the spec exactly, while the validation-level error from the domain layer follows Go convention. Both handle this identically.

**Priority as `*int` vs `int`:**

The spec says `--priority <0-4>: integer, default 2`. V4 uses `*int` to distinguish "not provided" from "provided as 0", passing the distinction to `task.NewTask()`. V5 uses `int` with the default set at parse time. Since the default is well-defined (2), both approaches correctly implement the spec. Neither deviates from it.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 5 (2 docs, 3 code) | 5 (2 docs, 3 code) |
| Lines added (total) | 1105 | 788 |
| Impl LOC (create.go) | 219 | 264 |
| Test LOC (create_test.go) | 877 | 519 |
| Test functions | 24 (24 t.Run subtests across 24 top-level functions) | 23 (23 t.Run subtests under 1 top-level function) |
| cli.go lines changed | +6 | +2/-1 |

V5's implementation is 45 lines longer despite fewer tests because it separates validation from construction, adds the `validateIDsExist()` helper, the `normalizeIDs()` helper, the `splitCSV()` function, and unknown flag detection. V4 is more compact in implementation by delegating more to `task.NewTask()`.

V4's tests are 358 lines longer due to more thorough assertions (timestamp range checking, cache.db verification, closed-field nil check, two-task uniqueness check, non-numeric priority test) and the boilerplate of separate top-level functions.

## Verdict

**V4 wins narrowly, primarily on test coverage.**

V5 has several genuine implementation advantages:
- Command map dispatch is more extensible than switch/case
- `map[string]int` for index-based lookup eliminates O(n*b) nested loop for `--blocks`
- `validateIDsExist()` helper is DRYer than repeated inline checks
- Unknown flag rejection is more robust
- `io.Writer` parameter on `printTaskDetails` is more idiomatic
- Validation before store opening fails faster for simple input errors
- Uses `task.FormatTimestamp()` instead of hardcoded format string

However, V4's test suite is significantly more thorough:
- Timestamp range and UTC verification (`TestCreate_TimestampsSetToUTC`) -- V5 has no equivalent
- Cache.db existence verification -- V5 doesn't test this acceptance criterion
- Non-numeric priority value ("abc") -- V5 only tests numeric out-of-range
- `--blocks` timestamp update verification -- V5 doesn't verify `Updated` was refreshed
- Two-task uniqueness check -- V5 only checks format, not uniqueness
- Closed field nil check -- V5 doesn't verify this
- Output contains "open" status -- V5 doesn't verify this in output tests

Test coverage is critical for a mutation command since `tick create` is the first write operation and validates the entire Mutate flow end-to-end. V4's 877-line test suite catches more potential regressions than V5's 519-line suite.

V5's implementation code is genuinely cleaner in structure, but the test coverage gaps -- particularly the missing timestamp test and cache verification -- mean V4 provides stronger confidence that the acceptance criteria are actually met. A version combining V5's implementation approach with V4's test thoroughness would be ideal.
