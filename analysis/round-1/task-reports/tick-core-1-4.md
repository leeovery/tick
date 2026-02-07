# Task tick-core-1-4: Storage engine with file locking

## Task Summary

Build a unified `Store` type that composes the JSONL reader/writer (tick-core-1-2) and SQLite cache (tick-core-1-3) into a single orchestration layer with file locking via `github.com/gofrs/flock`.

Two core operations:

**Mutate (write flow):** acquire exclusive lock on `.tick/lock` (5s timeout) -> read `tasks.jsonl` + compute SHA256 hash -> check SQLite freshness (rebuild if stale) -> pass `[]Task` to mutation function -> write modified tasks via atomic rewrite -> update SQLite in single transaction -> release lock via defer. If JSONL write succeeds but SQLite update fails, log warning to stderr and return success (self-heals on next read).

**Query (read flow):** acquire shared lock on `.tick/lock` (5s timeout) -> read `tasks.jsonl` + compute hash -> check freshness (rebuild if stale) -> execute query function against SQLite -> release lock via defer.

### Acceptance Criteria

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Store composes JSONL + SQLite into single interface | PASS - `Store` struct holds `*Cache`, `jsonlPath`, `lockPath` | PASS - `Store` struct holds paths, opens cache per-operation via `sqlite.EnsureFresh` | PASS - `Store` struct holds paths + `*flock.Flock`, opens cache per-operation via `EnsureFresh` |
| 2. Exclusive lock for writes, shared lock for reads | PASS - `acquireExclusiveLock()` / `acquireSharedLock()` | PASS - `TryLockContext` in Mutate / `TryRLockContext` in Query | PASS - `TryLockContext` in Mutate / `TryRLockContext` in Query |
| 3. Lock timeout of 5s with descriptive error | PASS - `defaultLockTimeout = 5 * time.Second`, correct message (lowercase "could") | PASS - `defaultLockTimeout = 5 * time.Second`, message uses uppercase "Could" matching spec exactly | PASS - `lockTimeout = 5 * time.Second`, correct message (lowercase "could") |
| 4. Concurrent shared locks allowed | PASS - tested with 5 concurrent goroutines | PASS - tested with 5 concurrent goroutines, pre-initializes cache | PASS - tested with 5 concurrent goroutines |
| 5. Exclusive lock blocks all others | PARTIAL - tests only timeout scenario, no explicit "shared blocked by exclusive" or "exclusive blocked by shared" tests | PASS - dedicated `TestSharedBlockedByExclusive` and `TestExclusiveBlockedByShared` tests | PASS - dedicated `TestStoreSharedBlocksExclusive` and `TestStoreExclusiveBlocksShared` tests |
| 6. Full write flow | PASS - tested via `TestStoreMutate/"executes full write flow"` | PASS - tested via `TestFullWriteFlow` with verification of both JSONL and SQLite | PASS - tested via `TestStoreWriteFlow` with verification of both JSONL and SQLite |
| 7. Full read flow | PASS - tested via `TestStoreQuery/"executes full read flow"` | PASS - tested via `TestFullReadFlow` with row-level verification | PASS - tested via `TestStoreReadFlow` with row-level verification |
| 8. Lock always released via defer | PASS - `defer fl.Unlock()` after acquire in both methods | PASS - `defer fl.Unlock()` after acquire in both methods | PASS - `defer s.flock.Unlock()` after acquire in both methods |
| 9. JSONL success + SQLite failure = warning, continue | PASS - `fmt.Fprintf(os.Stderr, "warning: ...")` and returns nil | PASS - `log.Printf("warning: ...")` and returns nil | PASS - `fmt.Fprintf(os.Stderr, "warning: ...")` and returns nil |
| 10. Stale cache rebuilt before mutation/query | PASS - `readAndEnsureFresh()` called at start of both Mutate and Query | PASS - `sqlite.EnsureFresh()` called in both Mutate and Query | PASS - `EnsureFresh()` called in both Mutate and Query |

## Implementation Comparison

### Approach

#### Package structure

**V1** and **V3** use a flat `internal/storage/` package. Their store lives in the same package as the JSONL and cache code. V1's parent commit has `cache.go` and `jsonl.go` side-by-side with `store.go`.

**V2** uses sub-packages: `internal/storage/jsonl/` and `internal/storage/sqlite/`. The store sits at the parent level `internal/storage/store.go` and imports from both sub-packages. V2 also modified the existing `jsonl.go` to add a new `ParseTasks(data []byte)` function for parsing raw bytes without file I/O, and refactored `ReadTasks` to delegate to `ParseTasks`.

#### Store struct design

**V1** keeps a persistent `*Cache` reference, opened once in `NewStore`:

```go
type Store struct {
    tickDir     string
    jsonlPath   string
    cachePath   string
    lockPath    string
    cache       *Cache
    lockTimeout time.Duration
}
```

The `NewStore` constructor calls `NewCacheWithRecovery(cachePath)` upfront. This means the SQLite connection is held open for the Store's lifetime. A `Close()` method delegates to `cache.Close()`.

**V2** does NOT hold a cache reference. It opens/closes the cache per operation:

```go
type Store struct {
    tickDir     string
    jsonlPath   string
    cachePath   string
    lockPath    string
    lockTimeout time.Duration
}
```

Each `Mutate`/`Query` call invokes `sqlite.EnsureFresh(s.cachePath, tasks, rawContent)` which returns a fresh `*Cache`, used within the operation and closed via `defer cache.Close()`. The `Close()` method is a no-op.

**V3** holds a persistent `*flock.Flock` instance (reused across operations) but opens cache per-operation like V2:

```go
type Store struct {
    tickDir   string
    jsonlPath string
    cachePath string
    lockPath  string
    flock     *flock.Flock
}
```

The flock is created once in `NewStore` and reused. Cache is opened per-operation via `EnsureFresh()`. `Close()` is a no-op. Notably, V3 does NOT have a configurable lock timeout -- `lockTimeout` is a package-level `const`, not a struct field.

#### Lock acquisition

**V1** creates a new `flock.Flock` per lock acquisition via helper methods `acquireExclusiveLock()` and `acquireSharedLock()`:

```go
func (s *Store) acquireExclusiveLock() (*flock.Flock, error) {
    fl := flock.New(s.lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
    defer cancel()
    locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
    if err != nil || !locked {
        return nil, fmt.Errorf("could not acquire lock on .tick/lock - another process may be using tick")
    }
    return fl, nil
}
```

Retry interval: 50ms. Returns the flock so the caller can `defer fl.Unlock()`.

**V2** inlines lock acquisition directly in `Mutate`/`Query`. Also creates a new flock per call:

```go
fl := flock.New(s.lockPath)
ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
defer cancel()
locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
if err != nil || !locked {
    return fmt.Errorf("Could not acquire lock on .tick/lock - another process may be using tick")
}
defer fl.Unlock()
```

Retry interval: 10ms (more aggressive). Error message uses uppercase "Could" matching the spec exactly.

**V3** reuses the single `*flock.Flock` from the struct. Inlines lock acquisition:

```go
locked, err := s.flock.TryLockContext(ctx, 50*time.Millisecond)
if err != nil || !locked {
    if errors.Is(err, context.DeadlineExceeded) || !locked {
        return errors.New("could not acquire lock on .tick/lock - another process may be using tick")
    }
    return fmt.Errorf("failed to acquire lock: %w", err)
}
defer s.flock.Unlock()
```

Retry interval: 50ms. V3 has the most nuanced error handling: it distinguishes between deadline exceeded (returns the user-facing message) and other errors (returns a generic wrapped error). However, reusing a single `*flock.Flock` instance is potentially problematic for concurrent access from multiple goroutines -- the flock library may not be safe for concurrent use from the same instance.

#### JSONL reading strategy

**V1** reads raw content once via `os.ReadFile`, then parses with `ReadJSONLBytes(content)` (a byte-based parser):

```go
func (s *Store) readAndEnsureFresh() ([]byte, []task.Task, error) {
    content, err := os.ReadFile(s.jsonlPath)
    tasks, err := ReadJSONLBytes(content)
    if err := s.cache.EnsureFresh(content, tasks); err != nil { ... }
    return content, tasks, nil
}
```

**V2** also reads raw content once, then parses with `jsonl.ParseTasks(rawContent)` (the new function V2 added to the jsonl package):

```go
rawContent, err := os.ReadFile(s.jsonlPath)
tasks, err := jsonl.ParseTasks(rawContent)
cache, err := sqlite.EnsureFresh(s.cachePath, tasks, rawContent)
```

