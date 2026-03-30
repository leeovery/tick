# Plan: Qualify Command Tree Leaks To Note

## Phase 1: Fix Tree Qualification Leak To Note
status: approved
approved_at: 2026-03-30

**Goal**: Prevent `qualifyCommand` from qualifying `"tree"` as a subcommand of `"note"`, and add regression tests that would have caught this bug.

**Rationale**: Single-phase fix. The bug has one root cause (a missing parent guard in one switch case at `app.go:374-376`), affects one function, and requires minimal code change. No distinct stages warrant separate checkpoints.

**Acceptance**:
- [ ] `qualifyCommand("note", ["tree"])` returns `("note", ["tree"])` — tree is not qualified under note
- [ ] `qualifyCommand("note", ["tree", "--foo"])` returns `("note", ["tree", "--foo"])` — args preserved unchanged
- [ ] `qualifyCommand("dep", ["tree", ...])` behavior unchanged (existing tests still pass)
- [ ] `qualifyCommand` for shared subcommands (`"add"`, `"remove"`) still qualifies correctly under both `"dep"` and `"note"`
- [ ] `tick note tree` produces `"unknown note sub-command 'tree'"` (no misleading "note tree" reference)
- [ ] `tick note tree --foo` error references `"note"`, not `"note tree"`
- [ ] All existing tests pass with no regressions

#### Tasks
status: approved
approved_at: 2026-03-30

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| qualify-command-tree-leaks-to-note-1-1 | Reproduce Bug and Fix qualifyCommand | qualifyCommand("note", ["tree"]) returns ("note", ["tree"]) not ("note tree", []), shared add/remove still qualify under both dep and note |
| qualify-command-tree-leaks-to-note-1-2 | Integration Regression Test via App.Run | tick note tree --foo error references "note" not "note tree", tick note tree produces "unknown note sub-command 'tree'", tick dep tree behavior unchanged |
