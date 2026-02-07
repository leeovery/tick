# Task tick-core-5-2: tick rebuild command

## Task Summary

Implement `tick rebuild` -- force a complete SQLite cache rebuild from JSONL, bypassing the freshness check. This is a diagnostic tool for corrupted cache, debugging, or after manual JSONL edits.

**Requirements:**
- CLI handler acquires exclusive file lock (same as write path)
- Delete existing `cache.db` if present
- Full rebuild: read JSONL, parse all records, create SQLite schema, insert all tasks
- Update hash in metadata table to current JSONL hash
- Release lock
- Output via `Formatter.FormatMessage()` -- confirmation showing count of tasks rebuilt
- `--quiet`: suppress output entirely
- `--verbose`: log each step (delete, read, insert count, hash update) to stderr

**Acceptance Criteria:**
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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Rebuilds SQLite from JSONL regardless of freshness | PASS -- `ForceRebuild()` bypasses freshness check entirely | PASS -- `ForceRebuild()` bypasses freshness check entirely | PASS -- `Rebuild()` bypasses freshness check entirely |
| Deletes existing cache before rebuild | PASS -- closes cache, calls `os.Remove(s.cachePath)` | PASS -- calls `os.Remove(s.cachePath)` (no close needed, opens per-op) | PASS -- calls `os.Remove(s.cachePath)` (no close needed, opens per-op) |
| Updates hash in metadata table | PASS -- delegates to `cache.Rebuild(tasks, content)` which updates hash | PASS -- delegates to `cache.Rebuild(tasks, rawContent)` which updates hash | PASS -- delegates to `cache.Rebuild(tasks, jsonlContent)` which updates hash |
| Acquires exclusive lock during rebuild | PASS -- calls `acquireExclusiveLock()` at top of `ForceRebuild` | PASS -- creates new `flock.New()` and calls `TryLockContext` inline | PASS -- uses stored `s.flock` and calls `TryLockContext` inline |
| Handles missing cache.db without error | PASS -- `os.Remove` is unconditional (no error check on missing file) | PASS -- `os.Remove` is unconditional | PASS -- `os.Remove` is unconditional |
| Handles empty JSONL (0 tasks rebuilt) | PASS -- test "handles empty JSONL" confirms 0 count | PASS -- test "it handles empty JSONL (0 tasks rebuilt)" confirms 0 count + verifies schema | PASS -- test "it handles empty JSONL with 0 tasks rebuilt" confirms 0 count + verifies schema |
| Outputs confirmation with task count | PASS -- `FormatMessage(stdout, "Cache rebuilt: %d tasks")` | PASS -- `FormatMessage(stdout, "Rebuilt cache: %d tasks")` | PASS -- `FormatMessage("Rebuilt cache: %d tasks")` via Formatter |
| `--quiet` suppresses output | PASS -- checks `a.opts.Quiet` before formatting | PASS -- checks `a.config.Quiet` before formatting | PASS -- checks `a.formatConfig.Quiet` before formatting |
| `--verbose` logs rebuild steps to stderr | PARTIAL -- `logf()` calls exist in `ForceRebuild` but test only checks for `"verbose:"` prefix, only 2 log points (delete + rebuild count) | PASS -- extensive `logVerbose()` calls covering lock acquire/release, read, delete, insert count, hash update | PARTIAL -- verbose messages are emitted from CLI handler (`WriteVerbose` calls in `runRebuild`), not from the store, so they fire even if the store operation fails |

## Implementation Comparison

### Approach

All three versions follow the same high-level architecture: a CLI handler in `rebuild.go` discovers the tick directory, opens a store, calls a force-rebuild method, and formats the output. The differences are in where the verbose logging lives, how locks are managed, and how the store is structured.

**V1: `cmdRebuild` + `Store.ForceRebuild` -- Stateful cache model**

V1's store keeps an open `*Cache` field on the `Store` struct. `ForceRebuild` must explicitly close this cache before deleting the file, then reopen a fresh one:

```go
// V1 store.go - ForceRebuild
s.logf("deleting existing cache")
if err := s.cache.Close(); err != nil {
    return 0, fmt.Errorf("closing cache: %w", err)
}
os.Remove(s.cachePath)

// Reopen cache (creates fresh).
cache, err := NewCacheWithRecovery(s.cachePath)
if err != nil {
    return 0, fmt.Errorf("reopening cache: %w", err)
}
s.cache = cache
```

Locking uses a shared `acquireExclusiveLock()` helper method that returns a `*flock.Flock` for deferred unlock. This is clean code reuse shared with `Mutate`.

The CLI handler is minimal (27 lines) -- pure delegation:

```go
// V1 rebuild.go
func (a *App) cmdRebuild(workDir string) error {
    tickDir, err := FindTickDir(workDir)
    // ...
    store, err := a.openStore(tickDir)
    // ...
    count, err := store.ForceRebuild()
    // ...
    return a.fmtr.FormatMessage(a.stdout, fmt.Sprintf("Cache rebuilt: %d tasks", count))
}
```

**V2: `runRebuild` + `Store.ForceRebuild` -- Per-operation cache model**

V2's store does not hold an open cache; it opens/closes per operation. The `ForceRebuild` method creates its own `flock.New()` and manages locking inline, duplicating the lock pattern from `Mutate` and `Query`:

```go
// V2 store.go - ForceRebuild
fl := flock.New(s.lockPath)
s.logVerbose("lock: acquiring exclusive lock")
ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
defer cancel()
locked, err := fl.TryLockContext(ctx, 10*time.Millisecond)
```

This version has the most verbose logging in the store layer, with 7 distinct `logVerbose` calls covering every step: lock acquire, lock release, reading JSONL, parsing task count, deleting cache, inserting tasks, and hash update.

The CLI handler is similarly minimal (31 lines) and purely delegates:

```go
// V2 rebuild.go
func (a *App) runRebuild() error {
    tickDir, err := DiscoverTickDir(a.workDir)
    // ...
    store, err := a.newStore(tickDir)
    // ...
    count, err := store.ForceRebuild()
    // ...
    msg := fmt.Sprintf("Rebuilt cache: %d tasks", count)
    return a.formatter.FormatMessage(a.stdout, msg)
}
```

**V3: `runRebuild` + `Store.Rebuild` -- Per-operation cache with CLI-level verbose**

V3 takes a fundamentally different approach to verbose logging. Instead of logging from within the store, V3 emits all verbose messages from the CLI handler itself:

```go
// V3 rebuild.go
a.WriteVerbose("lock acquire exclusive")
a.WriteVerbose("delete existing cache.db")
a.WriteVerbose("read JSONL tasks")
count, err := store.Rebuild()
a.WriteVerbose("insert %d tasks into cache", count)
a.WriteVerbose("hash updated in metadata table")
a.WriteVerbose("lock release")
```

This is architecturally questionable because the verbose messages describe what the store *should be doing internally*, but they fire before the store operation even starts. The "lock acquire exclusive", "delete existing cache.db", and "read JSONL tasks" messages all appear before `store.Rebuild()` is called. Even if the store fails, some of these messages will have already been emitted, giving misleading output.

V3 is also the only version that returns `int` from `runRebuild` instead of `error`, handling error formatting inline:

```go
// V3 rebuild.go
func (a *App) runRebuild() int {
    // ... error handling inline with fmt.Fprintf(a.Stderr, "Error: %s\n", err)
```

V3's store method is named `Rebuild()` (not `ForceRebuild`), uses a stored `*flock.Flock` on the struct rather than creating a new one per call, and has more granular error handling for lock acquisition:

```go
// V3 store.go - Rebuild
locked, err := s.flock.TryLockContext(ctx, 50*time.Millisecond)
if err != nil || !locked {
    if errors.Is(err, context.DeadlineExceeded) || !locked {
        return 0, errors.New("could not acquire lock...")
    }
    return 0, fmt.Errorf("failed to acquire lock: %w", err)
}
```

V3 is also the only version to add `rebuild` to the help text / usage output in `printUsage()`.

### Code Quality

**Naming:**
- V1 uses `cmdRebuild` / `ForceRebuild` -- the `cmd` prefix matches the V1 convention (`cmdList`, `cmdCreate`, etc.)
- V2 uses `runRebuild` / `ForceRebuild` -- the `run` prefix matches the V2 convention
- V3 uses `runRebuild` / `Rebuild` -- the shorter `Rebuild` name is arguably clearer since the method's behavior is always a force-rebuild; there's no conditional freshness-check variant on the public API

**Error Handling:**
- V1: `ForceRebuild` wraps each error with context (`"closing cache: %w"`, `"reopening cache: %w"`, etc.). 6 error-wrapping points.
- V2: `ForceRebuild` wraps each error similarly. 5 error-wrapping points. Verbose logging is most thorough.
- V3: `Rebuild` uses minimal error wrapping -- delegates to helper `readJSONLWithContent()` for JSONL read errors, wraps only cache creation and rebuild errors. Lock error handling is most granular, distinguishing `DeadlineExceeded` from other lock failures.

