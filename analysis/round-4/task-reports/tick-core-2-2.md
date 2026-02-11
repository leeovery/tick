# Task tick-core-2-2: tick start, done, cancel, reopen commands

## Task Summary

This task adds four CLI subcommands -- `start`, `done`, `cancel`, `reopen` -- that mutate a task's status. Each command parses a positional task ID argument, normalizes it to lowercase, looks up the task via the storage engine's `Mutate` flow, calls the pure `task.Transition()` function (built in tick-core-2-1), persists the result atomically, and outputs `{id}: {old_status} -> {new_status}` with a unicode arrow. `--quiet` suppresses success output. Errors go to stderr with exit code 1.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|----|----|
| All four commands transition correctly and output transition line | PASS | PASS |
| Invalid transitions return error to stderr with exit code 1 | PASS | PASS |
| Missing/not-found task ID returns error with exit code 1 | PASS | PASS |
| `--quiet` suppresses success output | PASS | PASS |
| Input IDs normalized to lowercase | PASS | PASS |
| Timestamps managed correctly (closed set/cleared, updated refreshed) | PASS | PASS |
| Mutation persisted through storage engine | PASS | PASS |

Both versions satisfy all acceptance criteria.

## Implementation Comparison

### Approach

**V5** (`internal/cli/transition.go`, 58 lines) uses the `Context`-based dispatch pattern with a higher-order function. The four commands are registered in the `commands` map in `cli.go`, each calling `runTransition` with the command name to get a closure:

```go
// V5 cli.go lines 103-106 — command registration
"start":  runTransition("start"),
"done":   runTransition("done"),
"cancel": runTransition("cancel"),
"reopen": runTransition("reopen"),
```

```go
// V5 transition.go lines 14-15 — higher-order function returns closure
func runTransition(command string) func(*Context) error {
    return func(ctx *Context) error {
```

The closure receives a `*Context` carrying `WorkDir`, `Stdout`, `Stderr`, `Quiet`, and `Args`. It accesses the storage engine through `engine.NewStore(tickDir)`:

```go
// V5 transition.go lines 27-28
store, err := engine.NewStore(tickDir)
if err != nil {
```

Output is written directly with `fmt.Fprintf`:

```go
// V5 transition.go lines 52-53
if !ctx.Quiet {
    fmt.Fprintf(ctx.Stdout, "%s: %s \u2192 %s\n", id, result.OldStatus, result.NewStatus)
}
```

**V6** (`internal/cli/transition.go`, 56 lines) uses the `App` struct pattern with an exported `RunTransition` function. The four commands share a single `case` arm in `app.go`'s switch dispatch:

```go
// V6 app.go lines 68-69 — single case for all four commands
case "start", "done", "cancel", "reopen":
    err = a.handleTransition(subcmd, flags, subArgs)
```

```go
// V6 app.go lines 185-192 — handler method wraps RunTransition
func (a *App) handleTransition(command string, flags globalFlags, subArgs []string) error {
    dir, err := a.Getwd()
    if err != nil {
        return fmt.Errorf("could not determine working directory: %w", err)
    }
    return RunTransition(dir, command, flags.quiet, subArgs, a.Stdout)
}
```

`RunTransition` is a flat, exported function that takes explicit parameters instead of a context struct:

```go
// V6 transition.go lines 14-15 — exported function, explicit params
func RunTransition(dir string, command string, quiet bool, args []string, stdout io.Writer) error {
    if len(args) == 0 {
```

V6 imports `storage.NewStore` directly rather than going through an `engine` package:

```go
// V6 transition.go lines 25-26
store, err := storage.NewStore(tickDir)
if err != nil {
```

Output uses the same direct `fmt.Fprintf` approach as V5:

```go
// V6 transition.go lines 51-53
if !quiet {
    fmt.Fprintf(stdout, "%s: %s \u2192 %s\n", id, result.OldStatus, result.NewStatus)
}
```

