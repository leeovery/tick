# Phase 4: Field selection on `show` — 8 tasks

## free-text-round-trip-4-1

### Task 1: `--field` and `--fields` are registered on `show` and parsed

**Problem**: `show` accepts no command-specific flags at all (`internal/cli/flags.go:72`, `"show": {}`), and `RunShow` reads the task ID straight out of `args[0]` (`internal/cli/show.go:35-46`). There is no way to name a field, and `ValidateFlags` refuses `--field` as an unknown flag before the handler ever runs. Field selection is the case that started this work — an agent wanting one task's description as a plain string went to `.tick/tasks.jsonl` instead — and every other task in this phase projects from a selection that nothing yet produces. The flag also has to be registered in `commandFlags` for the unrecognised-name error to fire at all, and `TestCommandFlagsMatchHelp` (`internal/cli/flag_validation_test.go:310`) fails the suite when a registered long flag has no matching help entry.

**Solution**: Register `--field` and its alias `--fields` against `show` alone, add the matching help entry so the drift test passes, and add a parser that separates the task ID from a validated `FieldSelection` — recognised names only, optional 1-based positional suffixes on list sections, whitespace trimmed, repeats collapsed. Rendering is untouched: a recognised selection still produces today's full output until Task 2.

**Outcome**: `tick show <id> --field title,status` exits zero and prints the full document unchanged; an unrecognised name, a blank name, a missing flag value and `--quiet` alongside a selection each exit non-zero with nothing on stdout; `tick create --field title` still fails as an unknown flag.

**Do**:
1. `internal/cli/flags.go` — add `"--field": {TakesValue: true}` and `"--fields": {TakesValue: true}` to `commandFlags["show"]`.
2. `internal/cli/help.go` — in the `show` entry, set `Usage` to `tick show <task-id> [flags]` and add one `flagInfo` whose `Name` is `--field, --fields`, `Arg` is `<name,...>` and `Desc` describes selecting fields by name with an optional `.N` position (`TestCommandFlagsMatchHelp` splits `Name` on `", "` and matches both long forms against the registry).
3. `internal/cli/show_fields.go` (new file) — add `FieldSelection` carrying the selected names in first-seen order, the set of names taken whole, and a `map[string][]int` of 1-based positions, with methods `Selected(name string) bool`, `Positions(name string) []int` (nil when the name is taken whole), `Len() int` (distinct names) and `Only() (string, bool)`. Add the recognised-name registry in one place: scalars `id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`; `description`; list sections `notes`, `tags`, `refs`, `children`, `blocked_by`.
4. `internal/cli/show_fields.go` — add `parseShowArgs(args []string) (string, *FieldSelection, error)`. It walks `args`, treats `--field` and `--fields` as value-taking and accumulates every occurrence, takes the first argument that is neither a flag nor a flag's value as the task ID, and returns a nil selection when neither spelling appeared. A flag with no following argument returns `--field requires a value`. Each value splits on `,`; each name is `strings.TrimSpace`d; a name containing `.` splits at the first dot, where the base must be a list section and the suffix must parse through `strconv.Atoi`. Any other name — including the empty string and a base that is a scalar or unrecognised — returns `unknown field %q for "show". Run 'tick help show' for usage.` with the name exactly as typed. A bare section name marks the section whole and discards any positions already recorded for it; a position on a section already marked whole is discarded.
5. `internal/cli/show.go` — in `RunShow`, replace the `args[0]` read with `parseShowArgs(args)`, keep the existing `task ID is required` error when the parsed ID is empty, and return `--quiet cannot be combined with --field` when `fc.Quiet` is set and the selection is non-nil. Both run before `openStore`, so neither reaches stdout. Pass the selection no further in this task.

**Acceptance Criteria**:
- [ ] `--field` and `--fields` are both registered against `show` and against no other command; `tick create --field title` still fails with the unknown-flag error
- [ ] `TestCommandFlagsMatchHelp` passes with both spellings present in `show`'s help entry
- [ ] `tick show --field title tick-a1b2` and `tick show tick-a1b2 --field title` both resolve the same task — the flag's value is never mistaken for the ID
- [ ] `--field "title, status"` parses the same selection as `--field title,status`
- [ ] `--field title --field status` parses the same selection as `--field title,status`, and `--fields` composes with `--field`
- [ ] `--field title,title` yields a selection whose `Len()` is 1
- [ ] Every name in the registry parses whether or not the task carries it
- [ ] `--field titel`, `--field ""`, `--field "title,,status"`, `--field "title,"`, `--field notes.x`, `--field notes.1.2` and `--field title.1` each exit non-zero with the unrecognised-name error and nothing on stdout
- [ ] `--field` with no following argument exits non-zero with `--field requires a value`
- [ ] `--quiet` with any selection exits non-zero with nothing on stdout, and `--quiet` without one still prints the task ID
- [ ] `notes.2` parses as the `notes` section narrowed to position 2; `notes,notes.2` parses as `notes` taken whole
- [ ] A recognised selection produces stdout byte-identical to the same command without the flag
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it accepts a comma-separated list of field names"` — `--field title,status` parses to two names in that order
- `"it accepts the plural spelling"` — `--fields title` parses identically to `--field title`
- `"it composes repeated flags"` — `--field title --fields status` parses to both names
- `"it trims whitespace around names"` — `--field "title, status"` equals `--field title,status`
- `"it collapses a repeated name"` — `--field title,title` has `Len() == 1`
- `"it reads the task ID past the flag value"` — `tick show --field title tick-a1b2` resolves `tick-a1b2`
- `"it rejects an unrecognised field name"` — exit 1, stderr names `titel`, stdout empty
- `"it rejects a blank name from an empty value"` — `--field ""` exits 1 with the unrecognised-name error
- `"it rejects a blank name from a doubled comma"` — `--field "title,,status"` exits 1
- `"it rejects a trailing comma"` — `--field "title,"` exits 1
- `"it rejects a non-numeric position"` — `--field notes.x` exits 1 with the unrecognised-name error naming `notes.x`
- `"it rejects a position on a scalar"` — `--field title.1` exits 1 with the unrecognised-name error
- `"it rejects a flag with no value"` — `tick show tick-a1b2 --field` exits 1 with `--field requires a value`
- `"it recognises a name the task does not carry"` — `--field closed` on an open task exits 0
- `"it refuses quiet alongside a selection"` — `tick show tick-a1b2 --quiet --field title` exits 1, stdout empty
- `"it still prints the ID under quiet with no selection"` — unchanged behaviour
- `"it takes a section whole when named both whole and by position"` — `--field notes,notes.2` records no positions for `notes`
- `"it keeps the flag off other commands"` — `tick create X --field title` exits 1 with the unknown-flag error
- `"it renders full output for a recognised selection"` — stdout equals the same command without the flag

**Edge Cases**:
- `--fields` is an alias, not a second flag with its own meaning, but both spellings must be registered and both must appear in help or the drift test fails
- The flag takes a value, so `ValidateFlags` skips that value and `RunShow` must find the ID by walking the arguments rather than reading `args[0]`
- A repeated name collapses and counts once toward the one-versus-several split of Task 2
- A blank name — from an empty value, a doubled comma or a trailing comma — is the unrecognised-name mistake, not an empty field
- A non-numeric suffix, a suffix containing a second dot, and a position on a scalar all take the unrecognised-name error
- `tick show <id> --field --json` reports `--field requires a value`: global flags are consumed wherever they appear (`internal/cli/app.go:346-372`), so `--json` never reaches the handler as the flag's value
- Recognition never depends on presence — parsing happens before the store is opened, so a name on the registry is recognised whatever the task holds
- `notes.0` and `notes.-1` parse as positions and are not rejected here; they are the out-of-range error of Task 7, which needs the task's data
- `--quiet` with any selection is refused before the store opens, so nothing reaches stdout
- The flag stays `show`'s alone: `create`, `update`, `note add` and `note remove` emit the same document but take no selection

