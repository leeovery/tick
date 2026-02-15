TASK: SQLite cache with freshness detection

ACCEPTANCE CRITERIA:
- [ ] SQLite schema matches spec exactly (3 tables, 3 indexes)
- [ ] Full rebuild from `[]Task` populates tasks and dependencies tables correctly
- [ ] SHA256 hash of JSONL content stored in metadata table after rebuild
- [ ] Freshness check correctly identifies fresh vs stale cache
- [ ] Missing cache.db triggers automatic creation and rebuild
- [ ] Corrupted cache.db is deleted, recreated, and rebuilt without failing the operation
- [ ] Empty task list handled (zero rows, hash still stored)
- [ ] Rebuild is transactional (all-or-nothing within single SQLite transaction)

STATUS: Complete

SPEC CONTEXT:
The specification defines SQLite as an expendable cache at `.tick/cache.db`, gitignored, always rebuildable from JSONL. Schema has 3 tables (tasks, dependencies, metadata) and 3 indexes (idx_tasks_status, idx_tasks_priority, idx_tasks_parent). SHA256 hash-based freshness detection runs on every operation: read JSONL, compute hash, compare with stored hash, rebuild if mismatch. Rebuild triggers: hash mismatch, SQLite missing, SQLite query errors. The hash update must be part of the same transaction as the data update.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/storage/cache.go` (lines 1-178): `Cache` struct, `OpenCache`, `Rebuild`, `IsFresh`, `computeHash`, and `schemaSQL` constant
  - `/Users/leeovery/Code/tick/internal/storage/store.go` (lines 245-314): `readAndEnsureFresh` and `ensureFresh` (the `EnsureFresh` gatekeeper logic)
- Notes:
  - Schema at `cache.go:13-40` matches spec exactly: 3 tables (tasks, dependencies, metadata) with correct columns, types, defaults, and primary keys; 3 indexes on status, priority, parent. Uses `IF NOT EXISTS` for idempotent creation.
  - `Rebuild` (cache.go:75-155) correctly: clears all existing data within a transaction, inserts all tasks with proper nullable field handling, inserts dependency rows from `BlockedBy` arrays, computes SHA256 hash of raw JSONL bytes, stores hash in metadata as `jsonl_hash`, commits the transaction. Uses `defer tx.Rollback()` for automatic rollback on error.
  - `IsFresh` (cache.go:159-171) correctly: queries stored hash, returns false on `sql.ErrNoRows` (missing hash = stale), compares hashes.
  - `ensureFresh` (store.go:267-314) correctly: lazy-inits cache, handles open failure by deleting and recreating, handles freshness check error by closing/deleting/recreating, rebuilds if stale, no-ops if fresh.
  - The task plan references a public `EnsureFresh` function; the implementation uses unexported `ensureFresh` as a private method on `Store`. This is appropriate since it is called internally by `readAndEnsureFresh` and should not be part of the public API.
  - Uses `github.com/mattn/go-sqlite3` as specified.
  - Uses `crypto/sha256` from stdlib as specified.
  - Hash computation operates on raw file bytes, not parsed structs, as specified.

TESTS:
- Status: Adequate
- Coverage:
  **In `cache_test.go` (unit tests for Cache struct):**
  - Schema verification (tables, columns, indexes) -- `TestCacheSchema` (line 15)
  - Full field round-trip on rebuild -- `TestCacheRebuild` subtest at line 127
  - Dependencies normalization -- `TestCacheRebuild` subtest at line 203
  - Hash storage in metadata -- `TestCacheRebuild` subtest at line 261
  - Fresh cache detection (hash match) -- `TestCacheFreshness` subtest at line 291
  - Stale cache detection (hash mismatch) -- `TestCacheFreshness` subtest at line 318
  - Empty task list (zero rows, hash stored) -- `TestCacheEdgeCases` subtest at line 348
  - Stale row replacement on rebuild -- `TestCacheEdgeCases` subtest at line 396
  - Transactional rebuild (rollback on duplicate IDs) -- `TestCacheEdgeCases` subtest at line 468

  **In `store_test.go` (integration tests for ensureFresh via Store):**
  - Missing cache.db auto-rebuild -- `TestStoreCacheFreshnessRecovery` subtest at line 742
  - Corrupted cache.db delete/recreate/rebuild -- `TestStoreCacheFreshnessRecovery` subtest at line 782
  - Stale cache hash mismatch rebuild -- `TestStoreCacheFreshnessRecovery` subtest at line 824
  - Corrupted metadata recovery -- `TestStoreCacheFreshnessRecovery` subtest at line 898
  - Stale cache during write -- `TestStoreStaleCacheRebuild` subtest at line 969
  - Stale cache during read -- `TestStoreStaleCacheRebuild` subtest at line 1035

  All 11 test cases from the task plan are covered. Tests verify behavior, not implementation details. Each test would fail if the feature broke.
- Notes: Test coverage is comprehensive and well-balanced. No over-testing observed. Each test verifies a distinct scenario. Helper functions (`queryColumns`, `queryIndexes`, `computeTestHash`, `setupTickDir`, `setupTickDirWithTasks`) use `t.Helper()` correctly.

CODE QUALITY:
- Project conventions: Followed. Uses `internal/` package layout per project-structure reference. Tests use subtests with descriptive names matching the plan's test descriptions. Error handling is explicit throughout; no ignored errors.
- SOLID principles: Good. `Cache` struct has single responsibility (SQLite cache operations). `Store` orchestrates JSONL+cache, separated from `Cache`. The `ensureFresh` method handles the freshness/recovery logic cleanly.
- Complexity: Low. `Rebuild` is linear through its steps. `ensureFresh` has clear conditional paths for lazy init, corruption recovery, and staleness. No deep nesting.
- Modern idioms: Yes. Uses `defer` for cleanup, prepared statements for batch inserts, `sql.NullString` for nullable columns, functional options for configuration.
- Readability: Good. Well-documented exported functions. Clear variable naming. The `schemaSQL` constant at the top of the file makes the schema instantly visible.
- Security: No concerns. SHA256 is appropriate for content hashing. No SQL injection risk (parameterized queries used throughout).
- Performance: Good. Uses prepared statements for batch inserts. Single transaction for rebuild avoids WAL churn. Uses `INSERT OR REPLACE` for metadata upsert.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `computeHash` function in `cache.go:174` could be made more discoverable by adding the algorithm name to the function name (e.g., `computeSHA256`), but the current name is fine given its proximity to the only call site and the import of `crypto/sha256` at the top.
- The corrupted cache.db recovery test in `store_test.go` writes "this is not a sqlite database" as garbage data. SQLite's `sql.Open` does not actually verify the file; it only fails on first query. The `OpenCache` function calls `db.Exec(schemaSQL)` which triggers the failure, so the test works correctly, but the failure path is the schema creation error rather than the open error. The `ensureFresh` function handles both paths (open failure and query failure), so both are covered.
