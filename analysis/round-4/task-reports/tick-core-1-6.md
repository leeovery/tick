# Task tick-core-1-6: tick create command

## Task Summary

This task implements `tick create`, the first mutation command in the CLI. It takes a required title and optional flags (`--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`), generates a `tick-{6 hex}` ID, validates all inputs (title length/content, priority range, reference IDs exist, no self-references), persists via the storage engine's `Mutate` flow (which provides atomic write), and outputs task details (or just the ID with `--quiet`). The `--blocks` flag is syntactic sugar that modifies other tasks' `blocked_by` arrays during creation.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|----|----|
| `tick create "<title>"` creates task with defaults (open, priority 2) | PASS | PASS |
| Generated ID follows `tick-{6 hex}` format, unique among existing | PASS | PASS |
| All optional flags work: `--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent` | PASS | PASS |
| `--blocks` correctly updates referenced tasks' `blocked_by` arrays | PASS | PASS |
| Missing or empty title returns error to stderr with exit code 1 | PASS | PASS |
| Invalid priority returns error with exit code 1 | PASS | PASS |
| Non-existent IDs in references return error with exit code 1 | PASS | PASS |
| Task persisted via atomic write through storage engine | PASS | PASS |
| SQLite cache updated as part of write flow | PASS (via engine.Mutate) | PASS (via storage.Mutate) |
| Output shows task details on success | PASS | PASS |
| `--quiet` outputs only task ID | PASS | PASS |
| Input IDs normalized to lowercase | PASS | PASS |
| Timestamps set to current UTC ISO 8601 | PASS | PASS |

Both versions satisfy all acceptance criteria.

## Implementation Comparison

### Approach

**V5** (`internal/cli/create.go`, 264 lines) uses a `Context`-based dispatch pattern. The create command is registered in a `commands` map in `cli.go` and receives a `*Context` struct containing `WorkDir`, `Stdout`, `Stderr`, `Quiet`, `Verbose`, `Format`, and `Args`:

```go
// V5 cli.go — command registration
var commands = map[string]func(*Context) error{
	"init":   runInit,
	"create": runCreate,
}
```

```go
// V5 create.go — handler signature
func runCreate(ctx *Context) error {
	title, opts, err := parseCreateArgs(ctx.Args)
	...
}
```

V5's `parseCreateArgs` returns `(string, createOpts, error)` -- title and options separately. It normalizes IDs **after** parsing, in the mutation closure:

```go
// V5 — normalization is deferred to mutation closure
blockedBy := normalizeIDs(opts.blockedBy)
blocks := normalizeIDs(opts.blocks)
parent := task.NormalizeID(opts.parent)
```

V5 also uses the task package's `ValidateBlockedBy` and `ValidateParent` helper functions for self-reference checks.

**V6** (`internal/cli/create.go`, 242 lines) uses an `App` struct pattern. The create command is dispatched via a `switch` in `app.go`, and the handler is an exported function `RunCreate(dir, quiet, args, stdout)`:

```go
// V6 app.go — switch dispatch
case "create":
    err = a.handleCreate(flags, subArgs)
```

```go
// V6 app.go — handler wrapping
func (a *App) handleCreate(flags globalFlags, subArgs []string) error {
	dir, err := a.Getwd()
	...
	return RunCreate(dir, flags.quiet, subArgs, a.Stdout)
}
```

V6's `parseCreateArgs` returns `(createOpts, error)` -- the title is embedded in the `createOpts` struct. It normalizes IDs **eagerly during parsing**:

```go
// V6 — normalization done during parse
ids := strings.Split(args[i], ",")
for _, id := range ids {
    normalized := task.NormalizeID(strings.TrimSpace(id))
    if normalized != "" {
        opts.blockedBy = append(opts.blockedBy, normalized)
    }
}
```

V6 also consolidates all reference validation into a single `validateRefs` function rather than using separate `task.ValidateBlockedBy`/`task.ValidateParent` calls.

**V5** builds the new task using `task.NewTask(id, title)` (which sets timestamps and defaults internally), then overrides fields:

```go
// V5 — uses task.NewTask constructor
newTask := task.NewTask(id, title)
newTask.Priority = opts.priority
newTask.Description = opts.description
if len(blockedBy) > 0 {
    newTask.BlockedBy = blockedBy
}
newTask.Parent = parent
```

