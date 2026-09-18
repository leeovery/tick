# Phase 5: Dash-leading free text and the whitespace invariant — 6 tasks

## free-text-round-trip-5-1

### Task 1: `--` ends flag parsing at the top level

**Problem**: An agent reads a note off a task, corrects a typo, and writes it back. If the note begins with a dash — `- read the header`, the shape a bulleted note naturally takes — the command refuses it: `unknown flag "- read the header" for "note add"`. `ValidateFlags` (`internal/cli/flags.go:117-147`) inspects every argument beginning with `-` that is not numeric and not a global flag, and free text passed as a bare argument is inspected alongside real flags. `tick create "- some title"` is refused identically. There is no way to tell tick that an argument is text: `--` is itself rejected on every command today, because `parseArgs` (`internal/cli/app.go:352-373`) does not recognise it and `ValidateFlags` finds it in no command's flag set. Phases 1 to 4 made this text readable; this is the first half of making it writable.

**Solution**: Teach `parseArgs` to recognise the first bare `--` as the end-of-flags marker — it stops consuming global flags from that point, drops the marker itself from the argument list, and records how many trailing arguments followed it. `App.Run` hands only the pre-marker arguments to `ValidateFlags` at all three of its call sites, and the count rides on `FormatConfig` so the command parsers can consume it in Task 2.

**Outcome**: `tick create -- "- title"` and `tick note add <id> -- "- text"` both succeed, `--` is accepted after every command, nothing after it reaches `ValidateFlags` or `applyGlobalFlag`, and every invocation that works today produces identical output and exit status.

**Do**:
1. `internal/cli/app.go` — add `literals int` to `globalFlags`. In `parseArgs`, track whether the marker has been seen: the first argument exactly equal to `--` sets the flag and is not appended to `rest`; from then on `applyGlobalFlag` is not called and every remaining argument — a later `--` included — is appended to `rest` and counted in `flags.literals`. Subcommand resolution is unchanged: when the marker appears before the subcommand, the first following argument is still taken as the subcommand and is not counted.
2. `internal/cli/flags.go` — add `splitLiteralArgs(args []string, n int) (flagArgs, literals []string)`, which splits the trailing `n` arguments off `args` and clamps `n` to `len(args)`. The boundary is carried as a trailing count rather than an index because `qualifyCommand` (`internal/cli/app.go:381-399`) and `handleNote` (`internal/cli/note.go:25`) both slice leading arguments off, which shifts an index and leaves a trailing count valid.
3. `internal/cli/format.go` — add `Literals int` to `FormatConfig`, populated from `flags.literals` in `NewFormatConfig`, and add `func (fc FormatConfig) SplitLiterals(args []string) ([]string, []string)` delegating to `splitLiteralArgs`. This task populates the field; Task 2 is the first caller of the method.
4. `internal/cli/app.go` — at each of the three `ValidateFlags` call sites — `doctor` (`app.go:71`), `migrate` (`app.go:78`) and the `qualifyCommand` site (`app.go:113`) — pass the first return of `splitLiteralArgs` over that site's argument slice and `flags.literals`, so post-marker arguments are never inspected.
5. `internal/cli/end_of_flags_test.go` (new file) — cover the marker end to end through `App.Run` for `create`, `note add`, `doctor` and `migrate`, plus unit tests of `parseArgs` asserting the returned subcommand, `rest` and `literals`.

**Acceptance Criteria**:
- [ ] `tick create -- "- title"` exits zero and stores the title `- title`
- [ ] `tick note add <id> -- "- text"` exits zero and stores the note text `- text`
- [ ] `--` is accepted after every command in `commandFlags`, including the no-flag commands and `doctor` and `migrate`
- [ ] An argument after the marker that spells a global flag exactly — `--json`, `--quiet` — does not set that flag and does not reach `ValidateFlags`
- [ ] Global flags before the marker still apply: `tick --json create -- "- title"` renders JSON
- [ ] The marker is absent from the arguments the handler receives, so it never becomes a title, a task ID or note text
- [ ] A second `--` after the first is an ordinary argument and is counted as a literal
- [ ] `--` with nothing after it exits zero and behaves as though it were not passed
- [ ] `--` before the subcommand is accepted and the following argument is still resolved as the subcommand
- [ ] `parseArgs` returns `literals` equal to the number of arguments appended to `rest` after the marker, and `0` when no marker appeared
- [ ] `splitLiteralArgs` clamps a count larger than the slice it is given, so a sub-subcommand consumed by `qualifyCommand` cannot produce an out-of-range split
- [ ] `--` appears in no `commandFlags` entry and in no `flagInfo.Name`, so `TestCommandFlagsMatchHelp` is unaffected
- [ ] `--` is not added to `globalFlagSet`
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean, with no existing test modified except where it asserted the old rejection

**Tests**:
- `"it accepts a dash-leading title after the marker"` — `tick create -- "- title"` exits 0 and the persisted task's title is `- title`
- `"it accepts dash-leading note text after the marker"` — `tick note add <id> -- "- text"` exits 0 and the persisted note text is `- text`
- `"it accepts the marker on a command with no flags"` — `tick show <id> --` exits 0 with today's output
- `"it accepts the marker on doctor"` — `tick doctor -- --bogus` exits with doctor's own status, not the unknown-flag error
- `"it accepts the marker on migrate"` — `tick migrate --from beads -- --bogus` does not report an unknown flag
- `"it treats a post-marker global flag as text"` — `tick create -- --json` exits 0 and the stored title is `--json`
- `"it applies a global flag placed before the marker"` — `tick --json create -- "- title"` writes a JSON document
- `"it drops the marker from the arguments"` — `tick note add <id> -- text` stores `text`, not `-- text`
- `"it treats a second marker as text"` — `tick create -- -- ` stores the title `--`
- `"it accepts a marker with nothing after it"` — `tick list --` exits 0 with the output `tick list` produces
- `"it resolves the subcommand after a leading marker"` — `tick -- create "x"` creates a task
- `"it counts only post-marker arguments"` — `parseArgs` over `["--quiet","create","a","--","b","c"]` returns subcommand `create`, `rest` `["a","b","c"]` and `literals` 2
- `"it reports no literals without a marker"` — `parseArgs` over today's arguments returns `literals` 0
- `"it clamps a literal count larger than the slice"` — `splitLiteralArgs([]string{"x"}, 3)` returns an empty flag slice and `["x"]`
- `"it rejects an unknown flag before the marker"` — `tick list --stauts open -- x` still reports `unknown flag "--stauts" for "list"`

