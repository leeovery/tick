# Phase 1: Walking Skeleton

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 1-1: Task model & ID generation | V5 | Moderate | Custom JSON marshal/unmarshal on Task gives consistent serialization everywhere; 23 vs 20 subtests |
| 1-2: JSONL storage with atomic writes | V5 | Strong | 86 vs 161 impl LOC -- V5 delegates to Task's MarshalJSON; V6 duplicates field mapping in storage-layer DTO |
| 1-3: SQLite cache with freshness | V6 (slight) | Marginal | V6 better test org and INSERT OR REPLACE; V5 DRYer recreate helper and idiomatic errors |
| 1-4: Storage engine with file locking | V6 | Moderate | Marshal-once pattern avoids file re-read; lazy cache init with corruption recovery; but V5 has stronger lock tests |
| 1-5: CLI framework & tick init | V6 | Strong | App struct more testable; correct error casing; table-driven flag tests; global flags after subcommand |
| 1-6: tick create command | V5 (slight) | Marginal | Stricter unknown-flag/extra-arg rejection; uses task.DefaultPriority constant; verifies no partial mutation |
| 1-7: tick list & tick show | V5 | Moderate | Column widths match spec exactly (12,12,4 vs 12,13,5); error casing matches spec; TrimSpace on description |

**Tally: V5 wins 4, V6 wins 2, tie 1 (1-3).**

## Cross-Task Architecture Analysis

### Serialization ownership: The phase-defining decision

The single most consequential architectural decision in Phase 1 is *where JSON serialization lives*. This choice, made in task 1-1, cascades through tasks 1-2, 1-4, 1-6, and 1-7.

**V5: Task owns its own serialization.**

In task 1-1, V5 defines `MarshalJSON`/`UnmarshalJSON` on the `Task` type with a private `taskJSON` shadow struct and `json:"-"` tags on timestamp fields:

```go
// V5: internal/task/task.go lines 52-54
Created     time.Time  `json:"-"`
Updated     time.Time  `json:"-"`
Closed      *time.Time `json:"-"`
```

This flows downstream: the JSONL storage layer (task 1-2) becomes a thin 86-line wrapper that simply feeds tasks to `json.NewEncoder`. The storage engine (task 1-4) calls `storage.WriteTasks` which delegates to `json.Encode(t)` -- no awareness of timestamps or field ordering. Any code anywhere in the codebase that calls `json.Marshal(task)` gets correct ISO 8601 timestamps and correct field ordering automatically.

**V6: Storage layer owns serialization.**

In task 1-1, V6 leaves `json:"created"` on the timestamp fields (no `-` tag) and provides no `MarshalJSON`/`UnmarshalJSON`. This means `task.Task` alone produces Go's default `time.Time` JSON output (RFC 3339 with nanoseconds). In task 1-2, V6 compensates by introducing a storage-layer `jsonlTask` DTO with explicit `toJSONL`/`fromJSONL` conversion functions -- 75 extra lines of field mapping that mirrors the Task struct:

```go
// V6: internal/storage/jsonl.go lines 17-28
type jsonlTask struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      string   `json:"status"`       // loses task.Status type safety
    Priority    int      `json:"priority"`
    Description string   `json:"description,omitempty"`
    BlockedBy   []string `json:"blocked_by,omitempty"`
    Parent      string   `json:"parent,omitempty"`
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

In task 1-4, V6 compensates for this split by introducing `MarshalJSONL` (a new function that wraps `toJSONL` + `json.Marshal` into bytes), `WriteJSONLRaw` (writes pre-marshaled bytes atomically), and `writeAtomic` (extracted atomic write helper). This decomposition enables the marshal-once pattern in `Store.Mutate`, which is genuinely efficient -- but the decomposition exists primarily *because* serialization was not centralized on the Task type.

**Phase-level impact:** V5's centralized approach means the Task type is self-documenting about its serialization contract. Any consumer (tests, future CLI JSON output, debugging) can `json.Marshal` a Task and get the right thing. V6's split means there is an implicit rule: "never marshal a Task directly; always go through the storage layer." This is never documented or enforced, and the `json:"created"` tag on `task.Task` is actively misleading -- it suggests direct marshaling would work but it would produce the wrong timestamp format.

### Package structure and import graph

**V5: 4 packages, layered hierarchy**
```
task -> storage -> cache -> engine -> cli
```

V5 creates `internal/engine` as a separate package from `internal/storage` and `internal/cache`. The engine imports both storage and cache. The CLI imports engine, task, and (implicitly via engine) storage. This creates clean layering with clear dependency direction.

**V6: 3 packages, shared storage**
```
task -> storage (jsonl + cache + store) -> cli
```

V6 co-locates JSONL, cache, and store in `internal/storage`. This reduces the import graph but puts everything into one package. The tradeoff: `OpenCache` is exported from the same package as `ReadJSONL`, which means internal implementation details of the cache are visible to anyone who imports `storage`.

Phase-level assessment: V5's extra package is justified because it enforces architectural boundaries. In V6, the store directly accesses `OpenCache` as a package-level function, while in V5 it goes through `cache.New` -- a package boundary that makes the dependency explicit.

### Command handler pattern: Context vs App

Every command in Phase 1 (init, create, list, show) follows a consistent handler pattern that was established in task 1-5 and extended in tasks 1-6 and 1-7.

**V5: Map dispatch with Context struct**

```go
// V5: internal/cli/cli.go lines 97-101
var commands = map[string]func(*Context) error{
    "init":   runInit,
    "create": runCreate,
    "list":   runList,
    "show":   runShow,
}
```

Every handler receives `*Context` with pre-resolved `WorkDir`, `Format`, `Quiet`, `Verbose`, and `Args`. Handlers are unexported (`runCreate`, `runList`). The pattern repeats identically: `DiscoverTickDir(ctx.WorkDir)` -> `engine.NewStore(tickDir, ctx.storeOpts()...)` -> `defer store.Close()` -> `store.Mutate/Query`.

**V6: Switch dispatch with App methods + exported functions**

```go
// V6: internal/cli/app.go lines 57-66
switch subcmd {
case "init":   err = a.handleInit(fc, fmtr, subArgs)
case "create": err = a.handleCreate(fc, fmtr, subArgs)
case "list":   err = a.handleList(fc, fmtr, subArgs)
case "show":   err = a.handleShow(fc, fmtr, subArgs)
```

Each `handleX` method resolves `Getwd()`, then delegates to an exported `RunX` function with explicit parameters. V6 extracts shared boilerplate into `helpers.go`:

```go
// V6: internal/cli/helpers.go lines 33-40
func openStore(dir string, fc FormatConfig) (*storage.Store, error) {
    tickDir, err := DiscoverTickDir(dir)
    if err != nil { return nil, err }
    return storage.NewStore(tickDir, storeOpts(fc)...)
}
```

**Phase-level impact:** V6's `openStore` helper eliminates the repeated 5-line `DiscoverTickDir` + `NewStore` sequence that V5 has in every single handler. Across 4 commands, V6 saves ~20 lines of pure boilerplate. V6 also extracts `outputMutationResult` which is reused by create and update commands. V5 has no equivalent shared helper -- `create.go` and `show.go` each independently construct output. This means V6 has better DRY at the CLI layer, while V5 has better DRY at the storage layer.

### SQL query sharing

Both versions share SQL fragments for the ready/blocked queries across `list.go`, `ready.go`, `blocked.go`, and `stats.go` (phases 3+5 commands). The approach diverges significantly:

**V5: String constants composed via concatenation**
```go
// V5: internal/cli/ready.go
const readyWhereClause = `t.status = 'open'
  AND NOT EXISTS (...blockers...)
  AND NOT EXISTS (...children...)`

const ReadyQuery = `SELECT ... FROM tasks t WHERE ` + readyWhereClause + ` ORDER BY ...`
```

**V6: Function-based SQL fragment builders**
```go
// V6: internal/cli/query_helpers.go
func ReadyConditions() []string {
    return []string{`t.status = 'open'`, ReadyNoUnclosedBlockers(), ReadyNoOpenChildren()}
}
func BlockedConditions() []string { ... }
```

V6's fragment builder approach is more composable -- the list command's `buildListQuery` can combine `ReadyConditions()` with additional status/priority/descendant filters by appending to the conditions slice. V5 achieves the same via wrapping the `ReadyQuery` in a subquery with `WHERE 1=1 AND status = ? ...`. V6's approach avoids the subquery nesting, producing simpler SQL at runtime.

### Verbose logging plumbing

Both versions thread verbose logging from CLI flags through to the storage engine, but with very different weight.

**V5: Full type with enabled flag (37 + 59 + 167 = 263 lines including tests)**
```go
// V5: internal/engine/verbose.go
type VerboseLogger struct { w io.Writer; enabled bool }
func (v *VerboseLogger) Log(msg string)  { if !v.enabled { return }; ... }
func (v *VerboseLogger) Logf(format string, args ...interface{}) { ... }
```

The `VerboseLogger` is created in every `NewStore` call (even when verbose is off) via `NewVerboseLogger(nil, false)`. Context has `newVerboseLogger()` and `storeOpts()` methods.

**V6: Nil-safe pointer with func callback (39 lines including tests)**
```go
// V6: internal/cli/verbose.go
type VerboseLogger struct { w io.Writer }
func (vl *VerboseLogger) Log(msg string) { if vl == nil { return }; ... }
```

The store accepts `func(msg string)` instead of a full type. The bridge is a 7-line `storeOpts` function. When verbose is off, no logger is created at all.

V6's approach is dramatically lighter (39 vs 263 lines) while achieving the same functionality. V5's `Logf` convenience method is the only addition of value, and it is trivially replaced by `fmt.Sprintf` at the call site.

## Code Quality Patterns

### Error message style

The two versions have a **systematic, consistent, and opposite** approach to error message formatting:

| Pattern | V5 | V6 |
|---------|----|----|
| Error creation (static) | `fmt.Errorf("...")` | `errors.New("...")` |
| Error prefix style | Gerund: `"creating temp file:"` | Failed-to: `"failed to create temp file:"` |
| Error string casing | Often capitalized: `"Task '%s' not found"` | Always lowercase: `"task '%s' not found"` |
| Spec-literal messages | Matches spec casing exactly | Follows Go convention instead |

V5's approach is more spec-faithful (reproducing exact error strings from the task plans). V6's approach is more Go-idiomatic per the Go Code Review Comments guide, which says error strings should not be capitalized or end with punctuation. However, V5's use of `fmt.Errorf` for static strings (where `errors.New` is preferred) is a minor anti-pattern.

These patterns are remarkably consistent *within* each version across all 7 tasks, suggesting a coherent agent style rather than ad-hoc choices.

### ID normalization consistency

Both versions call `task.NormalizeID()` at the same architectural point: at the boundary between user input and internal processing. In `create.go`, `show.go`, and `list.go`, input IDs are normalized before being passed to the storage layer. Neither version normalizes IDs inside the storage or cache layers -- it is always the CLI's responsibility. This is a consistent convention in both codebases.

### Store lifecycle: open-use-close

Every command handler in both versions follows the same lifecycle pattern:

```go
store, err := engine.NewStore(tickDir, ...)    // V5
store, err := openStore(dir, fc)                // V6
if err != nil { return err }
defer store.Close()
// ... store.Mutate or store.Query ...
```

V6 extracts this into the `openStore` helper; V5 repeats the 5-line block. Both correctly use `defer store.Close()`. Neither version ever forgets the defer. This consistency is good Go practice.

### Nil-pointer handling

Both versions handle `*time.Time` (the `Closed` field) carefully and consistently. In the cache `Rebuild`, both convert nil `Closed` to a nil `*string` for SQL. In `show.go`, both handle the nullable closed column (V5 via `sql.NullString`, V6 via `*string`). In the JSONL layer, both check `t.Closed != nil` before formatting. No nil dereference bugs were found in either version across all 7 tasks.

## Test Coverage Analysis

### Aggregate counts (Phase 1 tasks only)

| Metric | V5 | V6 |
|--------|----|----|
| Implementation LOC | 1,948 | 1,846 |
| Test LOC | 4,540 | 4,525 |
| Test:impl ratio | 2.33:1 | 2.45:1 |
| Test files | 10 | 7 |
| Top-level test functions | ~42 | ~35 |
| Total subtests (approx) | ~120 | ~130 |

The aggregate numbers are remarkably close. V6 has slightly more test lines per implementation line. V5 spreads tests across more files (separate `list_test.go` and `show_test.go` vs V6's combined `list_show_test.go`).

### Test helper reuse patterns

**V5:** Each command's test file is self-contained. `create_test.go` has `initTickProject` (calls the real `Run` init path), `initTickProjectWithTasks`, and `readTasksFromFile` (manual JSON parsing). `list_test.go` and `show_test.go` each have their own helper functions. There is no shared test utility file.

**V6:** `create_test.go` has `setupTickProject` (direct filesystem setup), `setupTickProjectWithTasks`, `readPersistedTasks` (uses `storage.ReadJSONL`), and `runCreate` (wrapper). `list_show_test.go` has `runList` and `runShow` wrappers. V6 also uses `helpers_test.go` in later phases for shared test utilities.

V5's `initTickProject` is more integration-like (tests init + create together) but couples tests. V6's direct filesystem setup is more isolated. V6's use of `storage.ReadJSONL` in tests reuses production code, which is both a strength (consistency) and a risk (test depends on production code being correct).

### Edge case distribution

Across all 7 tasks, the unique edge cases each version covers that the other misses:

**V5 only:**
- JSON serialization round-trip (3 tests in task 1-1)
- `\r` and `\r\n` newline variants in title validation
- Case-insensitive self-reference detection (blocked_by and parent)
- Empty task list write (0 tasks -> 0 bytes)
- Special characters in description (`<>&"'`)
- Description with embedded newlines
- Corrupted `.tick/` directory (exists but missing tasks.jsonl)
- `.tick/` found in starting directory itself
- No partial mutation verification on error (create tests verify 0 tasks after failed create)

**V6 only:**
- 501 multi-byte character rejection boundary
- Write error cleanup (nonexistent directory)
- Exact spec format string comparison for JSONL
- Cache corruption recovery (missing cache.db, corrupted metadata schema, garbage bytes)
- Stale cache via external JSONL modification (the real-world scenario)
- Global flags after subcommand
- TTY format resolution (6 subtests for ResolveFormat)
- Duplicate dependency skip in applyBlocks
- Rebuild method tests (count, lock, verbose, empty)

V5's unique edge cases are more focused on data correctness (serialization, encoding, self-references). V6's unique edge cases are more focused on operational resilience (corruption recovery, stale detection, integration scenarios).

## Phase Verdict

**V5 is the better Phase 1 implementation**, winning 4 of 7 tasks with moderate-to-strong margins, while V6 wins 2 with one near-tie.

The decisive factor is the **serialization architecture decision** made in task 1-1 and reverberating through tasks 1-2 and 1-4. V5's `MarshalJSON`/`UnmarshalJSON` on the Task type is the architecturally correct choice for this codebase. It produces:
- 86 vs 161 lines in the JSONL storage layer (task 1-2)
- No storage-layer DTO duplication
- Type safety preserved throughout (V6 downgrades `task.Status` to `string` in `jsonlTask`)
- Consistent serialization regardless of marshal context (V6's `task.Task` produces wrong timestamps if marshaled outside the storage layer)

V6 has genuine architectural strengths that are visible only at the phase level:
- The `openStore` and `outputMutationResult` helpers in `helpers.go` show better DRY at the CLI layer
- The `query_helpers.go` SQL fragment builders are more composable than V5's string constants
- Cache corruption recovery in `ensureFresh` is production-grade resilience
- The `MarshalJSONL` + `WriteJSONLRaw` decomposition enables marshal-once (even though the need arose from the serialization split)
- The `App` struct with injected `Getwd` is more testable for CLI integration
- Verbose logging is 85% lighter (39 vs 263 lines)

However, these V6 advantages are downstream optimizations that compensate for the fundamental serialization split. V5's architecture is simpler, more correct, and more maintainable at the foundation level. When building a walking skeleton, the foundation matters most.

The one area where V6 is clearly superior is **operational resilience testing** -- corruption recovery, stale cache detection via external modification, and lazy initialization with recovery. These tests catch real-world failure modes that V5 does not cover. A production deployment would benefit from V6's approach here. But for a Phase 1 skeleton that must prove the architecture works end-to-end, V5's cleaner layering and data-correctness focus is the right priority.

**Overall: V5 wins Phase 1.**