**V6** constructs the task struct directly as a literal:

```go
// V6 — direct struct literal
newTask := task.Task{
    ID:          id,
    Title:       trimmedTitle,
    Status:      task.StatusOpen,
    Priority:    opts.priority,
    Description: opts.description,
    BlockedBy:   opts.blockedBy,
    Parent:      opts.parent,
    Created:     now,
    Updated:     now,
}
```

The V6 approach differs because its `task.NewTask` has a different signature: `NewTask(title string, exists func(id string) bool) (*Task, error)`, which bundles ID generation and validation. Since `RunCreate` generates the ID separately inside the mutation closure, it cannot use `NewTask` and must build the struct directly.

**Title validation** differs between versions:

```go
// V5 — ValidateTitle returns (string, error), trims internally
title, err = task.ValidateTitle(title)
```

```go
// V6 — separate TrimTitle + ValidateTitle(title) returning only error
trimmedTitle := task.TrimTitle(opts.title)
if err := task.ValidateTitle(trimmedTitle); err != nil {
    return err
}
```

**--blocks handling** differs slightly:

```go
// V5 — uses index lookup via map[string]int
existing := make(map[string]int, len(tasks))
for i, t := range tasks { existing[t.ID] = i }
...
for _, blockID := range blocks {
    idx := existing[blockID]
    tasks[idx].BlockedBy = append(tasks[idx].BlockedBy, id)
    tasks[idx].Updated = now
}
```

```go
// V6 — linear scan with nested loop
if len(opts.blocks) > 0 {
    for i := range tasks {
        for _, blockID := range opts.blocks {
            if tasks[i].ID == blockID {
                tasks[i].BlockedBy = append(tasks[i].BlockedBy, id)
                tasks[i].Updated = now
            }
        }
    }
}
```

V5's approach is O(b) where b is the number of `--blocks` IDs (since it uses the pre-built index map). V6's approach is O(n * b) where n is the total task count. For typical usage (small task lists) this is negligible, but V5's approach is more algorithmically efficient.

### Code Quality

**Go idioms:**

- Both versions follow standard Go patterns: early returns, error wrapping, deferred Close.
- V5 uses unexported `runCreate` (method-on-nothing pattern via command map). V6 uses exported `RunCreate` (more testable independently, though the test still goes through `App.Run`).
- V5's `Context` struct is a common Go CLI pattern. V6's `App` struct with injected `Getwd` is also idiomatic and more testable.

**Naming:**

- V5: `runCreate`, `parseCreateArgs`, `validateIDsExist`, `splitCSV`, `normalizeIDs`, `printTaskDetails` -- all clear.
- V6: `RunCreate`, `parseCreateArgs`, `validateRefs`, `printTaskDetails` -- also clear. `validateRefs` is more concise than V5's `validateIDsExist`.

**Error handling:**

Both handle all errors explicitly. Key difference:

```go
// V5 — unknown flags are errors
case strings.HasPrefix(arg, "-"):
    return "", opts, fmt.Errorf("unknown flag '%s'", arg)
```

```go
// V6 — unknown flags are silently skipped
case strings.HasPrefix(arg, "-"):
    // Unknown flag — skip (global flags already extracted)
```

V5's strict approach is safer -- it will catch typos like `--prioirty`. V6 silently ignores them, which could lead to confusing behavior where a misspelled flag is silently discarded. However, V6's design assumes global flags have already been stripped by `parseArgs` in `app.go`, so this is intentional for flags like `--verbose` that might leak through.

**Error message for missing title:**

```go
// V5
fmt.Errorf("Title is required. Usage: tick create \"<title>\" [options]")
```

```go
// V6
fmt.Errorf("title is required. Usage: tick create \"<title>\" [options]")
```

V5 capitalizes "Title" which matches the spec verbatim: `Error: Title is required. Usage: tick create "<title>" [options]`. V6 uses lowercase "title" which is more conventional for Go error messages (per `go vet` / Go conventions that error strings should not be capitalized).

**DRY:**