**Edge Cases**:
- The marker itself must not become a task ID or note text — `parseArgs` drops it rather than passing it through
- A second `--` after the first is literal text; only the first occurrence is the boundary
- `--` with nothing after it leaves `literals` at 0 and changes nothing
- `--` before the subcommand: flag parsing ends there, but command resolution does not — the next argument is still the subcommand, and a flag-shaped argument in the subcommand slot keeps `parseArgs`' existing `unknown flag %q. Run 'tick help' for usage.` error
- Global flags before the marker still apply; `--json` and `--quiet` after it are text, because `applyGlobalFlag` is not consulted past the boundary
- `doctor` and `migrate` validate through their own `ValidateFlags` calls before the formatter machinery is built, so both take the boundary from `flags.literals` directly
- The boundary is retained as a count for Task 2 rather than discarded with the marker; the command parsers cannot recover it once the marker is gone
- `qualifyCommand` and `handleNote` slice leading arguments away, which is why the count is measured from the end and why `splitLiteralArgs` clamps
- `--` is not registered in `commandFlags` and is not a `flagInfo`, so the help drift test is untouched; it is also not a global flag, so it does not belong in `globalFlagSet`
- `--` is rejected on every command today, so no existing invocation regresses — with the single exception below
- `--` sitting where a value-taking flag's value belongs (`tick create --description -- x`) becomes the marker, so that invocation now reports `--description requires a value` where it previously stored a two-dash description

**Context**:
> §10.1: "An agent reads a note off a task, corrects a typo, and writes it back. If the note begins with a dash — `- read the header`, the shape a bulleted note naturally takes — the command refuses it… Wider than notes: `tick create \"- some title\"` is refused identically. `ValidateFlags` inspects every argument beginning with `-` that is not numeric and not a global flag, and rejects any it cannot find in the command's flag set… Free text passed as a bare argument — note text, task title — is inspected alongside real flags. `--description` escapes only because its value follows a registered value-taking flag, so validation skips it." And on what must survive: "The check itself is deliberate and worth keeping: it exists so `tick update tick-a1b2 --prioirty 3` refuses rather than silently reporting success. The defect is that it is applied to arguments that are free text by definition."
>
> §10.2: "`--` is supported as the end-of-flags marker, and becomes the canonical documented way to pass free text that may begin with a dash. Purely additive: `--` is currently rejected on every command (`unknown flag \"--\" for \"note add\"`), so no existing invocation uses it. Everything that works today works identically, and inputs that currently fail begin to succeed. Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text. Free text that spells a flag exactly, a note reading `--json`, is writable for the same reason a dash-leading one is." And: "`create` cannot take the second half: its title shares the argument list with real flags (`--priority`, `--description`), so a dash-leading title is indistinguishable from a mistyped flag without a marker. `create` relies on `--`."
>
> §2.2 sets the contract this closes: "Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was." And: "The bar also does not hold for free of charge across all three free-text carriers. Note text and task titles are rejected before reaching storage when they begin with a dash — see §10. Reaching byte-identity for them requires that fix, which this work carries."
>
> §3.3 keeps `doctor` and `migrate` out of the output work, but their input surface is not exempt: both validate through `ValidateFlags` at `internal/cli/app.go:71` and `internal/cli/app.go:78` before they bypass the formatter, so the boundary has to reach them or `--` would be accepted on eleven commands and refused on two.
>
> The specification does not say where the marker is recognised or how the boundary travels. Recognising it in `parseArgs` is chosen because that is the one place every argument passes through before any command sees it, and because global flags are consumed there — stopping them at the marker is what makes a note reading `--json` writable. The specification is also silent on `--` appearing before the subcommand and on `--` in a value position; both are resolved above by the least disruptive reading, and neither is an invocation any caller could have meant.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §10.1, §10.2, §2.2, §3.3

## free-text-round-trip-5-2

### Task 2: Post-marker text that spells a command flag is text

**Problem**: Task 1 stopped the top-level flag scanner at the marker, but every command parses its own flags by walking its argument list and matching exact strings wherever they appear. `parseCreateArgs` (`internal/cli/create.go:30-98`) matches `--description` at any position and consumes the argument after it as its value; `parseUpdateArgs` (`internal/cli/update.go:38`), `parseListFlags` (`internal/cli/list.go:36-101`), `parseRemoveArgs` (`internal/cli/remove.go:20-39`) and Phase 4's `parseShowArgs` all work the same way. So `tick create -- --priority 0` still loses its title to a flag match and `tick show <id> -- --field title` still parses a field selection out of what the caller marked as text. §10.2 requires that nothing after the marker is read as a flag — "not the command's own flags and not the global format flags" — and half of that is still unmet.

**Solution**: Split each handler's arguments through `fc.SplitLiterals` and hand the two halves to its parser: flags are matched against the pre-marker half only, and the post-marker half joins the parser's positional arguments in order.

**Outcome**: A post-marker argument is a positional wherever a command has one and is ignored wherever it does not, in every command that scans its arguments for flags; `tick create -- --priority` stores the title `--priority`, and `tick show <id> -- --field` renders full output.