**Context**:
> §9: "`show` accepts no command-specific flags today (`grep -n '\"show\":' internal/cli/flags.go` → `flags.go:72`, `\"show\": {}`), so this is its first, alongside the global `--quiet`. It is `show`'s flag and no other command's. `create`, `update`, `note add` and `note remove` emit the same detail document but take no field selection: a caller that wants one value out of them runs `tick show --field` afterwards, and the flag stays registered against a single command."
>
> §9.1: "`--field` takes a comma-separated list of field names, and `--fields` is an alias of it. Both spellings work; the plural exists so the flag reads naturally when selecting several." And: "The names the flag accepts are the names the output document uses — the task's own top-level fields (`id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`) and the section keys (`description`, `notes`, `tags`, `refs`, `children`, `blocked_by`), spelled as a full `tick show` spells them. There is no second vocabulary to learn… A positional suffix (`notes.2`, §9.3) attaches only to a section that holds a list; on anything else the whole name is unrecognised and takes §9.6's error."
>
> §9.1 on lenience: "The list is read leniently wherever its meaning is not in doubt. Whitespace around a name is not part of it… Repeating the flag composes rather than overrides… A field named more than once renders once, and counts once when the answer splits by how many fields were asked for… A section named both whole and by position comes back whole… None of this softens §9.6: a name that is empty once its whitespace is gone is still the blank-name mistake." And: "Recognition does not depend on presence: a name on this list is always recognised, and asking for one the task does not carry is an empty field (§9.6). A name absent from the list is unrecognised whatever the task holds."
>
> §9.6: "An unrecognised field name is an error with a non-zero exit… and gets what every other unrecognised flag value already gets — `ValidateFlags` refuses rather than silently ignoring (`grep -n 'unknown flag %q for %q' internal/cli/flags.go` → `flags.go:138`). A selection that names nothing is the same mistake. An empty value, or a stray or doubled comma leaving an empty name in the list, fails with the unrecognised-name error rather than printing nothing or falling back to the full record." And: "A suffix that is not a number is no positional claim at all and takes the unrecognised-name error."
>
> §9.8: "Passing `--quiet` and a field selection together is refused. `--quiet` prints a bare task ID and nothing else; a single-field request prints that field's bare value and nothing else. Two different single values have been asked for, and silently picking one hands back something the caller did not ask for. The refusal is an error with a non-zero exit and nothing on stdout, exactly as an unrecognised field name is."
>
> §12.1 on why registration is a constraint rather than a preference: "`--field` must be registered in `commandFlags` for §9.6's unrecognised-name error to fire at all, and `TestCommandFlagsMatchHelp` fails the suite when a registered long flag has no matching help entry."
>
> The specification does not dictate the error wording. `unknown field %q for "show". Run 'tick help show' for usage.` is chosen to mirror `ValidateFlags`' existing `unknown flag %q for %q. Run 'tick help %s' for usage.` rather than introducing a second grammar, and `--field requires a value` mirrors `parseListFlags`' `--status requires a value` (`internal/cli/list.go:44`).

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9, §9.1, §9.6, §9.8, §12.1

## free-text-round-trip-4-2

### Task 2: One field returns its bare value

**Problem**: The case that started this work is an agent wanting a task's description as a plain string. Every route the tool offers hands back a document instead: `tick show` wraps it in sections, `--json` wraps it in JSON syntax the caller has to strip, and Task 1's selection parses but changes nothing. Until a single named field comes back as its own bytes with nothing around it, the agent that needs the text still has to decode something to get at it — which is what sent it to `.tick/tasks.jsonl` in the first place.

**Solution**: When the parsed selection names exactly one field and that field resolves to exactly one value, print the value's bytes followed by a single newline and return, bypassing the formatter entirely. A field with no value prints no bytes at all. Everything else — several names, or one name that is a list section — falls through to the document path, which still renders full output until Task 3.

**Outcome**: `tick show <id> --field description` prints the stored description raw and unindented followed by one newline, in every format and in none; `--field closed` on an open task prints nothing and exits zero.

**Do**:
1. `internal/cli/show_fields.go` — add `bareFieldValue(detail TaskDetail, sel *FieldSelection) (string, bool)`. It returns `ok == false` unless `sel.Len() == 1` and the single name resolves to one value: the scalars `id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`, and `description`. A name that is a list section returns `ok == false`. `priority` renders through `strconv.Itoa`; `created`, `updated` and `closed` render through `task.FormatTimestamp`; an absent `closed` returns the empty string with `ok == true`.
2. `internal/cli/show.go` — in `RunShow`, after `queryShowData` and `showDataToTaskDetail` and before the quiet branch, call `bareFieldValue`. When it reports `ok`, write the value through `fmt.Fprintln(stdout, value)` unless the value is empty, in which case write nothing at all, and return nil.
3. `internal/cli/show_fields_test.go` (new file) — unit-test `bareFieldValue` over each scalar, `description`, an absent optional, and a list-section name.
4. `internal/cli/list_show_test.go` — add end-to-end subtests through `runShow` covering each bare case, including under `--json`, `--pretty` and `--toon`, asserting stdout as exact bytes.

**Acceptance Criteria**:
- [ ] `--field description` prints the stored description followed by exactly one newline, with no indentation, no quoting and no header
- [ ] The same request under `--json`, under `--pretty` and under `--toon` produces byte-identical stdout
- [ ] A multi-line description goes out with its own newlines intact and no per-line prefix
- [ ] A value beginning with `- `, a value containing a colon, and a value that looks like a section header all go out unescaped
- [ ] `--field closed` on a task that is not closed, and `--field type` on a task with no type, print zero bytes and exit zero — not a blank line
- [ ] `--field id` on a partial-ID request prints the resolved full ID
- [ ] `--field title,title` takes the bare path
- [ ] `--field notes`, `--field tags`, `--field refs`, `--field children` and `--field blocked_by` do not take the bare path
- [ ] `--field title,status` does not take the bare path
- [ ] Every non-bare selection still renders the full document unchanged
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it prints a single field's value bare"` — `--field title` stdout equals the title plus `"\n"`
- `"it prints a multi-line description raw"` — stdout equals the stored description plus `"\n"`, byte-for-byte
- `"it prints a dash-leading value unescaped"` — a title reading `- read the header` comes out as those bytes
- `"it prints a header-shaped value unescaped"` — a description line reading `Steps:` is not quoted
- `"it ignores --json for a bare value"` — stdout under `--json` equals stdout under no flag
- `"it ignores --pretty for a bare value"` — same
- `"it ignores --toon for a bare value"` — same
- `"it prints nothing for an absent value"` — `--field closed` on an open task gives empty stdout and exit 0
- `"it prints nothing for an empty description"` — empty stdout, exit 0
- `"it prints the resolved id for a partial id request"` — `tick show a1b2 --field id` prints the full `tick-a1b2`
- `"it prints the priority as a number"` — `--field priority` prints `2` and a newline
- `"it treats a repeated name as one field"` — `--field title,title` prints the bare title
- `"it does not print a list section bare"` — `--field notes` still renders the full document
- `"it does not print two fields bare"` — `--field title,status` still renders the full document

