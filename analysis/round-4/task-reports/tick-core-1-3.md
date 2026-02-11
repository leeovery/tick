# Task tick-core-1-3: SQLite cache with freshness detection

## Task Summary

This task requires implementing a SQLite-based query cache for Tick tasks that auto-rebuilds from the JSONL source of truth using SHA256 hash-based freshness detection. Specifically:

- Define a SQLite schema with three tables (`tasks`, `dependencies`, `metadata`) and three indexes (`idx_tasks_status`, `idx_tasks_priority`, `idx_tasks_parent`)
- Implement cache initialization: create `cache.db` at `.tick/cache.db`, create all tables/indexes if not present
- Implement full rebuild from JSONL: accept `[]Task` slice, clear all rows, insert all tasks and dependencies in a single transaction, compute and store SHA256 hash of raw JSONL file content in `metadata` as key `jsonl_hash`
- Implement freshness check: compare SHA256 hash of JSONL content with stored hash
- Implement `EnsureFresh`: checks freshness, triggers rebuild if stale, no-ops if fresh
- Handle missing `cache.db`: create from scratch and rebuild
- Handle corrupted `cache.db`: delete, recreate, and rebuild; log warning but do not fail
- Use `github.com/mattn/go-sqlite3` as the SQLite driver
- Hash computation: `crypto/sha256` from Go stdlib on raw file bytes

### Acceptance Criteria

1. SQLite schema matches spec exactly (3 tables, 3 indexes)
2. Full rebuild from `[]Task` populates tasks and dependencies tables correctly
3. SHA256 hash of JSONL content stored in metadata table after rebuild
4. Freshness check correctly identifies fresh vs stale cache
5. Missing cache.db triggers automatic creation and rebuild
6. Corrupted cache.db is deleted, recreated, and rebuilt without failing the operation
7. Empty task list handled (zero rows, hash still stored)
8. Rebuild is transactional (all-or-nothing within single SQLite transaction)

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| SQLite schema matches spec (3 tables, 3 indexes) | PASS - Schema constant defines all 3 tables and 3 indexes exactly per spec | PASS - Schema constant defines all 3 tables and 3 indexes exactly per spec |
| Full rebuild populates tasks and dependencies correctly | PASS - Rebuild clears and re-inserts all tasks and deps in a transaction | PASS - Rebuild clears and re-inserts all tasks and deps in a transaction |
| SHA256 hash stored in metadata after rebuild | PASS - Hash stored via `INSERT INTO metadata` with key `jsonl_hash` | PASS - Hash stored via `INSERT OR REPLACE INTO metadata` with key `jsonl_hash` |
| Freshness check identifies fresh vs stale | PASS - `IsFresh` compares computed hash with stored hash, returns false on `ErrNoRows` | PASS - `IsFresh` compares computed hash with stored hash, returns false on `ErrNoRows` |
| Missing cache.db triggers creation and rebuild | PASS - `EnsureFresh` calls `New` which opens/creates the file, then rebuilds | PASS - `EnsureFresh` calls `OpenCache` which opens/creates the file, then rebuilds |
| Corrupted cache.db deleted, recreated, rebuilt without failing | PASS - Two-phase recovery: catches open errors and query errors, deletes and recreates | PASS - Two-phase recovery: catches open errors and query errors, deletes and recreates |
| Empty task list handled (zero rows, hash stored) | PASS - Tested explicitly with `[]task.Task{}` and `[]byte{}` | PASS - Tested explicitly with `nil` tasks and `[]byte("")` |
| Rebuild is transactional (all-or-nothing) | PASS - Uses `tx.Begin()`, deferred `tx.Rollback()`, and `tx.Commit()` | PASS - Uses `tx.Begin()`, deferred `tx.Rollback()`, and `tx.Commit()` |

## Implementation Comparison

### Approach

Both versions implement structurally identical solutions with the same overall architecture: a `Cache` struct wrapping `*sql.DB` and a path, a constructor (`New` / `OpenCache`), `Rebuild`, `IsFresh`, and a package-level `EnsureFresh` function. The differences are in package placement, naming conventions, and minor implementation details.

