---
topic: auto-cascade-parent-status
status: planning
format: tick
work_type: feature
ext_id:
specification: ../specification/auto-cascade-parent-status/specification.md
spec_commit: 1f0cc547b94c612b74f1f051d505f1915721b11b
created: 2026-03-05
updated: 2026-03-05
external_dependencies: []
task_list_gate_mode: gated
author_gate_mode: gated
finding_gate_mode: gated
planning:
  phase: 1
  task: ~
---

# Plan: Auto-Cascade Parent Status

### Phase 1: StateMachine Core with Migration
status: approved
approved_at: 2026-03-05
ext_id:

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

### Phase 2: Cascade Logic and Transition History
status: approved
approved_at: 2026-03-05
ext_id:

**Goal**: Implement `Cascades()` and `ApplyWithCascades()` with queue-based cascade processing (Rules 2-6), plus the `Transitions` field on Task and `task_transitions` SQLite cache table.

**Why this order**: Cascade computation is the core new behavior of this feature. Transition history must land alongside cascades because cascades produce auto-flagged transition records. Both are domain/storage concerns that must be solid before CLI integration.

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

### Phase 3: CLI Integration and Cascade Display
status: approved
approved_at: 2026-03-05
ext_id:

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