**V3** reads the file TWICE -- once with `os.ReadFile` for raw bytes, once with `ReadJSONL(path)` which opens and reads the file again:

```go
func (s *Store) readJSONLWithContent() ([]task.Task, []byte, error) {
    content, err := os.ReadFile(s.jsonlPath)
    tasks, err := ReadJSONL(s.jsonlPath)  // reads file a second time
    return tasks, content, nil
}
```

This is an inefficiency: the file is read from disk twice per operation. V1 and V2 both avoid this by parsing from the already-loaded bytes.

#### Cache freshness approach

**V1** calls `s.cache.EnsureFresh(content, tasks)` on the persistent cache instance. This is a method on `*Cache` that checks hash and rebuilds in-place if stale.

**V2** calls `sqlite.EnsureFresh(s.cachePath, tasks, rawContent)` which is a package-level function that opens the cache, checks freshness, rebuilds if needed, and returns the cache handle. The caller is responsible for closing it.

**V3** calls `EnsureFresh(s.cachePath, tasks, jsonlContent)` -- same pattern as V2 (package-level function, returns `*Cache`, caller closes).

#### SQLite failure handling in Mutate

All three versions follow the same pattern: after successful JSONL write, re-read the file for hash computation, then attempt `cache.Rebuild`. If Rebuild fails, log a warning and return nil.

**V1:**
```go
if err := s.cache.Rebuild(modified, newContent); err != nil {
    fmt.Fprintf(os.Stderr, "warning: cache update failed (will self-heal on next read): %v\n", err)
}
return nil
```

**V2:**
```go
if err := cache.Rebuild(modified, newRawContent); err != nil {
    log.Printf("warning: failed to update SQLite cache: %v", err)
    return nil
}
```

**V3:**
```go
if err := cache.Rebuild(modified, newContent); err != nil {
    fmt.Fprintf(os.Stderr, "warning: failed to update SQLite cache: %v (will self-heal on next read)\n", err)
    return nil
}
```

V2 uses `log.Printf` (adds timestamp prefix). V1 and V3 use `fmt.Fprintf(os.Stderr, ...)`. The spec says "log warning to stderr" -- `log.Printf` defaults to stderr so all three comply, but V1/V3 are more explicit.

#### Error wrapping in Mutate/Query

**V1** wraps all errors with descriptive context: `fmt.Errorf("mutation failed: %w", err)`, `fmt.Errorf("reading tasks.jsonl: %w", err)`.

**V2** also wraps: `fmt.Errorf("mutation failed: %w", err)`, `fmt.Errorf("failed to read tasks.jsonl: %w", err)`, `fmt.Errorf("query failed: %w", err)`.

**V3** does NOT wrap errors from `readJSONLWithContent` or `EnsureFresh` -- it returns them bare. The mutation function error is also returned bare (no wrapping). Only the lock acquisition has special error handling. The Query method does not wrap the query function's error either -- it returns `fn(cache.db)` directly.

#### Timeout configurability

**V1** stores `lockTimeout` as a struct field, set to `defaultLockTimeout` in `NewStore`. Tests override it directly: `store.lockTimeout = 100 * time.Millisecond`.

**V2** provides `NewStoreWithTimeout(tickDir string, lockTimeout time.Duration)` as a dedicated constructor. Tests use it: `NewStoreWithTimeout(tickDir, 50*time.Millisecond)`.

**V3** uses a package-level `const lockTimeout = 5 * time.Second`. There is NO way to override the timeout. The lock timeout test (`TestStoreLockTimeout`) actually waits the full 5 seconds (guarded by `testing.Short()` skip). This makes tests significantly slower.

### Code Quality

#### Go idioms

**V1** follows good Go idioms: extracted helper methods (`acquireExclusiveLock`, `acquireSharedLock`, `readAndEnsureFresh`), consistent error wrapping with `%w`, clear method documentation.

**V2** has the cleanest package organization (sub-packages for jsonl/sqlite). It adds `ParseTasks` to the jsonl package to avoid the "read from bytes" duplication. The `NewStoreWithTimeout` constructor is good Go practice for test-friendly timeout configuration. However, the duplicate code between `Mutate` and `Query` for lock acquisition + JSONL read + freshness check is not extracted into a helper.

