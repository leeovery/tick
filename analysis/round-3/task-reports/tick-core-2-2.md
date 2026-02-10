# Task tick-core-2-2: tick start, done, cancel, reopen commands

## Task Summary

This task implements the CLI surface for the four status transition commands: `tick start`, `tick done`, `tick cancel`, and `tick reopen`. Each command shares a common flow: parse a positional ID argument, normalize to lowercase, look up the task via the storage engine's `Mutate` flow, apply the transition (from tick-core-2-1's pure domain logic), persist the result, and output `{id}: {old_status} -> {new_status}` with a unicode arrow. Missing ID should produce an error with usage hint. `--quiet` suppresses success output. Errors go to stderr with exit code 1.

### Acceptance Criteria (from plan)

1. All four commands transition correctly and output transition line
2. Invalid transitions return error to stderr with exit code 1
3. Missing/not-found task ID returns error with exit code 1
4. `--quiet` suppresses success output
5. Input IDs normalized to lowercase
6. Timestamps managed correctly (closed set/cleared, updated refreshed)
7. Mutation persisted through storage engine

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| All four commands transition correctly and output transition line | PASS -- all four commands registered in `cli.go` switch case (line 95); output via `a.Formatter.FormatTransition(a.Stdout, id, string(result.OldStatus), string(result.NewStatus))` at line 62 of `transition.go`; tests verify each transition path | PASS -- all four commands registered via `commands` map entries `runTransition("start")` etc. at lines 103-106 of `cli.go`; output via `ctx.Fmt.FormatTransition(ctx.Stdout, &TransitionData{...})` at line 53 of `transition.go`; tests verify each transition path |
| Invalid transitions return error to stderr with exit code 1 | PASS -- `task.Transition` returns error on invalid transition; `runTransition` returns it to `Run()` which calls `a.writeError(err)` and returns 1; tested in `TestTransition_ErrorInvalidTransition` | PASS -- same flow via `task.Transition` error; `handler(ctx)` returns error to `Run()` which calls `fmt.Fprintf(stderr, ...)` and returns 1; tested in subtest `"it errors on invalid transition"` |
| Missing/not-found task ID returns error with exit code 1 | PASS -- missing ID: `len(args) == 0` returns `fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)` at line 14; not found: `fmt.Errorf("Task '%s' not found", id)` at line 42; both tested | PASS -- missing ID: `len(ctx.Args) == 0` returns same error message at line 17; not found: `fmt.Errorf("Task '%s' not found", id)` at line 46; both tested |
| `--quiet` suppresses success output | PASS -- `if a.Quiet { return nil }` at lines 59-61 skips formatter call; tested in `TestTransition_QuietSuppressesOutput` | PASS -- `if !ctx.Quiet { ... }` at line 52 guards formatter call; tested in subtest `"it suppresses output with --quiet flag"` |
| Input IDs normalized to lowercase | PASS -- `id := task.NormalizeID(strings.TrimSpace(args[0]))` at line 17; tested in `TestTransition_NormalizesIDToLowercase` with `"TICK-AAA111"` input | PASS -- `id := task.NormalizeID(ctx.Args[0])` at line 20; tested in subtest `"it normalizes task ID to lowercase"` with `"TICK-AAAAAA"` input |
| Timestamps managed correctly (closed set/cleared, updated refreshed) | PASS -- delegated to `task.Transition()` which sets `Closed` on done/cancel and clears on reopen (from tick-core-2-1); tested in `TestTransition_SetsClosedTimestamp` and `TestTransition_ClearsClosedTimestampOnReopen`; also verifies `Updated` was refreshed in `TestTransition_PersistsViaAtomicWrite` | PASS -- same delegation to `task.Transition()`; tested in subtests `"it sets closed timestamp on done/cancel"` and `"it clears closed timestamp on reopen"` |
| Mutation persisted through storage engine | PASS -- mutation performed inside `s.Mutate()` callback at line 32; verified by re-reading from file in `TestTransition_PersistsViaAtomicWrite` | PASS -- mutation performed inside `store.Mutate()` callback at line 35; verified by re-reading from file in subtest `"it persists status change via atomic write"` |

