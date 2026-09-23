TASK: Order Show's Blockers By Declaration Order (same-second-tasks-sort-by-id-2-2, tick-5b3b57)

ACCEPTANCE CRITERIA:
- A task T whose `blocked_by` array declares its blockers in an order that contradicts both their ascending-ID order and their creation order, where the first-declared blocker was created last: `tick show T` lists the blockers in declaration order, first-declared first (§4.3, §8.4, §8 fixture constraints)
- Same fixture: T's blockers appear in the same order in `tick show T`, in T's `blocked_by` array in `tasks.jsonl`, and among the edges into T that `tick dep tree` emits (§3.3, §4.3, §8.4)
- A task whose blockers record one creation second and all carry the same non-zero sequence, declared in an order that contradicts their ascending-ID order: `tick show` lists them in declaration order, identically on repeated runs (§5.1)
- A project whose `cache.db` was built at schema version 2, with a `dependencies` table that has no ordinal. The next `tick show T` succeeds and lists T's blockers in declaration order. Afterwards `cache.db` reports `schema_version` 3, and its `dependencies` table, read directly, holds for each task an ordinal that orders the blockers as that task's `blocked_by` array does (§3.3, §3.4)
- Same fixture as the first scenario: `tick show T --field blocked_by.1` returns a one-row blocked_by section holding the first-declared blocker (§4.4, §8.4)

STATUS: complete

SPEC CONTEXT: §3.3 adds an ordinal to the cache's `dependencies` junction table carrying each blocker's position in the record's `blocked_by` array; the array order is explicit in the record and was lost only by flattening. §4.3 orders `tick show`'s blockers by that ordinal so show agrees with the stored record and with `tick dep tree` (which reads the task slice directly, `internal/cli/dep_tree_graph.go:189-203`). §5.1 notes the ordinal is unique within a task's blocker set, so the order is total with no ID term. §3.4 keeps the single 2→3 schema bump; `ensureFresh` deletes and rebuilds a version-mismatched cache. §4.4/§8.4 make `--field blocked_by.N` change meaning intentionally.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/storage/cache.go:32-37 — `dependencies` gains `ordinal INTEGER NOT NULL`; primary key stays `(task_id, blocked_by)`
  - internal/storage/cache.go:145 — dependency insert prepared with `(task_id, blocked_by, ordinal)`
  - internal/storage/cache.go:213-217 — each blocker inserted with its slice index `i`, so ordinals are 0..n-1, unique per task
  - internal/storage/cache.go:15 — `schemaVersion` stays 3 (no second bump)
  - internal/cli/show.go:137 — blockers query now `ORDER BY d.ordinal`
- Notes: The v2 upgrade path holds by reading: `OpenCache` runs `CREATE TABLE IF NOT EXISTS` over a v2 file without error (the indexes it creates reference only columns v2 already has), then `ensureFresh` (internal/storage/store.go:401-412) reads version 2, and `recreateCache` deletes the file and reopens it with the v3 schema before `Rebuild` fills in the ordinal. Blocker order is total, because the ordinal is distinct within each task's blocker set, so determinism across runs follows from the query alone. The pretty, toon and JSON formatters, and `selectedItems` for a nil or single position, all keep the slice order `queryShowData` returns, so every format carries declaration order. `blocked_by` has no `items` in the `showFields` registry (internal/cli/show_fields.go:61), so `--field blocked_by.1` renders a narrowed one-row section, not a bare value, as the criterion requires. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage: internal/cli/show_blockers_order_test.go has one subtest per criterion:
  - :109 AC1 — declared first/second/third = ccc333/aaa111/bbb222. The seqs (all at `sameSecond`) put the first-declared blocker last in creation order, and ascending ID and creation order both give second, third, first, so an ID-ordered or creation-ordered query fails.
  - :115 AC2 — the shown order is compared to the stored `blocked_by` array (via `storage.ReadJSONL`) and to the `dep_tree` rows whose `to` is T.
  - :123 AC3 — every record shares seq 7 and one second, the IDs contradict declaration order, and the check repeats three times.
  - :137 AC4 — `createV2Cache` (internal/cli/list_order_test.go:150) builds a two-column v2 `dependencies` table with a matching hash, so only the version is stale. The test then asserts `schema_version` 3 and reads `dependencies ORDER BY task_id, ordinal` directly for two blocked tasks, each declared against its ID and creation order, with a row-count check. A constant or reversed ordinal would fail it.
  - :166 AC5 — `--field blocked_by.1` returns exactly the first-declared blocker.
  - internal/storage/cache_test.go:46 — the schema test's expected `dependencies` columns now include `ordinal`.
- Notes: Each fixture follows the §8 constraints (IDs contradict the expected order; the declared order contradicts blocker creation order). Each subtest pins a different criterion, so none are redundant.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` subtests with "it ..." names, `t.Helper()` on helpers, toon output decoded rather than string-matched, error wrapping unchanged)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (`for range 3`, `slices.Equal` via `assertIDOrder`)
- Readability: Good. The comment at internal/cli/show_blockers_order_test.go:11-12 states the declared, creation and ID orders, and it matches the fixture.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
