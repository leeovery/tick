---
status: in-progress
created: 2026-03-04
cycle: 1
phase: Input Review
topic: auto-cascade-parent-status
---

# Review Tracking: auto-cascade-parent-status - Input Review

## Findings

### 1. Child reparented away -- no cascade reversal

**Source**: .workflows/discussion/auto-cascade-parent-status.md, "Edge cases with multiple children and partial completion" section
**Category**: Enhancement to existing topic
**Affects**: Cascade Rules (Reopen Behavior subsection, near Rule 5 note)

**Details**:
The discussion explicitly decided: "If a child is moved to a different parent (or made parentless), do not reverse the original parent's cascade. Same principle as rule 5 -- keep rules simple, the parent's state is its own now." This edge case was discussed and decided but not captured in the specification. It is relevant because reparenting is a supported operation in Tick (updating the parent field), and without this explicit statement, implementers might wonder whether reparenting should trigger cascade re-evaluation on the old parent.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added reparenting note to Reopen Behavior section.

---

### 2. Downward cascades ignore dependency state on target tasks

**Source**: .workflows/discussion/auto-cascade-parent-status.md, "Final hole check" section (Hole 3)
**Category**: Enhancement to existing topic
**Affects**: Cascade Rules (Downward Cascades subsection, Rule 4) or Core Concept

**Details**:
The discussion's Hole 3 analysis explicitly addresses: "if task A is blocked-by task B, and a downward cascade tries to mark A as done -- should that work?" The answer is yes, because dependencies are advisory and don't gate transitions. The spec's Core Concept section says "Dependencies remain advisory -- they affect queries (ready/blocked) not transitions. Cascades follow the same principle." This covers the general principle, but the specific interaction (downward cascade completing/cancelling a task that has unresolved dependencies) was explicitly analyzed in the discussion and could be stated more directly in Rule 4 to prevent implementation ambiguity.

**Proposed Addition**:

**Resolution**: Pending
**Notes**: The general principle is present in Core Concept. This finding is about whether the specific cascade-dependency interaction warrants explicit mention in Rule 4 to avoid implementer doubt.
