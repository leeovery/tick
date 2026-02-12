---
topic: tick-core
cycle: 3
total_proposed: 2
---
# Analysis Tasks: Tick Core (Cycle 3)

## Task 1: Prevent duplicate blocked_by entries in applyBlocks
status: approved
severity: medium
sources: standards

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

## Task 2: Extract post-mutation output helper from create.go and update.go
status: approved
severity: medium
sources: duplication

**Problem**: After a successful mutation, both `create.go` (lines 173-188) and `update.go` (lines 201-215) execute the same output block: check `fc.Quiet` (print ID only and return), then `queryShowData` -> `showDataToTaskDetail` -> `fmtr.FormatTaskDetail` -> `Fprintln`. The two blocks are structurally identical -- the only difference is the variable name holding the task ID. If the output format for mutation commands changes (e.g., adding a verbose mode or changing quiet behavior), both files must be updated in sync.

**Solution**: Extract a helper function like `outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error` in `helpers.go` that encapsulates the quiet-check, queryShowData, showDataToTaskDetail, and FormatTaskDetail sequence. Both `RunCreate` and `RunUpdate` call it after their mutation succeeds.

**Outcome**: Post-mutation output logic lives in one place. Changes to mutation output format require editing only the helper.

**Do**:
1. In `/Users/leeovery/Code/tick/internal/cli/helpers.go`, add a new function:
   ```go
   func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
       if fc.Quiet {
           fmt.Fprintln(stdout, id)
           return nil
       }
       data, err := queryShowData(store, id)
       if err != nil {
           return err
       }
       detail := showDataToTaskDetail(data)
       fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
       return nil
   }
   ```
2. In `/Users/leeovery/Code/tick/internal/cli/create.go`, replace lines 173-188 (the quiet check through FormatTaskDetail) with `return outputMutationResult(store, createdTask.ID, fc, fmtr, stdout)`.
3. In `/Users/leeovery/Code/tick/internal/cli/update.go`, replace lines 201-215 with `return outputMutationResult(store, updatedID, fc, fmtr, stdout)`.
4. Add necessary imports to `helpers.go` (`fmt`, `io`, and the storage package if not already imported).

**Acceptance Criteria**:
- `create.go` and `update.go` no longer contain the post-mutation output logic inline
- Both commands produce identical output to before (quiet mode prints ID, normal mode prints full task detail)
- The helper is the single source of truth for mutation command output

**Tests**:
- Existing tests for `RunCreate` and `RunUpdate` continue to pass with no changes (output is identical)
- Existing tests for quiet mode (`--quiet` flag) continue to pass with no changes
