TASK: Implement Cascades for upward start cascade (Rule 2)

ACCEPTANCE CRITERIA:
- Upward start cascade sets open ancestors to in_progress recursively (Rule 2)
- All existing tests pass with no regressions
- Edge cases: ancestor already in_progress skipped, deeply nested chain 5+ levels

STATUS: Complete

SPEC CONTEXT: Rule 2 states: "When a child transitions to in_progress, walk the ancestor chain and set any open ancestors to in_progress. Recursive -- applies to grandparents and beyond." The Cascades function is specified as pure (no mutation), returning []CascadeChange for the caller to apply.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:46-77 (cascadeUpwardStart method)
- Notes: Implementation correctly walks the parent chain using a task map built from the slice. Open ancestors get a CascadeChange with action "start" and StatusOpen->StatusInProgress. In_progress ancestors are skipped but the chain continues. Terminal ancestors (done/cancelled) stop the chain. The dispatching logic at cascades.go:31-32 routes "start" action to this method. Helper functions buildTaskMap (line 227) and NormalizeID are used for ID lookup. Implementation is pure -- no mutations, returns a list of changes. Matches spec precisely.

TESTS:
- Status: Adequate
- Coverage:
  - "it cascades start to open parent" -- basic single parent case (line 24)
  - "it cascades start through multiple open ancestors" -- recursive multi-level (line 48)
  - "it skips ancestor already in_progress" -- edge case from plan (line 78)
  - "it stops at done terminal ancestor" -- terminal boundary (line 95)
  - "it stops at cancelled terminal ancestor" -- terminal boundary (line 110)
  - "it handles deeply nested chain of 5+ levels" -- edge case from plan, 6 levels (line 125)
  - "it returns empty for task with no parent" -- no-parent boundary (line 148)
  - "it does not mutate any task" -- purity verification (line 395)
- Notes: All acceptance criteria and edge cases from the plan are covered. Tests verify the correct task IDs, actions, old/new statuses, and ordering of cascade changes. The purity test (no mutation) is thorough, checking both Status and Updated fields on all tasks including via the slice. Not over-tested -- each test targets a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, "it does X" naming, no testify.
- SOLID principles: Good. cascadeUpwardStart is single-responsibility (only Rule 2). CascadeChange is a clean data struct. StateMachine is stateless.
- Complexity: Low. Simple for-loop with switch on parent status. Linear walk up the parent chain with O(n) map construction.
- Modern idioms: Yes. Range-based iteration, map-based lookup, pointer-into-slice pattern for zero-copy task references.
- Readability: Good. Clear comments documenting the rule. Method name is self-documenting. Switch cases have inline comments explaining behavior.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
