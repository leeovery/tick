# Task Report: tick-core-1-4 -- Storage engine with file locking

## 1. Task Summary

Build a `Store` type that composes the JSONL reader/writer (tick-core-1-2) and SQLite cache (tick-core-1-3) into a single unified storage engine with file locking via `github.com/gofrs/flock`. The store must implement two distinct flows:

- **Mutate (write)**: exclusive lock, read JSONL, compute hash, check freshness (rebuild if stale), apply mutation, atomic write, update cache, release lock.
- **Query (read)**: shared lock, read JSONL, compute hash, check freshness (rebuild if stale), query SQLite, release lock.

Lock timeout is 5 seconds. If JSONL write succeeds but SQLite fails, log warning and return success. Locks must always be released via defer.

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

### Required Tests (from plan)

1. "it acquires exclusive lock for write operations"
2. "it acquires shared lock for read operations"
3. "it returns error after 5-second lock timeout"
4. "it allows concurrent shared locks (multiple readers)"
5. "it blocks shared lock while exclusive lock is held"
6. "it blocks exclusive lock while shared lock is held"
7. "it executes full write flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock"
8. "it executes full read flow: lock -> read JSONL -> freshness check -> query SQLite -> unlock"
9. "it releases lock on mutation function error (no leak)"
10. "it releases lock on query function error (no leak)"
11. "it continues when JSONL write succeeds but SQLite update fails"
12. "it rebuilds stale cache during write before applying mutation"
13. "it rebuilds stale cache during read before running query"
14. "it surfaces correct error message on lock timeout"

---

## 2. Acceptance Criteria Compliance

| # | Criterion | V4 | V5 | Notes |
|---|-----------|----|----|-------|
| 1 | Store composes JSONL + SQLite cache | PASS | PASS | Both create a Store type composing both subsystems. V4 uses `internal/store`, V5 uses `internal/engine`. |
| 2 | Exclusive lock for writes, shared for reads | PASS | PASS | V4: inline `TryLockContext`/`TryRLockContext` in Mutate/Query. V5: factored into `acquireExclusive`/`acquireShared` helpers. |
| 3 | Lock timeout 5s with descriptive error | PARTIAL | PASS | V4 uses the literal lock path in the error message (`could not acquire lock on /path/to/.tick/lock`), which deviates from the spec's prescribed message `"Could not acquire lock on .tick/lock - another process may be using tick"`. V5 returns the exact spec message via a constant. |
| 4 | Concurrent shared locks allowed | PASS | PASS | Both test with 5 concurrent readers. |
| 5 | Exclusive lock blocks readers and writers | PASS | PASS | Both test shared-blocked-by-exclusive and exclusive-blocked-by-shared. |
| 6 | Full write flow | PASS | PASS | Both implement lock -> read -> freshness -> mutate -> atomic write -> update cache -> unlock. |
| 7 | Full read flow | PASS | PASS | Both implement lock -> read -> freshness -> query -> unlock. |
| 8 | Lock released on errors/panics (defer) | PASS | PASS | V4 uses `defer func() { fl.Unlock() }()` inline. V5 returns an `unlock func()` from acquire helpers and callers `defer unlock()`. Both release on error. |
| 9 | JSONL success + SQLite failure = warning + success | PASS | PASS | V4 corrupts cache.db by replacing it with a directory. V5 closes the cache DB inside the mutation callback. Both verify Mutate returns nil. |
| 10 | Stale cache rebuilt before mutation/query | PASS | PASS | V4 tests stale cache more thoroughly by first creating a cache, then externally modifying JSONL. V5 tests with a never-built cache (simpler). |

---

## 3. Implementation Comparison

### 3.1 Package and File Layout

| Aspect | V4 | V5 |
|--------|----|----|
| Store package | `internal/store/` | `internal/engine/` |
| JSONL additions | Modified `internal/task/jsonl.go` (added `ReadJSONLFromBytes`, `SerializeJSONL`) | Added `internal/storage/jsonl.go` (added `ParseTasks`) |
| New files | `store.go`, `store_test.go` | `store.go`, `store_test.go`, modifications to `jsonl.go` / `jsonl_test.go` |

