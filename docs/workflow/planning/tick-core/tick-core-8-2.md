---
id: tick-core-8-2
phase: 8
status: completed
created: 2026-02-10
---

# Extract post-mutation output helper from create.go and update.go

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
