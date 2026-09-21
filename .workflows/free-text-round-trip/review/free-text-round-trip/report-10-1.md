TASK: free-text-round-trip-10-1 (tick-e8fce5) — The Field Selection Narrows Each List Section Once

ACCEPTANCE CRITERIA:
1. The three formatters call `selectedItems` nowhere; every remaining non-test hit is in `format.go` (inside `selectedSections`) or `show_fields.go`.
2. Each of `fieldBlockedBy`, `fieldChildren`, `fieldTags`, `fieldRefs`, `fieldNotes` is handed to `Positions` in exactly one place in non-test code.
3. Output is byte-identical against a binary built before the change, in all three formats, filtered and full.
4. A narrowed note still carries its whole-section position in the `index` column under toon and JSON.
5. The presence gates still read the unnarrowed collections.
6. No test file is changed; the diff is confined to `format.go`, `toon_formatter.go`, `json_formatter.go`, `pretty_formatter.go`.
7. `go test ./...`, `go vet ./...`, `golangci-lint run ./...` clean, `gofmt` applied.

STATUS: complete

SPEC CONTEXT: §9.2/§9.3 define field selection and positional narrowing — `notes.2`, `tags.1`, `refs.2`, `children.3`, `blocked_by.1`, 1-based, "a position narrows the section it names and nothing else" (specification.md:401-403). §9.3 also fixes the rule this task's centralisation must preserve: "Only notes carry a real position in the row itself, because only notes are addressed by position elsewhere in the CLI (§6.3)" (:403), and §6.3 (:223) requires the notes `index` column present whether filtered or not. The failure the task guards against is a narrowed note reported under one format at a position that does not address it under another, leading to a wrong `note remove`.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/format.go:111-121` — unexported `selectedSections` struct (`BlockedBy`, `Children []RelatedTask`, `Tags`, `Refs []string`, `Notes []task.Note`, `NotePositions []int`).
  - `internal/cli/format.go:123-140` — `func (d TaskDetail) selectedSections() selectedSections`, the one place each of the five field-name constants meets its collection (`:126-130`).
  - `internal/cli/toon_formatter.go:73` binds it; `:79`, `:83`, `:87`, `:91`, `:95` read from it; the five `selectedItems` calls are gone.
  - `internal/cli/json_formatter.go:107` binds it; `:121-123`, `:133-134` read from it; `toJSONNotes(sections.Notes, sections.NotePositions)` replaces the two-value pass-through.
  - `internal/cli/pretty_formatter.go:143` and `:184` each bind it; `:159`, `:188`, `:192`, `:196`, `:200-202` read from it. Both helpers keep their single `detail` parameter.
  - Commit `add91a8d` touches exactly the four files named by criterion 6 (+51/-30), no test file.
- Notes:
  - Criterion 1 holds: `grep -rn 'selectedItems(' internal/cli/ | grep -v _test` now returns 6 lines — `format.go:126-130` and `show_fields.go:140` (`barePositionValue`, untouched as the plan required).
  - Criterion 2 holds: the same grep restricted to `Positions(` shows each of the five constants exactly once, at `format.go:126-130`. The only other non-test `Positions` callers pass a dynamic `name` (`show_fields.go:123`, `:291`).
  - Criterion 4 holds by reading: `NotePositions` is `selectedItems`' second return, which carries whole-section positions for both the nil-positions branch (`show_fields.go:78-82`) and the narrowed branch (`:95`); `buildNotesSection` writes `positions[i]` into the toon `index` column (`toon_formatter.go:317`) and `toJSONNotes` into the JSON `index` key (`json_formatter.go:163`). The two slices are produced together and are equal-length by construction, so the `positions[i]` index is safe.
  - Criterion 5 holds: every gate still consults the unnarrowed collections — `toon_formatter.go:86`, `:90`; `pretty_formatter.go:158`, `:187`, `:191`, `:195`, `:199`. JSON's gate is `fields.includes` only, as before.
  - Behaviour preservation reads clean: each deleted expression is replaced by the identical value, and `selectedItems` is pure, so moving the narrowing out of the gates and computing it unconditionally cannot change output. `Positions` is nil-safe (`show_fields.go:171-176`), so a nil `d.Fields` keeps every item and every position through the same call.
  - Residual, not a defect: within each formatter both `detail.X` (for the gate) and `sections.X` (for the value) are now in scope, so rendering the unnarrowed collection by accident still compiles. That is the same class of risk as before the change, moved but not enlarged — the name-to-collection pairing this task targeted is genuinely single-sited now.

TESTS:
- Status: Adequate (pure refactor — no new test expected; criterion 6 forbids changing tests)
- Coverage: `TestShowFieldPositions` (`internal/cli/list_show_test.go:1085`) exercises narrowing in every section under toon — notes (`:1144`, `:1153`, `:1173`), tags (`:1157`), refs (`:1161`), children (`:1165`), blocked_by (`:1169`) — plus the real-index assertion for JSON (`:1232`) and pretty narrowing for notes (`:1256`), tags (`:1265`), children (`:1274`) and blocked_by (`:1283`). Because all three formatters now read the same `selectedSections`, the toon suite alone would fail on any mis-pairing of a field name with its collection inside `format.go`. `TestTaskDetailWithoutFieldSelection` (`internal/cli/show_fields_test.go:756`) covers the nil-`Fields` path through all three formatters, and `TestFieldSelectionNilReceiver` (`:717`) pins `Positions` nil-safety directly.
- Notes: JSON positional narrowing is pinned only for notes (`list_show_test.go:1232`); `json_formatter_test.go:1764` checks the key set for `title,notes.2`, not values. A per-format regression in JSON that rendered `detail.Tags`/`Refs`/`BlockedBy`/`Children` instead of the narrowed value would go unobserved. This gap pre-dates the task, is outside what it altered in substance (the values rendered are unchanged), and criterion 6 forbids the test change that would close it — recorded here rather than as a finding, and adjacent to what task 10-6 did for toon/pretty blockers.

CODE QUALITY:
- Project conventions: Followed. Unexported helper type beside the struct it derives from, value receiver, doc comments opening with the identifier, `internal/cli` layering untouched.
- SOLID principles: Good — narrowing (one concern) is separated from rendering (three format-specific concerns); formatters retain only their own gates and layout.
- Complexity: Low. Each formatter body loses a local binding per section; `selectedSections` is straight-line.
- Modern idioms: Yes. Generic `selectedItems` reused unchanged; no reflection or duplication introduced.
- Readability: Good. The type and method share the name `selectedSections`, which is legal and reads naturally at the call site (`detail.selectedSections()`); the struct's field names mirror `TaskDetail`'s, which makes the substitution obvious in review.
- Comment accuracy: Both new comments hold — `format.go:111-113` describes `NotePositions` correctly, and `:123-124`'s nil-selection claim is backed by `show_fields.go:77-82` and `:171-176`. No process-artifact references.
- Issues: None meeting the bar. (`prettyDetailHeader` and `prettyDetailBlocks` each compute `selectedSections()`, so a pretty detail render narrows twice — ten `selectedItems` calls over slices of single-digit length for one task; no consequence worth an edit.)

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Output is byte-identical against a binary built before the change, in all three formats, filtered and full: `tick show <id>`, `--field notes.2`, `--field title,tags.2`, `--field children.1` and `--field notes.1,notes.3`, each under `--toon`, `--json` and `--pretty`." — Reading settles semantic equivalence (every deleted expression is replaced by the identical pure-function value, and no gate moved), but the literal byte comparison needs the 15 invocations run against a binary built at `add91a8d^` and diffed.
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied." — Requires executing the toolchain; not settleable by reading.
