# Review Tracking: Free Text Round Trip - Gap Analysis

## Findings

### 1. A single field naming a section has no defined answer

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Critical
**Affects**: §9.2 (One field returns the bare value), §9.7 (Interaction with the format flags), §9.1 (accepted names)

**Problem**:
`--field notes` is named as a legitimate request — asking for the notes table is explicitly allowed — but the rule for a one-field request is that the answer is the bare value with no header, no indentation and no quoting. A table has no bare form. An agent running `tick show <id> --field notes` gets either the `notes[2]{index,text,created}:` section, or rows with their header stripped, or the same thing a full `tick show` would print, depending on which reading the builder picked; the same fork hits `tags`, `refs`, `children` and `blocked_by`. The rule that a one-field request ignores the format flags compounds it: if that request yields a document, `--json` is being ignored on an answer that is a document, which is the opposite of the stated reason for ignoring it (there is no document for the flag to act on).

**Proposal**:
Settled by the flag's own stated reasoning: the bare form exists so a caller can take a value away without parsing anything off it, and the format-flag exemption is justified on there being no document. So a selection that resolves to one value comes back bare, a selection that resolves to a row or a list comes back as its section in the one shape the library produces, and the format flags apply wherever a document is produced.

**Current**:
> **A single-field request ignores `--json`, `--pretty` and `--toon`; a multi-field request honours them.** A one-field answer is a raw value, so there is no document for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted. A multi-field request does produce a document, and the resolved format applies to it exactly as it applies to a full `tick show`.

**Proposed Text**:
Add to §9.2, after the "One field — the bare value" example and before "**Several fields**":

> **A single field naming a list section returns that section, not a bare value.** `tick show tick-a1b2 --field notes` prints the notes table exactly as full output renders it, header and all; `--field tags` prints `tags[2]: has space,plain`. A list has no bare form, and the flag is a projection rather than a single-value extractor (§9.1), so the section is handed over in the one shape the library produces for it.
>
> The bare form belongs to a selection that resolves to exactly one value: the task's own fields, `description`, and a position that names one — `notes.2`'s text, `tags.1`'s item (§9.3). A selection that resolves to a row rather than a value — `children.1` — comes back as its one-row section.

Replace §9.7's paragraph with:

> **A request that returns a bare value ignores `--json`, `--pretty` and `--toon`; a request that returns a document honours them.** A bare value is not a document, so there is nothing for a format flag to act on, and honouring one would re-quote the very string the flag exists to hand over unquoted. A multi-field request produces a document, and so does a single field naming a list section (§9.2); the resolved format applies to either exactly as it applies to a full `tick show`.

**Resolution**: Approved
**Notes**: Applied under auto to both §9.2 and §9.7.

---

### 2. The import trim names descriptions and titles but not note text

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §2.2 (The fidelity bar is byte-identity)

**Problem**:
The byte-identity bar rests on one invariant — no stored value carries edge whitespace — and the fix that makes it true system-wide covers imported descriptions and titles. Note text is the third free-text carrier and is named as one, but no write path is said to normalise it. An import that brings notes across with a trailing space or a leading newline reproduces exactly the hole the import trim was added to close: the note reads out of `tick show` intact, is written back, and comes back different. A builder implementing the import fix has no way to tell whether notes are inside it, and the round-trip fixture (§11) carries note text through the same bar.

**Proposal**:
Settled by the stated aim of the import fix — making the invariant true system-wide rather than documenting an exception a reader cannot predict from the output. Whatever free text an import carries is normalised on the way in, note text included; the sentence naming descriptions and titles is widened to say so.

**Current**:
> **`tick migrate` therefore trims descriptions and titles on import, exactly as `create` does.** This is in scope for this work: it makes the invariant true system-wide rather than documenting an exception a reader cannot predict from the output.

**Proposed Text**:
> **`tick migrate` therefore trims every free-text value it imports, exactly as `create` does.** Today that is descriptions and titles — the import framework carries no notes (`grep -rn 'Note' internal/migrate/ --include='*.go' | grep -v _test` → no matches; `MigratedTask` holds Title, Status, Priority, Description and the three timestamps, `sed -n '30,38p' internal/migrate/migrate.go`) — and a provider that later brings note text across is covered by the same rule rather than by a second decision. This is in scope for this work: it makes the invariant true system-wide rather than documenting an exception a reader cannot predict from the output.

