# Review Tracking: Free Text Round Trip - Gap Analysis

## Findings

### 1. The must-parse rule admits no exception, and the flag's whole purpose is one

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §3.1 (Output that must parse), §9.2 (One field returns the bare value), §11 (Conformance Verification, part 1)

**Problem**:
`tick show` is named as a command whose output a standard TOON reader decodes on every branch, and the same command is required to print a bare multi-line string — no header, no quoting — when a single field is asked for. A bare description with blank lines and indented bullets is not a TOON document and will not decode. The conformance suite is built from that same list, so a builder writing "decode every structured command's output" either excludes the field-selection path on their own judgement or writes a test that cannot pass, and a reader of the specification cannot tell which of the two statements is the rule.

**Proposal**:
Settled by the flag's own reasoning, already written down: §9.7 exempts a bare value from the format flags on the grounds that it is not a document at all. The same ground exempts it from the must-parse rule, and saying so once in §3.1 carries into §11, which takes its command list from there.

**Current**:
> Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8).

**Proposed Text**:
> Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8). The single exception is a bare value from `tick show --field` (§9.2), which is by design not a document: it is exempt here for the reason §9.7 exempts it from the format flags. Every document `show` produces — the full detail, and a filtered one — decodes.

**Resolution**: Pending
**Notes**:

---

### 2. Whether `note add`, `note remove` and `show` carry the changes section is never said

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §7.3 (The per-command split is kept), §7.4 (One document, never a document with loose lines after it), §3.1 (Output that must parse)

**Problem**:
Five commands print the task-detail document, and four of them share one helper to do it — the specification says so itself. The changes table is required to be present in `create` and `update` output even when nothing moved, justified by a reader never having to branch on which document arrived. A builder adding the section to the shared helper gives `note add` and `note remove` a permanent `changed[0]` section; a builder adding it to the two edit paths does not; and `tick show`, which renders the detail inline, is a third case again. The agent reading task-detail output then finds the section present, absent, or always empty depending on which command it ran — the branching the always-present rule exists to remove, reintroduced one level up.

**Proposal**:
Settled by what the section carries: only a parent/child structural change moves another task's status (§7.5), and adding or retracting a note changes no structure, so there is nothing for the section to hold on the note commands or on `show`. The section belongs to `create` and `update`, which is also the split §7.3 already draws for the full record.

**Proposed Text**:
Add to §7.3, after "The deciding factor: `create` and `update` are edits and the caller wants the result of the edit — the new ID, the merged fields — whereas a status change is something the caller already knows it did, so a full record is tokens it did not ask for.":

> `show`, `note add` and `note remove` carry no `changed` section at all. Only a parent/child structural change moves another task's status (§7.5), and none of the three performs one, so there is nothing for the section to hold. The section belongs to `create` and `update`, even though all four mutating commands share one detail helper (§3.1).

**Resolution**: Pending
**Notes**:

---

### 3. Which sections a detail document always carries is never stated

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §5.2 (The required shape), §9.1 (the names the flag accepts), §8 (Structured Output on Empty Branches)

**Problem**:
The worked document for `tick show` prints seven task fields, a children section and a description — and no notes section, no blocked-by section, no tags or refs. Elsewhere the specification says notes and related sections emit a count-zero header when empty, and that tags, refs, description, type, parent and closed appear only when the task carries them; it never says that presence rule holds after this work, and the requirement that the changes table be present even at count zero — argued from a reader never having to branch on which document arrived — reads as a principle that ought to generalise. A builder therefore either keeps today's mixed behaviour or emits every section unconditionally, and an agent handed a document with no `tags` line cannot tell whether the task has no tags or the section was dropped.

**Proposal**:
Settled by the two statements already in the document: §8 records that related and notes sections emit a count-zero header rather than prose, and §9.1 records that type, parent, closed, tags, refs and description are emitted only when set — and relies on that behaviour for its rule that recognition does not depend on presence. Nothing in this work changes either; the always-present rule of §7.4 is the changes table's. Stating it where the document's shape is defined stops the generalisation.

**Proposed Text**:
Add to §5.2, after the worked example and before "**The same treatment applies to the other two single-object sites**":

> **Which sections a document carries is unchanged by this work.** The example shows the form, not the full complement: `children`, `blocked_by` and `notes` are always present, carrying a count-zero header when empty (§8), while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them (§9.1). The always-present rule of §7.4 is the `changed` section's and does not extend to the rest.

**Resolution**: Pending
**Notes**:

---

### 4. What `--` does to the arguments after it is not stated

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §10.2 (The fix, both halves)

**Problem**:
`--` becomes the documented way to pass free text that may begin with a dash, and the agent that learns it will write `tick create -- "- fix the parser" --priority 2`. What comes back depends on a decision nobody made: if the marker only suppresses the unknown-flag check, the priority is set; if it ends flag parsing, `--priority 2` is text or a surplus argument and the command either errors or stores a task at the default priority. The global format flags sit in the same fog — an agent that puts `--json` after the marker gets JSON output or a task titled `--json`. A caller cannot be told to use a marker whose effect on everything after it is undefined.

