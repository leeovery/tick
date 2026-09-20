# Analysis Tasks: Free Text Round Trip (Cycle 2)

## Task 1: The Full-Graph Edge List Carries One Row Per Stored Dependency

severity: medium
sources: architecture, standards

**Problem**: `ToonFormatter.formatFullDepTree` derives the `dep_tree[N]{from,to}` rows by flattening
`result.Trees` (`internal/cli/toon_formatter.go:183-190` calling `collectDownstreamEdges` at
`:219-226`, which appends without deduplicating). A tree walk cannot represent a DAG without
repeating the shared part, so every path into a convergence point re-emits the whole subtree below
it. Ordinary CLI usage, no file editing: `create Alpha`, `create Alpha2`,
`create Beta --blocked-by=Alpha,Alpha2`, `create Gamma --blocked-by=Beta` stores three dependencies
and returns `dep_tree[4]` carrying `Beta,Gamma` twice, beside `chains: 1`. The seeded-tree path this
work added widens it to a second class: `Alpha→Beta→Gamma` plus a blocker ID no record matches above
`Beta` returns `dep_tree[4]` with `Beta,Gamma` twice. An agent that builds a graph from those rows
answers "what does Beta block" with 2 where `tick show Beta` answers 1 — it double-counts a
dependency, or acts on a graph carrying a phantom edge and never notices. Both halves of one
document then disagree in exactly the way §8's corrigendum was written to prevent: the summary says
one chain while the edge list lists four edges for three dependencies, and the reader is back to a
rule learned outside the output ("dedupe the rows yourself"). The two-root case predates this work;
the seeded case is new with it, and no test covers a seed whose walk overlaps an emitted tree.

**Solution**: Derive the full-graph rows from the relation that holds them rather than from a walk.
`BuildFullDepTree` already holds every `BlockedBy` entry; one pass over it yields exactly one
`{from,to}` row per stored dependency, covers every participant by construction, and cannot repeat —
so the toon edge list stops depending on the seeding machinery entirely. Carry the rows on
`DepTreeResult` (an `Edges []DepTreeEdge` field) and have `ToonFormatter.formatFullDepTree` render
those instead of walking `result.Trees`. `Trees` is left to the node-shaped renderings, where drawing
a shared subtree under each of its blockers is a drawing rather than a duplicated fact, so pretty and
JSON are byte-identical on every input and README.md:321's documented diamond duplication stays true
of them. Row order moves from walk order to record order, so the dep-tree edge assertions need their
expected rows reordered; the existing toon diamond assertion
(`internal/cli/toon_formatter_test.go:933`) expects four rows for four distinct stored dependencies
and survives on content, order aside. This is a change to shipped output, so it is recorded as a
dated corrigendum against §8 in the same form the earlier dep-tree output changes used.

Grounds for moving what phase 3's task 1 held fixed ("Root-reachable output is unchanged … the
diamond still duplicates the shared node"): that criterion scoped a task, it did not settle that
repeated rows are right, and the defect is measured — four rows for three dependencies beside
`chains: 1` is the self-contradiction §8's corrigendum widened the edge list to remove, and §1's bar
is output an agent can read without a rule learned outside it. Every participant still reaches the
edge list, so the settled direction of phase 3's task 1 is completed, not reversed.

**Outcome**: The edge list is the relation: one row per stored dependency, no row twice, and an
agent's count of what a task blocks agrees with `tick show`. Pretty and JSON full-graph output is
byte-identical to today on every input, and the summary fields do not move. The dated §8 corrigendum
recording this output change is the orchestrator's to write: this task touches code, tests and
`README.md` only, and nothing under `.workflows/`.

**Do**:
- Add a `DepTreeEdge` type (`From`, `To` strings) beside `DepTreeTask` in `internal/cli/format.go`,
  and an `Edges []DepTreeEdge` field in `DepTreeResult`'s full-graph block (`format.go:212-233`).
  Focused mode leaves it nil.
- Populate `Edges` in `BuildFullDepTree` (`internal/cli/dep_tree_graph.go:117-182`): one
  `{From: dep, To: t.ID}` per `BlockedBy` entry, tasks in slice order and each task's `BlockedBy` in
  stored order. Leave `Trees`, `ChainCount`, `LongestChain`, `BlockedCount` and `Message` exactly as
  they are built today, and leave `BuildFocusedDepTree` untouched.
- Render `result.Edges` in `ToonFormatter.formatFullDepTree`
  (`internal/cli/toon_formatter.go:183-199`) — map each `DepTreeEdge` to a `toonEdgeRow` and hand it
  to `buildEdgeSection("dep_tree", …)` — dropping the `collectDownstreamEdges` walk over
  `result.Trees`. `collectDownstreamEdges` and `collectUpstreamEdges` stay for focused mode
  (`toon_formatter.go:203-215`).
