# Task tick-core-1-4: Storage engine with file locking

## Task Summary

Build a `Store` type that composes the JSONL reader/writer (tick-core-1-2) and SQLite cache (tick-core-1-3) into a unified storage engine. The Store must use `github.com/gofrs/flock` for shared/exclusive file locking with a 5-second timeout. It exposes two flows: `Mutate` (exclusive lock, read JSONL, freshness check, apply mutation, atomic write, update cache, unlock) and `Query` (shared lock, read JSONL, freshness check, query SQLite, unlock). JSONL is the source of truth -- if JSONL write succeeds but SQLite update fails, log a warning and return success.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|---|---|---|
| Store composes JSONL reader/writer and SQLite cache into a single interface | PASS -- `engine.Store` holds `*cache.Cache` + delegates to `storage.WriteTasks`/`ParseTasks` | PASS -- `storage.Store` holds `*Cache` + delegates to `MarshalJSONL`/`WriteJSONLRaw`/`ParseJSONL` |
| Write operations acquire exclusive lock, read operations acquire shared lock | PASS -- `acquireExclusive()` calls `TryLockContext`, `acquireShared()` calls `TryRLockContext` | PASS -- inline `TryLockContext` in `Mutate`, `TryRLockContext` in `Query` |
| Lock timeout of 5 seconds returns descriptive error message | PASS -- `"Could not acquire lock on .tick/lock - another process may be using tick"` | **PARTIAL** -- `"could not acquire lock on .tick/lock - another process may be using tick"` (lowercase "could") -- spec says "Could" with capital C |
| Concurrent shared locks allowed (multiple readers) | PASS -- tested with 5 goroutines using atomic counters to prove concurrency | PASS -- tested with flock directly (two shared locks), but not through Store's `Query` |
| Exclusive lock blocks all other access (readers and writers) | PASS -- tested both directions (exclusive blocks shared, shared blocks exclusive) | PASS -- tested both directions with raw flock |
| Write flow executes full sequence: lock -> read -> freshness -> mutate -> atomic write -> update cache -> unlock | PASS -- verified end-to-end with JSONL + SQLite assertions | PASS -- verified end-to-end with JSONL file re-read + SQLite query |
| Read flow executes full sequence: lock -> read -> freshness -> query -> unlock | PASS -- verified with SELECT COUNT(*) | PASS -- verified with SELECT COUNT(*) + title check |
| Lock is always released, even on errors or panics (defer pattern) | PASS -- `defer unlock()` in both flows; tested with error-returning callbacks | PASS -- `defer func() { _ = s.fileLock.Unlock() }()` in both flows; tested with error-returning callbacks |
| JSONL write success + SQLite failure = log warning, return success | PASS -- closes `s.cache` inside mutation callback to force failure; verifies JSONL has new task | PASS -- replaces cache.db with a directory inside mutation callback; verifies JSONL + subsequent self-heal |
| Stale cache is rebuilt before mutation or query executes | PASS -- indirectly tested (fresh Store has no cache, so first access rebuilds) | PASS -- explicitly tested: builds cache, externally modifies JSONL, then verifies mutation/query sees new data |

## Implementation Comparison

### Approach

**V5** places the Store in a dedicated `internal/engine` package, separate from `internal/storage` (JSONL) and `internal/cache` (SQLite). It adds a `ParseTasks` function to `internal/storage/jsonl.go` (26 lines) and creates `internal/engine/store.go` (235 lines in the diff, 310 in the full worktree file including verbose logging and Rebuild method).

Key design decisions:
- Creates a new flock instance on **every lock acquisition** (`fl := flock.New(s.lockPath)` inside `acquireExclusive`/`acquireShared`). This means each call gets a fresh file descriptor.
- Returns an `unlock func()` from the lock helpers, which the caller defers. This pattern cleanly encapsulates both `fl.Unlock()` and context `cancel()`.
- Cache update after mutation **re-reads the JSONL file from disk** to get the bytes for hash computation:

```go
// store.go:300-309 (V5)
func (s *Store) updateCache(tasks []task.Task) {
    newJSONLData, err := os.ReadFile(s.jsonlPath)
    if err != nil {
        log.Printf("warning: could not read tasks.jsonl after write for cache update: %v", err)
        return
    }
    if err := s.cache.Rebuild(tasks, newJSONLData); err != nil {
        log.Printf("warning: SQLite cache update failed after successful JSONL write: %v", err)
    }
}
```

