# Consolidation Tasks: Free Text Round Trip (Phase 10)

## Task 1: Narrowing A Task's Blockers By Position Is Pinned Like Every Other Section
placement: phase 10
severity: behaviour

**Problem**: Task 10-1 merged the three per-formatter narrowing sites into `TaskDetail.selectedSections` (`internal/cli/format.go:122-132`), so one line now governs how `blocked_by` is narrowed by position in toon, pretty and JSON. Nothing pins that line. Measured by mutation in a scratch copy of the tree, twice — once by the task's reviewer and once by the phase's finder: replacing `d.Fields.Positions(fieldBlockedBy)` with `nil` leaves `go test ./internal/cli` green, while the identical mutation on each of `children`, `tags`, `refs` and `notes` fails the package. The behaviour is correct today — a binary built from the tree returns the second blocker for `--field blocked_by.2` in all three formats — so what is missing is the guard, not the code. `TestShowFieldPositions` (`internal/cli/list_show_test.go:1085`) builds its fixture with two children, three notes, two tags and two refs and no blockers at all, so its narrowing subtests reach every list section except this one; the only `blocked_by` positional coverage anywhere is error-path, on a one-blocker fixture where `blocked_by.1` is indistinguishable from the whole section. An agent that asked for one blocker and silently received all of them, or the wrong one, acts on a blocker ID it never asked for — removing a dependency against the wrong task, or reporting the wrong blocker upstream — with exit 0, a document that parses, and a count header that follows the selection.

**Solution**: Give `TestShowFieldPositions` a fixture carrying two blockers and assert `--field blocked_by.2` returns the second in toon and in pretty, mirroring the shapes the sibling sections already use — `"it narrows a table to one row"` (`internal/cli/list_show_test.go:1159`) for the toon section and `"it narrows a pretty children block"` (`:1266`) for the pretty block. Test-only: no production line changes, and the existing subtests keep their current fixtures and assertions. Confirm the guard bites by re-running the mutation the finder measured — `nil` in place of `d.Fields.Positions(fieldBlockedBy)` must now fail — and restoring.

**Outcome**: Every list section's positional narrowing is pinned by a test that fails when the pairing at the single narrowing site is lost or mispaired, so the consolidation task 10-1 performed cannot silently drop one section.

**Do**:
- Extend the `newProject` fixture in `TestShowFieldPositions` (`internal/cli/list_show_test.go:1096-1107`) with two open blocker tasks — `tick-b1b1b1` "Blocker one" and `tick-b2b2b2` "Blocker two" — declared ahead of `tick-a1b2c3`, and give `tick-a1b2c3` `BlockedBy: []string{"tick-b1b1b1", "tick-b2b2b2"}`, mirroring the fixture shape `TestShowFieldPositionOutOfRange`'s `richProject` already uses (`list_show_test.go:1291-1304`). Blockers reach the document in `ORDER BY t.id` (`internal/cli/show.go:137`), so position 2 is `tick-b2b2b2`; declare `secondBlocker := RelatedTask{ID: "tick-b2b2b2", Title: "Blocker two", Status: "open"}` beside the existing `firstChild` (`:1094`).
- Add a toon subtest beside `"it narrows a table to one row"` (`:1159`) asserting `assertToonRelatedRow(t, showToon(t, "blocked_by.2"), "blocked_by", secondBlocker)` — the same one-row shape the children subtest uses.
- Add a pretty subtest beside `"it narrows a pretty children block"` (`:1264`) running `--pretty --field title,blocked_by.2` and comparing stdout to `"Title:    Add login\n\nBlocked by:\n  tick-b2b2b2  Blocker two (open)\n"`.
- Run `go test ./internal/cli`; the two new subtests pass and every existing subtest in the package passes unchanged, including `"it renders every item without positions"` (`:1273`), which now decodes a document carrying a two-row `blocked_by` section.
- Confirm the guard bites: replace `d.Fields.Positions(fieldBlockedBy)` with `nil` at `internal/cli/format.go:126`, re-run `go test ./internal/cli -run TestShowFieldPositions` and see both new subtests fail, then restore the line verbatim and re-run green.

**Acceptance Criteria**:
- [ ] `TestShowFieldPositions`' fixture carries two blockers on `tick-a1b2c3`, and no existing subtest in that function has its arguments, expected values or assertions changed.
- [ ] A toon subtest asserts `--field blocked_by.2` yields a `blocked_by` section of exactly one row, carrying `tick-b2b2b2` / "Blocker two" / `open`.
- [ ] A pretty subtest asserts `--pretty --field title,blocked_by.2` prints exactly one entry under `Blocked by:`, byte-for-byte against the expected string.
- [ ] With `d.Fields.Positions(fieldBlockedBy)` replaced by `nil` at `internal/cli/format.go:126`, both new subtests fail; the line is restored before the task is committed.
- [ ] `go test ./...` passes, `go vet ./...` is clean, and `gofmt -l ./internal ./cmd` prints nothing.
- [ ] The committed diff touches `internal/cli/list_show_test.go` only — no production line changes, no `.workflows/` edits.

**Tests**:
- `"it narrows a blockers table to one row"`
- `"it narrows a pretty blocked-by block"`
