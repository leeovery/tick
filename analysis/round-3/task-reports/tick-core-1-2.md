# Task tick-core-1-2: JSONL storage with atomic writes

## Task Summary

Implement JSONL-based persistent storage for tasks that diffs cleanly in git and survives process crashes. The implementation requires:

1. A JSONL writer that serializes each `Task` as a single JSON line (no array wrapper, no trailing commas)
2. Omit optional fields when empty/null (omit the key entirely, not serialize as `null`)
3. A JSONL reader that parses line-by-line into `[]Task`, skipping empty lines
4. Atomic write pattern: write to temp file in same directory, `fsync`, `os.Rename(temp, tasks.jsonl)`
5. Full file rewrite on every mutation (no append mode)
6. Handle empty file (0 bytes) as valid (returns empty task list)
7. Handle missing file as an error

**Acceptance Criteria:**
- Tasks round-trip through write then read without data loss
- Optional fields omitted from JSON when empty/null
- Atomic write uses temp file + fsync + rename
- Empty file returns empty task list
- Each task occupies exactly one line (no pretty-printing)
- JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| Tasks round-trip through write/read without data loss | PASS -- `TestReadJSONL/"it round-trips all task fields without loss"` verifies every field | PASS -- `TestRoundTrip/"it round-trips all task fields without loss"` verifies every field |
| Optional fields omitted from JSON when empty/null | PASS -- checks `description`, `blocked_by`, `parent`, `closed` are absent in output | PASS -- checks all four optional fields are absent in output |
| Atomic write uses temp file + fsync + rename | PASS -- `WriteJSONL` calls `CreateTemp`, `Sync()`, `Close()`, `Rename()` in correct order | PASS -- `WriteTasks` calls `CreateTemp`, `Sync()`, `Close()`, `Rename()` in correct order |
| Empty file returns empty task list | PASS -- returns `[]Task{}` (explicit non-nil empty slice) | PASS -- returns `nil` slice (see analysis below for significance) |
| Each task occupies exactly one line | PASS -- writes `json.Marshal` + `\n` per task | PASS -- uses `json.NewEncoder` which appends `\n` per `Encode` call |
| JSONL output field ordering matches spec | PASS -- struct field order controls `json.Marshal` ordering; test verifies positions | PASS -- struct field order controls ordering; test verifies positions |

## Implementation Comparison

### Approach

**Package placement:**
- V4 places JSONL functions in `internal/task/` as `WriteJSONL` and `ReadJSONL`, co-located with the `Task` struct.
- V5 places JSONL functions in a new `internal/storage/` package as `WriteTasks` and `ReadTasks`, importing the `task` package.

V5's separation is arguably better architecture -- it separates concerns (storage vs domain model) and follows the Go convention of small, focused packages. V4's co-location is simpler but couples serialization with the domain model.

**Write implementation:**

Both versions use identical atomic write patterns: `CreateTemp` in same directory, deferred cleanup on error via a `success` bool, `Sync()`, `Close()`, `Rename()`.

V4 uses manual `json.Marshal` + `Write` + `WriteString("\n")`:
```go
// V4: internal/task/jsonl.go lines 31-42
for _, t := range tasks {
    data, err := json.Marshal(t)
    if err != nil {
        return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
    }
    if _, err := tmp.Write(data); err != nil {
        return fmt.Errorf("failed to write task %s: %w", t.ID, err)
    }
    if _, err := tmp.WriteString("\n"); err != nil {
        return fmt.Errorf("failed to write newline: %w", err)
    }
}
```

V5 uses `json.NewEncoder`:
```go
// V5: internal/storage/jsonl.go lines 35-40
encoder := json.NewEncoder(tmp)
for _, t := range tasks {
    if err := encoder.Encode(t); err != nil {
        return fmt.Errorf("encoding task %s: %w", t.ID, err)
    }
}
```

V5's approach is more idiomatic Go. `json.NewEncoder.Encode()` automatically appends a newline, reducing the code from 11 lines to 5 lines and eliminating a separate error path for the newline write.

**Read implementation:**

Both versions are nearly identical: open file, `bufio.NewScanner`, skip empty lines, `json.Unmarshal` per line, track line numbers for error context.