**Do**:
1. `internal/cli/app.go` — in `handleList`, `handleReady`, `handleBlocked` and `handleRemove`, and `internal/cli/create.go`, `internal/cli/update.go`, `internal/cli/show.go` in `RunCreate`, `RunUpdate` and `RunShow`, call `flagArgs, literals := fc.SplitLiterals(args)` before parsing and pass both halves to the parser. `handleReady` and `handleBlocked` keep prepending `--ready`/`--blocked` to the flag half only.
2. `internal/cli/create.go` — change `parseCreateArgs` to `parseCreateArgs(flagArgs, literals []string) (createOpts, error)`: walk `flagArgs` exactly as today, then run each literal through the same positional branch, so the first-positional-wins title rule is unchanged and a second post-marker argument is still ignored.
3. `internal/cli/update.go` — change `parseUpdateArgs` the same way, its positional being the task ID. `internal/cli/list.go` — change `parseListFlags` the same way; `list` has no positional, so its literals are ignored exactly as a stray positional is ignored today.
4. `internal/cli/show.go` — change Phase 4's `parseShowArgs` to take both halves: `--field`/`--fields` are recognised in `flagArgs` only, and a literal is a candidate for the task ID, never for a selection. `internal/cli/remove.go` — change `parseRemoveArgs` the same way, its literals joining the ID list in order and keeping the existing duplicate collapsing.
5. `internal/cli/end_of_flags_test.go` — add subtests per command asserting that a post-marker argument spelling one of that command's flags is text, and that commands which read positionals by index and never scan for flags (`start`, `done`, `cancel`, `reopen`, `dep add`, `dep remove`, `dep tree`, `note add`, `note remove`, `stats`, `init`, `rebuild`) still receive their arguments in order.

**Acceptance Criteria**:
- [ ] `tick create -- --priority` exits zero and stores the title `--priority`
- [ ] `tick create -- --description foo` stores the title `--description`, leaves the description empty and ignores `foo` — a post-marker value-taking flag swallows nothing
- [ ] `tick create --priority 0 -- "- title"` stores priority 0 and the title `- title` — flags before the marker still parse normally
- [ ] `tick show <id> -- --field title` renders full output rather than a field selection, and resolves the same task
- [ ] `tick show -- <id>` resolves the task
- [ ] `tick update <id> --title x -- --json` stores the title `x` and renders in the resolved default format
- [ ] `tick remove -- --force` treats `--force` as a task ID and reports it unresolved rather than skipping confirmation
- [ ] `tick list -- --status` exits zero and lists tasks, ignoring the literal as a stray positional
- [ ] `tick note add <id> -- "- text"` stores `- text`, with the marker absent from the joined text
- [ ] A command with no positional argument behaves as it does today apart from no longer reading post-marker text as a flag
- [ ] A selection flag whose value would sit after the marker (`tick show <id> --field -- title`) still reports `--field requires a value`
- [ ] Every invocation that carries no marker produces byte-identical output and the same exit status as before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it treats a post-marker command flag as a title"` — `tick create -- --priority` stores `--priority`
- `"it does not let a post-marker value-taking flag swallow the next argument"` — `tick create -- --description foo` stores the title `--description` and an empty description
- `"it parses flags placed before the marker"` — `tick create --priority 0 --type bug -- "- title"` stores priority 0, type `bug` and the dash-leading title
- `"it ignores a second post-marker positional on create"` — `tick create -- "- first" second` stores `- first`
- `"it treats a post-marker field flag as text on show"` — `tick show <id> -- --field title` renders the full document
- `"it resolves a post-marker task id on show"` — `tick show -- <id>` prints that task
- `"it treats a post-marker global flag as text on update"` — `tick update <id> --title x -- --json` renders in the default format
- `"it treats a post-marker force flag as an id on remove"` — `tick remove -- --force` reports an unresolved ID and removes nothing
- `"it ignores a post-marker flag name on list"` — `tick list -- --status` exits 0 and lists every task
- `"it joins note text without the marker"` — `tick note add <id> -- "- a" "b"` stores `- a b`
- `"it passes post-marker arguments to a command that scans no flags"` — `tick done -- <id>` transitions that task
- `"it still reports a missing flag value"` — `tick show <id> --field -- title` exits 1 with `--field requires a value`
- `"it leaves unmarked invocations unchanged"` — the existing create, update, list, show and remove suites pass unmodified

**Edge Cases**:
- Flags before the marker still parse normally; the marker ends flag parsing, it does not disable it
- `create`'s first-positional-wins title rule is unchanged, so a second post-marker argument is still ignored
- A post-marker `--field` on `show` is text rather than a selection, and a post-marker task ID is still a task ID
- A post-marker value-taking flag must not swallow the argument after it — the literals are positionals, never flag/value pairs
- Commands with no positional argument are unaffected beyond no longer reading post-marker text as a flag; `list` already ignores stray positionals
- `note add`'s text join must not pick the marker up — Task 1 dropped it, and the join is asserted here
- A flag whose value would fall after the marker keeps its own `requires a value` error: flags and their values both belong before the boundary
- Commands that read positionals by index and never scan for flags need no parser change, but must still receive post-marker arguments in order

**Context**:
> §10.2: "Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text. Free text that spells a flag exactly, a note reading `--json`, is writable for the same reason a dash-leading one is." And: "The existing bare-argument form keeps working. `--` is the recommended form, not a required one."
>
> §10.2 on why `create` in particular needs this: "`create` cannot take the second half: its title shares the argument list with real flags (`--priority`, `--description`), so a dash-leading title is indistinguishable from a mistyped flag without a marker. `create` relies on `--`."
>
> §10.1 on what must not be weakened: "The check itself is deliberate and worth keeping: it exists so `tick update tick-a1b2 --prioirty 3` refuses rather than silently reporting success. The defect is that it is applied to arguments that are free text by definition." Pre-marker arguments keep the full check.
>
> §9.1 fixes what a post-marker `--field` must not do: the flag "takes a comma-separated list of field names" and its names are validated by Phase 4's parser. A literal never reaches that parser, so `tick show <id> -- --field title` is a request for a task, not a selection.
>
> Task 1 carries the boundary as a trailing count on `FormatConfig`; `fc.SplitLiterals` is the only way a parser learns where it sits, and every handler already receives `fc`.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §10.1, §10.2, §9.1

## free-text-round-trip-5-3

### Task 3: `note add` stops inspecting flags after the task ID

