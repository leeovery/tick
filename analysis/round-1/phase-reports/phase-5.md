# Phase 5: Stats & Cache

## Task Scorecard
| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 5-1 `tick stats` command | V2 | Clear over V3 | V2 is the only version with correct JSON nesting (`by_status` + `workflow` as separate keys per spec). V2 also reuses ready/blocked SQL constants from `list.go` AND has the strongest test assertions (425 LOC, JSON-parsed exact count validation). V1 flat JSON + inline SQL duplication + weak substring assertions makes it a distant third. |
| 5-2 `tick rebuild` command | V2 | Moderate over V3 | V2 has the most correct verbose logging (7 `logVerbose` calls inside the store, firing at the exact moment each step occurs). V3's CLI-level verbose fires before `store.Rebuild()` runs, producing misleading output on failure. V2 also uniquely tests dependency preservation after rebuild. V1 is weakest (missing hash update and lock verification tests). |

## Cross-Task Architecture Analysis

### How stats and rebuild interact at the store level

Stats and rebuild represent the two fundamental store access patterns: **read-only query** (stats) and **destructive write** (rebuild). The architectural question is whether these two operations share infrastructure consistently, or whether rebuild introduces a parallel code path.

**V1: Stateful cache creates an asymmetric interaction.** Stats uses the standard `store.Query()` flow (shared lock, freshness check, query closure), while rebuild uses `store.ForceRebuild()` which must close and reopen the persistent `s.cache` field. This means stats and rebuild interact with the cache in fundamentally different ways -- stats reads through the long-lived `s.cache`, while rebuild destroys and replaces it:

```go
// V1 ForceRebuild -- must close stored cache before delete
if err := s.cache.Close(); err != nil {
    return 0, fmt.Errorf("closing cache: %w", err)
}
os.Remove(s.cachePath)
cache, err := NewCacheWithRecovery(s.cachePath)
s.cache = cache
```

If stats were to run after rebuild on the same `Store` instance, it would be reading through the newly-assigned `s.cache`. This is correct in single-threaded usage, but the stateful mutation of `s.cache` is a hidden coupling between the two operations. A stats call during a failed rebuild (after `s.cache.Close()` but before reassignment) would crash.

**V2: Per-operation cache creates symmetric independence.** Both stats (`store.Query()`) and rebuild (`store.ForceRebuild()`) open and close their own cache instances. There is zero shared mutable state between the two paths. Stats acquires a shared lock and gets a fresh cache via `sqlite.EnsureFresh()`. Rebuild acquires an exclusive lock, deletes the file, and creates a new cache. These operations cannot interfere with each other because they do not share any state on the `Store` struct:

```go
// V2 ForceRebuild -- local cache, no struct mutation
os.Remove(s.cachePath)
cache, err := sqlite.NewCache(s.cachePath)
defer cache.Close()
```

**V3: Same per-operation pattern as V2, but with a stored `*flock.Flock`.** Both stats and rebuild use local cache instances. However, V3 stores a single `s.flock` on the struct (created in `NewStore`), whereas V2 creates a new `flock.New()` per operation. This means V3's stats and rebuild do share one piece of mutable state: the file lock handle. In practice this is safe because file locks are OS-level, not struct-level, but it is a subtle difference.

### SQL reuse: the stats-to-Phase-3 pipeline

The single most important cross-task pattern in Phase 5 is how stats reuses the ready/blocked SQL from Phase 3 (task 3-3/3-4) -- because the spec explicitly requires it: "Ready/blocked counts reuse the ready query logic from Phase 3."

This creates a three-hop dependency chain: **Phase 3 defines SQL conditions** -> **Phase 3 task 3-5 decomposes them into reusable fragments** -> **Phase 5 task 5-1 stats consumes those fragments for COUNT queries**.

The versions handle this chain differently:

**V2** has the tightest coupling. In Phase 3 (task 3-5), V2 decomposed `ReadySQL` into `readyWhere` + boilerplate. In Phase 5, stats does `SELECT COUNT(*) FROM tasks WHERE ` + `readyWhere`. The fragment is the exact same Go constant. Any change to `readyWhere` in `list.go` automatically propagates to both `tick list --ready` AND `tick stats`. This is the ideal DRY outcome.