**Package placement:**

- V5 places the cache in its own package `internal/cache/cache.go`. This creates a dedicated package with a clear single responsibility.
- V6 places it in the existing `internal/storage/cache.go` package alongside the JSONL reader. This co-locates all persistence concerns, and V6 also updates the `jsonl.go` package doc from `"JSONL persistence for Tick tasks"` to `"JSONL persistence and SQLite cache management for Tick tasks"`.

Both approaches are valid. V5's separate package provides cleaner import boundaries but adds an extra package. V6's co-location follows a "storage layer" organizational pattern, which is arguably more cohesive since both JSONL and SQLite are storage concerns.

**Constructor naming:**

- V5: `New(dbPath string) (*Cache, error)` -- idiomatic Go for the primary constructor of a package
- V6: `OpenCache(path string) (*Cache, error)` -- necessary because `New` would be ambiguous in the shared `storage` package which already has JSONL concerns

**Schema constant naming:**

- V5: `const schema = ...`
- V6: `const schemaSQL = ...`

V6's name is slightly more descriptive.

**Rebuild metadata handling:**

- V5 uses `DELETE FROM metadata` to clear all metadata, then `INSERT INTO metadata`:
```go
if _, err := tx.Exec("DELETE FROM metadata"); err != nil {
    return fmt.Errorf("clearing metadata: %w", err)
}
// ...later...
if _, err := tx.Exec(`INSERT INTO metadata (key, value) VALUES ('jsonl_hash', ?)`, hash); err != nil {
```

- V6 skips the metadata delete and uses `INSERT OR REPLACE INTO metadata`:
```go
if _, err := tx.Exec(
    `INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)`,
    hash,
); err != nil {
```

V6's approach is marginally better: it avoids a separate DELETE step for metadata and handles the upsert atomically in a single statement. V5's approach of deleting ALL metadata rows first is broader than necessary -- it would wipe any non-hash metadata keys if they existed. In practice, the spec only defines one key (`jsonl_hash`), so this difference is inconsequential.

**Hash encoding:**

- V5: `fmt.Sprintf("%x", h)` -- uses `fmt` formatting on the `[32]byte` array directly
- V6: `hex.EncodeToString(h[:])` -- uses the dedicated `encoding/hex` package

```go
// V5
func computeHash(data []byte) string {
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h)
}

// V6
func computeHash(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
```

V6's approach is slightly more correct idiomatically -- `encoding/hex` is purpose-built for hex encoding, avoids the overhead of format string parsing, and requires the explicit `h[:]` slice conversion. V5's `%x` on a `[32]byte` works correctly but `Sprintf` has more overhead. Both produce identical output.

**Nullable field handling in Rebuild:**

Both versions handle nullable fields (`description`, `parent`, `closed`) identically by converting empty strings to `nil` pointers:

```go
// V5
var description, parent, closed *string
if t.Description != "" {
    description = &t.Description
}
if t.Parent != "" {
    parent = &t.Parent
}
if t.Closed != nil {
    s := task.FormatTimestamp(*t.Closed)
    closed = &s
}

// V6
var closedStr *string
if t.Closed != nil {
    s := task.FormatTimestamp(*t.Closed)
    closedStr = &s
}
var parentStr *string
if t.Parent != "" {
    parentStr = &t.Parent
}
var descStr *string
if t.Description != "" {
    descStr = &t.Description
}
```

V5 groups all declarations together then assigns; V6 declares each near its assignment. V6's pattern is slightly more readable for longer functions. Both are equivalent.

**EnsureFresh parameter order:**

- V5: `EnsureFresh(dbPath string, tasks []task.Task, jsonlData []byte) (*Cache, error)`
- V6: `EnsureFresh(dbPath string, rawJSONL []byte, tasks []task.Task) (*Cache, error)`

V6 puts raw bytes before tasks. V5 puts tasks before raw bytes. Neither order has a clear advantage.

**EnsureFresh corruption handling:**

Both use an identical two-phase approach. The only difference is that V5 factors the remove-and-recreate logic into a helper:

```go
// V5
func recreate(dbPath string) (*Cache, error) {
    if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("removing corrupt cache: %w", err)
    }
    c, err := New(dbPath)
    if err != nil {
        return nil, fmt.Errorf("recreating cache: %w", err)
    }
    return c, nil
}
```

V6 inlines the removal:
```go
// V6
os.Remove(dbPath)
cache, err = OpenCache(dbPath)
```

V5's `recreate` helper is more defensive -- it checks `os.Remove` errors (excluding `ErrNotExist`). V6 silently ignores `os.Remove` errors entirely. In practice, if `os.Remove` fails for a reason other than "not exists" (e.g., permissions), V6 would attempt `OpenCache` on the still-corrupt file and likely fail there, so the error would still surface. But V5 is more explicit about the failure point.

**Close() nil check:**

- V5 checks for nil before closing:
```go
func (c *Cache) Close() error {
    if c.db != nil {
        return c.db.Close()
    }
    return nil
}
```

- V6 calls Close unconditionally:
```go
func (c *Cache) Close() error {
    return c.db.Close()
}
```

V5 is more defensive. V6 would panic if `c.db` were nil, though in practice this cannot happen since the constructor always initializes `db` or returns an error.

### Code Quality

**Error messages:**

- V5: Terse, consistent prefix style: `"opening cache database"`, `"initializing cache schema"`, `"beginning rebuild transaction"`
- V6: Verbose, `"failed to"` prefix style: `"failed to open cache database"`, `"failed to create cache schema"`, `"failed to begin rebuild transaction"`