**V3** has some anti-patterns:
- Reusing a single `*flock.Flock` instance (`s.flock`) could cause issues with concurrent goroutines since flock instances track lock state.
- `readJSONLWithContent` reads the file twice (once for raw bytes, once for parsing).
- No error wrapping on most error paths.
- Package-level `const` for timeout prevents test configurability.

#### Naming

All three use consistent Go naming. V2's `ParseTasks` is well-named for a byte-parsing function. V1's helper methods `acquireExclusiveLock`/`acquireSharedLock` are descriptive. V3's `readJSONLWithContent` is reasonable.

#### DRY

**V1** is the DRYest: `readAndEnsureFresh()` is called by both Mutate and Query, and lock acquisition is extracted into two helpers.

**V2** has the most code duplication in `store.go`: the lock acquisition + JSONL read + parse + freshness check sequence is copied between `Mutate` and `Query` (approximately 15 lines duplicated).

**V3** extracts `readJSONLWithContent()` but the lock acquisition is duplicated between Mutate and Query.

#### Error message casing

The spec says: `"Could not acquire lock on .tick/lock - another process may be using tick"` (capital C).

- V1: `"could not acquire lock..."` (lowercase) -- does not match spec
- V2: `"Could not acquire lock..."` (uppercase) -- matches spec exactly
- V3: `"could not acquire lock..."` (lowercase) -- does not match spec

### Test Quality

#### V1 Test Functions (4 top-level, 10 subtests)

1. `TestNewStore`
   - `"opens store with valid tick directory"` -- happy path
   - `"errors if tick directory does not exist"` -- nonexistent dir
   - `"errors if tasks.jsonl does not exist"` -- dir exists, no JSONL

2. `TestStoreMutate`
   - `"executes full write flow"` -- creates task via mutate, verifies via query AND JSONL file
   - `"releases lock on mutation function error"` -- errors, then queries to prove lock released
   - `"rebuilds stale cache before applying mutation"` -- writes JSONL externally, verifies mutation sees it

3. `TestStoreQuery`
   - `"executes full read flow"` -- adds task via Mutate, queries by ID
   - `"releases lock on query function error"` -- errors, then queries again
   - `"rebuilds stale cache before running query"` -- writes JSONL externally, queries for it

4. `TestStoreLocking`
   - `"allows concurrent shared locks"` -- 5 goroutines, atomic counter
   - `"surfaces correct error message on lock timeout"` -- external lock, reduced timeout

#### V2 Test Functions (13 top-level, 13 subtests)

1. `TestNewStore`
   - `"it creates store when .tick/ directory exists with tasks.jsonl"`
   - `"it returns error when .tick/ directory does not exist"`
   - `"it returns error when tasks.jsonl does not exist in .tick/"`

2. `TestMutateExclusiveLock`
   - `"it acquires exclusive lock for write operations"` -- verifies lock is held during mutation by trying to acquire another exclusive lock from inside the callback

