# Task tick-core-1-3: SQLite Cache with Freshness Detection

## Task Summary

This task implements SQLite as an auto-rebuilding cache layer over the JSONL source of truth. The cache is expendable, gitignored, and always rebuildable. It requires:

- SQLite schema with 3 tables (`tasks`, `dependencies`, `metadata`) and 3 indexes (`idx_tasks_status`, `idx_tasks_priority`, `idx_tasks_parent`)
- Cache initialization: create `cache.db` at `.tick/cache.db` with all tables/indexes
- Full rebuild from `[]Task`: clear all rows, insert all tasks and dependencies in a single transaction, store SHA256 hash of raw JSONL content
- Freshness check: compare SHA256 of JSONL bytes against stored hash in `metadata`
- `EnsureFresh` gatekeeper: checks freshness, triggers rebuild if stale, no-ops if fresh
- Handle missing `cache.db`: create from scratch + full rebuild
- Handle corrupted `cache.db`: delete, recreate, rebuild; log warning but don't fail
- Use `github.com/mattn/go-sqlite3` driver
- Hash via `crypto/sha256` on raw file bytes

**Acceptance Criteria:**
1. SQLite schema matches spec exactly (3 tables, 3 indexes)
2. Full rebuild from `[]Task` populates tasks and dependencies tables correctly
3. SHA256 hash of JSONL content stored in metadata table after rebuild
4. Freshness check correctly identifies fresh vs stale cache
5. Missing cache.db triggers automatic creation and rebuild
6. Corrupted cache.db is deleted, recreated, and rebuilt without failing the operation
7. Empty task list handled (zero rows, hash still stored)
8. Rebuild is transactional (all-or-nothing within single SQLite transaction)

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Schema matches spec (3 tables, 3 indexes) | PASS -- Schema DDL identical to spec, verified by test | PASS -- Schema DDL identical to spec, verified by test |
| Full rebuild populates tasks and dependencies | PASS -- Rebuild inserts all fields, dependencies normalized | PASS -- Rebuild inserts all fields, dependencies normalized |
| SHA256 hash stored in metadata after rebuild | PASS -- Hash computed and stored in transaction | PASS -- Hash computed and stored in transaction |
| Freshness check identifies fresh vs stale | PASS -- IsFresh compares hashes, missing hash returns stale | PASS -- IsFresh compares hashes, missing hash returns stale |
| Missing cache.db triggers creation + rebuild | PASS -- EnsureFresh creates via NewCache, then rebuilds | PASS -- EnsureFresh creates via Open, then rebuilds |
| Corrupted cache.db deleted, recreated, rebuilt | PASS -- EnsureFresh handles both Open errors and query errors | PASS -- EnsureFresh handles both Open errors and query errors via recoverAndRebuild |
| Empty task list handled | PASS -- Tested with empty slice and empty bytes | PASS -- Tested with empty slice and empty bytes |
| Rebuild is transactional | PASS -- Uses tx.Begin/Commit with defer tx.Rollback; tested with duplicate-ID constraint violation | PASS -- Uses tx.Begin/Commit with defer tx.Rollback; tested via consistency check only |

## Implementation Comparison

### Approach

**Package Placement:**
- V2: `internal/storage/cache.go` -- cache lives alongside the JSONL reader in the same `storage` package
- V4: `internal/cache/cache.go` -- dedicated `cache` package, separate from storage concerns

V4's separate package is a better separation of concerns. The cache is a distinct layer with its own lifecycle and API. However, V2's co-location in `storage` has the practical advantage that later code in the same package can access unexported `db` fields directly.

**Constructor:**
- V2 (`NewCache`): stores `path` on the struct; ensures parent directory exists via `os.MkdirAll`
- V4 (`Open`): does not store `path`; does not create parent directories

```go
// V2: NewCache stores path, creates parent dir
func NewCache(path string) (*Cache, error) {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    db, err := sql.Open("sqlite3", path)
    // ...
    return &Cache{db: db, path: path}, nil
}

// V4: Open does not store path or create dirs
func Open(path string) (*Cache, error) {
    db, err := sql.Open("sqlite3", path)
    // ...
    return &Cache{db: db}, nil
}
```

