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
