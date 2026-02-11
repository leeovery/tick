# Task tick-core-6-3: Consolidate Cache Freshness/Recovery Logic (V6 Only -- Analysis Phase 6)

## Note
This is an analysis refinement task that only exists in V6. Standalone quality assessment, not a comparison.

## Task Summary

**Problem**: `Store.ensureFresh` (store.go) and a standalone `EnsureFresh` function (cache.go) implemented the same ~40-line corruption-recovery-and-rebuild pattern. The standalone function was only used in tests. Having two implementations risked silent divergence.

**Solution**: Remove the standalone `EnsureFresh` from cache.go entirely and migrate all tests to exercise freshness through the Store API (`store.Query()` which triggers `ensureFresh` internally).

**Acceptance Criteria**:
1. The standalone `EnsureFresh` function no longer exists in cache.go
2. All freshness and corruption recovery tests exercise the Store code path
3. No test coverage is lost -- every scenario previously tested through standalone `EnsureFresh` is tested through Store
4. All existing tests pass

**Required Tests**:
- Store handles missing cache.db (rebuilds automatically)
- Store handles corrupted cache.db (deletes and rebuilds)
- Store detects stale cache via hash mismatch and rebuilds
- Store handles freshness check errors (corrupted metadata)

## V6 Implementation

### Architecture & Design

The commit makes a clean, targeted refactoring change. The diff touches exactly 5 files:

- `internal/storage/cache.go`: Removes the standalone `EnsureFresh` function (38 lines deleted) and two now-unused imports (`log`, `os`).
- `internal/storage/cache_test.go`: Removes `TestEnsureFresh` (75 lines deleted, 2 subtests: "missing cache.db" and "corrupted cache.db").
- `internal/storage/store_test.go`: Adds `TestStoreCacheFreshnessRecovery` (227 lines added, 4 subtests covering all required scenarios).

The design decision is sound: the standalone `EnsureFresh` was a package-level function that duplicated logic already present in `Store.ensureFresh` (the method). By removing the standalone function and testing through the Store API, the codebase now has exactly one code path for cache freshness and corruption recovery. This eliminates the divergence risk the plan identified.

The `Store.ensureFresh` method (store.go lines 247-289) remains unchanged and handles all three recovery scenarios:
1. **Cache not yet opened (nil)**: Lazy init with corruption fallback (delete + recreate)
2. **IsFresh fails**: Close, delete, recreate, mark as not-fresh
3. **Stale hash**: Rebuild cache from current JSONL

### Code Quality

**Strengths**:
- The removal is clean -- unused imports (`log`, `os`) were properly cleaned from cache.go.
- No references to the standalone `EnsureFresh` remain anywhere in the codebase (verified via grep).
- The new test function `TestStoreCacheFreshnessRecovery` is well-named and groups all four recovery scenarios logically under one test function.
- Test subtests use descriptive names that map directly to the acceptance criteria.
- Error handling in tests uses `t.Fatalf` for setup failures and `t.Errorf` for assertion failures, which is the correct Go convention (fail-fast for preconditions, continue for assertions).
- Variable naming avoids shadowing: `wErr`, `cErr`, `qErr` are used for inner-scope errors to avoid conflict with the outer `err`.

**Style compliance**:
- All tests follow the `t.Run("it <does thing>", ...)` subtest pattern consistent with the rest of the codebase.
- Comments explain *why* specific operations are performed (e.g., "Corrupt the metadata table by dropping and replacing it with an incompatible schema").
- The tests use the existing `setupTickDirWithTasks` helper and `WriteJSONL` function rather than duplicating setup logic.

### Test Coverage

All four required test scenarios are present:

| Required Test | Implementation | Status |
|---|---|---|
| Missing cache.db | `"it rebuilds automatically when cache.db is missing"` -- removes cache.db, calls `store.Query`, verifies task count | Covered |
| Corrupted cache.db | `"it deletes and rebuilds when cache.db is corrupted"` -- writes garbage bytes, calls `store.Query`, verifies recovery | Covered |
| Stale cache via hash mismatch | `"it detects stale cache via hash mismatch and rebuilds"` -- primes cache, modifies JSONL externally, verifies both task count and updated title | Covered |
| Freshness check errors (corrupted metadata) | `"it handles freshness check errors from corrupted metadata"` -- primes cache, drops/replaces metadata table with incompatible schema, reopens store, verifies recovery | Covered |

