# Phase 2: Task Lifecycle

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| tick-core-2-1 (Transition validation) | V2 | Narrow | V2 tests all 9 invalid transitions for no-mutation; V4 tests only 1 |
| tick-core-2-2 (CLI transition commands) | V4 | Moderate | Type-safe test infrastructure, exit code verification, stderr testing |
| tick-core-2-3 (Update command) | V2 | Narrow | Validate-then-apply atomicity, shared error unwrapping, broader --blocks coverage |

## Cross-Task Architecture Analysis

### The Mutate Callback Pattern -- Central to All Phase 2 Work

Every Phase 2 command (transition, update) follows the same 5-step skeleton:

1. Parse args and normalize ID
2. Discover tick dir, open store
3. Call `store.Mutate(func(tasks []task.Task) ([]task.Task, error) { ... })`
4. Handle error from Mutate
5. Format and output

Both V2 and V4 replicate this skeleton in `transition.go` and `update.go`. Neither extracts the duplicated "find task by index" loop into a shared helper, despite it appearing identically 4 times across the two implementations (2 in each version):

```go
// V2: transition.go:34 and update.go:139 -- identical pattern
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
```

```go
// V4: transition.go:34 and update.go:35 -- identical pattern
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
```

This is a missed DRY opportunity in both codebases. A `findTaskIndex(tasks []task.Task, id string) (int, error)` helper would eliminate this duplication. Looking ahead at the `dep.go` files (Phase 3), the same pattern appears again, confirming this is a systemic cross-task duplication.

### The Error Unwrapping Divergence -- A Phase-Level Architecture Split

The most architecturally significant cross-task pattern is how the two implementations handle errors from `store.Mutate`. This stems from a foundational difference in their storage layers:

**V2's storage wraps callback errors:**
```go
// V2 internal/storage/store.go:130-131
modified, err := fn(tasks)
if err != nil {
    return fmt.Errorf("mutation failed: %w", err)
}
```

**V4's storage passes callback errors through:**
```go
// V4 internal/store/store.go:120-123
modified, err := fn(tasks)
if err != nil {
    return err
}
```

This creates a cascading difference across ALL Phase 2 commands:

**V2** must unwrap errors everywhere. It first did this with string prefix stripping in tick-core-2-2 (`transition.go`), then evolved it into a shared helper in tick-core-2-3 (`app.go`), and retroactively applied it to transition.go and create.go:

```go
// V2 app.go:174-179 -- shared helper used by transition, create, update, dep
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```

Usage across commands:
- `transition.go:57`: `return unwrapMutationError(err)`
- `update.go:232`: `return unwrapMutationError(err)`
- `create.go:220`: `return unwrapMutationError(err)`
- `dep.go:91,160`: `return unwrapMutationError(err)` (Phase 3)