## Implementation Comparison

### Approach

**V4: Method on `App` struct with switch-case dispatch**

V4 implements `runTransition` as a method on the `*App` struct (line 12):

```go
func (a *App) runTransition(command string, args []string) error {
```

The `cli.go` dispatches to it via a `switch` statement with combined cases (line 95):

```go
case "start", "done", "cancel", "reopen":
    if err := a.runTransition(subcommand, subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

The implementation flow is:
1. Check `len(args) == 0` for missing ID
2. Normalize with `task.NormalizeID(strings.TrimSpace(args[0]))` -- notably adds `strings.TrimSpace` before normalization
3. Discover tick dir, open store via `a.openStore(tickDir)` (which wires up verbose logging)
4. Inside `s.Mutate()`: linear scan for task by ID using an `idx` variable, then call `task.Transition(&tasks[idx], command)`
5. Output via `a.Formatter.FormatTransition(a.Stdout, id, string(result.OldStatus), string(result.NewStatus))`

The result is stored as `var result *task.TransitionResult` (pointer), matching V4's `task.Transition` which returns `(*TransitionResult, error)`.

**V5: Closure-returning function with command map dispatch**

V5 uses a higher-order function pattern -- `runTransition` returns a handler closure (line 14):

```go
func runTransition(command string) func(*Context) error {
    return func(ctx *Context) error {
```

The `cli.go` registers these in the `commands` map (lines 103-106):

```go
"start":  runTransition("start"),
"done":   runTransition("done"),
"cancel": runTransition("cancel"),
"reopen": runTransition("reopen"),
```

The implementation flow is:
1. Check `len(ctx.Args) == 0` for missing ID
2. Normalize with `task.NormalizeID(ctx.Args[0])` -- no `strings.TrimSpace` wrapping
3. Discover tick dir, open store via `engine.NewStore(tickDir, ctx.storeOpts()...)` (passes options directly)
4. Inside `store.Mutate()`: inline loop with early return on match, then `task.Transition(&tasks[i], command)`
5. Output via `ctx.Fmt.FormatTransition(ctx.Stdout, &TransitionData{...})`

The result is stored as `var result task.TransitionResult` (value type), matching V5's `task.Transition` which returns `(TransitionResult, error)`.

**Key structural differences:**

1. **Dispatch mechanism:** V4 uses a `switch` statement; V5 uses a map of handler functions. The map approach is more extensible -- adding a new command is a single map entry vs. a new `case` block with boilerplate. V5's approach eliminates the repeated `if err := ...; err != nil { a.writeError(err); return 1 }` pattern that V4 duplicates for every command.

2. **Context passing:** V4 passes `(command string, args []string)` and accesses `a.Stdout`, `a.Stderr`, `a.Quiet`, `a.Dir` from the receiver. V5 passes `*Context` which bundles all request state. This is a broader architectural difference inherited from the CLI framework, not specific to this task.

3. **Mutate callback structure:** V4 uses an index-scanning approach with a separate `idx` variable:
```go
idx := -1
for i := range tasks {
    if tasks[i].ID == id {
        idx = i
        break
    }
}
if idx == -1 {
    return nil, fmt.Errorf("Task '%s' not found", id)
}
r, err := task.Transition(&tasks[idx], command)
```

V5 uses an inline early-return pattern:
```go
for i := range tasks {
    if tasks[i].ID == id {
        r, err := task.Transition(&tasks[i], command)
        if err != nil {
            return nil, err
        }
        result = r
        return tasks, nil
    }
}
return nil, fmt.Errorf("Task '%s' not found", id)
```

V5's approach is more compact (no `idx` variable, no separate lookup-then-act phases) and is idiomatic Go -- the "find and act" is a single pass. V4's two-phase approach (find index, then act on index) is slightly more verbose but equally correct.

4. **FormatTransition interface:** V4's `FormatTransition` takes individual string parameters: `(w io.Writer, id string, oldStatus string, newStatus string)`. V5 uses a struct parameter: `(w io.Writer, data *TransitionData)`. V5's approach is more extensible -- adding fields to the transition output only requires updating the struct, not changing the method signature across all formatter implementations.

5. **`strings.TrimSpace` on ID input:** V4 wraps the ID argument with `strings.TrimSpace(args[0])` before normalization. V5 does not. Since `NormalizeID` only calls `strings.ToLower`, V4 handles edge cases like trailing whitespace on the ID argument. This is a minor V4 advantage for robustness, though in practice CLI arguments are already trimmed by the shell.

### Code Quality

**Imports:**

V4 imports `"strings"` for `strings.TrimSpace` and `"github.com/leeovery/tick/internal/store"`. V5 imports only `"fmt"` and uses `"github.com/leeovery/tick/internal/engine"` (different package name for the same concept). V5's leaner imports reflect the absence of `strings.TrimSpace`.

**Store creation:**

V4 uses a centralized helper on `*App` (from `cli.go` line 173):
```go
s, err := a.openStore(tickDir)
```
This `openStore` method creates the store and wires up verbose logging, providing consistent store setup across all commands.

V5 passes options directly:
```go
store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
```
Where `ctx.storeOpts()` returns `[]engine.Option{engine.WithVerbose(ctx.newVerboseLogger())}`. This uses the functional options pattern (a Go idiom) rather than post-construction mutation.

Both approaches are valid. V5's functional options pattern is more Go-idiomatic and recommended by the skill's reference to "functional options" as a configuration mechanism. V4's centralized helper is simpler but slightly less extensible.

**Error messages:**

Both versions produce identical error messages:
- Missing ID: `"Task ID is required. Usage: tick %s <id>"`
- Not found: `"Task '%s' not found"`

These match the spec's requirement: `"Error: Task ID is required. Usage: tick {command} <id>"`.

**Output format:**

Both produce the spec-required `{id}: {old_status} -> {new_status}` with unicode arrow, but delegate to their respective formatter implementations. V4 uses `fmt.Errorf` with `\u2192` in the formatter. V5 does the same via `TransitionData`.

**Quiet handling:**

V4:
```go
if a.Quiet {
    return nil
}
return a.Formatter.FormatTransition(...)
```

V5:
```go
if !ctx.Quiet {
    return ctx.Fmt.FormatTransition(ctx.Stdout, &TransitionData{...})
}
return nil
```

V4 uses early return for quiet; V5 uses a guard clause for non-quiet. Both are functionally equivalent. V5's approach is slightly more conventional -- the "normal" path (output) is in the if-body, and the "special" path (quiet/no-op) falls through.

**Return type for TransitionResult:**

V4 stores `var result *task.TransitionResult` (pointer) because its `task.Transition` returns `(*TransitionResult, error)`. V5 stores `var result task.TransitionResult` (value type) because its `task.Transition` returns `(TransitionResult, error)`. V5's value-type return is more idiomatic for small structs in Go -- it avoids heap allocation and nil-pointer concerns. The `TransitionResult` struct contains only two `Status` (string) fields, making value semantics appropriate.

**Documentation:**

V4's doc comment (line 10-11):
```go
// runTransition implements the shared handler for tick start, done, cancel, and reopen commands.
// It parses the ID, loads the task, applies the transition, persists via storage, and outputs the result.
```

V5's doc comment (lines 10-13):
```go
// runTransition returns a handler for a transition command (start, done, cancel,
// reopen). Each handler parses the positional task ID, looks up the task via the
// storage engine's Mutate flow, applies the transition, persists, and outputs
// the transition result.
```

Both are thorough and describe the function's purpose. V5's accurately describes its closure-returning nature.

### Test Quality

#### V4 Test Functions (14 top-level functions, 18 total subtests)

1. **`TestTransitionStart_TransitionsToInProgress`** (1 subtest)
   - `"it transitions task to in_progress via tick start"` -- creates open task, runs `tick start`, verifies status is `in_progress`

2. **`TestTransitionDone_FromOpen`** (1 subtest)
   - `"it transitions task to done via tick done from open"` -- creates open task, runs `tick done`, verifies status is `done`

3. **`TestTransitionDone_FromInProgress`** (1 subtest)
   - `"it transitions task to done via tick done from in_progress"` -- creates in_progress task, runs `tick done`, verifies status is `done`

4. **`TestTransitionCancel_FromOpen`** (1 subtest)
   - `"it transitions task to cancelled via tick cancel from open"` -- creates open task, runs `tick cancel`, verifies status is `cancelled`

5. **`TestTransitionCancel_FromInProgress`** (1 subtest)
   - `"it transitions task to cancelled via tick cancel from in_progress"` -- creates in_progress task, runs `tick cancel`, verifies status is `cancelled`

6. **`TestTransitionReopen_FromDone`** (1 subtest)
   - `"it transitions task to open via tick reopen from done"` -- creates done task with closed timestamp, runs `tick reopen`, verifies status is `open`

7. **`TestTransitionReopen_FromCancelled`** (1 subtest)
   - `"it transitions task to open via tick reopen from cancelled"` -- creates cancelled task with closed timestamp, runs `tick reopen`, verifies status is `open`

8. **`TestTransition_OutputsStatusTransitionLine`** (1 subtest)
   - `"it outputs status transition line on success"` -- runs `tick start`, verifies output is `"tick-aaa111: open -> in_progress"` (with unicode arrow) via `strings.TrimSpace` comparison

9. **`TestTransition_QuietSuppressesOutput`** (1 subtest)
   - `"it suppresses output with --quiet flag"` -- runs `tick --quiet start`, verifies stdout is empty

10. **`TestTransition_ErrorMissingID`** (1 parent + 4 subtests)
    - `"it errors when task ID argument is missing"` -- iterates over all 4 commands, verifies exit code 1, checks for "Error:", "Task ID is required", and command-specific usage hint `"Usage: tick {cmd} <id>"`

11. **`TestTransition_ErrorTaskNotFound`** (1 subtest)
    - `"it errors when task ID is not found"` -- runs `tick start tick-nonexist`, verifies exit code 1, checks stderr contains "Error:" and "tick-nonexist"

12. **`TestTransition_ErrorInvalidTransition`** (1 subtest)
    - `"it errors on invalid transition"` -- creates done task, runs `tick start`, verifies exit code 1 and error on stderr

13. **`TestTransition_ErrorsToStderr`** (1 subtest)
    - `"it writes errors to stderr"` -- runs `tick start tick-nonexist`, verifies stderr non-empty and stdout empty

14. **`TestTransition_ExitCode1OnError`** (1 subtest, 2 scenarios)
    - `"it exits with code 1 on error"` -- tests both missing ID and not-found scenarios in same subtest

15. **`TestTransition_NormalizesIDToLowercase`** (1 subtest)
    - `"it normalizes task ID to lowercase"` -- runs `tick start TICK-AAA111`, verifies task transitioned and output uses lowercase ID

16. **`TestTransition_PersistsViaAtomicWrite`** (1 subtest)
    - `"it persists status change via atomic write"` -- runs `tick start`, re-reads tasks from file, verifies status persisted and `Updated` timestamp is refreshed (after original `now`)

17. **`TestTransition_SetsClosedTimestamp`** (1 parent + 2 subtests via table)
    - `"it sets closed timestamp on done/cancel"` -- table-driven: runs `done` and `cancel`, verifies `Closed != nil`

18. **`TestTransition_ClearsClosedTimestampOnReopen`** (1 subtest)
    - `"it clears closed timestamp on reopen"` -- creates done task with closed timestamp, runs `tick reopen`, verifies `Closed == nil`

**V4 test structure notes:**
- Each test scenario is a separate top-level `Test*` function with a single nested `t.Run`
- Task setup uses manual struct construction: `task.Task{ID: "tick-aaa111", Title: "My task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}`
- Fixed timestamp: `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)` used consistently
- Helper functions: `setupInitializedDir(t)`, `setupInitializedDirWithTasks(t, tasks)`, `readTasksFromDir(t, dir)`
- CLI invocation: `app.Run([]string{"tick", ...})` via `App` struct construction
- Total: 14 top-level functions, 18 distinct test scenarios (including table-driven subtests)

#### V5 Test Functions (1 top-level function, 18 subtests)

All tests are nested under a single `TestTransition(t *testing.T)` function:

1. `"it transitions task to in_progress via tick start"` -- creates open task via `task.NewTask("tick-aaaaaa", "Start me")`, runs `tick start`, verifies status
2. `"it transitions task to done via tick done from open"` -- open task, runs `tick done`, verifies status
3. `"it transitions task to done via tick done from in_progress"` -- sets `tk.Status = task.StatusInProgress`, runs `tick done`
4. `"it transitions task to cancelled via tick cancel from open"` -- open task, runs `tick cancel`
5. `"it transitions task to cancelled via tick cancel from in_progress"` -- sets in_progress, runs `tick cancel`
6. `"it transitions task to open via tick reopen from done"` -- sets done with closed timestamp, runs `tick reopen`
7. `"it transitions task to open via tick reopen from cancelled"` -- sets cancelled with closed timestamp, runs `tick reopen`
8. `"it outputs status transition line on success"` -- runs `tick start`, checks exact output `"tick-aaaaaa: open -> in_progress\n"` (note: compares full string including newline, not trimmed)
9. `"it suppresses output with --quiet flag"` -- runs `tick --quiet start`, checks stdout empty
10. `"it errors when task ID argument is missing"` -- iterates 4 commands, checks exit code 1, checks exact expected message
11. `"it errors when task ID is not found"` -- runs `tick start tick-nonexist`, checks "not found" in stderr
12. `"it errors on invalid transition"` -- done task, runs `tick start`, checks "Cannot" in stderr
13. `"it writes errors to stderr"` -- runs `tick start` (no ID), checks stderr non-empty and stdout empty
14. `"it exits with code 1 on error"` -- runs `tick done tick-nonexist`, checks exit code 1
15. `"it normalizes task ID to lowercase"` -- runs `tick start TICK-AAAAAA`, verifies task transitioned
16. `"it persists status change via atomic write"` -- runs `tick start`, re-reads from file, verifies persisted status
17. `"it sets closed timestamp on done/cancel"` -- table-driven for done and cancel, verifies `Closed != nil`
18. `"it clears closed timestamp on reopen"` -- done task with closed, runs `tick reopen`, verifies `Closed == nil`

**V5 test structure notes:**
- Single top-level `TestTransition` with all subtests nested via `t.Run`
- Task setup uses constructor: `task.NewTask("tick-aaaaaa", "Start me")`, then mutates status where needed
- Uses `time.Now().UTC().Truncate(time.Second)` for closed timestamps (not fixed dates)
- Helper functions: `initTickProject(t)`, `initTickProjectWithTasks(t, tasks)`, `readTasksFromFile(t, dir)`
- CLI invocation: `Run([]string{"tick", ...}, dir, &stdout, &stderr, false)` -- free function, not method
- Total: 1 top-level function, 18 distinct test scenarios

#### Test Coverage Diff

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| start: open -> in_progress | Yes | Yes |
| done: open -> done | Yes | Yes |
| done: in_progress -> done | Yes | Yes |
| cancel: open -> cancelled | Yes | Yes |
| cancel: in_progress -> cancelled | Yes | Yes |
| reopen: done -> open | Yes | Yes |
| reopen: cancelled -> open | Yes | Yes |
| Output format verified | Yes (trimmed) | Yes (with newline) |
| --quiet suppresses output | Yes | Yes |
| Missing ID (all 4 commands) | Yes (4 subtests) | Yes (4 subtests) |
| Missing ID usage hint format | Yes (checks `Usage: tick {cmd} <id>`) | Yes (checks full message) |
| Task not found error | Yes | Yes |
| Not found error contains task ID | Yes (`"tick-nonexist"`) | No (checks `"not found"` only) |
| Invalid transition error | Yes | Yes |
| Invalid transition error content | Checks `"Error:"` | Checks `"Cannot"` |
| Errors written to stderr | Yes | Yes |
| No stdout on error | Yes | Yes |
| Exit code 1 on error (two scenarios) | Yes (missing + not found) | Yes (not found only) |
| Normalize ID to lowercase | Yes | Yes |
| Lowercase output verified | Yes (`strings.HasPrefix`) | No (not explicitly checked) |
| Persistence verified | Yes | Yes |
| Updated timestamp refreshed | Yes (`tasks[0].Updated.After(now)`) | No (not checked) |
| Closed set on done | Yes (table-driven) | Yes (table-driven) |
| Closed set on cancel | Yes (table-driven) | Yes (table-driven) |
| Closed cleared on reopen | Yes | Yes |

**Notable coverage differences:**

- V4 verifies the `Updated` timestamp was refreshed after transition (`tasks[0].Updated.After(now)`) in the persistence test. V5 does not check this, which is a gap given the spec says "updated refreshed" as part of timestamp management.
- V4 verifies the output starts with the lowercase ID (`strings.HasPrefix(output, "tick-aaa111:")`) in the normalize test. V5 only verifies the task status changed, not the output formatting with the normalized ID.
- V4 verifies the not-found error contains the actual task ID (`"tick-nonexist"`). V5 only checks for `"not found"`.
- V4's exit code test covers two scenarios (missing ID + not found) in one subtest. V5 tests only one scenario but the individual error tests also verify exit codes.
- V5 checks for "Cannot" in the invalid transition error, which matches the actual error message from `task.Transition`. V4 only checks for "Error:" which is the wrapper prefix, not the actual transition error content.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Handle all errors explicitly (no naked returns) | PASS -- all errors from `DiscoverTickDir`, `openStore`, `s.Mutate` are checked and returned | PASS -- all errors from `DiscoverTickDir`, `engine.NewStore`, `store.Mutate` are checked and returned |
| Write table-driven tests with subtests | PARTIAL -- uses `t.Run` subtests throughout, but most are individual top-level functions; only `TestTransition_SetsClosedTimestamp` and `TestTransition_ErrorMissingID` use table/iteration patterns | PARTIAL -- uses `t.Run` subtests throughout under single `TestTransition`; only closed timestamp and missing ID use table/iteration patterns |
| Document all exported functions, types, and packages | PASS -- `runTransition` is unexported but documented; package-level doc exists in `cli.go` | PASS -- `runTransition` is unexported but documented; package-level doc exists in `cli.go` |
| Propagate errors with fmt.Errorf("%w", err) | PARTIAL -- errors from `task.Transition` and store are returned directly (not wrapped with additional context); missing ID and not-found errors use `fmt.Errorf` without wrapping | PARTIAL -- same pattern; direct returns without wrapping. Neither version wraps the transition or store errors with additional context |
| No hardcoded configuration | PASS -- no magic values; error message format strings are appropriate inline constants | PASS -- no magic values |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Avoid _ assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |

### Spec-vs-Convention Conflicts

**1. Capitalized error messages**

- **Spec says:** `"Error: Task ID is required. Usage: tick {command} <id>"` and `"Task '{id}' not found"` -- both capitalized.
- **Go convention:** Error strings should not be capitalized or end with punctuation (per Go Code Review Comments and `golangci-lint` default checks).
- **V4 chose:** Capitalized errors: `"Task ID is required. Usage: tick %s <id>"`, `"Task '%s' not found"`.
- **V5 chose:** Capitalized errors: identical messages.
- **Assessment:** Both versions follow the spec verbatim. Since these are user-facing CLI error messages (not programmatic error values being wrapped/composed), capitalization is arguably appropriate. The Go convention applies more to library errors that get composed with `%w`. For CLI output, matching the spec is the right call.

**2. Error wrapping depth**

- **Spec says:** Nothing about error wrapping specifically.
- **Skill says:** "Propagate errors with fmt.Errorf("%w", err)".
- **Both versions chose:** Return errors from `task.Transition()` and the store directly without additional wrapping. For example, `return nil, err` rather than `return nil, fmt.Errorf("transition failed: %w", err)`.
- **Assessment:** Direct error return is acceptable here because the transition error already contains full context (task ID, command, status). Adding another wrapping layer would be redundant.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 5 (cli.go, transition.go, transition_test.go, 2 docs) | 5 (cli.go, transition.go, transition_test.go, 2 docs) |
| Lines added (total) | 520 | 406 |
| Lines added (internal/) | 517 | 403 |
| Impl LOC (transition.go) | 63 | 62 |
| Test LOC (transition_test.go) | 446 | 341 |
| cli.go lines changed | +6 | +4 |
| Top-level test functions | 14 | 1 |
| Total test subtests | 18 | 18 |

## Verdict

**V4 is the slightly better implementation.**

The implementations are very close -- both fully satisfy all 7 acceptance criteria and produce functionally identical behavior. The deciding factors are small but measurable:

1. **Test thoroughness (V4 advantage):** V4 has stronger test assertions in several areas:
   - Verifies `Updated` timestamp was refreshed after transition (`tasks[0].Updated.After(now)`) -- directly tests the "updated refreshed" acceptance criterion. V5 does not check this.
   - Verifies output uses lowercase ID in the normalize test (`strings.HasPrefix(output, "tick-aaa111:")`). V5 only checks status change.
   - Verifies the not-found error contains the actual task ID. V5 only checks for generic "not found" text.
   - Tests exit code 1 with two distinct scenarios in one test. V5 uses one.

2. **Defensive input handling (V4 minor advantage):** V4 applies `strings.TrimSpace(args[0])` before normalization, handling edge cases where the ID argument has trailing whitespace. V5 does not. This is a minor robustness difference since shells typically strip whitespace from arguments.

3. **Architectural approach (V5 advantage):** V5's closure-returning `runTransition` pattern and command map dispatch is more extensible and eliminates boilerplate. V5's `TransitionData` struct parameter to `FormatTransition` is more future-proof than V4's individual string parameters. V5's value-type `TransitionResult` return is more idiomatic for small structs.

4. **Mutate callback style (V5 slight advantage):** V5's inline early-return loop is more compact and idiomatic than V4's two-phase index-then-act approach.

5. **Test organization (different, not better/worse):** V4 uses 14 separate top-level functions for 18 subtests, making it easy to run individual scenarios but creating more boilerplate. V5 uses a single `TestTransition` with 18 subtests, which is more compact but makes individual scenario isolation slightly less convenient.

Overall, V4's test coverage advantage -- particularly the `Updated` timestamp verification which directly maps to an acceptance criterion about timestamp management -- gives it the edge. The architectural elegance of V5's dispatch pattern and closure approach is genuinely better at the framework level, but since this task is specifically about the transition commands' correctness, the more thorough verification wins. The margin is narrow.
