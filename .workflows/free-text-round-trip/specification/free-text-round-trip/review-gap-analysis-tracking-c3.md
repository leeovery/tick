# Review Tracking: Free Text Round Trip - Gap Analysis

## Findings

### 1. What a filtered record looks like on a terminal is never said

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Important
**Affects**: §9.7 (interaction with the format flags), §9.2 (one field vs several), §4.1 (pretty unchanged)

**Problem**:
On a terminal the format resolves to pretty unless a flag says otherwise, so a person typing `tick show tick-a1b2 --field notes` asks for a document and gets pretty. Nothing says what pretty prints for it. A builder either wraps the selection in pretty's usual detail chrome — the task header block and the labels for sections nobody asked for — or prints the selected sections alone. The two produce visibly different screens for the most ordinary interactive use of the new flag, and the rule that nothing rides along unasked was written for the machine formats.

**Proposal**:
The answer is already fixed by two decisions in the document: a document-returning request honours the resolved format, and you get exactly the fields you named and nothing else. Pretty therefore renders the named fields in its usual style with nothing around them. Say so, and note that a filtered record is output pretty does not produce today rather than a change to output it does, so the guarantee that terminal output is unchanged still holds.

**Current**:
**A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them.** A bare value is not a document, so there is nothing for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted. A multi-field request produces a document, and so does a single field naming a list section (§9.2); the resolved format applies to either exactly as it applies to a full `tick show`.

**Proposed Text**:
**A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them.** A bare value is not a document, so there is nothing for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted. A multi-field request produces a document, and so does a single field naming a list section (§9.2); the resolved format applies to either exactly as it applies to a full `tick show`.

In pretty — the format a terminal resolves to unless a flag overrides it — the filtered document is the named fields rendered in pretty's usual style and nothing else: no header block, no labels for sections outside the selection (§9.5). A filtered record is output pretty does not produce today rather than a change to output it does, so §4.1 stands untouched.

**Resolution**: Pending
**Notes**:

---

### 2. Whether a format flag after the task ID still works on `note add` is left open

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §10.2 (the fix, both halves)

**Problem**:
`tick note add tick-a1b2 "text" --json` returns a JSON document today. The rule that flag inspection stops after the task ID is written as "every remaining argument is text by definition", which read plainly turns that trailing `--json` into part of the note and drops the format flag — a working invocation that silently starts storing `--json` as note text. Read the other way, only the check stops and the flag keeps working. The neighbouring rule needs the second reading: it says free text that spells a flag exactly, a note reading `--json`, becomes writable because of the `--` marker, which is only true if without the marker that argument is still consumed as a flag. A builder has to pick, and the two picks differ on what a caller's existing command does.

**Proposal**:
What stops after the task ID is the validation check, not flag handling — the marker remains the only thing that turns a global flag into text. That is the reading the marker rule already depends on, and the other reading regresses an invocation that works today.

**Current**:
- **Flag inspection stops after the task ID on `note add`.** The command registers no flags at all (`grep -n '"note add":' internal/cli/flags.go` → `flags.go:80`, `"note add": {}`), so once the task ID is consumed every remaining argument is text by definition and nothing dash-leading there could be a flag the check would have caught. A dash-leading note then works with or without the marker.

**Proposed Text**:
- **Flag inspection stops after the task ID on `note add`.** The command registers no flags at all (`grep -n '"note add":' internal/cli/flags.go` → `flags.go:80`, `"note add": {}`), so nothing dash-leading after the ID could be a flag the check would have caught, and refusing it is the whole defect. What stops is the check, not flag handling: the global flags are still read wherever they appear, so `tick note add <id> "text" --json` prints a JSON document exactly as it does today, and note text that spells a global flag exactly still needs the marker. A dash-leading note that is not itself a global flag works with or without it.

**Resolution**: Pending
**Notes**:

---

### 3. How the field list is read is decided for blank names and nothing else

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §9.1 (`--field` and `--fields`), §9.6 (empty values, unrecognised names, out-of-range positions)