**V4** does nothing -- it just returns `err` directly from Mutate. This is simpler per-command, but means errors like `"Task 'tick-xxx' not found"` never get the `"mutation failed: "` prefix in the first place (because V4's storage layer does not wrap them). V4's cleaner approach is a consequence of better storage-layer design rather than better CLI-layer code.

This is a phase-level insight the task reports touch on individually but do not synthesize: **V4's simpler error handling in Phase 2 is a dividend of a better Phase 1 storage design**. V2's `unwrapMutationError` is a band-aid for a storage-layer flaw that V4 avoids entirely. However, V2 deserves credit for recognizing the pattern and centralizing the fix across all commands.

### Output Path Divergence: SQLite Query vs Direct Struct Conversion

After mutation, the update command must output full task details. The two versions take fundamentally different paths:

**V2** queries SQLite through the storage engine:
```go
// V2 update.go:242-249
data, err := queryShowData(store, updatedTask.ID)
if err != nil {
    // Fallback: if query fails, just print the ID
    fmt.Fprintln(a.stdout, updatedTask.ID)
    return nil
}
return a.formatter.FormatTaskDetail(a.stdout, data)
```

**V4** converts the in-memory task struct directly:
```go
// V4 update.go:140-141
detail := taskToDetail(updatedTask)
return a.Formatter.FormatTaskDetail(a.Stdout, detail)
```

V2's approach is more correct for showing relational data (blocked_by task titles, children) but is unnecessarily heavy for a create/update flow where the task was just modified in memory. V4's `taskToDetail` helper (defined in `create.go:214`) is reused by both `create` and `update` commands, showing better cross-task reuse. V2's `queryShowData` requires the store to still be open and does a full SQLite round-trip, while V4 avoids the database entirely. The tradeoff: V4's output after update will not include related task names (just IDs), while V2's will.

### Formatter Interface Type Safety

A subtle cross-task architectural difference visible only by examining the Formatter interface:

```go
// V2 formatter.go:36 -- uses task.Status type
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error

// V4 format.go:64 -- uses plain string
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
```

V2's Formatter interface uses the domain type `task.Status` for transition formatting, creating a compile-time type-safety boundary between the CLI layer and the task domain. V4 uses `string`, which requires the caller to cast: `string(result.OldStatus)` (see `transition.go:62`). This is a design decision with cross-phase implications: V2's approach means formatter implementations can switch on `task.Status` constants, while V4's formatters work with raw strings.

For `FormatTaskDetail`, the pattern reverses: V2 uses `*showData` (a CLI-layer struct populated from SQLite), while V4 uses `TaskDetail` (a CLI-layer struct converted from `task.Task`). V4's is more decoupled from the storage layer.

### ID Normalization Strategy

A cross-command pattern: V2 normalizes IDs on both sides of every comparison, V4 normalizes only the input and assumes stored IDs are already lowercase.

V2 inside Mutate callbacks (transition.go, update.go, dep.go):
```go
if task.NormalizeID(tasks[i].ID) == id {  // normalizes stored ID too
```

V4 inside Mutate callbacks (transition.go, update.go, dep.go):
```go
if tasks[i].ID == id {  // assumes stored IDs already normalized
```

V2's approach is defensive -- it handles the case where a task was stored with mixed-case IDs. V4's approach is more efficient but relies on an invariant (IDs are always lowercase in storage) that is never explicitly documented or enforced at the storage layer. Both use `task.NormalizeID` on input; they differ only on whether to also normalize the stored side. V2 is safer. V4 is cleaner if the invariant holds.

### Shared Test Helpers -- Divergent Philosophies

V2 test helpers (`create_test.go`):
```go
func setupTickDirWithContent(t *testing.T, content string) string { ... }  // raw JSONL string
func readTasksJSONL(t *testing.T, dir string) []map[string]interface{} { ... }  // untyped
func readTaskByID(t *testing.T, dir, id string) map[string]interface{} { ... }  // untyped
```

V4 test helpers (`create_test.go`):
```go
func setupInitializedDirWithTasks(t *testing.T, tasks []task.Task) string { ... }  // typed
func readTasksFromDir(t *testing.T, dir string) []task.Task { ... }  // typed
```

V4's helpers use `task.WriteJSONL`/`task.ReadJSONL` for setup and verification, ensuring tests go through the same serialization path as production code. V2's helpers write raw JSONL strings and parse into `map[string]interface{}`. This means:

- V4 tests catch serialization bugs (if `WriteJSONL` has a bug, both production and test break)
- V2 tests can verify the exact JSONL format (raw string comparison), catching serialization drift
- V4 assertions are compile-time safe: `tasks[0].Status != task.StatusInProgress`
- V2 assertions require runtime casts: `int(tk["priority"].(float64)) != 0`

Both sets of helpers are reused across all Phase 2 commands (transition_test.go, update_test.go), showing cross-task reuse. V4's approach is systematically better for maintainability.

## Code Quality Patterns

### Naming Consistency

**V2**: App uses unexported fields (`a.config.Quiet`, `a.stdout`, `a.workDir`), consistent nested config struct. Parameter names are descriptive (`task`, `command`, `currentStatus`).

**V4**: App uses exported fields (`a.Quiet`, `a.Stdout`, `a.Dir`), flat struct. Parameter names are short-idiomatic (`t`, `r`, `s`). The short `t` in the task package risks confusion with `*testing.T` in test files.

V2's encapsulation is better Go practice (unexported fields force construction through `NewApp()`). V4's exported fields allow direct struct literal construction in tests (`&App{Stdout: &stdout, Dir: dir}`), which is more convenient for testing but exposes internals.

### Error Message Consistency

V2 error messages across Phase 2:
- `"Task ID is required. Usage: tick %s <id>"` (transition, update)
- `"Task '%s' not found"` (transition, update)
- `"No update flags provided. Use at least one of: ..."` (update)
- `"unknown flag %q for update command"` (update)

V4 error messages across Phase 2:
- `"Task ID is required. Usage: tick %s <id>"` (transition, update)
- `"Task '%s' not found"` (transition, update)
- `"No flags provided. At least one flag is required.\n\n..."` (update)
- `"unexpected argument '%s'"` (update)

Both are internally consistent. V4's no-flags error includes formatted usage help (multi-line), which is more helpful for users. V2's error quote style uses `%q` for unknown flags (includes Go-style quotes), V4 uses `'%s'` (single quotes). V4 is more user-friendly.

### DRY Across Phase 2

**V2 shared utilities introduced/used in Phase 2:**
- `unwrapMutationError()` -- used by transition, update, create, dep (4 callers)
- `task.ValidateParent()` -- used by update and create
- `task.ValidateTitle()` -- used by update and create
- `task.NormalizeID()` -- used everywhere

**V4 shared utilities used in Phase 2:**
- `parseCommaSeparatedIDs()` -- used by update and create
- `taskToDetail()` -- used by update and create
- `task.ValidateTitle()` -- used by update and create
- `task.NormalizeID()` -- used everywhere
- `statusIn()` -- used by transition (single use, but extracted for clarity)

V2 has the more impactful shared helper (`unwrapMutationError` saves duplicated error-unwrapping logic across 4+ callers). V4's `parseCommaSeparatedIDs` and `taskToDetail` are smaller but well-scoped reusable pieces. Neither version extracts the "find task by index" pattern that appears in every Mutate callback.

### Pointer vs Value Optionality

V2 `updateFlags` uses boolean sentinels:
```go
type updateFlags struct {
    title               string
    titleProvided       bool      // sentinel
    description         string
    descriptionProvided bool      // sentinel
    priority            *int      // pointer (inconsistent)
    parent              string
    parentProvided      bool      // sentinel
    blocks              []string
}
```

V4 `updateFlags` uses pointer-based optionality uniformly:
```go
type updateFlags struct {
    title       *string
    description *string
    priority    *int
    parent      *string
    blocks      []string
}
```

V4's approach is more idiomatic Go. V2 mixes two patterns (boolean sentinels for strings, pointer for int), creating an inconsistency. V4's nil-check pattern is impossible to desync (you cannot have a value without it being non-nil), while V2's pattern allows a bug where `title` is set but `titleProvided` is false.

## Test Coverage Analysis

### Aggregate Counts

| Metric | V2 | V4 |
|--------|-----|-----|
| Task-level test LOC (transition logic) | ~390 lines in task_test.go | 226 lines in transition_test.go |
| CLI transition test LOC | 423 lines | 446 lines |
| CLI update test LOC | 609 lines | 790 lines |
| Total Phase 2 test LOC | ~1,422 (Phase 2 portions) | 1,462 |
| Leaf test cases (task-level) | 36 | 27 |
| Leaf test cases (CLI transition) | ~22 | ~21 |
| Leaf test cases (CLI update) | ~26 | ~24 |
| Total leaf tests | ~84 | ~72 |

Note: V2's task_test.go (933 lines) includes Phase 1 tests; only ~390 lines are Phase 2 transition tests. V4 has a separate transition_test.go (226 lines) containing only Phase 2 content.

### Edge Case Coverage Matrix

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| No-mutation on all 9 invalid transitions (task level) | YES | NO (1 of 9) |
| UTC timezone explicit assertion | YES | NO |
| Exit code tested directly | NO (inferred) | YES |
| Stderr content verified | NO | YES |
| Updated timestamp refresh (CLI level) | NO | YES |
| Created timestamp unchanged after update | NO | YES |
| --blocks duplicate skip | YES | NO |
| --blocks comma-separated multiple targets | YES | NO |
| --blocks combined with --title atomically | YES | NO |
| Invalid priority non-integer (e.g., "abc") | NO | YES |
| cache.db existence after update | NO | YES |
| "mutation failed:" prefix not leaking | YES (fixed) | NO (leaks in V2's storage, absent in V4's) |

### Testing Approach Differences

**V2** tests more edge cases in the domain layer (task package), ensuring the pure transition function is bulletproof. It then tests the CLI layer with lighter integration tests that focus on the wiring.

**V4** tests more edge cases at the CLI integration level, where it can verify exit codes, stderr content, and end-to-end persistence. Its domain-layer tests are more concise (table-driven) but less exhaustive.

Both approaches are valid. V2's bottom-up thoroughness catches domain-level regressions earlier. V4's top-down integration focus catches wiring issues (exit code routing, stderr formatting) that V2 misses entirely at the unit level.

## Phase Verdict

**V4 wins Phase 2 by a narrow margin.**

The decisive factors:

1. **Better storage-layer foundation**: V4's Mutate does not wrap callback errors, eliminating the need for `unwrapMutationError` across every command. This is not a Phase 2 decision per se, but it yields cleaner code across the entire phase. V2's `unwrapMutationError` is a well-engineered workaround for a flawed Phase 1 choice.

2. **Type-safe test infrastructure**: V4's `setupInitializedDirWithTasks([]task.Task)` and `readTasksFromDir() []task.Task` use production serialization paths and return typed values. This eliminates an entire class of test-specific bugs (wrong JSON field names, type assertion failures) that V2's `map[string]interface{}` approach is vulnerable to. This advantage compounds across every test file in the phase.

3. **Pointer-based flag optionality**: V4's uniform `*string`/`*int` approach for update flags is more idiomatic Go and structurally prevents sentinel-desync bugs. V2's mixed boolean-sentinel/pointer approach is inconsistent.

4. **CLI-level test completeness**: V4 directly tests exit codes, stderr content, and updated timestamp refresh at the integration level. V2 cannot test these due to its `Run() error` signature (V4 uses `Run() int`).

**Where V2 is genuinely better:**

1. **Domain-layer exhaustiveness**: V2 tests all 9 invalid transitions for no-mutation. V4 tests only 1. This is a real coverage gap.

2. **Cross-command DRY evolution**: V2 introduced `unwrapMutationError` during tick-core-2-3 and retroactively refactored tick-core-2-2 and tick-core-1-6 to use it. This shows codebase-level thinking -- improving existing code while adding new features.

3. **Validate-then-apply pattern**: V2's update command validates all flags before mutating any fields. V4 interleaves validation and mutation. While V4's Mutate callback rollback makes this functionally safe, V2's pattern is architecturally cleaner.

4. **Defensive ID normalization**: V2 normalizes both sides of ID comparisons inside Mutate callbacks. V4 relies on the unstated assumption that stored IDs are always lowercase.

The net assessment: V4's advantages are structural and compound across files (type safety, cleaner error flow, better test infrastructure). V2's advantages are point-specific (exhaustive domain tests, defensive normalization, retroactive refactoring). For a growing codebase, V4's structural choices will pay larger dividends over time.
