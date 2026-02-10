---
id: tick-core-6-3
phase: 6
status: pending
created: 2026-02-10
---

# Consolidate cache freshness/recovery logic

**Problem**: `Store.ensureFresh` (store.go lines 190-233) and the standalone `EnsureFresh` function (cache.go lines 177-209) implement the same ~40-line pattern: open cache (recover on error by deleting and reopening), check freshness (recover on error by closing/deleting/reopening), rebuild if stale. The standalone function is only used in tests. These two implementations could diverge silently.

**Solution**: Either remove the standalone `EnsureFresh` from cache.go entirely (testing freshness through Store) or extract the shared corruption-recovery-and-rebuild logic into a single private helper both can call. Since `Store.ensureFresh` is the runtime path, prefer removing the standalone function and adjusting tests to use the Store API.

**Outcome**: One code path for cache freshness and corruption recovery. No risk of the two implementations diverging.

**Do**:
1. Check which tests use the standalone `EnsureFresh` function in cache.go
2. Migrate those tests to exercise freshness through the Store API (e.g., `store.Query()` which triggers `ensureFresh` internally)
3. Remove the standalone `EnsureFresh` function from cache.go
4. If any non-test code references `EnsureFresh`, refactor to use Store instead
5. Verify all existing tests pass after removal

**Acceptance Criteria**:
- The standalone `EnsureFresh` function no longer exists in cache.go
- All freshness and corruption recovery tests exercise the Store code path
- No test coverage is lost -- every scenario previously tested through standalone EnsureFresh is tested through Store
- All existing tests pass

**Tests**:
- Test Store handles missing cache.db (rebuilds automatically)
- Test Store handles corrupted cache.db (deletes and rebuilds)
- Test Store detects stale cache via hash mismatch and rebuilds
- Test Store handles freshness check errors (corrupted metadata)
