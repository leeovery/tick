# Specification: Free Text Round Trip

## Specification

### 1. Purpose

`tick`'s toon output is the agent-facing format — the default whenever stdout is not a TTY. It does not parse. A standard TOON reader fails on the first line of `tick show` output and never reaches anything beneath it:

> `line 1: invalid unquoted key "task{id,title,status,priority,type,created,updated}"`

The failure that triggered this work was an agent abandoning the CLI and reading `.tick/tasks.jsonl` directly to obtain a task's description as a plain string. The file told it the truth without a decoding rule it had to already know.

The cause is uniform. Every section the formatter assembles by hand is malformed; every section it hands to the TOON library is correct.

| Written by | Sections | Parses |
|---|---|---|
| The library (`encodeToonSection`) | blockers, children, notes, priority breakdown, dep-tree edges | yes |
| Hand-assembled string building | task header, stats summary, dep-tree summary, tags, refs, description | no |

Each hand-rolled section exists because it wanted a shape the library does not produce directly — a singular object header, a list down the page, an unstructured text block. In every case the invented shape turned out to be invalid.

**This work delivers toon output that a standard TOON reader can read, for every command that returns data, and free text that survives an agent's read-edit-write cycle without a rule learned outside the output.**

Measured against `github.com/toon-format/toon-go@v0.0.0-20251202084852` (`grep toon-go go.mod`). Whether the TOON specification itself permits a single-object scope header was not verified — the claim is only that the reference Go implementation rejects it, and that the form is hand-constructed by tick rather than produced by the library.

### 2. Round-Trip Contract

#### 2.1 Both reading paths are in scope

An agent must be able to fetch one field bare, **and** must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback: the field flag (§9) does not excuse an ambiguous block in full output, and repairing the block does not remove the need for bare single-field output.

#### 2.2 The fidelity bar is byte-identity

Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was.

Existing whitespace trimming does not stand in the way of this. `create` and `update` both run a description through `TrimDescription` before storing (`grep -n 'TrimDescription(' internal/cli/create.go internal/cli/update.go` → `create.go:214`, `update.go:193`, `update.go:342`), which is `strings.TrimSpace` (`grep -n 'func TrimDescription' -A 2 internal/task/task.go` → `task.go:193-195`). No stored description can therefore carry leading or trailing whitespace, and the trim is idempotent over anything that came out of storage. The trimming behaviour is out of scope as a defect.

The bar does not hold for free of charge across all three free-text carriers. Note text and task titles are rejected before reaching storage when they begin with a dash — see §10. Reaching byte-identity for them requires that fix, which this work carries.

#### 2.3 What is not a defect

The current output is not lossy. The description block prefixes two spaces to every line, so blank lines emerge as two spaces and originally-indented lines at four; stripping exactly two from each line returns the original bytes. Note text passes through the library's tabular encoder, which quotes and escapes any string containing a newline, carriage return or tab, so a multi-line note survives as `"multi\nline\nnote"`.

The defect is that the two free-text fields use two different, mutually incompatible decoding rules, neither of which the output announces, and that the description block has no count and no terminator — it works today only because it is emitted last and runs to EOF (`sed -n '104,109p' internal/cli/toon_formatter.go`).

### 3. Output Inventory

#### 3.1 Output that must parse

Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8).

| Output | Commands |
|---|---|
| Task detail | `show`, `create`, `update`, `note add`, `note remove`. The four mutating commands share `outputMutationResult` (`grep -rn 'outputMutationResult' internal/cli/ \| grep -v _test` → `create.go:277`, `update.go:409`, `note.go:86`, `note.go:142`); `show` queries and renders the same detail inline (`sed -n '52,64p' internal/cli/show.go`), so a change made only at the helper leaves `show` — and §9's field selection, which never reaches the helper — untouched |
| Task list | `list`, `ready`, `blocked` |
| Stats | `stats` |
| Dependency graph | `dep tree` |
| Status change | `start`, `done`, `cancel`, `reopen` |

