---
status: complete
created: 2026-03-27
cycle: 1
phase: Gap Analysis
topic: dep-tree-visualization
---

# Review Tracking: dep-tree-visualization - Gap Analysis

## Findings

### 1. New Formatter method signature and input data type undefined

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Formatter Integration

**Details**:
The spec says "New method on the Formatter interface" but does not name the method or define its parameter type.

**Proposed Addition**: Added method name `FormatDepTree` to spec. Struct details left to planning phase.

**Resolution**: Approved
**Notes**: Method name added. Input struct shape is a planning/implementation concern — the spec defines the data requirements (modes, fields) and the planner designs the struct.

---

### 2. Full graph mode rendering detail insufficient

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure, Rendering

**Details**:
The full graph mode says "Each root task lists what it blocks" but does not define what a "root task" is or how the tree is structured.

**Proposed Addition**: Defined "root task" as a task that blocks others but has empty BlockedBy. Specified roots at top level with blocked tasks indented beneath, recursively.

**Resolution**: Approved
**Notes**: Added to Command Structure section.

---

### 3. Summary line specifics undefined

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
Summary line terms were ambiguous.

**Proposed Addition**: Defined format as `{N} chains, longest: {M}, {B} blocked` with chain = connected component, longest = edges, blocked = tasks with at least one BlockedBy entry. Summary line is full graph mode only.

**Resolution**: Approved
**Notes**: Added inline to full graph mode description.

---

### 4. Command alias -- `dep` vs `dependency`

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure

**Details**:
The spec used "tick dependency tree" but the codebase registers the command as `dep`.

**Proposed Addition**: Corrected all references from `tick dependency tree` to `tick dep tree`.

**Resolution**: Approved
**Notes**: Verified via codebase — help.go registers as "dep", dep.go uses handleDep.

---

### 5. Toon format output shape needs more definition

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Formatter Integration

**Details**:
Toon format for focused mode was unspecified.

**Proposed Addition**: Specified separate `blocked_by[N]{from,to}:` and `blocks[N]{from,to}:` sections for focused mode.

**Resolution**: Approved
**Notes**: Added to Formatter Integration section.

---

### 6. JSON format explicitly deferred

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Formatter Integration

**Details**:
JSON shape explicitly deferred to implementation.

**Proposed Addition**: N/A

**Resolution**: Skipped
**Notes**: Intentionally deferred in the discussion. The spec reflects that decision accurately.

---

### 7. Data access pattern not specified -- Query vs Mutate, SQL vs in-memory

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure (implied implementation)

**Details**:
Spec didn't explicitly state read-only via Store.Query().

**Proposed Addition**: Added "read-only operation via Store.Query()" to Overview.

**Resolution**: Approved
**Notes**: Brief clarification added to Overview section.

---

### 8. No flags specified for the tree subcommand

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure

**Details**:
No flags mentioned for the tree subcommand.

**Proposed Addition**: Added explicit "No command-specific flags. Only global flags (format, quiet, verbose) apply."

**Resolution**: Approved
**Notes**: Added to Command Structure section.

---

### 9. Title truncation lacks specifics

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
Terminal width detection, truncation character, and fallback width not specified.

**Proposed Addition**: N/A

**Resolution**: Skipped
**Notes**: Implementation detail. The spec defines the intent (truncate to fit). Truncation character, default width, and detection mechanism are implementation concerns the planner can address.

---

### 10. Focused view with asymmetric results not addressed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Edge Cases

**Details**:
Asymmetric focused view (one direction has results, the other doesn't) not addressed.

**Proposed Addition**: Added edge case: only show sections that have content, omit empty sections.

**Resolution**: Approved
**Notes**: Added to Edge Cases section.

---

### 11. Pretty format status rendering not specified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
Exact format string for each tree line not specified.

**Proposed Addition**: N/A

**Resolution**: Skipped
**Notes**: Implementation detail. The spec defines the data per line (ID + title + status). Exact format string, bracket style, and ordering are implementation concerns.
