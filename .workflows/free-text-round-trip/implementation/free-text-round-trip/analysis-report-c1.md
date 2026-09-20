# Analysis Report: Free Text Round Trip (Cycle 1)

## Stats

- Total findings: 7
- Deduplicated findings: 6
- Proposed tasks: 6

## Summary

The specification conforms closely to the tree — the standards pass checked §5 through §11 against a built binary, including every branch of §9's field selection, §10's marker semantics and §2.2's migrate trim, and the architecture pass found the field-selection module, the end-of-flags split and the conformance inventory composing cleanly. Two seams do not hold: the note index the detail document publishes is computed by a different ordering than the one `note remove` consumes, so on a task whose stored notes are out of chronological order the wrong note is deleted silently; and the widened full-graph dep-tree coverage reaches JSON and pretty as a key that lies about its contents and as a node with blank title and status where only an ID exists. Outside the code, §12.2's amendments to the `tick-core` and `auto-cascade-parent-status` specifications have not been made, leaving both superseded output shapes live in the knowledge base as validated context.

## Spec Defects

### S1: §8 and §4.1 put a non-task participant in the output and never say how it is presented

- **Claim**: §8's corrigendum of 2026-09-19 — "**The full-graph edge list therefore covers every participant, not only those a root reaches**" (§8, specification.md:324) — and §4.1's second corrigendum, which has pretty draw "the participants no root reaches alongside the rooted ones" through the existing tree rendering with no new visual form.
- **Observed**: Both amendments were written for exactly two producing states, one of which is a dangling blocker — an ID no task record matches. Neither decides what a *node* looks like when the participant is not a task. `buildUnrootedTrees` seeds `DepTreeTask{ID: id}` (`internal/cli/dep_tree_graph.go:228`) and the zero value reaches both node-shaped renderings: pretty draws `tick-ghost   ()` (`internal/cli/pretty_formatter.go:422`), JSON returns `{"id":"tick-ghost","title":"","status":""}` (`internal/cli/json_formatter.go:361`). Verified against a built binary. The toon edge list is unaffected — it carries IDs only.
- **Read**: Genuinely open — a gap rather than a stale claim. The specification reaches this graph class twice and decides edge coverage and tree coverage, and stops one step short of the node's contents, so the zero value is standing in for a decision nobody took. The code is wrong either way (an agent reads a task that does not exist), so it is staged as Task 4 with the direction settled against §1 and the ad hoc pass's terminal-coverage direction; the specification wants a corrigendum entry recording the shape whichever way the walk lands it.

## Comment Corrections

- internal/cli/helpers.go:99 — restates the name and signature of the one-line wrapper beneath it, carrying nothing the code cannot (found independently by the duplication and architecture passes)
  OLD: // outputStatusChanges writes a command's status changes to stdout.
  NEW:
- internal/cli/format.go:231 — the comment omits the constraint that makes the field safe: only pretty reads Message, and §8 requires toon and JSON never to
  OLD: 	// Message for edge cases (e.g., "No dependencies found.")
  NEW: 	// Message is the no-dependencies sentence. Only the pretty formatter renders it;
	// the machine formats answer an empty graph with the emptied document instead.

## Discarded Findings

None. All seven findings cleared the floor and none reversed a settled direction; the two `roots`-key findings deduplicated into one proposal, and the one low-severity finding was folded into the Corrections bundle rather than dropped, since its fix is a single deletion.

For the record, the duplication pass names three near-duplicate families it dropped against the floor before writing anything: the per-format detail projection across `toon_formatter.go`/`json_formatter.go`/`pretty_formatter.go` (guarded by `TestRegisteredFieldRendering`, consistent today); the fifteen `selectedItems(detail.X, sel.Positions(fieldX))` pairings (a typo risk rather than a rule that can silently diverge, with no type-concrete consolidation available); and `showListSections`' claim to mirror the toon document's section order (drift is silent but only reorders which section an out-of-range error names).

## Dedup and Grouping

- **`roots` key** — reported by the standards pass (`internal/cli/json_formatter.go:341`, `:388`) and the architecture pass (`:339-345`, `:388`, `internal/cli/format.go:217-239`, `README.md:321`) with the same binary-verified evidence. One proposal, Task 3; the architecture pass's extra step — collapsing the `Roots`/`Unrooted` pair no consumer reads separately — is carried in its Solution.
- Every other finding stands alone. No proposal depends on another: Task 3 and Task 4 both touch the full-graph dep tree but at disjoint sites (the document key and the domain field, versus the node's contents), and either lands alone.

## Ledger Check

Each proposal was tested against the seven settled directions (p1 T1; p2 T1-T2; p3 T1-T3; p4 T1-T5; p5 T1-T2; p6 T1-T2; ad-hoc-1 T1).

- Task 1 (note ordering): `ORDER BY created ASC, rowid ASC` was set by phase task 1-4 (`3376f7a9`), not by any settled proposal. No conflict.
- Task 3 (`trees` key): extends p3 T1's widened edge coverage and ad-hoc-1 T1's terminal coverage rather than reversing either — the root-versus-seeded distinction stays where ad-hoc-1's empty-answer condition consumes it, inside `BuildFullDepTree`. It follows the same precedent p6's corrigendum set when `blocked_by` became `blocker`.
- Task 4 (missing participant): the finder's alternative — keeping node renderings to participants that resolve to a task — would take the blocked real task out of pretty's tree with the ghost that seeds its walk, reversing ad-hoc-1 T1 without grounds. Dropped; the marker branch is what is staged.
- Task 5 (count-zero headers): the five literals predate and post-date several passes but no settled Solution fixes them as literals; p1 T1 and p4 T5 pin README samples of the *populated* forms, which the proposal deliberately leaves standing as the independent pin.
- Task 6 (delete `TestConformanceScopeBoundary`): created by phase task 6-2 (`d184e25f`). p6 T1 and T2 govern the inventory's command matching and its per-format prose exemption and do not touch this guard. No conflict.

## Decisions Staged

None. Three forks were weighed and settled rather than staged: the `roots` key's replacement (settled by this work unit's own `blocker` precedent, and no specification claim names the key); whether to publish the seeded trees under a second key (no consumer spends the distinction, per the architecture pass's read of the three formatters); and the presentation of a participant that names no task (one branch reverses a settled direction, the other makes an agent learn outside the output that an empty `status` means "no such task" — the rule §1 exists to delete). None had mirrored consequences with the tie-break in the user's hands.
