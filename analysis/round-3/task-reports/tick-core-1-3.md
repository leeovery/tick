# Task tick-core-1-3: SQLite cache with freshness detection

## Task Summary

This task implements an auto-rebuilding SQLite cache at `.tick/cache.db` that mirrors the JSONL source of truth. Requirements:

- Define SQLite schema: 3 tables (`tasks`, `dependencies`, `metadata`) and 3 indexes (`idx_tasks_status`, `idx_tasks_priority`, `idx_tasks_parent`)
- `tasks` table: `id TEXT PRIMARY KEY`, `title TEXT NOT NULL`, `status TEXT NOT NULL DEFAULT 'open'`, `priority INTEGER NOT NULL DEFAULT 2`, `description TEXT`, `parent TEXT`, `created TEXT NOT NULL`, `updated TEXT NOT NULL`, `closed TEXT`
- `dependencies` table: `task_id TEXT NOT NULL`, `blocked_by TEXT NOT NULL`, `PRIMARY KEY (task_id, blocked_by)` -- normalizes `blocked_by` array
- `metadata` table: `key TEXT PRIMARY KEY`, `value TEXT` -- stores `jsonl_hash`
- Full rebuild from `[]Task`: clear all rows, insert all tasks/deps in single transaction, compute SHA256 of raw JSONL bytes, store hash in metadata
- Freshness check: compare SHA256 of JSONL content with stored hash
- `EnsureFresh`: gatekeeper called on every operation -- check freshness, rebuild if stale, no-op if fresh
- Handle missing `cache.db`: create from scratch and rebuild
- Handle corrupted `cache.db`: delete, recreate, rebuild; log warning but do not fail
- Use `github.com/mattn/go-sqlite3`, `crypto/sha256`

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

| Criterion | V4 | V5 |
|-----------|-----|-----|
| 1. SQLite schema matches spec (3 tables, 3 indexes) | PASS -- Identical schema DDL with all 3 tables and 3 indexes using `CREATE TABLE/INDEX IF NOT EXISTS` | PASS -- Identical schema DDL with all 3 tables and 3 indexes using `CREATE TABLE/INDEX IF NOT EXISTS` |
| 2. Full rebuild populates tasks and dependencies | PASS -- `Rebuild()` clears and inserts all tasks/deps with prepared statements | PASS -- `Rebuild()` clears and inserts all tasks/deps with prepared statements |
| 3. SHA256 hash stored in metadata after rebuild | PASS -- Hash computed via `computeSHA256()` and stored with `INSERT OR REPLACE INTO metadata` | PASS -- Hash computed via `computeHash()` and stored with `INSERT INTO metadata` after `DELETE FROM metadata` |
| 4. Freshness check identifies fresh vs stale | PASS -- `IsFresh()` computes hash first, compares with stored; handles `sql.ErrNoRows` as stale | PASS -- `IsFresh()` queries stored hash first, compares with computed; handles `sql.ErrNoRows` as stale |
| 5. Missing cache.db triggers creation and rebuild | PASS -- `EnsureFresh()` opens via `Open()` which creates file if missing, then rebuilds since no hash exists | PASS -- `EnsureFresh()` opens via `New()` which creates file if missing, then rebuilds since no hash exists |
| 6. Corrupted cache.db deleted, recreated, rebuilt without failing | PASS -- `recoverAndRebuild()` removes file, recreates, rebuilds; logs warning via `log.Printf` | PASS -- `EnsureFresh()` catches errors from `New()` and `IsFresh()`, calls `recreate()` which removes and recreates; logs warning via `log.Printf` |
| 7. Empty task list handled (zero rows, hash still stored) | PASS -- Tested explicitly; zero tasks, hash stored | PASS -- Tested explicitly; zero tasks, hash stored |
| 8. Rebuild is transactional | PASS -- `BEGIN`/`COMMIT` with `defer tx.Rollback()`; tested with duplicate ID causing rollback | PASS -- `BEGIN`/`COMMIT` with `defer tx.Rollback()`; tested with duplicate ID causing rollback |

## Implementation Comparison

### Approach

Both versions follow the same high-level design: a `Cache` struct wrapping `*sql.DB`, a schema constant with `IF NOT EXISTS` guards, `Rebuild()` for full data replacement in a transaction, `IsFresh()` for hash comparison, and `EnsureFresh()` as the top-level gatekeeper. The structural differences are in naming, return types, and corruption handling flow.

**Struct definition:**

V4 stores only `*sql.DB`:
```go
// V4: internal/cache/cache.go line 46-48
type Cache struct {
    db *sql.DB
}
```

