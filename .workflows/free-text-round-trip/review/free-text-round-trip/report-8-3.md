TASK: free-text-round-trip-8-3 (tick-b845fe) — The Focused Dependency View's Edge List Is Also The Relation

ACCEPTANCE CRITERIA:
1. `tick dep tree <id> --toon` emits in `blocks` one row per stored dependency whose blocker and blocked task are both reached by the downstream walk from the target (target included), and in `blocked_by` one per stored dependency whose endpoints are both reached by the upstream walk; rows in record order (tasks in slice order, each task's `BlockedBy` in stored order), no row repeating within a section.
2. On the reproducer (A unblocked; B, C blocked by A; D blocked by B and C; E blocked by D), `tick dep tree tick-aaa111 --toon` returns five `blocks` rows with `tick-ddd444,tick-eee555` once — the same five rows the full graph returns.
3. A cycle inside the neighbourhood contributes every one of its stored dependencies, including the edge the walk suppresses as a re-entry; with the target in a two-task cycle both sections carry both dependencies, each once.
4. A blocker ID no task record matches contributes no row to either focused section.
5. Focused pretty and JSON output unchanged byte for byte; the diamond's shared branch still drawn under each blocker.
6. `grep -rn 'collectUpstreamEdges\|collectDownstreamEdges' internal/` returns nothing.
7. `README.md`'s `dep tree` paragraph no longer states unscoped a rule only the full graph kept.
8. No file under `.workflows/` is edited.
9. `go build ./...`, `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd`, `golangci-lint run ./...` are clean.

STATUS: complete

SPEC CONTEXT: §8 holds that "the counts and the edge list never disagree" and that the full-graph edge list "is therefore the stored relation itself": one row per recorded dependency, in record order, covering every participant by construction and repeating none, because "a walk cannot do this — it re-emits the whole subtree below every point two paths converge on". §1's bar is output an agent can read without a rule learned outside it. The specification carries the matching record at specification.md:561 (Corrigendum 2026-09-20): the focused sections are now the relation too, scoped to the neighbourhood the walk reaches, node trees for pretty and JSON untouched, with the two deliberate boundaries (a cycle inside the neighbourhood contributes every stored dependency; a blocker naming no task contributes no focused row) and the accepted asymmetry that the full graph additionally carries dangling blockers.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/format.go:268-269 — `BlockedByEdges []DepTreeEdge` and `BlocksEdges []DepTreeEdge` added beside the focused node trees; the type doc at format.go:252-255 states the new rule.
  - internal/cli/dep_tree_graph.go:189-202 — `collectScopedStoredEdges` iterates `tasks` in slice order and each task's `BlockedBy` in stored order, keeping an edge only when both endpoints are in `ids` (nil `ids` = no scope, which is how `collectStoredEdges` at :182-184 now reaches it after Tfree-text-round-trip-10-3's fold).
  - internal/cli/dep_tree_graph.go:355-363 — `BuildFocusedDepTree` populates both fields from `neighbourhoodIDs(targetID, blockedBy)` and `neighbourhoodIDs(targetID, downstream)`; `neighbourhoodIDs` at :365-369 seeds the set with the target and adds the tree's IDs via `collectTreeIDs`.
  - internal/cli/toon_formatter.go:203-204 — `formatFocusedDepTree` renders both sections through `buildEdgeSection`/`toonEdgeRows` (:209-215), the same path `formatFullDepTree` uses at :183, so an empty section still emits its count-zero header via `buildEdgeSection` (:218-223).
  - `collectUpstreamEdges`/`collectDownstreamEdges` deleted; `grep -rn 'collectUpstreamEdges\|collectDownstreamEdges' internal/` returns nothing (criterion 6 met).
  - README.md:321 — sentence now reads "the toon edge lists carry one row per stored dependency — `dep_tree` over the whole graph, and the focused `blocked_by`/`blocks` sections over the dependencies inside the target's neighbourhood" (criterion 7 met).
  - Commit e7bb74da touched README.md, internal/cli/{conformance_test.go, dep_tree_graph.go, dep_tree_test.go, format.go, toon_decode_test.go, toon_formatter.go, toon_formatter_test.go} — nothing under `.workflows/` (criterion 8 met).
