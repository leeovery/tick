TASK: Add reopen-under-cancelled-parent guard to Transition (acps-1-5)

ACCEPTANCE CRITERIA:
- Transition() blocks reopen under cancelled parent with correct error message (Rule 9)
- Cancelled grandparent with non-cancelled direct parent does not block (edge case)
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rule 9 states "Cannot reopen a child under a cancelled parent. Error: cannot reopen task under cancelled parent, reopen parent first." The spec checks only the direct parent -- cancelled grandparent with non-cancelled direct parent should not block. This is part of the "cancelled is a hard stop" principle.

IMPLEMENTATION:
- Status: Implemented (with architectural refinement)
- Location: /Users/leeovery/Code/tick/internal/task/apply_cascades.go:19-30
- Notes: Rule 9 was originally planned for `Transition()` but was moved to `ApplyWithCascades()` during analysis cycle 1 (task acps-4-2). This is a sound architectural decision: `Transition()` is a pure single-task state transition that should not need the full task list. `ApplyWithCascades()` already has the task list context needed to look up the parent. The guard checks only the direct parent (not ancestors), matching the spec exactly. A corresponding test in `state_machine_test.go:530-545` confirms `Transition()` itself intentionally does NOT check parent status.

TESTS:
- Status: Adequate
- Coverage:
  - Blocks reopen under cancelled direct parent (apply_cascades_test.go:388)
  - Verifies exact error message matches spec (apply_cascades_test.go:400)
  - Verifies task not mutated on error (apply_cascades_test.go:405-411)
  - Allows reopen with no parent (apply_cascades_test.go:414)
  - Allows reopen under open parent (apply_cascades_test.go:429)
  - Allows reopen under done parent (apply_cascades_test.go:445)
  - Allows reopen under in_progress parent (apply_cascades_test.go:462)
  - Allows reopen when grandparent is cancelled but direct parent is not (apply_cascades_test.go:478) -- the specified edge case
  - Allows reopen when parent ID references non-existent task (apply_cascades_test.go:497)
  - CLI integration test blocks reopening under cancelled parent (transition_test.go:255)
  - Explicit test that Transition() alone does NOT check parent status (state_machine_test.go:530)
- Notes: Good balance of positive/negative cases. Edge case from plan is covered. No over-testing.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, error wrapping, NormalizeID for case-insensitive comparison
- SOLID principles: Good -- Rule 9 placed in the method that has the necessary context (task list), keeping Transition() focused on single-task state changes
- Complexity: Low -- simple linear scan with early break
- Modern idioms: Yes -- standard Go patterns
- Readability: Good -- clear comment "Rule 9: block reopen if direct parent is cancelled"
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The plan task name says "Add reopen-under-cancelled-parent guard to Transition" but the guard lives in ApplyWithCascades. This is intentional per analysis cycle 1 findings (acps-4-2) and is the correct placement, but creates a minor naming mismatch with the original plan task description.
