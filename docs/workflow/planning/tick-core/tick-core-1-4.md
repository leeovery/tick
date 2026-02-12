---
id: tick-core-1-4
phase: 1
status: completed
created: 2026-01-30
---

# Storage engine with file locking

## Goal

The JSONL reader/writer (tick-core-1-2) and SQLite cache (tick-core-1-3) are independent components — nothing composes them into a unified storage engine, and nothing prevents concurrent access corruption. Tick needs a single entry point that orchestrates the full read and write flows from the specification: acquire lock, read JSONL, check freshness, rebuild if stale, apply mutation or query SQLite, write atomically, update cache, release lock. This task builds that orchestration layer using `github.com/gofrs/flock` for shared/exclusive file locking with a 5-second timeout.

## Implementation

- Define a `Store` type (or equivalent) that holds the `.tick/` root path and owns the JSONL reader/writer, SQLite cache, and flock instances
- Implement `NewStore(tickDir string)` constructor that validates the `.tick/` directory exists and contains `tasks.jsonl`
- Implement the **write mutation flow** as a single method (e.g., `Mutate(fn func(tasks []Task) ([]Task, error)) error`):
  1. Acquire exclusive lock on `.tick/lock` using `flock.TryLockContext` with a 5-second timeout context
  2. Read `tasks.jsonl` into memory, compute SHA256 hash
  3. Check SQLite freshness — rebuild cache if stale (delegates to tick-core-1-3 `EnsureFresh`)
  4. Pass parsed `[]Task` to the mutation function, receive modified `[]Task`
  5. Write modified tasks via atomic rewrite (delegates to tick-core-1-2 writer)
  6. Update SQLite in a single transaction: apply changes + store new hash
  7. Release lock (via defer)
  8. If JSONL write succeeds but SQLite update fails: log warning to stderr, continue (next read self-heals)
- Implement the **read query flow** as a method (e.g., `Query(fn func(db *sql.DB) error) error`):
  1. Acquire shared lock on `.tick/lock` using `flock.TryRLockContext` with a 5-second timeout context
  2. Read `tasks.jsonl` into memory, compute SHA256 hash
  3. Check SQLite freshness — rebuild cache if stale
  4. Execute the query function against SQLite
  5. Release lock (via defer)
- Implement lock timeout handling: if `TryLockContext` / `TryRLockContext` returns context deadline exceeded, return error with message: "Could not acquire lock on .tick/lock - another process may be using tick"
- Use `github.com/gofrs/flock` — create flock instance on `.tick/lock` file path
- Shared locks allow concurrent readers; exclusive lock blocks all others
- Ensure lock release always happens via `defer` even on panics
- The lock file (`.tick/lock`) is created automatically by the flock library if it doesn't exist

## Tests

- `"it acquires exclusive lock for write operations"`
- `"it acquires shared lock for read operations"`
- `"it returns error after 5-second lock timeout"`
- `"it allows concurrent shared locks (multiple readers)"`
- `"it blocks shared lock while exclusive lock is held"`
- `"it blocks exclusive lock while shared lock is held"`
- `"it executes full write flow: lock → read JSONL → freshness check → mutate → atomic write → update cache → unlock"`
- `"it executes full read flow: lock → read JSONL → freshness check → query SQLite → unlock"`
- `"it releases lock on mutation function error (no leak)"`
- `"it releases lock on query function error (no leak)"`
- `"it continues when JSONL write succeeds but SQLite update fails"`
- `"it rebuilds stale cache during write before applying mutation"`
- `"it rebuilds stale cache during read before running query"`
- `"it surfaces correct error message on lock timeout"`

## Edge Cases

- Lock timeout (5 seconds): return descriptive error "Could not acquire lock on .tick/lock - another process may be using tick". Do not proceed with the operation. The user can manually delete `.tick/lock` if a crashed process left it behind.
- Concurrent reads: multiple goroutines/processes can hold shared locks simultaneously. Reads must not block each other. Test with concurrent goroutines both acquiring shared locks and completing queries.
- Stale cache during write: a write operation must check freshness *after* acquiring the exclusive lock but *before* applying the mutation. This handles the scenario where another process wrote to JSONL between the last read and this write. The mutation function receives up-to-date data.
- JSONL write succeeds, SQLite fails: log warning to stderr, return success. The JSONL is the source of truth — the operation succeeded. The next read will detect the hash mismatch and rebuild the cache automatically. Do not propagate the SQLite error to the caller.
- Lock release on panic/error: always use `defer flock.Unlock()` immediately after acquiring the lock. The lock must never leak, even if the mutation or query function panics.
- Lock file doesn't exist: `gofrs/flock` creates it automatically. No special handling needed, but the `.tick/` directory must exist.

## Acceptance Criteria

- [ ] `Store` composes JSONL reader/writer and SQLite cache into a single interface
- [ ] Write operations acquire exclusive lock, read operations acquire shared lock
- [ ] Lock timeout of 5 seconds returns descriptive error message
- [ ] Concurrent shared locks allowed (multiple readers)
- [ ] Exclusive lock blocks all other access (readers and writers)
- [ ] Write flow executes full sequence: lock → read → freshness → mutate → atomic write → update cache → unlock
- [ ] Read flow executes full sequence: lock → read → freshness → query → unlock
- [ ] Lock is always released, even on errors or panics (defer pattern)
- [ ] JSONL write success + SQLite failure = log warning, return success
- [ ] Stale cache is rebuilt before mutation or query executes

## Context

The specification defines two distinct operation flows. **Write (mutation) flow**: (1) acquire exclusive lock on `.tick/lock`, (2) read `tasks.jsonl` + compute hash + check freshness (rebuild if stale), (3) apply mutation in memory, (4) write complete new content to temp file, (5) fsync temp file, (6) `os.Rename(temp, tasks.jsonl)`, (7) update SQLite in single transaction (apply mutation + store new hash), (8) release lock. **Read (query) flow**: (1) acquire shared lock on `.tick/lock` (allows concurrent reads, blocks writers), (2) read `tasks.jsonl` + compute hash + check freshness, (3) if stale rebuild from JSONL in memory, (4) query SQLite, (5) release lock.

File locking uses `github.com/gofrs/flock`. Exclusive lock for writes blocks all other readers and writers. Shared lock for reads allows other readers but blocks writers. Lock timeout is 5 seconds; on timeout return error, do not proceed.

JSONL-first, SQLite is expendable: if JSONL write succeeds but SQLite fails, log warning and continue. Next read detects hash mismatch and rebuilds automatically. The hash update is part of the same SQLite transaction as the data update.

The lock file lives at `.tick/lock`, separate from data files. If a crashed process leaves the lock file behind, the user can manually delete it.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
