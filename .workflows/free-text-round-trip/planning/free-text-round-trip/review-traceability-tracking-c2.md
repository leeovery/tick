# Review Tracking: Free Text Round Trip - Traceability

## Findings

### 1. The bound the specification puts on the `auto-cascade-parent-status` amendment is not carried into the plan

**Type**: Incomplete coverage
**Spec Reference**: §12.2 — "Both documents carry a point that is plainly and load-bearingly wrong, so both are amended… `auto-cascade-parent-status` fixes the arrow-and-`(auto)` lines as the machine-readable cascade form, which §7.2 replaces outright. **The amendment does not touch its requirement that unchanged terminal children be shown alongside a cascade (§7.6).**" Also §12.2's "Corrections are made by judgement, not as a blanket rewrite" and its account of the corrigendum route, and §7.6 ("it is not reinstated and not decided here").
**Plan Reference**: Phase 2, task `free-text-round-trip-2-6` (README transition and cascade samples match real output) — Context, final paragraph, and the task's Spec Reference line
**Move**: settled
**Change Type**: update-task

**Problem**:
The specification decides that another completed work unit's specification is amended, and it decides exactly how far that amendment reaches: the arrow-and-`(auto)` cascade form is replaced, and the requirement that unchanged terminal children be shown alongside a cascade is left standing. The plan carries the first half and not the second. The session that performs the corrigendum reads the plan's note, sees "fixes the arrow-and-`(auto)` lines as the machine-readable cascade form and is owed a correction", and has nothing telling it where to stop — so the natural move is to replace the whole passage, which deletes a live requirement of another work unit from a document that stays in the knowledge base at full confidence. That requirement is not this work's to retire: §7.6 says it is neither reinstated nor decided here. The sibling correction in Phase 1 (`tick-core`) carries the full route and the reason it is owed; this one is a single thin sentence, and the bound it is missing is the part that protects someone else's record. The task also cites §12.2 in its Context while omitting it from its Spec Reference, so an implementer chasing the claim has no pointer to the section that sets the bound.

**Proposal**:
Replace the Context's final paragraph with the same shape Phase 1 task `free-text-round-trip-1-3` already uses for the `tick-core` amendment — the reason the correction is owed, the bound on it, the route it runs through, and the statement that it is never this task's work — and add §12.2 to the task's Spec Reference. Every clause comes from §12.2 and §7.6 verbatim or near-verbatim; no new decision is taken, and the plan's own convention fixes both the placement and the wording. It stays a Context note rather than becoming task work because `task-design.md` forbids a task editing another work unit's artifact under `.workflows/` — the session handles the amendment through `correcting-historical-artifacts.md` behind its own gate.

**Current**:
```
> §12.2 notes that the `auto-cascade-parent-status` specification fixes the arrow-and-`(auto)` lines as the machine-readable cascade form and is owed a correction. That is a completed work unit's artifact: it is corrected through the session's corrigendum route, never by this task.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §7.2, §7.6, §4.1, §4.2
```

**Proposed Text**:
```
> §12.2 records that the `auto-cascade-parent-status` specification is owed a correction: it fixes the arrow-and-`(auto)` lines as the machine-readable cascade form, which §7.2 replaces outright — "plainly and load-bearingly wrong", so it is amended rather than left to supersession. Two bounds come with it. The amendment does not touch that specification's requirement that unchanged terminal children be shown alongside a cascade: §7.6 records it as "a pre-existing unimplemented requirement in another work unit's specification… not reinstated and not decided here", so it stands in its own document untouched. And "corrections are made by judgement, not as a blanket rewrite" — nothing else in that document is edited. That is a completed work unit's artifact: it is corrected through the session's corrigendum route — the amendment presented and confirmed, the wrong claim replaced in place, a dated corrigendum entry recording what the document used to claim and what is true instead, and the document re-indexed, which is the point of the route since a specification's content stays live in the knowledge base at full confidence — never by this task.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §12.2, §7.2, §7.6, §4.1, §4.2
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-2-6` (tick-477df7) — Context's final paragraph replaced with the bounded form and §12.2 added to the Spec Reference, in both the task detail file and the tick store.

---

### 2. Two of the four status commands never get a cascading document decoded

