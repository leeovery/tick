---
topic: task-removal
cycle: 2
total_proposed: 1
---
# Analysis Tasks: Task Removal (Cycle 2)

## Task 1: Replace read-only Mutate with Store.ReadTasks and restructure handleRemove into a single-open flow
status: pending
severity: high
sources: duplication, architecture

**Problem**: The non-force path in `handleRemove` (app.go:228-247) calls `store.Mutate` with `computeOnly=true` to compute the blast radius for the confirmation prompt. `Store.Mutate` always rewrites the JSONL file and rebuilds the SQLite cache regardless of whether the callback modifies the data (store.go:152-172). This means every non-forced remove rewrites the entire JSONL file unnecessarily before the user even sees the prompt. If the user declines, the file was rewritten for nothing -- and re-marshalling could introduce byte-level differences that change the file's MD5 hash and trigger unnecessary cache rebuilds. After the prompt, `handleRemove` delegates to `RunRemove` which opens the store a second time and runs `executeRemoval` again, creating a TOCTOU gap where another process could modify tasks between confirmation and execution. Additionally, `parseRemoveArgs` and the `len(ids)==0` check run twice (once in `handleRemove`, once in `RunRemove`), and the error message string `"task ID is required. Usage: tick remove <id> [<id>...]"` is duplicated verbatim. The `executeRemoval` function uses a `computeOnly bool` parameter to serve two fundamentally different purposes (read-only blast radius vs. actual removal), which is a boolean-parameter anti-pattern.

**Solution**: (1) Add a `Store.ReadTasks` method that reads and parses the JSONL file under a shared lock without writing. (2) Split `executeRemoval` into two functions: `computeBlastRadius(tasks, ids)` for read-only blast radius computation and `applyRemoval(tasks, ids)` for the actual mutation. Both share internal helpers (`collectDescendants`, validation, `stripIDsFromBlockedBy`). (3) Restructure `handleRemove` so the non-force path uses `Store.ReadTasks` + `computeBlastRadius` for the prompt, then calls `RunRemove` once for the actual mutation. This eliminates the spurious Mutate rewrite, the double store open, the double executeRemoval, and the boolean parameter.

**Outcome**: Non-forced removes no longer rewrite JSONL for the confirmation prompt. The store opens once for the read-only blast radius check (via `ReadTasks`) and once for the actual mutation (via `RunRemove`). No TOCTOU gap beyond the inherent confirmation pattern. `parseRemoveArgs` and ID validation run once in `handleRemove`, with pre-parsed IDs passed to `RunRemove`. Each function has a single responsibility. The `computeOnly` parameter is eliminated.

**Do**:
1. In `internal/storage/store.go`, add a `ReadTasks` method:
   ```go
   func (s *Store) ReadTasks() ([]task.Task, error) {
       unlock, err := s.acquireShared()
       if err != nil {
           return nil, err
       }
       defer unlock()
       _, tasks, err := s.readAndEnsureFresh()
       if err != nil {
           return nil, err
       }
       return tasks, nil
   }
   ```
   This provides read-only access to the in-memory task slice without writing.
2. In `internal/cli/remove.go`, split `executeRemoval` into two functions:
   - `computeBlastRadius(tasks []task.Task, ids []string) (blastRadius, error)` -- validates all IDs exist, expands descendants via `collectDescendants`, identifies cascaded tasks and affected deps. Returns a populated `blastRadius` struct.
   - `applyRemoval(tasks []task.Task, ids []string) ([]task.Task, RemovalResult, error)` -- validates all IDs exist, expands descendants, filters removed tasks from the slice, cleans up `BlockedBy` arrays on surviving tasks, returns the filtered slice and `RemovalResult`.
   - Both functions call shared helpers for validation and descendant expansion. Extract the common ID-validation loop and descendant expansion into a helper like `validateAndExpand(tasks, ids) (targetSet, removeSet, existingIDs map, error)` if the duplication between the two new functions warrants it.
3. Delete the `executeRemoval` function.
4. Update `RunRemove` to:
   - Accept pre-parsed IDs instead of raw args. Change signature to `RunRemove(dir string, fc FormatConfig, fmtr Formatter, ids []string, stdout io.Writer) error`. Remove the `parseRemoveArgs` call and `len(ids)==0` check from `RunRemove` (these now live only in `handleRemove`).
   - Call `store.Mutate` with `applyRemoval` (no `computeOnly` flag).
5. Update `handleRemove` in `internal/cli/app.go`:
   - Parse args once: `ids, force := parseRemoveArgs(subArgs)`. Validate `len(ids)==0` once.
   - Non-force path: open store via `openStore`, call `store.ReadTasks()` to get the task slice, call `computeBlastRadius(tasks, ids)`, close the store, run `confirmRemovalWithCascade`, then call `RunRemove(dir, fc, fmtr, ids, a.Stdout)`.
   - Force path: call `RunRemove(dir, fc, fmtr, ids, a.Stdout)` directly.
6. Update all tests that call `RunRemove` directly to pass pre-parsed ID slices instead of raw arg slices. If any tests pass `--force` in args to `RunRemove`, adjust them to pass only IDs since force handling is now in `handleRemove`.
7. Verify `go test ./internal/cli -count=1` and `go test ./internal/storage -count=1` pass.

**Acceptance Criteria**:
- `Store.ReadTasks` method exists and uses shared (not exclusive) locking
- `executeRemoval` function with `computeOnly bool` parameter no longer exists
- `computeBlastRadius` and `applyRemoval` are separate functions with distinct return types
- `RunRemove` signature matches the `Run*(dir, fc, fmtr, args/ids, stdout)` pattern with no raw-arg parsing
- `parseRemoveArgs` and the empty-ID validation run exactly once per invocation (in `handleRemove`)
- The error message `"task ID is required. Usage: tick remove <id> [<id>...]"` appears exactly once in the codebase
- Non-force path does not call `Store.Mutate` for blast radius computation
- All existing tests pass
- Force and non-force paths produce identical removal results for the same inputs

**Tests**:
- Run `go test ./internal/storage -run TestReadTasks -count=1` (new test for ReadTasks method) -- returns correct task slice without modifying JSONL file
- Run `go test ./internal/cli -run TestRemove -count=1` -- all existing remove tests pass
- Run `go test ./internal/cli -count=1` -- no regressions in other commands
- Run `go test ./... -count=1` -- full suite passes
- Verify JSONL file is not rewritten during non-force blast radius computation by checking file modification time or content hash in a test
