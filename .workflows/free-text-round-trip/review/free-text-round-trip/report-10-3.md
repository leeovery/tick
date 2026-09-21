TASK: free-text-round-trip-10-3 (tick-9502f1) — One Enumeration Of The Stored Dependency Edges

ACCEPTANCE CRITERIA:
- Only one function in `internal/cli/dep_tree_graph.go` ranges over `tasks` emitting `DepTreeEdge`; `collectStoredEdges` carries no loop of its own.
- A nil scope keeps every edge rather than none: over a project where every task carries a blocker, `tick dep tree` still reports one row per stored dependency, with `dep_tree[N]`'s count matching the rows beneath it.
- `tick dep tree` and `tick dep tree <id>` produce byte-identical output against a binary built before the change, under `--toon`, `--json` and `--pretty`, over a cycle, a diamond and a graph with a dangling blocker.
- Row order is unchanged: tasks in slice order, each task's blockers in stored order, on both views.
- No test file is changed; the diff is confined to `internal/cli/dep_tree_graph.go`.
- `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied.

STATUS: complete

SPEC CONTEXT: §8 ("Structured Output on Empty Branches") fixes that the full-graph edge list is the stored relation itself — "one row per recorded dependency, in record order, covering every participant by construction and repeating none" (specification.md:326) — while the node-shaped renderings (pretty, JSON) still walk (:328). The corrigendum of 2026-09-20 at specification.md:561 extends the same rule to the focused form: "The focused sections are now the relation too, scoped to the neighbourhood the walk reaches: one row per stored dependency among the target's transitive blockers and among its transitive dependents, in record order, neither repeating", and records the divergence this task removes by construction — five stored dependencies returning five rows from the full graph and six from the focused view. It also fixes the deliberate asymmetry the scope filter must preserve: a blocker naming no task reaches the full-graph list but never a focused section.

IMPLEMENTATION:
- Status: Implemented
- Location: `internal/cli/dep_tree_graph.go:182-184` (`collectStoredEdges` reduced to `return collectScopedStoredEdges(tasks, nil)`), `:186-203` (`collectScopedStoredEdges`, the single enumeration), `:173` (sole call site of `collectStoredEdges`), `:359-360` (the two focused call sites, unchanged, passing `neighbourhoodIDs(...)`), `:365-369` (`neighbourhoodIDs`, unchanged, always returning a non-nil map seeded with the target).
- Notes: The nil-scope reading is done by short-circuit in the `inScope` closure (`:190`, `return ids == nil || ids[id]`), so a nil map is never indexed for a membership decision — the failure mode the task's step 1 called out (a nil-map lookup returning false and dropping every edge) cannot occur. Row order is untouched: the outer range is over `tasks` in slice order and the inner over `t.BlockedBy` in stored order (`:192-200`), matching specification.md:326 and :561. The focused asymmetry still holds: `neighbourhoodIDs` is built from `collectTreeIDs` over the walked nodes, and `walkUpstream`/`walkDownstream` skip IDs no task record matches (`:82-85`, `:57-60`), so a dangling blocker is never a member and contributes no focused row, while the nil-scope full graph still carries it. I read the whole of `dep_tree_graph.go`: `collectScopedStoredEdges` is the only function in the file that constructs a `DepTreeEdge`, and `collectStoredEdges` carries no loop. Nothing downstream re-enumerates: `ToonFormatter.formatFullDepTree`/`formatFocusedDepTree` (`internal/cli/toon_formatter.go:182-207`) map `result.Edges` / `result.BlockedByEdges` / `result.BlocksEdges` straight through `toonEdgeRows`, and the JSON formatter renders trees rather than edges (`internal/cli/json_formatter.go:380-388`).

TESTS:
- Status: Adequate
- Coverage: Correct for a behaviour-preserving refactor — no new test, existing suite as the check. The nil-scope path is exercised end to end over exactly the shapes that would expose a dropped or duplicated edge: a cycle where every task carries a blocker (`internal/cli/dep_tree_test.go:365`), a dangling blocker (`:381`), a two-edge dangling chain (`:396`), a convergence point (`:412`), a dangling blocker above a chain (`:429`) and a diamond (`:442`, asserting four `dep_tree` rows). The scoped path is held by `:483` (focused `blocks` over a diamond-with-tail), `:506` (focused `blocked_by` in record order), `:520` (both cycle edges in each focused section) and `:533` (no focused row for a blocker no task record matches — the asymmetry specification.md:561 fixes). `:497` asserts the two views agree row-for-row over the same store, which is the invariant this task makes structural. Unit coverage of the builders sits in `internal/cli/dep_tree_graph_test.go:23` (`TestBuildFullDepTree`) and `:422` (`TestBuildFocusedDepTree`).
- Notes: If the nil scope had been read as an empty map, `internal/cli/dep_tree_test.go:365`, `:396`, `:412` and `:442` would all fail on an emptied `dep_tree` section, so the refactor's one real hazard is observed. Section counts cannot disagree with the rows beneath them by construction: the headers are written by the library from the row slice (`buildEdgeSection`/`encodeToonSection`, `internal/cli/toon_formatter.go:218-223`), and the assertions decode with a real TOON reader (`decodeToonDoc`, `internal/cli/toon_decode_test.go:17`) rather than comparing strings. No redundancy introduced by this task.

CODE QUALITY:
- Project conventions: Followed. Unexported helper, no doc comment on the one-line delegation and the rule stated once on the enumeration — consistent with `.claude/skills/golang-documentation` ("document complex internal functions"), and the surviving comment carries intent (what a nil scope means) rather than restating the loop.
- SOLID principles: Good — one function owns the enumeration rule; the scope predicate is the only variation point.
- Complexity: Low. One nested range plus a two-term predicate.
- Modern idioms: Yes.
- Readability: Good. `inScope` names the filter, and the nil case reads as "no scope" at the point of decision.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`tick dep tree` and `tick dep tree <id>` produce byte-identical output against a binary built before the change, under `--toon`, `--json` and `--pretty`, over a cycle, a diamond and a graph with a dangling blocker." — requires building the pre-change commit and diffing both commands' stdout in all three formats over the three fixtures; reading establishes that a nil scope keeps every edge in the same order, but not byte-identity against the prior binary.
- "No test file is changed; the diff is confined to `internal/cli/dep_tree_graph.go`." — requires inspecting the task's commit (`git show` over the commit carrying tick-9502f1); no test file in the change-set references `collectStoredEdges` or `collectScopedStoredEdges`, but the diff's confinement is a history fact reading the working tree cannot settle.
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied." — requires running the toolchain.
