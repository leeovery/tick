# Task tick-core-5-2: tick rebuild Command

## Task Summary

Implement `tick rebuild` -- a command that forces a complete SQLite cache rebuild from JSONL, bypassing the freshness check. This is a diagnostic tool for corrupted cache, debugging, or after manual JSONL edits.

Requirements:
- CLI handler acquires exclusive file lock (same as write path)
- Delete existing `cache.db` if present
- Full rebuild: read JSONL, parse all records, create SQLite schema, insert all tasks
- Update hash in metadata table to current JSONL hash
- Release lock
- Output via `Formatter.FormatMessage()` -- confirmation showing count of tasks rebuilt
- `--quiet`: suppress output entirely
- `--verbose`: log each step (delete, read, insert count, hash update) to stderr

### Acceptance Criteria

1. Rebuilds SQLite from JSONL regardless of current freshness
2. Deletes existing cache before rebuild
3. Updates hash in metadata table
4. Acquires exclusive lock during rebuild
5. Handles missing cache.db without error
6. Handles empty JSONL (0 tasks rebuilt)
7. Outputs confirmation with task count
8. `--quiet` suppresses output
9. `--verbose` logs rebuild steps to stderr

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Rebuilds SQLite from JSONL regardless of freshness | PASS -- `ForceRebuild()` bypasses freshness check entirely | PASS -- `Rebuild()` bypasses freshness check entirely |
| Deletes existing cache before rebuild | PASS -- `os.Remove(s.cachePath)` at store.go:195, ignores error silently | PASS -- `os.Remove(s.dbPath)` at store.go:182, checks error properly with `!os.IsNotExist(err)` |
| Updates hash in metadata table | PASS -- delegates to `cache.Rebuild(tasks, rawContent)` which handles hash | PASS -- delegates to `c.Rebuild(tasks, jsonlData)` which handles hash |
| Acquires exclusive lock during rebuild | PASS -- `fl.TryLockContext` with timeout at store.go:166-171 | PASS -- `fl.TryLockContext` with timeout at store.go:163-173 |
| Handles missing cache.db without error | PASS -- `os.Remove` ignoring error means no failure on missing file | PASS -- explicit `!os.IsNotExist(err)` guard on `os.Remove` |
| Handles empty JSONL (0 tasks rebuilt) | PASS -- test at rebuild_test.go:306 confirms 0 tasks | PASS -- test at rebuild_test.go:289 confirms 0 tasks |
| Outputs confirmation with task count | PASS -- `"Rebuilt cache: %d tasks"` at rebuild.go:29 | PASS -- `"Rebuilt cache: %d tasks"` at rebuild.go:28 |
| `--quiet` suppresses output | PASS -- `a.config.Quiet` check at rebuild.go:25 | PASS -- `a.Quiet` check at rebuild.go:24 |
| `--verbose` logs rebuild steps to stderr | PASS -- 7 `logVerbose` calls in `ForceRebuild` covering lock/read/delete/insert/hash | PASS -- 7 `vlog` calls in `Rebuild` covering lock/delete/read/insert/hash |

## Implementation Comparison

### Approach

Both versions follow a nearly identical architectural pattern: a thin CLI handler in `rebuild.go` that delegates to a store-level rebuild method, which performs locking, cache deletion, JSONL parsing, cache creation, and hash update.

**CLI Layer (`rebuild.go`)**

V2 (31 LOC):
```go
func (a *App) runRebuild() error {
    tickDir, err := DiscoverTickDir(a.workDir)
    // ...
    store, err := a.newStore(tickDir)
    // ...
    count, err := store.ForceRebuild()
    // ...
    if a.config.Quiet {
        return nil
    }
    msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
    return a.formatter.FormatMessage(a.stdout, msg)
}
```

V4 (30 LOC):
```go
func (a *App) runRebuild(args []string) error {
    tickDir, err := DiscoverTickDir(a.Dir)
    // ...
    s, err := a.openStore(tickDir)
    // ...
    count, err := s.Rebuild()
    // ...
    if a.Quiet {
        return nil
    }
    msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
    return a.Formatter.FormatMessage(a.Stdout, msg)
}
```

The CLI handlers are functionally identical. Minor differences:
- V2's `runRebuild()` takes no args; V4's `runRebuild(args []string)` accepts args but ignores them. V4's pattern is more consistent with other commands in its codebase.
- V2 accesses config via `a.config.Quiet` (private nested struct); V4 accesses via `a.Quiet` (exported field directly on App). This reflects existing codebase conventions, not a task-specific choice.
- V2 names the method `ForceRebuild()`; V4 names it `Rebuild()`. V4's name is simpler and sufficient since there's no non-force rebuild variant exposed at the store level.

**Store Layer**

