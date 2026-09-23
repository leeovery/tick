AGENT: architecture
FINDINGS: none
COMMENT_CORRECTIONS:
- internal/cli/list.go:148 — restates buildListQuery's ORDER BY and presents priority then created as the whole key, which this work made wrong: the sequence and ID terms now follow, and the ready band leads
  OLD: // RunList executes the list command: queries tasks from SQLite with optional filters
// and outputs them via the Formatter, ordered by priority ASC, then created ASC.
  NEW: // RunList executes the list command: queries tasks from SQLite with optional filters
// and outputs them via the Formatter.
- internal/storage/store.go:171-172 — the "full flow" line restates the body and has been false since this work added a numbering step between mutate and write. The contract callers need is that numbering: fn may return new tasks with no sequence, and the signature cannot say so.
  OLD: // Mutate executes a write mutation with exclusive file locking.
// The full flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock.
  NEW: // Mutate executes a write mutation with exclusive file locking. Tasks fn
// returns without a sequence are numbered, in order, above the highest sequence
// any returned task carries.
- CLAUDE.md:61 — the version value claim went stale when this work bumped the cache schema to 3, and every agent loads this file as instructions
  OLD: - **Cache schema versioning:** `schemaVersion` constant in `cache.go` (currently v2); `ensureFresh()` checks version before freshness hash — mismatch triggers delete+recreate+rebuild.
  NEW: - **Cache schema versioning:** `schemaVersion` constant in `cache.go`; `ensureFresh()` checks version before freshness hash — mismatch triggers delete+recreate+rebuild.
SUMMARY: The pieces fit together cleanly. One numbering rule (task.NextSeq) runs in ParseJSONL on read and in Store.Mutate on write, so the file and the cache always agree and the rule does not depend on callers. The list and show keys all end on a total term, and the doctor check's "carries a sequence" test matches storage's backfill test. Nothing clears the floor as a finding. Three pieces of text went stale with this change: the RunList sort claim, the Mutate flow line and the schema version in CLAUDE.md.
