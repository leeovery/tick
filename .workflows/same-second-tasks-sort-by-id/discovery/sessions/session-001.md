# Discovery Session 001

Date: 2026-09-21
Work unit: same-second-tasks-sort-by-id

## Description (as of session)

Tasks created within the same wall-clock second come back in random ID order instead of the order they were authored, because creation timestamps are stored at second granularity and the sort tie falls through to the task ID.

## Seed

- seeds/2026-09-21-same-second-tasks-sort-by-id.md (inbox:bug)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

The seed came from dogfooding tick as the plan store for the `free-text-round-trip` workflow. Five tasks in one phase were authored in a deliberate order — a consolidation task first, then the fix it enables, then the cleanups — and `tick ready` returned them shuffled. All five were created inside the same wall-clock second, so the creation-date sort tied on every row and fell through to the secondary key, the task ID, which is three random bytes.

The conditions are narrow to state and wide in practice: any batch created fast enough to share a second. A human typing `tick add` repeatedly will not hit it; anything scripted hits it every time — an agent authoring a plan, an importer bringing tasks from another tool, a shell loop. The faster the writer, the more reliably authoring order is lost.

The impact falls on every consumer that treats tick as an ordered list rather than a bag of tasks. The workflow system's plan adapter is exactly such a consumer: its reading instructions state that creation date preserves authoring order, and the implementation loop takes "the next available task" from that order. In the case observed the tasks were independent, so nothing broke — the ordering was merely visibly wrong in the "N of M in phase" positions and in the order the loop took tasks up. A batch with real sequencing between its tasks would have executed out of order with no signal that anything was amiss. The same tie affects `tick list` on a freshly imported project, the migration framework's output, and a dependency chain authored in one pass.

The tie is in the stored data, not only in the query: JSONL records carry timestamps at second granularity. The seed names the likely ground — `internal/cli/query_helpers.go` and the list/ready/blocked ORDER BY paths, the task record's `created` field in `internal/task/`, and the SQLite cache schema in `internal/storage/cache.go`.

Scope was narrowed in the shaping conversation: the user confirmed that getting creation order right for new tasks going forward is sufficient. What the fix should do about tasks already stored at second granularity — existing `tasks.jsonl` files that already contain ties — was raised and deliberately left to investigation rather than settled here.

Shape confirmed as a bugfix. Behaviour that is claimed and not delivered, with a concrete symptom, but the cause runs through the stored record format, the cache schema and the ordering clauses on several query paths — no single mechanical change is nameable yet, which is what rules out quick-fix.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

(none)
