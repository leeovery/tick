# Task tick-core-1-4: Storage Engine with File Locking

## Task Summary

This task builds a unified `Store` type that composes the JSONL reader/writer (tick-core-1-2) and SQLite cache (tick-core-1-3) into a single orchestration layer with file-based locking for concurrent access safety. The Store provides two operations:

- **Mutate (write flow)**: acquire exclusive lock on `.tick/lock` (5s timeout) -> read `tasks.jsonl` + compute hash -> check SQLite freshness (rebuild if stale) -> pass `[]Task` to mutation function -> atomic JSONL write -> update SQLite cache -> release lock. If JSONL write succeeds but SQLite update fails, log warning and return success.
- **Query (read flow)**: acquire shared lock on `.tick/lock` (5s timeout) -> read `tasks.jsonl` + compute hash -> check SQLite freshness (rebuild if stale) -> execute query against SQLite -> release lock.

Uses `github.com/gofrs/flock` for shared/exclusive file locking. Lock release must always happen via `defer`. Lock timeout returns error: "Could not acquire lock on .tick/lock - another process may be using tick".

### Acceptance Criteria (from plan)

1. `Store` composes JSONL reader/writer and SQLite cache into a single interface
2. Write operations acquire exclusive lock, read operations acquire shared lock
3. Lock timeout of 5 seconds returns descriptive error message
4. Concurrent shared locks allowed (multiple readers)
5. Exclusive lock blocks all other access (readers and writers)
6. Write flow executes full sequence: lock -> read -> freshness -> mutate -> atomic write -> update cache -> unlock
7. Read flow executes full sequence: lock -> read -> freshness -> query -> unlock
8. Lock is always released, even on errors or panics (defer pattern)
9. JSONL write success + SQLite failure = log warning, return success
10. Stale cache is rebuilt before mutation or query executes

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| 1. Store composes JSONL + SQLite into single interface | PASS - `Store` struct in `internal/storage/store.go` holds `tickDir`, `jsonlPath`, `cachePath`, `lockPath` and delegates to `jsonl` and `sqlite` sub-packages | PASS - `Store` struct in `internal/store/store.go` holds `tickDir`, `jsonlPath`, `dbPath`, `lockPath` and delegates to `task` and `cache` packages |
| 2. Exclusive lock for writes, shared lock for reads | PASS - `Mutate` calls `fl.TryLockContext`, `Query` calls `fl.TryRLockContext` | PASS - Same approach: `Mutate` calls `fl.TryLockContext`, `Query` calls `fl.TryRLockContext` |
| 3. Lock timeout 5s with descriptive error | PASS - `defaultLockTimeout = 5 * time.Second`; error message: `"Could not acquire lock on .tick/lock - another process may be using tick"` (matches spec exactly) | PARTIAL - `lockTimeout = 5 * time.Second`; error message: `"could not acquire lock on %s - another process may be using tick"` — uses lowercase "could" (spec says "Could") and includes full absolute path instead of `.tick/lock` |
| 4. Concurrent shared locks (multiple readers) | PASS - Tested with 5 concurrent goroutines all holding shared locks | PASS - Tested with 5 concurrent goroutines all holding shared locks |
| 5. Exclusive lock blocks all other access | PASS - Tested both directions: shared blocked by exclusive, exclusive blocked by shared | PASS - Tested both directions identically |
| 6. Full write flow sequence | PASS - Mutate: lock -> ReadFile -> ParseTasks -> EnsureFresh -> fn(tasks) -> WriteTasks -> ReadFile (re-read for hash) -> Rebuild -> unlock | PASS - Mutate: lock -> ReadFile -> ReadJSONLFromBytes -> EnsureFresh -> fn(tasks) -> SerializeJSONL -> WriteJSONL -> cache.Open -> Rebuild -> unlock |
| 7. Full read flow sequence | PASS - Query: lock -> ReadFile -> ParseTasks -> EnsureFresh -> fn(cache.DB()) -> unlock | PASS - Query: lock -> ReadFile -> ReadJSONLFromBytes -> EnsureFresh -> cache.Open -> fn(c.DB()) -> unlock |
| 8. Lock always released via defer | PASS - `defer fl.Unlock()` immediately after lock acquisition in both methods | PASS - `defer fl.Unlock()` immediately after lock acquisition in both methods |
| 9. JSONL success + SQLite failure = log warning, return success | PASS - `log.Printf("warning: ...")` + `return nil` after SQLite Rebuild failure; tested with duplicate-ID trick causing PRIMARY KEY violation | PASS - `log.Printf("warning: ...")` + `return nil` after cache Open/Rebuild failure; tested by replacing cache.db file with a directory |
| 10. Stale cache rebuilt before mutation/query | PASS - EnsureFresh called before mutation fn and before query fn; tested by externally modifying JSONL | PASS - EnsureFresh called before mutation fn and before query fn; tested by externally modifying JSONL |

## Implementation Comparison

### Approach

Both versions follow the same overall architectural pattern: a `Store` struct that holds paths and a lock timeout, with `Mutate` and `Query` methods implementing the write and read flows respectively. The key differences lie in package organization, API design of dependencies, and cache lifecycle management.

**Package Layout**

V2 places the store in `internal/storage/store.go` (package `storage`), alongside its existing `internal/storage/jsonl/` and `internal/storage/sqlite/` sub-packages. The store is a sibling file in the parent `storage` package that imports its own sub-packages.

V4 places the store in a new `internal/store/store.go` (package `store`), separate from the JSONL code in `internal/task/` and cache code in `internal/cache/`. This is a cleaner separation where the store is its own standalone package.

**Constructor Design**

V2 provides two constructors:
```go
func NewStore(tickDir string) (*Store, error)
func NewStoreWithTimeout(tickDir string, lockTimeout time.Duration) (*Store, error)
```

V4 provides only one constructor, with tests modifying the timeout directly on the struct field:
```go
func NewStore(tickDir string) (*Store, error)
// Tests do: s.lockTimeout = 50 * time.Millisecond
```

V2's approach is better for production use since `NewStoreWithTimeout` is an exported API. V4's direct field mutation in tests works but would break if `lockTimeout` were unexported or if the Store were behind an interface.

**Cache Lifecycle Management — Key Architectural Difference**

V2's `sqlite.EnsureFresh` returns `(*Cache, error)`, keeping the cache connection open. The Store uses this single cache reference throughout the operation:
```go
// V2 Mutate (lines 77-84)
cache, err := sqlite.EnsureFresh(s.cachePath, tasks, rawContent)
if err != nil {
    return fmt.Errorf("failed to ensure cache freshness: %w", err)
}
defer cache.Close()
// ... later uses cache.Rebuild(modified, newRawContent)
```

V4's `cache.EnsureFresh` returns only `error`, closing its internal connection. The Store must re-open the cache:
```go
// V4 Mutate (lines 93-95)
if err := cache.EnsureFresh(s.dbPath, jsonlData, tasks); err != nil {
    return fmt.Errorf("failed to ensure cache freshness: %w", err)
}
// ... later:
c, err := cache.Open(s.dbPath)  // re-opens cache
if err != nil {
    log.Printf("warning: failed to open cache for update: %v", err)
    return nil
}
defer c.Close()
```

V2's approach is more efficient (single open/close cycle). V4 opens the SQLite database twice per operation: once inside `EnsureFresh` and again for the actual work. However, V4's approach is simpler for the `EnsureFresh` API since the caller doesn't need to manage the cache lifecycle.

**Lock Error Handling**

V2 combines the error and locked-false cases into one condition:
```go
// V2 (line 58)
locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
if err != nil || !locked {
    return fmt.Errorf("Could not acquire lock on .tick/lock - another process may be using tick")
}
```

V4 checks them separately:
```go
// V4 (lines 72-77)
locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
if err != nil {
    return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
}
if !locked {
    return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
}
```

V4's separation is slightly more verbose but equivalent in behavior. However, V2 matches the spec's exact error message (`"Could not acquire lock on .tick/lock"` with capital C and literal `.tick/lock`), while V4 uses lowercase `"could"` and interpolates the full absolute path via `s.lockPath`. Go convention says error messages should start lowercase, so V4 follows Go idioms better, but deviates from the spec's literal text.

**Lock Retry Interval**

V2 uses a 10ms retry interval: `fl.TryLockContext(ctx, 10*time.Millisecond)`
V4 uses a 100ms retry interval: `fl.TryLockContext(ctx, 100*time.Millisecond)`

V2's shorter interval means faster lock acquisition when the lock becomes available, at the cost of slightly more CPU polling. V4's longer interval is gentler on the system but may add up to 100ms latency. Neither is clearly better; both are reasonable.

**Post-Write Hash Computation**

V2 re-reads the JSONL file after writing to compute the hash for the cache update:
```go
// V2 (lines 89-93)
newRawContent, err := os.ReadFile(s.jsonlPath)
if err != nil {
    log.Printf("warning: failed to read tasks.jsonl for cache update: %v", err)
    return nil
}
if err := cache.Rebuild(modified, newRawContent); err != nil {
```

