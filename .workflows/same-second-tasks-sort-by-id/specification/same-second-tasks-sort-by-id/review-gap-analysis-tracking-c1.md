# Review Tracking: Same-Second Tasks Sort By ID - Gap Analysis

## Findings

### 1. What "the highest sequence currently in the file" means on a file whose records arrived without one

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Critical
**Affects**: §2.2 (Assignment), with §2.3 (Backfill) as context

**Problem**:
The assignment rule reads as a single lookup — take the highest number in the file, add one — and the phrase "in the file" points at the stored bytes. That breaks on a file where some records reached it without a sequence. Those records are given numbers as the file is read, but the stored bytes still lack them. An implementer who computes the maximum from the stored values alone hands the next new task a number a just-numbered record already holds — a fresh duplicate minted by the upgrade itself, on the first task created after a merge, with two tasks then ordered by random ID and a doctor report the user cannot account for.

**Proposal**:
Say the maximum is taken over the task set as read rather than over the stored bytes. The record already determines it: backfill is placed in the single read funnel every mutation passes through (§2.3), so by the time a new task is numbered every record in hand already carries a sequence. Only the word "file" leaves the other reading open.

**Proposed Text**:
Append to the first paragraph of §2.2, after "so an absent or zero value on a record means it carries no sequence.":

"The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too."

**Resolution**: Approved
**Notes**: Narrowed at dispose. The finding's other half — that a single write creating many tasks would share one number — rests on a premise measured false: the migration framework opens its own `store.Mutate` per task (`internal/migrate/store_creator.go:36`), and every `Mutate` call site in the tree (`create.go:187`, `dep.go:78,156`, `remove.go:184`, `update.go:275`, `transition.go:36`, `note.go:68,124`, `store_creator.go:36`) creates at most one task. With no multi-task write path in existence, a rule for one is the builder's to settle if such a path is ever added. The same half was raised and declined in input review (finding 2). The surviving half applied to §2.2 as staged — the derivation is the record's own, §2.3 already placing backfill in the single read funnel every write passes through. Narrowed at dispose. The finding's other half — that a single write creating many tasks would share one number — rests on a premise measured false: the migration framework opens its own `store.Mutate` per task (`internal/migrate/store_creator.go:36`), and every `Mutate` call site in the tree (`create.go:187`, `dep.go:78,156`, `remove.go:184`, `update.go:275`, `transition.go:36`, `note.go:68,124`, `store_creator.go:36`) creates at most one task. With no multi-task write path in existence, a rule for one is the builder's to settle if such a path is ever added. The same half was raised and declined in input review (finding 2).

---

### 2. Records carrying no sequence must not be reported as duplicates of one another

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §5.2 (A duplicate-sequence doctor check)

**Problem**:
Every task in a project created before this change has no sequence, and a zero written by any other hand means the same thing. The doctor check reads the stored records so it can report line numbers, so it sees exactly those unnumbered records — and nothing says they are excluded from the comparison. Group them and a user upgrading an existing project runs `tick doctor` and gets a duplicate report naming every task in the file, flagged as needing a manual fix, on a file that is entirely healthy: the very next read gives each of those records its own distinct number. The first impression of the feature is a diagnostic screaming about a problem that does not exist.

**Proposal**:
Say that the check compares only records that carry a sequence. §2.2 already decides that absent or zero means the record has no sequence, and §2.3 already decides such records are each given a distinct number on read, so there is nothing for them to collide over; the check's silence is the only thing between that decision and a false report.

**Proposed Text**:
Append to the first paragraph of §5.2, after "returns a single passing result on a clean file.":

"Records carrying no sequence — absent or zero (§2.2) — are not compared with one another: backfill gives each of them a distinct number on read (§2.3), so they cannot collide. A project that predates the field reports clean."

**Resolution**: Approved
**Notes**: Applied to §5.2 as staged. The derivation is the record's own on two counts: §2.3 gives every unnumbered record its own number on read, and `DuplicateIdCheck` — the check §5.2 mirrors — already skips records with no ID (`internal/doctor/duplicate_id.go:24`, "missing/empty IDs are silently skipped").

---

### 3. Whether a duplicate sequence fails `tick doctor`, and what the user is told to do about it

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §5.2 (A duplicate-sequence doctor check), §5 (Duplicate Sequences)

**Problem**:
The check's verdict is left open on the two points a user experiences. First, does the duplicate make `tick doctor` come back failing, or come back passing with a note? The specification's own premise is that duplicates arrive through routine branch merges in this project, and tick offers no command that removes one — sequences are never renumbered. If the answer is "failing", every merge leaves the project permanently red to any script, agent or CI gate reading doctor's outcome, clearable only by hand-editing `tasks.jsonl`; if the answer is "passing with a note", a scripted consumer never learns that two tasks lost their authoring order. The implementer picks one of those user experiences blind.

