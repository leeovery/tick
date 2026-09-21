TASK: free-text-round-trip-6-6 (tick-f9b5f3) — No Toon Or JSON Assertion Pins A Full-Output String

ACCEPTANCE CRITERIA:
1. No toon or JSON test compares a whole rendered document against a string literal, apart from the retained count-zero section headers (`FormatTaskList(nil)` vs `tasks[0]{id,title,status,priority,type}:` is one of them)
2. Format detection in `format_integration_test.go` works by decoding and checking a key rather than by matching a section header
3. The empty toon list is asserted by decoding to an empty `tasks` list
4. The empty JSON list is asserted by unmarshalling to a non-nil zero-length slice
5. The retained text assertions are exactly the count-zero section headers, each beside a decoded assertion of the same empty section
6. Pretty's golden strings are all present: `No tasks found.`, `No dependencies found.`, `No dependencies.`, the single transition line and the box-drawing cascade tree
7. `--quiet` bare-ID assertions and the bare-value byte assertions are unchanged
8. (Amended by the orchestrator) The README guard is `TestREADMESamplesMatchRenderedOutput`, exact comparison, deliberately exempt from the sweep; `TestREADMEToonSamplesDecode` still resolves and decodes its anchors
9. Both stated greps return nothing
10. `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT:
§11 ("Conformance Verification") takes an explicit trade: "**No byte-level pinning is kept in the machine formats.** Golden strings pin the exact output shape ... but they are the mechanism that rotted into the defect this work undoes. Decoded-value assertions survive harmless reformatting while still failing when a section goes missing or a value is wrong. That trade is taken for toon and JSON." The same section carves out pretty ("Pretty output has no parser, so a decoded-value assertion does not exist for it"), and §3.1/§9.2 exempt the bare value and the zero-byte selection, whose bytes are fixed rather than documents. §7.4/§8 require count-zero section headers with their column lists — a shape no decoder can recover, since `notes[0]{index,text,created}:` and `notes[0]:` decode identically. The task resolves that collision by keeping the count-zero headers as text beside decoded emptiness checks, and by listing the retained set so the boundary is inspectable.

IMPLEMENTATION:
- Status: Implemented
- Location: commit 09377b8d ("Tfree-text-round-trip-6-6 — no toon or json assertion pins a full-output string"), 10 test files. Key sites:
  - `internal/cli/toon_decode_test.go:157-172` — new `assertCountZeroSection(t, rendered, header)`: asserts the header verbatim and that the same key decodes to an empty list; `assertChangedHeader` (a count-N text pin) deleted, its two call sites dropped, `changedHeader` retained for the count-zero comparison at `internal/cli/toon_decode_test.go:353-358` which does sit beside its decoded check
  - `internal/cli/format_integration_test.go:238-247` — the `tasks[` substring probe replaced by `toonRows(t, decodeToonDoc(t, stdout), "tasks")` with per-row id/title assertions
  - `internal/cli/format_integration_test.go:556-562` — empty toon list now `assertToonRowsEmpty(t, decodeToonDoc(t, stdout), "tasks")`
  - `internal/cli/format_integration_test.go:575-582` and `internal/cli/json_formatter_test.go:54-60` — the `"[]"` text comparisons replaced by `assertRenderedJSONIsEmptyArray` (`internal/cli/json_formatter_test.go:1874-1887`), which unmarshals into `[]any`, rejects `null` explicitly and asserts zero length
  - `internal/cli/toon_formatter_test.go:1364-1391` — the 24-line golden task-detail blob replaced by `assertToonKeySet` + `assertToonFields` + `assertToonRelatedRow` + `assertToonStringList` + `assertToonNoteRows`, so no value coverage was lost
  - `internal/cli/toon_formatter_test.go:874-880, 898-907, 924-933, 956-967, 1146-1157` — line-by-line `dep_tree` edge text comparisons converted to `assertToonEdgeRows`, preserving row count and order
  - Count-zero headers now routed through the helper at `internal/cli/create_test.go:1414`, `internal/cli/detail_changes_test.go:79`, `internal/cli/format_test.go:415`, `internal/cli/list_show_test.go:936,1413`, `internal/cli/toon_formatter_test.go:163,165,216,217,689,1093,1106,1344`, `internal/cli/update_test.go:1423`
- Criterion-by-criterion:
  - (1) Swept `internal/cli/*_test.go` for whole-document comparisons three ways — equality against bracket/brace-bearing literals, raw-string literals, and `DeepEqual`/`cmp` — and the only survivors are the retained count-zero headers (`toon_formatter_test.go:95,107,887`, `toon_formatter_test.go:1042`, `toon_decode_test.go:353`). No toon or JSON test compares a whole rendered document to a literal.
  - (2) `internal/cli/format_integration_test.go:608-624` (non-TTY) and `:650-665` (`--toon` override) decode and assert `id` plus the presence of `notes`; the pretty branches at `:639-642` and `:678-681` keep their `ID:` substring check, as the plan required.
  - (3) and (4) met at the sites above; `toon_formatter_test.go:92-113` additionally decodes both empty-list branches beside their retained headers.
  - (6) Present: `No tasks found.` (`blocked_test.go:192,206,290`, `format_integration_test.go:569`, `list_show_test.go:126`), `No dependencies found.` (`dep_tree_test.go:321,334`), `No dependencies.` (`dep_tree_test.go:771`, `pretty_formatter_test.go:1086`), the single transition line (`pretty_formatter_test.go:338`, `transition_test.go:174,401`, `cascade_formatter_test.go:283`) and the box-drawing cascade tree (`cascade_formatter_test.go:79-83,103-109,132-136`, `helpers_test.go:327`).
  - (7) The commit touched none of the `--quiet` or bare-value assertions: `conformance_test.go:1187-1255` is untouched by 09377b8d, and the quiet assertions at `create_test.go:541,789,1485`, `update_test.go:1434`, `toon_refusal_test.go:336` are unchanged.
  - (8) `internal/cli/readme_samples_test.go` is not in the commit; `TestREADMEToonSamplesDecode` (`:164`) and `TestREADMESamplesMatchRenderedOutput` (`:425`) both stand, the latter exempt as the amendment states.
  - (9) Both stated greps return nothing (exit 1, no output) against the current tree.
- Notes: the retained non-count-zero text probes are wider than criterion 5's word "exactly", and deliberately so. Four survive, each asserting a rendered shape decoding provably cannot recover, and each sitting beside a decoded assertion of the same value: `toon_formatter_test.go:253-256` (the description renders as one quoted line, beside `assertToonFields` at `:257`), `toon_formatter_test.go:607-609` (the §6.1 inline single-item refs list, beside `assertToonStringList` at `:610`), and `json_formatter_test.go:839-841` and `:1452-1459` (two-space JSON indentation and absence of tabs). Three of the four are enumerated in the task's own fix-tracking record (`.workflows/free-text-round-trip/implementation/free-text-round-trip/fix-tracking-free-text-round-trip-6-6.md`, NOTES), so the boundary stayed inspectable, which is what the criterion exists to guarantee. Judged sound, not a loss: deleting any of them would leave a shape §6.1/§6.2 requires with nothing checking it.

TESTS:
- Status: Adequate
- Coverage: this task is test-only, so the tests are the deliverable. Every converted assertion still fails on the behaviour it names: `decodeToonDoc` fatals on a document that will not parse; `toonRows` fatals on a missing key; `assertToonEdgeRows` holds row count and order; `assertToonFields` holds each value; `assertRenderedJSONIsEmptyArray` distinguishes `[]` from `null`, which is exactly what the old `!= "[]"` comparison was there for. The quoting tests keep their teeth after conversion — a comma or newline emitted unquoted would either fail to decode or land in the wrong column, so `toon_formatter_test.go:321-331` (`"Setup, Deploy"`) still catches the defect class that motivated the work.
- Notes: the plan's micro-acceptance names map onto existing subtests rather than new ones with those literal names (e.g. "it tells toon from pretty by a decoded key" is carried by `"non-TTY defaults to toon"` / `"--toon overrides TTY default"`), which is the right call — inventing parallel subtests would have duplicated coverage. Two narrowings were taken knowingly and are recorded in the fix-tracking file: `list_show_test.go:936` and `:1413` no longer pin that a `--field` toon document ends in a single newline (§9.2 fixes that byte for the bare value only, so nothing requires it for the document form), and `helpers_test.go:299-312` no longer pins that exactly one section is written. Neither names a behaviour the spec requires, and neither subtest's stated purpose is now unobserved. No over-testing: the largest conversion (`toon_formatter_test.go:1364-1391`) overlaps `TestToonTaskDetailConformance`, but it is the only place holding the unfiltered document's full key set against the filtered path, so it earns its keep.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on every new helper, "it does X" subtest naming, helpers living in `toon_decode_test.go`/`json_formatter_test.go` beside their kin
- SOLID principles: Good — `assertCountZeroSection` has one job and derives the section key from the header via `strings.Cut`, so callers state the required shape once instead of writing the contains-plus-decode pair by hand at fourteen sites
- Complexity: Low
- Modern idioms: Yes — `strings.Cut`, `slices.Equal`, `any` maps over the decoded document
- Readability: Good. The `assertRenderedJSONIsEmptyArray` name was chosen over the near-collision with the pre-existing `assertJSONEmptyArray(doc, key)`, which the fix-tracking record flagged; the two now read distinctly at their call sites
- Comment accuracy: the one new comment (`internal/cli/toon_decode_test.go:157-160`) states why the column list can only be checked as text, and it holds — decoding collapses `notes[0]{index,text,created}:` and `notes[0]:` to the same empty list. No process-artifact references in the changed code
- Issues: none

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settling this needs the four commands run; reading confirms only what precedes them: imports stay consistent with use after the sweep (`strings` still used in all ten touched files, `slices` still used in `toon_formatter_test.go` after the `slices.Equal` deletion at the old `:1005`), the deleted `assertChangedHeader` has no remaining callers, and `changedHeader` still has one consumer at `toon_decode_test.go:353`, so nothing is left dangling that would fail a build or a lint pass.
