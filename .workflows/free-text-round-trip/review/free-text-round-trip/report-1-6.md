TASK: free-text-round-trip-1-6 — README Samples Match Real Output (tick-a8523e)

ACCEPTANCE CRITERIA:
- The `tick show` sample was copied from real tool output, not hand-written, and its section order is head fields, `blocked_by`, `children`, `tags`, `refs`, `notes`, `description`
- The sample's head is the top-level named fields `id`, `title`, `status`, `priority`, `type`, `created`, `updated` — no `task` key, no `[1]` and no `parent` line, matching the target shape in Context
- `tags` and `refs` in the sample are inline lists, and the refs URL is quoted as the encoder quotes it
- The sample's `description` is one quoted value with `\n` escapes, not an indented block
- The sample's notes table header is `notes[1]{index,text,created}:` and its row starts with `1`
- The `tick list` sample's empty-type row ends `,""`
- `TestREADMEToonSamplesDecode` passes, and fails if any of the three anchored blocks is removed or made unparseable
- The dep-tree, transition and cascade samples are unchanged by this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §12.1 holds the README as live documentation, so a sample printing output the tool does not produce is a shipped defect. It assigns this phase the `tick show` full-detail sample (old `README.md:430-450`) and the `tick list` table (old `README.md:396-401`, whose empty `type` rendered as a bare trailing comma where `toonTaskRow.Type` carries no `omitempty` and the encoder quotes an empty string), and explicitly defers the dep-tree summary header, the arrow transition, the JSON transition and the cascade samples to §5.2/§7.2/§4.2, which land in later phases.

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:471-491 (show sample), README.md:437-441 (list sample), internal/cli/readme_samples_test.go (new file); delivered in commit a29d0ca1, which touches only README.md and the new test file
- Notes:
  - Show sample head is top-level named fields `id`/`title`/`status`/`priority`/`type`/`created`/`updated` (README.md:471-477) with no `task` key, no `[1]` and no `parent` line. This matches `buildTaskSection` (internal/cli/toon_formatter.go:228-242), which emits named fields and omits `type`, `parent` and `closed` when absent — the sample task carries no parent.
  - Section order in the sample (head, `blocked_by`, `children`, `tags`, `refs`, `notes`, `description`) matches the emission order in `ToonFormatter.FormatTaskDetail` (internal/cli/toon_formatter.go:76-104).
  - `tags[2]: auth,backend` (README.md:484) and `refs[1]: "https://github.com/org/repo/issues/42"` (README.md:486) are inline lists with the URL quoted; `description: "Full task description here.\nCan be multiple lines."` (README.md:491) is one quoted value; the notes header is `notes[1]{index,text,created}:` with the row starting `1` (README.md:488-489), matching `buildNotesSection` (internal/cli/toon_formatter.go:310-320), which carries the 1-based position as `index`.
  - The list sample's empty-type row is `  tick-d5c6,Update docs,open,3,""` (README.md:441).
  - `git show a29d0ca1 -- README.md` changes only those two samples: the dep-tree, transition, cascade and JSON samples are untouched by this commit. (They were later corrected by 7bbdbd2f, 9a0204c2 and 3d23b502 — the phases §12.1 assigns them to.)
  - Line numbers in the task text (429-450, 396-401) have shifted because later phases inserted the field-selection section; the samples themselves are the ones the task names.
  - Sound divergence, not a loss: the three anchored decode subtests this task wrote were superseded in 28b01b5e/b9782df7 by `TestREADMESamplesMatchRenderedOutput`, which renders each sample through `App.Run` and compares byte-for-byte (internal/cli/readme_samples_test.go:425-452). That is a strictly stronger check than decoding, and it covers all three anchors this task named — `id: tick-a1b2` (line 340), `tasks[3]{…}` (line 364) and `tasks[2]{…}` (line 366). Removal of a block still fails via `findREADMEBlock` (line 393-408); a block altered into something the tool does not print fails the byte comparison.

TESTS:
- Status: Adequate
- Coverage:
  - `TestREADMESamplesMatchRenderedOutput/it reproduces the README show sample/show toon` (internal/cli/readme_samples_test.go:338-341) seeds the documented task and blocker (lines 284-300) and asserts `tick --toon show tick-a1b2` equals the README block byte-for-byte — this is what enforces "copied from real tool output".
  - `…/it reproduces the README list samples/list toon` and `/toon format list` (lines 364, 366) do the same for both `tasks[N]` blocks.
  - `TestREADMEToonSamplesDecode/it renders an empty type as a quoted empty string` (lines 194-206) pins the third row of the list sample to `tick-d5c6,Update docs,open,3,""`, the §12.1 point, and fails if the block gains or loses a line.
  - `TestREADMESamplesMatchRenderedOutput/it fails when a documented sample has no matching README block` (lines 446-451) is the surviving form of the planned "it fails when an anchored sample is missing" — it proves a missing anchor is an error rather than a silent pass, using the `tasks[9]{nothing}:` fixture this task introduced.
  - `TestREADMEPromptedSamplesAreClaimed` (lines 514-542, later phases) closes the remaining hole by failing when any prompted README fence is rendered by no sample.
- Notes: the anchor scheme keys blocks by their first line (`readmeToonBlocks`, lines 97-108), so two no-info blocks sharing a first line would silently collide. Enumerating every fenced block in README.md shows no such collision today, and `TestREADMEPromptedSamplesAreClaimed` plus `findREADMEBlock`'s occurrence counter (lines 393-408) cover the prompted samples where one could arise. The empty-type subtest overlaps the `list toon` byte comparison, but it is the named guard for the §12.1 defect and states the expectation directly, so the overlap is earned rather than redundant.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run()` subtests with "it does X" names, `t.Helper()` on every helper, `strings.SplitSeq` and `slices`/`maps` idioms per the modernize lint set, `testutil.FindRepoRoot(t)` for locating the repo root as `cmd/tick/build_test.go` does.
- SOLID principles: Good — parsing (`fencesIn`), selection (`findREADMEBlock`), rendering (`runREADMECommand`) and coverage accounting (`readmeCoverageErrors`) are separate, and `readmeCoverageErrors` is a pure function over its inputs, which is what lets lines 521-541 test the harness itself.
- Complexity: Low — `fencesIn` is a single linear pass with one state flag; no nesting beyond two levels.
- Modern idioms: Yes
- Readability: Good — doc comments on each non-obvious helper, anchors named as constants rather than repeated literals.
- Issues: None. Every doc comment in the file holds against its code, and none references a task id, phase or spec section.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The `tick show` sample was copied from real tool output, not hand-written" — reading settles the structure (section order and field shapes match `ToonFormatter.FormatTaskDetail` and `buildTaskSection`); byte-for-byte equality with what the tool prints is settled only by running `go test ./internal/cli -run TestREADMESamplesMatchRenderedOutput`.
- "`TestREADMEToonSamplesDecode` passes, and fails if any of the three anchored blocks is removed or made unparseable" — the removal and alteration paths are verifiable by reading (`findREADMEBlock` returns an error; the byte comparison fails), but "passes" requires running `go test ./internal/cli -run 'TestREADME'`.
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — every symbol the new test file references was located (`testutil.FindRepoRoot`, `task.TimestampFormat`, `setupTickProjectWithTasks` in internal/cli/create_test.go:34, `findCommand` in internal/cli/help.go:246, `endOfFlagsMarker` in internal/cli/app.go:337), but the three commands have to be run to settle it.
