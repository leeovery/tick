TASK: free-text-round-trip-5-6 (tick-d35d1b) — README And Help Present The End-Of-Flags Marker

ACCEPTANCE CRITERIA:
- `tick help` lists `--` in its global-flags block
- `tick help --all` names `--` on its global-flags line, and the two lists agree
- `tick help create` and `tick help note` both describe `--` as the way to pass dash-leading free text
- `--` appears in no `flagInfo.Name` and in no `commandFlags` entry, and `TestCommandFlagsMatchHelp` passes unmodified in both directions
- The README's Global Flags block lists `--` with a one-line description
- README prose states that flags come before the marker, that everything after it is text, and that this covers an argument spelling a global flag exactly
- The `create` and `note` sections each carry a `--` invocation example, written as a shell command with no output block
- No new fenced output block is added (restated by the orchestrator's amendment: `readmeSampleGroups` unchanged and every entry still matches)
- The README does not contradict the `--field`/`--fields` documentation Phase 4 added, and does not present `--` as required
- The samples corrected in Phases 1, 2, 3 and 4 are byte-identical to what those phases left
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§10.2 makes `--` the canonical documented way to pass free text that may begin with a dash: "Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text. Free text that spells a flag exactly, a note reading `--json`, is writable for the same reason a dash-leading one is." It also fixes the two asymmetries this task must document without overstating: "The existing bare-argument form keeps working. `--` is the recommended form, not a required one" on `note add`, against "`create` cannot take the second half… `create` relies on `--`." §12.1 owes the README the new input surface on the grounds that "a flag documented nowhere is a flag nobody uses", and constrains where the help text can carry it — the marker is registered nowhere, so it can appear in no `Flags` list without failing `TestCommandFlagsMatchHelp` in the help-to-registry direction.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/help.go:272` — `--` entry in `printTopLevelHelp`'s `Global flags:` block, description column at offset 18, matching the seven existing lines.
  - `internal/cli/help.go:281` — `printAllHelp`'s single global-flags line extended with a trailing `--`.
  - `internal/cli/help.go:41-42` — `create` description gains "Put -- before a title that begins with a dash; everything after -- is text." No `flagInfo` added.
  - `internal/cli/help.go:154-155` — `note` description gains the equivalent sentence. No `flagInfo` added.
  - `README.md:633` — `--                End of flags; every argument after it is text` in the Global Flags fenced block.
  - `README.md:643` — prose covering the ordering rule, the global-flag-spelling case (`tick note add tick-a1b2 -- --json` stores the literal note `--json`), the `note add` "recommended rather than required" qualification, and the `create` case where the marker is needed.
  - `README.md:110` and `README.md:296` — one `--` invocation added to the existing `bash` example block in each of the `create` and `note` sections.
- Notes: The prose claims hold against the code. `tick create -- "--dry-run support"`: `parseArgs` (`internal/cli/app.go:361-392`) drops the marker and counts the tail into `flags.literals`, `splitLiteralArgs` (`app.go:43`) hands `create` an empty `flagArgs`, `ValidateFlags` sees nothing dash-leading, and `parseCreateArgs` (`internal/cli/create.go:104-106`) turns the literal into the title. `tick note add tick-a1b2 -- --json`: `applyGlobalFlag` is skipped past the marker, so `--json` reaches `RunNoteAdd` as a positional (`internal/cli/note.go:41-50`). The "recommended rather than required" qualification is exactly what `flagScanLimit["note add"] = 1` (`internal/cli/flags.go:101-103`) delivers, and the `create` exception is exactly what `ValidateFlags` rejecting an unregistered dash-leading first positional delivers. Nothing contradicts the Phase 4 `--field`/`--fields` entry, which still stands at `internal/cli/help.go:76` and in the README's `show` section.

TESTS:
- Status: Adequate
- Coverage: `TestHelpDocumentsEndOfFlagsMarker` (`internal/cli/help_test.go:479-530`) covers all five help criteria — the top-level block label, the `--all` token, list agreement, both command descriptions, and the registry exclusion. `TestREADMEDocumentsEndOfFlagsMarker` (`internal/cli/readme_samples_test.go:630-676`) covers the three README criteria, including one that the plan's test list did not name (the prose must mention both `--` and `--json` backticked) and a per-section count asserting exactly one marker invocation in a `bash` fence, which also fails if an output fence is added to either section. The parsing helpers are structural rather than substring-matched: `globalFlagLabels` cuts each line at its description column, so the assertion is that `--` is a *label*, not that the two characters appear somewhere in the output.
- Notes:
  - The orchestrator's amendment replaced the `"it adds no new toon anchor"` test with "assert `readmeSampleGroups` is unchanged and every entry still matches". No new test was added for this, and none is needed: the commit does not touch `readmeSampleGroups` (`readme_samples_test.go:262`), and `TestREADMESamplesMatchRenderedOutput` (`readme_samples_test.go:425-444`) already holds every entry against freshly rendered output on each run. The substance — nothing new needs covering — is delivered by an existing guard running unchanged.
  - `"it registers no flag for the marker"` (`help_test.go:517-529`) overlaps `"it registers the marker as neither a command flag nor a global flag"` (`internal/cli/end_of_flags_test.go:320-335`), which already walks `commandFlags` and `commands` for the same invariant. The newer one is marginally stronger — it splits alternative spellings via `longFlagsIn`, so it would catch `--` hidden inside a label like `"--field, --"`, where the older `f.Name == endOfFlagsMarker` would not. The plan listed the test explicitly, so the redundancy is deliberate and cheap; not worth undoing.
  - `"it keeps the two global flag lists in agreement"` (`help_test.go:492-505`) checks one direction only — every long flag in `printTopLevelHelp`'s block appears in `printAllHelp`'s line. That is exactly the direction the plan's Tests section specified, and `--` itself is pinned in both directions by the two preceding subtests. The unguarded direction (a flag added to the one-line `--all` string but not the aligned block) is the less likely drift, since the aligned block is the more visible edit site.
  - `TestCommandFlagsMatchHelp` (`internal/cli/flag_validation_test.go:317-360`) reads `info.Flags` only, so the two `Description` additions cannot reach it, and `--` is in no `Flags` list in either direction.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` only, `t.Run()` subtests, `t.Helper()` on all four new helpers, "it does X" subtest naming throughout.
- SOLID principles: Good. The `readmeShowSection` → `readmeSection(t, heading)` generalisation (`readme_samples_test.go:550-561`) derives the sibling-heading delimiter from the heading's own level rather than hard-coding `### `, and `readmeFences` → `fencesIn(content)` separates fence parsing from README loading so it can be applied to a single section. Both refactors preserve the prior behaviour exactly for the existing caller.
- Complexity: Low.
- Modern idioms: Yes — `strings.SplitSeq`, `strings.FieldsFuncSeq`, `strings.Cut`/`CutPrefix`, `slices.Contains`.
- Readability: Good. Every new helper carries a doc comment, and each states what it returns rather than restating its loop.
- Issues: None. Comments in the changed code hold against it: the `printAllHelp` doc comment, the `fencesIn` and `readmeSection` comments, and the help description sentences all describe what the code does.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the toolchain. Reading confirms no test was modified in a way that could break an existing assertion (the two `Description` additions are invisible to `TestCommandFlagsMatchHelp`, which reads `info.Flags` only; the two README `bash`-fence additions are invisible to the sample guards, which key on empty-info fences and `$ tick` prompt lines), but formatting and vet cleanliness cannot be judged by reading.