**Core logic is identical.** Both versions share the same Mutate callback structure -- iterate tasks by index, find matching ID, call `task.Transition`, capture result:

```go
// Both versions — identical Mutate callback logic
err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
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
    return nil, fmt.Errorf("Task '%s' not found", id)  // V5 capitalizes "Task"
})
```

**Key structural difference:** V5 uses a higher-order function (`runTransition` returns a `func(*Context) error`) which is registered once per command in the map. V6 uses a flat function dispatched through a `handleTransition` method. V5's approach is more DRY at the registration site (the command name is captured in the closure), while V6's approach adds a thin `handleTransition` wrapper but keeps `RunTransition` independently callable.

**DiscoverTickDir handling:** V5 calls `DiscoverTickDir(ctx.WorkDir)` directly in the transition function. V6 also calls `DiscoverTickDir(dir)` directly (not using the shared `openStore` helper from `helpers.go`, even though it exists in the V6 codebase). This is a minor inconsistency in V6 -- other V6 commands use `openStore(dir)` but `RunTransition` does its own discovery.

### Code Quality

**Go idioms:**

- Both versions follow idiomatic Go: early error returns, `defer store.Close()`, explicit error handling on every call.
- V5's closure-returning pattern (`func runTransition(command string) func(*Context) error`) is a legitimate Go idiom for parameterized handlers but slightly unusual.
- V6's exported flat function (`RunTransition`) is simpler and more conventional.

**Error message casing:**

```go
// V5 transition.go line 17
return fmt.Errorf("Task ID is required. Usage: tick %s <id>", command)
```

```go
// V6 transition.go line 16
return fmt.Errorf("task ID is required. Usage: tick %s <id>", command)
```

V5 capitalizes "Task" to match the spec text verbatim: `Error: Task ID is required. Usage: tick {command} <id>`. V6 uses lowercase "task" following Go convention that error strings should not be capitalized (per `go vet`). The spec says the prefix is `Error:` which is added by the dispatcher, so V6's lowercase is technically more Go-idiomatic.

Same difference for the not-found error:

```go
// V5 transition.go line 46
return nil, fmt.Errorf("Task '%s' not found", id)
```

```go
// V6 transition.go line 45
return nil, fmt.Errorf("task '%s' not found", id)
```

**Imports:**

- V5 imports `github.com/leeovery/tick/internal/engine` and `github.com/leeovery/tick/internal/task`.
- V6 imports `github.com/leeovery/tick/internal/storage`, `github.com/leeovery/tick/internal/task`, and `io`.

The `engine` vs `storage` difference reflects the different package naming conventions between the two codebases. V6 additionally imports `io` because `RunTransition` accepts `io.Writer` directly rather than getting it from a context struct.

**Function visibility:**

- V5: `runTransition` is unexported (lowercase), consistent with V5's pattern where all handlers are package-private and accessed only through the `commands` map.
- V6: `RunTransition` is exported (uppercase), making it independently testable and callable from the `handleTransition` wrapper.

**Line count comparison:**

- V5 `transition.go`: 58 lines
- V6 `transition.go`: 56 lines
- Nearly identical in size, reflecting how close the core logic is.

### Test Quality

**V5** (`internal/cli/transition_test.go`, 341 lines) has one top-level `TestTransition` function with 18 named subtests. Tests invoke the full CLI via `Run([]string{"tick", ...}, dir, &stdout, &stderr, false)`.

**V6** (`internal/cli/transition_test.go`, 365 lines) has one top-level `TestTransitionCommands` function with 18 named subtests. Tests use a dedicated `runTransition` helper that constructs an `App` struct directly.

**Complete list of tests -- both versions have the same 18 spec-mandated tests:**

