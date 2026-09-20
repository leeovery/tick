## Attempt 1

ISSUES:
- `internal/cli/dep_tree_graph.go:263-270` — the "resume the passes after the cycle fallback" behaviour the task mandates is untested. The reviewer mutated the code in a scratch copy to `return unrooted` immediately after the fallback seed and the entire `internal/cli` suite stayed green. With that mutation, a project holding two independent cycles renders only the first: `trees=[tick-aaa111]` beside a summary still reading `2 chains, longest: 2, 4 blocked` — half the graph missing from pretty and JSON while the counts claim it, which is the §8 "counts and edge list never disagree" contradiction this work exists to delete. The current code is correct here (`trees=[tick-aaa111 tick-ccc333]`); nothing holds it in place. This branch is new with this task — the old builder needed no loop at all — so it is the one mandated behaviour shipping without a guard.
  FIX: Add a fixture beside `cycleTasks` (`internal/cli/dep_tree_test.go:215`) — `twoCycleTasks(now)`: `tick-aaa111` ↔ `tick-bbb222` and `tick-ccc333` ↔ `tick-ddd444`, two independent two-task cycles — and a builder subtest beside "it still covers a cycle where every participant is blocked" (`internal/cli/dep_tree_graph_test.go:334`), e.g. `"it seeds a second cycle the first cycle's seed does not reach"`, asserting `len(result.Trees) == 2` with tree IDs `tick-aaa111` and `tick-ccc333`, each carrying its partner as its single child, and the counts `(2, 2, 4)`. Two independent cycles is the minimal shape that exercises the resume: a cycle plus a dangling chain does not, because the ghost is seeded in the first pass before the fallback ever fires.
  ALTERNATIVE: Pin it at the CLI level instead (a `--json` subtest asserting two entries under `trees`). That covers the same branch end to end but is slower and sits a layer away from the loop being protected; the existing cycle coverage is split builder-level plus format-level, so the builder test is the closer match. The reviewer recommends the builder test.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- internal/cli/dep_tree_graph.go:236-238 — "so each chain is drawn once" is falsified by the builder's own documented behaviour: a participant two paths reach still has its chain drawn under each blocker (README.md:321, and the tests at `internal/cli/dep_tree_test.go:406` and `:419` pin exactly that duplication). A reader of this function would conclude the builder deduplicates subtrees.
  OLD: // A participant another seed will reach is held back until that seed has run, so each chain is
// drawn once; when only mutually blocked participants remain — a cycle — the first of them is
// seeded so its edges still reach the output.
  NEW: // A participant whose blockers are not yet emitted is held back until they are; when only mutually
// blocked participants remain — a cycle — the first of them is seeded so its edges still reach the
// output.

NOTES:
- `internal/cli/dep_tree_test.go:587-588` spells the box-drawing characters differently from every neighbouring pretty assertion. Cosmetic only; the bytes are identical.
- The executor's "worth knowing" paragraph about `danglingAboveChainTasks` is correct: `tick-bbb222` carries both `tick-aaa111` and `tick-ghost1` as blockers, so it genuinely has two paths and the repeat there is the documented diamond, not this task's defect.
- The reviewer stress-checked termination and coverage over 5000 random graphs including self-loops, multi-blocker fan-in and ghosts: no participant is ever dropped from `trees` and no call failed to terminate; over 5000 random DAGs, no task carrying a `BlockedBy` entry is ever a top-level tree.
