---
topic: doctor-validation
cycle: 2
total_proposed: 3
---
# Analysis Tasks: Doctor Validation (Cycle 2)

## Task 1: Derive ParseTaskRelationships from ScanJSONLines output
status: approved
severity: medium
sources: architecture

**Problem**: ParseTaskRelationships in task_relationships.go independently opens tasks.jsonl, scans with bufio.Scanner, parses JSON into map[string]interface{}, skips blanks, and tracks line numbers -- all logic that ScanJSONLines in jsonl_reader.go already performs. This is a compose-don't-duplicate violation: two independent parsing pipelines that must stay in sync on blank-line handling, JSON decoding behavior, and line numbering. A change to one must be mirrored in the other or they drift.

**Solution**: Refactor ParseTaskRelationships into two layers: (1) a new unexported function `taskRelationshipsFromLines(lines []JSONLine) []TaskRelationshipData` that transforms pre-parsed JSONLine data into TaskRelationshipData, and (2) the existing exported ParseTaskRelationships becomes a thin wrapper that calls ScanJSONLines then passes the result to taskRelationshipsFromLines. Update getTaskRelationships to use the context-cached JSONLines when available (call getJSONLines then transform) instead of calling ParseTaskRelationships directly, eliminating the need for a separate TaskRelationshipsKey in the context.

**Outcome**: Single file-parsing pipeline (ScanJSONLines) with TaskRelationshipData derived by transformation. No duplicate parsing logic. Context caching simplified to one key (JSONLinesKey).

**Do**:
1. In `internal/doctor/task_relationships.go`, add an unexported function `taskRelationshipsFromLines(lines []JSONLine) []TaskRelationshipData` that iterates over the JSONLine slice, skips entries where Parsed is nil, extracts id/parent/blocked_by/status fields with the same type-assertion logic currently in ParseTaskRelationships, and sets Line from JSONLine.LineNum.
2. Rewrite `ParseTaskRelationships(tickDir string)` to call `ScanJSONLines(tickDir)` then return `taskRelationshipsFromLines(lines)`. Remove all direct file-open, bufio.Scanner, json.Unmarshal, and blank-line-skip code from this function.
3. Update `getTaskRelationships` in `internal/doctor/jsonl_reader.go` to: first call `getJSONLines(ctx, tickDir)` to get cached/fresh lines, then return `taskRelationshipsFromLines(lines), nil`. Remove the fallback to `ParseTaskRelationships`.
4. Remove `TaskRelationshipsKey` and `taskRelationshipsKeyType` from jsonl_reader.go since task relationships are now derived from JSONLines.
5. Update `internal/cli/doctor.go` to remove the TaskRelationshipsKey context injection (the pre-parsing of task relationships and context.WithValue for TaskRelationshipsKey). Only JSONLinesKey context caching is needed.
6. Run all existing tests (`go test ./internal/doctor/... ./internal/cli/...`) to verify no regressions. All existing behavior is preserved since the transformation logic is identical -- just sourced from ScanJSONLines output instead of independent parsing.

**Acceptance Criteria**:
- ParseTaskRelationships delegates to ScanJSONLines internally; no direct file I/O or JSON parsing in that function.
- taskRelationshipsFromLines is an unexported pure function taking []JSONLine and returning []TaskRelationshipData.
- TaskRelationshipsKey is removed from the codebase; only JSONLinesKey remains for context caching.
- All existing tests pass without modification (behavior is identical).
- getTaskRelationships derives data from getJSONLines, not from ParseTaskRelationships.

**Tests**:
- All existing ParseTaskRelationships tests pass (they call the public API which still works identically).
- All existing relationship check tests pass (they use getTaskRelationships which now derives from JSONLines).
- All existing CLI doctor tests pass (context caching still works via JSONLinesKey).
- Verify taskRelationshipsFromLines correctly skips JSONLine entries with nil Parsed (matching current behavior of skipping unparseable lines).

## Task 2: Extract assertReadOnly test helper
status: approved
severity: medium
sources: duplication

**Problem**: Every check test file (10 files) contains a near-identical "does not modify tasks.jsonl (read-only verification)" test: write content, read file bytes before check, run check, read file bytes after, compare. The pattern is 12-15 lines per file, totaling ~130 lines of duplicated scaffolding. Only the check type instantiated and the JSONL content vary.

**Solution**: Extract a shared test helper `assertReadOnly(t *testing.T, tickDir string, content []byte, runCheck func())` in a common test helper file. Each test file calls this with its specific content and a closure that runs the check.

**Outcome**: ~130 lines of duplicated test scaffolding replaced by ~15 lines of helper plus ~10 one-line calls. Read-only verification logic maintained in one place.

**Do**:
1. Create or identify an existing shared test helper file in `internal/doctor/` (e.g., `helpers_test.go` if it exists, or the file where setupTickDir and writeJSONL already live).
2. Add `assertReadOnly(t *testing.T, tickDir string, content []byte, runCheck func())` that: writes content via writeJSONL, reads file bytes before, calls runCheck(), reads file bytes after, compares with t.Helper() for correct line reporting.
3. In each of the 10 test files, replace the inline read-only verification test body with a call to assertReadOnly, passing the check-specific content and a closure like `func() { check := &OrphanedParentCheck{}; check.Run(ctx, tickDir) }`.
4. Run `go test ./internal/doctor/...` to verify all read-only tests still pass.

**Acceptance Criteria**:
- assertReadOnly helper exists in a shared test file in internal/doctor/.
- All 10 read-only verification tests use the helper instead of inline logic.
- All tests pass. No test behavior changes.
- Helper uses t.Helper() for proper error attribution.

**Tests**:
- All 10 existing "does not modify tasks.jsonl" tests pass using the new helper.
- No other tests are affected.

## Task 3: Use DiagnosticReport methods for issue count in FormatReport
status: approved
severity: low
sources: architecture

**Problem**: FormatReport in format.go maintains a local issueCount variable that counts all non-passing results in its own loop. DiagnosticReport already provides ErrorCount() and WarningCount() methods. The format function's count is logically ErrorCount() + WarningCount() but computed independently. If a new severity level is added or the definition of "issue" changes, FormatReport and the report methods could produce inconsistent numbers.

**Solution**: Replace the local issueCount tracking in FormatReport with `report.ErrorCount() + report.WarningCount()` computed after the display loop.

**Outcome**: Issue count derived from canonical methods. Single source of truth for what constitutes an "issue". FormatReport no longer maintains independent counting logic.

**Do**:
1. In `internal/doctor/format.go`, remove the `issueCount := 0` declaration and the `issueCount++` increment inside the loop.
2. After the display loop (before the switch statement), add `issueCount := report.ErrorCount() + report.WarningCount()`.
3. The rest of the switch statement remains unchanged.
4. Run `go test ./internal/doctor/...` to verify all format tests pass.

**Acceptance Criteria**:
- FormatReport does not maintain its own counter during iteration.
- issueCount is derived from report.ErrorCount() + report.WarningCount().
- All existing format tests pass without modification.

**Tests**:
- All existing FormatReport tests pass (behavior is identical since ErrorCount()+WarningCount() equals the current counting logic).
- ExitCode tests unaffected.
