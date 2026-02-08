# Task tick-core-2-2: tick start, done, cancel, reopen Commands

## Task Summary

This task requires implementing four CLI subcommands -- `tick start <id>`, `tick done <id>`, `tick cancel <id>`, and `tick reopen <id>` -- that wire up the pure transition logic (from tick-core-2-1) to the CLI framework and storage engine. All four share the same flow: parse positional ID argument, normalize to lowercase, execute via storage engine `Mutate` (look up by ID, call `Transition`, persist), and output `{id}: {old_status} -> {new_status}` (with unicode arrow). `--quiet` suppresses success output. Errors go to stderr with exit code 1.

### Acceptance Criteria (from plan)

1. All four commands transition correctly and output transition line
2. Invalid transitions return error to stderr with exit code 1
3. Missing/not-found task ID returns error with exit code 1
4. `--quiet` suppresses success output
5. Input IDs normalized to lowercase
6. Timestamps managed correctly (closed set/cleared, updated refreshed)
7. Mutation persisted through storage engine

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| All four commands transition correctly and output transition line | PASS -- Tests all 7 transition paths (start, done from open, done from in_progress, cancel from open, cancel from in_progress, reopen from done, reopen from cancelled); output test verifies exact unicode arrow format | PASS -- Same 7 transition paths tested; output test verifies exact unicode arrow format |
| Invalid transitions return error to stderr with exit code 1 | PARTIAL -- Tests error is returned (non-nil) and contains "Cannot start", but V2's `Run()` returns `error`, so exit code is inferred (handled by main.go), and stderr is not directly tested for content | PASS -- Tests `code != 1` directly since `Run()` returns int, and verifies error output on stderr via `stderr.String()` |
| Missing/not-found task ID returns error with exit code 1 | PARTIAL -- Verifies error is returned with "Task ID is required" and usage hint, but exit code is inferred from non-nil error return | PASS -- Directly verifies `code == 1` and stderr contains "Error:" prefix plus "Task ID is required" plus usage hint |
| `--quiet` suppresses success output | PASS -- Verifies stdout is empty after `--quiet` flag | PASS -- Verifies stdout is empty after `--quiet` flag |
| Input IDs normalized to lowercase | PASS -- Passes uppercase "TICK-AAA111", verifies task transitions and output uses lowercase | PASS -- Same approach |
| Timestamps managed correctly (closed set/cleared, updated refreshed) | PASS -- Separate tests for closed on done, closed on cancel, cleared on reopen; but does NOT test updated timestamp refresh | PASS -- Tests closed on done and cancel (table-driven), cleared on reopen; also explicitly tests updated timestamp refresh in persistence test |
| Mutation persisted through storage engine | PASS -- Reads back file directly with `os.ReadFile` and parses JSON to verify persisted status | PASS -- Uses `readTasksFromDir` helper to verify persisted status |

## Implementation Comparison

### Approach

Both versions create a single `transition.go` file with a `runTransition(command string, args []string) error` method on `*App`, and register all four commands in a single switch case in the routing file.

**Routing (app.go / cli.go)**

V2 adds 2 lines to `app.go`:
```go
case "start", "done", "cancel", "reopen":
    return a.runTransition(subcmd, cmdArgs)
```

