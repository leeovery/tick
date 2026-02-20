TASK: Beads Provider - Read & Map (migration-1-2)

ACCEPTANCE CRITERIA:
- BeadsProvider implements Provider interface (compile-time verified)
- Name() returns "beads"
- Tasks() reads .beads/issues.jsonl from configured base directory
- Beads status values mapped to tick equivalents (pending->open, in_progress->in_progress, closed->done)
- Priority values 0-3 passed through directly
- ISO 8601 timestamps parsed into time.Time fields
- Unparseable timestamps result in zero-value time.Time (not an error)
- Fields with no tick equivalent (id, issue_type, close_reason, created_by, dependencies) discarded
- Missing .beads directory returns descriptive error
- Missing issues.jsonl returns descriptive error
- Empty file returns empty slice and nil error
- Malformed JSON lines are skipped; valid lines still returned
- Lines with empty/whitespace-only title are skipped
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification defines migration as a one-time import using a plugin/provider pattern. The beads provider is file-based (local filesystem, no auth). The data mapping approach: "Map all available fields from source to tick equivalents. Missing data uses sensible defaults or is left empty. Extra source fields with no tick equivalent are discarded." Title is the only required field. Error strategy is "continue on error, report failures at end" which applies at line level within the provider. The spec's output format expects per-task success/failure reporting.

IMPLEMENTATION:
- Status: Implemented (with intentional evolution from Phase 3 and Phase 4 tasks)
- Location: /Users/leeovery/Code/tick/internal/migrate/beads/beads.go (139 lines)
- Notes:
  - BeadsProvider struct at line 44, compile-time interface check at line 49: `var _ migrate.Provider = (*BeadsProvider)(nil)`
  - NewBeadsProvider constructor at line 52 takes baseDir string
  - Name() at line 57 returns "beads"
  - Tasks() at lines 66-112: checks .beads dir, checks issues.jsonl, reads line by line with bufio.Scanner, skips blank lines, unmarshals JSON into beadsIssue, maps via mapToMigratedTask
  - beadsIssue struct at lines 28-41 matches planned fields; Priority is `*int` (updated by migration-4-1 to distinguish absent from zero)
  - statusMap at lines 19-23: pending->StatusOpen, in_progress->StatusInProgress, closed->StatusDone
  - mapToMigratedTask at lines 116-138: maps status via statusMap (unknown/empty -> zero value ""), parses timestamps with time.RFC3339 (failures produce zero time.Time), copies Title/Description/Status/Priority/Created/Updated/Closed only (discards id, issue_type, close_reason, created_by, dependencies)
  - Malformed JSON lines produce sentinel MigratedTask with Title "(malformed entry)" and Status "(invalid)" (lines 97-101) -- adapted by migration-3-2 to surface failures to the engine rather than silently dropping
  - Empty/whitespace titles are preserved and returned to the engine for validation (line 116 comment) -- adapted by migration-3-2
  - No Validate() call in provider -- validation delegated to engine per migration-3-2
  - Priority uses *int pointer with nil-check at lines 132-135 -- updated by migration-4-1

TESTS:
- Status: Adequate
- Coverage: All 19 planned tests from migration-1-2 are present (some adapted for Phase 3/4 behavioral changes). 6 additional tests added by Phase 3 (migration-3-2) and Phase 4 (migration-4-1) for new behaviors. Total: 25 test cases across 2 test functions (TestBeadsProvider, TestMapToMigratedTask).
- Test file: /Users/leeovery/Code/tick/internal/migrate/beads/beads_test.go (567 lines)
- Planned test coverage:
  - Name returns beads: line 28
  - Valid JSONL read: line 39
  - Status mappings (pending/in_progress/closed/unknown): lines 60, 77, 94, 111
  - Priority mapping 0-3 (table-driven): line 128
  - Timestamp parsing: line 199
  - Missing .beads dir error: line 222
  - Missing issues.jsonl error: line 236
  - Empty file: line 254
  - Blank lines only: line 267
  - Malformed JSON sentinel entries: line 280 (adapted from "skips malformed")
  - Empty title returned for engine: line 305 (adapted from "skips empty title")
  - Whitespace-only title returned for engine: line 326 (adapted from "skips whitespace title")
  - Discarded fields: line 408
  - Description mapping: line 437
  - closed_at timestamp: line 454
  - Timestamp parse failure (zero value): line 472
  - mapToMigratedTask full issue: line 499
- Additional Phase 3/4 tests:
  - Interface implementation compile check: line 35
  - Nil priority for omitted field: line 162
  - Non-nil *int(0) for explicit priority 0: line 179
  - Invalid priority passed through: line 347
  - Mixed valid/invalid/malformed comprehensive: line 375
  - mapToMigratedTask with empty title: line 549
- Notes: Tests are well-structured using t.Run subtests, t.Helper on the setupBeadsDir helper, and t.TempDir for isolation. The priority test at line 128 uses table-driven subtests. No redundant testing detected -- each test verifies a distinct behavior. The `intPtr` helper at line 496 is unused by any test that calls it via that name (it is used in TestMapToMigratedTask at line 505) -- this is fine.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Run subtests, t.TempDir for isolation, t.Helper on helpers, fmt.Errorf with %w for error wrapping, and proper package structure under internal/migrate/beads/.
- SOLID principles: Good. Single responsibility (BeadsProvider does one thing: read and map beads JSONL). Open/closed (new providers can be added without modifying BeadsProvider). Dependency inversion (depends on migrate.Provider interface and task.Status type, not concrete implementations).
- Complexity: Low. The Tasks() method has straightforward linear flow: check dir, check file, open, scan lines, unmarshal, map. No deeply nested conditionals. mapToMigratedTask is a simple field-by-field mapping.
- Modern idioms: Yes. Uses *int for optional priority (pointer semantics for distinguishing zero from absent). Compile-time interface satisfaction check. bufio.Scanner for line-by-line reading. Proper defer for file close.
- Readability: Good. Code is self-documenting with clear function names, doc comments on all exported types and functions, and inline comments explaining non-obvious decisions (sentinel entries, engine delegation). The statusMap is a clean declarative lookup table.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `intPtr` helper function at line 496 of the test file is a simple utility. It could be moved to a shared test helper if other test files need it, but for a single-file usage it is fine where it is.
- The sentinel entry approach for malformed JSON (Title "(malformed entry)", Status "(invalid)") is pragmatic but couples the provider to the engine's validation behavior. If the engine's validation logic changed, these sentinel entries might pass unexpectedly. This is a minor design coupling, not a defect -- the engine tests should (and do) cover this path.
- The plan originally specified `mapToMigratedTask` returning `(MigratedTask, error)`. The implementation returns just `MigratedTask` because migration-3-2 moved all validation to the engine. This is a deliberate and documented evolution, not drift.
