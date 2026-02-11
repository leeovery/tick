# Task tick-core-1-2: JSONL Storage with Atomic Writes

## Task Summary

Implement a JSONL reader/writer that serializes tasks one-per-line and uses the temp file + fsync + rename pattern for crash-safe writes. Specifically:

- Serialize each `Task` as a single JSON line, no array wrapper, no trailing commas
- Omit optional fields when empty/null (do not serialize as `null`)
- Parse file line-by-line into `[]Task`, skip empty lines
- Atomic write: write to temp file in same directory, fsync, os.Rename(temp, tasks.jsonl)
- Full file rewrite on every mutation (no append mode)
- Handle empty file (0 bytes) as valid (returns empty task list)
- Handle missing `tasks.jsonl` as an error

**Acceptance Criteria:**
1. Tasks round-trip through write then read without data loss
2. Optional fields omitted from JSON when empty/null
3. Atomic write uses temp file + fsync + rename
4. Empty file returns empty task list
5. Each task occupies exactly one line (no pretty-printing)
6. JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Tasks round-trip without data loss | PASS -- `TestRoundTrip` writes then reads, verifies all 10 fields | PASS -- `TestRoundTrip/"it round-trips all task fields without loss"` verifies all 10 fields |
| Optional fields omitted when empty/null | PASS -- `TestOptionalFieldOmission` checks absence of description, blocked_by, parent, closed; relies on `omitempty` tags on `task.Task` struct | PASS -- `TestRoundTrip/"it omits optional fields when empty"` checks absence of all four optional fields; uses `jsonlTask` DTO with `omitempty` tags |
| Atomic write: temp file + fsync + rename | PASS -- `WriteTasks` calls `os.CreateTemp`, `tmp.Sync()`, `tmp.Close()`, `os.Rename`; `TestAtomicWrite` verifies overwrite semantics and no temp files left | PASS -- `WriteJSONL` calls `os.CreateTemp`, `tmp.Sync()`, `tmp.Close()`, `os.Rename`; `TestAtomicWrite` verifies overwrite and no temp files; additionally tests write-error cleanup |
| Empty file returns empty task list | PASS -- `TestEmptyFile` creates 0-byte file, asserts 0 tasks | PASS -- `TestReadJSONL/"it returns empty list for empty file"` creates 0-byte file, asserts 0 tasks |
| Each task on exactly one line | PASS -- `TestWriteTasks` counts lines and checks no embedded newlines | PASS -- `TestWriteJSONL` counts lines and checks JSON object format |
| JSONL field ordering matches spec | PASS -- `TestFieldOrdering` checks positional ordering of all 10 field keys via `strings.Index` | PASS -- `TestJSONLFormat` performs exact string comparison against expected spec format for both minimal and fully-populated tasks |

## Implementation Comparison

### Approach

**V5: Direct JSON marshal/unmarshal via `Task` struct's custom methods**

V5 relies on the `MarshalJSON`/`UnmarshalJSON` methods defined on `task.Task` itself. The storage layer is a thin wrapper -- `WriteTasks` feeds tasks to `json.NewEncoder`, and `ReadTasks` uses `json.Unmarshal` per line. The Task struct has `json:"-"` tags on `Created`, `Updated`, and `Closed`, with a private `taskJSON` DTO inside `task.go` that handles timestamp formatting. The storage layer has zero awareness of timestamp formatting or field ordering.

```go
// V5 jsonl.go -- WriteTasks (line 36-40)
encoder := json.NewEncoder(tmp)
for _, t := range tasks {
    if err := encoder.Encode(t); err != nil {
        return fmt.Errorf("encoding task %s: %w", t.ID, err)
    }
}
```

```go
// V5 jsonl.go -- ReadTasks (line 73-79)
var t task.Task
if err := json.Unmarshal([]byte(line), &t); err != nil {
    return nil, fmt.Errorf("parsing task at line %d: %w", lineNum, err)
}
tasks = append(tasks, t)
```

Function names: `WriteTasks`, `ReadTasks`.

**V6: Intermediate `jsonlTask` DTO with explicit conversion functions**

V6 introduces a storage-layer-specific `jsonlTask` struct that duplicates the field layout and handles serialization independently from the `task.Task` type. Two conversion functions, `toJSONL` and `fromJSONL`, translate between `task.Task` and `jsonlTask`. Notably, in V6 the `task.Task` struct does NOT have `json:"-"` tags on its timestamp fields -- timestamps are plain `json:"created"` etc. This means V6's `task.Task` would serialize timestamps as Go's default time format (RFC 3339 with nanoseconds) if marshaled directly. The storage-layer DTO overrides this with explicit `task.FormatTimestamp()` calls.

