# Phase 2: Task Lifecycle

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 2-1: Transition validation | V2 | Large | Exhaustive test coverage (all 9 invalid non-mutation, all 7 updated timestamps), package-level map (no per-call alloc), correct spec error format |
| 2-2: CLI transition commands | V1 (impl) / V3 (tests) | Narrow | V1 is most concise (53 LOC, error-return pattern); V3 has strongest tests (direct exit code verification, round-trip persistence); V2 uniquely adds `unwrapMutationError` |
| 2-3: Update command | V2 | Large | 22 tests covering widest edge cases (dedup, atomicity, comma-separated blocks), V1 has a blocks-dedup bug, V2 adds `unwrapMutationError` retroactively to prior commands |

## Cross-Task Architecture Analysis

### The Domain-CLI Integration Pipeline

All three versions share the same fundamental three-layer flow for Phase 2:

```
CLI Router (dispatch) -> Command Handler (parse, validate, orchestrate) -> Domain Function (pure logic)
```

The critical cross-task pattern is how the **transition validation logic (2-1)** composes with the **CLI transition commands (2-2)** and the **update command (2-3)**. Each version makes a different architectural commitment that reverberates across all three tasks.

### Pattern 1: Error Return Type -- The Phase-Defining Divergence

The most significant cross-task pattern is the return type convention, which cascades from the router through every command handler.

**V1 and V2: `error` return, centralized error formatting**

The router converts errors to exit codes in one place:

```go
// V1 cli.go -- single error formatting point
if err != nil {
    fmt.Fprintf(a.stderr, "Error: %s\n", err)
    return 1
}
return 0
```

This means `cmdTransition`, `cmdUpdate`, and every other command return `error`. The router is the single point where errors become user-visible output. Both transition validation errors from task 2-1 and update validation errors from task 2-3 flow through the same exit path.

**V3: `int` return, distributed error formatting**

Every command handler writes its own errors:

```go
// V3 runTransition
if err != nil {
    fmt.Fprintf(a.Stderr, "Error: %s\n", err)
    return 1
}
// V3 runUpdate
if err != nil {
    fmt.Fprintf(a.Stderr, "Error: %s\n", err)
    return 1
}
```

This `fmt.Fprintf(a.Stderr, "Error: %s\n", err)` pattern appears **7 times** in V3's `runUpdate` and **5 times** in V3's `runTransition`. Every error site duplicates the same formatting. Across the entire phase, V3 has approximately 12+ instances of identical error formatting logic that V1/V2 handle once in the router.

### Pattern 2: The Mutate Callback -- Shared Orchestration Across 2-2 and 2-3

Both the transition commands (2-2) and update command (2-3) use `store.Mutate` with an identical callback pattern. This is the true compositional seam of Phase 2 -- the domain function from task 2-1 is invoked inside the Mutate callback from task 2-2, and the update mutations from task 2-3 follow the same structure.

**V1: Direct index iteration, in-place mutation**

```go
// 2-2 transition
for i := range tasks {
    if tasks[i].ID == taskID {
        r, err := task.Transition(&tasks[i], command)  // 2-1 domain call
        ...
    }
}

// 2-3 update -- uses map for O(1) lookup
taskMap := make(map[string]*task.Task, len(tasks))
for i := range tasks {
    taskMap[tasks[i].ID] = &tasks[i]
}
t, ok := taskMap[taskID]
```

V1 uses two different lookup strategies: linear scan in 2-2, map-based in 2-3. The map approach in 2-3 is necessary because update needs to look up parent and blocks targets too, but the inconsistency between the two commands reveals that V1 did not plan the lookup pattern ahead.

**V2: Double-normalization defense, consistent pattern**

```go
// 2-2 transition
if task.NormalizeID(tasks[i].ID) == id {
    idx = i
    break
}

// 2-3 update
if task.NormalizeID(tasks[i].ID) == flags.id {
    idx = i
    break
}
```

V2 consistently uses `task.NormalizeID()` on **stored** IDs during comparison across both commands. V1 and V3 normalize only the input but compare directly against stored IDs (`tasks[i].ID == taskID`). This means V2 is the only version that would correctly handle a JSONL file containing mixed-case IDs -- a defensive pattern that is consistent across tasks.

**V3: Pointer-based task capture**

```go
// 2-2 transition
var targetTask *task.Task
for i := range tasks {
    if tasks[i].ID == taskID {
        targetTask = &tasks[i]
        break
    }
}

// 2-3 update
var targetIdx int = -1
for i := range tasks {
    if tasks[i].ID == taskID {
        targetIdx = i
        break
    }
}
```