- Opens the cache eagerly in `NewStore` via `cache.New(cachePath)`.
- Includes a separate `VerboseLogger` type in `verbose.go` (37 lines) with `Log` and `Logf` methods, writing to an `io.Writer`.
- Adds a `Rebuild()` method (lines 166-210) not required by the task spec.

**V6** places the Store in the same `internal/storage` package alongside the JSONL and cache code. It refactors `jsonl.go` significantly -- extracting `MarshalJSONL`, `WriteJSONLRaw`, `writeAtomic`, `ParseJSONL` as public functions and refactoring `WriteJSONL` and `ReadJSONL` to delegate to them.

Key design decisions:
- Uses a **single shared flock instance** (`s.fileLock = flock.New(...)` set once in `NewStore`) reused across all lock acquisitions. This means the flock struct is shared state.
- Inlines the lock acquisition directly in `Mutate` and `Query` rather than helper functions -- slightly more repetitive but avoids the unlock-function allocation.
- Cache update after mutation **avoids re-reading the file** by marshaling tasks to bytes first, then using the same bytes for both atomic write and cache rebuild:

```go
// store.go:118-138 (V6)
newRawJSONL, err := MarshalJSONL(mutated)
if err != nil {
    return fmt.Errorf("failed to marshal tasks: %w", err)
}
if err := WriteJSONLRaw(s.jsonlPath, newRawJSONL); err != nil {
    return fmt.Errorf("failed to write tasks.jsonl: %w", err)
}
if err := s.cache.Rebuild(mutated, newRawJSONL); err != nil {
    log.Printf("warning: failed to update cache after write: %v", err)
    s.cache.Close()
    s.cache = nil
    return nil
}
```

- Opens the cache **lazily** in `ensureFresh` on first use, with corruption recovery (delete + recreate).
- Uses a simple `verboseLog func(msg string)` field instead of a separate VerboseLogger type.
- Also adds a `Rebuild()` method (lines 146-198) not required by the task spec.

### Code Quality

**Go Idioms & Naming**

| Aspect | V5 | V6 |
|---|---|---|
| Package location | `internal/engine` -- clean separation of concerns, but adds a new package the spec did not require | `internal/storage` -- co-located with JSONL and cache code, reduces import graph |
| Option type name | `Option` | `StoreOption` -- more specific, avoids collision with potential future option types in the same package |
| Lock error constant | `lockTimeoutMsg` | `lockErrMsg` -- shorter but less descriptive |
| Error message casing | `"Could not acquire lock..."` -- matches spec exactly | `"could not acquire lock..."` -- lowercase, deviates from spec |
| Lock helper pattern | Returns `(unlock func(), err error)` -- idiomatic Go pattern for resource cleanup | Inline `defer func() { _ = s.fileLock.Unlock() }()` -- standard but repeated |

**Error Handling**

V5 uses `fmt.Errorf("%s", lockTimeoutMsg)` for the lock error, which is slightly odd (could be `errors.New` or `fmt.Errorf` with `%w`). V6 uses `errors.New(lockErrMsg)` which is cleaner.

V5's `ensureFresh` (line 276-295) logs a warning on `IsFresh` error and continues with `fresh = false`, which is a soft recovery. V6's `ensureFresh` (line 247-289) takes a harder approach: on `IsFresh` error it closes the cache, deletes the file, and recreates it. V6's approach is more robust against persistent corruption.

V6 also handles cache corruption during open (`ensureFresh` lines 249-261):
```go
if s.cache == nil {
    cache, err := OpenCache(s.cachePath)
    if err != nil {
        log.Printf("warning: cache open failed, recreating: %v", err)
        os.Remove(s.cachePath)
        cache, err = OpenCache(s.cachePath)
        if err != nil {
            return fmt.Errorf("failed to recreate cache: %w", err)
        }
    }
    s.cache = cache
}
```

**DRY**

V5 has some code repetition between `acquireExclusive` and `acquireShared` (nearly identical 12-line functions). V6 has similar repetition between the lock blocks in `Mutate` and `Query`, plus the `Rebuild` method has yet another copy. Neither version extracts a generic lock helper.

V6's refactoring of `jsonl.go` to extract `MarshalJSONL` + `WriteJSONLRaw` + `writeAtomic` is good decomposition that enables the marshal-once-write-once pattern in `Mutate`.

**Type Safety**

Both versions use `func(tasks []task.Task) ([]task.Task, error)` for mutation and `func(db *sql.DB) error` for query, matching the spec signatures.

