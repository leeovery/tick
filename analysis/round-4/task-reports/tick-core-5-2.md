# Task tick-core-5-2: tick rebuild command

## Task Summary

Implement `tick rebuild` -- a CLI command that forces a complete SQLite cache rebuild from JSONL, bypassing the freshness check. This is a diagnostic tool for corrupted cache, debugging, or after manual JSONL edits.

**Required behavior:**
- CLI handler acquires exclusive file lock (same as write path)
- Delete existing `cache.db` if present
- Full rebuild: read JSONL, parse all records, create SQLite schema, insert all tasks
- Update hash in metadata table to current JSONL hash
- Release lock
- Output via `Formatter.FormatMessage()` -- confirmation showing count of tasks rebuilt
- `--quiet`: suppress output entirely
- `--verbose`: log each step (delete, read, insert count, hash update) to stderr

**Specified tests:**
1. "it rebuilds cache from JSONL"
2. "it handles missing cache.db (fresh build)"
3. "it overwrites valid existing cache"
4. "it updates hash in metadata table after rebuild"
5. "it acquires exclusive lock during rebuild"
6. "it outputs confirmation message with task count"
7. "it suppresses output with --quiet"
8. "it logs rebuild steps with --verbose"

**Acceptance Criteria:**
1. Rebuilds SQLite from JSONL regardless of current freshness
2. Deletes existing cache before rebuild
3. Updates hash in metadata table
4. Acquires exclusive lock during rebuild
5. Handles missing cache.db without error
6. Handles empty JSONL (0 tasks rebuilt)
7. Outputs confirmation with task count
8. --quiet suppresses output
9. --verbose logs rebuild steps to stderr

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Rebuilds SQLite from JSONL regardless of freshness | PASS -- `Store.Rebuild()` bypasses `ensureFresh()` entirely, reads JSONL and calls `cache.Rebuild()` directly | PASS -- `RunRebuild()` reads JSONL and calls `cache.Rebuild()` directly, no freshness check involved |
| Deletes existing cache before rebuild | PASS -- `os.Remove(s.cachePath)` with `!os.IsNotExist(err)` guard at store.go:180 | PASS -- `os.Remove(cachePath)` at rebuild.go:49, ignores error silently |
| Updates hash in metadata table | PASS -- delegated to `cache.Rebuild(tasks, jsonlData)` which stores SHA256 hash, confirmed by verbose log "hash updated" | PASS -- delegated to `cache.Rebuild(tasks, rawJSONL)` which stores SHA256 hash, confirmed by verbose log "hash updated" |
| Acquires exclusive lock during rebuild | PASS -- `s.acquireExclusive()` called at store.go:167, uses `flock.TryLockContext` with timeout | PASS -- `fileLock.TryLockContext()` called at rebuild.go:35, uses same flock mechanism with 5s timeout |
| Handles missing cache.db without error | PASS -- `os.Remove` returns `os.IsNotExist` which is explicitly ignored at store.go:180; `cache.New()` then creates fresh DB | PASS -- `os.Remove` return value discarded entirely at rebuild.go:49; `OpenCache()` creates fresh DB |
| Handles empty JSONL (0 tasks rebuilt) | PASS -- tested in "it handles empty JSONL (0 tasks rebuilt)" | PASS -- tested in "it handles empty JSONL with 0 tasks rebuilt" |
| Outputs confirmation with task count | PASS -- `ctx.Fmt.FormatMessage(ctx.Stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))` at rebuild.go:29 | PASS -- `fmt.Fprintln(stdout, fmtr.FormatMessage(msg))` at rebuild.go:78 |
| --quiet suppresses output | PASS -- gated by `!ctx.Quiet` at rebuild.go:28 | PASS -- gated by `!fc.Quiet` at rebuild.go:75 |
| --verbose logs rebuild steps to stderr | PASS -- via `VerboseLogger.Log()` calls in `Store.Rebuild()`: "deleting existing cache", "reading tasks.jsonl", "rebuilding cache (N tasks)", "hash updated" | PASS -- via `fc.Logger.Log()` calls in `RunRebuild()`: "acquiring exclusive lock", "lock acquired", "deleting cache.db", "reading JSONL", "rebuilding cache with N tasks", "hash updated", "lock released" |