V5 stores both `*sql.DB` and `path`:
```go
// V5: internal/cache/cache.go line 51-54
type Cache struct {
    db   *sql.DB
    path string
}
```
V5's `path` field is stored but never used outside the struct -- it is not referenced by any method. This is dead state. V4 avoids this by passing `dbPath` as a parameter to the functions that need it.

**Constructor naming:**

V4 uses `Open(path string) (*Cache, error)` -- idiomatic Go naming consistent with `os.Open`, `sql.Open`. V5 uses `New(dbPath string) (*Cache, error)` -- also idiomatic, following the `New` constructor convention. Both are valid Go conventions.

**EnsureFresh signature:**

V4:
```go
func EnsureFresh(dbPath string, jsonlData []byte, tasks []task.Task) error
```

V5:
```go
func EnsureFresh(dbPath string, tasks []task.Task, jsonlData []byte) (*Cache, error)
```

Two key differences:
1. **Return type**: V4 returns only `error` and manages the cache lifecycle internally (opens, uses, closes via `defer`). V5 returns `(*Cache, error)`, giving the caller ownership of the cache connection. V5's approach is more flexible -- callers can query the cache after ensuring freshness without reopening it. V4's approach requires reopening the database for any subsequent queries, which is wasteful.
2. **Parameter order**: V4 puts `jsonlData` before `tasks`; V5 puts `tasks` before `jsonlData`. Neither order is strictly better; V5 matches the `Rebuild(tasks, jsonlData)` parameter order which is slightly more consistent.

**Corruption handling:**

V4:
```go
func recoverAndRebuild(dbPath string, jsonlData []byte, tasks []task.Task) error {
    log.Printf("warning: cache at %s appears corrupted, rebuilding from scratch", dbPath)
    if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove corrupted cache: %w", err)
    }
    c, err := Open(dbPath)
    if err != nil {
        return fmt.Errorf("failed to create new cache after recovery: %w", err)
    }
    defer c.Close()
    return c.Rebuild(tasks, jsonlData)
}
```