**Problem**:
The flag takes a comma-separated list, and the document settles what a stray comma does — it fails as an unrecognised name. Three neighbouring cases a caller hits just as easily have no answer: `--field "title, status"` with a space after the comma, `--field title --field status` with the flag given twice, and the same field named twice or named both whole and by position (`--field notes,notes.2`). A builder invents an outcome for each, and the inventions differ in whether a request whose meaning is not in doubt fails with an error, silently loses the first flag, or comes back with the same section printed twice.

**Proposal**:
Each of the three has a side that costs the caller nothing and a side that fails a request nobody would call wrong, so the rule follows the harmless side: whitespace around a name is not part of the name, repeating the flag composes rather than overrides, and a field named more than once renders once. Blank names stay the mistake §9.6 already calls one.

**Current**:
The names the flag accepts are the names the output document uses — the task's own top-level fields (`id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`) and the section keys (`description`, `notes`, `tags`, `refs`, `children`, `blocked_by`), spelled as a full `tick show` spells them. There is no second vocabulary to learn: what you read in the output is what you ask for. A positional suffix (`notes.2`, §9.3) attaches only to a section that holds a list; on anything else the whole name is unrecognised and takes §9.6's error.

**Proposed Text**:
The names the flag accepts are the names the output document uses — the task's own top-level fields (`id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`) and the section keys (`description`, `notes`, `tags`, `refs`, `children`, `blocked_by`), spelled as a full `tick show` spells them. There is no second vocabulary to learn: what you read in the output is what you ask for. A positional suffix (`notes.2`, §9.3) attaches only to a section that holds a list; on anything else the whole name is unrecognised and takes §9.6's error.

**The list is read leniently wherever its meaning is not in doubt.** Whitespace around a name is not part of it, so `--field "title, status"` selects what `--field title,status` selects. Repeating the flag composes rather than overrides: `--field title --field status` is `--field title,status`. A field named more than once renders once — a section named both whole and by position comes back whole, and several positions on one section render it narrowed to those positions in output order. None of this softens §9.6: a name that is empty once its whitespace is gone is still the blank-name mistake.

**Resolution**: Pending
**Notes**:

---

### 4. The empty dependency answer tells a reader to read counts that cannot say anything

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Important
**Affects**: §8 (structured output on empty branches)

**Problem**:
An agent is told it reads the same document on both empty dependency branches and reads the counts to learn which one it hit — but every count on both branches is zero, so the counts distinguish nothing. What the reader actually needs to know is that empty and non-empty produce one shape and the counts say whether anything is there. The same sentence leaves a builder guessing whether the emptied document still identifies the task the caller named, since the populated one does: one reading strips the task's identity so both empty branches come out identical, the other keeps it. The output of `tick dep tree <id>` differs between the two.

**Proposal**:
The emptied document is the populated one with its numbers at zero, field for field, so a named task is still identified. That is what "the document the non-empty branch produces, emptied" already commits to, and it is the reading that makes the counts worth reading.

**Current**:
**What replaces them is the document the non-empty branch produces, emptied**: the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Both branches take that one shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent reads the same document whichever it hit, and reads the counts to learn which.

**Proposed Text**:
**What replaces them is the document the non-empty branch produces, emptied**: the same fields, with the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Where the caller named a task, the emptied document still identifies it exactly as the populated one does. Both empty branches take that shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent parses one document whether or not anything is blocked, and reads the counts to learn which it got.

**Resolution**: Pending
**Notes**:

---

### 5. Free text already imported by an earlier version still fails the round-trip bar

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: choice
**Priority**: Important
**Affects**: §2.2 (the fidelity bar is byte-identity), §11 (conformance verification)

**Problem**:
Byte-identity is made to hold by trimming free text as it is imported. Tasks imported before that lands keep their untrimmed values in storage, so reading one out of `tick show` and writing it back still silently trims it — the exact failure the bar exists to delete, on exactly the tasks nobody typed by hand. Nothing says whether this work touches values already sitting in `tasks.jsonl`, so a user with an imported project either gets the guarantee or gets it for everything except the tasks that motivated it.

**Options**:
- Leave stored values alone: the bar holds for everything written from this change onward, and a task imported by an earlier version can still lose edge whitespace on write-back (recommended — it keeps this work to output shapes and one import-path trim, and the exposure closes itself as those tasks are edited)
- Normalise once: a pass over stored descriptions and titles trims edge whitespace so the bar holds for every task in the file, at the cost of rewriting task records this work otherwise does not touch
- Trim on the way out: `tick show` emits the trimmed value for any stored value carrying edge whitespace, so write-back matches, at the cost of the read no longer returning the stored bytes

**Notes on the search**: §2.2 decides the import-path trim and rules out two alternatives to it; §3.3 scopes `migrate`'s output out while naming its write path as in; §11's fixture round-trips a task this work itself writes. Nothing in the record reaches values already stored.

**Resolution**: Pending
**Notes**:

---

### 6. What an empty selection prints has two answers for the always-present sections

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §9.6 (empty values), §9.2 (a single field naming a list section), §5.2 (which sections a document carries)

**Problem**:
`tick show tick-a1b2 --field notes` on a task with no notes has two answers in the document. One says an empty field prints nothing and exits successfully; the other says a single field naming a list section prints that section exactly as full output renders it, and full output always carries the notes section with a count-zero header. Nothing to parse versus `notes[0]{index,text,created}:` is the difference between a consumer getting a document and getting a zero-byte stream it has to special-case — on `notes`, `children` and `blocked_by` alike, the three sections a task most often has none of. The multi-field form inherits the same fork.

**Proposal**:
The two rules only collide because the empty-field rule was written for `description`. Settle it by making what an empty name prints be what full output prints for it: sections full output omits when absent print nothing, sections full output always carries print their count-zero header.

**Current**:
**An empty field prints nothing and exits successfully.** A task legitimately having no description is a fact about the task rather than a failure of the command, and it takes the same shape as an empty notes table in full output.

**Proposed Text**:
**An empty field prints what full output prints for it, and exits successfully.** A task legitimately having no description is a fact about the task rather than a failure of the command. A field or section that full output omits when the task does not carry it — `description`, `tags`, `refs`, `type`, `parent`, `closed` (§5.2) — prints nothing; a section full output always carries prints its count-zero header, so `--field notes` on a task with no notes returns `notes[0]{index,text,created}:`. The rule runs per name: in a multi-field selection each empty name contributes what it would contribute to full output, the rest of the document is unaffected, and a selection whose every name prints nothing prints nothing at all and still exits successfully.

**Resolution**: Pending
**Notes**:

---

### 7. The conformance check is written per command while the guarantee is per document

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Important
**Affects**: §11 (conformance verification), §3.1 (output that must parse)

**Problem**:
The promise is that every branch decodes — the empty one included — and that every document `show` produces, filtered ones included, decodes. The check that is supposed to hold the line is written as one decode per command. A suite that runs each command once with an ordinary invocation satisfies those words while never touching the two forms most likely to break: the emptied dependency document and a filtered `show`. That is how the current suite came to pass on a malformed header for the tool's entire life, so the check as written can leave the same hole in the same place.

**Proposal**:
State the coverage in the unit the guarantee is made in — every branch a listed command can take, and every document form `show` produces, naming the emptied forms and the filtered ones so neither can be read as covered by a single default invocation.

**Current**:
1. **Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse.** This alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content. The commands are §3.1's table.

**Proposed Text**:
1. **Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse.** This alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content. The commands are §3.1's table, and the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3).

**Resolution**: Pending
**Notes**:

---

### 8. Which fields a document carries only when the task has them is enumerated twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §9.1 (`--field` and `--fields`), §5.2 (the required shape)

**Problem**:
The list of fields and sections that appear only when the task carries them is written out in full in two places — once where the document's composition is defined, once where the flag's vocabulary is defined. Two copies of one enumeration drift the moment a field is added, and the reader who finds the shorter copy first has no way to tell which one is current.