## Implementation Comparison

### Approach

**V5: Store-level abstraction (engine layer)**

V5 adds a `Store.Rebuild() (int, error)` method directly on the `engine.Store` type (store.go lines 162-210). The CLI handler (`rebuild.go`, 33 lines) is a thin wrapper that:
1. Discovers the tick directory
2. Creates a Store via `engine.NewStore(tickDir, ctx.storeOpts()...)`
3. Calls `store.Rebuild()`
4. Formats the output

The rebuild logic lives in the engine layer where it belongs architecturally -- close to the cache and lock management code. The `Store.Rebuild()` method reuses the same `acquireExclusive()` helper used by `Mutate()`, ensuring consistent lock behavior. Key code:

```go
func (s *Store) Rebuild() (int, error) {
    unlock, err := s.acquireExclusive()
    if err != nil {
        return 0, err
    }
    defer unlock()

    s.verbose.Log("deleting existing cache")
    if err := s.cache.Close(); err != nil {
        return 0, fmt.Errorf("closing cache: %w", err)
    }

    if err := os.Remove(s.cachePath); err != nil && !os.IsNotExist(err) {
        return 0, fmt.Errorf("deleting cache: %w", err)
    }
    // ... read, parse, create fresh cache, rebuild ...
}
```

V5 also adds a `cachePath` field to the Store struct (previously only stored the cache object, not the path). This was necessary because the rebuild needs to delete and recreate the file.

**V6: CLI-level inline implementation**

V6's commit implements the entire rebuild logic inline in the CLI handler (`rebuild.go`, 81 lines). It directly:
1. Discovers tick directory
2. Manually creates a `flock.Flock` and acquires it
3. Deletes `cache.db` via `os.Remove`
4. Reads and parses JSONL using `storage.ParseJSONL`
5. Opens a fresh cache via `storage.OpenCache`
6. Calls `cache.Rebuild()`
7. Formats output

There is no modification to the store layer. The Store.Rebuild() method was added later in a separate commit (dce7d58). Key code:

```go
func RunRebuild(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
    tickDir, err := DiscoverTickDir(dir)
    if err != nil {
        return err
    }

    jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
    cachePath := filepath.Join(tickDir, "cache.db")
    lockPath := filepath.Join(tickDir, "lock")

    fileLock := flock.New(lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    fc.Logger.Log("acquiring exclusive lock")
    locked, err := fileLock.TryLockContext(ctx, 50*time.Millisecond)
    // ... manual lock management, file ops, cache creation ...
}
```

**Assessment:** V5's approach is architecturally superior. Placing `Rebuild()` on the Store type keeps all lock/cache/file logic in one layer, prevents the CLI from directly touching storage internals (lock files, cache paths), and makes the method reusable by other callers (e.g., tests, future programmatic API). V6 recognized this later and moved the logic behind the Store in a separate commit (tick-core-6-2), but at the time of this task's implementation, it violated separation of concerns.

### Code Quality

**V5 rebuild.go (33 lines)**

- Clean, minimal CLI handler. All storage logic delegated to `Store.Rebuild()`.
- Proper error propagation with `%w` wrapping at the Store level.
- Uses existing `ctx.storeOpts()` pattern for verbose logging injection.
- Formatter invocation: `ctx.Fmt.FormatMessage(ctx.Stdout, ...)` -- follows the same pattern as other V5 commands.
- `defer store.Close()` handles cleanup.

**V6 rebuild.go (81 lines)**

