TASK: Sort Same-Second Tasks By Creation Sequence (same-second-tasks-sort-by-id-1-1, tick-33a556)

ACCEPTANCE CRITERIA:
- A project holding a parent P and five open children step-1 … step-5 that record one creation second and one priority, carry sequences in authoring order, and have IDs whose ascending order contradicts authoring order: `tick list` returns step-1 … step-5 in authoring order (§1.3, §8.1)
- Same project: `tick list --parent P` returns step-1 … step-5 in authoring order (§8.1)
- Same project: `tick ready --parent P --count 1` returns step-1, not the ID-lowest child (§1.2, §8.1)
- Tasks recording one creation second and one priority, whose sequences contradict both their line order in `tasks.jsonl` and their ascending-ID order: `tick list` returns them in sequence order (§2.1, §3.2, §8.2)
- A project whose `cache.db` was built at schema version 2 (a `tasks` table with no `seq` column, `schema_version` 2 in metadata): the next `tick list` succeeds, and afterwards `cache.db` reports `schema_version` 3 and its `tasks` table's `seq` column holds each task's sequence, read from the table directly (§3.2, §3.4, §8.2)
- A task carrying a sequence is written to `tasks.jsonl`, read back and written again: the task read back carries the same sequence and the two writes are byte-identical (§3.1, §8.2)
- No command output carries the sequence: `tick show <id>` under toon, pretty and JSON has no `seq` key or line, and `tick show <id> --field seq` fails as an unknown field name (§2.4)
- The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified (§8.5)

STATUS: complete

SPEC CONTEXT: §1.1 names two compounding defects: whole-second creation times make a same-second batch tie, and the list-family ORDER BY ended at `t.created ASC`, so SQLite emitted tied rows in whatever order the chosen plan fed the sorter (file order under the priority/status indexes, ID order under the PK index). §1.2: `LIMIT` cuts that arbitrary order, so `ready --count 1` can pick the wrong task. §2.1/§3.1: the fix is an explicit `seq` field on the stored record. §3.2: the field is carried into a cache `seq` column, and a test asserts the column directly. §3.4: schema version 2 -> 3, with no migration because `ensureFresh` deletes and rebuilds on a version mismatch. §4.1/§4.2: `seq` sorts after `created`, which still outranks it. §2.4: seq appears in no command output. §8 fixture constraints: IDs contradict authoring order, creation seconds tie, and fixture size carries no weight.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/task/task.go:57 — `Seq int` on `Task` (`json:"seq,omitempty"`); internal/task/task.go:77 — `Seq` on `taskJSON`; carried through `MarshalJSON` (internal/task/task.go:98) and `UnmarshalJSON` (internal/task/task.go:136)
  - internal/storage/cache.go:15 — `schemaVersion = 3`
  - internal/storage/cache.go:29 — `seq INTEGER NOT NULL DEFAULT 0` on `tasks`
  - internal/storage/cache.go:139 — rebuild insert names `seq`; value bound at internal/storage/cache.go:208 (`t.Seq`)
  - internal/cli/list.go:313 — ready clause: `(t.status = 'in_progress') DESC, t.priority ASC, t.created ASC, t.seq ASC, t.id ASC`
  - internal/cli/list.go:315 — neutral clause: `t.priority ASC, t.created ASC, t.seq ASC, t.id ASC`
  - internal/cli/list.go:318-321 — `LIMIT` still appended after the `ORDER BY`
  - internal/storage/store.go:407-412 — the existing version-mismatch path recreates the cache and needs no migration, as planned
- Notes: `seq` is placed after `created` in both clauses, as §4.2 requires. The trailing `t.id ASC` belongs to Task 1-2 and sits correctly after `seq`. `OpenCache` on a v2 file does not fail: every `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` in `schemaSQL` references only columns a v2 `tasks` table already holds, so control reaches the `SchemaVersion` check and the recreate path. The seq never reaches command output: `queryShowData` selects no `seq` (internal/cli/show.go:102), the list scan reads only id/status/priority/title/type (internal/cli/list.go:308), and the `showFields` registry rejects `seq` through the generic unknown-field error (internal/cli/show_fields.go:242). The inline comment on `Task.Seq` ("zero means none") matches the backfill rule in `backfillSeqs` (internal/storage/jsonl.go:123-131). The timestamp format and the `Truncate(time.Second)` sites are unchanged, as §7.1 requires.

