TASK: free-text-round-trip-4-8 (tick-4de057) — README Documents Field Selection

ACCEPTANCE CRITERIA:
- All three samples were copied from real tool output rather than hand-written
- The filtered document sample is a decodable TOON document and is anchored in `TestREADMEToonSamplesDecode` by a first line unique among the README's fenced blocks (superseded by the orchestrator addendum: add it to the `readmeSampleGroups` fixture table instead)
- The two bare-value samples are not anchored — neither is a TOON document
- Both `--field` and `--fields` appear, with the plural documented as an alias of the singular
- The documented accepted names match the registry Task 1 added, and the documented flag description does not contradict `tick help show`
- The one-versus-several split, normal section order, the format-flag interaction, the `--quiet` refusal and the two error cases are all stated
- The positional form is documented as 1-based and available on every list section
- `grep -n '^\$ tick show' README.md` finds the new samples and the section carries no invented output
- No `--` documentation is added
- The samples corrected in Phases 1, 2 and 3 are unchanged by this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §12.1 owes the README the new input surface — "a flag documented nowhere is a flag nobody uses" — and forbids leaving the README describing output the tool does not produce. §9.1 fixes the vocabulary (the names the output document uses; `--fields` an alias of `--field`), §9.2 the one-versus-several split and normal section order, §9.3 the 1-based positional grammar on every list section (`notes.2`, `tags.1`, `refs.2`, `children.3`, `blocked_by.1`) with only `notes` carrying its real position in the row, §9.6 the unrecognised-name and out-of-range errors, §9.7 the format-flag interaction, §9.8 the `--quiet` refusal. §3.1 exempts bare output from the must-parse inventory. §11 leaves conformance coverage of filtered documents to Phase 6.

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:168-212 (`### show`); flag table at README.md:177-180; accepted-name list at README.md:182; the three samples at README.md:186-190, 194-201, 205-208; the behaviour prose at README.md:184, 192, 203, 210, 212. Test-side fixture: internal/cli/readme_samples_test.go:25 (`readmeFieldSelectionAnchor`), :311-330 (`fieldSelectionTasks`), :343-348 (the "it reproduces the README field selection sample" group), :544-606 (`readmeShowSection`, `TestREADMEDocumentsFieldSelection`, `backticked`, `showHelpFlagNames`). Delivered in commit 7c8cac7c.
- Notes:
  - Documented names at README.md:182 match the registry exactly: the 15 keys of `showFields` (internal/cli/show_fields.go:50-66) — id, title, status, priority, type, parent, created, updated, closed, description, notes, tags, refs, children, blocked_by.
  - The positional claim ("Every list section — notes, tags, refs, children, blocked_by — also accepts a 1-based position") matches `showListSections` (internal/cli/show_fields.go:70) and `isList()` gating in `addValue` (show_fields.go:229).
  - Sample section order (notes, then description) matches `ToonFormatter.FormatTaskDetail`'s fixed order (internal/cli/toon_formatter.go:71-107: task fields, blocked_by, children, tags, refs, notes, changed, description), so "in normal output order rather than the order they were typed" holds.
  - README.md:203 and :210 match the code: `bareFieldValue` returns bare only for a sole name resolving to one value, and `barePositionValue` requires a section with an `items` func — `children`/`blocked_by` have none (show_fields.go:61-62, 118-145), so a lone position on them falls through to the one-row document. `buildNotesSection` (toon_formatter.go:310-323) stamps each row with its whole-section position from `NotePositions`, which is what README.md:210 claims for `--field title,notes.2`.
  - README.md:212's three claims hold: the bare path prints before any formatter is consulted (internal/cli/show.go:71-76), `--quiet` plus a selection is refused (show.go:45-47), and both `unknownFieldError` (show_fields.go:241-243) and `ValidatePositions` (show_fields.go:284-298) return errors, which exit non-zero.
  - No contradiction with `tick help show` (internal/cli/help.go:72-79, `{"--field, --fields", "<name,...>", "Select fields by name; a section may be narrowed with .N (e.g. notes.2)"}`) — the README table is a superset phrased compatibly, and `--fields` is called the alias §9.1 says it is.
  - Scope boundary held: `git show 7c8cac7c -- README.md` modifies no existing output sample; the only change to pre-existing text is the added usage line README.md:174. No `--` (end-of-flags) documentation is in the commit — the `--` prose at README.md:110, 296, 633, 643 arrived later under task 5-6.
  - Deliberate, sound divergence from the written criteria, both from later tasks rather than drift: (a) the orchestrator's addendum replaced the decode anchor with the fixture table, which this task implemented; (b) task 4-13 (b9782df7) later added the two bare-value samples to `readmeSampleGroups` as `readmeBareDescriptionAnchor`/`readmeBareNoteAnchor` (readme_samples_test.go:26-27, 349-360). That does not violate "not anchored — neither is a TOON document": `TestREADMESamplesMatchRenderedOutput` compares rendered bytes and never decodes, and neither bare block is looked up by `readmeToonBlocks`, whose only consumer is `TestREADMEToonSamplesDecode` (readme_samples_test.go:165) against the list/transition/cascade anchors. §3.1's exemption is intact and the guard is strictly stronger.

