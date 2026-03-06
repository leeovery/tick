---
status: in-progress
created: 2026-03-06
cycle: 2
phase: Plan Integrity Review
topic: Auto-Cascade Parent Status
---

# Review Tracking: Auto-Cascade Parent Status - Integrity

## Findings

### 1. Phase 2 acceptance criterion references reparenting behavior delivered in Phase 3

**Severity**: Important
**Plan Reference**: Phase 2 (tick-08e6f9)
**Category**: Phase Structure
**Change Type**: update-task

**Details**:
Phase 2 acceptance includes "Reparenting away triggers Rule 3 re-evaluation on original parent" but this behavior is implemented in Phase 3 task acps-3-5 (Wire reparenting cascade logic into RunUpdate). Phase 2 builds the domain-level Cascades() and ApplyWithCascades() methods -- it does not wire reparenting logic. An implementer verifying Phase 2 completion would be unable to satisfy this criterion because the reparenting wiring does not exist yet.

**Current**:
```
**Acceptance**:
- [ ] Upward start cascade sets open ancestors to `in_progress` recursively (Rule 2)
- [ ] Upward completion cascade triggers parent `done` when at least one child done, `cancelled` when all cancelled (Rule 3)
- [ ] Downward cascade propagates `done`/`cancelled` to non-terminal children recursively, leaves terminal children untouched (Rule 4)
- [ ] Reopen child under done parent reopens parent recursively up ancestor chain (Rule 5)
- [ ] Adding non-terminal child to done parent triggers reopen to open (Rule 6 -- caller-side logic validated)
- [ ] Reparenting away triggers Rule 3 re-evaluation on original parent
- [ ] Queue-based processing with seen-map deduplication terminates correctly on deep hierarchies
- [ ] `Task.Transitions` array field serializes/deserializes correctly in JSONL with `auto` boolean
- [ ] `task_transitions` table created in SQLite cache schema; `schemaVersion` incremented
- [ ] `ApplyWithCascades()` returns primary `TransitionResult` plus all `[]CascadeChange` entries
```

**Proposed**:
```
**Acceptance**:
- [ ] Upward start cascade sets open ancestors to `in_progress` recursively (Rule 2)
- [ ] Upward completion cascade triggers parent `done` when at least one child done, `cancelled` when all cancelled (Rule 3)
- [ ] Downward cascade propagates `done`/`cancelled` to non-terminal children recursively, leaves terminal children untouched (Rule 4)
- [ ] Reopen child under done parent reopens parent recursively up ancestor chain (Rule 5)
- [ ] Adding non-terminal child to done parent triggers reopen to open (Rule 6 -- caller-side logic validated)
- [ ] Queue-based processing with seen-map deduplication terminates correctly on deep hierarchies
- [ ] `Task.Transitions` array field serializes/deserializes correctly in JSONL with `auto` boolean
- [ ] `task_transitions` table created in SQLite cache schema; `schemaVersion` incremented
- [ ] `ApplyWithCascades()` returns primary `TransitionResult` plus all `[]CascadeChange` entries
```

**Resolution**: Pending
**Notes**: The reparenting Rule 3 re-evaluation criterion belongs in Phase 3 acceptance (where acps-3-5 implements it). Removing from Phase 2 rather than moving to Phase 3 because Phase 3 acceptance already covers it via "Reparenting via update triggers Rule 6 (done parent reopen) and Rule 3 (original parent re-evaluation)".

---

### 2. acps-1-5 edge case description misaligned with direct-parent-only check

**Severity**: Minor
**Plan Reference**: Phase 1 / acps-1-5 (tick-dc1dbf)
**Category**: Acceptance Criteria Quality
**Change Type**: update-task

**Details**:
The plan table lists edge case "deeply nested ancestor chain with cancelled grandparent" for acps-1-5, but the task's Do section explicitly says "Only check the direct parent, not the full ancestor chain." The edge case as described is ambiguous -- it could be misread as requiring the grandparent check to block reopen. The tests section also lacks a test verifying that a cancelled grandparent with a non-cancelled direct parent does NOT block reopen, which would be the correct positive assertion for this edge case.

**Current**:
```
Edge Cases:
- Task whose parent ID references a non-existent task -- skip check, proceed with reopen (defensive)
```

Plan table edge case column: `deeply nested ancestor chain with cancelled grandparent`

**Proposed**:
```
Edge Cases:
- Task whose parent ID references a non-existent task -- skip check, proceed with reopen (defensive)
- Cancelled grandparent with non-cancelled direct parent does not block reopen (only direct parent checked)
```

Plan table edge case column: `cancelled grandparent with non-cancelled direct parent does not block`

Additionally, add to Tests:
```
- "it allows reopen when grandparent is cancelled but direct parent is not"
```

**Resolution**: Pending
**Notes**: The current edge case description in the plan table could mislead an implementer into thinking grandparent status should be checked. The proposed wording clarifies the boundary.

---
