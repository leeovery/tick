TASK: Orphaned Parent Reference Check (doctor-validation-3-1)

ACCEPTANCE CRITERIA:
- [x] `TaskRelationshipData` struct defined with `ID`, `Parent`, `BlockedBy`, `Status`, and `Line` fields
- [x] `ParseTaskRelationships` function reads `tasks.jsonl` and returns a slice of `TaskRelationshipData`
- [x] Parser skips blank lines, unparseable lines, and lines with missing/non-string IDs
- [x] Parser correctly extracts `parent` (string or empty), `blocked_by` (slice or empty), and `status`
- [x] Parser returns error for missing `tasks.jsonl`; returns empty slice for empty file
- [x] Parser is reusable -- not coupled to the orphaned parent check
- [x] `OrphanedParentCheck` implements the `Check` interface
- [x] Passing check returns `CheckResult` with Name `"Orphaned parents"` and Passed `true`
- [x] Each orphaned parent reference produces its own failing `CheckResult` with child ID and missing parent ID in details
- [x] Details follow spec wording: `"tick-{child} references non-existent parent tick-{parent}"`
- [x] Null or absent `parent` field treated as valid root task (not flagged)
- [x] Unparseable lines skipped by parser -- not flagged as orphaned references
- [x] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [x] Suggestion is `"Manual fix required"` for orphaned parent errors
- [x] All failures use `SeverityError`
- [x] Check is read-only -- never modifies `tasks.jsonl`
- [x] Parser tests written and passing for all parser behaviors
- [x] Orphaned parent check tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines orphaned parent references as Error #5: "Task references non-existent parent." The tick-core specification describes: "Orphaned children (parent reference points to non-existent task) -- Task remains valid, treated as root-level task. tick doctor flags: 'tick-child references non-existent parent tick-deleted'." Doctor reports but never modifies. Fix suggestion for all non-cache errors is "Manual fix required". Each error is listed individually. Warnings do not affect exit code; errors produce exit code 1.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/doctor/task_relationships.go:1-89` -- `TaskRelationshipData` struct (lines 5-16), `taskRelationshipsFromLines` pure function (lines 22-75), `ParseTaskRelationships` public entry point (lines 82-89)
  - `/Users/leeovery/Code/tick/internal/doctor/orphaned_parent.go:1-49` -- `OrphanedParentCheck` struct and `Run` method
  - `/Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:1-94` -- `ScanJSONLines` shared line reader, `getJSONLines` context-aware wrapper, `getTaskRelationships` context-aware relationship accessor
  - `/Users/leeovery/Code/tick/internal/doctor/helpers.go:1-22` -- `buildKnownIDs` and `fileNotFoundResult` shared helpers
  - `/Users/leeovery/Code/tick/internal/cli/doctor.go:23` -- check registered with runner
- Notes:
  - The architecture evolved through later refactoring phases (4-1, 5-1, 6-1) to derive `ParseTaskRelationships` from `ScanJSONLines` output and extract `buildKnownIDs`/`fileNotFoundResult` helpers. The final design is cleaner than the original plan specified: the parser is a thin composition of `ScanJSONLines` + `taskRelationshipsFromLines`, and the check uses context-aware `getTaskRelationships` to avoid redundant file reads when multiple checks run in the same `tick doctor` invocation.
  - The `OrphanedParentCheck` struct has no fields (zero-value struct) since `tickDir` is now passed as a `Run` parameter (Phase 4-2 refactoring).
  - All behavior matches spec and plan. No drift detected.

TESTS:
- Status: Adequate
- Coverage:
  - Parser tests (`TestParseTaskRelationships` in `/Users/leeovery/Code/tick/internal/doctor/task_relationships_test.go`): 12 subtests covering empty file, missing file, field extraction (id/parent/blocked_by/status), null parent, absent parent, null blocked_by, absent blocked_by, blank/unparseable/bad-id lines, line numbers with blanks, trailing newline, multiple blocked_by, read-only.
  - Internal converter tests (`TestTaskRelationshipsFromLines` in same file): 6 subtests covering nil Parsed skip, missing id skip, non-string id skip, full field extraction, empty input, BlockedBy initialization.
  - Orphaned parent check tests (`TestOrphanedParentCheck` in `/Users/leeovery/Code/tick/internal/doctor/orphaned_parent_test.go`): 16 subtests covering all parents valid, empty file, no parents (all root), non-existent parent, multiple orphaned parents, child+parent ID in details, spec wording, null parent, absent parent, unparseable lines, missing file, suggestion text, Name consistency (table-driven across 3 scenarios), SeverityError consistency (table-driven across 2 scenarios), no ID normalization, read-only.
  - Every edge case from the plan task is covered.
  - Tests use stdlib `testing` only with `t.Run()` subtests and `t.Helper()` on helpers -- consistent with project conventions.
- Notes:
  - Tests are well-structured and focused. Each test verifies one specific behavior.
  - The table-driven tests for Name and Severity cover multiple scenarios efficiently without redundancy.
  - The `assertReadOnly` helper avoids boilerplate in read-only verification tests.
  - No over-testing detected. No tests overlap significantly or test implementation details.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers, `fmt.Errorf("context: %w", err)` for error wrapping. Handler follows the `Check` interface pattern from task 1-1.
- SOLID principles: Good. Single responsibility (parser extracts data, check validates references, helpers provide shared utilities). Open/closed (new checks can be added without modifying existing ones). Interface segregation (the `Check` interface is minimal with one method). Dependency inversion (check depends on `ParseTaskRelationships`/`getTaskRelationships` abstraction, not direct file I/O).
- Complexity: Low. The `Run` method is ~30 lines with clear linear flow: get tasks, build ID set, iterate and check, return results. The parser function `taskRelationshipsFromLines` has straightforward field extraction with no deep nesting.
- Modern idioms: Yes. Uses `map[string]struct{}` for set semantics, `context.Context` propagation, zero-value struct for stateless check, `encoding/json` for JSON parsing.
- Readability: Good. Clear naming (`TaskRelationshipData`, `buildKnownIDs`, `fileNotFoundResult`). Well-documented exported types and functions. The code flow in `Run` is self-explanatory.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `ParseTaskRelationships` public function is now only used in tests (all production checks use `getTaskRelationships` via context). This is fine -- it provides a clean public API for external callers and test verification of the end-to-end parsing pipeline. Could be considered for removal if no external consumers are anticipated, but keeping it is reasonable for API completeness.