V4 places the store in `internal/store` and adds byte-parsing helpers directly to the existing `internal/task` package. V5 places the store in `internal/engine` and adds a `ParseTasks` function to `internal/storage`. Both are reasonable package names; `engine` arguably better conveys orchestration while `store` is more standard Go naming.

### 3.2 Store Struct and Constructor

**V4** (`internal/store/store.go`, lines 23-59):
```go
type Store struct {
    tickDir     string
    jsonlPath   string
    dbPath      string
    lockPath    string
    lockTimeout time.Duration
    LogFunc func(format string, args ...interface{})
}

func NewStore(tickDir string) (*Store, error) {
```

V4 stores the full `tickDir`, `dbPath`, and exposes `lockTimeout` as a mutable struct field. Logging uses a raw `LogFunc` function field. No functional options pattern. The cache is **not** opened eagerly -- each Mutate/Query opens and closes the cache independently via `cache.Open` and `cache.EnsureFresh`.

**V5** (`internal/engine/store.go`, lines 27-90):
```go
type Store struct {
    jsonlPath   string
    cachePath   string
    lockPath    string
    cache       *cache.Cache
    lockTimeout time.Duration
    verbose     *VerboseLogger
}

func NewStore(tickDir string, opts ...Option) (*Store, error) {
```

V5 opens the cache eagerly in the constructor (`cache.New(cachePath)`) and holds it for the lifetime of the Store. Uses the functional options pattern (`WithLockTimeout`, `WithVerbose`). The VerboseLogger is a separate struct in `verbose.go`.

**Key difference**: V4 opens/closes the cache on every operation, while V5 holds a persistent cache connection. V5's approach is more efficient (avoids repeated open/close overhead) but means `Close()` has real work to do. V4's `Close()` is a no-op.

### 3.3 Lock Acquisition

**V4** (inline in Mutate, lines 82-99):
```go
fl := flock.New(s.lockPath)
ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
defer cancel()

locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
if err != nil {
    return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
}
if !locked {
    return fmt.Errorf("could not acquire lock on %s - another process may be using tick", s.lockPath)
}
defer func() {
    fl.Unlock()
    s.vlog("exclusive lock released")
}()
```

V4 inlines lock acquisition in both `Mutate` and `Query`, duplicating ~15 lines each time. The error message includes the full filesystem path (`s.lockPath`), which deviates from the spec's prescribed message. Poll interval is 100ms.

**V5** (factored helpers, lines 214-251):
```go
func (s *Store) acquireExclusive() (unlock func(), err error) {
    fl := flock.New(s.lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)

    locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
    if !locked || err != nil {
        cancel()
        return nil, fmt.Errorf("%s", lockTimeoutMsg)
    }

    return func() {
        _ = fl.Unlock()
        cancel()
    }, nil
}
```

V5 extracts lock acquisition into `acquireExclusive()` and `acquireShared()` helper methods that return an `unlock` function. This eliminates code duplication and makes the Mutate/Query methods cleaner. The error message uses a constant `lockTimeoutMsg` that matches the spec exactly: `"Could not acquire lock on .tick/lock - another process may be using tick"`. Poll interval is 50ms. The context cancel is called in the unlock function rather than deferred at creation.

**Assessment**: V5's factored approach is cleaner and DRYer. V5 correctly matches the spec error message; V4 includes the full path which violates the spec.

### 3.4 Write Mutation Flow

**V4** (`Mutate`, lines 81-153):
1. Acquire exclusive lock (inline)
2. `os.ReadFile` -> `task.ReadJSONLFromBytes` to parse
3. `cache.EnsureFresh(s.dbPath, jsonlData, tasks)` -- package-level function that opens/closes cache
4. Apply mutation function
5. `task.SerializeJSONL(modified)` to get bytes (used for hash computation post-write)
6. `task.WriteJSONL(s.jsonlPath, modified)` for atomic write
7. Open cache via `cache.Open`, call `c.Rebuild(modified, newJSONLData)`, close cache
8. If cache open or rebuild fails, log warning and return nil (success)

