TASK: tick rebuild command (tick-core-5-2)

ACCEPTANCE CRITERIA:
- [ ] Rebuilds SQLite from JSONL regardless of current freshness
- [ ] Deletes existing cache before rebuild
- [ ] Updates hash in metadata table
- [ ] Acquires exclusive lock during rebuild
- [ ] Handles missing cache.db without error
- [ ] Handles empty JSONL (0 tasks rebuilt)
- [ ] Outputs confirmation with task count
- [ ] --quiet suppresses output
- [ ] --verbose logs rebuild steps to stderr

STATUS: Complete

SPEC CONTEXT: Spec section "Rebuild Command" states: "tick rebuild forces a complete rebuild of the SQLite cache from JSONL, bypassing the freshness check. Use when SQLite appears corrupted, debugging cache issues, or after manual JSONL edits. Output: Confirmation message showing tasks rebuilt." The rebuild is cache trigger #4 from the Synchronization section.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - CLI handler: /Users/leeovery/Code/tick/internal/cli/rebuild.go:1-30
  - App dispatch: /Users/leeovery/Code/tick/internal/cli/app.go:86-87 (case "rebuild") and :184-191 (handleRebuild)
  - Store.Rebuild(): /Users/leeovery/Code/tick/internal/storage/store.go:177-226
  - Cache.Rebuild(): /Users/leeovery/Code/tick/internal/storage/cache.go:73-155
- Notes:
  - Clean separation of concerns: CLI handler (rebuild.go) delegates to Store.Rebuild() which handles lock, delete, read, parse, create, populate, hash.
  - Store.Rebuild() acquires exclusive lock (line 181), closes existing cache (188-191), deletes cache.db (194-197), reads JSONL (201-203), parses (206-208), opens fresh cache (212-216), rebuilds with hash (219-223).
  - CLI handler checks fc.Quiet to suppress output (line 24), uses fmtr.FormatMessage for confirmation (line 26).
  - Verbose logging integrated via Store's verboseLog callback at each step: "deleting cache.db", "reading JSONL", "rebuilding cache with N tasks", "hash updated" plus lock acquire/release messages.
  - All acceptance criteria are met in the implementation.

TESTS:
- Status: Adequate
- Coverage:
  - CLI-level tests in /Users/leeovery/Code/tick/internal/cli/rebuild_test.go (8 test cases):
    1. "it rebuilds cache from JSONL" - verifies cache.db created, task count in SQLite, count in stdout
    2. "it handles missing cache.db (fresh build)" - removes cache.db first, verifies creation
    3. "it overwrites valid existing cache" - builds cache, modifies JSONL, rebuilds, verifies new count
    4. "it updates hash in metadata table after rebuild" - queries metadata table, verifies 64-char SHA256
    5. "it acquires exclusive lock during rebuild" - holds external lock, expects exit code 1 and stderr mention of "lock"
    6. "it outputs confirmation message with task count" - 3 tasks, expects exact "Cache rebuilt: 3 tasks\n"
    7. "it suppresses output with --quiet" - verifies stdout empty, cache still created
    8. "it logs rebuild steps with --verbose" - checks stderr for "deleting cache.db", "reading JSONL", "rebuilding cache with", "hash updated"; verifies "verbose: " prefix on all lines; verifies stdout still has confirmation
  - Store-level tests in /Users/leeovery/Code/tick/internal/storage/store_test.go (TestStoreRebuild, 7 test cases):
    1. "it rebuilds cache from JSONL and returns task count"
    2. "it works when cache.db does not exist"
    3. "it works when cache.db is corrupted"
    4. "it updates hash in metadata after rebuild"
    5. "it acquires exclusive lock during rebuild"
    6. "it logs verbose messages during rebuild"
    7. "it handles empty JSONL returning 0 tasks"
  - Edge cases covered: missing cache.db, valid cache overwritten, empty JSONL (0 tasks), concurrent access (lock held), corrupted cache.db
  - Tests use real file system (temp dirs), real SQLite, real file locks -- genuine integration-style tests
  - Test names match the planned test list from the task file exactly (all 8 CLI tests present)
  - Additional edge case from task (empty JSONL) covered as test #8 in CLI and #7 in store
- Notes: Test coverage is thorough. Both layers (CLI and storage) are tested independently. No over-testing detected -- each test verifies a distinct behavior. The verbose test checks specific message contents and prefix format, which is appropriate for verifying the logging contract.

CODE QUALITY:
- Project conventions: Followed. Table-driven subtests where applicable. Test helpers use t.Helper(). Error handling is explicit. Exported functions are documented.
- SOLID principles: Good. Single responsibility -- CLI handler only handles I/O and formatting, Store handles locking and data operations, Cache handles SQLite operations. Dependency inversion via FormatConfig/Formatter interface.
- Complexity: Low. RunRebuild is 18 lines. Store.Rebuild() is 46 lines with clear sequential flow. No branching complexity.
- Modern idioms: Yes. Functional options for Store configuration (WithVerbose, WithLockTimeout). Defer for cleanup. Error wrapping with %w.
- Readability: Good. Clear variable names, well-commented steps, logical flow. The rebuild.go handler is especially clean -- open store, rebuild, output.
- Issues: None found.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The plan mentions "Malformed JSONL lines: skip with warning to stderr" as an edge case, but the actual behavior delegates to ParseJSONL which returns an error on malformed lines rather than skipping them. This is noted in the task itself as "behavior defined by JSONL reader from tick-core-1-2 -- implementer decides error handling strategy there", so it is not a drift from this task's scope.
- The "concurrent access" edge case from the task spec ("exclusive lock prevents reads during rebuild") is tested by holding an external lock and verifying rebuild fails. A more thorough test would verify that a concurrent read is blocked during an active rebuild, but the current lock test adequately proves the exclusive lock is acquired.
