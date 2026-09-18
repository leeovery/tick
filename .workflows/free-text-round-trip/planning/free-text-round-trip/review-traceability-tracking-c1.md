# Review Tracking: Free Text Round Trip - Traceability

## Findings

### 1. The `tick-core` specification correction §12.2 requires is recorded nowhere in the plan

**Type**: Missing from plan
**Spec Reference**: §12.2 — "Two completed specifications are corrected selectively"; the `v1` / `tick-core` row of its table ("States 'long text fields get their own unstructured sections' as a principle and prints the indented description block as its worked example") and the closing paragraph: "Both documents carry a point that is plainly and load-bearingly wrong, so both are amended. `tick-core` states as a rule the exact thing §6.2 removes."
**Plan Reference**: Phase 1, task `free-text-round-trip-1-3` (Description becomes one TOON-quoted value)
**Move**: settled
**Change Type**: add-to-task

**Problem**:
Two published specifications describe output this work replaces, and the specification decided both are amended. One of them — `auto-cascade-parent-status` — is carried into the plan: Phase 2 task 6 names it in Context and routes it to the corrigendum facility. The other — `v1` / `tick-core` — appears nowhere in the plan. It states "long text fields get their own unstructured sections" as a *principle* and prints the indented description block as its worked example, which is the exact rule §6.2 deletes. Because a specification's content stays live in the knowledge base at full confidence, that superseded principle keeps being served as validated context to every later query — the precise failure §12.2's re-index exists to prevent. The session executing Phase 1 has no prompt to raise the amendment, so the work ships with one of the two owed corrections silently dropped.

**Proposal**:
Note the owed correction in the Context of `free-text-round-trip-1-3`, the task whose change (§6.2) is what shows `tick-core` wrong, and add §12.2 to its Spec Reference. This mirrors exactly how the plan already carries the sibling correction — Phase 2 task 6's Context names the `auto-cascade-parent-status` amendment and routes it to the session's corrigendum route rather than folding it into the task. The plan's own convention therefore fixes both the placement and the wording; no new decision is taken. It stays a Context note rather than becoming task work because `task-design.md` forbids a task editing another work unit's artifact under `.workflows/` — the executor writes code and tests, and the session handles the amendment through `correcting-historical-artifacts.md` behind its own gate.

**Current**:
```
> Verified against the project's toon-go version: a string carrying `\n\n`, two-space indentation, `"`, `\t` and `\r\n` encodes to one `description: "…"` line and decodes back byte-identical.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §6.2, §2.2, §2.3, §5.2
```

**Proposed Text**:
```
> Verified against the project's toon-go version: a string carrying `\n\n`, two-space indentation, `"`, `\t` and `\r\n` encodes to one `description: "…"` line and decodes back byte-identical.
>
> §12.2 records that the `v1` / `tick-core` specification is owed a correction on exactly this point: it states "long text fields get their own unstructured sections" as a principle and prints the indented description block as its worked example (`sed -n '693,714p' .workflows/v1/specification/tick-core/specification.md`) — "`tick-core` states as a rule the exact thing §6.2 removes", which is plainly and load-bearingly wrong and is therefore amended rather than left to supersession. That is a completed work unit's artifact: it is corrected through the session's corrigendum route — the amendment presented and confirmed, the wrong claim replaced in place, a dated corrigendum entry recording what the document used to claim, and the document re-indexed — never by this task.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §6.2, §2.2, §2.3, §5.2, §12.2
```

**Resolution**: Fixed
**Notes**: Applied verbatim to `free-text-round-trip-1-3` (tick-a119e7) — Context paragraph added and §12.2 appended to the Spec Reference, in both the task detail file and the tick store. Gate mode set to auto at this finding.

---

## Coverage Notes (no finding)

Recorded so the gaps that were checked and found closed are visible, and so the deliberate departures the plan already names are not re-raised in a later cycle.

- **Direction 1 (spec → plan)**: every bolded decision in §2.2, §3.1, §3.2, §3.3, §4.1, §4.2, §4.3, §5.1–§5.4, §6.1–§6.4, §7.1–§7.6, §8, §9.1–§9.8, §10.1–§10.3, §11 and §12.1 traces to at least one task with matching acceptance criteria. The three declined alternatives the specification names as out of scope (replicating §7's table shape in pretty, adding a title to pretty's transition line, a `note edit` command) appear nowhere in the plan, and §10.3's "no stdin or file input path" is honoured by absence.
- **Cross-phase deferrals**: each deferral recorded in a task's edge cases has a matching criterion in the receiving phase — Phase 1 task 5's create/update decode deferral → Phase 2's "emit a single decodable document" criterion; Phase 1 task 6's dep-tree sample deferral → Phase 3's README criterion; Phase 4 task 8's `--` deferral → Phase 5's README/help criterion; Phase 4 task 8's decode-coverage deferral → Phase 6's field-selection criterion.
- **§11's "no byte-level pinning is kept in the machine formats" versus the retained count-zero header comparisons** (Phase 6 task 6, and Phase 6 task 1's `tasks[0]{…}` assertion): the plan departs from the sentence and states why — `notes[0]{index,text,created}:` and `notes[0]:` decode to the same empty list, so no decoded assertion can check the columns that §7.4 and §8 require to be there. The departure is the only reading that satisfies both, it is named in the task rather than left implicit, and the retained set is enumerated. Settled by the record; not raised.
- **Phase 5 task 1's one accepted regression** (`tick create --description -- x` now reports `--description requires a value` where it previously stored a two-dash description) sits against §10.2's "Everything that works today works identically". The plan surfaces it as an edge case rather than hiding it, and honouring the sentence literally would require the top-level argument scanner to know each command's value-taking flags. No caller could have meant the invocation; no change proposed.
- **Spec-silent mechanisms the plan settles and labels as such** — the notes query ordering that makes §6.3's index truthful, the `changed` section's position between `notes` and `description`, the filtered-JSON `map[string]any` and its key order, the `--field` and out-of-range error wording, `parseArgs` as the marker's recognition point, the migrate Engine as the single normalisation point, `rebuild` declared with §3.2's prose commands, and the inventory-partition guard — are each how the plan builds a decided requirement, not new requirements of the product. Not traced.

---
