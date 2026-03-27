AGENT: architecture
FINDINGS:
- FINDING: walkDownstream/walkUpstream have no cycle guard -- stack overflow on corrupted data
  SEVERITY: medium
  FILES: internal/cli/dep_tree_graph.go:41, internal/cli/dep_tree_graph.go:63
  DESCRIPTION: Both walkDownstream and walkUpstream recurse without tracking visited nodes. While the domain layer (ValidateAddDep) prevents cycles during normal use, corrupted JSONL data (manual edit, import, partial write) containing a dependency cycle would cause unbounded recursion and a stack overflow crash. The doctor command can detect cycles but users are not required to run it before dep tree. countChains (line 200) correctly uses a visited set for the same graph -- the walkers should too.
  RECOMMENDATION: Add a visited (map[string]bool) parameter to walkDownstream and walkUpstream. When a node ID is already in the set, return nil (terminate that branch). This is a single-parameter addition to each function and mirrors the pattern already used in countChains. An alternative is to pass a depth counter and bail at a reasonable limit, but visited-set is cleaner since it also handles diamond dedup-free traversal correctly.

- FINDING: DepTreeResult is a union type discriminated by a nullable pointer field
  SEVERITY: low
  FILES: internal/cli/format.go:165
  DESCRIPTION: DepTreeResult serves both full-graph mode (Roots/Summary/ChainCount/LongestChain/BlockedCount populated) and focused mode (Target/BlockedBy/Blocks populated), discriminated by whether Target is nil. Every FormatDepTree implementation checks `result.Target != nil` to branch. This is functional but fragile -- callers must understand that certain fields are meaningless in each mode, and a constructor that populates both Target and Roots would produce undefined rendering behavior.
  RECOMMENDATION: No immediate change needed given the small surface area (two callers: BuildFullDepTree and BuildFocusedDepTree). If this pattern spreads, consider splitting into two result types and making FormatDepTree accept an interface or two methods. For now, a code comment documenting the mutual exclusivity of the two field groups would suffice.

SUMMARY: The implementation is well-structured with clean separation between graph building, command dispatch, and formatting. The one actionable concern is the lack of cycle protection in the recursive tree walkers, which would crash on corrupted data. The union-type result struct is a minor design note that works fine at current scale.
