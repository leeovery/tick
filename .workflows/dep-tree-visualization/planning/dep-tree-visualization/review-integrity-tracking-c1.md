---
status: in-progress
created: 2026-03-27
cycle: 1
phase: Plan Integrity Review
topic: Dep Tree Visualization
---

# Review Tracking: Dep Tree Visualization - Integrity

## Findings

### 1. Missing cross-phase dependency: Phase 2 not blocked by Phase 1

**Severity**: Critical
**Plan Reference**: Phase 2 (tick-fda1e3)
**Category**: Dependencies and Ordering
**Change Type**: update-task

**Details**:
Phase 2 tasks (ToonFormatter, JSONFormatter) depend on infrastructure created in Phase 1: the DepTreeResult/DepTreeNode/DepTreeTask types, the FormatDepTree interface method, and the BuildFullDepTree/BuildFocusedDepTree functions. Without an explicit blocked_by edge from Phase 2 to Phase 1, the `tick ready` system surfaces Phase 2 tasks immediately — before any Phase 1 work exists. The ReadyNoBlockedAncestor CTE walks the parent chain and checks for unclosed dependency blockers on ancestors; adding blocked_by from Phase 2 to Phase 1 causes Phase 2 tasks to inherit the blocked status through their parent (Phase 2).

**Current**:
Phase 2 (tick-fda1e3) has no blocked_by dependencies:
```
blocked_by[0]{id,title,status}:
```

**Proposed**:
Add dependency: Phase 2 (tick-fda1e3) blocked_by Phase 1 (tick-7f41cf).

```bash
tick dep add tick-fda1e3 tick-7f41cf
```

**Resolution**: Pending
**Notes**:

---

### 2. Phase 2 tasks lack self-contained context for data model types

**Severity**: Important
**Plan Reference**: Phase 2 / Task dep-tree-visualization-2-1 (tick-964a0b) and dep-tree-visualization-2-2 (tick-f0bfcd)
**Category**: Task Self-Containment
**Change Type**: add-to-task

**Details**:
Tasks 2-1 (ToonFormatter) and 2-2 (JSONFormatter) reference DepTreeResult, DepTreeNode, and DepTreeTask types throughout their Do steps and Acceptance Criteria but never describe their structure. An implementer picking up either task would need to read the codebase to understand what fields are available on these types. Including the type structure as Context ensures each task is self-contained.

**Current**:
Task 2-1 (tick-964a0b) description ends with the Spec Reference line and has no Context section.

Task 2-2 (tick-f0bfcd) description ends with the Spec Reference line and has no Context section.

**Proposed**:
Append the following Context section to **both** task descriptions (tick-964a0b and tick-f0bfcd):

```
Context:
> Data model types defined in format.go (created in Phase 1, Task 1-1):
>
> DepTreeResult has two modes indicated by a Mode field ("full" or "focused").
> Full-graph mode uses: Roots []DepTreeNode, Chains int, Longest int, Blocked int.
> Focused mode uses: Target DepTreeTask, BlockedBy []DepTreeNode, Blocks []DepTreeNode.
>
> DepTreeNode wraps a DepTreeTask with Children []DepTreeNode for recursive tree structure.
> DepTreeTask holds ID, Title, and Status strings.
>
> Diamond dependencies appear as duplicate DepTreeNode entries (no deduplication).
```

**Resolution**: Pending
**Notes**: The exact field names may vary slightly from what Task 1-1 implements, but this gives the implementer enough structural understanding to work independently.

---

### 3. Task 1-1 includes redundant formatter stub steps

**Severity**: Minor
**Plan Reference**: Phase 1 / Task dep-tree-visualization-1-1 (tick-38bd4e)
**Category**: Task Template Compliance
**Change Type**: update-task

**Details**:
Task 1-1 step 1 adds a stub FormatDepTree on baseFormatter. Steps 2 and 3 then add individual stubs on PrettyFormatter and ToonFormatter. However, both PrettyFormatter and ToonFormatter embed baseFormatter, so they automatically inherit the stub from step 1. Steps 2 and 3 are redundant for these two formatters. JSONFormatter and StubFormatter do NOT embed baseFormatter, so they genuinely need their own stubs. An implementer following these steps literally would write unnecessary code, or worse, be confused about the embedding relationship.

**Current**:
```
Do:
1. format.go - Define DepTreeResult, DepTreeNode, DepTreeTask types; add FormatDepTree(DepTreeResult) string to Formatter interface; stub on StubFormatter and baseFormatter
2. pretty_formatter.go - Add stub FormatDepTree returning empty string
3. toon_formatter.go - Add stub FormatDepTree returning empty string
4. json_formatter.go - Add stub FormatDepTree returning empty string
5. flags.go - Add "dep tree": {} to commandFlags
6. app.go - Extend qualifyCommand dep case to match "tree"
7. dep.go - Add case "tree": calling RunDepTree; create placeholder in new dep_tree.go
8. help.go - Update dep help entry to mention tree
```

**Proposed**:
```
Do:
1. format.go - Define DepTreeResult, DepTreeNode, DepTreeTask types; add FormatDepTree(DepTreeResult) string to Formatter interface; stub on baseFormatter (PrettyFormatter and ToonFormatter inherit via embedding)
2. json_formatter.go - Add stub FormatDepTree returning empty string (JSONFormatter does not embed baseFormatter)
3. format.go - Add stub FormatDepTree on StubFormatter returning empty string
4. flags.go - Add "dep tree": {} to commandFlags
5. app.go - Extend qualifyCommand dep case to match "tree"
6. dep.go - Add case "tree": calling RunDepTree; create placeholder in new dep_tree.go
7. help.go - Update dep help entry to mention tree
```

**Resolution**: Pending
**Notes**:
