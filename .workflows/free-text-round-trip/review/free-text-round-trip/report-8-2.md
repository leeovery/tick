TASK: free-text-round-trip-8-2 (tick-a69bd6) — A Participant Another Seeded Tree Reaches Is Not A Top-Level Tree

ACCEPTANCE CRITERIA:
- `BuildFullDepTree` on the two-level dangling chain returns exactly one tree: `tick-ghost1` (status `missing`, empty title) → `tick-bbb222` → `tick-aaa111`.
- Pretty draws that chain once, the ghost as the only top-level entry; no participant carrying a `BlockedBy` entry is drawn as a top-level entry on that input.
- JSON returns one object under `trees` for that input, with the `tick-bbb222` subtree appearing once.
- `dep_tree` carries two rows for the two-edge graph, one per stored dependency, neither repeated.
- The fixture's summary reads `1 chain, longest: 2, 2 blocked` in all three formats.
- Cycle coverage is unchanged: `cycleTasks` still yields the single `tick-aaa111` → `tick-bbb222` → `tick-aaa111` tree with `chains: 1`, `longest: 2`, `blocked: 2`.
- The single dangling blocker and the rooted chain are unchanged, and every participant still reaches every format.
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean.

STATUS: complete

SPEC CONTEXT: specification.md:328 ("The node-shaped renderings still walk") states the rule this task implements: after the walk from the roots, any participant not yet drawn seeds a further walk, "held back while any of its own blockers is still undrawn so that each chain is drawn once and a top-level entry is a participant nothing else drawn reaches"; where every remaining participant is blocked by another undrawn one — a cycle — "the first in order is seeded so its edges still reach the output, and the passes resume afterwards so an independent second cycle is drawn too". The same paragraph records the accepted limit: the fallback seeds the first remaining participant whether or not it is in a cycle, so on a cycle carrying dependents a subtree can still be drawn twice. The dated corrigenda at specification.md:557 and :559 record the output change and the limit, so the spec and the code agree. README.md:321 carries the user-facing form of the rule, updated in the phase-8 consolidation commit (890ffb69) to "then seeds a tree from each participant nothing already drawn reaches … with its downstream chain nested beneath it".

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/dep_tree_graph.go:246-279 (`buildSeededTrees`, rewritten into repeated passes with a `seed` closure and a `needsSeed` predicate), internal/cli/dep_tree_graph.go:283-285 (`blockedByUnemitted`), comment at internal/cli/dep_tree_graph.go:240-245. Commit 4cb71c27, comments refined in 890ffb69.
- Notes: Every bullet of the task's **Do** is present and nothing beyond it moved. The two existing skips survive inside `needsSeed` (`!emitted[id] && len(blocks[id]) > 0`); the hold-back reads the participant's `BlockedBy` out of `taskIdx`, and a participant no record matches yields a nil slice so a ghost is never held back. The root walk (`:123-144`), the `emitted`/`collectTreeIDs` bookkeeping, `depTreeMissingStatus` and the `chains`/`longest`/`blocked` computation are untouched, confirmed against the commit diff (the graph file's only hunks are inside `buildSeededTrees` and the new helper).

  Traced by hand against the fixtures:
  - `danglingChainTasks`: participants come out `[aaa111, bbb222, ghost1]`; `aaa111` blocks nothing, `bbb222` is held back on unemitted `ghost1`, `ghost1` seeds and its walk draws `bbb222` → `aaa111`. One tree, `longest: 2`, `chains: 1`, `blocked: 2` — matches criteria 1 and 5.
  - `cycleTasks`: both participants are held back, the pass emits nothing, `slices.IndexFunc(participants, needsSeed)` returns `aaa111`, and its walk reproduces the previous `aaa111 → bbb222 → aaa111` tree with the same counts — criterion 6 holds by construction, not merely by assertion.
  - `danglingAboveChainTasks` (a root plus a ghost over the same chain) still draws the shared branch under both blockers; that is the documented diamond case (README.md:321) and the task's problem statement explicitly excludes it, so no regression there.
  - Termination is guaranteed: every `seed` call marks its id emitted and `needsSeed` requires `!emitted`, so the unemitted-and-blocking set strictly shrinks on each outer iteration, including the fallback branch.
  - No participant is lost: at termination every unemitted participant blocks nothing, so it has a blocker; that blocker blocks it, hence must be emitted, and every emission expands the node's full child list on its first (non-cut) occurrence — so the child is emitted too. The "every participant still reaches every format" half of criterion 7 holds under reading.
- Drift: none. Ordering of top-level entries in multi-seed graphs moves (intended, and the substance of the change); the accepted fallback limit is recorded in the spec rather than fixed, as the task prescribes.

TESTS:
- Status: Adequate
- Coverage: All five tests named in the task exist, each pinning a different surface of the same fixture:
  - internal/cli/dep_tree_graph_test.go:316 "it seeds the dangling blocker rather than the chain beneath it" — one tree, ghost (`missing`, empty title) → `tick-bbb222` → `tick-aaa111`, plus the summary string.
  - internal/cli/dep_tree_test.go:693 "it renders a two-level dangling chain once in the terminal" — exact bytes, ghost line then `└── tick-bbb222`, `    └── tick-aaa111`, then the summary. The ghost line's three-space gap matches `writeDepTreeTaskLine` (internal/cli/pretty_formatter.go:422-425) rendering an empty title, the same shape the pre-existing dangling-blocker test pins.
  - internal/cli/dep_tree_test.go:638 "it nests a two-level dangling chain under the ghost in JSON" — `jsonDepTreeOnlyTree` (internal/cli/dep_tree_test.go:173-184) fails unless `trees` holds exactly one entry, and `jsonDepTreeOnlyChild` enforces the single-child nesting, so the "appears once" half of criterion 3 is genuinely observed.
  - internal/cli/dep_tree_test.go:396 "it emits two edges for a two-edge dangling chain" — two `dep_tree` rows plus the counts.
  - internal/cli/dep_tree_graph_test.go:349 "it still covers a cycle where every participant is blocked" — the fallback regression pin, with counts (1, 2, 2).
  New fixtures `danglingChainTasks` (internal/cli/dep_tree_test.go:241-248) and `twoCycleTasks` (:223-231) sit beside `danglingBlockerTasks` as the task asked.
  One test beyond the list: internal/cli/dep_tree_graph_test.go:373 "it seeds a second cycle the first cycle's seed does not reach". It covers the third **Do** bullet ("resume the passes"), which nothing else observes — without it, a fallback that returned instead of resuming would pass every other test. Justified, not scope creep.
- Notes: The tests would fail if the feature broke — reverting the hold-back gives the dangling chain two trees, which the builder, pretty, and JSON tests each catch independently. Assertion overlap with the older internal/cli/dep_tree_graph_test.go:726 "it terminates full graph with circular dependency" is partial (the new test adds the back-edge node and the three counts) and the plan names the test explicitly as the fallback pin; not over-tested. No existing assertion was weakened or deleted — the commit's test hunks are additions only.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing`, `t.Run()` subtests in "it does X" form, fixtures as helper constructors beside the existing ones, `t.Helper()` on the shared JSON helpers, CLI tests driving pretty explicitly. Tab-indented and gofmt-shaped.
- SOLID principles: Good. `blockedByUnemitted` is a named predicate with one job; `seed`/`needsSeed` keep the pass loop readable without leaking state outside the function.
- Complexity: Acceptable. The nested loop reads as "repeat passes until nothing moves, then break the deadlock", and each branch is three lines.
- Modern idioms: Yes — `slices.ContainsFunc` and `slices.IndexFunc` rather than hand-rolled loops, consistent with the repo's modernize lint set.
- Readability: Good. The function comment states the seeding rule, the hold-back and the fallback, and each sentence holds against the code: a ghost carries no blockers because `taskIdx[id]` returns the zero task, and the fallback fires exactly when a pass emits nothing.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean." — requires executing the suite, vet, gofmt and golangci-lint; judged sound by reading (formatting is tab-indented and gofmt-shaped, no unused symbols or imports introduced, `slices` was already imported for `slices.Concat`), but only a run settles it.
- "The fixture's summary reads `1 chain, longest: 2, 2 blocked` in all three formats." — derivation checked by hand (`chains` = 1 connected component over {aaa111, bbb222, ghost1}, `longest` = 2 edges from the ghost, `blocked` = 2 records carrying `BlockedBy`) and pinned by three tests, but the byte-level agreement across toon, pretty and JSON is settled only by running them.
- "The single dangling blocker and the rooted chain are unchanged … and every participant still reaches every format." — the unchanged-assertion half is settled by reading (no existing test was modified), and the participant-coverage half is argued above; a green suite run confirms both, including the conformance entries at internal/cli/conformance_test.go:284-305 that exercise the full graph on a populated graph and on a cycle.
