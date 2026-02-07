# Task tick-core-2-2: tick start, done, cancel, reopen commands

## Task Summary

This task adds four CLI subcommands (`start`, `done`, `cancel`, `reopen`) that transition a task's status. Each command shares the same flow: parse a positional ID argument, normalize it to lowercase, look up the task via the storage engine's `Mutate`, call `Transition()`, persist the result, and output `{id}: {old_status} → {new_status}`. The `--quiet` flag suppresses success output. Errors go to stderr with exit code 1.

**Acceptance Criteria:**
1. All four commands transition correctly and output transition line
2. Invalid transitions return error to stderr with exit code 1
3. Missing/not-found task ID returns error with exit code 1
4. `--quiet` suppresses success output
5. Input IDs normalized to lowercase
6. Timestamps managed correctly (closed set/cleared, updated refreshed)
7. Mutation persisted through storage engine

**Specified Tests (18 total):**
- Transitions: start (open->in_progress), done (from open), done (from in_progress), cancel (from open), cancel (from in_progress), reopen (from done), reopen (from cancelled)
- Output: transition line on success, --quiet suppression
- Errors: missing ID, not found, invalid transition, stderr output, exit code 1
- Normalization: lowercase ID
- Persistence: atomic write
- Timestamps: closed set on done/cancel, cleared on reopen

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| All four commands transition correctly and output transition line | PASS - all 7 transition tests present; output tested with `strings.Contains` | PASS - all 7 transition tests present; output tested with exact string match | PASS - all 7 transition tests present; output tested with exact string match |
| Invalid transitions return error to stderr with exit code 1 | PASS - tests exit code 1 and checks for "Cannot" in stderr | PASS - tests error return containing "Cannot start" | PASS - tests exit code 1 and checks for "cannot start" in stderr |
| Missing/not-found task ID returns error with exit code 1 | PASS - tests both missing and not-found; checks exit code 1 | PASS - tests both; checks error message content (not exit code directly - Run returns error) | PASS - tests both; checks exit code 1 directly |
| `--quiet` suppresses success output | PASS - verifies trimmed stdout is empty | PASS - verifies stdout.String() is empty | PASS - verifies stdout.String() is empty |
| Input IDs normalized to lowercase | PASS - passes uppercase ID, verifies exit code 0 | PASS - passes uppercase, verifies status changed and output uses lowercase | PASS - passes uppercase, verifies status changed and output uses lowercase |
| Timestamps managed correctly | PASS - separate tests for done/cancel closed set, reopen cleared; checks JSONL content | PASS - separate tests for done/cancel closed set, reopen cleared; uses typed assertion | PASS - combined done/cancel test, separate reopen test; uses struct field assertion |
| Mutation persisted through storage engine | PASS - reads tasks.jsonl directly, checks for "in_progress" | PASS - reads tasks.jsonl, parses JSON, checks status field | PASS - reads tasks via helper, then performs second transition to verify persistence chain |

## Implementation Comparison

### Approach

All three versions share the same high-level architecture: a single handler function registered for four command names via a `case "start", "done", "cancel", "reopen":` switch clause in the router. The key differences lie in method signature design, error handling patterns, and how tightly each version integrates with its pre-existing `App` structure.

**V1: `cmdTransition(workDir string, args []string, command string) error`**

V1's router passes `workDir` explicitly as a parameter alongside `cmdArgs` and the subcommand name. The function returns `error`, and the caller (`Run`) handles printing to stderr and returning exit code 1. This is the simplest implementation at 53 lines.

```go
func (a *App) cmdTransition(workDir string, args []string, command string) error {
    if len(args) == 0 {
        return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
    }
    taskID := task.NormalizeID(args[0])
    // ...
    var result task.TransitionResult
    err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
        for i := range tasks {
            if tasks[i].ID == taskID {
                r, err := task.Transition(&tasks[i], command)
                if err != nil { return nil, err }
                result = r
                return tasks, nil
            }
        }
        return nil, fmt.Errorf("task '%s' not found", taskID)
    })
```

The Mutate callback iterates directly with a `for i := range tasks` loop and returns immediately on match. The `TransitionResult` is captured via a closure variable. The task lookup compares `tasks[i].ID == taskID` — a direct equality check assuming IDs are already stored normalized.