**Resolution**: Pending
**Notes**: Disposal — move held as `settled`, Proposed Text amended before presentation. The staged wording ("note text where a provider carries notes") reads to a builder as a requirement against a surface that does not exist: the import framework has no notes field at all. Rewritten to state the rule over whatever free text an import carries, with the measurement showing today's set is descriptions and titles.

---

### 3. JSON's notes are not given the index the toon table gains

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Important
**Affects**: §4.2 (JSON moves with toon), §6.3 (The notes section carries an index column)

**Problem**:
The notes table gains a 1-based `index` column because position is the only handle a note has, and an agent that asked for one note and then calls `note remove` deletes the wrong note without it. JSON is said to move with toon, but the two changes spelled out for JSON are the status-change list and the empty dep-tree form — notes are not among them. A JSON consumer using the same positional selection therefore gets a one-element notes array with no position in it and walks into the exact deletion the index column exists to prevent, or the builder adds the key on a hunch and the two formats disagree about what a notes entry contains.

**Proposal**:
Settled by the index column's own reasoning, which is about what the consumer can do with a filtered notes section and not about toon's syntax: the JSON notes entries carry the same 1-based index. §4.2's enumeration gains it.

**Current**:
> A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`).

**Proposed Text**:
> A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`).
>
> Each note also carries its 1-based index, for the reason the toon table does (§6.3): a consumer that asked for one note (§9.3) needs the note's real position before it can call `note remove`, and that need is the same whichever format it parses.

**Resolution**: Pending
**Notes**:

---

### 4. Nothing says whether `create` and `update` carry the changes section when nothing moved

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §7.4 (One document, never a document with loose lines after it), §7.2 (One table, always)

**Problem**:
The common case of `tick create` moves no other task's status — nothing cascades unless the new task lands under a finished parent. The document a caller gets back in that case either carries an empty changes table or omits the section entirely, and the specification does not say which. An agent parsing `create` output has to handle both until it has seen enough outputs to know, which is the branching-on-which-document-arrived that the single-table decision exists to remove.

**Proposal**:
Settled by the table's own reason for existing — a reader must not have to branch on which document arrived — together with the established pattern that an empty notes or children section emits a count-zero header rather than vanishing. The section is always present on `create` and `update`, count zero when nothing moved.

**Proposed Text**:
Add to §7.4, after "**Where a command produces both a record and status changes, the result is one document with the changes as a section inside it.** Making each section valid is not sufficient on its own; the stream must be one document.":

> **The section is always there.** `tick create` with no parent moves no task's status; the document still carries `changed[0]{id,title,from,to,auto}:`, exactly as an empty notes or children section carries its count-zero header (§8). A reader that must first find out whether the section exists is branching on which document arrived, which §7.2 exists to prevent.

**Resolution**: Pending
**Notes**:

---

### 5. The structured empty dep-tree answer is required but never shown

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §8 (Structured Output on Empty Branches), §5.2 (the dep-tree summary as named fields)

**Problem**:
The two prose answers `tick dep tree` gives today — nothing blocked anywhere, and a named task with no dependencies either way — are the one branch an agent could not predict, and they are required to become structured. What replaces them is not stated anywhere: no shape, no example, and the two branches are not said to take the same shape as each other. A builder decides on their own whether the empty answer carries the chains/longest/blocked fields at zero, whether the edges section appears with a zero count, and whether the named-task branch keeps a title. Two builders produce two different documents, and an agent has to handle both.

**Proposal**:
Settled by the pattern the section already cites: an empty task list and an empty edge set come back as count-zero sections, and the dep-tree summary becomes top-level named fields beside its table (§5.2). The empty answer is the same document with its counts at zero and its edges section empty, identical on both branches.

**Proposed Text**:
Add to §8, after "**Both go, in toon and JSON; pretty keeps them (§4.3).**":

> **What replaces them is the document the non-empty branch produces, emptied**: the summary fields of §5.2 reading zero and the edges section carrying its count-zero header. Both branches take that one shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent reads the same document whichever it hit, and reads the counts to learn which.

**Resolution**: Pending
**Notes**:

---

### 6. Positional selection is defined on notes and generalised in one clause

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §9.3 (List fields are reachable by position), §9.1 (accepted names), §9.6 (out-of-range positions)

