# Task tick-core-1-3: SQLite cache with freshness detection

## Task Summary

This task requires implementing SQLite as an auto-rebuilding cache for task data, always rebuildable from the JSONL source of truth. Key requirements:

- **Schema**: 3 tables (`tasks`, `dependencies`, `metadata`) with 3 indexes (`idx_tasks_status`, `idx_tasks_priority`, `idx_tasks_parent`)
- **Full rebuild**: Accept `[]Task` slice, clear all rows, insert all tasks/dependencies in a single transaction, compute and store SHA256 hash of raw JSONL content in `metadata` as key `jsonl_hash`
- **Freshness check**: Compare SHA256 hash of JSONL content against stored `jsonl_hash`; return fresh/stale
- **EnsureFresh**: Gatekeeper called on every operation -- checks freshness, rebuilds if stale, no-ops if fresh
- **Missing cache.db**: Create from scratch and rebuild
- **Corrupted cache.db**: Delete, recreate, rebuild; log warning but do not fail
- **Driver**: `github.com/mattn/go-sqlite3`
- **Hash**: `crypto/sha256` on raw file bytes

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Schema matches spec (3 tables, 3 indexes) | PASS -- identical schema DDL, test verifies table and index names exist via `sqlite_master` | PASS -- identical schema DDL, test uses `PRAGMA table_info` to verify column names AND types for all 3 tables, plus index names | PASS -- identical schema DDL, test uses `PRAGMA table_info` to verify column names AND types for all 3 tables, plus index names |
| 2. Full rebuild populates tasks + deps | PASS -- round-trip test verifies all fields on a single task with all optional fields set | PASS -- round-trip test with 3 tasks covering all field combinations (open, done w/closed, in_progress w/description+newlines) | PASS -- round-trip test verifies all 9 fields on a single fully-populated task |
| 3. SHA256 hash stored after rebuild | PASS -- stores hash, test verifies stored value matches `hashBytes()` helper | PASS -- stores hash, test verifies stored value matches `hashBytes()` helper | PASS -- stores hash, test hardcodes expected SHA256 hex string for verification |
| 4. Freshness check (fresh vs stale) | PASS -- `IsFresh` tested for both matching and mismatching content, plus missing hash | PASS -- `IsFresh` tested for matching, mismatching, and missing hash (separate test function) | PASS -- `IsFresh` tested for matching, mismatching, and missing hash |
| 5. Missing cache.db triggers creation + rebuild | PASS -- test creates NewCache on non-existent path, verifies file exists afterward | PASS -- test calls `EnsureFresh` on non-existent path, verifies tasks inserted and hash stored | PASS -- test calls `EnsureFresh` on non-existent path, verifies task count |
| 6. Corrupted cache.db recovered without failure | PASS -- `NewCacheWithRecovery` deletes garbage file and recreates, test verifies rebuild works | PASS -- `EnsureFresh` detects corruption, deletes, recreates; tested with both garbage file AND wrong-schema DB | PASS -- `EnsureFresh` detects corruption, deletes, recreates; tested with garbage file |
| 7. Empty task list handled | PASS -- test verifies 0 rows in tasks and hash still stored | PASS -- test verifies 0 rows in tasks AND dependencies, plus hash stored | PASS -- test verifies 0 rows in tasks AND dependencies, plus hash stored |
| 8. Transactional rebuild | PARTIAL -- no explicit test for transaction rollback on failure | PASS -- test inserts good data, then attempts rebuild with duplicate primary keys, verifies original data intact and hash unchanged | PASS -- test inserts good data, then attempts rebuild with duplicate primary keys, verifies original data intact |

## Implementation Comparison

### Approach

#### File Organization

**V1** places the cache in `internal/storage/cache.go` (package `storage`), co-located with the existing JSONL reader. This is the flattest approach.

**V2** creates a new sub-package `internal/storage/sqlite/sqlite.go` (package `sqlite`). This is the most separated approach, giving the SQLite cache its own namespace.

**V3** places the cache in `internal/storage/cache.go` (package `storage`), same as V1.

#### EnsureFresh Architecture

