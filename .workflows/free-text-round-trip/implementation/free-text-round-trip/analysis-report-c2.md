# Analysis Report: Free Text Round Trip (Cycle 2)

## Stats

- Total findings: 3 (standards 2, architecture 1, duplication 0) plus 6 comment-correction entries
- Deduplicated findings: 2 actionable (5 unique comment corrections, already applied)
- Proposed tasks: 2

## Summary

The duplication pass returned nothing that clears the floor, and both remaining findings land on
one surface: `tick dep tree`'s full graph publishes the same dependency more than once. The toon
edge list is flattened out of a tree walk rather than read off the `BlockedBy` relation, so a
convergent path emits the shared subtree once per path into it; and `buildSeededTrees` seeds a
participant that a later seed would have reached, so a dangling chain is drawn twice in pretty and
JSON and its edges repeat in toon. Both were reproduced by reading the builder and the formatter,
and both are measurable against a binary built from HEAD. The standards pass's first finding (the
`auto-cascade-parent-status` amendment) was closed as a session step before this synthesis and is
not staged.

## Comment Corrections

**No action pending — all five were applied in commit 53a007b2 ("impl(free-text-round-trip):
analysis cycle 2 — comment corrections") and verified present in the working tree.** Recorded here
verbatim for the cycle's record; re-applying them would fail, since none of the OLD texts is still
in the tree.

- internal/cli/flags.go:11 — describes only the separated spelling, false since a value-taking flag now also carries its value attached as "--flag=value"; the field name carries what survives
  OLD: // TakesValue indicates whether the flag consumes the next argument as its value.
  NEW:
- internal/cli/format.go:258 — restates the line above it ("for text-based formatters (Toon and Pretty)") and makes a cardinality claim about embedders that ordinary additive change falsifies (found independently by the duplication and architecture passes)
  OLD: // Embedded by ToonFormatter and PrettyFormatter.
  NEW:
- internal/migrate/migrate.go:28 — "normalized" is false now that MigratedTask carries a Normalize method the Provider does not call
  OLD: // MigratedTask represents a normalized task ready for insertion into tick.
  NEW: // MigratedTask represents a task ready for insertion into tick.
- internal/migrate/migrate.go:68 — tells a Provider implementer the tasks it returns are normalized, which is now the opposite of true: Engine.Run normalizes what the Provider hands it
  OLD: 	// Tasks returns all normalized tasks from the source, or an error if the source cannot be read.
  NEW: 	// Tasks returns every task from the source, or an error if the source cannot be read.
- internal/migrate/engine.go:54 — enumerates Run's steps and omits the one that changes what gets stored, so the import path's trim contract is written down nowhere
  OLD: // Run fetches tasks from the provider, validates each one, inserts valid tasks
  NEW: // Run fetches tasks from the provider, trims edge whitespace off each one's free
  NEW: // text, validates each one, inserts valid tasks

## Discarded Findings

- The §12.2 amendment to `auto-cascade-parent-status` documents a row the `changed` table cannot carry (standards, high) — closed as a session step before this synthesis, not staged per the orchestrator's instruction. Verified: commit bf39ebf6 re-ran the corrigendum route, the toon section now shows `changed[5]` with no `from == to` row, and the unchanged-children requirement is back to the standing unimplemented requirement §7.6 leaves undecided. No code change was ever at stake — §7.2's behaviour ships correct.
- The duplication pass's seven examined candidates — the three formatters' parallel field projection, the four `build*Section` shapes, the `StatusChange` struct conversions, the note index per machine format, the trim-before-store call sites, the cascade from/to prediction, and the toon detail document's ungated changed section — all dropped by the pass itself against the floor (loud failure on divergence, or no nameable failure). Re-applied the floor and agree: none names a failure it prevents. The ungated changed section is a real divergence but unreachable today, since only `RunShow` sets Fields and only `outputMutationResult` sets Changes.
- No spec defect against this unit's `specification.md`: §8 is silent on whether the full-graph edge list carries one row per stored dependency or one per path. That is a gap the proposals settle, not a wrong claim, so it is routed as a corrigendum obligation inside Task 1 rather than reported here.
