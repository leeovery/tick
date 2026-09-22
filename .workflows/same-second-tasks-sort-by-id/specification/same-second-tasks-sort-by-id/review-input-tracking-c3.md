# Review Tracking: Same-Second Tasks Sort By ID - Input Review

## Findings

### 1. The new diagnostic is missing from the README's list of what `tick doctor` checks

**Source**: `investigation/same-second-tasks-sort-by-id.md` — Fix Direction, resolution 1 (the `DuplicateSeqCheck` diagnostic, reachable in this project after a branch merge) read against the investigation's documentation analysis (Testing Recommendations → "Documentation and regression floor"), which examines only the sort-contract sentence
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §6 Documentation; documents the check specified in §5.2

**Problem**:
Two branches that each numbered tasks from the same maximum merge into duplicate sequences, and the tasks in that group silently lose their authoring order relative to one another. Tick does have a way to tell the user — the new duplicate-sequence diagnostic reports the group and its line numbers — but the README's standing list of what `tick doctor` checks for names JSONL syntax errors, invalid IDs, duplicates, orphaned references, self-referential dependencies, dependency cycles, parent/child violations and cache staleness, and nothing about sequences or ordering. A user whose task list came back scrambled after a merge reads that list, sees no check that sounds like their problem, and concludes tick cannot tell them — the one diagnostic written for their situation goes unfound. The specification names a single documentation site the work must update, the sort-contract sentence, so the check ships with no user-facing mention anywhere.

**Proposal**:
Add a second documentation site to §6: the README's `tick doctor` check enumeration gains the duplicate-sequence check, named as duplicate creation sequences rather than left to the existing word "duplicates". What determined it: the sources decide the fix adds the diagnostic and treat the README as the user-facing documentation the work must keep true, and that enumeration is a standing list of every check — a new check belongs in it or the list is wrong. The record does not itself make this call; its documentation analysis was scoped to the sort contract. The alternative that also fits is relying on the existing "duplicates" to cover it — rejected because that word reads as duplicate IDs, the condition it was written for, and leaves the user nothing to search for.

**Current**:
```
`README.md:115` is the only site stating the sort contract (`rg -n 'sorted by|creation date' README.md` → one hit). It promises "sorted by priority (ascending), then creation date" and says nothing about what happens when creation dates are equal, so neither the code nor the docs commit to an answer today. It gains the tiebreak: within a priority band, tasks tied on creation date come back in creation order.
```
(The proposed text is additive — a new paragraph following this one; the paragraph above is unchanged and is quoted only to anchor the insertion point.)

**Proposed Text**:
```
`README.md:396` is the second site the work touches: it enumerates what `tick doctor` checks for — JSONL syntax errors, invalid IDs, duplicates, orphaned references, self-referential dependencies, dependency cycles, parent/child constraint violations and cache staleness. The duplicate-sequence check (§5.2) joins that enumeration, named as duplicate creation sequences rather than folded into the existing "duplicates", which reads as duplicate IDs — a user whose tasks lost their authoring order after a merge has to be able to find the diagnostic that reports it. The sentence is prose outside every fence, so no README sample renders it.
```

**Resolution**: Pending
**Notes**:

---

## Observations

- The check's user-facing display label — the name `tick doctor` prints beside each result, which for the mirrored duplicate-ID check reads as a short capability name — is unstated; mirroring settles it.
- The sources' H6 measured that the cache's `created` column is TEXT and compares lexically (`'Z'` above `'.'`); §3.2 names the new `seq` column without a type, and the same lexical trap would order sequence 10 before 2.
- The sources' cost list for the rejected sub-second route also counts 222 literal timestamps across 16 test files and the byte-compared README timestamp samples; §2.1 carries the other costs of that route.