**Proposal**:
The composition of the detail document is §5.2's subject, so the enumeration stays there and the flag section points at it, keeping its code citation at the point of use.

**Current**:
Several of these are emitted only when set — `type`, `parent` and `closed` among the task's fields (`sed -n '265,285p' internal/cli/toon_formatter.go`), and `tags`, `refs` and `description` among the sections. **Recognition does not depend on presence**: a name on this list is always recognised, and asking for one the task does not carry is an empty field, which prints nothing and exits successfully (§9.6). A name absent from the list is unrecognised whatever the task holds.

**Proposed Text**:
Several of these are emitted only when the task carries them (§5.2, `sed -n '265,285p' internal/cli/toon_formatter.go`). **Recognition does not depend on presence**: a name on this list is always recognised, and asking for one the task does not carry is an empty field (§9.6). A name absent from the list is unrecognised whatever the task holds.

**Resolution**: Pending
**Notes**:

---

### 9. The must-parse rule is stated in the inventory and again over the empty branches

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §8 (structured output on empty branches), §3.1 (output that must parse)

**Problem**:
The rule that every listed command decodes on every branch, the empty one included, is written once where the commands are listed and again as the opening of the section about empty branches. The second copy is the weaker one — it carries none of the exceptions the first carries — so a reader who takes it as the rule reaches a different answer about the bare value than the document intends.

**Proposal**:
The inventory owns the rule because it owns the list of commands it applies to; the empty-branches section opens on the work it actually does, which is removing the two prose answers.

**Current**:
A command in §3.1's must-parse table emits its structured form on **every** branch, the empty one included.

`tick dep tree` currently answers `No dependencies found.` when nothing in the project is blocked, and a title line plus `No dependencies.` when a named task has no dependencies either way (§4.3 locates both).

**Proposed Text**:
`tick dep tree` currently answers `No dependencies found.` when nothing in the project is blocked, and a title line plus `No dependencies.` when a named task has no dependencies either way (§4.3 locates both).

**Resolution**: Pending
**Notes**:

---

### 10. The description block's defect is named where the block is described and again where it is replaced

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §6.2 (the description is one TOON-quoted value), §2.3 (what is not a defect)

**Problem**:
That the description block has no count and no terminator, and that it prefixes two spaces to every line, is stated where the defect is diagnosed and restated where the replacement is proposed. The diagnosis is the one that carries the reasoning — that the block works today only because it runs to EOF — so the copy adds nothing and gives a later edit two places to correct.

**Proposal**:
Point the replacement section at the diagnosis and keep its code citation, which is the one thing the copy carries that the home does not.

**Current**:
Today the description is `description:` followed by every line of the text prefixed with two spaces, with no count and no terminator (`grep -n 'func buildDescriptionSection' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:346-355`).

**Proposed Text**:
Today the description is the indented raw block §2.3 describes (`grep -n 'func buildDescriptionSection' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:346-355`).

**Resolution**: Pending
**Notes**:

---

### 11. That unchanged terminal children are not decided here is said in two places

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §12.2 (two completed specifications are corrected selectively), §7.6 (unchanged terminal children are not reinstated)

**Problem**:
The point that the older requirement about unchanged terminal children is neither reinstated nor decided is made where the status-change table is defined and made again where the amendments are scoped. The second telling restates the first and then adds the only fact it owns — that the amendment leaves that requirement alone — so a reader meets the same non-decision twice and a later edit has two sentences to keep in step.

**Proposal**:
Keep the scoping fact the amendment section owns and let it point at the section that owns the non-decision.

**Current**:
Its requirement that unchanged terminal children be shown alongside a cascade is left standing — §7.6 neither reinstates nor decides it, and the amendment does not touch it.

**Proposed Text**:
The amendment does not touch its requirement that unchanged terminal children be shown alongside a cascade (§7.6).

**Resolution**: Pending
**Notes**:

---