**V2: `runTransition(command string, args []string) error`**

V2's router does not pass `workDir` — instead, `a.workDir` is read from the `App` struct. The function also returns `error`. V2 stores old/new status as separate `task.Status` variables instead of a `TransitionResult` struct, because V2's `Transition()` function has a different signature returning `(oldStatus, newStatus, err)` as three separate values.

```go
func (a *App) runTransition(command string, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
    }
    id := task.NormalizeID(args[0])
    // ...
    var oldStatus, newStatus task.Status
    err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
        idx := -1
        for i := range tasks {
            if task.NormalizeID(tasks[i].ID) == id {
                idx = i
                break
            }
        }
        if idx == -1 {
            return nil, fmt.Errorf("Task '%s' not found", id)
        }
        old, new, err := task.Transition(&tasks[idx], command)
        // ...
    })
```

Notable: V2 normalizes *both* the input ID and the stored ID during comparison (`task.NormalizeID(tasks[i].ID) == id`), providing double-normalization defense. V2 also includes explicit error message unwrapping to strip the `"mutation failed: "` prefix that `store.Mutate` wraps around errors:

```go
if strings.HasPrefix(errMsg, "mutation failed: ") {
    return fmt.Errorf("%s", strings.TrimPrefix(errMsg, "mutation failed: "))
}
```

This is the only version that handles this wrapping, which implies awareness of the storage engine's error wrapping behavior.

**V3: `runTransition(command string, args []string) int`**

V3 is unique: it returns `int` (exit code) directly rather than `error`. All error handling is done inline — the function writes to `a.Stderr` and returns `1` at each error point. This means errors are written to stderr *within* the transition handler itself, not delegated to the router.

```go
func (a *App) runTransition(command string, args []string) int {
    tickDir, err := DiscoverTickDir(a.Cwd)
    if err != nil {
        fmt.Fprintf(a.Stderr, "Error: %s\n", err)
        return 1
    }
    if len(args) < 3 {
        fmt.Fprintf(a.Stderr, "Error: Task ID is required. Usage: tick %s <id>\n", command)
        return 1
    }
    taskID := task.NormalizeID(args[2])
```

V3 also handles argument parsing differently: it receives the *full* args array (including `"tick"` and the subcommand), so it checks `len(args) < 3` and accesses `args[2]`. V1 and V2 receive only the arguments *after* the subcommand, so they check `len(args) == 0` and access `args[0]`.

V3 also discovers the tick directory *before* validating the ID argument, which means a "no .tick directory" error takes precedence over a "missing ID" error. V1 and V2 validate the ID first.

V3's Mutate callback uses a `*task.Task` pointer variable:
```go
var targetTask *task.Task
for i := range tasks {
    if tasks[i].ID == taskID {
        targetTask = &tasks[i]
        break
    }
}
if targetTask == nil {
    return nil, fmt.Errorf("task '%s' not found", taskID)
}
```

**V3 also adds help text** for the four commands to `printUsage()`:
```go
fmt.Fprintln(a.Stdout, "  start   Mark task as in-progress")
fmt.Fprintln(a.Stdout, "  done    Mark task as completed")
fmt.Fprintln(a.Stdout, "  cancel  Mark task as cancelled")
fmt.Fprintln(a.Stdout, "  reopen  Reopen a closed task")
```
This is the only version that updates the help text, which is a nice completeness touch even though not explicitly required by the task.

### Code Quality

**Naming:**
- V1 uses `cmdTransition` (matching `cmdInit`, `cmdCreate`, etc. in its codebase). Consistent with V1's naming pattern.
- V2 uses `runTransition` (matching `runCreate`, `runList`, etc.). Consistent with V2's naming pattern.
- V3 uses `runTransition` (matching `runInit`, `runCreate`, etc.). Consistent with V3's naming pattern.
- V1 names the ID variable `taskID`; V2 uses `id`; V3 uses `taskID`. Both `taskID` and `id` are reasonable.

**Error handling:**
- V1: Returns errors to the router, which prints them. Clean separation of concerns.
- V2: Returns errors to the router, but also strips `"mutation failed: "` prefix. This shows awareness of a leaky abstraction in the storage layer and actively corrects it. This is the most defensive approach.
- V3: Handles all errors inline by writing to stderr and returning exit codes. This duplicates the error-to-stderr logic that could be centralized in the router. Less DRY but more explicit.

