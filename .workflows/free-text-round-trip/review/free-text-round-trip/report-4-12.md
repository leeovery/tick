TASK: free-text-round-trip-4-12 — The Formatters Gate Off The Registry's Names

ACCEPTANCE CRITERIA:
- Each of the fifteen names is spelled once as a gate literal — in the constant declaration — with the registry and all three formatters referring to the constants; none of the 60 measured lines carries a bare name literal.
- A misspelt name in the registry or in any formatter gate fails to compile.
- The parity test fails when any one formatter gate is removed — checked by hand against toon, pretty and JSON in turn, then restored.
- The recognition subtest iterates `showFields`, so a name added to the registry is covered with no test edit.
- `tick show <id>` output is byte-identical before and after in all three formats, filtered and full, and a mutation command still renders its `changed` section in toon.
- `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass.
- `grep -n "it keeps key order stable" internal/cli/json_formatter_test.go` returns nothing

STATUS: complete

SPEC CONTEXT: §9.1–9.7 define `tick show --field`: a registry of selectable names, positional suffixes on list sections, an unrecognised name as an error, and §9.7's rule that a filtered request returning a document honours the format flags and renders in each format exactly as a full `show` does. The task is the structural guard behind that rule — one vocabulary shared by parser and all three renderers, so a registry name cannot be accepted by the parser and silently unrendered in one format (the human-surface/agent-surface divergence §9.7 closes). The spec mandates no particular mechanism; the constants and the parity test are the plan's means.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/show_fields.go:29-45` — the fifteen names declared once as `fieldID … fieldNotes`, documented as "the detail document spells it".
  - `internal/cli/show_fields.go:48-65` — `showFields` keyed off the constants; `:69` — `showListSections` built from them.
  - `internal/cli/toon_formatter.go:78,82,86,90,94,102` and `:231` (`buildTaskSection`'s `add`, called at `:240-254`) — every gate on a constant.
  - `internal/cli/json_formatter.go:110-133` — every `add` on a constant; `internal/cli/pretty_formatter.go:147-174` and `:187-207` — likewise.
  - `internal/cli/format.go:126-130` — `selectedSections` (introduced by a later phase) also resolves positions through the constants, so the vocabulary held as the code moved.
  - `internal/cli/toon_formatter.go:97-99` — the `changed` section now gated on `detail.Changes != nil` alone; the `sel.includes("changed")` conjunct is gone (`git show efea72ab -- internal/cli/toon_formatter.go`).
- Notes:
  - The measured command `rg -n 'includes\("|Selected\("|Positions\("|add\("' internal/cli/*_formatter.go` now returns exactly one line: `internal/cli/json_formatter.go:136` — `add("changed", …)`. `changed` is not one of the fifteen registry names, and the plan's Do list scopes JSON's `changed` gate out explicitly. Criterion 1 is met in substance.
  - Enumerated every remaining bare field-name literal in non-test `internal/cli` sources: they are `json:"…"`/`toon:"…"` struct tags, toon's `buildRelatedSection("blocked_by"|"children")` / `encodeToonSection("tags"|"refs"|"notes")` section names, `buildEdgeSection("blocked_by")` and pretty's display labels — the output-vocabulary literals the plan told the task to leave alone. No gate literal survives.
  - Dropping the toon `changed` gate cannot alter any command's output: `TaskDetail.Fields` is assigned in exactly one production site (`internal/cli/show.go:83`, the `show` path) and `TaskDetail.Changes` in exactly one (`internal/cli/helpers.go:29`, the mutation path); no path sets both. A nil selection's `includes` returns true (`internal/cli/show_fields.go:164-166`), so mutation documents rendered `changed` before the change and render it now.
  - `grep -n "it keeps key order stable" internal/cli/json_formatter_test.go` returns nothing; the superseding document-order subtests are present at `internal/cli/json_formatter_test.go:1758`, `:1774` and `:1793`. The JSON `changed` subtest (`:1743`) is untouched, as the plan required.
  - The task's constants are untyped string constants, so the compile-error guarantee covers a misspelt *identifier* (`fieldTitel` — undefined) rather than a stray literal. That is the limit of the mechanism the plan chose; no stronger Go construct would reject an untyped literal passed to `includes(string)`. Not a shortfall against the criterion as written.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/show_fields_test.go:118-129` — "it recognises every registered name" now iterates `slices.Sorted(maps.Keys(showFields))` instead of the hardcoded fifteen, so a name added to the registry is covered with no test edit.
  - `internal/cli/show_fields_test.go:799-819` — `TestRegisteredFieldRendering` / "it renders every registered name in every format": for each registry name it renders `richDetail()` (`internal/cli/toon_formatter_test.go:1195-1215`, which carries every section, `Parent`, `Type` and `Closed`) with only that name selected, through `ToonFormatter`, `PrettyFormatter` and `JSONFormatter`, asserting non-empty output.
  - `internal/cli/toon_formatter_test.go:1348-1364` — "it renders the changed section for a mutation document" replaces the deleted `"it never carries the changed section"`, decoding the `changed` rows and pinning id/title/from/to/auto.
  - `internal/cli/toon_formatter_test.go:1366-1391` — "it leaves unfiltered output unchanged" pins the full toon key set and values, unedited.
- Notes:
  - The parity test's sensitivity holds by reading: each formatter returns the empty string when nothing survives the selection — `JSONFormatter.FormatTaskDetail` returns `"", nil` for an empty object (`internal/cli/json_formatter.go:92-95`), `toonDoc.join` returns `joinToonSections(...)` which drops empty sections (`internal/cli/toon_formatter.go:417-422`, `:396-403`), and pretty returns `"", nil` when no group survives (`internal/cli/pretty_formatter.go:219-221`). Removing any single `add(field…)`/`includes(field…)` gate therefore yields an empty document for that one-name selection, which the test reports. Verified for all three formats by tracing, not by running.
  - Not over-tested: the parity assertion is deliberately a non-emptiness check — the per-format filtered suites (`TestToonFilteredTaskDetail`, `TestJSONFilteredTaskDetail`, pretty's equivalents) already pin content and order, so a stronger assertion here would duplicate them.
  - The deleted `"it keeps key order stable"` removed no coverage: it compared two renders of identical input, which any deterministic formatter satisfies for free; the three document-order subtests listed above assert the actual ordering it missed.
  - `assertToonKeysAbsent`, the helper the deleted `changed` subtest used, is still used at ten other sites in `internal/cli/toon_formatter_test.go` (e.g. `:154`, `:197`, `:236`), so no orphaned helper was left behind.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests, "it does X" subtest naming, `t.Helper()` on helpers, no testify.
- SOLID principles: Good — one declaration of the field vocabulary, consumed by the registry and the three renderers; no new coupling.
- Complexity: Low — a mechanical literal-to-constant substitution plus one removed conjunct.
- Modern idioms: Yes — `slices.Sorted(maps.Keys(...))` for deterministic iteration over the registry.
- Readability: Good — the constant block's doc comment states the invariant ("each spelled as the detail document spells it"), which is the reason the same constant serves both a gate and, at `internal/cli/toon_formatter.go:103`, a rendered key.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass." — requires running the three commands at the repo root; reading cannot settle a suite result.
