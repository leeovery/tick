# Consolidation Findings: oversized-text-field-bricks-the-store (Phase 2)

## Findings

None. The four tasks compose cleanly. Doctor and the store share `jsonl.Lines` and `storage.DecodeTaskLine`. The "store cannot load" predicate in `refreshSuggestion` is built from the same `loadFailures` and `errIncompleteRead` that the JSONL check and `linesUnavailableResult` use, so the two cannot drift apart. Every line-consuming check goes through `linesUnavailableResult`. No scaffolding from 2-1 survives superseded by 2-3 or 2-4.

## Comment Corrections

- internal/doctor/jsonl_reader.go:27 — restates the declaration; the name and its one use in `WithTasksOpener` already say it
  OLD: // tasksOpenerKeyType is an unexported type for the context key carrying the
// function that opens tasks.jsonl.
  NEW:

## Spec Defects

### S1: §5.3's "never reports `tasks.jsonl not found`" cannot hold for the cache check, which §5.5 leaves unchanged
- **Claim**: §5.3: "It never reports a pass over the lines it did read, and never reports `tasks.jsonl not found`." The same section says "Once the ceiling is gone, no file on disk makes the reader fail this way". §5.5 says "Everything else about the cache check is unchanged."
- **Observed**: The only real trigger for an incomplete read is an I/O error on `tasks.jsonl`, such as a failing disk or a dropped network mount. The cache check reads the same file separately with `os.ReadFile` (`internal/doctor/cache_staleness.go:28`). When that error persists, it reports `✗ Cache: tasks.jsonl not found or unreadable: read …/tasks.jsonl: <err>` with `→ Run tick init or verify .tick directory` (`internal/doctor/cache_staleness.go:30-36`). That is the text §5.3 rules out, and it advises initialising a store that already exists (`tick init` refuses: `internal/cli/init.go:23`). The §8.4 test seam cannot show this. `WithTasksOpener` (`internal/doctor/jsonl_reader.go:33-42`) replaces only `ScanJSONLines`' open, so under the seam the cache check hashes the intact real file. As a result, `internal/cli/doctor_incomplete_read_test.go:100-102` pins `✓ Cache: OK`, a state the real trigger cannot produce, and `internal/cli/doctor_incomplete_read_test.go:109-117` passes for the same reason.
- **Read**: Genuinely open. One option is to scope the claim to the checks that consume the file's lines: the spec is stale, and the cache check's hedged "not found or unreadable" stands under §5.5's freeze. The other is to bring the cache check's unreadable branch under §5.3, a code change that overrides §5.5's "unchanged". The spec makes both claims, and they only conflict on the trigger the seam simulates.
