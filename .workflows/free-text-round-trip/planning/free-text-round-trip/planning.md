# Plan: Free Text Round Trip

## Phases

### Phase 1: Task detail document becomes conformant TOON
status: draft

**Goal**: The document that `show`, `create`, `update`, `note add` and `note remove` all emit decodes with a standard TOON reader — the task's own fields at the top level with no wrapping key, tags and refs as library-produced inline lists, the description as one quoted value, and notes carrying a 1-based index column.

**Why this order**: It is the document five of the eight listed commands produce, and its header is the line a reader fails on before it sees anything else. Every later phase consumes it: the `changed` table becomes a section inside it, field selection projects from it, the round-trip fixture reads through it. It also deletes the mechanism behind every malformed section — hand-assembled string building and header surgery — establishing the pattern the remaining phases follow.

**Acceptance**:
- [ ] `tick show` output decodes with the project's TOON library for a task carrying every optional field (type, parent, closed, tags, refs, description, notes, children, blocked_by) and for one carrying none of them
- [ ] The task's own fields decode as top-level keys with no wrapping key, and no code strips a `[1]` from the detail document's header
- [ ] `tags` and `refs` decode as library-produced inline lists; a ref containing a comma decodes back as one value
- [ ] `description` decodes as a single string byte-identical to the stored value, for multi-line text carrying leading spaces, a colon and a leading dash
- [ ] The notes section carries a leading 1-based `index` column, present whether the section is filtered or not; JSON note objects carry the same index
- [ ] Which sections a document carries — always-present versus carried-only — is unchanged from today
- [ ] Pretty output for the same tasks is byte-identical to before the change
- [ ] README's `tick show` sample matches real output, and the `tick list` sample's empty-type row shows the quoted empty string the formatter emits
- [ ] Toon and JSON assertions for this document check decoded values rather than output text

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-1-1 | Task fields become top-level named fields | task carrying none of type/parent/closed (presence rules unchanged), title containing a comma/colon/leading dash, timestamps emitted quoted, marshal-error fallback, blank-line section joining must still decode |
| free-text-round-trip-1-2 | Tags and refs use the library's inline list form | ref containing a comma decodes back as one value, tag containing a space, URL colon quoted by the library, single-item list, empty tags/refs still omit their section entirely |
| free-text-round-trip-1-3 | Description becomes one TOON-quoted value | blank lines inside the text, interior lines with leading spaces, a header-shaped line ("Steps:"), leading dash, embedded double quotes/tabs/CR, empty description still omits the section, decoded value byte-identical to the stored value |
| free-text-round-trip-1-4 | Notes carry a 1-based index column | count-zero header carries the index column, multi-line note text stays library-quoted, index matches the 1-based addressing `note remove` takes |
| free-text-round-trip-1-5 | Detail-document assertions check decoded values | pretty's golden-string assertions must stay in place, JSON full-document golden blobs in create/update/note tests, assertions on sections whose shape did not change |
| free-text-round-trip-1-6 | README samples match real output | list sample's empty type renders as a quoted empty string, show sample's section order must match the formatter's real order |

### Phase 2: Status changes become one `changed` table
status: draft

**Goal**: Every command that moves a status returns a single `changed[N]{id,title,from,to,auto}` table, and `create`/`update` return one document carrying that table as a section rather than a document with transition lines appended after it.

**Why this order**: Phase 1 leaves `create` and `update` emitting a valid document with foreign lines stuck on the end — still unreadable as a stream — and this phase closes that, which requires Phase 1's document to nest into. It also carries the riskier of the two shared-code splits: `baseFormatter.FormatTransition` is embedded by both the toon and pretty formatters, so restructuring the toon form without moving pretty means splitting it deliberately.

