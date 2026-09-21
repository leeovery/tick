TASK: free-text-round-trip-9-2 (tick-ccc8ee) — A Task-List Refusal Names The Task That Caused It

ACCEPTANCE CRITERIA:
1. `tick list` over a project where one task's title carries `\x1b` exits 1, writes nothing to stdout, and its stderr names both the `tasks` section and that task's ID; `tick ready` and `tick blocked` behave identically through the shared call site.
2. The diagnostic names the offending task alone — the ID of an encodable task in the same list never appears in stderr.
3. Two refused tasks in one list name the first in row order, matching `fieldRefusal`'s first-failure rule.
4. Where the section refuses but no single row refuses on its own, the message is byte-identical to today's `cannot encode section tasks as TOON: …`.
5. The per-row re-marshal is reachable only from the section encoder's error branch: a list of encodable tasks marshals the section exactly once.
6. Every other refusal diagnostic is unchanged — `show`, `dep tree`, `note add`, `note remove`, `start`, `done`, `cancel`, `reopen` still name the section or field plus the document's subject, and the cascade subtest still passes with the parent named and the cascaded child absent.
7. An empty task list and `list --quiet` are untouched: neither reaches the section encoder.
8. `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied.

STATUS: complete

SPEC CONTEXT:
§3.1 ("Output that must parse", specification.md:60-64) requires a value TOON cannot carry to fail the command with a diagnostic "naming the field or section it could not encode and the task that carries the refused value", with "A task list names the offending row, not merely the section — an error that tells an agent only that a project-wide command failed leaves it bisecting the project or reading `.tick/tasks.jsonl`". "Where the offending row cannot be identified, the section name alone stands." The corrigendum of 2026-09-20 at specification.md:565 is the record of that amendment and names this exact defect — `tick list`/`ready`/`blocked` failing with the section name alone — and the precedent it follows: the field-level path re-encoding each field alone because the library error names none. The amended sentence is unqualified (the task carrying the value, not the document's subject), which is why the later task 10-2 legitimately reversed step 4 of this task (see IMPLEMENTATION notes).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/toon_formatter.go:345-352` — `encodeToonSectionIdentified(name, rows, identity)`; the refusal branch (`:349`) is the only caller of `sectionRefusal` in non-test code.
  - `internal/cli/toon_formatter.go:357-369` — `sectionRefusal`: builds `&toonEncodeError{part: "section " + name, err: err}`, returns it unchanged when `identity == nil`, otherwise re-marshals `[]T{row}` per row and sets `taskID` from the first row that fails alone, breaking there.
  - `internal/cli/toon_formatter.go:339-341` — `encodeToonSection(name, rows)` keeps its signature and delegates with a nil identity, so the identity-less call sites (`:87`, `:91`, `:127`, `:222`, `:322`) and the unit test at `internal/cli/toon_formatter_test.go:858-869` are untouched in behaviour.
  - `internal/cli/toon_formatter.go:67` — `FormatTaskList` supplies `func(r toonTaskRow) string { return r.ID }`; `list`, `ready` and `blocked` all reach it through `internal/cli/list.go:228` (`RunList` is called from `internal/cli/app.go:192`, `:223`, `:236`).
  - `internal/cli/toon_formatter.go:373-384` — identity is carried in `toonEncodeError.taskID`, never folded into `part`; the message becomes `cannot encode section tasks of task <id> as TOON: …` and stays section-only when `taskID` is empty.
