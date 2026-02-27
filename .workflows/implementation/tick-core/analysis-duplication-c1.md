AGENT: duplication
FINDINGS:
- FINDING: Cache corruption recovery logic duplicated between Store.ensureFresh and standalone EnsureFresh
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/storage/store.go:190-233, /Users/leeovery/Code/tick/internal/storage/cache.go:177-209
  DESCRIPTION: Store.ensureFresh (store.go) and the standalone EnsureFresh function (cache.go) implement the same pattern: open cache (recover on error by deleting and reopening), check IsFresh (recover on error by closing/deleting/reopening), rebuild if stale. The two implementations are ~40 lines each with nearly identical structure and error handling. Both handle the same two error recovery paths (open failure, freshness check failure) with the same strategy (delete file, reopen). The standalone EnsureFresh in cache.go does not appear to be used by Store at all, making it either dead code or a parallel implementation that drifted from the Store-integrated version.
  RECOMMENDATION: Remove the standalone EnsureFresh function from cache.go if unused. If it is needed for non-Store contexts, extract the shared cache-open-with-recovery and freshness-check-with-recovery logic into methods on Cache (e.g., OpenWithRecovery, CheckFreshnessWithRecovery) that both call sites can use.

- FINDING: FormatTransition and FormatDepChange identical across ToonFormatter and PrettyFormatter
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:90-100, /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:122-132
  DESCRIPTION: ToonFormatter.FormatTransition and PrettyFormatter.FormatTransition have identical implementations (both return fmt.Sprintf("%s: %s -> %s", ...)). ToonFormatter.FormatDepChange and PrettyFormatter.FormatDepChange are also identical (same if/else with the same format strings). Four methods total are exact copies.
  RECOMMENDATION: Extract a shared base implementation via an embedded struct (e.g., baseFormatter) that provides FormatTransition and FormatDepChange. Both ToonFormatter and PrettyFormatter embed it. This consolidates 4 methods into 2.

- FINDING: --blocks handling duplicated between create and update
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/create.go:165-174, /Users/leeovery/Code/tick/internal/cli/update.go:196-205
  DESCRIPTION: Both create.go and update.go contain the same loop to apply --blocks: iterate all tasks, match by blockID, append the new task's ID to BlockedBy, set Updated timestamp. The logic is structurally identical (nested for-range over tasks and opts.blocks, same append and timestamp update). If the semantics of --blocks ever change, both must be updated in sync.
  RECOMMENDATION: Extract a shared helper function like applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time) that both create and update call within their Mutate callbacks.

- FINDING: Comma-separated ID parsing + normalize duplicated in create and update arg parsers
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/create.go:54-60, /Users/leeovery/Code/tick/internal/cli/create.go:62-72, /Users/leeovery/Code/tick/internal/cli/update.go:74-84
  DESCRIPTION: The pattern strings.Split(args[i], ",") -> range -> NormalizeID(TrimSpace(id)) -> append if non-empty appears three times across two files (for --blocked-by and --blocks in create.go, for --blocks in update.go). Each is 6-7 lines of identical logic.
  RECOMMENDATION: Extract a helper function like parseCommaSeparatedIDs(s string) []string that splits, trims, normalizes, and filters empty values. All three call sites can use it.

- FINDING: Task-find-by-ID pattern repeated in Mutate callbacks
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/dep.go:79-88, /Users/leeovery/Code/tick/internal/cli/dep.go:92-97, /Users/leeovery/Code/tick/internal/cli/dep.go:154-161, /Users/leeovery/Code/tick/internal/cli/transition.go:35-46, /Users/leeovery/Code/tick/internal/cli/update.go:168-189
  DESCRIPTION: Multiple Mutate callbacks contain the same pattern of iterating tasks to find a task by ID (for-range, compare .ID, break/return if found, error "not found" if not). This appears 5 times across 3 files. Each instance is ~8-10 lines. However, the found-task handling differs per call site (some need the index, some need a bool, some return immediately), making a single generic helper less straightforward.
  RECOMMENDATION: Consider a findTaskIndex(tasks []task.Task, id string) (int, error) helper that returns the index or a "not found" error. Most call sites need the index. This would reduce each find pattern to a single call + error check. Low priority since each instance is small and the variations are minor.

SUMMARY: One high-severity finding: the cache freshness/corruption recovery logic is duplicated between Store.ensureFresh and the standalone EnsureFresh function (~40 lines each). Two medium-severity findings: formatter methods are identical across ToonFormatter and PrettyFormatter, and the --blocks application logic is duplicated between create and update. Two low-severity findings cover repeated ID-parsing helpers and task-lookup patterns.
