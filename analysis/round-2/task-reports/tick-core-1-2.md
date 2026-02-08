# Task tick-core-1-2: JSONL Storage with Atomic Writes

## Task Summary

Implement JSONL (JSON Lines) reader/writer for persistent task storage. Tasks are serialized one-per-line with no array wrapper. Optional fields (`description`, `blocked_by`, `parent`, `closed`) must be omitted when empty/nil, never serialized as `null`. Writes must be atomic using the temp file + fsync + rename pattern. Full file rewrite on every mutation (no append mode). Empty files (0 bytes) are valid and return an empty task list. Missing files must return an error.

**Acceptance Criteria:**
1. Tasks round-trip through write -> read without data loss
2. Optional fields omitted from JSON when empty/null
3. Atomic write uses temp file + fsync + rename
4. Empty file returns empty task list
5. Each task occupies exactly one line (no pretty-printing)
6. JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Tasks round-trip without data loss | PASS - `TestRoundTrip` verifies all fields including `Closed` pointer, `BlockedBy` slice, timestamps | PASS - `TestReadJSONL/"it round-trips all task fields without loss"` verifies all fields identically |
| Optional fields omitted when empty/null | PASS - `TestOmitOptionalFields` checks absence of `"description"`, `"blocked_by"`, `"parent"`, `"closed"`, and `"null"` in output. Achieved via `taskJSON` struct with `omitempty` tags and `string` type for `Closed` | PASS - `TestWriteJSONL/"it omits optional fields when empty"` checks same fields. Achieved via `omitempty` tags directly on `Task` struct |
| Atomic write: temp file + fsync + rename | PASS - `WriteTasks` creates temp file, uses `bufio.Writer` + `Flush` + `Sync` + `Close` + `Rename`. Deferred cleanup on error. `TestAtomicWrite` verifies full rewrite and no leftover temp files | PASS - `WriteJSONL` creates temp file, writes directly (no `bufio.Writer`), `Sync` + `Close` + `Rename`. Deferred cleanup on error. `TestWriteJSONL/"it writes atomically via temp file and rename"` verifies same |
| Empty file returns empty task list | PASS - `TestEmptyFile` creates 0-byte file, asserts `len(tasks) == 0` | PASS - `TestReadJSONL/"it returns empty list for empty file"` creates 0-byte file, asserts `len(tasks) == 0`. Also explicitly asserts `tasks != nil` (non-nil empty slice) |
| Each task on exactly one line | PASS - `TestWriteTasks/"it writes tasks as one JSON object per line"` verifies 2 tasks produce 2 lines, each valid JSON. `TestHandleAllFieldsPopulated` verifies 1 task = 1 line | PASS - `TestWriteJSONL/"it writes tasks as one JSON object per line"` verifies 2 tasks produce 2 lines, checks `{` prefix and `}` suffix |
| JSONL field ordering matches spec | PASS - `TestFieldOrdering` checks positional ordering of all 10 field keys in output. Ordering guaranteed by `taskJSON` struct field order | PASS - `TestWriteJSONL/"it outputs fields in spec order"` checks identical positional ordering. Ordering guaranteed by `Task` struct field order in `task.go` |

## Implementation Comparison

### Approach

**Package placement:**
- V2 places the JSONL code in a dedicated `internal/storage/jsonl` package, importing `internal/task`. This creates a clear separation between the data model and its serialization.
- V4 places the JSONL code directly in `internal/task` package (`internal/task/jsonl.go`). This co-locates serialization logic with the data model.

V4's approach is simpler and avoids cross-package imports, but V2's approach provides better separation of concerns. Neither is definitively better for a project this size.

**Serialization strategy -- the core architectural difference:**

V2 introduces an intermediate `taskJSON` struct that controls both field ordering and timestamp format:

```go
// V2: internal/storage/jsonl/jsonl.go
type taskJSON struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      string   `json:"status"`
    Priority    int      `json:"priority"`
    Description string   `json:"description,omitempty"`
    BlockedBy   []string `json:"blocked_by,omitempty"`
    Parent      string   `json:"parent,omitempty"`
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

Note that `Created`, `Updated`, and `Closed` are `string` fields in `taskJSON`, while they are `time.Time`/`*time.Time` in the `Task` struct. V2 manually formats timestamps using a custom `timeFormat = "2006-01-02T15:04:05Z"` constant, with explicit `toJSON()` and `fromJSON()` conversion functions (lines 37-82).

V4 relies entirely on Go's `encoding/json` default behavior with the `Task` struct's own JSON tags:

```go
// V4: internal/task/jsonl.go
func WriteJSONL(path string, tasks []Task) error {
    // ...
    for _, t := range tasks {
        data, err := json.Marshal(t)  // Direct marshal of Task
        // ...
    }
}
```

V4 has no intermediate struct, no custom time formatting, and no conversion functions. It trusts `time.Time`'s built-in `MarshalJSON()` method (which produces RFC 3339 format: `"2006-01-02T15:04:05.999999999Z07:00"`).

**This is a genuinely important difference.** V2's explicit time format (`2006-01-02T15:04:05Z`) matches the spec's required ISO 8601 UTC format exactly. V4 relies on `time.Time`'s default marshaling, which could produce nanosecond precision or timezone offsets (e.g., `"2026-01-19T10:00:00Z"` vs potentially `"2026-01-19T10:00:00.000000000Z"`). In practice, Go's `time.Time.MarshalJSON` truncates trailing zeros, so for second-precision UTC times, the output is identical. However, V2's approach is more defensively correct -- it guarantees the exact format regardless of the input time's precision or timezone.

**Write buffering:**

V2 uses a `bufio.Writer` wrapping the temp file:

```go
// V2: internal/storage/jsonl/jsonl.go lines 103-120
writer := bufio.NewWriter(tmpFile)
for _, t := range tasks {
    data, err := json.Marshal(toJSON(t))
    // ...
    if _, err := writer.Write(data); err != nil { ... }
    if _, err := writer.WriteString("\n"); err != nil { ... }
}
if err := writer.Flush(); err != nil { ... }
```

V4 writes directly to the `*os.File`:

```go
// V4: internal/task/jsonl.go lines 31-42
for _, t := range tasks {
    data, err := json.Marshal(t)
    // ...
    if _, err := tmp.Write(data); err != nil { ... }
    if _, err := tmp.WriteString("\n"); err != nil { ... }
}
```

V2's buffered approach is marginally more efficient for large task lists (fewer syscalls), but for the expected workload this is a negligible difference.

**Temp file naming:**

- V2: `os.CreateTemp(dir, "."+base+".tmp*")` -- e.g., `.tasks.jsonl.tmp123456`
- V4: `os.CreateTemp(dir, ".tasks-*.jsonl.tmp")` -- e.g., `.tasks-123456.jsonl.tmp`

Both are valid patterns. V2's pattern is derived from the actual filename which is slightly more flexible if the path isn't always `tasks.jsonl`.

**Nil vs empty slice on empty file:**

V2's `ReadTasks` returns `nil` for an empty file (since `tasks` is never appended to):

```go
// V2: ReadTasks returns nil when no tasks parsed
var tasks []task.Task
// ... scanner loop never appends ...
return tasks, nil  // returns nil
```

V4's `ReadJSONL` explicitly converts `nil` to an empty slice:

```go
// V4: ReadJSONL lines 88-90
if tasks == nil {
    tasks = []Task{}
}
return tasks, nil  // returns []Task{}, never nil
```

V4's approach is genuinely better. Returning a non-nil empty slice is more idiomatic Go for "zero results" vs "no data". V4's test also explicitly verifies this: `if tasks == nil { t.Error("expected empty slice, got nil") }`.

**Function naming:**

- V2: `WriteTasks(path, tasks)`, `ReadTasks(path)` -- generic names in a `jsonl` package, so the full call is `jsonl.WriteTasks(...)`.
- V4: `WriteJSONL(path, tasks)`, `ReadJSONL(path)` -- format-specific names in the `task` package, so the full call is `task.WriteJSONL(...)`.

Both are reasonable. V2's naming relies on the package name for context (idiomatic Go), while V4 encodes the format in the function name since it's in the broader `task` package.

### Code Quality

**Error handling:**

Both versions use consistent wrapped errors with `%w`. V2 and V4 are essentially identical in error handling quality:

```go
// V2
return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
// V4
return nil, fmt.Errorf("failed to parse task on line %d: %w", lineNum, err)
```

Both include line numbers in parse errors. Both use deferred cleanup with the `success` flag pattern for atomic writes.

**Deferred cleanup pattern:**

Both versions use an identical deferred cleanup pattern for the temp file:

```go
// Both V2 and V4
success := false
defer func() {
    if !success {
        tmpFile.Close()  // or tmp.Close()
        os.Remove(tmpPath)
    }
}()
```

This is the correct crash-safe pattern: the temp file is always cleaned up unless the rename succeeds.

**V2's explicit time parsing adds robustness but also boilerplate:**

V2's `fromJSON` function (lines 55-82) contains explicit timestamp validation:

```go
// V2: fromJSON validates timestamps on read
created, err := time.Parse(timeFormat, j.Created)
if err != nil {
    return task.Task{}, fmt.Errorf("invalid created timestamp %q: %w", j.Created, err)
}
```

V4 relies on `json.Unmarshal` to handle time parsing via `time.Time.UnmarshalJSON`. If the JSON contains a malformed timestamp, V4 will still get an error -- but the error message will be less specific (a generic JSON unmarshal error vs V2's "invalid created timestamp").

**Type safety:**

V2 converts `task.Status` to `string` in `taskJSON` (`Status string`) and back to `task.Status` via type cast. V4 lets `json.Marshal`/`json.Unmarshal` handle `Status` directly through its underlying `string` type. Both work correctly; V2's explicit conversion adds a tiny amount of unnecessary ceremony.

**DRY:**

V2 is less DRY due to the `taskJSON` struct duplicating every field from `Task` with different types for timestamps and status. This means any field added to `Task` must also be added to `taskJSON`, `toJSON()`, and `fromJSON()` -- three places to update instead of one.

V4 has zero duplication -- it uses `Task` directly for marshaling.

**LOC comparison:** V2's implementation is 183 lines vs V4's 99 lines. The 84-line difference is almost entirely due to the `taskJSON` struct and its `toJSON`/`fromJSON` conversion functions.

### Test Quality

**V2 Test Functions (10 total, in `internal/storage/jsonl/jsonl_test.go`, 523 lines):**

1. `TestWriteTasks/"it writes tasks as one JSON object per line"` -- writes 2 tasks, verifies 2 lines, checks each is valid JSON via `json.Valid()`
2. `TestReadTasks/"it reads tasks from JSONL file"` -- manually writes JSONL content, reads back, verifies task IDs
3. `TestRoundTrip/"it round-trips all task fields without loss"` -- all fields populated including `Closed`, `BlockedBy` with 2 entries; individual field comparisons
4. `TestOmitOptionalFields/"it omits optional fields when empty"` -- minimal task, checks absence of 4 optional field keys and `"null"` string
5. `TestEmptyFile/"it returns empty list for empty file"` -- 0-byte file, asserts `len(tasks) == 0`
6. `TestMissingFile/"it returns error for missing file"` -- nonexistent path, asserts error is non-nil
7. `TestAtomicWrite/"it writes atomically via temp file and rename"` -- writes, overwrites, verifies full rewrite, checks no temp files left
8. `TestHandleAllFieldsPopulated/"it handles tasks with all fields populated"` -- all fields present including description with embedded newlines, verifies all field keys in JSON, verifies single-line output
9. `TestHandleOnlyRequiredFields/"it handles tasks with only required fields"` -- minimal task, round-trips, verifies all optional fields are empty/nil/zero
10. `TestFieldOrdering/"it outputs fields in spec order..."` -- full task, checks positional ordering of 10 field keys
11. `TestReadSkipsEmptyLines/"it skips empty lines when reading"` -- JSONL with blank line between tasks, verifies both parsed

(Actually 11 test functions, counted by `func Test*`)

**V2 Edge Cases Tested:**
- Valid JSON per line
- Empty file (0 bytes)
- Missing file
- Empty lines between tasks
- All fields populated (including newlines in description)
- Only required fields
- Full rewrite (no append)
- No leftover temp files
- Field ordering
- No `null` in output

**V4 Test Functions (10 subtests across 2 top-level test functions, in `internal/task/jsonl_test.go`, 495 lines):**

Under `TestWriteJSONL`:
1. `"it omits optional fields when empty"` -- checks absence of 4 optional fields AND presence of 6 required fields
2. `"it writes atomically via temp file and rename"` -- writes, overwrites, verifies via `ReadJSONL`, checks no temp files
3. `"it outputs fields in spec order"` -- checks positional ordering of 10 field keys
4. `"it writes tasks as one JSON object per line"` -- writes 2 tasks, verifies 2 lines, checks `{`/`}` prefix/suffix

Under `TestReadJSONL`:
5. `"it round-trips all task fields without loss"` -- all fields including `Closed`, uses `reflect.DeepEqual` for `BlockedBy`
6. `"it returns empty list for empty file"` -- 0-byte file, asserts `len == 0` AND `tasks != nil`
7. `"it returns error for missing file"` -- nonexistent path
8. `"it handles tasks with all fields populated"` -- all fields, round-trips, also reads raw JSON to verify field keys present
9. `"it handles tasks with only required fields"` -- minimal task, verifies optional fields are empty/nil
10. `"it skips empty lines in JSONL file"` -- JSONL with blank lines (2 empty lines), verifies both tasks parsed
11. `"it reads tasks from JSONL file"` -- basic read of 2-task JSONL

(11 subtests across 2 top-level functions)

**Test Coverage Diff:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Valid JSON per line (`json.Valid`) | Yes | No (checks `{`/`}` instead) |
| Empty file returns empty list | Yes | Yes |
| Empty file returns non-nil slice | No | Yes |
| Missing file returns error | Yes | Yes |
| Empty lines skipped | Yes | Yes |
| All fields round-trip | Yes | Yes |
| Only required fields round-trip | Yes | Yes |
| All fields present in JSON | Yes | Yes |
| No temp files left behind | Yes | Yes |
| Full rewrite (not append) | Yes | Yes |
| Field ordering | Yes | Yes |
| No `null` in output | Yes | No |
| Required fields present in omit test | No | Yes |
| Description with embedded newlines | Yes | No |
| `BlockedBy` comparison via `reflect.DeepEqual` | No (manual loop) | Yes |
| Multiple `BlockedBy` entries (2) | Yes (2 entries) | V4 all-fields test has 2 entries too |

**Differences in assertion style:**

V2 uses individual field comparisons in the round-trip test:
```go
// V2
for i, dep := range got.BlockedBy {
    if dep != original[0].BlockedBy[i] {
        t.Errorf("BlockedBy[%d] = %q, want %q", ...)
    }
}
```

V4 uses `reflect.DeepEqual` for `BlockedBy`:
```go
// V4
if !reflect.DeepEqual(got.BlockedBy, want.BlockedBy) {
    t.Errorf("BlockedBy: got %v, want %v", ...)
}
```

V4's approach is more concise and handles nil vs empty slice correctly.

**Test organization:**

V2 uses one top-level `func Test*` per logical scenario (11 functions). V4 groups tests under two top-level functions (`TestWriteJSONL` and `TestReadJSONL`), each containing multiple subtests. V4's grouping is slightly better organized -- it groups by operation (write vs read).

**Unique V2 test detail:** The `TestHandleAllFieldsPopulated` test uses a description with embedded newlines (`"A detailed description\nwith newlines"`), which is a good edge case for JSONL format (newlines in values must be JSON-escaped, not literal newlines in the file). V4 does not test this.

**Unique V4 test detail:** The empty file test explicitly asserts the returned slice is non-nil (`if tasks == nil { t.Error(...) }`), verifying the nil-to-empty-slice conversion. V2 does not verify this (and V2's implementation would return `nil` rather than `[]Task{}`).

**V4 also additionally checks required field presence** in its omit-optional-fields test:
```go
// V4 only
requiredFields := []string{`"id"`, `"title"`, `"status"`, `"priority"`, `"created"`, `"updated"`}
for _, field := range requiredFields {
    if !strings.Contains(line, field) {
        t.Errorf("expected required field %s to be present", field, line)
    }
}
```

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 4 | 4 |
| Lines added | 709 | 597 |
| Impl LOC | 183 | 99 |
| Test LOC | 523 | 495 |
| Test functions | 11 (top-level) | 2 top-level, 11 subtests |

## Verdict

**V4 is the better implementation**, though the margin is not enormous.

**V4's advantages:**
1. **Dramatically simpler implementation** (99 vs 183 LOC). V4 eliminates the entire `taskJSON` intermediate struct and its `toJSON`/`fromJSON` conversion functions -- 84 lines of boilerplate that duplicates the `Task` struct's field definitions. Any future field additions to `Task` require changes in one place (V4) vs three places (V2).
2. **Non-nil empty slice guarantee.** V4 explicitly converts nil to `[]Task{}` and tests for it. This is more robust for downstream consumers.
3. **Uses `reflect.DeepEqual`** for slice comparison in tests, which is more idiomatic and handles edge cases better.
4. **Better test organization** -- grouping subtests under `TestWriteJSONL`/`TestReadJSONL` makes the test structure clearer.
5. **Also validates required field presence** in the omit test, providing bidirectional coverage.

**V2's advantages:**
1. **Explicit timestamp format control.** V2's `const timeFormat = "2006-01-02T15:04:05Z"` with manual formatting guarantees exact output format regardless of input time precision. V4 relies on `time.Time.MarshalJSON()` which uses RFC 3339 and could theoretically produce sub-second precision if input times have it. In practice this is unlikely to cause issues since times are created with `Truncate(time.Second)`, but V2's approach is more defensively correct.
2. **Checks for `"null"` in omit test.** V2 explicitly asserts `null` doesn't appear anywhere in the output -- V4 does not test this.
3. **Tests embedded newlines in description.** This exercises an important JSONL edge case that V4 skips.
4. **Buffered writes** via `bufio.Writer` -- marginally more efficient for large files.
5. **Better package separation** (`internal/storage/jsonl` vs everything in `internal/task`).

The timestamp format control is V2's strongest point, but it comes at a steep cost in code duplication. For a project where `time.Time` values are always created with second precision in UTC (as the `NewTask` function does), V4's reliance on default marshaling is pragmatic and correct. V4's overall simplicity, better nil-safety, and lower maintenance burden make it the stronger implementation.