**Error message wrapping (V2 only):**
```go
errMsg := err.Error()
if strings.HasPrefix(errMsg, "mutation failed: ") {
    return fmt.Errorf("%s", strings.TrimPrefix(errMsg, "mutation failed: "))
}
```
V1 and V3 pass through whatever error the store returns. If the store wraps errors with `"mutation failed: "`, V1 and V3 would expose that prefix to the user.

**Store error wrapping on open (V1 only):**
```go
store, err := storage.NewStore(tickDir)
if err != nil {
    return fmt.Errorf("opening store: %w", err)
}
```
V1 wraps the store-open error with context. V2 and V3 return it unwrapped.

**ID comparison in Mutate callback:**
- V1: `tasks[i].ID == taskID` — assumes stored IDs are already lowercase
- V2: `task.NormalizeID(tasks[i].ID) == id` — normalizes stored IDs too (defensive)
- V3: `tasks[i].ID == taskID` — assumes stored IDs are already lowercase

V2's approach is more robust against mixed-case IDs in storage.

**Argument access:**
- V1 receives pre-parsed `cmdArgs` (subcommand already stripped): `args[0]`
- V2 receives pre-parsed `cmdArgs` (subcommand already stripped): `args[0]`
- V3 receives full args array: `args[2]` — this couples the function to the exact position of arguments in the full command line

**Type safety for TransitionResult:**
- V1 and V3 use `task.TransitionResult` struct
- V2 uses raw `task.Status` variables (`oldStatus`, `newStatus`), because V2's Transition function returns tuple-style `(Status, Status, error)` rather than a struct

### Test Quality

**V1 Test Functions (16 tests, 244 lines):**

| # | Test Name | Approach |
|---|-----------|----------|
| 1 | `transitions task to in_progress via tick start` | Integration via `runCmd`; checks output contains "open" and "in_progress" |
| 2 | `transitions task to done via tick done from open` | Integration; checks output contains "done" |
| 3 | `transitions task to done via tick done from in_progress` | Integration; runs start first, then done |
| 4 | `transitions task to cancelled via tick cancel from open` | Integration; checks "cancelled" |
| 5 | `transitions task to cancelled via tick cancel from in_progress` | Integration; runs start first |
| 6 | `transitions task to open via tick reopen from done` | Integration; runs done first |
| 7 | `transitions task to open via tick reopen from cancelled` | Integration; runs cancel first |
| 8 | `outputs status transition line on success` | Checks for ID and "→" in output |
| 9 | `suppresses output with --quiet flag` | Checks trimmed stdout is empty |
| 10 | `errors when task ID argument is missing` | Tests all 4 commands in a loop |
| 11 | `errors when task ID is not found` | Checks "not found" in stderr |
| 12 | `errors on invalid transition` | Does done, then start; checks "Cannot" |
| 13 | `normalizes task ID to lowercase` | Passes uppercase, checks exit 0 |
| 14 | `persists status change via atomic write` | Reads tasks.jsonl directly |
| 15 | `sets closed timestamp on done` | Reads JSONL, checks for "closed" string |
| 16 | `sets closed timestamp on cancel` | Reads JSONL, checks for "closed" string |
| 17 | `clears closed timestamp on reopen` | Does done->reopen, checks no "closed" |

