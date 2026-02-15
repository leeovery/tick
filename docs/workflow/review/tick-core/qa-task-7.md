TASK: tick list & tick show commands

ACCEPTANCE CRITERIA:
- [ ] `tick list` displays all tasks in aligned columns (ID, STATUS, PRI, TITLE)
- [ ] `tick list` orders by priority ASC then created ASC
- [ ] `tick list` prints `No tasks found.` when empty
- [ ] `tick list --quiet` outputs only task IDs
- [ ] `tick show <id>` displays full task details
- [ ] `tick show` includes blocked_by with context (ID, title, status)
- [ ] `tick show` includes children with context
- [ ] `tick show` includes parent field when set
- [ ] `tick show` omits empty optional sections
- [ ] `tick show` errors when ID not found
- [ ] `tick show` errors when no ID argument
- [ ] Input IDs normalized to lowercase
- [ ] Both commands use storage engine read flow
- [ ] Exit code 0 on success, 1 on error

STATUS: Complete

SPEC CONTEXT: The specification defines `tick list` as showing all tasks with optional filters (--ready, --blocked, --status, --priority) -- filters are Phase 3. Phase 1 shows all tasks unfiltered. `tick show <id>` displays full task details with blocked_by, children, parent, description, and closed sections, omitting empty optional sections. Both commands use the storage engine read flow: shared lock, read JSONL, freshness check, query SQLite, release lock. Output format is TTY-detected (pretty for terminals, TOON for pipes).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/list.go` (lines 1-241): ListFilter, parseListFlags, RunList, queryDescendantIDs, buildListQuery
  - `/Users/leeovery/Code/tick/internal/cli/show.go` (lines 1-169): showData, RunShow, queryShowData, showDataToTaskDetail
  - `/Users/leeovery/Code/tick/internal/cli/app.go` (lines 71-73): list and show subcommand registration in dispatcher
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go` (lines 24-121): FormatTaskList and FormatTaskDetail rendering
- Notes:
  - `tick list` implemented via RunList: queries all tasks from SQLite ordered by priority ASC, created ASC. Uses PrettyFormatter.FormatTaskList for aligned columns (dynamic widths based on data). Empty result renders "No tasks found." via formatter. Quiet mode outputs IDs only.
  - `tick show` implemented via RunShow: validates ID argument, normalizes to lowercase via task.NormalizeID, queries task + dependencies + children + parent from SQLite. Uses PrettyFormatter.FormatTaskDetail for key-value output. Quiet mode outputs ID only.
  - Both commands use `openStore(dir, fc)` which discovers .tick/, opens a Store, and the Store.Query method acquires shared lock, reads JSONL, checks freshness, then runs the SQL callback.
  - All acceptance criteria are met by the implementation.

TESTS:
- Status: Adequate
- Coverage:
  - File: `/Users/leeovery/Code/tick/internal/cli/list_show_test.go` (517 lines)
  - TestList (4 subtests):
    - "it lists all tasks with aligned columns" -- verifies header and data rows with exact alignment
    - "it lists tasks ordered by priority then created date" -- verifies 4 tasks in correct order
    - "it prints 'No tasks found.' when no tasks exist" -- verifies empty state
    - "it prints only task IDs with --quiet flag on list" -- verifies quiet output
    - "it executes through storage engine read flow" -- verifies deterministic output across two calls (cache built then reused)
  - TestShow (16 subtests):
    - "it shows full task details by ID" -- verifies all core fields
    - "it shows blocked_by section with ID, title, and status of each blocker" -- verifies section content
    - "it shows children section with ID, title, and status of each child" -- verifies section content
    - "it shows description section when description is present" -- verifies section inclusion
    - "it omits blocked_by section when task has no dependencies" -- verifies omission
    - "it omits children section when task has no children" -- verifies omission
    - "it omits description section when description is empty" -- verifies omission
    - "it shows parent field with ID and title when parent is set" -- verifies parent rendering
    - "it omits parent field when parent is null" -- verifies omission
    - "it shows closed timestamp when task is done or cancelled" -- verifies closed field
    - "it omits closed field when task is open or in_progress" -- verifies omission
    - "it errors when task ID not found" -- verifies exit code 1 + error message to stderr
    - "it errors when no ID argument provided to show" -- verifies exit code 1 + usage hint
    - "it normalizes input ID to lowercase for show lookup" -- verifies TICK-AAA111 finds tick-aaa111
    - "it outputs only task ID with --quiet flag on show" -- verifies quiet output
    - "it executes through storage engine read flow" -- verifies deterministic output
    - "queryShowData populates RelatedTask fields for blockers and children" -- unit test of internal function verifying RelatedTask struct fields
  - All 20 planned tests from the task are covered (mapped to 21 actual subtests including the extra queryShowData unit test)
  - Tests run through the full App.Run dispatch path (integration-style), exercising the complete stack: CLI parsing, store opening, shared locking, freshness check, SQLite query, and formatter output
- Notes:
  - Test names match the planned test descriptions almost exactly
  - Tests verify behavior not implementation details -- checking stdout/stderr output and exit codes
  - The queryShowData unit test is slightly testing implementation details but is reasonable since it validates the RelatedTask struct population which is a key data boundary
  - No over-testing detected -- each test covers a distinct behavior/edge case

CODE QUALITY:
- Project conventions: Followed. Uses table-driven tests where appropriate (via subtests), t.Helper() in test helpers, proper error wrapping, injected App struct for testability.
- SOLID principles: Good.
  - Single responsibility: RunList handles list logic, RunShow handles show logic, PrettyFormatter handles rendering.
  - Open/closed: Formatter interface allows adding new formats without modifying commands.
  - Dependency inversion: Commands depend on Formatter interface, not concrete formatters.
- Complexity: Low. RunList and RunShow are straightforward query-then-format flows. buildListQuery has some branching for filters but is clear.
- Modern idioms: Yes. Uses Go idioms (error returns, defer for cleanup, sql.DB callback pattern, functional options for Store).
- Readability: Good. Code is well-structured with clear function names. showData/showDataToTaskDetail conversion is explicit. Comments are present on exported functions.
- Issues: None blocking.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The PrettyFormatter.FormatTaskDetail uses dynamic newline management (some fields end with "\n", optional sections prepend "\n\n") which works but is slightly fragile. The current approach is functionally correct but could benefit from a more systematic approach in later phases.
- Column widths in PrettyFormatter.FormatTaskList are dynamically computed (lines 30-51) rather than the fixed widths specified in the task description (ID=12, STATUS=12, PRI=4). The dynamic approach is arguably better since it adapts to actual data, but differs from the literal task spec. This is a non-issue as the output is still properly aligned.
- The `showData` struct (show.go:14-27) uses unexported fields and a conversion function `showDataToTaskDetail` to map to the exported `TaskDetail` struct. This is clean encapsulation but adds a small amount of boilerplate.