| # | Test Name | V5 | V6 |
|---|-----------|----|----|
| 1 | `it transitions task to in_progress via tick start` | line 13 | line 31 |
| 2 | `it transitions task to done via tick done from open` | line 33 | line 52 |
| 3 | `it transitions task to done via tick done from in_progress` | line 50 | line 70 |
| 4 | `it transitions task to cancelled via tick cancel from open` | line 68 | line 88 |
| 5 | `it transitions task to cancelled via tick cancel from in_progress` | line 85 | line 106 |
| 6 | `it transitions task to open via tick reopen from done` | line 103 | line 124 |
| 7 | `it transitions task to open via tick reopen from cancelled` | line 123 | line 143 |
| 8 | `it outputs status transition line on success` | line 143 | line 162 |
| 9 | `it suppresses output with --quiet flag` | line 160 | line 180 |
| 10 | `it errors when task ID argument is missing` | line 176 | line 199 |
| 11 | `it errors when task ID is not found` | line 196 | line 214 |
| 12 | `it errors on invalid transition` | line 210 | line 226 |
| 13 | `it writes errors to stderr` | line 228 | line 243 |
| 14 | `it exits with code 1 on error` | line 245 | line 255 |
| 15 | `it normalizes task ID to lowercase` | line 256 | line 264 |
| 16 | `it persists status change via atomic write` | line 273 | line 288 |
| 17 | `it sets closed timestamp on done/cancel` | line 294 (table-driven, 2 sub-tests: `done`, `cancel`) | line 314 (table-driven, 2 sub-tests: `done sets closed`, `cancel sets closed`) |
| 18 | `it clears closed timestamp on reopen` | line 322 | line 348 |

**Test fixture construction:**

V5 uses `task.NewTask(id, title)` to construct test tasks, then mutates fields as needed:

```go
// V5 transition_test.go line 14
tk := task.NewTask("tick-aaaaaa", "Start me")
dir := initTickProjectWithTasks(t, []task.Task{tk})
```

```go
// V5 transition_test.go lines 51-52 — mutating status for in_progress test
tk := task.NewTask("tick-aaaaaa", "Done from IP")
tk.Status = task.StatusInProgress
```

V6 constructs `task.Task` struct literals directly with all fields explicit:

```go
// V6 transition_test.go lines 32-35
openTask := task.Task{
    ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
    Priority: 2, Created: now, Updated: now,
}
dir, tickDir := setupTickProjectWithTasks(t, []task.Task{openTask})
```

V6's approach is more explicit -- every field is visible. V5 relies on `NewTask` defaults being correct. V6 also pre-computes `now` once at the top of `TestTransitionCommands` (line 29) and reuses it for all fixtures, while V5 creates `now` only in the tests that need it (reopen tests).

**Test helper differences:**

V5's `runTransition` invocations go through the full `Run` function:

```go
// V5 — full CLI invocation
code := Run([]string{"tick", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)
```

V6 has a dedicated `runTransition` helper (line 14-26) that constructs a minimal `App`:

```go
// V6 transition_test.go lines 14-26
func runTransition(t *testing.T, dir string, command string, args ...string) (stdout string, stderr string, exitCode int) {
    t.Helper()
    var stdoutBuf, stderrBuf bytes.Buffer
    app := &App{
        Stdout: &stdoutBuf,
        Stderr: &stderrBuf,
        Getwd:  func() (string, error) { return dir, nil },
    }
    fullArgs := append([]string{"tick", command}, args...)
    code := app.Run(fullArgs)
    return stdoutBuf.String(), stderrBuf.String(), code
}
```

V6's helper returns `(stdout, stderr string, exitCode int)` as strings directly, eliminating boilerplate `bytes.Buffer` declarations in each test. V5 declares `var stdout, stderr bytes.Buffer` in every single test body, which is repetitive.

**Missing ID error test:**

V5 tests all four commands in a loop with subtests:

```go
// V5 transition_test.go lines 179-193
commands := []string{"start", "done", "cancel", "reopen"}
for _, cmd := range commands {
    t.Run(cmd, func(t *testing.T) {
        ...
        expectedMsg := "Task ID is required. Usage: tick " + cmd + " <id>"
        if !strings.Contains(stderr.String(), expectedMsg) {
```

