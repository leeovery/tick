---
topic: auto-cascade-parent-status
status: concluded
format: tick
work_type: feature
ext_id: tick-b43d8e
specification: ../specification/auto-cascade-parent-status/specification.md
spec_commit: 1f0cc547b94c612b74f1f051d505f1915721b11b
created: 2026-03-05
updated: 2026-03-05
external_dependencies: []
task_list_gate_mode: auto
author_gate_mode: auto
finding_gate_mode: gated
review_cycle: 4
planning:
  phase: 3
  task: ~
---

# Plan: Auto-Cascade Parent Status

### Phase 1: StateMachine Core with Migration
status: approved
approved_at: 2026-03-05
ext_id: tick-38ca65

**Goal**: Create the StateMachine struct in `internal/task/` with `Transition()`, `ValidateAddChild()`, and `ValidateAddDep()` methods, migrating existing logic from `transition.go` and `dependency.go`. Implements Rules 1, 7, 8, 9, 10, 11.

**Why this order**: The StateMachine is the architectural foundation all cascade behavior builds on. Migrating existing transition and dependency validation logic first establishes the pattern and ensures no regression before adding new cascade rules.

**Acceptance**:
- [ ] `StateMachine.Transition()` passes all existing transition tests with identical behavior to `task.Transition()`
- [ ] `StateMachine.ValidateAddDep()` passes all existing dependency validation tests (cycle detection, child-blocked-by-parent)
- [ ] `ValidateAddChild()` blocks adding child to cancelled parent with correct error message (Rule 7)
- [ ] `ValidateAddDep()` blocks adding dependency on cancelled task with correct error message (Rule 8)
- [ ] `Transition()` blocks reopen under cancelled parent with correct error message (Rule 9)
- [ ] All existing callers in `internal/cli/` updated to use StateMachine methods
- [ ] All existing tests pass with no regressions

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| acps-1-1 | Create StateMachine struct with Transition method | unknown command, no-op on invalid transition (task unmodified) | authored | tick-7dc647 |
| acps-1-2 | Migrate ValidateAddDep into StateMachine | mixed-case IDs, self-reference, multi-hop cycles | authored | tick-74527d |
| acps-1-3 | Add ValidateAddChild for cancelled parent guard | none | authored | tick-cdaef4 |
| acps-1-4 | Add cancelled dependency guard to ValidateAddDep | none | authored | tick-e272cc |
| acps-1-5 | Add reopen-under-cancelled-parent guard to Transition | cancelled grandparent with non-cancelled direct parent does not block | authored | tick-dc1dbf |
| acps-1-6 | Update CLI callers to use StateMachine methods | none | authored | tick-f998b0 |

### Phase 2: Cascade Logic and Transition History
status: approved
approved_at: 2026-03-05
ext_id: tick-08e6f9

**Goal**: Implement `Cascades()` and `ApplyWithCascades()` with queue-based cascade processing (Rules 2-6), plus the `Transitions` field on Task and `task_transitions` SQLite cache table.

**Why this order**: Cascade computation is the core new behavior of this feature. Transition history must land alongside cascades because cascades produce auto-flagged transition records. Both are domain/storage concerns that must be solid before CLI integration.

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

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| acps-2-1 | Add Transition struct and Transitions field to Task | empty transitions array omitted from JSON, backward-compatible deserialization of tasks without transitions field | authored | tick-9e630d |
| acps-2-2 | Add task_transitions table to SQLite cache schema | pre-existing cache triggers delete+rebuild via version mismatch | authored | tick-c0b5c7 |
| acps-2-3 | Implement Cascades for upward start cascade (Rule 2) | ancestor already in_progress skipped, deeply nested chain 5+ levels | authored | tick-4fe6a9 |
| acps-2-4 | Implement Cascades for downward done/cancel cascade (Rule 4) | mixed terminal/non-terminal children, child with unresolved deps still cascaded | authored | tick-240557 |
| acps-2-5 | Implement Cascades for upward completion cascade (Rule 3) | single child trivial case, mix of done and cancelled, parent with no children | authored | tick-30f9f9 |
| acps-2-6 | Implement Cascades for reopen under done parent (Rule 5) | parent not done no-ops, deeply nested done ancestors, cancelled ancestor blocks | authored | tick-902c65 |
| acps-2-7 | Implement ApplyWithCascades with queue-based processing | multi-level cascades chain, seen-map deduplication, empty cascade list | authored | tick-d5cbbc |

### Phase 3: CLI Integration and Cascade Display
status: approved
approved_at: 2026-03-05
ext_id: tick-02a182

**Goal**: Wire `ApplyWithCascades()` into `RunTransition` and parent-modifying commands (create with parent, update/reparent). Add `FormatCascadeTransition` to the Formatter interface with Toon, Pretty, and JSON implementations showing cascaded and unchanged tasks.

**Why this order**: With domain logic and storage complete, this phase delivers the user-facing vertical slice -- cascades become visible through all three output formats and all relevant CLI commands.

**Acceptance**:
- [ ] `RunTransition` uses `StateMachine.ApplyWithCascades()` and persists all changes atomically in a single `Store.Mutate` call
- [ ] Creating a task with a done parent triggers reopen cascade (Rule 6) with cascade output
- [ ] Reparenting via update triggers Rule 6 (done parent reopen) and Rule 3 (original parent re-evaluation)
- [ ] Pretty format renders cascade tree with box-drawing characters, unchanged terminal children shown
- [ ] Toon format renders flat lines with `(auto)` and `(unchanged)` markers
- [ ] JSON format renders structured object with `transition`, `cascaded`, and `unchanged` keys
- [ ] Non-cascade single-task transitions still use existing `FormatTransition` with no visual regression
- [ ] Unchanged terminal children appear in all cascade output formats

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| acps-3-1 | Add CascadeResult type and FormatCascadeTransition to Formatter interface | empty cascaded list, empty unchanged list, both empty | authored | tick-c4dc82 |
| acps-3-2 | Implement FormatCascadeTransition for Toon, Pretty, and JSON formatters | deeply nested tree in Pretty, mixed cascaded and unchanged children, single cascade entry | authored | tick-079053 |
| acps-3-3 | Wire ApplyWithCascades into RunTransition | no cascades (single-task uses FormatTransition), quiet mode suppresses output, task not found | authored | tick-a24919 |
| acps-3-4 | Wire ValidateAddChild and done-parent reopen cascade into RunCreate | parent cancelled (error), parent open (no cascade), parent done (reopen cascade) | authored | tick-1dd3c8 |
| acps-3-5 | Wire reparenting cascade logic into RunUpdate | reparent away triggers Rule 3 on original parent, reparent to done triggers Rule 6, clear parent | authored | tick-2bf0f6 |