**Type**: Incomplete coverage
**Spec Reference**: §11, part 1 — "Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse… The commands are §3.1's table, and the coverage is counted in documents rather than commands: **every branch a listed command can take**, the emptied forms of §8 included". §3.1 lists `start`, `done`, `cancel` and `reopen` as the status-change commands.
**Plan Reference**: Phase 6, task `free-text-round-trip-6-3` (Mutation and status documents complete the toon inventory) — Do step 1, the first acceptance criterion, the Tests list and the matching edge case
**Move**: settled
**Change Type**: update-task

**Problem**:
The conformance inventory is meant to be the enumeration that stops a branch going undecoded — its own Problem statement says "a branch nobody thought of is a branch nobody decodes". It enumerates four `update` branches and two `create` branches, and then stops short on the status commands: `start` and `cancel` get a no-cascade document each and nothing more, on the stated ground that "`done` and `reopen` are the two that cascade naturally". They are not. `start` carries every open ancestor to `in_progress` (Rule 2) and `cancel` carries every non-terminal descendant down (Rule 4) — both fire on ordinary hierarchies, and `tick start` on a child under an open parent is about as common an invocation as the tool has. So two of the eight status documents the tool routinely emits are outside the table that exists to guarantee they parse, and the reason recorded for leaving them out is wrong, which means a later reader has no prompt to revisit it.

**Proposal**:
Give each of the four status commands both branches, which is the coverage §11 states and the standard the same task already applies to `create` and `update`. The cascade each command takes is fixed by the state machine documented in `CLAUDE.md` and the specification's §7.5 — Rule 2 up for `start`, Rule 4 down for `done` and `cancel`, Rule 5 up for `reopen` — so the seeding for each new entry is determined, not chosen. Do step 1, the first acceptance criterion, two test names and the edge case that carries the incorrect reason are updated together.

**Current**:
```
1. `internal/cli/conformance_test.go` — add the status entries: `start`, `done`, `cancel` and `reopen` each on a childless task with no cascade, plus `done` on a parent with an open child and `reopen` on a done task under a done parent, both of which cascade.
```
```
- [ ] Each of `start`, `done`, `cancel` and `reopen` has a decoded document for its no-cascade branch, and `done` and `reopen` additionally for a cascading branch
```
```
- `"it decodes a cascading done document"` — decoded `changed` carries the requested row and the child's row
- `"it decodes a cascading reopen document"` — decoded `changed` carries the requested row and the ancestor's row
```
```
- Each of `start`/`done`/`cancel`/`reopen` gets a no-cascade document; `done` and `reopen` are the two that cascade naturally, so they carry the cascading entries
```

**Proposed Text**:
```
1. `internal/cli/conformance_test.go` — add the status entries: `start`, `done`, `cancel` and `reopen` each on a childless task with no cascade, plus a cascading document for each of the four — `start` on a child under an open parent (Rule 2 carries the parent to `in_progress`), `done` on a parent with an open child (Rule 4), `cancel` on a parent with an open child (Rule 4), and `reopen` on a done task under a done parent (Rule 5).
```
```
- [ ] Each of `start`, `done`, `cancel` and `reopen` has a decoded document for its no-cascade branch and one for its cascading branch
```
```
- `"it decodes a cascading start document"` — decoded `changed` carries the requested row and the parent's row
- `"it decodes a cascading done document"` — decoded `changed` carries the requested row and the child's row
- `"it decodes a cascading cancel document"` — decoded `changed` carries the requested row and the descendant's row
- `"it decodes a cascading reopen document"` — decoded `changed` carries the requested row and the ancestor's row
```
```
- All four status commands cascade — `start` carries open ancestors to `in_progress` (Rule 2), `done` and `cancel` carry non-terminal descendants down (Rule 4), and `reopen` carries done ancestors back to open (Rule 5) — so each of the four carries both a no-cascade and a cascading entry, and the coverage is counted in branches as it is for `create` and `update`
```

**Resolution**: Pending
**Notes**:

---

## Coverage Notes (no finding)

Recorded so the gaps that were checked and found closed are visible, and so deliberate departures the plan already names are not re-raised in a later cycle.

