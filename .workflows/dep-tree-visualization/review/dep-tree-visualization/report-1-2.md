TASK: Dep Tree Data Model and Graph-Walking Algorithm

ACCEPTANCE CRITERIA:
- BuildFullDepTree with no deps returns empty result with zero counts
- Linear chain A->B->C returns 1 root, longest 2, 1 chain, 2 blocked
- Diamond (A->B,C both->D) duplicates D under both B and C
- Tasks with no dependency relationships are omitted
- BuildFocusedDepTree for mid-chain shows both upstream and downstream
- BuildFocusedDepTree for isolated task returns empty BlockedBy and Blocks
- BuildFocusedDepTree for nonexistent ID returns error
- Diamond dependencies duplicated (no deduplication)

STATUS: Complete

SPEC CONTEXT: The specification defines two modes for dep tree visualization. Full graph mode shows all tasks participating in dependency relationships with root tasks (block others but not blocked themselves) at the top level, walking downstream recursively. Focused mode walks both upstream (BlockedBy) and downstream (blocks) from a target task. Diamond dependencies are explicitly duplicated -- "No deduplication, no back-references, no special markers." Full transitive depth with no artificial cap. Scope is dependencies only -- no parent/child relationships.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/dep_tree_graph.go (274 lines)
- Data types: /Users/leeovery/Code/tick/internal/cli/format.go:150-180 (DepTreeTask, DepTreeNode, DepTreeResult)
- Notes:
  - All planned functions are present: buildBlocksIndex, buildTaskIndex, toDepTreeTask, walkDownstream, walkUpstream, longestPath, BuildFullDepTree, countChains, BuildFocusedDepTree
  - The plan mentioned a `Mode string` field on DepTreeResult; the implementation omits it, using `Target != nil` to distinguish modes instead. This is a reasonable simplification -- the Mode field would have been redundant, and the formatters correctly use `result.Target != nil` for mode dispatch (pretty_formatter.go:308).
  - Cycle guard (ancestors map with path-based tracking) was added to walkDownstream and walkUpstream despite the plan saying "no visited set." This is a defensive measure against corrupted data -- it does NOT prevent diamond duplication (ancestors are removed after processing via `delete(ancestors, id)`). The path-based approach correctly allows the same node to appear via different paths while preventing infinite recursion on cycles.
  - Dependencies-only scope constraint respected: no reference to Parent field anywhere in dep_tree_graph.go.
  - longestPath correctly counts edges (not nodes): a 3-node chain A->B->C yields longest=2.
  - countChains uses BFS on an undirected adjacency graph to count connected components -- correct algorithm.
  - Summary line has correct singular/plural handling ("1 chain" vs "N chains").
  - Edge case messages match spec: "No dependencies found." (full graph, no deps), "No dependencies." (focused, isolated task).

TESTS:
- Status: Adequate
- Coverage: All 17 planned tests are present plus 7 additional cycle-guard tests
- Test file: /Users/leeovery/Code/tick/internal/cli/dep_tree_graph_test.go (739 lines)
- Planned tests present:
  - "it returns empty result for project with no dependencies" (line 24)
  - "it returns empty result when task list is empty" (line 52)
  - "it builds a single linear chain" (line 75)
  - "it builds multiple independent chains" (line 122)
  - "it duplicates diamond dependency without deduplication" (line 162)
  - "it omits tasks with no dependency relationships" (line 196)
  - "it handles task blocked by multiple roots" (line 218)
  - "it computes longest chain across multiple chains" (line 243)
  - "it counts blocked tasks correctly" (line 270)
  - "it builds focused upstream tree" (line 296)
  - "it builds focused downstream tree" (line 334)
  - "it builds focused view for mid-chain task" (line 366)
  - "it returns no dependencies for isolated task in focused mode" (line 398)
  - "it returns error for nonexistent task ID in focused mode" (line 426)
  - "it duplicates diamond in focused downstream" (line 441)
  - "it handles focused view with only upstream dependencies" (line 471)
  - "it handles focused view with only downstream dependencies" (line 497)
- Additional tests (TestCycleGuard, line 524):
  - "it terminates walkDownstream with circular dependency A blocks B blocks A"
  - "it terminates walkUpstream with circular dependency A blocks B blocks A"
  - "it terminates walkDownstream with three-node cycle"
  - "it terminates full graph with circular dependency"
  - "it preserves acyclic diamond duplication after cycle guard addition"
  - "it duplicates diamond node with children under both paths"
  - "it preserves acyclic focused view after cycle guard addition"
- Notes: The cycle guard tests are valuable defensive coverage, not over-testing. They verify that the ancestor-based cycle detection does not break the diamond-duplication behavior. The deep-diamond test (line 645) specifically verifies that D's children (E) appear under both paths, which would fail with a permanent visited set.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, helper function with test data construction (makeTask), no external test dependencies.
- SOLID principles: Good. Pure functions with clear single responsibilities (buildBlocksIndex, buildTaskIndex, walkDownstream, walkUpstream, longestPath, countChains). No side effects. Data structures defined separately in format.go.
- Complexity: Low. Each function has a clear purpose. The recursive walk functions are straightforward with a single branching point (cycle guard check). countChains uses standard BFS.
- Modern idioms: Yes. Proper use of maps, slices, make with capacity hints where appropriate (buildTaskIndex). Error wrapping with fmt.Errorf in BuildFocusedDepTree.
- Readability: Good. Functions are well-documented with clear godoc comments explaining purpose, behavior, and the cycle guard mechanism. Variable names are descriptive (participants, ancestors, blocks).
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The omission of the `Mode` field from DepTreeResult is a minor deviation from the plan's "Do" instructions, but it is a net improvement -- the field would have been redundant and the mode is unambiguously determined by whether `Target` is nil or non-nil. The formatters and handler already rely on this convention.
- The `buildTaskIndex` function is called in both `BuildFullDepTree` and `BuildFocusedDepTree`, and `buildBlocksIndex` similarly. If these were ever called in sequence on the same task list, the indexes could be precomputed once. However, since the two functions serve different modes and are called independently, this is not an issue in practice.
