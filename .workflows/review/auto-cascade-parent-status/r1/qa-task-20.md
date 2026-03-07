TASK: Move Rule 9 out of Transition into ApplyWithCascades

ACCEPTANCE CRITERIA:
- Rule 9 (block reopen under cancelled parent) is handled in ApplyWithCascades, not in Transition
- Cancelled parent checking works with the cascade queue rather than requiring tasks array in Transition

STATUS: Complete

SPEC CONTEXT: Rule 9 states "Cannot reopen a child under a cancelled parent. Error: cannot reopen task under cancelled parent, reopen parent first." The spec places this as a validation rule that checks only the direct parent, not ancestors. The architectural motivation for moving it out of Transition is that Transition is a pure status-transition method operating on a single task pointer -- it has no access to the tasks slice needed for parent lookup. ApplyWithCascades already receives the tasks slice, making it the natural home.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/task/apply_cascades.go:19-30 (Rule 9 check in ApplyWithCascades)
- Location: internal/task/state_machine.go:35-66 (Transition method -- no parent checking)
- Notes: Clean separation. Rule 9 runs before the primary transition call, so on error no task is mutated. The check iterates the tasks slice to find the parent by normalized ID, checks if cancelled, and returns error. Only triggers for action == "reopen". Correct behavior: checks direct parent only, not ancestors.

TESTS:
- Status: Adequate
- Coverage:
  - Blocks reopen under cancelled direct parent (apply_cascades_test.go:388)
  - Allows reopen with no parent (apply_cascades_test.go:414)
  - Allows reopen under open parent (apply_cascades_test.go:429)
  - Allows reopen under done parent (apply_cascades_test.go:445)
  - Allows reopen under in_progress parent (apply_cascades_test.go:462)
  - Allows reopen when grandparent cancelled but direct parent is not (apply_cascades_test.go:478)
  - Handles missing parent gracefully (apply_cascades_test.go:497)
  - Skips Rule 9 for non-reopen actions (apply_cascades_test.go:512)
  - Transition directly allows reopen under cancelled parent (state_machine_test.go:530) -- confirms Transition itself is parent-agnostic
- Notes: Excellent coverage. The negative test (Transition allows it directly) is a smart verification that the rule truly moved. All parent status variants covered. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, error wrapping, NormalizeID for case-insensitive comparison
- SOLID principles: Good -- Transition stays single-responsibility (pure status transitions), ApplyWithCascades handles orchestration-level validation
- Complexity: Low -- simple linear scan with early break
- Modern idioms: Yes -- range over index for slice mutation safety
- Readability: Good -- clear comment "Rule 9: block reopen if direct parent is cancelled" documents intent
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