```go
// V6 jsonl.go -- jsonlTask struct (lines 16-27)
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

```go
// V6 jsonl.go -- toJSONL conversion (lines 29-43)
func toJSONL(t task.Task) jsonlTask {
    jt := jsonlTask{
        ID:          t.ID,
        Title:       t.Title,
        Status:      string(t.Status),
        Priority:    t.Priority,
        Description: t.Description,
        BlockedBy:   t.BlockedBy,
        Parent:      t.Parent,
        Created:     task.FormatTimestamp(t.Created),
        Updated:     task.FormatTimestamp(t.Updated),
    }
    if t.Closed != nil {
        jt.Closed = task.FormatTimestamp(*t.Closed)
    }
    return jt
}
```

V6 uses `json.Marshal` + manual newline append instead of `json.NewEncoder`:

```go
// V6 jsonl.go -- WriteJSONL (lines 93-101)
for _, t := range tasks {
    data, err := json.Marshal(toJSONL(t))
    if err != nil {
        return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
    }
    data = append(data, '\n')
    if _, err := tmp.Write(data); err != nil {
        return fmt.Errorf("failed to write task %s: %w", t.ID, err)
    }
}
```

Function names: `WriteJSONL`, `ReadJSONL`.

**Key Architectural Difference:**

V5 keeps serialization logic centralized on the `Task` type via `MarshalJSON`/`UnmarshalJSON`, making the storage layer simple and decoupled. V6 creates a separate serialization layer with its own DTO, duplicating field mapping. V5's approach means any consumer that marshals a `Task` gets correct formatting. V6's approach means only the JSONL storage layer controls formatting, while direct `json.Marshal(task)` would produce different output.

This is a genuine design tradeoff: V5's approach is more DRY and consistent (the Task always serializes the same way). V6's approach offers more flexibility if different serialization formats are needed later, but at the cost of duplication and potential inconsistency. For this particular spec -- which defines exactly one JSONL format -- V5's centralized approach is cleaner.

### Code Quality

**Error Messages:**

V5 uses gerund-style error prefixes (`"creating temp file:"`, `"encoding task:"`, `"opening tasks file:"`). V6 uses `"failed to"` style (`"failed to create temp file:"`, `"failed to marshal task:"`). Both wrap errors with `%w`. V5's style is more idiomatic Go per the Go error conventions (errors are typically lowercase and describe the action in progress, not prefixed with "failed to").

```go
// V5 -- idiomatic Go error style
return fmt.Errorf("creating temp file: %w", err)
return fmt.Errorf("encoding task %s: %w", t.ID, err)

