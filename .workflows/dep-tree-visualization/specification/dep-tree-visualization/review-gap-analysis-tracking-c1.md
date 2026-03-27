---
status: in-progress
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
The spec says "New method on the Formatter interface" but does not name the method or define its parameter type. Every existing formatter method has a well-defined input struct (e.g., `TaskDetail`, `CascadeResult`, `RemovalResult`, `Stats`). An implementer needs to know:

- The method name (e.g., `FormatDepTree`)
- The input data type and its fields -- specifically, the full graph mode and focused mode have quite different data shapes (root-with-blockers vs bidirectional upstream/downstream trees). Is this one method with a union type, or two methods? What fields does the struct carry?
- Whether the focused mode's two sections (blocked-by tree, blocks tree) come as separate data or a single struct with both directions

Without this, an implementer must make significant design decisions about the interface contract.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 2. Full graph mode rendering detail insufficient

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure, Rendering

**Details**:
The full graph mode says "Each root task lists what it blocks" but does not define:

- What constitutes a "root task" -- is it a task that is not blocked by anything (has empty BlockedBy), or a task that blocks other tasks but isn't blocked itself?
- How the tree is structured -- are roots listed at the top level, with the tasks they block indented beneath them? Is it one tree per root, or one merged graph?
- What the "summary line" format looks like -- "chain count, longest chain, blocked task count" is named but not defined. What exactly is a "chain"? Is it a connected component, a maximal path, or something else?
- Ordering of roots and children within the tree

An implementer would need to make multiple UX decisions here.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 3. Summary line specifics undefined

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
The spec mentions a "summary line" with "chain count, longest chain, blocked task count" for the full graph mode but:

- "Chain count" is ambiguous -- does it mean the number of distinct dependency chains (connected components)? The number of root-to-leaf paths?
- "Longest chain" -- length measured in edges or nodes? Example format?
- "Blocked task count" -- tasks with at least one BlockedBy entry? Or tasks whose blockers are all non-terminal (actually blocked right now)?
- The summary line is only mentioned for full graph mode, not focused mode. Is that intentional?
- No example format string provided (e.g., "3 chains, longest: 4, 7 blocked tasks")

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 4. Command alias -- `dep` vs `dependency`

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure

**Details**:
The spec uses "tick dependency tree" but the existing codebase and help text register the command as `dep` (see `help.go` line 151: `Name: "dep"`, and `dep.go`'s `handleDep` function). The discussion file also uses "tick dependency" in places. The spec should clarify the actual command name to match the existing codebase: is the subcommand `tick dep tree` (matching the existing `tick dep add/remove`) or `tick dependency tree`?

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 5. Toon format output shape needs more definition

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Formatter Integration

**Details**:
The toon format is described as "Flat edge list in standard toon format -- `dep_tree[N]{from,to}:` with one edge per line." This is clear for the full graph mode, but for the focused mode:

- Are there separate sections for upstream and downstream edges (e.g., `blocked_by_tree[N]{from,to}:` and `blocks_tree[N]{from,to}:`)? Or one combined `dep_tree` section?
- Should the target task itself be included as a node somehow?
- The "No dependencies" edge case -- what does the toon output look like? An empty section like `dep_tree[0]{from,to}:`? Or a message?

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 6. JSON format explicitly deferred

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Formatter Integration

**Details**:
The spec says for JSON: "Structured graph -- nodes array + edges array, or nested object mirroring the tree structure. Exact shape determined during implementation." This is explicitly deferred, which means an implementer has to make the design decision. While this may be intentional flexibility, it's inconsistent with how other formatter methods are specified (all have well-defined output shapes). Two possible shapes are mentioned but no guidance on which to prefer or when.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 7. Data access pattern not specified -- Query vs Mutate, SQL vs in-memory

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure (implied implementation)

**Details**:
The spec says "no data model changes" but doesn't address how the dependency graph is actually walked. The existing `Store.Query()` provides a `*sql.DB` for SQL queries, and the `dependencies` table has `(task_id, blocked_by)` pairs. For the tree command, the implementer needs to:

- Walk transitive dependencies in both directions (upstream: recursively follow BlockedBy; downstream: find all tasks where BlockedBy contains this ID, recursively)
- Decide whether to use recursive CTEs in SQL or load all tasks and walk in Go

This is arguably an implementation detail, but the spec's silence on whether this is a read-only query operation (vs Mutate) should be explicit since it affects the handler signature pattern.

**Proposed Addition**:

**Resolution**: Pending
**Notes**: This is borderline -- could be considered pure implementation detail. However, the spec says "no data model changes" without confirming "read-only via Store.Query()" which would help an implementer understand the scope quickly.

---

### 8. No flags specified for the tree subcommand

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Command Structure

**Details**:
The spec does not mention any flags for the `tree` subcommand. The existing `commandFlags` registry (flags.go) requires every command to be registered, including subcommands like `dep add` and `dep remove` (both registered with empty flag sets). The spec should explicitly state that `dep tree` accepts no command-specific flags (just the global format/quiet/verbose flags), or list any flags it does accept. This is needed for:

- The `commandFlags` registry entry
- The help registry entry (help.go)
- Flag validation

**Proposed Addition**:

**Resolution**: Pending
**Notes**: Even if the answer is "no flags", stating it explicitly prevents an implementer from wondering if they missed something.

---

### 9. Title truncation lacks specifics

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
The spec says "Truncate titles to fit available width after accounting for indentation + ID + status" but:

- How is terminal width detected? The existing codebase has no terminal width detection mechanism (the PrettyFormatter doesn't truncate anything currently). This would be new infrastructure.
- What truncation character/indicator is used? Ellipsis (`...`)? Unicode ellipsis?
- What is the minimum title length before truncation kicks in (i.e., if the available space is very small, show at least N chars)?
- What happens in non-TTY mode where terminal width may not be detectable? Fall back to a default width (80? 120?)?

**Proposed Addition**:

**Resolution**: Pending
**Notes**: This is the only place in the entire codebase that would need terminal width detection, making it a non-trivial infrastructure addition that deserves more specification.

---

### 10. Focused view with asymmetric results not addressed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Edge Cases

**Details**:
The focused mode walks both directions. The spec covers the case where neither direction has results ("No dependencies.") but doesn't address asymmetric cases:

- Task has upstream blockers but blocks nothing downstream -- show "Blocked by:" tree, then what for "Blocks:"? Empty section? Omitted?
- Task blocks things downstream but has no upstream blockers -- show nothing for "Blocked by:"? Or an explicit "None" marker?

These are distinct from the "no dependencies at all" case and affect the visual layout.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 11. Pretty format status rendering not specified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Rendering

**Details**:
The spec says each line shows "ID + title + status" but doesn't specify the format string. Looking at existing patterns, the pretty formatter uses aligned columns for task lists. For the tree, the indentation varies per depth level, making column alignment impractical. The spec should clarify:

- Exact format per line, e.g., `{indent}{tree_char} {id} {title} [{status}]` or `{indent}{tree_char} [{status}] {id} {title}`
- How status is rendered (brackets? parentheses? plain?)
- Separator between ID, title, and status

**Proposed Addition**:

**Resolution**: Pending
**Notes**: Minor but affects visual consistency. An implementer could reasonably pick different orderings.

