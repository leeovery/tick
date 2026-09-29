TASK: Doctor Shares One Scan Outcome, Failure Included, With Every Check (oversized-text-field-bricks-the-store-4-1, tick-c76fe7)

ACCEPTANCE CRITERIA:
1. Given a reader fault that fails only the first read of `tasks.jsonl`, cutting it short at line 3 after lines 1 and 2, `tick doctor` fails each of the ten line-consuming checks once with `tasks.jsonl could not be read in full: read line 3: <fault>`, and none of them passes (§5.3)
2. Given that same first-read-only fault and a stale or missing cache, the Cache check reports the stale or missing cache and suggests fixing the lines the JSONL syntax check names. No line of the report names `tick rebuild` (§5.5)
3. One `tick doctor` run opens `tasks.jsonl` through the line reader exactly once, whether the store is readable, the fault fails every read, or the fault fails only the first
4. A fault that fails every read still reports as it does today: ten read-error failures, `✓ Cache: OK`, and `10 issues found.` (§5.3)
5. With `tasks.jsonl` missing, every line-consuming check still reports `tasks.jsonl not found` (§5.3)
6. The line consumers judge a stored scan outcome as stored, without opening `tasks.jsonl`: its lines when the scan succeeded, its read error when it failed
7. A check run on its own, with no stored outcome, reads `tasks.jsonl` itself and judges those lines

STATUS: complete

SPEC CONTEXT: §5.3 requires every line-consuming check to fail with "that read error" when the read of tasks.jsonl stops partway, and reserves `tasks.jsonl not found` for a file that cannot be opened. §5.5 forbids any doctor output directing the user to `tick rebuild` while the read is incomplete; the Cache check keeps its verdict (hash of raw bytes, frozen by §5.5 and the 2026-09-29 corrigendum on §5.3) but its suggestion points at fixing the named lines. Both only hold across a whole report if every check and the Cache check's suggestion judge the same single read, which this task secures.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/doctor.go:28-29 — RunDoctor scans once and stores the outcome unconditionally via `doctor.WithScan(ctx, lines, err)`; the `if err == nil` success-only write is gone.
  - internal/doctor/jsonl_reader.go:77-88 — unexported `scanKeyType` / `scanOutcome{lines, err}` and exported `WithScan`; the lines-only `JSONLinesKey` is retired from production code and tests (no occurrence remains in internal/cli/doctor.go, internal/doctor/jsonl_reader.go or internal/doctor/jsonl_reader_test.go, the three files the plan enumerated; any stray reference elsewhere would fail to compile since the declaration is gone).
  - internal/doctor/jsonl_reader.go:90-95 — `getJSONLines` returns the stored lines and error as stored whenever an outcome is present, and calls `ScanJSONLines` only when none is stored.
  - internal/doctor/jsonl_reader.go:97-103 — `getTaskRelationships` routes through `getJSONLines`, so the six relationship checks see the stored outcome.
  - Consumers unchanged and all reaching the outcome through `getJSONLines`: jsonl_syntax.go:20, id_format.go:22, duplicate_id.go:26, duplicate_seq.go:27, orphaned_parent.go:18, orphaned_dependency.go:18, self_referential_dep.go:19, dependency_cycle.go:23, child_blocked_by_parent.go:23, parent_done_open_children.go:22, plus `refreshSuggestion` at cache_staleness.go:87.
  - Unchanged as required: `linesUnavailableResult` (internal/doctor/helpers.go:30) and the Cache check's own byte read (internal/doctor/cache_staleness.go:28).
- Notes: Reading settles every criterion. RunDoctor calls `ScanJSONLines` once and always stores the result, `getJSONLines` never re-scans when an outcome is present, and the Cache check's `os.ReadFile` does not go through the line reader or the opener seam. So one run opens tasks.jsonl through the line reader exactly once in all three fault modes (criterion 3). A stored incomplete-read error reaches all ten checks and `refreshSuggestion`, which returns the fix-lines advice (`errors.Is(err, errIncompleteRead)`, cache_staleness.go:88). A stored open error still maps to `fileNotFoundResult` through `linesUnavailableResult` (criterion 5). With the file missing, the Cache check returns early from its own ReadFile failure and never reaches `refreshSuggestion`.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/doctor_incomplete_read_test.go:150-243 `TestDoctorSharesOneScan` uses a counting opener (`countingOpener`, :139). With a first-read-only fault it asserts that each of the ten checks has exactly one report line with the read-error detail, and no pass line (:158-173; criterion 1). With a first-read-only fault it covers both missing and stale cache (:175-221), asserting the exact Cache line, the fix-lines suggestion line, and that no report line contains `tick rebuild` (criterion 2). It asserts `opens == 1` for readable, every-read-fault and first-read-only-fault stores (:223-242; criterion 3). If `getJSONLines` re-scanned on a stored error, the first-only cases would see successful re-reads through `os.Open` and pass five or more checks, and the opens count would be 11 or 12. Every one of these tests would fail.
  - internal/cli/doctor_incomplete_read_test.go:89-107 still pins the every-read-fault report: ten failures, `✓ Cache: OK`, `10 issues found.` (criterion 4). :119-136 pins `tasks.jsonl not found` for all ten checks with the file missing (criterion 5).
  - internal/doctor/jsonl_reader_test.go:222-272 and :274-332. The preloading subtests moved to `WithScan` (:223, :275). New subtests store an error and assert it comes back via `errors.Is` with nil lines (:238, :298). All four stored-outcome subtests run under `ctxWithoutOpening` (:207), which fails the test if tasks.jsonl is opened (criterion 6). The fallback subtests (:254, :314) keep asserting the no-outcome scan (criterion 7). The existing per-check unit tests run checks without a stored outcome and exercise the fallback throughout.
- Notes: No redundancy worth reporting. The first-only fault tests overlap in shape with the every-read-fault tests in TestDoctorIncompleteRead and TestDoctorRebuildAdvice (internal/cli/doctor_rebuild_advice_test.go:143), but each fault mode tests different behaviour.

CODE QUALITY:
- Project conventions: Followed. Context-value DI matches the existing `WithTasksOpener` seam. Tests use stdlib `testing`, `t.Run` subtests, `t.Helper()` and `t.TempDir()`, with "it does X" naming.
- SOLID principles: Good. The outcome type is unexported and only settable through `WithScan`, which takes lines and error together, so no caller can store a success-only outcome again.
- Complexity: Low
- Modern idioms: Yes
- Readability: Good. Doc comments on `WithScan`, `refreshSuggestion` and `cutShortReader` hold against the code, and none references a process artifact.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
