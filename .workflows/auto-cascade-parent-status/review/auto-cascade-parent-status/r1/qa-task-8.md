TASK: Add task_transitions table to SQLite cache schema (acps-2-2)

ACCEPTANCE CRITERIA:
- task_transitions table created in SQLite cache schema
- schemaVersion incremented
- Pre-existing cache triggers delete+rebuild via version mismatch
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: The specification defines a task_transitions junction table (same pattern as task_notes) with columns task_id, from_status, to_status, at, auto. Cache schema version must be incremented. The spec includes a FOREIGN KEY constraint which was deliberately omitted to match existing project patterns (no other cache tables use FKs -- analyzed and accepted in cycle 1).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/storage/cache.go:14 (schemaVersion = 2), :54-60 (table DDL), :72 (index), :116-117 (DELETE in Rebuild), :166-170 (prepared insert), :233-241 (insert loop)
- Notes: Schema matches spec exactly (minus FK, which is consistent with project patterns). The table has all 5 columns, the index on task_id exists, Rebuild clears and repopulates transitions, and the auto boolean is correctly mapped to INTEGER (0/1). schemaVersion incremented from 1 to 2.

TESTS:
- Status: Adequate
- Coverage:
  - Table creation with correct columns verified via PRAGMA table_info (:1516-1551)
  - Index existence verified (:1547-1550)
  - Rebuild populates transitions correctly with all field values (:1553-1612)
  - Auto flag stored as integer (true=1) (:1614-1658)
  - Task with no transitions produces zero rows (:1660-1695)
  - Rebuild clears old transitions before repopulating (:1697-1750)
  - Schema version constant is 2 (:1752-1758)
- Notes: Tests cover the table schema, data population, edge cases (no transitions, auto flag encoding), and idempotent rebuild. The edge case from the plan (pre-existing cache triggers delete+rebuild via version mismatch) is covered indirectly -- the schema version test verifies the constant changed, and the ensureFresh logic in store.go:395-410 handles the version mismatch -> delete+rebuild flow, which has its own existing tests. Test coverage is well-balanced.

CODE QUALITY:
- Project conventions: Followed. Follows existing junction table patterns (task_notes, task_tags, task_refs). Uses stdlib testing, t.Run subtests, t.TempDir, t.Helper on helpers.
- SOLID principles: Good. Cache schema changes are localized to cache.go. The Rebuild method follows the same pattern for all junction tables.
- Complexity: Low. Straightforward DDL addition and INSERT loop following established patterns.
- Modern idioms: Yes. Consistent with existing Go patterns in the codebase.
- Readability: Good. Schema SQL is clearly structured, rebuild logic follows the same pattern as other junction tables.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The schema version mismatch test (:1752-1758) only checks the constant value rather than exercising the actual delete+rebuild path with a version-1 cache. This is acceptable since the ensureFresh version mismatch logic is tested separately in store_test.go, but an integration-level test could strengthen confidence.