#### 3.2 Output that stays prose

`dep add`, `dep remove`, `remove`, `init`, and the general-purpose messages. These are confirmations of a command the caller issued: the caller already knows what it asked for and the exit code says whether it worked. Wrapping a one-line confirmation in a data format costs tokens to restate what the caller already knows and introduces a new way to fail.

The boundary this draws: a confirmation may be prose; an **answer to a query** may not. "No dependencies" is not a confirmation — it is the answer the caller ran the command to find out (§8).

#### 3.3 `doctor` and `migrate` are out of scope

Both bypass the formatter entirely and print straight to the terminal — `handleDoctor` documents this in its own comment (`sed -n '44,46p' internal/cli/doctor.go`), and `RunMigrate` presents through `migrate.Present` rather than a `Formatter` (`grep -n 'migrate.Present' internal/cli/migrate.go` → `migrate.go:124`). Neither honours the format flags.

The defect being fixed is output that *claims* to be machine-readable and is not — a header no reader accepts, a list missing its item markers. These two never claimed it. Bringing them in means building new formatter surface rather than repairing broken output, which is different work.

Cost accepted: an agent running the health check reads a paragraph and works out what to do from the words.

### 4. Formatter Scope

#### 4.1 Pretty is unchanged, everywhere

What a terminal prints today is what it prints after this work — the single transition line, the box-drawing cascade tree, the indented description block, the dep-tree prose on empty branches. Pretty is the human surface; this work is about the agent surface, and nothing in pretty is broken by the standard being applied: it never claimed to be machine-readable, and a human reads it fine.

Two candidate changes were declined explicitly and are **not** in scope:

- Replicating §7's table shape in pretty. The current cascade tree nests by which task caused which, so a grandchild closing because its parent closed shows as three levels of indentation. Flattened into rows, every knock-on looks equally directly caused. The toon table drops that too, but an agent holds the parent/child links and can reconstruct the chain; a human reading a terminal cannot, which is why the tree exists.
- Adding the task's title to pretty's single transition line. A genuine improvement rather than a defect fix, and out of scope.

#### 4.2 JSON moves with toon

A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`).

#### 4.3 Two shared code paths must be split, not edited

Pretty being unchanged while toon and JSON move is not free — the code is shared in two places, and editing it in place would change pretty by accident:

- **The single transition line** comes from `baseFormatter.FormatTransition` (`grep -n 'func (b \*baseFormatter) FormatTransition' internal/cli/format.go` → `format.go:211`), embedded by both the toon and pretty formatters, so the two emit byte-identical text today. Restructuring the toon form requires splitting that method.
- **The dep-tree empty messages** are not produced by a formatter at all. They are set on the result in the shared graph builder (`grep -n 'No dependencies' internal/cli/dep_tree_graph.go` → `dep_tree_graph.go:177`, `dep_tree_graph.go:255`) and consumed by all three formatters, so removing them at source would strip pretty's message too. Pretty keeps its sentence; only the machine formats take the structured empty form.

### 5. Single-Object Sections Become Top-Level Named Fields

#### 5.1 The current shape and its cause

Three places in the output describe one thing rather than a list of things: the task's own fields at the head of `tick show` (and of `create`, `update`, `note add`, `note remove`), the counts summary in `tick stats`, and the chains/longest/blocked summary in `tick dep tree`. All three are malformed, and this is the line a reader fails on before it sees anything else.

The cause is a hand-edit. The value is marshalled as a one-element array and the `[1]` is then deleted from the header with a string replace to make it read as singular — `buildTaskSection` does it inline (`grep -n 'strings.Replace(s, "task\[1\]"' internal/cli/toon_formatter.go` → `toon_formatter.go:294`) and `encodeToonSingleObject` does it generically for stats and the dep-tree summary (`grep -n 'func encodeToonSingleObject' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:371-379`). The result is a table header with no table beneath it, a shape TOON has no equivalent for.

#### 5.2 The required shape

**The task's own fields sit at the top level of the document, as named fields, with no wrapping key.** They are peers of the collection sections rather than nested inside a `task:` scope:

```
id: tick-a1b2
title: Add retry to the sync worker
status: in_progress
priority: 2
type: feature
created: "2026-09-10T09:14:00Z"
updated: "2026-09-17T16:00:00Z"

