TASK: free-text-round-trip-3-7 — The Full Dep-Tree Edge List Covers Every Participant (tick-98d2e0)

ACCEPTANCE CRITERIA:
1. The `dep_tree` section carries zero rows only when `chains`, `longest` and `blocked` are all zero, and the JSON `roots` array is empty only under the same condition.
2. Two tasks blocking each other emit rows `tick-aaa111,tick-bbb222` and `tick-bbb222,tick-aaa111` beside `chains: 1`, `blocked: 2` and `longest: 2`.
3. A project whose only blocker is an ID no task carries emits one row from that ID to the blocked task beside `chains: 1`, `blocked: 1`; in JSON that node carries its `id` with an empty `title` and `status`.
4. Root-reachable output is unchanged: the A → B → C fixture emits the same two rows and counts, the diamond still duplicates the shared node.
5. Pretty is unchanged on every input; the zero-roots branch still prints `No dependencies found.`
6. `BuildFullDepTree` terminates on every cycle shape and seeds each participant at most once.

STATUS: complete

SPEC CONTEXT: specification.md §8 (as amended by the corrigenda of 2026-09-19 and 2026-09-20) requires the full-graph document to be self-consistent: the counts derive from every participant, so the edge list must too. The corrigenda then moved past this task twice — the edge list became the stored `BlockedBy` relation itself (one row per recorded dependency, record order, no repeats) rather than a walk, and the seeding rule gained a hold-back with a first-in-order fallback for cycles. §8's "a participant that names no task says so" (corrigendum 2026-09-20) gives the ghost node status `missing` with an empty title; §4.1's third corrigendum has pretty draw the participants no root reaches instead of printing `No dependencies found.`

IMPLEMENTATION:
- Status: Implemented, then legitimately reshaped by later tasks in the same plan (7-2, 7-3, 8-1, 8-2, 10-3).
- Location: `internal/cli/dep_tree_graph.go:117-177` (`BuildFullDepTree`), `:208-228` (`collectParticipants`), `:230-235` (`collectTreeIDs`), `:237-238` (`depTreeMissingStatus`), `:240-279` (`buildSeededTrees`), `:281-285` (`blockedByUnemitted`), `internal/cli/format.go:249-273` (`DepTreeResult` with `Trees` and `Edges`), `internal/cli/toon_formatter.go:182-192` (`formatFullDepTree`), `internal/cli/json_formatter.go:380-388` (`formatFullDepTreeJSON`), `internal/cli/pretty_formatter.go:374-391`.
- Notes on the deltas from the task's own wording, all sanctioned by later plan tasks and recorded in the spec's corrigenda, and all preserving this task's outcome:
  - The task's `DepTreeResult.Unrooted` field no longer exists; `BuildFullDepTree` concatenates the root trees and the seeded trees into one `Trees` field (`dep_tree_graph.go:154`), and the seeded trees are built by `buildSeededTrees` (renamed from `buildUnrootedTrees`). Substance unchanged.
  - The toon `dep_tree` rows now come from `result.Edges` = `collectStoredEdges(tasks)` (`dep_tree_graph.go:180-201`, `toon_formatter.go:185`), one row per stored `BlockedBy` entry in record order. This covers every participant by construction — strictly stronger than this task's walk-seeding — so AC1/AC2/AC3's row assertions hold, with AC2's two rows in record order (`tick-bbb222,tick-aaa111` then `tick-aaa111,tick-bbb222`) rather than the order the criterion happened to list them in.
  - AC3's "empty `title` and `status`" was superseded by task 7-3: the ghost node carries `status: "missing"` (`dep_tree_graph.go:237-238, 250`), per §8's corrigendum of 2026-09-20. A deliberate improvement, not a loss — a blank status was the unmade decision the corrigendum names.
  - AC5's "pretty is unchanged" was superseded by task 3-10 and §4.1's third corrigendum: pretty draws every tree in `result.Trees` (`pretty_formatter.go:381`), so a cycle-only project now renders the cycle instead of `No dependencies found.`; the sentence survives only when the graph holds no participant at all (`dep_tree_graph.go:164-167`, `pretty_formatter.go:376-378`). Deliberate and recorded; this task's own commit (78e1cb5f) left pretty on `Roots` alone as the criterion required.
  - AC1 verified by reading: `Edges` is empty exactly when no task carries a `BlockedBy`, which is exactly when `collectParticipants` returns nothing — so `chains`, `blocked` and `longest` are all zero. Conversely, whenever an edge exists its blocker has a non-empty `blocks` entry, so either the root loop or `buildSeededTrees`' fallback produces a tree, keeping the JSON `trees` array (renamed from `roots` by task 7-2) non-empty. `longest` is computed over roots and seeded trees together (`dep_tree_graph.go:154-158`), so it tracks the rows.
  - AC6 verified by reading: `buildSeededTrees` seeds only ids for which `!emitted[id]` holds and sets `emitted[id] = true` inside `seed`, so no participant is seeded twice; each outer iteration either seeds at least one participant or takes the `slices.IndexFunc` fallback and seeds one, so the loop strictly shrinks a finite unemitted set and terminates. `walkDownstream`'s ancestor guard (`dep_tree_graph.go:44-69`) bounds each individual walk, including a self-block (`A` blocked by `A`) and n-node cycles.

