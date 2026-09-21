TASK: free-text-round-trip-3-2 — The Full Dep-Tree Summary Becomes Top-Level Named Fields (tick-904665)

ACCEPTANCE CRITERIA:
- `formatFullDepTree` output decodes without error via `toon.DecodeString` for a populated graph
- The decoded document carries `chains`, `longest` and `blocked` as top-level numbers and carries no `summary` key
- The `dep_tree` edge section is unchanged — same header, same rows, same order — and stays first in the document
- A summary field whose value is zero is still emitted with its name
- A `DepTreeResult` with no roots renders `dep_tree[0]{from,to}:` plus the three named fields, and the whole document decodes
- `DepTreeResult.Summary` is still populated by the graph builder and still rendered by `PrettyFormatter.formatFullDepTree`
- `grep -n 'encodeToonSingleObject\|toonDepTreeSummary\|toonStatsSummary' internal/cli/` returns nothing
- `grep -n 'strings.Replace' internal/cli/toon_formatter.go` returns nothing
- Pretty and JSON dep-tree output are byte-identical to before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §5.1 names three single-object sites whose TOON headers were hand-edited by stripping `[1]` with a string replace — the task-detail head, `tick stats`' counts, and the dep-tree chains/longest/blocked summary — producing a table header with no table beneath it. §5.2 requires the same treatment for all three: the stats counts and the dep-tree summary become top-level named fields beside their tables. §5.3/§5.4 argue named fields over a one-row table and over a wrapping key. §8 fixes the emptied full-graph form as the non-empty document emptied: count-zero edges header plus the summary fields reading zero. §4.1/§4.3 keep pretty as it is, so `DepTreeResult.Summary` (the human sentence) stays on the struct and in the graph builder.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/toon_formatter.go:182-193 — `formatFullDepTree` adds `buildEdgeSection("dep_tree", …)` first, then `encodeToonFields` over `chains`, `longest`, `blocked` in that order, sourced from `result.ChainCount`, `result.LongestChain`, `result.BlockedCount`.
  - internal/cli/toon_formatter.go:171 — `FormatDepTree` doc comment reads "Full graph: dep_tree[N]{from,to}: section + chains, longest and blocked named fields", matching the emitted shape.
  - Deletions confirmed: `grep -rn 'encodeToonSingleObject\|toonDepTreeSummary\|toonStatsSummary' internal/cli/` returns nothing; `grep -n 'strings.Replace' internal/cli/toon_formatter.go` returns nothing (the file's remaining `strings` uses are `Join` and `Cut`).
  - internal/cli/dep_tree_graph.go:164-174 still builds and sets `Summary`; internal/cli/pretty_formatter.go:390 still writes it as the trailing line.
  - Commit 2219d0c8 touched only `internal/cli/toon_formatter.go` and `internal/cli/toon_formatter_test.go`, so pretty and JSON dep-tree output are byte-identical by construction. JSON's `{mode, trees, chains, longest, blocked}` shape is untouched (internal/cli/json_formatter.go:338-340, 370).
- Notes: Later tasks in this plan refactored the surrounding code (the edge section now reads `result.Edges`; sections are collected through `toonDoc` with error propagation). The named-fields emission and the ordering this task required survive that refactor intact. The zero-value criterion holds structurally: `encodeToonFields` builds `toon.NewObject` from explicit `toon.Field`s, so no omit-empty path can drop a zero.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/toon_formatter_test.go:977-992 — decodes the populated document, asserts `chains`/`longest`/`blocked` as `float64(3)/(5)/(7)` and asserts `summary` absent.
  - internal/cli/toon_formatter_test.go:994-1017 — decodes and asserts the `dep_tree` rows and their order.
  - internal/cli/toon_formatter_test.go:1019-1038 — `ChainCount: 0` still decodes with `chains` == `float64(0)`.
  - internal/cli/toon_formatter_test.go:1040-1054 — `FormatDepTree(DepTreeResult{})`: first line is exactly `dep_tree[0]{from,to}:`, the decoded `dep_tree` list is empty, and all three fields read `float64(0)`. This is the only assertion pinning the edge section as the document's first line, and it covers §8's emptied form.
  - internal/cli/conformance_test.go:1005-1043 — end-to-end `tick dep tree` documents (populated, emptied, cycle) decode and carry the three fields.
  - internal/cli/pretty_formatter_test.go:878-892 — pretty output still ends with `result.Summary`; internal/cli/json_formatter_test.go:1029-1036 and 1396 keep the JSON `chains/longest/blocked` and full-graph key-set assertions.
  - internal/cli/readme_samples_test.go:30, 385 — the README toon dep-tree fence (README.md:343-349) is reproduced from real command output, so the documented sample is the named-fields form.
- Notes: The decode helpers (`decodeToonDoc`, `assertToonFields`, `assertToonKeysAbsent`, `toonRows` in internal/cli/toon_decode_test.go:17-88) fail on a decode error, so a malformed document fails these tests rather than passing silently. No redundant mocking or setup; assertions are behavioural (decoded values), not structural string matching, except where the header text is the behaviour under test.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it …", `t.Helper()` on helpers; formatter changes stay inside `internal/cli`; key literals are inline exactly as the sibling `FormatStats` does (internal/cli/toon_formatter.go:113-121), the `field*` constants being the task-detail selection registry's, not a general name table.
- SOLID principles: Good — `encodeToonFields` is the single named-field encoder for stats, task detail, focused dep tree and now the full graph; the bespoke header-surgery helper is gone.
- Complexity: Low — `formatFullDepTree` is two `doc.add` calls and a join.
- Modern idioms: Yes.
- Readability: Good.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires running the toolchain; reading shows gofmt-conformant formatting and no unused symbols left by the deletions, but pass/fail is an execution result.
- "`formatFullDepTree` output decodes without error via `toon.DecodeString` for a populated graph" — asserted by the tests above (which fatal on a decode error) and by the conformance suite; confirming the assertions hold needs the suite run.
