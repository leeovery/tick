## Attempt 1

ISSUES:
- /Users/leeovery/Code/tick/internal/cli/list.go:317 — the ready clause's final `t.id ASC` has no test, even though the task's Do step and §5.1 ("Every clause in §4.1 ends on the task ID") require it.
  - The two ORDER BY clauses are separate string literals, so they can drift apart without any error.
  - The duplicate-sequence subtest only runs `list` and `list --parent`, which both use the neutral clause.
  - The ready band test cannot catch this, because its fixture never ties on seq.
  - Mutation result: with the ID term removed from :317 only, every test passes. On the duplicate fixture, `tick ready` then returns file order c,a,d,b instead of a,b,c,d.
  - Why it matters: `tick ready --count 1` is the §1.2 wrong-selection case this work exists to fix. After a branch merge that duplicates a sequence, an edit that drops or reorders the ready clause's tail makes that selection depend on the query plan again, and nothing in the suite would notice.
  - The mutation results for the other terms match the executor's report: dropping the band or moving seq ahead of it fails the ready test, dropping seq fails the blocked test, and dropping id from the neutral clause fails the unfiltered list assertion.
  FIX: In the duplicate-sequence subtest's loop (list_order_test.go:416-422), add `assertIDOrder(t, listedIDs(t, dir, "ready"), children)` after the two existing `list` assertions. The children are open leaves, so all four are ready. In the scratchpad copy, this assertion failed with the ready ID term removed and passed with it restored. A `ready --parent` assertion may be added for symmetry, but it passes without the ID term: the filtered plan returns ID order by accident. The unfiltered `ready` assertion is the one that tells the two apart.
  ALTERNATIVE: A separate subtest (e.g. "it falls back to ascending ID order for ready tasks sharing a sequence") with its own fixture. That gives a clearer failure name but duplicates the fixture. I recommend the one-line addition to the existing loop.
  CONFIDENCE: high

NOTES:
- The `.tick/tasks.jsonl` diff only moves tick-69d78d (this task) from open to in_progress. That is workflow dogfooding, not executor output.
- The executor reports that the ready and blocked subtests passed before the change. That is expected: they guard the band's position and the blocked route's use of seq, which were already correct before this task.
- Children of P tie on seq, so the `list --parent` assertion in the duplicate subtest (list_order_test.go:418) passes even without the ID term: the parent-filtered plan returns ID order by accident. The unfiltered `list` assertion at :417 is the one that tells the two apart. This follows from the plans and is not a fixture defect.
- The three repeated runs query the same cache and plan, so the repetition proves little beyond the single run. It does meet the criterion's "identically on repeated runs" as written.
- The comment at list.go:313-316 existed before this task and is still accurate: both clauses gained the same tail, so the "byte-identical to the neutral clause" claim holds.
