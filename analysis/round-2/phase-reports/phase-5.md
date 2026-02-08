# Phase 5: Stats & Cache Rebuild

## Task Scorecard

| Task | Winner | Margin | Key Difference |
|------|--------|--------|----------------|
| 5-1 (stats) | V4 | Moderate | Type-safe formatter interface, decomposed query helpers, blocked = open - ready |
| 5-2 (rebuild) | V4 | Narrow | Real lock exclusion test, proper ENOENT guard on os.Remove, richer verbose output |

## Cross-Task Architecture Analysis

### Shared CLI Skeleton: DiscoverTickDir -> openStore -> delegate -> format

Both tasks in both implementations follow an identical CLI-layer skeleton. The pattern visible only at the phase level is how `stats` and `rebuild` share the exact same plumbing but use different store APIs:

**V2** -- both commands share `DiscoverTickDir -> a.newStore -> store.{Query|ForceRebuild} -> a.formatter.Format{Stats|Message}`:
```go
// stats.go (V2)
func (a *App) runStats() error {
    tickDir, err := DiscoverTickDir(a.workDir)
    store, err := a.newStore(tickDir)
    defer store.Close()
    err = store.Query(func(db *sql.DB) error { ... })
    return a.formatter.FormatStats(a.stdout, &stats)
}

// rebuild.go (V2)
func (a *App) runRebuild() error {
    tickDir, err := DiscoverTickDir(a.workDir)
    store, err := a.newStore(tickDir)
    defer store.Close()
    count, err := store.ForceRebuild()
    return a.formatter.FormatMessage(a.stdout, msg)
}
```

**V4** -- identical pattern using `DiscoverTickDir -> a.openStore -> s.{Query|Rebuild} -> a.Formatter.Format{Stats|Message}`:
```go
// stats.go (V4)
func (a *App) runStats(args []string) error {
    tickDir, err := DiscoverTickDir(a.Dir)
    s, err := a.openStore(tickDir)
    defer s.Close()
    err = s.Query(func(db *sql.DB) error { ... })
    return a.Formatter.FormatStats(a.Stdout, stats)
}

// rebuild.go (V4)
func (a *App) runRebuild(args []string) error {
    tickDir, err := DiscoverTickDir(a.Dir)
    s, err := a.openStore(tickDir)
    defer s.Close()
    count, err := s.Rebuild()
    return a.Formatter.FormatMessage(a.Stdout, msg)
}
```

Both implementations achieve good DRY at the CLI layer -- `newStore`/`openStore` handles logger wiring, the formatter dispatches output. The critical difference is **where** the cross-task shared code lives.

### Query Reuse: readyWhere vs readyConditionsFor

The `stats` command must reuse "ready" query logic from Phase 3. This is the most architecturally interesting cross-task dependency in the phase.

**V2** uses string constants (`readyWhere` and `blockedWhere` in `list.go`):
```go
// list.go (V2) -- constants without table alias
const readyWhere = `status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )`

// stats.go (V2) -- consumed via string concatenation
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + readyWhere).Scan(&readyCount)
err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + blockedWhere).Scan(&blockedCount)
```

**V4** uses a parameterized function (`readyConditionsFor` in `ready.go`):
```go
// ready.go (V4) -- function with table alias parameter
func readyConditionsFor(alias string) string {
    return `
  NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = ` + alias + `.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = ` + alias + `.id
      AND child.status IN ('open', 'in_progress')
  )`
}

// stats.go (V4) -- consumed via function call
readyCountQuery := `SELECT COUNT(*) FROM tasks t WHERE t.status = 'open' AND` + readyConditionsFor("t")
```

V4's approach is genuinely superior for cross-task reuse: the parameterized alias means the same function works in `ready.go` (for the ready command), `blocked.go` (for the blocked command, where it appears inside a NOT IN subquery), `list.go` (for combined filters with different alias contexts), and `stats.go` (for count queries). V2's raw string constant works but cannot be used in contexts requiring a table alias, forcing `blockedWhere` to exist as a separate constant that must be kept in sync.

