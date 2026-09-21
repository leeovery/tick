TASK: free-text-round-trip-4-6 — Positions Narrow The List Section They Name

ACCEPTANCE CRITERIA:
- `--field notes.2` inside a multi-field selection renders `notes[1]{index,text,created}:` with one row whose `index` decodes as `2`
- The narrowed section's count follows the selection while each note row carries its real position
- `--field title,tags.2` renders `tags[1]: <second tag>` — an inline list keeping its count and one item
- `--field children.1` renders the `children` table header and one row
- `--field notes.1,notes.3` renders two rows with indexes `1` and `3`, and `--field notes.3,notes.1` renders the same two rows in the same order
- `--field title,notes.2,notes.2` renders one row, and `--field notes.2,notes.2` alone collapses to one position and prints the note's text bare
- `--field notes,notes.2` renders the whole notes section
- A position narrows only the section it names — every other selected field comes back whole
- `--field notes.2` alone prints the note's text bare with one terminating newline
- `--field tags.1` alone prints the tag bare; `--field children.1` alone returns its one-row section, not a bare value
- Narrowing applies identically in toon, JSON and pretty
- An unselected or unfiltered document renders every item, unchanged from before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§9.3 requires a positional suffix on any list section (`notes.2`, `tags.1`, `refs.2`, `children.3`, `blocked_by.1`), 1-based, with the section rendered in its normal form narrowed to the named items — a table keeping its header and its row(s), an inline list keeping its count and its item — and only notes carrying a real position in the row, via the `index` column §6.3 added so an agent that asked for the third note cannot mistake it for the first and delete the wrong one. §9.3 also fixes that a position narrows the section it names and nothing else, and that `--field notes.2` alone prints the note's text bare by the one-field rule. §9.1 fixes that a section named both whole and by position comes back whole, that several positions render in output order rather than typed order, and that a repeated name counts once toward the one-versus-several split. §9.2 fixes that a selection resolving to a row rather than a value (`children.1`) comes back as its one-row section. §9.7 keeps a bare value out of the format flags' reach.