- Notes: criteria 1-4 verified by reading the two walks against the scoping. `walkDownstream`/`walkUpstream` (dep_tree_graph.go:43-100) build each node before recursing and only truncate children at a re-entry, so a task reachable in a direction is always in the tree's ID set even when its outgoing edge is suppressed — which is why a cycle through the target yields both of its stored dependencies in each section (criterion 3). Both walks `continue` past an ID absent from `taskIdx`, so a dangling blocker never enters the neighbourhood set and its edge is dropped by `inScope` (criterion 4); the rationale attached to that criterion also holds, since `show`'s blocked_by query INNER JOINs `tasks` (internal/cli/show.go:136-139) and drops the same ID. `RunDepTree` reads via `store.ReadTasks()` (internal/cli/dep_tree.go:25), so "record order" is genuine JSONL order. No row can repeat: `dep add` refuses a blocker already in the list (internal/cli/dep.go:104). Criterion 5 holds by construction — `grep -rn 'BlockedByEdges\|BlocksEdges' internal/` shows the new fields read only at toon_formatter.go:203-204; `PrettyFormatter.formatFocusedDepTree` (pretty_formatter.go:397-417) and `JSONFormatter.formatFocusedDepTreeJSON` (json_formatter.go:390-401) read only the node trees.

TESTS:
- Status: Adequate
- Coverage: internal/cli/dep_tree_test.go:483-495 (five `blocks` rows in record order on the `diamondTailTasks` fixture, `tick-ddd444,tick-eee555` once), :497-504 (focused `blocks` rows equal to the full graph's `dep_tree` rows, via the new `decodeToonEdgeRows` helper at toon_decode_test.go:376-388), :506-518 (focused `blocked_by` on E in record order rather than walk order), :520-531 (both cycle edges in both sections, each once), :533-541 (no row for `tick-ghost1`), :543-574 (focused pretty output byte-for-byte plus the JSON diamond branch under each blocker). internal/cli/toon_formatter_test.go:1151-1181 replaces the retired walk-rule case with one whose node trees repeat a branch and whose `BlocksEdges` does not, asserting the edge fields are what renders. The count-zero focused cases (toon_formatter_test.go:1085-1108) and the both-directions case (:1060-1070) populate the new fields alongside their trees. internal/cli/conformance_test.go:1065-1068 now expects the upstream-only rows in record order.
- Notes: `assertToonEdgeRows` (toon_decode_test.go:391-400) asserts both the exact row count and per-row order, so each of these would fail if the section reverted to the walk. No redundancy found: the full-graph diamond case at dep_tree_test.go:442 covers a different mode from the focused case at :543, and the cross-check at :497 asserts the agreement property criterion 2 names rather than restating the explicit-row test at :483.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` "it does X" subtests, `t.TempDir`-backed fixtures via `setupTickProjectWithTasks`, `t.Helper()` on the new decode helper; unexported graph helpers stay in `internal/cli/dep_tree_graph.go` beside their peers.
- SOLID principles: Good — the graph builder owns the derivation, the formatter only renders what the result carries; `collectScopedStoredEdges` generalised the existing collector rather than growing a parallel one.
- Complexity: Low — one guarded loop over tasks and their blockers; the scoping is a set membership test.
- Modern idioms: Yes.
- Readability: Good — `neighbourhoodIDs` names the concept the criteria use, and `inScope` makes the nil-means-unscoped contract read at the point of use.
- Comment accuracy: the `DepTreeResult` doc (format.go:252-255), the `collectScopedStoredEdges` doc (dep_tree_graph.go:186-188) and the `FormatDepTree` doc (toon_formatter.go:170-173) all hold against the code; no comment anywhere still describes the focused sections as derived from a walk.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go build ./...`, `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean." — settled only by running those five commands; nothing in reading contradicts them, but verification here is read-only.
