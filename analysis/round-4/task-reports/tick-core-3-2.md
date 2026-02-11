# Task tick-core-3-2: tick dep add & tick dep rm commands

## Task Summary

This task wires up two CLI sub-subcommands under `tick dep` -- `add` and `rm` -- for managing task dependencies post-creation. `tick dep add <task_id> <blocked_by_id>` looks up both tasks, validates constraints (self-ref, duplicate, cycle, child-blocked-by-parent via `task.ValidateDependency`), appends to `blocked_by`, refreshes the `updated` timestamp, and persists atomically. `tick dep rm <task_id> <blocked_by_id>` finds the task, verifies the dependency exists in its `blocked_by` slice, removes it, refreshes `updated`, and persists. `rm` does NOT validate that `blocked_by_id` exists as a task (supporting stale ref removal). Both output confirmation messages unless `--quiet` is set. Errors go to stderr with exit code 1.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|----|----|
| `dep add` adds dependency and outputs confirmation | PASS | PASS |
| `dep rm` removes dependency and outputs confirmation | PASS | PASS |
| Non-existent IDs return error | PASS | PASS |
| Duplicate/missing dep return error | PASS | PASS |
| Self-ref, cycle, child-blocked-by-parent return error | PASS | PASS |
| IDs normalized to lowercase | PASS | PASS |
| `--quiet` suppresses output | PASS | PASS |
| `updated` timestamp refreshed | PASS | PASS |
| Persisted through storage engine | PASS | PASS |

Both versions satisfy all acceptance criteria.

## Implementation Comparison

### Approach

**V5** (`internal/cli/dep.go`, 166 lines) uses the `Context`-based dispatch pattern. A single `runDep` function is registered in the `commands` map in `cli.go`, and it manually dispatches to `runDepAdd` or `runDepRm` based on `ctx.Args[0]`:

```go
// V5 cli.go — command registration
"dep":    runDep,
```

