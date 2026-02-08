# Phase 1: Walking Skeleton

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 1-1 Task Model & ID Generation | V2 | Moderate | V2 uses `NormalizeID()` in self-reference checks (spec-correct); broader test coverage (20 vs 17 subtests) |
| 1-2 JSONL Storage | V4 | Moderate | V4 eliminates 84 lines of `taskJSON` boilerplate; non-nil empty slice; DRYer. V2's explicit timestamp format is more defensive but costly |
| 1-3 SQLite Cache | Tie | Close | V2 wins API (`EnsureFresh` returns `*Cache`) and test rigor (rollback proof). V4 wins code quality (NULL handling, error wrapping, `log.Printf`) |
| 1-4 Storage Engine | V4 | Moderate | V4's `SerializeJSONL` avoids post-write re-read; 20 vs 16 test functions; cleaner package separation |
| 1-5 CLI Framework & init | V2 | Significant | V2 has 6 integration tests building real binary; 27 vs 24 tests; more edge cases (corrupted .tick/, unwritable dir) |
| 1-6 tick create | V4 | Moderate | V4 has type-safe test setup (`task.Task` structs vs raw JSON); 24 top-level test functions; DRY `parseCommaSeparatedIDs` helper |
| 1-7 tick list & show | V4 | Modest | V4 type-safe test data; multiline description handling; better error verification. V2 wins column alignment test and orphaned-parent handling |

**Score: V2 wins 2 tasks, V4 wins 4 tasks, 1 tie.**

## Cross-Task Architecture Analysis

### Data Flow: The Full Request Path

Both implementations share the same conceptual pipeline: `main -> App.Run -> parseGlobalFlags -> subcommand -> DiscoverTickDir -> Store.Mutate/Query -> JSONL + Cache -> output`. But the concrete wiring differs in ways that compound across tasks.

**V2's data flow through the storage layer:**
```
cli/create.go  -->  DiscoverTickDir(a.workDir)
               -->  a.newStore(tickDir)           // app.go:207
               -->  store.Mutate(fn)              // store.go:89
                    --> os.ReadFile(s.jsonlPath)
                    --> jsonl.ParseTasks(rawContent)  // uses taskJSON intermediate struct
                    --> sqlite.EnsureFresh(...)       // returns *Cache for reuse
                    --> fn(tasks)
                    --> jsonl.WriteTasks(...)
                    --> os.ReadFile(s.jsonlPath)      // RE-READ from disk for hash
                    --> cache.Rebuild(modified, newRawContent)
```

**V4's data flow through the storage layer:**
```
cli/create.go  -->  DiscoverTickDir(a.Dir)
               -->  a.openStore(tickDir)          // cli.go:173
               -->  s.Mutate(fn)                  // store.go:81
                    --> os.ReadFile(s.jsonlPath)
                    --> task.ReadJSONLFromBytes(jsonlData)  // direct unmarshal to Task
                    --> cache.EnsureFresh(...)              // opens+closes internally
                    --> fn(tasks)
                    --> task.SerializeJSONL(modified)       // serialize in memory
                    --> task.WriteJSONL(...)                // writes to disk
                    --> cache.Open(s.dbPath)                // RE-OPEN cache
                    --> c.Rebuild(modified, newJSONLData)
```

The critical difference is where inefficiency lives. V2 re-reads the JSONL from disk after writing (one extra I/O). V4 re-opens the SQLite database after `EnsureFresh` closes it (one extra `sql.Open` + schema init). V4's `SerializeJSONL` also means tasks are serialized twice (once for hash, once inside `WriteJSONL` which calls `SerializeJSONL` again). Neither is ideal, but V2's re-read is slightly more wasteful since disk I/O is slower than an in-memory `sql.Open`.

### Shared Helpers: Cross-Command Reuse Patterns

Both codebases establish helpers in Phase 1 that get reused across every subsequent command. The most important ones:

**V2 extracts `queryShowData` (show.go:70) -- reused by create and update:**
```go
// V2: show.go -- used by show, create, update
func queryShowData(store *storage.Store, lookupID string) (*showData, error) {
    var data *showData
    err := store.Query(func(db *sql.DB) error {
        // ... full task query with dependencies, children, parent
    })
    return data, err
}
```
After `create` and `update` mutate a task, they call `queryShowData(store, createdTask.ID)` to fetch the full detail for output. This means create/update perform *two* store operations: one `Mutate` then one `Query`. This is a strong DRY pattern -- the show rendering logic is written once and reused.

**V4 uses `taskToDetail` (create.go:214) -- a lightweight conversion without DB round-trip:**
```go
// V4: create.go -- used by create and update
func taskToDetail(t *task.Task) TaskDetail {
    detail := TaskDetail{
        ID: t.ID, Title: t.Title, Status: string(t.Status),
        Priority: t.Priority, Description: t.Description,
        // ... format timestamps
    }
    return detail
}
```
V4 avoids the second store round-trip by converting the in-memory `task.Task` directly. This is more efficient (no extra Query after Mutate) but the output is incomplete -- `BlockedBy` and `Children` lists are empty since they require a DB join. V2's approach shows richer output after create/update at the cost of an extra lock acquisition.

**V2's `unwrapMutationError` (app.go:174) -- used by create, update, dep, transition:**
```go
// V2: app.go -- strips "mutation failed: " wrapper from store errors
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```
This is used at 4 call sites across V2 to clean up error messages from `store.Mutate` (which wraps with `"mutation failed: %w"`). V4 does not need this because V4's `Mutate` returns the callback error directly without wrapping it (line 122: `return err` instead of `return fmt.Errorf("mutation failed: %w", err)`). V4's approach is cleaner -- the unwrapper shouldn't be needed if the store returns clean errors.

**Both share the `DiscoverTickDir -> openStore/newStore -> defer store.Close()` boilerplate:**
Every command in both codebases follows this exact 6-line preamble:
```go
tickDir, err := DiscoverTickDir(a.workDir)  // or a.Dir
if err != nil { return err }
store, err := a.newStore(tickDir)            // or a.openStore(tickDir)
if err != nil { return err }
defer store.Close()
```
Neither version extracts this into a helper. By Phase 1 task-7, this pattern appears at 3 call sites (create, list, show). By the full codebase, it appears at 9+ sites. Both miss an opportunity for a `withStore(fn)` abstraction.

### Formatter Interface: Phase-Level Architectural Decision

Both implementations introduce a `Formatter` interface in Phase 1 that every command renders through. The designs differ:

**V2** uses `OutputFormat` as a `string` type and `*showData` as pointer argument:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, tasks []TaskRow) error
    FormatTaskDetail(w io.Writer, data *showData) error
    // ...
}
```

**V4** uses `Format` as an `int` (iota) type and `TaskDetail` as value argument:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
    FormatTaskDetail(w io.Writer, detail TaskDetail) error
    // ...
}
```

Key differences: (1) V4 passes `quiet` into the formatter's `FormatTaskList`, pushing quiet-mode handling into the formatter; V2 handles quiet before calling the formatter. V4's approach is cleaner if different formatters need different quiet behaviors. (2) V2 uses a pointer to `showData` for detail; V4 uses a value type `TaskDetail`. V4's approach requires an extra conversion step from local types to the shared `TaskDetail` struct (visible in show.go:138-166 where V4 manually copies fields). V2 uses `showData` directly as the formatter data type, avoiding the copy but coupling the formatter to the query layer's types.

V4's explicit `RelatedTask` and `TaskDetail` types in `format.go` are better -- they establish a clean data contract between commands and formatters, independent of the query layer. V2's `showData` in `show.go` is specific to the show command's query logic but gets reused by the formatter, creating tighter coupling.

### Package Organization

**V2's package tree for Phase 1:**
```
internal/task/              -- Task struct, validation, ID generation
internal/storage/jsonl/     -- JSONL read/write with taskJSON intermediate
internal/storage/sqlite/    -- SQLite cache
internal/storage/store.go   -- Store orchestrator (sibling to sub-packages)
internal/cli/               -- All CLI commands + formatter
cmd/tick/main.go
```

**V4's package tree for Phase 1:**
```
internal/task/              -- Task struct, validation, JSONL read/write
internal/cache/             -- SQLite cache
internal/store/             -- Store orchestrator
internal/cli/               -- All CLI commands + formatter
cmd/tick/main.go
```

V2 nests JSONL and SQLite under `internal/storage/` with the store as a sibling file. This creates an unusual Go pattern where `storage/store.go` imports `storage/jsonl` and `storage/sqlite` -- a parent package importing its own children.

V4 keeps each concern in a dedicated top-level package (`task`, `cache`, `store`) with clean import edges: `store` imports both `task` and `cache`; `cache` imports `task`; `cli` imports all three. This is a more conventional Go layout with clearer dependency direction.

V4 co-locates JSONL code with the Task model (`internal/task/jsonl.go`), which has the advantage that serialization uses the `Task` struct's own JSON tags directly (no intermediate type). V2 separates JSONL into its own sub-package, requiring the `taskJSON` bridge struct.

## Code Quality Patterns

### Error Handling Consistency

**V2** is inconsistent about error wrapping vs raw returns. The cache's `Rebuild` in the original implementation returned raw `err` without context, though the final version wraps properly. Across Phase 1, V2 uses `errors.New()` for static messages (idiomatic) and `fmt.Errorf("...: %w")` for wrapped ones.

**V4** consistently wraps every error with `fmt.Errorf("context: %w", err)` throughout all layers. Even static error messages use `fmt.Errorf()` instead of `errors.New()`, which is slightly less idiomatic but more consistent.

Winner: V4 for consistency. The pattern in V4's cache (`"failed to begin rebuild transaction: %w"`, `"failed to clear dependencies: %w"`, etc.) makes debugging straightforward at every layer.

### Error Message Style

V2 uses capitalized error messages matching the spec verbatim:
```go
// V2 examples
"Could not create .tick/ directory: %w"
"Could not acquire lock on .tick/lock - another process may be using tick"
"Failed to generate unique ID after 5 attempts"
```

V4 uses lowercase following Go convention, deviating from spec:
```go
// V4 examples
"failed to create .tick/ directory: %w"
"could not acquire lock on %s - another process may be using tick"
"Failed to generate unique ID after 5 attempts"  // inconsistent - capitalized here
```

V2 is more spec-compliant. V4 is more Go-idiomatic. V4 is also internally inconsistent (some messages capitalized, some not).

### Naming Conventions

| Pattern | V2 | V4 |
|---------|-----|-----|
| App fields | `a.config.Quiet`, `a.stdout`, `a.workDir` (unexported) | `a.Quiet`, `a.Stdout`, `a.Dir` (exported) |
| Constructor | `cli.NewApp()` | `&cli.App{Stdout: os.Stdout, ...}` (struct literal) |
| Store creation | `a.newStore(tickDir)` | `a.openStore(tickDir)` |
| Run return | `error` | `int` (exit code) |
| JSONL functions | `jsonl.WriteTasks`, `jsonl.ReadTasks`, `jsonl.ParseTasks` | `task.WriteJSONL`, `task.ReadJSONL`, `task.ReadJSONLFromBytes`, `task.SerializeJSONL` |
| Cache constructor | `sqlite.NewCache(path)` | `cache.Open(path)` |
| Test names | `TestCreateCommand/"it creates..."` (monolithic) | `TestCreate_WithOnlyTitle/"it creates..."` (per-concern) |

V2's unexported fields with a constructor (`NewApp()`) provide better encapsulation. V4's exported fields allow direct struct literal construction in tests and main.go, which is simpler but less safe.

