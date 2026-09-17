# Review Tracking: Free Text Round Trip - Gap Analysis

## Findings

### 1. The round-trip test exercises one kind of free text, while the guarantee covers three

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Important
**Affects**: §11 (conformance verification, part 2 — the permanent awkward fixture), §2.2 (the byte-identity bar over all three free-text carriers), §6.4 (notes have no edit command)

**Problem**:
The promise this work makes is that an agent can read free text out of `tick`, write it back, and land the identical bytes — for descriptions, note text and titles alike. The one test that checks that promise is described as a single awkward task round-tripped end to end, without saying which of the three kinds of text it carries or how each is written back. A builder who puts the awkward content in the description alone ships a suite that never exercises the two carriers this work had to unblock: note text and titles, which are refused today when they begin with a dash. Writing a note back is not obvious either — a note cannot be edited, so "write the decoded text back" means adding it again and comparing the new note's stored text.

**Proposal**:
Say that the fixture carries the awkward text in all three carriers — title, description and a note — each taking what its shape allows, and that the note is written back by adding it again. The byte-identity bar is stated over all three carriers, and the dash-leading input this work unblocks reaches storage only through a bare title argument and note text, so a description-only fixture cannot fail on the paths the work repaired.

**Current**:
> 2. **One deliberately awkward task becomes a permanent fixture, round-tripped end to end.** Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets.

**Proposed Text**:
> 2. **One deliberately awkward task becomes a permanent fixture, round-tripped end to end.** Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. It carries that text in all three free-text carriers — title, description and a note — each taking what its shape allows: the title and the note text begin with a dash, and the description carries the multi-line content. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets. A note is written back by adding it again, since notes carry no edit (§6.4), and the assertion is made against the new note's stored text.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 2. The correction owed to the cascade specification names a requirement this work leaves standing

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §12.2 (two completed specifications are corrected selectively — the table row for `auto-cascade-parent-status`), colliding with the same section's prose and with §7.6, both of which state that the requirement to show unchanged terminal children alongside a cascade is not decided here and is not touched by the amendment

**Problem**:
Another work unit's completed specification is about to be amended, and the amendment is described twice with two different extents. The tabular summary of what is wrong in that document names two things: its arrow-and-`(auto)` cascade lines, and its requirement that unchanged terminal children be shown alongside a cascade. The prose immediately below names only the first and says the second is deliberately left alone. Someone carrying out the correction from the table deletes a standing requirement from another unit's permanent record — an edit that also re-indexes that document into the knowledge base, so the deletion is served as validated context afterwards.

**Proposal**:
Drop the second clause from the table cell so the row names only the claim that is wrong — the arrow-and-`(auto)` lines the status-change table replaces. The line range cited stays as it is; the prose beneath already states what is amended and what is not.

**Current**:
> | `auto-cascade-parent-status` specification | Fixes the arrow-and-`(auto)` lines as the machine-readable cascade form and requires unchanged terminal children to be shown alongside them (`sed -n '117,162p' .workflows/auto-cascade-parent-status/specification/auto-cascade-parent-status/specification.md`) | Work unit completed |

**Proposed Text**:
> | `auto-cascade-parent-status` specification | Fixes the arrow-and-`(auto)` lines as the machine-readable cascade form (`sed -n '117,162p' .workflows/auto-cascade-parent-status/specification/auto-cascade-parent-status/specification.md`) | Work unit completed |

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 3. Trimming imported text says nothing about a value that trims away to nothing

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §2.2 (the fidelity bar is byte-identity — the import-path trim)

**Problem**:
Imported tasks will have their free text trimmed on the way in, and the rule is given as "exactly as `create` does". On the command line a description that is nothing but whitespace is refused, with the caller pointed at the flag that clears a description instead. A builder reading the import rule as the same thing makes an import fail on a task the source tool was perfectly happy with — a migration aborting partway over invisible characters in a field the user never sees. Reading it as a trim and nothing more stores the empty string. Nothing in the document picks between them.

**Proposal**:
Say the import trim normalises rather than validates: a value that is nothing but whitespace stores as empty and no import fails over it. Import is a translation boundary, and the point of the trim is to make the byte-identity bar hold system-wide, which an empty stored value satisfies; refusing an import over whitespace would add a failure mode nobody hit before and the trim was not introduced to gate anything.

**Proposed Text**:
Append to the paragraph beginning "**`tick migrate` therefore trims every free-text value it imports…**":

> The trim normalises, it does not validate: a value that is nothing but whitespace stores as empty, and no import fails because of it.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 4. Nothing says a task appears once in the table of what changed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §7.2 (one table, always), §7.4 (two independent cascade blocks collapse into the single table), §7.5 (why a structural change produces a status cascade)

**Problem**:
Moving a task to a different parent fires two unrelated cascades at once — the new parent reopening, the old parent auto-completing — and each travels up its own chain. Where those chains meet on a shared ancestor, that one task's status is touched twice by a single command. The table of what changed is described as a row per task whose status moved, and the two cascade results are described as collapsing into it, but nothing says the collapse merges by task. A builder who concatenates the two results hands an agent the same task ID on two rows with conflicting `from`/`to` values; an agent that takes the first row and stops now holds a status the task does not have, and the count in the header no longer counts tasks.

**Proposal**:
State that a task appears at most once: the row reads from the status the task held before the command to the status it holds after, and a task that ends where it started carries no row. The table's stated job is to say what changed, and the value an agent needs is the state it will find on a later read, so the net move is the only reading that survives the collapse the section already requires.

**Proposed Text**:
Append to §7.2, after the paragraph beginning "The table carries the title so no second lookup is needed…":

