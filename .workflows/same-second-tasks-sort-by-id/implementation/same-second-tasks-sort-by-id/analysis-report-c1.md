# Analysis Report: Same Second Tasks Sort By Id (Cycle 1)

## Stats

- Total findings: 0
- Deduplicated findings: 0
- Proposed tasks: 0

## Summary
None of the three agents (duplication, standards, architecture) reported a finding that clears the floor. The creation-order key appears at three sites, but tests pin every term at each one. Every numbering path goes through `task.NextSeq`. The spec decisions and the §8 verification requirements match the implementation, and the consolidation-p1 direction (Store.Mutate numbers the slice it writes) is in place at internal/storage/store.go:191. This cycle leaves three stale comments, listed below, and nothing to stage.

## Comment Corrections

- internal/storage/store.go:172-173 — the "full flow" line restates Mutate's body and has been incomplete since this work added the numbering step (`backfillSeqs(mutated)`, :191) between mutate and the atomic write. What callers need stated is the numbering: fn may return tasks with no sequence, and the signature cannot say so. (Reported by duplication, standards and architecture. They proposed three conflicting edits, which are merged here into one: architecture's version, because it keeps the locking line and states the numbering rule exactly as `backfillSeqs`/`task.NextSeq` apply it. The duplication agent cited :173, the architecture agent :171-172; the lines are 172-173.)
  OLD: // Mutate executes a write mutation with exclusive file locking.
// The full flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock.
  NEW: // Mutate executes a write mutation with exclusive file locking. Tasks fn
// returns without a sequence are numbered, in order, above the highest sequence
// any returned task carries.
- internal/cli/list.go:148-149 — this comment restates buildListQuery's ORDER BY and presents priority then created as the whole key. That is now wrong: the seq and id terms follow (list.go:313, :315), and in the ready view the in_progress band comes first (source: architecture)
  OLD: // RunList executes the list command: queries tasks from SQLite with optional filters
// and outputs them via the Formatter, ordered by priority ASC, then created ASC.
  NEW: // RunList executes the list command: queries tasks from SQLite with optional filters
// and outputs them via the Formatter.
- CLAUDE.md:61 — the "currently v2" claim went stale when this work bumped the cache schema to 3 (`const schemaVersion = 3`, internal/storage/cache.go:15). Every agent loads this file as instructions (source: architecture)
  OLD: - **Cache schema versioning:** `schemaVersion` constant in `cache.go` (currently v2); `ensureFresh()` checks version before freshness hash — mismatch triggers delete+recreate+rebuild.
  NEW: - **Cache schema versioning:** `schemaVersion` constant in `cache.go`; `ensureFresh()` checks version before freshness hash — mismatch triggers delete+recreate+rebuild.

## Discarded Findings
- None. All three agents returned `FINDINGS: none`, so nothing reached synthesis to discard. The agents weighed three candidates and rejected them themselves:
  - The creation-order key repeated at list.go:313, list.go:315 and show.go:146. Tests pin every term at every site, so drift fails loudly.
  - The repeated test helpers. They are reuse only, and each fails its test when its section is missing.
  - The RunDoctor enumeration removal. code-quality.md's cardinality and restatement rules outrank the spec's request to grow the list.
