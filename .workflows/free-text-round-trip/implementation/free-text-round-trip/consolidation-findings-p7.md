# Consolidation Findings: free-text-round-trip (Phase 7)

No finding clears the floor on this phase's combined surface, and no spec
defect is owed. One comment correction is.

The bank was empty for this phase, so nothing was verdicted.

The five tasks touch three separate seams — the note query's ORDER BY
(`show.go`), the full-graph result type and its JSON key (`format.go`,
`dep_tree_graph.go`, the three formatters, `README.md`), the count-zero TOON
header (`toon_formatter.go`), and a deleted source-text guard
(`conformance_test.go`) — and they compose without leaving a seam. `go test
./...`, `go vet ./...`, `golangci-lint run ./...` and `gofmt -l` are all clean
on the assembled state.

Candidates considered and dropped, each with the test it failed:

- **`buildSeededTrees` still names its result and `BuildFullDepTree` still
  names its local slice `unrooted`** after 7-2 renamed the published field to
  `Trees` and 7-3 renamed the function (`dep_tree_graph.go:144`, `:225`,
  `:237`, `:241`). Names are never a finding (finding-floor.md → Duplication).

- **`emptyToonSection` derives the column list by reflection while
  `encodeToonSection` delegates the same derivation to toon-go**
  (`toon_formatter.go:337-344` against `:348-355`) — two derivations of one
  rule. Divergence is not silent: every count-zero header the helper produces
  is pinned as a literal in the suite (`toon_formatter_test.go:94`, `:106`,
  `:1046`, the fourteen `assertCountZeroSection` callers, `format_test.go:415`)
  and a column added, renamed or reordered fails those loudly. The helper is
  also not removable — `toon.MarshalString` of an empty typed slice emits
  `name[0]:` with no columns at all, so the library cannot produce the header
  §8 requires.

- **`encodeToonSection`'s marshal-error fallback returns `name[0]:`**
  (`toon_formatter.go:352`), a shape the phase's new helper could now supply
  correctly. The fallback is pre-existing, unreachable for the five concrete
  row structs, and switching it would not stop the header lying about the
  count. Names no failure.

- **A dangling blocker is a node with status `missing` in the full graph
  (7-3) but is dropped outright from the focused view and from `tick show`'s
  `blocked_by`** (`dep_tree_graph.go:92-95` in `walkUpstream`, `show.go:130`'s
  `JOIN tasks`). The drop and the full-vs-focused disagreement both predate
  this phase — the widened full-graph coverage landed in an earlier phase, and
  7-3 changed only the label the ghost node carries. Cause vs subject: the
  subject is wholly pre-existing code the phase sits next to.

- **`dep_tree_test.go:348` (`"it emits trees as an empty list when no task has
  dependencies"`, added by 7-2) duplicates the pre-existing
  `dep_tree_test.go:592` (`"it emits the emptied full document instead of a
  message"`)** — same fixture (`unconnectedTasks`), same format, same
  assertions, and the new one is the weaker of the two. A test file is in
  scope only for a failure-mode finding; duplicate coverage is not one.

- **The conformance inventory carries a two-task cycle but no dangling-blocker
  entry**, so the document shape 7-3 newly made distinct is decoded only by
  the dedicated tests in `dep_tree_test.go` and `dep_tree_graph_test.go`, not
  by the drivers. A coverage gap in a test file, not a failure-mode finding.

- **The `roots` → `trees` JSON key rename is a change to shipped output that
  the specification does not name.** Not raised: the analysis pass settled it
  explicitly — "no specification claim names `roots` — so the correction lands
  in the README and the shipped output, on the precedent the corrigendum of
  2026-09-20 set" (`analysis-tasks-c1.md:51`). Re-raising it would relitigate
  a decision already made and confirmed in this work unit.

## Comment Corrections

- `internal/cli/dep_tree_graph.go:154` — names the wrong algorithm.
  `countChains` builds an undirected adjacency list and walks it breadth-first
  (`dep_tree_graph.go:250-285`, and its own `// BFS to mark all reachable
  nodes` at `:272`); there is no union-find anywhere in the file. What is left
  of the line once the false half goes restates the call beneath it, which
  `countChains`'s own doc comment (`:244`) already carries.
  OLD: `	// Count connected components (chains) using union-find over participants`
  NEW:
