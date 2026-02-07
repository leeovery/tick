# Phase 1: Walking Skeleton

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 1-1: Task model & ID generation | V2 | Clear | Integrated `NewTask` factory with validation, rune-aware title length, `*int` options pattern |
| 1-2: JSONL storage with atomic writes | V2 | Clear | Best error wrapping, buffered I/O, field ordering test, null-check in omit test |
| 1-3: SQLite cache with freshness | V2 | Clear | Table-probing corruption recovery, logging on corruption (spec req), transaction rollback test |
| 1-4: Storage engine with file locking | V2 | Moderate | Exact spec error message casing, `ParseTasks` refactor, test-friendly timeout, safe flock usage |
| 1-5: CLI framework & tick init | V3 | Narrow | Testable `IsTTY(io.Writer)`, flexible flag parsing, thorough `os.Stat` error handling; V1 missing TTY entirely |
| 1-6: tick create command | V2 | Moderate | Best error messages with flag context, `*int` priority, no-partial-mutation tests, typed `createFlags` struct |
| 1-7: tick list & tick show | V2 | Clear | V1 dead code (commands not registered in dispatcher); V2 best architecture, column alignment test; V3 close second |

**Summary**: V2 wins 6 of 7 tasks. V3 wins 1 (CLI framework). V1 wins none.

## Cross-Task Architecture Analysis

This is where the phase-level view reveals patterns invisible to individual task analyses.

### Package Organization: A Deliberate Divergence

The most significant cross-task architectural decision is package layout. V2 consistently creates sub-packages while V1 and V3 use a flat structure:

- **V1**: `internal/storage/jsonl.go`, `internal/storage/cache.go`, `internal/storage/store.go` -- all in package `storage`
- **V2**: `internal/storage/jsonl/jsonl.go`, `internal/storage/sqlite/sqlite.go`, `internal/storage/store.go` -- store imports from sub-packages
- **V3**: `internal/storage/jsonl.go`, `internal/storage/cache.go`, `internal/storage/store.go` -- all in package `storage`

This has cascading effects across tasks 1-2 through 1-7:

1. **Function naming**: V2's JSONL functions are `jsonl.WriteTasks()` and `jsonl.ReadTasks()` (clean, non-redundant because the package name provides context). V1/V3 use `storage.WriteJSONL()` and `storage.ReadJSONL()` (redundant `JSONL` in the name because the flat package needs disambiguation).

2. **Cross-package imports in the store (task 1-4)**: V2's `store.go` explicitly imports `jsonl` and `sqlite` packages, making dependencies visible at the import level. V1/V3's store is in the same package as the JSONL and cache code, so there are no import boundaries -- any function can call any other function without explicit coupling signals.

3. **Test isolation**: V2's sub-packages get their own `_test.go` files in separate directories. V1/V3's tests share a package, meaning test helpers defined in one test file (e.g., `sampleTask()` in `jsonl_test.go`) are implicitly available to other test files. This is convenient but creates hidden coupling -- V1's `store_test.go` can use helpers from `jsonl_test.go` without any import.

### The Task Struct Decision Cascades Through Every Layer

V3's choice to use `string` timestamps in `task.Task` (task 1-1) creates a simplification cascade but also a type-safety debt that compounds:

**Task 1-1 (model)**: V3 stores `Created string`, `Updated string`, `Closed string`. V1/V2 store `Created time.Time`, `Updated time.Time`, `Closed *time.Time`.

**Task 1-2 (JSONL)**: V3 can use `json.NewEncoder` directly on `task.Task` with no intermediate struct. V1/V2 must define intermediate serialization structs (`jsonlTask`/`taskJSON`) with `toJSONL()`/`fromJSONL()` conversion functions. V3's JSONL implementation is 97 LOC vs V1's 216 and V2's 183.

**Task 1-3 (SQLite)**: V3 passes `t.Created` directly to `taskStmt.Exec()` as a string. V1 calls `task.FormatTimestamp(t.Created)`. V2 uses `t.Created.UTC().Format(timeFormat)`. V3 also stores empty strings instead of NULL for optional fields (no `sql.NullString` needed), while V1 uses `sql.NullString` and V2 uses `*string`.

