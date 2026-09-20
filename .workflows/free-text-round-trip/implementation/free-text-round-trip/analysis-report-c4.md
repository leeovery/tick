# Analysis Report: free-text-round-trip (Cycle 4)

## Stats

- Total findings: 5 (duplication 2, standards 2, architecture 1) plus 4 raw comment corrections
- Deduplicated findings: 5 findings, 3 comment corrections
- Proposed tasks: 5

## Summary

The shipped behaviour conforms to §2 through §12 on every branch the standards pass re-checked against a built binary, with one measured exception: phase 9's row-identity mechanism was wired to the task list alone, so a refused value carried by a child or a cascaded task names the document's subject — a task that encodes cleanly — rather than the task that carries it. The remaining four findings are structural and behaviour-preserving: the field selection's narrowing of the five list sections is restated fifteen times across the three formatters with nothing holding the name-to-collection pairing, the stored-dependency edge list is enumerated once for the full graph and again for the focused view, `FieldSelection` answers a nil receiver two different ways with the compensating guard in a caller, and the README's refusal paragraph still describes the pre-9-2 diagnostic. No Decisions are staged: the one fork in the cycle — fix the code or amend §3.1 — is broken by the specification, which is the later and governing statement.

## Notes On The Ledger

- **Task 2 and phase 9's task 1, step 4.** That step, and the acceptance criterion and subtest that pin it (`internal/cli/toon_refusal_test.go:231`), direct the cascade diagnostic to name the parent rather than the cascaded child. §3.1 was amended after that direction was set and requires the diagnostic to name "the task that carries the refused value", with the section name alone standing only "where the offending row cannot be identified" — a condition no row of `blocked_by`, `children` or `changed` meets, every one carrying a task ID. The finding is measured against a binary built from the working tree. Treated as a superseded test-level direction rather than a settled direction reversed; Task 2's Solution names the test it rewrites and the criterion it supersedes.
- **Task 1** extends phase 4's tasks 3 and 4 (one registry declaration; formatter gates off the registry's names) to the one part of that vocabulary still restated per formatter. No reversal.
- **Task 3** completes phase 8's task 1 and cycle 2's task 1 (both dep-tree views derive rows from the stored relation) by making the two forms agree by construction rather than by parallel maintenance. No reversal.
- **Task 4** extends phase 4's task 3, which settled one nil-safe gate on this type, over the two methods it did not reach. No reversal.

## Comment Corrections

- internal/task/transition.go:5-6 — the worked example prints the arrow line §7.2 replaced with the `changed` table in toon and JSON, so it names output the tool no longer produces (reported by both the duplication and standards passes)
  OLD: // TransitionResult holds the old and new status after a successful transition,
  // enabling the caller to format output like "tick-a3f2b7: open -> in_progress".
  NEW: // TransitionResult holds the old and new status after a successful transition.

- internal/cli/format.go:245 — the interface's error contract, the invariant the encoder-refusal work exists to preserve, is stated on one of its five error-returning methods instead of on the interface every implementation reads
  OLD: // Formatter defines the interface for rendering CLI output in different formats.
  NEW: // Formatter defines the interface for rendering CLI output in different formats.
  // A method returning an error returns one when the format cannot carry a value it
  // was handed; it never substitutes a document that contradicts the data — a dropped
  // section, or a count that disagrees with the rows behind it.

- internal/cli/format.go:260 — repeats on one method the contract that now sits on the interface
  OLD: 	// FormatCascadeTransition renders the status changes a command made. It
	// returns an error when the format cannot carry a value it was handed,
	// rather than a document that contradicts the data.
  NEW: 	// FormatCascadeTransition renders the status changes a command made.

## Discarded Findings

- None discarded on the floor. Two low-severity findings were kept rather than filtered:
  - *The stored-dependency edge list is enumerated twice* (low) — kept because the divergence it guards is measured rather than hypothetical: §8's corrigendum records this rule diverging once already, five stored dependencies returning five rows from one view and six from the other.
  - *FieldSelection's nil contract is split across its own methods* (low) — kept because it clusters with the cycle's medium duplication finding into one pattern: the selection's semantics are enforced by caller discipline (a guard in `bareFieldValue`, fifteen narrowing expressions in the formatters) rather than by the type that owns them.
- The architecture pass's secondary recommendation — unexporting `FieldSelection`'s methods, since nothing outside package `cli` reaches the type — is not carried into Task 4: visibility and naming are below the finding floor.
- The standards pass's stated alternative to Task 2 — keeping the shipped behaviour and writing a §3.1 corrigendum that bounds task-naming to a task list — is not staged as a Decision: §3.1 as amended breaks the tie, so there is no irreducible fork for the user to settle.