**V3** has equally correct reuse but through a different organizational structure. V3 places `ReadyCondition` in `ready.go` and `BlockedCondition` in `blocked.go` as exported constants. Stats references them with `SELECT COUNT(*) FROM tasks t WHERE ` + `ReadyCondition`. The one subtlety: V3's conditions assume a table alias `t`, so the stats query must use `FROM tasks t` (not just `FROM tasks`). This coupling is implicit -- if someone changed the alias in the condition constant, the stats query would break silently.

**V1** has no reuse at all. Stats duplicates the entire ready subquery inline. This means the Phase 3 -> Phase 5 pipeline is broken: changes to the ready/blocked logic in the list command would NOT propagate to stats. The blocked count is computed as `data.Open - data.Ready` (arithmetic) rather than querying the blocked condition. This creates a semantic divergence: if a future change introduced a third state (e.g., "waiting" tasks that are neither ready nor blocked), V1's stats would silently miscount.

### Verbose logging: how task 5-2 inherits from Phase 4 task 4-6

The rebuild command's verbose logging is directly shaped by the Phase 4 verbose architecture. This creates a cross-phase, cross-task interaction:

**V1:** Phase 4 established `store.SetLogger(a.verbose.Log)` via the `openStore` helper. In Phase 5, rebuild calls `a.openStore(tickDir)` which wires the logger, then `ForceRebuild` uses `s.logf()` internally. Only 2 log points exist ("deleting existing cache" and the rebuild count). The architecture is correct (log from inside the operation) but sparse.

**V2:** Phase 4 established `store.SetLogger(a.verbose)` via the `newStore` helper. In Phase 5, rebuild calls `a.newStore(tickDir)` which wires the logger, then `ForceRebuild` uses `s.logVerbose()` internally with 7 distinct messages covering every step. The verbose output tells a complete story: lock acquire -> read JSONL -> parse N tasks -> delete cache -> insert N tasks -> update hash -> lock release. This is the most informative and architecturally sound approach.

**V3:** Phase 4 established `WriteVerbose` calls at the CLI layer rather than injecting a logger into the store. In Phase 5, rebuild emits verbose messages from `runRebuild()` itself:

```go
// V3 rebuild.go -- verbose BEFORE the store call
a.WriteVerbose("lock acquire exclusive")
a.WriteVerbose("delete existing cache.db")
a.WriteVerbose("read JSONL tasks")
count, err := store.Rebuild()
a.WriteVerbose("insert %d tasks into cache", count)
```

The first three messages fire before `store.Rebuild()` is called. If `Rebuild()` fails to acquire the lock, the output will still say "lock acquire exclusive" and "delete existing cache.db" -- actions that never happened. This is a direct consequence of V3's Phase 4 architecture choice: without a logger injected into the store, the CLI cannot know when internal steps actually execute.

The stats command has the same pattern in V3: `WriteVerbose("lock acquire shared")` fires before `store.Query()`, so if the shared lock times out, the verbose output is misleading.

### Lock acquisition: DRY across stats and rebuild

Stats uses a shared lock (read path); rebuild uses an exclusive lock (write path). The question is whether lock boilerplate is shared or duplicated.

**V1** has the best DRY here. `acquireExclusiveLock()` and `acquireSharedLock()` are helper methods on `Store`, reused across `Mutate`, `Query`, and `ForceRebuild`. Stats goes through `Query` (which calls `acquireSharedLock`); rebuild calls `acquireExclusiveLock` directly.

**V2** duplicates the full lock boilerplate -- `flock.New()`, `context.WithTimeout()`, `TryLockContext()` -- in `Mutate`, `Query`, AND `ForceRebuild`. Three copies of essentially the same pattern.

**V3** uses a stored `s.flock` but still duplicates the `TryLockContext` + error handling in all three methods. The lock creation is centralized (one `flock.New()` in `NewStore`), but the acquisition ceremony is not.

