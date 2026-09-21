TASK: free-text-round-trip-3-4 (tick-78558c) — Nothing Blocked Anywhere Returns The Emptied Document

ACCEPTANCE CRITERIA:
- [x] `tick dep tree` toon output on an empty project decodes to a document with an empty `dep_tree` list and `chains`, `longest`, `blocked` all reading `float64(0)`
- [x] `tick dep tree` toon output on a project whose tasks carry no dependencies decodes the same way
- [~] `tick dep tree` toon output on a project that is a single dependency cycle decodes with an empty `dep_tree` list and the counts the builder produced — `blocked` reading 2 and `chains` reading 1 for a two-task cycle, not forced zeros (counts met; the "empty `dep_tree`" half was deliberately superseded later in the same plan — see IMPLEMENTATION notes)
- [x] `tick dep tree --pretty` stdout is byte-identical to before this task on both empty branches: `No dependencies found.` followed by one newline
- [~] `tick dep tree --json` output is byte-identical to before this task (superseded by task 3-5, which the criterion itself names as the task that removes the JSON message guard)
- [x] `tick dep tree --quiet` prints nothing
- [x] `grep -n 'FormatMessage' internal/cli/dep_tree.go` returns nothing
- [x] `ToonFormatter.FormatDepTree` carries no `result.Message` branch
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean — not settleable by reading, see UNSETTLED

STATUS: complete

SPEC CONTEXT: §3.2 draws the line this task lands on — "a confirmation may be prose; an answer to a query may not. 'No dependencies' is not a confirmation — it is the answer the caller ran the command to find out" (specification.md:78). §4.3 (specification.md:116-118) locates the sentence in the shared graph builder rather than in any formatter, and rules that the change belongs in the handler because `runFullDepTree` returned before any dep-tree formatter was reached, with the consequence that pretty must be handed its sentence explicitly or it prints a blank line. §8 (specification.md:322) requires the agent formats to answer with the emptied document on both empty branches while pretty keeps its sentence. Two later corrigenda (specification.md:547, :555) extend the full-graph document to cover participants no root reaches — a cycle, a blocker naming no task — and fire the sentence "only when the graph holds no participant at all"; that is the source of the two drifts noted below.

IMPLEMENTATION:
- Status: Implemented (with two deliberate, in-plan drifts from the task's literal wording)
- Location:
  - internal/cli/dep_tree.go:37-40 — `runFullDepTree` is now two lines: `fmtr.FormatDepTree(BuildFullDepTree(tasks))` into `printDocument`. No early return, no `FormatMessage` anywhere in the file (read in full, 59 lines).
  - internal/cli/pretty_formatter.go:375-378 — `formatFullDepTree` returns `result.Message` for zero trees, so pretty's stdout is unchanged where the handler used to print `FormatMessage(result.Message)`; `printDocument` (internal/cli/helpers.go:36-42) supplies the single trailing newline via `Fprintln`, giving exactly `"No dependencies found.\n"`.
  - internal/cli/toon_formatter.go:174-193 — `FormatDepTree` branches only on `result.Target`; `formatFullDepTree` emits the `dep_tree` edge section plus `chains`/`longest`/`blocked`. No `result.Message` branch remains.
  - Commit 9995f8dc touches exactly the four files the Do list names (dep_tree.go, pretty_formatter.go, toon_formatter.go, dep_tree_test.go).
- Notes:
  - Drift on the cycle criterion: a two-task cycle no longer decodes with an empty `dep_tree`. `BuildFullDepTree` now emits one row per stored dependency (`collectStoredEdges`, internal/cli/dep_tree_graph.go:173,182-203) and seeds trees for participants no root reaches (`buildSeededTrees`, internal/cli/dep_tree_graph.go:246-279). Both are later tasks of this same plan (3-7 "The Full Dep-Tree Edge List Covers Every Participant", 3-10 "The Terminal Stops Denying Dependencies That Exist") backed by the spec corrigenda at specification.md:547 and :555. The substance this criterion protected — real builder counts rather than forced zeros — holds: internal/cli/dep_tree_test.go:365-379 pins `chains: 1`, `longest: 2`, `blocked: 2` for the cycle. Nothing was lost; the emptied document now fires only where the graph truly holds no participant.
  - Drift on the JSON criterion: `JSONFormatter.FormatDepTree` (internal/cli/json_formatter.go:372-378) carries no message guard, so JSON on the empty branches is now the emptied full document rather than `{"message": …}`. The task's own Edge Cases note says task 3-5 removes it and that the JSON assertions belong there; internal/cli/dep_tree_test.go:872-897 holds that behaviour.
  - Pretty's zero-trees branch depends on the builder setting `Message` under exactly the condition that leaves `Trees` empty. That coupling holds today: `message` is set iff `len(trees) == 0` (internal/cli/dep_tree_graph.go:166-169), and `Trees` is empty iff no participant exists, since any `BlockedBy` entry makes its blocker a participant that blocks something and therefore gets seeded. The comment at internal/cli/pretty_formatter.go:374 states this correctly.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/dep_tree_test.go:339-350 and :352-363 — the two toon empty branches decode with an empty `dep_tree` and three zero counts. `toonRows` (internal/cli/toon_decode_test.go:37-42) fatals on a missing key, and `decodeToonDoc` (:17-28) fatals on a non-object, so a regression to `FormatMessage` fails these rather than passing vacuously.
  - internal/cli/dep_tree_test.go:313-324 and :326-337 — pretty stdout compared byte-for-byte against `"No dependencies found.\n"` on both branches, run through `App.Run` with `--pretty` (helper at :137-145). Reverting internal/cli/pretty_formatter.go:377 to `""` fails these with `"\n"`.
  - internal/cli/dep_tree_test.go:365-379 — cycle counts (`chains: 1`, `longest: 2`, `blocked: 2`) beside the cycle's edges.
  - internal/cli/dep_tree_test.go:731-745 — the populated graph still decodes with its edges and non-zero counts, unchanged by the handler change.
  - internal/cli/dep_tree_test.go:815-830 — `--quiet` stdout is empty.
  - internal/cli/dep_tree_test.go:872-897 — JSON emits the emptied full document with no `message` key.
- Notes: Not over-tested. The two empty-branch toon subtests differ in seeded state (no tasks vs unconnected tasks), which is the point the task makes — the root set is empty for a different reason in each. No redundant assertions, no mocking, no implementation-detail assertions; every subtest drives `App.Run` end to end.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` "it does X" subtests, `t.TempDir`-backed project setup helpers, `t.Helper()` on helpers; handler keeps the `Run<Command>(dir, fc, fmtr, flagArgs, literals, stdout)` signature.
- SOLID principles: Good — the change moves a rendering decision out of the handler and into the formatters, so each format owns its own empty-branch answer.
- Complexity: Low — `runFullDepTree` is now a single expression plus `printDocument`.
- Modern idioms: Yes.
- Readability: Good — the doc comment at internal/cli/pretty_formatter.go:373-374 states the condition under which the sentence stands in, and it holds against the builder.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires running the suite, vet and gofmt over the change-set; reading settled that the assertions are correct and would catch a regression, not that the toolchain reports clean.