V2 storing `path` on the struct is more forward-looking (useful for corruption recovery within a Cache method), though neither version uses it that way in this task. V2's `MkdirAll` call is a practical convenience that prevents "directory not found" errors, while V4 relies on the caller to ensure directories exist.

**EnsureFresh Return Type:**
- V2: `EnsureFresh(cachePath, tasks, jsonlContent) (*Cache, error)` -- returns the Cache for subsequent queries
- V4: `EnsureFresh(dbPath, jsonlData, tasks) error` -- returns only error, no Cache handle

```go
// V2: Returns cache handle for reuse
func EnsureFresh(cachePath string, tasks []task.Task, jsonlContent []byte) (*Cache, error) {
    // ...
    return cache, nil
}

// V4: Returns only error
func EnsureFresh(dbPath string, jsonlData []byte, tasks []task.Task) error {
    c, err := Open(dbPath)
    // ...
    defer c.Close()
    return c.Rebuild(tasks, jsonlData)
}
```

V2's approach is **genuinely better** -- by returning the `*Cache`, callers can use the same open connection for subsequent queries without re-opening. V4 opens the database, ensures freshness, then closes it immediately. This means every operation that needs the cache must open it again. The spec says EnsureFresh is "the gatekeeper called on every operation," implying the cache should remain usable afterward.

