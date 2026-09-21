TASK: free-text-round-trip-3-8 (tick-aa98d9) — One Test Holds Every README Sample Against Real Output

ACCEPTANCE CRITERIA:
1. Every fixture entry's rendered output equals its README block byte-for-byte, trailing newline aside; no covered sample is checked by decoding, substring or prefix alone.
2. All thirteen blocks are in the table, and a block whose first line or occurrence no longer matches fails the test rather than being skipped.
3. Renaming a documented key or shifting a documented value in any of the three formatters fails this test — verified locally by making one such change, watching the failure, and reverting it.
4. `readmeObjectHeaderMarker` and `readmeUnchangedMarker` and the subtests reading them are gone, and nothing replaces them with another one-off string marker.
5. No production code changes: the diff touches `internal/cli/readme_samples_test.go` and `README.md` only.
6. `go test ./...` green, `go vet ./...` clean, `gofmt -l ./internal` empty.
Plus the orchestrator's addition: correct the `dep tree` full-graph prose sentence so it describes what each format actually carries.

STATUS: complete

SPEC CONTEXT: §12.1 ("The README is updated as part of this work") holds the README to what the tool actually emits — every agent-format sample the work replaces is corrected, and the section states plainly that leaving live documentation describing output the tool does not produce is shipping a defect. This task is the guard that keeps that true after the fact: nothing previously compared an anchored README block to real formatter output, so a later rename or value shift could leave the README teaching a shape the tool no longer emits while the suite stayed green.

IMPLEMENTATION:
- Status: Implemented
- Location: `internal/cli/readme_samples_test.go:238-246` (`readmeSample` fixture type), `:262-389` (`readmeSampleGroups`, the fixture table and its seeds), `:393-408` (`findREADMEBlock` lookup by info string + first line + occurrence), `:410-423` (`runREADMECommand`), `:425-452` (`TestREADMESamplesMatchRenderedOutput`, including the missing-block guard at `:446-451`). Anchor constants at `:21-41`. README corrections in commit 28b01b5e: the full-graph prose sentence, the two pretty `list` blocks (column widths), and the JSON `list` sample gaining `"type": "feature"`.
- Notes:
  - All thirteen planned blocks are in the table and all thirteen fences exist in the current README: `dep tree` pretty `README.md:328-335` and toon `:341-350`; `list` toon `:436-442`, pretty `:448-454`, toon two-task `:464-468`, pretty two-task `:500-504`; `show` `:470-492`; `start` toon `:515-519`, pretty `:525-528`, JSON `:534-546`; `done` toon `:559-564`, pretty `:570-576`; `list` JSON `:586-596`. Three further samples (field selection, bare description, bare note position) were added to the same table by later phases.
  - The comparison is exact: `got != block` at `:438` over `strings.TrimRight(stdout, "\n")`, no decode, substring or prefix path for any covered sample. A sample whose first line or occurrence no longer resolves returns an error from `findREADMEBlock` and reaches `t.Fatalf` at `:434` — never a skip. An `occurrence` that cannot be reached (including the zero value) fails the same way.
  - Criterion 4 holds: `readmeObjectHeaderMarker`, `readmeUnchangedMarker` and `decodeAnchoredBlock` appear nowhere in the Go or Markdown sources (only in `.workflows` planning records), and no new one-off string marker replaced them — the new constants at `:34-40` are block first lines consumed by the lookup. Every constant in the block at `:21-41` has at least one use.
  - Criterion 5 holds: `git show --stat 28b01b5e` touches `README.md` and `internal/cli/readme_samples_test.go` only.
  - The deletions match the plan exactly against the pre-change file (28b01b5e~1): the five decode-only subtests (`:154-184`), the two negative-marker subtests (`:186-190`, `:265-269`), and the missing-anchor guard (`:219-223`) re-expressed through the new lookup. The three retained decoded-property subtests (`:192-200`, `:202-217`, `:225-237`) are present at `:167-206` and `:210-232` of the current file.
  - The orchestrator's addition landed: the `dep tree` sentence was rewritten in this commit and has since been revised again by phase 10. The current text (`README.md:321`) matches the implementation — `BuildFullDepTree` concatenates root trees with trees seeded from unreached participants (`internal/cli/dep_tree_graph.go:154`), so "all three formats cover every participant" is true as written.
  - The `occurrence` field currently disambiguates nothing: every (info string, first line) pair in the README is unique, because the two pretty `list` blocks the plan expected to collide now differ in column width after this task's correction. It remains a live part of the lookup contract and of the later coverage test's claim key, so it is a guard rather than dead weight.

TESTS:
- Status: Adequate
- Coverage: This task is a pure test addition, and the test is the deliverable. Sixteen fixture entries (the thirteen required plus three from later phases) each seed a project matching the sample's narrative and IDs, run the documented command through `App.Run` under an explicit format flag, and compare byte-for-byte. All three formatters are covered (toon: eight entries; pretty: five; JSON: two), so a rename or value shift in any of them lands on at least one compared block. Each mutating sample (`start`, `done`) seeds its own `t.TempDir()` project, so no ordering dependence between subtests.
- Notes:
  - Negative path is covered: `:446-451` asserts `findREADMEBlock` errors on the `readmeMissingAnchorFixture` first line, so the lookup's fatal path is itself tested rather than assumed.
  - Not over-tested: the three retained decode subtests now assert properties the byte comparison subsumes, but the plan retained them deliberately and they cost one decode each. Not worth unwinding.
  - The seeds hard-code `Created`/`Updated` timestamps parsed through `task.TimestampFormat` (`:253-260`), which is what keeps the `show` sample's `created`/`updated` lines reproducible.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.TempDir()` isolation via `setupTickProjectWithTasks`, `t.Helper()` on every helper, table-driven fixtures.
- SOLID principles: Good — lookup (`findREADMEBlock`), execution (`runREADMECommand`) and the fixture table are separate and each has one job.
- Complexity: Low — the lookup is a single counted scan; the test body is two nested loops over the table.
- Modern idioms: Yes — `strings.SplitSeq` in the shared fence parser, `slices`/`maps` helpers, `fmt.Errorf` for the lookup failure.
- Readability: Good — the `readmeSample` doc comment states the matching contract (info string, first line, occurrence counted from one in README order) and holds true against `findREADMEBlock`.
- Issues: None. `runREADMECommand` (`:410-423`) duplicates `runToonCommand` (`internal/cli/toon_decode_test.go:114-127`) with the format flag parameterised; both are used and the duplication is a few lines of test scaffolding, so it is a preference, not a defect.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Every fixture entry's rendered output equals its README block byte-for-byte, trailing newline aside" — reading settles that the comparison is exact and fails loudly on any mismatch, but whether it passes against today's formatters and today's README needs `go test ./internal/cli -run TestREADMESamplesMatchRenderedOutput`.
- "Renaming a documented key or shifting a documented value in any of the three formatters fails this test — verified locally by making one such change, watching the failure, and reverting it" — structurally the property holds either through the exact comparison or through the lookup fatal, but the prescribed mutate-watch-revert experiment was not run here.
- "`go test ./...` green, `go vet ./...` clean, `gofmt -l ./internal` empty" — requires execution.