**Edge Cases**:
- The bare form ignores `--json`, `--pretty` and `--toon` because it never reaches a formatter — a bare value is not a document, and honouring a format flag would re-quote the very string the flag exists to hand over unquoted
- Exactly one terminating newline, mirroring `fmt.Fprintln(stdout, id)` at `internal/cli/helpers.go:18`
- An empty or absent value prints no bytes at all rather than a blank line, and exits zero: the terminating newline belongs to a value
- A multi-line description goes out raw and unindented — the two-space prefix Phase 1 deleted from the toon block must not reappear here
- `--field id` prints the resolved full ID, because the detail is built from the resolved task
- A value beginning with a dash or looking like a section header goes out unescaped; making it writable again is Phase 5's `--` marker, not this task's concern
- `--field title,title` still counts as one field, by Task 1's dedup
- A single name of a list section is not bare — it returns the section, which Task 3 renders; until then it falls through to full output
- Values stored from this work onward carry no edge whitespace (§2.2), so the terminating newline is never ambiguous with the value's own bytes

**Context**:
> §9.2: "One field — the bare value. `tick show tick-a1b2 --field description`:
> ```
> Fix the parser.
>
> Steps:
>   - read the header
>   - validate
> ```
> The value goes out as a line: its own bytes followed by a single newline, as the bare task ID already is (`grep -n 'Fprintln(stdout, id)' internal/cli/helpers.go` → `helpers.go:18`). Values stored from this change onward carry no edge whitespace (§2.2), so that byte is the terminator and never part of the value."
>
> §9.2 on what is not bare: "A single field naming a list section returns that section, not a bare value… A list has no bare form, and the flag is a projection rather than a single-value extractor (§9.1), so the section is handed over in the one shape the library produces for it. The bare form belongs to a selection that resolves to exactly one value: the task's own fields, `description`, and a position that names one — `notes.2`'s text, `tags.1`'s item (§9.3)."
>
> §9.2 on the split being deliberate: "`--field description` and `--field description,notes` return different kinds of thing — a raw value versus a document — so an agent building the flag from a variable must know which it will get. It is the honest split between *fetch me this value* and *give me a trimmed record*, and collapsing them would cost the bare-value case the work exists to serve."
>
> §9.4: the task's own fields are selectable exactly like sections, and "§5.2 removed the wrapper, so the result has nothing around it."
>
> §9.6: "In the bare form that means no bytes at all: the terminating newline of §9.2 belongs to a value, so a field with no value produces an empty stream rather than a blank line."
>
> §9.7: "A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them. A bare value is not a document, so there is nothing for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted."
>
> §3.1 exempts this output from the must-parse inventory: "a bare value from `tick show --field` (§9.2), for the reason §9.7 exempts it from the format flags."
>
> Positions that resolve to one value (`notes.2`, `tags.1`) also take the bare path; they arrive in Task 6, which extends `bareFieldValue` rather than adding a second route.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.2, §9.4, §9.6, §9.7, §3.1

## free-text-round-trip-4-3

### Task 3: Several fields return a filtered toon document

**Problem**: A selection naming more than one field, or one field naming a list section, still renders the whole document. `ToonFormatter.FormatTaskDetail` (`internal/cli/toon_formatter.go:76-107`) assembles a fixed list of sections and has no way to be told that only some were asked for. That leaves the projection half of the flag unbuilt: an agent wanting the notes table plus the description gets every section, and an agent wanting just the notes table gets the same. §9.2 requires the document to be a full `tick show` minus the sections not asked for, in normal section order.

**Solution**: Carry the parsed selection on `TaskDetail` as an optional pointer and have the toon formatter emit only the selected pieces, keeping normal output order and each piece's existing presence rule. A selection that renders no bytes at all prints nothing, which the handler enforces by not writing an empty render.

**Outcome**: `tick show <id> --field description,notes` prints the notes section and the description and nothing else, in that order, decodable by the TOON library; a nil selection renders exactly what it renders today.

**Do**:
1. `internal/cli/format.go` — add `Fields *FieldSelection` to `TaskDetail`. Nil means the whole document.
2. `internal/cli/toon_formatter.go` — in `FormatTaskDetail`, gate each piece on `detail.Fields == nil || detail.Fields.Selected(name)`: the head block's nine scalars individually (keeping their existing non-empty guards for `type`, `parent` and `closed`), then `blocked_by`, `children`, `tags`, `refs`, `notes`, `changed` and `description` in today's order. Build the head block from only the selected scalars through `encodeToonFields`, and omit it entirely when none survive. Keep dropping empty strings from `sections` before joining with `"\n\n"`.
3. `internal/cli/show.go` — in `RunShow`, set `detail.Fields = sel` before formatting, and replace `fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))` with a render-then-print that writes nothing when the rendered string is empty.
4. `internal/cli/toon_formatter_test.go` — add filtered subtests decoding the output with `decodeToonDoc`: scalars only, sections only, a mix, an always-present section on a task carrying none, and a carried-only field on a task carrying none.
5. `internal/cli/list_show_test.go` — add end-to-end `--toon` subtests through `runShow` asserting the decoded document's keys and that no unselected key is present.

**Acceptance Criteria**:
- [ ] A filtered toon document decodes without error via `toon.DecodeString`
- [ ] The decoded document carries exactly the selected names as keys and no others — `id` included only when asked for
- [ ] Sections appear in normal output order regardless of the order the names were typed
- [ ] `--field notes` on a task with no notes prints `notes[0]{index,text,created}:`, and the same holds for `children` and `blocked_by`
- [ ] `--field tags` on a task with no tags prints nothing, and the same holds for `refs`, `description`, `type`, `parent` and `closed`
- [ ] A selection whose every name prints nothing produces zero bytes on stdout and exits zero
- [ ] A single name of a list section renders that section rather than a bare value
- [ ] Scalars and sections mix in one document with exactly one blank line between the head block and the first section
- [ ] No leading or trailing blank line appears when a selected field renders nothing
- [ ] A nil selection renders the document byte-identically to before this task, and `create`, `update`, `note add` and `note remove` never carry one
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders only the selected scalars"` — `--field title,status` decodes to a map with exactly those two keys
- `"it renders selected scalars in output order"` — `--field status,title` emits `title` before `status`
- `"it renders a selected section alone"` — `--field notes` decodes to a map whose only key is `notes`
- `"it mixes scalars and sections in output order"` — `--field description,notes,title` emits the head block, then notes, then description
- `"it does not carry id unless asked"` — `--field title` decodes to a map with no `id` key
- `"it renders a count-zero header for an always-present section"` — `--field notes` on a note-less task decodes to an empty `notes` list
- `"it renders nothing for a carried-only field the task lacks"` — `--field tags` on a tag-less task produces empty stdout and exit 0
- `"it renders nothing when every selected name is empty"` — `--field tags,refs` on a task carrying neither produces zero bytes and exit 0
- `"it has no leading blank line when the head block is empty"` — `--field notes,description` output starts with `notes[`
- `"it has no trailing blank line when the last selected field is empty"` — `--field notes,description` on a description-less task ends with the notes section
- `"it decodes a filtered document"` — end-to-end `--toon` output decodes and carries the expected values
- `"it leaves unfiltered output unchanged"` — a nil selection renders the string the existing full-document assertions expect