**Problem**:
A position may be attached to any section that holds a list, which makes `tags.1`, `refs.2`, `children.3` and `blocked_by.1` valid requests — but everything said about how a position behaves is said about notes: the 1-based numbering, the narrowed section keeping its header while each row carries its real position in the index column, and the out-of-range error. Notes are the only section with an index column, so a builder reaching `children.2` has to decide unaided whether the numbering is 1-based there too, what the narrowed section looks like without an index to carry, and whether an out-of-range position on a section other than notes errors or comes back empty. An agent that gets a silent empty answer from `tags.5` instead of an error has been told something false about the data — the outcome the out-of-range rule was written to avoid.

**Proposal**:
Settled by the rule already stated for note positions, whose reasoning is about the addressing grammar rather than about notes: one grammar, applied the same way wherever it is accepted. Positions are 1-based on every list-holding section, the section renders in its normal form narrowed to the named item, and a position past the end is the §9.6 error naming the range.

**Proposed Text**:
Add to §9.3, after "A position narrows the section it names and nothing else: every other field in the selection comes back whole.":

> **The grammar is the same on every section that holds a list** — `notes.2`, `tags.1`, `refs.2`, `children.3`, `blocked_by.1`. Positions are 1-based throughout, the section renders in its normal form narrowed to the named item (a table keeps its header and one row; an inline list keeps its count and carries one item), and a position past the end is the error of §9.6 on any of them, naming the range the same way. Only notes carry a real position in the row itself, because only notes are addressed by position elsewhere in the CLI (§6.3).

**Resolution**: Pending
**Notes**:

---

### 7. The round-trip fixture asserts a comparison the trimming rule makes impossible

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §11 (Conformance Verification, part 2), §2.2 (The fidelity bar is byte-identity)

**Problem**:
The permanent fixture carries trailing spaces and asserts that the text read back out is identical to the text written in. Every CLI write path trims edge whitespace before storing, which is stated as accepted behaviour and explicitly not a defect, so that assertion cannot pass on a value whose whitespace sits at the edge — a builder writes the test, it fails, and the only way to make it pass is to undo the trim the specification decided to keep. The guarantee the fixture is there to protect is not write-in-read-out either: it is read a value out, write it back, and find the stored value unchanged.

**Proposal**:
Settled by the bar as §2.2 defines it — read out, write back, stored value byte-for-byte unchanged — and by the trim being edge-only, so the fixture's trailing spaces belong at the end of an interior line where nothing trims them.

**Current**:
> 2. **One deliberately awkward task becomes a permanent fixture, round-tripped end to end.** Free text carrying newlines, quotes, commas, a leading dash, trailing spaces, and a line that looks like a section header. Write it in, read it out, decode it, assert the text is identical to what went in. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability.

**Proposed Text**:
> 2. **One deliberately awkward task becomes a permanent fixture, round-tripped end to end.** Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets. Whitespace at the very start or end of a value is trimmed on the way in and stays trimmed, which is why the fixture carries its trailing spaces mid-text. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability.

**Resolution**: Pending
**Notes**:

---

### 8. The README correction claims the `tick list` sample goes stale

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §12.1 (The README is updated as part of this work), §1 (which sections are malformed)

**Problem**:
Both worked samples in the README's Output Formats section are said to show output the tool no longer produces. The task-list table is library-written and parses today, and no decision in this work changes it — the shapes that change are the detail header, tags, refs, the description block, the stats and dep-tree summaries, and the status-change lines. A builder given this either rewrites a `tick list` sample that was correct, inventing a difference to justify the edit, or stops to work out which of the two statements to believe. Neither produces a documentation task anyone can size.

**Proposal**:
Settled by the inventory of what changes: the `tick show` sample is replaced because its header, its tags and refs lists and its description block all change; the `tick list` table is untouched by this work, so that sample needs no edit on that account.

**Current**:
> Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work those samples show output the tool no longer produces.

**Proposed Text**:
> Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work the `tick show` sample shows output the tool no longer produces — its header, its tags and refs lists and its description block all change (§5, §6). The `tick list` table is library-written and untouched by this work, so that sample stands.

**Resolution**: Pending
**Notes**:

---

### 9. Whether the cascade specification is amended is left unsettled

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §12.2 (Two completed specifications are corrected selectively), §7.6 (Unchanged terminal children are not reinstated)