V1 uses **integration-style tests** via `runCmd` helper (creates real App, runs commands, captures stdout/stderr/exit code). Tests create tasks via `createTask` helper which runs the actual `tick create` command. Output assertions use `strings.Contains` — loose matching. V1 has separate tests for "sets closed on done" and "sets closed on cancel" (matching the spec's "it sets closed timestamp on done/cancel" as two tests).

**Missing from spec:** "it writes errors to stderr" and "it exits with code 1 on error" are not explicit test names, though the behavior is implicitly tested via `runCmd` returning `code` and `stderr`.

**V2 Test Functions (18 tests, 423 lines):**

| # | Test Name | Approach |
|---|-----------|----------|
| 1 | `it transitions task to in_progress via tick start` | Unit; manually sets up JSONL, calls Run, reads back |
| 2 | `it transitions task to done via tick done from open` | Unit; same pattern |
| 3 | `it transitions task to done via tick done from in_progress` | Unit; uses `inProgressTaskJSONL` |
| 4 | `it transitions task to cancelled via tick cancel from open` | Unit |
| 5 | `it transitions task to cancelled via tick cancel from in_progress` | Unit |
| 6 | `it transitions task to open via tick reopen from done` | Unit; uses `doneTaskJSONL` |
| 7 | `it transitions task to open via tick reopen from cancelled` | Unit; uses `cancelledTaskJSONL` |
| 8 | `it outputs status transition line on success` | Exact string match: `"tick-aaa111: open → in_progress"` |
| 9 | `it suppresses output with --quiet flag` | Checks `stdout.String() == ""` |
| 10 | `it errors when task ID argument is missing` | Sub-tests for each of 4 commands; checks error message AND usage hint |
| 11 | `it errors when task ID is not found` | Checks error contains the ID |
| 12 | `it errors on invalid transition` | Uses `doneTaskJSONL`, tries start; checks "Cannot start" |
| 13 | `it writes errors to stderr` | Verifies error is returned (not nil) and stdout is empty |
| 14 | `it exits with code 1 on error` | Verifies error is non-nil (indirect — Run returns error, not int) |
| 15 | `it normalizes task ID to lowercase` | Passes "TICK-AAA111", verifies status and output prefix |
| 16 | `it persists status change via atomic write` | Reads and parses JSONL, checks 1 line, verifies status |
| 17 | `it sets closed timestamp on done` | Uses typed assertion `tk["closed"].(string)` |
| 18 | `it sets closed timestamp on cancel` | Same typed assertion |
| 19 | `it clears closed timestamp on reopen` | Checks `hasClosed` is false via map key existence |

V2 uses **unit-style tests** — directly creates `App` struct, injects a `strings.Builder` for stdout, and calls `app.Run()`. Test data is set up via JSONL helper functions (`openTaskJSONL`, `inProgressTaskJSONL`, etc.) that return raw JSON strings written to disk. This gives deterministic test data (fixed IDs like "tick-aaa111") without running `tick create`.

V2 has the **most thorough assertions**: the missing-ID test uses sub-tests per command AND checks both the "Task ID is required" message AND the "tick {cmd} <id>" usage hint. The output test does an exact string comparison. The "writes errors to stderr" and "exits with code 1" tests are explicit (even if indirect due to Run returning error).

**V3 Test Functions (18 tests, 499 lines):**

| # | Test Name | Approach |
|---|-----------|----------|
| 1 | `it transitions task to in_progress via tick start` | Unit; uses `setupTask` helper, creates App struct |
| 2 | `it transitions task to done via tick done from open` | Unit |
| 3 | `it transitions task to done via tick done from in_progress` | Unit; uses `setupTaskFull` with status "in_progress" |
| 4 | `it transitions task to cancelled via tick cancel from open` | Unit |
| 5 | `it transitions task to cancelled via tick cancel from in_progress` | Unit; `setupTaskFull` |
| 6 | `it transitions task to open via tick reopen from done` | Unit; `setupTaskFull` with "done" |
| 7 | `it transitions task to open via tick reopen from cancelled` | Unit; `setupTaskFull` with "cancelled" |
| 8 | `it outputs status transition line on success` | Exact string match with newline |
| 9 | `it suppresses output with --quiet flag` | Checks empty stdout |
| 10 | `it errors when task ID argument is missing` | Checks exit code 1, stderr contains "Task ID is required" AND "Usage: tick start <id>" |
| 11 | `it errors when task ID is not found` | Checks exit code 1, stderr contains ID and "not found" |
| 12 | `it errors on invalid transition` | Checks exit code 1, stderr contains "cannot start" |
| 13 | `it writes errors to stderr` | Checks stderr is non-empty AND stdout is empty |
| 14 | `it exits with code 1 on error` | Tests 3 error cases: missing ID, not-found, invalid transition |
| 15 | `it normalizes task ID to lowercase` | Passes "TICK-A1B2C3", verifies status and output contains lowercase |
| 16 | `it persists status change via atomic write` | Creates second App instance, performs second transition to verify chain |
| 17 | `it sets closed timestamp on done/cancel` | Combined test with two tasks, verifies both |
| 18 | `it clears closed timestamp on reopen` | Verifies initial closed is set, then reopen clears it |

V3 uses **unit-style tests** like V2 but with `bytes.Buffer` instead of `strings.Builder`. V3's tests directly verify exit codes (since `Run` returns `int`), making them the most faithful to the acceptance criteria about exit codes.

**Test Gap Analysis:**

| Test | V1 | V2 | V3 |
|------|-----|-----|-----|
| 7 transition tests | YES | YES | YES |
| Output format exact match | NO (loose `Contains`) | YES | YES |
| --quiet suppression | YES | YES | YES |
| Missing ID (all 4 commands) | YES (loop) | YES (sub-tests) | YES (single "start" only) |
| Not-found ID | YES | YES | YES |
| Invalid transition | YES | YES | YES |
| Errors to stderr (explicit) | NO (implicit) | YES (indirect) | YES (direct) |
| Exit code 1 (explicit) | NO (implicit) | YES (indirect) | YES (direct, 3 cases) |
| Lowercase normalization | YES | YES (+ output check) | YES (+ output check) |
| Persistence | YES (raw JSONL) | YES (parsed JSONL) | YES (second transition chain) |
| Closed on done | YES | YES | YES (combined) |
| Closed on cancel | YES | YES | YES (combined) |
| Closed cleared on reopen | YES | YES | YES (+ pre-condition check) |
| Usage hint in error message | NO | YES | YES |

**V1 test gaps:** No explicit "writes errors to stderr" or "exits with code 1" tests. Output assertions use loose matching. Missing-ID test does not verify the usage hint format.

**V2 test gaps:** The "exits with code 1" test is indirect (checks error is non-nil, since `Run()` returns `error` not `int`). Missing-ID test only tests at `Run()` error level, not actual exit code.

**V3 test gaps:** Missing-ID test only tests `start`, not all 4 commands. However, "exits with code 1" is the most thorough (3 error paths tested).

**V3 unique strengths:** The "persists via atomic write" test creates a *second* App instance and performs a second transition to verify the full persistence round-trip. The "clears closed on reopen" test verifies the pre-condition (closed IS set) before testing that reopen clears it. The "exits with code 1" test covers 3 distinct error paths.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 3 | 5 (incl 2 docs) | 6 (incl 2 docs) |
| Lines added | 299 | 501 (498 code) | 596 (579 code) |
| Impl LOC | 53 | 73 | 74 |
| Test LOC | 244 | 423 | 499 |
| Test functions | 17 (in 1 top-level) | 19 (in 1 top-level, with sub-tests) | 18 (in 1 top-level) |

## Verdict

**V1 is the best implementation** for this task, with a notable caveat about test precision.

**Implementation (V1 wins):** V1 is the most concise at 53 lines — 27% smaller than V2 and V3. It achieves the same functionality with less code because it delegates error presentation to the router (matching Go's idiomatic error-returns-up pattern) and receives pre-parsed arguments. The code is clean, linear, and has no special-case logic. V3's inline error handling duplicates logic that belongs in the router, and its raw-args approach (`args[2]`) couples it to command-line structure. V2's error-prefix stripping is defensive but suggests a storage layer design issue being patched at the wrong level.

**However, V2 and V3 have stronger tests.** V2's exact output matching (`"tick-aaa111: open → in_progress"`) and usage-hint verification in the missing-ID test are more precise than V1's loose `strings.Contains` checks. V3's explicit exit-code testing (for 3 error paths) and persistence round-trip test (second transition through a fresh App) are the most thorough integration proofs.

**If test quality is weighted equally with implementation quality, V3 is the best overall.** V3's tests are the most exhaustive: they verify exit codes directly (not indirectly through error returns), test 3 distinct error paths for exit code 1, perform a real round-trip persistence verification, and validate pre-conditions before asserting post-conditions. V3 also uniquely updates the help text for completeness. The implementation is slightly more verbose than V1 but functionally equivalent.

**Recommended synthesis:** Take V1's implementation approach (error-return pattern, pre-parsed args, minimal LOC) combined with V3's test thoroughness (exact output matching, multi-path exit code testing, round-trip persistence verification, pre-condition validation) and V2's double-normalization defense in the ID lookup and usage-hint verification in the missing-ID test.
