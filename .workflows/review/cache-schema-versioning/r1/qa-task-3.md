TASK: End-to-end query success after version-triggered rebuild

ACCEPTANCE CRITERIA:
- Test proves that a cache.db with a missing column (simulating old schema) is transparently replaced and a query referencing that column succeeds
- Test proves that after a version-triggered rebuild, subsequent queries do not re-trigger the rebuild (no extra cycle on second access)
- Test proves that task data is correct after the rebuild (no data loss from the delete-rebuild cycle)
- All existing tests in internal/storage/ continue to pass

STATUS: Complete

SPEC CONTEXT: The original bug manifested as "no such column: t.type" when querying after an upgrade that added the type column to schemaSQL. The cache has a content-freshness model (SHA256 hash) but no schema-freshness model. `CREATE TABLE IF NOT EXISTS` silently succeeds on old schemas without adding missing columns. The fix adds a schema version constant checked early in `ensureFresh()`, triggering delete-and-rebuild on mismatch or missing version. The spec explicitly requires: "After version-triggered rebuild, queries succeed normally."

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/storage/store.go:395-410` (schema version check in `ensureFresh`), `/Users/leeovery/Code/tick/internal/storage/store.go:133-147` (`recreateCache` helper)
- Notes: The implementation correctly places the schema version check before `IsFresh()` in `ensureFresh()`. On mismatch (including missing version returning 0), it calls `recreateCache()` which closes the DB, deletes the file, and opens a fresh empty cache. The subsequent `IsFresh()` call returns false on the empty cache, triggering a full `Rebuild()` that repopulates all data from JSONL. This is the correct two-step flow: (1) recreate empty cache on version mismatch, (2) rebuild data on hash mismatch. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - `TestStoreSchemaVersionUpgrade/"it recovers from pre-versioning cache with missing column and serves correct query results"` (store_test.go:1513-1623): Creates a task with `Type: "feature"`, primes cache, then tampers by (a) removing schema_version row and (b) dropping and recreating the tasks table WITHOUT the `type` column, then re-inserting task data without type. Reopens store and runs a query with `SELECT t.id, t.status, t.priority, t.title, t.type FROM tasks t` -- the exact query pattern that previously failed. Verifies all 5 fields including `type = "feature"` are correct after transparent rebuild. This directly replicates the "no such column: t.type" bug scenario.
  - `TestStoreSchemaVersionUpgrade/"it handles sequential queries after version-triggered rebuild without re-triggering rebuild"` (store_test.go:1625-1741): Primes cache, tampers schema_version to 0, reopens with verbose logging. First query verifies (a) version mismatch was logged, (b) rebuild was logged, (c) task count is correct. Clears logs. Second query verifies (a) "cache is fresh" was logged, (b) no mismatch or rebuild messages appear. This directly proves no extra rebuild cycle on second access.
  - Data correctness is verified in the first test (all 5 columns checked) and in the second test (task count verified on both queries).
  - Test naming matches the expected names from the plan task.
- Notes:
  - The tests are well-structured and test behavior rather than implementation details. The verbose logging capture is a clean way to verify the no-extra-rebuild criterion without adding test-only hooks.
  - The `containsSubstring` helper at line 1744 is overly complex for what it does -- `strings.Contains(s, substr)` alone would suffice since `strings.Contains` already handles all the edge cases the manual checks attempt. However, this is a minor stylistic point and does not affect correctness.
  - Not over-tested: Each test verifies a distinct acceptance criterion. The first test covers the exact bug scenario (missing column + missing version). The second test covers the no-extra-rebuild guarantee. No redundant assertions.
  - Not under-tested: The task-2 tests (`TestStoreSchemaVersionCheck`) already cover wrong version, missing version row, matching version, and SchemaVersion error scenarios. Task-3 tests correctly focus on the end-to-end query behavior aspect (querying a column from the new schema, sequential query efficiency).

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib `testing`, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers, `t.Fatalf` for setup errors, `t.Errorf` for assertion failures. Test names use the "it does X" format.
- SOLID principles: Good. `recreateCache()` has a single responsibility (close + delete + reopen). Schema version check is cleanly separated from freshness check. `SchemaVersion()` and `CurrentSchemaVersion()` provide clean separation of concerns.
- Complexity: Low. The `ensureFresh` flow is linear: lazy init -> version check -> freshness check -> rebuild if needed. Easy to follow.
- Modern idioms: Yes. Uses `sql.NullString` for nullable column scanning. Error wrapping with `%w`. Functional options for verbose logging.
- Readability: Good. Test setup steps are clearly commented ("Prime the cache", "Tamper with cache.db", "Reopen the store"). The test structure follows a clear arrange-act-assert pattern.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The `containsSubstring` helper (store_test.go:1743-1745) is unnecessarily complex. `strings.Contains(s, substr)` handles all the same cases and is more readable. The length checks and equality check are redundant. Consider simplifying.
- In the second test (line 1657), schema_version is set to "0" via UPDATE rather than DELETE. This simulates a numerically-mismatched version (0 != 1) rather than a truly "missing" row. It still triggers the same code path (since `SchemaVersion()` returns 0 and `0 != CurrentSchemaVersion()` is true), and the first test already covers the truly missing row scenario. This is adequate but could be slightly more explicitly commented to explain why "0" is equivalent to "missing" in this context.
