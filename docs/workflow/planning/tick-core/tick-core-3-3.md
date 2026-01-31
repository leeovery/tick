---
id: tick-core-3-3
phase: 3
status: pending
created: 2026-01-30
---

# Ready query & tick ready command

## Goal

The core value of Tick — "what should I work on next?" A task is ready when: `open`, all blockers closed, no open children. `tick ready` is an alias for `tick list --ready`. Build both the reusable query function and the CLI command.

## Implementation

- `ReadyQuery` — pure query (or SQLite query) returning tasks matching:
  1. Status is `open`
  2. All blockers `done` or `cancelled`
  3. No children with status `open` or `in_progress`
- Order: priority ASC, created ASC (deterministic)
- Register `ready` subcommand — alias for `list --ready`
- Output: aligned columns like `tick list`
- Empty: `No tasks found.`, exit 0
- `--quiet`: IDs only

## Tests

- `"it returns open task with no blockers and no children"`
- `"it excludes task with open/in_progress blocker"`
- `"it includes task when all blockers done/cancelled"`
- `"it excludes parent with open/in_progress children"`
- `"it includes parent when all children closed"`
- `"it excludes in_progress/done/cancelled tasks"`
- `"it handles deep nesting — only deepest incomplete ready"`
- `"it returns empty list when no tasks ready"`
- `"it orders by priority ASC then created ASC"`
- `"it outputs aligned columns via tick ready"`
- `"it prints 'No tasks found.' when empty"`
- `"it outputs IDs only with --quiet"`

## Edge Cases

- Cancelled blockers count as closed (unblock dependents)
- Leaf-only rule: parent not ready while children are open
- Deep nesting: only deepest incomplete leaf appears
- Empty result: exit 0, not error

## Acceptance Criteria

- [ ] Returns tasks matching all three conditions
- [ ] Open/in_progress blockers exclude task
- [ ] Open/in_progress children exclude task
- [ ] Cancelled blockers unblock
- [ ] Only `open` status returned
- [ ] Deep nesting handled correctly
- [ ] Deterministic ordering
- [ ] `tick ready` outputs aligned columns
- [ ] Empty → `No tasks found.`, exit 0
- [ ] `--quiet` outputs IDs only
- [ ] Query function reusable by blocked query and list filters

## Context

Spec: ready = open + unblocked + no open children. Leaf-only rule means parents wait for children. `tick ready` = alias for `list --ready`. Ordering: priority ASC, created ASC.

Specification reference: `docs/workflow/specification/tick-core.md`
