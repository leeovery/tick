TASK: free-text-round-trip-5-1 (tick-9ed70a) — End-Of-Flags Marker Ends Flag Parsing At The Top Level

ACCEPTANCE CRITERIA:
- `tick create -- "- title"` exits zero and stores the title `- title`
- `tick note add <id> -- "- text"` exits zero and stores the note text `- text`
- `--` is accepted after every command in `commandFlags`, including the no-flag commands and `doctor` and `migrate`
- An argument after the marker that spells a global flag exactly — `--json`, `--quiet` — does not set that flag and does not reach `ValidateFlags`
- Global flags before the marker still apply: `tick --json create -- "- title"` renders JSON
- The marker is absent from the arguments the handler receives, so it never becomes a title, a task ID or note text
- A second `--` after the first is an ordinary argument and is counted as a literal
- `--` with nothing after it exits zero and behaves as though it were not passed
- `--` before the subcommand is accepted and the following argument is still resolved as the subcommand
- `parseArgs` returns `literals` equal to the number of arguments appended to `rest` after the marker, and `0` when no marker appeared
- `splitLiteralArgs` clamps a count larger than the slice it is given
- `--` appears in no `commandFlags` entry and in no `flagInfo.Name`, so `TestCommandFlagsMatchHelp` is unaffected
- `--` is not added to `globalFlagSet`
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean, with no existing test modified except where it asserted the old rejection

STATUS: complete

SPEC CONTEXT: §10.1 records the defect — `ValidateFlags` inspects every dash-leading argument that is not numeric and not a global flag, so free text passed positionally (note text, task title) is refused, which is the one hole in §2.2's byte-identity round-trip bar. §10.2 fixes it with `--` as the canonical end-of-flags marker: nothing after the marker is read as a flag, not the command's own flags and not the global format flags, and text that spells a flag exactly (a note reading `--json`) becomes writable for the same reason a dash-leading one does. §3.3 keeps `doctor` and `migrate` out of the output work but not out of the input surface — both validate through `ValidateFlags` before bypassing the formatter, so the boundary has to reach them. The 2026-09-20 corrigendum to §10.2 records that the marker is not purely additive: `--` was previously accepted as the *value* of a registered value-taking flag, and that case is taken away here (restored by task 5-8's attached `--flag=value` form).

IMPLEMENTATION:
- Status: Implemented (with a later, sound refactor of how the boundary travels)
- Location:
  - `internal/cli/app.go:337` — `const endOfFlagsMarker = "--"`
  - `internal/cli/app.go:349` — `literals int` on `globalFlags`
  - `internal/cli/app.go:362-393` — `parseArgs` recognises the first bare `--`, drops it, stops consulting `applyGlobalFlag` past it, counts each post-marker argument appended to `rest`, and leaves subcommand resolution intact
  - `internal/cli/flags.go:181-188` — `splitLiteralArgs`, clamping `n` with `min` and clipping `flagArgs`' capacity
  - `internal/cli/app.go:43` — the single split in `App.Run`; `internal/cli/app.go:73`, `:80`, `:117` — all three `ValidateFlags` call sites take the flag half only (verified by grep: those are the only three call sites in non-test code)
- Notes:
  - Drift from the plan's step 3, and it is an improvement, not a loss. The task as committed (d8701f10) added `FormatConfig.Literals` and `FormatConfig.SplitLiterals`, splitting at each of the three validation sites separately. The final code splits once at `app.go:43` and threads `flagArgs, literals` through every handler signature; `FormatConfig.Literals` and `SplitLiterals` no longer exist (grep finds no reference). That is task 5-7's design ("Every Command Gets Its Arguments Split At The Marker"), it subsumes what 5-1 built, and none of 5-1's acceptance criteria name `FormatConfig`. Every criterion still holds under the final shape.
  - `splitLiteralArgs`' clamp is now defensive rather than load-bearing: with the split done once in `App.Run` over the whole of `subArgs`, `flags.literals` can never exceed the slice length, and `qualifyCommand`/`handleNote` slice the already-split flag half. The criterion asks for the clamp explicitly and it is tested (`end_of_flags_test.go:422`), so keeping it is right.
  - Behaviour worth recording, not a defect: a marker placed *before* a sub-subcommand — `tick note -- add <id> text`, `tick dep -- add a b` — leaves `flagArgs` empty, so `handleNote`/`handleDep` report "sub-command required" rather than routing. Both commands behave identically, nothing regresses (`--` was refused outright before this work), and it follows the spec's own framing that everything after the marker is text. No criterion or spec sentence covers it.
  - The marker also survives a value-taking flag correctly: `tick create --description -- x` reports `--description requires a value`, the one regression §10.2's corrigendum records and task 5-8 closes with `--description=--`.

TESTS:
- Status: Adequate
- Coverage: `internal/cli/end_of_flags_test.go` (667 lines) carries all fifteen tests the plan lists, under their planned names. `TestEndOfFlagsMarker` (`:35`) covers the end-to-end criteria — dash-leading title, dash-leading note text, the marker on a no-flag command, on `doctor`, on three `migrate` shapes, post-marker global flag as text, pre-marker global flag still applying, the marker dropped from handler arguments, a second marker as text, a bare trailing marker, a leading marker before the subcommand, an unknown flag before the marker still rejected, a loop over every `commandFlags` key asserting the marker is never reported as unknown, and a registry check that `--` is in no `commandFlags` entry, not in `globalFlagSet`, and in no `flagInfo.Name`. `TestParseArgsLiterals` (`:339`) and `TestSplitLiteralArgs` (`:401`) cover the unit-level criteria including the clamp. Tests genuinely fail if the feature breaks: the post-marker `--json` cases assert on stored values and on output format under `IsTTY: true`, so a consumed global flag would flip the format and fail the assertion.
- Notes:
  - The plan's single `"it accepts the marker on migrate"` test was replaced by three stronger ones (`--dry-run`, `--from` and `--pending-only` as post-marker text, plus a missing-`--from` case). Better coverage than planned.
  - Mild redundancy: the note-add half of `"it keeps the marker working on a command with a sub-subcommand"` (`:179`) repeats `"it accepts dash-leading note text after the marker"` (`:62`) exactly; only the `dep add` half is new. Not worth changing.
  - No existing test was modified by the task's commit (d8701f10 touched `app.go`, `flags.go`, `format.go` and the new test file only), which satisfies the last clause of the final criterion.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests in "it does X" form, `t.Helper()` on the new `runTick` helper, `t.TempDir()`-backed project setup, error wrapping unchanged.
- SOLID principles: Good. `parseArgs` keeps the single job of separating flags from the subcommand; `splitLiteralArgs` is a pure function with no knowledge of commands; the marker constant is declared once and referenced by both production code and tests.
- Complexity: Low. The marker adds one boolean and one guarded block to an existing loop; the `if !foundCmd { …; continue }` restructure preserves the prior semantics exactly.
- Modern idioms: Yes — builtin `min` for the clamp, three-index slicing to clip `flagArgs`' capacity so a later `append` cannot overwrite the first literal.
- Readability: Good. The doc comments on `parseArgs` (`app.go:353-361`), `splitLiteralArgs` (`flags.go:177-180`) and `globalFlagSet` (`flags.go:17-19`) each hold true against the code, name no process artifacts, and state the non-obvious parts (why the count is trailing, why the capacity is clipped).
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean, with no existing test modified except where it asserted the old rejection" — the second clause is settled by reading (the task commit modified no existing test, and no surviving test asserts the old `--` rejection). The first clause needs the three commands run over the current tree; verification here was by reading only.