**Task 1-6 (create)**: V3 calls `task.DefaultTimestamps()` which returns pre-formatted strings. V1/V2 let `task.NewTask()` handle `time.Now().UTC().Truncate(time.Second)` internally.

**Task 1-7 (show)**: V3 uses `COALESCE(description, '')` in SQL and scans directly into `string`. V1/V2 use `sql.NullString` and manual `.Valid` checks (3 checks per query).

The net effect: V3 has ~30% less code in storage layers but has traded away the ability to do time arithmetic (e.g., "how long has this task been open?") without parsing strings back into `time.Time`. This debt will compound in Phase 2 (status transitions set/clear `closed` timestamp) and Phase 5 (stats).

### The `NewTask` Design Creates Architectural Ripples

V2's integrated `NewTask(title, opts, existsFn)` factory (task 1-1) has a direct impact on task 1-6 (create command):

**V2's create command** simply builds a `TaskOptions` struct and calls `task.NewTask()`. The task package handles ID generation, title validation/trimming, priority defaulting, and all field validation:
```go
opts := &task.TaskOptions{Priority: flags.priority, Description: flags.description, ...}
newTask, err := task.NewTask(trimmedTitle, opts, existsFn)
```

**V1's create command** must manually coordinate: call `GenerateID(existsFn)`, then `ValidateTitle()`, then `TrimTitle()`, then `ValidatePriority()`, then `NewTask(id, title, priority)`, then set Description/Parent/BlockedBy on the returned struct. Six separate steps that must happen in the right order.

**V3's create command** has even more manual coordination: call `GenerateID(exists)`, `TrimTitle()`, `ValidateTitle()`, `ValidatePriority()`, `DefaultTimestamps()`, `DefaultPriority()`, then build the `task.Task{}` struct literal directly. Seven steps with no constructor guardrail.

This means V2's task package enforces a "pit of success" -- it is impossible to create an invalid task through the public API. V1 and V3 require callers to correctly orchestrate validation, which is error-prone as more commands (Phase 2's update, Phase 3's dep add) also construct or modify tasks.

### Error Handling Patterns: Three Distinct Philosophies

Across all 7 tasks, each version maintains a consistent (but different) error strategy:

**V1**: Gerund-prefix wrapping (`"opening JSONL file: %w"`, `"creating temp file: %w"`, `"inserting task %s: %w"`). This is concise and follows the Go standard library style (see `os` and `net/http` packages). Applied in storage layers (tasks 1-2, 1-3, 1-4) and CLI layer (tasks 1-5, 1-6, 1-7).

**V2**: "failed to" prefix wrapping (`"failed to open tasks file: %w"`, `"failed to insert task %s: %w"`, `"failed to acquire lock: %w"`). More verbose but human-readable. Applied uniformly across all tasks. V2 additionally includes contextual values in errors (e.g., `"invalid created timestamp %q: %w"` in JSONL, `"task %q not found (referenced in --blocked-by)"` in create). This contextual richness is unique to V2.

**V3**: Bare error returns with no wrapping (`return err`, `return nil, err`). Applied consistently across tasks 1-2, 1-3, 1-4, and 1-7. The only exceptions are in task 1-1 (validation errors have messages) and task 1-5 (CLI init has inline error messages). This means a caller receiving an error from V3's `ReadJSONL` gets a raw `os.Open` error with no indication it came from JSONL reading. A caller receiving an error from V3's `Rebuild` gets a raw `tx.Exec` error with no cache context. This pattern makes debugging significantly harder and would be caught by `go vet` linting in production.

### The Store Layer (Task 1-4) Reveals Integration Quality

Task 1-4's `Store` type is the integration point where tasks 1-1 through 1-3 compose together. The quality of this composition reveals cross-task thinking:

**V2** proactively modified the JSONL package (task 1-2) when building the store (task 1-4). It added `ParseTasks(data []byte)` to the `jsonl` package, refactoring `ReadTasks` to delegate to it. This allowed the store to read raw file bytes once and parse from memory, avoiding the double-read problem. This kind of cross-package improvement shows understanding of how the pieces fit together.

