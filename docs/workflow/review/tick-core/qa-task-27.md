TASK: Consolidate cache freshness/recovery logic (tick-core-6-3)

ACCEPTANCE CRITERIA:
- The standalone EnsureFresh function no longer exists in cache.go
- All freshness and corruption recovery tests exercise the Store code path
- No test coverage is lost -- every scenario previously tested through standalone EnsureFresh is tested through Store
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: The specification defines SHA256 hash-based freshness detection (compare stored hash with computed hash of tasks.jsonl). Cache rebuild triggers include hash mismatch, missing SQLite file, SQLite query errors, and explicit tick rebuild command. The Store.ensureFresh method is the single code path that implements all of these recovery scenarios per the spec's principle that "SQLite is a cache, not a peer."

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/storage/cache.go (no standalone EnsureFresh -- confirmed absent)
- Location: /Users/leeovery/Code/tick/internal/storage/store.go:267-314 (Store.ensureFresh -- the single code path)
- Notes: The standalone EnsureFresh function has been completely removed from cache.go. Only `IsFresh` (line 159) and `Rebuild` (line 75) remain as low-level Cache methods. The Store.ensureFresh method (store.go:267-314) is the sole orchestrator of freshness checking and corruption recovery. It handles three scenarios: (1) lazy init with corruption fallback at lines 269-283, (2) IsFresh query error recovery at lines 285-300, (3) stale hash rebuild at lines 302-311. No references to a standalone EnsureFresh function exist anywhere in the Go source.

TESTS:
- Status: Adequate
- Coverage:
  - TestStoreCacheFreshnessRecovery (store_test.go:741-966) covers all 4 required scenarios:
    1. "it rebuilds automatically when cache.db is missing" (line 742) -- removes cache.db, calls store.Query, verifies task appears
    2. "it deletes and rebuilds when cache.db is corrupted" (line 782) -- writes garbage bytes, calls store.Query, verifies recovery
    3. "it detects stale cache via hash mismatch and rebuilds" (line 824) -- primes cache, modifies JSONL externally, verifies count and updated title
    4. "it handles freshness check errors from corrupted metadata" (line 898) -- drops/replaces metadata table with incompatible schema, reopens store, verifies recovery
  - TestStoreStaleCacheRebuild (store_test.go:968-1108) additionally tests stale cache during both write and read paths
  - TestStoreSQLiteFailure (store_test.go:421-498) tests JSONL-first principle when SQLite update fails after write
  - cache_test.go retains low-level Cache tests for schema, rebuild mechanics, and IsFresh hash comparison -- these test the building blocks, not the freshness orchestration
- Notes: Tests are well-structured with descriptive names. Each test would fail if the feature broke. The corrupted-metadata test (line 898) is notably thorough, opening a second SQLite connection to corrupt the schema. Net coverage increased vs the old standalone EnsureFresh tests (2 old scenarios migrated + 2 new scenarios added). No over-testing observed -- each test covers a distinct failure mode.

CODE QUALITY:
- Project conventions: Followed -- tests use t.Run("it <does thing>", ...) pattern, helpers use t.Helper(), error variables avoid shadowing (wErr, cErr, qErr)
- SOLID principles: Good -- single responsibility maintained. Cache provides low-level operations (Open, Close, IsFresh, Rebuild), Store orchestrates recovery logic. No duplication of orchestration logic remains.
- Complexity: Low -- ensureFresh is ~47 lines handling 3 distinct scenarios with clear control flow
- Modern idioms: Yes -- uses Go idioms appropriately (nil checks for lazy init, error wrapping with %w)
- Readability: Good -- the ensureFresh method has clear comments explaining each recovery branch. Log warnings before destructive recovery operations (delete + recreate) aid debugging.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Minor: Test setup boilerplate (created time, task struct) is repeated across the 4 subtests in TestStoreCacheFreshnessRecovery. A shared fixture at the top could reduce repetition, though the current explicit approach is readable.
