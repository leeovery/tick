TASK: Ready query & tick ready command (tick-core-3-3)

ACCEPTANCE CRITERIA:
- Returns tasks matching all three conditions (open, unblocked, no open children)
- Open/in_progress blockers exclude task
- Open/in_progress children exclude task
- Cancelled blockers unblock
- Only `open` status returned
- Deep nesting handled correctly
- Deterministic ordering (priority ASC, created ASC)
- `tick ready` outputs aligned columns
- Empty -> `No tasks found.`, exit 0
- `--quiet` outputs IDs only
- Query function reusable by blocked query and list filters

STATUS: Complete

SPEC CONTEXT:
The spec defines "ready" as: status `open`, all `blocked_by` tasks closed (done/cancelled), no open children (leaf-only rule). Ordering is priority ASC, created ASC. `tick ready` is an alias for `list --ready`. Deep nesting handled naturally by the leaf-only rule -- only deepest incomplete tasks appear. Cancelled tasks count as closed and unblock dependents.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers.go:1-65` -- ReadyConditions(), ReadyNoUnclosedBlockers(), ReadyNoOpenChildren(), ReadyWhereClause(), BlockedConditions()
  - `/Users/leeovery/Code/tick/internal/cli/list.go:86-240` -- RunList() with buildListQuery() composing ready conditions via ReadyConditions()
  - `/Users/leeovery/Code/tick/internal/cli/app.go:150-160` -- handleReady() prepends `--ready` to args and delegates to RunList()
- Notes:
  - ReadyQuery is implemented as composable SQL conditions, not a standalone function returning tasks. This is actually a better design -- it allows reuse across list, stats, and blocked queries.
  - `handleReady` correctly acts as alias: it prepends `--ready` to subArgs and calls `parseListFlags` + `RunList`, matching the spec's "alias for list --ready" requirement.
  - ReadyWhereClause() is used by `stats.go:79` for the ready count query, confirming reusability.
  - ReadyConditions() is used by `buildListQuery` when `filter.Ready` is true.
  - BlockedConditions() uses the De Morgan inverse of the ready conditions.
  - Ordering `ORDER BY t.priority ASC, t.created ASC` is applied in buildListQuery (line 237).

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/ready_test.go:28-402` -- 14 subtests covering all planned test cases:
    1. "it returns open task with no blockers and no children" -- verifies basic ready condition
    2. "it excludes task with open blocker" -- verifies open blocker exclusion
    3. "it excludes task with in_progress blocker" -- verifies in_progress blocker exclusion
    4. "it includes task when all blockers done" -- verifies done blocker unblocking
    5. "it includes task when all blockers cancelled" -- verifies cancelled blocker unblocking (edge case)
    6. "it excludes parent with open children" -- verifies leaf-only rule
    7. "it excludes parent with in_progress children" -- verifies in_progress child exclusion
    8. "it includes parent when all children closed" -- verifies parent becomes leaf
    9. "it excludes in_progress tasks" -- verifies only open status returned
    10. "it excludes done tasks" -- verifies only open status returned
    11. "it excludes cancelled tasks" -- verifies only open status returned
    12. "it handles deep nesting - only deepest incomplete ready" -- verifies 3-level nesting
    13. "it returns empty list when no tasks ready" -- verifies empty result
    14. "it orders by priority ASC then created ASC" -- verifies deterministic ordering
    15. "it outputs aligned columns via tick ready" -- verifies human-readable output
    16. "it prints 'No tasks found.' when empty" -- verifies empty project output
    17. "it outputs IDs only with --quiet" -- verifies quiet mode
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers_test.go:1-56` -- Unit tests for the SQL condition helpers
  - Additional coverage from `list_filter_test.go` -- tests `--ready` flag via `tick list --ready`
  - Additional coverage from `blocked_test.go` -- cancel-unblocks-dependents tests use `runReady` to verify tasks move to ready
- Notes:
  - All 12 planned tests from the task are present, plus additional subtests (in_progress blocker, in_progress children as separate cases).
  - Tests exercise the full stack (CLI -> SQLite query -> formatter) via `runReady` helper, which constructs an App and calls `app.Run`.
  - Tests verify exit code 0 for empty results (exit 0, not error).
  - The query_helpers_test is lightweight (checks non-empty strings) but the real validation happens through the integration-style ready_test.go tests that exercise the SQL against real SQLite.
  - Not over-tested: each test targets a distinct behavioral scenario. No redundant assertions.

CODE QUALITY:
- Project conventions: Followed -- table-driven tests with subtests, `t.Helper()` on test helpers, proper error wrapping
- SOLID principles: Good
  - Single responsibility: query_helpers.go provides reusable SQL fragments; list.go handles query composition and execution; app.go handles dispatch
  - Open/closed: ReadyConditions() can be used by any query without modification
  - Dependency inversion: Formatter interface abstracts output format
- Complexity: Low -- clear linear code paths, no nested conditionals
- Modern idioms: Yes -- uses Go idioms appropriately (defer for cleanup, error wrapping, type-safe status constants)
- Readability: Good -- functions are well-named (ReadyNoUnclosedBlockers, ReadyNoOpenChildren), comments explain purpose
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The query_helpers_test.go tests are somewhat superficial (just checking non-empty strings). These helpers are adequately tested through the integration-level ready_test.go and blocked_test.go, so this is acceptable but could be noted as a style choice.
- Phase 7 (tick-core-7-1) is planned to extract shared ready-query SQL conditions further, which would address the minor duplication between ReadyConditions() and BlockedConditions() (the EXISTS subqueries are written out separately rather than composed from the NOT EXISTS forms). This is already tracked.
