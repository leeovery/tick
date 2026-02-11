# Phase 5: Stats & Cache

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 5-1 (stats) | V6 | Moderate | V6 tests verify actual values (TOON data rows, Pretty alignment, JSON correctness); V5 tests only check structure/headers |
| 5-2 (rebuild) | V5 | Large | V5 places Rebuild() on Store, reusing acquireExclusive(); V6 inlined storage internals in CLI then had to refactor later |

## Cross-Task Architecture Analysis

### The Store Boundary: Opposite Trajectories

The defining cross-task pattern in Phase 5 is how stats and rebuild interact with the Store -- and the two implementations diverge fundamentally on where the boundary sits.

**V5 is consistent across both tasks.** Stats uses `store.Query(func(db *sql.DB) error { ... })` to run SQL inside the Store's shared-lock scope. Rebuild uses `store.Rebuild()`, a new method that runs inside the Store's exclusive-lock scope. Both tasks go through the same `Store` API; the CLI handlers (`runStats` at 46 lines, `runRebuild` at 33 lines) contain zero storage implementation details. The import graph is clean: `cli -> engine -> cache/storage`.

**V6 had an internal contradiction.** Stats used `store.Query(func(db *sql.DB) error { ... })` -- going through the Store like V5. But the original rebuild implementation bypassed the Store entirely, directly creating a `flock.Flock`, calling `os.Remove`, reading JSONL with `storage.ParseJSONL`, and opening a cache with `storage.OpenCache` -- all in the CLI layer. This meant:
- Stats: CLI -> Store -> SQLite (correct boundary)
- Rebuild (original): CLI -> flock + os + storage.ParseJSONL + storage.OpenCache (boundary violation)

V6 recognized this inconsistency and refactored `rebuild.go` to delegate to `store.Rebuild()` in a later commit (dce7d58), producing the clean 30-line version visible in the current worktree. But the fact that the original implementation needed a corrective refactoring reveals that the architecture wasn't as internalized during Phase 5 authoring.

### Query Reuse: Compile-Time vs Runtime vs Arithmetic

The stats command's ready/blocked counts must reuse Phase 3 query semantics. The two implementations take three different approaches across the ready and blocked dimensions:

| | Ready Count | Blocked Count |
|---|---|---|
| **V5** | Compile-time const concat: `StatsReadyCountQuery = "SELECT COUNT(*) ... WHERE " + readyWhereClause` | Compile-time const concat: `StatsBlockedCountQuery = "SELECT COUNT(*) ... WHERE " + blockedWhereClause` |
| **V6** | Runtime function call: `"SELECT COUNT(*) ... WHERE " + ReadyWhereClause()` | Arithmetic derivation: `stats.Blocked = stats.Open - stats.Ready` |

V5's approach is the strongest DRY guarantee. The `readyWhereClause` and `blockedWhereClause` are Go `const` string fragments shared between `ReadyQuery` (used by `tick ready`), `BlockedQuery` (used by `tick blocked`), and the stats count queries. If the WHERE clause changes in `ready.go`, the stats query automatically changes. This is enforced at compile time -- no possibility of divergence.

V6's `ReadyWhereClause()` function provides DRY for the ready count, but with an indirection layer: `ReadyConditions()` returns `[]string`, and `ReadyWhereClause()` joins them with `AND`. This is more composable (the list module uses `ReadyConditions()` directly to append to its own condition slices at `list.go:204`), but the DRY coupling is runtime rather than compile-time.

For the blocked count, V6's `stats.Blocked = stats.Open - stats.Ready` is mathematically correct and elegant, but it completely sidesteps the `BlockedConditions()` function. V6 has a `BlockedConditions()` function available in `query_helpers.go` (line 41) that lists the exact SQL conditions, yet stats doesn't use it. This is the only place in either codebase where a stats dimension diverges from its Phase 3 query definition. If "blocked" ever gained conditions beyond "open AND NOT ready" (e.g., a "stalled" status that's neither ready nor blocked), V6's arithmetic would silently produce wrong results while V5's SQL query would remain correct.

### Store.Rebuild() as a Shared Abstraction

Both implementations end up with `Store.Rebuild()` (V5 from the start, V6 after refactoring). But the method reveals a design difference in the Store type itself.

V5's `Store.Rebuild()` reuses `acquireExclusive()` -- the same lock-acquire-and-return-unlock pattern used by `Mutate()`. The rebuild method slots cleanly into the existing API: `Query()` for reads, `Mutate()` for writes, `Rebuild()` for cache management. All three share the same lock infrastructure.

V6's `Store.Rebuild()` duplicates the lock acquisition code. The `Mutate()` method (line 92) inlines `s.fileLock.TryLockContext(ctx, 50*time.Millisecond)` with its own context/cancel pair. `Rebuild()` (line 146) does the same thing again with its own context/cancel pair. V6 has no `acquireExclusive()` helper -- every method that needs a lock repeats the same ~8 lines. This isn't a Phase 5-specific issue but it becomes visible here because Phase 5 adds a third lock consumer (Rebuild joins Mutate and Query). V5 added Rebuild with zero lock-management code; V6 added 8 more lines of lock boilerplate.

### The "Quiet" Short-Circuit Location

Both stats and rebuild support `--quiet`. The placement of the short-circuit reveals a subtle design difference:

- **Stats V5**: `if ctx.Quiet { return nil }` at line 44 -- before any store operations. No SQL queries run. Efficient.
- **Stats V6**: `if fc.Quiet { return nil }` at line 13 -- same pattern. No queries. Efficient.
- **Rebuild V5**: Quiet check is AFTER `store.Rebuild()` at line 28. Rebuild runs fully; only output is suppressed. Correct -- rebuild has side effects that should execute regardless of output mode.
- **Rebuild V6**: Same -- quiet check is after `store.Rebuild()` at line 24. Correct.

Both are consistent: stats short-circuits early (read-only, no side effects), rebuild runs fully (has side effects). This consistency holds across both implementations.

## Code Quality Patterns

### V5: Decomposition and Named Constants

V5's Phase 5 code is distinctly more decomposed:
- 4 named SQL constants at package scope (`StatsQuery`, `StatsPriorityQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery`)
- 2 helper functions (`queryStatusCounts`, `queryPriorityCounts`)
- Store method (`Rebuild`) with reused `acquireExclusive()`
- CLI handler is 33 lines for rebuild

This decomposition has diminishing returns for stats (the helper functions aren't tested independently), but for rebuild it pays dividends: the CLI handler is the shortest in the entire codebase.

### V6: Inline Density

V6's stats.go packs 4 SQL queries into a single closure at 95 lines. This is readable but dense. The rebuild.go (post-refactoring) is 30 lines -- comparable to V5's 33. Before refactoring, the 81-line inline version had:
- Direct `flock` import in CLI layer (storage implementation leak)
- Hardcoded `5 * time.Second` timeout (V5 uses configurable `s.lockTimeout`)
- Silently discarded `os.Remove` error (V5 checks `!os.IsNotExist(err)`)

The refactored V6 rebuild.go delegates to `store.Rebuild()` which properly uses the Store's configured timeout and handles the cache close/delete/recreate lifecycle. The code quality gap between original and refactored V6 rebuild demonstrates why the Store boundary matters.

### Error Handling Consistency

V5 uses descriptive gerund-form error messages: `"closing cache: %w"`, `"deleting cache: %w"`, `"reading tasks.jsonl: %w"`. V6 uses `"failed to ..."` prefix: `"failed to query total count: %w"`, `"failed to read tasks.jsonl: %w"`. Both are consistent within their own codebases. V5's style is more idiomatic Go (the Go convention is to describe the action that failed, not to prefix with "failed to").

The critical difference is in `os.Remove` for cache deletion during rebuild:
- V5 Store.Rebuild(): `if err := os.Remove(s.cachePath); err != nil && !os.IsNotExist(err) { return 0, fmt.Errorf(...) }` -- correctly ignores "file doesn't exist" but propagates permission errors
- V6 Store.Rebuild(): `os.Remove(s.cachePath)` -- silently discards ALL errors including permission failures

V6's Store.Rebuild() (line 169) still has this issue even after the refactoring. If cache.db is locked by another process or has restrictive permissions, V6 silently continues and likely fails later with a more confusing error.

## Test Coverage Analysis

### Cross-Task Test Pattern Divergence

V5 uses a single global `Run()` function entry point for all tests in both tasks. This is a genuine end-to-end integration test -- every test exercises the full CLI pipeline from argument parsing through output formatting. The cost is boilerplate: each test declares `var stdout, stderr bytes.Buffer` and calls `Run([]string{...}, dir, &stdout, &stderr, false)`.

V6 defines per-command test helpers (`runStats`, `runRebuild`) that construct an `App` struct with injected dependencies. This is cleaner ergonomically and reduces per-test boilerplate, but each helper is functionally equivalent to V5's approach -- both exercise the full pipeline.

### Stats Test Coverage Gap

The task reports correctly identify V5's stats test weaknesses. Viewed at the phase level, the gap is even more striking:

V5's format-specific tests (TOON, Pretty, JSON) are essentially smoke tests -- they verify the output _looks like_ the right format but don't verify correctness. The TOON test checks for header strings only. The Pretty test checks for label strings only (no alignment verification). The JSON test checks for key existence only. If a bug caused all counts to return 0 (except in the "it counts tasks by status correctly" subtest), the format tests would still pass.

V6's format tests are genuine correctness tests: exact TOON data rows (`"  2,1,0,1,0,1,0"`), exact Pretty right-aligned strings (`"Total:        2"`), and JSON value assertions (`parsed["total"] == float64(2)`). A bug in any count would be caught by multiple tests.

### Rebuild Test Coverage Gap

V6's lock contention test (acquiring the lock externally via `flock.New().TryLock()` before running rebuild) is the strongest individual test in this phase. V5's lock test only checks verbose log messages -- it proves the lock was logged but not that concurrent access is actually prevented.

V6 also verifies:
- SHA256 hash length (`len(hash) == 64`) vs V5's bare `hash != ""`
- Exact confirmation message (`stdout == "Cache rebuilt: 3 tasks\n"`) vs V5's `strings.Contains(output, "3")`
- Cache existence after quiet mode (rebuild side effect verified)
- All verbose lines have `"verbose: "` prefix
- Empty JSONL case verifies cache schema is valid (task count = 0 via SQL)

### Combined Test Surface

| Dimension | V5 | V6 |
|-----------|-----|-----|
| Stats subtests | 8 | 8 |
| Rebuild subtests | 8 (including empty JSONL) | 8 (including empty JSONL) |
| Total test lines | 549 (301 + 248) | 665 (343 + 322) |
| Assertion depth: structural only | 3 (TOON, Pretty, JSON format tests) | 0 |
| Assertion depth: value-verified | 5 | 8 |
| Lock contention test | Indirect (verbose log check) | Direct (external lock holder, expects exit code 1) |
| Hash verification | Non-empty check | Non-empty + length = 64 |
| Output precision | `strings.Contains` | Exact string match where applicable |

V6's 116 additional test lines aren't padding -- they're deeper assertions.

## Phase Verdict

**Split decision: V5 architecture, V6 testing.**

At the architecture level, V5 is the clear winner. Its Store boundary is maintained consistently across both tasks. Stats and rebuild both go through the Store API, reusing `acquireExclusive()` and `acquireShared()` for lock management. The compile-time const concatenation for ready/blocked query reuse is the strongest DRY guarantee possible in Go. The CLI handlers are thin and focused. V6's initial rebuild implementation violated its own Store boundary and required a corrective refactoring, even though the final result converged to V5's design.

At the testing level, V6 is the clear winner. Its stats format tests verify actual output correctness (not just structure), its rebuild lock test exercises real lock contention, its hash test verifies SHA256 length, and its output assertions use exact matching. V5's format tests have gaps that could mask real bugs -- particularly the Pretty test that claims to verify right-alignment but doesn't check any alignment at all.

**Overall: V5 edges ahead.** Architecture and query-reuse durability are more consequential than test depth -- tests can be strengthened incrementally, but an incorrect Store boundary causes cascading design debt (as V6's corrective refactoring proves). V5 also avoids the `os.Remove` error swallowing bug that V6 carries even after refactoring. However, V5 would benefit significantly from adopting V6's assertion patterns for format tests.
