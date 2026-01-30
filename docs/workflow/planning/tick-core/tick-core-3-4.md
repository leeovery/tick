---
id: tick-core-3-4
phase: 3
status: pending
created: 2026-01-30
---

# Blocked query, tick blocked & cancel-unblocks-dependents

## Goal

Inverse of ready: open tasks that can't be worked. `tick blocked` = alias for `list --blocked`. Also verify cancel-unblocks-dependents behavior end-to-end.

## Implementation

- `BlockedQuery` — open tasks that fail ready conditions (have unclosed blocker OR have open children)
- Simplest: blocked = open minus ready (reuse `ReadyQuery`)
- Order: priority ASC, created ASC
- Register `blocked` subcommand
- Verify cancelled blockers count as closed → dependents unblock
- Only `open` tasks in output

## Tests

- `"it returns open task blocked by open/in_progress dep"`
- `"it returns parent with open/in_progress children"`
- `"it excludes task when all blockers done/cancelled"`
- `"it excludes in_progress/done/cancelled from output"`
- `"it returns empty when no blocked tasks"`
- `"it orders by priority ASC then created ASC"`
- `"it outputs aligned columns via tick blocked"`
- `"it prints 'No tasks found.' when empty"`
- `"it outputs IDs only with --quiet"`
- `"cancel unblocks single dependent → moves to ready"`
- `"cancel unblocks multiple dependents"`
- `"cancel does not unblock dependent still blocked by another"`

## Edge Cases

- Open and in_progress blockers both keep task blocked
- Parent with open children blocked even without `blocked_by`
- Partial unblock: two blockers, one cancelled → still blocked
- Empty: exit 0

## Acceptance Criteria

- [ ] Returns open tasks with unclosed blockers or open children
- [ ] Excludes tasks where all blockers closed
- [ ] Only `open` in output
- [ ] Cancel → dependent unblocks
- [ ] Multiple dependents unblock simultaneously
- [ ] Partial unblock works correctly
- [ ] Deterministic ordering
- [ ] `tick blocked` outputs aligned columns
- [ ] Empty → `No tasks found.`, exit 0
- [ ] `--quiet` IDs only
- [ ] Reuses ready query logic

## Context

Spec: blocked = open AND NOT ready. `tick blocked` = `list --blocked`. Cancelled = closed for unblocking. Inverse of tick-core-3-3.

Specification reference: `docs/workflow/specification/tick-core.md`