### Store API: Query vs ForceRebuild/Rebuild

The two tasks exercise fundamentally different store APIs:
- `stats` uses `store.Query(func(db *sql.DB) error)` -- shared read lock, freshness check, then SQL queries
- `rebuild` uses `store.ForceRebuild()`/`store.Rebuild()` -- exclusive lock, delete cache, full rebuild

In V2, `ForceRebuild` (store.go lines 162-211) duplicates the lock acquisition pattern from `Mutate` (lines 89-157) and `Query` (lines 215-260). Three copies of lock-acquire-retry with `10*time.Millisecond` retry interval.

In V4, `Rebuild` (store.go lines 160-213) also duplicates the lock pattern from `Mutate` (lines 81-154) and `Query` (lines 221-268). Three copies with `100*time.Millisecond` retry interval.

Neither implementation extracts a shared lock-acquisition helper. This is a DRY gap visible only at the phase level -- the three store methods (`Mutate`, `Query`, `Rebuild`/`ForceRebuild`) all contain identical lock acquisition code. A `withExclusiveLock` / `withSharedLock` helper would eliminate this. V2 has 14 duplicated lines per copy; V4 has 16 (split `err`/`!locked` checks).

### Verbose Logger Wiring

Both tasks benefit from the logger being wired in `newStore`/`openStore`:

**V2**: `store.SetLogger(a.verbose)` -- interface-based, `Logger.Log(msg string)` takes a plain string. Each verbose message in the store is pre-formatted: `s.logVerbose("rebuild: deleting existing cache.db")`.

**V4**: `s.LogFunc = a.vlog.Log` -- function-based, `Log(format string, args ...interface{})` takes a format string. Store messages include runtime context: `s.vlog("deleting existing cache.db at %s", s.dbPath)`.

The function-based approach (V4) is a better cross-task pattern because:
1. It requires no interface definition (`Logger` in V2 is a single-method interface that could just be a function)
2. Format strings are resolved lazily at call site, avoiding unnecessary `fmt.Sprintf` when verbose is disabled (V2 uses `fmt.Sprintf` before calling `logVerbose`)
3. Adding path context to verbose messages is natural

### Formatter Interface: interface{} vs Concrete Type

This is the most impactful cross-task pattern. Both tasks use the Formatter -- `stats` calls `FormatStats`, `rebuild` calls `FormatMessage`. The interface difference ripples across every formatter implementation:

**V2** requires type assertions in EVERY formatter for EVERY stats method:
```go
// toon_formatter.go (V2)
func (f *ToonFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }

// pretty_formatter.go (V2)
func (f *PrettyFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }

// json_formatter.go (V2)
func (f *JSONFormatter) FormatStats(w io.Writer, stats interface{}) error {
    sd, ok := stats.(*StatsData)
    if !ok {
        return fmt.Errorf("FormatStats: expected *StatsData, got %T", stats)
    }
```

That is 3 copies of identical type assertion boilerplate (9 lines total) that would not compile-error if `StatsData` were renamed or restructured. V2 also passes `*StatsData` (pointer) while V4 passes `StatsData` (value) -- the value copy is appropriate for a small struct and avoids nil pointer risks.

**V4** uses a concrete type everywhere:
```go
// toon_formatter.go (V4)
func (f *ToonFormatter) FormatStats(w io.Writer, stats StatsData) error {
// pretty_formatter.go (V4)
func (f *PrettyFormatter) FormatStats(w io.Writer, stats StatsData) error {
// json_formatter.go (V4)
func (f *JSONFormatter) FormatStats(w io.Writer, stats StatsData) error {
```

Zero boilerplate. Compile-time type safety. This is the phase-level pattern: V2's `interface{}` choice in Phase 4 creates a recurring tax in every Phase 5 formatter method.

### FormatMessage: Shared Across rebuild + init

