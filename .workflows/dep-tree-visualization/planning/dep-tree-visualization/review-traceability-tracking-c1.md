---
status: in-progress
created: 2026-03-27
cycle: 1
phase: Traceability Review
topic: Dep Tree Visualization
---

# Review Tracking: Dep Tree Visualization - Traceability

## Findings

### 1. Focused mode "No dependencies" edge case does not show the task itself

**Type**: Incomplete coverage
**Spec Reference**: Edge Cases -- "Task with no dependencies (focused mode): Show the task itself with 'No dependencies.'"
**Plan Reference**: Phase 1 / dep-tree-visualization-1-3 (RunDepTree Command Handler)
**Change Type**: update-task

**Details**:
The specification says for the focused-mode "no dependencies" edge case: "Show the task itself with 'No dependencies.'" The Phase 2 acceptance criteria correctly mirrors this: "Task with no dependencies in focused mode shows the task itself with 'No dependencies.'" However, Task 1-3's implementation plan short-circuits to `FormatMessage("No dependencies.")` when both BlockedBy and Blocks are empty, bypassing the formatter entirely. `FormatMessage` renders a plain text string -- it does not include the task's ID, title, or status. The spec requires displaying the task alongside the message. The handler should pass the result through `FormatDepTree` (which carries the Target task info) and let each formatter render the task header plus the "No dependencies." message, or at minimum show the task before the message.

**Current**:
In the phase-1-tasks.md file, Task 3 Do step 8:
```
8. Focused: if error, return it; if empty both directions, output FormatMessage("No dependencies."); else FormatDepTree
```

And the tick task description for tick-8d0709 contains the same:
```
7. Full graph: if empty Roots, output FormatMessage("No dependencies found."); else FormatDepTree
8. Focused: if error, return it; if empty both directions, output FormatMessage("No dependencies."); else FormatDepTree
```

Acceptance Criteria item:
```
- [ ] `tick dep tree <id>` with a valid task ID that has no dependencies outputs "No dependencies."
```

Test:
```
- `"it outputs no dependencies for isolated task in focused mode"` -- setup task with no deps, run focused on it, output contains "No dependencies."
```

**Proposed**:
In the phase-1-tasks.md file, Task 3 Do step 8 replace with:
```
8. Focused: if error, return it; if empty both directions, output the target task line (ID + title + status) followed by FormatMessage("No dependencies."); else FormatDepTree
```

And the tick task description for tick-8d0709, replace steps 7-8 with:
```
7. Full graph: if empty Roots, output FormatMessage("No dependencies found."); else FormatDepTree
8. Focused: if error, return it; if empty both directions, show the target task (ID + title + status) then output FormatMessage("No dependencies."); else FormatDepTree
```

Acceptance Criteria item replace with:
```
- [ ] `tick dep tree <id>` with a valid task ID that has no dependencies shows the task itself (ID + title + status) with "No dependencies."
```

Test replace with:
```
- `"it outputs no dependencies for isolated task in focused mode"` -- setup task with no deps, run focused on it, output contains task ID and title and "No dependencies."
```

**Resolution**: Fixed
**Notes**: Applied to phase-1-tasks.md and tick task tick-8d0709.

---

### 2. JSON formatter asymmetric focused view contradicts spec and plan table edge cases

**Type**: Incomplete coverage
**Spec Reference**: Edge Cases -- "Asymmetric focused view -- if a task has upstream blockers but blocks nothing, show the 'Blocked by:' tree and omit the 'Blocks:' section (and vice versa). Only show sections that have content."
**Plan Reference**: Phase 2 / dep-tree-visualization-2-2 (JSONFormatter FormatDepTree)
**Change Type**: update-task

**Details**:
The specification says for the asymmetric focused view: "Only show sections that have content." The planning.md task table for dep-tree-visualization-2-2 lists the edge case as "asymmetric focused view omits empty arrays, empty arrays render as [] not null" which is self-contradictory -- "omits" implies the key is absent, while "render as []" implies the key is present. The phase-2-tasks.md task description then explicitly says "Always include both blocked_by and blocks keys in focused mode" and renders empty directions as `[]`, which contradicts both the spec's "Only show sections that have content" and the plan table's "omits empty arrays."

The task's Edge Cases section acknowledges this as an ambiguity and provides justification based on existing JSON conventions. While the reasoning is sound (Go/JSON convention of always including array keys), this is the plan inventing its own interpretation rather than faithfully translating the spec. The spec says "Exact shape determined during implementation" for JSON, which grants latitude on structure, but the asymmetric view rule is a cross-format requirement stated in the Edge Cases section, not in the JSON-specific formatter section.

