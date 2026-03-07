TASK: Implement Cascades for reopen under done parent (Rule 5)

ACCEPTANCE CRITERIA:
- Reopen child under done parent reopens parent recursively up ancestor chain (Rule 5)
- Edge cases: parent not done no-ops, deeply nested done ancestors, cancelled ancestor blocks
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rule 5 ("Auto-done undo") states that when a child is reopened under a done parent, the parent reopens to open. This applies recursively up the ancestor chain via the cascade queue. Non-done ancestors (open, in_progress, cancelled) stop the walk. The spec explicitly notes that cancelled ancestors block the reopen cascade -- the chain stops there.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:141-171 (cascadeUpwardReopen method)
- Notes: Implementation is clean and correct. The `cascadeUpwardReopen` method walks the parent chain, emitting CascadeChange entries for each done ancestor, and stops at any non-done ancestor (open, in_progress, cancelled). This matches the spec exactly. The method is invoked from `Cascades()` at line 38 when action is "reopen". The method is pure -- it does not mutate any tasks, consistent with the Cascades contract.

TESTS:
- Status: Adequate
- Coverage:
  - Basic reopen cascade to done parent (cascades_test.go:598)
  - Multiple done ancestors cascade recursively (cascades_test.go:623)
  - Stops at non-done (open) parent -- no-ops (cascades_test.go:655)
  - Stops at open parent with done grandparent above (cascades_test.go:667)
  - Stops at cancelled ancestor (cascades_test.go:681)
  - Deeply nested done ancestors 5+ levels (cascades_test.go:702)
  - Non-reopen actions don't trigger Rule 5 (cascades_test.go:730)
  - No parent -- returns empty (cascades_test.go:747)
  - Purity check -- no mutation on reopen cascade (cascades_test.go:758)
  - Integration: ApplyWithCascades reopen cascade chain (apply_cascades_test.go:284)
  - Integration: reopen under done parent (apply_cascades_test.go:445)
  - Integration: reopen under cancelled grandparent with done parent (apply_cascades_test.go:478)
- Notes: All three edge cases from the task are covered (parent not done no-ops, deeply nested done ancestors, cancelled ancestor blocks). Tests verify both the pure Cascades() output and the full ApplyWithCascades integration. The purity test ensures no side effects. Coverage is thorough without being redundant.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, t.Helper on helpers, error wrapping
- SOLID principles: Good -- cascadeUpwardReopen is a focused private method with single responsibility; Cascades() dispatches cleanly
- Complexity: Low -- simple linear walk up parent chain with a single conditional
- Modern idioms: Yes -- idiomatic Go patterns throughout
- Readability: Good -- method is self-documenting with clear comments referencing Rule 5
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
