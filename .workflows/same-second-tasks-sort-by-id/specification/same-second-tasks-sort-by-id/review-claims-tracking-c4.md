# Review Tracking: Same-Second Tasks Sort By ID - Claims Verification

## Findings

### 1. The history check backing the backfill cites a function that did not exist in the commit it names

**Source**: Tree measurement — `git show 4278ba09:internal/storage/jsonl.go | grep -n '^func'`, `git log --oneline --reverse -S 'func MarshalJSONL' -- internal/storage/jsonl.go`
**Category**: Source defect
**Move**: route
**Affects**: §2.3 (Backfill for records with no sequence); the accepted risk in §7.2 that leans on the same history check

**Problem**:
Every project that already has tasks gets its authoring order frozen permanently the first time it is written after this change — the backfill reads line order out of `tasks.jsonl` and turns it into numbers that never move again. If line order was ever anything other than authoring order, those users get a silently and permanently scrambled task list with no error and no way to tell. The record's proof that this is safe is two-part, and the second part names the wrong function: it says the earliest writer, `MarshalJSONL`, existed in commit `4278ba09` and wrote the task slice unsorted. That commit has no `MarshalJSONL` — its writer is `WriteJSONL`, and `MarshalJSONL` first appears two commits later. The proposition is true under the right names (both writers iterate the slice with no sort), but as written the one guarantee protecting existing projects cannot be checked by anyone who follows the citation.

**Evidence**:
Claim (specification §2.3): "For a file with no sequences at all this yields line order, which *is* authoring order. Verified across the project's entire history: `git log -S \"sort.\" -- internal/storage/ internal/task/ internal/cli/create.go` returns no commits — no sort call has ever existed on those paths — and the first `MarshalJSONL` (commit `4278ba09`) iterated the task slice unsorted exactly as today."

Measurement 1 — the named commit has no `MarshalJSONL`:
```
$ git show 4278ba09:internal/storage/jsonl.go | grep -n '^func'
29:func toJSONL(t task.Task) jsonlTask {
47:func fromJSONL(jt jsonlTask) (task.Task, error) {
81:func WriteJSONL(path string, tasks []task.Task) error {
126:func ReadJSONL(path string) ([]task.Task, error) {
```

Measurement 2 — `MarshalJSONL` first appears in a later commit:
```
$ git log --oneline --reverse -S 'func MarshalJSONL' -- internal/storage/jsonl.go
23e0dc0f impl(tick-core): T tick-core-1-4 — Storage engine with file locking
```

Measurement 3 — the underlying proposition holds under the correct names. `WriteJSONL` at `4278ba09` writes the slice in order with no sort:
```
$ git show 4278ba09:internal/storage/jsonl.go | grep -n 'for _, t := range tasks'
98:	for _, t := range tasks {
```
and `MarshalJSONL` as introduced at `23e0dc0f` does the same:
```
$ git show 23e0dc0f:internal/storage/jsonl.go | grep -n -A 14 'func MarshalJSONL'
82:func MarshalJSONL(tasks []task.Task) ([]byte, error) {
...
88-	for _, t := range tasks {
```

Measurement 4 — the other half of the claim holds:
```
$ git log --oneline -S "sort." -- internal/storage/ internal/task/ internal/cli/create.go
(no output)
```

Source carrying the claim: `.workflows/same-second-tasks-sort-by-id/investigation/same-second-tasks-sort-by-id.md` — line 76 ("**Verified across history after validation (2026-09-22):** … The first `MarshalJSONL` (`4278ba09`, \"JSONL storage with atomic writes\") iterated the slice unsorted exactly as today") and line 196 (the same pairing inside "Existing files need no migration").

**Proposed Text**:

**Resolution**: Routed
**Notes**: Re-measured before classifying: `git show 4278ba09:internal/storage/jsonl.go | grep -n '^func'` returns `toJSONL`, `fromJSONL`, `WriteJSONL`, `ReadJSONL`, and `git log --oneline --reverse -S 'func MarshalJSONL' -- internal/storage/jsonl.go` puts the first `MarshalJSONL` in `23e0dc0f`. The proposition survives under the corrected names — both writers iterate the slice unsorted, and `git log -S "sort." -- internal/storage/ internal/task/ internal/cli/create.go` still returns no commits — so the repair is the citation alone. It landed in the investigation at both sites that carry the pairing (H3's history verification and Fix Direction's "Existing files need no migration"); §2.3 is re-aligned to it.

---

## Observations

- `internal/cli/dep_tree.go:26` is cited for `store.ReadTasks()` (§7.1); the call is at line 25 (`rg -n 'store.ReadTasks\(\)' internal/cli/dep_tree.go` → `25:	tasks, err := store.ReadTasks()`).
- `list_filter_test.go:368` is cited alongside `ready_test.go:306` and `blocked_test.go:212` as a fixture site (§8); 306 and 212 are `t.Run` lines, while 368 is the assertion comment "// Priority 1 tasks first, ordered by created ASC" — that subtest opens at :349 and its fixture runs :350-355.
- "No test constructs a creation-time tie today" (§8) reads wider than what measurement supports: fixtures with two tasks sharing priority and creation second do exist (`blocked_test.go:575-576`, `create_test.go:325-326`, `cascade_formatter_test.go:238-240`), but none asserts row order over the tied pair, so the operative point — no ordering test exercises the tie branch — stands.
- `README.md:206` is cited for the positional `--field` form (§4.4); line 206 is the `notes.2` sample, while the sentence covering `children` and `blocked_by` positions is `README.md:212`. Neither states an order, so the documentation scope in §6 is unaffected.