V4's test naming with top-level functions per concern (`TestCreate_WithOnlyTitle`, `TestCreate_PriorityFlag`) is significantly better for Go's test runner -- `go test -run TestCreate_PriorityFlag` targets one test precisely. V2's monolithic `TestCreateCommand` with 23 subtests requires `-run "TestCreateCommand/it sets priority"` which is harder to type and remember.

### Normalization Consistency

A subtle cross-task pattern: V2 applies `task.NormalizeID()` at almost every ID comparison point -- in the Task model (ValidateBlockedBy/ValidateParent), in the existence lookup map during create, and in the --blocks update loop. This is defensive and consistent.

V4 uses raw `==` comparison in task model validators and raw `t.ID` keys in the existence map during create:
```go
// V4 create.go:36-38 -- no normalization in lookup map
for _, t := range tasks {
    existingIDs[t.ID] = true  // raw ID
}
```

V4 assumes IDs are already normalized in the JSONL file. This is generally true since `GenerateID` produces lowercase IDs and `NormalizeID` is applied at CLI input boundaries. But it creates a fragile assumption that breaks if JSONL is manually edited with uppercase IDs.

## Test Coverage Analysis

### Aggregate Counts

| Metric | V2 | V4 |
|--------|-----|-----|
| Phase 1 impl LOC | 2,002 | 1,642 |
| Phase 1 test LOC | 5,350 | 5,015 |
| Test-to-impl ratio | 2.67:1 | 3.05:1 |
| Test files | 11 | 10 |
| Integration tests | 6 (binary build) | 0 |
| Approximate subtests | ~124 | ~117 |

V4 achieves a higher test-to-impl ratio (3.05 vs 2.67) with 18% less implementation code -- strong evidence of better design (less code to cover, proportionally more tests per line).

V2's 6 integration tests (building and running the actual binary) are a significant unique asset. V4 never verifies the real `main()` -> `os.Exit()` -> stderr pipeline.

### Testing Approach Differences

**V2 test data setup:** Raw JSONL strings.
```go
// V2 pattern (create_test.go, show_test.go, list_test.go)
content := `{"id":"tick-aaa111","title":"Setup","status":"open","priority":1,"created":"2026-01-20T10:00:00Z","updated":"2026-01-20T10:00:00Z"}`
dir := setupTickDirWithContent(t, content)
```
This is fragile -- typos in JSON field names, missing required fields, or malformed timestamps fail at runtime with unhelpful JSON parse errors.

**V4 test data setup:** Typed `task.Task` structs.
```go
// V4 pattern (create_test.go, show_test.go, list_test.go)
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Setup", Status: task.StatusOpen, Priority: 1,
     Created: now, Updated: now},
}
dir := setupInitializedDirWithTasks(t, tasks)
```
Compile-time safety catches structural errors. Uses domain constants (`task.StatusOpen`). This pattern is used consistently across all V4 CLI tests.

**V2 assertion style:** Often parses raw JSON output into `map[string]interface{}`:
```go
// V2: create_test.go
var tk map[string]interface{}
json.Unmarshal([]byte(lines[0]), &tk)
if tk["priority"].(float64) != 2 { ... }  // fragile type assertion
```

**V4 assertion style:** Reads back typed structs:
```go
// V4: create_test.go
tasks, _ := task.ReadJSONL(filepath.Join(dir, ".tick", "tasks.jsonl"))
if tasks[0].Priority != 2 { ... }  // compile-time safe
```

### Edge Case Coverage Unique to Each

**V2-only edge cases:**
- Integration tests verifying real exit codes via `*exec.ExitError`
- Corrupted `.tick/` directory (exists but no `tasks.jsonl`)
- Unwritable directory (`chmod 0555`)
- Discovery stops at first `.tick/` match (two nested directories)
- Cache transaction rollback proof via duplicate-ID constraint violation
- Marker-based skip-rebuild verification
- Schema column type verification (TEXT vs INTEGER)
- Description with embedded newlines in JSONL
- `"null"` absence check in JSONL output
- Hardcoded expected SHA256 hash value

