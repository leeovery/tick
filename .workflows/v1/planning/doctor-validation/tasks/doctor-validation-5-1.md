---
id: doctor-validation-5-1
phase: 5
status: pending
created: 2026-02-13
---

# Derive ParseTaskRelationships from ScanJSONLines output

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
