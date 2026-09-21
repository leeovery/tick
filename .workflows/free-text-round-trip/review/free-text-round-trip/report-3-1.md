TASK: free-text-round-trip-3-1 — Stats Counts Become Top-Level Named Fields

ACCEPTANCE CRITERIA:
- `FormatStats` output decodes without error via `toon.DecodeString`
- The decoded document carries `total`, `open`, `in_progress`, `done`, `cancelled`, `ready` and `blocked` as top-level keys and carries no `stats` key
- Emitted field order is `total`, `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked`, unchanged from the old row order
- A count of zero is emitted with its name, not omitted
- `by_priority` still decodes to a five-element list of `{priority,count}` objects in priority order 0 to 4, including zero counts
- `tick stats` on a project with no tasks decodes with every count reading `0` and `by_priority` carrying its five rows
- Pretty and JSON `stats` output are byte-identical to before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §5.1 names three single-object sites rendered by marshalling a value as a one-element array and then deleting the `[1]` from the header with a string replace — a table header with no table beneath it, which a conformant TOON reader rejects on the document's first line. `tick stats`' counts summary is one of them (`encodeToonSingleObject("stats", summary)`). §5.2 requires those counts to become top-level named fields beside the `by_priority` table. §5.3 rejects the one-row-table alternative (positional reading, comma/quoting fragility); §5.4 rejects a wrapping key (consistency with `description`, mirrors flat JSONL storage, and the `--field` flag falls out of it). §4.1 keeps pretty unchanged; §4.2 moves JSON with toon only for the `changed` list and the structured empty dep-tree form — neither is stats.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/toon_formatter.go:109-130` — `FormatStats` builds a `toonDoc`, adding `encodeToonFields` over seven `toon.Field`s in the order `total`, `open`, `in_progress`, `done`, `cancelled`, `ready`, `blocked` (lines 113-121), then `encodeToonSection("by_priority", rows)` for the five priority rows (lines 123-127), and returns `doc.join("")`.
  - `internal/cli/toon_formatter.go:266-272` — `encodeToonFields` marshals `toon.NewObject(fields...)` through the library; nothing is hand-edited into the output.
  - `internal/cli/toon_formatter.go:285-294` / `:398-422` — `joinToonSections` drops empty sections before joining with a blank line; `toonDoc` collects sections and the first refusal. This is the plan's step 3 (`sections` filtering), arrived at via the later document/refusal consolidation rather than an inline filter — same behaviour, shared with `FormatTaskDetail`.
  - `toonStatsSummary` and `encodeToonSingleObject` are gone: `grep -rn "encodeToonSingleObject\|toonStatsSummary" --include="*.go" .` returns no matches anywhere in the repo.
  - Implementing commit: `8e5e7ff2` (`internal/cli/toon_formatter.go`, `stats_test.go`, `toon_formatter_test.go` only). Later commits changed the formatter signature to `(string, error)` and introduced `toonDoc`; neither altered the emitted stats text.
- Notes: Pretty and JSON stats are untouched in substance. `git diff b173ce4b..HEAD -- internal/cli/pretty_formatter.go` shows the only hunks inside `PrettyFormatter.FormatStats` are the signature line and `return b.String(), nil`; the same holds for `JSONFormatter.FormatStats` in `internal/cli/json_formatter.go` (`return marshalIndentJSON(obj), nil`) with `jsonStats`' `{total, by_status, workflow, by_priority}` nesting unchanged. Field keys are string literals rather than the `field*` constants — those constants belong to `--field` selection on `show`, which stats has no part in, so literals are right here.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_formatter_test.go:335-358` "it emits stats counts as top-level named fields" — decodes via `decodeToonDoc`, asserts all seven counts as `float64`, and `assertToonKeysAbsent(t, doc, "stats")`.
  - `internal/cli/toon_formatter_test.go:360-373` "it emits the counts in their established order" — splits the first block on `\n\n` and asserts the seven lines carry the field names in order (order survives the decode into an unordered map only as text, so this is the right level).
  - `internal/cli/toon_formatter_test.go:375-379` "it emits a zero count rather than omitting it" — `in_progress` present and `float64(0)`.
  - `internal/cli/toon_formatter_test.go:381-387` "it decodes counts as numbers" — `total` is a `float64`, not a string.
  - `internal/cli/toon_formatter_test.go:389-404` and `:406-424` — `by_priority` is five rows, `priority` 0..4 in order, counts matching `Stats.ByPriority`, zeros included.
  - `internal/cli/stats_test.go:250-288` "it formats stats in TOON format" — end-to-end `tick stats` on a seeded project, decoded stdout, all seven counts asserted, no `stats` key, `by_priority` counts checked.
  - `internal/cli/stats_test.go:290-314` "it decodes stats for a project with no tasks" — `setupTickProject` (empty `tasks.jsonl`, `internal/cli/create_test.go:19-31`), every count `float64(0)`, `by_priority` five rows all zero.
  - `internal/cli/conformance_test.go:973-1003` — the later conformance suite decodes the recorded `stats` documents on both the populated and empty branches, so the shape is also held from outside this package's unit tests.
  - Pretty and JSON stats assertions survive unchanged at `internal/cli/stats_test.go:316+` and `internal/cli/conformance_test.go:1889+`.
  - Every formatter call in these tests goes through `formatted(t).of(...)` (`internal/cli/format_test.go:496-506`), which fails the test on a refusal, so a refusal cannot be silently decoded as an empty document.
- Notes: "it keeps the by_priority table unchanged" and "it formats by_priority with 5 rows including zeros" assert the same invariant over different fixtures. Both were explicitly authored by the plan (Do step 4 kept the second, the Tests list added the first), the second acting as the untouched-behaviour witness; the overlap is deliberate rather than accidental, and folding them would be a preference.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.Helper()` on the shared helpers, `t.TempDir()` isolation via `setupTickProject`. Error wrapping is the package's `*toonEncodeError` route. `for i := range 5` is the modernize-era range-over-int the rest of the file uses.
- SOLID principles: Good. `FormatStats` composes two encoder-owned helpers and owns no serialization of its own; the refusal path is the shared `toonDoc`/`refusalForTask` mechanism rather than a local variant.
- Complexity: Low. One straight-line function, one loop over five priorities, no branching.
- Modern idioms: Yes.
- Readability: Good. The seven `toon.Field` literals state the emitted order at the call site, which is exactly what the acceptance criterion pins.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settling this needs the three commands run from the repo root; reading the sources cannot establish that the suite passes, that vet is silent, or that no file is unformatted.
