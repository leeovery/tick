TASK: Parent Done With Open Children Warning (doctor-validation-3-6)

ACCEPTANCE CRITERIA:
- ParentDoneWithOpenChildrenCheck implements the Check interface
- Check reuses ParseTaskRelationships from task 3-1
- Passing check returns CheckResult with Name "Parent done with open children" and Passed true
- Each done parent + open child pair produces its own failing CheckResult with SeverityWarning
- Details follow wording: "tick-{parent} is done but has open child tick-{child}"
- Suggestion is "Review whether parent was completed prematurely" for warning results
- "Open" child means status open or in_progress (not done, not cancelled)
- Only parents with status done are flagged (not open, in_progress, or cancelled)
- Parent done with all children done produces no warnings
- Parent done with all children cancelled produces no warnings
- Parent done with mix of done/cancelled/open children produces warnings only for open children
- Parent not done with open children produces no warnings
- Missing tasks.jsonl returns SeverityError failure with init suggestion (not SeverityWarning)
- Parent IDs not found in status map (non-existent parent tasks) are skipped
- Unparseable lines skipped by parser -- not included in analysis
- IDs compared as-is (no normalization)
- Check is read-only -- never modifies tasks.jsonl
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: The specification defines Warning #1: "Parent marked done while children still open -- allowed but suspicious." This is the only warning-severity check in the entire doctor suite. Exit code 0 means "All checks passed (no errors, warnings allowed)." Warnings appear in output but do not cause non-zero exit.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/parent_done_open_children.go:1-78
- Registration: /Users/leeovery/Code/tick/internal/cli/doctor.go:28
- Notes: All acceptance criteria met. Implementation is clean and correct:
  - Implements Check interface via Run(ctx, tickDir) (line 20)
  - Uses getTaskRelationships (refactored from ParseTaskRelationships) for parsing (line 21)
  - Missing file returns fileNotFoundResult with SeverityError (line 23)
  - Builds statusMap and childrenMap from parsed tasks (lines 26-34)
  - Skips parent IDs not in statusMap (orphaned parents, line 46)
  - Only flags parents with status "done" (line 49)
  - Flags children with status "open" or "in_progress" (line 58)
  - Uses SeverityWarning for findings (line 61)
  - Details format uses parentID/childID which already include tick- prefix, matching spec wording (line 63)
  - Suggestion matches spec: "Review whether parent was completed prematurely" (line 65)
  - Returns single passing result when no warnings found (lines 74-77)
  - Sorts parent IDs and children for deterministic output (lines 37-41, 54)
  - Check name "Parent done with open children" used consistently
  - Read-only -- no writes anywhere in the implementation

TESTS:
- Status: Adequate
- Coverage: All 24 test cases from the plan are implemented in /Users/leeovery/Code/tick/internal/doctor/parent_done_open_children_test.go:1-493
  - Passing cases: no done parent with open children, empty file, all children done, all children cancelled, mixed done/cancelled, parent open/in_progress/cancelled with open children (7 tests)
  - Warning cases: one open child, one in_progress child, multiple open children, mixed children with selective flagging, multiple parents (5 tests)
  - Severity verification: SeverityWarning for findings, SeverityError for missing file (2 tests)
  - Details and suggestion wording verification (3 tests)
  - Check name consistency across passing/warning/error states via table-driven test (1 test with 3 subtests)
  - Edge cases: orphaned parent skip, unparseable lines, no ID normalization, read-only verification (4 tests)
  - Integration: DiagnosticRunner with warnings-only produces exit code 0, verifies HasErrors false and WarningCount 1 (1 test)
- Notes: Tests are well-structured with t.Run subtests. The table-driven test for check name is a good pattern. Integration test validates the critical behavioral distinction (warnings do not affect exit code). No redundancy or over-testing observed -- each test verifies a distinct behavior or edge case.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.TempDir for isolation, t.Helper on helpers. Error wrapping with fmt.Errorf not needed here (no errors returned). Struct implements interface correctly.
- SOLID principles: Good. Single responsibility (only checks parent-done-with-open-children). Implements Check interface correctly. Uses shared helpers (fileNotFoundResult, getTaskRelationships) following DIP.
- Complexity: Low. Linear logic: parse tasks, build maps, iterate, check conditions. No nested complexity beyond the parent-children double loop which is inherent to the problem.
- Modern idioms: Yes. Clean Go patterns, appropriate use of maps and slices, deterministic sorting.
- Readability: Good. Clear variable names (statusMap, childrenMap, parentIDs). Well-documented struct and method with godoc comments. Logic flow is straightforward.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
