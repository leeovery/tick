# Review Tracking: Same-Second Tasks Sort By ID - Gap Analysis

## Findings

### 1. The assertions that pin the sequence can pass over a sequence that does not work

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Critical
**Affects**: §8 preamble (the constraints on every ordering fixture), governing §8.2 (The sequence itself) and §8.3 (Duplicate sequences)

**Problem**:
A recorded creation date outranks the sequence, and the sequence speaks only where two dates tie. Every ordering fixture in the verification list is therefore ordered by its timestamps unless those timestamps are made to tie — and the fixtures that exist to prove the sequence works are not told to tie them.

The two that matter most are silently self-proving as written. The backfill fixture — a file carrying no sequence at all, the case that carries every project created before this change — is required to come back in line position; give its records the ascending seconds a test author naturally writes and the date alone produces that order, so the assertion passes over a backfill that handed every record the same number, or none at all. The duplicate-sequence fixture is required to demonstrate the ID tiebreak; give its two tasks different seconds and the date decides before the tiebreak is ever reached, so the assertion never exercises the thing it names.

What ships is the defect this work exists to remove, with a green suite over it: a user upgrading an existing project runs `tick list` and gets tasks authored in one second back in random ID order, and nothing in the suite ever said otherwise. The preamble already names this exact trap for task IDs — an assertion that cannot fail proves nothing — and stops one term short of the timestamps.

**Proposal**:
Add the timestamp constraint alongside the two already there: wherever the sequence or the final ID term is what the fixture is testing, its records carry one creation second, so the result can only have come from the sequence. The record determines it — §4.2 puts `created` above `seq`, so any fixture with distinct seconds never reaches the terms under test — and the preamble's own first constraint is the same argument applied to the other tiebreak key.

**Current**:
Two constraints therefore apply to every ordering fixture below.

- **IDs must contradict the authoring order.** An ascending-ID result and a creation-order result must be distinguishable, or the assertion proves nothing.
- **Fixture size must not be load-bearing.** The query plan flips on data shape alone (§1.1), so a test keyed to "this command returns this order at this size" would be asserting a coincidence of its own fixture. Assertions state the required order, never the plan.

**Proposed Text**:
Three constraints therefore apply to every ordering fixture below.

- **IDs must contradict the authoring order.** An ascending-ID result and a creation-order result must be distinguishable, or the assertion proves nothing.
- **Creation seconds must tie wherever the sequence or the ID term is what is under test.** A recorded creation date outranks the sequence (§4.2), so a fixture whose records carry distinct seconds returns the expected order whether the sequence works or not — the backfill and duplicate-sequence assertions (§8.2, §8.3) would pass over a backfill that gave every record the same number, and over a duplicate group the ID term never reached. Records in those fixtures record one creation second.
- **Fixture size must not be load-bearing.** The query plan flips on data shape alone (§1.1), so a test keyed to "this command returns this order at this size" would be asserting a coincidence of its own fixture. Assertions state the required order, never the plan.

**Resolution**: Pending
**Notes**:

---

### 2. The decision section says the sequence alone carries authoring order; the sort puts the creation date above it

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §2.1 (The decision), colliding with §4.2 (Creation date stays above the sequence) and §1.3 (What tick guarantees after the fix)

**Problem**:
The section that states the decision tells the reader the sequence is the only thing tick relies on for authoring order. The sort rule says a recorded creation date outranks the sequence: two tasks whose dates differ are ordered by those dates and the sequence is never consulted, and the guarantee section says the same. Both readings are on the page.

Taken at its word, the decision section licenses a sort built on the sequence alone. That hands a migrated project its tasks in whatever order the importer happened to walk its source rather than in the chronology the source recorded — the trade §4.2 makes deliberately, reversed by a reader who planned from §2.1. It also tells whoever writes the user-facing promise that timestamps no longer bear on ordering, which promises away the misordering a backwards clock step still produces.

**Proposal**:
Qualify the claim to what the section is actually arguing: the sequence is the only explicit statement of authoring order tick stores — an observation like a timestamp is not one — and it decides where two creation dates tie. The derivation is the record's own: §4.2 argues the created-above-sequence ranking explicitly and trades against it, and §1.3 already states the ranking in the guarantee, so §2.1 is the loose statement of the three.

**Current**:
"The list-family sort ends on it, and it is the only thing tick relies on for authoring order."

**Proposed Text**:
"The list-family sort ends on it, and it is the only explicit statement of authoring order tick stores — a recorded creation date still outranks it, and the sequence decides where two dates tie (§4.2)."

**Resolution**: Pending
**Notes**:

---

### 3. The README's list of what `tick doctor` checks gains an entry that nothing holds in place

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §8.6 (The documented contract), pinning the second documentation site decided in §6

**Problem**:
Two README passages change. The sort-contract sentence gets a prose assertion of its own, on the reasoning that nothing in the suite reads it and the sample run never will. The doctor enumeration — which gains "duplicate creation sequences" as a named entry precisely so a user whose tasks lost their authoring order after a merge can find the diagnostic that reports it — is described in the same terms, prose outside every fence that no sample renders, and then the verification list is silent on it.

The implementer builds what the verification list asks for: one assertion. The named entry then sits in the README with nothing tying it to the check that exists, and the first rewrite of that paragraph that folds it back into the generic "duplicates" costs the user the only route to the diagnostic — with a green suite either way. The asymmetry is not a decision the reader can find on the page; it reads as an omission left by adding the second site.

**Proposal**:
Pin the doctor enumeration the same way as the sort sentence. §6 already decides the entry is named separately rather than folded in, and gives the user-facing reason; §8.6's own rationale — prose the sample run never reads, so it needs an assertion of its own — applies unchanged to the second site, and the suite's existing prose assertions are drift guards of exactly this shape. The alternative that also fits is leaving it unpinned on the grounds that a list of check names is not a behavioural contract; it was not preferred because the entry's whole purpose is discoverability, which is what silent drift removes.

**Current**:
- The README's sort-contract sentence is pinned by a prose assertion, following the two the suite already carries for README prose — `TestREADMEDocumentsFieldSelection` (`internal/cli/readme_samples_test.go:570`) and `TestREADMEDocumentsEndOfFlagsMarker` (`:637`). It is the only user-facing statement of the guarantee this work delivers, and the sample run never reads it (§6).

**Proposed Text**:
- The README's sort-contract sentence is pinned by a prose assertion, following the two the suite already carries for README prose — `TestREADMEDocumentsFieldSelection` (`internal/cli/readme_samples_test.go:570`) and `TestREADMEDocumentsEndOfFlagsMarker` (`:637`). It is the only user-facing statement of the guarantee this work delivers, and the sample run never reads it (§6).
- The README's doctor enumeration is pinned the same way: a prose assertion requires it to name the duplicate-sequence check as its own entry (§6). It is the only place a user meets the new diagnostic, it is prose the sample run never reads, and folding it back into the generic "duplicates" would otherwise pass unnoticed.

**Resolution**: Pending
**Notes**:

---

## Observations

- §4.3's children clause closes "so a duplicate sequence cannot leave the list plan-dependent"; the clause under discussion is `tick show`'s children, not the list family.
- §1.2 reports a measured `tick ready --parent P --count 1` returning step-4 and then states no stored batch is currently being read in the wrong order; whether the measured project was a constructed fixture is not said, and §7.3's release posture rests on the second statement.
- §5.2 tells the user to edit the sequences in `tasks.jsonl` so they differ, without noting that the replacement must also not match a number already in the file.
- The check's own name as `tick doctor` prints it is unstated; §6 fixes only how the README names it.
