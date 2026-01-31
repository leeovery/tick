---
id: doctor-validation-3-4
phase: 3
status: pending
created: 2026-01-31
---

# Dependency Cycle Detection Check

## Goal

Tasks 3-1 through 3-3 validate individual reference integrity: orphaned parents, orphaned dependencies, and self-referential dependencies. None of them detect multi-task circular dependency chains where A depends on B, B depends on C, and C depends on A. In such a cycle, every task in the chain is blocked by another task in the chain — none can ever become ready because they are all waiting on each other. This is a deadlock that is invisible when examining any single task's `blocked_by` in isolation; only graph traversal reveals it. This task implements `DependencyCycleCheck` using DFS with three-color marking to detect all circular dependency chains in the task graph. This is specification Error #8: "Circular dependency chains (A→B→C→A)."

Self-references (A depends on A) are excluded from this check — task 3-3 owns that detection. This check only reports cycles involving two or more distinct tasks. The specification requires each error to be reported individually, so each detected cycle produces its own failing `CheckResult`.

## Implementation

- Create a `DependencyCycleCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:

  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Dependency cycles"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the pattern established in tasks 3-1, 3-2, and 3-3.

  2. Build an adjacency list (dependency graph) from the parsed task data. For each task, map its `ID` to the list of IDs in its `BlockedBy` slice. When building the adjacency list, **exclude self-references** — if a task's `BlockedBy` contains its own ID, skip that entry. Self-references are handled by task 3-3 and must not be included in cycle detection to avoid double-reporting.

  3. Also build a set of all known task IDs from the parsed data. When iterating `BlockedBy` entries to build the adjacency list, **exclude references to IDs that do not exist** in the known-ID set. Orphaned dependencies are handled by task 3-2. Including non-existent IDs in the graph would create phantom nodes that cannot participate in real cycles. Only edges where both endpoints exist in the task data should be in the adjacency list.

  4. Run DFS with three-color marking (white/gray/black) to detect cycles:
     - **White** (unvisited): Node has not been visited yet.
     - **Gray** (in progress): Node is currently in the DFS recursion stack — it is being explored and its descendants are not yet fully processed.
     - **Black** (done): Node and all its descendants have been fully explored.
     - Iterate all known task IDs. For each white node, begin a DFS:
       - Mark the node gray.
       - For each neighbor (dependency) of the current node in the adjacency list:
         - If the neighbor is gray, a cycle has been found — the path from the neighbor to the current node (inclusive) forms a cycle. Record this cycle.
         - If the neighbor is white, recurse into it.
         - If the neighbor is black, skip it (already fully explored, no cycle through it).
       - After exploring all neighbors, mark the node black.
     - To reconstruct the cycle path when a gray neighbor is encountered, maintain a path/stack during DFS traversal. When a back-edge to a gray node is found, extract the cycle from the path: starting at the gray node's position in the path through the current node, back to the gray node. For example, if the path is `[A, B, C]` and C's neighbor B is gray, the cycle is `B→C→B`.

  5. Deduplicate cycles. The same cycle can be discovered from different starting nodes (e.g., starting DFS from A finds A→B→C→A, starting from B finds B→C→A→B — these are the same cycle). To deduplicate:
     - Normalize each cycle by rotating it so the lexicographically smallest ID is first. For example, cycle `[C, A, B]` becomes `[A, B, C]` after rotation.
     - Store normalized cycles in a set (e.g., using the joined string as a map key). Only add a cycle if its normalized form has not been seen before.

  6. For each unique cycle detected, produce a failing `CheckResult` with:
     - Name: `"Dependency cycles"`
     - Severity: `SeverityError`
     - Details: A human-readable description of the cycle chain. Format: `"Dependency cycle: tick-A → tick-B → tick-C → tick-A"` — the first node is repeated at the end to show the cycle closing. Use the normalized (lexicographically-first) ordering so the output is deterministic and testable.
     - Suggestion: `"Manual fix required"`

  7. After checking all nodes, if no cycles were found, return a single passing `CheckResult` with Name `"Dependency cycles"` and Passed `true`.

  8. If cycles were found, return all the failing `CheckResult` entries (one per unique cycle). Do not include a passing result alongside failures.

- The check does **not** normalize IDs to lowercase before comparison. IDs in `tasks.jsonl` should already be lowercase per write-time normalization. Compare as-is, consistent with tasks 3-1, 3-2, and 3-3.

- The adjacency list is built from `BlockedBy` semantics: if task A has `blocked_by: ["tick-B"]`, the edge is A→B (A depends on B). A cycle occurs when following dependency edges eventually leads back to the starting node.

- Tasks with empty `BlockedBy` (no dependencies) have no outgoing edges in the dependency graph. They cannot be part of a cycle as a source, but they can still be referenced by other tasks' `blocked_by`. They appear as nodes with no outgoing edges.

## Tests

- `"it returns passing result when no dependency cycles exist"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when tasks have dependencies but no cycles (valid DAG)"`
- `"it returns passing result when tasks have no dependencies (all empty blocked_by)"`
- `"it detects a simple 2-node cycle (A depends on B, B depends on A)"`
- `"it detects a 3-node cycle (A→B→C→A)"`
- `"it detects a longer cycle (4+ nodes)"`
- `"it detects multiple independent cycles in the same graph"`
- `"it does not report a chain that is not a cycle (A→B→C with no back-edge)"`
- `"it does not report self-references (handled by task 3-3)"`
- `"it excludes self-references from adjacency list to avoid false cycle detection"`
- `"it excludes orphaned dependency references from adjacency list (non-existent targets)"`
- `"it handles complex graph with both cycles and valid chains — reports only cycles"`
- `"it reports each unique cycle as a separate failing CheckResult"`
- `"it deduplicates cycles found from different starting nodes"`
- `"it formats cycle details as 'Dependency cycle: tick-A → tick-B → tick-C → tick-A'"`
- `"it uses deterministic ordering in cycle output (lexicographically smallest ID first)"`
- `"it skips unparseable lines — does not include them in dependency graph"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for cycle errors"`
- `"it uses CheckResult Name 'Dependency cycles' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with a suggestion to initialize. Consistent with the Phase 2 and Phase 3 pattern.

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no dependency graph to traverse. The check returns a single passing result.

- **Simple 2-node cycle**: Task A has `blocked_by: ["tick-B"]` and task B has `blocked_by: ["tick-A"]`. The DFS discovers A→B→A. This is the minimal multi-task cycle. One failing result is produced with details `"Dependency cycle: tick-A → tick-B → tick-A"` (assuming A is lexicographically first).

- **3+ node cycle**: Task A depends on B, B depends on C, C depends on A. The DFS discovers A→B→C→A. One failing result. This matches the specification's explicit example: "Circular dependency chains (A→B→C→A)."

- **Multiple independent cycles**: The graph contains two or more cycles that share no nodes. For example, A→B→A and C→D→C. Each cycle produces its own failing result. The deduplication ensures each cycle is reported exactly once.

- **Chain that is not a cycle**: Tasks form a dependency chain A→B→C where C has no dependencies. This is a valid DAG. The DFS fully explores each node, marks them black, and finds no back-edges. The check returns passing. This confirms no false positives on long chains.

- **Self-reference not double-reported**: If task A has `blocked_by: ["tick-A"]`, this is a self-reference handled exclusively by task 3-3. The cycle detection check filters out self-references when building the adjacency list, so A has no outgoing edge to itself and no cycle involving only A is reported. This prevents the same issue from being flagged by both task 3-3 and task 3-4.

- **Task with no dependencies**: Tasks with empty `BlockedBy` have no outgoing edges. They cannot initiate or participate in a cycle as a blocker. They may still be referenced in other tasks' `blocked_by` arrays. These tasks are simply leaf nodes in the dependency graph.

- **Complex graph with both cycles and valid chains**: A graph like A→B→C→A (cycle) plus D→E→F (valid chain) plus G→H→G (another cycle). The check reports the two cycles (A→B→C→A and G→H→G) but not the valid chain. The deduplication and DFS coloring ensure correct results even in complex mixed graphs.

- **Overlapping cycles (shared nodes)**: If A→B→C→A and A→B→D→A both exist (B is shared), these are two distinct cycles and both should be reported. The DFS discovers both because C→A and D→A are separate back-edges found during traversal.

- **Unparseable lines skipped**: Lines that are not valid JSON are skipped by the parser. They do not appear in the dependency graph. If a task's `blocked_by` references an ID on an unparseable line, that edge is excluded from the adjacency list (the target ID is not in the known-ID set), consistent with the orphaned reference filtering.

## Acceptance Criteria

- [ ] `DependencyCycleCheck` implements the `Check` interface
- [ ] Check reuses `ParseTaskRelationships` from task 3-1
- [ ] DFS with three-color marking (white/gray/black) used for cycle detection
- [ ] Self-references excluded from the adjacency list (task 3-3 owns self-reference detection)
- [ ] Orphaned dependency references excluded from the adjacency list (non-existent target IDs filtered out)
- [ ] Each unique cycle produces its own failing `CheckResult`
- [ ] Cycles are deduplicated — same cycle discovered from different starting nodes reported only once
- [ ] Cycle details formatted as `"Dependency cycle: tick-A → tick-B → tick-C → tick-A"` (first node repeated at end)
- [ ] Deterministic cycle output — lexicographically smallest ID first in the normalized cycle
- [ ] Passing check returns `CheckResult` with Name `"Dependency cycles"` and Passed `true`
- [ ] Valid DAGs (chains without back-edges) do not produce false positives
- [ ] Tasks with no dependencies do not cause errors
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for cycle errors
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases including 2-node, 3+-node, multiple independent, overlapping, and complex mixed graphs

## Context

The specification defines dependency cycles as Error #8: "Circular dependency chains (A→B→C→A)." The fix suggestion table maps this to "Manual fix required" (under "All other errors"). The specification requires that "Doctor lists each error individually" — each distinct cycle is a separate error.

The tick-core specification describes the `blocked_by` field as: type `array`, required `No`, default `[]`. It contains task IDs that must reach `done` or `cancelled` before the task becomes `ready`. A cycle in this dependency graph means a set of tasks that are mutually blocking — none can ever become ready because each waits on another member of the cycle.

The specification's design principle #4 ("Run all checks") means the cycle detection must complete even if cycles are found — it does not stop at the first cycle. All cycles in the graph must be discovered and reported.

Self-references (Error #7, task 3-3) are explicitly excluded from cycle detection to avoid double-reporting. Task 3-3's context states: "Task 3-4 (dependency cycle detection) explicitly excludes self-references from its scope per its edge case note: 'self-reference not double-reported (handled by task 3-3).'" The implementation enforces this by stripping self-referential edges before building the adjacency list.

DFS with three-color marking is the standard algorithm for cycle detection in directed graphs. White/gray/black coloring distinguishes unvisited, in-progress, and completed nodes. A back-edge from a node to a gray ancestor indicates a cycle. This runs in O(V+E) time where V is the number of tasks and E is the total number of dependency edges — efficient for any practical task count.

This is a Go project. The check implements the `Check` interface defined in task 1-1, reuses the `ParseTaskRelationships` parser from task 3-1, and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
