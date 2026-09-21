TASK: free-text-round-trip-10-2 (tick-9e5458) — A Refusal Names The Task Carrying The Refused Value, Not The Document's Subject

ACCEPTANCE CRITERIA:
1. `tick --toon show <parent>` with a child's title carrying `\x1b` exits 1, writes nothing to stdout, names the `children` section and the child's ID; the parent's ID appears nowhere.
2. `tick --toon show <id>` with a blocker's refused title names the `blocked_by` section and the blocker's ID, not the subject's.
3. `tick --toon done <parent>` with a cascaded child's refused title names the `changed` section and the child's ID, not the parent's (supersedes phase 9 task 1 step 4 and its criterion).
4. A refusal with no row identity still names the document's subject byte-identically (notes section; a refused `title`/`description` on the subject).
5. Task-list refusals unchanged — `list`, `ready`, `blocked` name `tasks` and the first offending row.
6. Every command in `refusedDocumentCommands()` still names `env.id`.
7. The per-row re-marshal stays on the error path; a document whose sections all encode marshals each section exactly once.
8. Diff confined to `internal/cli/toon_formatter.go` and `internal/cli/toon_refusal_test.go`; no README edit, no `.workflows/` edit.
9. `go test ./...`, `go vet ./...`, `golangci-lint run ./...` clean, `gofmt` applied.

STATUS: complete

SPEC CONTEXT: §3.1 (specification.md:62) requires a command whose TOON document would carry an unencodable C0 control character to "exit non-zero with a diagnostic naming the field or section it could not encode and the task that carries the refused value", writing nothing to stdout, with "Where the offending row cannot be identified, the section name alone stands." The corrigendum of 2026-09-20 (specification.md:565) replaced the earlier "the task where the document covers one" wording precisely because scoping task-naming to single-task documents leaves an agent bisecting the project. The implementation follows the amended, governing sentence; the superseded phase-9 direction is named in the task's own text, so the departure is recorded rather than silent.

IMPLEMENTATION:
- Status: Implemented (all five Do steps, commit 1539c58c, two files, 32 insertions / 10 deletions)
- Location:
  - `internal/cli/toon_formatter.go:305` — `buildRelatedSection` now calls `encodeToonSectionIdentified(name, rows, func(r toonRelatedRow) string { return r.ID })`, giving both `blocked_by` and `children` a row identity in every detail document.
  - `internal/cli/toon_formatter.go:155` — `buildChangedSection` does the same with `func(r toonChangedRow) string { return r.ID }`, covering both builders of the table: `FormatCascadeTransition` (`:159-162`) and `FormatTaskDetail` (`:97-99`).
  - `internal/cli/toon_formatter.go:390-396` — `refusalForTask` overrides only an anonymous refusal: `if taskID == "" || !errors.As(err, &refusal) || refusal.taskID != ""` returns `err` untouched; the empty-`taskID` case keeps the subject-naming rewrite. Both call sites keep their arguments (`:161`, `:419`).
  - `internal/cli/toon_formatter.go:371-372, 388-389` — the two doc comments were updated with the behaviour ("the task carrying it when one is identified"; "names no task of its own; one that does keeps it") and hold against the code.