V2 (`internal/storage/store.go`, lines 162-211, 50 LOC of new code):
```go
func (s *Store) ForceRebuild() (int, error) {
    // 1. Acquire lock
    // 2. Read JSONL
    // 3. Parse tasks
    // 4. Delete cache.db (os.Remove, ignore error)
    // 5. Create new cache via sqlite.NewCache
    // 6. cache.Rebuild(tasks, rawContent)
    // 7. Return len(tasks)
}
```

V4 (`internal/store/store.go`, lines 160-213, 54 LOC of new code):
```go
func (s *Store) Rebuild() (int, error) {
    // 1. Acquire lock
    // 2. Delete cache.db (os.Remove, check for non-ENOENT errors)
    // 3. Read JSONL
    // 4. Parse tasks
    // 5. Create new cache via cache.Open
    // 6. c.Rebuild(tasks, jsonlData)
    // 7. Return len(tasks)
}
```

**Key ordering difference**: V2 reads JSONL *before* deleting the cache; V4 deletes the cache *before* reading JSONL. V4's order matches the task spec more closely ("Delete existing cache.db if present" then "Full rebuild: read JSONL..."). However, V2's order is arguably safer: if JSONL reading fails, the old cache is preserved. V4's approach means a failed JSONL read leaves no cache at all. This is a genuinely meaningful difference.

**Error handling on cache deletion**:
- V2: `os.Remove(s.cachePath)` -- ignores *all* errors, not just ENOENT
- V4: `if err := os.Remove(s.dbPath); err != nil && !os.IsNotExist(err) { return 0, ... }` -- only ignores "file not found", surfaces real errors (e.g., permission denied)

V4's approach is genuinely better here.

**Lock retry interval**:
- V2: `10*time.Millisecond`
- V4: `100*time.Millisecond`

V2 polls 10x more frequently. Neither is categorically better; V4 is less aggressive on CPU.

### Code Quality

**Naming**: V2 uses `ForceRebuild` while V4 uses `Rebuild`. Since the store has no "soft rebuild" method, V4's simpler name avoids unnecessary prefix. V4 is slightly better.

**Error handling**: V4's explicit ENOENT check on `os.Remove` (store.go:182) is more correct than V2's silent discard of all errors (store.go:195). V2 could mask permission errors.

**Verbose logging**: Both versions log the same 7 steps with similar messages. V2 uses `s.logVerbose("rebuild: deleting existing cache.db")` (fixed strings); V4 uses `s.vlog("deleting existing cache.db at %s", s.dbPath)` (format strings with path context). V4's verbose messages include actual paths, which is more useful for debugging:

V2:
```go
s.logVerbose("rebuild: reading tasks.jsonl")
s.logVerbose("rebuild: deleting existing cache.db")
```

V4:
```go
s.vlog("reading JSONL from %s", s.jsonlPath)
s.vlog("deleting existing cache.db at %s", s.dbPath)
```

V4's format-string approach provides richer diagnostic output.

**Logger wiring**: V2 uses an interface-based `Logger` with `SetLogger(l Logger)`. V4 uses a function field `LogFunc func(format string, args ...interface{})`. Both are valid Go patterns; V4's function-field approach is simpler and more idiomatic for this use case (single method).

**Lock error handling**: V2 combines `err != nil || !locked` into one check and one error message. V4 separates them into two distinct checks:
```go
// V4
if err != nil {
    return 0, fmt.Errorf("could not acquire lock on %s - ...", s.lockPath)
}
if !locked {
    return 0, fmt.Errorf("could not acquire lock on %s - ...", s.lockPath)
}
```
V4's separation is cleaner structurally but returns the same message either way, so the practical benefit is nil.

**Cache constructor naming**: V2 calls `sqlite.NewCache(s.cachePath)` (store.go:199); V4 calls `cache.Open(s.dbPath)` (store.go:201). This reflects different package structures across the codebases, not task-specific choices.

### Test Quality

**V2 Test Functions** (1 top-level function, 8 subtests inside `TestRebuild`):
1. `TestRebuild/"it rebuilds cache from JSONL"` -- Creates 3 tasks (including one with a dependency), runs rebuild, verifies 3 tasks + 1 dependency row in SQLite
2. `TestRebuild/"it handles missing cache.db (fresh build)"` -- Verifies no cache.db before, runs rebuild, verifies cache.db created with 1 task
3. `TestRebuild/"it overwrites valid existing cache"` -- Creates cache via `list`, modifies JSONL to add a task, runs rebuild, verifies new count (3)
4. `TestRebuild/"it updates hash in metadata table after rebuild"` -- Runs rebuild, queries `metadata` table for `jsonl_hash`, asserts non-empty
5. `TestRebuild/"it acquires exclusive lock during rebuild"` -- Runs with `--verbose`, checks stderr for "exclusive lock" string
6. `TestRebuild/"it outputs confirmation message with task count"` -- 3 tasks, checks stdout contains "3" and "rebuilt"/"rebuild"
7. `TestRebuild/"it suppresses output with --quiet"` -- Checks stdout is empty
8. `TestRebuild/"it handles empty JSONL (0 tasks rebuilt)"` -- Empty JSONL, checks stdout contains "0" and cache has 0 tasks with correct schema

**V4 Test Functions** (8 top-level functions, each with 1 subtest):
1. `TestRebuild_RebuildsCacheFromJSONL/"it rebuilds cache from JSONL"` -- 3 tasks, verifies count = 3
2. `TestRebuild_HandlesMissingCacheDB/"it handles missing cache.db (fresh build)"` -- Explicitly removes cache.db, runs rebuild, verifies creation
3. `TestRebuild_OverwritesValidExistingCache/"it overwrites valid existing cache"` -- Creates cache via `list`, adds task via JSONL write, rebuilds, verifies count = 2
4. `TestRebuild_UpdatesHashInMetadataTable/"it updates hash in metadata table after rebuild"` -- Verifies `jsonl_hash` in metadata is non-empty
5. `TestRebuild_AcquiresExclusiveLock/"it acquires exclusive lock during rebuild"` -- **Holds an external flock, then runs rebuild, expects exit code 1 and lock error message**
6. `TestRebuild_OutputsConfirmationWithTaskCount/"it outputs confirmation message with task count"` -- 3 tasks, checks output for "3" and "rebuilt"
7. `TestRebuild_SuppressesOutputWithQuiet/"it suppresses output with --quiet"` -- Checks empty stdout
8. `TestRebuild_LogsRebuildStepsWithVerbose/"it logs rebuild steps with --verbose"` -- Checks stderr for "verbose:" prefix, delete, read, insert/rebuild, hash steps

**Critical test difference -- lock testing**:

V2's lock test (line 200-220) merely runs with `--verbose` and checks that stderr mentions "exclusive lock". This is a **weak test** -- it only verifies logging output, not actual lock behavior.

V4's lock test (line 162-191) is **genuinely superior**: it holds an external `flock.New(lockPath).TryLock()` before running rebuild, then asserts the command fails with exit code 1 and a lock error message. This actually verifies the lock exclusion behavior.

**Test setup differences**:

V2 uses raw JSONL string construction via `taskJSONL()` helper:
```go
content := strings.Join([]string{
    taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
}, "\n") + "\n"
dir := setupTickDirWithContent(t, content)
```

V4 uses structured `task.Task` objects:
```go
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
}
dir := setupInitializedDirWithTasks(t, tasks)
```

V4's approach is more type-safe and less prone to typos in raw JSON construction. V4 uses `task.StatusOpen`/`task.StatusDone` constants instead of raw strings like `"open"`/`"done"`.

**Dependency coverage**: V2's first test includes a task with `blockedBy: []string{"tick-aaa111"}` and verifies the `dependencies` table has 1 row. V4 does not test dependencies at all. This is a minor V2 advantage for thoroughness.

**Verbose test thoroughness**: V4's verbose test (line 245-286) checks for 4 distinct step categories (delete, read, insert/rebuild, hash) plus the "verbose:" prefix. V2's verbose test (line 270-303) checks for 3 categories (delete, read, hash) but not insert count explicitly. V4 is slightly more thorough.

**App construction pattern**: V2 uses `NewApp()` constructor + field assignment; V4 uses struct literal `&App{...}`. V4's pattern is more explicit and standard for tests.

**Exit code testing**: V4 tests return `int` exit codes from `app.Run()` and check `code != 0`; V2 tests return `error` from `app.Run()` and check `err != nil`. Both are valid; they reflect different `App.Run()` signatures across the codebases.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 6 | 6 |
| Lines added | 432 | 420 |
| Impl LOC (rebuild.go) | 31 | 30 |
| Impl LOC (store addition) | 54 | 59 |
| Test LOC | 341 | 321 |
| Test functions | 1 (8 subtests) | 8 (8 subtests total) |

## Verdict

**V4 is the better implementation**, though the margin is narrow.

The decisive factors:

1. **Lock test quality** (most significant): V4 actually tests lock exclusion by holding an external flock and verifying the command fails. V2 only checks that verbose output mentions "exclusive lock" -- this would pass even if locking were entirely broken. This is a meaningful correctness gap.

2. **Error handling on cache deletion**: V4's `!os.IsNotExist(err)` guard is correct; V2 silently discards all errors from `os.Remove`, which could mask permission errors in production.

3. **Verbose log messages**: V4 includes actual file paths in verbose output (`"deleting existing cache.db at %s"`), providing more useful diagnostic information.

4. **Type-safe test setup**: V4 constructs test data using `task.Task` structs with `task.StatusOpen` constants rather than raw JSON strings, reducing fragility.

V2's single advantage is testing dependency preservation during rebuild (the `blockedBy` + `dependencies` table assertion), which V4 omits. However, this is a minor edge case that's likely covered by the cache/rebuild implementation tests elsewhere, whereas V4's lock test gap in V2 represents a genuine missing verification of a core acceptance criterion.
