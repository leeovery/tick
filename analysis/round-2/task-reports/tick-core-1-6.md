# Task tick-core-1-6: tick create Command

## Task Summary

This task implements the `tick create` command -- the first mutation command in the CLI. It takes a title (required) and optional flags (`--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`), generates a `tick-{6 hex}` ID with collision check, validates all inputs, persists via the storage engine's `Mutate` write flow (exclusive lock, JSONL read, mutation, atomic write, cache update), and outputs the created task details.

**Acceptance Criteria (from plan):**
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

| Criterion | V2 | V4 |
|-----------|-----|-----|
| 1. Default status open, priority 2 | PASS -- tested in "creates a task with only a title" | PASS -- tested in `TestCreate_WithOnlyTitle`, also verifies empty description, empty blocked_by, empty parent, nil closed |
| 2. ID format `tick-{6 hex}`, unique | PASS -- test checks prefix "tick-" and length 11 | PASS -- test uses regex `^tick-[0-9a-f]{6}$` and creates two tasks verifying distinct IDs |
| 3. All optional flags | PASS -- tested individually and combined | PASS -- tested individually; combined test also uses --blocks |
| 4. --blocks updates target blocked_by | PASS -- test verifies target task's blocked_by contains new ID and updated timestamp changed | PASS -- same verification plus `targetTask.Updated.After(now)` time check |
| 5. Missing/empty title error exit 1 | PASS -- error returned from `app.Run()`, tests check error message | PASS -- tests check exit code == 1 and stderr contains "Error:" |
| 6. Invalid priority error exit 1 | PASS -- tests priority=5 and priority=-1 | PASS -- table-driven test covers -1, 5, 100, "abc" |
| 7. Non-existent ID references error exit 1 | PASS -- blocked-by, blocks, parent all tested; no-partial-mutation verified for blocked-by and blocks | PASS -- all three tested; no-partial-mutation verified for blocked-by and blocks |
| 8. Atomic write via storage engine | PASS -- uses `store.Mutate()` | PASS -- uses `s.Mutate()` |
| 9. SQLite cache updated | PASS (implicit -- storage engine handles this) | PASS -- test explicitly verifies `cache.db` file exists after create |
| 10. Output shows task details | PASS -- test verifies output contains ID, title, status | PASS -- test verifies output contains ID (regex), title, status |
| 11. --quiet outputs only ID | PASS -- test verifies prefix "tick-" and length 11 | PASS -- test uses regex `^tick-[0-9a-f]{6}$` for stricter validation |
| 12. IDs normalized to lowercase | PASS -- test uses "TICK-AAA111" in --blocked-by, verifies stored as "tick-aaa111" | PASS -- identical test scenario |
| 13. Timestamps set to UTC ISO 8601 | PASS (implicit -- `time.Now().UTC().Truncate(time.Second)`) | PASS -- dedicated `TestCreate_TimestampsSetToUTC` test verifies range, equality, and timezone |

## Implementation Comparison

### Approach

Both versions follow the same high-level architecture: a `runCreate` method on the `App` struct that parses args, validates the title, discovers the `.tick/` directory, opens the store, executes a `Mutate` callback that validates references and creates the task, then outputs results. The differences are in code organization, parsing strategy, and the CLI framework integration.

**CLI Integration (app.go / cli.go):**

V2 modified `parseGlobalFlags` to return a third value -- the remaining args after the subcommand:

```go
// V2: internal/cli/app.go
func (a *App) parseGlobalFlags(args []string) (string, []string, error) {
    // ...
    // First non-flag argument is the subcommand; rest are command args
    return arg, args[i+1:], nil
}
```

This is a refactor of the existing function signature from `(string, error)` to `(string, []string, error)`. The `create` case in `Run` then passes `cmdArgs` to `runCreate`.

V4 simply added a new `case "create"` block in `Run`, relying on an existing `subArgs` variable that was already being extracted in the V4 CLI framework:

```go
// V4: internal/cli/cli.go
case "create":
    if err := a.runCreate(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

V4's approach is less invasive -- only 6 lines added to `cli.go` with no changes to existing function signatures. V2's refactoring of `parseGlobalFlags` touches the existing code more deeply (18 lines changed in `app.go`, altering the return signature).

**Argument Parsing:**

V2 uses a standalone `parseCreateArgs(args []string) (*createFlags, error)` function (package-level, not a method). It separates title extraction from flag parsing in two phases: first it scans for the first non-flag argument as the title, then iterates flags:

```go
// V2: create.go lines 28-92
func parseCreateArgs(args []string) (*createFlags, error) {
    flags := &createFlags{}
    i := 0
    // First non-flag arg is the title
    for i < len(args) {
        arg := args[i]
        if strings.HasPrefix(arg, "--") {
            break
        }
        if !flags.titleProvided {
            flags.title = arg
            flags.titleProvided = true
        }
        i++
    }
    // Parse command-specific flags
    for i < len(args) {
        // ... switch on flag names
    }
}
```

V2's `createFlags` struct includes `title` and `titleProvided` fields. Title validation ("Title is required" vs "Title cannot be empty") happens separately in `runCreate`.

V4 uses a method `(a *App) parseCreateArgs(args []string) (string, *createFlags, error)` that returns the title as a separate string rather than embedding it in the flags struct. It uses a single loop with a `switch` on all patterns:

```go
// V4: create.go lines 145-208
func (a *App) parseCreateArgs(args []string) (string, *createFlags, error) {
    flags := &createFlags{}
    var title string
    for i := 0; i < len(args); i++ {
        arg := args[i]
        switch {
        case arg == "--priority":
            // ...
        default:
            if title == "" {
                title = arg
            } else {
                return "", nil, fmt.Errorf("unexpected argument '%s'", arg)
            }
        }
    }
    if title == "" {
        return "", nil, fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
    }
    return title, flags, nil
}
```

V4's single-loop approach is cleaner -- it handles flags and positional args in one pass and rejects unexpected extra positional arguments. V2's two-loop approach silently ignores extra positional arguments after the title.

**Title Validation:**

V2 checks title presence (`titleProvided` bool) and emptiness separately in `runCreate`:

```go
// V2: create.go lines 99-108
if !flags.titleProvided {
    return fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
}
trimmedTitle := strings.TrimSpace(flags.title)
if trimmedTitle == "" {
    return fmt.Errorf("Title cannot be empty")
}
```

V4 handles missing title in `parseCreateArgs` (returns "Title is required" error), and empty/whitespace is handled in `runCreate` after trimming. However, based on the diff, V4's `parseCreateArgs` treats an empty string `""` as a valid title (since `""` != `""` is false when title is already empty, so it sets `title = ""`). The empty/whitespace check must happen in the task creation layer (via `task.NewTask`) or in `runCreate` prior to calling it. Looking at V4's `runCreate`, there is no explicit empty-title check before calling into the store -- this validation is delegated to `task.NewTask` which validates titles per tick-core-1-1.

V2 explicitly checks in `runCreate` with clear error messages ("Title cannot be empty"). V4 relies on the task package's validation, which is more DRY but potentially gives a different error message format.

**Mutation Logic:**

Both versions are nearly identical in their `Mutate` callback structure:
1. Build existence lookup map
2. Validate --blocked-by, --blocks, --parent IDs exist
3. Build `TaskOptions`, call `task.NewTask` with collision-check function
4. Append new task
5. Handle --blocks by iterating tasks and updating target tasks' blocked_by

Minor difference: V2 uses `task.NormalizeID()` when building the lookup map (`existingIDs[task.NormalizeID(t.ID)] = true`), while V4 uses raw IDs (`existingIDs[t.ID] = true`). V2 also normalizes when checking (`existingIDs[task.NormalizeID(modified[i].ID)]`). This means V2 is more defensive about case-insensitive ID matching in the mutation callback, while V4 assumes IDs are already normalized in the JSONL file.

V2 also calls `task.ValidatePriority()` inside the mutation callback, while V4 defers this to `task.NewTask()`.

**Output:**

Both versions have identical `printTaskDetails` methods with only cosmetic formatting differences:

V2:
```go
fmt.Fprintf(a.stdout, "ID:       %s\n", t.ID)
fmt.Fprintf(a.stdout, "Title:    %s\n", t.Title)
```

V4:
```go
fmt.Fprintf(a.Stdout, "ID:          %s\n", t.ID)
fmt.Fprintf(a.Stdout, "Title:       %s\n", t.Title)
```

V4 uses wider padding for alignment. V4 uses `t.Created.Format("2006-01-02T15:04:05Z")` directly, while V2 extracts a `formatTime` helper function.

**Error Quoting Style:**

V2 uses Go-style `%q` quoting in error messages:
```go
return nil, fmt.Errorf("task %q not found (referenced in --blocked-by)", id)
```

V4 uses single quotes:
```go
return nil, fmt.Errorf("task '%s' not found (referenced in --blocked-by)", id)
```

V4's style is more user-friendly for CLI output. V2's `%q` would produce escaped output with backquotes, which is more Go-idiomatic for debugging but less CLI-friendly.

**Helper Function Extraction:**

V4 extracts `parseCommaSeparatedIDs(s string) []string` as a standalone function used by both `--blocked-by` and `--blocks` parsing, avoiding code duplication. V2 duplicates the `strings.Split` + `TrimSpace` + `NormalizeID` logic inline for both flags.

### Code Quality

**Go Idioms:**

V2 uses a package-level function `parseCreateArgs` rather than a method, which is idiomatic for pure parsing logic. V4 makes it a method `(a *App) parseCreateArgs`, even though `a` is never used inside the function. V2's approach is more idiomatic here -- functions that don't need receiver state should be standalone.

V2's `createFlags` struct includes `titleProvided bool` for distinguishing "no title argument" from "empty title argument". This is explicit but adds a field only used once. V4 returns the title separately, which is cleaner.

**Error Handling:**

V2's `runCreate` returns `error` and the caller (`Run` in `app.go`) also returns `error`:
```go
// V2 app.go
case "create":
    return a.runCreate(cmdArgs)
