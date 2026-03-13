TASK: cli-enhancements-3-2 -- Tags junction table in SQLite schema and Cache.Rebuild

ACCEPTANCE CRITERIA:
- SQLite `task_tags(task_id, tag)` junction table created in schema and populated during `Cache.Rebuild()`

STATUS: Complete

SPEC CONTEXT: The specification defines tags storage as "SQLite: junction table `task_tags(task_id, tag)` -- follows the established `blocked_by` -> `dependencies` pattern. Populated during `Cache.Rebuild()`." Edge cases from the plan: task with empty tags slice, rebuild clearing stale tags.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Schema definition: `/Users/leeovery/Code/tick/internal/storage/cache.go:36-40` -- `CREATE TABLE IF NOT EXISTS task_tags (task_id TEXT NOT NULL, tag TEXT NOT NULL, PRIMARY KEY (task_id, tag))`
  - Index: `/Users/leeovery/Code/tick/internal/storage/cache.go:62` -- `CREATE INDEX IF NOT EXISTS idx_task_tags_tag ON task_tags(tag)`
  - Clear on rebuild: `/Users/leeovery/Code/tick/internal/storage/cache.go:113-115` -- `DELETE FROM task_tags` executed before `DELETE FROM tasks` (correct order)
  - Prepared insert: `/Users/leeovery/Code/tick/internal/storage/cache.go:136-140` -- `INSERT INTO task_tags (task_id, tag) VALUES (?, ?)`
  - Insert loop: `/Users/leeovery/Code/tick/internal/storage/cache.go:197-201` -- iterates `t.Tags` and inserts each row
- Notes: Implementation follows the exact same pattern as the existing `dependencies` junction table. The composite primary key `(task_id, tag)` prevents duplicate tag entries per task. The tag index (`idx_task_tags_tag`) supports efficient filtering queries. The DELETE FROM task_tags is correctly placed before DELETE FROM tasks. All operations occur within a single transaction for atomicity.

TESTS:
- Status: Adequate
- Coverage:
  - Schema test: `/Users/leeovery/Code/tick/internal/storage/cache_test.go:82-111` -- "it creates task_tags table in schema" verifies columns (task_id, tag) exist and index `idx_task_tags_tag` is present
  - Populate test: `/Users/leeovery/Code/tick/internal/storage/cache_test.go:454-513` -- "it populates task_tags during rebuild for task with tags" creates a task with 3 tags, rebuilds, queries task_tags and verifies all 3 rows with correct task_id and tag values (ordered)
  - Empty tags test: `/Users/leeovery/Code/tick/internal/storage/cache_test.go:515-549` -- "it inserts no rows for task with empty tags slice" verifies COUNT=0 for task with no Tags field set (edge case from plan)
  - Stale tags test: `/Users/leeovery/Code/tick/internal/storage/cache_test.go:551-612` -- "it clears stale tags on rebuild" does two rebuilds (tags change from [frontend, urgent] to [backend]) and verifies only the new tag remains (edge case from plan)
  - Multi-task test: `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1017-1116` -- "it handles rebuild with multiple tasks having different tag sets" tests 3 tasks (2 tags, 1 tag, 0 tags), verifies counts per task, then re-rebuilds to verify idempotency
- Notes: Both edge cases specified in the plan (empty tags slice, stale tags cleared on rebuild) are covered with dedicated tests. Tests are well-structured using `t.Run` subtests. The tests directly query the SQLite database to verify row-level correctness, which is appropriate for a storage-layer test. No over-testing detected -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed -- uses stdlib `testing`, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers, `fmt.Errorf` with `%w` wrapping
- SOLID principles: Good -- Cache.Rebuild has a single responsibility (repopulate cache from task data), the tag insertion follows the same pattern as dependencies (open/closed via consistent extension pattern)
- Complexity: Low -- the tag-related code is a straightforward loop within the existing Rebuild method; no branching or conditional logic beyond the range loop
- Modern idioms: Yes -- uses prepared statements for batch inserts, composite primary key for uniqueness, deferred rollback with explicit commit
- Readability: Good -- code is self-documenting; the tag insertion block at lines 197-201 mirrors the dependency insertion at lines 191-195, making the pattern immediately recognizable
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