V6 tests only `start`:

```go
// V6 transition_test.go lines 199-211
_, stderr, exitCode := runTransition(t, dir, "start")
...
if !strings.Contains(stderr, "Error:") {
if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "start") {
```

V5 is more thorough here -- it verifies all four commands produce the correct usage hint with the right command name. V6 only tests `start` and checks for generic "Error:" and "Usage:" strings rather than the exact message.

**Invalid transition error assertion:**

```go
// V5 transition_test.go line 223
if !strings.Contains(stderr.String(), "Cannot") {
```

```go
// V6 transition_test.go line 238
if !strings.Contains(stderr, "cannot start") {
```

V6's assertion is stricter -- it checks for "cannot start" which validates both the error verb and the command name. V5 only checks for "Cannot" (capital C, matching V5's capitalized error messages).

**Quiet flag test:**

V5 passes `--quiet` as a global flag before the subcommand:

```go
// V5 transition_test.go line 165
code := Run([]string{"tick", "--quiet", "start", "tick-aaaaaa"}, dir, &stdout, &stderr, false)
```

V6 passes `--quiet` as a subcommand-level flag after the command:

```go
// V6 transition_test.go line 187
stdout, stderr, exitCode := runTransition(t, dir, "start", "--quiet", "tick-aaa111")
```

This tests different flag parsing paths: V5 tests global flag extraction, V6 tests that `--quiet` works when passed after the subcommand name. Both are valid since V6's `parseArgs` extracts global flags from all positions.

V6's quiet test additionally checks stderr is empty:

```go
// V6 transition_test.go lines 193-195
if stderr != "" {
    t.Errorf("stderr should be empty on success, got %q", stderr)
}
```

V5 does not check stderr in the quiet test.

**Closed timestamp precision test:**

V6's "sets closed timestamp" test captures `before` and `after` timestamps to bracket the expected closed time:

```go
// V6 transition_test.go lines 330-341
before := time.Now().UTC().Truncate(time.Second)
_, _, exitCode := runTransition(t, dir, tt.command, "tick-aaa111")
...
after := time.Now().UTC().Truncate(time.Second)
...
if tasks[0].Closed.Before(before) || tasks[0].Closed.After(after) {
    t.Errorf("closed timestamp %v not in expected range [%v, %v]", ...)
}
```

V5 only checks that Closed is non-nil:

```go
// V5 transition_test.go lines 315-316
if tasks[0].Closed == nil {
    t.Error("expected closed timestamp to be set")
}
```

V6's approach is more rigorous -- it validates the timestamp is within an expected range, not just present.

**Persistence test:**

V6 additionally checks the `Updated` timestamp was refreshed:

```go
// V6 transition_test.go lines 308-310
if !tasks[0].Updated.After(now.Add(-time.Second)) {
    t.Error("updated timestamp should be refreshed")
}
```

V5 does not verify the Updated timestamp in the persistence test.

**Normalize test:**

V6 additionally verifies the output contains the lowercase ID:

```go
// V6 transition_test.go lines 282-284
if !strings.Contains(stdout, "tick-aaa111") {
    t.Errorf("stdout should contain lowercase ID, got %q", stdout)
}
```

V5 only checks that the task status changed, not that the output uses the normalized ID.

### Skill Compliance

| Constraint | V5 | V6 |
|-----------|----|----|
| Handle all errors explicitly (no naked returns) | PASS | PASS |
| Document all exported functions/types/packages | PASS (`runTransition` is unexported, has doc comment) | PASS (`RunTransition` is exported, has doc comment) |
| Write table-driven tests with subtests | PASS (closed timestamp test uses table) | PASS (closed timestamp test uses table) |
| Propagate errors with `fmt.Errorf("%w", err)` | PARTIAL -- transition errors are plain strings, not wrapped | PARTIAL -- same; V6's `handleTransition` wraps Getwd error with `%w` |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic for normal error handling | PASS | PASS |
| MUST NOT hardcode configuration | PASS | PASS |
| Use `gofmt` compatible formatting | PASS | PASS |
| Run race detector on tests | Not verified at commit level | Not verified at commit level |

