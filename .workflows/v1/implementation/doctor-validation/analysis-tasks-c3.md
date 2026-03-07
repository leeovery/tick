---
topic: doctor-validation
cycle: 3
total_proposed: 1
---
# Analysis Tasks: Doctor Validation (Cycle 3)

## Task 1: Extract buildKnownIDs helper to eliminate 3-file duplication
status: approved
severity: medium
sources: duplication

**Problem**: The identical 3-line knownIDs map construction (`make(map[string]struct{}, len(tasks))` + range loop) is copy-pasted in `orphaned_parent.go`, `orphaned_dependency.go`, and `dependency_cycle.go`. The two orphaned-reference checks also share ~25 lines of near-identical scaffolding beyond knownIDs, but extracting a fully generic check would over-abstract for two callers.

**Solution**: Extract a `buildKnownIDs(tasks []TaskRelationshipData) map[string]struct{}` helper function in a shared file (e.g., `internal/doctor/helpers.go` or the existing helpers location). Replace the 3-line construction in all three check files with a call to this helper.

**Outcome**: The knownIDs construction exists in exactly one place. All three check files become slightly shorter and the intent ("build lookup set of known IDs") is expressed at the right abstraction level. No behaviour change.

**Do**:
1. In `internal/doctor/helpers.go` (or wherever `fileNotFoundResult` lives), add:
   ```go
   // buildKnownIDs returns a set of all task IDs from the given relationship data.
   func buildKnownIDs(tasks []TaskRelationshipData) map[string]struct{} {
       knownIDs := make(map[string]struct{}, len(tasks))
       for _, task := range tasks {
           knownIDs[task.ID] = struct{}{}
       }
       return knownIDs
   }
   ```
2. In `internal/doctor/orphaned_parent.go`, replace lines 23-26 with `knownIDs := buildKnownIDs(tasks)`.
3. In `internal/doctor/orphaned_dependency.go`, replace lines 23-26 with `knownIDs := buildKnownIDs(tasks)`.
4. In `internal/doctor/dependency_cycle.go`, replace lines 27-30 with `knownIDs := buildKnownIDs(tasks)`.
5. Run all existing doctor tests to confirm no regressions.

**Acceptance Criteria**:
- `buildKnownIDs` helper exists and is used by all three check files
- No inline `knownIDs` map construction remains in any check file
- All existing tests pass without modification

**Tests**:
- Existing tests for OrphanedParentCheck, OrphanedDependencyCheck, and DependencyCycleCheck continue to pass (no new tests needed -- this is a pure refactor with no behaviour change)