**Problem**: The invocation §10.1 opens the defect with still fails after Tasks 1 and 2 unless the caller knows to type the marker. `tick note add <id> "- read the header"` reaches `ValidateFlags("note add", ["<id>", "- read the header"], commandFlags)`, which inspects the second argument because it begins with `-`, finds `commandFlags["note add"]` empty (`internal/cli/flags.go:80`) and answers `unknown flag "- read the header" for "note add". Run 'tick help note' for usage.` The check is protecting a flag set that has no members: nothing dash-leading after the task ID could be a flag it would have caught, so every rejection it produces there is a false one.

**Solution**: Register a per-command limit on how many leading arguments `ValidateFlags` inspects, set to 1 for `note add`, so the task ID is still checked and everything after it is free text. Every other command, `note remove` included, keeps full validation.

**Outcome**: `tick note add <id> "- read the header"` stores that note with no marker and no error, while `tick note remove <id> --bogus 1` still reports the unknown flag.

**Do**:
1. `internal/cli/flags.go` — add `var flagScanLimit = map[string]int{"note add": 1}` beside `commandFlags`, documented as the number of leading arguments inspected for a command whose remaining arguments are free text by definition.
2. `internal/cli/flags.go` — in `ValidateFlags`, look the command up in `flagScanLimit` and stop the scan once the loop index reaches the limit; a command absent from the map is scanned in full, exactly as today.
3. `internal/cli/note_test.go` — add subtests through `App.Run` covering dash-leading note text with no marker, note text that spells a global flag, the first-argument check, and `note remove`'s unchanged validation.
4. `internal/cli/flag_validation_test.go` — add unit subtests over `ValidateFlags` asserting the limit applies to `note add` and to no other command.
5. `internal/cli/unknown_flag_test.go` — confirm its `note add` and `note remove` rows still pass unmodified; both put the unknown flag in the first position, which the limit still inspects.

**Acceptance Criteria**:
- [ ] `tick note add <id> "- read the header"` exits zero and stores the note text `- read the header`
- [ ] Any other dash-leading text after the ID is accepted, including text beginning with two dashes
- [ ] A flag-shaped first argument is still inspected: `tick note add --bogus <id> text` exits non-zero with the unknown-flag error
- [ ] `tick note add <id> text --json` prints a JSON document, unchanged from today
- [ ] Note text that spells a global flag exactly still needs the marker: `tick note add <id> -- --json` stores `--json`
- [ ] `tick note remove <id> --bogus 1` still exits non-zero with the unknown-flag error
- [ ] Every other command's validation is unchanged — `flagScanLimit` has exactly one entry
- [ ] `tick note add` with no arguments still reports `task ID is required. Usage: tick note add <task_id> <text>`
- [ ] `tick note add <id>` with no text still reports `note text is required and cannot be empty`
- [ ] Multi-word text still joins with single spaces
- [ ] `commandFlags["note add"]` stays empty and `note`'s help entry is unchanged, so `TestCommandFlagsMatchHelp` passes untouched
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it accepts dash-leading note text without the marker"` — `tick note add <id> "- read the header"` exits 0 and stores it
- `"it accepts note text beginning with two dashes"` — text reading `--not-a-flag` is stored as typed
- `"it still rejects a flag before the task id"` — `tick note add --bogus <id> text` exits 1 with the unknown-flag error
- `"it still resolves a global flag after the text"` — `tick note add <id> text --json` writes a JSON document
- `"it needs the marker for text that spells a global flag"` — `tick note add <id> -- --json` stores `--json`
- `"it keeps full validation on note remove"` — `tick note remove <id> --bogus 1` exits 1 with the unknown-flag error
- `"it applies the scan limit to note add alone"` — `ValidateFlags` rejects `--bogus` in the second position for every other command
- `"it reports a missing task id"` — unchanged error
- `"it reports empty note text"` — unchanged error
- `"it joins multi-word text with single spaces"` — `tick note add <id> a b c` stores `a b c`

**Edge Cases**:
- `note add` registers no flags, so stopping the check after the task ID loses nothing — there is no flag it could have caught there
- `tick note add <id> --json` still resolves JSON because global flags are consumed by `parseArgs` before the command ever sees its arguments
- Note text that spells a global flag exactly is consumed as that flag and still needs the marker; a dash-leading note that is not a global flag works with or without one
- `note remove` keeps full validation, so a mistyped flag there is still caught
- The missing-ID and empty-text errors are unchanged — the limit affects validation only, not argument handling
- Multi-word text still joins with single spaces, since the join is `RunNoteAdd`'s and is untouched
- The limit is keyed on the fully-qualified command name `note add`, which is what `qualifyCommand` produces

**Context**:
> §10.2, second half of the fix: "Flag inspection stops after the task ID on `note add`. The command registers no flags at all (`grep -n '\"note add\":' internal/cli/flags.go` → `flags.go:80`, `\"note add\": {}`), so nothing dash-leading after the ID could be a flag the check would have caught, and refusing it is the whole defect. What stops is the check, not flag handling: global flags are consumed wherever they appear, before the command sees its arguments (`sed -n '352,373p' internal/cli/app.go`), so `tick note add <id> \"text\" --json` prints a JSON document exactly as it does today, and note text that spells a global flag exactly still needs the marker. A dash-leading note that is not itself a global flag works with or without it."
>
> §10.2: "The existing bare-argument form keeps working. `--` is the recommended form, not a required one."
>
> §10.1: "This is the one place the §2.2 round-trip guarantee has a hole — text an agent can read out of tick and cannot put back. It is fixed here rather than left as a follow-up, because leaving it means shipping the contract with an exception nobody wrote down."
>
> §6.4 fixes what a note round trip costs and why the write path matters: notes stay an append-and-retract log with no edit command, so correcting a note means adding it again — which is exactly the invocation this task unblocks.
>
> The specification says the check stops after the task ID but not how. A per-command scan limit is chosen over a `note add` special case inside `ValidateFlags` so the rule sits beside `commandFlags`, where the next person looking for `note add`'s flag behaviour will find it.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §10.1, §10.2, §2.2, §6.4

## free-text-round-trip-5-4

### Task 4: `tick migrate` trims imported free text

