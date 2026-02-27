TASK: Move rebuild logic behind Store abstraction (tick-core-6-2)

ACCEPTANCE CRITERIA:
- `tick rebuild` produces the same user-visible output and behavior as before
- `RunRebuild` no longer directly uses `flock`, `ReadJSONL`, `OpenCache`, or other low-level storage functions
- All lock management and file operations for rebuild flow through Store
- Existing rebuild tests continue to pass

STATUS: Complete

SPEC CONTEXT: The specification defines `tick rebuild` as a forced complete rebuild of the SQLite cache from JSONL, bypassing freshness checks. The rebuild command acquires an exclusive lock, deletes cache.db, reads tasks.jsonl, creates a fresh cache, populates it, and updates the hash in metadata. The spec states all storage operations should flow through coordinated lock management (section: Synchronization, Cache Rebuild Triggers, Write Operations).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/storage/store.go:177-226` - `Store.Rebuild()` method
  - `/Users/leeovery/Code/tick/internal/cli/rebuild.go:1-30` - `RunRebuild` refactored to use Store
- Notes:
  - `Store.Rebuild()` correctly encapsulates the full rebuild flow: acquire exclusive lock, close existing cache, delete cache.db, read JSONL, parse tasks, open fresh cache, rebuild with tasks, update hash, return count.
  - `RunRebuild` is now 18 lines of pure CLI orchestration: open store, call `store.Rebuild()`, format output. No low-level storage primitives remain.
  - Imports in `rebuild.go` are only `fmt` and `io` -- no `flock`, `storage.OpenCache`, `storage.ReadJSONL`, or `os` imports.
  - Verbose logging is wired through `openStore` -> `storeOpts` -> `storage.WithVerbose`, preserving all debug messages (deleting cache.db, reading JSONL, rebuilding cache with N tasks, hash updated).
  - The `Store.Rebuild()` method returns `(int, error)` providing the task count for the confirmation message, which is a clean API design choice.

TESTS:
- Status: Adequate
- Coverage:
  - **Store-level tests** (`/Users/leeovery/Code/tick/internal/storage/store_test.go:500-739`):
    - Rebuilds cache from JSONL and returns correct task count (line 501)
    - Works when cache.db does not exist (line 539)
    - Works when cache.db is corrupted (line 570)
    - Updates hash in metadata after rebuild (line 613)
    - Acquires exclusive lock during rebuild -- verifies lock timeout error (line 650)
    - Logs verbose messages during rebuild -- verifies exact message sequence (line 678)
    - Handles empty JSONL returning 0 tasks (line 722)
  - **CLI-level tests** (`/Users/leeovery/Code/tick/internal/cli/rebuild_test.go:32-322`):
    - Rebuilds cache from JSONL (line 35)
    - Handles missing cache.db / fresh build (line 75)
    - Overwrites valid existing cache (line 103)
    - Updates hash in metadata table after rebuild (line 152)
    - Acquires exclusive lock during rebuild (line 185)
    - Outputs confirmation message with task count (line 210)
    - Suppresses output with --quiet (line 229)
    - Logs rebuild steps with --verbose (line 251)
    - Handles empty JSONL with 0 tasks rebuilt (line 293)
  - All four test scenarios from the task's "Tests" section are covered:
    1. store.Rebuild() successfully rebuilds cache from JSONL -- covered
    2. store.Rebuild() works when cache.db does not exist -- covered
    3. store.Rebuild() works when cache.db is corrupted -- covered
    4. RunRebuild integration still produces correct output -- covered
- Notes: Tests are well-structured with clear subtests. No over-testing detected; each test verifies a distinct scenario. The verbose message test at the store level verifies the exact sequence and count of messages, which is thorough without being excessive.

CODE QUALITY:
- Project conventions: Followed. Table-driven-style subtests with t.Run, proper error handling, t.Helper on test helpers, defer for cleanup.
- SOLID principles: Good. Store.Rebuild() has single responsibility (forced rebuild). The method cleanly extends the Store API alongside Mutate() and Query(). RunRebuild delegates entirely to Store -- no mixed responsibilities.
- Complexity: Low. Store.Rebuild() is a linear sequence of steps with clear error handling at each point. No branching complexity.
- Modern idioms: Yes. Uses functional options (WithVerbose), deferred unlock, proper error wrapping with fmt.Errorf and %w.
- Readability: Good. The method is well-commented with clear step-by-step flow. The CLI function is minimal and self-explanatory.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