children[1]{id,title,status}:
  tick-9f3c,Parse the header,done

description: "Fix it.\n\nSteps."
```

**The same treatment applies to the other two single-object sites**: `tick stats`' counts and the dep-tree chains/longest/blocked summary become top-level named fields beside their tables.

#### 5.3 Why named fields rather than a one-row table

Two conformant options existed; both were encoded and decoded back with the project's TOON library, and both round-trip.

The one-row table (`task[1]{id,title,…}:` with a single row beneath) is three characters longer than today's broken output and is the library's own output with nothing stripped. Its cost is positional reading: a consumer counts commas across to the matching name in the header. The table layout earns its keep when many rows would otherwise repeat the field names — that is `tick list`'s case, not this one. With a single row it repeats nothing, so it compresses nothing, and it trades that for a positional read that can go wrong.

Named fields put every value beside its name, so adding or reordering a field cannot break a positional read and a value containing a comma is safe without relying on quoting discipline.

Neither form is an invention. TOON is a compact way of writing JSON: an object inside an object is written as the key with its fields indented beneath (the named-fields form), and a list of same-shaped objects is written as a header plus rows (the table form, which is what `tick list` uses). What tick invented was neither.

#### 5.4 Why no wrapping key

Measured: 256 characters flat against 276 wrapped, for the same content, both parsing and round-tripping.

Three reasons, the size being the least of them:

1. **It removes an inconsistency the wrapper created.** `description` is a task field and sits at the top level. `title` is a task field and sat nested. The same kind of thing at two different depths, for no reason beyond one of them being long.
2. **It mirrors storage.** The JSONL file already holds each task as a flat object — `{"id":…,"title":…,"status":…,"description":…}` — so the output stops inventing a grouping that exists nowhere else in the system.
3. **The field flag falls out of it.** `--field title,status` returns two lines with nothing wrapped around them, rather than a trimmed block (§9.4).

Given up: a consumer can no longer grab "the task's own fields" as one object without naming them. Nothing identified wants that.

### 6. Collection and Free-Text Sections

#### 6.1 Tags and refs use the library's inline list form

Today tags and refs are emitted as a header followed by one raw item per indented line (`grep -n 'func buildStringListSection' -A 9 internal/cli/toon_formatter.go` → `toon_formatter.go:320-328`):

```
tags[2]:
  has space
  plain
