---
id: tick-core-6-3
phase: 6
status: completed
created: 2026-02-09
---

# Consolidate ReadTasks/ParseTasks duplicate JSONL parsing

**Problem**: `ReadTasks` (reads from file path) and `ParseTasks` (reads from `[]byte`) in `internal/storage/jsonl.go` contain identical scanner-based parsing logic: bufio.Scanner loop, empty-line skip, line-by-line json.Unmarshal, lineNum tracking, and error wrapping. The only difference is the io.Reader source (os.File vs bytes.NewReader). This is 20+ lines of duplicated logic that must be kept in sync.

**Solution**: Have `ReadTasks` open the file, read its contents into `[]byte`, then delegate to `ParseTasks`. This collapses the duplicate parsing loop into a single implementation.

**Outcome**: JSONL parsing logic exists in one place (`ParseTasks`). `ReadTasks` becomes a thin wrapper that handles file I/O and delegates.

**Do**:
1. In `internal/storage/jsonl.go`, modify `ReadTasks` to: open the file, read all bytes (e.g. via `io.ReadAll`), close the file, then call `ParseTasks(data)` and return its result.
2. Remove the duplicated scanner loop from `ReadTasks`.
3. Run all storage tests to verify behavior is identical.

**Acceptance Criteria**:
- `ReadTasks` delegates to `ParseTasks` instead of duplicating the parsing loop
- All existing storage/JSONL tests pass unchanged
- Error messages from ReadTasks remain consistent (file-not-found errors still originate from ReadTasks)

**Tests**:
- All existing storage tests pass -- this is a pure refactor
- Verify ReadTasks still returns appropriate error for non-existent file path
