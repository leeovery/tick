# Task tick-core-6-2: Move Rebuild Logic Behind Store Abstraction (V6 Only -- Analysis Phase 6)

## Note
This is an analysis refinement task that only exists in V6. Standalone quality assessment, not a comparison.

## Task Summary

**Problem**: `RunRebuild` in `internal/cli/rebuild.go` manually acquires a file lock, reads JSONL, opens cache, and rebuilds -- re-implementing the locking, file-reading, and cache-management responsibilities that `Store` encapsulates. This creates a parallel code path that does not share the same error recovery, verbose logging integration, or corruption handling as Store.

**Solution**: Add a `Rebuild()` method to `Store` that encapsulates forced-rebuild semantics: exclusive lock, delete cache, read JSONL, rebuild cache. Refactor `RunRebuild` to delegate entirely to `store.Rebuild()`.

**Acceptance Criteria**:
1. `tick rebuild` produces the same user-visible output and behavior as before
2. `RunRebuild` no longer directly uses `flock`, `ReadJSONL`, `OpenCache`, or other low-level storage functions
3. All lock management and file operations for rebuild flow through Store
4. Existing rebuild tests continue to pass

**Required Tests**:
- `store.Rebuild()` successfully rebuilds cache from JSONL
- `store.Rebuild()` works when cache.db does not exist
- `store.Rebuild()` works when cache.db is corrupted
- `RunRebuild` integration still produces correct output

## V6 Implementation

### Architecture & Design

The refactoring is clean and follows the existing `Store` API patterns precisely. The new `Rebuild() (int, error)` method sits alongside `Mutate()` and `Query()` as a third top-level Store operation, forming a natural triad: read, write, rebuild. All three share the same locking pattern (acquire lock, do work, release lock via defer), which makes the Store API internally consistent.

Key design decisions:
- **Returns `(int, error)` instead of just `error`**: The count return value allows the CLI layer to report the number of tasks rebuilt without needing a separate query. This is a pragmatic choice that keeps `rebuild.go` trivially simple.
- **Closes existing cache before delete**: Lines 162-165 properly handle the case where a cache connection is already open before deleting the file, preventing SQLite resource leaks.
- **Reassigns `s.cache` after creating a fresh one**: Line 188 ensures subsequent `Query()` or `Close()` calls on the same Store instance operate on the newly created cache.
- **Exclusive lock (not shared)**: Correctly uses `TryLockContext` (exclusive), not `TryRLockContext` (shared), matching the destructive nature of the operation.

The resulting `RunRebuild` in `rebuild.go` is reduced from 79 lines to 35 lines (at commit time), with all low-level storage imports (`flock`, `os`, `filepath`, `context`, `time`) removed. The function now reads as a straightforward orchestration: discover dir, open store, rebuild, print result.

### Code Quality

**Strengths**:
- Excellent error wrapping with `fmt.Errorf("failed to X: %w", err)` throughout, consistent with `Mutate()` and `Query()`.
- The verbose logging calls mirror the exact same messages the old `RunRebuild` emitted (acquiring lock, deleting cache, reading JSONL, rebuilding, hash updated, lock released), preserving backward-compatible verbose output.
- The `lockErrMsg` constant is reused across all three operations rather than duplicated.
- Unchecked `os.Remove(s.cachePath)` on line 169 is intentional and consistent -- the file may not exist, and that is acceptable.

**Minor observations**:
- The error from `s.cache.Close()` on line 163 is silently ignored. In the `Rebuild` context this is acceptable since we are about to delete the file anyway, but a verbose log noting the close would be marginally more observable.
- The verbose message `"rebuilding cache with 1 tasks"` is grammatically imprecise for count=1 ("1 tasks" vs "1 task"), but this is pre-existing behavior from the original code and not introduced by this task.

### Test Coverage

**Store-level tests** (`internal/storage/store_test.go`): 7 new subtests under `TestStoreRebuild`:

| Test | What it covers |
|------|----------------|
| "rebuilds cache from JSONL and returns task count" | Happy path with 2 tasks, verifies count and cache contents via Query |
| "works when cache.db does not exist" | No pre-existing cache file |
| "works when cache.db is corrupted" | Garbage bytes in cache.db, verifies recovery |
| "updates hash in metadata after rebuild" | Verifies SHA256 hash stored in metadata table |
| "acquires exclusive lock during rebuild" | External lock held, verifies timeout error message |
| "logs verbose messages during rebuild" | Captures verbose callback, asserts exact message sequence and count |
| "handles empty JSONL returning 0 tasks" | Edge case with zero tasks |

