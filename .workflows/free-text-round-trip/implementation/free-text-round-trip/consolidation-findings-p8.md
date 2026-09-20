# Consolidation Findings: free-text-round-trip (Phase 8)

## Findings

### F1: the focused toon edge list still repeats a shared branch the full-graph list no longer repeats

- **Class**: behaviour
- **Failure**: `tick dep tree --toon` and `tick dep tree <id> --toon` now emit the same `from,to` row vocabulary under two different rules, with nothing in either document marking which rule produced it. An agent (or a reader of README.md:321, which this phase wrote as an unscoped promise — "the toon edge list carries one row per stored dependency") that counts rows, or treats a row as a stored fact, over-counts on the focused document: every dependency below a point two paths converge on comes back once per converging path. Noticed as a literally repeated `from,to` row in the `blocks` or `blocked_by` section, and as a row count larger than the number of dependencies `tick dep add` ever recorded.
- **Evidence**:
  - `internal/cli/toon_formatter.go:183-188` — `formatFullDepTree` now converts `result.Edges` (the stored relation) row for row.
  - `internal/cli/dep_tree_graph.go:187-195` — `collectStoredEdges`, one row per `BlockedBy` entry, cannot repeat.
  - `internal/cli/toon_formatter.go:203-215` — `formatFocusedDepTree` still builds both sections from the walk, via `collectUpstreamEdges` / `collectDownstreamEdges` (`internal/cli/toon_formatter.go:217-236`), over trees `BuildFocusedDepTree` (`internal/cli/dep_tree_graph.go:329-354`) grows with `walkDownstream` / `walkUpstream`, whose own doc records the consequence: "No deduplication — diamond dependencies are duplicated" (`internal/cli/dep_tree_graph.go:41`, `:72`).
  - Reproducer: A unblocked, B blocked by A, C blocked by A, D blocked by B and C, E blocked by D — five stored dependencies. `tick dep tree --toon` returns five rows. `tick dep tree A --toon` returns six, with `tick-D,tick-E` twice (walk order A→B, B→D, D→E, A→C, C→D, D→E).
  - The phase caused the divergence, not the repetition: before `Tfree-text-round-trip-8-1` both modes derived rows from a walk and agreed. Spec §8 legislates the full-graph list only; the argument it makes there — "a walk cannot represent a graph without repeating the shared part below every point two paths converge on" — is untouched by mode.
- **Proposed shape**: either (a) build both focused sections from the stored relation restricted to the IDs the corresponding walk reaches — one row per stored dependency inside the focused neighbourhood — which retires `collectUpstreamEdges` and `collectDownstreamEdges` along with their last callers; or (b) hold focused mode walk-shaped by decision and scope the README sentence to the full graph ("…the full-graph toon edge list carries one row per stored dependency"). (a) changes shipped output and wants the same spec treatment §8 gave the full graph; (b) is a documentation edit to the last sentence of README.md:321.

## Comment Corrections

- README.md:321 — describes the superseded seeding rule: every participant no root reaches is no longer drawn as a top-level entry, only those nothing already drawn reaches (a two-task chain under a dangling blocker now draws one entry, not three — `internal/cli/dep_tree_test.go`, "it renders a two-level dangling chain once in the terminal").
  OLD: then the tasks no root reaches — a cycle's members, or a task blocked by an ID that no longer exists
  NEW: then seeds a tree from each participant nothing already drawn reaches — a cycle's member, or a task blocked by an ID that no longer exists — with its downstream chain nested beneath it

