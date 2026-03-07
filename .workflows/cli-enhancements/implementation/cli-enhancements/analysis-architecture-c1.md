AGENT: architecture
FINDINGS:
- FINDING: showData query omits type column -- show output always displays empty type
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/cli/show.go:73, /Users/leeovery/Code/tick/internal/cli/show.go:216
  DESCRIPTION: The `queryShowData` function queries `SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?` but never selects the `type` column added to the `tasks` table in this implementation. The `showData` struct also lacks a `taskType` field, and `showDataToTaskDetail` never sets `task.Task.Type`. This means `tick show` always renders Type as empty/dash, even when the task has a type set. The same `queryShowData` function is used by `outputMutationResult` (called after create, update, and note add/remove), so the post-mutation output also shows an empty type. This affects all three formatters (Pretty, Toon, JSON) since they all render type from `TaskDetail.Task.Type`.
  RECOMMENDATION: Add `type` to the `showData` struct (as `taskType string`), add it to the SQL SELECT in `queryShowData`, scan it (as `*string` like `descPtr`/`parentPtr`), and populate it in `showDataToTaskDetail` when constructing the `task.Task`.
- FINDING: ResolveID acquires two separate shared locks for exact-match then prefix-search
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/storage/store.go:272-329
  DESCRIPTION: `ResolveID` calls `s.Query()` up to twice -- once for the exact full-ID match (when hex length is 6) and once for the prefix search fallback. Each `Query()` call acquires a shared lock, reads the JSONL file, checks cache freshness, then releases the lock. The second call re-reads the same file and re-checks the same hash. While functionally correct (shared locks are non-exclusive), this doubles the I/O and lock overhead for every full-ID resolution. Since `ResolveID` is called for every command that accepts an ID (which is nearly all of them), this adds unnecessary latency on every invocation.
  RECOMMENDATION: Factor the body of `ResolveID` into a single `Query()` call that performs both the exact match attempt and the prefix search within one lock acquisition. Something like: `s.Query(func(db *sql.DB) error { ... try exact; if not found, try prefix ... })`.
SUMMARY: One high-severity bug where `type` is never queried in `show`/post-mutation output, and one medium-severity performance issue where `ResolveID` acquires the shared lock twice per call.