**Edge Cases**:
- Sections keep normal output order, not typed order, so the shape does not shift with how the flag was written
- Nothing rides along, `id` included — the caller passed the ID on the command line to make the request
- An always-present section selected on a task carrying none prints its count-zero header; a carried-only field prints nothing. Each keeps the presence rule full output already applies to it
- A selection whose every name prints nothing prints nothing at all and exits zero — it is nothing, not an empty document, so the handler must not `Fprintln` an empty string
- A single name of a list section returns the section rather than a bare value; Task 2's `bareFieldValue` already declines list sections
- Scalars and sections mix in one document — the selected head scalars form one block, joined to the sections by a blank line as the full head block is
- The filtered document must decode with the TOON library, which is what makes it a document rather than a trimmed block
- A nil selection renders full output unchanged; `create`, `update`, `note add` and `note remove` build their detail through `outputMutationResult` and never set the field
- The `changed` section is not a selectable name and `show` never carries one, so it is gated like the rest and never appears

**Context**:
> §9.2: "Several fields — the normal document with only those sections in it. `tick show tick-a1b2 --field description,notes`:
> ```
> notes[2]{index,text,created}:
>   1,Retried twice before it stuck,"2026-09-14T10:02:00Z"
>   2,"multi\nline\nnote","2026-09-16T08:30:00Z"
>
> description: "Fix the parser.\n\nSteps:\n  - read the header\n  - validate"
> ```
> Identical to a full `tick show` minus the sections not asked for. Sections keep their usual output order, not the order they were typed, so the shape does not shift with how the flag was written."
>
> §9.4: "The task's own fields are selectable individually. Exactly like sections. `tick show tick-a1b2 --field title,status`:
> ```
> title: Add retry to the sync worker
> status: in_progress
> ```
> Refusing — on the grounds that only whole sections are selectable — would make the grammar depend on which side of a boundary a name happens to sit, which is a rule to learn rather than read. §5.2 removed the wrapper, so the result has nothing around it."
>
> §9.5: "You get exactly the fields you named, in both forms… Scoping the rule — `id` present in the document form, absent in the bare form — was available and rejected as a second rule to learn. The caller passed the task's ID on the command line to make the request; handing it back tells them something they just typed."
>
> §9.6: "An empty field prints what full output prints for it, and exits successfully… A field or section full output omits when the task does not carry it (§5.2) prints nothing; a section full output always carries prints its count-zero header, so `--field notes` on a task with no notes returns `notes[0]{index,text,created}:`… The rule runs per name: in a multi-field selection each empty name contributes what it would contribute to full output, the rest of the document is unaffected, and a selection whose every name prints nothing prints nothing at all and still exits successfully."
>
> §3.1: "Every document `show` produces — the full detail, and a filtered one — decodes." And the exemption: "a field selection that prints no bytes at all (§9.6), which is nothing rather than an empty document, in every format."
>
> §5.2 fixes the presence rules this task must not disturb: "`children`, `blocked_by` and `notes` are always present, carrying a count-zero header when empty, while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them."
>
> §9.7: "A multi-field request produces a document, and so does a single field naming a list section (§9.2); the resolved format applies to either exactly as it applies to a full `tick show`."

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.2, §9.4, §9.5, §9.6, §9.7, §3.1, §5.2

## free-text-round-trip-4-4

### Task 4: JSON renders the filtered document

**Problem**: `--json` with a field selection still returns every key. `JSONFormatter.FormatTaskDetail` (`internal/cli/json_formatter.go:79-120`) marshals a fixed `jsonTaskDetail` struct, and a struct cannot express "only these keys": every field it declares is emitted, and the two that carry `omitempty` (`parent`, `closed`) vanish precisely when the caller selected them and the task does not carry them — collapsing the empty-but-selected case into absence. §9.7 requires the format flags to apply to a filtered document exactly as they apply to a full one, so a JSON consumer asking for two fields must get an object with two keys.

**Solution**: When the detail carries a selection, build the JSON object as a `map[string]any` populated only with the selected keys, each applying the presence rule full JSON output already applies to it, and return the empty string when no key survives so the handler prints nothing. Unfiltered output keeps the struct and is untouched.

**Outcome**: `tick show <id> --field title,notes --json` parses as one JSON object carrying exactly `title` and `notes`, with `notes` objects keeping their `index` field, and unfiltered `--json` output is byte-identical to before.

**Do**:
1. `internal/cli/json_formatter.go` — in `FormatTaskDetail`, keep the existing `jsonTaskDetail` path when `detail.Fields == nil`. Otherwise build a `map[string]any` adding each selected key from the same values the struct path uses: `id`, `title`, `status`, `priority`, `type`, `tags`, `refs`, `notes`, `description`, `created` and `updated` always when selected; `parent` and `closed` only when non-empty; `blocked_by` and `children` through `toJSONRelated` always when selected.
2. `internal/cli/json_formatter.go` — return `""` from that path when the map is empty, so `RunShow`'s empty-render guard from Task 3 prints nothing.
3. `internal/cli/json_formatter_test.go` — add filtered subtests unmarshalling into `map[string]any` and asserting exactly which keys are present, the values of each, and that `tags`/`refs` are non-nil empty slices when selected and empty.
4. `internal/cli/list_show_test.go` — add end-to-end `--json` subtests through `runShow` asserting the object parses, carries only the selected keys, and is empty stdout when every selected key would be omitted.

**Acceptance Criteria**:
- [ ] Filtered JSON output parses as exactly one JSON object
- [ ] The object carries exactly the selected keys — `id` included only when asked for
- [ ] Key order is stable across runs for the same selection
- [ ] `notes` objects carry `index`, `text` and `created`, with `index` the note's 1-based position
- [ ] `tags` and `refs` selected on a task carrying none render `[]`, never `null` and never absent
- [ ] `description` selected on a task with no description renders `""`, and `type` renders `""` — matching what unfiltered JSON carries for them
- [ ] `parent` and `closed` selected on a task carrying neither are absent, matching unfiltered JSON's `omitempty`
- [ ] A selection whose every key would be absent produces zero bytes on stdout and exits zero — not `{}`
- [ ] Unfiltered `--json` output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders only the selected keys"` — `--field title,status --json` unmarshals to a map with exactly those two keys
- `"it does not carry id unless asked"` — `--field title --json` has no `id` key
- `"it keeps the index on selected notes"` — `--field notes --json` note objects carry `index` 1 and 2
- `"it renders selected empty tags as an empty array"` — `--field tags,title --json` on a tag-less task gives a non-nil zero-length `[]any`
- `"it renders selected empty refs as an empty array"` — same for refs
- `"it renders a selected empty description as an empty string"` — the key is present with `""`
- `"it omits a selected absent parent"` — `--field parent,title --json` on a parentless task has no `parent` key
- `"it prints nothing when no selected key survives"` — `--field parent,closed --json` on a task carrying neither gives empty stdout and exit 0
- `"it parses as one object"` — `json.Unmarshal` of the whole stdout succeeds
- `"it keeps key order stable"` — two runs of the same selection produce identical bytes
- `"it leaves unfiltered json output unchanged"` — the existing full-document assertions still pass