**Proposal**:
Settled by the name the specification already gives it — the end-of-flags marker — which has one meaning: nothing after it is a flag, at any level. Saying so makes the ordering rule explicit for the caller, and closes the last corner of the round-trip hole §10 exists to fix, where free text spells a registered flag exactly.

**Proposed Text**:
Add to §10.2, after the `--` bullet's paragraph and before the `note add` bullet, as a continuation of the first bullet:

> Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text. Free text that spells a flag exactly, a note reading `--json`, is writable for the same reason a dash-leading one is.

**Resolution**: Pending
**Notes**:

---

### 5. A position that names nothing has one rule for one case out of three

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §9.6 (Empty values, unrecognised names, and out-of-range positions), §9.3 (List fields are reachable by position), §9.1 (the names the flag accepts)

**Problem**:
Asking for the fourth note of a two-note task is an error naming the range. Three neighbouring requests have no answer. `notes.1` on a task with no notes at all reads as the empty-field case — a name the task does not carry, which prints nothing and exits zero — and as the out-of-range case at the same time, so a builder picks one and an agent probing a note-less task gets either silence or a failure. `notes.0` is worse: positions are 1-based, so zero names nothing, but a builder passing it to a zero-based slice hands back the first note and the caller then retracts the wrong one by index. And a suffix that is not a number at all, `notes.x`, falls under neither rule.

**Proposal**:
Settled by the reason already given for the out-of-range error — a selector that resolves to nothing is not a field that happens to be empty, because the position is a claim about the data that is false. That covers an absent section and a zero position alike. A suffix that is not a number makes no positional claim at all, so it is the unrecognised name of §9.1.

**Proposed Text**:
Add to §9.6, after the out-of-range paragraph:

> **A position that names nothing is out of range whatever the reason.** `notes.4` on a two-note task, `notes.0` where positions start at 1, and `tags.1` on a task carrying no tags all fail the same way: non-zero exit and a message naming the range. A section the task does not carry is not the empty-field case above — `tags` alone on a tag-less task is a field that happens to be empty, while `tags.1` is a claim that a first tag exists. A suffix that is not a number is no positional claim at all and takes the unrecognised-name error (§9.1).

**Resolution**: Pending
**Notes**:

---

### 6. The bare value's terminator is left to the builder

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Minor
**Affects**: §9.2 (One field returns the bare value)

**Problem**:
The bare form exists so a caller can take a value away without parsing anything off it, and the fidelity bar is byte-identity. Whether the value arrives with a trailing newline or not is the one byte the specification does not account for: a builder who omits it leaves a terminal prompt running into the last line of a description, and one who adds it gives a caller comparing bytes a value that differs from what is stored by exactly that byte.

**Proposal**:
Settled by the treatment of the neighbouring bare answer: §9.8 puts a single-field request and `--quiet`'s bare task ID side by side as two answers of the same kind, and a task ID is written as a line. The value goes out followed by one newline, and §2.2's no-edge-whitespace invariant makes that byte unambiguously the terminator rather than part of the value.

**Proposed Text**:
Add to §9.2, after the `--field description` example and before "**A single field naming a list section returns that section, not a bare value.**":

> The value goes out as a line: its own bytes followed by a single newline. Stored values carry no edge whitespace (§2.2), so that byte is the terminator and never part of the value.

**Resolution**: Pending
**Notes**:

---

### 7. One section, two names

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Minor
**Affects**: §1 (Purpose, the parses/does-not-parse table), §9.1 (the names the flag accepts), §9.3 (List fields are reachable by position)

**Problem**:
The section listing what a task is blocked by is called `blockers` where the output inventory is drawn and `blocked_by` where the selectable names are listed. The field flag accepts exactly the names the output uses and errors on anything else, so the two spellings cannot both be right: a caller who reads one and types it gets a non-zero exit, and a builder writing the accepted-name list has two candidates for the same key.

**Proposal**:
Settled by §9.1, which is the one place that states the accepted vocabulary and pins it to the output's own spelling — `blocked_by` there and in §9.3's positional grammar, against a single prose mention elsewhere. The inventory row takes the same spelling.

**Current**:
> | The library (`encodeToonSection`) | blockers, children, notes, priority breakdown, dep-tree edges | yes |

**Proposed Text**:
> | The library (`encodeToonSection`) | blocked_by, children, notes, priority breakdown, dep-tree edges | yes |

**Resolution**: Pending
**Notes**:

---

### 8. The shared transition line is explained twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §7.1 (The current shape), §4.3 (Two shared code paths must be split, not edited)

