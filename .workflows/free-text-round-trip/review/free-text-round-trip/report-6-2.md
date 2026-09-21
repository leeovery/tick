TASK: free-text-round-trip-6-2 — Stats, Dependency-Tree And Detail Documents Join The Table (tick-f5455a)

ACCEPTANCE CRITERIA:
- `tick stats --toon` decodes on a populated project and on an empty one, the empty one carrying all-zero counts and `by_priority`'s five rows
- `tick dep tree --toon` decodes on a populated graph, on a project with no dependencies, and on a two-task cycle
- The cycle document decodes with an empty `dep_tree` list beside the counts the builder produced, not forced zeros
- `tick dep tree <id> --toon` decodes for all four combinations of upstream and downstream edges, each carrying top-level `id`, `title` and `status`
- The no-dependencies focused document decodes with both `blocked_by` and `blocks` as empty lists
- `tick show --toon` decodes for a task carrying every optional field and for one carrying none
- The `dep tree --quiet` entry is present in the inventory with a stated reason and is not decoded
- Phase 1's `TestToonTaskDetailConformance` and Phase 3's stats and dep-tree decoded-value subtests still exist and pass unmodified
- Every entry added by this task is a document in the inventory, not an exemption
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT: §11.1 requires every structured command's output to be decoded by a real TOON reader, counted in documents rather than commands — "every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces". §3.1 lists stats, `dep tree` and `show` among the must-parse commands. §8 replaces the two dep-tree prose branches with the emptied document (count-zero edge section beside the summary counts) and — decisively for this task — rules that the full-graph edge list is the stored relation itself, "one row per recorded dependency, in record order, covering every participant by construction", precisely so a cycle cannot report an empty edge list beside non-zero `chains`/`blocked`. §5.2 fixes the two `show` shapes: `children`, `blocked_by` and `notes` always present, `type`/`parent`/`closed`/`tags`/`refs`/`description` only when carried.

IMPLEMENTATION:
- Status: Implemented (one criterion delivered differently from its wording — see Notes)
- Location:
  - Inventory entries, `internal/cli/conformance_test.go`: stats 268, 276; full `dep tree` 284, 292, 300; focused `dep tree` 308, 316, 324, 332; `dep tree --quiet` exemption 340; `show` 345, 353 — eleven documents plus the one `NotADocument`, exactly the outcome the task states.
  - Seeds: `conformanceStatsTasks` 68, `conformanceDepGraph` 85, `conformanceUnconnectedTasks` 95, `conformanceCyclePair` 102, `conformanceDetailTasks` 118.
  - Value assertions: `TestToonStatsConformance` 973, `TestToonDepTreeConformance` 1005, `TestToonDetailConformance` 1099, `assertConformancePriorityCounts` 1307.
  - Original delivery: commit d184e25f (conformance_test.go only, +373 lines).
- Notes:
  - The cycle criterion ("decodes with an empty `dep_tree` list beside the counts the builder produced") is delivered as an assertion of two real edges — `internal/cli/conformance_test.go:1031-1043` expects `{from tick-b88888, to tick-a99999}` then `{from tick-a99999, to tick-b88888}` beside `chains: 1, longest: 2, blocked: 2`. That is the spec, not drift away from it: §8 makes the full-graph edge list the stored relation (`collectStoredEdges` / `collectScopedStoredEdges`, `internal/cli/dep_tree_graph.go:182-203`), so a cycle emits one row per stored `BlockedBy` entry in record order. The plan's wording predates that ruling; the criterion's own rationale cites zero *roots*, which remains true — `BuildFullDepTree` finds no root for the cycle and falls back to seeding a tree (`internal/cli/dep_tree_graph.go:117-180`). The intent — assert what the builder produced, never forced zeros — is met, and §8's emptied full-graph form is still covered by the separate no-dependencies entry (284 is the populated one; the emptied entry is 292, asserted at `internal/cli/conformance_test.go:1020-1029`). No loss.
  - The grep-based guard `TestConformanceScopeBoundary`, added by this task to pin the phase 1/3 test names as source text, was deleted later by commit a8305929 (task 7-5 corrections). The tests it guarded all still exist: `TestToonTaskDetailConformance` (`internal/cli/toon_decode_test.go:184`), `TestToonDepTreeFocusedConformance` (`internal/cli/toon_decode_test.go:402`), `internal/cli/toon_formatter_test.go:335` and `:977`, `internal/cli/stats_test.go:290`, `internal/cli/dep_tree_test.go:352`. Deleting a test that asserted on source text rather than behaviour is a sound correction, and the criterion it served is satisfied directly.
  - Asserted values check out against the producing code: `ToonFormatter.FormatStats` always emits five `by_priority` rows (`internal/cli/toon_formatter.go:123-127`), so the empty-project branch carries them; `RunStats` derives `Blocked = (Open + InProgress) − Ready` (`internal/cli/stats.go:85`), giving the asserted `ready: 2, blocked: 1` for the five-task seed; `formatFullDepTree` and `formatFocusedDepTree` always emit their edge sections, count-zero when empty (`internal/cli/toon_formatter.go:182-223`), so both §8 branches are documents rather than prose; the focused edge order asserted at `internal/cli/conformance_test.go:1045-1096` is what `collectScopedStoredEdges` produces over `neighbourhoodIDs` (`internal/cli/dep_tree_graph.go:359-369`). Seeds are written to JSONL in slice order and read back in file order (`store.ReadTasks`, `internal/cli/dep_tree.go:25`), so the ordered edge assertions are deterministic despite every fixture sharing `conformanceTime`.
  - `dep tree --quiet` (340) is declared with the reason "--quiet prints nothing at all rather than a document", matching `RunDepTree`'s early return before the store is opened (`internal/cli/dep_tree.go:15-17`).
  - Every helper this task added is still reachable from a call site; nothing was orphaned when the scope-boundary test was later removed (its `os`, `path/filepath` and `testutil` imports went with it).

