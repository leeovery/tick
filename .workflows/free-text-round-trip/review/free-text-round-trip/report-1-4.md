TASK: free-text-round-trip-1-4 (tick-df8092) — Notes Carry A 1-Based Index Column

ACCEPTANCE CRITERIA:
- The toon notes header is `notes[N]{index,text,created}:` and each decoded row's `index` is its 1-based position, in order
- The empty toon notes section is `notes[0]{index,text,created}:` and decodes to an empty list
- Each JSON note object carries `index` with the same 1-based value
- `tick note remove <id> <index>` applied to the index shown for a note removes that note and no other, including on a task whose notes share a created timestamp
- Multi-line note text stays library-quoted and decodes byte-identical
- Pretty's notes block is unchanged — no index is rendered there
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §6.3 requires the notes section to gain a leading `index` column carrying each note's 1-based position, present whether the section is filtered (§9.3) or not — position is the only handle a note has, since there is no note ID and `note remove` takes a 1-based index. §4.2 carries the same index into JSON, for the same reason. §6.4 fixes that no `note edit` command is added. §4.1 keeps pretty untouched, making the index a machine-format column only. §9.3 adds the later requirement that a position-narrowed notes section keeps each row's real whole-section position while the count follows the selection.

IMPLEMENTATION:
- Status: Implemented (with one sound divergence from the plan's step 4 — see Notes)
- Location:
  - `internal/cli/toon_formatter.go:40-44` — `toonNoteRow` carries `Index int` with `toon:"index"` as its first field
  - `internal/cli/toon_formatter.go:310-323` — `buildNotesSection` sets each row's `Index` from the position slice
  - `internal/cli/toon_formatter.go:327-334` — `emptyToonSection[toonNoteRow]("notes")` derives `notes[0]{index,text,created}:` by reflecting the row struct's toon tags, so the count-zero header cannot drift from the populated one
  - `internal/cli/json_formatter.go:50-54` — `jsonNote` carries `Index int` with `json:"index"` first
  - `internal/cli/json_formatter.go:159-169` — `toJSONNotes` sets `Index` from the same position slice
  - `internal/cli/format.go:111-140` — `selectedSections` carries `NotePositions` alongside the narrowed notes
  - `internal/cli/show_fields.go:76-98` — `selectedItems` returns items and their whole-section positions together, so the two slices are always the same length and `positions[i]` cannot go out of range
  - `internal/cli/show.go:173` — notes query orders `ORDER BY rowid ASC`
  - `internal/cli/pretty_formatter.go:199-205` — pretty's notes block renders `created  text` only, no index
- Notes: The plan's step 4 asked for `ORDER BY created ASC, rowid ASC`; the delivered query is `ORDER BY rowid ASC` (later refined under free-text-round-trip-7-1). This is an improvement, not a loss. `note remove` indexes the JSONL `Notes` slice directly (`internal/cli/note.go:130-134`), notes enter `task_notes` in JSONL order during a full cache rebuild (`internal/storage/cache.go:109` clears the table, `:161` inserts in task-then-note order), and `task_notes` is an ordinary rowid table (`internal/storage/cache.go:49-53`), so rowid order is JSONL order. Ordering by `created` first would publish an index `note remove` does not accept whenever stored stamps run backwards — exactly the case `note_test.go:618` and `:640` now pin. There is no incremental insert path into `task_notes`; `Rebuild` is the only writer, so the published index and the index `note remove` takes cannot diverge.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_formatter_test.go:663` — two notes decode with `index` 1 and 2 in order, alongside text and created
  - `internal/cli/toon_formatter_test.go:686` and `:165` — the count-zero header is `notes[0]{index,text,created}:` and decodes to zero rows (`assertCountZeroSection` at `toon_decode_test.go:161-173` asserts both the header text and the decoded row count)
  - `internal/cli/toon_formatter_test.go:692` — multi-line text decodes back identical with `index` 1 intact; `:710` — dash-leading text decodes unchanged
  - `internal/cli/json_formatter_test.go:914` — JSON notes carry `index` 1 and 2; `:1639` — a filtered JSON document keeps the index
  - `internal/cli/note_test.go:576` — end-to-end: three notes added via the CLI, the second's published index read out of `tick show --toon`, `note remove` with that index, remaining notes asserted as the first and third from the persisted JSONL
  - `internal/cli/note_test.go:674` — two notes sharing a created stamp decode with indexes matching JSONL order
  - `internal/cli/note_test.go:618`, `:640`, `:654` — the harder ordering case: stamps running backwards against stored order still publish, remove and narrow by stored position
  - `internal/cli/conformance_test.go:1145-1159` — a position-narrowed selection decodes to one row carrying `index` 2, which is the §9.3 requirement that the count follows the selection while the row keeps its real position; `:1785` pins `index` as a JSON number
  - `internal/cli/pretty_formatter_test.go:711-716`, `:1135` — pretty's notes block pinned as exact strings, so an index leaking into it would fail
  - README samples at `README.md:196` and `README.md:488` show the header with the column and are pinned against rendered output by `internal/cli/readme_samples_test.go:425`
- Notes: No redundancy worth naming. The five subtests under `TestNoteIndex` each pin a distinct case (CLI round trip, backwards stamps on read, backwards stamps on remove, backwards stamps on narrow, tied stamps) rather than restating one.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it …", `t.Helper()` on helpers, `--toon`/`--pretty` explicit in CLI tests
- SOLID principles: Good — the index is produced once by `selectedItems` and consumed by both machine formatters, so neither recomputes position and they cannot disagree
- Complexity: Low
- Modern idioms: Yes — `slices.Sorted`/`slices.Compact` in `selectedItems`, `reflect.TypeFor` in `emptyToonSection`
- Readability: Good — the doc comments on `buildNotesSection` (`toon_formatter.go:308-309`) and `toJSONNotes` (`json_formatter.go:157-158`) both state that the row carries the position in the *whole* section, which is what the narrowed case makes non-obvious, and both hold against the code
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands; nothing in reading contradicts them, but the pass is not mine to make.