**V1** had already built `ReadJSONLBytes(data []byte)` in task 1-2 as an extra utility function. When task 1-4 needed byte-based parsing, it was already available. Whether this was prescient design or coincidence, V1's store cleanly uses the existing API.

**V3** did NOT have a byte-based parser from task 1-2, and task 1-4 did not add one. Instead, the store reads the file twice: once with `os.ReadFile` for raw bytes (hash computation), once with `ReadJSONL(path)` for parsing (which reopens and re-reads the file). This is a cross-task integration failure -- the task 1-2 and task 1-4 implementations were not designed with each other in mind.

### CLI Command Dispatch: Compounding Integration Debt

The CLI dispatcher (task 1-5) must be updated every time a new command is added (tasks 1-6 and 1-7). This creates a cross-task integration point:

**V1** has a critical defect: task 1-7's `cmdList` and `cmdShow` methods exist but are NOT registered in the `switch subcmd` block in `cli.go`. The help text was updated, but the actual routing was not. This means `tick list` and `tick show` produce "Unknown command" errors. This is a cross-task integration failure that would be immediately caught by any manual test.

**V2** correctly registers all commands in `app.go` across tasks 1-5, 1-6, and 1-7. Each task's PR modifies the dispatcher.

**V3** correctly registers all commands in `cli.go` across tasks 1-5, 1-6, and 1-7.

### Shared Helpers and Cross-Task Code Reuse

**V1** reuses `task.FormatTimestamp()` (task 1-1) in the cache layer (task 1-3) and the CLI output (task 1-6). It also reuses `ReadJSONLBytes` (task 1-2) in the store (task 1-4). However, V1's test helpers are not shared across tasks -- each test file defines its own.

**V2** reuses the `TaskOptions` pattern (task 1-1) in the create command (task 1-6). The `*int` priority idiom flows naturally from model to CLI. V2 also reuses `jsonl.ParseTasks` (added in task 1-4) as a cross-package utility. V2's test helpers are minimal but don't overlap.

**V3** extracted shared test helpers in task 1-7, creating `test_helpers_test.go` with `setupTickDir()`, `setupTask()`, `setupTaskWithPriority()`, `setupTaskFull()`, and `readTasksFromDir()`. It also refactored `create_test.go` to use these shared helpers. This is the only version that explicitly manages test helper debt across tasks. V3 also defines `normalizeIDs()` as a reusable helper in the create command.

## Code Quality Patterns

### Naming Consistency

Each version maintains internal consistency across all 7 tasks:

- **V1**: `cmd` prefix for CLI handlers (`cmdInit`, `cmdCreate`, `cmdList`, `cmdShow`), `Find` for discovery (`FindTickDir`), `GlobalOpts` for flags. Status constants are `StatusOpen`, etc.
- **V2**: `run` prefix for CLI handlers (`runInit`, `runCreate`, `runList`, `runShow`), `Discover` for discovery (`DiscoverTickDir`), `Config` for flags, typed `OutputFormat` constants. The `run` prefix is more idiomatic Go for internal dispatchers.
- **V3**: `run` prefix for CLI handlers (same as V2), `Discover` for discovery, `GlobalFlags` for flags, `ParseGlobalFlags` exported. Mixed exported/unexported: `CreateFlags` is exported but only used internally.

### Return Type Convention

This is a phase-level pattern that affects every task:

- **V1**: CLI handlers return `error`. The central `Run` method maps to exit codes and prints `Error: ` prefix. Consistent across all tasks.
- **V2**: CLI handlers return `error`. Same centralized pattern. The `Run` method itself returns `error` (not `int`), pushing exit code mapping to `main.go`.
- **V3**: CLI handlers return `int` (exit code). Each handler formats its own `Error: ` prefix. This means every error path in every handler has `fmt.Fprintf(a.Stderr, "Error: %s\n", err)` + `return 1` -- counted across tasks 1-5 through 1-7, this pattern appears 20+ times in V3 vs 0 times in V1/V2 (they use centralized formatting).

### Constants and Magic Numbers

Across all 7 tasks, V2 consistently names every magic number:

- Task 1-1: `idPrefix`, `idRandomBytes`, `maxIDRetries`, `maxTitleLength`, `minPriority`, `maxPriority`, `defaultPriority`
- Task 1-3: `timeFormat` constant for timestamp formatting
- Task 1-4: `defaultLockTimeout`