```go
// V5 dep.go lines 14-29 — subcommand dispatch
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

`runDepAdd` and `runDepRm` are unexported functions that accept `*Context` and `[]string` args. Each function normalizes IDs inline at the top, then opens the store:

```go
// V5 dep.go lines 35-41 — ID normalization in runDepAdd
func runDepAdd(ctx *Context, args []string) error {
    if len(args) < 2 {
        return fmt.Errorf("dep add requires two IDs: tick dep add <task_id> <blocked_by_id>")
    }
    taskID := task.NormalizeID(args[0])
    blockedByID := task.NormalizeID(args[1])
```

V5 uses `engine.NewStore(tickDir)` for the storage engine:

```go
// V5 dep.go lines 48-49
store, err := engine.NewStore(tickDir)
if err != nil {
```

For the add Mutate callback, V5 builds a map-based lookup for efficiency:

```go
// V5 dep.go lines 54-57 — map-based ID lookup
existing := make(map[string]int, len(tasks))
for i, t := range tasks {
    existing[t.ID] = i
}
```

V5 checks duplicate deps by normalizing stored values:

```go
// V5 dep.go lines 69-72 — duplicate check normalizes existing deps
for _, dep := range tasks[taskIdx].BlockedBy {
    if task.NormalizeID(dep) == blockedByID {
        return nil, fmt.Errorf("Task '%s' is already blocked by '%s'", taskID, blockedByID)
    }
}
```

For rm, V5 uses linear scan (no map) and also normalizes stored deps:

```go
// V5 dep.go lines 118-124 — rm linear scan with normalization
for i, dep := range tasks[taskIdx].BlockedBy {
    if task.NormalizeID(dep) == blockedByID {
        depIdx = i
        break
    }
}
```

**V6** (`internal/cli/dep.go`, 193 lines) uses the `App` struct pattern. A `handleDep` method is registered in the switch dispatch in `app.go`, which resolves the working directory and dispatches to exported `RunDepAdd`/`RunDepRm` functions:

```go
// V6 app.go lines 55-56 — dispatch
case "dep":
    err = a.handleDep(flags, subArgs)
```

```go
// V6 dep.go lines 14-36 — handleDep method
func (a *App) handleDep(flags globalFlags, subArgs []string) error {
    dir, err := a.Getwd()
    if err != nil {
        return fmt.Errorf("could not determine working directory: %w", err)
    }
    if len(subArgs) == 0 {
        return fmt.Errorf("sub-command required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>")
    }
    subCmd := subArgs[0]
    rest := subArgs[1:]
    switch subCmd {
    case "add":
        return RunDepAdd(dir, flags.quiet, rest, a.Stdout)
    case "rm":
        return RunDepRm(dir, flags.quiet, rest, a.Stdout)
    default:
        return fmt.Errorf("unknown dep sub-command '%s'. Usage: tick dep <add|rm> <task_id> <blocked_by_id>", subCmd)
    }
}
```

V6 extracts a shared `parseDepArgs` helper that strips flag-like args and extracts two positional IDs:

```go
// V6 dep.go lines 39-56 — shared arg parser
func parseDepArgs(args []string, subCmd string) (string, string, error) {
    var positional []string
    for _, arg := range args {
        if strings.HasPrefix(arg, "-") {
            continue
        }
        positional = append(positional, arg)
    }
    if len(positional) < 2 {
        return "", "", fmt.Errorf("two IDs required. Usage: tick dep %s <task_id> <blocked_by_id>", subCmd)
    }
    taskID := task.NormalizeID(positional[0])
    blockedByID := task.NormalizeID(positional[1])
    return taskID, blockedByID, nil
}
```

V6 uses `storage.NewStore(tickDir)` and performs linear scans for task lookup (no map):

```go
// V6 dep.go lines 82-90 — linear scan for task_id
taskIdx := -1
for i := range tasks {
    if tasks[i].ID == taskID {
        taskIdx = i
        break
    }
}
if taskIdx == -1 {
    return nil, fmt.Errorf("task '%s' not found", taskID)
}
```

V6 does NOT normalize stored deps during duplicate/rm checks -- it compares directly:

```go
// V6 dep.go lines 103-106 — duplicate check, direct comparison
for _, dep := range tasks[taskIdx].BlockedBy {
    if dep == blockedByID {
        return nil, fmt.Errorf("dependency already exists: %s is already blocked by %s", taskID, blockedByID)
    }
}
```

```go
// V6 dep.go lines 156-161 — rm lookup, direct comparison
for i, dep := range tasks[taskIdx].BlockedBy {
    if dep == blockedByID {
        depIdx = i
        break
    }
}
```

**Key structural differences:**

1. **Arg parsing**: V5 inlines ID extraction at the top of each function. V6 extracts a shared `parseDepArgs` helper used by both `RunDepAdd` and `RunDepRm`, which is DRYer and also strips unknown flags from the arg list.

2. **Task lookup (add)**: V5 builds a `map[string]int` for O(1) lookup of both IDs. V6 uses two sequential linear scans. For small task lists this is irrelevant; V5 is technically more efficient for large datasets.

3. **Duplicate/rm normalization**: V5 calls `task.NormalizeID(dep)` on each stored dependency during duplicate and rm checks. V6 compares directly (`dep == blockedByID`). Since IDs are already normalized at write time, V6's direct comparison is correct, but V5's approach is more defensive against data written by older code paths.

4. **Function visibility**: V5 keeps `runDep`, `runDepAdd`, `runDepRm` unexported. V6 exports `RunDepAdd` and `RunDepRm` as independently callable functions with explicit parameter lists (`dir string, quiet bool, args []string, stdout io.Writer`).

5. **Error wrapping**: V6's `handleDep` wraps the `Getwd` error with `%w`. V5 has no equivalent since `WorkDir` is pre-resolved in `Context`.

6. **Error message casing**: V5 uses `"Task '%s' not found"` and `"Task '%s' is already blocked by '%s'"` (capital T). V6 uses `"task '%s' not found"` and `"dependency already exists: %s is already blocked by %s"` (lowercase, Go convention).

7. **Missing subcommand error**: V5 says `"dep requires a subcommand: add, rm"`. V6 says `"sub-command required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>"` -- V6 includes full usage syntax.

8. **Rm error message**: V5 uses `"Task '%s' is not blocked by '%s'"`. V6 uses `"%s is not a dependency of %s"` -- different phrasing, both clear.

### Code Quality

**Go idioms:**

- Both versions follow idiomatic Go: early error returns, `defer store.Close()`, explicit error handling.
- V6's `parseDepArgs` helper is a good DRY extraction. It also silently skips flag-like args (`strings.HasPrefix(arg, "-")`), which is a pragmatic approach to handling flags that may appear between positional args.
- V5's map-based lookup is a micro-optimization pattern common in Go code that processes collections.

**Error message casing:**

V5 capitalizes error nouns (`"Task '%s' not found"`, `"Task '%s' is already blocked by '%s'"`), matching spec text verbatim. V6 uses lowercase (`"task '%s' not found"`, `"dependency already exists: ..."`), following the Go convention that error strings should not be capitalized (`go vet` recommendation). Since the dispatcher prepends `"Error: "`, V6's lowercase is more Go-idiomatic.

**Imports:**

- V5 imports `fmt`, `time`, `engine`, `task` (4 imports).
- V6 imports `fmt`, `io`, `strings`, `time`, `storage`, `task` (6 imports). The extra `io` and `strings` imports come from the explicit `io.Writer` parameter and `parseDepArgs` flag stripping.

**Defensive normalization:**

V5's choice to call `task.NormalizeID(dep)` on stored dependencies during duplicate/rm checks is more defensive:

```go
// V5 dep.go line 70 — defensive normalization on stored values
if task.NormalizeID(dep) == blockedByID {
```

V6 trusts that stored data is already normalized:

```go
// V6 dep.go line 105 — direct comparison
if dep == blockedByID {
```

V6's approach is cleaner but less robust against data corruption or legacy entries.

### Test Quality

**V5** (`internal/cli/dep_test.go`, 503 lines) has two top-level functions: `TestDepAdd` (12 subtests) and `TestDepRm` (10 subtests). Tests invoke the full CLI via `Run([]string{"tick", ...}, dir, &stdout, &stderr, false)`.

**V6** (`internal/cli/dep_test.go`, 615 lines) has three top-level functions: `TestDepAdd` (12 subtests), `TestDepRm` (10 subtests), `TestDepNoSubcommand` (1 subtest). Tests use a dedicated `runDep` helper that constructs an `App` struct.

**V6 test helper:**

```go
// V6 dep_test.go lines 12-25 — dedicated helper
func runDep(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
    t.Helper()
    var stdoutBuf, stderrBuf bytes.Buffer
    app := &App{
        Stdout: &stdoutBuf,
        Stderr: &stderrBuf,
        Getwd:  func() (string, error) { return dir, nil },
    }
    fullArgs := append([]string{"tick", "dep"}, args...)
    code := app.Run(fullArgs)
    return stdoutBuf.String(), stderrBuf.String(), code
}
```

V5 has no dedicated dep test helper -- each test declares its own `bytes.Buffer` variables.

**Complete test list:**

| # | Spec Test Name | V5 Location | V6 Location |
|---|---------------|-------------|-------------|
| 1 | `it adds a dependency between two existing tasks` | `TestDepAdd` line 14 | `TestDepAdd` line 31 |
| 2 | `it outputs confirmation on success (add)` | `TestDepAdd` line 34 | `TestDepAdd` line 51 |
| 3 | `it updates task's updated timestamp (add)` | `TestDepAdd` line 53 | `TestDepAdd` line 72 |
| 4 | `it errors when task_id not found (add)` | `TestDepAdd` line 80 | `TestDepAdd` line 94 |
| 5 | `it errors when blocked_by_id not found (add)` | `TestDepAdd` line 95 | `TestDepAdd` line 109 |
| 6 | `it errors on duplicate dependency (add)` | `TestDepAdd` line 110 | `TestDepAdd` line 123 |
| 7 | `it errors on self-reference (add)` | `TestDepAdd` line 128 | `TestDepAdd` line 141 |
| 8 | `it errors when add creates cycle` | `TestDepAdd` line 143 | `TestDepAdd` line 155 |
| 9 | `it errors when add creates child-blocked-by-parent` | `TestDepAdd` line 165 | `TestDepAdd` line 174 |
| 10 | `it normalizes IDs to lowercase (add)` | `TestDepAdd` line 182 | `TestDepAdd` line 193 |
| 11 | `it suppresses output with --quiet (add)` | `TestDepAdd` line 205 | `TestDepAdd` line 216 |
| 12a | `it errors when fewer than two IDs provided (add) — no IDs` | `TestDepAdd` line 221 (table-driven, sub: "no IDs") | `TestDepAdd` line 244 (separate subtest) |
| 12b | `it errors when fewer than two IDs provided (add) — one ID` | `TestDepAdd` line 221 (table-driven, sub: "one ID") | `TestDepAdd` line 234 (separate subtest) |
| 13 | `it persists via atomic write (add)` | `TestDepAdd` line 247 | `TestDepAdd` line 254 |
| 14 | `it removes an existing dependency` | `TestDepRm` line 271 | `TestDepRm` line 282 |
| 15 | `it outputs confirmation on success (rm)` | `TestDepRm` line 296 | `TestDepRm` line 305 |
| 16 | `it updates task's updated timestamp (rm)` | `TestDepRm` line 318 | `TestDepRm` line 328 |
| 17 | `it errors when task_id not found (rm)` | `TestDepRm` line 347 | `TestDepRm` line 356 |
| 18 | `it errors when dependency not found in blocked_by (rm)` | `TestDepRm` line 362 | `TestDepRm` line 371 |
| 19 | `it removes stale dependency without validating blocked_by_id exists` | `TestDepRm` line 378 | `TestDepRm` line 543 |
| 20 | `it normalizes IDs to lowercase (rm)` | `TestDepRm` line 405 | `TestDepRm` line 388 |
| 21 | `it suppresses output with --quiet (rm)` | `TestDepRm` line 430 | `TestDepRm` line 416 |
| 22a | `it errors when fewer than two IDs provided (rm) — no IDs` | `TestDepRm` line 452 (table-driven, sub: "no IDs") | `TestDepRm` line 449 (separate subtest) |
| 22b | `it errors when fewer than two IDs provided (rm) — one ID` | `TestDepRm` line 452 (table-driven, sub: "one ID") | `TestDepRm` line 438 (separate subtest) |
| 23 | `it persists via atomic write (rm)` | `TestDepRm` line 477 | `TestDepRm` line 462 |
| 24 | `it errors when no sub-subcommand given` | NOT PRESENT | `TestDepNoSubcommand` line 578 |

**Test fixture construction:**

V5 uses `task.NewTask(id, title)` and mutates fields as needed:

```go
// V5 dep_test.go lines 15-16
t1 := task.NewTask("tick-aaaaaa", "Task A")
t2 := task.NewTask("tick-bbbbbb", "Task B")
dir := initTickProjectWithTasks(t, []task.Task{t1, t2})
```

V6 constructs full `task.Task` struct literals with all fields explicit:

```go
// V6 dep_test.go lines 32-39
taskA := task.Task{
    ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
    Priority: 2, Created: now, Updated: now,
}
taskB := task.Task{
    ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
    Priority: 2, Created: now, Updated: now,
}
dir, tickDir := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})
```

V6's approach is more verbose but fully explicit about all field values.

**Timestamp test approach:**

V5 uses `time.Sleep(1100 * time.Millisecond)` to ensure timestamp difference:

```go
// V5 dep_test.go lines 58-59
originalUpdated := t1.Updated
...
time.Sleep(1100 * time.Millisecond)
```

V6 sets the initial timestamp to 1 hour in the past and brackets the expected range:

```go
// V6 dep_test.go lines 73-74
pastTime := now.Add(-1 * time.Hour)
taskA := task.Task{
    ...Created: pastTime, Updated: pastTime,
}
...
before := time.Now().UTC().Truncate(time.Second)
_, _, exitCode := runDep(t, dir, "add", "tick-aaa111", "tick-bbb222")
...
after := time.Now().UTC().Truncate(time.Second).Add(time.Second)
...
if found.Updated.Before(before) || found.Updated.After(after) {
```

V6's approach avoids the 1.1-second sleep, making tests faster.

**Quiet flag placement:**

V5 passes `--quiet` as a global flag before the subcommand:

```go
// V5 dep_test.go line 210
code := Run([]string{"tick", "--quiet", "dep", "add", "tick-aaaaaa", "tick-bbbbbb"}, dir, &stdout, &stderr, false)
```

V6 passes `--quiet` after the dep subcommand, which ends up between positional args:

```go
// V6 dep_test.go line 268
stdout, stderr, exitCode := runDep(t, dir, "add", "--quiet", "tick-aaa111", "tick-bbb222")
```

V6's `parseArgs` extracts global flags from all positions, so `--quiet` is caught globally. The `parseDepArgs` helper also skips `-` prefixed args, so `--quiet` in `rest` is harmlessly ignored. Both placements work.

**Missing argument error assertion:**

V5 uses table-driven tests and checks for `"requires two"`:

```go
// V5 dep_test.go lines 222-240
tests := []struct {
    name string
    args []string
}{
    {"no IDs", []string{"tick", "dep", "add"}},
    {"one ID", []string{"tick", "dep", "add", "tick-aaaaaa"}},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        ...
        if !strings.Contains(stderr.String(), "requires two") {
```

V6 uses separate subtests and checks for `"Usage:"`:

```go
// V6 dep_test.go lines 234-241
t.Run("it errors when fewer than two IDs provided (add)", func(t *testing.T) {
    ...
    _, stderr, exitCode := runDep(t, dir, "add", "tick-aaa111")
    ...
    if !strings.Contains(stderr, "Usage:") {
```

V5's table-driven approach is slightly more idiomatic per the skill constraints. V6's `"Usage:"` assertion is looser but still validates a usage hint is shown.

**Additional V6 normalize assertions:**

V6's normalize tests verify that stdout contains lowercase IDs:

```go
// V6 dep_test.go lines 218-220
if !strings.Contains(stdout, "tick-aaa111") || !strings.Contains(stdout, "tick-bbb222") {
    t.Errorf("stdout should contain lowercase IDs, got %q", stdout)
}
```

V5 does not check that output uses normalized IDs -- only that the persisted data is correct.

**Additional V6 quiet stderr assertion:**

V6 checks stderr is empty during quiet tests:

```go
// V6 dep_test.go lines 275-277
if stderr != "" {
    t.Errorf("stderr should be empty on success, got %q", stderr)
}
```

V5 does not check stderr in quiet tests.

**V6 extra test -- no subcommand:**

V6 has an additional `TestDepNoSubcommand` test (line 578) that verifies `tick dep` with no sub-subcommand returns an error with usage hint. V5 does not test this path.

### Skill Compliance

| Constraint | V5 | V6 |
|-----------|----|----|
| Handle all errors explicitly (no naked returns) | PASS | PASS |
| Document all exported functions/types/packages | PASS (all functions have doc comments, even unexported) | PASS (all exported functions have doc comments; `parseDepArgs` unexported also documented) |
| Write table-driven tests with subtests | PASS (missing-arg tests use table-driven pattern) | PARTIAL (missing-arg tests use separate subtests instead of table) |
| Propagate errors with `fmt.Errorf("%w", err)` | PARTIAL -- dep-specific errors are plain strings | PARTIAL -- dep-specific errors are plain strings; `handleDep` wraps Getwd with `%w` |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic for normal error handling | PASS | PASS |
| MUST NOT hardcode configuration | PASS | PASS |
| Use `gofmt` compatible formatting | PASS | PASS |
| Run race detector on tests | Not verified at commit level | Not verified at commit level |

### Spec-vs-Convention Conflicts

1. **Error message capitalization**: V5 uses `"Task '%s' not found"` and `"Task '%s' is already blocked by '%s'"` (matches spec tone). V6 uses `"task '%s' not found"` and `"dependency already exists: %s is already blocked by %s"` (Go convention). Since the dispatcher prepends `"Error: "`, V6's lowercase is more Go-idiomatic.

2. **Rm error phrasing**: The spec says `"it errors when dependency not found (rm)"`. V5 outputs `"Task '%s' is not blocked by '%s'"` (mirrors the data model). V6 outputs `"%s is not a dependency of %s"` (uses "dependency" vocabulary). Both are spec-compliant; phrasing is not specified.

3. **Missing subcommand error phrasing**: V5 says `"dep requires a subcommand: add, rm"` (concise). V6 says `"sub-command required. Usage: tick dep <add|rm> <task_id> <blocked_by_id>"` (includes full usage). V6 is more user-friendly.

4. **Duplicate normalization on stored deps**: V5 normalizes stored deps during comparison (`task.NormalizeID(dep) == blockedByID`). V6 compares directly (`dep == blockedByID`). The spec does not specify this behavior -- V5 is more defensive, V6 trusts the data layer.

## Diff Stats

| Metric | V5 | V6 |
|--------|----|----|
| Files changed | 3 (code) + docs | 3 (code) + docs |
| `dep.go` lines | 166 | 193 |
| `dep_test.go` lines | 503 | 615 |
| Dispatcher change | +1 line in `cli.go` (1 map entry) | +2 lines in `app.go` (1 case arm) |
| Total lines added (code only) | 670 | 810 |
| Test helper functions | 0 (reuses `initTickProjectWithTasks`, `readTasksFromFile`) | 1 (`runDep` helper, 14 lines; reuses `setupTickProjectWithTasks`, `readPersistedTasks`) |
| Top-level test functions | 2 (`TestDepAdd`, `TestDepRm`) | 3 (`TestDepAdd`, `TestDepRm`, `TestDepNoSubcommand`) |
| Total `t.Run` subtests | 22 (12 add + 10 rm, incl table sub-subtests) | 23 (12 add + 10 rm + 1 no-subcommand) |
| Imports (`dep.go`) | `fmt`, `time`, `engine`, `task` | `fmt`, `io`, `strings`, `time`, `storage`, `task` |
| Shared arg parsing | None (inline in each function) | `parseDepArgs` helper shared by both |
| Timestamp test approach | `time.Sleep(1100ms)` | 1-hour offset + range bracket |

## Verdict

Both implementations are functionally complete and satisfy all 9 acceptance criteria. The core Mutate callback logic is structurally identical -- find task, validate, mutate `blocked_by`, update timestamp, persist. Differences are architectural and testing-related.

**V5 strengths:**
- Map-based ID lookup in add (`make(map[string]int, len(tasks))`) is O(1) vs V6's dual linear scan -- a micro-optimization that shows awareness of algorithmic efficiency
- Defensive normalization of stored deps during duplicate/rm checks (`task.NormalizeID(dep)`) protects against data inconsistencies
- Missing-arg tests use table-driven pattern (per skill constraint), with both "no IDs" and "one ID" cases in a single loop
- More compact overall (670 vs 810 lines added)

**V6 strengths:**
- Shared `parseDepArgs` helper eliminates argument-parsing duplication between add and rm (DRY)
- `parseDepArgs` strips unknown flags from positional args, making the command more resilient to misplaced flags
- Dedicated `runDep` test helper eliminates repeated `bytes.Buffer` boilerplate
- Timestamp tests avoid 1.1-second sleep by using 1-hour offset + range bracket -- significantly faster test execution
- Normalize tests also verify output contains lowercase IDs (not just persisted data)
- Quiet tests also assert stderr is empty
- Extra `TestDepNoSubcommand` test covers the bare `tick dep` error path
- `RunDepAdd`/`RunDepRm` are exported and independently callable
- `handleDep` wraps Getwd error with `%w` for proper error chaining
- Error messages include full usage syntax in the missing-subcommand case

**Winner: Slight edge to V6.** V6's `parseDepArgs` extraction is a clear structural improvement that removes code duplication between the two commands. The timestamp test approach (avoiding `time.Sleep`) is meaningfully better for test suite speed. The additional test coverage (no-subcommand path, lowercase output verification, stderr assertion in quiet tests) adds real value. V5's map-based lookup and defensive normalization are thoughtful but provide marginal benefit for the expected dataset size. V5's table-driven missing-arg test is slightly more idiomatic per the skill constraints, but this is a minor point against V6's overall stronger test quality.
