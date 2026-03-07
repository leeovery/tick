TASK: Extract shared ready-query SQL conditions (tick-core-7-1)

ACCEPTANCE CRITERIA:
- The ready NOT EXISTS subqueries appear in exactly one location (the shared helper)
- buildListQuery uses the shared helper for both ready and blocked filters
- RunStats uses the shared helper for the ready count query
- All existing list and stats tests pass unchanged
- `tick ready`, `tick blocked`, and `tick stats` produce identical output to before

STATUS: Complete

SPEC CONTEXT: The specification defines "ready" as: status is `open`, all `blocked_by` tasks are closed (done or cancelled), and has no open children. "Blocked" is the De Morgan inverse: open status AND (has unclosed blockers OR has open children). These conditions were previously duplicated across `list.go` (buildListQuery, lines ~211-245) and `stats.go` (RunStats, lines ~86-99), creating drift risk.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers.go` (lines 1-65): New file defining `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()`, `ReadyConditions()`, `BlockedConditions()`, and `ReadyWhereClause()`
  - `/Users/leeovery/Code/tick/internal/cli/list.go:204`: `buildListQuery` uses `ReadyConditions()` for `--ready` filter
  - `/Users/leeovery/Code/tick/internal/cli/list.go:208`: `buildListQuery` uses `BlockedConditions()` for `--blocked` filter
  - `/Users/leeovery/Code/tick/internal/cli/stats.go:79`: `RunStats` uses `ReadyWhereClause()` for the ready count query
- Notes: The ready/blocked SQL conditions exist in exactly one location (`query_helpers.go`). A grep for `blocker.status NOT IN` confirms no other file contains these SQL fragments. The blocked conditions are correctly the De Morgan inverse (EXISTS instead of NOT EXISTS, with OR). The stats blocked count is derived arithmetically (`Open - Ready`) rather than re-querying, which is correct and avoids another SQL invocation using the same conditions.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers_test.go`: Unit tests verify `ReadyNoUnclosedBlockers()` returns non-empty, `ReadyNoOpenChildren()` returns non-empty, `ReadyConditions()` returns 3 conditions with correct structure, `BlockedConditions()` returns 2 conditions with correct structure, and `ReadyWhereClause()` returns non-empty.
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go`: Integration tests for `--ready` and `--blocked` filters verify correct task inclusion/exclusion via the actual SQL query path.
  - `/Users/leeovery/Code/tick/internal/cli/stats_test.go`: Integration tests for stats verify ready/blocked counts with dependencies and parent/child relationships ("it counts ready and blocked tasks correctly").
- Notes: The unit tests in `query_helpers_test.go` are appropriately lightweight -- they verify structural correctness (correct number of conditions, correct composition) without testing SQL semantics, which is already covered by the integration tests in `list_filter_test.go` and `stats_test.go`. The integration tests exercise the actual SQL against real SQLite databases with realistic task scenarios (blockers, children, etc.), which is where the real validation happens. Not over-tested.

CODE QUALITY:
- Project conventions: Followed. Functions are exported with doc comments. File naming (`query_helpers.go`) is idiomatic Go.
- SOLID principles: Good. Single responsibility -- `query_helpers.go` is solely responsible for SQL fragment definitions. The DRY principle is the core motivation for this task and is fully satisfied.
- Complexity: Low. Each function returns a string or string slice. No branching logic.
- Modern idioms: Yes. Simple Go functions returning string constants. `strings.Join` for composition.
- Readability: Good. Each function has a clear doc comment explaining its purpose and assumption (outer query aliases tasks as "t"). The naming convention (`ReadyConditions`, `BlockedConditions`, `ReadyWhereClause`) is self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `BlockedConditions()` function (lines 41-58) inlines the EXISTS subqueries rather than deriving them from `ReadyNoUnclosedBlockers()`/`ReadyNoOpenChildren()` via string manipulation. This is a reasonable choice -- programmatic negation of SQL fragments is fragile and error-prone. The duplication within `query_helpers.go` itself is acceptable since both the ready and blocked definitions co-locate in the same file, making them easy to update together.