- **Direction 1 (spec → plan)**: every numbered subsection of the specification except §1 (purpose) and §10.3 (a decision not to build an input path, honoured by absence) is named in at least one task's Spec Reference, and every bolded decision in §2.2, §3.1–§3.3, §4.1–§4.3, §5.1–§5.4, §6.1–§6.4, §7.2–§7.6, §8, §9.1–§9.8, §10.1–§10.2, §11 and §12.1–§12.2 traces to a task with matching acceptance criteria. §1's inventory of hand-assembled sections — task header, stats summary, dep-tree summary, tags, refs, description — is fixed one site per task across Phases 1 and 3, with the focused dep tree's identity line and the two prose branches carried in Phase 3.
- **§12.1's five README samples plus the `tick list` row**: `tick show` and `tick list` in Phase 1 task 6, the arrow/JSON/cascade samples in Phase 2 task 6, the dep-tree summary in Phase 3 task 6, `--field`/`--fields` in Phase 4 task 8, `--` in Phase 5 task 6. Verified against the live README: lines 397, 424, 430, 433, 437, 445, 448, 474, 502-504 and 513-517 are the only toon-shaped or arrow-shaped samples in the file, and each is allocated.
- **Phase 6's inventory partition**: the live `commandFlags` registry carries 21 keys; the plan's three declared sets (14 must-parse, 5 prose, 2 out of scope) account for all of them with none doubled, so the guard `free-text-round-trip-6-3` adds is satisfiable as written.
- **Cross-phase deferrals**: every deferral recorded in a task's edge cases has a matching criterion in the receiving phase — Phase 1 task 5's create/update decode deferral → Phase 2's single-document criterion; Phase 1 task 4's dash-leading note deferral and Phase 4 task 2's dash-value deferral → Phase 5's writability criteria; Phase 1 task 6's dep-tree sample deferral → Phase 3's README criterion; Phase 4 task 8's `--` deferral → Phase 5's README/help criterion; Phase 4 task 8's decode-coverage deferral → Phase 6's field-selection criterion.
- **§8's emptied full dep tree versus a dependency cycle**: §8 describes the emptied document with "the summary fields of §5.2 reading zero", while the plan (Phase 3 task 4, Phase 6 task 2) requires the builder's real counts on the cycle branch, where zero roots sit beside non-zero blocked/chains. The specification never identified the cycle case, and §8's own "reads the counts to learn which it got" only works if the counts are true. Settled by the record; not raised.
- **Focused dep tree always carrying both directions** (Phase 3 task 3), which changes today's omit-when-empty rule on the populated branch: derived from §8's "the document the non-empty branch produces, emptied: the same fields", which cannot hold if the populated branch's key set varies with the data. Named in the task as a change rather than left implicit; not raised.
- **§11's "no byte-level pinning is kept in the machine formats" versus the retained count-zero header comparisons** (Phase 6 tasks 1 and 6): the plan departs from the sentence and states why — `notes[0]{index,text,created}:` and `notes[0]:` decode to the same empty list, so no decoded assertion can check columns §7.4 and §8 require to exist. The retained set is enumerated. Raised and settled in cycle 1; not re-raised.
- **Phase 5 task 1's one accepted regression** (`tick create --description -- x` now reports `--description requires a value`) against §10.2's "Everything that works today works identically": surfaced as an edge case rather than hidden, and no caller could have meant the invocation. Raised and settled in cycle 1; not re-raised.
- **Spec-silent mechanisms the plan settles and labels as such** — the notes query ordering that makes §6.3's index truthful, the `changed` section's position between `notes` and `description`, the dep-tree summary fields sitting after the edge section, the filtered-JSON `map[string]any` and its key order, the `--field` and out-of-range error wording, `parseArgs` as the marker's recognition point and its rulings on a leading `--` and a `--` in a value slot, the migrate Engine as the single normalisation point, `flagScanLimit` as the shape of the `note add` exemption, `rebuild` declared with §3.2's prose commands, and the inventory-partition guard — are each how the plan builds a decided requirement, not new requirements of the product. Not traced.
- **The `CLAUDE.md` Formatter-interface line** (Phase 2 task 2, Do step 6): not product content and not in §12's list of documents owed a correction, but a one-line consequence of a deletion the specification decides, landed by the cycle 1 integrity review. Not raised.

---