The plan table edge case description in planning.md should be corrected to remove the contradiction. The task should either faithfully implement the spec's "omit empty sections" (by using `omitempty` JSON tags or conditional marshalling), or flag this as a deliberate deviation with `[needs-info]` for the user to decide.

**Current**:
In planning.md, the task table for dep-tree-visualization-2-2:
```
| dep-tree-visualization-2-2 | JSONFormatter FormatDepTree | diamond dependencies produce duplicate nodes in tree, asymmetric focused view omits empty arrays, empty arrays render as [] not null |
```

In phase-2-tasks.md, the Solution paragraph:
```
Empty arrays render as `[]` not `null`, following the existing JSON formatter convention. Asymmetric focused views include both keys but render the empty direction as `[]`.
```

In phase-2-tasks.md, the Do step 3 focused mode section:
```
   - **Focused mode** (`result.Mode == "focused"`):
     a. Convert `result.Target` to `jsonDepTreeTask`.
     b. Convert `result.BlockedBy` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes`. If nil or empty, use `make([]jsonDepTreeNode, 0)` to ensure `[]` in JSON.
     c. Convert `result.Blocks` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes`. Same nil-safety.
     d. Build `jsonDepTreeFocused{Mode: "focused", Target: target, BlockedBy: blockedBy, Blocks: blocks}`.
     e. Return `marshalIndentJSON(obj)`.
```

In phase-2-tasks.md, Acceptance Criteria:
```
- [ ] Empty `blocked_by` or `blocks` in focused mode renders as `[]` not `null`
```

In phase-2-tasks.md, Edge Cases section:
```
- Asymmetric focused view: unlike the toon format which omits empty sections, the JSON format always includes both `blocked_by` and `blocks` keys, rendering the empty direction as `[]`. This follows the existing JSON convention where arrays are always present (e.g., `blocked_by` and `children` in `FormatTaskDetail` are always `[]` not omitted). The spec says "omits empty arrays" for focused view, but this should be interpreted as "empty arrays render as `[]`" to maintain JSON structural consistency. This is noted as an ambiguity -- the spec's phrasing "Asymmetric focused view omits empty arrays" could mean either "omit the key" or "render as empty array". Given the project's strong convention of always including array keys (never omitting them), we render as `[]`.
```

**Proposed**:
In planning.md, the task table for dep-tree-visualization-2-2:
```
| dep-tree-visualization-2-2 | JSONFormatter FormatDepTree | diamond dependencies produce duplicate nodes in tree, asymmetric focused view omits empty key from JSON object, empty arrays for non-omitted keys render as [] not null |
```

In phase-2-tasks.md, the Solution paragraph:
```
Empty arrays render as `[]` not `null`, following the existing JSON formatter convention. Asymmetric focused views omit the empty direction's key from the JSON object entirely, matching the spec's "Only show sections that have content" rule. Use conditional marshalling -- only include `blocked_by` when non-empty, only include `blocks` when non-empty.
```

In phase-2-tasks.md, the Do step 3 focused mode section:
```
   - **Focused mode** (`result.Mode == "focused"`):
     a. Convert `result.Target` to `jsonDepTreeTask`.
     b. Convert `result.BlockedBy` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes` if non-empty, otherwise leave as nil.
     c. Convert `result.Blocks` to `[]jsonDepTreeNode` via `toJSONDepTreeNodes` if non-empty, otherwise leave as nil.
     d. Build `jsonDepTreeFocused{Mode: "focused", Target: target, BlockedBy: blockedBy, Blocks: blocks}`. The `BlockedBy` and `Blocks` fields use `json:"blocked_by,omitempty"` and `json:"blocks,omitempty"` tags so empty directions are omitted from JSON output.
     e. Return `marshalIndentJSON(obj)`.