Second, a failing check owes the user an action. Nothing states what a person should do when the report names a duplicate group, and the plausible guesses are actively misleading — rebuilding the cache, for instance, changes nothing, because the duplicate lives in the file.

**Proposal**:
Report the duplicate as a warning rather than an error, and say in the report what it costs and what would undo it. Tick's diagnostics already carry the distinction and define it: `sed -n '13,17p' internal/doctor/doctor.go` → `SeverityError` is "a failure that breaks tick and affects exit code", `SeverityWarning` "a suspicious but allowed state that does not affect exit code". A duplicate sequence breaks nothing — §5.1 keeps the order total, the group simply falls back to ID order among themselves — so it is the second of those by the codebase's own definition. The precedent matches: `grep -rn SeverityWarning internal/doctor` → the single existing user is `ParentDoneWithOpenChildrenCheck` (`parent_done_open_children.go:59`), a state tick permits and flags; the other nine checks are errors. The alternative that also fits is failing the run, on the strength of §5.2 mirroring the duplicate-identity check — rejected because that check's duplicates break task identity and partial-ID resolution outright, where this one costs bounded ordering information, and because a merge leaves no command that could clear a failing verdict.

**Proposed Text**:
Append to §5.2, after the first paragraph:

"The report does not fail the run — a duplicate does not break anything, since order stays total (§5.1), and the condition arrives through a merge that tick offers no command to undo, so a failing outcome would be a state the user could clear only by hand. What the report tells the user is which tasks share a number, on which lines, and that those tasks have lost their authoring order relative to one another and fall back to ID order among themselves; restoring the order means editing the sequences in `tasks.jsonl` so they differ."

**Resolution**: Routed
**Notes**: The call is this session's, derived from the tree rather than from the sources, so it landed first in the investigation (Fix Direction → resolution 1), which now records the warning severity, the `SeverityError`/`SeverityWarning` definitions it rests on, the `ParentDoneWithOpenChildrenCheck` precedent, and the failing alternative it was preferred over. §5.2 is re-aligned to it.

---

### 4. The guarantee promises imports read back in import order; the sort rule gives them historical order

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §1.3 (What tick guarantees after the fix), colliding with §4.2 (Creation date stays above the sequence)

**Problem**:
The guarantee states that a batch written by an importer reads back in the sequence it was written. The sort rule states the opposite for the import case it names: recorded creation date outranks the sequence precisely so that an import carrying real historical timestamps keeps its true chronology and import order does not override it. Both readings are on the page, and they produce different products — one where a migrated project lists in the order the importer happened to walk its source, one where it lists in the order the tasks were originally created elsewhere.

The cost lands twice. The documented promise (§6) is written from the guarantee, so the user is told the wrong thing about the one command that creates hundreds of tasks at once. And the implementer building the import fixture required by §8.1 has no way to know which behaviour the assertion should pin.

**Proposal**:
Keep the sort rule and qualify the guarantee to match it: a batch that records one creation instant — an agent, a shell loop, an import whose source carries no timestamps — reads back in write order, while an import supplying real historical times is ordered by those times. This is the reading §4.2 argues for explicitly and trades against, and the one §8.1's fixture note ("its timestamps are identical") already assumes.

**Current**:
"A batch written one after another by an agent, an importer or a shell loop reads back in the sequence it was written."

**Proposed Text**:
"A batch written one after another by an agent or a shell loop reads back in the sequence it was written, as does an import whose source carries no creation times of its own. An import that supplies real historical creation times is ordered by those times instead: recorded chronology outranks the sequence (§4.2)."

**Resolution**: Approved
**Notes**: Applied to §1.3 as staged. The derivation is the record's own — §4.2 decides import chronology explicitly and trades against it, so the guarantee was the loose statement of the two.

---

## Observations

- §2.2 defines absent and zero as "carries no sequence" but says nothing about a negative value in a hand-edited record; it would sort ahead of everything and survive backfill untouched.
- Running-maximum backfill can itself mint a duplicate when an unnumbered record precedes a record whose stored sequence is at or below the running maximum plus one; §5.1's fallback and §5.2's check absorb it, so no rule is missing.
- §6 scopes documentation to the list-family sort contract, leaving `tick show`'s new children and blocker ordering undocumented in the README.
- §4.4 names the README passage documenting positional `--field` addressing without saying whether it changes; the meaning of "position N" is unchanged, so only a rendered sample whose output shifts needs re-rendering.
- On a merged file, `tick dep tree` stays on file order while the list family moves to sequence order, so the two can disagree; both positions are stated (§7.1, §7.2).
- The direction of the new sort terms is not stated; ascending follows from the creation-order guarantee.