V1 names some (`maxRetries = 5`) but not others (500 chars, priority bounds). V3 names some (`idPrefix`, `maxRetries`, `lockTimeout`) but also has dead code (`idHexLen = 6` defined but unused).

### DRY Across the Phase

**V2's best DRY achievement**: The `task.NewTask()` factory (task 1-1) is the single point of task creation used by the create command (task 1-6). No other code path can create a task without validation.

**V3's worst DRY violation**: In `create_test.go` (task 1-6), an anonymous struct with 10 JSON-tagged fields is defined inline 7 separate times instead of being extracted to a type. In `list_show_test.go` (task 1-7), V3 addressed this by creating shared test helpers -- showing awareness of the problem, but only after it had already accumulated.

**V1's hidden DRY problem**: V1's `ValidateTitle()` trims internally but discards the trimmed result (returns only `error`). This means every caller must call both `TrimTitle()` and `ValidateTitle()` -- a two-step coordination that is repeated in task 1-6 (create command). V2's `ValidateTitle()` returns `(string, error)`, eliminating this duplication.

## Test Coverage Analysis

### Aggregate Test Counts

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Task 1-1 test functions | 10 | 9 | 8 |
| Task 1-2 test subtests | 12 | 12 | 14 |
| Task 1-3 test subtests | 13 | 15 | 14 |
| Task 1-4 test subtests | 10 | 13 | 14 |
| Task 1-5 test cases | 12 | 31 | 26 |
| Task 1-6 test cases | 16 | 20 | 22 |
| Task 1-7 test cases | 17 | 20 | 21 |
| **Phase 1 total tests** | **~90** | **~120** | **~119** |
| Test LOC (impl files only) | ~2,025 | ~3,621 | ~3,942 |

V2 and V3 have roughly 33-45% more tests than V1. V3 has the most test LOC but V2 has comparable test count with more concise tests.

### Systematic Edge Cases

**V2 systematically covers** across the phase:
- Unicode boundary testing (task 1-1: 500-rune Chinese title)
- Field ordering verification (task 1-2: JSON field positions)
- Corruption with wrong schema (task 1-3: valid SQLite, wrong tables)
- No partial mutation (task 1-6: 3 separate tests)
- Column alignment verification (task 1-7: programmatic position checks)
- Integration tests building actual binary (task 1-5: 6 tests)
- Bidirectional lock blocking (task 1-4: shared-blocks-exclusive AND exclusive-blocks-shared)

**V3 systematically covers** across the phase:
- Error prefix verification (task 1-7: checks `"Error: "` prefix)
- Timestamp format testing (task 1-6: explicit UTC ISO 8601 test)
- Exit code verification (tasks 1-5 through 1-7)
- Special characters in data (task 1-2: quotes and backslashes)
- Nil vs empty slice distinction (task 1-1: `nil` vs `[]string{}`)
- Skip-rebuild proof via marker technique (task 1-3)

**V1 systematically misses** across the phase:
- No TTY detection at all (task 1-5)
- No SQLite failure test (task 1-4)
- No `--blocks` non-existent task test (task 1-6)
- No transaction rollback test (task 1-3)
- Commands not registered in dispatcher (task 1-7: dead code)
- Fewest tests in every task except 1-1

### Testing Patterns

**Table-driven tests**: All three versions use table-driven tests extensively in task 1-1 (validation tests). V1 uses them most consistently across the phase.

**Integration-style tests**: V2's task 1-5 uniquely builds the actual binary and tests via `exec.Command`, verifying real exit codes and stderr separation. This is the only version with true end-to-end integration tests.

**Test helper evolution**: V3 shows progressive improvement -- task 1-6 has DRY violations (repeated anonymous structs), but task 1-7 extracts shared helpers into `test_helpers_test.go` and refactors task 1-6's tests. This suggests V3 was developed sequentially with learning between tasks.

**Test data strategy**: V1 uses the actual CLI (`tick init`, `tick create`) to set up test data -- the most realistic but most fragile (a bug in create breaks list/show tests). V2/V3 write JSONL directly for test setup, which is more isolated. V3's `readTasksFromDir` calls the real `storage.ReadJSONL`, adding partial integration coverage to otherwise isolated tests.