V5 `NewStore` validates the tick directory exists AND is a directory (two checks):
```go
info, err := os.Stat(tickDir)
if err != nil {
    return nil, fmt.Errorf("tick directory does not exist: %w", err)
}
if !info.IsDir() {
    return nil, fmt.Errorf("tick path is not a directory: %s", tickDir)
}
```

V6 only checks if `tasks.jsonl` exists via `os.IsNotExist`, skipping the directory validation entirely.

### Test Quality

#### V5 Test Functions and Subtests

**`internal/engine/store_test.go`** (559 lines) -- single top-level `TestStore` with subtests:

1. `TestStore/"it acquires exclusive lock for write operations"` (lines 79-119) -- Goroutine-based: runs Mutate in background, signals from inside callback, tries to acquire exclusive lock from outside. Verifies lock is held.
2. `TestStore/"it acquires shared lock for read operations"` (lines 121-169) -- Same goroutine pattern: runs Query in background, verifies shared lock is compatible (another shared succeeds) and exclusive lock is blocked.
3. `TestStore/"it returns error after lock timeout"` (lines 171-198) -- Holds external exclusive lock, creates Store with 100ms timeout, verifies Mutate returns exact error message.
4. `TestStore/"it allows concurrent shared locks (multiple readers)"` (lines 200-247) -- 5 goroutines run concurrent Query calls, tracks max concurrent readers with atomics, asserts >= 2 concurrent.
5. `TestStore/"it blocks shared lock while exclusive lock is held"` (lines 249-275) -- Holds external exclusive lock, verifies Query times out.
6. `TestStore/"it blocks exclusive lock while shared lock is held"` (lines 277-303) -- Holds external shared lock, verifies Mutate times out.
7. `TestStore/"it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock"` (lines 305-357) -- Mutates pre-populated tasks, verifies JSONL + SQLite via follow-up Query.
8. `TestStore/"it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock"` (lines 359-379) -- Queries pre-populated tasks, verifies count.
9. `TestStore/"it releases lock on mutation function error (no leak)"` (lines 381-408) -- Returns error from mutation callback, then acquires lock externally to prove it was released.
10. `TestStore/"it releases lock on query function error (no leak)"` (lines 410-437) -- Returns error from query callback, then acquires lock externally.
11. `TestStore/"it continues when JSONL write succeeds but SQLite update fails"` (lines 439-479) -- Closes `s.cache` inside mutation callback, verifies Mutate succeeds and JSONL contains new task.
12. `TestStore/"it rebuilds stale cache during write before applying mutation"` (lines 481-502) -- Fresh Store with no prior cache, Mutate forces rebuild, verifies tasks passed to callback.
13. `TestStore/"it rebuilds stale cache during read before running query"` (lines 504-525) -- Fresh Store, Query forces rebuild, verifies count via SQLite.
14. `TestStore/"it surfaces correct error message on lock timeout"` (lines 527-558) -- Tests both Mutate and Query timeout error message.

**`internal/engine/store_verbose_test.go`** (167 lines):

15. `TestStoreVerbose/"it logs lock/cache/hash/write operations when verbose is on"` (lines 13-49)
16. `TestStoreVerbose/"it logs lock/cache/hash operations during query when verbose is on"` (lines 51-83)
17. `TestStoreVerbose/"it writes nothing when verbose is off"` (lines 85-108)
18. `TestStoreVerbose/"it logs hash comparison on freshness check"` (lines 110-138)
19. `TestStoreVerbose/"it prefixes all lines with verbose:"` (lines 140-167)

**`internal/engine/verbose_test.go`** (59 lines):

20. `TestVerboseLogger/"it writes to writer with verbose prefix when verbose is true"` (lines 9-19)
21. `TestVerboseLogger/"it writes nothing when verbose is false"` (lines 21-29)
22. `TestVerboseLogger/"it supports formatted output"` (lines 31-41)
23. `TestVerboseLogger/"it writes nothing for formatted output when verbose is false"` (lines 43-51)
24. `TestVerboseLogger/"it handles nil writer gracefully when verbose is false"` (lines 53-58)

**`internal/storage/jsonl_test.go`** (added tests only):

25. `TestParseTasks/"it parses tasks from a byte slice of JSONL content"` (lines 519-543)
26. `TestParseTasks/"it returns empty list for empty byte slice"` (lines 545-553)
27. `TestParseTasks/"it skips empty lines in byte slice"` (lines 555-568)
28. `TestParseTasks/"it returns error for invalid JSON in byte slice"` (lines 570-576)

