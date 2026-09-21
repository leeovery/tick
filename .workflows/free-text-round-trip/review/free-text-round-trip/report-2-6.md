TASK: free-text-round-trip-2-6 (tick-477df7) — README Transition And Cascade Samples Match Real Output

ACCEPTANCE CRITERIA:
- Both toon samples were copied from real tool output and decode via `toon.DecodeString`
- The simple-transition toon sample is a one-row `changed` table carrying the task's title and `auto` reading `false`
- The cascade toon sample is one table whose count matches its rows, the requested row first with `auto` `false` and each cascaded row `true`
- The JSON sample is a single object whose only key is `changed`, with `auto` as an unquoted boolean
- The pretty samples match real pretty output, and the cascade one is still a box-drawing tree
- `grep -n '(unchanged)' README.md` returns nothing
- No sample is labelled as shared between TOON and Pretty
- `TestREADMEToonSamplesDecode` covers both new anchors and fails if either block is removed or made unparseable
- The `tick show`, `tick list` and dep-tree samples are unchanged by this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§7.2 fixes the status-change shape as one `changed[N]{id,title,from,to,auto}:` table carrying the title, the requested row first with `auto` false and each knock-on true, the old trailing ` (auto)` marker folded into the column. §7.6 declines to reinstate the unchanged-terminal-children row (`(unchanged)`), leaving it to the `auto-cascade-parent-status` document. §4.2 moves JSON with toon — the `changed` list replaces the `transition`/`cascaded` pair. §4.1 keeps pretty exactly as it is: the bare arrow line with no title (adding the title was declined explicitly) and the nesting cascade tree (flattening it to the table shape was declined explicitly). §12.1 treats the README as live documentation, so a sample describing output the tool does not produce is a shipped defect.

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:506-580 (Transition & Cascade Output), internal/cli/readme_samples_test.go:28-29, 127-233
- Notes:
  - Simple transition, three separately labelled cells: TOON (README.md:514-519) `changed[1]{id,title,from,to,auto}:` / `tick-a1b2,Setup auth,open,in_progress,false`; Pretty (README.md:524-528) `tick-a1b2: open → in_progress`; JSON (README.md:533-546) `{"changed":[{"id","title","from","to","auto"}]}` with `"auto": false` unquoted. Field order matches `jsonStatusChange` (internal/cli/json_formatter.go:293-299) and the 2-space indent matches `marshalIndentJSON` (json_formatter.go:318).
  - Cascade, two cells: TOON (README.md:558-564) `changed[2]{…}:` with `tick-a1b2,…,in_progress,done,false` then `tick-c3d4,Subtask one,open,done,true`; Pretty (README.md:569-576) keeps the box-drawing tree (`└─ tick-c3d4 "Subtask one": open → done`), byte-identical to `PrettyFormatter.cascadeTransition` + `writeCascadeTree` (internal/cli/pretty_formatter.go:293-357).
  - The toon/JSON column set matches `toonChangedRow` (internal/cli/toon_formatter.go:137-144). Row order — requested first, then cascaded — matches `CascadeResult.Changed()` (internal/cli/format.go:209-228) feeding `statusChangeSet.rows()`, which returns first-seen order (internal/cli/transition.go:139-147), so the README's "the task you named first, then any task the change cascaded to" (README.md:508) holds.
  - Labels: "**Simple transition** (TOON)" / "(Pretty)" / "(JSON)", "**TOON** (`changed` table)" / "**Pretty** (tree with box-drawing)". No cell claims a shared shape, and "flat lines" is gone.
  - Read the whole document (README.md:1-648): no `(unchanged)` anywhere, and no arrow-plus-`(auto)` line survives.
  - `tick show` (README.md:470-492), `tick list` (README.md:436-454) and dep-tree (README.md:328-350) samples are intact and each is still claimed by its own `readmeSample` entry; nothing in this task's section reaches them.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/readme_samples_test.go:28-29 adds `readmeTransitionAnchor` = `changed[1]{id,title,from,to,auto}:` and `readmeCascadeAnchor` = `changed[2]{id,title,from,to,auto}:`; `readmeChangedRows` (127-138) locates each block by first line, `t.Fatalf`s when the anchor is absent, decodes with `toon.DecodeString`, `t.Fatalf`s on a decode error, and `changedRowsOf` (140-162) asserts the document's only key is `changed`. Removing either block or making it unparseable fails the suite.
  - `TestREADMEToonSamplesDecode` subtests cover the title column (167-175) and the auto column with an exact row-count check per anchor — `{false}` for the one-row sample, `{false, true}` for the cascade (177-192), which is criteria 2 and 3.
  - `TestREADMETransitionSamples` (209-233) asserts the README carries exactly one JSON object sample, that it unmarshals, that its only key is `changed`, and that `auto` is a Go `bool` rather than a string — criterion 4.
  - `TestREADMESamplesMatchRenderedOutput` (425-452) renders each sample from seeded tasks through `App.Run` and compares byte-for-byte, covering all five transition/cascade blocks (`start toon`, `start pretty`, `start json`, `done toon`, `done pretty`, lines 372-380) — the strongest available form of "copied from real tool output", including the pretty box-drawing tree.
  - `TestREADMEPromptedSamplesAreClaimed` (514-542) fails if any `$ tick`-prompted fence is claimed by no sample, so a future sample added to this section cannot go unverified.
- Notes:
  - The plan listed "it decodes the README simple transition sample", "it decodes the README cascade sample" and "it fails when an anchored sample is missing" as separate subtests; they are folded into `readmeChangedRows`' fatal guards, exercised by both decode subtests. The substance — decode failure and missing anchor both fail rather than silently pass — is covered.
  - Not over-tested: the decode layer and the byte-for-byte layer assert different properties (the block parses as TOON with the right row semantics; the block equals real output), and the decode layer is required by criterion 8 in its own right.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run` subtests, `t.Helper()` on every helper, table-driven anchors, no testify.
- SOLID principles: Good — `readmeChangedRows` (locate + decode) and `changedRowsOf` (shape assertion) split so the JSON test reuses the shape assertion without the TOON decode.
- Complexity: Low.
- Modern idioms: Yes — `strings.SplitSeq`, `maps.Keys` with `slices.Sorted`, `slices.Equal`.
- Readability: Good — anchors are named constants, so a README edit that moves a sample fails with the anchor text in the message.
- Comment accuracy: Comments on `readmeToonBlocks`, `fenceBody` and `readmeChangedRows` hold against the code; no process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the three commands run; every other criterion was settled by reading. The README assertions in `readme_samples_test.go` are static-document checks, so a run is what confirms `TestREADMEToonSamplesDecode`, `TestREADMETransitionSamples`, `TestREADMESamplesMatchRenderedOutput` and `TestREADMEPromptedSamplesAreClaimed` all pass against the current README text.