**Problem**: The byte-identity bar of §2.2 rests on an invariant that is not currently true. `create` and `update` trim before storing — `TrimTitle` at `internal/cli/create.go:117` and `internal/cli/update.go:182`, `internal/cli/update.go:337`, `TrimDescription` at `internal/cli/create.go:214` and `internal/cli/update.go:193`, `internal/cli/update.go:342` — and `note add` trims through `TrimNoteText` at `internal/cli/note.go:50`, so a value that came out of storage survives a write-back untouched. `tick migrate` is a third write path and stores the source tool's value as it arrives: `internal/migrate/store_creator.go:75-84` builds the task with `Title: mt.Title` and `Description: mt.Description`, and `MigratedTask.Validate` (`internal/migrate/migrate.go:44`) trims only for its emptiness check. An imported description with a leading newline or trailing spaces reads out of `tick show` intact and is silently trimmed on write-back — the bar failing on precisely the tasks nobody typed by hand.

**Solution**: Normalise a `MigratedTask`'s free text once, in the migrate Engine, before validation — so the value that is validated, the value that is persisted and the value that is reported are the same one, and so the store creator and the dry-run creator inherit the rule rather than each implementing it.

**Outcome**: `tick migrate` stores titles and descriptions with edge whitespace removed, a whitespace-only description imports as empty rather than failing, and the invariant §2.2 rests on holds on every write path in the system.

**Do**:
1. `internal/migrate/migrate.go` — add `func (mt MigratedTask) Normalize() MigratedTask` returning a copy whose `Title` is `task.TrimTitle(mt.Title)` and whose `Description` is `task.TrimDescription(mt.Description)`, leaving every other field alone.
2. `internal/migrate/engine.go` — in `Run`, rename the loop variable off `task` (it shadows the imported package) and reassign it from `Normalize()` before `Validate()` is called, so validation, creation and the `Result` all see the normalised value.
3. `internal/migrate/engine_test.go` — add subtests over `Run` with a stub provider: edge whitespace on title and description, a whitespace-only description, a whitespace-only title, interior whitespace and newlines, and a dry-run creator asserting the reported titles.
4. `internal/cli/migrate_test.go` — add an end-to-end subtest asserting the persisted task's `Title` and `Description` are the trimmed values, read through `readPersistedTasks`.
5. Confirm the scope boundary: `grep -rn 'TrimTitle\|TrimDescription\|TrimSpace' internal/migrate/ --include='*.go' | grep -v _test` shows normalisation at the single `Normalize` site and `Validate`'s existing emptiness check, the providers under `internal/migrate/beads/` are unmodified, and no code path rewrites tasks already in storage.

**Acceptance Criteria**:
- [ ] An imported title carrying leading or trailing whitespace is stored trimmed
- [ ] An imported description carrying leading or trailing whitespace, including a leading newline, is stored trimmed
- [ ] A whitespace-only description imports as the empty string and the import succeeds
- [ ] No import fails because a description was whitespace-only
- [ ] A whitespace-only title still fails with today's `title is required and cannot be empty` and is reported under the `(untitled)` fallback title
- [ ] Interior whitespace, blank lines and newlines inside a description are preserved byte-for-byte
- [ ] A dry run reports the trimmed titles, because it shares the Engine
- [ ] Normalisation happens at exactly one point, so a provider that later carries note text inherits the rule without a second decision
- [ ] The providers under `internal/migrate/beads/` are unchanged
- [ ] No pass rewrites stored records: a task already in `tasks.jsonl` carrying edge whitespace is byte-identical after a migration runs
- [ ] `tick show` emits stored bytes unmodified — a stored description carrying edge whitespace comes back with that whitespace
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it trims edge whitespace from an imported title"` — a provider title of `"  Fix parser  "` is persisted as `Fix parser`
- `"it trims edge whitespace from an imported description"` — a leading newline and trailing spaces are gone from the persisted description
- `"it imports a whitespace-only description as empty"` — the persisted description is `""` and the result is a success
- `"it does not fail an import because the description was whitespace-only"` — the run reports one imported, zero failed
- `"it keeps today's error for a whitespace-only title"` — the result is a failure carrying `title is required and cannot be empty` and the title `(untitled)`
- `"it preserves interior whitespace and newlines"` — a description with blank lines and an indented line is persisted byte-for-byte
- `"it reports the trimmed title in a dry run"` — `DryRunTaskCreator` plus an untrimmed provider title yields a `Result.Title` with no edge whitespace
- `"it normalises before validation"` — a title that is only whitespace is rejected once, with the normalised value reaching `Validate`
- `"it stores the trimmed values end to end"` — after `tick migrate`, `readPersistedTasks` shows the trimmed title and description
- `"it leaves values already in storage untouched"` — a seeded task whose description carries edge whitespace is unchanged after a migration
- `"it emits stored bytes unmodified from show"` — `tick show <id> --field description` on that seeded task prints the stored bytes including their edge whitespace

**Edge Cases**:
- A whitespace-only description imports as empty and no import fails because of it — the trim normalises, it does not validate
- A whitespace-only title keeps today's validation error, since `Validate` already trims for its check and the normalised title is empty either way
- Interior whitespace and newlines are preserved; only the edges are touched
- Dry-run shares the Engine, so its reported titles are the trimmed ones and the two paths cannot drift
- One normalisation point, so a provider that later carries note text inherits the rule — `MigratedTask` holds no note field today
- `Engine.Run`'s loop variable is named `task`, which shadows the imported `task` package; the trim helpers live on `MigratedTask` partly for that reason and the variable is renamed regardless
- Values already in storage are left alone: no pass rewrites stored records, and a task imported by an earlier version keeps its untrimmed value until it is next edited
- `tick show` still emits stored bytes — trimming on the way out would buy the match by breaking the read the bar is stated over

