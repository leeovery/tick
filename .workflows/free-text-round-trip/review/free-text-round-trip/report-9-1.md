TASK: free-text-round-trip-9-1 (tick-6b8ce7) — The Toon Encoder's Refusal Emits A Document That Denies The Data

ACCEPTANCE CRITERIA:
- With a C0 control character other than tab, newline or carriage return stored in any free-text carrier, no toon document is printed that omits or under-counts it: `tick show`, `tick list`, `tick dep tree <id>`, `tick create`, `tick update`, `tick note add`, `tick note remove` and the four transition commands exit 1 with a stderr diagnostic and write nothing to stdout.
- The diagnostic names the field or section that could not be encoded, and names the task's ID for every document that covers a single task.
- `tick stats` and the full-graph `tick dep tree` carry no free text — integers, IDs and the fixed priority table only — and their output is byte-unchanged.
- `--json` and `--pretty` over the same data are byte-unchanged: both exit 0 and carry the value.
- `--quiet` paths exit 0 and print IDs as they do today, since no document is encoded on them.
- `tick show <id> --field title|description|notes.1` returns a refused value bare and byte-identically at exit 0.
- A mutation whose document is refused still commits.
- A section that genuinely holds no rows still renders its `name[0]{cols}:` header, and a detail document narrowed to a selection that keeps no field still exits 0 printing nothing.
- `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass, the conformance inventory included, and no test asserts a substituted document.

STATUS: issues_found

SPEC CONTEXT:
§1 states the failure class this work exists to delete: toon output an agent cannot trust, which sends it back to `.tick/tasks.jsonl`. §2.2 sets byte-identity as the fidelity bar on the read-back path, and declines an announced substitution rather than write an exception down. §11 makes every structured command's output decodable by a real TOON reader and keeps a permanently round-tripped awkward fixture. A silently dropped section or a count-zero header over live rows is exactly the shape §1 and §2.2 forbid — it parses, so §11's decoders pass on it, while the document denies the data.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/toon_formatter.go:266-283` — `encodeToonFields` returns `("", *toonEncodeError)` via `fieldRefusal`, which re-marshals each field alone to name the one the library error does not.
  - `internal/cli/toon_formatter.go:334-376` — `encodeToonSection` / `encodeToonSectionIdentified` / `sectionRefusal`: the substituted `fmt.Sprintf("%s[0]:", name)` is gone; the refusal names the section and, where an identity is supplied, the refused row's task ID.
  - `internal/cli/toon_formatter.go:378-402` — `toonEncodeError` (part, taskID, wrapped err) plus `refusalForTask`, which attributes a refusal that names no task of its own to the document's subject.
  - `internal/cli/toon_formatter.go:404-422` — `toonDoc.add`/`join`: sections are collected, the first refusal short-circuits the rest, and `join(taskID)` attributes it.
  - All fourteen encoder-helper lines carry the error (`grep -n 'encodeToonFields(\|encodeToonSection(\|encodeToonSectionIdentified(' internal/cli/toon_formatter.go` → 67, 87, 91, 103, 113, 127, 155, 186, 198, 222, 260, 305, 322, 340, plus the three definitions at 266/336/340): `buildTaskSection`, `buildRelatedSection`, `buildNotesSection`, `buildChangedSection`, `buildEdgeSection`, `FormatTaskList`, `FormatTaskDetail`, `FormatStats`, `FormatCascadeTransition`, both dep-tree builders.
  - `internal/cli/format.go:264-286` — the five document-building `Formatter` methods take a second return value; `StubFormatter`, `PrettyFormatter` and `JSONFormatter` follow with `nil`.
  - Seven print sites turned into failures via the new `printDocument` (`internal/cli/helpers.go:34-41`): `list.go:228-229`, `show.go:84-92`, `helpers.go:30-31` (`outputMutationResult`, behind `create.go:278`, `update.go:398`, `note.go:87`, `note.go:145`), `helpers.go:109-112` (`outputStatusChanges`, now returning an error, handled at its single caller `transition.go:58`), `dep_tree.go:38-39`, `dep_tree.go:56-57`, `stats.go:93-94`. Each returns to `App.Run`, which prints `Error: %s` to stderr and returns 1 (`app.go:152-159`).
  - `README.md:494` — the paragraph naming the uncarryable characters, the failure mode and the `--json` / `--field` escape hatches.
- Notes:
  - The distinction the task asked to preserve is preserved: `emptyToonSection` (`toon_formatter.go:326-334`) still emits `name[0]{cols}:` for a genuinely empty section, and `buildTaskSection` still returns `("", nil)` when a selection keeps no top-level field, which `joinToonSections` (`:286-294`) drops. `grep -rn '\[0\]:' internal/cli --include='*.go'` returns nothing — no substituted form survives anywhere.
  - Divergence, judged sound: where a refused value sits on a *related* task (a child, a blocker, a cascaded row), the diagnostic names that task rather than the document's subject, and `toon_refusal_test.go` asserts the subject is left unnamed. The acceptance criterion's wording asks for "the task's ID" of a single-task document; what ships names the task carrying the refused value, which is the actionable one, is what `README.md:494` documents, and is the shape tasks 9-2 and 10-2 refined deliberately. No loss — the caller already knows which task it asked for.
  - Divergence, judged sound: the refused character is pinned in a dedicated `TestRefusedCharacterRoundTrip` with its own constants rather than by adding a refused character into `fixtureTitle`/`fixtureDescription`/`fixtureNoteText` as the Do list's wording suggested. Folding it into the shared fixture would have made every existing round-trip document assertion refuse. The outcome the task named — round-trip assertions pinning stored byte-identity, bare `--field` byte-identity and a non-zero exit on the document path, for all three carriers — is delivered (`round_trip_test.go:380-455`).
  - The bare `--field` path is reached before any encoding (`show.go:73-77` → `bareFieldValue`, `show_fields.go:118-131`), so a refused value still returns bare at exit 0; `--field notes` with no position falls through to the document path and refuses, closing the "count-zero notes section that `--field notes` agreed with" hole named in the task's Problem.
  - Mutations commit before rendering on every path (`create.go`, `update.go`, `note.go`, `transition.go:35-58`), so a refused document leaves the record written.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_refusal_test.go` carries all fourteen named micro-acceptances plus the breadth cases: a ten-command table (`refusedDocumentCommands`, `:88-114`) covering show, list, dep tree, update, note add, note remove and the four transitions, each asserting exit 1, empty stdout and a diagnostic naming the task; `ready`/`blocked` alongside `list`; create-still-stores; note-still-stored; bare `--field`; `--json`; `--pretty`; `--quiet` across show/list/update/start; and stats / full-graph dep tree byte-compared between a project holding a refused title and one that does not.
  - `assertRefused` (`:59-71`) pins all three halves of the contract at once — exit 1, empty stdout, named strings — so a regression to silent omission fails on exit code, and a regression to a substituted document fails on stdout.
  - Unit level: `toon_formatter_test.go:844-855` and `:857-868` assert the encoder's error comes out of `encodeToonFields`/`encodeToonSection` naming the field/section; `:870-879` covers the `sectionRefusal` fallback where the section refuses but no single row does; `:881-889` and `:891-903` pin that the empty-section header and the omitted head survive unchanged.
  - `round_trip_test.go:429-455` pins stored byte-identity, bare `--field` byte-identity and document refusal for each of the three carriers.
  - `formatted(t).of(...)` (`format_test.go:492-505`) converts the new error into a test failure everywhere a formatter is called directly, so no existing assertion can silently swallow a refusal.
- Notes:
  - Some overlap between the plan's named `show`/`list`/`dep tree` tests, the ten-command stdout table and `TestRefusedCharacterRoundTrip`'s per-carrier `show` case. Each has a distinct job (named micro-acceptance / command breadth / round-trip contract per carrier), so this reads as deliberate breadth rather than redundancy — not reported as over-testing.
  - Ordering in "it names the first offending task when two tasks in the list carry refused values" is fixed by priority 1 vs 3 against `ORDER BY ... t.priority ASC` (`list.go:317-319`), so the test is deterministic.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` only, `t.Run()` "it does X" subtests, `t.Helper()` on helpers, `t.TempDir()`-based project setup, `fmt.Errorf`-style wrapping via `Unwrap` on the new error type, DI untouched.
- SOLID principles: Good. `toonDoc` isolates the collect-and-short-circuit concern from every document builder; `printDocument` isolates the print-or-fail decision from all seven call sites; `refusalForTask` isolates attribution. `PrettyFormatter.FormatCascadeTransition` correctly splits into an error-returning interface method and an internal `cascadeTransition`, so `FormatTaskDetail`'s internal use does not have to discard an error it cannot get.
- Complexity: Low. The error paths are linear; the extra `MarshalString` passes in `fieldRefusal`/`sectionRefusal` run only after a refusal has already occurred.
- Modern idioms: Yes. Multi-value call forwarding (`doc.add(buildTaskSection(...))`), generics on `encodeToonSectionIdentified`/`emptyToonSection`, `errors.As` for attribution, `reflect.TypeFor`.
- Readability: Good. Error messages carry both facts an agent needs — `cannot encode field title of task tick-xxxxxx as TOON: toon: unsupported control character U+001B in string`.
- Issues: One inaccurate fixture label, below — nothing in the production code.

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/round_trip_test.go:30 — `refusedTitle`, `refusedDescription` and `refusedNoteText` (`:30-32`) spell their own payload "bell" while `refusedChar` (`:29`) is `"\x1b"`, ESC (U+001B), not BEL (U+0007); the same wrong word is copied into `toon_formatter_test.go:846`, `:859` and `:873`. Replace "bell" with "escape" in all six literals. — FAILS: the fixtures state a character they do not carry, so a maintainer reading a refusal failure (whose diagnostic says `U+001B`) or grepping for the BEL case is sent looking for a character no test uses.

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass, the conformance inventory included, and no test asserts a substituted document." — the second half is settled by reading (`grep -rn '\[0\]:' internal/cli --include='*.go'` returns nothing, and every direct formatter call in the suite routes through `formatted(t).of`, which fails on a refusal); the three commands themselves need an executing pass to settle.
