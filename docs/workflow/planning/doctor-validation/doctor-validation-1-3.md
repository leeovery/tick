---
id: doctor-validation-1-3
phase: 1
status: pending
created: 2026-01-30
---

# Cache Staleness Check

## Goal

The cache staleness check is the first real diagnostic check for the doctor framework. It detects when the SQLite cache (`cache.db`) is out of sync with the JSONL source of truth (`tasks.jsonl`) by comparing SHA256 content hashes. Without this check, a stale cache could cause tick to return incorrect query results until the next write triggers an automatic rebuild. This task implements a `CacheStalenessCheck` that conforms to the `Check` interface (from task 1-1) and produces a `CheckResult` with error severity and the specific fix suggestion "Run `tick rebuild` to refresh cache".

## Implementation

- Create a `CacheStalenessCheck` struct that implements the `Check` interface from task 1-1. It needs access to the `.tick` directory path (provided via context) to locate `tasks.jsonl` and `cache.db`.
- Implement the `Run` method with the following logic:
  1. Attempt to read `.tick/tasks.jsonl`. If the file does not exist, return a single failing `CheckResult` with Name `"Cache"`, Severity `SeverityError`, Details explaining that `tasks.jsonl` is missing, and Suggestion `"Run tick init or verify .tick directory"`. A missing JSONL file means the data store is broken, not just stale — but this check still reports it as an error rather than silently passing.
  2. Compute the SHA256 hash of the raw `tasks.jsonl` file contents (hash the bytes, not parsed structs — consistent with tick-core's freshness detection).
  3. Attempt to open `.tick/cache.db`. If the file does not exist, return a single failing `CheckResult` with Name `"Cache"`, Severity `SeverityError`, Details `"cache.db not found — cache has not been built"`, and Suggestion `"Run \`tick rebuild\` to refresh cache"`.
  4. Query the SQLite `metadata` table for the row where `key = 'jsonl_hash'`. If the metadata table doesn't exist, the key is missing, or the query fails, treat as stale (hash mismatch).
  5. Compare the computed JSONL hash with the stored `jsonl_hash` value. If they match, return a single passing `CheckResult` with Name `"Cache"`, Passed `true`. If they don't match, return a single failing `CheckResult` with Name `"Cache"`, Severity `SeverityError`, Details `"cache.db is stale — hash mismatch between tasks.jsonl and cache"`, and Suggestion `"Run \`tick rebuild\` to refresh cache"`.
- The check must be read-only. It opens `cache.db` in read-only mode (e.g., `?mode=ro` SQLite URI or equivalent). It never writes to any file.
- The check must not trigger a cache rebuild. Doctor diagnoses; it does not fix. This is design principle #1: "Report, don't fix."
- Close the SQLite connection after querying. The check should not hold database handles open.

## Tests

- `"it returns passing result when tasks.jsonl and cache.db hashes match"`
- `"it returns failing result with stale details when hashes do not match"`
- `"it returns failing result when cache.db does not exist"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it returns failing result when metadata table has no jsonl_hash key"`
- `"it returns passing result for empty tasks.jsonl with matching hash"`
- `"it suggests 'Run tick rebuild to refresh cache' when cache is stale"`
- `"it suggests 'Run tick rebuild to refresh cache' when cache.db is missing"`
- `"it uses CheckResult Name 'Cache' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not modify tasks.jsonl or cache.db (read-only verification)"`
- `"it returns exactly one CheckResult (single result, not multiple)"`

## Edge Cases

- **Missing `cache.db`**: Report as error. The cache has never been built or was deleted. This is a valid scenario — for example, after `tick init` before any operation, or after manually deleting the cache. Suggestion directs user to `tick rebuild`. Do not attempt to create or rebuild the cache.
- **Missing `tasks.jsonl`**: Report as error with a different detail message. The source of truth is gone — this is more severe than a stale cache but still reported through the same check mechanism. The suggestion should point toward reinitializing or verifying the `.tick` directory, not `tick rebuild` (rebuilding from a missing source makes no sense).
- **Empty `tasks.jsonl` with matching hash**: A valid, healthy state. An empty file has a well-defined SHA256 hash (`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`). If `cache.db` stores this same hash, the check passes. This confirms the check works with zero tasks, not just populated data stores.
- **Hash mismatch**: The primary failure case. Could be caused by external JSONL edits (manual, git pull, merge), a failed write that updated JSONL but not SQLite, or database corruption that lost the hash. All produce the same diagnostic: stale cache, run rebuild.
- **Corrupted or unreadable `cache.db`**: If the SQLite file exists but can't be opened or queried (e.g., not a valid SQLite database, permission denied), treat as stale — the check should not panic or crash. Return a failing result indicating the cache is unreadable.
- **Missing metadata table or missing key**: If `cache.db` exists but lacks the `metadata` table or the `jsonl_hash` key, treat as stale. This covers partially built or schema-incompatible cache files.

## Acceptance Criteria

- [ ] `CacheStalenessCheck` implements the `Check` interface
- [ ] Passing check returns `CheckResult` with Name `"Cache"` and Passed `true`
- [ ] Hash mismatch returns error-severity failure with details and `"Run \`tick rebuild\` to refresh cache"` suggestion
- [ ] Missing `cache.db` returns error-severity failure with rebuild suggestion
- [ ] Missing `tasks.jsonl` returns error-severity failure with appropriate (non-rebuild) suggestion
- [ ] Empty `tasks.jsonl` with matching hash returns passing result
- [ ] Check is read-only — never modifies `tasks.jsonl` or `cache.db`
- [ ] Check never triggers a cache rebuild
- [ ] Corrupted/unreadable `cache.db` treated as stale (does not panic)
- [ ] Tests written and passing for all edge cases

## Context

The specification defines cache staleness as Error #1: "Hash mismatch between JSONL and SQLite cache." The fix suggestion table specifies the exact text: "Run `tick rebuild` to refresh cache."

The hash mechanism is defined in tick-core's specification: SHA256 hash of the raw `tasks.jsonl` file contents, stored in the SQLite `metadata` table under key `jsonl_hash`. The doctor check replicates the read side of this mechanism — it computes the hash the same way tick-core does and compares it against the stored value — but never writes.

Doctor design principle #1 ("Report, don't fix") is critical here. Tick's normal read path auto-rebuilds stale caches, but doctor explicitly must not. It diagnoses and suggests `tick rebuild`; the user decides.

The specification states doctor "never modifies data" and exit code 1 means "one or more errors found." Cache staleness is an error (not a warning), so it contributes to exit code 1 via the runner and formatter built in tasks 1-1 and 1-2.

This is a Go project. Use `crypto/sha256` from stdlib for hash computation and `github.com/mattn/go-sqlite3` for SQLite access (consistent with tick-core). Open `cache.db` in read-only mode.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