**Edge Cases**:
- `jsonTaskDetail`'s fixed struct cannot express "only these keys", and `omitempty` on `parent`/`closed` collapses the empty-but-selected case — hence the map rather than a struct with more tags
- Key order must be stable; `encoding/json` sorts `map[string]any` keys, which satisfies stability without a second ordering mechanism
- `notes` keeps the `index` field Phase 1 added, because a consumer that asked for one note needs its real position before it can call `note remove`
- `tags` and `refs` stay `[]` and never `null` when selected and empty — the existing `make(...)` allocations carry that
- Unselected keys are absent entirely, `id` included
- The output parses as one JSON object; a selection that survives to zero keys prints nothing rather than `{}`, matching §3.1's "nothing rather than an empty document, in every format"
- Unfiltered JSON output is unchanged — `create`, `update`, `note add` and `note remove` never carry a selection, so their documents take the struct path
- Each key applies JSON's own full-output presence rule, which is not identical to toon's: JSON always carries `type`, `tags`, `refs` and `description`, where toon omits them when empty

**Context**:
> §9.7: "A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them… A multi-field request produces a document, and so does a single field naming a list section (§9.2); the resolved format applies to either exactly as it applies to a full `tick show`."
>
> §9.5: "You get exactly the fields you named, in both forms."
>
> §9.6: "An empty field prints what full output prints for it, and exits successfully… The rule runs per name: in a multi-field selection each empty name contributes what it would contribute to full output, the rest of the document is unaffected, and a selection whose every name prints nothing prints nothing at all and still exits successfully."
>
> §4.2: "Each note also carries its 1-based index, for the reason the toon table does (§6.3): a consumer that asked for one note (§9.3) needs the note's real position before it can call `note remove`, and that need is the same whichever format it parses."
>
> §11: "JSON output for each listed command is parsed and asserted by decoded value" and "No byte-level pinning is kept in the machine formats."
>
> The specification fixes that filtered JSON carries exactly the selected keys but says nothing about their order beyond the keys themselves; it is an open point. A `map[string]any` is chosen because it is the only shape that expresses both "only these keys" and "selected but empty", and Go's sorted map-key marshalling gives a deterministic order. Filtered key order therefore differs from unfiltered struct order, which the specification neither requires nor forbids.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.5, §9.6, §9.7, §4.2, §3.1, §11

## free-text-round-trip-4-5

### Task 5: Pretty renders the filtered document

**Problem**: Pretty is the format a terminal resolves to unless a flag overrides it, so `tick show <id> --field title,status` on a terminal is the common case — and `PrettyFormatter.FormatTaskDetail` (`internal/cli/pretty_formatter.go:117-184`) writes every label unconditionally. A caller asking for two fields gets the whole record with its header block, and §9.7 requires the filtered document in pretty to be the named fields in pretty's usual style and nothing else.

**Solution**: Gate each of pretty's pieces on the selection and assemble the output from groups — the header lines as one group, each block as its own — joined by a blank line, so an unfiltered render reproduces today's bytes exactly and a filtered one carries no leading or trailing blank line. A filtered record is output pretty does not produce today, so nothing existing changes.

**Outcome**: `tick show <id> --field title,status --pretty` prints two labelled lines and nothing else; unfiltered pretty output for every command is byte-identical to before, and its golden-string assertions are untouched.

