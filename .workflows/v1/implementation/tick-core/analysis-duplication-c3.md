AGENT: duplication
FINDINGS:
- FINDING: Post-mutation "show as output" pattern duplicated across create.go and update.go
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/create.go:173-188, /Users/leeovery/Code/tick/internal/cli/update.go:201-215
  DESCRIPTION: After a successful mutation, both create.go and update.go execute the same 10-line output block: check fc.Quiet (print ID only and return), then queryShowData -> showDataToTaskDetail -> fmtr.FormatTaskDetail -> Fprintln. The two blocks are structurally identical -- the only difference is the variable name holding the task ID (createdTask.ID vs updatedID). show.go has a similar but shorter variant (it already has the data). If the output format for mutation commands changes (e.g., adding a verbose mode or changing quiet behavior), both create.go and update.go must be updated in sync.
  RECOMMENDATION: Extract a helper like outputTaskDetail(store, id, fc, fmtr, stdout) error that encapsulates the quiet-check, queryShowData, showDataToTaskDetail, and FormatTaskDetail sequence. Both RunCreate and RunUpdate call it after mutation. show.go could also use it, reducing its post-query output to a single call.

- FINDING: Find-task-by-index pattern duplicated within dep.go
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/dep.go:73-81, /Users/leeovery/Code/tick/internal/cli/dep.go:142-149
  DESCRIPTION: RunDepAdd and RunDepRm contain identical 8-line blocks that find a task by ID in the tasks slice (taskIdx := -1, for-range with ID comparison, break, check -1 and return "not found" error). This same pattern also appears in a slightly different form in transition.go:29-39 and update.go:156-180. Within dep.go it is an exact copy. Across other files the post-find behavior varies enough that a generic helper is less clean, but within dep.go the two instances are byte-for-byte identical.
  RECOMMENDATION: Extract a findTaskIndex(tasks []task.Task, id string) (int, error) helper in helpers.go. Both RunDepAdd and RunDepRm call it. transition.go and update.go could also benefit but their post-find logic is more interleaved, making adoption optional.

- FINDING: Exclusive lock acquisition boilerplate repeated in Store.Mutate and Store.Rebuild
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/storage/store.go:93-105, /Users/leeovery/Code/tick/internal/storage/store.go:147-159
  DESCRIPTION: Mutate and Rebuild both contain the same 10-line exclusive lock acquisition block: context.WithTimeout, verbose("acquiring exclusive lock"), TryLockContext, error check, verbose("lock acquired"), defer unlock with verbose("lock released"). The only difference is the return type on error (error vs (int, error)). Query has a similar block but uses shared locking (TryRLockContext). All three are in the same file.
  RECOMMENDATION: Low priority since all instances are in a single file and the shared-vs-exclusive difference is meaningful. Could extract a withExclusiveLock(fn) and withSharedLock(fn) pattern, but the return type variations reduce the benefit. Acknowledge and leave as-is unless the file grows further.

SUMMARY: One medium-severity finding: the post-mutation output pattern (quiet check + queryShowData + format) is duplicated identically across create.go and update.go (~10 lines each). Two low-severity findings cover the find-task-by-index pattern duplicated within dep.go and lock acquisition boilerplate in store.go. All prior C1/C2 findings (cache freshness, baseFormatter, applyBlocks, parseCommaSeparatedIDs, openStore, ready-query SQL, relatedTask struct) have been successfully consolidated.