**V5** (`Mutate`, lines 111-140):
1. `acquireExclusive()` -> `defer unlock()`
2. `readAndEnsureFresh()` -- combined helper that reads, parses, checks freshness
3. Apply mutation function
4. `storage.WriteTasks(s.jsonlPath, modified)` for atomic write
5. `updateCache(modified)` -- re-reads JSONL file for new hash bytes, calls `s.cache.Rebuild`

**Key differences**:
- V4 serializes tasks to bytes separately for hash computation (`SerializeJSONL`), then writes atomically. V5 re-reads the file after writing it (`os.ReadFile` inside `updateCache`) to get the bytes for the new hash. V5's approach is simpler but adds an extra file read.
- V4 uses `cache.EnsureFresh` (a package-level convenience) which opens/closes the cache. V5 uses `s.cache.IsFresh` on the persistently-held cache instance.
- V4's `Mutate` is 72 lines; V5's is 29 lines (thanks to helper extraction).

### 3.5 Read Query Flow

**V4** (`Query`, lines 220-268):
1. Acquire shared lock (inline)
2. Read + parse JSONL
3. `cache.EnsureFresh`
4. `cache.Open` to get a DB handle, defer close
5. Call query function with `c.DB()`

**V5** (`Query`, lines 148-160):
1. `acquireShared()` -> `defer unlock()`
2. `readAndEnsureFresh()`
3. Call query function with `s.cache.DB()`

V5 is dramatically more concise (12 lines vs 48 lines) because it holds the cache persistently and extracts helpers.

### 3.6 Rebuild Method (bonus, not in spec)

Both versions implement a `Rebuild()` method that is not in the task spec. This forces a full cache rebuild.

**V4** (`Rebuild`, lines 160-213): Acquires exclusive lock, deletes cache.db, reads JSONL, opens new cache, rebuilds. Returns task count.

**V5** (`Rebuild`, lines 166-210): Same logic but closes existing cache connection first, then deletes and recreates. Uses `s.cache = c` to replace the cache reference.

### 3.7 JSONL Byte Parsing Addition

**V4** added to `internal/task/jsonl.go`:
- `ReadJSONLFromBytes(data []byte) ([]Task, error)` -- parses from bytes, returns `[]Task{}` (non-nil) for empty input
- `SerializeJSONL(tasks []Task) ([]byte, error)` -- serializes tasks to JSONL bytes

**V5** added to `internal/storage/jsonl.go`:
- `ParseTasks(data []byte) ([]task.Task, error)` -- parses from bytes, returns `nil` for empty input

V4 adds two functions; V5 adds one. V4's `SerializeJSONL` is used for post-write hash computation; V5 avoids this by re-reading the file. V4's `ReadJSONLFromBytes` returns non-nil empty slice for empty input; V5's `ParseTasks` returns nil for empty input. The difference is minor but V4's non-nil empty slice is arguably more robust (avoids nil slice issues for callers).

V4 also refactored the existing `ReadJSONL` to call `ReadJSONLFromBytes` internally, consolidating the parsing logic. The original `ReadJSONL` was previously ~30 lines of inline parsing that is now replaced by a 2-line delegation:
```go
func ReadJSONL(path string) ([]Task, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open tasks file: %w", err)
    }
    return ReadJSONLFromBytes(data)
}
```

V5's `ReadTasks` delegates to `ParseTasks` similarly but was likely structured this way from a prior task.

### 3.8 Verbose/Debug Logging

**V4**: Uses a `LogFunc func(format string, args ...interface{})` field on Store. When nil, logging is a no-op via `vlog()` helper. The logging calls are embedded directly in the Mutate/Query/Rebuild methods.

**V5**: Uses a separate `VerboseLogger` struct (in `verbose.go`) with `Log(msg string)` and `Logf(format string, args ...interface{})` methods. The logger has an `enabled` bool and `io.Writer`. The Store gets a default disabled logger in the constructor. Has `WithVerbose` option. V5's verbose files and tests (`verbose.go`, `verbose_test.go`, `store_verbose_test.go`) are NOT part of the task commit -- they were pre-existing.