**Problem**:
That toon and pretty emit byte-identical transition lines because both formatters embed the same base method is stated in full in the shared-code-path section, where it carries the requirement to split that method rather than edit it, and stated in full again where the current arrow form is described. The second copy cites the first while restating it, so an edit to either — a different method, a third embedder — leaves two accounts of one code fact and no way to tell which is current.

**Proposal**:
The fact's home is §4.3: the requirement to split rather than edit rests on it, and it is measured there. §7.1 needs only that the arrow form is shared with pretty, which the pointer already carries.

**Current**:
> Reading it requires knowing that the ID precedes the colon, that the arrow separates old state from new, and that `(auto)` marks a knock-on rather than the requested change. It is a bespoke line format, and it is byte-identical in `--toon` and `--pretty`: both formatters embed `baseFormatter.FormatTransition` (§4.3), and `ToonFormatter.FormatCascadeTransition` is the same construction with ` (auto)` appended (`grep -n 'func (f \*ToonFormatter) FormatCascadeTransition' internal/cli/toon_formatter.go` → `toon_formatter.go:145`).

**Proposed Text**:
> Reading it requires knowing that the ID precedes the colon, that the arrow separates old state from new, and that `(auto)` marks a knock-on rather than the requested change. It is a bespoke line format, shared with pretty (§4.3), and `ToonFormatter.FormatCascadeTransition` is the same construction with ` (auto)` appended (`grep -n 'func (f \*ToonFormatter) FormatCascadeTransition' internal/cli/toon_formatter.go` → `toon_formatter.go:145`).

**Resolution**: Pending
**Notes**:

---

### 9. The commands that print the detail document are enumerated twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §5.1 (The current shape and its cause), §3.1 (Output that must parse)

**Problem**:
Which five commands print the task-detail document is established in the output inventory, with the evidence for the shared helper and for `show` rendering it inline. The single-object section restates the same five in a parenthesis. If the set ever moves — a sixth command, or one of the four dropping the helper — one of the two lists is updated and the other quietly disagrees.

**Proposal**:
The fact's home is §3.1, where the set is enumerated with its measurement and where it carries the must-parse requirement. §5.1 needs only to name the document.

**Current**:
> Three places in the output describe one thing rather than a list of things: the task's own fields at the head of `tick show` (and of `create`, `update`, `note add`, `note remove`), the counts summary in `tick stats`, and the chains/longest/blocked summary in `tick dep tree`.

**Proposed Text**:
> Three places in the output describe one thing rather than a list of things: the task's own fields at the head of the task-detail document (§3.1), the counts summary in `tick stats`, and the chains/longest/blocked summary in `tick dep tree`.

**Resolution**: Pending
**Notes**:

---

### 10. The failure that triggered the work is told twice

**Source**: Specification analysis
**Category**: Duplication
**Move**: settled
**Priority**: Minor
**Affects**: §9 (Field Selection), §1 (Purpose)

**Problem**:
An agent abandoning the CLI and reading `.tick/tasks.jsonl` to get a description as a plain string is the opening of the purpose section and is told again as the opening of the field-selection section. Two accounts of one event drift apart at the first edit to either, and the second adds nothing the first has not said.

**Proposal**:
The account's home is §1, where it motivates the whole piece of work. §9 needs only the pointer, which keeps the connection without a second telling.

**Current**:
> The companion to the format repair: a way to ask for one field's value and get it with nothing around it — no header, no indentation, no quoting. This is the case that started the work, where an agent needed a task's description as a plain string and went to the raw data file instead.

**Proposed Text**:
> The companion to the format repair: a way to ask for one field's value and get it with nothing around it — no header, no indentation, no quoting. This is the case that started the work (§1).

**Resolution**: Pending
**Notes**:

---

## Working Notes

Cycle 2 gap analysis, standalone reading only. Ten findings: five important, five minor, none critical. Cycle 1's fifteen findings are all incorporated and none is re-raised.

Considered and not raised:

- **Section order inside a document** — settled by §9.2's "sections keep their usual output order" and unobservable to a decoder; cycle 1 reached the same conclusion.
- **The stats and dep-tree summaries having no worked example** — the named-fields form is shown once in §5.2 and the field names and values are unchanged by this work, so the conversion is mechanical from the current output.
- **Whether `--fields` needs its own registry and help entry** — a builder implementing "both spellings work" registers both, and the drift-detection test §12.1 cites fails the suite if only one is registered.
- **Whether the quoted description form is always quoted or quoted only when the library needs to** — §6.2 pins the form to what the library produces from a Go string, and a decoder reads both identically.
- **`remove` producing a status cascade while staying prose** — §3.2 decides the prose boundary and §7.3 records the declined option; reopening it would be re-litigating a made decision rather than closing a gap.
- **The mutual pointer between §7.6 and §12.2 on the unchanged-terminal-children requirement** — each side answers a different reader question (what the changes table lists; what the amendment touches), and the overlap is one clause.
