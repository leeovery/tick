---
id: tick-core-8-1
phase: 8
status: pending
created: 2026-02-10
---

# Prevent duplicate blocked_by entries in applyBlocks

**Problem**: The `applyBlocks` helper in `helpers.go` blindly appends `sourceID` to target tasks' `BlockedBy` slices without checking for duplicates. When `tick update T1 --blocks T2` is called and T1 is already in T2's `blocked_by`, T1 gets appended a second time. This contradicts the spec's `blocked_by` semantics (an array of IDs implying unique blockers) and is inconsistent with `tick dep add`, which explicitly rejects duplicates with "dependency already exists: %s is already blocked by %s" (dep.go:97-101). The `ValidateDependency` call that follows in update.go only checks cycles and child-blocked-by-parent, not duplicates.

**Solution**: Add duplicate checking in `applyBlocks` before appending. For each target task, check whether `sourceID` already exists in its `BlockedBy` slice and skip the append if so. This aligns the `--blocks` flag behavior with `dep add`'s existing duplicate rejection.

**Outcome**: The `--blocks` flag on both `create` and `update` silently skips already-present dependencies instead of creating duplicates, consistent with how `dep add` handles duplicates.

**Do**:
1. In `/Users/leeovery/Code/tick/internal/cli/helpers.go`, modify the `applyBlocks` function.
2. Before `tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)`, add a check: iterate `tasks[i].BlockedBy` to see if `sourceID` is already present. If found, skip the append (do not update the timestamp either, since no change was made).
3. Only append and set `tasks[i].Updated = now` when `sourceID` is not already in `BlockedBy`.

**Acceptance Criteria**:
- `applyBlocks` does not create duplicate entries in `BlockedBy` when called with a sourceID already present
- Existing non-duplicate behavior is unchanged
- The `Updated` timestamp is only modified when a new dependency is actually added

**Tests**:
- Unit test: call `applyBlocks` with a sourceID already in the target task's `BlockedBy`; verify the slice length is unchanged and no duplicate exists
- Unit test: call `applyBlocks` with a new sourceID; verify it is appended and `Updated` is set
- Integration test: `tick create "A" --blocks T1` where A already blocks T1 (via prior dep add); verify T1's `blocked_by` has no duplicates