**Assessment**: Both approaches work. V4's is simpler but less structured. V5's is more testable and cleanly separated. However, since V5's verbose infrastructure pre-existed, only the `s.verbose.Log(...)` calls in store.go are part of this task.

### 3.9 Error Message Compliance

The spec prescribes exactly: `"Could not acquire lock on .tick/lock - another process may be using tick"`

**V4**: `"could not acquire lock on /full/path/to/.tick/lock - another process may be using tick"` -- lowercase "could" and includes full filesystem path. V4's test (`TestStore_LockTimeout`, line 194) constructs the expected message dynamically with the full path:
```go
expectedMsg := fmt.Sprintf("could not acquire lock on %s - another process may be using tick", lockPath)
```
This passes V4's test but does NOT match the spec.

**V5**: `"Could not acquire lock on .tick/lock - another process may be using tick"` -- exact match via `lockTimeoutMsg` constant. V5's test verifies against this exact string.

**Assessment**: V5 is correct per spec. V4 deviates. This is a real defect in V4.

---

## 4. Test Quality

### 4.1 V4 Test Functions

File: `/private/tmp/tick-analysis-worktrees/v4/internal/store/store_test.go` (617 lines)

| Test Function | Description | Matches Spec Test |
|---------------|-------------|-------------------|
| `TestStore_WriteFlow/"it executes full write flow..."` | Writes 3 tasks, verifies JSONL and SQLite cache and hash | #7 |
| `TestStore_LockTimeout/"it returns error after lock timeout"` | External exclusive lock, Mutate times out | #3 |
| `TestStore_LockTimeout/"it surfaces correct error message on lock timeout"` | Tests Query timeout message (uses full path) | #14 |
| `TestStore_ReadFlow/"it executes full read flow..."` | Query reads 2 tasks from SQLite | #8 |
| `TestStore_ExclusiveLock/"it acquires exclusive lock for write operations"` | Inside Mutate callback, tries TryLock on same lock file | #1 |
| `TestStore_SharedLock/"it acquires shared lock for read operations"` | Inside Query callback, tries TryRLock (succeeds) | #2 |
| `TestStore_ConcurrentReaders/"it allows concurrent shared locks..."` | 5 goroutines each create their own Store and Query concurrently | #4 |
| `TestStore_SharedBlockedByExclusive/"it blocks shared lock while exclusive lock is held"` | External exclusive lock, Query times out | #5 |
| `TestStore_ExclusiveBlockedByShared/"it blocks exclusive lock while shared lock is held"` | External shared lock, Mutate times out | #6 |
| `TestStore_LockReleaseOnMutationError/"it releases lock on mutation function error..."` | Mutation returns error, then Query succeeds | #9 |
| `TestStore_LockReleaseOnQueryError/"it releases lock on query function error..."` | Query returns error, then Mutate succeeds | #10 |
| `TestStore_SQLiteFailureAfterJSONLWrite/"it continues when JSONL write succeeds..."` | Replaces cache.db with directory, Mutate still succeeds | #11 |
| `TestStore_StaleCacheRebuild/"it rebuilds stale cache during write before applying mutation"` | Creates cache, externally modifies JSONL, Mutate sees new data | #12 |
| `TestStore_StaleCacheRebuild/"it rebuilds stale cache during read before running query"` | Creates cache, externally modifies JSONL, Query returns new data | #13 |

V4 JSONL test additions (`internal/task/jsonl_test.go`, 199 lines added):

| Test Function | Description |
|---------------|-------------|
| `TestReadJSONLFromBytes/"it parses tasks from in-memory bytes"` | Basic round-trip |
| `TestReadJSONLFromBytes/"it returns empty list for empty bytes"` | Empty slice, non-nil |
| `TestReadJSONLFromBytes/"it skips empty lines"` | Blank lines skipped |
| `TestReadJSONLFromBytes/"it returns error for invalid JSON"` | Invalid JSON error |
| `TestReadJSONLFromBytes/"it produces same results as ReadJSONL for same content"` | Equivalence test |
| `TestSerializeJSONL/"it serializes tasks to JSONL bytes"` | Serialize + parse round-trip |
| `TestSerializeJSONL/"it produces bytes identical to WriteJSONL output"` | File vs memory equivalence |
| `TestSerializeJSONL/"it returns empty bytes for empty task list"` | Empty list handling |

**Total V4 test subtests**: 14 (store) + 8 (jsonl) = 22

### 4.2 V5 Test Functions

File: `/private/tmp/tick-analysis-worktrees/v5/internal/engine/store_test.go` (559 lines)

All tests are under a single `TestStore` function:

| Subtest | Description | Matches Spec Test |
|---------|-------------|-------------------|
| `"it acquires exclusive lock for write operations"` | Goroutine holds Mutate, main thread tries TryLockContext | #1 |
| `"it acquires shared lock for read operations"` | Goroutine holds Query, main tries shared (pass) and exclusive (fail) | #2 |
| `"it returns error after lock timeout"` | External exclusive lock, Mutate with 100ms timeout | #3 |
| `"it allows concurrent shared locks (multiple readers)"` | 5 goroutines, tracks max concurrent via atomic | #4 |
| `"it blocks shared lock while exclusive lock is held"` | External exclusive lock, Query times out | #5 |
| `"it blocks exclusive lock while shared lock is held"` | External shared lock, Mutate times out | #6 |
| `"it executes full write flow..."` | Mutate adds task, Query verifies cache has 3 tasks | #7 |
| `"it executes full read flow..."` | Query reads 2 tasks | #8 |
| `"it releases lock on mutation function error (no leak)"` | Mutation error, then TryLockContext succeeds | #9 |
| `"it releases lock on query function error (no leak)"` | Query error, then TryLockContext succeeds | #10 |
| `"it continues when JSONL write succeeds but SQLite update fails"` | Close cache in callback, Mutate succeeds, JSONL verified | #11 |
| `"it rebuilds stale cache during write before applying mutation"` | Never-built cache, Mutate receives 2 tasks | #12 |
| `"it rebuilds stale cache during read before running query"` | Never-built cache, Query returns 2 tasks | #13 |
| `"it surfaces correct error message on lock timeout"` | Tests both Mutate and Query timeout messages | #14 |

V5 JSONL test additions (`internal/storage/jsonl_test.go`, 61 lines added for ParseTasks):

| Test Function | Description |
|---------------|-------------|
| `TestParseTasks/"it parses tasks from a byte slice of JSONL content"` | Basic parsing |
| `TestParseTasks/"it returns empty list for empty byte slice"` | Empty input |
| `TestParseTasks/"it skips empty lines in byte slice"` | Blank line handling |
| `TestParseTasks/"it returns error for invalid JSON in byte slice"` | Error case |

**Total V5 test subtests**: 14 (store) + 4 (jsonl) = 18

### 4.3 Test Quality Comparison

| Aspect | V4 | V5 |
|--------|----|----|
| All 14 spec tests covered | Yes | Yes |
| Test names match spec exactly | Yes | Yes |
| Concurrent reader test | Each goroutine creates own Store (more realistic for multi-process) | Single Store, tracks `maxConcurrent` with atomics (more rigorous concurrency proof) |
| Lock release verification | V4 does a follow-up operation to prove lock freed | V5 directly tries `TryLockContext` on the lock path |
| Stale cache test | V4 creates cache first, then externally mutates JSONL (proves stale detection). More rigorous. | V5 tests with never-built cache (simpler, less complete -- proves rebuild but not stale detection specifically) |
| SQLite failure test | V4 replaces cache.db with a directory (filesystem corruption) | V5 closes cache DB in callback (in-process sabotage) |
| Write flow verification | V4 verifies JSONL file, SQLite count, AND stored hash | V5 verifies via follow-up Query with SQLite count + task lookup |
| Shared lock test | V4 checks shared-lock-acquirable inside callback | V5 additionally checks exclusive-lock-NOT-acquirable during shared hold |
| JSONL helper tests | 8 subtests (roundtrip, equivalence, edge cases) | 4 subtests (basic parsing, empty, empty lines, error) |

