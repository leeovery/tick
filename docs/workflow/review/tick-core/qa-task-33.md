TASK: Add relationship context to create command output (tick-core-7-2)

ACCEPTANCE CRITERIA:
- `tick create "test" --blocked-by tick-abc` output includes the blocker's title and status in the blocked_by section
- `tick create "test" --parent tick-abc` output includes the parent's title
- Create output matches show output for the same task (when viewed immediately after creation)
- `--quiet` mode still outputs only the task ID (no change)
- All existing create tests pass

STATUS: Complete

SPEC CONTEXT: Spec line 631 states create output should be "full task details (same format as tick show), TTY-aware." The show format includes blocked_by entries with id/title/status, parent with title, and children sections. The task identified that create was previously constructing a `TaskDetail{Task: createdTask}` with empty relationship slices instead of querying SQLite for the full context.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:173` -- calls `outputMutationResult(store, createdTask.ID, fc, fmtr, stdout)`
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:16-30` -- `outputMutationResult` function: handles quiet mode (ID only) and non-quiet mode (calls `queryShowData` then `showDataToTaskDetail` then `FormatTaskDetail`)
  - `/Users/leeovery/Code/tick/internal/cli/show.go:61-139` -- `queryShowData` queries SQLite for full task detail including blocked_by with titles/statuses, parent title, and children
  - `/Users/leeovery/Code/tick/internal/cli/show.go:143-169` -- `showDataToTaskDetail` converts query results to `TaskDetail`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:201` -- `RunUpdate` also uses same `outputMutationResult`, confirming shared approach
- Notes: The implementation matches the plan exactly. Instead of manually constructing `TaskDetail{Task: createdTask}` with empty slices, it now calls `outputMutationResult` which uses `queryShowData` to retrieve full relationship context from SQLite. This is the same code path used by `RunUpdate` and `RunShow`, ensuring output parity. The shared helper `outputMutationResult` in `helpers.go` is cleanly factored.

TESTS:
- Status: Adequate
- Coverage:
  - `create_test.go:642-668` -- "it shows blocker title and status in output when created with --blocked-by": verifies "Blocked by:" section present, blocker title "Blocker task", blocker ID "tick-aaa111", and status "open" in output
  - `create_test.go:670-693` -- "it shows parent title in output when created with --parent": verifies "Parent:" field, parent ID "tick-ppp111", and parent title "Parent task" in output
  - `create_test.go:695-722` -- "it shows relationship context when created with --blocks": verifies new task ID and title appear in output after --blocks usage
  - `create_test.go:724-749` -- "it outputs only task ID with --quiet flag after create with relationships": verifies quiet mode with --blocked-by still outputs only the task ID
  - `create_test.go:751-778` -- "it produces correct output without relationships (empty blocked_by/children)": verifies standard fields present and no "Blocked by:"/"Children:" sections when no relationships
  - `helpers_test.go:203-290` -- `TestOutputMutationResult` unit tests: quiet mode outputs ID only, non-quiet outputs full detail, error on non-existent ID
- Notes: Tests cover all 5 acceptance criteria from the task description. The test for --blocks is slightly weaker (only checks task ID and title appear, not relationship sections specifically), but this is reasonable since --blocks modifies the *target* task's blocked_by, not the created task's. The created task itself has no blocked_by or children from --blocks alone.

CODE QUALITY:
- Project conventions: Followed -- table-driven tests in helpers_test.go, explicit error handling, proper use of t.Helper()
- SOLID principles: Good -- `outputMutationResult` follows single responsibility (output only), shared by create and update (DRY). The `queryShowData`/`showDataToTaskDetail` pipeline is cleanly separated.
- Complexity: Low -- the helper is 14 lines with a single branch (quiet vs non-quiet)
- Modern idioms: Yes -- proper Go error propagation, defer for cleanup, io.Writer for testability
- Readability: Good -- function name `outputMutationResult` clearly communicates intent, code flow is linear
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The --blocks test at line 695 could be slightly more thorough by verifying that the output format matches the show format structurally (e.g., checking for "Status:" and "Priority:" fields), but this is covered by other tests in the file and the `outputMutationResult` unit tests in `helpers_test.go`.
