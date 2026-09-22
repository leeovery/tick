# Investigation: Same Second Tasks Sort By Id

## Symptoms

### Problem Description

**Expected behavior:**
Tasks listed back in creation order come back in the order they were authored. A batch written one after another — by an agent, an importer, or a shell loop — reads back in the sequence it was written.

**Actual behavior:**
Tasks created within the same wall-clock second come back in task-ID order, which is effectively random — IDs are three random bytes. The creation-date sort ties on every row in the batch and falls through to the secondary key.

### Manifestation

- `tick ready` returned a five-task phase shuffled out of authoring order
- The plan adapter's "N of M in phase" positions were visibly wrong
- No error, no warning — the ordering is silently wrong; a batch with real sequencing between tasks would execute out of order with no signal

### Reproduction Steps

1. Create several tasks in quick succession (scripted, so they share a wall-clock second)
2. List them back — `tick list`, `tick ready`, or `tick blocked`
3. Observe the order does not match the order of creation

**Reproducibility:** Always, when the batch shares a second. A human typing `tick add` repeatedly will not hit it; anything scripted hits it every time.

### Environment

- **Affected environments:** Local — all usage
- **User conditions:** Any batch created fast enough to share a wall-clock second. The faster the writer, the more reliably authoring order is lost.

### Impact

- **Severity:** Medium — silent wrong ordering, no data loss
- **Scope:** Every consumer that treats tick as an ordered list rather than a bag of tasks: the workflow system's plan adapter and its implementation loop, `tick list` on a freshly imported project, the migration framework's output, a dependency chain authored in one pass
- **Business impact:** Trust in tick as a plan store. The observed case had independent tasks so nothing broke; a sequenced batch would run out of order undetected.
- **Live exposure:** None. No active work units — every plan authored into tick has been executed, so no stored batch is currently being read in the wrong order. Nothing is being worked around by hand.
- **Urgency:** None. The investigation can take the time it needs; no pressure to land a narrow fix fast.

### References

- Seed: `seeds/2026-09-21-same-second-tasks-sort-by-id.md` (inbox:bug)
- Discovery session: `.workflows/same-second-tasks-sort-by-id/discovery/sessions/session-001.md`
- Observed while dogfooding tick as the plan store for the `free-text-round-trip` workflow (Phase 10)

### Scope Note (from discovery)

Getting creation order right for new tasks going forward is confirmed sufficient. What the fix should do about tasks already stored at second granularity — existing `tasks.jsonl` files that already carry ties — was deliberately left to this investigation.

Confirmed at symptom gathering: there is no live affected data. Existing-tie handling is a question about other people's projects and about tick's own tolerance for older files, not about rescuing a plan currently on disk.

---

## Analysis

### Hypotheses

**Checkpoint depth:** straight-through

- **H1: Creation timestamps are truncated to whole-second granularity before storage, so a batch written inside one wall-clock second stores identical `created` values and the ordering information never reaches disk.** [suspected]
  Basis: `task.TimestampFormat = "2006-01-02T15:04:05Z"` carries no fractional part, and every stamping site applies `time.Now().UTC().Truncate(time.Second)` — `create.go:208`, `task.go:285` (`NewTask`), `state_machine.go:49`, `note.go:71`, `update.go:318`, `dep.go:116,186`, `migrate/store_creator.go:62`.

- **H2: The list-family `ORDER BY` terminates at `t.created ASC` with no deterministic tiebreak, so tied rows emerge in whatever order the SQLite query plan produces — empirically task-ID order.** [suspected]
  Basis: `list.go:317` (`ready` view) and `list.go:319` (neutral view) both end on `t.created ASC`; no ID term is written anywhere. `tasks` is a rowid table with `id TEXT PRIMARY KEY`, so the observed ID ordering is emergent from the plan, not requested by the code.

- **H3: The JSONL source of truth still holds true authoring order — records are written from the in-memory slice, which appends — so ordering is lost on read, not on write.** [suspected]
  Basis: `jsonl.go:16` `MarshalJSONL(tasks)` serialises slice order; creation appends. If it holds, existing files are repairable and row position is available as a tiebreak.

- **H4: Other fixed-order presentation paths share the defect or sort by ID deliberately — the blast radius exceeds the three list views.** [suspected]
  Basis: `show.go:137` (blockers), `show.go:146` (children) both `ORDER BY t.id` / `ORDER BY id`; `show.go:173` (notes) uses `ORDER BY rowid ASC`. The migrate framework builds many tasks in one pass and hits the same second every time.

- **H5: Second granularity is structural across every timestamp field, so raising precision changes the on-disk format for all of them and for files written by other versions.** [suspected]
  Basis: `created`, `updated`, `closed`, note `created` and transition `at` all share `TimestampFormat`; `task.go:112,116,137` parse with strict `time.Parse(TimestampFormat, …)`, which rejects a value carrying a fraction.

**Trace lines (in intended order):**
1. `internal/task/task.go` — `TimestampFormat`, `NewTask`, the JSON marshal/unmarshal round trip
2. Every stamping site: `create.go`, `update.go`, `note.go`, `dep.go`, `state_machine.go`, `migrate/store_creator.go`
3. `internal/cli/list.go:310-322` — the `ORDER BY`, plus an empirical `EXPLAIN QUERY PLAN` and live ordering check against a seeded project
4. `internal/storage/jsonl.go` and `cache.go` `Rebuild()` — whether authoring order survives the round trip and whether `rowid` tracks it
5. Other ordered read paths — `show.go` children/blockers/notes, dep tree, migrate output
6. Compatibility surface — strict `time.Parse`, `schemaVersion`, `internal/doctor/` checks, tests pinning the format

### Code Trace

(to be populated)

**Ground named by the seed:**
- `internal/cli/query_helpers.go` and the list/ready/blocked query paths carrying the `ORDER BY` clauses
- the task record's `created` field in `internal/task/`
- the SQLite cache schema in `internal/storage/cache.go`

### Root Cause

(to be populated)

### Contributing Factors

(to be populated)

### Why It Wasn't Caught

(to be populated)

### Blast Radius

(to be populated)

---

## Fix Direction

(to be populated after findings sign-off)

---

## Notes

**User's fix lean (symptom gathering, 2026-09-21):** raise timestamp precision — store creation times at millisecond granularity or finer so the sort no longer ties. Stated as a lean, explicitly open to alternatives. To be tested against the root cause at Step 10 rather than assumed.
