TASK: free-text-round-trip-3-10 (tick-d05b7e) — The Terminal Stops Denying Dependencies That Exist

ACCEPTANCE CRITERIA:
- A project whose only dependency is a task blocked by an unresolvable ID renders that edge in the terminal and does not print `No dependencies found.`
- A project whose only dependencies form a cycle renders those tasks in the terminal and does not print the sentence
- A project with no dependency relationships at all still prints `No dependencies found.` and nothing else
- The terminal's summary line is unchanged on every input
- Root-reachable projects render byte-identically to before this task
- The toon and JSON dep-tree documents are unchanged on every input
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§4.1 ("Pretty is unchanged, everywhere") carries the amended exception clause: pretty's shape is preserved, not a wrong figure inside it. It now records the exception as having arisen twice, both in `tick dep tree`, and states the end state this task delivers — "pretty's tree draws the participants no root reaches alongside the rooted ones, and the sentence fires only when the graph holds no participant at all". §8 describes the same rendering from the data side: "The node-shaped renderings still walk … any participant not yet drawn seeds a further walk", and §8's dangling-participant paragraph fixes the `(missing)` status marker that a later task (7-3) put on the seeded bare-ID node. Implementation matches both.

IMPLEMENTATION:
- Status: Implemented (evolved past the task's wording by later tasks, soundly)
- Location:
  - internal/cli/dep_tree_graph.go:167 — `if len(trees) == 0` guards the `No dependencies found.` assignment (the task's commit 66cf5517 changed `len(roots) == 0` to `len(roots) == 0 && len(unrooted) == 0`; task 7-2 later merged `Roots`/`Unrooted` into the single `Trees` field, so the guard now reads over the concatenation — the same condition).
  - internal/cli/pretty_formatter.go:375-392 — `formatFullDepTree` loops over `result.Trees` (roots first, then the seeded unrooted trees, in `slices.Concat` order at dep_tree_graph.go:154) and returns `result.Message` only at dep_tree_graph.go's empty condition.
  - README.md:321 — prose updated so all three formats are described as covering every participant (further revised by later tasks; current text matches current behaviour).
- Notes:
  - The plan's `Roots`/`Unrooted` field names no longer exist (`grep -rn "fullGraphTrees\|Unrooted" internal/` returns nothing). The merge into `Trees` came from task 7-2, after this one; no orphaned helper or field is left behind. This is a later, deliberate consolidation, not a loss.
  - The `Message` field's setting condition and pretty's read guard are now the same predicate, so `Message` can never disagree with what pretty draws. A residual concern recorded in the task's fix-tracking file (a wrong `Message` being unobservable) was dissolved by that consolidation.
  - `Message` is read by no other formatter (`grep -rn "\.Message" internal/cli/*.go` excluding tests → pretty_formatter.go:377, 401, 403 only), and the same was true at this task's commit, so the changed condition cannot move the toon or JSON documents.
  - Trees empty implies no participants: every participant is either a blocked task (drawn as a child) or a blocker ID, and every blocker ID has a non-empty `blocks` entry, so it is either already emitted or seeded by `buildSeededTrees` (dep_tree_graph.go:230-262). The sentence therefore fires exactly on a dependency-free project.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/dep_tree_test.go:656 "it renders a cycle in the terminal" — full stdout pinned byte-for-byte, tree plus `1 chain, longest: 2, 2 blocked`; the sentence cannot appear.
  - internal/cli/dep_tree_test.go:675 "it renders a dangling blocker as missing in the terminal" — pins `tick-ghost1   (missing)` over the edge (the `(missing)` marker arrived with task 7-3; this subtest is this task's, re-pinned there).
  - internal/cli/dep_tree_test.go:693 "it renders a two-level dangling chain once in the terminal" — the no-duplication case.
  - internal/cli/dep_tree_test.go:712 "it leaves rooted terminal output unchanged" and :442 "it still duplicates the diamond in the pretty and JSON trees" — both pin exact pretty bytes for root-reachable graphs.
  - internal/cli/dep_tree_test.go:313 and :326 — the sentence still stands for an empty project and for a project whose tasks carry no dependencies, asserted as whole-stdout equality (`"No dependencies found.\n"`), which also covers "and nothing else".
  - internal/cli/dep_tree_graph_test.go:24 and :52 pin `Message` at the build level for both empty inputs; :294-:392 pin the seeded trees, statuses and counts for the dangling, dangling-chain, cycle and two-cycle fixtures.
  - internal/cli/pretty_formatter_test.go:771 "it renders multiple roots in full graph mode" pins the blank-line separator the multi-tree loop emits, so a mixed rooted+seeded graph's rendering is covered at the unit level even though no e2e fixture mixes them.
- Notes: Every test asserts rendered output or built structure, not internals; no redundant duplicates of the same assertion; fixtures were extracted into shared helpers (`danglingBlockerTasks`, `chainTasks`) rather than inlined twice. Would fail if the behaviour regressed: reverting the pretty loop to roots-only breaks the cycle and dangling subtests.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` "it does X" subtests, `--pretty` in the CLI test helper because `bytes.Buffer` is non-TTY (internal/cli/dep_tree_test.go:145), `t.Helper()` on helpers.
- SOLID principles: Good — the builder decides what to draw, the formatter decides how; the change did not push graph knowledge into the formatter.
- Complexity: Low — one condition widened, one loop's input widened; no new branch.
- Modern idioms: Yes — `slices.Concat` for the tree join, `max` builtin for the longest path.
- Readability: Good.
- Comment accuracy: The rewritten doc comment at internal/cli/pretty_formatter.go:373-374 ("The no-dependencies message stands in only when the graph holds no participant at all") holds against the code. The commit also deleted a comment the change falsified at internal/cli/dep_tree_graph_test.go ("No roots … so should get 'No dependencies found.'"). No process-artifact references in the changed code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires executing the suite, vet and gofmt over the repository; not settleable by reading.
