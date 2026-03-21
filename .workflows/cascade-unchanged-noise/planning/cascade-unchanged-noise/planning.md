# Plan: Cascade Unchanged Noise

## Phases

### Phase 1: Remove unchanged cascade output
status: approved
approved_at: 2026-03-21

**Goal**: Eliminate the "unchanged" feature from cascade output — remove the type, collection logic, rendering in all three formatters, and associated tests — then add a negative-case test confirming terminal siblings no longer appear in cascade output.

**Why this order**: Single-phase fix. The bug has one root cause (intentional but noisy unchanged collection), the fix is pure deletion across one package, and all changes are interdependent. No meaningful checkpoint exists between removing the type and updating the formatters that use it.

**Acceptance**:
- [ ] `UnchangedEntry` type and `Unchanged` field no longer exist in `CascadeResult`
- [ ] `buildCascadeResult()` no longer collects unchanged terminal descendants (involvedIDs map and collection loop removed)
- [ ] Toon, Pretty, and JSON formatters no longer render unchanged entries
- [ ] A test confirms that terminal siblings are NOT included in cascade output (the negative case)
- [ ] All existing tests in `internal/cli/` pass (updated to remove Unchanged references)
- [ ] `go vet ./...` and full test suite `go test ./...` pass with no failures

#### Tasks
status: draft

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| cascade-unchanged-noise-1-1 | Add negative-case test confirming terminal siblings excluded from cascade output | terminal child with done status, terminal child with cancelled status |
| cascade-unchanged-noise-1-2 | Remove UnchangedEntry type and all unchanged collection and rendering | pretty formatter tree node count with only cascaded entries, JSON output no longer includes unchanged key, empty cascade result still renders correctly |
