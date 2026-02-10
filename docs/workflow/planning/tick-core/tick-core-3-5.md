---
id: tick-core-3-5
phase: 3
status: completed
created: 2026-01-30
---

# tick list filter flags — --ready, --blocked, --status, --priority

## Goal

Wire ready/blocked queries into `tick list` as flags, plus add `--status` and `--priority` filters. Completes list to full spec.

## Implementation

- Four flags on `list`: `--ready` (bool), `--blocked` (bool), `--status <s>`, `--priority <p>`
- `--ready`/`--blocked` mutually exclusive → error if both
- `--status` validates: open, in_progress, done, cancelled
- `--priority` validates: 0-4
- Filters combine as AND
- Contradictory combos (e.g., `--status done --ready`) → empty result, no error
- No filters = all tasks (backward compatible)
- Reuses ReadyQuery/BlockedQuery from 3-3/3-4
- Output: aligned columns, `--quiet` IDs only

## Tests

- `"it filters to ready tasks with --ready"`
- `"it filters to blocked tasks with --blocked"`
- `"it filters by --status (all 4 values)"`
- `"it filters by --priority"`
- `"it combines --ready with --priority"`
- `"it combines --status with --priority"`
- `"it errors when --ready and --blocked both set"`
- `"it errors for invalid status/priority values"`
- `"it returns 'No tasks found.' when no matches"`
- `"it outputs IDs only with --quiet after filtering"`
- `"it returns all tasks with no filters"`
- `"it maintains deterministic ordering"`

## Edge Cases

- `--ready` + `--blocked` → mutual exclusion error
- Invalid status/priority → error with valid options
- Contradictory filters → empty result, no error
- No matches → exit 0, not error

## Acceptance Criteria

- [ ] `list --ready` = same as `tick ready`
- [ ] `list --blocked` = same as `tick blocked`
- [ ] `--status` filters by exact match
- [ ] `--priority` filters by exact match
- [ ] Filters AND-combined
- [ ] `--ready` + `--blocked` → error
- [ ] Invalid values → error with valid options
- [ ] No matches → `No tasks found.`, exit 0
- [ ] `--quiet` outputs filtered IDs
- [ ] Backward compatible (no filters = all)
- [ ] Reuses query functions

## Context

Spec: `tick list` options --ready, --blocked, --status, --priority. Aliases `tick ready`/`tick blocked` already built. Status enum: open, in_progress, done, cancelled. Priority: 0-4.

Specification reference: `docs/workflow/specification/tick-core.md`