### Formatter usage across stats and rebuild

Stats and rebuild exercise different parts of the Formatter interface: stats uses `FormatStats()` (the most complex formatter method, producing three output formats with nested data) while rebuild uses `FormatMessage()` (a simple string-to-format pass-through). This reveals formatter design differences:

**V1 and V2** both use `Formatter.FormatFoo(io.Writer, data)` -- the formatter writes directly to a writer. The caller's responsibility is minimal.

**V3** uses `Formatter.FormatFoo(data) string` -- the formatter returns a string, and the caller writes it via `fmt.Fprint(a.Stdout, ...)`. This adds a step for every call site but gives the caller control over the write target.

For rebuild, this distinction is trivial. For stats, it matters more because `FormatStats` produces multi-line output. V3's approach means the entire stats output is buffered in a string before being written, while V1/V2 stream it to the writer. For the small output volumes involved, this is not a performance concern, but it is an architectural divergence.

## Code Quality Patterns

### Error handling consistency

Both stats and rebuild wrap errors with context, but the wrapping style differs:

| Pattern | V1 | V2 | V3 |
|---------|-----|-----|-----|
| Stats query errors | `fmt.Errorf("querying status counts: %w", err)` | `fmt.Errorf("failed to query status counts: %w", err)` | Bare `return nil, err` from `queryStats` |
| Rebuild errors | 6 distinct wrapped errors | 5 distinct wrapped errors | Minimal wrapping, delegates to helpers |
| Error prefix style | Present tense gerund ("querying...", "closing...") | Past tense ("failed to...") | Mixed / minimal |

V3's `queryStats` function returns bare errors, which means a stats failure would produce something like `sql: no rows in result set` with no indication that it came from the stats query. V1 and V2 would produce `querying status counts: sql: no rows in result set` -- immediately locatable.

### SQL patterns: GROUP BY reuse

Both stats and rebuild touch SQLite, but in opposite directions: stats reads aggregated data; rebuild writes raw data. There is no shared SQL between the two tasks. However, stats internally runs 4 separate queries (status GROUP BY, priority GROUP BY, ready COUNT, blocked COUNT), and the question is whether these could be consolidated.

All three versions run these as independent queries within a single `store.Query()` transaction. None attempt to combine them into a single query with window functions or CTEs. This is a reasonable design choice -- individual queries are simpler to understand and debug -- but it means 4 round-trips to SQLite per stats call.

### DRY between stats and rebuild: shared infrastructure

| Shared Component | V1 | V2 | V3 |
|-----------------|-----|-----|-----|
| Lock helpers | Shared `acquireExclusiveLock`/`acquireSharedLock` | Duplicated inline | Shared `s.flock`, duplicated acquisition |
| Store open helper | `a.openStore()` used by both | `a.newStore()` used by both | No shared helper; both call `storage.NewStore()` directly |
| Verbose logger wiring | Centralized in `openStore` | Centralized in `newStore` | No wiring needed (CLI-level logging) |
| Formatter interface | Same `Formatter` interface, different methods | Same `Formatter` interface, different methods | Same `Formatter` interface, different methods |
| Cache lifecycle | Stateful (close/reopen in rebuild) | Per-operation (local in both) | Per-operation (local in both) |

V2's `newStore` helper is the most effective shared infrastructure between stats and rebuild. One call handles store creation, logger wiring, and error wrapping. V3 has no equivalent -- each command directly calls `storage.NewStore()` and has no logger to inject.

## Test Coverage Analysis

### Aggregate test counts

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Stats tests | 8 | 8 | 8 |
| Rebuild tests | 7 | 8 | 8 |
| **Total tests** | **15** | **16** | **16** |
| Stats test LOC | 140 | 425 | 388 |
| Rebuild test LOC | 121 | 341 | 294 |
| **Total test LOC** | **261** | **766** | **682** |
| Test-to-impl ratio (stats) | 1.3:1 | 4.1:1 | 3.2:1 |
| Test-to-impl ratio (rebuild) | 1.7:1 | 4.0:1 | 3.5:1 |