**Problem**:
The status-change decision points forward to "the correction owed" to the `auto-cascade-parent-status` specification, while the corrections section says correcting a completed unit's record is not obligatory, ranks one document as the more obviously wrong, and leaves the other unresolved. Two amendments, one amendment, or none are all readings a planner can defend, and each amendment is a separate task that touches another work unit's record and needs the user's confirmation before it is made. The plan either contains that task or it does not.

**Proposal**:
Settled by the rule the section itself sets — amend where a point is plainly and load-bearingly wrong. The cascade specification fixes the arrow-and-`(auto)` lines as the machine-readable cascade form, which the status-change table replaces outright, so both documents are amended; the unimplemented terminal-children requirement is left standing, since it is neither reinstated nor decided here.

**Current**:
> The `tick-core` unstructured-long-text principle is the most obviously wrong of the two, since it states as a rule the exact thing §6.2 removes.

**Proposed Text**:
> Both documents carry a point that is plainly and load-bearingly wrong, so both are amended. `tick-core` states as a rule the exact thing §6.2 removes. `auto-cascade-parent-status` fixes the arrow-and-`(auto)` lines as the machine-readable cascade form, which §7.2 replaces outright. Its requirement that unchanged terminal children be shown alongside a cascade is left standing — §7.6 neither reinstates nor decides it, and the amendment does not touch it.

**Resolution**: Pending
**Notes**:

---

### 10. The note-quoting behaviour is stated twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §6.3 (The notes section carries an index column), §2.3 (What is not a defect)

**Problem**:
That note text goes through the library's tabular encoder, which quotes and escapes newlines, carriage returns and tabs so a multi-line note survives as one quoted string, is stated in full in two places. The two copies agree today; an edit to either — the set of characters that force quoting, say — leaves the reader with two accounts of the same behaviour and no way to tell which is current.

**Proposal**:
The fact's home is §2.3, where it carries the "current output is not lossy" claim alongside the description block. §6.3 needs only the conclusion and a pointer, since its subject is the schema change.

**Current**:
> Notes already round-trip on the read side: note text goes through the library's tabular encoder, which quotes and escapes newlines, carriage returns and tabs, so a multi-line note is emitted as one unambiguous quoted string. What changes is the schema.

**Proposed Text**:
> Notes already round-trip on the read side (§2.3). What changes is the schema.

**Resolution**: Pending
**Notes**:

---

### 11. The migrate scope boundary is drawn in two places

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §2.2 (The fidelity bar is byte-identity), §3.3 (`doctor` and `migrate` are out of scope)

**Problem**:
The boundary — `migrate`'s import path is touched, its output is not — is stated whole in the fidelity section and stated whole again in the scope section, each pointing at the other. Two statements of one boundary is one more than can be kept in step, and the scope section is where a reader looks to find out what is in and out.

**Proposal**:
The boundary's home is §3.3, which owns scope and already carries the carve-out and the pointer back. §2.2's parenthetical restates it and can go.

**Current**:
> Import is already a translation boundary — statuses, priorities and timestamps are all mapped on the way in — so normalising whitespace there is the same kind of move, and it costs a reader nothing they would notice. (This is the only part of `migrate` this work touches; its output remains out of scope per §3.3.)

**Proposed Text**:
> Import is already a translation boundary — statuses, priorities and timestamps are all mapped on the way in — so normalising whitespace there is the same kind of move, and it costs a reader nothing they would notice.

**Resolution**: Pending
**Notes**:

---

### 12. Which formats keep the dep-tree prose is stated three times

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §4.3 (Two shared code paths must be split, not edited), §8 (Structured Output on Empty Branches), §4.1 (Pretty is unchanged, everywhere)

**Problem**:
That pretty keeps its empty-dep-tree sentence while the machine formats take the structured form is stated in the empty-branches requirement, listed among the things pretty keeps, and stated again at the end of the shared-code bullet. The bullet's own point — that the messages are set in the shared graph builder, so deleting them at source would strip pretty's too — already lands without the restatement, and three copies of one rule drift apart under editing.

**Proposal**:
The requirement's home is §8, and §4.1 already lists the dep-tree prose among what pretty keeps. §4.3's closing sentence adds nothing its preceding sentence has not established, so it goes.

