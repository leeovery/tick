TASK: free-text-round-trip-4-5 (tick-86b409) — Pretty Renders The Filtered Document

ACCEPTANCE CRITERIA:
- Unfiltered pretty output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task, including blank-line spacing and column alignment
- `create` and `update` still print their cascade tree, appended after the joined groups with the single `"\n"` separator Phase 2 gave it
- A filtered pretty render carries no header block for unselected fields and no labels for them
- Selected fields keep pretty's label text, padding and alignment exactly as full output renders them
- `--field type` on a task with no type prints `Type:     -`
- `--field tags` on a tag-less task prints nothing, and the same for `refs`, `parent`, `closed`, `blocked_by`, `children`, `notes` and `description`
- A selection that renders nothing produces zero bytes on stdout and exits zero
- A filtered render has no leading blank line when no header line survives and no trailing blank line when the last selected piece renders nothing
- A bare-value request never reaches the pretty formatter
- `internal/cli/pretty_formatter_test.go` carries every golden-string assertion it carried before, none removed or weakened
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§9.7 (specification.md:440) requires that in pretty — the format a terminal resolves to unless a flag overrides it — a filtered document is "the named fields rendered in pretty's usual style and nothing else: no header block, no labels for sections outside the selection", and holds that a filtered record is new output rather than a change to existing output, so §4.1's bound on pretty stands. §9.6 (specification.md:426) requires an empty field to print what full output prints for it and still exit zero, read per format: pretty always prints `Type:     -` and omits `Tags:` when there are none. §9.5 forbids `id` riding along unasked. §11 keeps pretty's golden-string assertions, since pretty has no parser and a decoded-value assertion does not exist for it.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/pretty_formatter.go:116-120` — `prettyDetailLine` pads the label to the column `Priority:` sets (`%-9s %s`), reproducing the literal `"ID:       %s"` / `"Title:    %s"` / `"Priority: %d"` widths the pre-task code hard-coded
  - `internal/cli/pretty_formatter.go:122-137` — `prettyDetailBlock` / `prettyRelatedEntries` reproduce the two-space-indented block bodies
  - `internal/cli/pretty_formatter.go:139-178` — `prettyDetailHeader` gates each of the ten header lines on `sel.includes(name)` in document order, keeping the existing non-empty guards for `Tags`, `Parent` and `Closed`
  - `internal/cli/pretty_formatter.go:180-212` — `prettyDetailBlocks` gates `Blocked by`, `Children`, `Refs`, `Notes`, `Description` on the same predicate, each still behind its existing non-empty guard
  - `internal/cli/pretty_formatter.go:217-238` — `FormatTaskDetail` joins the header group with `"\n"`, joins groups with `"\n\n"`, returns `""` when no group survives, and appends the cascade tail with a single `"\n"` per block outside the group join
  - `internal/cli/show.go:84-89` — the empty-render guard suppresses the `Fprintln` when the document is `""`
  - `internal/cli/show.go:71-76` — the bare-value branch returns before the formatter is reached
  - Name constants come from the `showFields` registry (`internal/cli/show_fields.go:30-45`), so the label→selection-name mapping the task prescribed cannot drift from the names the flag parser accepts
- Notes:
  - Byte-identity of unfiltered output verified by reading the pre-task rendering against the new one (`git show f1431cc1 -- internal/cli/pretty_formatter.go`): every literal width, the `"\n\n"` block separation, the absence of a trailing newline and the cascade tail's single `"\n"` all reproduce.
  - The header's `Tags` line and every block render the narrowed `sections.*` values (positions arrived in tasks 4-6/10-6) while their non-empty guards read the whole `detail.*` slice. A narrowed-to-empty section would render a bare label, but `FieldSelection.ValidatePositions` (`internal/cli/show_fields.go:265-280`) rejects an out-of-range position before the formatter runs, and `show` is the only caller that sets `detail.Fields`, so the state is unreachable.
  - `FormatTaskDetail` drops the cascade tail when no group survives. Only `outputMutationResult` (`internal/cli/helpers.go:17-31`) sets `Changes`, and it leaves `Fields` nil, so the header always carries `ID`/`Title`/`Status`/`Priority`/`Type` and the branch is unreachable.
  - The criterion "`--field type` ... prints `Type:     -`" holds at the formatter, which is what this task owns; end-to-end `--field type` alone is a bare value (§9.2) and returns before pretty, consistent with the task's own edge case "A bare-value request never reaches pretty". The end-to-end test uses `--field type,tags` to force the document form, which is the correct reading.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/pretty_formatter_test.go:1099-1260` (`TestPrettyFilteredTaskDetail`) — golden strings for scalars only, each list block alone (`notes`, `children`, `blocked_by`), a header-plus-block mix, the `Type:     -` dash, a selected field pretty omits, a selection whose every name is empty, and no leading blank line when no header line survives. Two regression goldens pin the unfiltered document and the unfiltered document plus cascade tail.
  - `internal/cli/list_show_test.go:953-1009` — end-to-end through `runShow` (which sets `IsTTY: true`, so pretty is resolved without a flag): exact stdout for `title,status`, `tags`, `title,notes`, `type,tags`, the eight-field empty sweep with a zero exit code, and `notes,description`.
  - `internal/cli/detail_changes_test.go:166-192` (`TestTaskDetailChangedSectionPretty`) pins the cascade tail as `body + "\n" + block` for one and two blocks — the criterion about `create`/`update` keeping their tree.
  - `internal/cli/list_show_test.go:188-195` and the rest of the pre-existing pretty show/create/update/note goldens are untouched by this task's commit, so unfiltered byte-identity is pinned by assertions written before it.
  - Golden-string preservation confirmed: `git diff f1431cc1^ HEAD -- internal/cli/pretty_formatter_test.go` removes only `result := f.Format*` call lines (the later `(string, error)` signature change) and a struct field rename — no `want` string removed or loosened.
- Notes: The formatter-level and end-to-end suites overlap on four shapes, but they assert different things — the former pins the rendering, the latter pins format resolution, the trailing newline and the empty-render guard in `RunShow`. The plan asked for both (Do steps 4 and 5). Not over-tested.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` with `t.Run` "it does X" subtests and `t.Helper()` on helpers; the formatter keeps the `Formatter` interface shape; field names come from the central `showFields` registry rather than string literals.
- SOLID principles: Good. The single 70-line `FormatTaskDetail` is split into a line renderer, a block renderer, a header collector and a block collector, each with one job.
- Complexity: Low. The selection predicate is one call per piece; `FormatTaskDetail` is now a join over two groups plus the tail.
- Modern idioms: Yes. `cmp.Or` for the type dash, `strings.Join` over manual `Fprintf` accumulation, generic `selectedItems`.
- Readability: Good. `prettyDetailLine`'s comment explains the padding width by naming the label that sets it, and both collector comments state the document-order and empty-section rules; all three hold against the code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands from the repo root; reading cannot confirm a clean suite, vet or formatting pass.
