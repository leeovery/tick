# Consolidation Findings: same-second-tasks-sort-by-id (Phase 2)

## Findings

None. The phase's source surface is two ORDER BY clauses in `queryShowData` (internal/cli/show.go:137, :146) and one cache column with its insert (internal/storage/cache.go:35, :145, :213-214). Neither task duplicates the other or leaves anything dead. Both sub-list keys discriminate under mutation in a scratch copy:
- Constant ordinals (`insertDep.Exec(t.ID, dep, 0)`) fail every `TestShowBlockersOrder` and `TestMutationDetailSubListOrder` subtest. The query falls back to the primary-key autoindex order.
- Dropping `seq ASC` from the children clause fails `TestShowChildrenOrder`'s three authoring-order subtests and three mutation subtests.
- Dropping `id ASC` fails the shared-sequence subtest.

## Comment Corrections

- internal/cli/list_order_test.go:147-150 — the phase extended an enumeration that restates `v2SchemaSQL` directly above it, broke the line wrap, and proved the clause goes stale on the first edit; the hash-matches-so-only-version-is-stale clause is the part that carries anything
  OLD: // createV2Cache writes a cache.db shaped as schema version 2 — no seq column,
// no dependency ordinal —
// holding the project's tasks and a hash matching its tasks.jsonl, so only the
// version is stale.
  NEW: // createV2Cache writes a cache.db shaped as schema version 2, holding the
// project's tasks and a hash matching its tasks.jsonl, so only the version is
// stale.
- internal/cli/show_order_test.go:13-14 — restates the three-line helper's parameters and body
  OLD: // childLine renders one stored child of showParentID with the given
// sequence, priority and creation instant.
  NEW:
