TASK: cli-enhancements-4-2 -- Refs junction table in SQLite schema and Cache.Rebuild

ACCEPTANCE CRITERIA:
- SQLite `task_refs(task_id, ref)` junction table populated during `Cache.Rebuild()`
- Edge cases: task with empty refs slice, rebuild clearing stale refs

STATUS: Complete

SPEC CONTEXT: Spec section "External References / Storage" states: SQLite junction table `task_refs(task_id, ref)` following the same pattern as tags and dependencies, populated during Cache.Rebuild(). Refs are not filterable on list/ready/blocked -- they are view-only via `show`.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Schema DDL: `/Users/leeovery/Code/tick/internal/storage/cache.go:42-46` -- `CREATE TABLE IF NOT EXISTS task_refs (task_id TEXT NOT NULL, ref TEXT NOT NULL, PRIMARY KEY (task_id, ref))`
  - Clear stale refs: `/Users/leeovery/Code/tick/internal/storage/cache.go:110-112` -- `DELETE FROM task_refs` in Rebuild transaction
  - Prepare insert: `/Users/leeovery/Code/tick/internal/storage/cache.go:142-146` -- prepared statement for ref insertion
  - Insert loop: `/Users/leeovery/Code/tick/internal/storage/cache.go:203-207` -- iterates `t.Refs` and inserts each row
  - Query usage: `/Users/leeovery/Code/tick/internal/cli/show.go:136` -- `SELECT ref FROM task_refs WHERE task_id = ? ORDER BY ref`
- Notes: Implementation follows the exact same pattern as `task_tags` and `dependencies`. The table uses a composite primary key `(task_id, ref)` which prevents duplicates at the database level. No extra index is created on `task_refs`, which is correct since refs are not filterable (spec says "not filterable on list, ready, blocked"). The entire rebuild runs in a single transaction for atomicity.

TESTS:
- Status: Adequate
- Coverage:
  - Schema verification: "it creates task_refs table in schema" (`cache_test.go:144`) -- verifies table exists with correct columns (task_id, ref)
  - Populate during rebuild: "it populates refs in task_refs during rebuild" (`cache_test.go:614`) -- inserts task with 2 refs, queries back and verifies both rows with correct task_id and ref values
  - Empty refs edge case: "it inserts no rows for task with empty refs slice" (`cache_test.go:675`) -- confirms zero rows inserted when task has no refs
  - Stale refs edge case: "it clears stale refs on rebuild" (`cache_test.go:711`) -- rebuilds with 2 refs, then rebuilds again with 1 different ref, verifies only the new ref remains
- Notes: All edge cases specified in the plan (empty refs slice, stale ref clearing) are covered. Tests directly query the SQLite database to verify data integrity. Tests would fail if the feature broke (e.g., if refs were not inserted or stale data not cleared). No over-testing -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed -- uses stdlib testing, t.Run subtests, t.TempDir for isolation, t.Helper on helpers, t.Fatalf for setup failures, t.Errorf for assertion failures
- SOLID principles: Good -- Rebuild handles all junction tables uniformly; single transaction responsibility
- Complexity: Low -- straightforward loop-based insertion, standard DELETE + INSERT pattern
- Modern idioms: Yes -- prepared statements, deferred Close, error wrapping with %w
- Readability: Good -- follows the established pattern from dependencies and tags exactly, making it immediately recognizable
- Issues: None

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
