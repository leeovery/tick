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
agent's count of what a task blocks agrees with `tick show`.

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
and the toon edge list for a two-edge graph carries two rows.
