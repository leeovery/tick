TASK: Storage engine with file locking

ACCEPTANCE CRITERIA:
- [x] `Store` composes JSONL reader/writer and SQLite cache into a single interface
- [x] Write operations acquire exclusive lock, read operations acquire shared lock
- [x] Lock timeout of 5 seconds returns descriptive error message
- [x] Concurrent shared locks allowed (multiple readers)
- [x] Exclusive lock blocks all other access (readers and writers)
- [x] Write flow executes full sequence: lock -> read -> freshness -> mutate -> atomic write -> update cache -> unlock
- [x] Read flow executes full sequence: lock -> read -> freshness -> query -> unlock
- [x] Lock is always released, even on errors or panics (defer pattern)
- [x] JSONL write success + SQLite failure = log warning, return success
- [x] Stale cache is rebuilt before mutation or query executes

STATUS: Complete

SPEC CONTEXT:
The spec defines two operation flows. Write (mutation) flow: acquire exclusive lock, read JSONL + compute hash + check freshness, apply mutation, write to temp, fsync, rename, update SQLite in single transaction, release lock. Read (query) flow: acquire shared lock, read JSONL + hash + freshness, query SQLite, release lock. File locking uses `gofrs/flock`. JSONL is source of truth; if JSONL write succeeds but SQLite fails, log warning and continue. SHA256 hash-based freshness detection. Lock timeout 5 seconds.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/storage/store.go` (lines 1-314)
- Notes:
  - `Store` struct (line 22) holds tickDir, jsonlPath, cachePath, lockTimeout, fileLock (*flock.Flock), and cache (*Cache) -- composes JSONL and SQLite as required.
  - `NewStore` (line 54) validates `tasks.jsonl` exists, creates flock instance on `.tick/lock`, defaults to 5s timeout. Uses functional options pattern (WithLockTimeout, WithVerbose).
  - `acquireExclusive` (line 91) uses `TryLockContext` with timeout context and 50ms poll interval. Returns unlock function.
  - `acquireShared` (line 108) uses `TryRLockContext` similarly.
  - `Mutate` (line 134) follows the full write flow: lock -> readAndEnsureFresh -> apply fn -> MarshalJSONL -> WriteJSONLRaw -> cache.Rebuild. On SQLite failure, logs warning via `log.Printf` and returns nil (success).
  - `Query` (line 230) follows the full read flow: lock -> readAndEnsureFresh -> fn(cache.DB()).
  - `readAndEnsureFresh` (line 247) reads JSONL once, parses, then calls `ensureFresh`.
  - `ensureFresh` (line 267) does lazy cache init, checks IsFresh via hash comparison, rebuilds if stale. Handles corruption by deleting and recreating cache.
  - Lock release is via `defer unlock()` immediately after acquisition in both Mutate and Query.
  - Error message: `"could not acquire lock on .tick/lock - another process may be using tick"` -- lowercase per Go convention. The spec says uppercase "Could" but this was explicitly accepted as the right Go idiom choice per analysis-standards-c2.
  - `Rebuild` method (line 180) also provided for forced rebuilds.

TESTS:
- Status: Adequate
- Coverage: All 14 test scenarios from the task are covered. Specifically:
  - `TestStoreMutate/"it acquires exclusive lock for write operations"` (line 42)
  - `TestStoreQuery/"it acquires shared lock for read operations"` (line 67)
  - `TestStoreLockTimeout/"it returns error after lock timeout"` (line 94) -- uses external flock hold + short timeout
  - `TestStoreLockTimeout/"it surfaces correct error message on lock timeout"` (line 125) -- tests Query path too
  - `TestStoreConcurrentLocks/"it allows concurrent shared locks (multiple readers)"` (line 159) -- uses flock directly
  - `TestStoreConcurrentLocks/"it blocks shared lock while exclusive lock is held"` (line 185) -- TryRLock fails
  - `TestStoreConcurrentLocks/"it blocks exclusive lock while shared lock is held"` (line 208) -- TryLock fails
  - `TestStoreWriteFlow/full write flow` (line 233) -- verifies JSONL and SQLite both updated
  - `TestStoreReadFlow/full read flow` (line 315) -- verifies query against SQLite
  - `TestStoreLockRelease/"it releases lock on mutation function error (no leak)"` (line 368) -- verifies subsequent Mutate succeeds
  - `TestStoreLockRelease/"it releases lock on query function error (no leak)"` (line 394) -- verifies subsequent Query succeeds
  - `TestStoreSQLiteFailure/"it continues when JSONL write succeeds but SQLite update fails"` (line 422) -- corrupts cache during mutation, verifies success return and self-heal
  - `TestStoreStaleCacheRebuild/"it rebuilds stale cache during write before applying mutation"` (line 969) -- externally modifies JSONL, verifies mutation sees updated data
  - `TestStoreStaleCacheRebuild/"it rebuilds stale cache during read before running query"` (line 1035) -- externally modifies JSONL, verifies query sees updated data
  - Additional tests for Rebuild method, cache freshness recovery (missing cache, corrupted cache, corrupted metadata, hash mismatch), verbose logging
- Notes:
  - The concurrent locks tests (lines 159-229) test flock behavior directly rather than through Store's Mutate/Query. This is acceptable since they verify the underlying mechanism, but a test with concurrent goroutines both using Store.Query simultaneously would provide stronger integration coverage. This is minor since the flock library's shared lock semantics are well-established.
  - Tests are well-structured, not over-tested. Each test verifies a distinct behavior. No redundant assertions.
  - Test helper functions use `t.Helper()` correctly.

CODE QUALITY:
- Project conventions: Followed. Uses `internal/` package structure. Error wrapping with `%w`. Functional options pattern for configuration.
- SOLID principles: Good. Store has a single responsibility (orchestrating JSONL + cache + locking). Cache and JSONL operations are delegated to separate types/functions. Open for extension via StoreOption functional options.
- Complexity: Low. Mutate and Query are linear flows. `ensureFresh` has moderate branching for corruption recovery but is clear and well-commented.
- Modern idioms: Yes. Context-based timeouts, functional options, defer for cleanup, `errors.New` for static errors.
- Readability: Good. Methods are well-documented. Flow comments in Mutate/Query match the spec's described sequence. Verbose logging aids debugging.
- Issues:
  - The `Mutate` method rebuilds the entire cache after every write (line 166: `s.cache.Rebuild(mutated, newRawJSONL)`). The spec says step 7 is "Update SQLite in single transaction: apply mutation + store new hash." The implementation does a full rebuild (DELETE all + INSERT all) rather than an incremental update. For the expected scale (<500 tasks), this is acceptable and simpler, but it is a minor deviation from the spec's "apply mutation" wording. The same transaction atomicity guarantee is maintained.
  - `log.Printf` is used for the SQLite failure warning (line 167). This goes to stderr via the default logger, which aligns with the spec's "log warning to stderr." However, it is not configurable and could be noisy in tests. The verbose logging system (`s.verbose`) is not used here, which is the correct choice since this is a warning, not debug output.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The lock error message uses lowercase "could" instead of the spec's uppercase "Could". This was consciously decided as the Go-idiomatic choice and documented in analysis-standards-c2. No action needed.
- The concurrent shared lock test (lines 159-183) tests flock directly rather than via Store.Query with goroutines. A goroutine-based integration test would be stronger but is not strictly necessary given the flock library's well-tested behavior.
- The `Mutate` method does a full cache Rebuild after writes rather than an incremental SQLite update. This is simpler and correct for the expected scale but deviates slightly from the spec's "apply mutation" language.