One subtle difference: V4 calls `strings.TrimSpace(scanner.Text())` while V5 compares `scanner.Text()` directly to `""`. V4's approach is more robust since it handles lines containing only whitespace, though this is unlikely in practice given the writer never produces such lines.

**Empty slice handling:**

V4 explicitly converts a nil slice to an empty slice at the end of `ReadJSONL`:
```go
// V4: internal/task/jsonl.go lines 91-93
if tasks == nil {
    tasks = []Task{}
}
```

V5 does not do this, returning a nil slice for empty files. This is a behavioral difference: while `len(nil) == 0` is true, `nil != []Task{}` for JSON serialization and `reflect.DeepEqual`. The spec says "returns empty task list" which is ambiguous, but V4's explicit empty slice is more defensive.

**Timestamp handling:**

A critical architectural difference lies in how each version's `Task` struct handles timestamp serialization. This comes from the underlying `task.go`, not this task's code, but it directly affects the JSONL output.

V4's `Task` struct uses `time.Time` with default `json:"created"` tags, relying on Go's default `time.Time` JSON marshaling (RFC 3339 with nanoseconds). The `json.Marshal` in V4's `WriteJSONL` delegates entirely to struct tags.

V5's `Task` struct uses `json:"-"` on `Created`, `Updated`, and `Closed` fields and implements custom `MarshalJSON`/`UnmarshalJSON` methods via a `taskJSON` helper struct. This ensures timestamps are formatted as `2006-01-02T15:04:05Z` (no fractional seconds, always UTC). V5's approach gives explicit control over the exact timestamp format.

### Code Quality

**Naming:**

V4 uses `WriteJSONL`/`ReadJSONL` -- names that describe the format. V5 uses `WriteTasks`/`ReadTasks` -- names that describe the domain operation. Since V5 is in a `storage` package, the fully qualified names `storage.WriteTasks`/`storage.ReadTasks` read naturally. V4's names `task.WriteJSONL`/`task.ReadJSONL` are also clear but leak implementation detail (JSONL format) into the function name. Both are acceptable; V5 is slightly more Go-idiomatic by following the "package name provides context" convention.

**Error messages:**

V4 uses `"failed to ..."` prefix style:
```go
return fmt.Errorf("failed to create temp file: %w", err)
return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
```

V5 uses gerund style without prefix:
```go
return fmt.Errorf("creating temp file: %w", err)
return fmt.Errorf("encoding task %s: %w", t.ID, err)
```

V5's style follows the Go convention documented in the standard library and Effective Go: error strings should not be capitalized or end with punctuation, and gerund-based descriptions are preferred for wrapped errors. V4's style is common in practice but less idiomatic.

**Error wrapping:**

Both versions correctly wrap all errors with `%w`, complying with the skill constraint "Propagate errors with fmt.Errorf("%w", err)".

**Package documentation:**

V4 has no package-level doc comment in `jsonl.go` (the package doc is in `task.go`). V5 has an explicit package doc comment:
```go
// Package storage provides JSONL-based persistent storage for tasks with
// atomic write support using the temp file + fsync + rename pattern.
package storage
```

V5 is better here -- creating a new package demands a package doc comment.

**DRY:**

Both implementations are clean with no code duplication. V5 is slightly DRYer due to using `json.NewEncoder` (avoids separate Write + WriteString calls).

**Exported function documentation:**

Both versions document all exported functions with godoc-style comments, complying with the skill constraint.

### Test Quality

**V4 Test Functions (in `internal/task/jsonl_test.go`):**

Under `TestWriteJSONL`:
1. `"it omits optional fields when empty"` -- verifies optional fields absent, required fields present
2. `"it writes atomically via temp file and rename"` -- writes, overwrites, verifies content and no temp files
3. `"it outputs fields in spec order"` -- verifies field position ordering in JSON string
4. `"it writes tasks as one JSON object per line"` -- verifies 2 tasks produce 2 lines, each starting with `{`

