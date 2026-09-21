TASK: free-text-round-trip-4-3 — Several Fields Return A Filtered Toon Document

ACCEPTANCE CRITERIA:
- A filtered toon document decodes without error via `toon.DecodeString`
- The decoded document carries exactly the selected names as keys and no others — `id` included only when asked for
- Sections appear in normal output order regardless of the order the names were typed
- `--field notes` on a task with no notes prints `notes[0]{index,text,created}:`, and the same holds for `children` and `blocked_by`
- `--field tags` on a task with no tags prints nothing, and the same holds for `refs`, `description`, `type`, `parent` and `closed`
- A selection whose every name prints nothing produces zero bytes on stdout and exits zero
- A single name of a list section renders that section rather than a bare value
- Scalars and sections mix in one document with exactly one blank line between the head block and the first section
- No leading or trailing blank line appears when a selected field renders nothing
- A nil selection renders the document byte-identically to before this task, and `create`, `update`, `note add` and `note remove` never carry one
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§9.2 requires a multi-field selection to return "the normal document with only those sections in it… identical to a full `tick show` minus the sections not asked for", with sections keeping their usual output order rather than the typed order. §9.4 makes the task's own scalars selectable individually and unwrapped. §9.5 forbids anything riding along unasked, `id` included. §9.6 fixes the per-name emptiness rule: a field full output omits prints nothing, an always-present section prints its count-zero header, and a selection whose every name prints nothing prints nothing at all and still exits zero. §9.7 says a document-returning request honours the resolved format. §3.1 requires every `show` document to decode, exempting only the zero-byte selection. §5.2 fixes the presence rules the task must not disturb.

IMPLEMENTATION:
- Status: Implemented (evolved past the plan's wording by later tasks 4-6/4-12/10-1/10-4, consistently)
- Location:
  - internal/cli/format.go:105 — `Fields *FieldSelection` on `TaskDetail`, nil meaning the whole document
  - internal/cli/toon_formatter.go:71-103 — `FormatTaskDetail` gates the head block and each of `blocked_by`, `children`, `tags`, `refs`, `notes`, `description` on `sel.includes(name)`, in today's output order
  - internal/cli/toon_formatter.go:228-260 — `buildTaskSection` adds each of the nine scalars only when selected, keeps the non-empty guards on `type`, `parent` and `closed`, and returns `""` when no field survives
  - internal/cli/toon_formatter.go:398-422 — `toonDoc`/`joinToonSections` drop empty pieces before joining with `"\n\n"`
  - internal/cli/show_fields.go:164-176 — `includes`/`Positions` answer a nil receiver as the whole document (task 10-4's refinement of the plan's `Selected`)
  - internal/cli/show.go:83-90 — `detail.Fields = selection`, then render-then-print that writes nothing for an empty document
- Notes:
  - The plan's `detail.Fields == nil || detail.Fields.Selected(name)` became the nil-safe method `(*FieldSelection).includes`, which answers `true` for a nil receiver. Same behaviour, one call site shape; a legitimate move (task 10-4 made it explicit).
  - `id` is gated like every other scalar (toon_formatter.go:238 via `add(fieldID, t.ID)`), so §9.5 holds.
  - The presence rules survive: `buildRelatedSection` and `buildNotesSection` emit the count-zero header when their (possibly narrowed) slice is empty, while `tags`, `refs`, `description`, `type`, `parent` and `closed` are guarded on what the task carries.
  - The `changed` section (toon_formatter.go:98) is gated on `detail.Changes != nil` alone rather than on the selection, diverging from the task's edge-case wording. It cannot appear under a selection: `Fields` is set only at show.go:83 and `Changes` only at helpers.go:29 (`outputMutationResult`), and neither path sets the other — confirmed by grepping every assignment of both fields across internal/cli. `changed` is also not a registry name (show_fields.go:30-46), so gating it off the registry (task 4-12) would be meaningless. No behaviour is lost; not reported as a finding.
  - `tags`/`refs` presence is judged on the unnarrowed `detail.Tags`/`detail.Refs` while the rows come from the narrowed `sections`. Safe as delivered because `RunShow` calls `ValidatePositions` (show.go:63) before formatting, so a surviving position always keeps at least one item.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/toon_formatter_test.go:1243-1382 (`TestToonFilteredTaskDetail`) — selected scalars only; output order under reversed typing; a section alone; scalars mixed with sections in output order; exactly one blank line between head and first section; `id` absent unless asked; count-zero header for `notes`, `children` and `blocked_by` on a bare task; empty render for `tags`, `refs`, `description`, `type`, `parent` and `closed`; zero bytes when every name is empty; no leading blank line when the head is empty; no trailing blank line when the last name renders nothing; unfiltered output unchanged (full key set plus decoded values).
  - internal/cli/toon_formatter_test.go:892-905 — `buildTaskSection` returns `""` rather than a blank line when no scalar is selected, and `joinToonSections` drops it.
  - internal/cli/list_show_test.go:916-953 — end-to-end `--toon` through `runShow`: a filtered document decodes and carries the expected values; a lone list section is a document not a bare value; count-zero header for an always-present section; zero bytes and exit 0 when every name is empty; no unselected keys.
  - Every filtered assertion decodes through `decodeToonDoc` (internal/cli/toon_decode_test.go:17), which is `toon.DecodeString` plus a single-object check, so the documents are asserted by decoded value rather than by pinned strings.
  - Phase 6's inventory reaches the same ground from outside: conformance_test.go:361-392 declares the multi-field, positional and lone-section selections as documents and the bare-value and zero-byte selections as the two exemptions.
- Notes: The duplication between the formatter unit tests and the end-to-end subtests is the plan's own split (Do steps 4 and 5) — unit for the render, end-to-end for the handler's print-nothing branch — and each layer observes something the other cannot. Not over-tested.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` "it does X" subtests, `t.Helper()` on helpers, `t.TempDir()`-based project fixtures, error wrapping unchanged.
- SOLID principles: Good. The selection is data on `TaskDetail` and the presence decision stays in the formatter; the handler decides only whether to print.
- Complexity: Low. `FormatTaskDetail` is a flat sequence of guarded `doc.add` calls; `buildTaskSection` is a single `add` closure over the nine scalars.
- Modern idioms: Yes — nil-receiver methods on `*FieldSelection` remove the `sel == nil ||` repetition at every call site; generics carry `selectedItems` and `emptyToonSection`.
- Readability: Good. Doc comments on `FormatTaskDetail`, `buildTaskSection`, `TaskDetail.Fields`, `includes` and `RunShow` all state the nil-means-whole-document rule and hold against the code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands from the repo root; reading cannot establish that the suite passes or that the toolchain reports nothing.
