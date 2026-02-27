TASK: tick list filter flags -- --ready, --blocked, --status, --priority

ACCEPTANCE CRITERIA:
- `list --ready` = same as `tick ready`
- `list --blocked` = same as `tick blocked`
- `--status` filters by exact match
- `--priority` filters by exact match
- Filters AND-combined
- `--ready` + `--blocked` -> error
- Invalid values -> error with valid options
- No matches -> `No tasks found.`, exit 0
- `--quiet` outputs filtered IDs
- Backward compatible (no filters = all)
- Reuses query functions

STATUS: Complete

SPEC CONTEXT:
The spec defines `tick list` with --ready, --blocked, --status, --priority options. `tick ready` is an alias for `list --ready`; `tick blocked` is an alias for `list --blocked`. Status enum: open, in_progress, done, cancelled. Priority: 0-4. The spec also shows `--parent` as a list option (covered by a separate task 3-6). Output should be aligned columns (TTY) or TOON (non-TTY), with `--quiet` outputting IDs only. Empty results: "No tasks found." in human-readable.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/list.go:14-84 (ListFilter struct and parseListFlags), /Users/leeovery/Code/tick/internal/cli/list.go:88-166 (RunList), /Users/leeovery/Code/tick/internal/cli/list.go:199-240 (buildListQuery)
- Notes:
  - Four flags parsed: --ready (bool), --blocked (bool), --status (string), --priority (int with HasPriority sentinel)
  - Mutual exclusion of --ready/--blocked enforced at line 61-63
  - Status validation against 4 valid values at lines 65-75
  - Priority range validation 0-4 at lines 77-81
  - buildListQuery composes SQL with AND-combined conditions
  - Reuses ReadyConditions() and BlockedConditions() from query_helpers.go (shared with ready.go/blocked.go)
  - `tick ready` and `tick blocked` aliases implemented in app.go:150-173 by prepending --ready/--blocked to args and calling RunList
  - --quiet mode outputs only IDs at lines 157-162
  - No filters = no WHERE conditions = all tasks returned
  - Ordering: priority ASC, created ASC

TESTS:
- Status: Adequate
- Coverage:
  - "it filters to ready tasks with --ready" -- verifies ready task appears, blocked task excluded
  - "it filters to blocked tasks with --blocked" -- verifies blocked task appears, unblocked excluded
  - "it filters by --status open/in_progress/done/cancelled" -- all 4 status values individually tested
  - "it filters by --priority" -- exact match verified, non-matching excluded
  - "it combines --ready with --priority" -- AND combination verified
  - "it combines --status with --priority" -- AND combination verified
  - "it errors when --ready and --blocked both set" -- mutual exclusion, exit 1, error message
  - "it errors for invalid status value" -- exit 1, error message lists valid options
  - "it errors for invalid priority value" -- exit 1, range 0-4 in message
  - "it errors for non-numeric priority value" -- bonus edge case
  - "it returns 'No tasks found.' when no matches" -- exit 0, exact message
  - "it outputs IDs only with --quiet after filtering" -- exact output verified
  - "it returns all tasks with no filters" -- backward compatibility
  - "it maintains deterministic ordering" -- priority ASC, created ASC verified
  - "contradictory filters return empty result no error" -- --status done + --ready -> empty, exit 0
- Notes: All 12 tests from the task plan are covered, plus 2 additional useful edge case tests (non-numeric priority, contradictory filters). Tests are behavioral, not testing implementation details. Each test creates a fresh project with specific task data and verifies through the full CLI path. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed. Table-driven subtests pattern used where appropriate. Helper functions (runList, setupTickProjectWithTasks) follow Go testing conventions with t.Helper(). Tests are in the same package (white-box but testing through public CLI interface).
- SOLID principles: Good. ListFilter is a data struct, parseListFlags handles parsing/validation, RunList handles execution, buildListQuery handles SQL composition. Single responsibility respected. ReadyConditions/BlockedConditions are reused from query_helpers.go (DRY, shared with ready.go and blocked.go commands).
- Complexity: Low. parseListFlags is a linear flag parser. buildListQuery builds SQL conditions in a straightforward manner. No nested loops or complex branching.
- Modern idioms: Yes. Uses proper error wrapping, switch statements for flag parsing, variadic args for SQL placeholders.
- Readability: Good. Code is self-documenting with clear function names and comments. ListFilter struct has doc comments on non-obvious fields (HasPriority).
- Issues: None blocking.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `parseListFlags` function uses manual flag parsing rather than Go's `flag` package. This is consistent with the rest of the codebase (global flags also parsed manually) so it is a deliberate design choice, not a deficiency.
- The `ready.go` and `blocked.go` files contain only `package cli` declarations. The actual ready/blocked alias logic lives in `app.go` (handleReady/handleBlocked). This is fine architecturally but the empty files could be removed to reduce clutter.