**Context**:
> §2.2: "The invariant that derivation rests on is not currently true. `tick migrate` is a third write path and it stores the source tool's value as it arrives… `sed -n '75,84p' internal/migrate/store_creator.go` builds the task with `Description: mt.Description`. Titles carry the same hole: `grep -n 'TrimSpace(mt.Title)' internal/migrate/migrate.go` → `migrate.go:44` validates that a trimmed title is non-empty, and `store_creator.go:78` then stores the untrimmed one. An imported description with a leading newline or trailing spaces reads out of `tick show` intact and is silently trimmed on write-back — the bar failing on precisely the tasks nobody typed by hand."
>
> §2.2: "**`tick migrate` therefore trims every free-text value it imports, exactly as `create` does.** Today that is descriptions and titles — the import framework carries no notes… and a provider that later brings note text across is covered by the same rule rather than by a second decision. This is in scope for this work: it makes the invariant true system-wide rather than documenting an exception a reader cannot predict from the output. The trim normalises, it does not validate: a value that is nothing but whitespace stores as empty, and no import fails because of it. Import is already a translation boundary — statuses, priorities and timestamps are all mapped on the way in — so normalising whitespace there is the same kind of move, and it costs a reader nothing they would notice."
>
> §2.2 on the two declined alternatives: "Dropping the trim from `create` and `update` would make byte-identity hold with no invariant at all, but changes behaviour for every user to serve a case only importers hit, and reopens whitespace-only descriptions, which `ValidateDescriptionUpdate` currently routes to `--clear-description`. Writing the exception down — the bar covering CLI-authored descriptions only — costs no code and hands the reader the kind of unpredictable exception this work exists to delete."
>
> §2.2 on the boundary this task must not cross: "**Values already in storage are left alone.** The bar holds for everything written from this change onward; a task imported by an earlier version keeps its untrimmed value and can still lose edge whitespace on a write-back, and that exposure closes itself as those tasks are edited. No pass rewrites stored records, and `tick show` never emits anything but the stored bytes — trimming on the way out would buy the match by breaking the read the bar is stated over."
>
> §2.2 states the invariant in full: "every path that stores free text trims edge whitespace before storing it — `TrimNoteText` on note text… and `TrimTitle` on titles… exactly as `TrimDescription` does above. The import path is the only one where it does not, which is what the trim above adds."
>
> §3.3: "This covers their **output** only. `migrate`'s import write path is touched by §2.2, which trims descriptions and titles on the way in." Nothing about `migrate`'s printed output changes here.
>
> The specification does not name the normalisation point. The Engine is chosen because it is the one place both creators pass through, which is what makes the dry run report the same titles the real run stores.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §2.2, §3.3

## free-text-round-trip-5-5

### Task 5: The awkward-task fixture round-trips byte-identically

**Problem**: Every test this work has added so far checks a shape — that a document decodes, that a section carries the right rows, that a flag parses. None checks the guarantee the work actually made, which is the round trip: read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was. §11 requires one deliberately awkward task as a permanent fixture for exactly that reason — a single test that would have caught the original description defect, the tags item-marker defect and the refs comma defect, none of which a parse check alone catches.

**Solution**: One permanent test that writes a fixture task whose title, description and note each carry text the old output mangled, and which also carries a tag and a ref so the sections the old builder wrote by hand are in the document too, reads every value back through both read paths — the decoded toon document and `tick show --field` — writes the decoded text back, and asserts the stored value byte-for-byte at each step.

**Outcome**: `internal/cli/round_trip_test.go` fails if any free-text carrier stops surviving write → read → decode → write-back, and if the tags or refs sections stop decoding to their stored values, and it asserts stored values rather than output text.

**Do**:
1. `internal/cli/round_trip_test.go` (new file) — declare the five fixture constants given in Context and create the task through `App.Run` with `--description`, `--tags` and `--refs` before the marker and the dash-leading title after it, capturing the ID from `--quiet` output, then add the note with `tick note add <id> -- <text>`.
2. Assert the write half: `readPersistedTasks` shows `Title`, `Description`, `Tags`, `Refs` and `Notes[0].Text` equal to the fixture constants byte-for-byte.
3. Read path A: run `tick --toon show <id>`, decode with `decodeToonDoc` (added in Phase 1), and assert the decoded `title`, `description`, `tags`, `refs` and first `notes` row's `text` equal the stored values.
4. Read path B: run `tick show <id> --field title`, `--field description` and `--field notes.1`, strip exactly one trailing newline from each, and assert each equals the stored value.
5. Write the decoded values back — `tick update <id> --title <decoded title> --description <decoded description>` and `tick note add <id> -- <decoded note text>` — then assert from `readPersistedTasks` that the stored title and description are unchanged and that `Notes[1].Text` equals `Notes[0].Text`. Add a second subtest creating a task whose title and description carry edge whitespace, asserting they store trimmed and stay trimmed across a write-back.

**Acceptance Criteria**:
- [ ] One task carries all three free-text carriers: a dash-leading title containing a comma, a multi-line description, and a dash-leading note
- [ ] The same task carries at least one kebab-case tag and one colon-bearing URL ref, so the document it produces contains the two sections the old hand-written builder wrote as unmarked indented items
- [ ] The description carries blank lines, a header-shaped line, an interior line ending in trailing spaces, an embedded double quote, a comma and a line beginning with a dash
- [ ] The note text begins with a dash and carries an embedded double quote and a comma
- [ ] `create` and `note add` both succeed for that content
- [ ] The stored title, description, tags, refs and note text are byte-identical to the fixture constants
- [ ] The decoded toon document's `title`, `description` and note `text` are byte-identical to the stored values
- [ ] The decoded toon document's `tags` and `refs` are element-wise byte-identical to the stored tags and refs, in the same order
- [ ] `tick show <id> --field title`, `--field description` and `--field notes.1` each print the stored value followed by exactly one newline
- [ ] Writing the decoded title and description back leaves the stored values byte-identical
- [ ] Re-adding the decoded note text stores a second note whose text is byte-identical to the first's, and the task then carries two notes
- [ ] Every fidelity assertion compares a stored value or a decoded value — none compares rendered output text
- [ ] A value carrying edge whitespace stores trimmed and stays trimmed across a write-back
- [ ] The test runs under `go test ./internal/cli` with no built binary and no network
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it stores the awkward title byte-identically"` — the persisted title equals the fixture constant
- `"it stores the awkward description byte-identically"` — the persisted description equals the fixture constant
- `"it stores the awkward note byte-identically"` — the persisted note text equals the fixture constant
- `"it stores the fixture tags and refs"` — the persisted `Tags` and `Refs` equal the fixture constants element for element
- `"it decodes the stored title out of the show document"` — decoded `title` equals the stored title
- `"it decodes the stored description out of the show document"` — decoded `description` equals the stored description
- `"it decodes the stored note text out of the show document"` — the first `notes` row's `text` equals the stored text, and its `index` reads 1
- `"it decodes the stored tags out of the show document"` — decoded `tags` equals the stored tags element for element
- `"it decodes the stored refs out of the show document"` — decoded `refs` equals the stored refs element for element, the URL's colon quoted by the library
- `"it returns the stored title bare from --field"` — stdout equals the stored title plus one newline
- `"it returns the stored description bare from --field"` — stdout equals the stored description plus one newline
- `"it returns the stored note text bare from --field notes.1"` — stdout equals the stored note text plus one newline
- `"it writes the decoded title back unchanged"` — after `tick update --title`, the persisted title is byte-identical
- `"it writes the decoded description back unchanged"` — after `tick update --description`, the persisted description is byte-identical
- `"it writes the decoded note back as a second note with identical text"` — `Notes[1].Text == Notes[0].Text` and the task carries two notes
- `"it trims edge whitespace on the way in and keeps it trimmed"` — a title and description created with surrounding whitespace store trimmed, and a write-back of the read value stores the same bytes

