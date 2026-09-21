TASK: free-text-round-trip-7-2 — The Full-Graph JSON Document Names Its Key For What It Holds

ACCEPTANCE CRITERIA:
1. `tick dep tree --json` in full-graph mode returns exactly the keys `mode`, `trees`, `chains`, `longest`, `blocked`, and no shipped output carries a key named `roots`.
2. On a two-task cycle, `trees` carries the cycle's participants; on a task blocked by an ID no record matches, `trees` carries that participant's tree — neither under a key claiming rootness.
3. On a project where no task has dependencies, `trees` is `[]` rather than `null` and `chains`, `longest`, `blocked` read zero.
4. `grep -rn 'Unrooted\|fullGraphTrees' internal/cli` returns nothing; `DepTreeResult` publishes one tree field and `BuildFullDepTree` keeps the root-versus-seeded distinction local.
5. Toon and pretty `dep tree` output is byte-identical to before the change on the rooted, cycle and dangling-blocker projects.
6. `jsonConformanceListKeys` carries `trees` in place of `roots`, and the JSON conformance driver decodes the full dep-tree document with no null-list problem.
7. `README.md` names no JSON `roots` array, and `"it reproduces the README dep tree samples"` still passes.

STATUS: issues_found

SPEC CONTEXT: §8 of the specification (and its corrigenda of 2026-09-19 and 2026-09-20) governs the full-graph dep-tree document: every participant is covered in all three formats, the counts and the edge list never disagree, and the document must be readable without a decoding rule learned outside it (§1). The spec names no JSON key called `roots` — it describes the node-shaped renderings (pretty, JSON) as walks that draw roots first and then seed from participants no drawn tree reaches. The rename is therefore a correction landing in shipped output and the README, on the precedent of the `blocked_by` → `blocker` rename recorded in the corrigendum of 2026-09-20.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/format.go:257` — `DepTreeResult` publishes the single `Trees []DepTreeNode`; `Roots`/`Unrooted` and the `fullGraphTrees()` helper are gone (commit 777f67e3 deleted them along with the now-unused `slices` import in that file).
  - `internal/cli/format.go:249-254` — the type's doc comment describes the single field ("Trees holds the drawn trees") and holds true against it.
  - `internal/cli/dep_tree_graph.go:124,142,154,172` — `roots` and `unrooted` stay locals inside `BuildFullDepTree`; the returned field is `slices.Concat(roots, unrooted)`.
  - `internal/cli/json_formatter.go:337` — `Trees []jsonDepTreeNode` with tag `json:"trees"`; `:383` feeds `result.Trees`; the `FormatDepTree` doc comment at `:369-371` lists `{mode, trees, chains, longest, blocked}`.
  - `internal/cli/pretty_formatter.go:376,381` — reads `result.Trees`.
  - `internal/cli/toon_formatter.go` — no longer reads the field at all (a later task, 10-3, moved the full-graph toon section to the stored `Edges` relation); the commit's own change pointed it at `result.Trees`.
  - `internal/cli/conformance_test.go:703` — `jsonConformanceListKeys` carries `"trees"`.
  - `README.md:321` — the clause naming "the JSON `roots` array" is dropped; the sentence keeps its statement that all three formats cover every participant.
- Notes:
  - Criterion 4 verified by search: `grep -rn 'Roots\|Unrooted\|fullGraphTrees' internal/cli` returns exactly one hit, `dep_tree_graph.go:123`, a comment describing the local. `grep -rn 'roots' --include='*.go' --include='*.md'` outside `.workflows` returns only local variables, comments, subtest names and `dep_tree_test.go:581-582` (the assertion that the key is absent) — no shipped output carries `roots`.
  - Deliberate divergence from the plan's "Do" list, and a sound one: the plan asked that the `longest` fold and the empty-answer condition keep reading `roots` and `unrooted` separately; the implementation folds both over `trees` (`dep_tree_graph.go:155-157`, `:167`). `trees` is exactly `slices.Concat(roots, unrooted)`, so `max` over it and `len(trees) == 0` are identical to the two-slice forms. No loss.
  - `buildUnrootedTrees` was renamed to `buildSeededTrees` in the same commit — not in the task's "Do" list, but it is the same mislabel (the function's output is no longer "unrooted" in any meaningful sense) and its doc comment at `dep_tree_graph.go:240-245` holds true against the current body.

TESTS:
- Status: Adequate, with one redundant subtest
- Coverage:
  - Criterion 1: `dep_tree_test.go:581-583` asserts the `roots` key is absent from the live CLI document; `json_formatter_test.go:1396` pins the full-graph key set as `{mode, trees, chains, longest, blocked}`. The five-field struct at `json_formatter.go:335-341` makes the "exactly" half structural.
  - Criterion 2: `"it carries a cycle's participants under trees"` (`dep_tree_test.go:610`) asserts A → B → A under `trees` plus `chains:1, longest:2, blocked:2`; `"it marks a blocker no task carries as missing in JSON"` (`dep_tree_test.go:628`) is the retargeted dangling-blocker test the plan named (renamed by the later `missing`-status work) and reads the ghost's tree from `trees`.
  - Criterion 3: `"it emits trees as an empty list when no task has dependencies"` (`dep_tree_test.go:591`) and `"it emits the emptied full document instead of a message"` (`dep_tree_test.go:872`), plus the formatter-level `"it renders trees as [] not null when empty"` (`json_formatter_test.go:1117`).
  - Criterion 4: `"it terminates full graph with circular dependency"` (`dep_tree_graph_test.go:726`) is re-expressed over the single field — it now asserts the cycle's one seeded tree (A with child B) rather than the meaningless `Roots = 0`, keeping the termination the subtest exists for. Every `result.Roots` read in `dep_tree_graph_test.go` follows the rename: the file now carries 43 `result.Trees` occurrences and no `Roots` or `Unrooted`.
  - Criterion 5: toon and pretty expectations in `toon_formatter_test.go` and `pretty_formatter_test.go` are unchanged apart from the literal field name, which is what byte-identical output looks like at the test surface.
  - Criterion 6: `conformance_test.go:703` carries `trees`; `jsonMemberProblem` (`:773-788`) applies the non-null list rule to it, and `TestJSONOutputConformance` (`:1692`) drives the three full-graph `dep tree` inventory entries (`:284`, `:292`, `:300`).
- Notes: the two CLI-level empty-graph subtests are the same test twice — see FINDINGS. No over-mocking; fixtures are the shared `chainTasks`/`cycleTasks`/`danglingBlockerTasks`/`unconnectedTasks` helpers.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.Helper()` on the renamed `jsonDepTreeOnlyTree` helper, `slices.Concat`/`max` per the modernize lint set.
- SOLID principles: Good. Collapsing the published pair removes a distinction no consumer spent while keeping it where it is still needed — local to the one builder that uses it.
- Complexity: Low. The change is a field collapse plus a rename; `BuildFullDepTree` loses one branch (`len(roots) == 0 && len(unrooted) == 0` → `len(trees) == 0`).
- Modern idioms: Yes.
- Readability: Good. The key now claims only what the value holds, which was the point.
- Comment accuracy: Holds. `format.go:249-254`, `json_formatter.go:369-371` and `dep_tree_graph.go:240-245` all describe the current code; the stale `pretty_formatter.go` line "Supports both full-graph mode (Roots populated)" was deleted rather than left to rot.
- Issues: none beyond the test duplication below. (`buildSeededTrees` still names its accumulator `unrooted` at `dep_tree_graph.go:247` — a naming leftover, not reported.)

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/dep_tree_test.go:591-608 — the added subtest `"it emits trees as an empty list when no task has dependencies"` is a strict subset of the pre-existing sibling `"it emits the emptied full document instead of a message"` (`dep_tree_test.go:872-897`): same fixture (`unconnectedTasks(now)`), same command (`runDepTreeJSON(t, dir)`), and every assertion it makes — `trees` decodes to a non-nil `[]any`, `len(trees) == 0`, `chains`/`longest`/`blocked` are `0` — is already made there, which additionally pins `mode == "full"` and the absence of `message`. Delete the subtest at `:591-608`; criterion 3 stays covered by `:872` and by the formatter-level `"it renders trees as [] not null when empty"` (`json_formatter_test.go:1117`). — FAILS: the subtest cannot fail unless `:872` fails with it, so it buys no case; any future change to the emptied full-graph document produces two identical failures to triage and two sites to edit for one behaviour.

UNSETTLED:
- "`README.md` names no JSON `roots` array, and `"it reproduces the README dep tree samples"` still passes" — the first half is settled by reading (no occurrence of `roots` survives in `README.md`, verified by search). The second half names a test run: `readme_samples_test.go:382-387` reproduces only the pretty and toon fenced samples, and the commit's README edit is prose outside any fence, so reading found nothing that would break it — but only running `go test ./internal/cli -run TestREADMESamples` settles it.