TESTS:
- Status: Adequate
- Coverage:
  - Criteria 1–3: `TestSameSecondOrdering` (internal/cli/list_order_test.go:87-106). The fixture from `phaseLines` (internal/cli/list_order_test.go:58-64) uses one second and one priority, sequences in authoring order, and IDs that descend (tick-f0000f, tick-e00005 … tick-a00001), so ascending-ID order is the reverse of authoring order. The subtests assert the unfiltered `list`, `list --parent P`, and `ready --parent P --count 1` == [step-1].
  - Criterion 4: "it orders by sequence against both line order and ID order" (internal/cli/list_order_test.go:108-123). Sequence order, line order and ascending-ID order are three different orderings, so the test catches both the file-order accident and the PK-plan accident.
  - Criterion 5: `TestSchemaV2CacheUpgrade` (internal/cli/list_order_test.go:192-244). The test builds a real v2-shaped cache.db with a matching hash and `schema_version` 2, so only the version is stale (internal/cli/list_order_test.go:150-190). It runs `list` (`runToonCommand` fails the test on a non-zero exit), then reads `schema_version` and `SELECT id, seq FROM tasks` directly. If no rebuild ran, that SELECT would fail against the v2 table.
  - Criterion 6: "it round-trips the creation sequence byte-identically" (internal/storage/jsonl_test.go:210-242). The test marshals, parses and re-marshals with `Seq: 7`, asserting `"seq":7` in the first write, `Seq == 7` after the parse, and byte-identical writes. It exercises `MarshalJSONL`/`ParseJSONL`, the functions `WriteJSONL`/`ReadJSONL` wrap.
  - Criterion 7: `TestSequenceNotSurfaced` (internal/cli/list_order_test.go:265-321) covers toon, JSON and pretty `show` and `--field seq`. Toon and JSON use a recursive key search; pretty uses a substring check; the `--field seq` case asserts the exact stderr text, which matches internal/cli/show_fields.go:242.
  - Cache column asserted directly in the storage package too: "it stores each task's creation sequence in the seq column" (internal/storage/cache_test.go:214-239). The schema test includes `seq` in the expected `tasks` columns (internal/storage/cache_test.go:33), and the version pins moved to 3 (internal/storage/cache_test.go:1166, :1189, :1270, :1289, :1781).
  - Criterion 8: ready_test.go, blocked_test.go and list_filter_test.go are outside the change-set's file list. Their fixtures (internal/cli/ready_test.go:306-339, internal/cli/blocked_test.go:212-244) give distinct `created` values within each priority band, so the appended `seq`/`id` terms never decide an order they assert.
- Notes: Each ordering test would fail if the `seq` term were dropped: with the final `t.id ASC` present, the fallback is deterministic ascending-ID order, which every fixture contradicts. No redundant or over-mocked tests found.

CODE QUALITY:
- Project conventions: Followed. The tests use stdlib `testing` only, `t.Run` subtests named "it …", `t.Helper()` on helpers, `t.TempDir()` isolation via `setupTickProject`, and toon assertions that decode output (`decodeToonDoc`/`toonRows`) instead of comparing golden strings.
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good. The fixture comments hold against the fixtures they describe, including the ID-descent comment at internal/cli/list_order_test.go:21-22 and the three-orderings comment at :109-110.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The existing priority-then-created ordering tests (`ready_test.go:306`, `blocked_test.go:212`, `list_filter_test.go:368`) pass unmodified (§8.5)" — this needs a run of `go test ./internal/cli -run 'TestReady|TestBlocked|TestListFilter'` (or the full suite) to confirm the tests pass. On reading: the three files are absent from the change-set, and their fixtures differ on `created` within every priority band, so the new final terms cannot reorder them.
