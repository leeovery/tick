# Task 5-2: tick rebuild command

## Task Plan Summary

Implement `tick rebuild` -- a CLI command that forces a complete SQLite cache rebuild from JSONL, bypassing the freshness check. This is a diagnostic tool for corrupted caches, debugging, or manual JSONL edits.

Key requirements from the plan:
- Acquire exclusive file lock (same as write path)
- Delete existing `cache.db` if present
- Full rebuild: read JSONL, parse all records, create SQLite schema, insert all tasks
- Update hash in metadata table to current JSONL hash
- Release lock
- Output via `Formatter.FormatMessage()` showing count of tasks rebuilt
- `--quiet`: suppress output entirely
- `--verbose`: log each step (delete, read, insert count, hash update) to stderr
- Handle edge cases: missing cache.db, valid cache overwritten, empty JSONL, concurrent access

Nine acceptance criteria are defined, along with eight required test cases.

---

## V4 Implementation

### Architecture & Design

V4 uses a method-receiver pattern where `runRebuild` is a method on the `*App` struct:

```go
func (a *App) runRebuild(args []string) error {
```

The `App` struct holds global flags (`Quiet`, `Verbose`), I/O writers, and the working directory. Command dispatch is a `switch` statement in `App.Run()` with each case following an identical pattern:

```go
case "rebuild":
    if err := a.runRebuild(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

The `Store.Rebuild()` method in `internal/store/store.go` handles all the logic. V4's Store does not hold a persistent `*cache.Cache` -- it opens/closes the cache within each operation. The `Rebuild()` method directly inlines the locking logic, duplicating the same lock acquisition pattern found in `Mutate()` and `Query()`:

```go
func (s *Store) Rebuild() (int, error) {
    s.vlog("acquiring exclusive lock on %s", s.lockPath)
    fl := flock.New(s.lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
    defer cancel()

    locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
    if err != nil {
        return 0, fmt.Errorf("could not acquire lock on %s - ...", s.lockPath)
    }
    if !locked {
        return 0, fmt.Errorf("could not acquire lock on %s - ...", s.lockPath)
    }
    // ...
```

This is 15 lines of boilerplate lock acquisition repeated verbatim across three methods (`Mutate`, `Query`, `Rebuild`).

V4 uses `cache.Open(s.dbPath)` which returns a `*Cache` that the caller defers close on. Since V4's Store does not hold a persistent cache connection, it can safely delete the `cache.db` file and open a fresh one.

The verbose logging in V4 uses a `LogFunc func(format string, args ...interface{})` field on the Store, injected by the CLI via `s.LogFunc = a.vlog.Log`. The V4 `VerboseLogger.Log` method is variadic-format-based:

```go
func (v *VerboseLogger) Log(format string, args ...interface{}) {
    msg := fmt.Sprintf(format, args...)
    fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
```

### Code Quality

The CLI handler is clean and minimal (30 lines). Error wrapping uses `fmt.Errorf("failed to ...: %w", err)` consistently. The output message format is `"Rebuilt cache: %d tasks"`.

Notable concerns:
1. **Lock duplication**: The lock acquisition boilerplate is copied verbatim from `Mutate()` rather than extracted into a helper. This is a DRY violation.
2. **No cache connection management issue**: Since V4's Store doesn't hold a persistent `*cache.Cache`, there's no need to close/reopen it during rebuild. This is simpler but means every operation (Query, Mutate) has to open its own cache.
3. **`vlog` naming**: The Store uses `s.vlog()` internally, which is a private method. The public API exposes `LogFunc` as a settable field rather than using functional options.
4. **Error message inconsistency**: Lock errors include the full lock path in the message (`could not acquire lock on /path/to/lock`), which leaks internal paths.

### Test Coverage

V4 has 9 test functions, one per file-level function, each containing a single `t.Run` subtest:

| Test Function | What It Tests |
|---|---|
| `TestRebuild_RebuildsCacheFromJSONL` | Basic rebuild with 3 tasks, verifies DB count |
| `TestRebuild_HandlesMissingCacheDB` | Removes cache.db before rebuild, verifies creation |
| `TestRebuild_OverwritesValidExistingCache` | Populates cache via list, modifies JSONL, rebuilds, verifies new count |
| `TestRebuild_UpdatesHashInMetadataTable` | Checks metadata table has non-empty hash after rebuild |
| `TestRebuild_AcquiresExclusiveLock` | Holds external flock, expects exit code 1 with "lock" in error |
| `TestRebuild_OutputsConfirmationWithTaskCount` | Checks stdout contains "3" and "rebuilt" (case-insensitive) |
| `TestRebuild_SuppressesOutputWithQuiet` | Passes `--quiet`, checks stdout is empty |
| `TestRebuild_LogsRebuildStepsWithVerbose` | Passes `--verbose`, checks stderr for delete/read/insert/hash keywords |
| `TestRebuild_HandlesEmptyJSONL` | Empty JSONL, checks output contains "0" and DB has 0 rows |

**Strengths**:
- The lock test (`TestRebuild_AcquiresExclusiveLock`) uses an actual external `flock.New()` to hold the lock and then verifies the rebuild fails with exit code 1. This is a genuine concurrency test.
- Each test verifies the actual SQLite database content by opening `cache.db` directly.
- The verbose test checks for four distinct log categories (delete, read, insert/rebuild, hash) using substring matching.

**Weaknesses**:
- Tests are split into separate top-level functions rather than using a single parent `TestRebuild` with subtests. This is a minor structural issue -- the golang-pro skill says "write table-driven tests with subtests."
- Task creation in tests uses struct literals with manual field assignment including `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)`, which is verbose.
- The verbose test checks are loose (`strings.ToLower(stderrStr)` with `"delet"`, `"read"`, `"insert"` etc.), not verifying the exact verbose format.
- The confirmation output test has a convoluted dual check for "rebuilt"/"Rebuilt" that is unnecessary.
- Direct import of `github.com/gofrs/flock` in test file -- this is a test-only dependency that couples tests to the lock implementation.
- Each test creates a new `App` struct manually.

### Spec Compliance

| Acceptance Criterion | Met? | Notes |
|---|---|---|
| Rebuilds SQLite from JSONL regardless of freshness | Yes | `Rebuild()` skips `EnsureFresh`, does full rebuild |
| Deletes existing cache before rebuild | Yes | `os.Remove(s.dbPath)` with `!os.IsNotExist` guard |
| Updates hash in metadata table | Yes | `c.Rebuild(tasks, jsonlData)` handles this |
| Acquires exclusive lock during rebuild | Yes | Uses `TryLockContext` with timeout |
| Handles missing cache.db without error | Yes | `os.Remove` error ignored if not-exist |
| Handles empty JSONL (0 tasks rebuilt) | Yes | Works with empty byte slice |
| Outputs confirmation with task count | Yes | `"Rebuilt cache: %d tasks"` via `FormatMessage` |
| --quiet suppresses output | Yes | `if a.Quiet { return nil }` |
| --verbose logs rebuild steps to stderr | Yes | Logs delete, read, count, hash steps |

Full spec compliance.

### golang-pro Skill Compliance

| Rule | Compliant? | Notes |
|---|---|---|
| Handle all errors explicitly | Yes | All errors checked |
| Propagate errors with fmt.Errorf("%w", err) | Yes | Consistent wrapping |
| Write table-driven tests with subtests | Partial | Has subtests but not table-driven; tests are separate functions |
| Document all exported functions | Yes | `Rebuild()` has doc comment |
| No panic for error handling | Yes | No panics |
| No ignored errors without justification | Yes | Clean |
| Add context.Context to blocking operations | Partial | Lock acquisition uses context but `Rebuild()` itself does not accept a context parameter |

---

## V5 Implementation

### Architecture & Design

V5 uses a function-based handler pattern where `runRebuild` is a package-level function taking a `*Context`:

```go
func runRebuild(ctx *Context) error {
```

The `Context` struct holds parsed CLI state (WorkDir, Stdout, Stderr, Quiet, Verbose, Format, Fmt, Args). Command dispatch uses a map:

```go
var commands = map[string]func(*Context) error{
    // ...
    "rebuild": runRebuild,
}
```

This is cleaner than V4's switch statement -- adding a new command is one line in the map rather than a copy-pasted case block.

V5's `Store` in `internal/engine/store.go` holds a persistent `*cache.Cache` connection and stores `cachePath` as a field. The constructor opens the cache eagerly:

```go
func NewStore(tickDir string, opts ...Option) (*Store, error) {
    // ...
    c, err := cache.New(cachePath)
    // ...
    s := &Store{
        jsonlPath:   jsonlPath,
        cachePath:   cachePath,
        cache:       c,
        // ...
    }
```

This means `Rebuild()` must close the existing cache connection before deleting the file, then open a new one:

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

    // ... read and parse JSONL ...

    c, err := cache.New(s.cachePath)
    if err != nil {
        return 0, fmt.Errorf("creating cache: %w", err)
    }
    s.cache = c

    if err := s.cache.Rebuild(tasks, jsonlData); err != nil {
        return 0, fmt.Errorf("rebuilding cache: %w", err)
    }

    s.verbose.Log("hash updated")
    return len(tasks), nil
}
```

Critically, V5 has extracted lock acquisition into a reusable helper:

```go
func (s *Store) acquireExclusive() (unlock func(), err error) {
    fl := flock.New(s.lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)

    locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
    if !locked || err != nil {
        cancel()
        return nil, fmt.Errorf("%s", lockTimeoutMsg)
    }

    s.verbose.Log("lock acquired (exclusive)")

    return func() {
        _ = fl.Unlock()
        cancel()
        s.verbose.Log("lock released")
    }, nil
}
```

This is used by `Mutate()`, `Query()`, and `Rebuild()` -- no duplication. The returned `unlock` function encapsulates both the flock release and context cancellation.

V5 uses functional options (`WithVerbose`, `WithLockTimeout`) instead of V4's public field injection:

```go
store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
```

The `VerboseLogger` in V5 lives in the `engine` package and has separate `Log(string)` and `Logf(string, ...interface{})` methods:

```go
func (v *VerboseLogger) Log(msg string) { ... }
func (v *VerboseLogger) Logf(format string, args ...interface{}) { ... }
```

V5 uses `storage.ParseTasks(jsonlData)` from a dedicated `internal/storage` package instead of V4's `task.ReadJSONLFromBytes(jsonlData)`. This is a cleaner separation of concerns -- the `task` package defines the model, the `storage` package handles serialization.

### Code Quality

The CLI handler is 33 lines, slightly longer than V4 due to the explicit `engine.NewStore` call with options. The output message is `"Cache rebuilt: %d tasks"` (note: different wording than V4's `"Rebuilt cache: %d tasks"` -- both are acceptable).

One notable issue: the `FormatMessage` return value is silently ignored:

```go
if !ctx.Quiet {
    ctx.Fmt.FormatMessage(ctx.Stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))
}
return nil
```

V4 returns the error from `FormatMessage`:

```go
msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
return a.Formatter.FormatMessage(a.Stdout, msg)
```

This is a defect in V5: if `FormatMessage` returns an error (e.g., write failure), V5 swallows it. V4 propagates it correctly. The golang-pro skill says "Handle all errors explicitly (no naked returns)" -- V5 violates this.

**Strengths**:
1. **DRY lock management**: `acquireExclusive()` returns an `unlock` func, eliminating all lock boilerplate duplication.
2. **Functional options**: `WithVerbose`, `WithLockTimeout` follow idiomatic Go patterns.
3. **Proper cache lifecycle**: Closes existing cache before delete, reassigns `s.cache` to new connection. This is necessary given V5's persistent cache design.
4. **Separated storage package**: `storage.ParseTasks` vs V4's `task.ReadJSONLFromBytes`.
5. **Consistent error messages**: Uses a constant `lockTimeoutMsg` instead of formatting the lock path into each error.
6. **Map-based command dispatch**: Eliminates the switch statement boilerplate.

**Concerns**:
1. **Swallowed FormatMessage error**: As noted above, `ctx.Fmt.FormatMessage()` return value is discarded.
2. **Cache state after error**: If `s.cache.Close()` succeeds but `os.Remove` fails with a non-IsNotExist error, the Store's `s.cache` field points to a closed cache. Subsequent operations would fail. However, the function returns an error in this case, so the Store is effectively dead.
3. **Reassigning `s.cache`**: Mutating a struct field (`s.cache = c`) in the middle of an operation is somewhat unusual, but necessary given the architecture.

### Test Coverage

V5 has a single parent `TestRebuild` with 9 subtests:

| Subtest | What It Tests |
|---|---|
| `it rebuilds cache from JSONL` | 2 tasks, verifies DB count and cache.db existence |
| `it handles missing cache.db (fresh build)` | Removes cache.db, verifies creation |
| `it overwrites valid existing cache` | Populates via list, appends task to JSONL, rebuilds, verifies count |
| `it updates hash in metadata table after rebuild` | Checks metadata table for non-empty hash |
| `it acquires exclusive lock during rebuild` | Uses `--verbose` to check for "lock acquired (exclusive)" in stderr |
| `it outputs confirmation message with task count` | 3 tasks, checks for "3" and "rebuilt" in output |
| `it suppresses output with --quiet` | `--quiet` flag, checks stdout is empty |
| `it logs rebuild steps with --verbose` | Checks for exact phrases in stderr |
| `it handles empty JSONL (0 tasks rebuilt)` | Empty project, checks output contains "0" |

**Strengths**:
- Single parent `TestRebuild` with subtests aligns better with the golang-pro skill's guidance.
- Uses `task.NewTask("tick-aaaaaa", "Task A")` constructor instead of manual struct literals. This is cleaner and leverages the domain model properly.
- The verbose test checks exact expected phrases rather than loose substring matching:
  ```go
  expectedPhrases := []string{
      "verbose: deleting existing cache",
      "verbose: reading tasks.jsonl",
      "verbose: rebuilding cache",
      "verbose: hash updated",
  }
  ```
- Uses `Run()` function directly rather than constructing `App` structs, matching V5's functional CLI design.
- Test helper `initTickProjectWithTasks` uses `task.MarshalJSON()` for serialization, which is more correct than V4's `task.WriteJSONL` (though both work).

**Weaknesses**:
- The lock test does NOT use an actual external flock to verify mutual exclusion. Instead, it only checks verbose output for "lock acquired (exclusive)". This verifies that the log message is emitted but does not prove that an exclusive lock is actually acquired. V4's approach of holding an external lock and expecting failure is a much stronger test.
- The empty JSONL test does not verify that the cache.db was created with correct schema (no DB query). V4's test does verify this.
- Tests are still not table-driven (each subtest has unique setup logic, which is arguably necessary for these varied scenarios).
- No `_ "github.com/mattn/go-sqlite3"` import in the test file -- this works because V5's `initTickProject` helper calls `Run([]string{"tick", "init"}, ...)` which triggers the import chain through the engine package. This is fragile and relies on transitive import.

### Spec Compliance

| Acceptance Criterion | Met? | Notes |
|---|---|---|
| Rebuilds SQLite from JSONL regardless of freshness | Yes | `Rebuild()` skips `readAndEnsureFresh`, deletes and recreates |
| Deletes existing cache before rebuild | Yes | Closes cache, `os.Remove(s.cachePath)` |
| Updates hash in metadata table | Yes | `s.cache.Rebuild(tasks, jsonlData)` handles this |
| Acquires exclusive lock during rebuild | Yes | `s.acquireExclusive()` |
| Handles missing cache.db without error | Yes | `!os.IsNotExist(err)` guard |
| Handles empty JSONL (0 tasks rebuilt) | Yes | Works with empty data |
| Outputs confirmation with task count | Yes | `"Cache rebuilt: %d tasks"` |
| --quiet suppresses output | Yes | `if !ctx.Quiet` guard |
| --verbose logs rebuild steps to stderr | Yes | Four distinct log messages |

Full spec compliance.

### golang-pro Skill Compliance

| Rule | Compliant? | Notes |
|---|---|---|
| Handle all errors explicitly | **No** | `FormatMessage` return value ignored in CLI handler |
| Propagate errors with fmt.Errorf("%w", err) | Yes | Consistent wrapping |
| Write table-driven tests with subtests | Partial | Has subtests but not table-driven |
| Document all exported functions | Yes | `Rebuild()`, `NewStore()`, etc. all documented |
| No panic for error handling | Yes | No panics |
| No ignored errors without justification | **No** | `FormatMessage` error swallowed |
| Add context.Context to blocking operations | Partial | Lock uses context internally but `Rebuild()` doesn't accept one |

---

## Comparative Analysis

### Where V4 is Better

1. **Lock test quality**: V4's `TestRebuild_AcquiresExclusiveLock` holds an actual external `flock` and verifies that rebuild fails with exit code 1 when the lock is held. This is a genuine concurrency test that proves mutual exclusion works. V5 merely checks that a verbose log message was emitted, which only proves logging works -- not that locking actually happens. This is a significant testing gap in V5.

2. **FormatMessage error handling**: V4 correctly returns the error from `a.Formatter.FormatMessage(a.Stdout, msg)`. V5 discards it:
   ```go
   // V4 (correct)
   return a.Formatter.FormatMessage(a.Stdout, msg)

   // V5 (error swallowed)
   ctx.Fmt.FormatMessage(ctx.Stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))
   return nil
   ```
   This is a clear golang-pro skill violation in V5.

3. **Empty JSONL test thoroughness**: V4's empty JSONL test verifies both the output AND the actual database state (opens cache.db, queries `SELECT COUNT(*) FROM tasks`, asserts 0). V5 only checks the output string.

4. **Simpler cache lifecycle**: Because V4's Store does not hold a persistent `*cache.Cache`, the Rebuild method can simply delete the file and open a new cache. There is no risk of a stale `s.cache` field. V5 must carefully close, delete, reopen, and reassign.

### Where V5 is Better

1. **DRY lock management**: V5 extracts lock acquisition into `acquireExclusive()` returning an unlock function. This eliminates ~15 lines of duplicated boilerplate per method. V4 copies the same lock code in `Mutate()`, `Query()`, and `Rebuild()`.

2. **Functional options pattern**: `engine.NewStore(tickDir, ctx.storeOpts()...)` with `WithVerbose()` and `WithLockTimeout()` is idiomatic Go. V4 uses a public field `s.LogFunc = a.vlog.Log` which is less encapsulated and allows mutation after construction.

3. **Command dispatch architecture**: V5's map-based dispatch (`var commands = map[string]func(*Context) error{...}`) is cleaner than V4's 60-line switch statement. Adding a command is a single map entry.

4. **Test organization**: Single parent `TestRebuild` with subtests is cleaner than V4's nine separate top-level test functions.

5. **Task construction in tests**: V5 uses `task.NewTask("tick-aaaaaa", "Task A")` vs V4's manual struct literals with explicit timestamps. The constructor is more concise and less error-prone.

6. **Verbose log verification**: V5 checks for exact expected phrases in a loop:
   ```go
   expectedPhrases := []string{
       "verbose: deleting existing cache",
       "verbose: reading tasks.jsonl",
       "verbose: rebuilding cache",
       "verbose: hash updated",
   }
   for _, phrase := range expectedPhrases {
       if !strings.Contains(stderrStr, phrase) {
           t.Errorf("expected stderr to contain %q, got:\n%s", phrase, stderrStr)
       }
   }
   ```
   This is more precise and maintainable than V4's ad-hoc substring checks with `strings.ToLower()`.

7. **Package separation**: V5 places task serialization in `internal/storage` and the store engine in `internal/engine`, keeping `internal/task` purely as the domain model. V4 mixes serialization into the `task` package.

8. **VerboseLogger API**: V5 separates `Log(string)` from `Logf(string, ...interface{})`, which is cleaner than V4's single `Log(format string, args ...interface{})` that always runs through `fmt.Sprintf`.

### Differences That Are Neutral

1. **Output message wording**: V4 says `"Rebuilt cache: %d tasks"`, V5 says `"Cache rebuilt: %d tasks"`. Both satisfy the spec requirement of "confirmation showing count of tasks rebuilt."

2. **Cache constructor naming**: V4 uses `cache.Open()`, V5 uses `cache.New()`. Both are acceptable Go naming.

3. **Lock poll interval**: V4 uses `100ms`, V5 uses `50ms`. Both are reasonable.

4. **Error message format for lock failure**: V4 includes the full lock path, V5 uses a constant message. V5's approach is arguably better (no path leakage) but both are acceptable.

5. **Test helper design**: V4 uses `setupInitializedDir`/`setupInitializedDirWithTasks` (manual dir + file creation), V5 uses `initTickProject`/`initTickProjectWithTasks` (runs `tick init` then writes tasks). V5's approach exercises more of the actual code path, but V4's is more isolated.

---

## Verdict

**Winner: V5**, with one notable defect that must be addressed.

V5 demonstrates superior architecture across every structural dimension: DRY lock management via `acquireExclusive()`, functional options, map-based command dispatch, cleaner package separation, better test organization, and more precise verbose log assertions. These are not cosmetic differences -- they represent meaningfully better code that is easier to extend, maintain, and reason about.

However, V5 has two concrete deficiencies:

1. **The FormatMessage error is swallowed** -- this is a golang-pro skill violation ("Handle all errors explicitly") and a real bug. If stdout is a broken pipe, the error disappears silently. V4 handles this correctly.

2. **The lock test is weak** -- V5 only verifies verbose log output to prove locking, while V4 actually holds an external flock and demonstrates that the rebuild fails when the lock is contended. V5's test proves that a log message is printed, not that mutual exclusion works.

Despite these two issues, V5 wins because the architectural improvements are substantial and systemic, while the defects are localized and easily fixable (one-line fix for the error, swap the lock test approach). V4's lock duplication across three methods is a maintainability problem that compounds as the codebase grows, and its switch-statement dispatch pattern doesn't scale well either.

If the FormatMessage error handling were fixed and the lock test used an actual external flock (as V4 does), V5 would be unambiguously better in every dimension.
