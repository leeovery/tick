---
status: in-progress
created: 2026-03-26
cycle: 1
phase: Input Review
topic: dep-tree-visualization
---

# Review Tracking: dep-tree-visualization - Input Review

## Findings

### 1. Zero-dependency task omission was explicitly parked as an open question

**Source**: discussion/dep-tree-visualization.md, "What should the command structure and UX look like?" section, final paragraph
**Category**: Gap/Ambiguity
**Affects**: Command Structure (Full graph mode description)

**Details**:
The discussion explicitly parks the question of whether tasks with zero dependencies should appear in the full graph: "Leaning towards omitting them (they're not interesting in a dependency view) but feels slightly incomplete. Parking this -- will revisit after seeing it in practice." The specification states "Tasks with zero dependencies are omitted" as a firm decision without acknowledging this was deferred. Either the decision needs to be formally made, or the spec should note this as a deferred/revisitable choice.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 2. Focused view "Blocked by" and "Blocks" as labeled output sections

**Source**: discussion/dep-tree-visualization.md, "What should the command structure and UX look like?" section, Decision subsection
**Category**: Enhancement to existing topic
**Affects**: Rendering, Formatter Integration (Pretty format)

**Details**:
The discussion describes the focused view as having distinct named sections: "'Blocked by' section walks upstream... 'Blocks' section walks downstream." This implies labeled section headers in the rendered output (e.g., a "Blocked by:" header followed by the upstream tree, then a "Blocks:" header followed by the downstream tree). The specification mentions these directional labels in passing under Command Structure but doesn't specify that the pretty formatter should render them as distinct labeled sections with headers. For the pretty formatter implementation this distinction matters -- it determines whether the output has clear section breaks or is a single undifferentiated tree.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:
