# A repeated value in a task's tags, refs or blocked_by list bricks the store

A line in `.tick/tasks.jsonl` whose own `tags`, `refs` or `blocked_by` list holds the same value twice — for example `"tags":["x","x"]` — loads as a task without complaint, but from then on every tick command fails. Opening the store rebuilds the SQLite cache from the loaded tasks, and the cache refuses the repeat:

```text
Error: failed to ensure cache freshness: failed to rebuild cache: failed to insert tag tick-fa204e -> x: constraint failed: UNIQUE constraint failed: task_tags.task_id, task_tags.tag (1555)
```

Reproduced on a build from current source: `tick init`, `tick create "A"`, hand-edit the line to add `"tags":["x","x"]`, then `tick list` exits 1 with the error above. The same applies to a repeated ref or a repeated `blocked_by` entry, since those lists are stored with the same kind of per-task uniqueness in the cache (`internal/storage/cache.go`, around lines 213–229, where tags, refs and dependencies are inserted).

`tick doctor` on that store passes its JSONL syntax check and every relationship check, flags only `Cache: cache.db is stale`, and advises running `tick rebuild` — which fails with the same constraint error. So doctor points the user at a remedy that cannot work, and the only way out is hand-editing the file. The rebuild's error does at least name the task and the repeated value.

Conditions: tick's own writes do not produce this. `create --tags x,x` stores a single `x`, `update --tags` likewise, a repeated `dep add` is refused with "dependency already exists", and `migrate` carries no tags or dependencies. It is reachable through hand edits of `tasks.jsonl` or other tools writing the file — the same routes the oversized-text-field-bricks-the-store work covers for lines that fail to load.

Impact: the whole store becomes unusable through tick until the line is fixed by hand, and doctor's advice sends an agent to a rebuild that fails the same way. Agents run doctor to decide whether they can trust the store, so the misleading advice lands on the party least able to see past it.

This was found while planning oversized-text-field-bricks-the-store and deliberately left outside that fix, whose doctor changes judge whether each line loads as a task; a line that loads but breaks the cache build falls outside that boundary.