```

V4's `runCreate` returns `error`, but the caller in `cli.go` handles it by writing to stderr and returning an exit code:
```go
// V4 cli.go
case "create":
    if err := a.runCreate(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

V4's pattern is better for a CLI application -- it properly converts errors to stderr output with exit codes at the dispatch layer, rather than propagating errors up further.

**DRY:**

V4 is DRYer with its extracted `parseCommaSeparatedIDs` helper. V2 duplicates the comma-split-trim-normalize logic in both `--blocked-by` and `--blocks` branches.

**Naming:**

V2: `store` imported as `storage`, struct field `a.config.Quiet`, `a.stdout`, `a.workDir`
V4: `store` imported as `store`, struct fields `a.Quiet`, `a.Stdout`, `a.Dir`

V4 uses exported fields on `App`, which is appropriate for a struct that tests create directly. V2 uses unexported fields, requiring the `NewApp()` constructor plus field assignment.

**Type Safety:**

Both use `*int` for optional priority. Both use `task.NormalizeID` for ID normalization. Equivalent.

### Test Quality

**V2 Test Functions (all inside single `TestCreateCommand`):**

1. `"it creates a task with only a title (defaults applied)"` -- checks title, status, priority via JSON parsing
2. `"it creates a task with all optional fields specified"` -- checks priority, description, blocked_by, parent; uses pre-populated JSONL content
3. `"it generates a unique ID for the created task"` -- checks prefix, length=11
4. `"it sets status to open on creation"` -- checks status field
5. `"it sets default priority to 2 when not specified"` -- checks priority field
6. `"it sets priority from --priority flag"` -- uses priority=0
7. `"it rejects priority outside 0-4 range"` -- tests priority=5 AND priority=-1 (two separate app instances)
8. `"it sets description from --description flag"` -- checks description field
9. `"it sets blocked_by from --blocked-by flag (single ID)"` -- checks single blocked_by entry
10. `"it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)"` -- checks 2 blocked_by entries
11. `"it updates target tasks' blocked_by when --blocks is used"` -- checks target task's blocked_by and updated timestamp
12. `"it sets parent from --parent flag"` -- checks parent field
13. `"it errors when title is missing"` -- checks error contains "Title is required"
14. `"it errors when title is empty string"` -- checks error contains "Title cannot be empty"
15. `"it errors when title is whitespace only"` -- checks error contains "Title cannot be empty"
16. `"it errors when --blocked-by references non-existent task"` -- checks error and no tasks written
17. `"it errors when --blocks references non-existent task"` -- checks error and no tasks written
18. `"it errors when --parent references non-existent task"` -- checks error and no tasks written
19. `"it persists the task to tasks.jsonl via atomic write"` -- reads back, checks title, timestamps present
20. `"it outputs full task details on success"` -- checks output contains ID, title, status
21. `"it outputs only task ID with --quiet flag"` -- checks prefix, length=11
22. `"it normalizes input IDs to lowercase"` -- uses "TICK-AAA111" in --blocked-by, verifies lowercase
23. `"it trims whitespace from title"` -- uses "  Trimmed title  ", verifies trimmed

Total: 23 subtests

**V4 Test Functions (each a separate top-level test function):**

1. `TestCreate_WithOnlyTitle` / `"it creates a task with only a title (defaults applied)"` -- checks title, status (using `task.StatusOpen` constant), priority, description, blocked_by, parent, closed (nil)
2. `TestCreate_WithAllOptionalFields` / `"it creates a task with all optional fields specified"` -- uses 3 pre-existing tasks (including --blocks target), checks priority, description, blocked_by, parent
3. `TestCreate_GeneratesUniqueID` / `"it generates a unique ID for the created task"` -- uses regex `^tick-[0-9a-f]{6}$`, creates TWO tasks and verifies different IDs
4. `TestCreate_SetsStatusOpen` / `"it sets status to open on creation"` -- uses `task.StatusOpen` constant
5. `TestCreate_DefaultPriority` / `"it sets default priority to 2 when not specified"` -- checks priority
6. `TestCreate_PriorityFlag` / `"it sets priority from --priority flag"` -- uses priority=0
7. `TestCreate_RejectsPriorityOutsideRange` / `"it rejects priority outside 0-4 range"` -- TABLE-DRIVEN: {"-1", "5", "100", "abc"} -- checks exit code=1 and stderr non-empty
8. `TestCreate_DescriptionFlag` / `"it sets description from --description flag"` -- checks description
9. `TestCreate_BlockedBySingleID` / `"it sets blocked_by from --blocked-by flag (single ID)"` -- uses typed `task.Task` setup
10. `TestCreate_BlockedByMultipleIDs` / `"it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)"` -- uses set-based verification (order-independent)
11. `TestCreate_BlocksUpdatesTargetTasks` / `"it updates target tasks' blocked_by when --blocks is used"` -- checks blocked_by and `targetTask.Updated.After(now)`
12. `TestCreate_ParentFlag` / `"it sets parent from --parent flag"` -- checks parent
13. `TestCreate_ErrorMissingTitle` / `"it errors when title is missing"` -- checks exit code=1, stderr "Error:", "Title is required"
14. `TestCreate_ErrorEmptyTitle` / `"it errors when title is empty string"` -- checks exit code=1, stderr "Error:"
15. `TestCreate_ErrorWhitespaceOnlyTitle` / `"it errors when title is whitespace only"` -- checks exit code=1, stderr "Error:"
16. `TestCreate_ErrorBlockedByNonExistent` / `"it errors when --blocked-by references non-existent task"` -- checks exit code=1, stderr, no tasks written
17. `TestCreate_ErrorBlocksNonExistent` / `"it errors when --blocks references non-existent task"` -- checks exit code=1, stderr, no tasks written
18. `TestCreate_ErrorParentNonExistent` / `"it errors when --parent references non-existent task"` -- checks exit code=1, stderr
19. `TestCreate_PersistsViaAtomicWrite` / `"it persists the task to tasks.jsonl via atomic write"` -- reads raw JSONL, verifies valid JSON, checks title. **Also checks `cache.db` exists.**
20. `TestCreate_OutputsTaskDetails` / `"it outputs full task details on success"` -- uses regex for ID, checks title and status in output
21. `TestCreate_QuietFlag` / `"it outputs only task ID with --quiet flag"` -- uses regex `^tick-[0-9a-f]{6}$`
22. `TestCreate_NormalizesInputIDs` / `"it normalizes input IDs to lowercase"` -- "TICK-AAA111" in --blocked-by, verifies lowercase
23. `TestCreate_TrimsWhitespaceFromTitle` / `"it trims whitespace from title"` -- verifies trimmed
24. `TestCreate_TimestampsSetToUTC` / `"it sets timestamps to current UTC ISO 8601"` -- captures before/after times, verifies range, equality of created/updated, UTC location

Total: 24 subtests (within 24 top-level functions)

**Test Structure Comparison:**

V2 uses a single `TestCreateCommand` function with all 23 subtests. V4 uses 24 separate top-level test functions, each containing one `t.Run` subtest. V4's structure is better for Go testing -- it allows running individual test functions with `go test -run TestCreate_QuietFlag` and makes test output clearer. V2's monolithic approach makes it harder to isolate failures.

**Edge Cases Unique to V4:**
- `TestCreate_RejectsPriorityOutsideRange` is table-driven with 4 cases including "100" and "abc" (non-integer). V2 only tests 5 and -1.
- `TestCreate_TimestampsSetToUTC` is a wholly new test not present in V2 -- verifies timestamps are within expected range, that created == updated, and timezone is UTC.
- `TestCreate_PersistsViaAtomicWrite` additionally checks `cache.db` exists (SQLite cache criterion).
- `TestCreate_GeneratesUniqueID` creates two tasks and verifies distinct IDs; V2 only creates one.
- `TestCreate_WithOnlyTitle` checks 7 fields (title, status, priority, description, blocked_by, parent, closed); V2 checks only 3 (title, status, priority).

**Edge Cases Unique to V2:**
- None. V2 covers the same scenarios as V4 minus the extras listed above.

**Assertion Quality:**

V2 reads tasks by parsing raw JSON into `map[string]interface{}`, requiring type assertions like `tk["priority"].(float64)`. This is fragile -- a missing field causes a nil-pointer panic rather than a clear test failure.

V4 reads tasks using `task.ReadJSONL` which returns typed `[]task.Task` structs. This is more robust -- field access is direct struct access (e.g., `tasks[0].Priority`), with compile-time type safety. V4 also uses `task.StatusOpen` constant instead of raw string "open", catching any future status name changes at compile time.

**Test Setup:**

V2 uses `setupTickDirWithContent(t, content string)` which takes raw JSONL strings. This is error-prone -- typos in JSON won't be caught until runtime.

V4 uses `setupInitializedDirWithTasks(t, tasks []task.Task)` which takes typed `[]task.Task` and calls `task.WriteJSONL` to serialize. This is type-safe and catches structural errors at compile time.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 3 (app.go, create.go, create_test.go) | 3 (cli.go, create.go, create_test.go) |
| Lines added | 884 | 1102 |
| Impl LOC (create.go) | 250 | 219 |
| Impl LOC (app.go/cli.go changes) | 11 net | 6 |
| Test LOC (create_test.go) | 623 | 877 |
| Test functions | 23 subtests in 1 function | 24 subtests in 24 functions |
| App integration changes | 18 lines changed (signature refactor) | 6 lines added (no signature change) |

## Verdict

**V4 is the better implementation.** The evidence:

1. **Test comprehensiveness:** V4 has 24 tests vs V2's 23, including the dedicated `TestCreate_TimestampsSetToUTC` test that directly validates acceptance criterion #13 and the explicit `cache.db` existence check for criterion #9. V4's priority rejection test is table-driven with 4 cases (including non-integer input) vs V2's 2 ad-hoc cases. V4's unique-ID test actually verifies uniqueness by creating two tasks.

2. **Test quality:** V4's tests use typed `task.Task` structs for both setup and assertions, providing compile-time safety. V2 parses raw JSON into `map[string]interface{}`, requiring fragile runtime type assertions. V4 uses `task.StatusOpen` constants rather than string literals.

3. **Test organization:** V4's 24 separate top-level test functions are superior to V2's single monolithic `TestCreateCommand` for Go test runner integration, parallel execution, and failure isolation.

4. **Code cleanliness:** V4 extracts `parseCommaSeparatedIDs` as a reusable helper (DRY), while V2 duplicates the comma-split logic. V4's single-loop arg parser rejects unexpected extra positional arguments; V2's two-loop parser silently ignores them.

5. **CLI integration:** V4's 6-line addition to `cli.go` is non-invasive, while V2 refactors the `parseGlobalFlags` return signature (changing 3 return values), touching more existing code.

6. **Implementation LOC:** V4 achieves equivalent (or better) functionality in 219 lines of implementation vs V2's 250 lines, while producing 877 lines of more thorough tests vs V2's 623 lines.

The only area where V2 is marginally more careful is normalizing IDs in the mutation callback's lookup map (`existingIDs[task.NormalizeID(t.ID)]`), which guards against mixed-case IDs in the JSONL file. V4 assumes IDs are already normalized. This is a minor defensive coding advantage for V2 but not significant enough to change the overall verdict.