This is the most significant architectural divergence between versions.

**V1** makes `EnsureFresh` a method on `*Cache`:

```go
// V1: EnsureFresh is a method on Cache
func (c *Cache) EnsureFresh(jsonlContent []byte, tasks []task.Task) error {
    fresh, err := c.IsFresh(jsonlContent)
    if err != nil {
        return c.Rebuild(tasks, jsonlContent)
    }
    if fresh {
        return nil
    }
    return c.Rebuild(tasks, jsonlContent)
}
```

The caller must separately create the `Cache` via `NewCache` or `NewCacheWithRecovery`, then call `EnsureFresh`. This means corruption recovery and freshness checking are decoupled into separate call sites.

**V2** makes `EnsureFresh` a package-level function that returns a `*Cache`:

```go
// V2: EnsureFresh is a standalone function, returns *Cache
func EnsureFresh(dbPath string, tasks []task.Task, rawContent []byte) (*Cache, error) {
    cache, err := tryOpen(dbPath)
    if err != nil {
        log.Printf("warning: cache db unusable, recreating: %v", err)
        return recreateAndRebuild(dbPath, tasks, rawContent)
    }
    fresh, err := cache.IsFresh(rawContent)
    if err != nil {
        cache.Close()
        log.Printf("warning: cache freshness check failed, recreating: %v", err)
        os.Remove(dbPath)
        return recreateAndRebuild(dbPath, tasks, rawContent)
    }
    if fresh {
        return cache, nil
    }
    if err := cache.Rebuild(tasks, rawContent); err != nil {
        return nil, fmt.Errorf("failed to rebuild cache: %w", err)
    }
    return cache, nil
}
```

This is the most complete implementation: it handles all three recovery scenarios (missing DB, corrupted file, corrupted schema) in one unified entry point. It also includes `tryOpen` which probes all three tables, and `recreateAndRebuild` as a helper.

**V3** also makes `EnsureFresh` a package-level function returning `*Cache`:

```go
// V3: EnsureFresh is a standalone function, returns *Cache
func EnsureFresh(cachePath string, tasks []task.Task, jsonlContent []byte) (*Cache, error) {
    cache, err := NewCache(cachePath)
    if err != nil {
        os.Remove(cachePath)
        cache, err = NewCache(cachePath)
        if err != nil {
            return nil, err
        }
    }
    fresh, err := cache.IsFresh(jsonlContent)
    if err != nil {
        cache.Close()
        os.Remove(cachePath)
        cache, err = NewCache(cachePath)
        if err != nil {
            return nil, err
        }
        fresh = false
    }
    if !fresh {
        if err := cache.Rebuild(tasks, jsonlContent); err != nil {
            cache.Close()
            return nil, err
        }
    }
    return cache, nil
}
```

V3's approach is similar to V2 but simpler -- no separate `tryOpen`/`recreateAndRebuild` helpers, no schema probing, no logging.

#### Corruption Recovery

**V1** uses a separate `NewCacheWithRecovery` constructor that wraps `NewCache`:

```go
func NewCacheWithRecovery(dbPath string) (*Cache, error) {
    cache, err := NewCache(dbPath)
    if err != nil {
        os.Remove(dbPath)
        cache, err = NewCache(dbPath)
        if err != nil {
            return nil, fmt.Errorf("recovery failed: %w", err)
        }
    }
    return cache, nil
}
```

This only catches errors from `sql.Open` + schema creation. It does not catch corrupted-but-openable databases (e.g., wrong schema). The `EnsureFresh` method partially covers this by falling through to `Rebuild` on `IsFresh` error, but does not delete/recreate the file.

**V2** has the most thorough corruption detection. Its `tryOpen` function explicitly checks file existence, then probes all three tables with `SELECT 1 FROM ... LIMIT 0`:

```go
func tryOpen(dbPath string) (*Cache, error) {
    if _, err := os.Stat(dbPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("cache db does not exist: %w", err)
    }
    cache, err := NewCache(dbPath)
    if err != nil {
        return nil, err
    }
    if _, err := cache.db.Exec("SELECT 1 FROM tasks LIMIT 0"); err != nil {
        cache.Close()
        return nil, fmt.Errorf("tasks table unusable: %w", err)
    }
    // ... same for dependencies and metadata
    return cache, nil
}
```