Under `TestReadJSONL`:
5. `"it round-trips all task fields without loss"` -- field-by-field comparison of all 10 fields
6. `"it returns empty list for empty file"` -- empty file returns `[]Task{}` (checks non-nil)
7. `"it returns error for missing file"` -- nonexistent file returns error
8. `"it handles tasks with all fields populated"` -- all optional fields present, verifies round-trip + raw JSON content
9. `"it handles tasks with only required fields"` -- minimal task, verifies empty optionals
10. `"it skips empty lines in JSONL file"` -- hand-written JSONL with blank lines
11. `"it reads tasks from JSONL file"` -- basic 2-task read from hand-written JSONL

Total: 11 test cases across 2 top-level test functions.

**V5 Test Functions (in `internal/storage/jsonl_test.go`):**

1. `TestWriteTasks/"it writes tasks as one JSON object per line"` -- 2 tasks, 2 lines, no pretty-printing
2. `TestReadTasks/"it reads tasks from JSONL file"` -- basic 2-task read
3. `TestRoundTrip/"it round-trips all task fields without loss"` -- field-by-field comparison (includes description with newlines)
4. `TestOptionalFieldOmission/"it omits optional fields when empty"` -- verifies optional fields absent, required present
5. `TestEmptyFile/"it returns empty list for empty file"` -- empty file returns 0 tasks
6. `TestMissingFile/"it returns error for missing file"` -- nonexistent file returns error
7. `TestAtomicWrite/"it writes atomically via temp file and rename"` -- write, overwrite, verify content + no temp files
8. `TestAllFieldsPopulated/"it handles tasks with all fields populated"` -- all fields present, round-trip + raw check
9. `TestOnlyRequiredFields/"it handles tasks with only required fields"` -- minimal task, verify empty optionals
10. `TestFieldOrdering/"it outputs fields in spec order"` -- field position verification
11. `TestSkipsEmptyLines/"it skips empty lines when reading"` -- JSONL with blank lines
12. `TestWriteEmptyTaskList/"it writes empty file for empty task list"` -- empty input produces 0-byte file

Total: 12 test cases across 10 top-level test functions.

**Structural comparison:**

V4 groups tests under 2 parent `Test*` functions (`TestWriteJSONL` and `TestReadJSONL`) with subtests. This is valid table-driven-adjacent style but is somewhat monolithic -- all read-related tests are under one function regardless of what aspect they test.

V5 separates each logical concern into its own top-level `Test*` function with a single subtest inside. This makes each test independently runnable with `go test -run TestRoundTrip`, which is better for CI granularity and debugging.

**Edge cases unique to V5:**
- `TestWriteEmptyTaskList` -- verifies writing an empty `[]task.Task{}` produces a 0-byte file. V4 does not test this case.
- Round-trip test includes description with embedded newlines: `"Detailed description\nwith newlines"`. V4's round-trip uses `"Details here"` (no special characters).

**Edge cases unique to V4:**
- `"it returns empty list for empty file"` explicitly checks that the result is non-nil: `if tasks == nil { t.Error("expected empty slice, got nil") }`. V5's equivalent test only checks `len(tasks) != 0`, missing the nil vs empty distinction.

**Edge cases in V5 but with richer assertion:**
- `TestAllFieldsPopulated` in V5 uses a description with special characters: `"A detailed description with special chars: <>&\"'"`. V4's equivalent uses `"Full description with details"`. V5 tests JSON escaping of special characters.

**Assertion quality:**
Both versions use direct field-by-field assertions rather than `reflect.DeepEqual` on the whole struct (except V4 uses `reflect.DeepEqual` for `BlockedBy` slice comparison). Both import only `testing` and standard library -- no third-party assertion libraries. Both use `t.Fatalf` for preconditions and `t.Errorf` for field mismatches, which is correct practice.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code appears formatted | PASS -- code appears formatted |
| Add context.Context to all blocking operations | N/A -- file I/O here does not use context (neither version does; see Spec-vs-Convention) | N/A -- same |
| Handle all errors explicitly (no naked returns) | PASS -- every error is checked | PASS -- every error is checked |
| Write table-driven tests with subtests | PARTIAL -- uses subtests but no table-driven tests (each test is a separate subtest with inline data) | PARTIAL -- uses subtests but no table-driven tests |
| Document all exported functions, types, and packages | PASS -- `WriteJSONL` and `ReadJSONL` have godoc comments | PASS -- `WriteTasks`, `ReadTasks`, and package comment present |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- all errors wrapped with `%w` | PASS -- all errors wrapped with `%w` |
| Must not ignore errors | PASS -- `os.Remove(tmpPath)` in deferred cleanup ignores error, but this is justified (best-effort cleanup) | PASS -- same justified `os.Remove` in deferred cleanup |
| Must not use panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Must not hardcode configuration | PASS -- file path is passed as parameter | PASS -- file path is passed as parameter |

