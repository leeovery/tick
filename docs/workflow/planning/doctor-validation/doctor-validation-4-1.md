---
id: doctor-validation-4-1
phase: 4
status: pending
created: 2026-02-13
---

# Extract shared JSONL line iterator and parse tasks.jsonl once per doctor run

**Problem**: Three line-level checks (JsonlSyntaxCheck, DuplicateIdCheck, IdFormatCheck) each independently implement the same ~20-line JSONL file-scanning loop: open tasks.jsonl, handle file-not-found, create bufio.Scanner, maintain lineNum counter, skip blank lines via strings.TrimSpace, and json.Unmarshal each line. Additionally, six relationship checks each independently call ParseTaskRelationships which re-opens, re-reads, and re-parses the entire tasks.jsonl file. The file is opened and fully parsed ~10 times per doctor run, wasting I/O and risking inconsistent snapshots if the file changes mid-run.

**Solution**: (1) Extract a shared JSONL line iterator (e.g., `ScanJSONLines(tickDir string) ([]JSONLine, error)` in a new `jsonl_reader.go`) that opens the file, scans lines, skips blanks, parses JSON, and returns (lineNum, raw line, parsed map) tuples. The three line-level checks use this instead of duplicating the scanning loop. (2) In `RunDoctor` (internal/cli/doctor.go), call `ParseTaskRelationships` once and store the result in the context (add a new context key `TaskRelationshipsKey` in doctor.go) so the six relationship checks pull pre-parsed data from context rather than re-parsing. Alternatively, add a `SetTaskRelationships` method or pass the data via a struct field on each relationship check.

**Outcome**: tasks.jsonl is opened and scanned at most twice per doctor run (once for line-level checks via the shared iterator, once for relationship data via ParseTaskRelationships). Line-level checks contain only their validation logic. Relationship checks no longer independently parse the file.

**Do**:
1. Create `internal/doctor/jsonl_reader.go` with a `JSONLine` struct (`LineNum int`, `Raw string`, `Parsed map[string]interface{}`) and a `ScanJSONLines(tickDir string) ([]JSONLine, error)` function that implements the shared open/scan/skip-blank/parse loop
2. Refactor `JsonlSyntaxCheck.Run` to call `ScanJSONLines` -- note: syntax check needs the raw line for `json.Valid` on unparsed lines, so `ScanJSONLines` should return raw lines even when JSON parsing fails (or provide a lower-level variant that yields raw lines and parse errors)
3. Refactor `DuplicateIdCheck.Run` to call `ScanJSONLines` and iterate the returned slice
4. Refactor `IdFormatCheck.Run` to call `ScanJSONLines` and iterate the returned slice
5. Add a `TaskRelationshipsKey` context key (similar pattern to `TickDirKey`) in `internal/doctor/doctor.go`
6. In `RunDoctor` (internal/cli/doctor.go), call `ParseTaskRelationships(tickDir)` once before `runner.RunAll`, store the result in the context via `context.WithValue`
7. Refactor all six relationship checks (orphaned_parent, orphaned_dependency, self_referential_dep, dependency_cycle, child_blocked_by_parent, parent_done_open_children) to extract task relationships from context instead of calling `ParseTaskRelationships` directly. Include a fallback: if the context key is missing, call `ParseTaskRelationships` directly (defensive coding)
8. Run all existing tests to ensure no regressions

**Acceptance Criteria**:
- tasks.jsonl is opened at most twice per full doctor run (once for line-level scanning, once for relationship parsing)
- No line-level check contains file-open, bufio.Scanner, or line-counting boilerplate
- No relationship check calls ParseTaskRelationships directly in its Run method (it reads from context)
- All existing doctor tests pass without modification (or with minimal test setup changes)
- JsonlSyntaxCheck still correctly reports line numbers for malformed JSON

**Tests**:
- Existing test suite passes (jsonl_syntax_test, duplicate_id_test, id_format_test, all relationship check tests)
- Verify ScanJSONLines correctly skips blank lines and maintains accurate line numbers
- Verify relationship checks work both with pre-parsed context data and with fallback (missing context key)