**Current**:
> - **The dep-tree empty messages** are not produced by a formatter at all. They are set on the result in the shared graph builder (`grep -n 'No dependencies' internal/cli/dep_tree_graph.go` → `dep_tree_graph.go:177`, `dep_tree_graph.go:255`) and consumed by all three formatters, so removing them at source would strip pretty's message too. Pretty keeps its sentence; only the machine formats take the structured empty form.

**Proposed Text**:
> - **The dep-tree empty messages** are not produced by a formatter at all. They are set on the result in the shared graph builder (`grep -n 'No dependencies' internal/cli/dep_tree_graph.go` → `dep_tree_graph.go:177`, `dep_tree_graph.go:255`) and consumed by all three formatters, so removing them at source would strip pretty's message too.

**Resolution**: Pending
**Notes**:

---

### 13. The commands that accept the field flag are never named

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Minor
**Affects**: §9 (Field Selection), §9.1 (`--field` and `--fields` are the same flag)

**Problem**:
Every example puts the flag on `show`, but four other commands emit the same task-detail document, and the flag has to be registered against a specific command for the unrecognised-name error to fire at all and for the help-text consistency check to pass. A builder registering it has to guess whether `create`, `update` and the `note` subcommands take it too; guessing wide means help text and documentation for a surface nobody decided to offer, guessing narrow means a caller who tried `tick update <id> --title x --field id` gets an unknown-flag error the specification never mentions.

**Proposal**:
Settled by the section's framing throughout — the flag is introduced as `show`'s first command-specific flag, every example is a `show` invocation, and the README entry is written as `--field`/`--fields` on `show`. It is registered for `show` alone.

**Proposed Text**:
Add to §9, after "`show` accepts no command-specific flags today (`grep -n '\"show\":' internal/cli/flags.go` → `flags.go:72`, `\"show\": {}`), so this is its first, alongside the global `--quiet`.":

> It is `show`'s flag and no other command's. `create`, `update`, `note add` and `note remove` emit the same detail document but take no field selection: a caller that wants one value out of them runs `tick show --field` afterwards, and the flag stays registered against a single command.

**Resolution**: Pending
**Notes**:

---

### 14. "Refused" does not say what the caller gets

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Minor
**Affects**: §9.8 (Interaction with `--quiet`)

**Problem**:
Passing `--quiet` alongside a field selection is refused, but refusal could mean an error on stderr with a non-zero exit, or a warning with output still produced. A script that pairs the two flags by accident needs the exit code to tell it what happened, and the builder has no statement of what to return.

**Proposal**:
Settled by the treatment the neighbouring rejections already take — an unrecognised field name and an out-of-range position both fail with a non-zero exit and no stdout. The same applies here.

**Proposed Text**:
Add to §9.8, after the existing paragraph:

> The refusal is an error with a non-zero exit and nothing on stdout, exactly as an unrecognised field name is (§9.6).

**Resolution**: Pending
**Notes**:

---

### 15. A selection that names nothing has no defined outcome

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Minor
**Affects**: §9.6 (Empty values, unrecognised names, and out-of-range positions)

**Problem**:
The flag takes a comma-separated list, and three outcomes are defined: a field that is empty, a name that is not a field, and a position past the end. A list that names nothing at all — an empty value, or a stray or doubled comma from a shell variable that came back blank — is none of the three. An agent building the flag from a variable is exactly the caller who hits it, and gets silence, a full record, or an error depending on which the builder chose.

**Proposal**:
Settled by the rule already covering a name that is not a field: an empty name is a caller mistake, not a field that happens to be empty, so it takes the unrecognised-name error rather than printing nothing or falling back to the full record.

**Proposed Text**:
Add to §9.6, after the unrecognised-name paragraph:

> **A selection that names nothing is the same mistake.** An empty value, or a stray or doubled comma leaving an empty name in the list, fails with the unrecognised-name error rather than printing nothing or falling back to the full record. A blank name is a caller mistake, not a field that happens to be empty.

**Resolution**: Pending
**Notes**:

---

## Working Notes

Cycle 1 gap analysis, standalone reading only. Fifteen findings: one critical (the shape of a single field naming a section), eight important, six minor.

Nothing in §1–§5 reads as under-specified beyond finding 2's carrier gap; the cost-accepted and declined-alternative passages throughout carry their reasoning and need nothing added. Section ordering within the detail document ("sections keep their usual output order", §9.2) was considered and not raised: order is not observable to a decoder, and the existing formatter order stands unchanged.
