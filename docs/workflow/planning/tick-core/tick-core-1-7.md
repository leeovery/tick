---
id: tick-core-1-7
phase: 1
status: completed
created: 2026-01-30
---

# tick list & tick show commands

## Goal

Tasks 1-5 and 1-6 set up the CLI and the `tick create` mutation command, but there is no way to retrieve tasks. The walking skeleton is incomplete without read commands. `tick list` displays a summary table of all tasks (no filters in Phase 1 — filtering is Phase 3). `tick show <id>` displays full details of a single task. Both execute through the storage engine's read flow (shared lock, freshness check, SQLite query) and produce basic output (full formatting is Phase 4).

## Implementation

- Register `list` and `show` subcommands in the CLI dispatcher (tick-core-1-5)

### tick list

- No arguments, no filters in Phase 1 (filters are Phase 3)
- Query all tasks from SQLite, ordered by priority ASC (highest first), then created ASC (oldest first)
- Output as aligned columns:
  ```
  ID          STATUS       PRI  TITLE
  tick-a1b2   done         1    Setup Sanctum
  tick-c3d4   in_progress  1    Login endpoint
  ```
- Column widths: ID (12), STATUS (12), PRI (4), TITLE (remainder)
- Empty result: print `No tasks found.` — no headers, no table
- With `--quiet`: output only task IDs, one per line

### tick show

- Requires one positional argument `<id>`. If missing: `Error: Task ID is required. Usage: tick show <id>`
- Normalize input ID to lowercase
- Query task + dependencies (blocked_by tasks' ID, title, status) + children (ID, title, status)
- If not found: `Error: Task 'tick-xyz' not found`
- Output as key-value format:
  ```
  ID:       tick-c3d4
  Title:    Login endpoint
  Status:   in_progress
  Priority: 1
  Created:  2026-01-19T10:00:00Z
  Updated:  2026-01-19T14:30:00Z

  Blocked by:
    tick-a1b2  Setup Sanctum (done)

  Children:
    tick-e5f6  Sub-task one (open)

  Description:
    Implement the login endpoint...
  ```
- Omit sections with no data (blocked_by, children, description, parent, closed)
- Include `Closed:` only when not null; `Parent:` only when set (show ID and title)
- With `--quiet`: output only the task ID

## Tests

- `"it lists all tasks with aligned columns"`
- `"it lists tasks ordered by priority then created date"`
- `"it prints 'No tasks found.' when no tasks exist"`
- `"it prints only task IDs with --quiet flag on list"`
- `"it shows full task details by ID"`
- `"it shows blocked_by section with ID, title, and status of each blocker"`
- `"it shows children section with ID, title, and status of each child"`
- `"it shows description section when description is present"`
- `"it omits blocked_by section when task has no dependencies"`
- `"it omits children section when task has no children"`
- `"it omits description section when description is empty"`
- `"it shows parent field with ID and title when parent is set"`
- `"it omits parent field when parent is null"`
- `"it shows closed timestamp when task is done or cancelled"`
- `"it omits closed field when task is open or in_progress"`
- `"it errors when task ID not found"`
- `"it errors when no ID argument provided to show"`
- `"it normalizes input ID to lowercase for show lookup"`
- `"it outputs only task ID with --quiet flag on show"`
- `"it executes through storage engine read flow (shared lock, freshness check)"`

## Edge Cases

- No tasks (empty list): prints `No tasks found.` to stdout, exit code 0 (not an error). Normal state after `tick init` before any `tick create`.
- Task ID not found: error to stderr, exit code 1. Error message uses the normalized (lowercase) ID.
- No ID argument to show: error with usage hint, exit code 1.
- Case-insensitive lookup: `TICK-A1B2` normalized to `tick-a1b2`.
- Tasks with all optional fields: show renders every section.
- Tasks with only required fields: show renders only core fields, no optional sections.

## Acceptance Criteria

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

## Context

Specification defines `tick list` with filter options `--ready`, `--blocked`, `--status`, `--priority` — Phase 3 scope. Phase 1 shows all tasks unfiltered. `tick ready` and `tick blocked` aliases are also Phase 3.

Both commands use the storage engine's read query flow (tick-core-1-4): shared lock → read JSONL → freshness check → query SQLite → release lock. Concurrent reads don't block each other.

Error format: `Error: ` prefix convention from tick-core-1-5. All errors to stderr.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
