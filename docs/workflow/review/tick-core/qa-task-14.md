TASK: Blocked query, tick blocked & cancel-unblocks-dependents (tick-core-3-4)

ACCEPTANCE CRITERIA:
- Returns open tasks with unclosed blockers or open children
- Excludes tasks where all blockers closed
- Only `open` in output
- Cancel -> dependent unblocks
- Multiple dependents unblock simultaneously
- Partial unblock works correctly
- Deterministic ordering
- `tick blocked` outputs aligned columns
- Empty -> `No tasks found.`, exit 0
- `--quiet` IDs only
- Reuses ready query logic

STATUS: Complete

SPEC CONTEXT: The specification defines blocked as: status is `open` AND either has at least one `blocked_by` task that is not closed (done or cancelled), OR has at least one open child (status open or in_progress). This is the De Morgan inverse of the ready conditions. `tick blocked` is an alias for `list --blocked`. Cancelled tasks count as closed for unblocking purposes.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/app.go:80,162-173` - `blocked` subcommand registered, `handleBlocked` delegates to `RunList` with `--blocked` prepended
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers.go:41-58` - `BlockedConditions()` returns SQL WHERE conditions (open AND (EXISTS unclosed blockers OR EXISTS open children))
  - `/Users/leeovery/Code/tick/internal/cli/list.go:86-240` - `RunList` + `buildListQuery` compose the SQL query using `BlockedConditions()`
- Notes:
  - `handleBlocked` correctly implements the alias pattern by prepending `--blocked` to subArgs and calling `parseListFlags` + `RunList`, mirroring `handleReady`
  - `BlockedConditions()` is the De Morgan inverse of `ReadyConditions()` as specified: open AND (EXISTS unclosed blockers OR EXISTS open children)
  - Query reuses the shared `buildListQuery` infrastructure with `BlockedConditions()` from `query_helpers.go`
  - Ordering is `ORDER BY t.priority ASC, t.created ASC` (line 237) - deterministic as required
  - `--quiet` mode handled in `RunList` lines 157-162 - outputs IDs only
  - Empty results handled by the formatter (`No tasks found.` for pretty format)

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/blocked_test.go:28-328` - `TestBlocked` (14 subtests):
    - Open task blocked by open dep (line 31)
    - Open task blocked by in_progress dep (line 55)
    - Parent with open children (line 72)
    - Parent with in_progress children (line 89)
    - Excludes task when all blockers done/cancelled (line 106)
    - Excludes in_progress tasks from output (line 125)
    - Excludes done tasks from output (line 141)
    - Excludes cancelled tasks from output (line 158)
    - Returns empty when no blocked tasks (line 175)
    - Returns empty when no tasks exist (line 193)
    - Orders by priority ASC then created ASC (line 207)
    - Outputs aligned columns (line 244)
    - Prints 'No tasks found.' when empty (line 277)
    - Partial unblock: two blockers one cancelled still blocked (line 310)
  - `/Users/leeovery/Code/tick/internal/cli/blocked_test.go:330-464` - `TestCancelUnblocksDependents` (3 subtests):
    - Cancel unblocks single dependent moves to ready (line 333)
    - Cancel unblocks multiple dependents (line 381)
    - Cancel does not unblock dependent still blocked by another (line 426)
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers_test.go:36-48` - `BlockedCondition` unit test
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:36-50` - `--blocked` flag integration via `tick list`
- Notes:
  - All 12 planned test cases from the task are covered, plus 5 additional subtests (parent with in_progress children, excludes done/cancelled separately, returns empty when no tasks exist, and the query_helpers unit test)
  - Cancel-unblocks tests are end-to-end: they create tasks, run `tick cancel`, then verify via both `runBlocked` and `runReady`, confirming the full flow
  - Tests are focused and not redundant; each subtest covers a distinct scenario

CODE QUALITY:
- Project conventions: Followed - table-driven where appropriate, subtests with descriptive names, test helpers using `t.Helper()`
- SOLID principles: Good - BlockedConditions() is a single-purpose function; handleBlocked delegates to RunList (SRP); query_helpers.go provides reusable SQL conditions (DRY)
- Complexity: Low - handleBlocked is 10 lines, BlockedConditions is straightforward SQL, all logic composes through RunList
- Modern idioms: Yes - proper Go error handling, defer for cleanup, builder pattern for SQL
- Readability: Good - BlockedConditions has a clear comment explaining it is the De Morgan inverse; code is self-documenting
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `BlockedConditions()` function in `query_helpers.go` duplicates the logic from `ReadyNoUnclosedBlockers()` and `ReadyNoOpenChildren()` with EXISTS instead of NOT EXISTS. This is noted in the plan as Phase 7 task tick-core-7-1 ("Extract shared ready-query SQL conditions"), so it is a known deferred improvement, not a current deficiency.
