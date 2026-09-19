# Consolidation Tasks: free-text-round-trip (Phase 4)

## Task 1: Help Flags Print Their Description In Its Own Column
placement: phase 4
severity: behaviour

**Problem**: `tick help show`'s only Flags line reads `  --field, --fields <name,...>Select fields by name; a section may be narrowed with .N (e.g. notes.2)` — the argument spec and the description fused with no separator, so `show`'s entire Flags block is garbled. `tick help --all`, the agent discovery surface, prints the same fused line. Both renderers format with `"  %-24s%s\n"` (`internal/cli/help.go:286` and `:308`), which loses the gap entirely for any label wider than 24 columns; this phase's label is 28. The same overflow already hits `create`/`update`'s `--type <bug|feature|task|chore>` (31 columns) and `list`'s `--status` (42), but `show` has one flag line and this phase created it. §12.1 makes the help text a deliverable of this work, and a flag has to be readable to be discoverable at all.

**Solution**: Give the flag column a two-space minimum gap by computing the column width from the widest label in the set being printed, rather than a fixed 24. The two renderers duplicate both the label construction and the format string; extract one helper and change it once. Rendered help for `create`, `list`, `remove`, `migrate` and `--all` shifts with it, so the help expectations across the suite move too — that reach beyond the phase diff is expected and is why this is a consolidation task rather than a task-time fix.

**Outcome**: Every command's help prints each flag's description separated from its label, whatever the label's width, and one helper owns the column.

**Do**:
- Add one helper to `internal/cli/help.go` that writes a `[]flagInfo` block: it builds each label (`f.Name`, plus `" " + f.Arg` when `f.Arg` is non-empty), takes the column from the widest label in that slice plus two spaces, and writes each line as `"  %-*s%s\n"`.
- Call it from `printAllHelp` in place of `:281-287` and from `printCommandHelp` in place of `:303-309`, both passing `cmd.Flags`, so the two renderers emit the same block for a command and `%-24s` disappears from the file.
- Size the column per command's own flag set with no fixed floor, so a block narrower than today's 24 narrows: `create`/`update` go to 33 (`--type <bug|feature|task|chore>`, 31), `list`/`ready`/`blocked` to 44 (`--status <open|in_progress|done|cancelled>`, 42), `show` to 30 (`--field, --fields <name,...>`, 28), `remove` to 13 (`--force, -f`, 11), `migrate` to 19 (`--from <provider>`, 17).
- Leave the top-level listing's `"  %-14s%s\n"` command column (`:257`) and the hand-written global-flags lines (`:261-267`) alone — they are not flag blocks.
- Add the help tests below; the existing `help_test.go` assertions are substring matches on flag names and usage lines (`grep -n 'strings.Contains' internal/cli/help_test.go`), so they stay green without edits.

