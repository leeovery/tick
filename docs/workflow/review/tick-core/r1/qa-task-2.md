TASK: JSONL storage with atomic writes

ACCEPTANCE CRITERIA:
- Tasks round-trip through write -> read without data loss
- Optional fields omitted from JSON when empty/null
- Atomic write uses temp file + fsync + rename
- Empty file returns empty task list
- Each task occupies exactly one line (no pretty-printing)
- JSONL output matches spec format (field ordering: id, title, status, priority, then optional fields)

STATUS: Complete

SPEC CONTEXT: The spec defines JSONL as the source of truth for tasks, committed to git. One JSON object per line, no trailing commas, no array wrapper. Optional fields (description, blocked_by, parent, closed) omitted when empty/null -- not serialized as null. Atomic rewrite pattern: temp file -> fsync -> os.Rename. Full file rewrite on every mutation. Empty file is valid (returns empty task list). Missing file is an error.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/storage/jsonl.go (lines 1-128)
- Notes:
  - MarshalJSONL (line 16): serializes tasks to JSONL bytes, one JSON object per line, newline-terminated
  - WriteJSONL (line 34): delegates to writeAtomic after marshaling
  - writeAtomic (line 49): correct temp file + fsync + rename pattern with cleanup on error via deferred function
  - ParseJSONL (line 89): parses raw bytes line-by-line, skips empty lines, returns empty list for empty input
  - ReadJSONL (line 121): reads file, returns error for missing file, delegates to ParseJSONL
  - WriteJSONLRaw (line 44): additional utility for pre-marshaled data, used by Store layer
  - Field ordering controlled by taskJSON struct in /Users/leeovery/Code/tick/internal/task/task.go (lines 57-68) with correct ordering: id, title, status, priority, description, blocked_by, parent, created, updated, closed
  - Optional fields use omitempty tags: description, blocked_by, parent, closed
  - Custom MarshalJSON/UnmarshalJSON on Task type handles timestamp formatting (ISO 8601 UTC)
  - All acceptance criteria are met

TESTS:
- Status: Adequate
- Coverage:
  - TestJSONLFormat: verifies spec-exact field ordering for minimal task and all-fields task (lines 13-83)
  - TestWriteJSONL: verifies one JSON object per line, no pretty-printing (lines 86-134)
  - TestReadJSONL: empty file returns empty list, missing file returns error, reads tasks correctly (lines 137-205)
  - TestRoundTrip: optional fields omitted when empty, all fields round-trip without loss (lines 208-329)
  - TestAtomicWrite: verifies atomic overwrite and no temp files left behind, error path for non-existent directory (lines 332-423)
  - TestParseJSONL: parses bytes, empty bytes returns empty, skips empty lines, invalid JSON returns error (lines 426-483)
  - TestMarshalJSONL: serializes to JSONL bytes, empty list returns empty bytes, round-trip through ParseJSONL (lines 486-586)
  - TestFieldVariations: all fields populated, only required fields, skips empty lines (lines 589-723)
- Notes:
  - All 8 planned tests from the task file are covered
  - Edge cases (empty file, missing file, empty lines, invalid JSON) are well tested
  - The MarshalJSONL round-trip test at line 536 partially duplicates the WriteJSONL round-trip test at line 254, but is acceptable since it tests the in-memory path (Marshal/Parse) separately from the file path (Write/Read)
  - Tests would fail if the feature broke -- format assertions are string-exact, round-trip checks verify all 10 fields

CODE QUALITY:
- Project conventions: Followed -- internal package structure, table-driven subtests, t.TempDir() for cleanup, proper error wrapping with %w
- SOLID principles: Good -- single responsibility (MarshalJSONL, ParseJSONL, writeAtomic are each focused), functions compose cleanly (WriteJSONL calls MarshalJSONL then writeAtomic)
- Complexity: Low -- each function is short and linear, no complex branching
- Modern idioms: Yes -- uses bytes.Buffer for efficient string building, bufio.Scanner for line parsing, deferred cleanup with success flag pattern
- Readability: Good -- clear function names, doc comments on all exported functions, logical grouping
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- ParseJSONL returns nil (not empty slice) for empty input, and MarshalJSONL returns nil for empty task list. This is idiomatic Go (nil slices serialize as null in JSON), and since the caller (ReadJSONL) passes through directly, it works correctly. However, if a caller ever json.Marshal'd the result of ParseJSONL with zero tasks, they'd get "null" instead of "[]". This is acceptable given the current usage pattern where JSONL is the serialization format, not JSON arrays.
- WriteJSONLRaw is not directly tested in jsonl_test.go, but it is a thin wrapper over writeAtomic (which is tested via WriteJSONL), and is exercised through the Store layer. Acceptable.