TESTS:
- Status: Adequate
- Coverage: `TestREADMESamplesMatchRenderedOutput` (internal/cli/readme_samples_test.go:425-452) seeds `fieldSelectionTasks` and compares `tick --toon show tick-a1b2 --field description,notes` byte-for-byte with the README block found by info string + first line + occurrence — the sample cannot drift from the tool without the test failing, which is the whole point of §12.1. Its negative case at :446-451 covers the missing-block failure the task asked for. `TestREADMEDocumentsFieldSelection` (:563-581) covers "it documents both field flag spellings" and "it matches the help text for show" (every long flag in show's help entry must appear backticked in the `### show` section, so help/README drift fails). The earlier anchors still resolve: `readmeListSampleAnchor`, `readmeTransitionAnchor`, `readmeCascadeAnchor` decode in `TestREADMEToonSamplesDecode`, and `readmeShowSampleAnchor`/`readmeDepTreeAnchor` are claimed by fixture entries.
- Notes:
  - Anchor uniqueness verified by enumerating the first line of every fence in README.md: `notes[2]{index,text,created}:`, `Full task description here.` and `Blocked on the migration landing` each occur exactly once as a fence first line, so `occurrence: 1` cannot collide.
  - Not over-tested: "it documents both field flag spellings" is not redundant with "it matches the help text for show" — the first guards the README independently of the help registry, so removing `--fields` from help would not silently drop the README assertion.
  - The prose claims at README.md:212 (format-flag interaction, `--quiet` refusal, error exits) carry no assertion in this task's tests, which is right: they are behaviours owned and tested by the Phase 4 implementation tasks, and this task's obligation was that the statements be present.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run()` subtests in "it does X" form, `t.Helper()` on helpers, `--toon`/`--pretty` passed explicitly because `bytes.Buffer` is non-TTY.
- SOLID principles: Good — the fixture table keeps sample declaration data-only; helpers (`readmeShowSection`, `showHelpFlagNames`, `backticked`) each do one thing.
- Complexity: Low.
- Modern idioms: Yes — `strings.SplitSeq`, `strings.Cut`, consistent with the rest of the file.
- Readability: Good — doc comments on `readmeShowSection` and `showHelpFlagNames` state what they return without restating the code, and carry no task/phase/spec references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "All three samples were copied from real tool output rather than hand-written" — reading confirms each sample is consistent with the renderer (section order, row shape, bare-value path) and that all three are claimed by `readmeSampleGroups`, but provenance is settled only by running `go test ./internal/cli -run TestREADMESamplesMatchRenderedOutput`, which compares each block to freshly rendered output.
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires executing the three commands.
