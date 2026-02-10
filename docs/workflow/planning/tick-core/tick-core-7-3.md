---
id: tick-core-7-3
phase: 7
status: pending
created: 2026-02-10
---

# Extract store-opening boilerplate into shared helper

**Problem**: Every Run* function that accesses the store repeats the same 8-line sequence: `DiscoverTickDir(dir)`, `storage.NewStore(tickDir, storeOpts(fc)...)`, `defer store.Close()`. This identical block appears 9 times across 8 files (dep.go has it twice). Each instance uses the same arguments (dir, fc) and produces the same local variables (tickDir, store, err). The pattern is mechanical boilerplate with no variation.

**Solution**: Extract an `openStore(dir string, fc FormatConfig) (*storage.Store, func(), error)` helper that encapsulates DiscoverTickDir + NewStore. Return a cleanup function or let callers defer store.Close() themselves. Callers reduce from 8 lines to approximately 3 lines.

**Outcome**: Store opening logic exists in one place. If the initialization sequence changes (e.g., adding a new option, changing DiscoverTickDir behavior), only one location needs updating. Approximately 45 lines of boilerplate eliminated across 9 call sites.

**Do**:
1. Create a helper function `openStore(dir string, fc FormatConfig) (*storage.Store, error)` in an appropriate shared file (e.g., `internal/cli/helpers.go` or `internal/cli/store_helpers.go`)
2. The function calls `DiscoverTickDir(dir)` and `storage.NewStore(tickDir, storeOpts(fc)...)`, returning the store or any error
3. Replace the boilerplate in all 9 call sites: create.go, dep.go (twice), list.go, rebuild.go, show.go, stats.go, transition.go, update.go
4. Each call site retains its own `defer store.Close()` since Go defers are scope-bound
5. Run all tests to verify no behavioral changes

**Acceptance Criteria**:
- No inline DiscoverTickDir + NewStore sequence remains in any Run* function
- All 9 call sites use the shared openStore helper
- Each call site still has its own defer store.Close()
- All existing tests pass unchanged

**Tests**:
- Test openStore returns a valid store for a valid tick directory
- Test openStore returns appropriate error when no .tick directory exists
- Test that all commands still function correctly after refactor (covered by existing integration tests)
