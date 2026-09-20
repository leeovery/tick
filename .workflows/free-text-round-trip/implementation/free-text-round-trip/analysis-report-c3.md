# Analysis Report: Free Text Round Trip (Cycle 3)

## Stats

- Total findings: 1
- Deduplicated findings: 1
- Proposed tasks: 1

## Summary

The cycle produced one finding, from the architecture pass, and it is high: the toon encoding seam this work built answers an encoder refusal by emitting a well-formed document that denies the data — a dropped top-level header block, a count-zero section over rows that exist, an empty task list — silently and with exit 0. The orchestrator reproduced it independently against a binary built from the working tree and it is worse than reported: one task whose title carries an escape character makes `tick list` print `tasks[0]:` for the whole project while `--json` returns both tasks, and `tick show` on that task prints a detail document with no `id`, `title` or `status`. The duplication and standards passes found nothing that clears the floor — duplication examined seven candidate families and dropped each as cross-pinned by a test that fails loudly on drift, and standards re-verified §2 through §12 including the three phase-8 changes no earlier cycle saw — leaving six comment corrections between them.

## Notes on scope

Rejecting or normalising C0 control characters at the write path (`create`, `update`, `note add`, `tick migrate`) is deliberately not proposed. It is additive behaviour the specification does not carry, and it would not cover values already stored or arriving through import — those still have to read back honestly, which is what the proposed task fixes. The architecture finding says the same: a separate change that does not remove this one.

## Comment Corrections

Five distinct targets, six entries — the architecture and duplication passes both corrected `internal/cli/format.go:240`, with different replacement text. Both are recorded verbatim below; they edit the same two lines (240-241), so one must be chosen. The architecture version keeps the machine-format clause and drops only the cardinality claim; the duplication version drops both sentences' tails.

- internal/cli/format.go:240 — names the one formatter that reads the field, a cardinality claim ordinary additive change falsifies; the clause after it carries the part the code cannot *(architecture pass)*
  OLD: 	// Message is the no-dependencies sentence. Only the pretty formatter renders it;
	// the machine formats answer an empty graph with the emptied document instead.
  NEW: 	// Message is the no-dependencies sentence; the machine formats answer an empty
	// graph with the emptied document instead.

- internal/cli/format.go:240 — makes a cardinality claim about which formatter reads the field, which ordinary additive change falsifies far from the comment *(duplication pass)*
  OLD: 	// Message is the no-dependencies sentence. Only the pretty formatter renders it;
  OLD: 	// the machine formats answer an empty graph with the emptied document instead.
  NEW: 	// Message is the no-dependencies sentence.

- internal/cli/toon_formatter.go:196 — states the focused document's shape a second time, three lines under the FormatDepTree doc that already states it, so an edit to one leaves the other stale
  OLD: // formatFocusedDepTree renders focused mode as the target's id, title and status
  OLD: // followed by its blocked_by and blocks sections.
  NEW:

- internal/cli/json_formatter.go:384 — restates the function name, and the FormatDepTree doc above already states the full-graph shape
  OLD: // formatFullDepTreeJSON renders the full graph as nested JSON.
  NEW:

- internal/cli/json_formatter.go:395 — restates the function name, and the FormatDepTree doc above already states the focused shape
  OLD: // formatFocusedDepTreeJSON renders focused mode as JSON.
  NEW:

- internal/cli/dep_tree_graph.go:236 — restates the name and the four-line body beside it, carrying nothing the code cannot
  OLD: // collectTreeIDs adds the ID of every node in the given trees to seen.
  NEW:

## Discarded Findings

None. The cycle's single finding clears the floor — it names the failure (an agent acting on data the output says is absent), who bears it (any agent reading toon, the non-TTY default), and how it would be noticed (never as an error; only by contradiction against `--json` or the JSONL file) — and reverses no settled direction. The phase-3 consolidation task that routed the two dep-tree documents through `joinToonSections` noted the same `""`-on-encoder-error return as its mechanism; carrying the error out extends that direction rather than reversing it, and the reproduction is measured grounds in any case.