> **A task appears at most once.** One command can move the same task's status twice — the two cascades of §7.5 meeting on a shared ancestor — and the table still carries one row for it, reading from the status it held before the command to the status it holds after. A task that ends where it started carries no row: the table lists what changed.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 5. A field request that answers with nothing is still held to the parsing rule

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §3.1 (output that must parse — the single stated exception), colliding with §9.6, which has a field selection that resolves to nothing print nothing at all and exit successfully; §9.7 (a request returning a document honours the format flags)

**Problem**:
Asking for a field the task does not carry prints nothing and succeeds — `--field tags` on a task with no tags, or a multi-name selection where every name is empty. That answer is not a bare value, so the rule that every document `show` produces must decode reaches it, and a builder has two ways to satisfy it: emit no bytes, or emit an empty document. In JSON the fork is visible to the caller — zero bytes against `{}` — and an agent that pipes the output into a JSON reader gets a parse error on one branch and an empty object on the other. The conformance suite inherits the same fork: it must either decode a zero-byte answer or leave it uncovered.

**Proposal**:
Extend the exception to output with no bytes, in every format: a request that prints nothing is nothing rather than an empty document, so the parsing rule does not reach it. The behaviour is already fixed — such a request prints nothing and exits successfully, stated without regard to format — so the only open point is the parsing rule's reach, and an empty stream cannot be a document in any of the three formats.

**Current**:
> Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8). The single exception is a bare value from `tick show --field` (§9.2), which is by design not a document: it is exempt here for the reason §9.7 exempts it from the format flags. Every document `show` produces — the full detail, and a filtered one — decodes.

**Proposed Text**:
> Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8). Two things are exempt, both because they are not documents: a bare value from `tick show --field` (§9.2), for the reason §9.7 exempts it from the format flags, and a field selection that prints no bytes at all (§9.6), which is nothing rather than an empty document, in every format. Every document `show` produces — the full detail, and a filtered one — decodes.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 6. A field named twice leaves the shape of the answer undecided

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Minor
**Affects**: §9.1 (the list is read leniently), §9.2 (one field returns the bare value; several return a filtered document)

**Problem**:
The answer's shape turns on how many fields were asked for: one gets the raw value with nothing around it, several get a document with labels. A caller who names the same field twice — assembling the list from a variable, or repeating the flag — has asked for one field written twice, and the document does not say which side of that split it lands on. One builder returns the bare text, another returns `title: …`, and an agent reading the value gets a label it then has to strip.

**Proposal**:
Say that a repeated name counts once for the split, so a selection naming one field however many times returns the bare value. A repeat already renders once in the output, so the selection is the set of distinct names; counting the typed list instead would make the same request answer in two shapes depending on typing, which is the branching the flag's design avoids.

**Current**:
> **The list is read leniently wherever its meaning is not in doubt.** Whitespace around a name is not part of it, so `--field "title, status"` selects what `--field title,status` selects. Repeating the flag composes rather than overrides: `--field title --field status` is `--field title,status`. A field named more than once renders once — a section named both whole and by position comes back whole, and several positions on one section render it narrowed to those positions in output order. None of this softens §9.6: a name that is empty once its whitespace is gone is still the blank-name mistake.

**Proposed Text**:
> **The list is read leniently wherever its meaning is not in doubt.** Whitespace around a name is not part of it, so `--field "title, status"` selects what `--field title,status` selects. Repeating the flag composes rather than overrides: `--field title --field status` is `--field title,status`. A field named more than once renders once, and counts once when the answer splits by how many fields were asked for (§9.2) — `--field title,title` is `--field title` and comes back as the bare value. A section named both whole and by position comes back whole, and several positions on one section render it narrowed to those positions in output order. None of this softens §9.6: a name that is empty once its whitespace is gone is still the blank-name mistake.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

### 7. Why the trailing newline is a terminator is settled in two places

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §2.2 (the whitespace invariant over all three carriers) — the fact's home is §9.2, which decides the bare value's form and states that stored values carry no edge whitespace, so the closing byte terminates the value rather than belonging to it

**Problem**:
The same consequence is drawn in two places: where the bare value's form is decided, and again in the passage about trimming on the way in. Two copies of one rule drift apart the first time either is edited, and a reader who finds the weaker copy first has to go looking for the other to be sure of what the closing byte means.

**Proposal**:
Leave the consequence where the bare value is decided and keep the trimming passage to its own point — that the invariant covers note text and titles, not descriptions alone.

**Current**:
> Stating it over all three is what makes the bar hold for a note read out and written back, and what makes the terminating newline of §9.2 a terminator rather than part of the value, whichever field was asked for.

**Proposed Text**:
> Stating it over all three is what makes the bar hold for a note or a title read out and written back, not descriptions alone.

**Resolution**: Approved
**Notes**: Applied verbatim under auto.

---

## Working Notes

Cycle 5 reads the specification cold, as an implementer. The document is close to buildable: the shapes in §5–§9 carry examples, the declined alternatives carry their reasoning, and the four earlier cycles have closed the large holes (section identity on the focused dep-tree branch, the bare value's terminator, the empty-selection outcome, the lenient list reading).

Considered and not raised:

- **Row order in the `changed` table** — not observable to a decoder, and the `auto` column identifies the requested change without relying on position, so order is the builder's.
- **Section order in a filtered document** — settled by "sections keep their usual output order", and the existing formatter order stands unchanged.
- **Field names in the `stats` and dep-tree summaries** — "which sections a document carries is unchanged by this work" holds them at their current names; only their shape moves.
- **Where `--` appears in help text** — placement is the builder's; that it is documented is already required.
- **Case sensitivity of field names** — "spelled as a full `tick show` spells them" settles it.