V4 adds 6 lines to `cli.go` due to its different App architecture where `Run()` returns `int` rather than `error`:
```go
case "start", "done", "cancel", "reopen":
    if err := a.runTransition(subcommand, subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

This difference is purely architectural -- V4's pattern is consistent with all its other command handlers in the same file. V2's is more concise because error handling is centralized in `main.go`.

**Transition handler (transition.go)**

Both implementations follow the same 5-step flow from the spec. Key structural differences:

1. **ID normalization**: V4 adds `strings.TrimSpace()` around the arg before normalizing:
   ```go
   // V4
   id := task.NormalizeID(strings.TrimSpace(args[0]))
   // V2
   id := task.NormalizeID(args[0])
   ```
   This is a minor defensive addition in V4. Unlikely to matter in practice since CLI args are already trimmed, but technically more robust.

2. **Task lookup within Mutate**: V2 compares normalized IDs on both sides; V4 compares the already-normalized input ID directly against the stored task ID:
   ```go
   // V2
   if task.NormalizeID(tasks[i].ID) == id {
   // V4
   if tasks[i].ID == id {
   ```
   V2's approach is more correct per spec: it normalizes both sides, so a task stored as "TICK-AAA111" would still be found. V4 assumes tasks are already stored with lowercase IDs, which may be true but is a stronger assumption.

3. **Transition function return type**: V2 calls `task.Transition()` which returns `(oldStatus, newStatus, error)` as three values. V4 calls `task.Transition()` which returns `(*TransitionResult, error)`:
   ```go
   // V2
   old, new, err := task.Transition(&tasks[idx], command)
   ...
   oldStatus = old
   newStatus = new
   // V4
   r, err := task.Transition(&tasks[idx], command)
   ...
   result = r
   ```
   V4's struct-based return is more idiomatic Go for returning multiple related values, but this difference originates from the prior task (tick-core-2-1), not this one.

4. **Error unwrapping**: V2 has special logic to strip the "mutation failed: " prefix that `store.Mutate` wraps around callback errors:
   ```go
   // V2
   if strings.HasPrefix(errMsg, "mutation failed: ") {
       return fmt.Errorf("%s", strings.TrimPrefix(errMsg, "mutation failed: "))
   }
   ```
   V4 has no such logic and lets the "mutation failed: " prefix propagate to the user. This means V4's error messages for "Task not found" or "Cannot start" will be prefixed with "mutation failed: ", which is a leaky abstraction and arguably a bug. V2 is genuinely better here.

5. **Package naming**: V2 imports `internal/storage`, V4 imports `internal/store`. Different naming conventions from the prior storage engine task, not a choice made in this task.

6. **App field access**: V2 uses `a.config.Quiet` and `a.stdout`; V4 uses `a.Quiet` and `a.Stdout`. This reflects different App struct designs: V2 uses a nested `Config` struct with unexported fields, V4 uses exported fields directly on `App`.

### Code Quality

**V2 transition.go (73 lines)**

```go
func (a *App) runTransition(command string, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
    }
    id := task.NormalizeID(args[0])
    ...
    var oldStatus, newStatus task.Status
    err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
        ...
        old, new, err := task.Transition(&tasks[idx], command)
        ...
        oldStatus = old
        newStatus = new
        return tasks, nil
    })
    if err != nil {
        errMsg := err.Error()
        if strings.HasPrefix(errMsg, "mutation failed: ") {
            return fmt.Errorf("%s", strings.TrimPrefix(errMsg, "mutation failed: "))
        }
        return err
    }
    ...
}
```

Uses named variables `old`, `new` for the transition result. Note: `new` shadows the Go builtin `new()` function, which is a minor code smell but harmless here.

The error-unwrapping logic is string-manipulation-based and fragile (if the storage layer changes its prefix, this breaks). But it does produce cleaner user-facing error messages.

**V4 transition.go (65 lines)**

```go
func (a *App) runTransition(command string, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
    }
    id := task.NormalizeID(strings.TrimSpace(args[0]))
    ...
    var result *task.TransitionResult
    err = s.Mutate(func(tasks []task.Task) ([]task.Task, error) {
        ...
        r, err := task.Transition(&tasks[idx], command)
        ...
        result = r
        return tasks, nil
    })
    if err != nil {
        return err
    }
    ...
    fmt.Fprintf(a.Stdout, "%s: %s -> %s\n", id, result.OldStatus, result.NewStatus)
}
```

Cleaner: 8 fewer lines, no string manipulation for error unwrapping, uses a struct result. However, the "mutation failed:" prefix leaks to users.

**Naming**: Both use clear, descriptive names. V4's `result *task.TransitionResult` is slightly more self-documenting than V2's `oldStatus, newStatus task.Status` pair.

**Error handling**: V2 is more thorough (unwraps storage prefix), V4 is simpler but lets an internal prefix leak.

### Test Quality

**V2 Test Functions** (all within a single `TestTransitionCommands` parent):

| # | Test name | What it tests |
|---|-----------|---------------|
| 1 | `it transitions task to in_progress via tick start` | start from open |
| 2 | `it transitions task to done via tick done from open` | done from open |
| 3 | `it transitions task to done via tick done from in_progress` | done from in_progress |
| 4 | `it transitions task to cancelled via tick cancel from open` | cancel from open |
| 5 | `it transitions task to cancelled via tick cancel from in_progress` | cancel from in_progress |
| 6 | `it transitions task to open via tick reopen from done` | reopen from done |
| 7 | `it transitions task to open via tick reopen from cancelled` | reopen from cancelled |
| 8 | `it outputs status transition line on success` | exact output format with unicode arrow |
| 9 | `it suppresses output with --quiet flag` | `--quiet` flag |
| 10 | `it errors when task ID argument is missing` | missing ID for all 4 commands (subtests) |
| 11 | `it errors when task ID is not found` | non-existent ID |
| 12 | `it errors on invalid transition` | done task + start = error |
| 13 | `it writes errors to stderr` | errors don't appear in stdout |
| 14 | `it exits with code 1 on error` | non-nil error returned |
| 15 | `it normalizes task ID to lowercase` | uppercase input normalized |
| 16 | `it persists status change via atomic write` | reads file back directly |
| 17 | `it sets closed timestamp on done` | done sets closed |
| 18 | `it sets closed timestamp on cancel` | cancel sets closed |
| 19 | `it clears closed timestamp on reopen` | reopen clears closed |

Total: 19 subtests (counting the 4 missing-ID subtests individually: 22)

**V4 Test Functions** (each is a top-level `Test*` function with one subtest):

| # | Function | Subtest name | What it tests |
|---|----------|-------------|---------------|
| 1 | `TestTransitionStart_TransitionsToInProgress` | it transitions task to in_progress via tick start | start from open |
| 2 | `TestTransitionDone_FromOpen` | it transitions task to done via tick done from open | done from open |
| 3 | `TestTransitionDone_FromInProgress` | it transitions task to done via tick done from in_progress | done from in_progress |
| 4 | `TestTransitionCancel_FromOpen` | it transitions task to cancelled via tick cancel from open | cancel from open |
| 5 | `TestTransitionCancel_FromInProgress` | it transitions task to cancelled via tick cancel from in_progress | cancel from in_progress |
| 6 | `TestTransitionReopen_FromDone` | it transitions task to open via tick reopen from done | reopen from done |
| 7 | `TestTransitionReopen_FromCancelled` | it transitions task to open via tick reopen from cancelled | reopen from cancelled |
| 8 | `TestTransition_OutputsStatusTransitionLine` | it outputs status transition line on success | exact output format |
| 9 | `TestTransition_QuietSuppressesOutput` | it suppresses output with --quiet flag | quiet flag |
| 10 | `TestTransition_ErrorMissingID` | it errors when task ID argument is missing | missing ID for all 4 commands (subtests) |
| 11 | `TestTransition_ErrorTaskNotFound` | it errors when task ID is not found | non-existent ID |
| 12 | `TestTransition_ErrorInvalidTransition` | it errors on invalid transition | done task + start |
| 13 | `TestTransition_ErrorsToStderr` | it writes errors to stderr | stderr has content, stdout empty |
| 14 | `TestTransition_ExitCode1OnError` | it exits with code 1 on error | exit code 1 for missing ID and not found |
| 15 | `TestTransition_NormalizesIDToLowercase` | it normalizes task ID to lowercase | uppercase input |
| 16 | `TestTransition_PersistsViaAtomicWrite` | it persists status change via atomic write | reads back via helper |
| 17 | `TestTransition_SetsClosedTimestamp` | it sets closed timestamp on done/cancel | table-driven for done and cancel |
| 18 | `TestTransition_ClearsClosedTimestampOnReopen` | it clears closed timestamp on reopen | reopen clears closed |

Total: 18 top-level test functions (counting table-driven subtests: 21)

**Test structure differences**:

- V2 uses a single parent `TestTransitionCommands` with all subtests nested. V4 uses individual top-level `Test*` functions each containing one `t.Run`. The V4 approach allows running individual test functions in isolation more easily via `go test -run`, but the extra `t.Run` inside each top-level function is redundant.
- V4 uses `bytes.Buffer` for stdout/stderr; V2 uses `strings.Builder` for stdout only and doesn't capture stderr (relying on errors being returned).
- V4's `TestTransition_ExitCode1OnError` tests two scenarios (missing ID and not found) in one function. V2's equivalent only checks one scenario.
- V4's `TestTransition_SetsClosedTimestamp` is table-driven combining done and cancel. V2 has separate tests for each.

**Test data setup differences**:

- V2 constructs JSONL strings manually via helper functions (`openTaskJSONL`, `inProgressTaskJSONL`, etc.) and writes raw content. This is brittle -- if the JSON schema changes, all helpers must be updated. The `readTasksJSONL` helper returns `[]map[string]interface{}`, requiring string-based field access.
- V4 constructs typed `task.Task` structs directly and uses `task.WriteJSONL` / `task.ReadJSONL` for serialization. This is more robust, type-safe, and aligned with Go idioms. The `readTasksFromDir` helper returns `[]task.Task`, allowing typed field access.

**Assertion quality**:

- V2 checks `tk["status"] != "in_progress"` (untyped string comparison on map values)
- V4 checks `tasks[0].Status != task.StatusInProgress` (typed constant comparison)
- V4 is strictly better: compile-time safety vs runtime string matching.

**Edge case: V4 tests exit code directly; V2 cannot**:

V4's `App.Run()` returns `int` (exit code), so tests like `TestTransition_ErrorsToStderr` verify `code != 1` and check stderr content directly. V2's `App.Run()` returns `error`, so the tests can only verify the error is non-nil and stdout is empty. V2 has comments explaining that exit code 1 is handled by `main.go`. This means V2 does not actually test stderr output or exit codes at the CLI level.

**Edge case: V4 tests updated timestamp refresh; V2 does not**:

V4's `TestTransition_PersistsViaAtomicWrite` includes:
```go
if !tasks[0].Updated.After(now) {
    t.Errorf("expected updated timestamp to be refreshed, got %v (original: %v)", tasks[0].Updated, now)
}
```
V2's equivalent test only checks the status, not the updated timestamp.

**Test gap in V2**: No test verifies the updated timestamp is refreshed.
**Test gap in V4**: The "mutation failed:" prefix leaks are not tested (arguably an implementation bug rather than a test gap).

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 5 | 5 |
| Lines added | 501 | 520 |
| Impl LOC (transition.go) | 73 | 65 |
| Routing LOC added (app.go/cli.go) | 2 | 6 |
| Test LOC (transition_test.go) | 423 | 446 |
| Top-level test functions | 1 (19 subtests) | 18 (21 subtests incl. table-driven) |

## Verdict

**V4 is the better implementation**, though V2 has one notable advantage.

V4 wins on:

1. **Type safety in tests**: V4 uses typed `task.Task` structs and `task.Status` constants throughout, while V2 uses raw JSONL strings and `map[string]interface{}` with untyped string comparisons. This is a significant quality difference that affects maintainability and compile-time error detection.

2. **Test completeness**: V4 verifies exit codes directly (its `Run()` returns `int`), tests stderr output content, and checks the `Updated` timestamp refresh. V2 can only infer exit codes from error returns and does not test stderr content or updated timestamps.

3. **Test infrastructure**: V4's `setupInitializedDirWithTasks` and `readTasksFromDir` use the actual `task.WriteJSONL` / `task.ReadJSONL` functions, ensuring test data goes through the same serialization path as production code. V2's manual JSONL string construction bypasses this.

4. **Implementation conciseness**: 65 lines vs 73 lines, achieved by removing the error-unwrapping logic.

V2 wins on:

1. **Error message quality**: V2 strips the "mutation failed: " prefix from storage errors, producing cleaner user-facing messages like `Task 'tick-nonexist' not found` instead of `mutation failed: Task 'tick-nonexist' not found`. This is genuinely better UX, though the string-prefix-stripping approach is fragile.

2. **ID lookup correctness**: V2 normalizes both sides of the ID comparison inside Mutate (`task.NormalizeID(tasks[i].ID) == id`), while V4 assumes stored IDs are already lowercase (`tasks[i].ID == id`). V2's approach is more defensive.

On balance, V4's systematic advantages in type safety, test completeness, and test infrastructure outweigh V2's advantage in error message cleanup and defensive ID lookup. The error prefix issue in V4 is a real bug but a minor one that could be fixed in the storage layer rather than worked around in every command handler.
