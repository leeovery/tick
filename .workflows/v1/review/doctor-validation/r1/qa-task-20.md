TASK: Extract assertReadOnly test helper

ACCEPTANCE CRITERIA:
- assertReadOnly helper exists in a shared test file in internal/doctor/
- All 10 read-only verification tests use the helper instead of inline logic
- All tests pass. No test behavior changes
- Helper uses t.Helper() for proper error attribution

STATUS: Issues Found

SPEC CONTEXT: Doctor is diagnostic only -- it reports problems and suggests remedies but never modifies data. The read-only verification tests confirm each check upholds this invariant. There are 10 checks (9 errors + 1 warning), each of which should have a read-only test.

IMPLEMENTATION:
- Status: Partial
- Location: `/Users/leeovery/Code/tick/internal/doctor/cache_staleness_test.go:79-101` (helper definition)
- Notes:
  - The `assertReadOnly` helper exists and is well-implemented: it writes content, reads before bytes, runs the check closure, reads after bytes, compares, and uses `t.Helper()`.
  - 9 of 10 check test files use the helper correctly via closure pattern.
  - `cache_staleness_test.go` (lines 361-397) still uses inline read-only verification logic. It checks both `tasks.jsonl` and `cache.db`, which the `assertReadOnly` helper does not support. This is a legitimate deviation since the cache staleness check is the only check that has a cache.db file to verify, but the acceptance criteria say "All 10 read-only verification tests use the helper instead of inline logic."
  - The helper is defined in `cache_staleness_test.go` rather than `helpers_test.go`. The task says "Create or identify an existing shared test helper file in internal/doctor/ (e.g., helpers_test.go if it exists, or the file where setupTickDir and writeJSONL already live)." Since `setupTickDir` and `writeJSONL` also live in `cache_staleness_test.go`, this meets the "or the file where..." clause. However, `helpers_test.go` exists and would be the more natural home for shared helpers -- having shared helpers scattered in a single check's test file is misleading.

  Files using the helper:
  - `/Users/leeovery/Code/tick/internal/doctor/orphaned_parent_test.go:326`
  - `/Users/leeovery/Code/tick/internal/doctor/jsonl_syntax_test.go:344`
  - `/Users/leeovery/Code/tick/internal/doctor/id_format_test.go:447`
  - `/Users/leeovery/Code/tick/internal/doctor/duplicate_id_test.go:354`
  - `/Users/leeovery/Code/tick/internal/doctor/orphaned_dependency_test.go:376`
  - `/Users/leeovery/Code/tick/internal/doctor/self_referential_dep_test.go:315`
  - `/Users/leeovery/Code/tick/internal/doctor/dependency_cycle_test.go:491`
  - `/Users/leeovery/Code/tick/internal/doctor/child_blocked_by_parent_test.go:389`
  - `/Users/leeovery/Code/tick/internal/doctor/parent_done_open_children_test.go:466`
  - `/Users/leeovery/Code/tick/internal/doctor/task_relationships_test.go:231` (bonus -- not one of the 10 checks)

  File NOT using the helper:
  - `/Users/leeovery/Code/tick/internal/doctor/cache_staleness_test.go:361-397` (inline logic, also checks cache.db)

TESTS:
- Status: Adequate
- Coverage: All 10 checks have read-only verification tests. 9 use the helper, 1 is inline. The helper is also used by the task_relationships_test.go utility.
- Notes: The existing tests still verify the correct behavior. No test behavior was changed in the 9 files that were migrated. The cache staleness inline test is functionally correct and tests more (cache.db), so it's not a regression.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Helper(), t.Run() subtests
- SOLID principles: Good -- helper has single responsibility (verify file not modified)
- Complexity: Low -- helper is 19 lines, clear logic
- Modern idioms: Yes -- closure pattern for check injection is idiomatic Go
- Readability: Good -- helper is self-documenting with clear doc comment
- Issues:
  - Helper placement in `cache_staleness_test.go` alongside check-specific tests is suboptimal. Other shared helpers (`setupTickDir`, `writeJSONL`, `ctxWithTickDir`, `computeTestHash`, `createCacheWithHash`, `createCacheWithoutHash`) are also in this file, making it a mixed-purpose file. The project already has `helpers_test.go` which only contains `TestFileNotFoundResult`. Moving shared test utilities to `helpers_test.go` (or a dedicated `testhelpers_test.go`) would improve discoverability.

BLOCKING ISSUES:
- cache_staleness_test.go still uses inline read-only verification (lines 361-397) instead of the assertReadOnly helper. The acceptance criteria require "All 10 read-only verification tests use the helper." Either the cache staleness test should be refactored to use the helper (potentially by extending the helper to optionally verify additional files), or the acceptance criteria should be clarified to exclude the cache staleness special case.

NON-BLOCKING NOTES:
- Consider moving all shared test helpers (setupTickDir, writeJSONL, assertReadOnly, ctxWithTickDir, computeTestHash) from cache_staleness_test.go to helpers_test.go. This would make cache_staleness_test.go purely about cache staleness tests, and helpers_test.go the canonical location for shared test infrastructure. Currently helpers_test.go only tests `fileNotFoundResult` which is somewhat inconsistent.
- The assertReadOnly helper could be extended with a variadic parameter for additional file paths to verify (e.g., `assertReadOnly(t, tickDir, content, runCheck, "cache.db")`) which would let cache_staleness_test.go use it too.