**CLI-level tests** (`internal/cli/rebuild_test.go`): 8 subtests under `TestRebuild`:

| Test | What it covers |
|------|----------------|
| "rebuilds cache from JSONL" | End-to-end through `App.Run` |
| "handles missing cache.db" | Fresh build scenario |
| "overwrites valid existing cache" | Double rebuild with JSONL mutation between |
| "updates hash in metadata table" | SHA256 verification at integration level |
| "acquires exclusive lock" | Lock contention with exit code verification |
| "outputs confirmation message with task count" | Exact stdout match |
| "suppresses output with --quiet" | Quiet flag behavior |
| "logs rebuild steps with --verbose" | Verbose to stderr, prefix check, expected messages |

This is thorough coverage at both the unit (Store) and integration (CLI) levels. Each acceptance criterion from the plan is directly tested. The tests are well-structured with clear setup, assertion, and cleanup patterns.

### Spec Compliance

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Same user-visible output and behavior | PASS | CLI tests verify exact stdout, verbose stderr messages, quiet mode |
| RunRebuild no longer uses low-level storage functions | PASS | `rebuild.go` imports only `fmt`, `io`, and `storage` -- no `flock`, `os`, `filepath`, `context`, or `time` |
| All lock/file ops for rebuild flow through Store | PASS | `Store.Rebuild()` handles lock, delete, read, cache create, populate, hash update |
| Existing rebuild tests continue to pass | PASS | CLI-level `TestRebuild` still present and exercised |
| Test: Rebuild from JSONL | PASS | Both store and CLI level |
| Test: No existing cache.db | PASS | Both store and CLI level |
| Test: Corrupted cache.db | PASS | Store-level test with garbage bytes |
| Test: RunRebuild integration | PASS | Full CLI integration test suite |

All acceptance criteria and required tests are satisfied.

### golang-pro Skill Compliance

| Requirement | Status | Notes |
|-------------|--------|-------|
| context.Context on blocking ops | PASS | Lock acquisition uses `context.WithTimeout` |
| Handle all errors explicitly | PASS | Every error is checked and wrapped or returned |
| Table-driven tests with subtests | PASS | All tests use `t.Run()` subtests |
| Document exported functions/types | PASS | `Rebuild()` has full godoc comment |
| Propagate errors with `fmt.Errorf("%w")` | PASS | All error returns use `%w` wrapping |
| No panic for error handling | PASS | No panics anywhere |
| No goroutines without lifecycle management | N/A | No goroutines introduced |
| No hardcoded configuration | PASS | Lock timeout configurable via `WithLockTimeout` |

## Quality Assessment

### Strengths
- **Exemplary refactoring discipline**: The task achieves exactly what it promises -- moving low-level storage operations behind the Store abstraction -- without scope creep or unnecessary changes.
- **Consistent API design**: `Rebuild()` follows the same structural pattern as `Mutate()` and `Query()` (lock, work, unlock), making the Store API predictable and easy to reason about.
- **Dramatic simplification of CLI code**: `rebuild.go` is reduced to a thin orchestration layer with zero storage-related imports, which is the ideal boundary between CLI and storage concerns.
- **Comprehensive test coverage at two levels**: Unit tests on Store.Rebuild test the mechanism; integration tests on RunRebuild test the behavior. Edge cases (empty JSONL, missing cache, corrupted cache, lock contention) are covered at both levels.
- **Verbose logging preserved**: The exact same verbose messages are emitted in the same order, ensuring observable behavior is unchanged.

### Weaknesses
- **Minor**: `s.cache.Close()` error on line 163 of `store.go` is silently discarded. While acceptable in context (file is about to be deleted), a verbose log would aid debugging.
- **Minor**: The `"1 tasks"` grammar issue in verbose output is pre-existing but could have been addressed opportunistically.
- **Trivial**: No test verifies that `Rebuild()` can be called multiple times on the same Store instance (double-rebuild scenario), though the CLI-level "overwrites valid existing cache" test effectively covers this path.

### Overall Quality Rating

**Excellent** -- This is a textbook refactoring task executed with precision. The implementation correctly moves all low-level storage operations behind the Store abstraction, the resulting CLI code is minimal and focused, the API design is consistent with existing patterns, and the test coverage is thorough at both unit and integration levels. All acceptance criteria are satisfied without any functional issues.