```

In phase-2-tasks.md, Acceptance Criteria, replace the empty arrays criterion:
```
- [ ] Asymmetric focused view omits the empty direction key from JSON output entirely (key absent, not `[]`)
- [ ] Leaf node `children` renders as `[]` not `null` (children is always present on nodes)
- [ ] Empty `roots` in full graph mode renders as `[]` not `null`
```

In phase-2-tasks.md, Edge Cases section replace the asymmetric bullet:
```
- Asymmetric focused view: the spec says "Only show sections that have content." For JSON focused mode, this means omitting the `blocked_by` key entirely when there are no upstream dependencies, and omitting the `blocks` key when there are no downstream dependencies. Use `omitempty` JSON struct tags on the `BlockedBy` and `Blocks` fields of `jsonDepTreeFocused`. Note: this differs from the full-graph `roots` field and node `children` fields which are always present (rendered as `[]` not `null`) because those are structural elements, not directional sections.
```

Also update the tests in phase-2-tasks.md:
Replace:
```
- `"it renders focused blocked_by as [] when only downstream"` -- root task A blocking B: `blocked_by` is `[]`, `blocks` has content
- `"it renders focused blocks as [] when only upstream"` -- leaf task C blocked by B: `blocks` is `[]`, `blocked_by` has content
```
With:
```
- `"it omits blocked_by key when only downstream exists"` -- root task A blocking B: JSON has no `blocked_by` key, `blocks` has content
- `"it omits blocks key when only upstream exists"` -- leaf task C blocked by B: JSON has no `blocks` key, `blocked_by` has content
```

And update the struct definition in Do step 1:
Replace:
```
   type jsonDepTreeFocused struct {
       Mode      string             `json:"mode"`
       Target    jsonDepTreeTask    `json:"target"`
       BlockedBy []jsonDepTreeNode  `json:"blocked_by"`
       Blocks    []jsonDepTreeNode  `json:"blocks"`
   }
```
With:
```
   type jsonDepTreeFocused struct {
       Mode      string             `json:"mode"`
       Target    jsonDepTreeTask    `json:"target"`
       BlockedBy []jsonDepTreeNode  `json:"blocked_by,omitempty"`
       Blocks    []jsonDepTreeNode  `json:"blocks,omitempty"`
   }
```

**Resolution**: Pending
**Notes**: The existing codebase convention of always including array keys (e.g., blocked_by in FormatTaskDetail) is noted but the spec's explicit "Only show sections that have content" directive for the asymmetric focused view should take precedence. The spec's "Exact shape determined during implementation" for JSON grants latitude on structure but the asymmetric omission rule is a cross-format edge case requirement, not a JSON-specific design choice. If the user prefers to keep both keys as [] for JSON consistency, they can reject this finding.

---

### 3. Dependencies-only scope constraint not documented in tasks

**Type**: Incomplete coverage
**Spec Reference**: Scope -- "Dependencies only. No parent/child relationships, no parent annotations."
**Plan Reference**: Phase 1 / dep-tree-visualization-1-2 (Dep Tree Data Model and Graph-Walking Algorithm)
**Change Type**: add-to-task

**Details**:
The specification has an explicit "Scope" section stating: "Dependencies only. No parent/child relationships, no parent annotations. Parent/child (decomposition) and dependencies (ordering) have different semantics. Mixing them in one tree creates ambiguity." This is a key design constraint that prevents scope creep. While the graph algorithm in Task 1-2 implicitly only uses BlockedBy (dependency relationships) and ignores Parent (hierarchy), this constraint is not documented anywhere in the task content. An implementer might reasonably add parent annotations to tree output (e.g., showing "parent: X" on each task line) thinking it's helpful context. The constraint should be explicit in the graph algorithm task's Context section.

**Current**:
In phase-1-tasks.md, Task 2 Context section:
```
**Context**:
> The specification explicitly states: "Diamond dependencies (task reachable via multiple paths): Duplicate the task wherever it appears in the graph. No deduplication, no back-references, no special markers." This means the recursive walk must NOT use a visited set. The specification also states: "Tasks with zero dependencies (neither blocking nor blocked) are omitted" from the full graph view. The "Depth: Full transitive — walk the entire chain with no artificial cap" constraint means no depth limit. For the summary line: "chain = connected component of the dependency graph, longest = longest path measured in edges, blocked = tasks with at least one BlockedBy entry."
```

**Proposed**:
In phase-1-tasks.md, Task 2 Context section:
```
**Context**:
> The specification explicitly states: "Diamond dependencies (task reachable via multiple paths): Duplicate the task wherever it appears in the graph. No deduplication, no back-references, no special markers." This means the recursive walk must NOT use a visited set. The specification also states: "Tasks with zero dependencies (neither blocking nor blocked) are omitted" from the full graph view. The "Depth: Full transitive — walk the entire chain with no artificial cap" constraint means no depth limit. For the summary line: "chain = connected component of the dependency graph, longest = longest path measured in edges, blocked = tasks with at least one BlockedBy entry."
>
> Scope constraint: "Dependencies only. No parent/child relationships, no parent annotations." The graph algorithm must only use BlockedBy fields (dependency relationships) and must not traverse or display Parent fields (hierarchy relationships). Parent/child and dependencies have different semantics -- mixing them creates ambiguity about whether B is under A because A blocks B or because B is a child of A.
```

**Resolution**: Pending
**Notes**: The algorithm is already correct by construction since it only reads BlockedBy fields. This finding adds explicit documentation of the constraint to prevent scope creep during implementation, particularly in the formatter tasks where someone might add parent annotations to task display lines.