V4 pre-serializes the JSONL data in memory before writing, then uses that same buffer for the cache update:
```go
// V4 (lines 102-105)
newJSONLData, err := task.SerializeJSONL(modified)
if err != nil {
    return fmt.Errorf("failed to serialize tasks: %w", err)
}
// ... WriteJSONL writes to disk ...
// ... later uses newJSONLData for cache.Rebuild
```

V4's approach is genuinely better here — it avoids the extra disk read and guarantees the hash matches exactly what was written. V2's re-read introduces a theoretical TOCTOU race (though unlikely under exclusive lock) and unnecessary I/O.

**Helper Function Additions**

V2 adds `ParseTasks(data []byte)` to `internal/storage/jsonl/jsonl.go` — a new function that parses tasks from raw bytes without opening a file.

V4 adds both `ReadJSONLFromBytes(data []byte)` and `SerializeJSONL(tasks []Task)` to `internal/task/jsonl.go`. The `SerializeJSONL` function enables the in-memory serialization used to avoid the re-read. V4 also refactors `WriteJSONL` to use `SerializeJSONL` internally, reducing code duplication.

### Code Quality

**Go Idioms**

V2 error message: `"Could not acquire lock on .tick/lock - another process may be using tick"` — starts with uppercase, violating Go convention that error strings should not be capitalized (per Go Code Review Comments). However, this is the spec's exact text.

V4 error message: `"could not acquire lock on %s - another process may be using tick"` — follows Go convention with lowercase. Uses format string with `s.lockPath` which is more flexible but produces a different message than the spec.

**Error Handling**

Both versions correctly handle all error paths with appropriate wrapping (`fmt.Errorf("...: %w", err)`). Both log SQLite failures instead of propagating them.

V4 has a subtle issue in the Mutate method: it calls `SerializeJSONL` before `WriteJSONL`, but `WriteJSONL` internally calls `SerializeJSONL` again (since V4 refactored `WriteJSONL` to use `SerializeJSONL`). This means the tasks are serialized twice — once for hash computation and once for writing. A more efficient approach would be a `WriteJSONLBytes` function that accepts pre-serialized data.

V2 avoids this by re-reading from disk, but that has its own cost (the extra I/O).

**DRY / Code Duplication**

V2's `Mutate` and `Query` methods share identical code for lock acquisition, JSONL reading/parsing, and freshness checking. This duplication could be extracted into a helper.

V4 has the same duplication pattern. Neither version extracts common pre-amble logic.

V4's refactoring of `WriteJSONL` to use `SerializeJSONL` improves DRY within the JSONL package itself.

**Type Safety**

Both versions are type-safe. Both properly use `defer` for cleanup. Both correctly use `context.WithTimeout` for lock deadlines.

**Naming**

V2: `cachePath`, `ParseTasks`, `WriteTasks`, `ReadTasks` — follows the existing naming in the storage sub-packages.
V4: `dbPath`, `ReadJSONLFromBytes`, `SerializeJSONL`, `WriteJSONL`, `ReadJSONL` — more descriptive names that make the format (JSONL) explicit.

### Test Quality

**V2 Test Functions** (in `internal/storage/store_test.go`, 647 lines):

| Test Function | Edge Cases Covered |
|---|---|
| `TestNewStore` | (1) success with valid .tick dir + tasks.jsonl, (2) error for nonexistent dir, (3) error for missing tasks.jsonl |
| `TestMutateExclusiveLock` | Verifies TryLock fails from inside mutation callback (exclusive lock held) |
| `TestQuerySharedLock` | Verifies TryLock (exclusive) fails from inside query callback (shared lock held) |
| `TestLockTimeout` | (1) Timeout error when external exclusive lock held, (2) Exact error message verification |
| `TestConcurrentSharedLocks` | 5 concurrent goroutines all successfully acquire shared locks; verifies all 5 started |
| `TestSharedBlockedByExclusive` | Query times out when external exclusive lock is held |
| `TestExclusiveBlockedByShared` | Mutate times out when external shared lock is held |
| `TestFullWriteFlow` | Full mutation: reads 1 task, adds 1, verifies JSONL has 2, verifies SQLite cache has 2 |
| `TestFullReadFlow` | Full query: reads 2 tasks via SQLite COUNT(*) |
| `TestReleasesLockOnMutationError` | Mutation returns error, then TryLock succeeds (lock released) |
| `TestReleasesLockOnQueryError` | Query returns error, then TryLock succeeds (lock released) |
| `TestContinuesWhenSQLiteFails` | Uses duplicate task IDs to trigger PRIMARY KEY violation; verifies Mutate returns nil and JSONL was written |
| `TestRebuildsStaleOnWrite` | Establishes cache, externally modifies JSONL, verifies mutation sees updated tasks (2 instead of 1) |
| `TestRebuildsStaleOnRead` | Establishes cache, externally modifies JSONL, verifies query sees updated count (2 instead of 1) |

