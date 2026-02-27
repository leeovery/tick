TASK: Derive ParseTaskRelationships from ScanJSONLines output

ACCEPTANCE CRITERIA:
- ParseTaskRelationships delegates to ScanJSONLines internally; no direct file I/O or JSON parsing in that function.
- taskRelationshipsFromLines is an unexported pure function taking []JSONLine and returning []TaskRelationshipData.
- TaskRelationshipsKey is removed from the codebase; only JSONLinesKey remains for context caching.
- All existing tests pass without modification (behavior is identical).
- getTaskRelationships derives data from getJSONLines, not from ParseTaskRelationships.

STATUS: Complete

SPEC CONTEXT: This is a refactoring task from cycle 2 analysis. The doctor validation system requires parsing tasks.jsonl for relationship data (parent, blocked_by, status). Previously, ParseTaskRelationships had its own independent file-open/scan/parse pipeline duplicating ScanJSONLines logic. The task eliminates this duplication by making ParseTaskRelationships a thin composition of ScanJSONLines + a pure transformation function.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/doctor/task_relationships.go:22-75` -- `taskRelationshipsFromLines` unexported pure function
  - `/Users/leeovery/Code/tick/internal/doctor/task_relationships.go:82-89` -- `ParseTaskRelationships` delegates to `ScanJSONLines` then `taskRelationshipsFromLines`
  - `/Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:88-94` -- `getTaskRelationships` derives from `getJSONLines` + `taskRelationshipsFromLines`
  - `/Users/leeovery/Code/tick/internal/cli/doctor.go:32-35` -- Only `JSONLinesKey` used for context caching
- Notes:
  - `task_relationships.go` has no import block -- confirms zero direct file I/O or JSON parsing.
  - `TaskRelationshipsKey` and `taskRelationshipsKeyType` are completely absent from all Go source files (only referenced in docs/planning files).
  - `ParseTaskRelationships` is no longer called from `getTaskRelationships` or `doctor.go`; it remains as a public API for external callers (tests use it).
  - All six relationship checks (orphaned parent, orphaned dependency, self-referential dep, dependency cycle, child-blocked-by-parent, parent-done-with-open-children) use `getTaskRelationships` which routes through the shared JSONLines pipeline.

TESTS:
- Status: Adequate
- Coverage:
  - `TestParseTaskRelationships` (12 subtests) at `/Users/leeovery/Code/tick/internal/doctor/task_relationships_test.go:7-235` -- covers empty file, missing file, field extraction, null/absent parent, null/absent blocked_by, blank line skipping, unparseable JSON skipping, line numbering, trailing newline, multiple blocked_by, read-only verification.
  - `TestTaskRelationshipsFromLines` (6 subtests) at `/Users/leeovery/Code/tick/internal/doctor/task_relationships_test.go:237-355` -- covers nil Parsed skipping, missing id skipping, non-string id skipping, full field extraction, empty input, absent BlockedBy initialization.
  - `TestGetTaskRelationships` (2 subtests) at `/Users/leeovery/Code/tick/internal/doctor/jsonl_reader_test.go:177-220` -- covers derivation from cached JSONLines context and fallback to ScanJSONLines.
- Notes:
  - The `taskRelationshipsFromLines` tests directly verify the pure function with controlled JSONLine input, matching the acceptance criteria requirement to "verify taskRelationshipsFromLines correctly skips JSONLine entries with nil Parsed."
  - Tests are focused and not over-tested: each subtest verifies a distinct behavior.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.TempDir via setupTickDir helper, error wrapping with fmt.Errorf and %w.
- SOLID principles: Good. Single responsibility is well-maintained: ScanJSONLines handles file I/O and parsing, taskRelationshipsFromLines handles pure transformation, ParseTaskRelationships composes the two. Open/closed principle followed -- adding new field extraction only requires changing the transform function.
- Complexity: Low. taskRelationshipsFromLines is a straightforward linear iteration with simple type assertions. ParseTaskRelationships is 4 lines. getTaskRelationships is 4 lines.
- Modern idioms: Yes. Clean composition pattern, context-based caching, nil-safe slice initialization.
- Readability: Good. Function names clearly express intent. Doc comments present on all exported and key unexported functions. The two-layer decomposition (I/O wrapper + pure transform) is immediately understandable.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. The refactoring cleanly achieves its goal of eliminating duplicate parsing pipelines with a minimal, well-tested design.
