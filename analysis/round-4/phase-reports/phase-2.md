# Phase 2: Task Lifecycle

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| tick-core-2-1: Transition validation logic | V5 | Moderate | Nested-map O(1) lookup; tests cover all 7 updated-timestamp paths and inline closed verification |
| tick-core-2-2: start/done/cancel/reopen commands | V6 | Slight | Dedicated test helper, timestamp range assertions, stricter error checks |
| tick-core-2-3: tick update command | V6 | Moderate | Shared `outputMutationResult` with SQL enrichment; `applyBlocks` deduplication; deterministic timestamps |

## Cross-Task Architecture Analysis

### The "Helpers Divergence" -- Biggest Phase-Level Pattern

The three tasks in this phase reveal a **fundamental architectural fork** in how the two codebases evolve shared code. This pattern is invisible at the individual task level but defines the phase.

**V5: Accrete-in-place.** Shared functions (`validateIDsExist`, `splitCSV`, `normalizeIDs`) live inside `create.go` where they were first needed. When `update.go` (task 2-3) needs them, it imports them by package visibility -- no refactoring occurs. The blocks application logic (`tasks[idx].BlockedBy = append(...)`) is inlined identically in both `create.go` (line 126-130) and `update.go` (line 126-129). This is copy-paste duplication across two commands:

```go
// V5 create.go lines 126-130 (blocks application)
for _, blockID := range blocks {
    idx := existing[blockID]
    tasks[idx].BlockedBy = append(tasks[idx].BlockedBy, id)
    tasks[idx].Updated = now
}

// V5 update.go lines 126-129 (same logic, same variable names)
for _, blockID := range opts.blocks {
    bIdx := existing[blockID]
    tasks[bIdx].BlockedBy = append(tasks[bIdx].BlockedBy, id)
    tasks[bIdx].Updated = now
}
```

**V6: Extract-and-share.** Task 2-3 introduces `helpers.go` with four purpose-built functions (`openStore`, `outputMutationResult`, `parseCommaSeparatedIDs`, `applyBlocks`). These are used by both `create.go` and `update.go`. `openStore` alone is called from 8 command files. The `show.go` refactoring to extract `queryShowData` enables both `RunShow` and `RunUpdate` to share the identical full-SQL query path.

Usage counts across the codebase (non-test, non-definition):

| Helper | Files Using It |
|--------|---------------|
| `openStore` | 8 (transition, update, create, show, list, dep, stats, rebuild) |
| `outputMutationResult` | 2 (create, update) |
| `parseCommaSeparatedIDs` | 2 (create, update) |
| `applyBlocks` | 2 (create, update) |

This is the kind of cross-task architecture decision that accumulates: V5's duplication is harmless at 3 tasks but becomes a maintenance burden at 10+ commands. V6 paid the extraction cost during Phase 2 and every subsequent phase benefits.

### The "Function Signature" Fork

Across all three tasks, the two codebases use consistently different function signatures for command handlers:

**V5 pattern -- Context struct, unexported closures:**
```go
func runTransition(command string) func(*Context) error  // task 2-2
func runUpdate(ctx *Context) error                       // task 2-3
```

**V6 pattern -- Explicit params, exported flat functions:**
```go
func RunTransition(dir string, command string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error  // task 2-2
func RunUpdate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error                      // task 2-3
```

V6's approach is more verbose (6 params vs 1 context struct) but the parameter list is **identical** across `RunCreate`, `RunShow`, `RunTransition`, and `RunUpdate` -- the `(dir, fc, fmtr, args, stdout)` quintuple is effectively an implicit interface. This means V6's functions are independently testable and callable without constructing a full App, while V5's require a `*Context` that bundles everything.

The tradeoff: V5 can add new context fields (e.g., a logger) without changing any function signature. V6 must thread new params through every function (they did this with `FormatConfig` and `Formatter` which appear in every handler).

### The "Mutate Callback" Convergence

Despite architectural differences, both versions converge on an identical Mutate callback structure across tasks 2-2 and 2-3. The pattern is:

```go
err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
    // 1. Build index/lookup
    // 2. Find target task
    // 3. Validate references (parent, blocks)
    // 4. Apply mutations
    // 5. Set timestamp
    // 6. Capture result for output
    return tasks, nil
})
```

Both `transition.go` and `update.go` in both versions follow this exact 6-step structure inside the Mutate closure. The callback returns the same `[]task.Task` slice (mutated in place by index), never creates new slices. This is a strong signal that both AI agents internalized the same storage engine contract correctly.

### ID Lookup Strategy -- O(1) vs O(n) at Phase Scale

V5 consistently builds a `map[string]int` (ID -> index) across both `create.go` and `update.go`, enabling O(1) lookup and direct index-based mutation:

```go
// V5 pattern (create.go and update.go)
existing := make(map[string]int, len(tasks))
for i, t := range tasks { existing[t.ID] = i }
// ...
idx := existing[id]
tasks[idx].Field = value
```

V6 consistently builds a `map[string]bool` for existence checks but uses linear iteration for the actual task lookup:

```go
// V6 pattern (update.go)
idSet := make(map[string]bool, len(tasks))
for _, t := range tasks { idSet[t.ID] = true }
// ...
for i := range tasks {
    if tasks[i].ID != opts.id { continue }
    // mutate tasks[i]
}
```

V5's approach is O(1) for both existence check and index lookup. V6 does O(1) for existence check but O(n) for the actual find-and-mutate. With typical task lists (hundreds, not millions), this is immaterial for performance -- but V5's map stores strictly more useful information (index vs boolean) at the same construction cost.

### Validation Ordering -- Pre-mutation vs Post-mutation

A subtle cross-task pattern emerges in how dependency validation interacts with mutation. In task 2-3 (`update --blocks`):

**V5** validates *before* applying blocks (lines 95-106 of `update.go`):
```
validate -> apply blocks -> set timestamp
```

**V6** applies blocks *first*, then validates against the post-mutation state (lines 184-192 of `update.go`):
```
apply blocks -> validate (on mutated graph) -> return
```

V6's post-mutation validation is actually more correct for cycle detection: by applying the new edges first, `ValidateDependency` BFS traverses the *resulting* graph, catching cycles that only exist after all edges are added. V5 validates each edge against the *pre-mutation* graph, which could theoretically miss a cycle created by the combination of multiple simultaneous `--blocks` targets.

However, V6's approach has a safety concern: if validation fails after mutation, the Mutate callback returns an error and the storage engine discards the entire mutation (no persist). This relies on the atomicity guarantee of the storage engine -- which is correct, but couples the correctness of validation to the transactional behavior of the persistence layer.

## Code Quality Patterns

### Error Message Casing -- Consistent Philosophy Split

Across all three tasks, V5 and V6 take opposite positions on error message casing, and each is internally consistent:

| Error String | V5 | V6 |
|-------------|-----|-----|
| Transition rejection | `"Cannot %s task..."` | `"cannot %s task..."` |
| Missing ID | `"Task ID is required..."` | `"task ID is required..."` |
| Not found | `"Task '%s' not found"` | `"task '%s' not found"` |

V5 capitalizes to match spec text verbatim. V6 uses lowercase per Go convention (`go vet` flags capitalized error strings). Both are internally consistent across all three tasks. Since both dispatchers prepend `"Error: "`, V6's lowercase produces `"Error: task ID is required"` -- which reads naturally. V5 produces `"Error: Task ID is required"` -- which reads as a proper sentence but violates `go vet`.

### Flag Parser Strictness -- Consistent Philosophy Split

V5 rejects unknown flags across all commands:
```go
case strings.HasPrefix(arg, "-"):
    return "", opts, fmt.Errorf("unknown flag '%s'", arg)
```

V6 silently skips unknown flags across all commands:
```go
case strings.HasPrefix(arg, "-"):
    // Unknown flag -- skip (global flags already extracted)
```

This is not a per-task decision -- it's a consistent architectural choice. V5 is defensive (catch typos), V6 is tolerant (allow flag passthrough from global parsing). V6's approach avoids a class of bugs where `--quiet` or `--pretty` flags that were already consumed by the global parser would be rejected by the subcommand parser. V5 must ensure global flag stripping is complete before reaching subcommand parsers.

### Output Formatting -- In-Memory vs SQL Re-query

A cross-task pattern in output handling:

**V5** captures the mutated `task.Task` struct from the Mutate callback and converts it to output format in-memory via `taskToShowData()`. This function (defined in `show.go`) converts fields 1:1 but cannot enrich relationships (blocked_by titles, children, parent title) because the in-memory struct only has IDs, not full objects.

**V6** captures only the task ID from Mutate, then calls `outputMutationResult()` which performs a fresh `queryShowData()` SQL query. This produces the same enriched output as `tick show`, including blocked_by task titles/statuses, children lists, and parent names.

This matters because the spec says update output should be "full task details (like `tick show`)." V6 achieves literal spec compliance by sharing the exact query path. V5 produces a structurally similar but informationally poorer output.

## Test Coverage Analysis

### Test Helper Evolution Across Tasks

**V5** never introduces command-specific test helpers across all three tasks. Every test body declares `var stdout, stderr bytes.Buffer`, calls `Run([]string{...})`, and manually unpacks results. This is repeated ~60 times across the three test files (341 + 449 + lines of transition/update tests).

**V6** introduces a dedicated `runXxx` helper for each command file:
- `runTransition(t, dir, command, args...)` -- task 2-2
- `runUpdate(t, dir, args...)` -- task 2-3

