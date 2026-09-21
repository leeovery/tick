TASK: free-text-round-trip-8-1 (tick-9e2a3b) — The Full-Graph Edge List Carries One Row Per Stored Dependency

ACCEPTANCE CRITERIA:
- `dep_tree` carries exactly one row per stored dependency: its row count equals the total number of `BlockedBy` entries across all tasks, on every fixture in `internal/cli/dep_tree_test.go`.
- Two blockers converging above a further-blocked task emit the shared subtree once — Alpha/Alpha2 → Beta → Gamma returns `dep_tree[3]` with the Beta→Gamma row present once.
- A blocker ID no task record matches still reaches the edge list as the `from` of its dependency's row, and emits nothing extra when its walk overlaps an already-emitted tree.
- Row order is record order: tasks in stored order, each task's `BlockedBy` in stored order.
- `chains`, `longest` and `blocked` are unchanged on every fixture, and an empty project still emits `dep_tree[0]{from,to}:`.
- Pretty and JSON full-graph output is byte for byte what it is today, including the diamond's duplicated subtree; focused mode's `blocked_by` and `blocks` sections are unchanged.
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean.

STATUS: complete

SPEC CONTEXT: §8 ("Structured Output on Empty Branches"), specification.md:320-330, holds that the counts and the edge list never disagree and states the rule this task implements: "**The full-graph edge list is therefore the stored relation itself**: one row per recorded dependency, in record order, covering every participant by construction and repeating none. A walk cannot do this — it re-emits the whole subtree below every point two paths converge on, so three stored dependencies came back as four rows beside a summary reading one chain." The same section keeps the walk for the node-shaped renderings ("Pretty and JSON draw trees, where a task two things block appears under each of them — a drawing, not a duplicated fact"). The dated corrigendum recording this shipped-output change exists at specification.md:557 (2026-09-20), written by the orchestrator as the task prescribed.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/format.go:248-251` — `DepTreeEdge{From, To}`.
  - `internal/cli/format.go:258` — `Edges []DepTreeEdge` in `DepTreeResult`'s full-graph block; `format.go:249-255` documents it.
  - `internal/cli/dep_tree_graph.go:173` — `BuildFullDepTree` populates `Edges: collectStoredEdges(tasks)`; `Trees`, `ChainCount`, `LongestChain`, `BlockedCount` and `Message` are built exactly as before (commit f86873fe adds one struct field and one helper, nothing else in that file).
  - `internal/cli/dep_tree_graph.go:182-202` — `collectStoredEdges` → `collectScopedStoredEdges(tasks, nil)`: one `{From: dep, To: t.ID}` per `BlockedBy` entry, outer loop over `tasks` in slice order, inner loop over `t.BlockedBy` in stored order. Deduplication is impossible by construction; no walk, no seeding.
  - `internal/cli/toon_formatter.go:184-193` — `formatFullDepTree` renders `buildEdgeSection("dep_tree", toonEdgeRows(result.Edges))`; the `collectDownstreamEdges` walk over `result.Trees` is gone. `grep` over non-test `internal/cli` confirms `result.Trees` no longer feeds any toon path (only `Edges`, `BlockedByEdges`, `BlocksEdges` reach `buildEdgeSection`).
  - `README.md:321` — the closing claim was narrowed to the node renderings (later widened again by task 8-3 to cover the focused sections).
- Notes:
  - Record order is genuinely stored order: `RunDepTree` reads via `store.ReadTasks()` (`internal/cli/dep_tree.go:26`), which parses `tasks.jsonl` with no reordering (`internal/storage/store.go:151-170`).
  - Focused mode leaves `Edges` nil as the task required; `BuildFocusedDepTree` (`internal/cli/dep_tree_graph.go:337-363`) sets only its own fields. The focused sections later moved to the relation too under task 8-3 — that is its scope, not a drift in this one.
  - Pretty and JSON are untouched: commit f86873fe changes neither `pretty_formatter.go` nor `json_formatter.go`, and `jsonDepTreeFull` (`internal/cli/json_formatter.go:334-341`) carries `mode/trees/chains/longest/blocked` only, so `Edges` cannot leak into JSON.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/dep_tree_test.go:412` "it emits one edge per stored dependency where two blockers converge" — `convergentTasks` (3 stored deps) asserts exactly three rows with Beta→Gamma once, beside `chains: 1, longest: 2, blocked: 2`.
  - `internal/cli/dep_tree_test.go:429` "it emits no repeated edge when a dangling blocker sits above a chain" — `danglingAboveChainTasks` (4 stored deps, one blocker `tick-ghost1` no record matches, sitting above a real chain) asserts four rows, none repeated, with the ghost present as a `from`.
  - `internal/cli/dep_tree_test.go:365` cycle case and `:396` two-edge dangling chain — both repointed to record order.
  - `internal/cli/dep_tree_test.go:442` "it still duplicates the diamond in the pretty and JSON trees" — pins pretty byte for byte (`tick-ddd444` drawn under both `tick-bbb222` and `tick-ccc333`), the same duplication in the JSON tree, and four `dep_tree` rows for the diamond's four stored dependencies.
  - Empty branches: `:339` and `:352` assert the emptied document; `internal/cli/toon_formatter_test.go:1040-1053` pins the literal header `dep_tree[0]{from,to}:` alongside zero counts.
  - Row-count-equals-stored-dependencies holds on every fixture carrying a full-graph edge assertion: `cycleTasks` 2/2, `danglingBlockerTasks` 1/1, `danglingChainTasks` 2/2, `chainTasks` 2/2 (`:736`), `convergentTasks` 3/3, `danglingAboveChainTasks` 4/4, `diamondTasks` 4/4, `unconnectedTasks` and the empty project 0/0.
  - Conformance driver updated in step: `internal/cli/conformance_test.go:1034-1037` decodes the two-task-cycle document and asserts record order.
  - Unit-level toon assertions (`internal/cli/toon_formatter_test.go:911-1018`) now supply `Edges` rather than `Trees`, so they exercise the rendered field rather than the abandoned walk.
- Notes:
  - `assertToonEdgeRows` (`internal/cli/toon_decode_test.go:391-400`) fails on both count and per-row content in order, so a reintroduced duplicate or a reversion to walk order fails the suite loudly.
  - No over-testing: each subtest pins a distinct input class (convergence, dangling-above-chain, cycle, diamond, empty), and the diamond subtest is the only one asserting all three formats — justified, since it is the criterion that pretty and JSON did not move.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` "it does X" subtests and `t.TempDir`-backed fixtures, `DepTreeEdge` mirrors the existing `DepTreeTask` placement in `format.go`, and the toon row conversion reuses the established `buildEdgeSection`/`emptyToonSection` helpers so the count-zero header stays derived from the row struct's tags.
- SOLID principles: Good — derivation (`dep_tree_graph.go`) stays separate from rendering (`toon_formatter.go`); the formatter reads a field rather than re-deriving the graph.
- Complexity: Low — the derivation is two nested loops over data already in hand; it replaces a recursive walk, so the full-graph edge path got strictly simpler.
- Modern idioms: Yes.
- Readability: Good — `collectStoredEdges`/`collectScopedStoredEdges` state the ordering rule in their doc comment, and it matches the code.
- Issues: None. Comments in the changed code hold: `format.go:249-255` describes `Edges`, `BlockedByEdges` and `BlocksEdges` as they are built; `dep_tree_graph.go:186-188` describes the scoping parameter accurately.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean." — needs those four commands executed at the current HEAD; reading cannot settle a toolchain result.