**Key test gaps**:
- V5's stale cache tests are weaker: they only test with a never-built cache, not with a cache that was built and then became stale due to external JSONL modification. V4 explicitly tests the "git pull changed JSONL" scenario.
- V4 has more JSONL helper tests (SerializeJSONL tests, equivalence tests between file and memory functions).
- V5's shared lock test is slightly more thorough by also verifying that an exclusive lock cannot be acquired during a shared hold (two assertions in one test).

---

## 5. Skill Compliance

| Skill Requirement | V4 | V5 |
|-------------------|----|----|
| Document all exported functions/types/packages | PASS -- package doc, all exported funcs documented | PASS -- same |
| Handle all errors explicitly | PASS | PASS |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS | PASS |
| Table-driven tests with subtests | PARTIAL -- uses subtests but not table-driven | PARTIAL -- same |
| Context.Context for blocking operations | PASS -- `context.WithTimeout` for lock acquisition | PASS -- same |

Neither version uses table-driven tests for the store tests. This is reasonable given the nature of the tests (each test has distinct setup/teardown requirements that don't lend themselves to table-driven patterns).

---

## 6. Spec-vs-Convention Conflicts

### Error message format
The spec requires `"Could not acquire lock on .tick/lock - another process may be using tick"`. V4 chose to include the full filesystem path and use lowercase, which provides more debugging information but violates the spec. V5 matches the spec exactly. **The spec should be followed here** -- the prescribed message is user-facing and the full path adds noise for the user.

### Cache lifecycle
The spec says `NewStore` should "validate the .tick/ directory exists and contains tasks.jsonl" but does not prescribe whether the cache should be opened eagerly. V5 opens it eagerly; V4 opens it on-demand per operation. V5's approach is more efficient and conventional for a Go type that has a `Close()` method. V4's no-op `Close()` is a code smell.

### Rebuild method
Neither the spec nor the acceptance criteria mention a `Rebuild()` method. Both versions add it. This is acceptable forward-looking engineering but goes beyond the task scope.

---

## 7. Diff Stats

| Metric | V4 | V5 |
|--------|----|----|
| Files changed (internal/) | 4 | 4 |
| Lines added | 1026 | 881 |
| Lines removed | 15 | 0 |
| store.go lines | 179 (268 total) | 235 (310 total) |
| store_test.go lines | 617 | 559 |
| JSONL additions | 46 lines modified | 26 lines added |
| JSONL test additions | 199 lines | 61 lines |

V5's store.go is longer (235 vs 179 lines) because it includes the `Rebuild` method and uses helper methods. V4's store.go is shorter but has more code duplication between Mutate/Query. V4 has more total lines added due to more extensive JSONL helper tests.

---

## 8. Verdict

**V5 is the stronger implementation.**

### Critical differentiator: Error message compliance
V4's lock timeout error message deviates from the spec. The spec explicitly prescribes: `"Could not acquire lock on .tick/lock - another process may be using tick"`. V4 produces: `"could not acquire lock on /full/path/.tick/lock - another process may be using tick"`. V4's test is written to pass against its own implementation rather than the spec. This is a genuine defect.

### Code quality advantage: V5
V5's factored approach (`acquireExclusive`/`acquireShared` returning unlock functions, `readAndEnsureFresh` helper) results in a Mutate method of 29 lines vs V4's 72 lines. The functional options pattern (`WithLockTimeout`) is more idiomatic than V4's approach of mutating an exported struct field (`s.lockTimeout = 50 * time.Millisecond`) in tests.

V5's persistent cache connection (opened in constructor, closed in `Close()`) is more efficient and conventional than V4's open-close-per-operation pattern. V4's no-op `Close()` method is misleading.

### Test quality advantage: V4 (partial)
V4's stale cache tests are genuinely stronger -- they first build a cache, externally modify JSONL, and verify the store detects staleness and rebuilds. V5 only tests with a never-built cache, which proves rebuild capability but not stale detection. V4 also has more JSONL helper test coverage (8 vs 4 subtests). However, these are minor advantages that don't outweigh V4's error message defect.

### Summary
V5 wins on: spec-correct error messages, cleaner code structure (DRY, functional options, persistent cache), and equivalent acceptance criteria coverage. V4 has marginally better stale-cache testing but a real spec compliance defect that would require a fix.