- internal/cli/format.go:219-221 — the type doc restates the producer's seeding rule, which `Tfree-text-round-trip-8-2` changed: participants an earlier *seeded* tree reaches are skipped too, and a participant is held back while any of its own blockers is undrawn. Nothing binds this doc to `buildSeededTrees`, so it goes stale again on the next change to it.
  OLD: // DepTreeResult holds all data needed to render a dep tree command output.
  // For full graph mode: Trees holds every tree covering the graph — those grown from
  // unblocked tasks first, then those seeded from participants no such tree reaches — Edges
  // holds one entry per stored dependency, and summary stats are populated.
  // For focused mode: BlockedBy and Blocks contain upstream/downstream trees.
  NEW: // DepTreeResult holds all data needed to render a dep tree command output.
  // For full graph mode: Trees holds the drawn trees, Edges holds one entry per stored
  // dependency, and summary stats are populated.
  // For focused mode: BlockedBy and Blocks contain upstream/downstream trees.

- internal/cli/dep_tree_graph.go:233-238 — the stated purpose is false since the sibling task landed: seeding no longer feeds the edge list at all (that comes from `collectStoredEdges`, whatever the walk does), and "the target of a blocker's edge" reads as an edge-list row in a file where `DepTreeEdge` now means exactly that. What a seed buys is a drawing in the node-shaped renderings.
  OLD: // buildSeededTrees seeds a downstream walk from each participant the walk from the roots
  // left unemitted, so the edges of a cycle or of a dangling blocker still reach the output.
  // Participants that block nothing need no seed: each is reached as the target of a blocker's edge.
  // A participant whose blockers are not yet emitted is held back until they are; when every remaining
  // participant is blocked by another that is also unemitted, the first in order is seeded so its
  // edges still reach the output.
  NEW: // buildSeededTrees seeds a downstream walk from each participant the walk from the roots
  // left unemitted, so a cycle's members and a dangling blocker are still drawn.
  // Participants that block nothing need no seed: each is drawn as a child of one of its blockers.
  // A participant whose blockers are not yet emitted is held back until they are; when every remaining
  // participant is blocked by another that is also unemitted, the first in order is seeded so it is
  // drawn at all.

- internal/cli/toon_formatter.go:228 — a claim about where the helper is called, falsified by any later caller and checked by nothing; this phase deleted the twin claim from `collectDownstreamEdges` and left this one.
  OLD: // Used for the "blocked_by" direction of focused mode.
  NEW:

- internal/cli/dep_tree_graph.go:132 — restates the two guards directly above it (pre-existing, inside the function this phase edited).
  OLD: // This task participates and is not blocked — it's a root
  NEW:

- internal/cli/dep_tree_graph.go:146 — restates the loop below it (pre-existing, inside the function this phase edited).
  OLD: // Count blocked tasks (tasks with at least one BlockedBy entry)
  NEW:

- internal/cli/dep_tree_graph.go:162 — restates the code below it (pre-existing, inside the function this phase edited).
  OLD: // Build summary
  NEW:

## Spec Defects

### S1: §8 states the hold-back invariant unqualified in the body, and records its exception only in a corrigendum

- **Claim**: §8, "The node-shaped renderings still walk": "any participant not yet drawn seeds a further walk, held back while any of its own blockers is still undrawn **so that each chain is drawn once and a top-level entry is a participant nothing else drawn reaches**. Where every remaining participant is blocked by another that is also undrawn — a cycle — the first in order is seeded…"
- **Observed**: the fallback the same sentence-pair prescribes cannot preserve the invariant the first sentence states, and the landed code — `internal/cli/dep_tree_graph.go:266-270`, `slices.IndexFunc(participants, needsSeed)` — implements the fallback verbatim. On a cycle carrying dependents, the seeded participant can be one a later seed reaches, so a subtree is drawn twice and a blocked task appears as a top-level entry. The corrigendum of 2026-09-20 records exactly this ("Known limit, recorded rather than fixed…"), so the document as a whole is correct; the body paragraph read alone is not.
- **Read**: spec stale — the limit belongs in the body beside the invariant it qualifies, otherwise the next reader implementing from §8 takes "a top-level entry is a participant nothing else drawn reaches" as a guarantee to hold and re-opens a decision already made. Editorial only: no code or behaviour follows from it.