V3 uses two different approaches even between 2-2 and 2-3: pointer capture in transition, index capture in update. This inconsistency means a developer reading the codebase encounters two different patterns for the same conceptual operation.

### Pattern 3: Timestamp Type -- A Phase 1 Decision That Echoes Through Phase 2

The task model definition (from Phase 1) forces a timestamp handling approach in every Phase 2 function:

**V1/V2: `time.Time` / `*time.Time`**
```go
// Both transition (2-1) and update (2-3) share this pattern
now := time.Now().UTC().Truncate(time.Second)
t.Updated = now
t.Closed = &now   // pointer for nullable
t.Closed = nil     // clear on reopen
```

**V3: `string`**
```go
// Both transition (2-1) and update (2-3) share this pattern
now := time.Now().UTC().Format(time.RFC3339)
task.Updated = now
task.Closed = now    // string for nullable via omitempty
task.Closed = ""     // clear on reopen
```

This is not just a task-2-1 decision -- it propagates into 2-3 where `t.Updated = now` is used in update. The string approach requires RFC3339 parsing in tests, adding `time.Parse(time.RFC3339, task.Updated)` calls in 2-1 and 2-3 test code. V1/V2 tests can compare `time.Time` values directly. This is a **Phase 1 architectural decision** that compounds across all three Phase 2 tasks.

### Pattern 4: Cross-Task Refactoring -- V2's `unwrapMutationError`

V2 is the only version that, during Phase 2, recognized a cross-cutting concern and addressed it retroactively. When implementing task 2-3 (update), V2 added `unwrapMutationError` to `app.go` as a shared helper:

```go
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```

Then V2 **went back** and applied this to task 2-2's `transition.go` and even to `create.go` from Phase 1. The diff stats confirm this: V2's Phase 2 touched **9 files** including `app.go` (+16), `create.go` (+16/-3), `create_test.go` (+38), and `transition.go` (+1/-7).

V1 and V3 both leave the `"mutation failed: "` prefix leaking to users. This is only visible at the phase level -- task-level analysis sees V2's `unwrapMutationError` usage but misses that it was retroactively applied to prior code.

### Pattern 5: Argument Parsing -- Pre-Stripped vs Full Args

The router's argument handling creates a phase-wide consistency pattern:

**V1/V2: Pre-stripped arguments** -- the router strips the subcommand and passes only remaining args. Both transition (2-2) and update (2-3) receive `args[0]` as the task ID.

**V3: Full args array** -- the router passes the entire `[]string{"tick", "start", "tick-abc123"}`. Commands must know their positional index: `args[2]` in transition (2-2), `args[2]` in update (2-3). This coupling is consistent but fragile -- if any global flag processing changes the indices, every command breaks.

The consequence: V3's `parseUpdateArgs` starts at index 2, skipping "tick" and "update". V2's `parseUpdateArgs` starts at index 0, receiving only post-subcommand args. V1 doesn't extract a separate parse function at all. This affects how the flag parser from 2-3 could be reused or tested independently.

### Pattern 6: Formatter Integration

The transition output formatting reveals how the formatter abstraction (from later phases but already present in the final code) connects to Phase 2:

```go
// V1: passes a struct through the formatter
a.fmtr.FormatTransition(a.stdout, TransitionData{...})

// V2: passes arguments directly
a.formatter.FormatTransition(a.stdout, id, oldStatus, newStatus)

// V3: calls formatter, writes result manually
formatter := a.formatConfig.Formatter()
fmt.Fprint(a.Stdout, formatter.FormatTransition(...))
```

V1 and V2 let the formatter write directly to the writer (side-effect inside the formatter). V3 gets a string back and writes it explicitly. This means V3's formatter is a pure function (returns string), while V1/V2's formatters are impure (write to io.Writer). V3's approach is more testable in isolation but means every call site must handle the `fmt.Fprint` boilerplate.

## Code Quality Patterns

### Error Handling Consistency

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| Error return type | `error` across all tasks | `error` across all tasks | `int` across all tasks |
| Error formatting | Centralized in router | Centralized in router + unwrap | Distributed in every handler |
| Mutation error handling | Pass through raw | `unwrapMutationError()` shared helper | Pass through raw |
| Unknown command (2-1) | Not handled | `"unknown command: %s"` | `"unknown command: %s"` |
| Error message case | Spec-compliant "Cannot" | Spec-compliant "Cannot" | Non-compliant "cannot" |

V2 is the most consistent. It handles errors the same way everywhere, and when it discovered that `store.Mutate` wraps errors, it fixed the problem once and applied it everywhere.