**Edge Cases**:
- The title and the note text both begin with a dash, so the fixture cannot be written at all without Tasks 1 to 3
- Trailing spaces sit on an interior line, not at the end of a value: edge whitespace is trimmed on the way in and stays trimmed, so a value carrying it could never round-trip and is not what the bar covers
- The description carries blank lines, a header-shaped line, embedded quotes and commas — each the shape a hand-built section mangled before Phase 1
- The fixture carries tags and refs as well as the three free-text carriers: a task carrying neither section exercises neither, and the old `tags[2]:` form with unmarked indented items makes the whole document undecodable, which is what read path A fails on
- A comma-bearing ref cannot reach storage — `task.ValidateRef` (`internal/task/refs.go:17-34`) rejects a ref containing a comma or whitespace, and `task.ValidateTag` (`internal/task/tags.go:24-35`) requires kebab-case — so the fixture's ref is a colon-bearing URL and the comma case stays the formatter-level assertion of Phase 1 task `free-text-round-trip-1-2`
- Tags and refs carry no write-back: they are not free-text carriers, and the byte-identity bar of §2.2 is stated over the title, the description and note text
- The title must stay single-line, since `ValidateTitle` rejects newlines; the multi-line content lives in the description
- Note text is capped at 2000 characters, which the fixture is far under
- The note is written back by adding it again, since notes carry no edit command, and the assertion is made against the new note's stored text
- Both read paths are exercised — the decoded document and `tick show --field` — because §2.1 makes neither a fallback for the other
- The assertion reads the stored value rather than output text, so a reformatting of the document cannot make the test pass or fail spuriously
- The write-back of the title needs no marker because its value follows a registered value-taking flag; the note write-back does need one

**Context**:
> §11, part 2: "One deliberately awkward task becomes a permanent fixture, round-tripped end to end. Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. It carries that text in all three free-text carriers — title, description and a note — each taking what its shape allows: the title and the note text begin with a dash, and the description carries the multi-line content. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets. A note is written back by adding it again, since notes carry no edit (§6.4), and the assertion is made against the new note's stored text. Whitespace at the very start or end of a value is trimmed on the way in and stays trimmed, which is why the fixture carries its trailing spaces mid-text. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability."
>
> §2.2: "Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was."
>
> §2.1: "An agent must be able to fetch one field bare, **and** must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback: the field flag (§9) does not excuse an ambiguous block in full output, and repairing the block does not remove the need for bare single-field output." Both paths are therefore asserted.
>
> §6.1 is what the tag and the ref on this task are here for: today's form writes "a header followed by one raw item per indented line", which "a TOON reader rejects… an item written on its own line carries a leading `- ` marker, and without it the decoder reports a length mismatch", and "the items are also written raw, so a ref containing a comma comes back as two values rather than one, and a URL's colon goes unquoted where the format's own rules would quote it".
>
> §9.2 fixes the bare form this test reads: "The value goes out as a line: its own bytes followed by a single newline." §9.3: "Asked for alone, `--field notes.2` prints that note's text bare, by the one-field rule."
>
> Fixture content, as Go string literals:
> ```go
> const fixtureTitle = `- read the header, carefully`
> const fixtureDescription = "Fix the parser.\n\nSteps:\n  - read the header   \n  - validate \"strictly\", then stop\nDone."
> const fixtureNoteText = `- retried "twice", then it stuck`
> var fixtureTags = []string{"round-trip", "fixture"}
> var fixtureRefs = []string{"https://example.com/issues/42"}
> ```
> The description's second bullet line ends in three spaces and is followed by two further lines, so the trailing whitespace is interior and survives; the value itself starts and ends on non-whitespace. The tags are kebab-case and the ref carries no comma or whitespace because the tool's own validation allows nothing else; the ref's colon is the character the old raw form left unquoted.
>
> Phase 1 added `decodeToonDoc` in `internal/cli/toon_decode_test.go`; `setupTickProject` and `readPersistedTasks` are in `internal/cli/create_test.go`; `runShow` in `internal/cli/list_show_test.go` constructs the `App` with `IsTTY: true`, so an explicit `--toon` is needed for the decoded read path.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §2.1, §2.2, §6.1, §6.4, §9.2, §9.3, §10.2

## free-text-round-trip-5-6

### Task 6: README and help present `--`

**Problem**: Tasks 1 to 3 made `--` work and left it undiscoverable. It appears nowhere in `README.md` and in no help text: `tick help`'s global-flags block (`internal/cli/help.go:257-264`), `tick help --all`'s one-line equivalent (`internal/cli/help.go:273`) and the `create` and `note` command descriptions all predate it. §12.1 owes the README the new input surface for the same reason it owed it corrected samples — "Calling `--` the canonical form only means something if a caller can find it written down, and a flag documented nowhere is a flag nobody uses."