IMPLEMENTATION:
- Status: Implemented (sound divergence from the plan's step 2-4 shape, see Notes)
- Location:
  - `internal/cli/show_fields.go:76` — `selectedItems[T any](items []T, positions []int) ([]T, []int)`: nil positions return every item paired with `1..len(items)`; otherwise a sorted (`slices.Sorted(slices.Values(...))`, a copy, so the selection's own slice is not mutated) and `slices.Compact`-ed position list, out-of-range positions skipped, items paired with their real 1-based positions.
  - `internal/cli/format.go:114-140` — `selectedSections` struct and `TaskDetail.selectedSections()`, narrowing `BlockedBy`, `Children`, `Tags`, `Refs` and `Notes` through `selectedItems` off `d.Fields.Positions(...)`, keeping the note positions.
  - `internal/cli/toon_formatter.go:73-96` — the detail document builds every list section from `sections.*`; `buildNotesSection(sections.Notes, sections.NotePositions)` at line 95, and `internal/cli/toon_formatter.go:317` sets each row's `Index` from `positions[i]` rather than the loop counter.
  - `internal/cli/json_formatter.go:107,121-134` — the filtered object takes `tags`, `refs`, `notes`, `blocked_by` and `children` from `sections.*`; `toJSONNotes` (`internal/cli/json_formatter.go:159-169`) takes each object's `index` from the paired position.
  - `internal/cli/pretty_formatter.go:143,159` (Tags) and `:184-205` (Blocked by, Children, Refs, Notes) — narrowed through the same sections, positions discarded since pretty renders no index.
  - `internal/cli/show_fields.go:118-145` — `bareFieldValue` delegates to `barePositionValue`, which resolves bare only for a section carrying an `items` accessor (`notes`, `tags`, `refs` — registered at `internal/cli/show_fields.go:63-65`) with exactly one position after collapse; `children` and `blocked_by` register no `items` and so fall through to the document path.
- Notes:
  - The plan's steps 2-4 asked each formatter to thread `detail.Fields.Positions(name)` into its own sections. The delivered code instead narrows once in `TaskDetail.selectedSections()` and lets all three formatters read it. Same observable behaviour, one narrowing rule instead of three copies — better than what was written, and no loss.
  - Positional gating still reads the unnarrowed slice (`len(detail.Tags) > 0`, `len(detail.Notes) > 0`, toon lines 86/90, pretty lines 187/191/195/199), so a narrowing that survived to formatting can never produce an empty section header: `ValidatePositions` (`internal/cli/show_fields.go:284`) rejects an out-of-range position in `RunShow` (`internal/cli/show.go:67`) before any formatter runs, which is the task's own stated edge case.
  - `buildNotesSection` and `toJSONNotes` index `positions[i]`; their only production callers are `internal/cli/toon_formatter.go:95` and `internal/cli/json_formatter.go:123`, both passing the pair `selectedItems` returned, so the lengths always match. The only other caller is `internal/cli/toon_formatter_test.go:883`, passing `(nil, nil)` into the count-zero early return.
  - `detail.Fields` is assigned in exactly one production site, `internal/cli/show.go:83`, so narrowing cannot leak into the detail document `create`, `update` and `note add/remove` emit.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/show_fields_test.go:545-604` (`TestSelectedItems`) — the helper directly: every item with positions for nil, a named item with its position, ascending output order for `{3,1}`, a repeated position collapsed, out-of-range (`0`, `4`, `-1`) skipped, empty section.
  - `internal/cli/list_show_test.go:1085-1301` (`TestShowFieldPositions`) — end-to-end through `App.Run`: one narrowed notes row (1144), its `index` decoding as `2` (1153), inline `tags`/`refs` narrowed to one item (1157, 1161), `children`/`blocked_by` tables narrowed to one row (1165, 1169), `notes.1,notes.3` and `notes.3,notes.1` both decoding to indexes 1 then 3 (1173), repeated position collapsing in the document and to a bare value (1179), `notes,notes.2` whole (1188), `notes.2,tags` leaving tags whole (1192), bare note/tag/ref for a lone position (1199, 1207, 1215), `children.1` returning its one-row section rather than a bare value (1223), JSON one note object with `index` 2 (1232), pretty notes/tags/children/blocked_by narrowing (1256-1290), and an unfiltered document rendering every item with positions 1..3 (1292).
  - `internal/cli/show_fields_test.go:485-530` — `bareFieldValue` for `notes.2`, `tags.2`, `refs.1`; bare for a repeated position; not bare for two distinct positions, for `children.1`/`blocked_by.1`, or for a position outside the section.
- Notes:
  - The toon assertions go through `decodeToonDoc` (`internal/cli/toon_decode_test.go:17`), which fails the test on a decode error, so a section header whose count disagreed with its rows would not pass silently; `assertNoteRows` (`internal/cli/list_show_test.go:1129`) then asserts the row's `index` against the expected real position, which is the assertion that would fail if the row were renumbered from its slice index.
  - Not over-tested: the extra subtests beyond the plan's list (`refs.2`, `blocked_by.2`, pretty tags/children/blocked_by) each cover a distinct narrowing call site rather than repeating one, and the criterion asks for narrowing to hold identically across three formats and five sections.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` "it does X" subtests and `t.Helper()` on helpers; `runShow` sets `IsTTY: true` so unflagged cases resolve to pretty, and the toon cases pass `--toon` explicitly; formatter/field-registry patterns from CLAUDE.md are kept.
- SOLID principles: Good — narrowing is one responsibility in one place (`selectedSections`), the formatters consume it; adding a list section means registering it in `showFields` and reading it from `sections`.
- Complexity: Low — `selectedItems` is a single guarded loop; the formatters gained no branches.
- Modern idioms: Yes — generics for the shared helper, `slices.Sorted`/`slices.Values`/`slices.Compact`, capacity-preallocated slices.
- Readability: Good — doc comments on `selectedItems`, `selectedSections`, `buildNotesSection`, `toJSONNotes` and `barePositionValue` each state the rule they implement and hold true against the code; no process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands at the delivered tree; reading cannot establish a clean suite, vet or gofmt result.