// V6 -- "failed to" prefix style
return fmt.Errorf("failed to create temp file: %w", err)
return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
```

**DRY Principle:**

V5's implementation is 86 lines. V6's is 161 lines -- nearly double. The extra 75 lines come from the `jsonlTask` struct (12 lines), `toJSONL` (15 lines), `fromJSONL` (30 lines), and more verbose error paths. V5 avoids this duplication entirely because the `task.Task` type handles its own serialization.

**Type Safety:**

V6's `jsonlTask.Status` is `string` rather than `task.Status`:
```go
// V6 jsonlTask
Status string `json:"status"`
```
This loses type safety during the conversion. In `fromJSONL`, V6 does a raw cast `task.Status(jt.Status)` without validation:
```go
// V6 fromJSONL (line 64)
Status: task.Status(jt.Status),
```
V5 gets validation automatically because `task.Task.Status` is `task.Status` and the struct tags are on the task type directly.

**Naming:**

V5: `WriteTasks`, `ReadTasks` -- clear, concise, describes the domain operation.
V6: `WriteJSONL`, `ReadJSONL` -- names the format rather than the domain concept. This is a style choice; for a package named `storage` that currently only has one format, V5's naming reads slightly better (`storage.WriteTasks` vs `storage.WriteJSONL`).

**Package Documentation:**

Both provide package-level doc comments.
- V5: `"Package storage provides JSONL-based persistent storage for tasks with atomic write support using the temp file + fsync + rename pattern."`
- V6: `"Package storage provides JSONL persistence for Tick tasks with atomic writes."`

Both are adequate; V5 is more descriptive.

**Cleanup Pattern:**

Both use the identical `success` flag with deferred cleanup pattern:

```go
// Both versions (identical pattern)
success := false
defer func() {
    if !success {
        tmp.Close()
        os.Remove(tmpPath)
    }
}()
```

This is correct and handles error paths well.

### Test Quality

**V5 Test Functions (11 total):**

1. `TestWriteTasks/"it writes tasks as one JSON object per line"` -- writes 2 tasks, checks line count = 2, verifies no pretty-printing, checks task IDs in correct lines
2. `TestReadTasks/"it reads tasks from JSONL file"` -- writes raw JSONL, reads back, verifies 2 tasks with correct fields
3. `TestRoundTrip/"it round-trips all task fields without loss"` -- full task with all 10 fields, write then read, field-by-field comparison
4. `TestOptionalFieldOmission/"it omits optional fields when empty"` -- minimal task, checks absence of description/blocked_by/parent/closed keys, checks presence of required keys
5. `TestEmptyFile/"it returns empty list for empty file"` -- 0-byte file, asserts 0 tasks
6. `TestMissingFile/"it returns error for missing file"` -- nonexistent path, asserts error
7. `TestAtomicWrite/"it writes atomically via temp file and rename"` -- write initial, verify, overwrite, verify new content replaced old, check no .tmp files
8. `TestAllFieldsPopulated/"it handles tasks with all fields populated"` -- special chars in description, priority 0, 2-element BlockedBy; checks JSON contains all optional field keys; round-trips and verifies
9. `TestOnlyRequiredFields/"it handles tasks with only required fields"` -- minimal task, round-trips, verifies empty optional fields are zero/nil
10. `TestFieldOrdering/"it outputs fields in spec order"` -- writes full task, checks positional ordering of all 10 field keys using `strings.Index`
11. `TestSkipsEmptyLines/"it skips empty lines when reading"` -- JSONL with blank lines between entries, verifies 2 tasks
12. `TestWriteEmptyTaskList/"it writes empty file for empty task list"` -- empty slice, verifies 0 bytes written

Correction: V5 has 12 subtests across 10 top-level test functions.

**V6 Test Functions (12 total):**

1. `TestJSONLFormat/"it matches spec format with correct field ordering"` -- minimal task, exact string comparison against expected output
2. `TestJSONLFormat/"it matches spec format with all fields populated"` -- full task, exact string comparison against expected output
3. `TestWriteJSONL/"it writes tasks as one JSON object per line"` -- 2 tasks, checks line count, verifies JSON object structure
4. `TestReadJSONL/"it returns empty list for empty file"` -- 0-byte file, asserts 0 tasks
5. `TestReadJSONL/"it returns error for missing file"` -- nonexistent path, asserts error
6. `TestReadJSONL/"it reads tasks from JSONL file"` -- writes raw JSONL, reads back, verifies fields
7. `TestRoundTrip/"it omits optional fields when empty"` -- checks absence of optional keys, presence of required keys
8. `TestRoundTrip/"it round-trips all task fields without loss"` -- full task, field-by-field comparison
9. `TestAtomicWrite/"it writes atomically via temp file and rename"` -- write initial, overwrite, verify, check no .tmp files
10. `TestAtomicWrite/"it cleans up temp file on write error"` -- writes to nonexistent directory, asserts error
11. `TestFieldVariations/"it handles tasks with all fields populated"` -- full task with 2-element BlockedBy, verifies all fields present, round-trips
12. `TestFieldVariations/"it handles tasks with only required fields"` -- minimal task, round-trips, verifies zero/nil optional fields
13. `TestFieldVariations/"it skips empty lines when reading"` -- blank lines between entries, verifies 2 tasks

Correction: V6 has 13 subtests across 6 top-level test functions.

**Edge Cases Covered:**

| Edge Case | V5 | V6 |
|-----------|-----|-----|
| Empty file (0 bytes) | Yes | Yes |
| Missing file | Yes | Yes |
| Empty lines between tasks | Yes | Yes |
| All fields populated | Yes | Yes |
| Only required fields | Yes | Yes |
| Special characters in description | Yes (`<>&"'`) | No |
| Priority 0 (boundary) | Yes | Yes |
| Multiple BlockedBy entries | Yes (2) | Yes (2) |
| Description with newlines | Yes (`\n`) | No |
| Empty task list write | Yes | No |
| Write error cleanup | No | Yes (nonexistent dir) |
| Exact spec format match | No (positional check) | Yes (string equality) |

**Test Gap Analysis:**

- V5 uniquely tests: writing an empty task list (edge case of 0 tasks producing 0 bytes), special characters in description, description with embedded newlines.
- V6 uniquely tests: write error cleanup (nonexistent directory), exact spec format matching via string equality.
- V6's `TestJSONLFormat` with exact string comparison is a stronger assertion for spec compliance than V5's positional index check. If the output ever drifts from the spec, V6 catches it immediately.
- V5's `TestWriteEmptyTaskList` is a genuine gap in V6 -- the spec says "Empty file (0 bytes): valid, returns empty task list" which implies writing 0 tasks should produce an empty file.