```

A TOON reader rejects this: an item written on its own line carries a leading `- ` marker, and without it the decoder reports a length mismatch. The items are also written raw, so a ref containing a comma comes back as two values rather than one, and a URL's colon goes unquoted where the format's own rules would quote it.

**Both become the inline form, produced by the library:**

```
tags[2]: has space,plain
```

The deciding factor is that this deletes the hand-written builder rather than correcting its output, and quoting becomes the library's responsibility — so a comma-bearing ref is safe by construction. The alternative, one item per line each marked `- `, is not a form the encoder emits: it would have to be hand-built, preserving the exact class of defect this work removes.

Cost accepted: a long refs list becomes a long line. The reader of toon output is an agent; a human reading tags reads pretty output, which renders them separately and is unchanged (§4.1).

#### 6.2 The description is one TOON-quoted value

Today the description is `description:` followed by every line of the text prefixed with two spaces, with no count and no terminator (`grep -n 'func buildDescriptionSection' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:346-355`).

**It becomes a single TOON-quoted string, exactly as the library's `toon.MarshalString` produces from a Go string:**

```
description: "Retry the sync worker on transient failures.\n\nCurrent behaviour: a single 500 …"
```

Three reasons, in the order they carried:

1. **It is produced by the library, not hand-assembled.** The alternative — a dash-list of lines, one item per line with a declared count — is not a form the encoder produces from a `[]string` (given one it emits the inline form). It would have to be built by hand with our own quoting rules, which is the mechanism behind every malformed section this work removes.
2. **The dash-list's advantage evaporates on real content.** Measured on an actual 13-line, 1090-character task description from a live project: the quoted value is 1117 characters on one line; the dash-list is 1185 characters over 14 lines with **13 of 13 lines needing quotes** — for a colon in a heading, a leading dash on a bullet, leading spaces on an indented line. The line structure it exists to preserve is buried under quote marks, and it is larger as well.
3. **It is the rule note text already obeys.** One decoding rule for all free text in the output, which is what the reader needed and never had.

The quoted form's overhead on that text is 27 characters, about 2.5%, all of it newline escapes. On a 4500-character description it is roughly a hundred characters and one very long line.

Cost accepted: a long description is one long line and unpleasant to read in a terminal. A human reading a description reads pretty output, which this work does not touch.

A third candidate — keeping the indented raw block and adding a terminator or a line count — cannot be conformant however it is delimited: it is a block form TOON has no concept of.

#### 6.3 The notes section carries an index column

Notes already round-trip on the read side: note text goes through the library's tabular encoder, which quotes and escapes newlines, carriage returns and tabs, so a multi-line note is emitted as one unambiguous quoted string. What changes is the schema.

**The notes section gains a leading `index` column carrying each note's 1-based position, present whether the section is filtered (§9.3) or not:**

```
notes[2]{index,text,created}:
  1,Retried twice before it stuck,"2026-09-14T10:02:00Z"
  2,"multi\nline\nnote","2026-09-16T08:30:00Z"
```

Position is the only handle a note has. There is no note ID; `note remove` takes a 1-based index (`grep -n 'index %d out of range' internal/cli/note.go` → `note.go:128`); and notes stay an append-and-retract log (§6.4), so nothing else identifies one. Without the column, an agent that asked for the third note alone sees `notes[1]` with one row, decides the note is wrong, runs `tick note remove <id> 1`, and deletes the first note on the task.

One shape either way, so there is no rule about when the column appears. It also retires a smaller oddity: in full output today an agent must count rows to work out what to pass to `note remove`.

#### 6.4 Notes stay read-side only — no edit command

Editing a note today means removing it and adding it back, which gives the corrected text a fresh `created` timestamp and moves it to the bottom of the list. The `note` command has exactly two subcommands (`grep -n 'case "add":\|case "remove":' internal/cli/note.go` → `note.go:28`, `note.go:30`).

**No `note edit` command is added by this work.** A note carries a `created` stamp and sits in a positional log you retract from by index — an append-only record of what was observed when, not a mutable field. Adding an edit would turn it into a list of editable strings, and would do so as a side effect of a formatting fix rather than as a decision about the annotation model. If notes should become editable, that is its own piece of work with its own reasoning.

Cost accepted: correcting a note still costs its timestamp and its position. The counter-argument — agents write these notes and agents typo — was weighed and did not carry, because the remedy it asks for changes the data model to serve a convenience.

What remains in scope for notes: their text is readable out of `tick show` without a rule learned elsewhere (already true, via TOON quoting), field extraction reaches it (§9), and note text that begins with a dash becomes writable (§10).

### 7. Status Change Output

#### 7.1 The current shape

When an agent closes a task and other tasks move with it, what comes back is an arrow diagram:

```
tick-a1b2: in_progress → done
tick-9f3c: open → done (auto)
tick-77ab: in_progress → done (auto)
```

Reading it requires knowing that the ID precedes the colon, that the arrow separates old state from new, and that `(auto)` marks a knock-on rather than the requested change. It is a bespoke line format, and it is byte-identical in `--toon` and `--pretty`: both formatters embed `baseFormatter.FormatTransition` (§4.3), and `ToonFormatter.FormatCascadeTransition` is the same construction with ` (auto)` appended (`grep -n 'func (f \*ToonFormatter) FormatCascadeTransition' internal/cli/toon_formatter.go` → `toon_formatter.go:145`).

#### 7.2 One table, always

**Every task whose status moved gets a row, and an `auto` column says whether that row is the change the caller asked for:**

```
changed[3]{id,title,from,to,auto}:
  tick-a1b2,Add retry to the sync worker,in_progress,done,false
  tick-9f3c,Parse the header,open,done,true
  tick-77ab,Validate fields,in_progress,done,true