V2 also adds tests in `internal/storage/jsonl/jsonl_test.go` (33 lines):

| Test Function | Edge Cases |
|---|---|
| `TestParseTasks` | (1) Parses 2 tasks from bytes, verifies IDs. (2) Returns empty list for empty bytes |

Total V2 test functions: **14 store tests + 2 ParseTasks tests = 16 test functions**

**V4 Test Functions** (in `internal/store/store_test.go`, 617 lines):

| Test Function | Edge Cases Covered |
|---|---|
| `TestStore_WriteFlow` | Full mutation: reads 2 tasks, adds 1, verifies JSONL has 3 via ReadJSONL, verifies SQLite has 3 + hash stored in metadata |
| `TestStore_LockTimeout` | (1) Mutate timeout with external lock, (2) Query timeout with exact error message comparison |
| `TestStore_ReadFlow` | Full query: reads 2 tasks via SQLite COUNT(*) |
| `TestStore_ExclusiveLock` | TryLock fails from inside mutation (exclusive lock held) |
| `TestStore_SharedLock` | TryRLock succeeds from inside query (verifies shared, not exclusive) |
| `TestStore_ConcurrentReaders` | 5 concurrent goroutines all query successfully; verifies each gets count=2 |
| `TestStore_SharedBlockedByExclusive` | Query times out when external exclusive lock held |
| `TestStore_ExclusiveBlockedByShared` | Mutate times out when external shared lock held |
| `TestStore_LockReleaseOnMutationError` | Mutation returns error, then successful Query proves lock was released |
| `TestStore_LockReleaseOnQueryError` | Query returns error, then successful Mutate proves lock was released |
| `TestStore_SQLiteFailureAfterJSONLWrite` | Replaces cache.db with directory to force Open failure; verifies Mutate returns nil and JSONL has 3 tasks |
| `TestStore_StaleCacheRebuild` | (1) Write: establishes cache, externally modifies JSONL, mutation sees 1 external task. (2) Read: establishes cache, externally modifies JSONL, query sees 1 task with correct ID |

V4 also adds tests in `internal/task/jsonl_test.go` (199 lines):

| Test Function | Edge Cases |
|---|---|
| `TestReadJSONLFromBytes` | (1) Parses 2 tasks with timestamp verification, (2) empty bytes returns empty non-nil slice, (3) skips empty lines, (4) error for invalid JSON, (5) produces same results as ReadJSONL for same content |
| `TestSerializeJSONL` | (1) Serializes 2 tasks, round-trips back, (2) output identical to WriteJSONL file content, (3) empty task list returns empty bytes |

Total V4 test functions: **12 store tests (some with multiple subtests) + 8 jsonl tests = 20 test functions**

**Test Quality Comparison**

Lock release verification:
- V2 verifies by directly calling `flock.TryLock()` after the failed operation — tests the lock mechanism directly.
- V4 verifies by performing a subsequent `Query`/`Mutate` operation — tests the full flow, proving the lock is released in a more integration-style way. This is arguably more robust since it tests the real usage pattern.

Concurrent readers:
- V2 pre-establishes the cache with an initial query to avoid race conditions during concurrent cache creation, then uses `started` channel to verify all readers entered their callbacks. This is more rigorous.
- V4 lets all goroutines race on cache creation; stores errors and counts in indexed slices. Simpler but slightly less rigorous.

SQLite failure test:
- V2 uses duplicate task IDs to trigger a PRIMARY KEY violation in Rebuild. Creative but depends on SQLite's UNIQUE constraint enforcement inside Rebuild.
- V4 replaces cache.db with a directory to make `cache.Open()` fail. This tests a different failure path (Open vs Rebuild) but is more reliable since it doesn't depend on internal cache behavior.

Error message test:
- V2 uses `strings.Contains` to check the error message — tolerant of wrapping.
- V4 uses exact `err.Error() == expectedMsg` comparison in the Query timeout test — stricter.

