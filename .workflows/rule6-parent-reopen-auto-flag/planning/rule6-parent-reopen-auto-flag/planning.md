# Plan: Rule6 Parent Reopen Auto Flag

### Phase 1: Fix auto flag on system-initiated transitions
status: approved
ext_id: tick-359ce4
approved_at: 2026-03-12

**Goal**: Parameterize the auto flag in ApplyWithCascades so system-initiated transitions (Rule 6 parent reopen, Rule 3 reparent auto-completion) correctly record auto=true on the primary target's TransitionRecord.

**Why this order**: Single-phase bugfix — one root cause, contained fix, all changes are highly cohesive. The refactoring, call-site updates, rename, and tests cannot be meaningfully separated without creating intermediate states that have no independent value.

**Acceptance**:
- [ ] `ApplyWithCascades` is unexported (`applyWithCascades`); `ApplyUserTransition` and `ApplySystemTransition` are the only public entry points
- [ ] `evaluateRule3` is renamed to `autoCompleteParentIfTerminal`
- [ ] All three call sites use the correct wrapper (`RunTransition` calls `ApplyUserTransition`; `validateAndReopenParent` and `autoCompleteParentIfTerminal` call `ApplySystemTransition`)
- [ ] Existing 13 `ApplyWithCascades` subtests pass unchanged under `ApplyUserTransition`
- [ ] New unit tests verify `ApplyUserTransition` records `auto=false` and `ApplySystemTransition` records `auto=true` on the primary target
- [ ] Integration test confirms `create --parent <done-parent>` produces `auto=true` on parent reopen transition in JSONL
- [ ] Integration test confirms `update --parent` reparent triggers auto-completion with `auto=true` in JSONL
- [ ] `go test ./...` passes with no regressions

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| rule6-parent-reopen-auto-flag-1-1 | Refactor ApplyWithCascades into user/system wrappers with failing test | none | authored | tick-0930d3 |
| rule6-parent-reopen-auto-flag-1-2 | Update call sites and rename evaluateRule3 | none | authored | tick-be26d8 |
| rule6-parent-reopen-auto-flag-1-3 | Integration tests for auto flag in JSONL | none | authored | tick-d6e894 |
