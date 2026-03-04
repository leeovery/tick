---
status: complete
created: 2026-03-04
cycle: 3
phase: Gap Analysis
topic: auto-cascade-parent-status
---

# Review Tracking: auto-cascade-parent-status - Gap Analysis

## Findings

### 1. Rule 5 recursive upward reopen through done ancestor chain is ambiguous

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Cascade Rules > Rule 5, Reopen Behavior

**Details**:
Rule 5 says: "When a child is reopened under a done parent, the parent reopens to open." The follow-up note says: "No reverse cascade on reopen otherwise -- reopening a child does not revert a started parent; reopening a parent does not reopen cancelled children."

Consider a three-level hierarchy where grandparent and parent were both auto-completed to `done` via Rule 3. A leaf child is reopened by the user. Rule 5 reopens the parent to `open`. Now the parent (a child of the grandparent) has been reopened, and the grandparent is `done`. Rule 5's trigger condition is met again: "a child is reopened under a done parent." Should the grandparent also reopen?

The queue-based cascade processing would naturally propagate this if the parent's reopen is enqueued as a cascade change. But the "no reverse cascade on reopen otherwise" note creates ambiguity about whether Rule 5 is intended to apply recursively up the ancestor chain or only to the direct parent.

The two "no reverse cascade" clarifications cover different cases (in_progress parent stays, cancelled children stay) and do not directly address the done-ancestor-chain scenario. An implementer could reasonably read Rule 5 as either single-level or recursive.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added recursive upward reopen clarification to Rule 5.
