TASK: auto-cascade-parent-status-2-4 — Implement Cascades for downward done/cancel cascade (Rule 4)

ACCEPTANCE CRITERIA:
- Downward cascade propagates done/cancelled to non-terminal children recursively, leaves terminal children untouched (Rule 4)
- Edge cases: mixed terminal/non-terminal children, child with unresolved deps still cascaded
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: Rule 4 states: "When a parent is marked done or cancelled, non-terminal children (open, in_progress) copy the parent's terminal status. Children already done or cancelled are left untouched. Recursive — applies to grandchildren and beyond. Dependency state on target children does not gate cascade transitions — a child with unresolved dependencies is still cascaded. Consistent with the advisory dependency principle."

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:81-107 (`cascadeDownwardTerminal`)
- Notes: Implementation uses BFS via a queue starting from the changed task's ID. For each task in the queue, it finds children via `buildChildrenMap`, skips terminal children (done/cancelled), and emits a `CascadeChange` for non-terminal children with the target status derived from `transitions[action].to`. Non-terminal children are also enqueued to recurse into grandchildren. The approach correctly handles recursive descent without mutating tasks. The method is called from `Cascades()` at line 34 for both "done" and "cancel" actions.

TESTS:
- Status: Adequate
- Coverage:
  - "it cascades done to open children" (line 177) — basic done cascade
  - "it cascades cancel to in_progress children" (line 201) — cancel cascade to in_progress
  - "it cascades cancel to open children" (line 225) — cancel cascade to open
  - "it skips children already done" (line 249) — terminal skip (done)
  - "it skips children already cancelled" (line 266) — terminal skip (cancelled)
  - "it handles mixed terminal and non-terminal children" (line 283) — mixed edge case from plan
  - "it cascades recursively to grandchildren" (line 316) — recursive descent
  - "it cascades child with unresolved deps" (line 349) — advisory dependency edge case from plan
  - "it returns empty when all children are terminal" (line 366) — no-op case
  - "it returns empty for done/cancel on task with no children" (line 381) — leaf node
  - "it does not mutate any task on downward cascade" (line 806) — purity verification
- Notes: All edge cases from the plan (mixed terminal/non-terminal, unresolved deps) are explicitly tested. Tests verify action, old status, new status, and correct task targeting. Purity (no mutation) is verified separately. Test count is appropriate — each test covers a distinct scenario without redundancy.

CODE QUALITY:
- Project conventions: Followed — stdlib testing only, t.Run subtests with "it does X" naming, no external dependencies
- SOLID principles: Good — `cascadeDownwardTerminal` has single responsibility (compute downward cascade changes), `Cascades` dispatches by action type, `CascadeChange` is a clean data struct separating computation from mutation
- Complexity: Low — BFS loop with simple terminal-status check, straightforward queue pattern
- Modern idioms: Yes — idiomatic Go with range loops, map-based lookups, slice append
- Readability: Good — clear variable names (`childrenMap`, `targetStatus`, `queue`), well-commented function docs
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `transitions[action]` lookup on line 83 does not guard against an unknown action, but this is safe because `Cascades()` only calls this method for "done" and "cancel" actions (line 33-34), both of which exist in the transitions map. Still, a defensive check could be considered for future-proofing.
