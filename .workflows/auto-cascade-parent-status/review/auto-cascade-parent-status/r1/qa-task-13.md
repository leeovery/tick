TASK: auto-cascade-parent-status-2-7 — Implement ApplyWithCascades with queue-based processing

ACCEPTANCE CRITERIA:
- ApplyWithCascades() returns primary TransitionResult plus all []CascadeChange entries
- Queue-based processing with seen-map deduplication terminates correctly on deep hierarchies
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: The spec defines ApplyWithCascades as the main orchestration entry point. It calls Transition() on the target, then runs a cascade queue: compute cascades, pop from queue, apply each via Transition(), compute further cascades from each applied change, track processed tasks in a seen-map to deduplicate, loop until queue empty. Returns primary TransitionResult plus all CascadeChange entries. The spec also states Rule 9 (block reopen under cancelled parent) should be enforced. Transition history records with Auto: true/false must be appended.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/apply_cascades.go:18-99
- Notes: Implementation follows the spec closely. Queue-based processing with seen-map deduplication is correctly implemented. Key design decisions:
  - Rule 9 check is performed before the primary transition (lines 20-30), preventing mutation on error
  - Primary transition applied via sm.Transition() (line 33), transition history recorded with Auto: false (lines 39-44)
  - taskMap built from slice for pointer-stable lookups (lines 48-49)
  - Initial cascades computed and copied into queue (lines 51-55)
  - Target pre-seeded in seen-map (line 58)
  - Queue loop pops front, checks seen-map, applies transition, records Auto: true history, computes further cascades (lines 63-95)
  - Invalid cascade transitions are silently skipped (line 77) — reasonable since cascade logic should only produce valid transitions
  - Value receiver on StateMachine is consistent with all other methods (stateless struct)
  - Spec says pointer receiver (`*StateMachine`) but all methods consistently use value receiver — acceptable since struct has no fields

TESTS:
- Status: Adequate
- Coverage: 14 test cases covering:
  1. Primary transition with no cascades (empty cascade list edge case)
  2. Transition history recording on primary task (Auto: false)
  3. Single-level upward start cascade
  4. Multi-level downward cancel cascade (3 levels)
  5. Chained upward completion cascade (Rule 3 chains through grandparent)
  6. Seen-map deduplication verification
  7. Empty cascades for leaf task
  8. Invalid primary transition returns error without mutation
  9. Reopen cascade chain (Rule 5, 3 levels deep)
  10. Auto flag verification on cascaded tasks
  11. Block reopen under cancelled parent (Rule 9)
  12. Allow reopen with no parent
  13. Allow reopen under open/done/in_progress parent
  14. Allow reopen when grandparent cancelled but direct parent is not
  15. Non-existent parent ID handling
  16. Non-reopen actions skip Rule 9 check
- Notes: All three edge cases from the plan task are covered: multi-level cascade chains (tests 4, 5, 9), seen-map deduplication (test 6), empty cascade list (tests 1, 7). The tests verify both the returned results and in-place mutations on the tasks slice. Transition history with correct Auto flags is verified on cascaded tasks. Test count is appropriate — each test covers a distinct behavior or edge case without redundancy.

CODE QUALITY:
- Project conventions: Followed — stdlib testing only, t.Run subtests, "it does X" naming, t.Helper on helpers, error wrapping patterns
- SOLID principles: Good — single responsibility (ApplyWithCascades orchestrates, Cascades computes, Transition mutates), clean separation of concerns
- Complexity: Low — linear queue processing loop, straightforward control flow, no nested conditionals beyond the Rule 9 guard
- Modern idioms: Yes — range over slice indices for pointer stability, map-based deduplication, copy for queue initialization
- Readability: Good — well-commented steps (Step 1-6), clear variable names, doc comment explains contract and pointer requirements
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The queue implementation uses `queue[1:]` slice reslicing which is fine for the expected cascade sizes but would not release memory for very large queues. Not a practical concern given task hierarchies are shallow.
