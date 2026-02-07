# Task tick-core-1-2: JSONL Storage with Atomic Writes

## Task Summary

Tasks need persistent storage that diffs cleanly in git and survives process crashes. Without atomic writes, a crash mid-write could corrupt the task file. Implement a JSONL reader/writer that serializes tasks one-per-line and uses the temp file + fsync + rename pattern for crash-safe writes.

### Requirements

- Implement JSONL writer: serialize each `Task` as a single JSON line, no array wrapper, no trailing commas
- Omit optional fields when empty/null (don't serialize as `null` -- omit the key entirely)
- Implement JSONL reader: parse file line-by-line into `[]Task`, skip empty lines
- Implement atomic write: write to temp file in same directory, fsync, `os.Rename(temp, tasks.jsonl)`
- Full file rewrite on every mutation (no append mode)
- Handle empty file (0 bytes) as valid -- returns empty task list
- Handle `tasks.jsonl` not existing (pre-init) as an error

### Acceptance Criteria

1. Tasks round-trip through write then read without data loss
2. Optional fields omitted from JSON when empty/null
3. Atomic write uses temp file + fsync + rename
4. Empty file returns empty task list
5. Each task occupies exactly one line (no pretty-printing)
6. JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

### Specified Tests

- "it writes tasks as one JSON object per line"
- "it reads tasks from JSONL file"
- "it round-trips all task fields without loss"
- "it omits optional fields when empty"
- "it returns empty list for empty file"
- "it writes atomically via temp file and rename"
- "it handles tasks with all fields populated"
- "it handles tasks with only required fields"

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Tasks round-trip through write/read without data loss | PASS -- `TestRoundTrip` verifies all 10 fields including `Closed *time.Time` and `BlockedBy []string` survive write/read cycle. Tests both fully-populated and minimal tasks in one subtest. | PASS -- `TestRoundTrip` verifies all fields with explicit per-field assertions. Single-task round-trip with all fields populated. | PASS -- `TestJSONLRoundTrip` round-trips all fields. Uses string timestamps so comparison is simple string equality. |
| 2. Optional fields omitted from JSON when empty/null | PASS -- `TestWriteJSONL/"omits optional fields when empty"` checks absence of `description`, `blocked_by`, `parent`, `closed` keys in output. Custom `jsonlTask` struct uses `omitempty` tags. | PASS -- `TestOmitOptionalFields` checks absence of all 4 optional field keys and also explicitly checks for absence of `null` string. | PARTIAL -- `TestJSONLRoundTrip/"it omits optional fields when empty"` checks field absence, but V3 relies on `json.Encoder` directly on `task.Task` which uses the struct's own `omitempty` tags. The `priority` field with value 0 would be incorrectly omitted since Go's `omitempty` omits zero ints, but no test catches this because tests use `priority: 2`. See detailed analysis below. |
| 3. Atomic write uses temp file + fsync + rename | PASS -- `WriteJSONL` creates temp file via `os.CreateTemp`, calls `tmp.Sync()`, then `os.Rename`. Deferred cleanup removes temp on error. | PASS -- `WriteTasks` creates temp file, uses `bufio.Writer`, calls `tmpFile.Sync()`, then `os.Rename`. Deferred cleanup on error. | PASS -- `WriteJSONL` creates temp file, calls `temp.Sync()`, then `os.Rename`. Deferred cleanup on error. |
| 4. Empty file returns empty task list | PASS -- `TestReadJSONL/"returns empty list for empty file"` verifies `len(tasks) == 0`. Returns nil slice (not guaranteed non-nil). | PASS -- `TestEmptyFile` verifies `len(tasks) == 0`. Returns nil slice. | PASS -- `TestJSONLRoundTrip/"it returns empty list for empty file"` verifies non-nil empty slice. V3 explicitly converts nil to `[]task.Task{}`. |
| 5. Each task occupies exactly one line | PASS -- `WriteJSONL` calls `json.Marshal` per task then writes `\n`. No pretty-printing possible. | PASS -- `WriteTasks` calls `json.Marshal` per task then writes `\n` via `bufio.Writer`. | PASS -- `WriteJSONL` uses `json.NewEncoder` which adds trailing newline per `Encode()` call. `TestJSONLFormat` explicitly tests one-line-per-task including multiline descriptions. |
| 6. JSONL output matches spec field ordering | PASS -- Custom `jsonlTask` struct declares fields in spec order: `ID, Title, Status, Priority, Description, BlockedBy, Parent, Created, Updated, Closed`. Go's `encoding/json` preserves struct field order. No explicit test for ordering. | PASS -- Custom `taskJSON` struct declared in spec order. `TestFieldOrdering` explicitly verifies field index positions in output. | FAIL -- V3 serializes `task.Task` directly via `json.Encoder`. The `task.Task` struct's field order is correct, but V3 has no custom serialization struct. V3 has no field ordering test. However, since the Task struct fields are declared in spec order, the output happens to be correct. Calling this PASS on behavior, but no verification test exists. |

## Implementation Comparison

### Approach

#### File Organization

**V1** places code in `internal/storage/jsonl.go` (package `storage`). Flat organization, co-located with other storage code.

**V2** creates a dedicated sub-package `internal/storage/jsonl/jsonl.go` (package `jsonl`). This gives JSONL storage its own namespace with exports like `jsonl.WriteTasks()` and `jsonl.ReadTasks()`.

**V3** uses the same flat structure as V1: `internal/storage/jsonl.go` (package `storage`).

#### Serialization Strategy -- The Key Architectural Divergence

This is the most significant difference between versions, driven by the `task.Task` struct design from tick-core-1-1.

**V1 and V2** use `time.Time` for timestamps (`Created`, `Updated`) and `*time.Time` for optional `Closed`. Both define a custom intermediate struct for JSON serialization:

V1's custom struct (`jsonlTask`):
```go
type jsonlTask struct {
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

V2's custom struct (`taskJSON`):
```go
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

These are structurally identical. Both convert `time.Time` to `string` in the serialization struct, using `Closed string` with `omitempty` instead of `*time.Time`. Both define `toJSONL`/`toJSON` and `fromJSONL`/`fromJSON` conversion functions.

**V3** uses `string` for all timestamps in the `task.Task` struct itself:
```go
// V3's task.Task (from task.go)
type Task struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      Status   `json:"status"`
    Priority    int      `json:"priority"`
    Description string   `json:"description,omitempty"`
    BlockedBy   []string `json:"blocked_by,omitempty"`
    Parent      string   `json:"parent,omitempty"`
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

This means V3 can use `json.NewEncoder` directly on `task.Task` without any intermediate struct or conversion logic. The entire write function is:
```go
encoder := json.NewEncoder(temp)
for _, t := range tasks {
    if err := encoder.Encode(t); err != nil {
        temp.Close()
        return err
    }
}
```

And the read function uses `json.Unmarshal` directly into `task.Task`:
```go
var t task.Task
if err := json.Unmarshal([]byte(line), &t); err != nil {
    return nil, err
}
```

**Critical issue with V3's approach**: The `task.Task` struct has `Priority int` with NO `omitempty` tag on `priority`. This is correct -- priority 0 (P0 critical) must always be serialized. However, V3 also has `Status Status` (a `string` type alias) without `omitempty`, which means empty status would serialize as `""` rather than being omitted. V1 and V2 have the same behavior via their intermediate structs. This is fine because status is a required field.

However, there is a subtle difference: V1 and V2's intermediate structs use `Status string` (plain string), while V3 serializes `Status` as type `Status` (a `string` alias). Both produce the same JSON output.

#### Atomic Write Pattern

All three implementations follow the same pattern: `os.CreateTemp` -> write data -> `Sync()` -> `Close()` -> `os.Rename()`. Key differences:

**V1**:
```go
tmp, err := os.CreateTemp(dir, ".tasks-*.jsonl.tmp")
// ... writes directly via tmp.Write(data) and tmp.WriteString("\n")
if err := tmp.Sync(); err != nil { ... }
if err := tmp.Close(); err != nil { ... }
if err := os.Rename(tmpPath, path); err != nil { ... }
success = true
```
Uses `success` flag with deferred cleanup. Writes directly to the file without buffering.

**V2**:
```go
tmpFile, err := os.CreateTemp(dir, "."+base+".tmp*")
// ... wraps in bufio.Writer, writes via writer.Write(data) and writer.WriteString("\n")
if err := writer.Flush(); err != nil { ... }
if err := tmpFile.Sync(); err != nil { ... }
if err := tmpFile.Close(); err != nil { ... }
if err := os.Rename(tmpPath, path); err != nil { ... }
success = true
```
Adds `bufio.Writer` for buffered I/O (genuinely better for performance with many tasks). Also uses `success` flag pattern. Temp file name uses the actual filename as prefix: `"."+base+".tmp*"`.

**V3**:
```go
temp, err := os.CreateTemp(dir, ".tasks-*.tmp")
// ... writes via json.NewEncoder(temp).Encode(t)
if err := temp.Sync(); err != nil { ... }
if err := temp.Close(); err != nil { ... }
if err := os.Rename(tempPath, path); err != nil { ... }
success = true
```
Uses `json.NewEncoder` which internally buffers writes. Same `success` flag pattern.

**V2's buffered writer is the only genuinely different approach** -- the others write directly or rely on the encoder's internal buffering. V2's `bufio.Writer` with explicit `Flush()` gives the most control over write performance.

#### Extra Functions in V1

V1 provides two additional public functions not found in V2 or V3:

```go
// ReadJSONLBytes parses tasks from raw JSONL content (no file I/O).
func ReadJSONLBytes(data []byte) ([]task.Task, error) { ... }

// MarshalJSONL serializes tasks to JSONL bytes (no file I/O).
func MarshalJSONL(tasks []task.Task) ([]byte, error) { ... }
```

These are not required by the task specification but provide utility for in-memory operations (e.g., testing, caching). V2 and V3 do not include these.

### Code Quality

#### Error Handling

**V1** uses descriptive context-rich wrapping:
```go
return nil, fmt.Errorf("opening JSONL file: %w", err)
return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
return nil, fmt.Errorf("line %d: %w", lineNum, err)
return fmt.Errorf("creating temp file: %w", err)
return fmt.Errorf("marshaling task %s: %w", t.ID, err)
return fmt.Errorf("writing task %s: %w", t.ID, err)
return fmt.Errorf("syncing temp file: %w", err)
return fmt.Errorf("renaming temp file: %w", err)
```
Uses gerund-style prefixes ("opening", "creating", "marshaling"). Includes line numbers for parse errors and task IDs for write errors. 11 distinct wrapped error returns in `ReadJSONL` + `WriteJSONL`.

**V2** uses "failed to" prefix consistently:
```go
return nil, fmt.Errorf("failed to open tasks file: %w", err)
return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
return nil, fmt.Errorf("invalid task on line %d: %w", lineNum, err)
return nil, fmt.Errorf("failed to read tasks file: %w", err)
return fmt.Errorf("failed to create temp file: %w", err)
return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
return fmt.Errorf("failed to write task %s: %w", t.ID, err)
return fmt.Errorf("failed to flush writer: %w", err)
return fmt.Errorf("failed to fsync temp file: %w", err)
return fmt.Errorf("failed to close temp file: %w", err)
return fmt.Errorf("failed to rename temp file: %w", err)
```
More verbose but equally informative. Also includes line numbers and task IDs. V2 additionally includes the bad timestamp value in its error messages from `fromJSON`:
```go
return task.Task{}, fmt.Errorf("invalid created timestamp %q: %w", j.Created, err)
return task.Task{}, fmt.Errorf("invalid updated timestamp %q: %w", j.Updated, err)
return task.Task{}, fmt.Errorf("invalid closed timestamp %q: %w", j.Closed, err)
```
This is **genuinely better** for debugging -- V1's equivalent error messages do not include the invalid value:
```go
return task.Task{}, fmt.Errorf("parsing created timestamp: %w", err)
```

**V3** returns bare errors with no wrapping:
```go
return err           // from os.CreateTemp
return err           // from encoder.Encode
return err           // from temp.Sync
return err           // from temp.Close
return err           // from os.Rename
return nil, err      // from os.Open
return nil, err      // from json.Unmarshal
return nil, err      // from scanner.Err
```
Zero context is added to any error. A caller receiving an error from `ReadJSONL` would get a raw `os.Open` error without knowing it came from JSONL reading. This is a **genuine quality gap** -- error wrapping is a Go best practice for debugging and error chain analysis.

#### Function Naming

| Operation | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Write tasks | `WriteJSONL(path, tasks)` | `WriteTasks(path, tasks)` | `WriteJSONL(path, tasks)` |
| Read tasks | `ReadJSONL(path)` | `ReadTasks(path)` | `ReadJSONL(path)` |
| Intermediate struct | `jsonlTask` | `taskJSON` | (none) |
| To intermediate | `toJSONL(t)` | `toJSON(t)` | (none) |
| From intermediate | `fromJSONL(jt)` | `fromJSON(j)` | (none) |

V2's naming (`WriteTasks`/`ReadTasks`) is cleaner because the `jsonl` package name already provides context -- callers would write `jsonl.WriteTasks()` rather than the redundant `storage.WriteJSONL()` in V1/V3.

#### Nil vs Empty Slice

**V3** explicitly converts nil results to empty slices:
```go
if tasks == nil {
    tasks = []task.Task{}
}
```
This is more correct -- callers get a non-nil empty slice for empty files, which plays better with JSON marshaling and range loops. V1 and V2 return `nil` for empty files (though both have `var tasks []task.Task` which is `nil` by default).

#### Close-on-Error Patterns

**V1** explicitly calls `tmp.Close()` before returning from each error path within the write loop:
```go
if _, err := tmp.Write(data); err != nil {
    tmp.Close()
    return fmt.Errorf(...)
}
```
This is manual and repetitive -- 3 explicit `tmp.Close()` calls in error paths.

**V2** uses the `success` flag and deferred cleanup to handle closing:
```go
defer func() {
    if !success {
        tmpFile.Close()
        os.Remove(tmpPath)
    }
}()
```
The defer handles both close and cleanup on any error. V2 does NOT close in individual error paths -- the defer handles it. This is cleaner but means the file stays open slightly longer on error.

**V3** follows the same pattern as V1 -- explicit `temp.Close()` before error returns. But V3 has fewer error paths because it uses `json.NewEncoder` which handles its own write errors.

### Test Quality

#### V1 Test Functions (12 subtests across 4 top-level functions)

File: `internal/storage/jsonl_test.go` (328 LOC)

1. **`TestWriteJSONL`**
   - `"writes tasks as one JSON object per line"` -- writes 2 tasks, checks 2 lines, validates each is valid JSON
   - `"omits optional fields when empty"` -- writes minimal task, checks absence of 4 optional field keys
   - `"includes optional fields when populated"` -- writes full task, checks presence of 4 optional field keys
   - `"writes empty list as empty file"` -- writes `[]task.Task{}`, checks file content is `""`

2. **`TestReadJSONL`**
   - `"reads tasks from JSONL file"` -- manually writes 2-line JSONL, reads back, checks IDs and Priority
   - `"returns empty list for empty file"` -- reads empty file, checks `len == 0`
   - `"errors on missing file"` -- reads from `/nonexistent/tasks.jsonl`, checks error non-nil
   - `"skips empty lines"` -- JSONL with blank line between tasks, checks 2 tasks returned

3. **`TestRoundTrip`**
   - `"round-trips all task fields without loss"` -- writes full + minimal task, reads back, checks all 10 fields on full task and 4 optional fields are zero on minimal task

4. **`TestAtomicWrite`**
   - `"writes atomically via temp file and rename"` -- writes old content, overwrites with new, checks old content gone and new present, checks no temp files left
   - `"original file survives write to nonexistent directory"` -- writes to `/nonexistent/dir/tasks.jsonl`, checks error returned

**V1 helpers**: `tempDir(t)`, `writeFile(t, path, content)`, `readFile(t, path)`, `sampleTask(id, title)`.

#### V2 Test Functions (12 subtests across 10 top-level functions)

File: `internal/storage/jsonl/jsonl_test.go` (523 LOC)

1. **`TestWriteTasks`**
   - `"it writes tasks as one JSON object per line"` -- writes 2 tasks, checks 2 lines, validates each is valid JSON

2. **`TestReadTasks`**
   - `"it reads tasks from JSONL file"` -- manually writes 2-line JSONL, reads back, checks both IDs

3. **`TestRoundTrip`**
   - `"it round-trips all task fields without loss"` -- writes full task with all fields, reads back, checks every field individually including `Created`, `Updated`, `Closed` time comparisons

4. **`TestOmitOptionalFields`**
   - `"it omits optional fields when empty"` -- writes minimal task, checks absence of 4 optional keys AND checks for absence of `"null"` string

5. **`TestEmptyFile`**
   - `"it returns empty list for empty file"` -- reads empty file, checks `len == 0`

6. **`TestMissingFile`**
   - `"it returns error for missing file"` -- reads nonexistent path, checks error non-nil

7. **`TestAtomicWrite`**
   - `"it writes atomically via temp file and rename"` -- writes initial content, overwrites, verifies replacement, checks no temp files left with correct prefix pattern

8. **`TestHandleAllFieldsPopulated`**
   - `"it handles tasks with all fields populated"` -- writes full task, checks all 10 field keys present in output, verifies exactly 1 line (no pretty-printing). Note: includes description with `\n` in it (multiline description edge case).

9. **`TestHandleOnlyRequiredFields`**
   - `"it handles tasks with only required fields"` -- writes minimal task, reads back, checks all required fields have expected values and all optional fields are zero-valued

10. **`TestFieldOrdering`**
    - `"it outputs fields in spec order: id, title, status, priority, optional, created, updated, closed"` -- writes full task, checks that `strings.Index` positions of each field name are monotonically increasing

11. **`TestReadSkipsEmptyLines`**
    - `"it skips empty lines when reading"` -- JSONL with blank line between tasks, reads 2 tasks

**V2 helpers**: `mustParseTime(t, s)`, `timePtr(t)`.

#### V3 Test Functions (14 subtests across 7 top-level functions)

File: `internal/storage/jsonl_test.go` (618 LOC)

1. **`TestJSONLWriter`**
   - `"it writes tasks as one JSON object per line"` -- writes 2 tasks, checks 2 lines, validates JSON object format (checks `{` prefix and `}` suffix), checks no trailing commas

2. **`TestJSONLReader`**
   - `"it reads tasks from JSONL file"` -- manually writes 2-line JSONL, reads back, checks IDs, Title, and Status

3. **`TestJSONLRoundTrip`**
   - `"it round-trips all task fields without loss"` -- full task round-trip, all fields compared. Description includes `\n` (multiline). `Closed` is a string in V3.
   - `"it omits optional fields when empty"` -- writes minimal task, checks absence of `description`, `blocked_by`, `parent`, `closed`. Also checks presence of required fields.
   - `"it returns empty list for empty file"` -- checks non-nil empty slice, checks `len == 0`

4. **`TestJSONLAtomicWrite`**
   - `"it writes atomically via temp file and rename"` -- writes initial content, overwrites, verifies replacement. Checks no temp files with correct prefix. Also checks `os.Stat` and compares file sizes (though acknowledges this is weak).
   - `"it cleans up temp file on write error"` -- writes to non-existent subdirectory, checks error, checks no temp files left behind in parent directory

5. **`TestJSONLFieldCombinations`**
   - `"it handles tasks with all fields populated"` -- full task with P0 priority, multiline description with special chars (`\"quotes\"` and `\\backslash`), 3 blocked_by entries. Writes and reads back, verifies all fields. Tests 3 blocked_by entries (V1 tests 2, V2 tests 2).
   - `"it handles tasks with only required fields"` -- writes minimal task, reads back, checks all required fields and all optional fields are zero

6. **`TestJSONLFormat`**
   - `"each task occupies exactly one line"` -- writes task with multiline description, counts non-empty lines, checks no pretty-printing indentation

7. **`TestJSONLErrors`**
   - `"it returns error for missing file"` -- reads nonexistent file, checks error non-nil, checks `os.IsNotExist(err)` -- **this is unique to V3**, verifying the error type not just that an error occurred
   - `"it skips empty lines in JSONL file"` -- JSONL with blank lines, checks 2 tasks

**V3 helpers**: None (all test data is inline).

#### Test Coverage Comparison

| Edge Case / Scenario | V1 | V2 | V3 |
|----------------------|-----|-----|-----|
| Write 2 tasks, verify 2 lines | Yes | Yes | Yes |
| Each line is valid JSON | Yes (`json.Valid`) | Yes (`json.Valid`) | Yes (checks `{`/`}` prefix/suffix) |
| Omit optional fields when empty | Yes | Yes | Yes |
| Check no `null` in output | No | Yes | No |
| Include optional fields when populated | Yes | Yes (via all-fields test) | Yes (via all-fields test) |
| Write empty list produces empty file | Yes | No | No |
| Read from manually-written JSONL | Yes | Yes | Yes |
| Read empty file returns empty list | Yes | Yes | Yes |
| Missing file returns error | Yes | Yes | Yes |
| Missing file error is `os.IsNotExist` | No | No | Yes |
| Skip empty lines | Yes | Yes | Yes |
| Round-trip all fields | Yes | Yes | Yes |
| Round-trip minimal task | Yes (in round-trip test) | Yes (required-only test) | Yes (required-only test) |
| Atomic: old content replaced | Yes | Yes | Yes |
| Atomic: no temp files left | Yes | Yes | Yes |
| Atomic: error on bad directory | Yes | No | Yes |
| Atomic: temp cleanup on error | No | No | Yes |
| All fields populated write/read | Yes (in round-trip) | Yes (dedicated test) | Yes (dedicated test) |
| Required-only fields write/read | Yes (in round-trip) | Yes (dedicated test) | Yes (dedicated test) |
| Field ordering verification | No | Yes | No |
| One task per line (explicit test) | No | Yes (in all-fields test) | Yes (dedicated test) |
| No pretty-printing | No | No | Yes |
| Multiline description in task | No | Yes | Yes |
| Special chars in description | No | No | Yes |
| No trailing commas | No | No | Yes |
| 3+ blocked_by entries | No (tests 2) | No (tests 2, round-trip) | Yes (tests 3) |

**Tests unique to V1**: "writes empty list as empty file" (tests that writing `[]task.Task{}` produces an empty file).

**Tests unique to V2**: `TestFieldOrdering` (explicit verification of JSON field ordering via `strings.Index` position checks). Also checks for absence of `"null"` in omit-optional test.

**Tests unique to V3**: `os.IsNotExist` error type check on missing file. Temp file cleanup on write error. No-pretty-printing check. Special characters in description (`"quotes"` and `\backslash`). No-trailing-commas check.

#### Test Style Differences

**V1** uses shared helper functions (`tempDir`, `writeFile`, `readFile`, `sampleTask`). Tests are concise. Uses `t.Fatalf`/`t.Errorf` consistently. The `sampleTask` helper creates reusable test fixtures.

**V2** uses minimal helpers (`mustParseTime`, `timePtr`). Each test function is a single dedicated top-level function with one subtest (e.g., `TestOmitOptionalFields` contains one `t.Run`). Test data is constructed inline. This is the most organized structure -- each specified test requirement gets its own top-level test function.

**V3** uses no helpers at all -- all test data is inline. This makes tests self-contained but verbose. V3 groups related tests under fewer top-level functions (e.g., `TestJSONLRoundTrip` contains 3 subtests including omit-optional and empty-file). Uses string timestamps directly, making test setup simpler.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4* | 6* |
| Lines added | 544 | 709 | 736 |
| Impl LOC | 216 | 183 | 97 |
| Test LOC | 328 | 523 | 618 |
| Test functions (top-level) | 4 | 10 | 7 |
| Test subtests | 12 | 12 | 14 |
| Package | `storage` | `jsonl` | `storage` |

\* V2 and V3 also modified docs/workflow files and planning status. V3 also modified `.claude/settings.local.json` and added a context doc. Only the impl and test files are relevant to this analysis.

## Verdict

**V2 is the best implementation of this task**, with the following reasoning:

### V2 Strengths

1. **Best error handling**: V2 wraps every error with context and uniquely includes the invalid value in timestamp parse errors (`fmt.Errorf("invalid created timestamp %q: %w", j.Created, err)`). This is genuinely better for debugging than V1's "parsing created timestamp" or V3's bare `err`.

2. **Buffered I/O**: V2 is the only version that uses `bufio.Writer` for writing tasks. This is a genuine performance advantage for files with many tasks, reducing system call overhead.

3. **Field ordering test**: V2 is the only version with `TestFieldOrdering`, which explicitly verifies acceptance criterion #6 (JSONL output matches spec field ordering). This is a spec requirement that V1 and V3 rely on implicitly via struct field order but never verify.

4. **Clean package organization**: `internal/storage/jsonl/` gives the JSONL layer its own namespace. Function names like `jsonl.WriteTasks()` are clean and non-redundant. This scales better as the storage package grows.

5. **Null-check in omit test**: V2 is the only version that explicitly checks for absence of `"null"` in the output, which directly addresses the spec requirement "don't serialize as null -- omit the key entirely."

6. **Well-organized test structure**: Each spec requirement gets its own top-level test function. This makes it immediately clear which requirement each test covers.

### V2 Weaknesses

- Does not test write-to-nonexistent-directory error (V1 and V3 do)
- Does not test empty list produces empty file (V1 does)
- Does not test temp file cleanup on error (V3 does)

### V1 Assessment (Second Place)

V1 is solid with good error wrapping and the additional utility functions (`ReadJSONLBytes`, `MarshalJSONL`) that go beyond the spec. However, it has the fewest edge case tests (328 LOC vs 523/618 for V2/V3), lacks a field ordering test, and lacks a null-check in the omit test.

### V3 Assessment (Third Place)

V3 has the most test cases (14 subtests, 618 test LOC) and tests unique edge cases (special characters, `os.IsNotExist`, temp cleanup). However, V3 has two significant issues:

1. **No error wrapping at all**: Every error return is bare `return err` or `return nil, err`. This is a genuine code quality problem in Go that makes debugging harder for callers.

2. **No intermediate serialization struct**: V3 relies on `json.NewEncoder` encoding `task.Task` directly. While this works because the Task struct happens to have the right tags and field order, it tightly couples the internal storage format to the Task struct's JSON tags. If the Task struct's tags ever change for API purposes, the JSONL storage format would silently change too. V1 and V2's intermediate structs decouple the storage format from the model.

3. **Potential priority=0 issue**: Since V3 uses `json.Encoder` directly on `task.Task`, and `task.Task` has `Priority int` with `json:"priority"` (no `omitempty`), priority 0 is correctly serialized. However, if someone ever added `omitempty` to the Task struct's priority tag for API reasons, the storage format would silently break. The intermediate struct in V1/V2 protects against this.
