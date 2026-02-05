---
id: tick-core-2-2
phase: 2
status: completed
created: 2026-01-30
---

# tick start, done, cancel, reopen commands

## Goal

tick-core-2-1 built the pure transition logic but has no CLI surface. Users and agents need `tick start <id>`, `tick done <id>`, `tick cancel <id>`, and `tick reopen <id>` commands. Each parses the ID, loads the task, applies the transition, persists via the storage engine, and outputs the result.

## Implementation

- Register four subcommands: `start`, `done`, `cancel`, `reopen`
- All share the same flow (parameterized by command name):
  1. Parse positional ID argument. If missing: `Error: Task ID is required. Usage: tick {command} <id>`
  2. Normalize ID to lowercase
  3. Execute via storage engine `Mutate`:
     a. Look up task by ID — not found → error
     b. Call `Transition(task, command)` — invalid → error
     c. Return modified task list
  4. Output: `{id}: {old_status} → {new_status}`
  5. `--quiet`: no output on success
- Errors to stderr, exit code 1. Success to stdout, exit code 0.

## Tests

- `"it transitions task to in_progress via tick start"`
- `"it transitions task to done via tick done from open"`
- `"it transitions task to done via tick done from in_progress"`
- `"it transitions task to cancelled via tick cancel from open"`
- `"it transitions task to cancelled via tick cancel from in_progress"`
- `"it transitions task to open via tick reopen from done"`
- `"it transitions task to open via tick reopen from cancelled"`
- `"it outputs status transition line on success"`
- `"it suppresses output with --quiet flag"`
- `"it errors when task ID argument is missing"`
- `"it errors when task ID is not found"`
- `"it errors on invalid transition"`
- `"it writes errors to stderr"`
- `"it exits with code 1 on error"`
- `"it normalizes task ID to lowercase"`
- `"it persists status change via atomic write"`
- `"it sets closed timestamp on done/cancel"`
- `"it clears closed timestamp on reopen"`

## Edge Cases

- Output format: `{id}: {old_status} → {new_status}` with unicode arrow
- `--quiet` suppresses stdout; errors still go to stderr
- Task not found: error with normalized ID
- Case-insensitive: `TICK-A1B2` → `tick-a1b2`
- Missing ID: error with usage hint

## Acceptance Criteria

- [ ] All four commands transition correctly and output transition line
- [ ] Invalid transitions return error to stderr with exit code 1
- [ ] Missing/not-found task ID returns error with exit code 1
- [ ] `--quiet` suppresses success output
- [ ] Input IDs normalized to lowercase
- [ ] Timestamps managed correctly (closed set/cleared, updated refreshed)
- [ ] Mutation persisted through storage engine

## Context

Output format from spec: `tick-a3f2b7: open → in_progress`. Storage engine (tick-core-1-4) handles locking, freshness, atomic write. Transition function (tick-core-2-1) is pure domain logic.

Specification reference: `docs/workflow/specification/tick-core.md`