**DRY Principle:**
- V1: Lock acquisition is extracted into `acquireExclusiveLock()` shared with `Mutate`. Best DRY compliance.
- V2: Lock acquisition is duplicated inline in `ForceRebuild`, `Mutate`, and `Query` -- each creates its own `flock.New()`, context, and timeout. Worst DRY compliance.
- V3: Lock uses a stored `s.flock` on the struct (created once in `NewStore`), but the `TryLockContext` + error handling is duplicated inline across `Mutate`, `Query`, and `Rebuild`.

**Verbose Logging Architecture:**
- V1: `logf()` calls in store using a `LogFunc` callback set via `SetLogger`. Minimalist -- only 2 log points in `ForceRebuild`.
- V2: `logVerbose()` calls in store using a `Logger` interface set via `SetLogger`. Most thorough -- 7 log points covering every rebuild step.
- V3: `WriteVerbose()` calls in CLI handler. Decoupled from store internals but architecturally misleading -- messages describe store behavior but are emitted from outside the store, before the operation runs.

**Formatter Usage:**
- V1: `a.fmtr.FormatMessage(a.stdout, msg)` -- writes to writer
- V2: `a.formatter.FormatMessage(a.stdout, msg)` -- writes to writer
- V3: `fmt.Fprint(a.Stdout, formatter.FormatMessage(msg))` -- formatter returns string, caller writes. Different API pattern from V1/V2.

**Cache Lifecycle:**
- V1: Must explicitly close the stored `s.cache` before deleting the file, then reassign `s.cache`. This is fragile -- if `ForceRebuild` fails after setting `s.cache = cache` but before rebuild, the old cache is lost.
- V2: Creates a local `cache` with `defer cache.Close()`. Clean lifecycle, no mutation of store state.
- V3: Creates a local `cache` with `defer cache.Close()`. Same clean pattern as V2.

### Test Quality

**V1 Test Functions (7 tests, 121 lines):**

1. `TestRebuildCommand/"rebuilds cache from JSONL"` -- creates 2 tasks, runs rebuild, checks output contains "2", verifies list still shows tasks
2. `TestRebuildCommand/"handles missing cache.db (fresh build)"` -- creates task, deletes cache.db, rebuilds, verifies list shows task
3. `TestRebuildCommand/"overwrites valid existing cache"` -- rebuilds twice, checks both succeed (exit 0)
4. `TestRebuildCommand/"handles empty JSONL"` -- empty dir, rebuild, checks output contains "0"
5. `TestRebuildCommand/"outputs confirmation message with task count"` -- 3 tasks, checks output has "3" and "rebuilt"
6. `TestRebuildCommand/"suppresses output with --quiet"` -- creates App directly, checks `outBuf.Len() == 0`
7. `TestRebuildCommand/"logs rebuild steps with --verbose"` -- checks stderr contains `"verbose:"`

**V2 Test Functions (8 tests, 341 lines):**

1. `TestRebuild/"it rebuilds cache from JSONL"` -- 3 tasks with dependencies, directly queries SQLite for task count (3) and dependency count (1)
2. `TestRebuild/"it handles missing cache.db (fresh build)"` -- verifies cache.db doesn't exist before, exists after, queries task count
3. `TestRebuild/"it overwrites valid existing cache"` -- runs list to build cache (2 tasks), modifies JSONL to 3 tasks, rebuilds, verifies cache has 3
4. `TestRebuild/"it updates hash in metadata table after rebuild"` -- queries `metadata WHERE key = 'jsonl_hash'`, checks non-empty
5. `TestRebuild/"it acquires exclusive lock during rebuild"` -- runs with `--verbose`, checks stderr for `"exclusive lock"`
6. `TestRebuild/"it outputs confirmation message with task count"` -- 3 tasks, checks stdout for "3" and "rebuilt"/"rebuild"
7. `TestRebuild/"it suppresses output with --quiet"` -- checks `output == ""`
8. `TestRebuild/"it handles empty JSONL (0 tasks rebuilt)"` -- empty dir, checks "0" in output, queries SQLite for count=0 and schema existence

**V3 Test Functions (8 tests, 294 lines):**

1. `TestRebuildCommand/"it rebuilds cache from JSONL"` -- 2 tasks, queries SQLite for count=2
2. `TestRebuildCommand/"it handles missing cache.db (fresh build)"` -- removes cache.db, rebuilds, verifies created, queries count=1
3. `TestRebuildCommand/"it overwrites valid existing cache"` -- runs list first, adds task to JSONL, rebuilds, queries count=2
4. `TestRebuildCommand/"it updates hash in metadata table after rebuild"` -- queries metadata hash, checks non-empty AND verifies 64-char SHA256 length
5. `TestRebuildCommand/"it acquires exclusive lock during rebuild"` -- runs twice (once normal, once verbose), checks stderr for `"lock acquire exclusive"`
6. `TestRebuildCommand/"it outputs confirmation message with task count"` -- 3 tasks, checks "3" and "Rebuilt" (capital R, exact)
7. `TestRebuildCommand/"it suppresses output with --quiet"` -- checks `stdout.String() == ""`
8. `TestRebuildCommand/"it logs rebuild steps with --verbose"` -- checks 6 specific messages: lock acquire, delete, read JSONL, insert count, hash, lock release