### Where Test Quality Matters Most

For Phase 1 (walking skeleton), the most critical test scenarios are:

1. **Round-trip integrity** (data survives write/read cycle): All three versions cover this adequately.
2. **Corruption recovery** (cache can be rebuilt from JSONL): V2 covers this most thoroughly with wrong-schema tests.
3. **Concurrent access safety**: V2 and V3 both test bidirectional lock blocking; V1 only tests one direction.
4. **Command actually works end-to-end**: V1 critically fails here -- list and show are unreachable.

## Phase Verdict

**V2 is the clear winner of Phase 1**, winning 6 of 7 tasks and demonstrating the strongest cross-task integration.

### Why V2 Wins at the Phase Level

1. **Architectural coherence across tasks**: V2's sub-package organization (`jsonl/`, `sqlite/`) creates clean import boundaries that make cross-package dependencies explicit. The `task.NewTask()` factory established in task 1-1 flows naturally through task 1-6's create command. The `ParseTasks()` addition in task 1-4 shows deliberate cross-task thinking -- modifying an earlier package to serve a later task's needs.

2. **No integration defects**: Every command is properly registered in the dispatcher. The store reads files exactly once. The flock is safely created per-operation. V1 has the critical dead-code defect (task 1-7). V3 has the double file-read inefficiency (task 1-4).

3. **Consistent quality floor**: V2 wraps every error with context across all 7 tasks. V2 names every magic number. V2 uses typed constants (`OutputFormat`, `Status`). V3 has bare error returns in 4 of 7 tasks. V1 is inconsistent about constants.

4. **Strongest test coverage for this phase's purpose**: Phase 1 is about "proving the dual-storage architecture end-to-end." V2 uniquely tests: field ordering in JSONL (proving storage format correctness), wrong-schema corruption recovery (proving cache rebuildability), column alignment in list output (proving UI correctness), and integration tests building the real binary (proving end-to-end behavior).

5. **Best error diagnostics**: V2's error messages include flag context in create (`"task %q not found (referenced in --blocked-by)"`), invalid values in JSONL parsing (`"invalid created timestamp %q: %w"`), and warning logs on cache corruption. These support the debugging workflow that Phase 1 needs to be reliable.

### Where V2 Falls Short

- **Missing TTY `io.Writer` testability**: V2's `detectTTY()` is hardcoded to `os.Stdout`, while V3's `IsTTY(io.Writer)` is injectable and testable. This is V2's only task loss (task 1-5, to V3).
- **No space between format columns**: V2's list output uses `"%-12s%-12s%-4s%s\n"` with no separator spaces, which could cause columns to visually merge.
- **Multiline description in show**: V2's `printShowOutput` uses single-line output for descriptions, not handling newlines. V1 and V3 both split on `\n` and indent.
- **No `filepath.Abs` normalization gap**: This is actually one of V2's strengths -- it normalizes paths in both `runInit` and `DiscoverTickDir`.

### V3's Strengths as Runner-Up

V3 wins on: testable TTY detection, flexible flag positioning, shared test helpers (task 1-7), timestamp format testing (task 1-6), and the most total test LOC. V3 would be a stronger contender if not for: string timestamps losing type safety, bare error returns throughout storage layers, double file reads in the store, and the `int`-returning command pattern that creates repetitive error formatting.

### V1's Position

V1 is consistently third. Its most critical failure is the dead-code defect in task 1-7 where commands are not registered in the CLI dispatcher -- this means the walking skeleton literally cannot walk. V1 also has the fewest tests in every task, no TTY detection, and a `NewTask` function that does no validation. V1's only structural advantage is the `FormatTimestamp()` utility and the `ReadJSONLBytes()` function that happened to be useful for the store layer.

### Final Phase 1 Ranking

**V2 >> V3 > V1**

V2 is the strongest by a clear margin at the phase level. The gap between V2 and V3 is moderate -- V3 has comparable test coverage but weaker architecture and error handling. The gap between V3 and V1 is significant -- V1 has a critical integration defect and consistently weaker test coverage.