**Acceptance Criteria**:
- [ ] `tick help show` prints `  --field, --fields <name,...>  Select fields by name; a section may be narrowed with .N (e.g. notes.2)` — two spaces between label and description.
- [ ] `tick help --all` prints that same line for `show`.
- [ ] Every flag line in every command's help has at least two spaces between its label and its description, `list`'s 42-column `--status` and `create`'s 31-column `--type` included.
- [ ] The flag block a command renders under `tick help <cmd>` is byte-identical to the block it renders inside `tick help --all`.
- [ ] Label construction and the flag-line format string exist once in `internal/cli/help.go`; `grep -n '%-24s' internal/cli/help.go` returns nothing.
- [ ] A command with no flags still prints no `Flags:` section, and `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass.

**Tests**:
- `"it separates every flag label from its description"` — over every entry in `commands`, each flag line of `tick help <name>` has two or more spaces after its label
- `"it prints the show field flag with its description in its own column"` — the exact `--field, --fields` line, asserted byte-for-byte
- `"it renders the same flag block in --all as in per-command help"` — for every command carrying flags
- `"it sizes the column to the widest label in the command's flag set"` — `create` at 33, `remove` at 13
- `"it prints no flags section for a command with no flags"` — `tick help init`

## Task 2: Filtered JSON Returns Its Keys In The Document's Order
placement: phase 4
severity: behaviour

**Problem**: The filtered JSON document is built as a `map[string]any` (`internal/cli/json_formatter.go:119-158`), so `encoding/json` sorts its keys alphabetically. Measured: `--json --field type,tags` returns `tags` then `type` while full output returns `type` then `tags`; `--field title,notes.2` returns `notes` then `title`. Toon and pretty both preserve document order under a selection, so the three formats disagree, and `README.md:191` — written by this phase — tells the reader a filtered document comes back "in normal output order rather than the order they were typed". A caller diffing a filtered document against a full one gets a reordering nothing warned them about. Nothing catches it: the JSON tests assert key *sets*, and the subtest named "it keeps key order stable" compares two identical renders, which map marshalling satisfies for free. The same fork also encodes §9.6's per-key rule twice — key presence, the omit-when-absent behaviour for `parent`/`closed`, and the empty-array and empty-string shapes are written once at `:91-107` and again at `:128-152`, so a change to one is silent in the other.

**Solution**: Render JSON from one ordered key table, gated per key, so the full and filtered paths share both the order and the per-key rule. An ordered emission replaces the map — the shape still has to express "only these keys" and "selected but empty", which is what the map was chosen for, so the replacement keeps both properties rather than reverting to a struct. Pin the resulting order in a test rather than only its stability. Settled by §9.7 rather than by preference: the resolved format applies to a filtered document "exactly as it applies to a full `tick show`", and a full `tick show --json` returns document order — which also makes the README sentence true of all three formats instead of two.

**Outcome**: A filtered JSON document carries its keys in the same order the full document does, one table decides both, and a test fails if either drifts.

**Do**:
- Replace both halves of the fork in `internal/cli/json_formatter.go` — the `jsonTaskDetail` marshal at `:84-115` and the `map[string]any` builder at `:119-158` — with one ordered emission walked in the order `jsonTaskDetail` declares: `id, title, status, priority, type, tags, refs, notes, description, parent, created, updated, closed, blocked_by, children, changed`. Marshal through a type whose `MarshalJSON` writes its members in slice order so `marshalIndentJSON` still yields today's 2-space indentation.
- Gate each key once through the selection: a nil `detail.Fields` keeps every key, and the list sections narrow with `selectedItems(..., detail.Fields.Positions(name))`, which is nil-safe (`show_fields.go:173-178`) and so serves the unfiltered path with the same call.
- Write each key's rule once in that table: `parent` and `closed` emitted only when the task carries them, `tags`/`refs`/`notes`/`blocked_by`/`children` always arrays (`[]` when empty), `description` an empty string rather than omitted, and `changed` only on a document that carries changes and no selection (`json_formatter_test.go:1760` pins it out of a filtered document).
- Keep `""` as the output when a selection leaves no key (`json_formatter_test.go:1746`, `list_show_test.go:1064`).
- Add a test helper that reads a document's top-level keys in emitted order from the raw bytes (`json.Decoder.Token()`), and use it for the order assertions below; leave `assertJSONKeySet` (`json_formatter_test.go:1776`) in place for the presence assertions that already call it.

**Acceptance Criteria**:
- [ ] `tick show <id> --json --field type,tags` returns `type` before `tags`; `--field title,notes.2` returns `title` before `notes`; `--field status,title,id` returns `id`, `title`, `status`.
- [ ] For any selection, the filtered document's key order is the full document's order restricted to the selected keys.
- [ ] Full `tick show --json` output is byte-identical to today's, `changed` last when a mutation carries it.
- [ ] Each key's presence rule is written once: one `parent`/`closed` omit-when-absent guard, one empty-array shape, one `description` guard; `map[string]any` no longer appears on `FormatTaskDetail`'s path.
- [ ] The README's "in normal output order rather than the order they were typed" (`README.md:191`) is true of JSON as well as toon and pretty.
- [ ] The existing key-set, shape and position tests in `json_formatter_test.go` and `list_show_test.go` pass unedited apart from the order subtest below.

**Tests**:
- `"it returns a filtered document's keys in document order"` — `type,tags`, `title,notes.2` and `status,title,id`
- `"it orders a filtered document as the full document restricted to the selection"` — driven off the full document's key order
- `"it renders a full document's keys in the declared order"`
- `"it keeps json key order in document order across runs"` — replaces `list_show_test.go:1072`'s identical-render comparison, which map marshalling satisfied for free
- `"it omits parent and closed from a filtered document when the task does not carry them"`
- `"it prints nothing when no selected key survives"`

## Task 3: Each Field Is Declared Once In The Registry
placement: phase 4
severity: drift

**Problem**: Three separate declarations describe the same fields, and two gates answer the same question differently. `showFields[…].items` (`internal/cli/show_fields.go:45-49`, added by task 4-6) and `showSections` (`:54-64`, added by task 4-7) both enumerate the five list sections, but only the second range-checks positions: a section added to one alone is accepted with a positional suffix and never checked, so `--field <section>.99` silently returns an empty or whole section instead of §9.6's out-of-range error, or a lone position returns the section instead of the bare value. Neither divergence is a compile error and coverage is written per section, so nothing fails. Separately, `showFieldScalar` and `showFieldDescription` are never compared anywhere — only `showFieldUnknown` and `showFieldList` are, and `bareFieldValue` branches on `bare == nil` rather than on kind — so a maintainer keying a later rule on `showFieldScalar` silently excludes `description`, the sole member of an identical kind. And `Selected` (`:160-162`, exported, panics on a nil receiver) and `includes` (`:166-168`, nil-safe) are two gates over one question: `TaskDetail.Fields` is nil for every document `create`, `update`, `note add` and `note remove` produce, so a formatter path reaching for the exported half — the natural one to reach for — turns a mutation command into a runtime panic.

**Solution**: Fold `noun` and `length` onto `showField` for the list kinds and derive the position-validation walk from one ordered slice of list-section names, so a section is declared once and the document-order property the error message depends on stays explicit. Collapse `showFieldDescription` into `showFieldScalar`; with list-versus-not-list the only remaining distinction and recognition driven by the map lookup's `ok`, the kind reduces to a field on `showField` and the `showFieldUnknown` zero-value convention goes with it. Keep one nil-safe gate: make `Selected` nil-safe and delete `includes`, routing every gate site through it, or keep `includes` and remove the exported half — nothing outside package `cli` uses `FieldSelection`, so the export buys nothing.

**Outcome**: A list section is declared in one place with its items, its length and its noun; recognition is a map lookup; and one nil-safe gate serves every formatter, so a nil selection cannot panic.

**Do**:
- Fold `noun` and `length` onto `showField` for the five list entries and delete `showSections` (`internal/cli/show_fields.go:54-64`); drive `ValidatePositions` (`:277-290`) off one ordered slice of list-section names in document order — `blocked_by`, `children`, `tags`, `refs`, `notes` — so the order the error message depends on stays one explicit declaration.
- Collapse `showFieldDescription` into `showFieldScalar`, reduce the kind to list-versus-not on `showField`, and drive recognition off the map lookup's `ok` in `addValue` (`:220` for a bare name, `:226` for a positional suffix), retiring `showFieldUnknown` and its zero-value convention (`:12-19`).
- Keep exactly one nil-safe gate: delete the exported `Selected` (`:160-162`), route its callers through `includes` (`:166-168`), and convert every site — `rg -n '\.Selected\(' internal/cli` → `json_formatter.go:123` plus `show_fields_test.go:112`, `:129`, `:137`, `:167`; nothing outside package `cli` refers to `FieldSelection` (`rg -n 'FieldSelection' --glob '!internal/cli/**'` → no hits).
- Leave `bareFieldValue`'s `bare == nil` branch (`:124`) and `barePositionValue`'s `items == nil` branch (`:133-135`) as they stand — with the kind gone, those accessors carry the distinction they already tested for.
- Change no rendered byte and no error message: `TestValidatePositions`'s table (`show_fields_test.go:650-661`), `notes.3 out of range: task has 2 note(s)` included, stays exact.

**Acceptance Criteria**:
- [ ] `showSections` no longer exists; each list section's `noun`, `length` and `items` sit on its single `showFields` entry.
- [ ] `ValidatePositions` walks one ordered list of section names, so a list section added to the registry is range-checked without a second edit.
- [ ] `showFieldDescription` and `showFieldUnknown` are gone; `description` and the nine other single-value fields share one kind, and an unrecognised name is detected by the map lookup failing.
- [ ] One gate answers "does this name belong in the document", and it is nil-safe; no exported half survives to be reached for.
- [ ] A `TaskDetail` with nil `Fields` — what `create`, `update`, `note add` and `note remove` produce — renders in toon, pretty and JSON without panicking.
- [ ] Behaviour is unchanged: `go test ./...` passes with no test file edited beyond the four `Selected` call sites, and `go vet ./...` plus `golangci-lint run ./...` are clean.

**Tests**:
- Existing, unchanged in meaning and expected to pass as written: `TestValidatePositions`, `TestParseShowArgs`, `TestShowFieldFlag`, `TestBareFieldValue`, `TestSelectedItems` (`internal/cli/show_fields_test.go`), `TestShowFieldPositions` (`internal/cli/list_show_test.go:1083`), and the toon, pretty and JSON formatter suites
- `"it renders a document with no field selection in every format"` — the one new test: nil `Fields` through `ToonFormatter`, `PrettyFormatter` and `JSONFormatter`, pinning the surviving gate as nil-safe

## Task 4: The Formatters Gate Off The Registry's Names
placement: phase 4
severity: drift

**Problem**: The fifteen field names exist as five independent sets of bare string literals with nothing tying them together — the registry (`internal/cli/show_fields.go:34-50`), the toon gates (`internal/cli/toon_formatter.go:75-103` and the `add(...)` calls at `:257-274`), the JSON gates (`internal/cli/json_formatter.go:128-152`), the pretty gates (`internal/cli/pretty_formatter.go:151-174` and `:186-210`), and the test at `internal/cli/show_fields_test.go:117-133` whose subtest is named "it recognises every registered name" but iterates a hardcoded list rather than `showFields`. A name added to or respelled in the registry is accepted by the parser and silently unrendered by any formatter whose literal was not updated in the same edit: `tick show <id> --field <name>` prints zero bytes and exits 0 in that format while another format prints the field, with no compile error and no failing test. The user reads the empty output as "this task has no value for that field" — the human-surface/agent-surface divergence §9.7 exists to close. The drift is already present in the inverse direction: `internal/cli/toon_formatter.go:99` gates on `"changed"`, a name no registry entry can select, while JSON drops that section under a selection and pretty appends it regardless — three answers for one section key.

**Solution**: Give the fifteen names one ordered set of constants shared by the registry and all three formatter gate lists, so a typo is a compile error rather than silence. Drive the registry test off `showFields` itself instead of its hardcoded copy. Add one table-driven parity assertion: for every name in the registry, a fixture carrying every field renders non-empty output in toon, pretty and JSON — which is what turns a registry name with a missing gate into a test failure. Settle `changed` by dropping the toon gate rather than registering the name: `changed` is not selectable, and only `show` carries a selection while only the mutation commands carry changes, so the two never meet and the gate decides nothing.

**Outcome**: One declaration of the vocabulary, a compile error for a typo, and a parity test that fails when a registry name goes unrendered in any format.

**Do**:
- Declare the fifteen names once in `internal/cli/show_fields.go` as an ordered constant set (`fieldID`, `fieldTitle`, `fieldStatus`, … `fieldBlockedBy`) and key `showFields` (`:34-50`) off the constants.
- Convert every gate and selector literal in the three formatters to the constants — measured `rg -n 'includes\("|Selected\("|Positions\("|add\("' internal/cli/*_formatter.go` → 60 lines (toon 21, json 19, pretty 20). Convert all 60, not the sections this phase touched. Leave the output-vocabulary literals alone: toon's `buildRelatedSection`/`buildEdgeSection` section names, the `json:"…"` struct tags and pretty's display labels are the document's own spelling, not gates.
- Drop the `sel.includes("changed")` gate at `internal/cli/toon_formatter.go:99`, leaving the section gated on `detail.Changes != nil` alone: `changed` is not a registry name, and only `show` carries a selection while only the mutation commands carry changes, so nothing reaches the gate. Delete the subtest that pinned it, `"it never carries the changed section"` (`internal/cli/toon_formatter_test.go:1425-1430`), which builds a selection-plus-changes document no command produces; JSON's own `changed` gate and its test (`json_formatter_test.go:1760`) are out of scope here.
- Rewrite the subtest at `internal/cli/show_fields_test.go:117-133` to iterate `showFields` rather than its hardcoded copy of the fifteen names, so the guard checks what its name claims.
- Add one table-driven parity test over `showFields`: for each registered name, render `richDetail()` (`internal/cli/toon_formatter_test.go:1286`, which carries every section and every optional scalar) with only that name selected through `ToonFormatter`, `PrettyFormatter` and `JSONFormatter`, asserting non-empty output from each.

**Acceptance Criteria**:
- [ ] Each of the fifteen names is spelled once as a gate literal — in the constant declaration — with the registry and all three formatters referring to the constants; none of the 60 measured lines carries a bare name literal.
- [ ] A misspelt name in the registry or in any formatter gate fails to compile.
- [ ] The parity test fails when any one formatter gate is removed — checked by hand against toon, pretty and JSON in turn, then restored.
- [ ] The recognition subtest iterates `showFields`, so a name added to the registry is covered with no test edit.
- [ ] `tick show <id>` output is byte-identical before and after in all three formats, filtered and full, and a mutation command still renders its `changed` section in toon.
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass.

**Tests**:
- `"it recognises every registered name"` — the existing subtest, now driven off `showFields`
- `"it renders every registered name in every format"` — the table-driven parity assertion over toon, pretty and JSON
- `"it renders the changed section for a mutation document"` — pins that dropping the toon gate leaves mutation output untouched
- `"it leaves unfiltered output unchanged"` and the filtered toon, pretty and JSON suites — existing, expected green, the deleted `changed` subtest aside

## Task 5: Every README Sample Is Pinned To Real Output
placement: phase 4
severity: drift

**Problem**: The two bare-value samples this phase added to the README — `$ tick show tick-a1b2 --field description` (`README.md:185-189`) and `$ tick show tick-a1b2 --field notes.2` (`:204-207`) — appear in no `readmeSampleGroup`, while every other `$ tick` fence in the file is compared byte-for-byte against real rendered output. A later change to the bare-value path (a trailing newline, a prefix, quoting, a different resolution for a position) leaves both blocks describing output the tool no longer produces and the whole suite stays green, which is precisely the defect §12.1 names the README work as existing to prevent. Task 4-8 was instructed to leave them out, and phase 6's conformance task pins the bare value's bytes against the tool but never against the README, so no later task closes this. The gap is invisible to a reader of the test file: nothing distinguishes "deliberately exempt" from "forgotten".

**Solution**: Add both fences to `readmeSampleGroups` against the existing field-selection fixture, which already carries the description and the two notes the samples print — `readmeSample` compares a fence body to stdout byte-for-byte and its format field is inert for a bare value, which is itself the §9.7 claim worth pinning. Then close the class rather than the instance: assert that every `$ tick`-prompted fence in the README is claimed by some sample, with the one error-output fence (`README.md:634`) declared as a named exemption. Settled rather than raised: which guard shape to use is test structure, and the class-closing form is what stops the next unpinned sample arriving the same way this one did.

**Outcome**: Every `$ tick` sample in the README is rendered and compared against real output, and a new sample that is not is a test failure rather than a silent gap.

**Do**:
- Add a `readmeSampleGroup` to `readmeSampleGroups` (`internal/cli/readme_samples_test.go:321-361`) carrying both bare-value samples against the existing `fieldSelectionTasks` fixture (`:296-315`): `show tick-a1b2 --field description` anchored on first line `Full task description here.`, and `show tick-a1b2 --field notes.2` anchored on `Blocked on the migration landing`, each with an empty `info` and `occurrence: 1`.
- Give both the format the neighbouring samples use (`--toon`); it is inert for a bare value, and the samples passing under it is the §9.7 claim the fences are worth pinning for.
- Keep the prompt line `fenceBody` strips (`:102-110`) on `readmeFence` (`:41-44`) so a guard can tell a prompted fence from an unprompted one, leaving `readmeFences`, `readmeToonBlocks` and `findREADMEBlock` behaviour as they are.
- Add a guard that every prompted fence is claimed by exactly one `readmeSample`, matching on `info`, first line and occurrence, with one named exemption table holding the unknown-flag error fence (`README.md:634`, `$ tick list --stauts open`); an unclaimed fence fails naming its prompt line, and an exemption matching no fence fails too.
- Measured today (`grep -c '^\$ tick' README.md` → 12): nine fences are already claimed (README.md lines 194, 327, 340, 435, 447, 512, 522, 556, 567), two are the ones this task adds (186, 205), one is the exemption (634) — the guard must pass over all twelve with no README prose or sample bytes edited.

**Acceptance Criteria**:
- [ ] `tick show tick-a1b2 --field description` renders the body of the fence at `README.md:185-189` byte-for-byte from the seeded fixture, and `--field notes.2` renders the body of the fence at `README.md:204-207`.
- [ ] Each of the two new samples resolves to exactly one README fence.
- [ ] Every `$ tick`-prompted fence in the README is claimed by a sample or by the single declared exemption.
- [ ] Adding an unclaimed `$ tick` fence fails the suite with a message naming its prompt line — checked by hand, then reverted.
- [ ] An exemption naming no fence fails the suite, so the exemption cannot outlive the sample it covers.
- [ ] README content is unchanged; the diff touches `internal/cli/readme_samples_test.go` only, and `go test ./...` passes.

**Tests**:
- `"it reproduces the README bare description sample"`
- `"it reproduces the README bare note position sample"`
- `"it claims every prompted README sample"`
- `"it fails when a prompted sample is claimed by nothing"`
- `"it fails when an exemption names no fence"`