- Repoint the existing expectations: the five full-graph subtests in `TestToonFormatDepTree` that
  build `Trees` and assert edge rows (`internal/cli/toon_formatter_test.go:863`, `:882`, `:907`,
  `:933`, `:989`) supply `Edges` instead, and the cycle assertion in
  `internal/cli/dep_tree_test.go:307-310` swaps its two rows into record order
  (`tick-bbb222,tick-aaa111` then `tick-aaa111,tick-bbb222`).
- Narrow `README.md:321`'s closing sentence ("Diamond dependencies are duplicated at each path") to
  the node-shaped renderings, so it no longer claims the duplication of the toon edge list.

**Acceptance Criteria**:
- [ ] `dep_tree` carries exactly one row per stored dependency: its row count equals the total number
      of `BlockedBy` entries across all tasks, on every fixture in `internal/cli/dep_tree_test.go`.
- [ ] Two blockers converging above a further-blocked task emit the shared subtree once — `create
      Alpha`, `create Alpha2`, `create Beta --blocked-by=Alpha,Alpha2`, `create Gamma
      --blocked-by=Beta` returns `dep_tree[3]` with the Beta→Gamma row present once.
- [ ] A blocker ID no task record matches still reaches the edge list as the `from` of its
      dependency's row, and emits nothing extra when its walk overlaps an already-emitted tree.
- [ ] Row order is record order: tasks in stored order, each task's `BlockedBy` in stored order.
- [ ] `chains`, `longest` and `blocked` are unchanged on every fixture, and an empty project still
      emits `dep_tree[0]{from,to}:`.
- [ ] Pretty and JSON full-graph output is byte for byte what it is today, including the diamond's
      duplicated subtree; focused mode's `blocked_by` and `blocks` sections are unchanged.
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are
      clean.

**Tests**:
- `"it emits one edge per stored dependency where two blockers converge"` — Alpha/Alpha2 → Beta →
  Gamma: three rows, Beta→Gamma once, beside `chains: 1`.
- `"it emits no repeated edge when a dangling blocker sits above a chain"` — `tick-aaa111` →
  `tick-bbb222` → `tick-ccc333` plus a ghost ID blocking `tick-bbb222`: four rows for four stored
  dependencies, none repeated.
- `"it carries each edge of a cycle once"` — the `cycleTasks` fixture, two rows in record order.
- `"it still emits the count-zero edge section when no task has dependencies"`.
- `"it still duplicates the diamond in the pretty and JSON trees"` — the four-task diamond renders
  `tick-ddd444` under both `tick-bbb222` and `tick-ccc333` in both formats while `dep_tree` carries
  four rows.

## Task 2: A Participant Another Seeded Tree Reaches Is Not A Top-Level Tree

severity: medium
sources: standards, architecture

**Problem**: `buildSeededTrees` (`internal/cli/dep_tree_graph.go:220-241`) skips a participant an
*earlier* tree already emitted, but nothing holds back a participant a *later* seed will reach, and
it ignores `emitted` for the duration of its own walk. `collectParticipants`
(`:183-206`) adds a blocked task before its blockers, so the blocked-but-blocking middle of a
dangling chain is always visited before the dangling blocker that reaches it — the ordering makes the
redundant seed the normal case rather than a corner. In a project holding an orphaned dependency one
level above a real chain — task X blocked by Y, Y blocked by an ID no task carries —
`tick dep tree` prints `dep_tree[3]{from,to}:` for a two-edge graph, repeating `Y,X`; pretty draws
the whole `Y → X` chain twice, once as a top-level entry and once nested under the ghost; and JSON
returns the same subtree twice. A terminal reader sees Y presented as a top-level tree although Y is
blocked, which is the one thing a top-level entry is supposed to rule out: README.md:321 states the
rule the output breaks ("the pretty tree draws root tasks (tasks that block others but aren't
blocked themselves) with their downstream chains, then the tasks no root reaches"). This is not the
documented diamond duplication — a diamond repeats a node that genuinely has two paths, whereas here
Y has exactly one. `internal/doctor/orphaned_dependency.go` exists because the project expects this
state, and before this work neither the ghost nor the chain reached the output at all, so the whole
rendering is new with it. No test pins the current shape: the dangling-blocker fixture
(`internal/cli/dep_tree_test.go:223`, `danglingBlockerTasks`) is one task directly under the ghost,
where the ordering never bites.

**Solution**: Seed only participants that no other seeded tree reaches — hold back a participant that
an unemitted participant blocks, so the dangling blocker is seeded first and the chain beneath it is
drawn once, inside that seed's walk. When a pass emits nothing because every remaining participant is
blocked by another remaining participant — a cycle — fall back to seeding the first remaining
participant, which keeps cycle coverage exactly as it is today, including the cycle fixture's pinned
`chains: 1`, `longest: 2`, `blocked: 2`. Every participant still reaches every format, so phase 3's
task 1 (the edge list covers every participant), the ad hoc pass's direction (the terminal stops
denying dependencies that exist) and cycle 1's task 4 (a participant naming no task renders as
`missing`) all keep their outcomes; only which participant is drawn as a top-level entry moves. The
root walk and rooted output are untouched. Cover it with the two-level dangling chain above, the
shape that exposes the ordering, in the builder and in all three formats.