**Do**:
1. `internal/cli/pretty_formatter.go` — restructure `FormatTaskDetail` to collect a `[]string` of header lines (`ID`, `Title`, `Status`, `Priority`, `Type`, `Tags`, `Parent`, `Created`, `Updated`, `Closed` in today's order and with today's padding) and a `[]string` of blocks (`Blocked by`, `Children`, `Refs`, `Notes`, `Description` in today's order), then join the header group with `"\n"`, and join that group and each non-empty block with `"\n\n"`. The cascade tail Phase 2 task `free-text-round-trip-2-4` appended to this function stays exactly as that task left it: when `detail.Changes != nil && len(detail.Changes.Blocks) > 0`, `"\n"` followed by each block rendered through `f.FormatCascadeTransition(block)` joined by `"\n"`, appended to the joined groups rather than added to either list. It is not a group and must not take the `"\n\n"` join — that would put a blank line into `create` and `update` output that is not there today.
2. `internal/cli/pretty_formatter.go` — gate each header line and each block on `detail.Fields == nil || detail.Fields.Selected(name)`, mapping each label to its selection name: `ID`→`id`, `Title`→`title`, `Status`→`status`, `Priority`→`priority`, `Type`→`type`, `Tags`→`tags`, `Parent`→`parent`, `Created`→`created`, `Updated`→`updated`, `Closed`→`closed`, `Blocked by`→`blocked_by`, `Children`→`children`, `Refs`→`refs`, `Notes`→`notes`, `Description`→`description`. Keep every existing non-empty guard as it is.
3. `internal/cli/pretty_formatter.go` — return `""` when no group survives, so `RunShow`'s empty-render guard from Task 3 prints nothing.
4. `internal/cli/pretty_formatter_test.go` — leave every existing golden-string assertion in place and add golden-string subtests for the filtered forms: scalars only, one block only, a mix, and a selection that renders nothing.
5. `internal/cli/list_show_test.go` — add end-to-end subtests through `runShow` (which runs with `IsTTY: true`, so pretty is the resolved format) asserting filtered stdout as exact strings.

**Acceptance Criteria**:
- [ ] Unfiltered pretty output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task, including blank-line spacing and column alignment
- [ ] `create` and `update` still print their cascade tree, appended after the joined groups with the single `"\n"` separator Phase 2 gave it
- [ ] A filtered pretty render carries no header block for unselected fields and no labels for them
- [ ] Selected fields keep pretty's label text, padding and alignment exactly as full output renders them
- [ ] `--field type` on a task with no type prints `Type:     -`, because that is what pretty's full output prints for it
- [ ] `--field tags` on a tag-less task prints nothing, and the same for `refs`, `parent`, `closed`, `blocked_by`, `children`, `notes` and `description`
- [ ] A selection that renders nothing produces zero bytes on stdout and exits zero
- [ ] A filtered render has no leading blank line when no header line survives and no trailing blank line when the last selected piece renders nothing
- [ ] A bare-value request never reaches the pretty formatter
- [ ] `internal/cli/pretty_formatter_test.go` carries every golden-string assertion it carried before, none removed or weakened
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders only the selected header lines"` — `--field title,status` output equals `"Title:    X\nStatus:   open"`
- `"it renders only the selected block"` — `--field notes` output equals the `Notes:` block with no header lines above it
- `"it mixes a header line and a block"` — `--field title,notes` output equals the title line, a blank line, then the notes block
- `"it renders a dash for a selected empty type"` — `--field type` output equals `"Type:     -"`
- `"it renders nothing for a selected field pretty omits"` — `--field tags` on a tag-less task gives empty stdout and exit 0
- `"it renders nothing when every selected field is empty"` — empty stdout, exit 0
- `"it has no leading blank line when no header line survives"` — `--field notes,description` starts with `Notes:`
- `"it leaves unfiltered pretty detail unchanged"` — the existing golden full-detail string still matches
- `"it leaves unfiltered pretty detail with cascades unchanged"` — the golden detail-plus-cascade string from Phase 2 still matches

**Edge Cases**:
- Unfiltered pretty stays byte-identical: today's output is one header group followed by blocks separated by `"\n\n"`, so the group-join restructure must reproduce it exactly, including the absence of a trailing newline
- The cascade tail is neither a header line nor a block: it is appended after the joined groups with a single `"\n"`, and a filtered render never carries one, because `--field` is `show`'s flag alone and `show` builds its detail with `Changes` nil
- No header block and no labels for unselected fields — the filtered render is the named fields in pretty's usual style and nothing else
- A selected field the task does not carry prints nothing wherever pretty's full output omits it; `Type` is the exception, because pretty's full output always prints it as `-`
- A filtered record is output pretty does not produce today, so §4.1's "pretty is unchanged" is not disturbed by adding it
- Pretty keeps golden-string assertions, so this task adds goldens rather than decoded checks — pretty has no parser and a decoded-value assertion does not exist for it
- A bare-value request never reaches pretty: Task 2 returns before the formatter is called
- `--quiet` with a selection is already refused, so pretty never sees a quiet filtered render

**Context**:
> §9.7: "In pretty — the format a terminal resolves to unless a flag overrides it — the filtered document is the named fields rendered in pretty's usual style and nothing else: no header block, no labels for sections outside the selection (§9.5). A filtered record is output pretty does not produce today rather than a change to output it does, so §4.1 stands untouched."
>
> §4.1: "What a terminal prints today is what it prints after this work — the single transition line, the box-drawing cascade tree, the indented description block, the dep-tree prose on empty branches. Pretty is the human surface; this work is about the agent surface."
>
> §9.6: "An empty field prints what full output prints for it, and exits successfully. A task legitimately having no description is a fact about the task rather than a failure of the command." Read per format: pretty's full output always prints `Type:     -` and omits `Tags:` when there are none, so a selection of each reproduces that.
>
> §11: "Pretty keeps golden-string assertions. Pretty output has no parser, so a decoded-value assertion does not exist for it; removing its golden strings would replace its only form of assertion with nothing."
>
> `runShow` in `internal/cli/list_show_test.go:29-41` constructs the `App` with `IsTTY: true`, so end-to-end show tests already resolve to pretty without a flag.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.7, §9.5, §9.6, §4.1, §11

## free-text-round-trip-4-6

### Task 6: Positions narrow the list section they name

**Problem**: Task 1 parses `notes.2` and every formatter ignores it — a positional selector renders the whole section. That leaves the selector the caller asked for unimplemented, and it is the one an agent reaching for a single note needs: without it, asking for "the third note" means reading the whole table. The narrowing also cannot be done by re-slicing: Phase 1's notes table numbers rows from their slice position (`buildNotesSection`, `internal/cli/toon_formatter.go`), so a naive `notes[2:3]` would render the third note as index 1 — and an agent acting on that index would call `tick note remove <id> 1` and delete the first note on the task, which is the exact failure §6.3 added the column to prevent.

**Solution**: Add one shared helper that maps a section's items and selected positions to the surviving items paired with their real 1-based positions, and call it from all three formatters' list sections so the count follows the selection while each note's row carries its true position. Extend the bare-value resolution so a position naming one scalar value comes back bare.

**Outcome**: `tick show <id> --field description,notes.2` prints `notes[1]{index,text,created}:` with a single row whose index reads `2`, alongside the whole description; `--field notes.2` alone prints that note's text bare; `--field children.1` alone returns a one-row `children` section.

**Do**:
1. `internal/cli/show_fields.go` — add `selectedItems[T any](items []T, positions []int) ([]T, []int)`. Nil positions returns every item paired with `1..len(items)`. Otherwise the positions are sorted ascending and deduplicated, any position outside `1..len(items)` is skipped, and the surviving items are returned paired with their 1-based positions.
2. `internal/cli/toon_formatter.go` — thread `detail.Fields.Positions(name)` into the notes, `blocked_by`, `children`, `tags` and `refs` sections: build each from `selectedItems`, and set each note row's `Index` from the returned position rather than from its loop counter.
3. `internal/cli/json_formatter.go` — in the filtered map path, narrow `notes`, `blocked_by`, `children`, `tags` and `refs` through `selectedItems`, taking each note object's `index` from the returned position.
4. `internal/cli/pretty_formatter.go` — narrow the `Blocked by`, `Children`, `Refs`, `Notes` and `Tags` renderings through `selectedItems`, discarding the positions since pretty renders no index.
5. `internal/cli/show_fields.go` — extend `bareFieldValue`: a single name that is `notes`, `tags` or `refs` with exactly one selected position resolves to that item's value (a note's text, a tag, a ref); `children` and `blocked_by` never resolve bare, and a name with more than one position never resolves bare.

**Acceptance Criteria**:
- [ ] `--field notes.2` inside a multi-field selection renders `notes[1]{index,text,created}:` with one row whose `index` decodes as `2`
- [ ] The narrowed section's count follows the selection while each note row carries its real position
- [ ] `--field tags.2` renders `tags[1]: <second tag>` — an inline list keeping its count and one item
- [ ] `--field children.1` renders the `children` table header and one row
- [ ] `--field notes.1,notes.3` renders two rows with indexes `1` and `3`, and `--field notes.3,notes.1` renders the same two rows in the same order
- [ ] A repeated position renders once
- [ ] `--field notes,notes.2` renders the whole notes section
- [ ] A position narrows only the section it names — every other selected field comes back whole
- [ ] `--field notes.2` alone prints the note's text bare with one terminating newline
- [ ] `--field tags.1` alone prints the tag bare; `--field children.1` alone returns its one-row section, not a bare value
- [ ] Narrowing applies identically in toon, JSON and pretty
- [ ] An unselected or unfiltered document renders every item, unchanged from before this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it narrows the notes section to one position"` — decoded `notes` has one row
- `"it keeps the real index on a narrowed note"` — that row's `index` decodes as `2`
- `"it narrows an inline list to one item"` — decoded `tags` has one element equal to the second stored tag
- `"it narrows a table to one row"` — decoded `children` has one row equal to the first child
- `"it narrows to several positions in output order"` — `notes.3,notes.1` decodes to indexes `1` then `3`
- `"it collapses a repeated position"` — `notes.2,notes.2` decodes to one row
- `"it returns the whole section when named both whole and by position"` — `notes,notes.2` decodes to every note
- `"it leaves other selected fields whole"` — `--field notes.2,tags` decodes with one note and every tag
- `"it prints a note's text bare for a lone position"` — `--field notes.2` stdout equals that note's text plus `"\n"`
- `"it prints a tag bare for a lone position"` — `--field tags.1` stdout equals the tag plus `"\n"`
- `"it returns a one-row section for a lone children position"` — `--field children.1` output is the section, not a bare value
- `"it narrows notes in json with the real index"` — `--field notes.2,title --json` gives one note object with `index` 2
- `"it narrows notes in pretty"` — `--field notes.2,title --pretty` prints one note line under `Notes:`
- `"it renders every item without positions"` — an unfiltered document is unchanged

**Edge Cases**:
- Narrowing must preserve each note's real 1-based index, so the notes slice cannot be re-sliced and renumbered — the helper returns the positions alongside the items and the row takes its index from them
- The section's count follows the selection while the row carries its real position, so `notes[1]` with a row reading `2` is correct rather than inconsistent
- An inline list keeps its count and one item (`tags[1]: plain`); a table keeps its header and one row
- Several positions on one section narrow to those items in output order, not the order they were typed
- A section named both whole and by position comes back whole — Task 1's parser already discards the positions in that case
- A position narrows only the section it names; every other field in the selection is untouched
- `notes.2` alone prints the note's text bare while `children.1` alone returns its one-row section, because a row is not a value
- The rule applies in toon, JSON and pretty — one helper, three call sites per format
- A position outside the section's range is skipped by the helper here; Task 7 rejects it before formatting, so the skip is never the observed behaviour once that task lands