Each returns `(stdout, stderr string, exitCode int)`, eliminating `bytes.Buffer` boilerplate. The helpers also set `IsTTY: true` consistently, ensuring tests use `PrettyFormatter` rather than defaulting to TOON.

At the phase level, V6's pattern produces **measurably DRYer test code** despite being ~10% more total lines. The extra lines come from explicit struct literals and richer assertions, not boilerplate.

### Timestamp Testing Strategy -- Deterministic vs Real-Time

A cross-task pattern in how timestamps are validated:

**V5** uses real-time `time.Sleep` in one update test (1.1 seconds) and `before/after` bracketing in transition tests. The `time.Sleep` is a test smell -- it adds wall-clock delay to CI and introduces flakiness risk.

**V6** uses fixed `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)` as the fixture timestamp across all update tests. Since `time.Now()` during the test will always be after this past date, the assertion `Updated.After(now)` is deterministic. No sleeps needed.

V6's approach is strictly better: faster tests, no flakiness, and the fixed date serves as documentation of the expected test state.

### Test Fixture Construction -- NewTask vs Struct Literals

**V5** uses `task.NewTask(id, title)` then mutates fields:
```go
tk := task.NewTask("tick-aaaaaa", "Done from IP")
tk.Status = task.StatusInProgress
```

**V6** uses struct literals with all fields explicit:
```go
ipTask := task.Task{
    ID: "tick-aaa111", Title: "IP task", Status: task.StatusInProgress,
    Priority: 2, Created: now, Updated: now,
}
```

V6's approach makes every field visible at the test site -- no hidden defaults from `NewTask`. This matters for tests that assert on `Created`, `Updated`, or `Priority`, where `NewTask`'s defaults could mask bugs. V5's approach is more concise but relies on knowing `NewTask` internals.

### Coverage Gaps at Phase Level

| Gap | V5 | V6 |
|-----|----|----|
| Updated timestamp: all 7 valid transitions tested (task 2-1) | Yes (7/7) | No (4/7) |
| Closed verified inline in valid-transition table (task 2-1) | Yes (wantClosed field) | No (separate test only) |
| Missing-ID tested for all 4 commands (task 2-2) | Yes (loop over 4 commands) | No (only `start`) |
| No-mutation on invalid title (task 2-3) | Yes (reads back file, verifies title unchanged) | No (checks exit code only) |
| Blocks deduplication tested (task 2-3) | No | Yes (dedicated test) |
| Created-timestamp immutability tested (task 2-3) | No | Yes (`Created.Equal(now)`) |
| Target's Updated refreshed in blocks test (task 2-3) | No | Yes (`target.Updated.After(now)`) |
| time.Sleep in tests | Yes (1.1s in update timestamp test) | No |

V5 has better coverage at the domain/unit level (task 2-1). V6 has better coverage at the CLI integration level (tasks 2-2, 2-3). Neither achieves perfect coverage.

## Phase Verdict

**V6 wins Phase 2 overall**, driven by architectural decisions that compound across tasks.

The decisive factors are cross-task in nature and invisible at the individual task level:

1. **Shared helpers (`helpers.go`)** -- `openStore`, `outputMutationResult`, `parseCommaSeparatedIDs`, `applyBlocks` eliminate duplication between create and update commands. V5 duplicates the blocks-application logic verbatim. This is a 4-function extraction with 8+ usage sites across the codebase -- the ROI increases with every subsequent phase.

2. **`show.go` refactoring** -- Extracting `queryShowData` enables both `RunShow` and `RunUpdate` to share the identical SQL query path. V5's `taskToShowData` is a lossy in-memory conversion that produces informationally poorer output, failing to match the spec's "like `tick show`" requirement.

3. **`applyBlocks` deduplication** -- V6's shared helper prevents duplicate `blocked_by` entries; V5's inline logic will silently create duplicates on repeated `--blocks` invocations. This is a correctness advantage, not just a style one.

4. **Post-mutation validation** -- V6 validates dependency cycles against the post-mutation graph state, which is more correct for detecting cycles created by the combination of new edges.

5. **Test infrastructure** -- V6's per-command test helpers and deterministic timestamp strategy produce faster, more maintainable tests. V5's `time.Sleep(1100ms)` is a CI tax paid on every run.

**V5's advantages are real but narrower:** the nested-map data structure in task 2-1 is more elegant, test coverage at the unit level is more exhaustive (7/7 updated-timestamp paths, all 4 commands for missing-ID), and strict flag rejection catches more user errors. These are task-local wins that don't compound across the phase.

The margin is **moderate**. V5 wins task 2-1 and V6 wins tasks 2-2 and 2-3, but the cross-task architecture patterns (helpers extraction, output enrichment, deduplication logic) provide V6 with compounding advantages that extend beyond this phase into Phases 3-5.