```

A re-parenting, where nothing the caller asked for was itself a status change, is the same shape with every row marked as a consequence:

```
changed[2]{id,title,from,to,auto}:
  tick-1111,Phase 5,done,open,true
  tick-2222,Phase 4,in_progress,done,true
```

Two reasons carried it:

1. **A reader must not have to branch on which document arrived before it can read either.** An agent parsing status output sees one table whatever command produced it, and the count is always right. A singular block for the requested change beside a table of knock-ons would be two documents for one kind of event.
2. **The marking column is not new vocabulary.** Every task's transition history already records each change with an `auto` flag — false when a user or agent asked for it, true when the system produced it as a consequence — stored per task in the JSONL and in the `task_transitions` table. Inventing a "requested" column instead would be exactly the move that produced the malformed output this work removes: a shape someone wanted, built by hand, outside the vocabulary that already existed.

The table carries the title so no second lookup is needed to know what moved. The trailing `(auto)` marker of the old arrow lines disappears into the column that always meant it.

#### 7.3 The per-command split is kept

`done`, `start`, `cancel` and `reopen` return **only** the `changed` table. `create` and `update` return the task's full record with the `changed` table as a section inside it.

The deciding factor: `create` and `update` are edits and the caller wants the result of the edit — the new ID, the merged fields — whereas a status change is something the caller already knows it did, so a full record is tokens it did not ask for.

A third option was put up and declined: leave status output alone entirely, treating `tick done` as a prose confirmation like `tick dep add`, with an agent running `tick show` afterwards if it needed to know what cascaded. The single-transition case is indeed change for consistency rather than repair. It was declined once the cascade case was seen beside it: the multi-task output carries titles the agent would otherwise have to look up, and one command's output parsing while another's does not — depending on whether a cascade fired — is exactly the branching rule this work exists to delete.

#### 7.4 One document, never a document with loose lines after it

`create` and `update` today print the full task detail and then append transition lines after it (`grep -n 'outputTransitionOrCascade' internal/cli/create.go internal/cli/update.go` → `create.go:283`, `update.go:415`, `update.go:420`). A reader handed that stream sees a task-detail document with foreign lines stuck on the end.

**Where a command produces both a record and status changes, the result is one document with the changes as a section inside it.** Making each section valid is not sufficient on its own; the stream must be one document.

`update` can carry two independent cascade blocks today because two unrelated changes can fire at once — this collapses into the single `changed` table along with everything else.

#### 7.5 Why a structural change produces a status cascade

Recorded because it is not obvious from the code, and the implementation must preserve it. `create` and `update` emit cascades not because the edited task's status moved, but because *other* tasks' statuses moved as a consequence of the parent/child structure changing:

- **Adding work under a finished parent.** `tick create --parent <done task>` reopens that parent — it is no longer complete — and that reopen travels further up if its own parent was done (Rule 6, then Rule 5).
- **Moving a task to a different parent.** `tick update <id> --parent <other>` can fire two unrelated changes at once: the new parent reopens if it was finished, and the old parent may auto-complete if the moved task was the last unfinished thing under it (Rule 6 and Rule 3 together).

These tasks are elsewhere in the tree and are not in the edited task's record, which is why they cannot be folded into it.

#### 7.6 Unchanged terminal children are not reinstated

The `changed` table lists what changed, exactly as today's output does. The `auto-cascade-parent-status` specification's requirement that unchanged terminal children be shown alongside a cascade is a pre-existing unimplemented requirement in another work unit's specification; it is not reinstated and not decided here. See §12.2 for the correction owed to that document.

### 8. Structured Output on Empty Branches

A command in §3.1's must-parse table emits its structured form on **every** branch, the empty one included.

`tick dep tree` currently answers `No dependencies found.` when nothing in the project is blocked, and a title line plus `No dependencies.` when a named task has no dependencies either way (§4.3 locates both). That is prose on the one branch an agent could not predict, from the formatter whose purpose is machine-readable output. **Both go, in toon and JSON; pretty keeps them (§4.3).**

This is not an exception to §3.2's prose rule, it is that rule's boundary: the exemption covers confirmations of a command the caller issued, and "no dependencies" is the answer to a query — the answer the caller ran the command to find out.

The formatter already has the shape. An empty task list comes back as a structured empty section, and so does an empty edge set — `buildRelatedSection` and `buildNotesSection` both emit a count-zero header rather than prose (`grep -n '\[0\]{' internal/cli/toon_formatter.go` → `toon_formatter.go:300`, `toon_formatter.go:333`). The two dep-tree prose branches are the exception, not the pattern.

### 9. Field Selection

The companion to the format repair: a way to ask for one field's value and get it with nothing around it — no header, no indentation, no quoting. This is the case that started the work, where an agent needed a task's description as a plain string and went to the raw data file instead.

`show` accepts no command-specific flags today (`grep -n '"show":' internal/cli/flags.go` → `flags.go:72`, `"show": {}`), so this is its first, alongside the global `--quiet`.

#### 9.1 `--field` and `--fields` are the same flag

**`--field` takes a comma-separated list of field names, and `--fields` is an alias of it.** Both spellings work; the plural exists so the flag reads naturally when selecting several. The flag is a projection, not a single-value extractor — asking for the notes table is a legitimate thing to want, since editing and writing back is not the only reason to read a field.

The answer's shape is split by how many fields were asked for.

#### 9.2 One field returns the bare value; several return a filtered document

**One field — the bare value.** `tick show tick-a1b2 --field description`:

```
Fix the parser.