V2 has nearly 3x the test code of V1. V3 is between them but closer to V2.

### Cross-task test patterns

**Test setup consistency:** V1 uses CLI-driven setup for stats (creating tasks via `createTask()`) but the same pattern for rebuild. V2 uses JSONL fixture injection for both. V3 uses structured fixture injection for both. Within each version, the setup approach is consistent across tasks, which is good for maintainability.

**Assertion strength gradient:** Both V2 and V3 use JSON parsing for stats verification and direct SQLite queries for rebuild verification. V1 uses substring matching for stats and CLI output checking for rebuild. The assertion strategy is consistent within each version across the two tasks.

**Coverage gaps that span tasks:**
- V1 is missing hash verification in rebuild AND does not verify JSON nesting in stats. Both are structural correctness checks -- V1 consistently trusts the happy path.
- No version tests the interaction between stats and rebuild (e.g., run stats, corrupt cache, rebuild, run stats again and verify counts match).
- No version tests stats immediately after a rebuild with zero tasks to verify the freshly-built cache produces correct zero-counts.

### Unique test contributions per version

| Unique Test | Version | Task |
|-------------|---------|------|
| Dependency preservation after rebuild | V2 | 5-2 |
| SHA256 hash length validation (64 chars) | V3 | 5-2 |
| All 6 verbose step messages checked | V3 | 5-2 |
| `workflow` JSON key separate from `by_status` | V2 | 5-1 |
| Right-alignment pattern check (` 3`) | V2 | 5-1 |

## Phase Verdict

**V2 is the clear phase winner.**

The evidence spans both tasks and their integration:

**1. Spec compliance across the phase.** V2 is the only version where both tasks fully comply with the specification. Stats produces the correct JSON structure with separate `by_status` and `workflow` keys. Rebuild has the correct verbose logging granularity. V1 fails on JSON nesting (flat stats output) and has sparse verbose logging. V3 fails on JSON structure (merges workflow into by_status) and has architecturally misleading verbose logging.

**2. Cross-task infrastructure is strongest.** V2's `newStore` helper, `Logger` interface, and per-operation cache model create the cleanest shared infrastructure between stats and rebuild. Both commands benefit from the same store creation, logger wiring, and cache lifecycle patterns. The interaction between Phase 4's verbose architecture and Phase 5's commands is most correct in V2 because verbose messages fire from inside the store, at the moment each step actually occurs.

**3. SQL reuse pipeline is complete.** V2's stats command reuses `readyWhere`/`blockedWhere` from `list.go`, maintaining the Phase 3 -> Phase 5 query consistency requirement. V3 also achieves this (via `ReadyCondition`/`BlockedCondition`), but V2's approach is simpler -- the constants live in the same file (`list.go`) where they are also used for list queries, making the DRY relationship immediately visible.

**4. Test coverage dominates.** V2 has 766 total test LOC across the phase vs V1's 261 and V3's 682. More importantly, V2's tests are the strongest structurally: JSON-parsed assertions for stats, direct SQLite queries for rebuild, dependency preservation checks, and full JSON key validation including the `workflow` nested object.

**5. V2's weaknesses are inherited, not introduced.** The `interface{}` formatter parameter (losing type safety) and the duplicated lock boilerplate are pre-existing architectural choices from earlier phases, not decisions introduced in Phase 5. Within Phase 5's scope, V2 makes no new architectural mistakes.

**Ranking: V2 > V3 > V1.**

V3 is a solid second -- its exported SQL constants and extracted `queryStats` function show good separation of concerns, and its test suite is nearly as thorough as V2's. But V3's JSON spec deviation (no `workflow` key) and the fundamental flaw of CLI-level verbose logging (messages that describe store internals but fire before the store operation runs) keep it behind V2.

V1 is clearly third. Its inline SQL duplication, flat JSON structure, stateful cache lifecycle, and weak substring-only test assertions represent the lowest quality across every dimension measured.
