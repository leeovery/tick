TASK: free-text-round-trip-3-3 (tick-df8fc7) — The Focused Dep Tree Names Its Task On Both Branches

ACCEPTANCE CRITERIA:
- `formatFocusedDepTree` output decodes without error via `toon.DecodeString` on every branch
- The decoded document carries `id`, `title` and `status` as top-level keys equal to the target's values, on both the populated and the empty branch
- `blocked_by` and `blocks` are both present in every focused document; an empty direction renders `blocked_by[0]{from,to}:` / `blocks[0]{from,to}:` and decodes to an empty list
- The edge rows for a populated direction are unchanged — same `{from,to}` pairs in the same order, diamond duplication preserved
- No hand-built identity line and no `No dependencies.` string is produced by the toon formatter
- A target whose title contains a comma decodes back as one value, and the same for an ID
- `PrettyFormatter.FormatDepTree` output for a focused result is byte-identical to before this task on both branches
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §5.2 requires the line identifying a named task in `tick dep tree` to become top-level `id`, `title` and `status` fields, carried on both branches, so a focused dependency document opens exactly as a task-detail document does. §8 requires the two dep-tree prose answers (`No dependencies found.`, `No dependencies.`) to go from toon and JSON — an answer to a query may not be prose — and the empty branch to be the populated document emptied, with edge sections carrying their count-zero header. §4.3 records that the named-task branch already reaches the formatters, so no handler change is needed, and that `DepTreeResult.Message` stays populated because pretty renders it. Two later spec corrigenda (2026-09-19, 2026-09-20) govern subsequent tasks that reshaped the edge source; they do not disturb this task's requirement.

IMPLEMENTATION:
- Status: Implemented (subsequently evolved by planned later tasks, all consistent with the criteria)
- Location:
  - internal/cli/toon_formatter.go:195-207 — `formatFocusedDepTree` builds three sections: `encodeToonFields` over `fieldID`/`fieldTitle`/`fieldStatus` (constants declared at internal/cli/show_fields.go:31-33 as "id"/"title"/"status") from `result.Target`, then `buildEdgeSection("blocked_by", ...)` and `buildEdgeSection("blocks", ...)`, unconditionally, joined via `toonDoc.join`.
  - internal/cli/toon_formatter.go:218-223 — `buildEdgeSection` returns `emptyToonSection[toonEdgeRow](name)` for a zero-length slice; internal/cli/toon_formatter.go:327-334 derives the header columns from the `toonEdgeRow` toon tags, so an empty direction renders exactly `blocked_by[0]{from,to}:` / `blocks[0]{from,to}:`.
  - internal/cli/toon_formatter.go:170-173 — `FormatDepTree` doc comment now states the focused shape ("both always present"); it matches the code.
  - Original delivery: commit e7d0b989. The early return `len(result.BlockedBy) == 0 && len(result.Blocks) == 0 && result.Message != ""` with its `fmt.Sprintf("%s  %s (%s)\n%s", …)` identity line, and both `len(...) > 0` guards, are deleted, not repaired.
- Notes:
  - Two later planned tasks reshaped this function and are not drift from 3-3: 3-9 (d38deefa) routed the join through `joinToonSections`, 8-3 (e7bb74da) swapped the walk-derived `collectUpstreamEdges`/`collectDownstreamEdges` for the stored-relation fields `result.BlockedByEdges`/`result.BlocksEdges` (spec corrigendum 2026-09-20), and 9-1 (cabd1deb) made the function error-returning via `toonDoc`. `collectUpstreamEdges`/`collectDownstreamEdges` no longer exist anywhere in `internal/` — no dead helper left behind.
  - The `formatFocusedDepTree` doc comment this task added was later removed by a642d906 ("analysis cycle 3 — comment corrections") as pure paraphrase; the surviving `FormatDepTree` comment above it still describes the focused shape accurately, so nothing in the file makes a false claim.
  - No toon-side reference to `DepTreeResult.Message` remains (`Message` appears in internal/cli/toon_formatter.go only inside the unrelated `FormatMessage` at line 132-135). `BuildFocusedDepTree` still sets `Message: "No dependencies."` (internal/cli/dep_tree_graph.go:352) for pretty, exactly as §4.3 requires.
  - Pretty's focused view is untouched: internal/cli/pretty_formatter.go:394-417 keeps the target header line, the `Blocked by:` / `Blocks:` labels, omit-when-empty, and the `No dependencies.` sentence. Commit e7d0b989 touched only toon files.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/toon_formatter_test.go:1056 — populated branch: `id`/`title`/`status` decode to the target's values alongside both edge lists.
  - internal/cli/toon_formatter_test.go:1074 — no-dependencies branch: same three keys, both edge lists assert empty via `assertToonRowsEmpty`, which routes through `toonRows` (internal/cli/toon_decode_test.go:37-56) and `t.Fatalf`s on a missing key — so it fails if a section is omitted rather than passing vacuously.
  - internal/cli/toon_formatter_test.go:1084 and :1097 — the two "omits" subtests became count-zero assertions, checking both the literal `blocked_by[0]{from,to}:` / `blocks[0]{from,to}:` header and the decoded empty list, with the populated direction's edges asserted by value.
  - internal/cli/toon_formatter_test.go:1110 — a comma-bearing title decodes back as one value.
  - internal/cli/toon_formatter_test.go:1119 — no `No dependencies.` substring on the empty branch, plus a decode.
  - internal/cli/toon_formatter_test.go:1151 — the diamond case (renamed by task 8-3 to "it renders the focused sections from the result's edge fields") still asserts the same four `{from,to}` rows in the same order, duplication preserved.
  - internal/cli/toon_decode_test.go:402-459 — end-to-end through `App.Run([]string{"tick","--toon","dep","tree",id})` on a seeded project, one subtest per branch (both directions with a comma-bearing title, upstream only, downstream only, neither).
  - internal/cli/pretty_formatter_test.go:1052-1096 — pretty's focused header line and `No dependencies.` assertions remain and are unmodified by this task's commit.
- Notes:
  - The phase 6 conformance table (internal/cli/conformance_test.go:308-338) later added the same four focused branches, but it asserts decodability only; the 3-3 tests assert decoded values. The overlap is thin and deliberate, not redundancy worth removing.
  - The criterion's "and the same for an ID" has no dedicated subtest. Not reported as a finding: `id` travels the same `encodeToonFields` path as `title`, and `DepTreeResult.Target.ID` comes from the stored task, whose IDs are `tick-` + 6 hex characters, so no comma-bearing ID is reachable — nothing breaks that a test would catch.

CODE QUALITY:
- Project conventions: Followed — field keys come from the `show_fields.go` registry constants rather than string literals, the section-building helpers are shared with the other documents, and errors flow through the `toonDoc` accumulator like every other formatter method.
- SOLID principles: Good — the formatter renders, the graph builder decides content; nothing in the formatter branches on emptiness any more.
- Complexity: Low — the function is straight-line, no conditionals.
- Modern idioms: Yes — generic `emptyToonSection[T]`/`encodeToonSection[T]`, tag-derived headers.
- Readability: Good — three `doc.add` calls read as the three sections of the document.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the toolchain; not judged here. Reading confirms the package compiles coherently (signatures, helpers and constants all resolve) and the assertions match the rendered shape, but an executing pass is needed to confirm the suite, vet and gofmt are clean.
- "`formatFocusedDepTree` output decodes without error via `toon.DecodeString` on every branch" — the tests that assert this exist and are correct by reading (internal/cli/toon_formatter_test.go:1056-1131, internal/cli/toon_decode_test.go:402-459); confirming they pass against the real `toon-go` decoder requires running them.
