## Attempt 1

ISSUES:
- `internal/cli/pretty_formatter.go:184,188` — the `blocked_by` and `children` gates are the only two of the five new block gates with no test that asserts the block *renders* when selected. `TestPrettyFilteredTaskDetail` covers `notes` and `description`; `refs` and `tags` are covered end-to-end by the rewritten `TestShowBareField` table (`internal/cli/list_show_test.go:864-877`); `children` and `blocked_by` were the two entries that table dropped, and nothing replaced them. Today's behaviour is correct (the reviewer ran both selections against `richDetail()`: `"Children:\n  tick-e5f6  Child (open)"` and `"Blocked by:\n  tick-c3d4  Blocker (open)"`), so this is a regression hole rather than a live defect: a mistyped or removed gate name would leave `tick show <id> --field children` printing nothing on a terminal — the default format — while `--toon` prints the section, and the suite would stay green.
  FIX: add two subtests to `TestPrettyFilteredTaskDetail` in `internal/cli/pretty_formatter_test.go`, alongside `"it renders only the selected block"`, using the existing `filtered` helper and `richDetail()`: `"it renders only the selected children block"` asserting `"Children:\n  tick-e5f6  Child (open)"`, and `"it renders only the selected blocked_by block"` asserting `"Blocked by:\n  tick-c3d4  Blocker (open)"`. Both strings are exact against the current fixture.
  ALTERNATIVE: restore `children` and `blocked_by` to the `TestShowBareField` table instead, which needs a second project carrying a parent/child pair and a dependency — more setup for the same guarantee at a lower level of the stack. The formatter-level pair is cheaper and sits beside the tests it belongs with; the reviewer recommends it.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- `internal/cli/pretty_formatter.go:211-214` — the rewritten doc names three of the five omit-when-empty sections; `Refs` and `Notes` are omitted the same way, so the parenthetical reads as an exhaustive set and is not one.
  OLD: // FormatTaskDetail renders a single task in key-value format, narrowed to
// detail.Fields when it is set. Sections (Blocked by, Children, Description)
// are omitted when empty, and the whole document is empty when the selection
// keeps nothing.
  NEW: // FormatTaskDetail renders a single task in key-value format, narrowed to
// detail.Fields when it is set. A section the task has nothing for is omitted,
// and the whole document is empty when the selection keeps nothing.
- `internal/cli/pretty_formatter.go:122` — restates the five lines beneath it; the label-and-entries shape is in the signature and the two-space indent is in the one `Fprintf` format string.
  OLD: // prettyDetailBlock renders a labelled block whose entries are indented two spaces.
  NEW:

NOTES:
- Byte-identity was verified independently rather than accepted: the reviewer copied the tree to a scratch directory, re-added the pre-change `FormatTaskDetail` body verbatim from `HEAD`, and compared old against new across 9216 permutations (tags 0/1/2, parent absent/untitled/titled, type empty/set, closed nil/set, blocked_by, children, refs, notes, five description shapes including trailing-newline and leading-space, and four `Changes` shapes including `nil` and `Blocks: nil`). Zero mismatches.
- The `--field type` acceptance criterion is unreachable as literally worded — at the CLI that selection takes task 4-2's bare path. The reviewer agrees with the executor's reading and states no production change should be made to chase the literal wording.
- `prettyDetailHeader(detail, sel)` and `prettyDetailBlocks(detail, sel)` take the selection as a second parameter while `sel` is always `detail.Fields`. Harmless today, but the two can be passed apart; folding the read inside the helpers would remove the possibility. Not worth a round on its own.
- `internal/cli/pretty_formatter.go:144-151` uses positional composite literals for the function-local `candidate` type, where `.claude/skills/golang-code-style` states field names MUST be used. Not raised: all three fields are strings and the golden tests would fail loudly on any reorder, so no silent divergence is possible. If ever revisited, the sibling `buildTaskSection` (`internal/cli/toon_formatter.go:245-251`) shows the cheaper shape — a local `add(name, label, value)` closure appending to `lines`, which removes the intermediate type entirely.
- Forward note for task 4-6: its step 4 lists `Tags` among the pretty renderings to narrow through `selectedItems`, but pretty renders tags as a header line, so that narrowing lands in `prettyDetailHeader`'s candidate list rather than in `prettyDetailBlocks`.
- `--field tags` on a tag-carrying task is now asserted in two places with the same fixture shape and the same expected string (`internal/cli/list_show_test.go:872` and `:961`). Duplicate rather than wrong; no action.