**V4-only edge cases:**
- Non-integer priority input (`"abc"`)
- Explicit `cache.db` existence check after create
- Dedicated timestamp UTC verification test
- Two-task uniqueness verification for ID generation
- 20-ID pattern test (vs V2's single ID)
- Non-nil empty slice guarantee from `ReadJSONLFromBytes`
- `ReadJSONLFromBytes` equivalence with `ReadJSONL`
- `SerializeJSONL` byte-for-byte equivalence with `WriteJSONL`
- Format flag override tests (--toon/--pretty/--json override TTY default)

## Phase Verdict

**V4 is the stronger Phase 1 implementation**, though V2 has notable advantages that a merged implementation should preserve.

### V4 advantages that accumulate across the phase:

1. **18% less implementation code** (1,642 vs 2,002 LOC) with equivalent or better functionality. The biggest single saving is eliminating the `taskJSON` intermediate struct (84 lines) by co-locating JSONL serialization with the Task model.

2. **Type-safe test infrastructure.** Every CLI test in V4 uses `task.Task` structs for setup and `task.ReadJSONL` for assertions. This pattern, established in task 1-6 and carried through 1-7, prevents an entire class of test-data bugs that V2 is vulnerable to with its raw JSON strings.

3. **Better package architecture.** V4's `task/`, `cache/`, `store/` layout follows standard Go conventions with clean one-directional import edges. V2's `storage/jsonl/` and `storage/sqlite/` nested under a parent `storage/` package that imports them is structurally unusual.

4. **`SerializeJSONL` helper.** V4 introduced this in task 1-4, enabling in-memory hash computation without re-reading from disk. This is the architecturally correct approach to the post-write hash problem, and it pays forward into all future mutation flows.

5. **Per-concern test functions.** V4's `TestCreate_PriorityFlag`, `TestShow_BlockedBySection` pattern is unambiguously better than V2's monolithic `TestCreateCommand` with 23 subtests. It enables targeted test execution, clearer failure output, and easier maintenance.

6. **Cleaner formatter data contract.** V4 defines `TaskDetail`, `RelatedTask`, and `listRow` as explicit shared types in `format.go`, creating a clean boundary between command logic and output rendering. V2 uses `showData` from show.go as the formatter type, coupling the formatter to one command's query logic.

### V2 advantages that should not be lost:

1. **Integration tests.** V2's 6 binary-level tests (build, exec, check exit code and stderr) verify the full pipeline including `main()` -> `os.Exit()`. V4 never tests this boundary. This is the single most important thing V4 is missing.

2. **`EnsureFresh` returning `*Cache`.** V2's cache API design avoids the double-open problem that V4 has in every `Query` and `Mutate` call. This is the more correct API for a "gatekeeper called on every operation."

3. **Case-insensitive normalization throughout.** V2 applies `NormalizeID()` at every comparison point (validators, existence maps, --blocks loop). V4 skips normalization in validators and existence maps, relying on the assumption that IDs are already lowercase. V2's defensive approach is more robust.

4. **Spec-verbatim error messages.** V2's `"Could not acquire lock on .tick/lock"` and `"Could not create .tick/ directory"` match the specification exactly. V4 deviates with lowercase and dynamic paths.

5. **`queryShowData` reuse.** V2 reuses the full show query for create and update output, giving richer results (with blocked_by and children context). V4's `taskToDetail` is faster but outputs incomplete data.

6. **Stronger transaction and corruption tests.** V2's duplicate-ID constraint violation test actually proves rollback. V2's marker technique proves skip-rebuild. These are more rigorous proofs of correctness than V4's consistency-only checks.

### Net assessment:

V4 wins on engineering quality (less code, better types, cleaner packages, DRYer tests) while V2 wins on spec fidelity and edge-case test rigor at critical boundaries. The ideal implementation would take V4's architecture with V2's integration tests, case-insensitive normalization discipline, and `EnsureFresh` API design.