V6's `handleTransition` in `app.go` line 189 wraps the Getwd error with `%w`:
```go
return fmt.Errorf("could not determine working directory: %w", err)
```

V5 does not have this wrapper since `WorkDir` is already resolved in `Context`.

### Spec-vs-Convention Conflicts

1. **Error message capitalization**: The spec says `Error: Task ID is required. Usage: tick {command} <id>`. V5 outputs `Error: Task ID is required...` (matching spec). V6 outputs `Error: task ID is required...` (Go convention for lowercase error strings). Since the `Error: ` prefix is added by the dispatcher, V6's lowercase is more Go-idiomatic but deviates from the spec text.

2. **Not-found message capitalization**: V5 uses `Task '%s' not found` (capital T); V6 uses `task '%s' not found` (lowercase t). Same convention-vs-spec tension.

3. **Output format**: The spec requires `{id}: {old_status} -> {new_status}` with a unicode arrow (U+2192). Both versions correctly use `\u2192` in `fmt.Fprintf`. Both produce, e.g., `tick-aaaaaa: open -> in_progress\n`.

## Diff Stats

| Metric | V5 | V6 |
|--------|----|----|
| Files changed | 4 (`cli.go` +4, `transition.go` +58, `transition_test.go` +341, docs) | 4 (`app.go` +11, `transition.go` +56, `transition_test.go` +365, docs) |
| Lines added (code, excl docs) | 403 | 432 |
| `transition.go` lines | 58 | 56 |
| `transition_test.go` lines | 341 | 365 |
| Dispatcher change | +4 lines in `cli.go` (4 map entries) | +11 lines in `app.go` (1 case + `handleTransition` method) |
| Test helper functions added | 0 (reuses `initTickProject*`, `readTasksFromFile` from create_test.go) | 1 (`runTransition` helper, 13 lines) |
| Test subtests (total `t.Run`) | 18 + 4 sub-sub (missing-ID tests all 4 cmds) + 2 sub-sub (closed timestamp) = 24 | 18 + 2 sub-sub (closed timestamp) = 20 |
| Imports (transition.go) | `fmt`, `engine`, `task` | `fmt`, `io`, `storage`, `task` |

## Verdict

Both implementations are functionally equivalent and satisfy all 7 acceptance criteria. The core Mutate callback logic is nearly character-for-character identical. Differences are architectural and testing-related.

**V5 strengths:**
- Missing-ID error test covers all four commands via loop with subtests (24 total `t.Run` calls vs V6's 20), verifying each command produces its own usage hint
- Higher-order function pattern (`runTransition("start")`) is more DRY at the registration site -- 4 map entries vs a case arm plus a 7-line wrapper method
- Error message capitalization matches spec text verbatim

**V6 strengths:**
- Dedicated `runTransition` test helper eliminates repeated `bytes.Buffer` boilerplate (DRYer test bodies)
- More precise timestamp assertions: closed timestamp is range-checked, Updated timestamp is verified in persistence test
- Normalize test also checks that output contains the lowercase ID
- Quiet test also asserts stderr is empty
- Invalid-transition test asserts `"cannot start"` (stricter) vs V5's generic `"Cannot"`
- `RunTransition` is exported, independently callable/testable
- `handleTransition` wraps Getwd error with `%w` (better error chain)

**Winner: Slight edge to V6.** The test quality advantage is the deciding factor. V6's timestamp range assertions, Updated-timestamp verification, lowercase-output check in the normalize test, and stderr-empty check in the quiet test all provide meaningfully better coverage for the same 18 test names. V5's advantage of testing all four commands for the missing-ID error is real but less impactful since the underlying code path is shared (the same `RunTransition`/`runTransition` function handles all four). V6's dedicated test helper also produces cleaner, more maintainable test code. The core implementation quality is effectively tied.