**Assertion Style:**

Both use `t.Fatalf`/`t.Errorf` (standard library assertions). Neither uses a third-party assertion library. Both use subtests within top-level test functions. Neither uses table-driven tests with `[]struct` -- all tests are individual subtests. This is technically a skill constraint deviation (the skill says "Write table-driven tests with subtests"), though the nature of these tests (each scenario has unique setup) makes table-driven tests a poor fit.

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| **MUST: Use gofmt and golangci-lint** | PASS -- code is formatted consistently | PASS -- code is formatted consistently |
| **MUST: Handle all errors explicitly** | PASS -- every error checked, wrapped with context | PASS -- every error checked, wrapped with context |
| **MUST: Write table-driven tests with subtests** | PARTIAL -- uses subtests but not table-driven format; individual scenarios with unique setup justify this | PARTIAL -- same; subtests but not table-driven |
| **MUST: Document all exported functions, types, and packages** | PASS -- `WriteTasks`, `ReadTasks` have doc comments; package has doc comment | PASS -- `WriteJSONL`, `ReadJSONL`, `jsonlTask` (unexported but documented via its role) have doc comments; package has doc comment |
| **MUST: Propagate errors with fmt.Errorf("%w", err)** | PASS -- all error returns use `%w` | PASS -- all error returns use `%w` |
| **MUST NOT: Ignore errors** | PASS -- no `_` assignments for errors | PASS -- no `_` assignments for errors |
| **MUST NOT: Use panic for error handling** | PASS -- no panics | PASS -- no panics |
| **MUST NOT: Hardcode configuration** | PASS -- paths are parameters | PASS -- paths are parameters |

### Spec-vs-Convention Conflicts

**Field ordering guarantee:** The spec requires field ordering `id, title, status, priority, then optional fields`. Both versions achieve this via struct field order in a serialization DTO. In Go, `encoding/json` marshals struct fields in declaration order, so this works reliably. However, this relies on an implementation detail of `encoding/json` that is not formally part of Go's specification. Both versions make the same assumption. Neither version uses a custom `MarshalJSON` with an ordered map or similar, so both are equally reliant on this behavior.

**V6 timestamp handling in Task struct:** V6's `task.Task` uses `json:"created"` (no `-` tag) on `Created time.Time`, meaning direct `json.Marshal` of a `Task` would produce RFC 3339 timestamps with nanosecond precision, not the spec-required `2026-01-19T10:00:00Z` format. V5's `task.Task` uses `json:"-"` on timestamp fields and has custom `MarshalJSON`/`UnmarshalJSON`, so `Task` always serializes correctly regardless of context. This is a potential consistency hazard in V6 if `Task` is ever marshaled outside the storage layer.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 4 (2 code, 2 docs) | 4 (2 code, 2 docs) |
| Lines added | 650 | 725 |
| Impl LOC | 86 | 161 |
| Test LOC | 561 | 561 |
| Test functions (subtests) | 12 | 13 |

## Verdict

**V5 is the better implementation.**

The decisive advantages are:

1. **Architectural cleanliness (86 vs 161 impl LOC):** V5's storage layer is a thin, focused wrapper because serialization logic lives on `task.Task` via `MarshalJSON`/`UnmarshalJSON`. V6 duplicates the entire field mapping into a storage-layer DTO (`jsonlTask`, `toJSONL`, `fromJSONL`), nearly doubling implementation size without gaining anything for this spec's requirements.

2. **Consistency guarantee:** V5's `task.Task` always serializes the same way due to custom marshal methods. V6's `task.Task` would produce different JSON output depending on whether it is marshaled by the storage layer (via DTO) or directly (via `encoding/json`). The timestamp fields in V6 lack `json:"-"` tags, creating a latent inconsistency hazard.

3. **Type safety:** V6 downgrades `Status` from `task.Status` to `string` in `jsonlTask`, losing type safety and performing an unchecked cast back in `fromJSONL`. V5 retains the `task.Status` type throughout.

4. **Idiomatic error style:** V5 uses Go-idiomatic gerund-style error prefixes; V6 uses `"failed to"` prefixes.

V6 has two minor advantages: it includes a test for write-error cleanup (nonexistent directory) that V5 lacks, and its `TestJSONLFormat` exact-string-match assertion is stronger than V5's positional-index check for field ordering. However, V5 covers additional edge cases (empty task list write, special characters in description, newlines in description) that V6 misses, and V5's `TestFieldOrdering` is still a reasonable verification of field order. These minor V6 advantages do not offset the significant architectural and DRY advantages of V5.