TESTS:
- Status: Adequate
- Coverage: All eleven branches are both driven by `TestToonOutputConformance` (parseability, `internal/cli/conformance_test.go:694`) and asserted by value. Stats: seven counts plus five priority rows on both branches. Full dep tree: populated edges + counts, emptied edges + zero counts, cycle edges + the builder's non-zero counts. Focused: all four upstream/downstream combinations, each asserting top-level `id`/`title`/`status`; `assertToonRowsEmpty` (`internal/cli/toon_decode_test.go:175`) goes through `toonRows`, which fatals on a missing key, so "empty list" is asserted as present-and-empty rather than merely absent — which is what §8 requires. Detail: the full document asserts every optional key with its stored value; the bare document asserts `type`/`parent`/`closed`/`tags`/`refs`/`description` absent and `blocked_by`/`children`/`notes` present and empty, which is §5.2's pair exactly.
- Notes: The new value assertions overlap the phase 1/3 per-document tests substantially — `TestToonDetailConformance`'s two subtests (`internal/cli/conformance_test.go:1100-1134`) restate `TestToonTaskDetailConformance`'s two (`internal/cli/toon_decode_test.go:216-255`) against a different seed, its four focused subtests restate `TestToonDepTreeFocusedConformance`'s four (`internal/cli/toon_decode_test.go:411-461`), and the empty-stats subtest restates `internal/cli/stats_test.go:290`. That overlap is what the plan directed (its Edge Cases keep both layers deliberately, and its Tests section specifies value-level assertions for the new ones), so it is recorded here rather than raised as a finding. Nothing is under-tested, and no assertion reaches into implementation detail.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests in "it does X" form, `t.Helper()` on every helper, `t.TempDir()` isolation via `setupTickProject`/`setupTickProjectWithTasks`, seeds as package-level constructors alongside the existing ones.
- SOLID principles: Good — each seed constructor builds one fixture shape; entries carry data only and the driver owns execution.
- Complexity: Low — `assertConformancePriorityCounts` is the only new helper and it is a flat loop over five rows.
- Modern idioms: Yes — `[5]int` fixed array for the priority expectation, `append` over the shared graph in `conformanceDetailTasks`, no redundant conversions.
- Readability: Good — entry names state the branch they cover, and each `Setup` returns a seeded directory plus the exact argv.
- Comment accuracy: `conformanceDetailTasks`'s doc comment (`internal/cli/conformance_test.go:115-117`) holds against the code — `full.Parent` is the graph's `tick-a11111` and `full.BlockedBy` its `tick-c33333`. No comment in the added code references a task id, phase or spec section.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — requires running the toolchain; reading confirms only that the added code is well-formed Go consistent with the package's existing style.
- "Phase 1's `TestToonTaskDetailConformance` and Phase 3's stats and dep-tree decoded-value subtests still exist and pass unmodified" — existence is settled by reading (all six named tests located, and commit d184e25f touched `internal/cli/conformance_test.go` alone, so this task modified none of them); that they pass needs a suite run.
- The eight decode criteria (stats both branches, full dep tree three branches, focused four branches, `show` two branches) — reading settles that each branch is an inventory entry driven by the toon decoder and that every asserted value matches what the builders and `ToonFormatter` produce; that each document actually parses under `toon.DecodeString` is settled only by running `TestToonOutputConformance`, `TestToonStatsConformance`, `TestToonDepTreeConformance` and `TestToonDetailConformance`.
