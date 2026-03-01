TASK: Check schema version in ensureFresh and delete-rebuild on mismatch

ACCEPTANCE CRITERIA:
- ensureFresh() queries schema_version before calling IsFresh()
- Cache with wrong schema version triggers close + delete + reopen (and subsequent rebuild)
- Cache with missing schema_version row (pre-versioning cache.db) triggers close + delete + reopen (and subsequent rebuild)
- Cache with correct schema version is preserved -- no unnecessary delete or rebuild (beyond normal hash-based freshness)
- Errors from SchemaVersion() are handled gracefully (delete + recreate, same as existing corruption handling)
- All existing tests in internal/storage/ continue to pass

STATUS: Complete

SPEC CONTEXT: The spec identifies a critical upgrade-path bug: when the SQLite cache schema changes, existing cache.db files become incompatible. The fix requires checking a schema version constant early in ensureFresh(), before any queries, and deleting+rebuilding the cache on mismatch or absence. The version check must happen before IsFresh() to avoid querying a schema-incompatible cache. No ALTER TABLE migrations -- full rebuild is correct since the cache is ephemeral.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - store.go:135-147 -- recreateCache() helper: close + nil + delete + reopen
  - store.go:395-410 -- Schema version check in ensureFresh(), positioned after lazy init but before IsFresh()
  - cache.go:258-272 -- SchemaVersion() method: reads from metadata, returns 0 for missing row
  - cache.go:275-277 -- CurrentSchemaVersion() exported accessor
- Notes:
  - The version check at store.go:399-410 correctly runs before IsFresh() at line 412, matching the spec requirement.
  - Error handling (line 400-404): on SchemaVersion() error, logs warning and calls recreateCache("schema version error") -- consistent with the existing corruption handling pattern.
  - Mismatch handling (line 405-409): on ver != CurrentSchemaVersion(), logs verbose message and calls recreateCache("schema version mismatch").
  - recreateCache() (lines 135-147) is a well-factored helper that encapsulates the close+nil+delete+reopen sequence, reused for both schema version errors and freshness check corruption.
  - The flow after recreateCache proceeds to IsFresh() which returns false on the empty cache, triggering Rebuild() which stores the correct schema_version. This is the documented acceptable extra cycle for new/incompatible caches.
  - No drift from the plan or spec.

TESTS:
- Status: Adequate
- Coverage:
  - "it deletes and rebuilds cache when schema_version is wrong" (store_test.go:1265): Primes cache, tampers schema_version to 999, opens new store, verifies Query succeeds with correct task count, title, and schema_version after rebuild.
  - "it deletes and rebuilds cache when schema_version row is missing" (store_test.go:1337): Primes cache, deletes schema_version row, opens new store, verifies Query succeeds with correct task count and schema_version.
  - "it preserves cache when schema_version matches" (store_test.go:1400): Primes cache, runs second query, verifies no "schema version mismatch" verbose log was emitted. Confirms no unnecessary rebuild.
  - "it handles SchemaVersion query error gracefully" (store_test.go:1448): Drops and recreates metadata table with wrong schema (broken INTEGER), opens new store, verifies Query succeeds with correct task count and schema_version.
  - All four planned tests are present and match the expected test names exactly.
  - Tests verify both the behavioral outcome (tasks queryable, correct count/title) and the metadata state (schema_version correct after rebuild).
  - Edge case: pre-versioning cache (missing row returning 0) is explicitly tested.
  - Edge case: corrupt metadata table forcing SchemaVersion() error is tested.
- Notes:
  - The "preserves cache" test uses verbose log inspection to confirm no mismatch occurred. This is a reasonable behavioral check -- it verifies the code path rather than implementation details, since the verbose message is a documented feature.
  - Tests are not over-tested: each test covers a distinct scenario with focused assertions. No redundant checks.

CODE QUALITY:
- Project conventions: Followed
  - Error wrapping with fmt.Errorf("context: %w", err) throughout
  - stdlib testing only, t.Run subtests, "it does X" naming
  - Functional options pattern (WithVerbose) used in tests
  - Store.Query/Mutate pattern maintained
- SOLID principles: Good
  - recreateCache() follows single responsibility -- one helper, one job (close+delete+reopen)
  - SchemaVersion() and CurrentSchemaVersion() on cache.go keep version logic in the cache layer
  - ensureFresh() orchestrates the check in store.go, maintaining the existing separation of concerns
- Complexity: Low
  - ensureFresh() adds a single if/else-if block (error vs mismatch) before the existing IsFresh check. No deep nesting.
  - recreateCache() is a simple linear sequence of 4 operations.
- Modern idioms: Yes
  - Proper use of sql.ErrNoRows sentinel
  - strconv.Atoi for parsing stored version string
  - INSERT OR REPLACE for metadata upsert
- Readability: Good
  - Clear inline comment at store.go:395-398 explains the rationale for the version check placement
  - recreateCache "reason" parameter provides context for error messages
  - SchemaVersion() doc comment explains the 0-return convention for missing rows
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The containsSubstring helper (store_test.go:1744) is functionally equivalent to strings.Contains and its extra length checks add no value. strings.Contains("", "") returns true and handles all edge cases. Minor cleanup opportunity.
- recreateCache() at store.go:136 calls s.cache.Close() without checking for nil first. This is safe because all current callers are in contexts where s.cache is guaranteed non-nil (after lazy init or after a successful prior open). However, a nil guard would make it more defensive. Very minor.
