TASK: Orphaned Dependency Reference Check (doctor-validation-3-2)

ACCEPTANCE CRITERIA:
- OrphanedDependencyCheck implements the Check interface
- Check reuses ParseTaskRelationships from task 3-1 (no duplicate file parsing)
- Passing check returns CheckResult with Name "Orphaned dependencies" and Passed true
- Each orphaned dependency reference produces its own failing CheckResult with task ID and missing dep ID in details
- Details follow pattern: "tick-{task} depends on non-existent task tick-{dep}"
- Mixed valid/invalid refs in same blocked_by array: only invalid refs produce failures
- Null, absent, or empty blocked_by treated as valid (no references to check)
- Unparseable lines skipped by parser -- not flagged as orphaned dependencies
- Missing tasks.jsonl returns error-severity failure with init suggestion
- Suggestion is "Manual fix required" for orphaned dependency errors
- All failures use SeverityError
- Check is read-only -- never modifies tasks.jsonl
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: Specification Error #6: "Task depends on non-existent task." Doctor must report each error individually. blocked_by is optional (default []), must reference existing task IDs. Doctor is diagnostic-only, never modifies data. "Manual fix required" suggestion for all non-cache errors.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/orphaned_dependency.go:1-48
- Notes: Clean, minimal implementation. The check struct is stateless (empty struct). Run method delegates to getTaskRelationships (context-aware wrapper around the shared parser from task 3-1), builds a known-ID set via the shared buildKnownIDs helper (/Users/leeovery/Code/tick/internal/doctor/helpers.go:4-9), iterates all tasks and their BlockedBy slices, and produces one failing CheckResult per orphaned reference. Missing file case delegates to shared fileNotFoundResult helper (/Users/leeovery/Code/tick/internal/doctor/helpers.go:14-22). Passing case returns single result with Passed true. All acceptance criteria met. No drift from plan.

TESTS:
- Status: Adequate
- Coverage: 19 subtests in /Users/leeovery/Code/tick/internal/doctor/orphaned_dependency_test.go covering all 19 test scenarios from the plan:
  - Happy paths: all refs valid, empty file, no blocked_by entries, empty/null/absent blocked_by
  - Failure paths: single orphaned dep, multiple orphaned deps on same task, multiple across tasks
  - Details verification: task ID and dep ID in details, exact pattern match
  - Mixed valid/invalid refs: only invalid flagged
  - Unparseable lines: silently skipped
  - Missing file: correct error result with init suggestion
  - Suggestion text: "Manual fix required" verified
  - Check name: "Orphaned dependencies" verified across passing, failing, and missing-file cases (table-driven)
  - Severity: SeverityError verified for orphaned dep and missing file cases (table-driven)
  - No normalization: case-sensitive comparison verified (tick-AAA111 vs tick-aaa111)
  - Read-only: assertReadOnly helper verifies file contents unchanged after check execution
- Notes: Tests are well-structured, use stdlib testing only, use t.Run subtests, use shared test helpers (setupTickDir, writeJSONL, ctxWithTickDir, assertReadOnly). The "Name" and "SeverityError" tests use table-driven subtests, which is idiomatic Go. No over-testing detected -- each test verifies a distinct behavior or edge case.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf, functional struct pattern matching other checks.
- SOLID principles: Good. Single responsibility (one check, one concern: orphaned deps). Implements Check interface cleanly. Reuses shared helpers (buildKnownIDs, fileNotFoundResult, getTaskRelationships) -- good DI via context for pre-scanned data.
- Complexity: Low. Linear scan with map lookup. No nested conditionals beyond the two-level for loop (tasks x blocked_by). Cyclomatic complexity is minimal.
- Modern idioms: Yes. Uses context.Context, struct{} for set values, empty struct for stateless check, nil slice append pattern.
- Readability: Good. 48 lines total, clear variable names, doc comments on struct and method. Logic flow is immediately understandable.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The implementation is structurally identical to OrphanedParentCheck (orphaned_parent.go), differing only in iterating BlockedBy vs checking Parent. This duplication was noted in the plan's Phase 6 analysis (task 6-1: extract buildKnownIDs helper, which has already been done). The remaining structural similarity is inherent to the two checks having parallel logic and is acceptable.
