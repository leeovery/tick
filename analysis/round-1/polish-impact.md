# V3 Polish Agent Impact Analysis

Commit `dcdb7a8` on `implementation-v3`. Net: -181 lines (77 added, 258 removed).
Files touched: blocked.go, cli.go, cli_test.go, create.go, format.go, format_test.go, list.go, ready.go, update.go.

**Critical context**: The analysis reports were produced per-task (evaluating V3 at each task's commit), meaning they evaluated the PRE-polish V3 code. The polish commit came after all 23 implementation tasks were complete. However, the phase reports and final synthesis aggregate from these task-level snapshots, so they describe the pre-polish state as V3's final quality profile.

## What the Polish Agent Fixed

### 1. Duplicated row-scanning code (IMPROVED)

**Pre-polish**: The identical 8-line `rows.Scan` loop appeared 5 times across `list.go` (3x), `ready.go` (1x), and `blocked.go` (1x). The analysis explicitly flagged this: "V3 has 5 query functions, 3 output formatters, 3 store-opening sequences, and **5 row scanners**" (phase-3 report). Task 3-5's verdict called "the code duplication in its three query functions" a "notable weakness."

**Post-polish**: All 5 instances replaced by calls to a single `scanTaskRows(rows)` helper extracted into `ready.go`. This is a genuine quality improvement -- the kind of DRY refactoring the analysis wished V3 had.

**Impact on score**: Would have narrowed the gap with V2 on task 3-5 (where V2's "one row scanner" was cited as an advantage). The verdict might have softened from "notable weakness" to a minor concern. Would not have changed the winner (V2 still wins on composable WHERE fragments and unified `buildListQuery`), but the margin would be narrower.

### 2. Duplicated output-formatting logic across list/ready/blocked (IMPROVED)

**Pre-polish**: The identical ~15-line block (quiet mode ID output, TaskListData struct building, formatter creation, fmt.Fprint) was copy-pasted across `runList()`, `runReady()`, and `runBlocked()`. Task 3-4's verdict explicitly flagged: "V3 duplicates the full output-formatting logic between `runReady()` and `runBlocked()`."

**Post-polish**: All three replaced by a single `renderTaskList(tasks)` method call. The shared method lives in `ready.go` and handles both quiet mode and formatted output.

**Impact on score**: This was directly cited in the task 3-4 verdict as a V3 weakness. Fixing it would have improved V3's standing on tasks 3-3, 3-4, and 3-5, though V2 would still win all three (V2's architectural advantage of routing `ready`/`blocked` through `runList` is a deeper unification than extracting a shared render helper).

### 3. Dead code removal: StubFormatter (IMPROVED)

**Pre-polish**: `StubFormatter` (34 lines, 6 empty methods) existed in `format.go` with 48 lines of tests in `format_test.go`. This was scaffolding from task 4-1 that became unnecessary once the concrete formatters (TOON, Pretty, JSON) were implemented.

**Post-polish**: `StubFormatter` removed entirely. The interface satisfaction test was updated to verify concrete formatters (`ToonFormatter`, `PrettyFormatter`, `JSONFormatter`) instead.

**Impact on score**: Minor positive. Dead code was not a major factor in any verdict. The analysis mentioned `StubFormatter` in the task 4-1 report neutrally ("correct 'stub' behavior"). Removing it is good housekeeping but wouldn't shift any task ranking.

### 4. Dead code removal: `printTaskDetails` (IMPROVED)

**Pre-polish**: `printTaskDetails` (18 lines in `create.go`) existed as a plain-text task detail printer. It was superseded by the formatter system in Phase 4 -- once `FormatTaskDetail` existed on the Formatter interface, `printTaskDetails` became dead code.

**Post-polish**: Removed entirely.

**Impact on score**: Negligible. No verdict cited this as a V3 weakness. Pure housekeeping.

### 5. Dead code removal: `DefaultOutputFormat` (IMPROVED)

**Pre-polish**: `DefaultOutputFormat()` (10 lines in `cli.go`, 12 lines of test in `cli_test.go`) was superseded by the `FormatConfig` system introduced in task 4-1. It duplicated TTY-based format selection logic that now lives in `NewFormatConfig`.

**Post-polish**: Removed along with its test.

**Impact on score**: Negligible. Task 1-5's verdict praised V3's `DefaultOutputFormat` as a strength. By Phase 4, the FormatConfig system replaced it. Removing the dead version is correct.

### 6. Missing dependency validation in `create --blocked-by` and `update --blocks` (IMPROVED -- BUG FIX)

**Pre-polish**: `create.go` validated that `--blocked-by` and `--blocks` task IDs existed, but did NOT run cycle detection or child-blocked-by-parent validation on them. Similarly, `update.go` validated `--blocks` IDs existed but skipped cycle/parent validation. This meant a user could create circular dependencies or have a child blocked by its parent via the `create` and `update` commands, bypassing the validation that `dep add` enforced (task 3-1/3-2).

**Post-polish**: `create.go` now calls `task.ValidateDependencies()` for `--blocked-by` and `task.ValidateDependency()` for each `--blocks` ID. `update.go` now calls `task.ValidateDependency()` for each `--blocks` ID.

**Impact on score**: This is the most significant fix in the polish commit. It is a correctness bug -- the spec requires dependency validation on all mutation paths, and `create --blocked-by` / `update --blocks` were silently skipping it. If the analysis had been run on pre-polish code and noticed this gap, it would have been a meaningful deduction for V3 on tasks 1-6 (create) and 2-3 (update). The analysis did not explicitly flag this specific gap (it evaluated per-task snapshots where the `--blocked-by` flag on create predated the dependency validation system from Phase 3). This is the kind of cross-phase integration bug that per-task analysis misses.

## What the Polish Agent Did NOT Fix (Pre-existing Issues)

### 1. String timestamps instead of `time.Time` (task 1-1 decision -- UNFIXED)

The most consequential V3 design flaw. `Created`/`Updated` stored as `string` instead of `time.Time`, losing type safety and pushing parse obligations to every consumer. This was a Phase 1 decision that compounded through all subsequent phases. The polish agent did not touch the task model at all.

**Why it was unfixable by polish**: Changing the timestamp type would require modifying the `Task` struct, all serialization/deserialization code, every consumer that formats or compares timestamps, and all tests. This is a foundational refactor, not a polish.

### 2. String-returning Formatter interface instead of `io.Writer` (task 4-1 decision -- UNFIXED)

V3's `FormatTaskList(data) string` / `FormatStats(data) string` signature requires full output buffering instead of streaming. Every verdict from Phase 4 onward cited this as a V3 weakness. The polish agent did not modify the Formatter interface.

**Why it was unfixable by polish**: Changing the return type from `string` to writing to `io.Writer` would require rewriting all 3 formatter implementations (TOON, Pretty, JSON), all 6 interface methods, and all callers. Architectural change, not polish.

### 3. CLI-level verbose logging instead of store-level (task 4-6 decision -- UNFIXED)

V3's `WriteVerbose()` calls fire from the CLI handler BEFORE the store operation executes. The phase 5 analysis flagged that rebuild verbose messages describe "actions that never happened" if the store operation fails early. The polish agent did not touch the verbose logging system.

**Why it was unfixable by polish**: Fixing this requires injecting a Logger interface into the Store, moving all `WriteVerbose` calls from CLI handlers into store methods, and restructuring the store API. Architectural change.

### 4. Per-command formatter creation instead of single resolution (task 4-5 -- PARTIALLY ADDRESSED)

The task 4-5 verdict explicitly called out: "V3 fails this criterion by calling `a.formatConfig.Formatter()` in every command handler." The polish agent consolidated the list/ready/blocked rendering into a shared `renderTaskList` method (which calls `Formatter()` once per render), but `create.go` and `update.go` still independently call `a.formatConfig.Formatter()`. The fundamental issue -- formatter not resolved once at startup like V2 does -- remains.

### 5. Five separate query functions in `list.go` (task 3-5 -- UNFIXED)

The polish removed the duplicated row-scanning code inside each query function but did NOT consolidate the 5 query functions (`queryListTasks`, `queryReadyTasksWithFilters`, `queryBlockedTasksWithFilters`, `queryReadyTasks`, `queryBlockedTasks`) into a unified query builder like V2's `buildListQuery`. The structural problem (5 functions doing logically 3 patterns) persists.

### 6. Bare error returns without context wrapping (Phase 5 -- UNFIXED)

V3's `queryStats` and other functions return bare errors without wrapping (e.g., raw `sql: no rows in result set` with no operational context). The polish agent did not add any error wrapping.

### 7. `int` exit code return convention (task 1-5 decision -- UNFIXED)

V3 returns `int` from command handlers instead of `error`, breaking Go's standard error propagation pattern. Architectural decision, not touched by polish.

### 8. No `workflow` JSON key (tasks 4-4, 5-1 -- UNFIXED)

V3 merges ready/blocked into `by_status` instead of providing a separate `workflow` key per spec. The final synthesis called this "the single most significant spec deviation in V3." Not touched by polish.

### 9. 12+ duplicate `fmt.Fprintf(a.Stderr, "Error: %v\n", err)` calls (Phase 2 -- UNFIXED)

V3 distributes error formatting across many call sites rather than centralizing it. Not touched by polish.

## Did the Polish Agent Cause Any Issues?

### No regressions detected.

The polish commit makes only safe, behavior-preserving transformations:

1. **`scanTaskRows` extraction**: Pure mechanical refactor. The extracted function contains the exact same code that was inlined 5 times. No logic change.

2. **`renderTaskList` extraction**: Pure mechanical refactor. The extracted method contains the exact same quiet-mode + formatter logic that was inlined in 3 places. No logic change.

3. **Dead code removal**: `StubFormatter`, `printTaskDetails`, and `DefaultOutputFormat` were all superseded by later implementations. Removing them cannot affect behavior since they were unreachable from the active code paths.

4. **Dependency validation additions**: These ADD validation that was previously missing. They call existing, tested functions (`ValidateDependencies`, `ValidateDependency`) from task 3-1. The only behavioral change is that `create --blocked-by` and `update --blocks` now correctly reject cycles and parent-blocks-child relationships. This is a bug fix, not a regression.

5. **Test updates**: The `StubFormatter` interface satisfaction test was replaced with concrete formatter type assertions. The `DefaultOutputFormat` test was removed along with the dead function. Both are correct: the tests were testing dead code.

One minor concern: the dependency validation added to `create.go` uses `append(tasks, newTask)` without the typical Go `tasks = append(...)` reassignment, but this is inside a `Mutate` closure where `tasksWithNew` is a new variable, so it is safe (the original `tasks` slice is not mutated if it has capacity).

## Conclusion: Would V3 Have Scored Better or Worse Without Polish?

**V3 scored the same with or without polish, because the analysis evaluated per-task snapshots (pre-polish code).**

The analysis reports were generated by examining V3 at each task's commit, not at the branch tip. This means:

- The "5 row scanners" criticism (phase-3 report) accurately describes the pre-polish state
- The "duplicated output-formatting logic" criticism (task 3-4) accurately describes the pre-polish state
- The dead code (`StubFormatter`, `printTaskDetails`, `DefaultOutputFormat`) was present when each task was evaluated
- The missing dependency validation in `create --blocked-by` and `update --blocks` was present but never flagged (because per-task analysis evaluates create at task 1-6, before the dependency system exists in task 3-1)

**If the analysis HAD been run on the post-polish code, V3 would have scored marginally better:**

- Tasks 3-3, 3-4, 3-5: The DRY improvements (`scanTaskRows`, `renderTaskList`) would have softened the "code duplication" criticisms. The margin with V2 would be narrower on 3-5 (currently "narrow"), but V2 would still win all three tasks due to its deeper architectural unification (`buildListQuery`, `runList` routing).
- Task 4-1: The dead `StubFormatter` removal is cosmetically better but would not shift the verdict.
- Task 4-5: The `renderTaskList` consolidation partially addresses "per-command formatter creation" but does not achieve V2's "resolved once" pattern. No verdict change.
- Cross-phase integration: The dependency validation fix in `create --blocked-by` and `update --blocks` would have been invisible to per-task analysis but visible to a holistic code review. It fixes a real correctness gap.

**Net assessment**: The polish agent improved V3's code quality. It made safe, correct changes that address real issues flagged by the analysis. However, it could not fix V3's fundamental architectural decisions (string timestamps, string-returning formatters, CLI-level verbose logging, `int` exit codes, missing `workflow` key) because those require deep structural refactoring beyond a polish pass.

The polish agent's contribution is best characterized as: **fixed the fixable, left the foundational**. The issues it fixed (DRY helpers, dead code, missing validation) are real but secondary. The issues it left (string timestamps, formatter design, verbose architecture, spec compliance) are the ones that actually determined V3's ranking. V3 would remain firmly in second place regardless of whether the polish was applied.