**Context**:
> §9.3: "`notes.2` selects the second note. In a multi-field selection the section renders as normal, its count following the selection while each row carries its real position via the `index` column (§6.3) — `tick show tick-a1b2 --field description,notes.2`:
> ```
> notes[1]{index,text,created}:
>   2,"multi\nline\nnote","2026-09-16T08:30:00Z"
>
> description: "Fix the parser.\n\nSteps:\n  - read the header\n  - validate"
> ```
> A position narrows the section it names and nothing else: every other field in the selection comes back whole."
>
> §9.3 on the grammar: "The grammar is the same on every section that holds a list — `notes.2`, `tags.1`, `refs.2`, `children.3`, `blocked_by.1`. Positions are 1-based throughout, the section renders in its normal form narrowed to the named item (a table keeps its header and one row; an inline list keeps its count and carries one item)… Only notes carry a real position in the row itself, because only notes are addressed by position elsewhere in the CLI (§6.3). Asked for alone, `--field notes.2` prints that note's text bare, by the one-field rule."
>
> §9.3 on the rejected alternative: "to refuse filtering on notes altogether — `notes.3` returning the whole table, leaving position implicit in row order. That keeps the output minimal at the cost of the selector the caller asked for."
>
> §9.1: "A section named both whole and by position comes back whole, and several positions on one section render it narrowed to those positions in output order."
>
> §9.2: "A selection that resolves to a row rather than a value — `children.1` — comes back as its one-row section."
>
> §6.3 on why the index must stay truthful: "Without the column, an agent that asked for the third note alone sees `notes[1]` with one row, decides the note is wrong, runs `tick note remove <id> 1`, and deletes the first note on the task." And: "present whether the section is filtered (§9.3) or not."

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.3, §9.1, §9.2, §6.3

## free-text-round-trip-4-7

### Task 7: Out-of-range positions are an error

**Problem**: A position that names nothing currently narrows to nothing and succeeds. `--field notes.4` on a task carrying two notes prints `notes[0]{index,text,created}:`, `--field tags.1` on a tag-less task prints an empty list, and `--field notes.0` does the same — each handing back a successful empty answer to a claim about the data that is false. §9.6 requires all three to fail with a non-zero exit and a message naming the range, and requires that the grammar match the one `note remove` already uses for the same 1-based addressing (`internal/cli/note.go:128`, `index %d out of range: task has %d note(s)`) rather than inventing a second answer under a different command.

**Solution**: Validate every selected position against the task's real section lengths once the detail is loaded and before anything is written, returning an error naming the section, the position and the count. A section the task does not carry stays the empty-field success case when named whole and becomes this error when named by position.

**Outcome**: `tick show <id> --field notes.4` on a two-note task exits non-zero with `notes.4 out of range: task has 2 note(s)` on stderr and nothing on stdout, and `--field tags` on a tag-less task still exits zero.

**Do**:
1. `internal/cli/show_fields.go` — add `(s *FieldSelection) ValidatePositions(detail TaskDetail) error`. For each section carrying positions it compares every position against the section's length — `notes` against `len(detail.Notes)`, `tags` against `len(detail.Tags)`, `refs` against `len(detail.Refs)`, `children` against `len(detail.Children)`, `blocked_by` against `len(detail.BlockedBy)` — and returns on the first position outside `1..length`.
2. `internal/cli/show_fields.go` — format the error as `%s.%d out of range: task has %d %s`, where the trailing noun comes from a per-section table: `note(s)`, `tag(s)`, `ref(s)`, `child(ren)`, `blocker(s)`. Check sections in the document's normal order and positions in ascending order so the reported failure is deterministic.
3. `internal/cli/show.go` — in `RunShow`, call `ValidatePositions` immediately after `showDataToTaskDetail` and before the bare-value path, the quiet branch and any rendering, returning its error.
4. `internal/cli/show_fields_test.go` — unit-test the validator across each section, the zero and negative positions, and a section the task does not carry.
5. `internal/cli/list_show_test.go` — add end-to-end subtests asserting exit code, the stderr message and empty stdout, including a selection whose other names would have rendered.

**Acceptance Criteria**:
- [ ] `--field notes.4` on a two-note task exits non-zero with `notes.4 out of range: task has 2 note(s)`
- [ ] `--field notes.0` exits non-zero with the same grammar, reporting the task's real note count
- [ ] `--field notes.-1` exits non-zero the same way
- [ ] `--field tags.1` on a tag-less task exits non-zero with `tags.1 out of range: task has 0 tag(s)`
- [ ] `--field children.2`, `--field blocked_by.1` and `--field refs.3` fail the same way against their own counts and nouns
- [ ] Stdout is empty on every one of those, including when another name in the selection would have rendered
- [ ] `--field tags` on a tag-less task still exits zero and prints what full output prints for it
- [ ] `--field notes` on a note-less task still exits zero with its count-zero header
- [ ] A non-numeric suffix still takes Task 1's unrecognised-name error, not this one
- [ ] The first reported failure is deterministic for a selection carrying several out-of-range positions
- [ ] An in-range position still renders the narrowed section, in all three formats
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it rejects a position past the end of the notes"` — `notes.4` on two notes, exit 1, message names the range
- `"it rejects position zero"` — `notes.0`, exit 1
- `"it rejects a negative position"` — `notes.-1`, exit 1
- `"it rejects a position on a section the task does not carry"` — `tags.1` on a tag-less task, exit 1, `task has 0 tag(s)`
- `"it rejects an out-of-range children position"` — `children.2` with one child, message reads `child(ren)`
- `"it rejects an out-of-range blocked_by position"` — message reads `blocker(s)`
- `"it prints nothing on stdout when a position is out of range"` — `--field title,notes.4` gives empty stdout and exit 1
- `"it still succeeds for a whole section the task does not carry"` — `--field tags` on a tag-less task exits 0
- `"it still succeeds for a whole always-present section that is empty"` — `--field notes` on a note-less task exits 0 with the count-zero header
- `"it keeps the unrecognised-name error for a non-numeric suffix"` — `notes.x` reports the unknown-field error, not the range error
- `"it reports the first failure deterministically"` — `--field refs.9,notes.9` reports the same message on repeated runs
- `"it renders an in-range position normally"` — `--field notes.2` still works

**Edge Cases**:
- `notes.4` on a two-note task, `notes.0` where positions start at 1, and `tags.1` on a tag-less task all fail the same way — a selector that resolves to nothing is not a field that happens to be empty
- The message names the range as `note remove` already does, rather than inventing a second grammar for the same 1-based addressing
- `tags` alone on a tag-less task stays the empty-field success case, while `tags.1` is a claim that a first tag exists and is false
- Nothing reaches stdout even when other names in the selection would have rendered — validation runs before any output
- A non-numeric suffix is no positional claim at all and stays the unrecognised-name error of Task 1
- Validation needs the task's data, so it cannot run in the parser; it runs once the detail is built and before the bare-value path, so a bare request fails the same way
- `--quiet` with a selection is already refused, so there is no interaction to resolve here

