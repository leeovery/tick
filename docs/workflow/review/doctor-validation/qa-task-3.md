TASK: Cache Staleness Check

ACCEPTANCE CRITERIA:
- CacheStalenessCheck implements the Check interface
- Passing check returns CheckResult with Name "Cache" and Passed true
- Hash mismatch returns error-severity failure with details and "Run `tick rebuild` to refresh cache" suggestion
- Missing cache.db returns error-severity failure with rebuild suggestion
- Missing tasks.jsonl returns error-severity failure with appropriate (non-rebuild) suggestion
- Empty tasks.jsonl with matching hash returns passing result
- Check is read-only -- never modifies tasks.jsonl or cache.db
- Check never triggers a cache rebuild
- Corrupted/unreadable cache.db treated as stale (does not panic)
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines cache staleness as Error #1: "Hash mismatch between JSONL and SQLite cache." The fix suggestion table specifies exact text: "Run `tick rebuild` to refresh cache." Doctor is diagnostic-only and never modifies data. The hash mechanism uses SHA256 of raw tasks.jsonl bytes, stored in SQLite metadata table under key "jsonl_hash". Doctor must replicate the read side of this mechanism but never write.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/cache_staleness.go:1-102
- Notes:
  - CacheStalenessCheck struct (line 18) implements Check interface via Run method (line 22)
  - Run method signature matches interface: `Run(_ context.Context, tickDir string) []CheckResult`
  - Step 1 (line 27-36): Reads tasks.jsonl, returns error result with "Run tick init or verify .tick directory" suggestion if missing
  - Step 2 (line 39-40): Computes SHA256 hash using crypto/sha256 + hex encoding -- matches storage layer's computeHash() in /Users/leeovery/Code/tick/internal/storage/cache.go:174-177
  - Step 3 (line 43-51): Checks cache.db existence, returns "cache.db not found" error with rebuild suggestion
  - Step 4 (line 54-64): Queries stored hash via queryStoredHash helper; corrupted/missing table/key all treated as stale
  - Step 5 (line 67-75): Compares hashes, returns mismatch error if different
  - Passing path (line 77-80): Returns single result with Name "Cache" and Passed true
  - queryStoredHash (line 87-102): Opens cache.db with `?mode=ro` (read-only), queries metadata table, defers db.Close()
  - All return paths produce exactly one CheckResult with Name "Cache"
  - Check is purely read-only: uses os.ReadFile, os.Stat, and read-only SQLite mode
  - No cache rebuild is triggered anywhere in the check
  - Registration confirmed in /Users/leeovery/Code/tick/internal/cli/doctor.go:19

TESTS:
- Status: Adequate
- Coverage: All 12 planned test cases covered plus 2 additional edge cases
- Test file: /Users/leeovery/Code/tick/internal/doctor/cache_staleness_test.go:1-520
- Tests present:
  1. "it returns passing result when tasks.jsonl and cache.db hashes match" (line 104)
  2. "it returns failing result with stale details when hashes do not match" (line 121)
  3. "it returns failing result when cache.db does not exist" (line 141)
  4. "it returns failing result when tasks.jsonl does not exist" (line 159)
  5. "it returns failing result when metadata table has no jsonl_hash key" (line 177)
  6. "it returns passing result for empty tasks.jsonl with matching hash" (line 196) -- uses known SHA256 of empty content
  7. "it suggests Run tick rebuild to refresh cache when cache is stale" (line 214)
  8. "it suggests Run tick rebuild to refresh cache when cache.db is missing" (line 231)
  9. "it uses CheckResult Name Cache for all results" (line 247) -- table-driven across 4 scenarios
  10. "it uses SeverityError for all failure cases" (line 303) -- table-driven across 4 failure scenarios
  11. "it does not modify tasks.jsonl or cache.db (read-only verification)" (line 361) -- compares file bytes before/after
  12. "it returns exactly one CheckResult (single result, not multiple)" (line 399) -- table-driven across 4 scenarios
  13. "it handles corrupted cache.db without panicking" (line 452) -- writes non-SQLite bytes to cache.db
  14. "it returns failing result when tick directory is empty string" (line 475)
  15. "it handles cache.db with no metadata table" (line 493) -- valid SQLite file without metadata table
- Notes:
  - Tests use t.Helper(), t.TempDir(), t.Run() subtests correctly per project conventions
  - Table-driven subtests used where appropriate (Name check, Severity check, single-result check)
  - Helper functions (setupTickDir, writeJSONL, createCacheWithHash, createCacheWithoutHash, computeTestHash) are well-factored and use t.Helper()
  - The assertReadOnly helper is defined but not used by cache staleness tests (the read-only test at line 361 does its own more thorough verification including cache.db comparison, which assertReadOnly does not cover)
  - Tests 9-12 have some redundancy with earlier tests (e.g., Name and Severity already implicitly verified in earlier specific tests), but this is acceptable since these are structured as cross-cutting verification and use table-driven patterns
  - All spec edge cases covered: missing cache.db, missing tasks.jsonl, empty tasks.jsonl with matching hash, hash mismatch, corrupted cache.db, missing metadata table, missing jsonl_hash key

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.TempDir for isolation, error wrapping with %w, functional handler patterns
- SOLID principles: Good -- CacheStalenessCheck has single responsibility (cache freshness verification), implements the Check interface cleanly, queryStoredHash is a focused helper
- Complexity: Low -- linear flow with early returns, no loops, clear step-by-step logic
- Modern idioms: Yes -- uses crypto/sha256, database/sql with DSN URI parameters, deferred Close
- Readability: Good -- well-commented steps (Steps 1-5), descriptive function/variable names, clear separation between file I/O and database querying
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The cache staleness check constructs its own "tasks.jsonl not found or unreadable" message inline rather than using the shared fileNotFoundResult helper. This is intentional per task 4-3 which explicitly notes the cache check has a different message format that includes the OS error detail. This is acceptable.
- The ctxWithTickDir helper in the test file (line 75-77) is a no-op stub that ignores its argument and returns context.Background(). The comment explains it is "retained for test compatibility" after the interface was refactored (task 4-2 made tickDir an explicit parameter). This is a minor vestige but does not affect correctness.
- Minor observation: queryStoredHash could be an unexported method on CacheStalenessCheck rather than a package-level function, but as a package-internal unexported function it is fine.
