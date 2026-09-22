# Review Tracking: Same-Second Tasks Sort By ID - Input Review

## Findings

### 1. A batch created in one mutation must advance the sequence per task

**Source**: `.workflows/same-second-tasks-sort-by-id/investigation/same-second-tasks-sort-by-id.md` — Fix Direction → Chosen Approach ("the next number above the highest currently in the file, assigned at creation"); Testing Recommendations → Ordering ("An in-process batch, not just separate invocations — the migration framework is the natural fixture"); Blast Radius ("Anything importing through `internal/migrate/` — the whole import shares one second")
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §2.2 Assignment

**Problem**:
The rule for what number a new task gets is written against the task set as it was read at the start of the write. An import, or any pass that authors several tasks in one go, creates every one of its tasks against that same set — so every task in the batch takes the same number. The scrambled ordering the work exists to remove survives untouched for exactly the writer it was reported against: a migrated project, or an agent authoring a whole phase, reads back in random task-ID order. `tick doctor` then reports the entire import as one duplicate-sequence group, telling the user their authoring order is lost — correctly, and with no way to fix it short of hand-editing every record.

**Proposal**:
State that the running maximum advances with each task assigned, not once per write, so a batch numbers 1, 2, 3 … rather than landing on one number. This is determined by the record rather than chosen: duplicates are described as reachable only through a branch merge (§5), and the verification requirements demand that an in-process batch — the migration framework, whose timestamps are identical — comes back in authoring order. Neither holds unless the batch numbers upward within a single write.

**Current**:
A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence. The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too.

**Proposed Text**:
A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence. The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too.

Where a single write creates several tasks — an import, or any batch authored in one pass — the maximum advances with each assignment rather than being taken once for the write. The first task of the batch takes the next number above the file's highest, the second takes the number above that, and so on. Taking one number for the whole batch would give every member of an import the same sequence and drop it straight back to ID order, which is the case this work exists to fix.

**Resolution**: Declined
**Notes**: Third raise of this premise, and it measures false each time. No path in tick creates more than one task in a single write: the migration framework opens its own `store.Mutate` per task (`internal/migrate/store_creator.go:36`), and every `Mutate` call site in the tree (`create.go:187`, `dep.go:78,156`, `remove.go:184`, `update.go:275`, `transition.go:36`, `note.go:68,124`, `store_creator.go:36`) creates at most one. The investigation's "the whole import shares one second" is about the timestamp, not the write, and its "in-process batch" fixture is a sequence of separate mutations. With no multi-task write path in existence, a rule for one is the builder's to settle if such a path is ever added; writing it in to quiet a recurring finding would be the review adding scope no source asked for. Previously declined in input review c1 (finding 2) and narrowed out of gap analysis c1 (finding 1).

---

### 2. Import chronology is preserved only to whole-second resolution

**Source**: `.workflows/same-second-tasks-sort-by-id/investigation/same-second-tasks-sort-by-id.md` — Analysis → H4 ("When the provider *does* supply a time (`beads.go:119` parses RFC3339, which may carry a fraction), `FormatTimestamp` flattens it to the second on write, so imported sub-second precision is discarded too"); Contributing Factors ("Sub-second precision is discarded even when a source has it")
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §1.3 What tick guarantees after the fix, §4.2 Creation date stays above the sequence

**Problem**:
The promise made to someone importing a project is that real historical creation times win over import order. That is only true down to the second. Tick flattens whatever precision the source carried, so tasks the source recorded a fraction of a second apart arrive sharing one timestamp and are ordered by the position the importer happened to visit them in — which for a provider that iterates by ID is not chronology at all. Stated unqualified, the promise is one the product does not keep, and it is the sentence a reader would carry into the user-facing documentation and into a test fixture built with sub-second source times.

**Proposal**:
Qualify the guarantee at whole-second resolution in both places it is stated. The source settles the behaviour: the provider parses RFC3339 times that may carry a fraction, and the write path flattens them, which is unchanged by this work (§7.1). No new behaviour is being chosen — only the promise being brought in line with what ships.

**Current**:
(§1.3) An import that supplies real historical creation times is ordered by those times instead: recorded chronology outranks the sequence (§4.2).

(§4.2) This cuts both ways and the trade is deliberate. Keeping `created` above preserves true chronology on imports, where the provider supplies real historical timestamps (`internal/migrate/beads/beads.go:119` parses RFC3339) and import order would otherwise override them; it also confines duplicate-sequence damage (§5) to within a single second.

**Proposed Text**:
(§1.3) An import that supplies real historical creation times is ordered by those times instead, to the second: recorded chronology outranks the sequence (§4.2).

(§4.2) This cuts both ways and the trade is deliberate. Keeping `created` above preserves true chronology on imports, where the provider supplies real historical timestamps (`internal/migrate/beads/beads.go:119` parses RFC3339) and import order would otherwise override them; it also confines duplicate-sequence damage (§5) to within a single second. The chronology carried across is whole-second: a fraction in the source is flattened on write as it is today, so imported tasks that share a second tie on `created` and fall to the sequence — within one second the import's own order decides, not the source's.

**Resolution**: Adjusted
**Notes**: Applied to §1.3 and §4.2; the derivation is the record's own — H4 states the flattening directly. §1.3's wording adjusted from "to the second" to "to whole-second resolution", which reads as the guarantee rather than as an aside; §4.2's addition landed as staged.

---

## Observations

- The investigation names the new diagnostic `DuplicateSeqCheck`; §5.2 describes it without naming it.
- §2.2 states numbering begins at 1 but not why the zero value is reserved — the investigation records that under Go's `omitempty` a zero sequence vanishes from the record, so a 0-based first task would be re-backfilled to a different number on every read until a write froze it wrong.
- The investigation flags blocker declaration order as the one place the fix reaches past the reported symptom and notes a specification could reasonably split it out; §3.3/§4.3 keep it in scope without recording that.
- The investigation's overall severity rating (Medium) is not carried into §1.2; §7.3's release posture carries the consequence.
