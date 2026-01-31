---
id: tick-core-3-1
phase: 3
status: pending
created: 2026-01-30
---

# Dependency validation — cycle detection & child-blocked-by-parent

## Goal

Reject two categories of invalid dependencies at write time: circular dependencies (deadlocks) and child-blocked-by-parent (unworkable due to leaf-only ready rule). Pure validation functions called before persisting.

## Implementation

- `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error` — checks cycle + child-blocked-by-parent
- **Cycle detection**: DFS/BFS from `newBlockedByID` following `blocked_by` edges to detect if `taskID` is reachable. Return error with full cycle path.
- **Child-blocked-by-parent**: If task's parent equals `newBlockedByID`, reject.
- **Allowed**: parent blocked by child, sibling deps, cross-hierarchy deps
- Error formats: `Error: Cannot add dependency - creates cycle: tick-a → tick-b → tick-a` and `Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent`
- `ValidateDependencies(tasks, taskID, blockedByIDs)` for batch — validate sequentially, fail on first error
- Pure domain logic — no I/O

## Tests

- `"it allows valid dependency between unrelated tasks"`
- `"it rejects direct self-reference"`
- `"it rejects 2-node cycle with path"`
- `"it rejects 3+ node cycle with full path"`
- `"it rejects child blocked by own parent"`
- `"it allows parent blocked by own child"`
- `"it allows sibling dependencies"`
- `"it allows cross-hierarchy dependencies"`
- `"it returns cycle path format: tick-a → tick-b → tick-a"`
- `"it validates multiple blocked_by IDs, fails on first error"`
- `"it detects cycle through existing multi-hop chain"`

## Edge Cases

- Self-reference: defense-in-depth (also caught by model validation)
- 2-node cycle: simplest cycle
- 3+ node cycle: must reconstruct full path, not just "cycle detected"
- Child-blocked-by-parent: only direct parent checked, not grandparent
- Batch validation: sequential, fail on first

## Acceptance Criteria

- [ ] Self-reference rejected
- [ ] 2-node cycle detected with path
- [ ] 3+ node cycle detected with full path
- [ ] Child-blocked-by-parent rejected with descriptive error
- [ ] Parent-blocked-by-child allowed
- [ ] Sibling/cross-hierarchy deps allowed
- [ ] Error messages match spec format
- [ ] Batch validation fails on first error
- [ ] Pure functions — no I/O

## Context

Spec: `blocked_by` must not create cycles. Cycle detection at write time. Child-blocked-by-parent creates deadlock via leaf-only rule. Validation called by `tick create --blocked-by`, `tick dep add`, and future commands.

Specification reference: `docs/workflow/specification/tick-core.md`