Additionally, V2 handles the case where `IsFresh` fails (indicating corruption discovered after open), closing the cache, removing the file, and recreating. V2 also logs warnings via `log.Printf`.

**V3** handles corruption at both `NewCache` failure and `IsFresh` failure, with delete-and-recreate at both points. It does not probe tables separately.

#### Nullable Field Handling

**V1** uses `sql.NullString` for description, parent, and closed:

```go
var description, parent, closed sql.NullString
if t.Description != "" {
    description = sql.NullString{String: t.Description, Valid: true}
}
```

**V2** uses `*string` pointers:

```go
var description, parent, closed *string
if t.Description != "" {
    description = &t.Description
}
```

**V3** passes fields directly without nullable wrappers:

```go
_, err := taskStmt.Exec(t.ID, t.Title, string(t.Status), t.Priority, t.Description, t.Parent, t.Created, t.Updated, t.Closed)
```

V3's approach is the simplest but stores empty strings instead of NULL for optional fields. This is because V3's `Task` struct uses `string` for timestamps and has `Closed string` (not `*time.Time`), so there's no nil/zero distinction at the type level -- empty string is the zero value.

#### Timestamp Handling

**V1** uses `task.FormatTimestamp()` which is a dedicated helper:

```go
task.FormatTimestamp(t.Created)  // calls t.UTC().Format("2006-01-02T15:04:05Z")
```

**V2** formats inline using its own `timeFormat` constant:

```go
const timeFormat = "2006-01-02T15:04:05Z"
// ...
t.Created.UTC().Format(timeFormat)
```

**V3** passes `t.Created` directly as a string (since the Task struct uses `string` for timestamps):

```go
_, err := taskStmt.Exec(t.ID, t.Title, string(t.Status), t.Priority, t.Description, t.Parent, t.Created, t.Updated, t.Closed)
```

V1 and V2 both convert `time.Time` to strings at the cache boundary, while V3 stores the string as-is (no conversion needed since the model already uses strings). This is a difference inherited from the `Task` struct definition in the prior task, not a decision made in this task.

#### Hash Encoding

**V1 and V2** both use `fmt.Sprintf("%x", h)`:

```go
func computeHash(data []byte) string {
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h)
}
```

**V3** uses `encoding/hex`:

```go
hash := sha256.Sum256(jsonlContent)
hashStr := hex.EncodeToString(hash[:])
```

Both produce identical output. V3's approach avoids the format string overhead (marginally) and is arguably more explicit.

#### Directory Creation

**V3** uniquely includes parent directory creation in `NewCache`:

```go
func NewCache(path string) (*Cache, error) {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    // ...
}
```

V1 and V2 do not create parent directories, relying on the caller to ensure the path exists. V3's tests exercise this by using `filepath.Join(dir, ".tick", "cache.db")` where the `.tick` subdirectory doesn't exist yet.

### Code Quality

#### Error Messages

**V1** uses short, context-rich prefix style:

```go
return nil, fmt.Errorf("opening cache database: %w", err)
return nil, fmt.Errorf("creating cache schema: %w", err)
return fmt.Errorf("inserting task %s: %w", t.ID, err)
```

**V2** uses "failed to ..." prefix style consistently:

```go
return nil, fmt.Errorf("failed to open cache db: %w", err)
return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
return fmt.Errorf("failed to insert task %s: %w", t.ID, err)
```

**V3** returns bare errors without wrapping:

```go
return nil, err  // from sql.Open
return nil, err  // from db.Exec(createSchema)
return err       // from tx.Begin, tx.Exec, taskStmt.Exec, etc.
```

V1 and V2 both properly wrap errors with context (Go best practice). V3 loses error context entirely, making debugging harder for callers. This is a genuine quality gap.

#### Close Method

**V2** includes a nil guard on `Close`:

```go
func (c *Cache) Close() error {
    if c.db != nil {
        return c.db.Close()
    }
    return nil
}
```

V1 and V3 call `c.db.Close()` directly, which would panic if `db` is nil. In practice this shouldn't happen since the constructors always set `db`, but V2's defensive coding is better.

#### Resource Cleanup in EnsureFresh

**V2** carefully closes the cache before removing the file when corruption is detected mid-operation:

```go
cache.Close()
log.Printf("warning: cache freshness check failed, recreating: %v", err)
os.Remove(dbPath)
return recreateAndRebuild(dbPath, tasks, rawContent)
```

**V3** does the same:

```go
cache.Close()
os.Remove(cachePath)
cache, err = NewCache(cachePath)
```

**V1** doesn't handle this scenario because `EnsureFresh` is a method on an already-opened cache. If `IsFresh` errors, it simply calls `Rebuild` on the same (potentially corrupted) cache without recreating it.

#### Logging

**V2** is the only version that logs warnings on corruption, per the spec requirement "Log a warning but don't fail the operation":

```go
log.Printf("warning: cache db unusable, recreating: %v", err)
log.Printf("warning: cache freshness check failed, recreating: %v", err)
```

V1 and V3 silently handle corruption with no logging, which does not meet the spec's logging requirement.

#### DB() Accessor

**V1** and **V2** both expose a `DB()` method returning `*sql.DB` for direct queries:

```go
func (c *Cache) DB() *sql.DB {
    return c.db
}
```

**V3** does not expose this accessor. Its tests access `cache.db` directly (the field is unexported but accessible within the same package for tests).

### Test Quality

#### V1 Test Functions (6 top-level, 13 subtests)

| Function | Subtests |
|----------|----------|
| `TestNewCache` | "creates cache.db with correct schema" |
| `TestCacheRebuild` | "rebuilds cache from parsed tasks with all fields", "normalizes blocked_by into dependencies table", "stores JSONL content hash in metadata table", "handles empty task list", "replaces all existing data on rebuild" |
| `TestCacheFreshness` | "detects fresh cache and skips rebuild", "detects stale cache on hash mismatch", "treats missing hash as stale" |
| `TestEnsureFresh` | "rebuilds when stale", "skips rebuild when fresh" |
| `TestCacheMissingDB` | "rebuilds from scratch when cache.db is missing" |
| `TestCacheCorrupted` | "deletes and recreates cache.db when corrupted" |

V1 groups related subtests under parent functions (e.g., all rebuild tests under `TestCacheRebuild`). It uses a shared `fullTask()` helper for the fully-populated task and `sampleTask()` (from `jsonl_test.go`) for simpler tasks.

**Test gap**: No transaction atomicity test. V1 never verifies that a failed rebuild rolls back.

**Schema test gap**: V1 only checks that table and index names exist in `sqlite_master`. It does not verify column names, types, or constraints.

#### V2 Test Functions (15 top-level, 15 subtests)

| Function | Subtest |
|----------|---------|
| `TestCreatesCacheDBWithCorrectSchema` | "it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)" |
| `TestRebuildFromParsedTasks` | "it rebuilds cache from parsed tasks -- all fields round-trip correctly" |
| `TestNormalizesBlockedByIntoDependencies` | "it normalizes blocked_by array into dependencies table rows" |
| `TestStoresHashInMetadata` | "it stores JSONL content hash in metadata table after rebuild" |
| `TestDetectsFreshCache` | "it detects fresh cache (hash matches) and skips rebuild" |
| `TestDetectsStaleCache` | "it detects stale cache (hash mismatch) and triggers rebuild" |
| `TestRebuildsWhenCacheDBMissing` | "it rebuilds from scratch when cache.db is missing" |
| `TestDeletesAndRecreatesCorruptedDB` | "it deletes and recreates cache.db when corrupted" |
| `TestHandlesEmptyTaskList` | "it handles empty task list (zero rows, hash still stored)" |
| `TestReplacesAllExistingDataOnRebuild` | "it replaces all existing data on rebuild (no stale rows)" |
| `TestRebuildIsTransactional` | "it rebuilds within a single transaction (all-or-nothing)" |
| `TestEnsureFreshSkipsRebuildWhenFresh` | "EnsureFresh skips rebuild when cache is fresh" |
| `TestEnsureFreshRebuildsWhenStale` | "EnsureFresh triggers rebuild when cache is stale" |
| `TestIsFreshMissingHashTreatedAsStale` | "it treats missing jsonl_hash in metadata as stale" |
| `TestEnsureFreshWithCorruptedSchemaDB` | "it handles corrupted schema by deleting and rebuilding" |

