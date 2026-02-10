---
id: tick-core-6-2
phase: 6
status: pending
created: 2026-02-10
---

# Move rebuild logic behind Store abstraction

**Problem**: `RunRebuild` in `internal/cli/rebuild.go` manually acquires a file lock, reads JSONL, opens cache, and rebuilds -- re-implementing the locking, file-reading, and cache-management responsibilities that `Store` encapsulates. This creates a parallel code path that does not share the same error recovery, verbose logging integration, or corruption handling as Store. If Store's locking or freshness logic changes, RunRebuild will not benefit.

**Solution**: Add a `Rebuild` method to `Store` that encapsulates forced-rebuild semantics: exclusive lock, delete cache, read JSONL, rebuild cache. `RunRebuild` then calls `store.Rebuild()` similar to how other commands call `store.Mutate()` or `store.Query()`.

**Outcome**: All storage operations (read, write, rebuild) flow through the Store API. No CLI code directly manages locks, reads JSONL, or manipulates the cache file.

**Do**:
1. Add a `Rebuild(verbose *VerboseLogger) error` method (or similar signature) to `Store` in `internal/storage/store.go`
2. The method should: acquire exclusive lock, delete the existing cache.db file, read tasks.jsonl, create a new cache, populate it, update the hash in metadata, release lock
3. Refactor `RunRebuild` in `internal/cli/rebuild.go` to instantiate a Store and call `store.Rebuild()` instead of directly using low-level storage primitives
4. Ensure verbose logging is preserved (rebuild should log the same messages as before when --verbose is set)
5. Remove any now-unused imports or helper usage from rebuild.go

**Acceptance Criteria**:
- `tick rebuild` produces the same user-visible output and behavior as before
- `RunRebuild` no longer directly uses `flock`, `ReadJSONL`, `OpenCache`, or other low-level storage functions
- All lock management and file operations for rebuild flow through Store
- Existing rebuild tests continue to pass

**Tests**:
- Test that store.Rebuild() successfully rebuilds cache from JSONL
- Test that store.Rebuild() works when cache.db does not exist
- Test that store.Rebuild() works when cache.db is corrupted
- Test that RunRebuild integration still produces correct output
