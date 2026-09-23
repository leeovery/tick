TASK: Keep The Order Total Across The Ready Band, Blocked And Duplicate Sequences (same-second-tasks-sort-by-id-1-2, tick-69d78d)

ACCEPTANCE CRITERIA:
- Open tasks recording one creation second and one priority, plus an `in_progress` task of the same priority authored after them in the same second, all with IDs contradicting authoring order: `tick ready` lists the `in_progress` task first, then the open tasks in authoring order (§1.3, §8.1)
- Tasks blocked by a common open blocker, recording one creation second and one priority, with IDs contradicting authoring order: `tick blocked` returns them in authoring order (§4.1, §8.1)
- Children of a parent P recording one creation second and one priority and all carrying the same non-zero sequence, with IDs whose ascending order contradicts their line order in `tasks.jsonl`: `tick list` and `tick list --parent P` both return them in ascending task-ID order, identically on repeated runs (§5.1, §8.3)
- The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified (§7.3, §8.5)

STATUS: complete

SPEC CONTEXT: §4.1 requires both `buildListQuery` clauses to end priority, created, sequence, id (ready view with the `in_progress` band first). §5.1 makes the task ID the absolute final term so a duplicate sequence (reachable via git-tracked `tasks.jsonl` merged across worktree branches, §5) falls back to ID order deterministically instead of varying with the query plan. §8.1 requires separate tie assertions for the ready band (band must beat the sequence) and for `tick blocked` (different WHERE shape from `BlockedConditions()`). §8 fixture constraints: IDs contradict authoring order, creation seconds tie, and for the duplicate case line order must not coincide with ascending-ID order.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/list.go:313 (ready clause: `(t.status = 'in_progress') DESC, t.priority ASC, t.created ASC, t.seq ASC, t.id ASC`), internal/cli/list.go:315 (neutral clause: `t.priority ASC, t.created ASC, t.seq ASC, t.id ASC`)
- Notes: Both clauses end on `t.id`, which is `TEXT PRIMARY KEY` (internal/storage/cache.go:19), so the order is total under every WHERE shape and plan. The band term stays first in the ready clause. `tick ready` and `tick blocked` reach this builder via `parseListFlags` with `--ready`/`--blocked` prepended (internal/cli/app.go:219, :232) and `RunList` -> `buildListQuery` (internal/cli/list.go:186); `tick blocked` takes the neutral clause with `BlockedConditions()` as its WHERE (internal/cli/list.go:271-273). `LIMIT` is still appended after the now-total ORDER BY (internal/cli/list.go:318-321), so `--count` cuts a defined order. The line numbers differ from the plan's (:317/:319) because the band comment above the ready clause was later removed in the analysis-cycle comment corrections; no drift in substance. Backfill only fills zero/absent sequences (internal/storage/jsonl.go:126), so a fixture's duplicate non-zero sequences reach the cache intact.

TESTS:
- Status: Adequate
- Coverage: `TestTotalListFamilyOrdering` in internal/cli/list_order_test.go:
  - :365 ready band. Open e00001/d00002/c00003 (seq 1-3) plus in_progress a00004 (seq 4), all same second and priority 2, IDs reversed against authoring. It expects [a00004, e00001, d00002, c00003]. This fails if the band is dropped (seq puts a00004 last), if seq is moved ahead of the band (same result), or if seq is dropped (ID order reverses the open tasks).
  - :383 blocked. A common open blocker plus three blocked tasks with reversed IDs, reached through `tick blocked`. This guards the seq term on the BlockedConditions route.
  - :402 duplicate sequence. Parent seq 1; children all seq 7; line order c,a,d,b against ascending a,b,c,d. It asserts `list`, `list --parent P` and `ready` across three runs. The `ready` assertion at :424 covers the ready clause's ID term and was added in fix round 1.
  - Criterion 4: `git diff 937a299e HEAD` shows no change to internal/cli/ready_test.go, blocked_test.go or list_filter_test.go. In each fixture (ready_test.go:306, blocked_test.go:211, list_filter_test.go:349), the rows being compared never tie on (priority, created), so the new final terms cannot reorder them.
- Notes: Every fixture uses non-zero sequences and a single creation second, as §8 requires. The three repeated runs query one cache and one plan, so the repetition adds little beyond a single run, but it meets the criterion as written. Determinism across plans comes from the total ORDER BY, and the unfiltered assertions (the ones that separate file order from ID order) are present. No over-testing.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` "it ..." subtests, `t.Helper()` on helpers, toon output decoded via `decodeToonDoc`/`toonRows`)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (`for run := range 3`, `slices.Equal`)
- Readability: Good. The fixture comments (list_order_test.go:323-324, :403) are accurate.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified" — reading settles "unmodified" (git diff empty) and shows no (priority, created) ties that the new terms could reorder; the pass itself needs `go test ./internal/cli -run 'TestReady|TestBlocked|TestListFilter'` to be run