V2 uses one top-level function per spec test case (1:1 mapping to the task's "Tests" section). Each has exactly one subtest with the spec's exact test description. V2 uses `sampleTasks(t)` returning 3 tasks covering open, done-with-closed, and in_progress-with-multiline-description. V2 also includes `mustParseTime` and `timePtr` helpers.

**Unique to V2**: `TestEnsureFreshWithCorruptedSchemaDB` -- creates a valid SQLite DB with wrong tables (not just garbage bytes), exercises schema mismatch recovery. `TestEnsureFreshSkipsRebuildWhenFresh` passes `nil` for tasks on second call to verify no rebuild occurs (data preserved from first call).

**Thoroughness of rebuild test**: V2's `TestRebuildFromParsedTasks` verifies 3 different tasks, checks the closed field for the second task specifically, and verifies total task count. V1 only verifies 1 task.

**Thoroughness of dependencies test**: V2's `TestNormalizesBlockedByIntoDependencies` checks 3 different tasks' dependencies: one with 2 deps, one with 1, one with 0. V1 only checks one task's 2 deps.

**Transaction test**: V2's `TestRebuildIsTransactional` inserts good data, attempts rebuild with duplicate IDs (primary key violation), then verifies original task ID, count, AND hash are all intact. This is thorough.

#### V3 Test Functions (7 top-level, 14 subtests)

| Function | Subtests |
|----------|----------|
| `TestCacheSchema` | "it creates cache.db with correct schema (tasks, dependencies, metadata tables and indexes)" |
| `TestCacheRebuild` | "it rebuilds cache from parsed tasks -- all fields round-trip correctly", "it normalizes blocked_by array into dependencies table rows", "it stores JSONL content hash in metadata table after rebuild" |
| `TestCacheFreshness` | "it detects fresh cache (hash matches) and skips rebuild", "it detects stale cache (hash mismatch) and triggers rebuild", "it treats missing hash as stale" |
| `TestEnsureFresh` | "it rebuilds from scratch when cache.db is missing", "it skips rebuild when cache is fresh", "it triggers rebuild when cache is stale", "it deletes and recreates cache.db when corrupted" |
| `TestCacheEmptyTasks` | "it handles empty task list (zero rows, hash still stored)" |
| `TestCacheReplaceAllData` | "it replaces all existing data on rebuild (no stale rows)" |
| `TestCacheTransaction` | "it rebuilds within a single transaction (all-or-nothing)" |

V3 uses a mix of grouping styles: some test functions have multiple subtests (like `TestEnsureFresh` with 4 subtests), while others have just one. V3 constructs test tasks inline rather than using a shared helper function.

**Unique to V3**: `TestEnsureFresh/"it skips rebuild when cache is fresh"` -- uses a clever marker technique: inserts a `marker` key into `metadata` after first rebuild, then calls `EnsureFresh` again and verifies the marker still exists (proving no rebuild occurred). This is a stronger assertion than V1's approach (which just calls EnsureFresh and checks for no error) and V2's approach (which passes `nil` tasks and checks data survives).

**Unique to V3**: `TestCacheReplaceAllData` starts with 3 tasks and 1 dependency, rebuilds with 1 task and 0 dependencies, then explicitly checks that old task IDs do not exist using `WHERE id IN (...)`. This is more thorough than V1's equivalent test.

**Unique to V3**: The hash test (`TestCacheRebuild/"it stores JSONL content hash..."`) hardcodes the expected SHA256 hex string rather than computing it via a helper. This serves as a regression anchor but is fragile if the test input changes.

**Schema test**: V3 uses `PRAGMA table_info` to verify column names AND types for all three tables (same depth as V2), plus verifies indexes using `WHERE sql IS NOT NULL` to filter auto-indexes.

**Transaction test**: V3's `TestCacheTransaction` verifies original task title and that duplicate IDs don't exist after failed rebuild. Does not verify hash is unchanged (V2 does).

#### Edge Cases Tested Per Version

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Schema column names and types | No | Yes | Yes |
| Multiple tasks in rebuild | No (1 task) | Yes (3 tasks) | No (1 task in round-trip) |
| Task with all optional fields | Yes | Yes | Yes |
| Task with no optional fields | Yes (via sampleTask) | Yes (second task) | Yes (inline) |
| Multiline description | No | Yes ("Details\nwith newlines") | Yes ("Full description with\nmultiple lines") |
| Dependencies: multiple per task | Yes (2 deps) | Yes (2 deps) | Yes (3 deps) |
| Dependencies: single per task | No | Yes | No |
| Dependencies: zero per task | No | Yes | No |
| Dependencies cleared on rebuild | No | Yes | Yes |
| Missing hash = stale | Yes | Yes | Yes |
| Corrupted DB (garbage file) | Yes | Yes | Yes |
| Corrupted DB (wrong schema) | No | Yes | No |
| Transaction rollback | No | Yes | Yes |
| Transaction rollback preserves hash | No | Yes | No |
| EnsureFresh skip-rebuild proof | No (just no error) | Partial (nil tasks) | Yes (marker technique) |
| EnsureFresh stale triggers rebuild | Yes | Yes | Yes |
| Rebuild clears old tasks verified by ID | No | Yes | Yes |
| Logging on corruption | N/A (not logged) | Tested implicitly | N/A (not logged) |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 4 | 6 | 7 |
| Lines added | 602 | 1143 | 966 |
| Impl LOC | 203 | 261 | 196 |
| Test LOC | 395 | 874 | 748 |
| Top-level test functions | 6 | 15 | 7 |
| Subtests | 13 | 15 | 14 |

## Verdict

**V2 is the best implementation.**

**Evidence:**

1. **Most complete corruption handling**: V2 is the only version that probes table usability (`SELECT 1 FROM tasks LIMIT 0`) via `tryOpen`, detects wrong-schema corruption (not just garbage files), and tests for it with `TestEnsureFreshWithCorruptedSchemaDB`. V1 cannot detect corrupted-but-openable databases at the `EnsureFresh` level. V3 relies solely on `NewCache` failure and `IsFresh` query failure.

2. **Logging on corruption**: V2 is the only version that logs warnings when corruption is detected (`log.Printf("warning: ...")`), as required by the task spec: "Log a warning but don't fail the operation." V1 and V3 silently recover.

3. **Most thorough tests**: V2 tests 3 tasks with all status variants (open, done, in_progress), verifies zero-dependency tasks, single-dependency tasks, and multi-dependency tasks separately. Its transaction test verifies the hash is preserved after rollback (V3 does not). It has a dedicated wrong-schema corruption test. It has 874 test lines vs V1's 395 and V3's 748.

4. **Proper error wrapping**: V2 wraps all errors with `fmt.Errorf("failed to ...: %w", err)` context. V3 returns bare errors throughout its implementation, losing all call-site context.

5. **Defensive coding**: V2 includes a nil guard in `Close()`, properly closes the cache before removing files on corruption, and separates concerns with `tryOpen` and `recreateAndRebuild` helpers.

6. **API design**: V2's `EnsureFresh` as a package-level function returning `*Cache` is a better API than V1's method-on-cache approach. The caller gets a single entry point that handles creation, corruption recovery, freshness checking, and rebuilding. V3 shares this same API design.

**V3 is second-best** due to its clever skip-rebuild marker test, parent directory creation in `NewCache`, and transaction test -- but is let down by bare error returns and no corruption logging.

**V1 is third** due to missing transaction test, shallow schema verification, V1's `EnsureFresh` being a method (requiring separate construction), and no corruption logging.