**Test Comparison:**

| Test Scenario | V1 | V2 | V3 |
|---------------|-----|-----|-----|
| Basic rebuild from JSONL | Yes (output check) | Yes (SQLite query) | Yes (SQLite query) |
| Missing cache.db | Yes (list verify) | Yes (stat + SQLite query) | Yes (stat + SQLite query) |
| Overwrite existing cache | Yes (double rebuild) | Yes (list -> edit JSONL -> rebuild) | Yes (list -> add task -> rebuild) |
| Hash update in metadata | No | Yes | Yes (with 64-char check) |
| Exclusive lock verification | No | Yes (verbose output check) | Yes (verbose output check) |
| Confirmation message | Yes | Yes | Yes |
| Quiet flag | Yes | Yes | Yes |
| Verbose logging | Yes (basic prefix check) | Yes (checks delete/read/hash) | Yes (checks all 6 steps) |
| Empty JSONL | Yes (output check only) | Yes (output + SQLite schema) | Yes (output + SQLite schema) |
| Dependency preservation | No | Yes (checks dep count) | No |

**Tests unique to V1:** None -- all V1 tests are a subset of V2/V3.

**Tests unique to V2:**
- Dependency count verification in "rebuilds cache from JSONL" test
- Uses `taskJSONL()` helper for explicit JSONL construction (rather than `createTask`/`setupTask`)

**Tests unique to V3:**
- SHA256 hash length validation (`len(hash) != 64`)
- Most thorough verbose step validation (6 specific message checks)
- Lock acquire AND release verification in verbose test

**Key test gaps:**
- V1 is missing: hash update test, lock acquisition test (2 criteria untested)
- V1 overwrite test only runs rebuild twice rather than verifying cache actually changes
- V2/V3 use direct SQLite queries for verification (stronger), V1 relies on subsequent `tick list` output (weaker, indirect)
- None test concurrent access (exclusive lock preventing reads)

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 4 | 6 | 7 |
| Lines added | 193 | 432 | 409 |
| Impl LOC (rebuild.go) | 27 | 31 | 50 |
| Impl LOC (store.go addition) | 43 | 54 | 42 |
| Test LOC | 121 | 341 | 294 |
| Test functions | 7 | 8 | 8 |

## Verdict

**V2 is the best implementation.**

**Test thoroughness:** V2 has the most comprehensive test suite at 341 lines with 8 tests. It is the only version that verifies dependency preservation after rebuild (checking the `dependencies` table), which is a meaningful data-integrity concern for a rebuild operation. V2's "overwrites valid existing cache" test is the most realistic -- it creates a cache via `list`, manually edits the JSONL to add a task, then verifies rebuild picks up the new task. This directly tests the rebuild use case described in the spec ("after manual JSONL edits").

**Verbose logging architecture:** V2 has the most correct verbose logging design. All 7 `logVerbose()` calls are inside `ForceRebuild()` in the store layer, meaning they fire at the exact moment each step occurs. V3's approach of emitting verbose messages from the CLI handler before calling `store.Rebuild()` is architecturally flawed -- messages like "lock acquire exclusive" and "delete existing cache.db" fire before those operations actually happen, and would still appear even if the store operation fails immediately.

**Store design:** V2 and V3 both use the per-operation cache model (open/close per call), which is cleaner than V1's stateful cache that requires explicit close-and-reassign in `ForceRebuild`. However, V2 duplicates lock boilerplate across `Mutate`, `Query`, and `ForceRebuild` (creating `flock.New()` each time), which is worse DRY than V1's shared `acquireExclusiveLock()` helper.

**Minor deductions for V2:** Lock acquisition code is duplicated (not extracted to a helper). The `FormatCfg` field is exported on `App` (capitalized), which is unusual for an internal package field.

**V3 runner-up notes:** V3 uniquely adds `rebuild` to the help text, which is a nice completeness touch. V3's SHA256 hash length check (`len(hash) != 64`) in the test is a stronger assertion than V2's simple non-empty check. However, V3's CLI-level verbose logging is a genuine design flaw that outweighs these advantages.

**V1 is weakest:** Fewest tests (missing hash update and lock verification), smallest test surface area, and the stateful cache model creates fragility in the `ForceRebuild` implementation.