- Notes:
  - Ordering behind criterion 3 holds: `internal/cli/list.go:317-319` orders `priority ASC, created ASC` (with an `in_progress` band first for `ready`), so the priority-1 task the test creates is row 0 and `sectionRefusal`'s break-on-first yields it.
  - Criterion 6's second clause ("the parent named and the cascaded child absent") no longer holds, and correctly so: task 10-2 ("A Refusal Names The Task Carrying The Refused Value, Not The Document's Subject") explicitly supersedes step 4 of this task and that criterion, on the grounds that §3.1 was amended afterwards and scopes task-naming to the task carrying the value. `refusalForTask` (`internal/cli/toon_formatter.go:390-396`) now overrides only an anonymous refusal, and `buildRelatedSection`/`buildChangedSection` (`:305`, `:155`) carry row identity. That is a later, recorded, sound divergence, not a loss — not reported as a finding.
  - Criterion 6's first clause still holds for the commands it names: each `refusedDocumentCommands()` entry's document covers the task carrying the refused value, and the table still asserts `env.id` appears (`internal/cli/toon_refusal_test.go:220-226`).
  - Criterion 7 holds by reading: `FormatTaskList` returns the count-zero header before the encoder for an empty list (`internal/cli/toon_formatter.go:54-56`), and `RunList` returns from the quiet branch before formatting (`internal/cli/list.go:221-226`).
  - Criterion 5 holds by reading: `sectionRefusal` has exactly one non-test call site, inside `if err != nil` at `internal/cli/toon_formatter.go:349`; a healthy section marshals once.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_refusal_test.go:156-165` — `list` over an encodable task plus a refused one: asserts exit 1, empty stdout, `"tasks"` and the refused ID in stderr, and `assertUnnamed` for the encodable sibling (criteria 1 and 2). The encodable task is created first, so the row the hunt must skip actually precedes the culprit.
  - `internal/cli/toon_refusal_test.go:167-176` — two refused tasks at priorities 1 and 3: first named, second unnamed (criterion 3).
  - `internal/cli/toon_refusal_test.go:178-190` — the same assertion driven through `ready` and `blocked` (criterion 1's second half), with `blockAgainstOrdinaryTask` first proving the dependency landed via `blocked --quiet`.
  - `internal/cli/toon_formatter_test.go:871-880` — the section-only fallback: `sectionRefusal` over two encodable rows with an identity produces a message byte-identical to the identity-less refusal (criterion 4). Calling the unexported helper directly is the only way to reach this branch — no pair of rows refuses jointly but not singly through the public path — so it is warranted rather than a test of internals.
  - `internal/cli/toon_refusal_test.go:211-229` — the per-command table now asserts `"cannot encode"`, `env.id` and, for `list`, the `tasks` section, so the list row expects the ID as the task required.
  - Pre-existing branches stay pinned: `internal/cli/toon_formatter_test.go:858-869` (identity-less `encodeToonSection`), `internal/cli/toon_refusal_test.go:192-200` (notes section falls back to the subject), `:318-340` (`list --quiet` still exits 0 printing IDs).
- Notes: No redundancy worth flagging — the `ready`/`blocked` loop repeats the list assertion deliberately, since criterion 1 names the shared call site as the thing under test. No mocking or superfluous setup.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` with `t.Run` subtests in "it does X" form, `t.Helper()` on every helper, `t.TempDir()` isolation via `setupTickProject`, error wrapping through `*toonEncodeError`'s `Unwrap`, generics used in the existing style of `encodeToonSection`/`emptyToonSection`.
- SOLID principles: Good. `sectionRefusal` is one responsibility (attribute a refusal to a row) and is the section-level twin of `fieldRefusal` (`internal/cli/toon_formatter.go:276-283`); identity is injected as a function rather than constraining `T`, so scalar sections (`tags`, `refs`) still compile through the same encoder.
- Complexity: Low. One loop with an early break on the error path; the happy path gains one function call and no work.
- Modern idioms: Yes. Generic helper, `errors.As` in `refusalForTask`, no reflection added.
- Readability: Good. The comments at `:343-344` and `:354-356` state the rule and the fallback without restating the code, and neither references a task id, phase or spec section.
- Issues: None. `gofmt -l` reports nothing for `internal/cli/toon_formatter.go`, `internal/cli/toon_refusal_test.go` and `internal/cli/toon_formatter_test.go`.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied." — the gofmt half is settled (`gofmt -l` lists none of the three changed files); the rest needs `go test ./...`, `go vet ./...` and `golangci-lint run ./...` executed over the tree, which reading cannot settle. The behavioural criteria above were judged by reading the code path and the assertions that pin it; a suite run is what confirms those assertions actually pass.