V3 has the most repetitive error handling. The `fmt.Fprintf(a.Stderr, "Error: %s\n", err)` + `return 1` pattern appears repeatedly in both transition and update handlers with no abstraction.

V1 is clean but incomplete -- it doesn't handle the mutation error wrapping at all.

### Naming Consistency

| Element | V1 | V2 | V3 |
|---------|-----|-----|-----|
| Command handler | `cmdTransition`, `cmdUpdate` | `runTransition`, `runUpdate` | `runTransition`, `runUpdate` |
| Task parameter | `t` (2-1) | `task` (2-1) | `task` (2-1) |
| ID variable | `taskID` (2-2, 2-3) | `id` (2-2), `flags.id` (2-3) | `taskID` (2-2, 2-3) |
| Result capture | `result` (TransitionResult) | `oldStatus, newStatus` (bare) | `transitionResult` (TransitionResult) |
| Flags struct | raw pointers (2-3) | `updateFlags` unexported (2-3) | `UpdateFlags` exported (2-3) |

V1 uses `cmd` prefix for all handlers (consistent with `cmdInit`, `cmdCreate`). V2 and V3 use `run` prefix. All three are internally consistent.

V3's exported `UpdateFlags` struct is over-scoped -- it is only used within the `cli` package but has public visibility.

### DRY Across Tasks

**V1** is the most concise per task but does not share abstractions across tasks. Each command is self-contained. The transition handler is 53 LOC, the update handler is 175 LOC.

**V2** actively identifies shared patterns and extracts them. The `unwrapMutationError` helper is used by transition (2-2), update (2-3), and was retrofitted to create (Phase 1). The `parseUpdateArgs` / `hasAnyFlag()` pattern is the cleanest separation of parsing from execution. Total Phase 2 code is larger (1851 lines) but the per-command complexity is lower.

**V3** has the most total code (1962 lines) with the least sharing. The `normalizeIDs` helper in `create.go` is reused by `update.go`, which is the only cross-command helper. The verbose logging calls (`a.WriteVerbose(...)`) appear in both transition and update but are not abstracted further.

### Blocks Deduplication Bug

V1 has a bug that only becomes visible when examining tasks 2-2 and 2-3 together: the `--blocks` flag in the update command appends to `blocked_by` without checking for duplicates.

```go
// V1 update.go -- no dedup check
for _, blockID := range blocksFlag {
    target.BlockedBy = append(target.BlockedBy, taskID)
    target.Updated = now
}
```

V2 and V3 both check for duplicates, but with different robustness:

```go
// V2 -- uses NormalizeID for case-insensitive dedup
if task.NormalizeID(existingID) == task.NormalizeID(sourceID) {

// V3 -- direct string comparison
if existing == taskID {
```

V2's approach is strictly more correct since it handles the edge case where stored IDs might have inconsistent casing.

## Test Coverage Analysis

### Aggregate Test Counts

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| **2-1 (transition logic)** | | | |
| Test functions | 7 | 21 | 21 |
| Test LOC | 192 | 390 | 413 |
| Custom helpers | 3 (containsAll, contains, searchString) | 1 (newTestTask) | 2 (makeTask, makeClosedTask) |
| **2-2 (CLI transitions)** | | | |
| Test functions | 17 | 19 | 18 |
| Test LOC | 244 | 423 | 499 |
| **2-3 (update command)** | | | |
| Test functions | 17 | 22 | 20 |
| Test LOC | 308 | 609 | 643 |
| **Phase total** | | | |
| Test functions | 41 | 62 | 59 |
| Test LOC | 744 | 1422 | 1555 |
| Impl LOC | 293 | 368 | 398 |
| Test:impl ratio | 2.5:1 | 3.9:1 | 3.9:1 |

### Cross-Task Test Patterns

**V1: Integration-style CLI tests, table-driven domain tests**

V1's domain tests (2-1) use table-driven subtests with shared data structures. V1's CLI tests (2-2, 2-3) use full-pipeline integration: `initTickDir` -> `createTask` -> `runCmd`. This means every CLI test exercises the entire stack including `tick init` and `tick create`, which is realistic but slow and couples transition tests to create behavior.

The `runCmd` helper invokes the full `App.Run()` pipeline. The `createTask` helper runs `tick create` to set up test data. This means V1's transition tests implicitly test that create works correctly too.

**V2: Unit-style CLI tests with raw JSONL fixtures**

V2 bypasses the create command entirely by writing JSONL directly:

```go
func openTaskJSONL(id string) string {
    return `{"id":"` + id + `","title":"Test task","status":"open",...}`
}
dir := setupTickDirWithContent(t, openTaskJSONL("tick-aaa111")+"\n")
```

