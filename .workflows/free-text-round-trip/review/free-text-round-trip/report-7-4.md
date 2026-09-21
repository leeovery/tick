TASK: free-text-round-trip-7-4 (tick-02d03b) — Each Count-Zero TOON Header Derives Its Columns From The Row Struct

ACCEPTANCE CRITERIA:
- `grep -n '\[0\]{' internal/cli/toon_formatter.go` matches only the helper's format string; the five hand-written column lists are gone.
- Each of the five call sites passes the same row type its populated branch marshals.
- Every count-zero header is byte-identical to before: `tasks[0]{id,title,status,priority,type}:`, `changed[0]{id,title,from,to,auto}:`, `dep_tree[0]{from,to}:`, `blocked_by[0]{from,to}:`, `blocks[0]{from,to}:`, `blocked_by[0]{id,title,status}:`, `children[0]{id,title,status}:`, `notes[0]{index,text,created}:`.
- No test file, no README sample and no other production file is edited.
- `go test ./...`, `go vet ./...` and `golangci-lint run ./...` all clean.

STATUS: complete

SPEC CONTEXT: §7.4 (specification.md:303) requires the `changed` section to be always present, carrying `changed[0]{id,title,from,to,auto}:` when nothing moved, "exactly as an empty notes or children section carries its count-zero header (§8)"; §8 (:324) requires the emptied document to take the shape of the populated one; §7.2 is the branch-on-which-document-arrived defect the count-zero header exists to delete; §5.2/:428 requires `--field notes` on a task with no notes to return `notes[0]{index,text,created}:`. A count-zero header whose columns disagree with the populated branch's columns is exactly the two-schema state those sections exist to prevent, so deriving both from one struct serves the spec's intent directly.

IMPLEMENTATION:
- Status: Implemented
- Location: helper at internal/cli/toon_formatter.go:325-334; call sites at :55 (`emptyToonSection[toonTaskRow]("tasks")`), :149 (`[toonChangedRow]("changed")`), :220 (`[toonEdgeRow](name)`), :299 (`[toonRelatedRow](name)`), :312 (`[toonNoteRow]("notes")`). Commit e0b40a83 touches internal/cli/toon_formatter.go only (17 insertions, 5 deletions).
- Criterion 1: settled. `grep -rn '\[0\]{' --include='*.go'` returns one production match, the helper's format string at toon_formatter.go:333; every other match is a test file. No production file outside toon_formatter.go holds a count-zero literal.
- Criterion 2: settled. Each call site passes the row type its own populated branch marshals: FormatTaskList marshals `[]toonTaskRow` (:57-67), buildChangedSection `[]toonChangedRow` (:151-155), buildEdgeSection `[]toonEdgeRow` (:222, rows produced by `toonEdgeRows` at :209-215), buildRelatedSection `[]toonRelatedRow` (:301-305), buildNotesSection `[]toonNoteRow` (:314-322). `grep -rn 'emptyToonSection'` shows exactly five call sites and one definition.
- Criterion 3: settled by reading the tags the helper reflects over — toonTaskRow (:24-30) id,title,status,priority,type; toonRelatedRow (:33-37) id,title,status; toonNoteRow (:40-44) index,text,created; toonChangedRow (:138-144) id,title,from,to,auto; toonEdgeRow (:165-168) from,to. `strings.Cut(tag, ",")` yields the whole tag for these comma-free tags, so all eight named headers render byte-identical, including the three `toonEdgeRow` names passed through `buildEdgeSection` (`dep_tree` at :185, `blocked_by` at :203, `blocks` at :204).
- Criterion 4: settled. `git show --stat e0b40a83` lists internal/cli/toon_formatter.go as the only file. The test literals the task asked to leave standing are intact (toon_formatter_test.go:95, :107, :1042-1043 and the assertCountZeroSection callers), and README.md:482 still carries `children[0]{id,title,status}:`.
- Notes: the helper walks every field of T rather than only toon-tagged exported ones, while toon-go's encoder (internal/codec/structmeta.go:30-55) skips unexported fields and `toon:"-"`, and falls back to the Go field name for an untagged field. Every field of all five row types is exported and toon-tagged, so the two derivations agree today, and an untagged field added later would produce a visibly empty column that the unedited literal pins in toon_formatter_test.go and the assertCountZeroSection callers fail on immediately — the pin the task was built to buy. Recorded as context, not reported as a finding: nothing is currently wrong and nothing here is actionable.

TESTS:
- Status: Adequate
- Coverage: the task is a behaviour-preserving refactor and correctly adds no test. The unedited literals are the independent pin: "it formats zero tasks as empty section" (toon_formatter_test.go:91) and its nil-slice twin (:103) compare full output against `tasks[0]{id,title,status,priority,type}:`; "it renders the emptied full document for a result with no trees" (:1040-1046) pins `dep_tree[0]{from,to}:` as the first line; the assertCountZeroSection callers pin `children`/`blocked_by` (:163, :216, :217), `notes` (:165, :689, :1344, list_show_test.go:936, :1413), `blocked_by`/`blocks` edge headers (:1093, :1106) and `changed` (create_test.go:1414, update_test.go:1423, detail_changes_test.go:79, format_test.go:415). assertCountZeroSection (toon_decode_test.go:161-175) compares the header as text and then asserts the section decodes to zero rows, so a wrong column list fails on the text comparison — the derived production header cannot drift without one of these failing.
- Notes: no over-testing introduced; the helper gets no direct unit test, which is right — testing it in isolation would restate the reflection rather than the rendered header the callers already pin.

CODE QUALITY:
- Project conventions: Followed. Generic helper sits beside encodeToonSection as the task directed, doc comment states what the header is rather than restating the loop, `reflect.TypeFor[T]()` is the current API (go 1.26 in go.mod), and `for i := range cols` is the modernize-preferred integer range form already used at :124.
- SOLID principles: Good. One reason to change — the column list now lives only in the row struct, and both branches of each section read it.
- Complexity: Low. One loop, no branching.
- Modern idioms: Yes. Type parameter over `reflect.TypeOf(new(T)).Elem()`, `strings.Cut` over `strings.Split`, blank-discard of Cut's unused returns.
- Readability: Good. The call sites read as the schema they render.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` all clean." — needs the toolchain run; reading settles that no test file was edited by the commit and that each pinned literal matches the tags the helper derives from, but not that the suite, vet and the linter pass over the delivered tree.
