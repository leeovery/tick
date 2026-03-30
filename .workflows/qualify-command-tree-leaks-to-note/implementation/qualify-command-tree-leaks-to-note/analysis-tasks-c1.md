---
topic: qualify-command-tree-leaks-to-note
cycle: 1
total_proposed: 1
---
# Analysis Tasks: qualify-command-tree-leaks-to-note (Cycle 1)

## Task 1: Remove redundant dep-tree regression tests from note_test.go
status: approved
external_id: tick-30a977
severity: medium
sources: duplication

**Problem**: Two subtests in `TestNoteTreeRejection` (note_test.go:569-593) duplicate coverage that already exists in `TestDepTreeWiring` (dep_tree_test.go:34-64). Specifically: "it preserves dep tree dispatch" is functionally identical to "it dispatches dep tree without error", and "it preserves dep tree flag validation" is functionally identical to "it rejects unknown flag on dep tree". These were authored by separate task executors and independently verify the same dep-tree behavior.

**Solution**: Remove the two redundant subtests from `TestNoteTreeRejection` in `internal/cli/note_test.go`. The canonical dep-tree tests in `dep_tree_test.go` already provide this coverage. The remaining note-specific subtests in `TestNoteTreeRejection` (tree rejection, unknown sub-command error, flag error not referencing "note tree") are sufficient regression coverage for the note side of the fix.

**Outcome**: `TestNoteTreeRejection` contains only note-specific regression tests. Dep-tree dispatch and flag validation are tested in exactly one place (`dep_tree_test.go`). All tests pass.

**Do**:
1. Open `internal/cli/note_test.go`
2. Delete the subtest `"it preserves dep tree dispatch"` (lines 569-582)
3. Delete the subtest `"it preserves dep tree flag validation"` (lines 584-593)
4. Run `go test ./internal/cli/` to confirm all tests pass
5. Run `go vet ./internal/cli` to confirm no issues

**Acceptance Criteria**:
- `TestNoteTreeRejection` no longer contains subtests that test dep-tree behavior
- `TestDepTreeWiring` in dep_tree_test.go continues to cover dep-tree dispatch and flag validation
- `go test ./internal/cli/` passes with zero failures

**Tests**:
- `go test ./internal/cli/ -run TestNoteTreeRejection` passes and only runs note-specific subtests
- `go test ./internal/cli/ -run TestDepTreeWiring` passes (canonical dep-tree coverage unaffected)
- `go test ./internal/cli/` passes (no regressions)