Settled rather than staged, against the finder's alternative of pruning the seeded walk at nodes
`emitted` already holds and rendering them as leaves: that leaves Y a top-level entry and still draws
Y twice, once as its own tree and once as a leaf under the ghost, which README.md:321's rule rules
out, while holding the seed back satisfies the rule as written. Pretty and JSON change shape on this
input class, which is new with this work, so the change is recorded as a dated corrigendum against §8
alongside the other dep-tree output changes.

**Outcome**: Each dependency chain is drawn once. A top-level entry in pretty or JSON is a
participant nothing else in the drawn graph reaches, which is what README.md:321 already promises,
and the toon edge list for a two-edge graph carries two rows. The summary fields and every existing
cycle, dangling-blocker and rooted assertion stay as they are. The dated §8 corrigendum recording
this output change is the orchestrator's to write: this task touches code and tests only, and
nothing under `.workflows/`.

**Do**:
- In `buildSeededTrees` (`internal/cli/dep_tree_graph.go:223-241`), hold back a participant any
  unemitted participant blocks: before seeding `id`, take its blockers — the `BlockedBy` of its
  record in `taskIdx`, none when no record matches it — and skip it while any of them is unemitted.
  Keep the two existing skips (`emitted[id]`, and `len(blocks[id]) == 0` for a participant that
  blocks nothing).
- Wrap the walk over `participants` in repeated passes in the same first-seen order, stopping when a
  pass emits nothing, so a participant freed by an earlier seed in the same run is picked up.
- When a pass emits nothing and a remaining participant still blocks something — every such
  participant is then blocked by another remaining one, a cycle — seed the first of them in
  first-seen order and resume the passes, so the loop always terminates and cycle coverage is what
  it is today.
- Leave the rest of the builder untouched: the root walk at `dep_tree_graph.go:123-144`, the
  `emitted`/`collectTreeIDs` bookkeeping, `depTreeMissingStatus` for a participant no record
  matches, and the `chains`/`longest`/`blocked` computation.
- Add a two-level dangling-chain fixture beside `danglingBlockerTasks`
  (`internal/cli/dep_tree_test.go:223-228`): `tick-aaa111` blocked by `tick-bbb222`, `tick-bbb222`
  blocked by `tick-ghost1`, which no task record matches.

**Acceptance Criteria**:
- [ ] `BuildFullDepTree` on the two-level dangling chain returns exactly one tree: `tick-ghost1`
      (status `missing`, empty title) → `tick-bbb222` → `tick-aaa111`.
- [ ] Pretty draws that chain once, the ghost as the only top-level entry; no participant carrying a
      `BlockedBy` entry is drawn as a top-level entry on that input.
- [ ] JSON returns one object under `trees` for that input, with the `tick-bbb222` subtree appearing
      once.
- [ ] `dep_tree` carries two rows for the two-edge graph, one per stored dependency, neither
      repeated.
- [ ] The fixture's summary reads `1 chain, longest: 2, 2 blocked` in all three formats.
- [ ] Cycle coverage is unchanged: `cycleTasks` still yields the single `tick-aaa111` →
      `tick-bbb222` → `tick-aaa111` tree with `chains: 1`, `longest: 2`, `blocked: 2`
      (`internal/cli/dep_tree_test.go:302`, `:367`, `:395`;
      `internal/cli/dep_tree_graph_test.go:316`, `:647`).
- [ ] The single dangling blocker and the rooted chain are unchanged
      (`internal/cli/dep_tree_test.go:318`, `:385`, `:414`, `:432`, `:451`;
      `internal/cli/dep_tree_graph_test.go:294`), and every participant still reaches every format.
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are
      clean.

**Tests**:
- `"it seeds the dangling blocker rather than the chain beneath it"` — builder: one tree, rooted at
  the ghost, `tick-bbb222` and `tick-aaa111` nested beneath it in that order.
- `"it renders a two-level dangling chain once in the terminal"` — pretty, exact bytes: the ghost
  line, `└── tick-bbb222`, `    └── tick-aaa111`, then `1 chain, longest: 2, 2 blocked`.
- `"it nests a two-level dangling chain under the ghost in JSON"` — one entry under `trees`, the
  `tick-bbb222` subtree present once.
- `"it emits two edges for a two-edge dangling chain"` — toon: two rows, neither repeated.
- `"it still covers a cycle where every participant is blocked"` — the fallback seed keeps
  `cycleTasks` at one tree and its pinned counts.
