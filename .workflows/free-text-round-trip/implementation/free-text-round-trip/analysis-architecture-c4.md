AGENT: architecture
FINDINGS:
- FINDING: FieldSelection's nil contract is split across its own methods, and the guard that makes it safe lives in a caller
  SEVERITY: low
  FAILURE: `*FieldSelection` documents nil as "the whole document" and three of its five methods honour that — `includes`, `Positions` and `ValidatePositions` all return sensible values on a nil receiver. `Len` and `Only` dereference the receiver and panic. Verified against HEAD: calling `sel.Only()` or `sel.Len()` on a nil `*FieldSelection` panics with `runtime error: invalid memory address or nil pointer dereference`. `detail.Fields` is nil on every document `create`, `update`, `note add`, `note remove` and an unfiltered `show` produce, and all three formatters hold that pointer. The first production code that consults the selection's size — a formatter asking "was more than one field asked for?", the natural question given `Only` exists — crashes the CLI with a Go stack trace on the commands an agent runs most, instead of printing the task. Nothing in the type stops it: the only nil guard is `bareFieldValue`'s `if sel == nil` at show_fields.go:119, in a caller, and `Len` has no production caller at all today, so the split has been carried this far untested from the production side.
  FILES: internal/cli/show_fields.go:165-192, internal/cli/show_fields.go:119, internal/cli/format.go:236
  DESCRIPTION: The nil-means-whole-document convention is the whole reason `TaskDetail.Fields` is a pointer rather than a value — it lets `outputMutationResult` leave the field zero and lets each formatter call `sel.includes(...)` unguarded, which is why the toon, JSON and pretty detail builders read as cleanly as they do. That convention is a property of the type and every method has to hold it for the calling style the rest of the package adopted to be safe. Two do not, and the compensating check was written in `bareFieldValue` instead of in `Only` — correctness moved out of the type and into caller discipline, which is exactly the arrangement that holds until the next caller. `Len` compounds it: it is exported, reached only from tests, and is the method whose first production use would land on the panicking path.
  RECOMMENDATION: Give `Len` and `Only` the same nil handling the other three have — `Len` returning 0 and `Only` returning `("", false)` for a nil receiver — and drop the `if sel == nil` guard from `bareFieldValue`, which the type then covers. If `Len` is wanted only by the tests, delete it instead and leave `Only` as the one size question the type answers; either way the type stops having two contracts. Consider unexporting `FieldSelection`'s methods to match `includes`, since nothing outside package `cli` reaches the type and the exported/unexported split currently tracks nothing.

COMMENT_CORRECTIONS:
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

SUMMARY: The pieces this work built compose well — the field-selection registry is held to the formatters by a rendering guard, the end-of-flags split reaches every handler through one signature, the toon refusal path carries the encoder's error out of every document builder rather than substituting one, and the dep-tree edge list is read off the stored relation so counts and edges agree on the cycle, diamond and dangling-blocker graphs I reproduced against a built binary. One seam is soft: `FieldSelection` answers a nil receiver two different ways depending on the method, with the compensating guard in a caller rather than in the type.
