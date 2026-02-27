AGENT: architecture
FINDINGS:
- FINDING: Duplicate freshness/corruption-recovery logic between Store.ensureFresh and package-level EnsureFresh
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/storage/cache.go:177, /Users/leeovery/Code/tick/internal/storage/store.go:190
  DESCRIPTION: The package-level function `EnsureFresh` in cache.go (lines 177-209) duplicates the same open-check-rebuild-with-corruption-recovery logic that lives in `Store.ensureFresh` (store.go lines 190-233). Both implement the identical pattern: open cache, check freshness, handle corruption by delete-and-recreate, rebuild if stale. The package-level function is only used in tests (`cache_test.go`). This creates two independent code paths for the same concern that could diverge.
  RECOMMENDATION: Either remove the package-level `EnsureFresh` entirely (test through Store instead) or extract the shared corruption-recovery-and-rebuild logic into a single private helper that both call. Since `Store.ensureFresh` is the real runtime path, prefer removing the standalone function and testing freshness through the Store API.

- FINDING: RunRebuild bypasses Store and directly uses low-level storage primitives
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/cli/rebuild.go:18
  DESCRIPTION: `RunRebuild` manually acquires a file lock, reads JSONL, opens cache, and rebuilds -- re-implementing the locking, file-reading, and cache-management responsibilities that `Store` encapsulates. This creates a parallel code path that does not share the same error recovery, verbose logging integration (it calls `fc.Logger.Log` directly on a potentially nil logger without nil-check -- see line 33), or corruption handling as `Store`. If `Store`'s locking or freshness logic changes, `RunRebuild` will not benefit. More critically, `fc.Logger.Log("acquiring exclusive lock")` on line 33 will panic if `--verbose` is not set because `fc.Logger` is nil and `Log` is only nil-safe on the receiver, but the `FormatConfig.Logger` field itself is nil -- however, inspecting the `VerboseLogger.Log` method, it IS nil-receiver-safe, so this is actually safe due to Go's nil pointer method dispatch. Still, the architectural bypass of Store is the real issue.
  RECOMMENDATION: Add a `Rebuild` method to `Store` (or a standalone function that accepts a `Store`) that encapsulates forced-rebuild semantics: exclusive lock, delete cache, read JSONL, rebuild cache. This keeps all storage operations behind a single API surface. `RunRebuild` would then call `store.Rebuild()` similar to how other commands call `store.Mutate()` or `store.Query()`.

- FINDING: Store.Query exposes raw *sql.DB to callers, coupling CLI to SQLite internals
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/storage/store.go:145, /Users/leeovery/Code/tick/internal/cli/list.go:110, /Users/leeovery/Code/tick/internal/cli/show.go:76, /Users/leeovery/Code/tick/internal/cli/stats.go:31
  DESCRIPTION: `Store.Query(fn func(db *sql.DB) error)` passes the raw `*sql.DB` handle to CLI code. Every CLI command (list, show, stats) writes inline SQL queries against the `tasks` and `dependencies` tables. This means the CLI package has detailed knowledge of the SQLite schema -- column names, table structure, index availability. If the schema changes (e.g., renaming a column, adding a view), every CLI command file must be updated. The storage layer provides no query abstraction -- it handles only write orchestration and cache freshness, while reads are fully delegated to raw SQL in the CLI.
  RECOMMENDATION: For v1 scope this is acceptable as pragmatic -- the schema is small and stable. However, consider adding a few typed query methods to Store (e.g., `ListTasks(filter)`, `GetTask(id)`, `GetStats()`) as the query count grows. This would confine SQL to the storage package and let the CLI work with domain types. Flag this for v2 if the current approach causes friction.

- FINDING: Parallel relatedTask types in show.go and format.go
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/show.go:30, /Users/leeovery/Code/tick/internal/cli/format.go:88
  DESCRIPTION: There are two structurally identical types for representing related tasks: `relatedTask` (unexported, in show.go, line 30) and `RelatedTask` (exported, in format.go, line 88). The `showDataToTaskDetail` function (show.go:155) manually converts between them field by field. Both have identical fields (id/ID, title/Title, status/Status). The conversion is boilerplate that exists only because the show command defines its own parallel type rather than using the formatter's type directly.
  RECOMMENDATION: Remove the unexported `relatedTask` type and use `RelatedTask` directly in `showData` and the SQL scanning logic. This eliminates the conversion function and one redundant type.

- FINDING: No end-to-end workflow integration test
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/
  DESCRIPTION: Tests cover individual commands well (create, list, show, transition, dep, stats, rebuild) and format integration across all three formatters. However, there is no test that exercises the full agent workflow described in the spec: init -> create tasks with dependencies/hierarchy -> ready (verify correct tasks) -> start -> done -> ready (verify unblocking). The closest are `parent_scope_test.go` and individual command tests, but none chain multiple mutations and verify the emergent behavior of the ready/blocked query logic across a realistic multi-step workflow.
  RECOMMENDATION: Add one integration test that exercises the primary workflow: create a parent task, create child tasks with inter-dependencies, verify `ready` returns only the correct leaf tasks, transition a blocker to done, verify the dependent task appears in `ready`, and verify `stats` reflects the changes. This catches seam issues between mutation and query paths.

SUMMARY: The architecture is generally sound with clean separation between task model, storage, and CLI layers. The main structural issue is that RunRebuild bypasses the Store abstraction entirely, creating a parallel code path for lock management and cache operations. The raw *sql.DB exposure in Store.Query is pragmatically acceptable for v1 but worth flagging as a future concern. The duplicate EnsureFresh logic and parallel relatedTask types are minor cleanup items. Adding one end-to-end workflow test would strengthen confidence in cross-command integration.