**Coverage analysis of migrated tests**:
The old `TestEnsureFresh` had 2 subtests:
1. "it rebuilds from scratch when cache.db is missing" -- now covered by `TestStoreCacheFreshnessRecovery/"it rebuilds automatically when cache.db is missing"`
2. "it deletes and recreates cache.db when corrupted" -- now covered by `TestStoreCacheFreshnessRecovery/"it deletes and rebuilds when cache.db is corrupted"`

The new test suite adds 2 additional scenarios (stale hash, corrupted metadata) that were not tested by the old `TestEnsureFresh`, representing a net increase in test coverage.

**Test design quality**:
- The stale-hash test is particularly thorough: it verifies both the count of tasks (2) and that the updated title reflects the new data, confirming the rebuild used the current JSONL content.
- The corrupted-metadata test is well-crafted: it opens a second SQLite connection to corrupt the metadata table schema, closes the store to force reopen, then verifies recovery. This tests a realistic corruption scenario where the schema exists but is incompatible.

### Spec Compliance

All four acceptance criteria are fully met:

1. **Standalone `EnsureFresh` removed**: Confirmed via code review of `cache.go` and grep across the entire `internal/` tree. Zero references remain.
2. **Tests exercise Store code path**: All four tests create a `Store`, call `store.Query()`, and verify results through the Store's cache. No direct `Cache` freshness testing remains.
3. **No test coverage lost**: Both old scenarios (missing cache, corrupted cache) are migrated. Two new scenarios are added (stale hash, corrupted metadata).
4. **All existing tests pass**: The commit was accepted, indicating tests passed (though this cannot be re-verified from the diff alone).

### golang-pro Skill Compliance

| Requirement | Status | Notes |
|---|---|---|
| Handle all errors explicitly | Pass | Every error is checked; no naked returns |
| Write table-driven tests with subtests | Partial | Uses subtests (`t.Run`) but not table-driven format. However, the scenarios are sufficiently distinct that table-driven would not improve clarity |
| Document all exported functions/types/packages | N/A | No new exported symbols were added; the change removed one |
| Propagate errors with `fmt.Errorf("%w", err)` | Pass | Existing `ensureFresh` uses `%w` wrapping throughout |
| No panic for normal error handling | Pass | No panics |
| No goroutines without lifecycle management | N/A | No goroutines involved |
| No ignored errors without justification | Pass | All errors handled |

## Quality Assessment

### Strengths

1. **Precise scope**: The change does exactly what the plan specifies -- no more, no less. It removes exactly one function and its tests, replaces them with Store-level tests, and cleans up unused imports.
2. **Net positive test coverage**: Not only were the two existing scenarios migrated, but two new scenarios were added (stale hash detection, corrupted metadata recovery), improving overall resilience testing.
3. **The corrupted-metadata test is especially well-designed**: It tests a subtle failure mode (IsFresh query error due to schema mismatch) that the old standalone `EnsureFresh` tests did not cover. The technique of opening a second SQLite connection to corrupt the schema is realistic and effective.
4. **Clean diff**: No unnecessary changes, no formatting-only changes, no unrelated modifications. The import cleanup in cache.go demonstrates attention to detail.
5. **Consistent style**: The new tests match the existing test conventions in the file (naming, helpers, error handling patterns).

### Weaknesses

1. **Minor: Test setup repetition**: The task/created-time boilerplate is repeated across all 4 subtests. A shared test fixture or table at the top of `TestStoreCacheFreshnessRecovery` could reduce repetition. This is a minor style point -- the current approach is explicit and readable.
2. **Minor: The corrupted-metadata test is somewhat complex**: It requires closing the store, opening a second DB connection, corrupting the schema, closing that connection, then reopening the store. While well-commented, this multi-step setup could be fragile if Store internals change. However, this complexity is inherent to testing the scenario and the comments adequately explain each step.
3. **Commit message typo**: The commit message reads "Ttick-core-6-3" (double T). This is cosmetic and does not affect code quality.

### Overall Quality Rating

**Excellent** -- This is a clean, well-scoped refactoring that eliminates code duplication, migrates all existing test coverage to the correct abstraction layer, adds new test scenarios beyond what was previously covered, and leaves the codebase in a strictly better state. The implementation precisely follows the plan with no deviations and no omissions.