`FormatMessage` is used by `rebuild` and also by `init` (from Phase 1). Both implementations have identical `FormatMessage` implementations across all three formatters:
- TOON: `fmt.Fprintln(w, message)` -- plain text
- Pretty: `fmt.Fprintln(w, message)` -- plain text
- JSON: `marshalIndentTo(w, jsonMessage{Message: message})` / `f.writeJSON(w, jsonMessage{Message: msg})` -- wrapped in `{"message": "..."}`

This is correctly shared infrastructure. No cross-task issue here.

## Code Quality Patterns

### Naming Consistency

| Pattern | V2 | V4 |
|---------|-----|-----|
| Store variable | `store` | `s` |
| Store constructor | `a.newStore(tickDir)` | `a.openStore(tickDir)` |
| Store rebuild method | `store.ForceRebuild()` | `s.Rebuild()` |
| App fields | `a.config.Quiet`, `a.workDir`, `a.stdout` | `a.Quiet`, `a.Dir`, `a.Stdout` |
| Verbose logger | `a.verbose` (interface) | `a.vlog` (struct pointer) |
| Stats data location | `StatsData` in `toon_formatter.go` | `StatsData` in `format.go` |

V4 is more consistent: shorter store variable name throughout, simpler method names (no "Force" prefix when there is no non-force variant), and exported App fields for easy test construction. V4 also correctly places `StatsData` in `format.go` alongside the `Formatter` interface, while V2 oddly defines it in `toon_formatter.go`.

### Error Handling

Both implementations handle errors consistently within their own codebase:
- V2: `fmt.Errorf("failed to ...: %w", err)` wrapping pattern
- V4: Same pattern with slightly different messages

The one genuinely different error handling is `os.Remove` in the rebuild store method:
```go
// V2 -- silently discards ALL errors
os.Remove(s.cachePath)

// V4 -- only ignores ENOENT
if err := os.Remove(s.dbPath); err != nil && !os.IsNotExist(err) {
    return 0, fmt.Errorf("failed to delete cache.db: %w", err)
}
```

V4 is correct. V2 could mask permission errors or filesystem corruption.

### SQL Patterns

Both use identical SQL for status and priority grouping:
```sql
SELECT status, COUNT(*) FROM tasks GROUP BY status
SELECT priority, COUNT(*) FROM tasks GROUP BY priority
```

The key SQL difference is in the ready/blocked counting approach:
- V2 runs 4 queries: status GROUP BY, priority GROUP BY, `COUNT(*) WHERE readyWhere`, `COUNT(*) WHERE blockedWhere`
- V4 runs 3 queries: status GROUP BY, priority GROUP BY, `COUNT(*) WHERE readyConditionsFor("t")`, then computes `blocked = open - readyCount`

V4's approach is mathematically correct (every open task is either ready or blocked), eliminates a query, and avoids maintaining a separate `blockedWhere` constant. The `NOT EXISTS` subquery style in V4 is also more performant than V2's `NOT IN` style for large datasets (NOT EXISTS can short-circuit).

## Test Coverage Analysis

### Aggregate Counts

| Metric | V2 | V4 |
|--------|-----|-----|
| Stats test functions | 8 (subtests in 1 top-level) | 8 (8 top-level, 1 subtest each) |
| Rebuild test functions | 8 (subtests in 1 top-level) | 8 (8 top-level, 1 subtest each) |
| Stats test LOC | 425 | 393 |
| Rebuild test LOC | 341 | 321 |
| Total test LOC (phase) | 766 | 714 |

V4 achieves equivalent coverage in fewer lines due to type-safe test helpers and typed JSON assertion structs.

### Test Organization

V2 uses a single top-level `TestStats`/`TestRebuild` with subtests via `t.Run`. V4 uses separate top-level functions (`TestStats_CountsTasksByStatusCorrectly`, `TestRebuild_RebuildsCacheFromJSONL`, etc.) each containing a single subtest. V4's approach provides better isolation: a panic in one test function does not skip later tests. It also produces cleaner `go test -v` output with named top-level functions.

### Test Setup Pattern: Cross-Task Reuse

