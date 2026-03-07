TASK: Child Blocked-By Parent Check (doctor-validation-3-5)

ACCEPTANCE CRITERIA:
- ChildBlockedByParentCheck implements the Check interface
- Check reuses ParseTaskRelationships from task 3-1 (refactored to getTaskRelationships)
- Passing check returns CheckResult with Name "Child blocked by parent" and Passed true
- Each child blocked by its parent produces its own failing CheckResult with child ID and parent ID in details
- Details follow wording: "tick-{child} is blocked by its parent tick-{parent}"
- Only direct parent-child relationships flagged -- grandparent/ancestor not checked
- Multiple children blocked by same parent each produce separate failing results
- Child blocked by parent detected even among other valid blocked_by entries
- Duplicate parent entries in blocked_by produce one error per child (not per entry)
- Tasks with no parent (root tasks) are not flagged
- Tasks with parent but empty blocked_by are not flagged
- Unparseable lines skipped by parser -- not examined by this check
- Missing tasks.jsonl returns error-severity failure with init suggestion
- Suggestion mentions deadlock with leaf-only ready rule
- All failures use SeverityError
- Check is read-only -- never modifies tasks.jsonl
- Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: Spec Error #9: "Child blocked_by parent -- Deadlock condition -- child can never become ready." The leaf-only ready rule means a parent with open children never appears in ready, so parent cannot complete while child is open, and child cannot become ready while blocked by parent. Creates permanent deadlock. Fix suggestion: "Manual fix required." Doctor is diagnostic-only and never modifies data.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/doctor/child_blocked_by_parent.go:1-59
- Registration: /Users/leeovery/Code/tick/internal/cli/doctor.go:27
- Notes: Clean, correct implementation. All acceptance criteria met:
  - Implements Check interface (Run(ctx, tickDir) []CheckResult) -- matches interface at doctor.go:40-42
  - Reuses getTaskRelationships (the refactored shared parser from task 3-1, updated in Phases 4-5)
  - Skips root tasks (Parent == "") and tasks with no deps (len(BlockedBy) == 0) at line 29
  - Inner loop searches BlockedBy for Parent match, breaks on first hit (line 37) ensuring one error per child even with duplicates
  - Details format: "%s is blocked by its parent %s" using task.ID and task.Parent -- matches spec wording
  - Suggestion includes "deadlock with leaf-only ready rule" text with em-dash (\u2014)
  - Missing file handled via fileNotFoundResult helper (helpers.go:14) -- consistent with other Phase 3 checks
  - No ancestor/grandparent traversal -- purely local per-task comparison
  - No write operations -- strictly read-only
  - Registered in RunDoctor at cli/doctor.go:27

TESTS:
- Status: Adequate
- Coverage: All 19 planned test cases implemented, covering:
  - 5 passing scenarios: no overlap, empty file, parent+blocked_by no overlap, parent+empty blocked_by, root task with blocked_by
  - 4 failure detection: direct parent blocked, parent among other deps, multiple children same parent, duplicate parent entries
  - 1 grandparent negative test: verifies only direct parent flagged
  - 3 output verification: details wording, suggestion text, CheckResult Name consistency
  - 2 severity/format: SeverityError for all failures, Name field for all result types (table-driven)
  - 1 parser integration: unparseable lines skipped
  - 1 missing file: returns error with correct details and suggestion
  - 1 ID normalization: compares as-is without normalization
  - 1 read-only: assertReadOnly helper verifies file not modified
- Test location: /Users/leeovery/Code/tick/internal/doctor/child_blocked_by_parent_test.go:1-394
- Notes: Tests are well-structured with t.Run subtests. Table-driven tests used for Name and SeverityError checks across multiple scenarios. Each test has a distinct purpose -- no redundant tests. Tests verify behavior rather than implementation details.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only (no testify), t.Run subtests, t.TempDir via setupTickDir helper, t.Helper on helpers, error wrapping with fmt.Errorf pattern, functional Check interface implementation. Registered in RunDoctor alongside other checks.
- SOLID principles: Good. Single responsibility (one check, one concern). Implements Check interface (open/closed). Struct has no fields -- stateless, simple dependency injection via interface.
- Complexity: Low. Linear scan of tasks, simple string comparison within each task's data. No graph traversal needed. Early continue for tasks without parent or blocked_by.
- Modern idioms: Yes. Clean Go idioms -- range loops, append for dynamic slice, early returns, break on first match.
- Readability: Good. Thorough godoc on both the struct and Run method. The code is straightforward: parse tasks, skip irrelevant, check parent in blocked_by, collect failures or return pass. Intent is immediately clear.
- Issues: None found.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. Implementation is clean, correct, and well-tested. All acceptance criteria and edge cases from the plan are fully covered.
