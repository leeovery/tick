# Consolidation Findings: oversized-text-field-bricks-the-store (Phase 4)

## Findings

None. The phase's one task retired `JSONLinesKey` everywhere (no Go, README or CLAUDE.md reference is left) and routes every line consumer through `getJSONLines`. That covers the ten line-consuming checks, `refreshSuggestion` (`internal/doctor/cache_staleness.go:87`) and `getTaskRelationships` (`internal/doctor/jsonl_reader.go:98`). All of them read the outcome `RunDoctor` stores (`internal/cli/doctor.go:28-29`), and `RunAll` passes that ctx unchanged (`internal/doctor/doctor.go:101-105`). No scaffolding survives.

The one bank entry was dropped. It proposed retiring or rerouting `ParseTaskRelationships` (`internal/doctor/task_relationships.go:79-91`). That function has had no production caller since cb301f60 (2026-02-13), which removed its preload from `RunDoctor`. Its `context.Background()` scan dates from task 2-3 (5eefe7c9), and this phase did not touch the file. It bypassed the lines-only key before this phase exactly as it now bypasses `WithScan`. The subject is wholly pre-existing code, and the failure depends on a caller that does not exist.

## Comment Corrections

- internal/cli/doctor_incomplete_read_test.go:139-140 — restates the eight-line body (the count, the `faultOn` branch, the real `os.Open`) and hand-traces "line 3" from the fixture the caller passes. It carries nothing the code does not.
  OLD:
  // countingOpener opens tasks.jsonl for real, except that each open faultOn
  // reports true for is cut short at line 3 with boom. It counts every open.
  NEW:

- internal/doctor/jsonl_reader_test.go:207 — restates the body. The `t.Errorf` in the opener says it fails the test on open.
  OLD:
  // ctxWithoutOpening returns a context whose tasks.jsonl opener fails the test.
  NEW:
