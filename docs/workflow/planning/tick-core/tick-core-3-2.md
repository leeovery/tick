---
id: tick-core-3-2
phase: 3
status: pending
created: 2026-01-30
---

# tick dep add & tick dep rm commands

## Goal

Wire up CLI for managing dependencies post-creation. `tick dep add <task_id> <blocked_by_id>` and `tick dep rm <task_id> <blocked_by_id>` — parse two positional IDs, validate, call dependency validation (tick-core-3-1 for add), mutate via storage engine.

## Implementation

- Register `dep` with sub-subcommands `add` and `rm`
- Both: two positional IDs (task first, dependency second), normalize to lowercase
- **add**: look up both IDs, check self-ref, check duplicate, call `ValidateDependency`, add to `blocked_by`, update timestamp. Output: `Dependency added: {task_id} blocked by {blocked_by_id}`
- **rm**: look up task_id, check blocked_by_id in array, remove from `blocked_by`, update timestamp. Output: `Dependency removed: {task_id} no longer blocked by {blocked_by_id}`
- `--quiet`: no output. Errors to stderr, exit 1.

## Tests

- `"it adds a dependency between two existing tasks"`
- `"it removes an existing dependency"`
- `"it outputs confirmation on success (add/rm)"`
- `"it updates task's updated timestamp"`
- `"it errors when task_id not found (add/rm)"`
- `"it errors when blocked_by_id not found (add)"`
- `"it errors on duplicate dependency (add)"`
- `"it errors when dependency not found (rm)"`
- `"it errors on self-reference (add)"`
- `"it errors when add creates cycle"`
- `"it errors when add creates child-blocked-by-parent"`
- `"it normalizes IDs to lowercase"`
- `"it suppresses output with --quiet"`
- `"it errors when fewer than two IDs provided"`
- `"it persists via atomic write"`

## Edge Cases

- `rm` does not validate blocked_by_id exists as a task — only checks array membership (supports removing stale refs)
- Duplicate dep on add → error, no mutation
- Missing arguments → error with usage hint
- Cycle/child-blocked-by-parent delegated to tick-core-3-1

## Acceptance Criteria

- [ ] `dep add` adds dependency and outputs confirmation
- [ ] `dep rm` removes dependency and outputs confirmation
- [ ] Non-existent IDs return error
- [ ] Duplicate/missing dep return error
- [ ] Self-ref, cycle, child-blocked-by-parent return error
- [ ] IDs normalized to lowercase
- [ ] `--quiet` suppresses output
- [ ] `updated` timestamp refreshed
- [ ] Persisted through storage engine

## Context

Spec: `tick dep add tick-c3d4 tick-a1b2` = "c3d4 blocked by a1b2". Validation from tick-core-3-1. Storage engine (tick-core-1-4) handles locking/write.

Specification reference: `docs/workflow/specification/tick-core.md`
