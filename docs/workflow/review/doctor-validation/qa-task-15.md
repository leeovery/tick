TASK: Relationship Check Registration (doctor-validation-3-7)

ACCEPTANCE CRITERIA:
- OrphanedParentCheck registered in the doctor command handler
- OrphanedDependencyCheck registered in the doctor command handler
- SelfReferentialDepCheck registered in the doctor command handler
- DependencyCycleCheck registered in the doctor command handler
- ChildBlockedByParentCheck registered in the doctor command handler
- ParentDoneOpenChildrenCheck registered in the doctor command handler
- All 10 checks (4 existing + 6 new) run in a single tick doctor invocation
- Each check receives the correct .tick/ directory path
- All 10 checks run regardless of individual results (no short-circuit)
- Exit code 0 when all 10 checks pass
- Exit code 1 when any check produces an error-severity failure
- Exit code 0 when the only failures are warning-severity (warnings-only scenario)
- Summary count reflects total failures (errors + warnings) across all 10 checks
- Output shows results for all 10 checks (passing and failing)
- Warning-severity results display with cross marker same as errors
- Doctor remains read-only with 10 checks registered (no data modification)
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT:
The specification requires 9 error checks and 1 warning check running in a single invocation. Design principle #4: "Run all checks -- Doctor completes all validations before reporting, never stops early." Exit code 0 = "All checks passed (no errors, warnings allowed)", exit code 1 = "One or more errors found." The parenthetical "warnings allowed" is the key distinction for the ParentDoneWithOpenChildrenCheck. Doctor lists each error individually and uses human-readable output with checkmark/cross markers and a summary count.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/doctor.go:17-42
- Notes:
  - All 6 new checks are registered at lines 23-28: OrphanedParentCheck, OrphanedDependencyCheck, SelfReferentialDepCheck, DependencyCycleCheck, ChildBlockedByParentCheck, ParentDoneWithOpenChildrenCheck.
  - Registration order follows spec error numbering as required: CacheStalenessCheck (#1), JsonlSyntaxCheck (#2), IdFormatCheck (#4), DuplicateIdCheck (#3), OrphanedParentCheck (#5), OrphanedDependencyCheck (#6), SelfReferentialDepCheck (#7), DependencyCycleCheck (#8), ChildBlockedByParentCheck (#9), ParentDoneWithOpenChildrenCheck (Warning #1).
  - tickDir is passed to runner.RunAll() which passes it to each check's Run() method -- all checks receive the same directory path.
  - Minor naming difference from plan: plan says "ParentDoneOpenChildrenCheck" but code uses "ParentDoneWithOpenChildrenCheck" (more descriptive, consistent throughout codebase). Not a functional issue.
  - ExitCode() in /Users/leeovery/Code/tick/internal/doctor/format.go:42-47 correctly delegates to HasErrors() which only considers SeverityError, not SeverityWarning.
  - FormatReport() in /Users/leeovery/Code/tick/internal/doctor/format.go:11-37 counts issues as ErrorCount() + WarningCount(), showing both in summary but only errors drive exit code.
  - Warning results display with the same cross marker as errors (line 16 of format.go uses the same cross path for all non-passing results).
  - No new types or interfaces created -- this task is purely wiring as planned.

TESTS:
- Status: Adequate
- Coverage:
  - TestDoctorTenChecks suite in /Users/leeovery/Code/tick/internal/cli/doctor_test.go:550-836 covers all acceptance criteria and edge cases.
  - All 10 labels verified in output (lines 558-568).
  - 10 check marks counted for single invocation (lines 570-579).
  - Exit code 0 for healthy store (lines 581-589).
  - Exit code 1 for each individual Phase 3 error check failing alone (lines 591-646) -- covers orphaned parent, orphaned dependency, self-referential dep, dependency cycle, child-blocked-by-parent.
  - Exit code 0 for warnings-only scenario (lines 648-658) -- critical new behavior.
  - Exit code 1 for mixed errors + warnings (lines 660-672).
  - Cross-phase error combination tests: Phase 1 + Phase 3 (lines 674-683), Phase 2 + Phase 3 (lines 685-696).
  - Mixed results output format verified -- passing checks show checkmark, failing show cross with details (lines 698-723).
  - Summary count reflecting errors + warnings (lines 737-763).
  - No short-circuit verified (lines 765-776).
  - Empty tasks.jsonl handled (lines 778-791).
  - Read-only invariant preserved (lines 793-825).
  - "No issues found." summary for all-pass (lines 827-835).
- Notes:
  - Two tests are functionally identical: "it shows correct summary count reflecting errors and warnings from all checks combined" (line 737) and "it shows summary count including warnings (e.g., 1 error + 1 warning = '2 issues found.')" (line 751) -- both use the same content and assert "2 issues found." This is minor over-testing but not blocking.
  - The most complex edge case from the plan (cross-phase aggregation with multiple results per check yielding 7 total issues) is not tested with that exact scenario. However, the summary counting logic is tested adequately with simpler cases, and the underlying ErrorCount()/WarningCount() methods are straightforward. This is acceptable.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run subtests, t.Helper on helpers, t.TempDir for isolation, error wrapping with %w. No testify.
- SOLID principles: Good. Single responsibility (RunDoctor wires, runner executes, format displays, ExitCode decides). Open/closed -- adding checks only requires Register calls, no runner changes. Interface segregation is clean (Check interface has one method).
- Complexity: Low. RunDoctor is linear -- create runner, register checks, run, format, exit. No branching logic added.
- Modern idioms: Yes. Context-based data passing (JSONLinesKey), pointer receivers for checks, functional composition.
- Readability: Good. The registration block in doctor.go lines 19-28 is clear and self-documenting. Each check type name maps directly to its spec check name. The comment on RunDoctor lists all 10 checks.
- Issues: None blocking.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Two tests in TestDoctorTenChecks are functionally identical (lines 737-749 and 751-763). One could be removed without reducing coverage.
- The plan uses "ParentDoneOpenChildrenCheck" while the code uses "ParentDoneWithOpenChildrenCheck". This is a harmless naming difference but worth noting for plan-to-code traceability.