**Acceptance**:
- [ ] `start`, `done`, `cancel` and `reopen` emit only the `changed` table, and it decodes
- [ ] A task whose status moved twice in one command carries exactly one row, reading from its pre-command status to its post-command status; a task that ends where it started carries no row
- [ ] `create` and `update` emit a single decodable document with `changed` as a section inside it, present with count 0 when no task's status moved
- [ ] `update`'s two independent cascade blocks collapse into the single table
- [ ] `show`, `note add` and `note remove` carry no `changed` section
- [ ] JSON emits the same `changed` list in place of the current `transition` object beside `cascaded`
- [ ] Pretty's single transition line and box-drawing cascade tree are byte-identical to before the change
- [ ] README's arrow transition, JSON transition and cascade samples match real output

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-2-1 | Collapse a command's status movements into one changed set | a task moved twice by two cascades collapses to one row reading pre-command to post-command status, a task that ends where it started carries no row, every row auto=true when the caller asked for no status change, deterministic row order, an empty set is valid, pretty's tree inputs (head row, ParentID links) survive alongside the flat rows |
| free-text-round-trip-2-2 | Status commands emit the changed table in toon | a single transition with no cascades still renders a one-row table never a bare line, toon and pretty stop being byte-identical so the base-formatter equality assertion goes, a title containing a comma is library-quoted, auto renders as an unquoted boolean, quiet mode still prints nothing |
| free-text-round-trip-2-3 | JSON status output becomes the changed list | `changed` is `[]` and never null, `auto` is a JSON boolean, the `transition` and `cascaded` keys disappear entirely, single-transition and cascade branches produce one shape |
| free-text-round-trip-2-4 | The detail document carries a changed section | no section versus an always-present count-0 section must be distinguishable (nil versus empty), pretty's detail plus trailing transition line/tree stays byte-identical including blank-line spacing, the section's position in document order is fixed once and identical for create and update, JSON stops emitting a second top-level document |
| free-text-round-trip-2-5 | create and update emit one document | update's Rule 6 and Rule 3 blocks collapse into one table including when they meet on a shared ancestor, a shared ancestor that ends where it started carries no row, create with no parent carries `changed[0]`, quiet mode still prints only the task ID, pretty byte-identical with both blocks in today's order |
| free-text-round-trip-2-6 | README transition and cascade samples match real output | the `(unchanged)` marker in both the toon and pretty cascade samples was never implemented and goes rather than being reproduced, the pretty column stays a tree corrected to real output, sample rows must carry the new title column |

### Phase 3: Stats and dependency-tree documents
status: draft

**Goal**: The remaining two hand-built single-object headers become top-level named fields, a focused dep tree identifies its task exactly as a task-detail document does on both branches, and both empty branches return the populated document emptied instead of an English sentence.

**Why this order**: These are the last two malformed headers and the last prose answer to a query. They share no code with Phases 1 and 2, so the phase is self-contained; it follows them because it reuses the named-field form Phase 1 establishes, and because its risk profile differs — the empty-branch fix lands in the dep-tree handler rather than a formatter, and pretty prints nothing at all on that branch unless handed its sentence explicitly.