V5:
```go
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

V5's `recreate()` is a smaller, more focused function that only handles file removal and database creation. The rebuild is done by the caller (`EnsureFresh`). This is a cleaner separation of concerns. V4's `recoverAndRebuild()` does everything in one shot, which is simpler but less composable.

V5's `EnsureFresh` has a two-level corruption catch:
```go
// V5: catches error from New() and also error from IsFresh()
c, err := New(dbPath)
if err != nil {
    log.Printf("warning: cache corrupt or unreadable, recreating: %v", err)
    c, err = recreate(dbPath)
    ...
}
fresh, err := c.IsFresh(jsonlData)
if err != nil {
    log.Printf("warning: cache query failed, recreating: %v", err)
    c.Close()
    c, err = recreate(dbPath)
    ...
}
```

V4's `EnsureFresh` also catches both errors but routes both to the same `recoverAndRebuild`:
```go
// V4
c, err := Open(dbPath)
if err != nil {
    return recoverAndRebuild(dbPath, jsonlData, tasks)
}
defer c.Close()
fresh, err := c.IsFresh(jsonlData)
if err != nil {
    c.Close()
    return recoverAndRebuild(dbPath, jsonlData, tasks)
}
```

Both handle corruption correctly. V5 is more explicit about closing the existing connection before recreating (in the `IsFresh` error path). V4 also closes via `c.Close()` before calling `recoverAndRebuild`, which is correct.

**Metadata clearing in Rebuild:**

V4 uses `INSERT OR REPLACE INTO metadata` without clearing metadata first:
```go
// V4 line 138
if _, err := tx.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('jsonl_hash', ?)", hash); err != nil {
```

V5 clears metadata before inserting:
```go
// V5 lines 97-99
if _, err := tx.Exec("DELETE FROM metadata"); err != nil {
    return fmt.Errorf("clearing metadata: %w", err)
}
// ... then later:
if _, err := tx.Exec(`INSERT INTO metadata (key, value) VALUES ('jsonl_hash', ?)`, hash); err != nil {
```

V4's `INSERT OR REPLACE` is functionally equivalent and more efficient (one SQL statement vs two). V5's `DELETE` + `INSERT` is more explicit and consistent with the pattern used for tasks and dependencies. Both are correct.

**Timestamp formatting:**

V4 formats timestamps inline:
```go
// V4 line 119-120
created := t.Created.UTC().Format("2006-01-02T15:04:05Z")
updated := t.Updated.UTC().Format("2006-01-02T15:04:05Z")
```

V5 reuses the `task.FormatTimestamp()` utility:
```go
// V5 line 119
task.FormatTimestamp(t.Created),
task.FormatTimestamp(t.Updated),
```

V5's approach is genuinely better: it avoids duplicating the timestamp format string and reuses an existing utility from the task package. If the format ever changes, V4 would need to update it in two places (task package and cache package) while V5 only needs one.

**Close() nil guard:**

V5 has a nil guard on Close:
```go
// V5 line 72-76
func (c *Cache) Close() error {
    if c.db != nil {
        return c.db.Close()
    }
    return nil
}
```

V4 does not:
```go
// V4 line 72-74
func (c *Cache) Close() error {
    return c.db.Close()
}
```

V5's nil guard is defensive but unnecessary since `*Cache` is only constructed by `New()` which always sets `db`. V4's version is simpler and fine given the invariant.

### Code Quality

**Error message style:**

V4 uses `"failed to ..."` prefix:
```go
return nil, fmt.Errorf("failed to open cache database: %w", err)
return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
return fmt.Errorf("failed to begin rebuild transaction: %w", err)
```

V5 uses gerund phrases:
```go
return nil, fmt.Errorf("opening cache database: %w", err)
return nil, fmt.Errorf("initializing cache schema: %w", err)
return fmt.Errorf("beginning rebuild transaction: %w", err)
```

V5's style follows the Go convention more closely. The Go standard library and effective Go both recommend wrapping errors without "failed to" prefixes since the error itself implies failure. V5 is the more idiomatic choice.

**Package documentation:**

V4:
```go
// Package cache provides SQLite-based caching for task data.
// The cache is expendable and always rebuildable from the JSONL source of truth.
// It uses SHA256 hash-based freshness detection to self-heal on every operation.
```

V5:
```go
// Package cache provides a SQLite-based query cache for tasks that
// auto-rebuilds from the JSONL source of truth using SHA256 freshness detection.
```

Both are well-documented. V4 is more verbose (3 lines); V5 is more concise (2 lines). Both adequately describe the package's purpose.

**All exported functions documented:** Both versions document all exported functions (`Open`/`New`, `DB`, `Close`, `Rebuild`, `IsFresh`, `EnsureFresh`). PASS for both.

**Error handling:** Both versions handle all errors explicitly with `%w` wrapping. Neither uses naked returns. Both use `defer tx.Rollback()` for transaction safety. No `_ =` error suppression without justification in V4. V5 uses `_ = tx.Rollback()` in the deferred call which is the standard pattern -- V4 uses bare `defer tx.Rollback()` which is equivalent (the error from Rollback after Commit is harmless).

**DRY:**

V5 reuses `task.FormatTimestamp()` from the task package, avoiding timestamp format duplication. V4 hardcodes `"2006-01-02T15:04:05Z"` in the cache package, duplicating the format constant that already exists in the task package.

V4's test file imports `_ "github.com/mattn/go-sqlite3"` -- this is unnecessary since the production code already imports it with the blank identifier. V5's test file does not import it. Both work correctly.

### Test Quality

**V4 test functions (4 top-level, 14 subtests):**

Top-level functions:
- `TestCache_CreateSchema` (1 subtest)
- `TestCache_Rebuild` (5 subtests)
- `TestCache_Freshness` (2 subtests)
- `TestCache_EnsureFresh` (4 subtests)

Subtests:
1. `"it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)"` -- verifies all 3 tables, 3 indexes, tasks columns, dependencies columns, metadata columns
2. `"it rebuilds cache from parsed tasks -- all fields round-trip correctly"` -- queries all tasks back into structs, verifies each field including nullable fields
3. `"it normalizes blocked_by array into dependencies table rows"` -- checks 2 deps for task with blockers, verifies total count is 2
4. `"it stores JSONL content hash in metadata table after rebuild"` -- computes expected hash, queries metadata, compares
5. `"it handles empty task list (zero rows, hash still stored)"` -- zero tasks, zero deps, hash still present
6. `"it replaces all existing data on rebuild (no stale rows)"` -- rebuilds twice, verifies old data replaced, checks ID of remaining task, checks hash updated
7. `"it rebuilds within a single transaction (all-or-nothing)"` -- verifies consistency after successful rebuild (does NOT test rollback on failure)
8. `"it detects fresh cache (hash matches) and skips rebuild"` -- rebuild then IsFresh with same data
9. `"it detects stale cache (hash mismatch) and triggers rebuild"` -- rebuild then IsFresh with different data
10. `"it rebuilds from scratch when cache.db is missing"` -- EnsureFresh on non-existent path
11. `"it skips rebuild when cache is fresh"` -- EnsureFresh twice with same data
12. `"it rebuilds when cache is stale"` -- EnsureFresh with different data, verifies count changes
13. `"it deletes and recreates cache.db when corrupted"` -- writes garbage to file, EnsureFresh recovers

Edge cases tested by V4:
- All 3 table schemas verified with PRAGMA table_info
- Metadata table columns verified (extra vs V5)
- EnsureFresh fresh-skip tested (extra vs V5)
- EnsureFresh stale-rebuild tested (extra vs V5)
- Hash updated after second rebuild verified (extra vs V5)
- Task ID verified after second rebuild (extra vs V5)

**V5 test functions (1 top-level, 11 subtests):**

Top-level functions:
- `TestCache` (11 subtests)

Subtests:
1. `"it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)"` -- verifies tables, indexes, tasks columns, dependencies columns
2. `"it rebuilds cache from parsed tasks - all fields round-trip correctly"` -- queries individual fields directly (not scanning into structs), checks closed timestamp on second task
3. `"it normalizes blocked_by array into dependencies table rows"` -- checks 2 deps for task with blockers, verifies no deps for task without blockers
4. `"it stores JSONL content hash in metadata table after rebuild"` -- computes expected hash, queries, compares
5. `"it detects fresh cache (hash matches) and skips rebuild"` -- rebuild then IsFresh with same data
6. `"it detects stale cache (hash mismatch) and triggers rebuild"` -- rebuild then IsFresh with different data
7. `"it rebuilds from scratch when cache.db is missing"` -- EnsureFresh on non-existent path, verifies count
8. `"it deletes and recreates cache.db when corrupted"` -- writes garbage, EnsureFresh recovers, verifies count
9. `"it handles empty task list (zero rows, hash still stored)"` -- zero tasks, hash present
10. `"it replaces all existing data on rebuild (no stale rows)"` -- rebuilds twice, verifies old data replaced
11. `"it rebuilds within a single transaction (all-or-nothing)"` -- attempts rebuild with duplicate IDs, verifies original data preserved (TESTS ACTUAL ROLLBACK)

Edge cases tested by V5 but not V4:
- Transaction rollback on failure (duplicate ID test) -- V5's transaction test is **genuinely superior** because it verifies the rollback behavior by inserting duplicate IDs that cause an error, then confirming original data survives

Edge cases tested by V4 but not V5:
- Metadata table column verification (V4's schema test also checks metadata table columns)
- EnsureFresh skip-when-fresh behavior
- EnsureFresh stale-rebuild with data verification
- Hash update verification after second rebuild
- Task ID verification after second rebuild

**Test data differences:**

V5's sample data includes a description with newlines (`"A description\nwith newlines"`) which exercises multiline content round-tripping. V4's sample data uses a simple description (`"A description"`). V5's test is more thorough on this point.

V4's round-trip test queries all rows into structs and verifies them comprehensively. V5's round-trip test queries individual fields with direct SQL and checks each one, which is equally thorough but less elegant.

**Test organization:**

V4 uses 4 top-level test functions organized by concern (`CreateSchema`, `Rebuild`, `Freshness`, `EnsureFresh`). V5 uses 1 top-level `TestCache` function with all subtests flat. V4's organization is better -- it groups related subtests and makes test output easier to navigate. The Go testing convention favors multiple top-level functions over a single monolithic one.

**Assertion quality:** Both versions use `t.Fatalf` for precondition failures and `t.Errorf` for assertion failures, which is correct Go testing practice. Neither uses a third-party assertion library, which is idiomatic Go.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| MUST: Use gofmt and golangci-lint | PASS -- code appears formatted | PASS -- code appears formatted |
| MUST: Add context.Context to all blocking operations | N/A -- SQLite operations are local I/O, not network-blocking; neither version uses context.Context. This is a reasonable judgment call for a local SQLite cache. | N/A -- Same assessment |
| MUST: Handle all errors explicitly | PASS -- All errors handled with `%w` wrapping | PASS -- All errors handled with `%w` wrapping |
| MUST: Write table-driven tests with subtests | PARTIAL -- Uses subtests via `t.Run()` but no table-driven tests. All tests are individual subtests, not table-driven. | PARTIAL -- Uses subtests via `t.Run()` but no table-driven tests. All tests are individual subtests, not table-driven. |
| MUST: Document all exported functions, types, and packages | PASS -- Package doc, all exported funcs/types documented | PASS -- Package doc, all exported funcs/types documented |
| MUST: Propagate errors with fmt.Errorf("%w", err) | PASS -- All error returns use `%w` | PASS -- All error returns use `%w` |
| MUST NOT: Ignore errors without justification | PASS -- No suppressed errors. `defer tx.Rollback()` is standard practice (return value ignored because Commit() already called). | PASS -- Uses `_ = tx.Rollback()` explicitly, which is the conventional way to document intentional error suppression. |
| MUST NOT: Use panic for normal error handling | PASS -- No panics | PASS -- No panics |
| MUST NOT: Hardcode configuration | PASS -- `dbPath` is a parameter, not hardcoded | PASS -- `dbPath` is a parameter, not hardcoded |

### Spec-vs-Convention Conflicts

**1. `context.Context` on SQLite operations**

- **Spec says**: No mention of `context.Context`
- **Skill requires**: "Add context.Context to all blocking operations"
- **V4 chose**: No context.Context
- **V5 chose**: No context.Context
- **Assessment**: Reasonable judgment call. SQLite operations on a local file are not truly "blocking" in the network sense. Adding context would complicate the API for minimal benefit. Both versions made the same pragmatic choice.

**2. Table-driven tests**

- **Spec says**: Lists 11 specific test descriptions, each testing a distinct scenario
- **Skill requires**: "Write table-driven tests with subtests"
- **V4 chose**: Individual subtests, no table-driven structure
- **V5 chose**: Individual subtests, no table-driven structure
- **Assessment**: The spec's test scenarios are fundamentally different operations (schema creation, rebuild, freshness, corruption) that don't share a common input/output shape. Table-driven tests would not be appropriate here. Both versions correctly used individual subtests. This is not a real conflict -- the skill's guidance is for cases where test scenarios share a common structure.

**3. `EnsureFresh` return type**

- **Spec says**: "`EnsureFresh` function: takes raw JSONL bytes and parsed `[]Task`, checks freshness, triggers rebuild if stale, no-ops if fresh."
- **Convention**: Functions that open resources should return them so callers can manage lifecycle.
- **V4 chose**: Returns `error` only, manages cache lifecycle internally. Spec-literal.
- **V5 chose**: Returns `(*Cache, error)`, gives caller the connection. Convention-aligned.
- **Assessment**: V5's approach is more practical. The spec describes `EnsureFresh` as "the gatekeeper called on every operation" -- callers will need the cache to perform queries. V4's approach forces a second `Open()` call after `EnsureFresh()`, which is wasteful. V5's deviation from spec is a positive design choice.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 6 | 6 |
| Lines added | 960 | 784 |
| Impl LOC | 214 | 226 |
| Test LOC | 739 | 550 |
| Test functions (top-level) | 4 | 1 |
| Test subtests | 14 (incl. 3 bonus) | 11 |

## Verdict

**V5 is the better implementation**, though V4 has a meaningful advantage in test coverage breadth.

**V5 wins on implementation quality:**
- Reuses `task.FormatTimestamp()` instead of duplicating the timestamp format string (DRY principle)
- Returns `(*Cache, error)` from `EnsureFresh`, giving callers the connection they need for subsequent queries
- Uses idiomatic Go error message style (gerund phrases, no "failed to" prefix)
- Cleaner `recreate()` helper with better separation of concerns
- Transaction test actually verifies rollback behavior with duplicate IDs, which is the most meaningful test of transactional atomicity

**V4 wins on test breadth:**
- 14 subtests vs 11, with 3 additional EnsureFresh scenarios (fresh-skip, stale-rebuild, stale-rebuild-with-data-verification)
- 4 organized top-level test functions vs 1 monolithic function
- Metadata table column verification in schema test
- Hash update verification after second rebuild

**V4 weaknesses:**
- Duplicates timestamp format string instead of reusing `task.FormatTimestamp()`
- `EnsureFresh` returns `error` only, requiring callers to reopen the database
- Transaction test only verifies consistency after success, does not test rollback on failure
- "failed to" error prefix style is less idiomatic

**V5 weaknesses:**
- Dead `path` field in `Cache` struct (stored but never used)
- Single monolithic `TestCache` function instead of organized groups
- Fewer EnsureFresh integration tests
- Unnecessary nil guard in `Close()`

The implementation quality gap (especially DRY timestamp formatting, practical `EnsureFresh` return type, and the superior transaction rollback test) outweighs V4's test breadth advantage. V5's code would be easier to maintain and integrate with the rest of the codebase.
