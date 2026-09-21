TASK: free-text-round-trip-3-5 (tick-49e3d7) — JSON Dep-Tree Documents Drop Their Message Forms

ACCEPTANCE CRITERIA:
- `tick dep tree --json` on a project with nothing blocked emits `{"mode":"full","roots":[],"chains":0,"longest":0,"blocked":0}` and parses as one object
- `tick dep tree <id> --json` for a task with no dependencies either way emits `mode`, `target`, `blocked_by` and `blocks`, with both directions `[]`
- `roots`, `blocked_by` and `blocks` unmarshal to non-nil empty slices, never `null`, on every branch
- No dep-tree JSON document carries a `message` key
- `target` stays a nested object carrying `id`, `title` and `status`
- `mode` still reads `full` or `focused`
- `JSONFormatter.FormatMessage` still renders `{"message": …}`, and `tick init` and the cache-rebuild message still emit it
- Toon and pretty dep-tree output are unchanged by this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§8 holds that `tick dep tree`'s two English answers ("No dependencies found." for the full graph, "No dependencies." for a named task) go in toon and JSON, replaced by the populated document emptied — same fields, summary fields reading zero — while pretty keeps them (§4.3). §4.2 requires a JSON consumer to get the same structured answer as a toon one. §3.2 keeps the prose exemption for confirmations (`init`, the cache-rebuild notice), whose renderer is `FormatMessage`. §11 requires the rewritten assertions to check decoded values rather than output text.

IMPLEMENTATION:
- Status: Implemented (with one explained downstream rename, below)
- Location: internal/cli/json_formatter.go:343-350 (`jsonDepTreeFocused` — no `Message` field, no `omitempty` on `BlockedBy`/`Blocks`), :369-378 (`FormatDepTree` — no `result.Message` branch), :390-401 (`formatFocusedDepTreeJSON` — both directions assigned unconditionally through `toJSONDepTreeNodes`), :334-341 (`jsonDepTreeFull`), :255-263 (`jsonMessage`/`FormatMessage` intact). Commit 158a8f20.
- Notes:
  - The task's first criterion names the full-graph list key `roots`; the shipped key is `trees` (internal/cli/json_formatter.go:337). That is the deliberate work of the later plan task 7-2 ("the full-graph json document names its key for what it holds", commit 777f67e3), which also added internal/cli/dep_tree_test.go:576-589 asserting `roots` is absent and `trees` present. The criterion is met in substance — an always-present, never-null list on the full document under the name the plan later settled on. Not a loss, not a finding.
  - `DepTreeResult.Message` (internal/cli/format.go:271-273) is still set by the graph builder (internal/cli/dep_tree_graph.go:178, :361) and consumed only by pretty (internal/cli/pretty_formatter.go:377, :401-403) — I checked every non-test `.Message` reference in internal/cli, and those three are all of them. That matches §4.3's split rather than leaving orphaned state.
  - The handler no longer short-circuits the nothing-blocked branch through `FormatMessage`: `runFullDepTree` (internal/cli/dep_tree.go:37-40) calls `FormatDepTree` unconditionally, so the full-graph JSON branch is genuinely reachable and the empty document is what a consumer gets.
  - `FormatMessage`'s two prose callers survive: internal/cli/init.go:37 and internal/cli/rebuild.go:26.
  - The commit touched only internal/cli/json_formatter.go plus the two test files, so toon and pretty dep-tree rendering are untouched by this task.

TESTS:
- Status: Adequate
- Coverage:
  - Unit, internal/cli/json_formatter_test.go: "it emits both directions on a focused document with no dependencies" (:1463), "it emits blocked_by and blocks as empty arrays never null" (:1477), "it keeps target as a nested object" (:1491), "it emits the emptied full document instead of a message" (:1509), "it keeps mode on both documents" (:1536), "it still renders a message for init" (:1557). All unmarshal and assert decoded values, per §11; the shared setup is `parseFocusedNoDepsJSON` (:1571), which feeds a `Message` into the formatter so the deleted guard would resurface if it came back.
  - The full-document empty-list criterion is covered twice: "it renders trees as [] not null when empty" (:1117) and the trees/counts assertions inside :1509.
  - End-to-end through `App.Run` with `--json`: internal/cli/dep_tree_test.go:832 (focused, no deps) and :872 (full graph, nothing blocked), both via `runDepTreeJSON` (:151), which fails on a non-zero exit and on unparseable output.
  - Pretty's retained sentences are pinned separately at internal/cli/dep_tree_test.go:313 and :326 (both run through `runDepTree`, which passes `--pretty` at :145) and internal/cli/pretty_formatter_test.go:1086, so a regression that stripped prose from pretty would fail.
- Notes: Would fail if the feature broke — restoring `omitempty` breaks the `[]any` type assertions at json_formatter_test.go:1481 and dep_tree_test.go:862; restoring the message guard breaks the absence checks at :1466/:1517/:843/:880. No over-testing worth naming: the unit and App-level pairs check different layers (document shape vs. handler wiring), and the three `parseFocusedNoDepsJSON` subtests assert disjoint facts.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests, `t.Helper()` on `runDepTreeJSON` and `parseFocusedNoDepsJSON`, `t.TempDir()`-backed project setup, "it does X" subtest naming.
- SOLID principles: Good — the change removes branching from the formatter and leans on the existing `toJSONDepTreeNodes` converter rather than duplicating its non-nil guarantee.
- Complexity: Low — `formatFocusedDepTreeJSON` is now a single struct literal; three `len(...)` guards and a mutable local are gone.
- Modern idioms: Yes.
- Readability: Good.
- Issues: None. The doc comments hold against the code: `FormatDepTree` (json_formatter.go:369-371) names `{mode, trees, chains, longest, blocked}` and `{mode, target, blocked_by, blocks}`, both accurate; `jsonDepTreeFocused` (:344) says both directions are always present, which the tags and the unconditional assignment make true.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs those three commands run over the repo; I verified by reading that the renamed `trees` key is consistent across formatter and tests (the only remaining `"roots"` literal in internal/ is the absence assertion at internal/cli/dep_tree_test.go:581), but compilation, suite pass and formatting cannot be settled by reading.