This isolates each task's tests from other commands' behavior. The `openTaskJSONL`, `inProgressTaskJSONL`, `doneTaskJSONL`, `cancelledTaskJSONL` helpers in 2-2 are reusable JSONL factories that could serve any test. V2's `readTaskByID` and `readTasksJSONL` helpers provide structured readback.

**V3: Unit-style with typed helpers**

V3 uses `setupTask` and `setupTaskFull` helpers that write directly to JSONL but use typed `storage.ReadJSONL` for readback, producing typed structs rather than `map[string]interface{}`. The `readTasksFromDir` helper returns a concrete struct type, providing IDE autocompletion in tests.

However, V3 defines this struct inline (anonymous struct in the return type of `readTasksFromDir`) rather than reusing `task.Task`. This is because V3's string-based timestamps don't need `time.Time` parsing, so the test struct uses plain strings.

### Test Infrastructure Quality

| Infrastructure | V1 | V2 | V3 |
|---------------|-----|-----|-----|
| Test data setup | Full CLI pipeline (create) | Raw JSONL strings | Direct JSONL write via helper |
| Data readback | `os.ReadFile` + string search | `readTasksJSONL` (parsed maps) | `readTasksFromDir` (typed structs) |
| Output assertion | `strings.Contains` (loose) | Exact string match | Exact string match |
| Error assertion | `strings.Contains` on stderr | Error message substring | stderr content + exit code |
| Exit code testing | Direct (Run returns int) | Indirect (Run returns error) | Direct (Run returns int) |

V2's test infrastructure is the most portable across tasks -- the JSONL factory functions and structured readback could be reused for any command's tests. V3's typed readback is the most developer-friendly. V1's integration approach is the most realistic but the least isolated.

## Phase Verdict

**V2 is the best Phase 2 implementation.**

### Reasoning

**1. Cross-task architectural consistency.** V2 is the only version that maintains consistent patterns across all three tasks: `error` returns everywhere, `NormalizeID` on both input and stored IDs, `unwrapMutationError` applied retroactively. When V2 discovered the mutation error wrapping problem during task 2-3, it went back and fixed tasks 2-2 and even Phase 1's create command. This is the behavior of a codebase being maintained, not just implemented task-by-task.

**2. Most thorough test coverage at every level.** V2 has the highest test function count (62 vs 41/59), validates every edge case in the spec, and provides the most precise assertions (exact string matching for error messages and output). The test-to-impl ratio (3.9:1) matches V3 but V2's tests cover more edge cases per LOC.

**3. Correct spec compliance.** V2 matches the specification's error format exactly (`"Cannot %s task %s -- status is '%s'"` with capital C and em dash) across both domain and CLI layers. V3 deviates with lowercase "cannot" and a regular hyphen, which cascades through 2-1 and 2-2.

**4. No bugs.** V1 has the blocks deduplication bug in task 2-3. V3 is correct but V2 is more robust (normalized dedup comparison).

**5. Best domain layer design.** V2's package-level `validTransitions` map (task 2-1) allocates once at init rather than per-call like V1. The `from []Status` slice pattern is more extensible than V3's switch and more efficient than V1's per-call map. V2's `updateFlags` struct with explicit `*Provided` booleans (task 2-3) is the clearest representation of "was this flag given" vs "what value was given."

**6. Phase-level refactoring.** V2's Phase 2 diff touches 9 files including retroactive improvements to Phase 1 code. V1 touches 7 files with no retroactive fixes. V3 touches 7 files with no retroactive fixes. V2 treats the codebase as a whole rather than as isolated task deliverables.

**V2's weaknesses:**
- Task 2-1 uses bare return values `(oldStatus, newStatus, err)` instead of `TransitionResult` struct. Less self-documenting at call sites.
- Task 2-3 calls `ValidateTitle` twice (once to check, once to get cleaned value). Minor waste.
- Total LOC (1851) is between V1 (1041) and V3 (1962), but the extra code is justified by broader test coverage and cross-task helpers.

**V3 is a close second** -- its tests are thorough (1555 test LOC), it has direct exit code verification, and the round-trip persistence test is creative. But the lowercase error messages, exported `UpdateFlags` type, distributed error formatting (12+ duplicate `fmt.Fprintf` calls), and inconsistent Mutate callback patterns (pointer vs index) weaken it.

**V1 is third** -- the most concise implementation (293 impl LOC, 1041 total) with the cleanest error flow, but the blocks dedup bug, weakest test coverage (41 functions, no exact output matching), and reimplemented `strings.Contains` helpers place it last.

### Final Ranking: V2 > V3 > V1