**Context**:
> §9.6: "A position that names nothing is out of range whatever the reason. `notes.4` on a task carrying two notes, `notes.0` where positions start at 1, and `tags.1` on a task carrying no tags all fail the same way: a non-zero exit and a message naming the range, rather than printing nothing and succeeding. A selector that resolves to nothing is not a field that happens to be empty — the position is a claim about the data that is false. A section the task does not carry is not the empty-field case above either: `tags` alone on a tag-less task is a field that happens to be empty, while `tags.1` is a claim that a first tag exists. `tick note remove` already answers this for the same 1-based addressing, reporting the index as out of range and naming how many notes the task has (§6.3); giving the same grammar a different answer under a different command would be a second rule for a reader to learn. A suffix that is not a number is no positional claim at all and takes the unrecognised-name error (§9.1)."
>
> §9.3: "a position past the end is the error of §9.6 on any of them, naming the range the same way."
>
> `note remove`'s existing message is `index %d out of range: task has %d note(s)` (`internal/cli/note.go:128`). The specification requires the same grammar without fixing the exact string for a selector that names its section; `%s.%d out of range: task has %d %s` keeps the "out of range: task has N x" half verbatim and puts the selector as typed in front of it, so the caller sees the token they wrote.
>
> §9.6 on the empty-field case this must not swallow: "An empty field prints what full output prints for it, and exits successfully. A task legitimately having no description is a fact about the task rather than a failure of the command."

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §9.6, §9.3, §9.1, §6.3

## free-text-round-trip-4-8

### Task 8: README documents field selection

**Problem**: `--field`/`--fields` exists in the tool and nowhere in the README. Its `### show` section (`README.md:167-174`) documents `tick show <task-id>` and nothing else, so the flag that the whole phase exists to deliver is findable only by running `tick help show`. §12.1 owes the README the new input surface for the same reason it owes it corrected samples: a flag documented nowhere is a flag nobody uses, and the README is live documentation someone reads to learn the tool.

**Solution**: Document both spellings in the `### show` section with samples copied from the real binary — the bare single-value form, the filtered document form, and a positional selector — plus the rules a caller cannot guess: the one-versus-several split, normal section order, format-flag interaction, the `--quiet` refusal, and the two error cases. Anchor the filtered toon sample in the README decode test.

**Outcome**: The `### show` section documents `--field` and `--fields` with output copied from the tool, and `TestREADMEToonSamplesDecode` fails if the filtered toon sample stops parsing or disappears.

**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, create a task carrying a multi-line description and two notes, then capture real output for `tick show <id> --field description`, `tick show <id> --field description,notes` and `tick show <id> --field notes.2`.
2. `README.md:167-174` — extend the `### show` section: keep the existing `tick show <task-id>` usage block, add a flags line documenting `--field <name,...>` with `--fields` as its alias, list the accepted names (`id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`, `description`, `notes`, `tags`, `refs`, `children`, `blocked_by`), and state that a name may carry a 1-based position on any list section.
3. `README.md` — add the three captured samples beneath it, each under its own `$ tick show …` prompt line, and prose covering: one field returns the bare value while several return the document with only those sections in normal output order; a single name of a list section returns the section; the bare form ignores `--toon`/`--pretty`/`--json` while the document form honours them; `--quiet` with a selection is refused; an unrecognised name and an out-of-range position both exit non-zero.
4. `internal/cli/readme_samples_test.go` — add the filtered document sample's first line to the anchor list of `TestREADMEToonSamplesDecode`, and leave the two bare-value samples unanchored.
5. Confirm the scope boundary: the `tick show`, `tick list`, dep-tree, transition and cascade samples corrected in Phases 1 to 3 are byte-identical to what those phases left, and no `--` documentation is added.

**Acceptance Criteria**:
- [ ] All three samples were copied from real tool output rather than hand-written
- [ ] The filtered document sample is a decodable TOON document and is anchored in `TestREADMEToonSamplesDecode` by a first line unique among the README's fenced blocks
- [ ] The two bare-value samples are not anchored — neither is a TOON document
- [ ] Both `--field` and `--fields` appear, with the plural documented as an alias of the singular
- [ ] The documented accepted names match the registry Task 1 added, and the documented flag description does not contradict `tick help show`
- [ ] The one-versus-several split, normal section order, the format-flag interaction, the `--quiet` refusal and the two error cases are all stated
- [ ] The positional form is documented as 1-based and available on every list section
- [ ] `grep -n '^\$ tick show' README.md` finds the new samples and the section carries no invented output
- [ ] No `--` documentation is added
- [ ] The samples corrected in Phases 1, 2 and 3 are unchanged by this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it decodes the README filtered field sample"` — the newly anchored block parses as TOON
- `"it fails when an anchored sample is missing"` — the existing anchor-miss failure covers the new anchor
- `"it keeps the earlier anchors decoding"` — the `tasks[N]`, `id:`, `changed[N]` and `dep_tree[N]` anchors from Phases 1 to 3 still resolve and decode
- `"it documents both field flag spellings"` — `README.md` contains `--field` and `--fields`
- `"it matches the help text for show"` — every long flag in `show`'s help entry appears in the README's `### show` section

**Edge Cases**:
- Samples are copied from real tool output rather than hand-written — the defect this phase's README work exists to prevent is documentation describing output the tool does not produce
- The filtered toon document sample is added as an anchor to `TestREADMEToonSamplesDecode`; its first line must be unique among the README's fenced blocks or the test's first-line index collides with an existing sample
- The bare-value samples are not TOON documents and must not be anchored — §3.1 exempts bare output from the must-parse inventory
- The help text for both spellings landed in Task 1, and the README must not contradict it
- `--` documentation belongs to Phase 5 and is not added here
- The existing `tick show`, `tick list`, dep-tree, transition and cascade samples stay untouched
- A bare-value sample of a multi-line description spans several lines with no prefix; the surrounding prose must make clear those are the value's own bytes rather than a formatted block

**Context**:
> §12.1: "The README also gains the new input surface, for the same reason: `--field`/`--fields` on `show` (§9), and `--` as the way to pass free text that may begin with a dash (§10.2). Calling `--` the canonical form only means something if a caller can find it written down, and a flag documented nowhere is a flag nobody uses." The `--` half belongs to Phase 5; this task carries the field-selection half.
>
> §12.1: the README "is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect."
>
> §9.1: "`--field` takes a comma-separated list of field names, and `--fields` is an alias of it… The names the flag accepts are the names the output document uses… There is no second vocabulary to learn: what you read in the output is what you ask for."
>
> §9.2 supplies the shape of both samples: the bare value with nothing around it, and "the normal document with only those sections in it… Sections keep their usual output order, not the order they were typed."
>
> §9.7: "A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them."
>
> §9.8: "Passing `--quiet` and a field selection together is refused."
>
> §11: "Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes." That coverage is Phase 6's; this task only anchors the README's own sample.
>
> `TestREADMEToonSamplesDecode` was added by Phase 1 task free-text-round-trip-1-6 in `internal/cli/readme_samples_test.go`: it locates `README.md` via `testutil.FindRepoRoot(t)`, collects every fenced block with an empty info string, strips a leading `$ tick …` prompt line, indexes the blocks by their first remaining line, and fails both when an anchor has no matching block and when `toon.DecodeString` rejects one.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §9.1, §9.2, §9.3, §9.6, §9.7, §9.8, §3.1