- V5 has dedicated `splitCSV` and `normalizeIDs` helper functions, keeping parsing and normalization separate. The normalization happens in the mutation closure.
- V6 inlines the split-and-normalize logic directly in `parseCreateArgs`, which is slightly less DRY if the pattern were reused elsewhere. V6 consolidates reference validation into a single `validateRefs` function.

**Type safety:**

- V5's `existing` map is `map[string]int` (ID -> index), serving dual purpose: existence check and index lookup for `--blocks` modification.
- V6 uses `map[string]bool` for existence checking and does a separate linear scan for `--blocks`. This is slightly less efficient but simpler.

**Unknown-argument handling:**

```go
// V5 — rejects extra positional args
default:
    if !titleFound {
        title = arg
        titleFound = true
    } else {
        return "", opts, fmt.Errorf("unexpected argument '%s'", arg)
    }
```

```go
// V6 — silently ignores extra positional args
default:
    if opts.title == "" {
        opts.title = arg
    }
```

V5 rejects extra positional arguments with a clear error. V6 silently ignores them. V5's approach is strictly better here -- it prevents user confusion from silently dropped arguments.

### Test Quality

Both versions have one top-level test function `TestCreate` with 21 named subtests plus 3 table-driven sub-subtests for priority validation (24 `t.Run` calls total).

**Complete list of subtests (identical in both):**

1. `it creates a task with only a title (defaults applied)`
2. `it creates a task with all optional fields specified`
3. `it generates a unique ID for the created task`
4. `it sets status to open on creation`
5. `it sets default priority to 2 when not specified`
6. `it sets priority from --priority flag`
7. `it rejects priority outside 0-4 range`
   - `negative` (-1)
   - `too high` / `above max` (5)
   - `way too high` / `way above` (99 / 100)
8. `it sets description from --description flag`
9. `it sets blocked_by from --blocked-by flag (single ID)`
10. `it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)`
11. `it updates target tasks' blocked_by when --blocks is used`
12. `it sets parent from --parent flag`
13. `it errors when title is missing`
14. `it errors when title is empty string`
15. `it errors when title is whitespace only`
16. `it errors when --blocked-by references non-existent task`
17. `it errors when --blocks references non-existent task`
18. `it errors when --parent references non-existent task`
19. `it persists the task to tasks.jsonl via atomic write`
20. `it outputs full task details on success`
21. `it outputs only task ID with --quiet flag`
22. `it normalizes input IDs to lowercase`
23. `it trims whitespace from title`

All 22 tests from the spec (plus the priority subtests) are covered in both versions.

**Test helper differences:**

V5 uses `initTickProject` which calls `Run([]string{"tick", "init"}, ...)` to create a project via the actual init command. V6 uses `setupTickProject` which directly creates the `.tick/` directory and `tasks.jsonl` file:

```go
// V5 — integration-style setup
func initTickProject(t *testing.T) string {
    dir := t.TempDir()
    var stdout, stderr bytes.Buffer
    code := Run([]string{"tick", "init"}, dir, &stdout, &stderr, false)
    ...
}
```

```go
// V6 — unit-style setup
func setupTickProject(t *testing.T) (string, string) {
    dir := t.TempDir()
    tickDir := filepath.Join(dir, ".tick")
    os.Mkdir(tickDir, 0755)
    os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte{}, 0644)
    ...
}
```

V5's approach is more integration-like (tests the init flow too), but couples create tests to init correctness. V6's approach is more isolated (pure unit test setup). V6 also returns `(dir, tickDir)` tuple, while V5 returns only `dir`.

V5 reads tasks via manual JSON parsing:
```go
// V5
func readTasksFromFile(t *testing.T, dir string) []task.Task {
    data, _ := os.ReadFile(filepath.Join(dir, ".tick", "tasks.jsonl"))
    for _, line := range strings.Split(string(data), "\n") {
        json.Unmarshal([]byte(line), &tk)
        ...
    }
}
```

V6 uses the storage package's `ReadJSONL`:
```go
// V6
func readPersistedTasks(t *testing.T, tickDir string) []task.Task {
    tasks, err := storage.ReadJSONL(jsonlPath)
    ...
}
```

V6's approach reuses production code for reading, which is cleaner and ensures consistency. V5's manual parsing is more isolated from production code but duplicates logic.

**V6 test runner helper:**

V6 has a dedicated `runCreate` helper that constructs an `App` directly, making tests more focused:

```go
// V6
func runCreate(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
    app := &App{
        Stdout: &stdoutBuf, Stderr: &stderrBuf,
        Getwd: func() (string, error) { return dir, nil },
    }
    fullArgs := append([]string{"tick", "create"}, args...)
    code := app.Run(fullArgs)
    ...
}
```

V5 calls the full `Run` function directly, requiring full args including the program name and subcommand.

**Assertion quality differences:**

V6's "defaults" test checks more fields:
```go
// V6 — also checks these defaults are empty
if tk.Description != "" { t.Errorf(...) }
if len(tk.BlockedBy) != 0 { t.Errorf(...) }
if tk.Parent != "" { t.Errorf(...) }
```

V5's defaults test only checks title, status, and priority -- it does not verify that description, blocked_by, and parent are empty.

V6's "generates unique ID" test creates TWO tasks and checks uniqueness:
```go
// V6 — creates 2 tasks, checks IDs differ
runCreate(t, dir, "Task one")
runCreate(t, dir, "Task two")
tasks := readPersistedTasks(t, tickDir)
if tasks[0].ID == tasks[1].ID { t.Errorf(...) }
```

V5 only creates one task and checks it against a regex.

V6's "--blocks" test also verifies the timestamp was refreshed:
```go
// V6
if !target.Updated.After(now.Add(-time.Second)) {
    t.Error("target's updated timestamp should be refreshed")
}
```

V5 does not verify the timestamp update on the blocked target.

V6's persistence test does deeper validation (raw JSON parsing):
```go
// V6
var raw map[string]interface{}
json.Unmarshal([]byte(lines), &raw)
if raw["title"] != "Persisted task" { ... }
```

V5 just checks `strings.Contains(data, "Persisted task")`.

V6's "outputs full task details" test checks for status "open" in the output; V5 does not.

V6's "--quiet" test does an exact string match:
```go
// V6
expected := tk.ID + "\n"
if stdout != expected { ... }
```

V5 uses a regex match after trimming:
```go
// V5
output := strings.TrimSpace(stdout.String())
pattern := regexp.MustCompile(`^tick-[0-9a-f]{6}$`)
if !pattern.MatchString(output) { ... }
```

V6's exact match is stricter (verifies no extra output), though V5's regex validates the ID format more rigorously.

V6's error tests for non-existent refs do NOT verify that no task was created. V5 does:
```go
// V5 — verifies no partial mutation
tasks := readTasksFromFile(t, dir)
if len(tasks) != 0 {
    t.Errorf("expected 0 tasks (no partial mutation), got %d", len(tasks))
}
```

V6 omits this check, which means V6 doesn't test that the mutation was fully rolled back.

**Existing task setup differences:**

V5 uses `task.NewTask(id, title)` to construct test fixtures. V6 constructs `task.Task{...}` literals directly with explicit timestamps:
```go
// V6
now := time.Now().UTC().Truncate(time.Second)
existingTask := task.Task{
    ID: "tick-aaa111", Title: "Blocker", Status: task.StatusOpen,
    Priority: 2, Created: now, Updated: now,
}
```

V6's approach is more explicit and doesn't depend on `NewTask`'s implementation details. V5's is more concise.

### Skill Compliance

| Constraint | V5 | V6 |
|-----------|----|----|
| Handle all errors explicitly (no naked returns) | PASS | PASS |
| Document all exported functions/types/packages | PASS (all funcs unexported except helpers in task pkg) | PASS (`RunCreate` and `validateRefs` documented) |
| Write table-driven tests with subtests | PASS (priority validation uses table) | PASS (priority validation uses table) |
| Propagate errors with `fmt.Errorf("%w", err)` | PARTIAL -- uses `%w` in engine calls but create errors are plain `fmt.Errorf` | PARTIAL -- same pattern, create-level errors are plain strings |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic for normal error handling | PASS | PASS |
| MUST NOT hardcode configuration | PASS (priority default from `task.DefaultPriority`) | PASS (priority default hardcoded as `2` in `parseCreateArgs`) |
| Use `gofmt` compatible formatting | PASS | PASS |
| Run race detector on tests | Not verified at commit level | Not verified at commit level |

