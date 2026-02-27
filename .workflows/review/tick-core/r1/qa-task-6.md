TASK: tick create command

ACCEPTANCE CRITERIA:
- `tick create "<title>"` creates task with correct defaults (status: open, priority: 2)
- Generated ID follows `tick-{6 hex}` format, unique among existing tasks
- All optional flags work: `--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`
- `--blocks` correctly updates referenced tasks' `blocked_by` arrays
- Missing or empty title returns error to stderr with exit code 1
- Invalid priority returns error with exit code 1
- Non-existent IDs in references return error with exit code 1
- Task persisted via atomic write through storage engine
- SQLite cache updated as part of write flow
- Output shows task details on success
- `--quiet` outputs only task ID
- Input IDs normalized to lowercase
- Timestamps set to current UTC ISO 8601

STATUS: Complete

SPEC CONTEXT:
The specification defines `tick create "<title>" [options]` with flags `--priority <0-4>`, `--blocked-by <id,id,...>`, `--blocks <id,id,...>`, `--parent <id>`, `--description "<text>"`. Default priority is 2, status is always `open`. `--blocks` is inverse of `--blocked-by` -- syntactic sugar that modifies other tasks during creation. Only `blocked_by` is stored in the data model. All mutations go through the storage engine: exclusive lock, JSONL read + freshness check, mutation, atomic write, cache update, lock release. Validation of `blocked_by` references (exist check, no self-reference) happens here since it requires the full task list. Cycle detection and child-blocked-by-parent validation also apply.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/cli/create.go` (full file, 205 lines)
- Supporting code:
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go` (`parseCommaSeparatedIDs`, `applyBlocks`, `outputMutationResult`, `openStore`)
  - `/Users/leeovery/Code/tick/internal/cli/app.go:68-69` (command dispatch via `handleCreate`)
  - `/Users/leeovery/Code/tick/internal/task/task.go` (Task struct, GenerateID, ValidateTitle, ValidatePriority, NormalizeID, TrimTitle)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:134` (Mutate with locking, atomic write, cache update)
- Notes:
  - `parseCreateArgs` handles title as first positional arg and all flags (--priority, --description, --blocked-by, --blocks, --parent)
  - Default priority 2 set in `createOpts` struct initialization (line 26)
  - Title validation: trimmed via `task.TrimTitle`, validated via `task.ValidateTitle` (non-empty, no newlines, max 500 chars)
  - Priority validation via `task.ValidatePriority` (0-4 range)
  - ID generation via `task.GenerateID` with collision check against existing task set
  - Reference validation in `validateRefs`: checks existence and self-reference for blocked-by, blocks, and parent
  - `--blocks` applied via `applyBlocks` in helpers.go -- adds new task ID to target tasks' `blocked_by` and refreshes `updated` timestamp
  - Dependency validation (cycle detection + child-blocked-by-parent) called after task appended to list
  - Mutation runs inside `store.Mutate` which handles exclusive locking, JSONL read, atomic write, and cache rebuild
  - Output via `outputMutationResult`: quiet mode prints ID only; normal mode queries show data and formats via Formatter
  - Timestamps set to `time.Now().UTC().Truncate(time.Second)` matching ISO 8601 format
  - `Closed` field not set (zero value for `*time.Time` is nil), correctly omitted from serialization
  - ID normalization happens in `parseCommaSeparatedIDs` for --blocked-by and --blocks, and explicitly for --parent (line 65)
  - All acceptance criteria are addressed in the implementation

TESTS:
- Status: Adequate
- Coverage:
  - "it creates a task with only a title (defaults applied)" -- verifies defaults (open, priority 2, empty description/blocked_by/parent)
  - "it creates a task with all optional fields specified" -- verifies all flags together
  - "it generates a unique ID for the created task" -- creates two tasks, verifies uniqueness and tick- prefix and 11-char length
  - "it sets status to open on creation" -- verifies status field
  - "it sets default priority to 2 when not specified" -- verifies default
  - "it sets priority from --priority flag" -- verifies priority 0
  - "it rejects priority outside 0-4 range" -- table-driven: -1, 5, 100
  - "it sets description from --description flag" -- verifies description
  - "it sets blocked_by from --blocked-by flag (single ID)" -- single dep
  - "it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)" -- two deps
  - "it updates target tasks' blocked_by when --blocks is used" -- verifies target's blocked_by updated and timestamp refreshed
  - "it sets parent from --parent flag" -- verifies parent
  - "it errors when title is missing" -- no args, exit 1
  - "it errors when title is empty string" -- empty string, exit 1
  - "it errors when title is whitespace only" -- whitespace, exit 1
  - "it errors when --blocked-by references non-existent task" -- exit 1
  - "it errors when --blocks references non-existent task" -- exit 1
  - "it errors when --parent references non-existent task" -- exit 1
  - "it persists the task to tasks.jsonl via atomic write" -- reads raw file, verifies JSONL
  - "it outputs full task details on success" -- checks ID, title, status in stdout
  - "it outputs only task ID with --quiet flag" -- exact match on ID + newline
  - "it normalizes input IDs to lowercase" -- TICK-AAA111 normalized to tick-aaa111
  - "it trims whitespace from title" -- leading/trailing whitespace removed
  - Additional tests beyond plan scope (defense in depth):
    - "it rejects --blocked-by that would create child-blocked-by-parent dependency"
    - "it rejects --blocked-by that would create a cycle"
    - "it rejects --blocks that would create child-blocked-by-parent dependency" (skipped with architectural justification)
    - "it rejects --blocks that would create a cycle"
    - "it shows blocker title and status in output when created with --blocked-by"
    - "it shows parent title in output when created with --parent"
    - "it shows relationship context when created with --blocks"
    - "it outputs only task ID with --quiet flag after create with relationships"
    - "it produces correct output without relationships (empty blocked_by/children)"
    - "it allows valid dependencies through create --blocked-by and --blocks"
  - Supporting helper tests in `helpers_test.go`:
    - `TestParseCommaSeparatedIDs` (7 subtests: single, multiple, whitespace, empty, commas, lowercase, trailing comma)
    - `TestApplyBlocks` (6 subtests: append, timestamp update, no-op, duplicate skip, case-insensitive match, multiple IDs)
    - `TestOutputMutationResult` (3 subtests: quiet mode, full detail, non-existent ID)
- Notes: All 22 tests from the plan task are present. The additional tests cover dependency validation at creation time and output formatting with relationships -- these are appropriate given that Phase 6 added dependency validation to the create path. Test balance is good; tests are behavior-focused rather than implementation-detail-focused.

CODE QUALITY:
- Project conventions: Followed -- table-driven tests with subtests, error wrapping, idiomatic Go patterns
- SOLID principles: Good
  - Single responsibility: `parseCreateArgs` handles parsing, `RunCreate` orchestrates, `validateRefs` validates, `applyBlocks` applies side-effects
  - Dependency inversion: `Formatter` interface for output, `store.Mutate` for persistence
- Complexity: Low -- `parseCreateArgs` has a switch-based loop (straightforward), `RunCreate` follows a clear linear flow
- Modern idioms: Yes -- uses `time.Time` directly, `Truncate(time.Second)` for clean timestamps, `map[string]bool` for set membership
- Readability: Good -- clear function names, comments on each function, validation logic separated from mutation logic
- Issues: None significant

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `Closed` field is not explicitly set in the `newTask` struct (line 134-144 of create.go). This works correctly because `*time.Time` zero value is nil, and `omitempty` handles serialization. Explicit `Closed: nil` would make intent clearer but is not required.
- The `parseCreateArgs` silently skips unknown flags (line 66-67). This is by design since global flags are pre-extracted, but could mask typos like `--proirity`. This is a general CLI design choice, not specific to this task.