- Notes: Traced end to end for each criterion.
  - AC1/AC2: `FormatTaskDetail` adds `blocked_by` then `children`; the first failing section short-circuits `toonDoc.add`, `doc.join(detail.Task.ID)` calls `refusalForTask`, which now returns the identified refusal untouched, so `toonEncodeError.Error()` (`:379-384`) renders `cannot encode section children|blocked_by of task <row-id> as TOON: …`. `RunShow` returns that error unwrapped (`internal/cli/show.go:84-87`), `App.Run` prints `Error: %s` to stderr and returns 1 (`internal/cli/app.go:118-119` and the shared tail), and `printDocument` (`internal/cli/helpers.go:36-42`) returns before writing, so stdout stays empty. The subject ID appears in no part of the message.
  - AC3: `CascadeResult.Changed()` (`internal/cli/format.go:209-227`) emits the primary row then the cascaded rows, `statusChangeSet.rows()` (`internal/cli/transition.go:139-147`) preserves each row's original ID, so `sectionRefusal` (`:357-369`) attributes the refusal to the cascaded child, and `FormatCascadeTransition` no longer rewrites it to `result.TaskID`.
  - AC4: `buildNotesSection` (`:317`) and `encodeToonFields`/`fieldRefusal` (`:264-284`) still produce refusals with an empty `taskID`, so the subject-naming branch of `refusalForTask` is reached unchanged; `tags`, `refs`, `by_priority` and the edge sections also still route through the nil-identity `encodeToonSection` (`:339-341`).
  - AC5/AC6: `FormatTaskList` (`:67`) is untouched; for every command in `refusedDocumentCommands()` the refused value sits on the subject itself, so either the subject-field path or a `changed`/`tasks` row whose identity *is* `env.id` names it.
  - AC7: `sectionRefusal` is called only inside the `err != nil` branch of `encodeToonSectionIdentified` (`:348-350`); `sectionRefusal` and `encodeToonSectionIdentified` have no other non-test call sites (four production call sites: `:67`, `:155`, `:305`, `:340`). A document that encodes marshals each section once.
  - AC8: `git show --stat 1539c58c` lists exactly `internal/cli/toon_formatter.go` and `internal/cli/toon_refusal_test.go`. `README.md:494`'s paragraph already states §3.1's rule ("the task carrying the refused value — the offending row, where the document is a task list") and needed no edit; nothing under `.workflows/` was touched, consistent with §3.1 as amended.
  - No drift from the plan: the implementation is step-for-step what the task prescribed, and the direction it reverses is the one the task explicitly supersedes with grounds.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_refusal_test.go:231` — rewritten to "it names the cascaded task carrying the refused value": asserts `"cannot encode section changed"` plus the child, and `assertUnnamed(t, stderr, parent)` (AC3).
  - `:242` — "it names the child when a child's title cannot be encoded": `show <parent>` names `"section children"` plus the child, parent unnamed (AC1; `assertRefused` also pins exit 1 and empty stdout).
  - `:253` — "it names the blocker when a blocker's title cannot be encoded": `dep add` then `show <blocked>` names `"section blocked_by"` plus the blocker, subject unnamed (AC2).
  - Untouched branches stay pinned: `:147` (field on the subject), `:192` (notes fallback to the subject), `:156`/`:167`/`:178` (task list, first offending row, `ready`/`blocked`), `:211` (the `refusedDocumentCommands()` table, each still asserting `env.id`).
  - Each new subtest would fail if the feature broke: reverting either builder to `encodeToonSection` leaves `taskID` empty and `refusalForTask` rewrites to the subject, failing both the "want it to name" and the `assertUnnamed` assertions.
- Notes: Not over-tested — three subtests for three distinct producers (`changed`, `children`, `blocked_by`), each the only coverage of its path; setup reuses the existing `createChildCarryingRefusedTitle` (`:137`) and `createRefusedTitleTask` (`:13`) helpers rather than duplicating fixtures. No unit test asserts the per-row hunt stays off the happy path (AC7), but the structure settles it by reading and `toon_formatter_test.go:870-880` already exercises `sectionRefusal` directly for the no-single-row fallback.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing`, `t.Run` subtests with "it does X" names, `t.Helper()` on helpers, `--toon` via `runToon`; error values unwrapped on the way out; the formatter keeps its one-section-per-builder shape.
- SOLID principles: Good. The identity hunt stays inside `sectionRefusal`; builders only supply the projection from row to task ID.
- Complexity: Low. The change is two call swaps and one added disjunct in an existing guard.
- Modern idioms: Yes. Generic `encodeToonSectionIdentified` with a `func(T) string` projection; `errors.As` for the typed refusal.
- Readability: Good. `refusalForTask`'s three-part guard reads as "no subject to name, not a refusal, or already named".
- Comment accuracy: The three comments the commit touched (`:371-372`, `:388-389`, and `sectionRefusal`'s at `:354-356`) each describe what the code now does; no process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied." — needs the three commands run over the current tree; reading confirms the diff is gofmt-shaped and introduces no unused symbol, but clean suite/vet/lint exit codes cannot be established by reading.