Note on hardcoded configuration: V5 references `task.DefaultPriority` constant for the default priority, while V6 hardcodes `2` directly in `parseCreateArgs`:

```go
// V5
opts := createOpts{priority: task.DefaultPriority}
```

```go
// V6
opts := createOpts{priority: 2}
```

V5 is slightly better here -- if the default priority constant changes, V5 updates automatically.

### Spec-vs-Convention Conflicts

1. **Error message capitalization**: The spec says `Error: Title is required. Usage: ...`. V5 outputs `Error: Title is required. Usage: tick create "<title>" [options]` (matching spec). V6 outputs `Error: title is required. Usage: tick create "<title>" [options]` (following Go convention that error strings should not be capitalized). Since the CLI wraps errors with `Error: ` prefix, V6's lowercase `title` is technically more Go-idiomatic but deviates from the spec text.

2. **Unknown flag handling**: The spec does not explicitly address behavior for unknown flags. V5 rejects them with an error. V6 silently skips them. Neither approach conflicts with the spec, but V5's strictness is more user-friendly.

3. **Error quote style**: V5 uses single quotes in error messages (`task 'tick-xxx' not found`), V6 uses Go `%q` formatting (`task "tick-xxx" not found`). Minor cosmetic difference.

## Diff Stats

| Metric | V5 | V6 |
|--------|----|----|
| Files changed | 3 (`cli.go`, `create.go`, `create_test.go`) | 3 (`app.go`, `create.go`, `create_test.go`) |
| Lines added (total) | 785 | 787 |
| `create.go` lines | 264 | 242 |
| `create_test.go` lines | 519 | 534 |
| Dispatcher change | +2 lines in `cli.go` (map entry) | +11 lines in `app.go` (case + `handleCreate` method) |
| Helper functions in create.go | 6 (`parseCreateArgs`, `validateIDsExist`, `splitCSV`, `normalizeIDs`, `printTaskDetails`, `runCreate`) | 4 (`parseCreateArgs`, `validateRefs`, `printTaskDetails`, `RunCreate`) |
| Test helper functions | 3 (`initTickProject`, `initTickProjectWithTasks`, `readTasksFromFile`) | 4 (`setupTickProject`, `setupTickProjectWithTasks`, `readPersistedTasks`, `runCreate`) |
| Test subtests | 21 + 3 sub-subtests | 21 + 3 sub-subtests |
| Imports (create.go) | `engine`, `task` | `storage`, `task` |
| Imports (create_test.go) | `task`, stdlib only | `storage`, `task`, `time` |

## Verdict

Both implementations are solid and fully satisfy all acceptance criteria. The differences are largely stylistic and architectural rather than functional.

**V5 strengths:**
- Stricter argument validation (rejects unknown flags and extra positional args)
- Uses `task.DefaultPriority` constant instead of hardcoded `2`
- More efficient `--blocks` handling via pre-built index map (O(b) vs O(n*b))
- Error tests verify no partial mutation occurred
- Error message capitalization matches spec verbatim
- Uses existing `task.ValidateBlockedBy` / `task.ValidateParent` for self-reference checks (better reuse)
- Fewer helper functions (6 vs 4 in create.go) -- achieved through better factoring (`splitCSV`, `normalizeIDs` as separate utilities)

**V6 strengths:**
- More thorough default-value assertions in tests (checks description, blocked_by, parent are empty)
- Unique ID test creates two tasks and verifies they differ
- `--blocks` test verifies timestamp refresh on target task
- Persistence test does structured JSON validation (not just string search)
- Dedicated `runCreate` test helper reduces boilerplate
- Uses `storage.ReadJSONL` and `storage.MarshalJSONL` in tests (production code reuse)
- Test setup is more isolated (doesn't depend on `tick init`)
- `validateRefs` consolidates all reference checks into one function (easier to maintain)
- Exported `RunCreate` is independently callable/testable

**Winner: Slight edge to V5.** V5's stricter input validation (rejecting unknown flags and extra positional arguments) is a meaningful correctness advantage that prevents user confusion. V5's no-partial-mutation assertions in error tests also provide better safety verification. V6 has better test coverage for edge cases (timestamp refresh, default field assertions, unique ID deduplication) but V5's architectural choices around strictness and constant usage are more important for production reliability. The differences are minor overall -- both are production-quality implementations.