V2 uses string-based setup across both tasks:
```go
// Used in both stats_test.go and rebuild_test.go (V2)
content := strings.Join([]string{
    taskJSONL("tick-aaa111", "Task one", "open", 2, nil, "", "2026-01-19T10:00:00Z"),
}, "\n") + "\n"
dir := setupTickDirWithContent(t, content)
```

V4 uses type-safe setup across both tasks:
```go
// Used in both stats_test.go and rebuild_test.go (V4)
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
}
dir := setupInitializedDirWithTasks(t, tasks)
```

V4's setup helper is shared across both test files, providing: (1) compile-time field validation, (2) `task.StatusOpen` constants instead of raw `"open"` strings, (3) explicit `Created`/`Updated` fields using `time.Time` instead of string timestamps.

### Edge Case Comparison

| Edge Case | V2 | V4 | Winner |
|-----------|-----|-----|--------|
| TOON priority rows (all 5) | Checks all 5 (P0-P4) | Checks only 3 (P0-P2) | V2 |
| Pretty right-alignment | Checks `" 3"` alignment | Checks labels only, not alignment | V2 |
| Pretty sub-labels | Section headers only | All sub-labels (Open, In Progress, etc.) | V4 |
| Lock exclusion (rebuild) | Checks verbose log for "exclusive lock" | Holds external flock, asserts exit code 1 | V4 |
| Dependency in rebuild | Tests `dependencies` table row count | Not tested | V2 |
| JSON typed assertions | `map[string]interface{}` + float64 casts | `jsonStats` struct | V4 |
| Verbose step coverage | 3 categories (delete, read, hash) | 4 categories (delete, read, insert/rebuild, hash) | V4 |

### Critical Gap: V2's Lock Test is Meaningless

V2's lock test (rebuild_test.go line 200) just checks that `--verbose` output mentions "exclusive lock". This would pass even if the locking code were entirely removed, as long as the verbose logger printed the string. V4's lock test (line 162) actually holds an external `flock.New(lockPath).TryLock()` and verifies the command fails with exit code 1. This is a genuine behavioral test vs a log-message assertion. The V2 lock test provides zero confidence that locking works.

## Phase Verdict

**V4 wins the phase.** The margin is moderate for task 5-1 (stats) and narrow for task 5-2 (rebuild), but the cross-task patterns all favor V4, widening the overall gap:

1. **Query reuse architecture**: V4's `readyConditionsFor(alias)` function is used across 4 files (ready.go, blocked.go, list.go, stats.go) with different table alias contexts. V2's string constants work for simple concatenation but cannot adapt to alias-required contexts, forcing a separate `blockedWhere` constant. This is the clearest phase-level architectural win.

2. **Formatter type safety**: V2's `interface{}` in `FormatStats` causes 9 lines of duplicated type assertion boilerplate across 3 formatter files, with only runtime error detection. V4's concrete `StatsData` parameter eliminates this entirely with compile-time safety. This is a Phase 4 decision that taxes every Phase 5 formatter method.

3. **Blocked computation elegance**: V4's `blocked = open - ready` eliminates one SQL query per stats invocation, removes the need for a `blockedWhere` constant, and is mathematically sound. V2 maintains two complex WHERE fragments that must stay in sync.

4. **Lock test quality**: V4's rebuild lock test actually verifies lock exclusion by holding an external flock. V2's test only checks log output -- a false positive waiting to happen.

5. **Store-level logger pattern**: V4's `LogFunc func(format string, args ...interface{})` is simpler than V2's `Logger` interface (single-method interfaces that should be functions), and naturally supports format strings with runtime path context.

6. **StatsData placement**: V4 defines `StatsData` in `format.go` alongside the `Formatter` interface. V2 defines it in `toon_formatter.go` -- a poor location since it's consumed by all formatters and the stats command.

V2's advantages are limited to a few test coverage edges: checking all 5 TOON priority rows (vs V4's 3), verifying Pretty right-alignment numerically, and testing dependency preservation during rebuild. These are minor gaps that do not overcome V4's structural advantages.
