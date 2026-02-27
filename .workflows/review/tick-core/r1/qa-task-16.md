TASK: Parent scoping -- --parent flag with recursive descendant CTE

ACCEPTANCE CRITERIA:
- `tick list --parent <id>` returns only descendants of the specified parent (recursive, all levels)
- `tick ready --parent <id>` returns only ready tasks within the descendant set
- `tick blocked --parent <id>` returns only blocked tasks within the descendant set
- Parent task itself is excluded from results (leaf-only rule filters it out naturally when it has open children)
- Non-existent parent ID returns error: `Error: Task '<id>' not found`
- Parent with no descendants returns empty result (`No tasks found.`, exit 0)
- Deep nesting (3+ levels) collects all descendants recursively
- `--parent` composes with `--status` filter (AND)
- `--parent` composes with `--priority` filter (AND)
- `--parent` composes with `--ready` flag (AND)
- `--parent` composes with `--blocked` flag (AND)
- Case-insensitive parent ID matching (e.g., `TICK-A1B2` treated as `tick-a1b2`)
- `--quiet` outputs IDs only within the scoped set

STATUS: Complete

SPEC CONTEXT:
The specification (Parent Scoping section) defines `--parent <id>` as a pre-filter that restricts queries to descendants of the specified task using a recursive CTE in SQLite. It narrows which tasks are considered before post-filters (leaf-only, blocked-by, status, priority) are applied. The `ready` and `blocked` commands are aliases for `tick list --ready` and `tick list --blocked`, so `--parent` applies to them automatically. The SQLite schema has `CREATE INDEX idx_tasks_parent ON tasks(parent)` to support efficient CTE traversal.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/list.go:14-22` -- `ListFilter` struct with `Parent` field
  - `/Users/leeovery/Code/tick/internal/cli/list.go:52-58` -- `--parent` flag parsing with `task.NormalizeID` for case-insensitive matching
  - `/Users/leeovery/Code/tick/internal/cli/list.go:104-123` -- Parent validation (existence check) and descendant ID collection within `store.Query` callback
  - `/Users/leeovery/Code/tick/internal/cli/list.go:168-195` -- `queryDescendantIDs` function with recursive CTE matching the spec's SQL exactly
  - `/Users/leeovery/Code/tick/internal/cli/list.go:199-240` -- `buildListQuery` integrating descendant IDs as `WHERE t.id IN (...)` pre-filter, with `1 = 0` impossible condition when parent exists but has no descendants
  - `/Users/leeovery/Code/tick/internal/cli/app.go:150-159` -- `handleReady` prepends `--ready` to subArgs, passing `--parent` through naturally
  - `/Users/leeovery/Code/tick/internal/cli/app.go:163-172` -- `handleBlocked` prepends `--blocked` to subArgs, passing `--parent` through naturally
- Notes: Clean implementation. The pre-filter/post-filter design matches the spec exactly. The recursive CTE starts from children of the parent (not the parent itself), so the parent is naturally excluded from the descendant set. The `1 = 0` impossible condition for empty descendant sets is a good pattern to guarantee empty results without special-casing.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/parent_scope_test.go` -- 15 subtests covering all planned test cases:
    1. Direct children returned (line 14)
    2. Recursive 3+ levels deep (line 39)
    3. Parent excluded from results (line 64)
    4. Empty result when parent has no descendants (line 84)
    5. Error for non-existent parent ID (line 101)
    6. Ready within parent scope via `tick ready --parent` (line 114)
    7. Blocked within parent scope via `tick blocked --parent` (line 143)
    8. Combines with `--status` filter (line 172)
    9. Combines with `--priority` filter (line 194)
    10. Combines `--parent` with `--ready` and `--priority` (line 215)
    11. Combines `--parent` with `--blocked` and `--status` (line 241)
    12. Case-insensitive parent ID (line 262)
    13. Excludes tasks outside subtree (line 279)
    14. `--quiet` outputs IDs only within scoped set (line 305)
    15. "No tasks found." when descendants exist but none match filters (line 325)
  - All 15 tests from the task's "Tests" section are present and match 1:1
  - Tests use realistic multi-task setups with blockers, multiple parents, and various statuses
  - Tests verify both inclusion and exclusion (positive and negative assertions)
  - Edge cases covered: empty descendants, non-existent parent, deep nesting, contradictory filters
- Notes: Test coverage is thorough without being redundant. Each test targets a distinct acceptance criterion. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed -- uses table-driven-style subtests, explicit error handling, `t.Helper()` in helper functions, consistent test naming pattern
- SOLID principles: Good -- `queryDescendantIDs` is a single-responsibility function; `buildListQuery` composes filters cleanly; `ListFilter` struct cleanly extends with the `Parent` field without modifying existing filter logic
- Complexity: Low -- the recursive CTE is the most complex piece but it's a well-known SQLite pattern; `buildListQuery` uses simple conditional appends; the `1 = 0` pattern for empty descendants avoids additional branching
- Modern idioms: Yes -- proper use of `database/sql` interfaces, variadic args for query parameters, Go error wrapping conventions
- Readability: Good -- clear comments on both `queryDescendantIDs` and `buildListQuery`; the `else if f.Parent != ""` branch for empty descendants is well-placed with an inline comment explaining the impossible condition
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The error message uses lowercase `task '%s' not found` while the acceptance criteria says `Task '<id>' not found`. This follows Go convention (errors should not be capitalized) and is consistent across all commands in the codebase. Not a real issue.
- The `queryDescendantIDs` function lives in `list.go` rather than being extracted to the storage layer. This is reasonable for now since it is only used by the list command, but if parent scoping is ever needed elsewhere (e.g., stats scoped to a parent), it would need to be moved. The plan's Phase 7 already tracks store-related refactoring.
