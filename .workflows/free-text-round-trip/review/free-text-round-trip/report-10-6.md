TASK: free-text-round-trip-10-6 (tick-9e1023) — Narrowing A Task's Blockers By Position Is Pinned Like Every Other Section

ACCEPTANCE CRITERIA:
- `TestShowFieldPositions`' fixture carries two blockers on `tick-a1b2c3`, and no existing subtest in that function has its arguments, expected values or assertions changed.
- A toon subtest asserts `--field blocked_by.2` yields a `blocked_by` section of exactly one row, carrying `tick-b2b2b2` / "Blocker two" / `open`.
- A pretty subtest asserts `--pretty --field title,blocked_by.2` prints exactly one entry under `Blocked by:`, byte-for-byte against the expected string.
- With `d.Fields.Positions(fieldBlockedBy)` replaced by `nil` at `internal/cli/format.go:126`, both new subtests fail; the line is restored before the task is committed.
- `go test ./...` passes, `go vet ./...` is clean, and `gofmt -l ./internal ./cmd` prints nothing.
- The committed diff touches `internal/cli/list_show_test.go` only — no production line changes, no `.workflows/` edits.

STATUS: complete

SPEC CONTEXT: §9.3 ("List fields are reachable by position") states the positional grammar holds on every list section and names `blocked_by.1` explicitly alongside `notes.2`, `tags.1`, `refs.2`, `children.3`: the section renders in its normal form narrowed to the named item, a table keeping its header and one row. §9.2 adds that a selection resolving to a row rather than a value — `children.1`, and by the same rule `blocked_by.2` — comes back as its one-row section rather than a bare value. This task adds no behaviour; it pins the `blocked_by` half of that grammar, which task 10-1's consolidation into `TaskDetail.selectedSections` left governed by a single unguarded line.

IMPLEMENTATION:
- Status: Implemented (test-only, as planned)
- Location:
  - Fixture: `internal/cli/list_show_test.go:1095` (`secondBlocker := RelatedTask{ID: "tick-b2b2b2", Title: "Blocker two", Status: "open"}`) and `:1100-1106` (two open blocker tasks declared ahead of `tick-a1b2c3`, which now carries `BlockedBy: []string{"tick-b1b1b1", "tick-b2b2b2"}`).
  - Toon subtest: `internal/cli/list_show_test.go:1169-1171` — `"it narrows a blockers table to one row"`, `assertToonRelatedRow(t, showToon(t, "blocked_by.2"), "blocked_by", secondBlocker)`.
  - Pretty subtest: `internal/cli/list_show_test.go:1283-1290` — `"it narrows a pretty blocked-by block"`, comparing stdout byte-for-byte to `"Title:    Add login\n\nBlocked by:\n  tick-b2b2b2  Blocker two (open)\n"`.
  - Guarded site intact and unmutated: `internal/cli/format.go:126` reads `blockedBy, _ := selectedItems(d.BlockedBy, d.Fields.Positions(fieldBlockedBy))`.
- Notes:
  - Ordering assumption holds: blockers reach the document through `ORDER BY t.id` (`internal/cli/show.go:137`), so position 2 of `{tick-b1b1b1, tick-b2b2b2}` is `tick-b2b2b2`.
  - The pretty expected string reproduces the renderer exactly: `prettyDetailLine` pads the label with `%-9s ` giving `"Title:    Add login"` (`internal/cli/pretty_formatter.go:118-119`), blocks join on `"\n\n"` (`:229`), and `prettyDetailBlock`/`prettyRelatedEntries` produce `"Blocked by:\n  tick-b2b2b2  Blocker two (open)"` (`:122-137`, `:188`).
  - The lone `--field blocked_by.2` selection correctly renders a document rather than a bare value: `showFields[fieldBlockedBy]` has no `items` and no `bare` func (`internal/cli/show_fields.go:61`), so `bareFieldValue`/`barePositionValue` both decline (`show_fields.go:118-145`), matching §9.2.
  - The fixture is seeded through `setupTickProjectWithTasks` (`internal/cli/create_test.go:34`), which writes JSONL and lets the cache rebuild, so the `dependencies` rows exist — the same shape `TestShowFieldPositionOutOfRange`'s fixture (`list_show_test.go:1313-1318`) already relies on.

TESTS:
- Status: Adequate
- Coverage: `blocked_by` positional narrowing is now asserted in toon (exactly one row, with id/title/status checked by value after decoding) and in pretty (whole-stdout byte comparison). All five list sections — `blocked_by`, `children`, `tags`, `refs`, `notes` — now have at least one positional-narrowing assertion in `TestShowFieldPositions`.
- Notes:
  - The guard bites as required by criterion 4, by construction: with `nil` in place of `d.Fields.Positions(fieldBlockedBy)`, `selectedItems` takes its `positions == nil` branch (`internal/cli/show_fields.go:77-83`) and returns both blockers, so `assertToonRelatedRow` fatals on `rows = 2, want 1` (`internal/cli/toon_decode_test.go:147-149`) and the pretty comparison sees a second `  tick-b1b1b1  Blocker one (open)` line. Both new subtests fail; the re-run itself is recorded under UNSETTLED.
  - No over-testing: two subtests, one per format, each mirroring the sibling section's existing shape. JSON is not separately pinned, but the narrowing has exactly one site (`format.go:125-140`), consumed by all three formatters (`toon_formatter.go:79`, `json_formatter.go:133`, `pretty_formatter.go:188`), so a third assertion would add no discrimination.
  - Existing subtests are untouched in substance. Line arithmetic against the plan's pre-change citations corroborates additions only: `"it narrows a table to one row"` moved :1159 → :1165 (+6, the fixture lines), `"it narrows a pretty children block"` :1264 → :1274 (+10, fixture plus the 4-line toon subtest), `"it renders every item without positions"` :1273 → :1292 (+19, plus the 9-line pretty subtest). The added blockers touch no existing assertion: the note/tag/ref/children subtests select fields that exclude `blocked_by`, and `assertToonKeySet(t, doc, "title", "notes")` (:1147) is unaffected because the selection gates the key set.
  - `"it renders every item without positions"` (:1292) now decodes a full document carrying a two-row `blocked_by` section, which incidentally widens that subtest's conformance reach without changing its assertions.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.Helper()` on the local helpers, `t.TempDir()` isolation via `setupTickProjectWithTasks`, shared assertion helpers reused rather than re-rolled.
- SOLID principles: N/A (test-only change); the new subtests reuse `assertToonRelatedRow` rather than duplicating row-shape assertions.
- Complexity: Low.
- Modern idioms: Yes.
- Readability: Good — `secondBlocker` is declared beside `firstChild` at the top of the test function, and each new subtest sits beside the sibling section it mirrors.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "With `d.Fields.Positions(fieldBlockedBy)` replaced by `nil` at `internal/cli/format.go:126`, both new subtests fail; the line is restored before the task is committed." — the restore half is settled by reading (`internal/cli/format.go:126` carries the line verbatim) and the failure half follows from `selectedItems`' nil-positions branch; confirming it empirically needs the mutation applied and `go test ./internal/cli -run TestShowFieldPositions` run, then the line restored.
- "`go test ./...` passes, `go vet ./...` is clean, and `gofmt -l ./internal ./cmd` prints nothing." — needs those three commands run; not settleable by reading.
- "The committed diff touches `internal/cli/list_show_test.go` only — no production line changes, no `.workflows/` edits." — the in-file evidence is consistent with additions to `list_show_test.go` alone, but the commit's file set needs `git show --stat 93ff21ad` to settle.
