# Review Tracking: Free Text Round Trip - Gap Analysis

## Findings

### 1. The named-task dep-tree document has no stated way of identifying its task

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §5.2 (the required shape for single-object sections), §8 (structured output on empty branches), §3.1 (every branch must decode)

**Problem**:
`tick dep tree <task>` prints a line naming the task whose dependencies are being shown, above the edges. Every branch of that command's toon output has to decode with a standard TOON reader, and the empty branch has to identify the named task the same way the populated one does — but nothing says what that identifying line becomes. Three hand-written single-object sites get a stated replacement: the task-detail header, the `stats` counts, and the whole-project chains/longest/blocked summary. The named-task line is a fourth, and it is left where it is. A builder either keeps today's line, in which case the focused dep-tree document can still fail on its first line — the precise failure this work exists to delete, and the conformance test of §11 would then fail with nothing in the specification telling the builder what to write instead — or invents a shape: a `task:` wrapper, a bare title, an id-and-title pair. Agents asking the same question of two builds would get two different documents.

**Proposal**:
State, in §5.2 alongside the other single-object sites, that where `tick dep tree` names a task the identifying line becomes top-level `id` and `title` named fields, in the same form the task-detail header takes. §5.4 already ruled out a wrapping key and §5.2 already fixed named fields as the form for content describing one thing, so those two decisions leave one defensible shape; carrying the ID as well as the title matches what the line identifies today and what a filtered document is said to preserve in §8.

**Proposed Text**:
Append to the final paragraph of §5.2, after "become top-level named fields beside their tables.":

"Where `tick dep tree` names a task, the line identifying that task becomes top-level `id` and `title` fields in the same form, so a focused dependency document opens exactly as a task-detail document does."

**Resolution**: Pending
**Notes**:

---

### 2. The bare-value terminator rests on a claim that does not hold for imported tasks

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Minor
**Affects**: §9.2 (one field returns the bare value) — colliding with §2.2, which states that values already in storage are left alone and a task imported by an earlier version keeps its untrimmed value

**Problem**:
An agent is told that the single newline closing a bare field value is always the terminator and never part of the value, on the grounds that no stored value carries edge whitespace. That ground is stated without limit, while the round-trip contract explicitly exempts tasks imported before this change — those keep whatever whitespace they arrived with and no pass rewrites them. On exactly those tasks an agent that trusts the unlimited claim will strip a newline that belonged to the description and write back a value one byte short, on the very tasks nobody typed by hand.

**Current**:
Stored values carry no edge whitespace (§2.2), so that byte is the terminator and never part of the value.

**Proposed Text**:
Values stored from this change onward carry no edge whitespace (§2.2), so that byte is the terminator and never part of the value.

**Resolution**: Pending
**Notes**:

---

### 3. The list of conditionally-present fields is written out twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §9.6 (empty values) — the fact's home is §5.2, which states which fields appear only when the task carries them

**Problem**:
Which fields a task-detail document omits when the task does not carry them is settled in one place and then written out name-for-name in a second. The moment a field is added to the model, the two copies answer differently, and a builder asking what `--field <new-field>` prints on a task that lacks it gets one answer from the document's shape rules and another from the field-selection rules.

**Current**:
**An empty field prints what full output prints for it, and exits successfully.** A task legitimately having no description is a fact about the task rather than a failure of the command. A field or section that full output omits when the task does not carry it — `description`, `tags`, `refs`, `type`, `parent`, `closed` (§5.2) — prints nothing; a section full output always carries prints its count-zero header, so `--field notes` on a task with no notes returns `notes[0]{index,text,created}:`.

**Proposed Text**:
**An empty field prints what full output prints for it, and exits successfully.** A task legitimately having no description is a fact about the task rather than a failure of the command. A field or section full output omits when the task does not carry it (§5.2) prints nothing; a section full output always carries prints its count-zero header, so `--field notes` on a task with no notes returns `notes[0]{index,text,created}:`.

**Resolution**: Pending
**Notes**:

---

### 4. The out-of-range position rule is given twice in the same section

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §9.6 (unrecognised names and out-of-range positions)

**Problem**:
The answer for a position past the end of a list — `notes.4` on a two-note task, non-zero exit, a message naming the range — is stated in one paragraph and then stated again in the next, which widens it to every list section. A builder reading the second cannot tell whether it adds a rule or repeats one, and a later edit to the wording of either leaves two rules for one behaviour.

