TASK: Order Show's Children By Creation, Not Task ID (same-second-tasks-sort-by-id-2-1, tick-2951f6)

ACCEPTANCE CRITERIA:
- A parent P whose children c1, c2, c3 record one creation second and one priority and carry sequences in that authoring order. Their ascending-ID order and their line order in `tasks.jsonl` both contradict authoring order. `tick show P` lists the children c1, c2, c3 (§4.3, §8.4, §8 fixture constraints)
- A parent whose two children record one creation second, where the later-authored child carries a higher priority (a lower number) than its earlier-authored sibling and has an ID that sorts first: `tick show` lists the earlier-authored child first (§4.3, §8.4)
- A parent with one child recorded in an earlier creation second and one in a later second, where the earlier-second child carries the higher sequence and the later-second child's ID sorts first: `tick show` lists the earlier-second child first, because creation date outranks the sequence (§4.2, §4.3)
- A parent whose children record one creation second and all carry the same non-zero sequence, with a line order in `tasks.jsonl` that differs from ascending-ID order: `tick show P` lists them in ascending task-ID order, identically on repeated runs (§5.1, §8.3)
- Same fixture as the first scenario: `tick show P --field children.1` returns a one-row children section holding c1, and `--field children.3` returns one holding c3 (§4.4, §8.4)

STATUS: complete

SPEC CONTEXT: §4.3 moves the show children sub-list off the unconditional `ORDER BY id` onto created, then sequence, then task ID, with no priority term (a better-priority child must not float above an earlier-created sibling). §4.2 keeps `created` above `seq`. §5.1 makes the task ID the absolute final term so a duplicate sequence falls back to ID order deterministically. §8 fixture constraints: IDs and line order must contradict the expected order, creation seconds tie wherever seq or ID is under test, assertions state the order and never the plan. §4.4/§8.4: positional `--field children.N` changes meaning by design and must be asserted. §2.4: seq is never surfaced.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/show.go:145-148 (children query `SELECT id, title, status FROM tasks WHERE parent = ? ORDER BY created ASC, seq ASC, id ASC`); seq column at internal/storage/cache.go:29, populated by the rebuild insert at internal/storage/cache.go:139 and :208
- Notes: The clause matches §4.3 exactly: created, seq, id, no priority term. `created` is stored via `task.FormatTimestamp` (internal/storage/cache.go:205), a fixed-width UTC layout, so text ordering equals chronological ordering and §4.2 holds. `id` is the table's PRIMARY KEY (internal/storage/cache.go:19), so the order is total and repeated runs cannot differ. The children rows keep their three columns (id, title, status), so seq is not surfaced (§2.4). `outputMutationResult` (internal/cli/helpers.go:23) renders through the same `queryShowData`, so create/update/note detail documents inherit the order with no second query to drift. `--field children.N` is not a bare value (`fieldChildren` has no `items`, internal/cli/show_fields.go:62), so a lone position renders a one-row section, matching README.md:210.

TESTS:
- Status: Adequate
- Coverage: internal/cli/show_order_test.go has one subtest per criterion, all driving `tick show` end to end through toon output.
  - Criterion 1 (lines 50-54): IDs tick-bbb222/ccc333/aaa111 for c1/c2/c3 give ID order c3,c1,c2 and line order c2,c3,c1 (lines 40-47). Both contradict the expected c1,c2,c3. One second, priority 2 throughout.
  - Criterion 2 (lines 56-68): the later child has priority 0, the lower ID and the first line. A priority term, ID order or file order would each fail the test.
  - Criterion 3 (lines 70-82): the later-second child has the lower seq (2 against 5), the lower ID and the first line. A clause with seq above created would fail it.
  - Criterion 4 (lines 84-96): shared non-zero seq 7, line order c,a,b against ID order a,b,c. Asserted three times.
  - Criterion 5 (lines 98-110): `children.1` and `children.3` over the criterion-1 fixture, each asserting a single-row children section holding the expected ID.
- Notes: Every fixture keeps line order and ID order apart from the expected order, as §8 requires. The tests are focused, with no redundant assertions. Formats: testing toon only is sufficient because pretty and JSON render the same `data.children` slice. The pretty formatter's imports (internal/cli/pretty_formatter.go:3-11) include neither `sort` nor `slices`.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` "it ..." naming, `t.Helper()` on helpers, decoded toon assertions via `decodeToonDoc`/`toonRows`, fixtures via the shared `setupRawProject`/`seqLine`/`assertIDOrder` in internal/cli/list_order_test.go)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (`for range 3`, Go 1.22 per-iteration loop variables in the table loop)
- Readability: Good. The comment at internal/cli/show_order_test.go:33 states the fixture's property and holds true against lines 35-46.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