### Spec-vs-Convention Conflicts

**1. context.Context on file I/O operations**

- **Spec says:** Implement JSONL reader/writer with atomic writes (no mention of context).
- **Skill requires:** "Add context.Context to all blocking operations."
- **Both versions chose:** No `context.Context` parameter on `WriteJSONL`/`WriteTasks` or `ReadJSONL`/`ReadTasks`.
- **Assessment:** Reasonable judgment call. These are synchronous file I/O operations on local filesystem. Adding context would require either `os.File` operations that don't support context, or wrapping in goroutines with select -- adding complexity for no practical benefit. The standard library's own `os` package does not accept context for file operations. Both versions correctly omit it.

**2. Table-driven tests**

- **Spec says:** Lists 8 specific test scenarios as individual test descriptions (e.g., `"it writes tasks as one JSON object per line"`).
- **Skill requires:** "Write table-driven tests with subtests."
- **Both versions chose:** Individual subtests matching the spec's test descriptions, not table-driven format.
- **Assessment:** Reasonable. The spec's 8 tests each have fundamentally different setup, assertions, and verification logic. Forcing these into a table-driven format would result in an overly generic test structure with many special-case branches inside the loop. Individual subtests are the right call here.

**3. Empty file return value: nil vs empty slice**

- **Spec says:** "Empty file (0 bytes): valid, returns empty task list."
- **Go convention:** A nil slice is functionally equivalent to an empty slice for most operations (`len`, `range`, `append`).
- **V4 chose:** Explicitly return `[]Task{}` (non-nil empty slice).
- **V5 chose:** Return nil (the default when no tasks are appended).
- **Assessment:** V4 is more defensive and aligns more closely with the spec's "returns empty task list" wording. V5's nil return is idiomatic Go but could cause issues if callers check `tasks == nil` to detect error conditions. Neither is wrong, but V4 is marginally safer.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 4 (2 impl/test + 2 docs) | 4 (2 impl/test + 2 docs) |
| Lines added | 597 | 650 |
| Impl LOC | 99 | 86 |
| Test LOC | 495 | 561 |
| Test functions | 11 subtests (2 top-level) | 12 subtests (10 top-level) |

## Verdict

**V5 is the better implementation**, though both are solid.

Key advantages of V5:

1. **Package architecture:** Placing storage concerns in `internal/storage/` rather than `internal/task/` properly separates the domain model from its persistence mechanism. This is a meaningful structural advantage that becomes more important as the codebase grows.

2. **More idiomatic Go code:** V5 uses `json.NewEncoder` (reducing write code from 11 lines to 5), gerund-style error messages matching Go conventions, and includes a package-level doc comment for the new package.

3. **Better test organization:** V5 uses 10 top-level test functions (one per logical concern) vs V4's 2 monolithic top-level functions. This provides better `go test -run` granularity and clearer test output.

4. **One additional test case:** V5 includes `TestWriteEmptyTaskList` (writing empty input produces 0-byte file), a valid edge case V4 misses.

5. **Richer edge case data:** V5's round-trip test uses a description containing embedded newlines and its all-fields test uses special characters (`<>&"'`), providing stronger confidence in JSON serialization correctness.

6. **Custom timestamp marshaling:** V5's `Task` struct (from prior task) uses custom `MarshalJSON`/`UnmarshalJSON` with explicit timestamp formatting, giving deterministic output format. V4 relies on Go's default `time.Time` JSON encoding which includes nanosecond precision and timezone info that may not match the spec's exact format.

V4's one notable advantage is the explicit nil-to-empty-slice conversion for empty files, which is more defensive. However, this is a minor point that does not outweigh V5's structural and idiomatic advantages.
