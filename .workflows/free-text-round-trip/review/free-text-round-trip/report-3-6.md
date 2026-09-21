TASK: free-text-round-trip-3-6 (tick-b3a485) — README Dep-Tree Sample Matches Real Output

ACCEPTANCE CRITERIA:
- The TOON dep-tree sample was copied from real tool output rather than hand-written
- Its summary is three top-level `chains:` / `longest:` / `blocked:` lines, and no `summary{` header remains anywhere in the README
- The `dep_tree[2]{from,to}:` edge section and its two rows are unchanged
- `TestREADMEToonSamplesDecode` locates the block by its first line and decodes it, and fails if the block is removed or made unparseable
- The Pretty cell still shows the box-drawing tree and the `1 chain, longest: 2, 2 blocked` sentence, matching real pretty output
- The prose introducing the section still describes the two modes and the summary line
- No `tick stats` sample is added
- The samples corrected in Phases 1 and 2 are byte-identical to what those phases left
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §12.1 lists the dep-tree summary header (`README.md:307`, inside `### dep`) among the README samples this work replaces, on the grounds that the README is live documentation and "leaving it describing output the tool does not produce is shipping a defect". §5.2 is the shape change it reflects: a single-object `summary{...}:` section becomes top-level named fields sitting as peers of the collection section, blank-line separated. §4.1 leaves pretty unchanged, so the Pretty cell is verified rather than rewritten.

IMPLEMENTATION:
- Status: Implemented (drifted in test routing, deliberately and for the better)
- Location: README.md:341-350 (TOON cell), README.md:328-335 (Pretty cell), README.md:321 (prose), internal/cli/readme_samples_test.go:30 (`readmeDepTreeAnchor`), internal/cli/readme_samples_test.go:384-385 (dep-tree samples in `readmeSampleGroups`). Delivered in commit 9a0204c2.
- Notes:
  - The TOON cell now reads `dep_tree[2]{from,to}:` with rows `tick-a1b2,tick-c3d4` and `tick-c3d4,tick-f3e4`, a blank line, then `chains: 1`, `longest: 2`, `blocked: 2` — exactly the document `ToonFormatter.formatFullDepTree` builds (internal/cli/toon_formatter.go:182-193: edge section added first, then `chains`/`longest`/`blocked` as `toon.Field`s). The commit's diff touched only the two summary lines, so the edge header and its two rows are unchanged.
  - The summary values agree with the graph the sample describes: `collectStoredEdges` emits one row per stored `BlockedBy` in task order (internal/cli/dep_tree_graph.go:180-200), `blocked` counts tasks with blockers = 2, `countChains` counts one undirected component = 1, and `longestPath` counts edges, giving 2 for the three-node chain (internal/cli/dep_tree_graph.go:107-113, 144-157).
  - No `summary{` occurs anywhere in README.md or under internal/ (grep returns nothing).
  - No `tick stats` sample exists: `grep -n '^\$ tick stats' README.md` returns nothing; the `### stats` section (README.md:356-362) carries only a bash usage block.
  - Drift from the plan's step 4, and it is an improvement: the anchor was added to `TestREADMEToonSamplesDecode` in this task's own commit, then task 3-8 (28b01b5e, "compare every README sample against rendered output") retired the decode-only subtest in favour of `TestREADMESamplesMatchRenderedOutput`, which seeds `depTreeTasks` and byte-compares the rendered `--toon dep tree` and `--pretty dep tree` output against the two README blocks. Byte equality with real output subsumes "decodes as TOON", so the criterion is met in substance by a stronger check. The anchor constant is not orphaned — it is consumed at readme_samples_test.go:385.

TESTS:
- Status: Adequate
- Coverage: `TestREADMESamplesMatchRenderedOutput` → "it reproduces the README dep tree samples" covers both cells: "dep tree pretty" (anchor `tick-a1b2  Setup auth (done)`, unique in README at line 330) and "dep tree toon" (anchor `dep_tree[2]{from,to}:`, unique at line 343). `findREADMEBlock` returns an error — escalated by `t.Fatalf` — when no fence matches the anchor, so removing the block fails the suite; any edit to the block's bytes fails the comparison. "it fails when a documented sample has no matching README block" is the surviving form of the planned anchor-miss test. `TestREADMEPromptedSamplesAreClaimed` guarantees both `$ tick dep tree` prompted fences stay claimed by a rendering sample.
- Notes:
  - The planned micro test "it has no single-object section header in the README" was added here and removed again by task 3-8 together with the decode subtests. The state it asserted still holds (no `summary{` in the README), and the realistic regression path — the dep-tree sample itself — is now byte-pinned rather than merely parseable, so the guard was traded up rather than lost. The residual gap is only an unprompted fence reintroducing `summary{`, which no sample renders; not worth a test of its own.
  - Not over-tested: the dep-tree sample rides the existing table-driven group rather than getting a bespoke test, as the task's edge cases required.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in "it does X" form, table-driven samples, helpers marked `t.Helper()`; the change adds one anchor constant alongside the existing block and two rows to the sample table.
- SOLID principles: Good — the sample table stays the single place a README block is bound to the command that renders it.
- Complexity: Low — no new control flow.
- Modern idioms: Yes.
- Readability: Good — `depTreeTasks` names the seed the sample describes, and the anchors read as the lines they match.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The TOON dep-tree sample was copied from real tool output rather than hand-written" — provenance is not observable by reading; what is observable is byte equality with rendered output, which needs `go test ./internal/cli -run TestREADMESamplesMatchRenderedOutput` to confirm (the "dep tree toon" and "dep tree pretty" subtests).
- "The Pretty cell still shows the box-drawing tree and the `1 chain, longest: 2, 2 blocked` sentence, matching real pretty output" — the tree and sentence are present at README.md:330-334 and match what `PrettyFormatter.formatFullDepTree` composes; exact byte equality needs the same test run.
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires running the three commands.