- All storage logic inlined: lock acquisition, file deletion, JSONL reading, cache creation.
- Hardcoded lock timeout (`5 * time.Second`) rather than using the store's configurable `lockTimeout`.
- Hardcoded lock retry interval (`50 * time.Millisecond`).
- Direct import of `github.com/gofrs/flock` in the CLI layer -- a storage implementation detail leaking into the presentation layer.
- `fc.Logger.Log("acquiring exclusive lock")` -- calls Logger.Log on a potentially nil `*VerboseLogger`. This works because V6's VerboseLogger has a nil-safe `Log()` method, but it's a subtle dependency.
- Error from `os.Remove(cachePath)` is silently discarded (line 49). V5 explicitly checks for `!os.IsNotExist(err)` and returns unexpected errors. V6's approach could mask permission errors.
- `defer cache.Close()` manages cache lifecycle, but the cache is created mid-function, meaning an error between lock acquisition and cache creation leaves no cache to close (handled correctly by Go's nil defer behavior, but less explicit).

**V5 store.go changes (55 new lines)**

- Adds `cachePath string` field to Store struct -- necessary and well-placed.
- `Rebuild()` method: closes existing cache connection before file deletion (preventing SQLite file handle leaks), checks `os.Remove` error properly, creates fresh cache with `cache.New()`, stores it back on the struct.
- Verbose logging uses `s.verbose.Log()` and `s.verbose.Logf()` -- consistent with existing Store methods.
- Error wrapping: `fmt.Errorf("closing cache: %w", err)`, `fmt.Errorf("deleting cache: %w", err)`, etc. -- all explicitly wrapped.

**V6 app.go changes (11 new lines)**

- Standard boilerplate: `handleRebuild()` method with `a.Getwd()` and delegation to `RunRebuild()`.
- Consistent with V6's existing command handler pattern.
- `case "rebuild":` added to the switch statement in `Run()`.

**V5 cli.go change (1 line)**

- `"rebuild": runRebuild` added to the `commands` map. Consistent with V5's map-based dispatch pattern.

### Test Quality

Both versions implement all 8 specified tests. Here is a detailed comparison:

**V5 Test Functions (248 lines, 8 subtests):**

| Test | Approach | Assertions |
|------|----------|------------|
| `"it rebuilds cache from JSONL"` | Creates 2 tasks via `initTickProjectWithTasks`, runs rebuild, opens cache.db directly with `sql.Open` | Checks exit code, cache.db existence via `os.Stat`, task count via `SELECT COUNT(*)` |
| `"it handles missing cache.db (fresh build)"` | Creates 1 task, removes cache.db, runs rebuild | Checks exit code, cache.db re-created via `os.Stat` |
| `"it overwrites valid existing cache"` | Creates 1 task, runs list (populates cache), appends 2nd task to JSONL manually, runs rebuild | Checks exit code, opens cache.db, verifies count = 2 |
| `"it updates hash in metadata table after rebuild"` | Creates 1 task, runs rebuild, queries metadata | Checks exit code, queries `SELECT value FROM metadata WHERE key='jsonl_hash'`, asserts non-empty |
| `"it acquires exclusive lock during rebuild"` | Creates 1 task, runs with `--verbose`, checks stderr | Checks exit code, checks stderr contains "verbose: lock acquired (exclusive)" and "verbose: lock released" |
| `"it outputs confirmation message with task count"` | Creates 3 tasks, runs rebuild | Checks exit code, checks stdout contains "3" and "rebuilt" |
| `"it suppresses output with --quiet"` | Creates 1 task, runs with `--quiet` | Checks exit code, checks `stdout.String() == ""` |
| `"it logs rebuild steps with --verbose"` | Creates 1 task, runs with `--verbose` | Checks exit code, checks stderr contains 4 expected verbose phrases |
| `"it handles empty JSONL (0 tasks rebuilt)"` | Creates empty project via `initTickProject`, runs rebuild | Checks exit code, checks stdout contains "0" |

V5 uses the global `Run()` function for integration testing. Tasks are created via `task.NewTask()` factory. The lock test indirectly verifies locking via verbose log output (not by actually contesting the lock).

**V6 Test Functions (322 lines, 8 subtests):**

| Test | Approach | Assertions |
|------|----------|------------|
| `"it rebuilds cache from JSONL"` | Creates 2 tasks with explicit struct literals, runs rebuild via `runRebuild` helper | Checks exit code, cache.db existence, task count via SQL, stdout contains "2" |
| `"it handles missing cache.db (fresh build)"` | Creates 1 task, removes cache.db, verifies removal, runs rebuild | Checks exit code, cache.db re-created, stdout contains "1" |
| `"it overwrites valid existing cache"` | Creates 1 task, runs rebuild, adds 2nd task via `storage.MarshalJSONL` + `os.WriteFile`, runs rebuild again | Checks exit code, opens cache.db, verifies count = 2 |
| `"it updates hash in metadata table after rebuild"` | Creates 1 task, runs rebuild, queries metadata | Checks exit code, queries hash, asserts non-empty AND asserts `len(hash) == 64` (SHA256 verification) |
| `"it acquires exclusive lock during rebuild"` | Creates 1 task, acquires lock via `flock.New().TryLock()`, runs rebuild | Checks exit code = 1, checks stderr contains "lock" |
| `"it outputs confirmation message with task count"` | Creates 3 tasks, runs rebuild | Checks `stdout == "Cache rebuilt: 3 tasks\n"` (exact match) |
| `"it suppresses output with --quiet"` | Creates 1 task, runs with `--quiet` | Checks exit code, `stdout == ""`, verifies cache.db still exists |
| `"it logs rebuild steps with --verbose"` | Creates 1 task, runs with `--verbose` | Checks stderr non-empty, checks 4 expected messages, verifies ALL verbose lines have "verbose: " prefix, checks stdout still has confirmation |
| `"it handles empty JSONL with 0 tasks rebuilt"` | Creates empty project, runs rebuild | Checks `stdout == "Cache rebuilt: 0 tasks\n"` (exact match), opens cache.db and verifies count = 0 |

V6 uses a dedicated `runRebuild()` test helper that constructs an `App` directly. Tasks use explicit struct literals with all fields.

**Test Quality Differences:**

1. **Lock test:** V6 is genuinely stronger. It acquires the lock externally via `flock.New().TryLock()` and verifies rebuild fails with exit code 1 -- a true concurrency contention test. V5 only checks verbose log output for "lock acquired (exclusive)", which proves the lock was taken but not that concurrent access is blocked.

2. **Hash verification:** V6 additionally asserts `len(hash) == 64` (SHA256 hex length). V5 only checks `hash != ""`.

3. **Output assertions:** V6 uses exact string matching (`stdout == "Cache rebuilt: 3 tasks\n"`) for confirmation messages. V5 uses `strings.Contains(output, "3")` and `strings.Contains(strings.ToLower(output), "rebuilt")`, which is more lenient.

4. **Verbose line prefix validation:** V6 iterates all stderr lines and asserts each starts with "verbose: ". V5 does not validate this prefix consistency.

5. **Quiet test coverage:** V6 additionally verifies cache.db exists after `--quiet` rebuild (confirming rebuild happened despite no output). V5 only checks stdout is empty.

6. **Empty JSONL test:** V6 opens the cache.db and verifies task count = 0 AND correct schema. V5 only checks stdout contains "0".

7. **Test helper pattern:** V6 has a reusable `runRebuild()` helper that returns stdout, stderr, exitCode as strings/int. V5 uses the global `Run()` function with raw `bytes.Buffer` variables -- more boilerplate per test.

8. **Task construction:** V5 uses `task.NewTask("tick-aaaaaa", "Task A")` factory (6-char hex IDs). V6 uses explicit struct literals `{ID: "tick-aaa111", Title: "Task one", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}` -- more verbose but explicit about what fields are set. V6 also uses different statuses (StatusDone with Closed pointer) for richer test data.

### Skill Compliance

**MUST DO requirements from golang-pro SKILL.md:**

| Requirement | V5 | V6 |
|-------------|-----|-----|
| Use gofmt and golangci-lint | PASS (standard formatting) | PASS (standard formatting) |
| Add context.Context to blocking operations | PASS -- `acquireExclusive()` uses `context.WithTimeout` internally | PASS -- `context.WithTimeout` used directly in rebuild.go |
| Handle all errors explicitly | PASS -- all errors checked and wrapped | PARTIAL -- `os.Remove(cachePath)` error silently discarded at rebuild.go:49 |
| Write table-driven tests with subtests | N/A -- tests use subtests but not table-driven (appropriate for integration tests) | N/A -- same |
| Document exported functions | PASS -- `Rebuild()` has doc comment, `runRebuild` is unexported | PASS -- `RunRebuild()` has doc comment |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- all error returns use `%w` wrapping | PASS -- all error returns use `%w` wrapping |

**MUST NOT DO requirements:**

| Requirement | V5 | V6 |
|-------------|-----|-----|
| Ignore errors without justification | PASS | PARTIAL -- `os.Remove(cachePath)` at rebuild.go:49 ignores ALL errors including permission errors |
| Use panic for error handling | PASS | PASS |
| Hardcode configuration | PASS -- uses Store's configurable lockTimeout | FAIL -- hardcodes `5 * time.Second` lock timeout and `50 * time.Millisecond` retry at rebuild.go:32-33 |

### Spec-vs-Convention Conflicts

**Verbose log message format:**

The spec says verbose should "log each step (delete, read, insert count, hash update) to stderr." Both versions comply, but with different message text:

- V5: `"deleting existing cache"`, `"reading tasks.jsonl"`, `"rebuilding cache (N tasks)"`, `"hash updated"`
- V6: `"deleting cache.db"`, `"reading JSONL"`, `"rebuilding cache with N tasks"`, `"hash updated"` plus additional lock messages `"acquiring exclusive lock"`, `"lock acquired"`, `"lock released"`

V6 logs more steps (lock lifecycle), which exceeds spec requirements but provides better diagnostic value. V5 relies on the Store's `acquireExclusive()` to log lock events, so they ARE logged when verbose is on -- just with different message text ("lock acquired (exclusive)" vs "lock acquired").

**Formatter invocation:**

The spec says "Output via `Formatter.FormatMessage()`". V5 calls `ctx.Fmt.FormatMessage(ctx.Stdout, msg)` where `FormatMessage` takes a writer and string. V6 calls `fmtr.FormatMessage(msg)` which returns a string, then passes it to `fmt.Fprintln(stdout, ...)`. Both achieve the spec requirement; the difference is in their respective `FormatMessage` signatures (V5's takes a writer, V6's returns a string).

**Separation of concerns:**

The spec says "CLI handler acquires exclusive file lock (same as write path)." V6 literally follows this by acquiring the lock in the CLI handler. V5 delegates lock acquisition to `Store.Rebuild()`, which means the CLI handler does NOT directly acquire the lock -- the Store does. This is technically a deviation from spec letter, but is the correct Go architectural decision. The spec's "same as write path" phrase suggests reusing the same locking mechanism, which V5 achieves more faithfully by reusing `acquireExclusive()`.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed (code only) | 3 (cli.go, rebuild.go, store.go) | 3 (app.go, rebuild.go, rebuild_test.go) |
| New files | 2 (rebuild.go, rebuild_test.go) | 2 (rebuild.go, rebuild_test.go) |
| Lines added (code, excl. docs) | ~336 | ~414 |
| rebuild.go lines | 33 | 81 |
| rebuild_test.go lines | 248 | 322 |
| store.go changes | +55 lines (Rebuild method + cachePath field) | 0 (no store modification) |
| Test count | 8 subtests | 8 subtests |
| External imports in CLI handler | 1 (engine) | 4 (context, flock, os, storage) |

## Verdict

**V5 is the stronger implementation.**

V5's architectural decision to place `Rebuild()` on the `Store` type is clearly superior. It maintains the existing separation of concerns (CLI as thin handler, Store as storage orchestrator), reuses the `acquireExclusive()` helper for consistent lock behavior, avoids leaking storage implementation details (flock, file paths) into the CLI layer, and produces a CLI handler that is 33 lines vs V6's 81 lines. V6 itself acknowledged this was the right approach by later refactoring the logic behind the Store in commit dce7d58 (tick-core-6-2).

V6's tests are stronger in specific areas: the lock contention test actually contests the lock (vs V5's verbose-log check), the hash length assertion is more thorough, output assertions use exact string matching, and the verbose prefix validation is more rigorous. V6 also verifies cache.db existence after `--quiet` mode and validates the empty JSONL case more deeply.

V6's code quality issues are notable: hardcoded lock timeout (violates MUST NOT "hardcode configuration"), silently discarded `os.Remove` error (violates MUST DO "handle all errors explicitly"), and direct flock import in the CLI layer (leaks storage internals).

Overall: V5 delivers better architecture with a cleaner, more maintainable implementation. V6 delivers better test coverage but with a fundamentally flawed separation of concerns that required a follow-up refactoring commit.
