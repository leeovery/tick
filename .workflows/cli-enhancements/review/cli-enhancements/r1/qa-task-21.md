TASK: cli-enhancements-4-6 -- Notes table in SQLite schema and Cache.Rebuild

ACCEPTANCE CRITERIA:
- SQLite `task_notes(task_id, text, created)` table populated during `Cache.Rebuild()`

EDGE CASES FROM PLAN:
- Task with empty notes slice
- Rebuild clearing stale notes
- Note ordering preserved

STATUS: Complete

SPEC CONTEXT:
The specification (Notes section) defines: JSONL stores `[]Note` with `omitempty`; SQLite uses a `task_notes(task_id, text, created)` table; notes are displayed chronologically. The `Note` type has `Text string` and `Created time.Time`. This task specifically covers the SQLite schema definition and the cache rebuild logic for populating notes into the table.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Schema: `/Users/leeovery/Code/tick/internal/storage/cache.go:48-52` -- `task_notes` table DDL with columns `task_id TEXT NOT NULL`, `text TEXT NOT NULL`, `created TEXT NOT NULL`
  - Index: `/Users/leeovery/Code/tick/internal/storage/cache.go:63` -- `idx_task_notes_task_id` index
  - Rebuild clear: `/Users/leeovery/Code/tick/internal/storage/cache.go:107-109` -- `DELETE FROM task_notes` at start of rebuild
  - Rebuild insert prepare: `/Users/leeovery/Code/tick/internal/storage/cache.go:148-152` -- prepared statement for note insertion
  - Rebuild insert loop: `/Users/leeovery/Code/tick/internal/storage/cache.go:209-213` -- iterates `t.Notes` and inserts each with `task.FormatTimestamp(note.Created)`
- Notes:
  - Table correctly has no composite PRIMARY KEY, unlike `task_tags` and `task_refs`. This is correct because a task can have duplicate note text at different timestamps (or even same timestamp). The rowid-based implicit ordering preserves insertion order.
  - The `DELETE FROM task_notes` runs before task inserts, ensuring stale data is cleared on rebuild.
  - Timestamp is formatted via `task.FormatTimestamp()` which produces ISO 8601 UTC strings, consistent with all other timestamp storage.
  - The prepared statement is properly closed via `defer insertNote.Close()`.
  - The entire rebuild runs in a single transaction, ensuring atomicity.

TESTS:
- Status: Adequate
- Coverage:
  1. Schema test (`cache_test.go:113-142`): Verifies `task_notes` table exists with correct columns (`task_id`, `text`, `created`) and that `idx_task_notes_task_id` index is created.
  2. Populate test (`cache_test.go:774-848`): Rebuilds with a task having 2 notes, then queries `task_notes` and verifies all three columns (task_id, text, created) for both rows. Validates timestamp formatting.
  3. Empty notes slice test (`cache_test.go:850-884`): Rebuilds a task with no notes, verifies 0 rows in `task_notes`. Covers the "empty notes slice" edge case.
  4. Stale notes clearing test (`cache_test.go:886-953`): Performs two rebuilds -- first with 2 notes, second with 1 different note. Verifies only 1 row remains with the new text. Covers "rebuild clearing stale notes" edge case.
  5. Ordering test (`cache_test.go:955-1015`): Inserts 3 notes with timestamps deliberately out of chronological order, verifies that `ORDER BY rowid` returns them in insertion (slice) order, not timestamp order. Covers "note ordering preserved" edge case.
- Notes: All three edge cases from the plan are directly tested. Tests query the SQLite database directly to verify behavior -- appropriate for a storage-layer task. Tests would fail if the feature broke (e.g., if notes loop were removed or table schema changed).

CODE QUALITY:
- Project conventions: Followed. Uses `t.Run()` subtests with "it does X" naming, `t.TempDir()` for isolation, `t.Helper()` on shared helpers, stdlib testing only (no testify). Error wrapping uses `fmt.Errorf("context: %w", err)` pattern.
- SOLID principles: Good. The `Rebuild` method handles all table population in a single transaction -- single responsibility for the cache rebuild operation. The note insertion follows the exact same pattern as dependencies, tags, and refs (prepare, loop, exec).
- Complexity: Low. The notes insertion is a straightforward loop with a prepared statement. No branching complexity.
- Modern idioms: Yes. Prepared statements with deferred close, transaction with deferred rollback, proper error propagation.
- Readability: Good. The code follows the established pattern for other junction-table-like inserts (dependencies, tags, refs), making it immediately recognizable.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