TESTS:
- Status: Adequate
- Coverage: `internal/cli/dep_tree_test.go:365-379` pins the cycle's two rows beside `chains: 1`, `longest: 2`, `blocked: 2`; `:381-394` the dangling-blocker row beside `chains: 1`, `blocked: 1`; `:339-363` the two zero-count emptied-document states (AC1's zero case); `:610-626` the cycle under the JSON `trees` key; `:628-636` the ghost node's `missing` status in JSON; `:656-672` and `:675-691` the cycle and the ghost in the terminal (the post-3-10 replacement for the criterion's pretty subtest); `:712-729` and `:731-745` pin the rooted A → B → C output unchanged in both pretty and toon (AC4); `:442-480` and `internal/cli/dep_tree_graph_test.go:750-779` pin diamond duplication. Unit-level seeding is pinned at `dep_tree_graph_test.go:294-314` (ghost seeded with `missing`), `:316-347` (the dangling blocker seeded rather than the chain beneath it), `:349-372` (cycle coverage with counts), `:373-394` (a second, independent cycle seeded — the fallback resuming), `:395-420` (real statuses preserved on cycle members) and `:726-748` (full-graph termination on a cycle).
- Notes: No redundancy worth flagging — each subtest carries a distinct fixture (`cycleTasks`, `twoCycleTasks`, `danglingBlockerTasks`, `danglingChainTasks`, `danglingAboveChainTasks`, `chainTasks`, `diamondTasks`) and asserts a different output surface (toon rows, JSON trees, pretty text, builder result). The tests would fail if the seeding or the edge enumeration regressed: removing the seeded-tree pass zeroes `longest` and empties `Trees` on every cycle and ghost fixture, which four subtests assert against directly.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` "it does X" subtests, shared fixtures in the package's test files, pretty asserted through `runDepTree` and toon through `runToonCommand`.
- SOLID principles: Good — graph construction stays in `dep_tree_graph.go`, rendering decisions stay in the three formatters; `buildSeededTrees`, `collectParticipants`, `collectTreeIDs` and `blockedByUnemitted` each hold one responsibility.
- Complexity: Acceptable — the `for {}` pass loop in `buildSeededTrees` is the one non-obvious construct and its comment (`dep_tree_graph.go:240-245`) states both the hold-back rule and the cycle fallback.
- Modern idioms: Yes — `slices.Concat`, `slices.IndexFunc`, `slices.ContainsFunc`, the `max` builtin.
- Readability: Good.
- Comment accuracy: The doc comments on `BuildFullDepTree`, `buildSeededTrees`, `blockedByUnemitted`, `collectParticipants`, `depTreeMissingStatus` and `DepTreeResult` all hold against the current code; none references a task id, phase or spec section. `README.md:321` describes the final seeding and edge-list behaviour correctly.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Root-reachable output is unchanged: the A → B → C fixture emits the same two rows and the same counts as today, and the diamond still duplicates the shared node — `internal/cli/dep_tree_graph_test.go:615-643` stays green unedited." — reading confirms the diamond subtest's assertions are unchanged in substance (only the `Roots` → `Trees` field rename by a later task touched it, `dep_tree_graph_test.go:750-779`) and that the rooted fixtures are pinned at `dep_tree_test.go:712-745`; whether those tests are green needs `go test ./internal/cli`.
- "Pretty is unchanged on every input: ... `internal/cli/dep_tree_test.go:183-207` and `internal/cli/dep_tree_graph_test.go:599-613` stay green unedited." — superseded in substance by task 3-10 and §4.1's third corrigendum; what remains to settle by running is that the surviving pretty assertions (`dep_tree_test.go:313-338`, `:656-745`) pass.
- "`BuildFullDepTree` terminates on every cycle shape and seeds each participant at most once." — reading establishes termination (each outer pass of `buildSeededTrees` shrinks the unemitted-seed set) and at-most-once seeding (`emitted[id]` set inside `seed`); a run of `go test ./internal/cli` is what would show no shape hangs in practice.
