# Analysis Tasks: Free Text Round Trip (Cycle 3)

## Task 1: The toon encoder's refusal emits a document that denies the data

severity: high
sources: architecture

**Problem**: The encoding seam this work built has exactly one error path and it substitutes a well-formed falsehood for the failure. `encodeToonFields` (`internal/cli/toon_formatter.go:268-274`) returns `""` when `toon.MarshalString` rejects a value, and `joinToonSections` (`:277-286`) then drops the section — so one refused value deletes the entire top-level header block from the document. `encodeToonSection` (`:329-336`) returns `name[0]:` on the same error — a count-zero header asserting the section holds nothing while it holds rows. Both are silent, both exit 0, and both produce documents that parse, so §11's conformance drivers pass on them however many branches the inventory covers. The trigger is any C0 control character other than tab, newline or carriage return — an ANSI escape inside pasted terminal output is the everyday way to get one — and nothing on any write path rejects or normalises it: `create`, `update`, `note add` and `tick migrate` all store it, and no test in the suite carries such a value, the awkward fixture included. Reproduced against a binary built from the working tree, ordinary CLI input only, and confirmed independently by a second reproduction:

- one task whose title carries an escape makes `tick list` print `tasks[0]:` — exit 0, the whole project reading as empty — while `tick list --json` returns both tasks;
- `tick show <id>` on that task prints a detail document with no `id`, no `title` and no `status` at all, only `blocked_by[0]`, `children[0]` and `notes[0]`, so the document identifies no task;
- a description carrying an escape vanishes from the detail document entirely, no `description:` line, while `tick show <id> --json` returns it intact;
- `tick note add <id> $'bell \x07 note'` then `tick show <id>` prints `notes[0]:`, which `--field notes` and the `notes.N` range check then agree with.

The failure is noticed as an agent acting on absent data — re-adding a description the task already holds, concluding a project is empty, or going back to `.tick/tasks.jsonl` — and never as an error. That is the exact failure class §1 says this work exists to delete, and it inverts §2.2's bar on the values it hits.

**Solution**: Stop substituting. Carry the encoder's error out of the three helpers rather than discarding it, so `FormatTaskList`, `FormatTaskDetail`, `FormatStats`, `FormatCascadeTransition`, `FormatDepTree` and the section builders surface it and the command exits non-zero naming the task and the field, exactly as an unresolvable ID already does. Whether the error travels as a second return value on the `Formatter` methods or as state on `ToonFormatter` is the author's choice; the settled invariant is that no path emits a count, a section or a document that contradicts the data it was handed, and that a refusal is a non-zero exit with a diagnostic rather than a silent omission. The direction is derived, not judged: toon cannot carry a C0 codepoint at all, so no encoding choice preserves the value, leaving a diagnostic or an announced substitution — and an announced substitution breaks §2.2's byte-identity on the read-back path, which is the same trade §2.2 declines when it refuses to write the exception down. Refusing the input at the write path is a separate, additive change and does not remove this one: values already stored, and values arriving through `tick migrate`, still have to read back honestly. Extend the awkward fixture's carriers (`internal/cli/round_trip_test.go:15-19`) with one refused character so the round-trip assertion pins the behaviour chosen.

**Outcome**: Every toon document either carries the data it was handed or the command fails loudly naming what it could not encode; no toon path reports a count of zero over rows that exist, and no detail document omits the fields that identify its task. JSON and pretty output are unaffected — both already carry these values — so the contradiction between formats closes from the toon side.

**Do**:

- Make the refusal a value the callers can see. `encodeToonFields` (`internal/cli/toon_formatter.go:266-272`) and `encodeToonSection` (`:327-334`) hold the tree's only two encoder calls (`grep -rn 'toon\.MarshalString' --include='*.go' .` → exactly those two lines): return the encoder's error instead of `""` and `fmt.Sprintf("%s[0]:", name)`, and carry it through every one of their twelve call sites (`grep -n 'encodeToonFields(\|encodeToonSection(' internal/cli/toon_formatter.go` → fourteen lines, twelve calls plus the two definitions) — `buildTaskSection`, `buildRelatedSection`, `buildNotesSection`, `buildChangedSection`, `buildEdgeSection` and `joinToonSections`. Leave `emptyToonSection` and `joinToonSections`'s skip of an empty string as they are: a section that genuinely holds nothing and a section that was refused must stay distinguishable, and `buildTaskSection`'s `""` for a selection that keeps no top-level field is the former.
- Carry it out of the five toon methods that build documents — `FormatTaskList`, `FormatTaskDetail`, `FormatStats`, `FormatCascadeTransition`, `FormatDepTree` — as a second return value on the `Formatter` methods or as state on `ToonFormatter`, whichever the executor prefers (the second-return route also reaches `StubFormatter` and `PrettyFormatter.FormatTaskDetail`'s internal call at `pretty_formatter.go:236`). `PrettyFormatter` and `JSONFormatter` emit the bytes they emit today.
- Turn the seven document print sites into failures: `list.go:228`, `show.go:84`, `helpers.go:30` (`outputMutationResult`, behind `create`, `update` and both `note` paths), `helpers.go:100` (`outputStatusChanges`), `dep_tree.go:39`, `dep_tree.go:57`, `stats.go:93` — the document-producing subset of `grep -rn 'fmtr\.Format' internal/cli --include='*.go' | grep -v _test` (twelve sites; the other five are `FormatMessage`, `FormatRemoval` and `FormatDepChange`, plain text with no encoder, untouched). Each returns the error to `App.Run`, which already prints `Error: %s` to stderr and exits 1; stdout stays empty on that path. `outputStatusChanges` (`helpers.go:99`) returns nothing today — give it an error and handle it at its one caller, `transition.go:55`.
- Put the two facts an agent needs into the message: the field or section that was refused, and the task's ID wherever the document covers a single task. The library supplies neither — its error reads `toon: unsupported control character U+001B in string`.
- Move the tests off the substituted forms and onto the behaviour chosen: `toon_formatter_test.go:844` asserts `encodeToonFields` returns `""` on a rejected value, and `:850` names the empty-head case as an encoding failure when what it now exercises is the no-selected-field case. Extend the awkward fixture (`round_trip_test.go:15-19`) with carriers holding one refused character — a title, a description and a note text — and pin what happens to each: stored byte-identically, returned byte-identically by bare `--field`, and refused with a non-zero exit on the document path.
- Add a short paragraph to the README's `### TOON` section (`README.md:460`): which characters TOON cannot carry, that a command carrying one fails naming the task and the field rather than printing a document without it, and that `--json` returns the value.

**Acceptance Criteria**:

- [ ] With a C0 control character other than tab, newline or carriage return stored in any free-text carrier, no toon document is printed that omits or under-counts it: `tick show`, `tick list`, `tick dep tree <id>`, `tick create`, `tick update`, `tick note add`, `tick note remove` and the four transition commands exit 1 with a stderr diagnostic and write nothing to stdout.
- [ ] The diagnostic names the field or section that could not be encoded, and names the task's ID for every document that covers a single task.
- [ ] `tick stats` and the full-graph `tick dep tree` carry no free text — integers, IDs and the fixed priority table only — and their output is byte-unchanged.
- [ ] `--json` and `--pretty` over the same data are byte-unchanged: both exit 0 and carry the value.
- [ ] `--quiet` paths exit 0 and print IDs as they do today, since no document is encoded on them: `create --quiet`, `update --quiet`, `list --quiet`, `show --quiet` and the transitions under `--quiet`.
- [ ] `tick show <id> --field title|description|notes.1` returns a refused value bare and byte-identically at exit 0 — the bare path never reaches the encoder.
- [ ] A mutation whose document is refused still commits: the new or updated record is in `.tick/tasks.jsonl` and only the rendering fails.
- [ ] A section that genuinely holds no rows still renders its `name[0]{cols}:` header, and a detail document narrowed to a selection that keeps no field still exits 0 printing nothing.
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass, the conformance inventory included, and no test asserts a substituted document.

**Tests**:

- `"it fails naming the task and the field when the stored title cannot be encoded"`
- `"it fails rather than printing an empty task list when one task's title cannot be encoded"`
- `"it fails rather than printing a count-zero notes section when a note's text cannot be encoded"`
- `"it fails when the focused dep tree target's title cannot be encoded"`
- `"it writes nothing to stdout when the document is refused"`
- `"it stores the note when the document confirming it is refused"`
- `"it returns the refused title bare from --field at exit zero"`
- `"it returns the refused description intact under --json at exit zero"`
- `"it prints only the ID under --quiet for a task whose title cannot be encoded"`
- `"it returns the encoder's error from encodeToonFields"`
- `"it returns the encoder's error from encodeToonSection"`
- `"it keeps the emptied section header when the section holds no rows"`
- `"it omits the head rather than emitting a blank line when no top-level field is selected"`
- `"it leaves stats output unchanged since it carries no free text"`
