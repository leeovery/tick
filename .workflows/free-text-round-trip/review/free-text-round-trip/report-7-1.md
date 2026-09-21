TASK: free-text-round-trip-7-1 (tick-6ef31f) — The Published Note Index Addresses The Note `note remove` Deletes

ACCEPTANCE CRITERIA:
- The notes query in `queryShowData` orders by `rowid ASC` and carries no `created` term.
- On a task whose stored notes carry descending stamps, `tick show <id> --toon` lists them in stored order, `index` 1 naming the first note in `tasks.jsonl`.
- On that same task, `tick note remove <id> N` removes the note published at row N, for each N the document carries.
- `--field notes.1` on that task returns the first stored note, and a position past the last note still fails as out of range.
- Notes sharing one `created` stamp still render in insertion order.
- Ordinary ascending-stamp output is unchanged: `go test ./...` green with no existing test edited.

STATUS: complete

SPEC CONTEXT: §6.3 (specification.md:219-231) makes position the only handle a note has — there is no note ID, `note remove` takes a 1-based index, and notes are an append-and-retract log. The index column exists so an agent that read a narrowed section can call `note remove` on the right row; §9.3 (specification.md:392-395) and the out-of-range rule (specification.md:434) reuse the same addressing for `--field notes.N`. The whole point of the column is that the published position and the removed position are the same position.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/show.go:173 — `SELECT text, created FROM task_notes WHERE task_id = ? ORDER BY rowid ASC`; commit 82f7c489 changes only this line plus test additions.
- Notes: The `created` ordering term is gone; `created` remains in the projection because it is the value the section renders, which the criterion's "no `created` term" is about ordering, not selection. The correctness of `rowid` as the JSONL position holds on inspection: `task_notes` (internal/storage/cache.go:49-53) is a plain rowid table with no `INTEGER PRIMARY KEY` and no `WITHOUT ROWID`, the only writes are a full `DELETE FROM task_notes` (cache.go:120) followed by sequential inserts in task-then-note slice order (cache.go:161, cache.go:228), the only callers of `Cache.Rebuild` are full rebuilds (storage/store.go:206, :260, :427 — no incremental note path), and nothing in the repo runs `VACUUM`, so per-task rowid order equals stored order on every rebuild. `note remove` indexes the JSONL slice directly (internal/cli/note.go:130-134), so published index N and removed slice position N-1 now coincide by construction rather than by the stamps agreeing.
- The fix reaches every consumer, not just toon: `data.notes` feeds `TaskDetail.Notes`, which `format.go:130` narrows and the toon (toon_formatter.go:95), JSON (json_formatter.go:123) and pretty (pretty_formatter.go:199-201) formatters all render, and `show_fields.go:65` derives `notes` length and items from the same slice for `--field notes.N` and `ValidatePositions` (show_fields.go:283-297). No other production query reads `task_notes`, and nothing sorts notes anywhere else.

TESTS:
- Status: Adequate
- Coverage: Three new subtests on a shared fixture `descendingStampTask` (note_test.go:707-716 — first slice entry carries the later stamp): "it renders notes in stored order when created stamps run backwards" (note_test.go:618) decodes the toon document and asserts rows 1/2 by index and text; "it removes the note the published index names when stamps run backwards" (note_test.go:640) runs `note remove 1` and asserts against the persisted JSONL that the row published as index 1 is the one gone; "it narrows to the first stored note when stamps run backwards" (note_test.go:654) asserts `--field notes.1` returns the first stored note and `--field notes.3` fails with `notes.3 out of range`. The pre-existing "it matches the index note remove accepts" (note_test.go:576) and "it keeps insertion order for notes sharing a created timestamp" (note_test.go:674) are untouched — the commit diff is additions only.
- Notes: The discriminating assertion is the render subtest — removal behaviour never changed, so the removal subtest would have passed before the fix too; read as a pair they pin the correspondence, and the removal subtest guards the other direction (a future change sorting the removal side). The third criterion says "for each N the document carries" and only N=1 is executed; N=2 is not exercised, but the mapping is structural (published index = rowid position = slice position = removal index), so no residual behaviour is unobserved. Assertions decode the document rather than pinning full-output strings, consistent with the phase 6 conformance rule. No redundancy: each subtest covers a distinct consumer (document, removal, field selection).

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` "it does X" subtest naming, `t.TempDir()`-backed fixtures via `setupTickProjectWithTasks`, `t.Helper()` on `descendingStampTask`'s callees, no testify.
- SOLID principles: Good — the change is one ordering term in the single query that owns note retrieval; every downstream view inherits it rather than repeating an order.
- Complexity: Low — one-line change.
- Modern idioms: Yes — `slices.Equal` for the remaining-notes comparison, table-driven row assertions in the render subtest.
- Readability: Good — the fixture helper's doc comment states exactly the property that makes it a fixture ("stamps running backwards against their stored order"), which is what the subtests depend on.
- Issues: None. The `// Query notes.` comment above the query is a restatement, but it matches the established rhythm of the surrounding block (`// Query tags.`, `// Query refs.`) and predates this change.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Ordinary ascending-stamp output is unchanged: `go test ./...` green with no existing test edited." — the no-edit half is settled: the diff of commit 82f7c489 is additions only across internal/cli/note_test.go and the one-line show.go change. The suite-green half needs `go test ./...` executed; reading the ascending-stamp assertions (note_test.go:576, :674) shows nothing that the loss of the `created ASC` key would alter, but that is an argument, not a run.