**Solution**: Document the marker in both help global-flag blocks and in the `create` and `note` command descriptions, and add it to the README's Global Flags section with the rule a caller cannot guess plus one invocation example in each of the `create` and `note` sections. It stays out of every `Flags` list, because a help flag with no `commandFlags` entry fails the drift test.

**Outcome**: `tick help`, `tick help --all`, `tick help create`, `tick help note` and the README all present `--` as the way to pass free text that may begin with a dash, and `TestCommandFlagsMatchHelp` passes unchanged.

**Do**:
1. `internal/cli/help.go` — add a `--` entry to `printTopLevelHelp`'s `Global flags:` block, in the existing column alignment, describing it as the end-of-flags marker after which every argument is text.
2. `internal/cli/help.go` — extend `printAllHelp`'s single `Global flags:` line with `--` so the two lists agree.
3. `internal/cli/help.go` — extend the `create` and `note` `commandInfo.Description` strings with one sentence each naming `--` as the way to pass a title or note text that begins with a dash. Neither gets a `flagInfo`.
4. `README.md:564-576` — add `--` to the Global Flags fenced block and a sentence beneath it stating that flags come before the marker and everything after it is text, including an argument that spells a global flag; `README.md:86-110` and `README.md:245-257` — add one `--` invocation to the existing `bash` example block in each of the `create` and `note` sections.
5. `internal/cli/help_test.go` — add subtests asserting both help blocks name `--`, that they agree, and that `create` and `note` help mention it; `internal/cli/readme_samples_test.go` — add a subtest asserting the README's Global Flags block names `--`, and leave `TestREADMEToonSamplesDecode`'s anchor list unchanged.

**Acceptance Criteria**:
- [ ] `tick help` lists `--` in its global-flags block
- [ ] `tick help --all` names `--` on its global-flags line, and the two lists agree
- [ ] `tick help create` and `tick help note` both describe `--` as the way to pass dash-leading free text
- [ ] `--` appears in no `flagInfo.Name` and in no `commandFlags` entry, and `TestCommandFlagsMatchHelp` passes unmodified in both directions
- [ ] The README's Global Flags block lists `--` with a one-line description
- [ ] README prose states that flags come before the marker, that everything after it is text, and that this covers an argument spelling a global flag exactly
- [ ] The `create` and `note` sections each carry a `--` invocation example, written as a shell command with no output block
- [ ] No new fenced output block is added, so `TestREADMEToonSamplesDecode`'s anchors are unchanged
- [ ] The README does not contradict the `--field`/`--fields` documentation Phase 4 added, and does not present `--` as required
- [ ] The samples corrected in Phases 1, 2, 3 and 4 are byte-identical to what those phases left
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it names the end-of-flags marker in top level help"` — `tick help` stdout contains the `--` entry
- `"it names the end-of-flags marker in the all-commands help"` — `tick help --all`'s global-flags line contains `--`
- `"it keeps the two global flag lists in agreement"` — every flag named in `printTopLevelHelp`'s block appears in `printAllHelp`'s line
- `"it documents the marker in create help"` — `tick help create` mentions `--`
- `"it documents the marker in note help"` — `tick help note` mentions `--`
- `"it registers no flag for the marker"` — no `commandFlags` entry and no `flagInfo.Name` contains `--` as a flag name
- `"it documents the marker in the README global flags block"` — the README's Global Flags fenced block contains `--`
- `"it adds no new toon anchor"` — `TestREADMEToonSamplesDecode`'s anchor list is the one Phases 1 to 4 left, and every anchor still decodes

**Edge Cases**:
- `--` cannot appear in a command's help `Flags` list without being registered in `commandFlags`, or `TestCommandFlagsMatchHelp` fails in the help-to-registry direction — so it belongs in the global-flags block and the `create` and `note` descriptions, which the drift test does not read
- `printTopLevelHelp` and `printAllHelp` both carry a global-flags list and must agree, which is what the agreement test pins
- The README block is prose rather than a TOON sample, so it is not anchored to `TestREADMEToonSamplesDecode`
- `--field` documentation landed in Phase 4 Task 8 and must not be contradicted — the marker is documented beside it, not in place of it
- Earlier phases' samples stay untouched; this task adds prose and shell examples only
- `--` is the recommended form, not a required one: the documentation must not claim that dash-leading note text needs it, since Task 3 makes the bare form work for text that is not itself a global flag

**Context**:
> §12.1: "The README also gains the new input surface, for the same reason: `--field`/`--fields` on `show` (§9), and `--` as the way to pass free text that may begin with a dash (§10.2). Calling `--` the canonical form only means something if a caller can find it written down, and a flag documented nowhere is a flag nobody uses."
>
> §12.1: the README "is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect."
>
> §12.1 on why the help text carries it alongside, and on the constraint that shapes where: "The command's own help text carries both alongside — not a preference but a constraint: `--field` must be registered in `commandFlags` for §9.6's unrecognised-name error to fire at all, and `TestCommandFlagsMatchHelp` (`grep -n 'func TestCommandFlagsMatchHelp' internal/cli/flag_validation_test.go` → `flag_validation_test.go:310`) fails the suite when a registered long flag has no matching help entry." The marker takes the opposite side of the same constraint: it is registered nowhere, so it can appear in no `Flags` list.
>
> §10.2: "`--` is supported as the end-of-flags marker, and becomes the canonical documented way to pass free text that may begin with a dash… Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text. Free text that spells a flag exactly, a note reading `--json`, is writable for the same reason a dash-leading one is." And: "The existing bare-argument form keeps working. `--` is the recommended form, not a required one." And: "`create` cannot take the second half… `create` relies on `--`."
>
> `TestREADMEToonSamplesDecode` was added by Phase 1 task free-text-round-trip-1-6 in `internal/cli/readme_samples_test.go`: it collects every fenced block with an empty info string, strips a leading `$ tick …` prompt line, indexes the blocks by their first remaining line, and fails both when an anchor has no matching block and when `toon.DecodeString` rejects one. The `bash` example blocks this task extends carry an info string and are outside that collection.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §10.2, §9.1