**Current**:
**A note position that does not exist is an error too.** `notes.4` on a task carrying two notes fails with a non-zero exit and a message naming the range, rather than printing nothing and succeeding. A selector that resolves to nothing is not a field that happens to be empty — the position is a claim about the data that is false. `tick note remove` already answers this for the same 1-based addressing, reporting the index as out of range and naming how many notes the task has (§6.3); giving the same grammar a different answer under a different command would be a second rule for a reader to learn.

**A position that names nothing is out of range whatever the reason.** `notes.4` on a two-note task, `notes.0` where positions start at 1, and `tags.1` on a task carrying no tags all fail the same way: non-zero exit and a message naming the range. A section the task does not carry is not the empty-field case above — `tags` alone on a tag-less task is a field that happens to be empty, while `tags.1` is a claim that a first tag exists. A suffix that is not a number is no positional claim at all and takes the unrecognised-name error (§9.1).

**Proposed Text**:
**A position that names nothing is out of range whatever the reason.** `notes.4` on a task carrying two notes, `notes.0` where positions start at 1, and `tags.1` on a task carrying no tags all fail the same way: a non-zero exit and a message naming the range, rather than printing nothing and succeeding. A selector that resolves to nothing is not a field that happens to be empty — the position is a claim about the data that is false. A section the task does not carry is not the empty-field case above either: `tags` alone on a tag-less task is a field that happens to be empty, while `tags.1` is a claim that a first tag exists. `tick note remove` already answers this for the same 1-based addressing, reporting the index as out of range and naming how many notes the task has (§6.3); giving the same grammar a different answer under a different command would be a second rule for a reader to learn. A suffix that is not a number is no positional claim at all and takes the unrecognised-name error (§9.1).

**Resolution**: Pending
**Notes**:

---

### 5. A bare field that is empty does not say whether anything at all is printed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Minor
**Affects**: §9.6 (empty values), §9.2 (the bare value goes out as its own bytes followed by a single newline)

**Problem**:
`tick show <id> --field description` on a task with no description is required to print what full output prints — nothing — and to exit successfully. It is separately required, as a bare-value request, to emit the value followed by a single newline. A builder writing the obvious one-line print emits a lone newline; a builder following the empty rule emits zero bytes. An agent testing whether a task carries a description by measuring the output gets a different answer from the two builds, on a contract whose whole point is that byte counts are trustworthy.

**Proposal**:
Say in §9.6 that in the bare form an empty field produces no bytes at all, the newline of §9.2 belonging to a value rather than to the request. Determined by §9.6's own rule that an empty field prints what full output prints for it — full output emits nothing for an absent field, and a lone newline is not nothing.

**Proposed Text**:
Append to the first paragraph of §9.6, after the sentence ending "returns `notes[0]{index,text,created}:`.":

"In the bare form that means no bytes at all: the terminating newline of §9.2 belongs to a value, so a field with no value produces an empty stream rather than a blank line."

**Resolution**: Pending
**Notes**:

---

### 6. A cross-reference sends the reader away from the fact it is citing

**Source**: Specification analysis
**Category**: Enhancement to existing topic
**Move**: settled
**Priority**: Minor
**Affects**: §5.2 (which sections a document carries)

**Problem**:
The rule that some task fields appear only when the task carries them is stated, then cited to the field-selection section — which states nothing of the kind and cites the rule straight back. A reader checking the point makes two hops and returns to where they started, and a builder looking for a second, narrower rule about presence spends the search finding there isn't one.

**Current**:
while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them (§9.1).

**Proposed Text**:
while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them.

**Resolution**: Pending
**Notes**:

---

## Working Notes

Read end-to-end as an implementer. Areas checked and found internally complete, recorded so a later cycle does not re-tread them:

- **Field selection grammar (§9.1–§9.8)**: the one-field/several-fields split, positional suffixes on every list section, repeated and duplicate names, recognition-without-presence, the unrecognised-name and out-of-range errors, `--quiet` refusal, and the format-flag interaction all resolve without guessing. `--field title,title` resolving to the bare form follows from §9.2's "resolves to exactly one value" read against §9.1's "renders once".
- **`changed` table (§7)**: column set, the always-present rule for `create`/`update`, its absence from `show`/`note add`/`note remove`, and the created task never appearing in its own table (§7.4's zero-count example) are all determined.
- **`--` marker (§10.2)**: reads as CLI-wide — it is stated as currently rejected on every command and as stopping global flag consumption, not merely command flag validation.
- **Empty branches (§8)** and **conformance verification (§11)**: actionable as written; §11's "every branch" is bounded by the enumerated inclusions.
- No open-decision markers ("TBD", "decision required") anywhere in the document.
