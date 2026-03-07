---
id: tick-core-2-1
phase: 2
status: completed
created: 2026-01-30
---

# Status transition validation logic

## Goal

Tasks need to move through a defined lifecycle: open → in_progress → done/cancelled, with reopen to reverse closures. Without validated transitions, tasks could reach invalid states. Implement a pure validation function that enforces the 4 valid transitions, rejects all invalid ones, and manages the `closed` and `updated` timestamps as side effects.

## Implementation

- Add a `Transition(task *Task, command string) error` function that applies a status transition by command name (`start`, `done`, `cancel`, `reopen`)
- Valid transitions:
  - `start`: `open` → `in_progress`
  - `done`: `open` or `in_progress` → `done`
  - `cancel`: `open` or `in_progress` → `cancelled`
  - `reopen`: `done` or `cancelled` → `open`
- On valid transition: set new status, set `updated` to current UTC, set `closed` on done/cancelled, clear `closed` on reopen
- On invalid transition: return error without modifying task
- Error format: `Error: Cannot {command} task tick-{id} — status is '{current_status}'`
- Pure domain logic — no I/O. Caller handles persistence.
- Return old and new status for output formatting (`tick-a3f2b7: open → in_progress`)

## Tests

- `"it transitions open to in_progress via start"`
- `"it transitions open to done via done"`
- `"it transitions in_progress to done via done"`
- `"it transitions open to cancelled via cancel"`
- `"it transitions in_progress to cancelled via cancel"`
- `"it transitions done to open via reopen"`
- `"it transitions cancelled to open via reopen"`
- `"it rejects start on in_progress task"`
- `"it rejects start on done task"`
- `"it rejects start on cancelled task"`
- `"it rejects done on done task"`
- `"it rejects done on cancelled task"`
- `"it rejects cancel on done task"`
- `"it rejects cancel on cancelled task"`
- `"it rejects reopen on open task"`
- `"it rejects reopen on in_progress task"`
- `"it sets closed timestamp when transitioning to done"`
- `"it sets closed timestamp when transitioning to cancelled"`
- `"it clears closed timestamp when reopening"`
- `"it updates the updated timestamp on every valid transition"`
- `"it does not modify task on invalid transition"`

## Edge Cases

- 7 valid from→to pairs, 9 invalid pairs — each invalid must return error and leave task unmodified
- Closed timestamp set on done/cancelled, cleared on reopen
- Updated timestamp refreshed on every valid transition
- Task not mutated on error

## Acceptance Criteria

- [ ] All 7 valid status transitions succeed with correct new status
- [ ] All 9 invalid transitions return error
- [ ] Task not modified on invalid transition
- [ ] `closed` set to current UTC on done/cancelled
- [ ] `closed` cleared on reopen
- [ ] `updated` refreshed on every valid transition
- [ ] Error messages include command name and current status
- [ ] Function returns old and new status

## Context

4 status values: `open`, `in_progress`, `done`, `cancelled`. The `closed` field is optional datetime — set on done/cancelled, cleared on reopen. `updated` always refreshed on any mutation.

Specification reference: `docs/workflow/specification/tick-core.md`