Steps:
  - read the header
  - validate
```

**Several fields — the normal document with only those sections in it.** `tick show tick-a1b2 --field description,notes`:

```
notes[2]{index,text,created}:
  1,Retried twice before it stuck,"2026-09-14T10:02:00Z"
  2,"multi\nline\nnote","2026-09-16T08:30:00Z"

description: "Fix the parser.\n\nSteps:\n  - read the header\n  - validate"
```

Identical to a full `tick show` minus the sections not asked for. **Sections keep their usual output order**, not the order they were typed, so the shape does not shift with how the flag was written.

The split between the two kinds of answer was locked in knowingly. `--field description` and `--field description,notes` return different kinds of thing — a raw value versus a document — so an agent building the flag from a variable must know which it will get. It is the honest split between *fetch me this value* and *give me a trimmed record*, and collapsing them would cost the bare-value case the work exists to serve.

#### 9.3 List fields are reachable by position

`notes.2` selects the second note. In a multi-field selection the section renders as normal, its count following the selection while each row carries its real position via the `index` column (§6.3) — `tick show tick-a1b2 --field description,notes.2`:

```
notes[1]{index,text,created}:
  2,"multi\nline\nnote","2026-09-16T08:30:00Z"
```

Asked for alone, `--field notes.2` prints that note's text bare, by the one-field rule.

The alternative considered and rejected was to refuse filtering on notes altogether — `notes.3` returning the whole table, leaving position implicit in row order. That keeps the output minimal at the cost of the selector the caller asked for.

#### 9.4 The task's own fields are selectable individually

**Exactly like sections.** `tick show tick-a1b2 --field title,status`:

```
title: Add retry to the sync worker
status: in_progress
```

Refusing — on the grounds that only whole sections are selectable — would make the grammar depend on which side of a boundary a name happens to sit, which is a rule to learn rather than read. §5.2 removed the wrapper, so the result has nothing around it.

#### 9.5 Nothing rides along unasked, including `id`

**You get exactly the fields you named**, in both forms.

The opening position was that `id` should always be present so a filtered document identifies the task it describes. It does not survive the bare form: `--field description` prints the bare value with nothing around it, so an `id` riding along would wreck the case the flag exists for. Scoping the rule — `id` present in the document form, absent in the bare form — was available and rejected as a second rule to learn. The caller passed the task's ID on the command line to make the request; handing it back tells them something they just typed.

#### 9.6 Empty values, unrecognised names, and out-of-range positions

**An empty field prints nothing and exits successfully.** A task legitimately having no description is a fact about the task rather than a failure of the command, and it takes the same shape as an empty notes table in full output.

**An unrecognised field name is an error with a non-zero exit.** A field name that is not a field at all is a caller mistake, and gets what every other unrecognised flag value already gets — `ValidateFlags` refuses rather than silently ignoring (`grep -n 'unknown flag %q for %q' internal/cli/flags.go` → `flags.go:138`).

**A note position that does not exist is an error too.** `notes.4` on a task carrying two notes fails with a non-zero exit and a message naming the range, rather than printing nothing and succeeding. A selector that resolves to nothing is not a field that happens to be empty — the position is a claim about the data that is false. `tick note remove` already answers this for the same 1-based addressing, reporting the index as out of range and naming how many notes the task has (§6.3); giving the same grammar a different answer under a different command would be a second rule for a reader to learn.

#### 9.7 Interaction with the format flags

**A single-field request ignores `--json`, `--pretty` and `--toon`; a multi-field request honours them.** A one-field answer is a raw value, so there is no document for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted. A multi-field request does produce a document, and the resolved format applies to it exactly as it applies to a full `tick show`.

#### 9.8 Interaction with `--quiet`

**Passing `--quiet` and a field selection together is refused.** `--quiet` prints a bare task ID and nothing else; a single-field request prints that field's bare value and nothing else. Two different single values have been asked for, and silently picking one hands back something the caller did not ask for.

### 10. Free Text That Begins With a Dash

#### 10.1 The defect

An agent reads a note off a task, corrects a typo, and writes it back. If the note begins with a dash — `- read the header`, the shape a large share of this project's own notes take — the command refuses it:

```
unknown flag "- read the header" for "note add"
```

Wider than notes: `tick create "- some title"` is refused identically. `ValidateFlags` inspects every argument beginning with `-` that is not numeric and not a global flag, and rejects any it cannot find in the command's flag set (`grep -n 'func ValidateFlags' -A 30 internal/cli/flags.go` → `flags.go:117-147`). Free text passed as a bare argument — note text, task title — is inspected alongside real flags. `--description` escapes only because its value follows a registered value-taking flag, so validation skips it.

The check itself is deliberate and worth keeping: it exists so `tick update tick-a1b2 --prioirty 3` refuses rather than silently reporting success. The defect is that it is applied to arguments that are free text by definition.

This is the one place the §2.2 round-trip guarantee has a hole — text an agent can read out of tick and cannot put back. It is fixed here rather than left as a follow-up, because leaving it means shipping the contract with an exception nobody wrote down.

#### 10.2 The fix, both halves

- **`--` is supported as the end-of-flags marker, and becomes the canonical documented way to pass free text that may begin with a dash.** Purely additive: `--` is currently rejected on every command (`unknown flag "--" for "note add"`), so no existing invocation uses it. Everything that works today works identically, and inputs that currently fail begin to succeed.
- **Flag inspection stops after the task ID on `note add`.** The command registers no flags at all (`grep -n '"note add":' internal/cli/flags.go` → `flags.go:80`, `"note add": {}`), so once the task ID is consumed every remaining argument is text by definition and nothing dash-leading there could be a flag the check would have caught. A dash-leading note then works with or without the marker.
- **The existing bare-argument form keeps working.** `--` is the recommended form, not a required one.

`create` cannot take the second half: its title shares the argument list with real flags (`--priority`, `--description`), so a dash-leading title is indistinguishable from a mistyped flag without a marker. `create` relies on `--`.

#### 10.3 No alternative input path

**Descriptions are passed as command-line arguments; no stdin or file input path is added.**

Measured: `getconf ARG_MAX` → `1048576`, and passing a 200 KB argument through a process call succeeds. A real task description taken from a live project runs to roughly 4.5 KB — under half a percent of the limit. Descriptions carry no length cap of their own, unlike note text, which is capped at 2000 characters (`grep -n 'maxNoteTextLen' internal/task/notes.go` → `notes.go:12`), so an arbitrarily large description remains possible in principle; building an input path for it would serve a case that does not occur.

### 11. Conformance Verification

Every other decision here is a shape. Nothing in them stops the next change adding a hand-built section and breaking the output again — which is exactly how it broke the first time.

The existing suite cannot catch it. Its assertions compare output against a string written down alongside the code, so a malformed header passed for the tool's entire life: the test compared a wrong string to the same wrong string. Every one of those assertions has to be rewritten regardless, since every shape this work touches changes.

Three parts:

1. **Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse.** This alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content. The commands are §3.1's table.

2. **One deliberately awkward task becomes a permanent fixture, round-tripped end to end.** Free text carrying newlines, quotes, commas, a leading dash, trailing spaces, and a line that looks like a section header. Write it in, read it out, decode it, assert the text is identical to what went in. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability.

3. **Rewritten assertions check decoded values, not output text.** "The notes section has two rows and the second row's text is X", not "the output equals this blob".

**No byte-level pinning is kept in the machine formats.** Golden strings pin the exact output shape, so a future change cannot reshape a section without a test noticing — but they are the mechanism that rotted into the defect this work undoes. Decoded-value assertions survive harmless reformatting while still failing when a section goes missing or a value is wrong. That trade is taken for toon and JSON.

**Pretty keeps golden-string assertions.** Pretty output has no parser, so a decoded-value assertion does not exist for it; removing its golden strings would replace its only form of assertion with nothing.

### 12. Documentation Owed a Correction

Three published documents describe output this work replaces.

#### 12.1 The README is updated as part of this work

Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work those samples show output the tool no longer produces. It is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect.

#### 12.2 Two completed specifications are corrected selectively

Both belong to completed work units, so the correcting route is the one that presents each proposed amendment and confirms before editing another unit's record.

| Document | What is wrong | Status |
|---|---|---|
| `v1` / `tick-core` specification | States "long text fields get their own unstructured sections" as a principle and prints the indented description block as its worked example (`sed -n '693,714p' .workflows/v1/specification/tick-core/specification.md`) | Work unit completed |
| `auto-cascade-parent-status` specification | Fixes the arrow-and-`(auto)` lines as the machine-readable cascade form and requires unchanged terminal children to be shown alongside them (`sed -n '117,162p' .workflows/auto-cascade-parent-status/specification/auto-cascade-parent-status/specification.md`) | Work unit completed |

**Corrections are made by judgement, not as a blanket rewrite.** Specifications are forever documents that do not churn out of the knowledge base, and the facility for amending them exists, so a correction can be made wherever something is obviously wrong. But this specification supersedes those decisions regardless, so correcting them is not obligatory. Where a point is plainly and load-bearingly wrong, amend it; otherwise let supersession carry it.

The `tick-core` unstructured-long-text principle is the most obviously wrong of the two, since it states as a rule the exact thing §6.2 removes.

---

## Working Notes