3. `TestQuerySharedLock`
   - `"it acquires shared lock for read operations"` -- verifies shared lock is held by trying to acquire another shared lock from inside the callback (proves it's shared, not exclusive)

4. `TestLockTimeout`
   - `"it returns error after lock timeout"` -- with 50ms timeout
   - `"it surfaces correct error message on lock timeout"` -- checks exact message string

5. `TestConcurrentSharedLocks`
   - `"it allows concurrent shared locks (multiple readers)"` -- 5 goroutines, pre-initializes cache first, uses channel to track started count

6. `TestSharedBlockedByExclusive`
   - `"it blocks shared lock while exclusive lock is held"` -- external exclusive lock, store query times out

7. `TestExclusiveBlockedByShared`
   - `"it blocks exclusive lock while shared lock is held"` -- external shared lock, store mutate times out

8. `TestFullWriteFlow`
   - `"it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock"` -- pre-populated JSONL, adds task, verifies both JSONL content and SQLite count

9. `TestFullReadFlow`
   - `"it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock"` -- 2 tasks in JSONL, verifies count

10. `TestReleasesLockOnMutationError`
    - `"it releases lock on mutation function error (no leak)"` -- verifies by acquiring lock externally after error

11. `TestReleasesLockOnQueryError`
    - `"it releases lock on query function error (no leak)"` -- verifies by acquiring lock externally after error

12. `TestContinuesWhenSQLiteFails`
    - `"it continues when JSONL write succeeds but SQLite update fails"` -- uses duplicate task IDs to trigger PRIMARY KEY violation in SQLite, verifies JSONL written

13. `TestRebuildsStaleOnWrite`
    - `"it rebuilds stale cache during write before applying mutation"` -- establishes cache, externally modifies JSONL, verifies mutation sees 2 tasks

14. `TestRebuildsStaleOnRead`
    - `"it rebuilds stale cache during read before running query"` -- establishes cache, externally modifies JSONL, verifies query sees 2 tasks

#### V3 Test Functions (14 top-level, 14 subtests)

1. `TestStoreExclusiveLock`
   - `"it acquires exclusive lock for write operations"` -- similar to V2, tries to acquire exclusive lock from inside mutation callback with 100ms timeout

2. `TestStoreSharedLock`
   - `"it acquires shared lock for read operations"` -- tries to acquire another shared lock from inside query callback

3. `TestStoreLockTimeout`
   - `"it returns error after 5-second lock timeout"` -- waits FULL 5 seconds (has `testing.Short()` skip), also checks elapsed time is between 4-7 seconds

4. `TestStoreConcurrentSharedLocks`
   - `"it allows concurrent shared locks (multiple readers)"` -- 5 goroutines, 100ms sleep

5. `TestStoreSharedBlocksExclusive`
   - `"it blocks exclusive lock while shared lock is held"` -- external shared lock, goroutine Mutate, 200ms context timeout

6. `TestStoreExclusiveBlocksShared`
   - `"it blocks shared lock while exclusive lock is held"` -- external exclusive lock, goroutine Query, 200ms context timeout

7. `TestStoreWriteFlow`
   - `"it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock"` -- initial JSONL with 1 task, adds second, verifies JSONL lines and SQLite count

8. `TestStoreReadFlow`
   - `"it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock"` -- 2 tasks in JSONL, queries IDs, verifies order

9. `TestStoreMutationErrorReleasesLock`
   - `"it releases lock on mutation function error (no leak)"` -- verifies by acquiring external lock with 100ms timeout

10. `TestStoreQueryErrorReleasesLock`
    - `"it releases lock on query function error (no leak)"` -- verifies by acquiring external lock with 100ms timeout

11. `TestStoreJSONLSuccessSQLiteFailure`
    - `"it continues when JSONL write succeeds but SQLite update fails"` -- creates cache, makes `cache.db` read-only (chmod 0444), mutates, verifies success and JSONL content

12. `TestStoreStaleCache_Write`
    - `"it rebuilds stale cache during write before applying mutation"` -- establishes cache, externally modifies JSONL, verifies mutation sees 2 tasks

13. `TestStoreStaleCache_Read`
    - `"it rebuilds stale cache during read before running query"` -- establishes cache, externally modifies JSONL, verifies query sees 2 tasks

14. `TestNewStoreValidation`
    - `"it validates .tick directory exists and contains tasks.jsonl"` -- tests nonexistent dir, dir without JSONL, and valid dir in a single subtest

#### Test Coverage Diff

| Test Scenario | V1 | V2 | V3 |
|---------------|-----|-----|-----|
| NewStore with valid dir | YES | YES | YES |
| NewStore with nonexistent dir | YES | YES | YES |
| NewStore with missing tasks.jsonl | YES | YES | YES |
| Exclusive lock acquired during Mutate | NO (implicit) | YES (explicit lock check inside callback) | YES (explicit lock check inside callback) |
| Shared lock acquired during Query | NO (implicit) | YES (explicit lock check inside callback) | YES (explicit lock check inside callback) |
| Lock timeout error | YES | YES | YES |
| Lock timeout error message exact text | YES | YES | YES |
| Lock timeout actually waits ~5s | NO | NO | YES (measures elapsed 4-7s) |
| Concurrent shared locks (multiple readers) | YES (5 goroutines) | YES (5 goroutines, pre-init cache) | YES (5 goroutines) |
| Shared blocked by exclusive | NO | YES | YES |
| Exclusive blocked by shared | NO | YES | YES |
| Full write flow with JSONL + cache verification | YES | YES | YES |
| Full read flow with SQLite query | YES | YES | YES |
| Lock released on mutation error | YES (uses subsequent Query) | YES (uses external flock TryLock) | YES (uses external flock TryLock) |
| Lock released on query error | YES (uses subsequent Query) | YES (uses external flock TryLock) | YES (uses external flock TryLock) |
| JSONL success + SQLite failure | NO | YES (duplicate IDs cause PK violation) | YES (chmod cache.db to 0444) |
| Stale cache on write | YES | YES | YES |
| Stale cache on read | YES | YES | YES |
| `ParseTasks` (new JSONL function) | N/A | YES (2 tests added to jsonl_test.go) | N/A |

**Tests unique to V1:** None -- V1 has the smallest test surface.

**Tests unique to V2:** `TestContinuesWhenSQLiteFails` uses duplicate IDs to trigger PK violation; also adds `TestParseTasks` tests for the new `ParseTasks` function.

**Tests unique to V3:** `TestStoreLockTimeout` measures actual elapsed time (4-7s). `TestStoreJSONLSuccessSQLiteFailure` uses `os.Chmod` to make cache read-only.

**V1 gaps:** No test for JSONL success + SQLite failure. No tests for shared-blocked-by-exclusive or exclusive-blocked-by-shared. No explicit verification that the lock type is correct during Mutate/Query (just tests that operations complete).

**V2 gaps:** Does not measure actual timeout duration.

**V3 gaps:** The 5-second lock timeout test makes the test suite slow (mitigated by `testing.Short()` skip, but will be slow in full runs).

#### SQLite failure simulation comparison

V2's approach (duplicate IDs causing PRIMARY KEY violation) is clever but fragile -- it depends on implementation details of how `Rebuild` works internally.

V3's approach (`os.Chmod` to make `cache.db` read-only) is more robust because it fails regardless of the data being written. However, it may behave differently on different operating systems (file permissions are OS-dependent, and on some systems the root user can bypass permissions).

V1 does not test this scenario at all.

#### Lock release verification comparison

V1 verifies lock release by attempting another Store operation (Query after failed Mutate). This is indirect -- it proves the lock isn't held but relies on the Store's own locking being correct.

V2 and V3 verify by creating an external `flock.Flock` and calling `TryLock` on the lock file directly. This is a more direct and reliable verification.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 4 (go.mod, go.sum, store.go, store_test.go) | 8 (go.mod, go.sum, store.go, store_test.go, jsonl.go, jsonl_test.go, 2 docs) | 7 (go.mod, go.sum, store.go, store_test.go, 2 docs, context doc) |
| Lines added | 460 | 876 | 1025 |
| Impl LOC (store.go) | 158 | 167 | 179 |
| Test LOC (store_test.go) | 293 | 647 | 810 |
| Test functions (top-level) | 4 | 13 | 14 |
| Test subtests | 10 | 13 | 14 |

## Verdict

**V2 is the best implementation** for the following reasons:

1. **Correctest error message:** V2 is the only version that uses the exact error message casing from the spec: `"Could not acquire lock on .tick/lock - another process may be using tick"` with capital C. V1 and V3 use lowercase.

2. **Best package architecture:** V2 cleanly separates concerns with sub-packages (`jsonl/`, `sqlite/`) and deliberately added `ParseTasks` to the jsonl package to enable byte-based parsing without file I/O duplication. This is a genuine improvement to the prior code.

3. **Test-friendly timeout configuration:** V2 provides `NewStoreWithTimeout` as a proper constructor. V1 exposes the timeout as a mutable struct field (works but less clean API). V3 uses a package-level constant with no override, forcing the lock timeout test to wait 5 full seconds.

4. **Comprehensive test coverage:** V2 covers all 14 specified test scenarios. It has dedicated tests for every locking behavior (exclusive lock acquired, shared lock acquired, shared blocked by exclusive, exclusive blocked by shared), all edge cases (SQLite failure, stale cache), and also tests the new `ParseTasks` function it added. The SQLite failure test using duplicate IDs is creative.

5. **No double file read:** V2 reads `tasks.jsonl` once and parses from bytes, whereas V3 reads the file twice per operation.

6. **Safe flock usage:** V2 creates a new `flock.Flock` per operation (like V1), which is safer for concurrent access. V3 reuses a single flock instance, which could be problematic if multiple goroutines call Mutate/Query concurrently.

**V1** is the most compact and DRYest implementation but has notable test gaps (no SQLite failure test, no bidirectional lock blocking tests).

**V3** has the most thorough individual tests (including timing verification for lock timeout) but has implementation flaws: double file read, non-configurable timeout making tests slow, shared flock instance, and bare (unwrapped) errors.