Go convention per the standard library and the [Go blog](https://go.dev/blog/error-handling-and-go) is to NOT prefix error messages with "failed to" since the caller's `fmt.Errorf` wrapping makes that implicit. V5's style is more idiomatic.

```go
// V5 (idiomatic)
return nil, fmt.Errorf("opening cache database: %w", err)

// V6 (verbose)
return nil, fmt.Errorf("failed to open cache database: %w", err)
```

**Package documentation:**

- V5: `// Package cache provides a SQLite-based query cache for tasks that auto-rebuilds from the JSONL source of truth using SHA256 freshness detection.`
- V6: No dedicated package doc for cache (it's in the `storage` package, which has its doc on `jsonl.go`)

V5 has a complete, descriptive package doc. V6 updated the existing `storage` package doc on `jsonl.go` to include cache concerns.

**Exported function documentation:**

Both versions document all exported functions. V5 provides slightly more detailed doc comments:

```go
// V5
// EnsureFresh opens the cache at dbPath, checks freshness against the given
// JSONL content, and triggers a full rebuild if stale or missing. If the cache
// file is corrupted, it is deleted, recreated, and rebuilt.

// V6
// EnsureFresh checks if the cache is up-to-date with the given JSONL content and tasks.
// If stale (or on any error), it rebuilds the cache. If fresh, it no-ops.
```

**DRY principle:**

V5 extracts the `recreate` helper, called twice in `EnsureFresh`. V6 duplicates the `os.Remove` + `OpenCache` pattern inline twice. V5 is DRYer.

**Prepared statement handling:**

Both versions correctly use prepared statements for batch inserts within the transaction, with proper `defer stmt.Close()`. This is correct and efficient Go database usage.

### Test Quality

**V5 Test Functions (1 top-level, 11 subtests):**

Top-level: `TestCache`
1. `"it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)"` -- verifies 3 tables, 3 indexes, all task columns, dependency columns
2. `"it rebuilds cache from parsed tasks - all fields round-trip correctly"` -- round-trips all 9 task fields including nullable fields, checks closed task, verifies 3 total rows
3. `"it normalizes blocked_by array into dependencies table rows"` -- verifies 2 deps for task with `blocked_by`, verifies 0 deps for task without
4. `"it stores JSONL content hash in metadata table after rebuild"` -- computes expected hash, queries metadata table
5. `"it detects fresh cache (hash matches) and skips rebuild"` -- rebuilds, then checks IsFresh with same data
6. `"it detects stale cache (hash mismatch) and triggers rebuild"` -- rebuilds, then checks IsFresh with different data
7. `"it rebuilds from scratch when cache.db is missing"` -- calls EnsureFresh with non-existent path, verifies 3 tasks inserted
8. `"it deletes and recreates cache.db when corrupted"` -- writes garbage to file, calls EnsureFresh, verifies recovery
9. `"it handles empty task list (zero rows, hash still stored)"` -- rebuilds with empty slice and empty bytes, verifies 0 tasks and hash stored
10. `"it replaces all existing data on rebuild (no stale rows)"` -- rebuilds twice (3 tasks then 1 task), verifies only 1 task remains and 0 deps
11. `"it rebuilds within a single transaction (all-or-nothing)"` -- rebuilds valid data, then attempts rebuild with duplicate IDs, verifies original 3 tasks preserved

**V6 Test Functions (5 top-level, 11 subtests):**

Top-level: `TestCacheSchema`, `TestCacheRebuild`, `TestCacheFreshness`, `TestEnsureFresh`, `TestCacheEdgeCases`

Under `TestCacheSchema`:
1. `"it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)"` -- verifies tables, columns (with count comparison), indexes via helper functions

Under `TestCacheRebuild`:
2. `"it rebuilds cache from parsed tasks -- all fields round-trip correctly"` -- round-trips all 9 fields including closed timestamp
3. `"it normalizes blocked_by array into dependencies table rows"` -- verifies 2 deps with both columns scanned
4. `"it stores JSONL content hash in metadata table after rebuild"` -- uses `nil` tasks (no task data), just verifies hash

Under `TestCacheFreshness`:
5. `"it detects fresh cache (hash matches) and skips rebuild"` -- same approach as V5
6. `"it detects stale cache (hash mismatch) and triggers rebuild"` -- same approach as V5

Under `TestEnsureFresh`:
7. `"it rebuilds from scratch when cache.db is missing"` -- verifies 1 task inserted
8. `"it deletes and recreates cache.db when corrupted"` -- same approach as V5, verifies 1 task

Under `TestCacheEdgeCases`:
9. `"it handles empty task list (zero rows, hash still stored)"` -- uses `nil` tasks and empty bytes, also checks 0 deps explicitly
10. `"it replaces all existing data on rebuild (no stale rows)"` -- rebuilds with task A (with dep), then task C (no dep), verifies only C remains
11. `"it rebuilds within a single transaction (all-or-nothing)"` -- same approach as V5, additionally verifies original task ID is preserved

**Test structure differences:**

V5 uses a single `TestCache` parent function with all 11 subtests flat underneath. This is simpler but less organized.

V6 uses 5 top-level test functions grouped by concern: schema, rebuild, freshness, EnsureFresh, edge cases. Each contains related subtests. This provides better test organization and makes it easier to run specific test groups.

**Test helper differences:**

V5 uses `sampleTasks()` and `sampleJSONLContent()` helper functions to provide reusable test fixtures shared across many tests. This reduces duplication.

V6 constructs test data inline for each test. This makes each test self-contained but results in more code. V6 also extracts `queryColumns` and `queryIndexes` as reusable test helpers for schema verification with `t.Helper()` annotations.

**Edge case coverage differences:**

Both versions test all 11 specified test cases. Specific differences:

- V5's empty task test passes `[]task.Task{}` (empty slice); V6 passes `nil` (nil slice). Both are valid in Go but V6's `nil` is a slightly different edge case.
- V6's schema test verifies column *count* matches expected (catches extra columns): `if len(taskCols) != len(expectedTaskCols)`. V5 only checks expected columns exist but would not catch extra unexpected columns.
- V6's "replaces all existing data" test verifies the remaining task's `id` is the expected one, not just that the count is correct. V5 checks count and dependency count but not the specific remaining task ID.
- V6's "transaction" test verifies the preserved task ID after rollback; V5 only checks the count.
- V6's empty task test additionally verifies 0 dependency rows; V5 only checks tasks and hash.
- V5's round-trip test checks a second task's closed timestamp explicitly AND verifies the first task's closed IS null. V6's round-trip test only checks one task with all fields populated including closed.
- V5 uses 3 sample tasks (open, done, in_progress) giving broader coverage of status values in a single test. V6 uses 1 task per test.

**Test gaps:**

Neither version tests:
- Concurrent access to the cache (race conditions)
- Very large task sets (performance)
- Unicode in task fields (though this is handled by SQLite transparently)
- The `DB()` accessor returning a usable connection

V5 has a minor advantage in that its sample data covers more task states in the round-trip test. V6 has a minor advantage in verifying specific IDs after operations and checking column counts.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint | PASS - code formatting is consistent with gofmt | PASS - code formatting is consistent with gofmt |
| Handle all errors explicitly | PASS - every error is checked and wrapped | PASS - every error is checked and wrapped (except `os.Remove` in `EnsureFresh`) |
| Write table-driven tests with subtests | PARTIAL - uses subtests but not table-driven format; tests are individual subtests | PARTIAL - uses subtests but not table-driven format; tests are individual subtests |
| Document all exported functions/types/packages | PASS - all exports documented | PASS - all exports documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS - all errors wrapped with %w | PASS - all errors wrapped with %w |
| Do not ignore errors without justification | PASS - only `tx.Rollback()` is ignored (justified by deferred cleanup) | PARTIAL - `os.Remove(dbPath)` return value ignored in `EnsureFresh` without comment |
| No panic for normal error handling | PASS - no panics | PASS - no panics |
| No reflection without justification | PASS - no reflection used | PASS - no reflection used |

### Spec-vs-Convention Conflicts

**Error message style:** The spec does not prescribe error message format. V5 follows Go convention (lowercase, no "failed to" prefix). V6 uses "failed to" prefix which is more readable for end users but less idiomatic in Go libraries. This is a convention-only difference with no spec conflict.

**Table-driven tests:** The skill mandates table-driven tests with subtests. Neither version uses table-driven tests -- both use individual subtests with inline assertions. However, the task spec explicitly lists 11 specific test case descriptions, each testing a unique scenario that does not lend itself well to a shared table structure (each test has different setup, different assertions). Using individual subtests matching the spec's test list is a reasonable interpretation. Neither version is penalized for this.

**Package placement:** The spec says "implement cache initialization: create `cache.db` at `.tick/cache.db`" but does not mandate a specific package name. V5's `internal/cache` and V6's `internal/storage` are both valid. The spec mentions the cache should be "auto-rebuilding" and work with the JSONL reader from tick-core-1-2. V6's co-location with the JSONL reader is slightly more aligned with this integration intent.

No spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 6 | 7 |
| Lines added | 784 | 840 |
| Impl LOC | 226 | 215 |
| Test LOC | 550 | 617 |
| Test functions | 1 top-level + 11 subtests | 5 top-level + 11 subtests |

## Verdict

Both versions are well-implemented and pass all acceptance criteria. The differences are minor and come down to style and organization rather than correctness.

**V5 is slightly better** on the following dimensions:
- More idiomatic Go error messages (no "failed to" prefix)
- DRYer code via the `recreate` helper function
- Defensive nil check in `Close()`
- Explicit error handling on `os.Remove` in the recreate path
- Richer sample data in tests (3 tasks covering multiple statuses)

**V6 is slightly better** on the following dimensions:
- Better test organization (5 top-level groups vs 1 flat list)
- Reusable test helpers with `t.Helper()` (`queryColumns`, `queryIndexes`)
- Column count verification in schema test (catches extra columns)
- `INSERT OR REPLACE` for metadata avoids unnecessary DELETE
- Co-location with JSONL reader in `storage` package is cohesive
- More precise assertions (verifying specific task IDs after operations, not just counts)
- Slightly more tests LOC (617 vs 550) with marginally broader edge case coverage in some tests

**Overall: Tie, with a slight edge to V6.** V6's superior test organization and more precise assertions slightly outweigh V5's more idiomatic error messages and DRYer implementation code. Both are production-quality implementations that fully satisfy the spec.
