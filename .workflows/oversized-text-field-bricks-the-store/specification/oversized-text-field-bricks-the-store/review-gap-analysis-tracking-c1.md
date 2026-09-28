# Review Tracking: oversized-text-field-bricks-the-store - Gap Analysis

## Findings

### 1. The overview demands every refusal carry elements the title and note refusals are decided not to carry

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: 1.4 Who meets these behaviours (colliding with 7. Unchanged by this work, and 6.3 The refusal)

**Problem**:
The overview says every write-time refusal must name the field, the limit and that nothing was saved. The "unchanged" list says the opposite for titles and notes: their refusals keep their current messages, which name the field and limit but not the submitted length or that nothing was saved, and aligning them is out of this work. A builder who follows the overview rewrites existing title and note error messages that agents and tests already see, which the specification says must not change. The same sentence also governs the title refusal that migrate newly adopts, so that skip reason could go either way. It has also drifted from the description refusal's own definition, which requires a fourth element, the submitted length, that the overview leaves out. A builder working from the overview would ship a description refusal that doesn't tell the agent how much to cut.

**Proposal**:
Scope the overview sentence to the description cap's refusal and point to the refusal section for its contents, without listing the elements a second time. The record settles this: the "unchanged" list explicitly keeps title and note messages as they are, and the refusal section is where the description refusal's elements are defined.

**Current**:
Agents write the tasks, not people, so an agent is what meets a write-time refusal. Every refusal must say which field, what limit, and that nothing was saved.

**Proposed Text**:
Agents write the tasks, not people, so an agent is what meets a write-time refusal. The description cap's refusal therefore carries everything an agent needs to cut the description and retry the write (§6.3).

**Resolution**: Pending
**Notes**:

---

### 2. The length the description refusal reports is not tied to the length the cap counts

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: 6.3 The refusal

**Problem**:
The cap counts a description's characters after leading and trailing whitespace is trimmed. The refusal must report "the length submitted" so the agent knows how much to cut. Read literally, that is the raw input length, whitespace included. An agent whose description arrives with surrounding whitespace, such as a trailing newline or indented heredoc padding, would be told a number larger than the one the product compared against the limit. It would then cut more real content than needed, and the reported number would disagree with the limit it sits beside. The boundary tests in the testing section would pin whichever reading the builder happens to pick.

**Proposal**:
Report the counted length: the description's characters after trimming, the same measure the cap is checked against. Two decisions already in the specification settle this. The counting rule says whitespace is trimmed before counting, and the refusal's purpose is to tell the agent how much to cut. Reporting the raw input length also fits the phrase "length submitted", but nobody who knows the cap ignores surrounding whitespace would want it, because its only effect is to overstate how much must go.

**Proposed Text**:
(Add after the example sentence in §6.3.) The length reported is the description's counted length, its characters after trimming (§6.1), so it compares directly with the limit.

**Resolution**: Pending
**Notes**:

---

## Observations

- Doctor's per-line load failure names the line and the loader's reason but not the task ID that store read errors carry. The line number is enough for the hand fix doctor suggests.
- The incomplete-read failure in §5.3 has no stated suggestion. The path is reachable only through a test seam, so its wording is the implementer's.
- §3's "every error from reading tasks.jsonl names the line" reads more broadly than intended for failures that happen before any line is read, such as a file that cannot be read. Only parse failures have a line to name.
- On migrate, the refusal's "nothing was saved" element refers to the one skipped issue while the rest of the import proceeds.
- §5.4 opens with "Partial" without an antecedent in that subsection. It refers to the incomplete read of §5.3.
