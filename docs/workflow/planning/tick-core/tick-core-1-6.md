---
id: tick-core-1-6
phase: 1
status: pending
created: 2026-01-30
---

# tick create command

## Goal

Tasks 1-1 through 1-4 built the data layer and task 1-5 set up the CLI framework with `tick init`. But there is no way to add tasks. `tick create` is the first mutation command — it takes a title (required) and optional flags (priority, description, blocked-by, blocks, parent), generates an ID, validates inputs, persists via the storage engine's write flow, and outputs the created task details.

## Implementation

- Register `create` subcommand in the CLI dispatcher (tick-core-1-5)
- Parse positional argument: first non-flag argument after `create` is the title (required). If missing: `Error: Title is required. Usage: tick create "<title>" [options]`
- Parse command-specific flags:
  - `--priority <0-4>`: integer, default 2
  - `--description "<text>"`: string, optional
  - `--blocked-by <id,id,...>`: comma-separated task IDs, optional
  - `--blocks <id,id,...>`: comma-separated task IDs (inverse — adds this task to their `blocked_by`), optional
  - `--parent <id>`: single task ID, optional
- Validate title (tick-core-1-1): non-empty after trimming, max 500 chars, no newlines
- Validate priority (tick-core-1-1): integer 0-4
- Generate task ID (tick-core-1-1) with collision check
- Build `Task` struct: id (generated), title (trimmed), status (`open`), priority (flag or 2), description (flag or empty), blocked_by (flag or empty), parent (flag or null), created/updated (current UTC ISO 8601), closed (null)
- Execute via storage engine `Mutate` (tick-core-1-4):
  1. Validate `--blocked-by`, `--blocks`, `--parent` IDs exist
  2. Validate no self-reference
  3. Append new task
  4. For `--blocks` IDs: add new task's ID to their `blocked_by`, update their `updated` timestamp
  5. Return modified task list for atomic write
- Output created task details (basic format for Phase 1; full formatting is Phase 4)
- With `--quiet`: output only the task ID

## Tests

- `"it creates a task with only a title (defaults applied)"`
- `"it creates a task with all optional fields specified"`
- `"it generates a unique ID for the created task"`
- `"it sets status to open on creation"`
- `"it sets default priority to 2 when not specified"`
- `"it sets priority from --priority flag"`
- `"it rejects priority outside 0-4 range"`
- `"it sets description from --description flag"`
- `"it sets blocked_by from --blocked-by flag (single ID)"`
- `"it sets blocked_by from --blocked-by flag (multiple comma-separated IDs)"`
- `"it updates target tasks' blocked_by when --blocks is used"`
- `"it sets parent from --parent flag"`
- `"it errors when title is missing"`
- `"it errors when title is empty string"`
- `"it errors when title is whitespace only"`
- `"it errors when --blocked-by references non-existent task"`
- `"it errors when --blocks references non-existent task"`
- `"it errors when --parent references non-existent task"`
- `"it persists the task to tasks.jsonl via atomic write"`
- `"it outputs full task details on success"`
- `"it outputs only task ID with --quiet flag"`
- `"it normalizes input IDs to lowercase"`
- `"it trims whitespace from title"`

## Edge Cases

- Missing title: error with usage hint, exit code 1. `tick create` with no args must fail.
- Empty title (`""`): after trim, empty → `Error: Title cannot be empty`
- Whitespace-only title: same as empty.
- Non-existent IDs in `--blocked-by`, `--blocks`, `--parent`: error before writing, no partial mutation, task not created.
- `--blocks` modifies other tasks: `tick create "X" --blocks tick-abc` adds new task ID to tick-abc's `blocked_by` and refreshes its `updated` timestamp. Both written atomically.
- ID normalization: `--blocked-by TICK-A1B2` normalized to `tick-a1b2`.

## Acceptance Criteria

- [ ] `tick create "<title>"` creates task with correct defaults (status: open, priority: 2)
- [ ] Generated ID follows `tick-{6 hex}` format, unique among existing tasks
- [ ] All optional flags work: `--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`
- [ ] `--blocks` correctly updates referenced tasks' `blocked_by` arrays
- [ ] Missing or empty title returns error to stderr with exit code 1
- [ ] Invalid priority returns error with exit code 1
- [ ] Non-existent IDs in references return error with exit code 1
- [ ] Task persisted via atomic write through storage engine
- [ ] SQLite cache updated as part of write flow
- [ ] Output shows task details on success
- [ ] `--quiet` outputs only task ID
- [ ] Input IDs normalized to lowercase
- [ ] Timestamps set to current UTC ISO 8601

## Context

Specification: `tick create "<title>" [options]` with flags `--priority <0-4>`, `--blocked-by <id,id,...>`, `--blocks <id,id,...>`, `--parent <id>`, `--description "<text>"`. Default priority 2. Status always `open`.

`--blocks` is inverse of `--blocked-by` — syntactic sugar that modifies other tasks during creation. Only `blocked_by` is stored in the data model.

All mutations go through the storage engine (tick-core-1-4): exclusive lock, JSONL read + freshness check, mutation, atomic write, cache update, lock release. This task provides the mutation function.

Validation of `blocked_by` references (exist check, no self-reference) happens here since it requires the full task list. Cycle detection is Phase 3 scope.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