SharedLock test verification:
- V2 verifies the shared lock by trying to acquire an exclusive lock (should fail). This confirms a shared lock is held.
- V4 verifies the shared lock by trying to acquire another shared lock (should succeed). This confirms the lock IS shared (not exclusive), which is a subtly better test of the requirement.

**Test Gaps**

V2 missing:
- No test for Query lock timeout (only tests Mutate timeout, though `TestLockTimeout` second subtest checks message)
- No test that `NewStore` rejects a file path (not directory) for tickDir
- No tests for `SerializeJSONL` (V2 doesn't have this function)

V4 missing:
- No test for `NewStore` constructor (no validation tests for missing dir, missing tasks.jsonl)
- No test that explicitly verifies the lock file path is `.tick/lock`
- `TestStore_StaleCacheRebuild` write subtest verifies mutation sees correct data but doesn't verify the JSONL and cache are updated afterward

V4's jsonl_test.go additions are significantly more thorough: they test empty lines, invalid JSON, round-trip equivalence with ReadJSONL, and byte-for-byte equivalence with WriteJSONL. V2's ParseTasks tests are minimal (2 subtests).

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 8 (4 internal Go files + go.mod/sum + 2 docs) | 8 (4 internal Go files + go.mod/sum + 2 docs) |
| Lines added | 876 | 1047 |
| Lines removed | 7 | 18 |
| Impl LOC (store.go) | 167 | 179 |
| Test LOC (store_test.go) | 647 | 617 |
| Additional impl changes | +12 lines in jsonl.go (ParseTasks) | +46/-15 lines in jsonl.go (ReadJSONLFromBytes + SerializeJSONL + WriteJSONL refactor) |
| Additional test changes | +33 lines in jsonl_test.go | +199 lines in jsonl_test.go |
| Store test functions | 14 (across 14 top-level tests) | 12 (with subtests; 14 total leaf tests) |
| JSONL test functions added | 2 | 8 |
| Total new test functions | 16 | 20 |

## Verdict

**V4 is the better implementation**, with V2 having one notable advantage.

**V4 advantages:**

1. **Better helper function design**: V4's `SerializeJSONL` enables in-memory hash computation without re-reading from disk, which is the architecturally correct approach. V2's post-write re-read is wasteful and introduces an unnecessary I/O operation.

2. **More thorough JSONL tests**: V4 adds 199 lines of tests for `ReadJSONLFromBytes` and `SerializeJSONL`, covering empty lines, invalid JSON, round-trip equivalence, and byte-for-byte file output comparison. V2 adds only 33 lines of basic tests for `ParseTasks`.

3. **Better Go idioms**: V4 uses lowercase error messages (Go convention), dynamic path interpolation in errors, and better test naming with `TestStore_*` prefix pattern.

4. **Cleaner package separation**: V4's `internal/store/` as a standalone package is a cleaner architecture than V2's approach of placing `store.go` inside the `internal/storage/` package that also contains the `jsonl` and `sqlite` sub-packages.

5. **More robust lock verification test**: V4's `TestStore_SharedLock` tries to acquire a second shared lock (should succeed) — proving the lock is shared. V2 tries to acquire an exclusive lock (should fail) — only proves a lock is held, not that it's shared specifically.

6. **Better SQLite failure test**: V4's directory-replacement technique is simpler and more reliable than V2's duplicate-ID approach.

7. **DRY refactoring**: V4 refactors `WriteJSONL` to use `SerializeJSONL`, removing internal duplication.

**V2 advantages:**

1. **Exact spec compliance for error message**: V2's `"Could not acquire lock on .tick/lock - another process may be using tick"` matches the specification verbatim. V4's lowercase/dynamic-path message deviates from the spec.

2. **More efficient cache lifecycle**: V2's `EnsureFresh` returns `*Cache`, avoiding the double-open that V4 requires (EnsureFresh opens+closes, then Mutate/Query re-opens).

3. **Constructor API**: V2's `NewStoreWithTimeout` is a cleaner API for custom timeouts vs. V4's direct field mutation in tests.

4. **Concurrent reader test quality**: V2 pre-establishes the cache before concurrent reads and uses a channel to verify all readers entered their callbacks, which is more rigorous than V4's approach.

5. **NewStore validation tests**: V2 tests constructor error paths (missing dir, missing tasks.jsonl) while V4 omits these entirely.

Overall, V4's advantages in helper design (avoiding re-read), test breadth (20 vs 16 functions), package organization, and Go idiom compliance outweigh V2's advantages in spec-literal error messages and cache efficiency. The double-open inefficiency in V4 is minor (SQLite opens are fast), and the constructor test gap is a notable but small omission.
