TASK: Add Cycle Guard to walkDownstream and walkUpstream

ACCEPTANCE CRITERIA:
- walkDownstream and walkUpstream both accept and use a visited set
- A dependency cycle in the data does not cause a stack overflow or panic
- Normal (acyclic) dep tree output is unchanged

STATUS: Complete

SPEC CONTEXT: The specification requires diamond dependencies to be duplicated wherever they appear in the graph -- no deduplication, no back-references, no special markers. The cycle guard must therefore use path-based ancestor tracking (not a permanent visited set) to prevent infinite recursion while still allowing the same node to appear multiple times via different paths.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/dep_tree_graph.go:43-68 (walkDownstream), :74-103 (walkUpstream)
- Notes: Both functions accept an `ancestors map[string]bool` parameter. On entry, if the current ID is already in `ancestors`, the function returns nil (cycle detected). Otherwise it adds itself, recurses, then deletes itself from `ancestors` after processing. This path-based approach is superior to a simple permanent visited set because it correctly preserves diamond duplication (same node reachable via different paths is fully expanded each time) while preventing infinite recursion on actual cycles. All call sites at lines 151, 259, and 260 pass fresh `make(map[string]bool)` maps. The approach is consistent with the spec's explicit requirement to duplicate diamond dependencies.

TESTS:
- Status: Adequate
- Coverage:
  - Two-node downstream cycle (A blocks B, B blocks A): line 525 -- verifies termination and finite node count
  - Two-node upstream cycle (same data, walkUpstream direction): line 554 -- verifies termination and finite node count
  - Three-node cycle (A->B->C->A): line 579 -- verifies termination with longer cycle
  - Full graph mode with all-cyclic data: line 599 -- verifies no roots found (all tasks have BlockedBy)
  - Diamond duplication preserved (acyclic): line 615 -- verifies D appears under both B and C
  - Deep diamond with children preserved (acyclic): line 645 -- verifies D->E subtree appears under both paths, confirming ancestor tracking does not suppress diamond duplication
  - Acyclic focused view preserved: line 687 -- verifies normal upstream/downstream still works correctly
- Notes: Tests are well-structured and cover all three acceptance criteria. The `countNodes > 10` threshold is a pragmatic bound -- without the guard, the test would stack overflow rather than merely exceed the threshold, so the test effectively validates termination. The diamond duplication tests (lines 615 and 645) are particularly valuable as they verify the ancestor-tracking approach does not regress the spec's diamond requirement.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.Helper on helpers, "it does X" naming convention
- SOLID principles: Good -- walkDownstream/walkUpstream have single responsibility (tree walking), the ancestors parameter is injected (dependency inversion at function level)
- Complexity: Low -- straightforward recursive descent with a single guard check at top
- Modern idioms: Yes -- idiomatic Go map-as-set pattern, clean delete-after-processing for path tracking
- Readability: Good -- clear doc comments explain the ancestors set semantics and why nodes are removed after processing
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The parameter name `ancestors` is well-chosen over `visited` since it communicates the path-based (not permanent) semantics. No changes suggested.