**V4-only feature: `DB()` accessor:**
```go
// V4 only
func (c *Cache) DB() *sql.DB {
    return c.db
}
```
V4 exports the underlying `*sql.DB` via a public accessor. V2 accesses `cache.db` directly from tests (since they're in the same package). The `DB()` accessor in V4 is needed because tests are in a separate package conceptually, but since `cache_test.go` uses `package cache` (not `package cache_test`), it can actually access unexported fields too. The tests do use `c.DB()` throughout, which is cleaner but adds unnecessary API surface.

**Corruption Recovery:**
- V2: Inline recovery in `EnsureFresh` -- calls `os.Remove(cachePath)` then `NewCache(cachePath)` directly
- V4: Extracted `recoverAndRebuild` helper with `log.Printf` warning

```go
// V4: Dedicated recovery function with logging
func recoverAndRebuild(dbPath string, jsonlData []byte, tasks []task.Task) error {
    log.Printf("warning: cache at %s appears corrupted, rebuilding from scratch", dbPath)
    if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove corrupted cache: %w", err)
    }
    c, err := Open(dbPath)
    // ...
}
```

V4 is **genuinely better** here: (1) the spec explicitly says "Log a warning but don't fail the operation" and V4 does this with `log.Printf` while V2 has no logging at all; (2) V4 checks for `os.IsNotExist` on the Remove call which is more robust; (3) V4 extracts the recovery logic into a named helper function.

**Rebuild -- Nullable Field Handling:**

This is a **critical difference** driven by the underlying Task model:

- V2's Task model uses `string` for `Created`, `Updated`, `Closed` -- timestamps are raw strings
- V4's Task model uses `time.Time` for `Created`/`Updated` and `*time.Time` for `Closed`

```go
// V2: Passes all fields directly, no null handling
for _, t := range tasks {
    _, err := taskStmt.Exec(t.ID, t.Title, string(t.Status), t.Priority,
        t.Description, t.Parent, t.Created, t.Updated, t.Closed)
    // ...
}

// V4: Explicit null handling for optional fields
for _, t := range tasks {
    var desc, parent, closed *string
    if t.Description != "" {
        desc = &t.Description
    }
    if t.Parent != "" {
        parent = &t.Parent
    }
    if t.Closed != nil {
        s := t.Closed.UTC().Format("2006-01-02T15:04:05Z")
        closed = &s
    }
    created := t.Created.UTC().Format("2006-01-02T15:04:05Z")
    updated := t.Updated.UTC().Format("2006-01-02T15:04:05Z")
    // ...
}
```

V4's null-aware approach is **genuinely better** for data correctness. V2 inserts empty strings `""` for optional fields (Description, Parent, Closed) instead of SQL NULLs. This means V2's cache stores `description = ""` instead of `description = NULL`, which could cause subtle bugs in queries like `WHERE description IS NULL`. V4 correctly inserts NULL for empty optional fields.

V4 also explicitly formats `time.Time` values into ISO 8601 strings for SQLite storage, while V2 relies on Go's default `time.Time` → string conversion (or in this case, already-string fields being passed through).

**Hash Computation:**
```go
// V2: Uses encoding/hex
hash := sha256.Sum256(jsonlContent)
hashStr := hex.EncodeToString(hash[:])

// V4: Uses fmt.Sprintf
h := sha256.Sum256(data)
return fmt.Sprintf("%x", h)
```

Functionally equivalent. V2's `hex.EncodeToString` is marginally more explicit about what's happening; V4's `fmt.Sprintf` is more concise. No meaningful difference.

**Parameter Ordering in EnsureFresh:**
- V2: `EnsureFresh(cachePath, tasks, jsonlContent)` -- tasks before content
- V4: `EnsureFresh(dbPath, jsonlData, tasks)` -- content before tasks

The spec says EnsureFresh "takes raw JSONL bytes and parsed []Task," which matches V4's ordering better. Different but equivalent.

### Code Quality

**Error Handling:**
- V2: Returns raw errors without wrapping (e.g., `return err` throughout Rebuild)
- V4: Wraps every error with context using `fmt.Errorf("...: %w", err)`

```go
// V2: Raw errors
func (c *Cache) Rebuild(tasks []task.Task, jsonlContent []byte) error {
    tx, err := c.db.Begin()
    if err != nil {
        return err  // no context
    }
    if _, err := tx.Exec(`DELETE FROM tasks`); err != nil {
        return err  // no context
    }
    // ...
}

// V4: Wrapped errors with context
func (c *Cache) Rebuild(tasks []task.Task, jsonlData []byte) error {
    tx, err := c.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin rebuild transaction: %w", err)
    }
    if _, err := tx.Exec("DELETE FROM dependencies"); err != nil {
        return fmt.Errorf("failed to clear dependencies: %w", err)
    }
    // ...
}
```

V4 is **genuinely better** here. Error wrapping is a Go best practice for debugging. When an error propagates through multiple layers, V4's messages tell you exactly where the failure occurred (e.g., "failed to insert task tick-a1b2c3: UNIQUE constraint failed").

**Go Naming Conventions:**
- V2: `NewCache`, `TestCacheSchema`, `TestCacheRebuild` -- standard Go constructor naming
- V4: `Open`, `TestCache_CreateSchema`, `TestCache_Rebuild` -- uses underscores in test names for readability

V4's `Open` follows the pattern of `sql.Open`, `os.Open` etc. which is slightly more idiomatic for a function that opens an existing resource. V2's `NewCache` implies construction which is also appropriate. V4's underscore-separated test names (`TestCache_Rebuild`) are more readable but less common in Go.

**Package Documentation:**
- V2: `// Package storage provides JSONL file storage and SQLite cache for tasks.` -- reuses existing package doc
- V4: `// Package cache provides SQLite-based caching for task data. / The cache is expendable and always rebuildable...` -- dedicated multi-line doc

V4's dedicated package doc is more informative.

**DRY:**
- V4: Extracts `computeSHA256` helper and `recoverAndRebuild` helper; test file has shared `sampleTasks()` and `sampleJSONLContent()` fixtures
- V2: Hash computation is inline (2 lines duplicated in Rebuild and IsFresh); test data is constructed inline in each test

V4 is more DRY. V2 repeats task construction boilerplate across many tests.

**Type Safety:**
- V2's Task model uses `string` for all timestamps, which means the cache layer doesn't need to do any time parsing/formatting -- timestamps are opaque strings passed through. This is simpler but loses type safety.
- V4's Task model uses `time.Time`, which requires explicit formatting in the cache layer but provides compile-time type safety and prevents malformed timestamps from entering the system.

### Test Quality

**V4 Test Functions (4 top-level, 14 subtests):**

Top-level:
1. `TestCache_CreateSchema`
2. `TestCache_Rebuild`
3. `TestCache_Freshness`
4. `TestCache_EnsureFresh`

Subtests:
1. "it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)" -- verifies 3 tables, 3 indexes, column names for all tables
2. "it rebuilds cache from parsed tasks -- all fields round-trip correctly" -- uses 3 sample tasks, reads all back, verifies ID/Title/Status/Priority/Description/Parent/Closed on multiple tasks
3. "it normalizes blocked_by array into dependencies table rows" -- queries dependencies for a task with 2 blockers, checks total count
4. "it stores JSONL content hash in metadata table after rebuild" -- computes expected hash, compares with stored
5. "it handles empty task list (zero rows, hash still stored)" -- 0 tasks, 0 deps, hash present
6. "it replaces all existing data on rebuild (no stale rows)" -- rebuilds from 3 tasks to 1 task, verifies old data gone, hash updated
7. "it rebuilds within a single transaction (all-or-nothing)" -- verifies consistency after two sequential rebuilds (hash + data match). Note: does NOT test rollback on failure
8. "it detects fresh cache (hash matches) and skips rebuild" -- rebuild then IsFresh with same data
9. "it detects stale cache (hash mismatch) and triggers rebuild" -- rebuild then IsFresh with different data
10. "it rebuilds from scratch when cache.db is missing" -- EnsureFresh on non-existent path
11. "it skips rebuild when cache is fresh" -- EnsureFresh twice with same data
12. "it rebuilds when cache is stale" -- EnsureFresh with changed data, verifies new task count
13. "it deletes and recreates cache.db when corrupted" -- writes garbage bytes, EnsureFresh recovers

**V2 Test Functions (7 top-level, 14 subtests):**

Top-level:
1. `TestCacheSchema`
2. `TestCacheRebuild`
3. `TestCacheFreshness`
4. `TestEnsureFresh`
5. `TestCacheEmptyTasks`
6. `TestCacheReplaceAllData`
7. `TestCacheTransaction`

Subtests:
1. "it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)" -- verifies 3 tables with column types, 3 indexes. Also checks column types (TEXT, INTEGER) not just names.
2. "it rebuilds cache from parsed tasks -- all fields round-trip correctly" -- single task with all fields including multiline description, verifies every field individually
3. "it normalizes blocked_by array into dependencies table rows" -- task with 3 blockers (not 2), verifies order and count
4. "it stores JSONL content hash in metadata table after rebuild" -- hardcodes expected hash value
5. "it detects fresh cache (hash matches) and skips rebuild"
6. "it detects stale cache (hash mismatch) and triggers rebuild"
7. "it treats missing hash as stale" -- **unique to V2**: explicitly tests IsFresh on a fresh cache with no prior rebuild
8. "it rebuilds from scratch when cache.db is missing"
9. "it skips rebuild when cache is fresh" -- **uses marker row technique**: inserts a marker into metadata, verifies it survives EnsureFresh (proving no rebuild happened)
10. "it triggers rebuild when cache is stale" -- verifies old task gone, new task present
11. "it deletes and recreates cache.db when corrupted"
12. "it handles empty task list (zero rows, hash still stored)"
13. "it replaces all existing data on rebuild (no stale rows)" -- 3 tasks to 1 task, checks old IDs don't exist
14. "it rebuilds within a single transaction (all-or-nothing)" -- **uses duplicate-ID constraint violation** to force a rollback, then verifies original data is intact

**Test Gap Analysis:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Missing hash treated as stale | Dedicated test | Implicitly covered by EnsureFresh "missing" test |
| Rollback on rebuild failure (true transactionality proof) | TESTED via duplicate-ID insertion | NOT TESTED -- only checks consistency after successful rebuild |
| Marker-based skip-rebuild verification | TESTED (metadata marker survives) | Not tested -- only checks task count |
| Schema column types (not just names) | TESTED (checks TEXT vs INTEGER) | Not tested -- only checks column existence |
| Multiline description round-trip | TESTED (`"Full description with\nmultiple lines"`) | Not tested |
| 3 blockers (vs 2) | TESTED | 2 blockers tested |
| Hardcoded expected hash | TESTED (verifies exact SHA256 output) | Computed dynamically (less brittle but less explicit) |
| Null vs empty string for optional fields | Not tested (V2 stores empty strings, not NULLs) | Implicitly handled by *string nil checks in Rebuild |

V2's transaction test is **genuinely better** -- it actually proves rollback works by forcing a constraint violation:

```go
// V2: Forces rollback by inserting duplicate IDs
badTasks := []task.Task{
    {ID: "tick-newone", Title: "Task 1", ...},
    {ID: "tick-newone", Title: "Task 2 (duplicate ID)", ...},
}
err = cache.Rebuild(badTasks, []byte("bad"))
if err == nil {
    t.Fatal("expected error for duplicate ID")
}
// Original data should still be intact due to transaction rollback
var title string
err = cache.db.QueryRow(`SELECT title FROM tasks WHERE id = 'tick-a1b2c3'`).Scan(&title)
```

V2's "skip rebuild when cache is fresh" test is also more rigorous -- the marker technique provides proof-positive that Rebuild was not called:

```go
// V2: Inserts marker, verifies it survives EnsureFresh
_, err = cache2.db.Exec(`INSERT INTO metadata (key, value) VALUES ('marker', 'test')`)
// ... call EnsureFresh ...
var marker string
err = cache3.db.QueryRow(`SELECT value FROM metadata WHERE key = 'marker'`).Scan(&marker)
// Marker survives = no rebuild happened
```

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 7 | 6 |
| Lines added | 966 | 960 |
| Impl LOC | 196 | 214 |
| Test LOC | 748 | 739 |
| Top-level test functions | 7 | 4 |
| Test subtests | 14 | 13 |

V2 also adds a `docs/workflow/implementation/tick-core-context.md` file documenting integration notes, which V4 does not.

## Verdict

**V2 is the stronger implementation overall, despite V4 having better code quality in the implementation file.**

The analysis breaks down into two distinct dimensions:

**V4 wins on implementation code quality:**
- Proper error wrapping with `fmt.Errorf("...: %w", err)` throughout
- Correct NULL handling for optional fields (uses `*string` nil pointers instead of empty strings)
- Extracted `recoverAndRebuild` helper with `log.Printf` warning (spec requirement)
- Dedicated `cache` package with better separation of concerns
- DRY test fixtures (`sampleTasks()`, `sampleJSONLContent()`)
- `time.Time` type safety in the Task model (though this is a task-1-1 decision, not a task-1-3 decision)

**V2 wins on API design and test rigor:**
- `EnsureFresh` returns `*Cache` for reuse, avoiding unnecessary re-opens
- Transaction test actually proves rollback works via constraint violation (V4's transaction test only checks post-success consistency)
- "Skip rebuild" test uses marker technique to prove rebuild was not called
- Schema test verifies column types, not just column existence
- Tests multiline description round-trip
- Dedicated "missing hash treated as stale" test
- Hardcoded expected hash value provides regression protection
- Integration context documentation

The V2 `EnsureFresh` returning `*Cache` is the most consequential API difference -- it avoids the pattern where every caller must separately open the cache after ensuring freshness, which V4 would require. V2's transaction test is also significantly more rigorous, actually proving the "all-or-nothing" claim rather than merely checking consistency after success.

However, V2's failure to handle NULLs correctly in optional fields (storing `""` instead of `NULL`) is a real bug that could cause issues in later tasks when querying `WHERE description IS NULL` or similar. V4 handles this correctly.

**Net assessment: Close call. V2 has the better API and tests; V4 has better implementation hygiene. Neither is clearly superior across all dimensions.**