Total V5 subtests for this task: 28 (14 store + 5 verbose store + 5 verbose logger + 4 ParseTasks)

#### V6 Test Functions and Subtests

**`internal/storage/store_test.go`** (1108 lines):

1. `TestStoreMutate/"it acquires exclusive lock for write operations"` (lines 42-61) -- Checks lock file exists during mutation. Does NOT verify the lock is actually exclusive (only checks file existence via `os.Stat`).
2. `TestStoreQuery/"it acquires shared lock for read operations"` (lines 63-89) -- Same pattern: checks lock file exists during query. Does NOT verify the lock is shared.
3. `TestStoreLockTimeout/"it returns error after lock timeout"` (lines 91-120) -- Holds external exclusive lock, verifies Mutate error message. Also asserts mutation function was not called.
4. `TestStoreLockTimeout/"it surfaces correct error message on lock timeout"` (lines 122-153) -- Tests Query error message (separate from #3 which tests Mutate).
5. `TestStoreConcurrentLocks/"it allows concurrent shared locks (multiple readers)"` (lines 155-180) -- Uses raw flock (NOT Store.Query), acquires two shared locks directly. Does not prove concurrent Store queries work.
6. `TestStoreConcurrentLocks/"it blocks shared lock while exclusive lock is held"` (lines 182-204) -- Uses raw flock TryRLock, not Store.
7. `TestStoreConcurrentLocks/"it blocks exclusive lock while shared lock is held"` (lines 206-228) -- Uses raw flock TryLock, not Store.
8. `TestStoreWriteFlow/"it executes full write flow:..."` (lines 230-308) -- Mutate adds a task, verifies JSONL via ReadJSONL + SQLite via follow-up Query.
9. `TestStoreReadFlow/"it executes full read flow:..."` (lines 310-362) -- Query pre-populated tasks, verifies count + title.
10. `TestStoreLockRelease/"it releases lock on mutation function error (no leak)"` (lines 364-389) -- Returns error from mutation, then runs another Mutate to prove lock released.
11. `TestStoreLockRelease/"it releases lock on query function error (no leak)"` (lines 391-414) -- Returns error from query, then runs another Query.
12. `TestStoreSQLiteFailure/"it continues when JSONL write succeeds but SQLite update fails"` (lines 416-494) -- Replaces cache.db with a directory inside mutation to force failure. Verifies JSONL has new task. Also verifies subsequent Query self-heals.
13. `TestStoreRebuild/"it rebuilds cache from JSONL and returns task count"` (lines 496-598) -- Tests the Rebuild method: verifies returned count, verifies SQLite has correct data, verifies hash is stored.
14. `TestStoreRebuild/"it acquires exclusive lock during rebuild"` (lines 600-633) -- Tests Rebuild lock timeout.
15. `TestStoreRebuild/"it logs verbose messages during rebuild"` (lines 634-676) -- Tests verbose output during Rebuild.
16. `TestStoreRebuild/"it handles empty JSONL returning 0 tasks"` (lines 678-695) -- Tests Rebuild on empty file.
17. `TestStoreCacheFreshnessRecovery/"it rebuilds automatically when cache.db is missing"` (lines 697-736) -- Deletes cache.db before first use, verifies Query works.
18. `TestStoreCacheFreshnessRecovery/"it deletes and rebuilds when cache.db is corrupted"` (lines 738-778) -- Writes garbage to cache.db, verifies Query recovers.
19. `TestStoreCacheFreshnessRecovery/"it detects stale cache via hash mismatch and rebuilds"` (lines 780-852) -- Primes cache, externally modifies JSONL, verifies Query sees updated data.
20. `TestStoreCacheFreshnessRecovery/"it handles freshness check errors from corrupted metadata"` (lines 854-921) -- Corrupts metadata table schema, verifies Query recovers.
21. `TestStoreStaleCacheRebuild/"it rebuilds stale cache during write before applying mutation"` (lines 924-989) -- Primes cache, externally modifies JSONL, verifies Mutate sees updated data with modified title.
22. `TestStoreStaleCacheRebuild/"it rebuilds stale cache during read before running query"` (lines 991-1063) -- Same pattern for Query.

**`internal/storage/jsonl_test.go`** (added tests only):

23. `TestParseJSONL/"it parses tasks from raw JSONL bytes"` (lines 426-447)
24. `TestParseJSONL/"it returns empty list for empty bytes"` (lines 449-457)
25. `TestParseJSONL/"it skips empty lines in byte input"` (lines 459-472)
26. `TestParseJSONL/"it returns error for invalid JSON in bytes"` (lines 474-483)
27. `TestMarshalJSONL/"it serializes tasks to JSONL bytes"` (lines 486-524)
28. `TestMarshalJSONL/"it returns empty bytes for empty task list"` (lines 526-534)
29. `TestMarshalJSONL/"it round-trips through ParseJSONL"` (lines 536-587)

Total V6 subtests for this task: 29 (22 store + 7 JSONL)

#### Test Quality Diff

| Aspect | V5 | V6 |
|---|---|---|
| Lock verification method | Uses goroutines + channels to verify lock is **actually held** during callbacks. Tries to acquire external lock while inside mutation/query -- strong proof. | Only checks lock **file exists** via `os.Stat` -- weak proof. A file existing does not prove a lock is held. |
| Concurrent readers test | 5 goroutines running `Store.Query` concurrently with atomic max-reader tracking -- tests the real Store path | Tests raw `flock.TryRLock` -- does not prove Store.Query supports concurrent readers |
| Lock release tests | Acquires external flock after error to prove lock was released | Runs another Store operation after error -- indirect proof, could pass even with a leaked-then-timed-out lock |
| Stale cache tests | Indirectly tested (fresh Store has no cache, first access rebuilds) -- does not simulate external JSONL modification between operations | **Explicitly tested**: primes cache, then externally modifies JSONL, then verifies mutation/query sees new data. This is the correct test. |
| Cache corruption recovery | Not tested | Extensively tested: missing cache.db, corrupted cache.db (garbage bytes), corrupted metadata table schema, all with recovery verification |
| SQLite failure + self-heal | Tests JSONL persistence after SQLite failure but does NOT verify self-heal on next read | Tests both: JSONL persistence after failure AND subsequent Query self-heal |
| Rebuild method tests | Not tested (Rebuild exists in worktree but no tests in this task's diff) | 4 tests covering Rebuild: count, lock, verbose, empty file |
| Verbose/debug tests | 10 tests across 2 files (verbose_test.go + store_verbose_test.go) | 1 test (verbose during Rebuild) |

**Edge case coverage:**

V5 covers all 14 spec test cases directly. V6 covers all 14 plus adds cache corruption recovery (4 extra tests) and Rebuild tests (4 extra). However, V6's lock-related tests are weaker in their assertions about actual locking behavior.

### Skill Compliance

| Constraint | V5 | V6 |
|---|---|---|
| Use gofmt and golangci-lint on all code | PASS -- code is properly formatted | PASS -- code is properly formatted |
| Add context.Context to all blocking operations | PASS -- `context.WithTimeout` for lock acquisition | PASS -- `context.WithTimeout` for lock acquisition |
| Handle all errors explicitly | PASS -- all errors checked or logged | PASS -- all errors checked or logged |
| Write table-driven tests with subtests | **PARTIAL** -- uses subtests (`t.Run`) but not table-driven pattern | **PARTIAL** -- uses subtests (`t.Run`) but not table-driven pattern |
| Document all exported functions, types, and packages | PASS -- package doc, all exported symbols documented | PASS -- package doc, all exported symbols documented |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS -- all error wrapping uses `%w` | PASS -- all error wrapping uses `%w` |
| No panic for normal error handling | PASS | PASS |
| No goroutines without lifecycle management | PASS -- goroutines only in tests | PASS -- no goroutines in implementation |
| No hardcoded configuration | PASS -- `WithLockTimeout` option | PASS -- `WithLockTimeout` option |
| No ignored errors without justification | PASS -- `_ = fl.Unlock()` justified (lock release in defer) | PASS -- `_ = s.fileLock.Unlock()` justified |

### Spec-vs-Convention Conflicts

1. **Lock timeout error message casing**: The spec requires `"Could not acquire lock on .tick/lock - another process may be using tick"` (capital "C"). V5 matches exactly. V6 uses lowercase `"could"`.

2. **Package placement**: The spec says "Define a Store type (or equivalent)" without prescribing package location. V5 creates `internal/engine`, V6 co-locates in `internal/storage`. Neither violates the spec, but V6's choice means the Store shares a package with its dependencies (JSONL, cache), reducing encapsulation.

3. **Flock instance lifecycle**: The spec says "create flock instance on `.tick/lock` file path". V5 creates a new instance per operation (stateless, no sharing issues). V6 creates one instance in `NewStore` and reuses it. The `gofrs/flock` documentation states the Flock type is safe for concurrent use, so V6's approach is valid, but V5's approach is safer against potential state leakage between operations.

4. **NewStore validation**: The spec says `NewStore(tickDir string)` should validate the `.tick/` directory exists and contains `tasks.jsonl`. V5 validates both (directory existence + is-a-directory + tasks.jsonl existence). V6 only validates tasks.jsonl existence via `os.IsNotExist`.

5. **Cache opening**: The spec does not prescribe eager vs. lazy cache initialization. V5 opens the cache eagerly in `NewStore` (fails fast if cache.db is uncreateable). V6 opens lazily on first use (defers failure to first operation).

6. **Verbose logging**: Not in the spec for this task. Both versions add it. V5 creates a full `VerboseLogger` type with 37 lines of code + 59 lines of unit tests + 167 lines of integration tests. V6 uses a simple `func(msg string)` callback -- far more lightweight.

7. **Rebuild method**: Not in the spec for this task. Both versions add it. This is scope creep, though harmless.

## Diff Stats

| Metric | V5 | V6 |
|---|---|---|
| Files changed (internal/) | 4 | 4 |
| Lines added (internal/) | 881 | 1,054 |
| Lines removed (internal/) | 0 | 16 |
| store.go lines (full file) | 310 | 290 |
| store_test.go lines (full file) | 559 | 1,108 |
| Additional files | verbose.go (37), verbose_test.go (59), store_verbose_test.go (167) | -- |
| New JSONL functions added | `ParseTasks` (26 lines) | `ParseJSONL`, `MarshalJSONL`, `WriteJSONLRaw`, `writeAtomic` (refactored 69 lines net) |
| Total new test subtests | 28 | 29 |
| Spec-required Store tests | 14/14 | 14/14 |
| Extra tests beyond spec | 14 (verbose, ParseTasks) | 15 (Rebuild, corruption recovery, MarshalJSONL) |

## Verdict

**V6 is the stronger implementation**, with important caveats.

**V6 advantages:**
- The marshal-once pattern (`MarshalJSONL` -> `WriteJSONLRaw` + `cache.Rebuild`) avoids an unnecessary file re-read after writing, which is both more efficient and more correct (no TOCTOU gap between write and re-read for cache update).
- Lazy cache initialization with corruption recovery (delete + recreate) is significantly more robust than V5's eager open with no recovery.
- The `ensureFresh` corruption handling (close, delete, reopen on `IsFresh` failure) is production-grade resilience that V5 lacks.
- The refactoring of `jsonl.go` into `MarshalJSONL`/`ParseJSONL`/`writeAtomic` is good decomposition that creates a cleaner API for the Store.
- The stale cache tests explicitly simulate external JSONL modification, which is the real-world scenario. V5 only tests the implicit "first access triggers rebuild" path.
- Cache corruption recovery tests (4 extra) cover a critical edge case the spec mentions ("next read self-heals").
- Lighter verbose implementation (`func(msg string)` vs. full `VerboseLogger` type) is appropriate given verbose logging is not in scope.

**V5 advantages:**
- Lock verification tests are **significantly stronger**: using goroutines + channels to prove the lock is actually held during callbacks, and using external flock acquisition attempts to verify exclusivity. V6's tests merely check the lock file exists (`os.Stat`) which proves nothing about locking semantics.
- Concurrent reader test actually runs 5 `Store.Query` calls concurrently -- V6 only tests raw flock. This is a meaningful gap because it does not prove Store.Query works under concurrency.
- Matches the spec's error message casing exactly ("Could" vs. V6's "could").
- More thorough `NewStore` validation (checks directory existence, is-a-directory, tasks.jsonl existence).
- The unlock-function pattern (`acquireExclusive() (unlock func(), err error)`) is cleaner Go idiom and avoids the repeated inline lock blocks.
- Creates a new flock per operation, avoiding any potential state sharing issues.

**Net assessment:** V6 wins on implementation correctness (no re-read, lazy init, corruption recovery) and real-world edge case testing (stale cache, corruption). V5 wins on lock testing rigor and spec-literal compliance. The lock testing gap in V6 is concerning but not a correctness bug -- it is a test coverage gap. The casing difference in the error message is a minor spec deviation. If one had to ship one version, V6's production resilience outweighs V5's testing elegance, but V6 should adopt V5's goroutine-based lock tests.