**Acceptance**:
- [ ] `tick stats` counts decode as top-level named fields beside the `by_priority` table
- [ ] `tick dep tree` chains/longest/blocked decode as top-level named fields beside the edges section
- [ ] `tick dep tree <id>` carries top-level `id`, `title` and `status` on both the populated and the empty branch
- [ ] Nothing blocked anywhere, and a named task with no dependencies either way, both emit the populated document emptied — summary fields reading zero, edge sections carrying count-zero headers — and both decode, in toon and JSON
- [ ] Pretty still prints `No dependencies found.` and `No dependencies.` on those branches
- [ ] No code path anywhere in the formatter strips a count from a section header
- [ ] README's dep-tree summary sample matches real output

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-3-1 | Stats counts become top-level named fields | a zero count must still be emitted (no omitempty), a project with no tasks emits all-zero counts plus by_priority's five rows, by_priority's shape and order unchanged, decoded numbers arrive as float64, no `stats` key in the decoded document, pretty and JSON stats unchanged |
| free-text-round-trip-3-2 | The full dep-tree summary becomes top-level named fields | zero-valued summary fields still emitted, no `summary` key in the decoded document, `DepTreeResult.Summary` prose stays for pretty, the emptied form is unreachable through the handler until task 4 so it is unit-tested on the formatter, `encodeToonSingleObject` deleted and no code path strips a count from a header |
| free-text-round-trip-3-3 | The focused dep tree names its task on both branches | the hand-built `id  title (status)` line and the `No dependencies.` early return go, both `blocked_by` and `blocks` always present with count-zero headers (changes today's omit-when-empty rule on the populated branch), the no-dependencies branch is the populated document emptied, a comma-bearing title or ID quoted by the library, pretty's focused view byte-identical |
| free-text-round-trip-3-4 | Nothing blocked anywhere returns the emptied document | pretty returns `""` for zero roots today and must be handed its sentence explicitly, pretty stdout byte-identical, a dependency cycle leaves zero roots with non-zero blocked/chains so counts are real not forced zeros, the dead `result.Message` guard in `ToonFormatter.FormatDepTree` goes, empty project and unconnected-tasks project both take the branch, `--quiet` prints nothing |
| free-text-round-trip-3-5 | JSON dep-tree documents drop their message forms | `roots`/`blocked_by`/`blocks` are `[]` never null, `omitempty` comes off both direction fields, `message` disappears from both dep-tree branches while `FormatMessage` stays for the prose commands, JSON's nested `target` is not flattened, `mode` stays |
| free-text-round-trip-3-6 | README dep-tree sample matches real output | the `summary{…}:` block becomes top-level `chains`/`longest`/`blocked` lines, sample copied from real output, the pretty companion block and its prose stay, the dep-tree sample is added as an anchor to `TestREADMEToonSamplesDecode`, no `tick stats` sample exists so none is added, earlier phases' samples untouched |

### Phase 4: Field selection on `show`
status: draft

**Goal**: `tick show --field`/`--fields` returns one field's value bare with nothing around it, or a document carrying only the named sections, with list sections addressable by position.

**Why this order**: It is the case that started the work, and it is a projection of the document Phase 1 defines — the names it accepts are the names that document uses, and the bare form exists only because Phase 1 removed the wrapper. It depends on nothing from Phases 2 and 3, since `show` carries no `changed` section, so it follows once every document it could project is conformant.

**Acceptance**:
- [ ] `--field` and `--fields` are registered against `show` alone with matching help text, and the flag/help drift test passes
- [ ] A selection resolving to exactly one value prints that value's bytes followed by a single newline — no header, indentation or quoting — and ignores `--json`, `--pretty` and `--toon`
- [ ] A multi-field selection, or a single field naming a list section, prints a document identical to full output minus the sections not asked for, in normal section order, honouring the format flags in all three formats
- [ ] Positional selectors work on every list section, narrow only the section they name, and leave every other selected field whole
- [ ] Nothing rides along unasked, `id` included, in either form
- [ ] An empty field prints what full output prints for it and exits zero; unrecognised names, blank names and out-of-range positions exit non-zero with nothing on stdout
- [ ] `--quiet` combined with a field selection is refused with a non-zero exit and nothing on stdout
- [ ] README documents `--field`/`--fields`

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-4-1 | `--field` and `--fields` are registered on `show` and parsed | `--fields` is an alias not a second flag but both must appear in help for the drift test, the flag takes a value so show's ID lookup must skip it rather than read args[0], whitespace around a name is trimmed and repeated flags compose, a repeated name collapses and counts once toward the one-versus-several split, a blank name from an empty value or a stray/doubled comma is the unrecognised-name mistake not an empty field, `--field` with no value at all, a non-numeric suffix and a position on a scalar take the unrecognised-name error, recognition never depends on presence, `--quiet` with any selection is refused with nothing on stdout, the flag stays show's alone so `tick create --field title` still fails as an unknown flag, a recognised selection still renders full output until task 2 |
| free-text-round-trip-4-2 | One field returns its bare value | the bare form ignores `--json`/`--pretty`/`--toon`, exactly one terminating newline mirroring `Fprintln(stdout, id)` at helpers.go:18, an empty or absent value prints no bytes at all rather than a blank line and exits zero, a multi-line description goes out raw and unindented, `--field id` prints the resolved full ID for a partial-ID request, a value beginning with a dash or looking like a section header goes out unescaped, `--field title,title` still counts as one field, a single name of a list section is not bare |
| free-text-round-trip-4-3 | Several fields return a filtered toon document | sections keep normal output order not typed order, nothing rides along including `id`, an always-present section selected on a task carrying none prints its count-zero header while a carried-only field prints nothing, a selection whose every name prints nothing prints nothing at all and exits zero, a single name of a list section returns the section rather than a bare value, scalars and sections mix in one document, the filtered document decodes with the TOON library, a nil selection renders full output unchanged and create/update/note never carry one, no leading or trailing blank line when a selected field renders nothing |
| free-text-round-trip-4-4 | JSON renders the filtered document | `jsonTaskDetail`'s fixed struct cannot express "only these keys" and `omitempty` collapses the empty-but-selected case, key order stable, `notes` keeps its index field and `tags`/`refs` stay `[]` never null when selected and empty, unselected keys absent entirely including `id`, the output parses as one JSON object, unfiltered JSON output unchanged |
| free-text-round-trip-4-5 | Pretty renders the filtered document | unfiltered pretty stays byte-identical, no header block and no labels for unselected fields, selected fields keep pretty's label and alignment style, a selected field the task does not carry prints nothing, a filtered record is output pretty does not produce today so nothing existing changes, pretty keeps golden-string assertions so this task adds goldens rather than decoded checks, a bare-value request never reaches pretty |
| free-text-round-trip-4-6 | Positions narrow the list section they name | narrowing must preserve each note's real 1-based index so the notes slice cannot be re-sliced and renumbered, the section's count follows the selection while the row carries its real position, an inline list keeps its count and one item and a table keeps its header and one row, several positions on one section narrow to those items in output order not typed order, a section named both whole and by position comes back whole, a position narrows only the section it names, `notes.2` alone prints the note's text bare while `children.1` alone returns its one-row section, the rule applies in toon, JSON and pretty |
| free-text-round-trip-4-7 | Out-of-range positions are an error | `notes.4` on a two-note task, `notes.0` and `tags.1` on a tag-less task all fail the same way, the message names the range as `note remove` already does rather than inventing a second grammar, `tags` alone on a tag-less task stays the empty-field success case while `tags.1` is not, nothing on stdout even when other names in the selection would have rendered, a non-numeric suffix stays the unrecognised-name error |
| free-text-round-trip-4-8 | README documents field selection | samples copied from real tool output rather than hand-written, the filtered toon document sample is added as an anchor to `TestREADMEToonSamplesDecode`, the bare-value sample is not a TOON document and must not be anchored, the help text for both spellings landed in task 1 and README must not contradict it, `--` documentation belongs to Phase 5, the existing show/list/dep-tree/transition samples stay untouched |

### Phase 5: Dash-leading free text and the whitespace invariant
status: draft

**Goal**: Text an agent reads out of tick can be written back — dash-leading titles and note text are accepted, and every write path trims edge whitespace so byte-identity holds across all three free-text carriers.

**Why this order**: It is the write half of the round-trip contract and the read half now exists: Phases 1 to 4 make the text readable, this makes it writable. The byte-identity fixture that proves the contract needs both halves and the dash fix specifically, since the fixture's title and note both begin with a dash.

**Acceptance**:
- [ ] `--` is accepted on every command as the end-of-flags marker; nothing after it is read as a flag, including an argument that spells a global flag exactly
- [ ] `tick create -- "- title"` and `tick note add <id> "- text"` both succeed, and dash-leading note text that is not itself a global flag works with or without the marker
- [ ] Every invocation that works today produces identical output and exit status, with one accepted exception recorded in task `free-text-round-trip-5-1`: `tick create --description -- x` now reports `--description requires a value`, because a `--` sitting where a value-taking flag's value belongs becomes the marker
- [ ] `tick migrate` trims edge whitespace from imported titles and descriptions; a whitespace-only value imports as empty and no import fails because of it
- [ ] Values already in storage are untouched — no pass rewrites stored records, and `show` emits stored bytes unmodified
- [ ] A permanent fixture task carrying newlines, quotes, commas, a leading dash, trailing spaces on an interior line and a header-shaped line — across title, description and a note — round-trips write → read → decode → write-back with the stored value byte-for-byte what it was
- [ ] README and the command help text present `--` as the canonical way to pass free text that may begin with a dash

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-5-1 | `--` ends flag parsing at the top level | the marker itself must not become a task ID or note text, a second `--` after the first is literal text, `--` with nothing after it, `--` appearing before the subcommand, global flags before the marker still apply while `--json`/`--quiet` after it are text, `doctor` and `migrate` validate through their own `ValidateFlags` calls at app.go:71 and app.go:78 and must take the same boundary, the boundary must be retained for task 2 rather than discarded with the marker, `--` is not registered in `commandFlags` so the help drift test is untouched, `--` is rejected on every command today so no existing invocation regresses |
| free-text-round-trip-5-2 | Post-marker text that spells a command flag is text | flags before the marker still parse normally, create's first-positional-wins title rule is unchanged so a second post-marker argument is still ignored, a post-marker `--field` on `show` is text rather than a selection, a post-marker value-taking flag must not swallow the argument after it, commands with no positional arguments are unaffected, `note add`'s text join must not pick the marker up |
| free-text-round-trip-5-3 | `note add` stops inspecting flags after the task ID | `note add` registers no flags so stopping the check loses nothing, `tick note add <id> --json` still resolves JSON because global flags are consumed before the command sees its args, note text that spells a global flag exactly still needs the marker, `note remove` keeps full validation so a mistyped flag there is still caught, the missing-ID and empty-text errors are unchanged, multi-word text still joins with single spaces |
| free-text-round-trip-5-4 | `tick migrate` trims imported free text | a whitespace-only description imports as empty and no import fails because of it, a whitespace-only title keeps today's validation error since `Validate` already trims for its check, interior whitespace and newlines are preserved, dry-run shares the Engine so its reported titles are the trimmed ones, one normalisation point so a provider that later carries note text inherits the rule, no pass rewrites stored records and `show` still emits stored bytes unmodified |
| free-text-round-trip-5-5 | The awkward-task fixture round-trips byte-identically | the title and note text begin with a dash so the fixture depends on tasks 1-3, trailing spaces sit on an interior line because edge whitespace is trimmed on the way in and stays trimmed, the description carries blank lines, a header-shaped line, embedded quotes and commas, the note is written back by adding it again and asserted against the new note's stored text since notes carry no edit, both read paths are exercised — the decoded document and `tick show --field` — and the assertion reads the stored value rather than output text |
| free-text-round-trip-5-6 | README and help present `--` | `--` cannot appear in a command's help `Flags` list without being registered in `commandFlags` or `TestCommandFlagsMatchHelp` fails in the help-to-registry direction, so it belongs in the global-flags block and the create/note descriptions, `printTopLevelHelp` and `printAllHelp` both carry a global-flags list and must agree, the README block is prose rather than a TOON sample so it is not anchored to `TestREADMEToonSamplesDecode`, `--field` documentation landed in Phase 4 task 8 and must not be contradicted, earlier phases' samples stay untouched |

### Phase 6: Conformance verification across the output inventory
status: draft

**Goal**: Every document the tool produces is decoded by a real TOON reader in the suite, on every branch, and no machine-format assertion pins output text.

**Why this order**: The coverage is counted in documents rather than commands, so it can only be enumerated once every document exists in its final shape. It is the guard that stops the next change re-introducing a hand-built section — the failure mode that produced this work in the first place.

**Acceptance**:
- [ ] Every command in the must-parse inventory has its output decoded by the TOON library in the suite, on every branch it can take, the emptied forms included
- [ ] Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes
- [ ] JSON output for each listed command is parsed and asserted by decoded value
- [ ] No toon or JSON test asserts against a pinned full-output string, with one retained set recorded in task `free-text-round-trip-6-6`: the count-zero section headers — `tasks[0]{…}`, `notes[0]{…}`, `changed[0]{…}`, `blocked_by[0]{…}`, `blocks[0]{…}`, `children[0]{…}` and `dep_tree[0]{…}` — stay as text assertions, because a decoder collapses them to the same empty list a bare `name[0]:` produces, and each sits beside a decoded assertion of the same empty section
- [ ] Pretty's golden-string assertions remain in place
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| free-text-round-trip-6-1 | Task-list documents decode through one conformance table | `runList` and friends resolve to pretty via `IsTTY: true` so the table must drive `--toon` explicitly, the empty branch emits the hand-written `tasks[0]{id,title,status,priority,type}:` literal and must decode to an empty list, an empty project and a filter matching nothing are two handler branches producing the same document, a row whose `type` is empty decodes as `""` rather than a shifted column, `--quiet` prints bare IDs and is declared excluded rather than silently omitted, pretty's `No tasks found.` branch keeps its golden assertion, `ready` and `blocked` are separate handlers and each needs its own rows, the driver names the failing document and prints its text, `toon_formatter_test.go`'s list golden strings become decoded-value assertions here rather than in task 6's sweep |
| free-text-round-trip-6-2 | Stats, dependency-tree and detail documents join the table | `stats` on an empty project emits all-zero counts plus `by_priority`'s five rows, the full dep tree's populated / nothing-blocked / cycle branches with the cycle carrying real non-zero counts beside an empty edge list, the focused dep tree's four branches (both directions, upstream only, downstream only, neither), `show`'s two detail branches (every optional field, and none), Phase 1's `TestToonTaskDetailConformance` and Phase 3's stats and dep-tree subtests keep their decoded-value assertions so the table adds enumerated parseability rather than replacing them, the emptied forms of §8 are documents rather than exemptions and must be rows, `dep tree --quiet` prints nothing and is not a document |
| free-text-round-trip-6-3 | Mutation and status documents complete the toon inventory | the guard must place every `commandFlags` key in exactly one of must-parse / prose (`dep add`, `dep remove`, `remove`, `init`, `rebuild`) / out of scope (`doctor`, `migrate`) so a newly added command fails the suite until declared, `ready` and `blocked` are registered by `init()` so the guard reads the map after initialisation, `create` with no parent carries `changed[0]` while `create --parent <done task>` carries rows, `update`'s four branches (no status movement, Rule 6 alone, Rule 3 alone, both meeting on a shared ancestor), `note add` and `note remove` carry no `changed` section and that absence is part of the shape, each of `start`/`done`/`cancel`/`reopen` with no cascade plus a cascading document for `done` and `reopen`, `--quiet` on a mutating command prints the bare ID and is not a document |
| free-text-round-trip-6-4 | Field-selection documents and the two exemptions are covered | a multi-field selection and a position-narrowed selection are documents while a bare value and a zero-byte selection are §3.1's two exemptions, the bare-value assertion compares exact bytes including the single terminating newline and holds identically under `--json`/`--pretty`/`--toon`, a selection whose every name prints nothing produces zero bytes which is nothing rather than an empty document, the exemptions are declared exempt in the inventory rather than absent from it, a single name of a list section is a document and belongs in the table, `--quiet` with a selection is refused and produces no document, the JSON counterparts of these forms belong to task 5 |
| free-text-round-trip-6-5 | JSON output is parsed and asserted by decoded value | one inventory drives both formats so a document present in one table and not the other is the drift this removes, JSON's presence rules differ from toon's (`type`/`tags`/`refs`/`description` always carried, `parent`/`closed` `omitempty`) so assertions cannot be copied across, lists unmarshal to non-nil empty slices never `null` (`changed`, `roots`, `blocked_by`, `blocks`, `tags`, `refs`), `auto` is a JSON boolean and `index` a number, each document must parse as exactly one JSON object which is what proves no command emits two concatenated documents, `stats` JSON keeps its own nested `{total, by_status, workflow, by_priority}` shape which §4.2 does not move, pretty is not in the table because it has no parser |
| free-text-round-trip-6-6 | No toon or JSON assertion pins a full-output string | pretty's goldens stay including `No tasks found.`, `No dependencies found.`, `No dependencies.`, the transition line and the cascade tree, a substring probe used to tell one format from another (`format_integration_test.go`) is a pinned shape and becomes a decoded-key check, an assertion on one value inside a decoded document is not a pin so only whole-output comparison goes, `TestREADMEToonSamplesDecode` stays because it decodes rather than compares, `--quiet` and bare-value byte assertions stay because that output is not a document, the sweep's result is checked by a stated grep so it can be re-run |
