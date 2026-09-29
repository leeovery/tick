TASK: Doctor Fails A Read Of Tasks.jsonl It Cannot Complete (oversized-text-field-bricks-the-store-2-3, tick-28f4a7)

ACCEPTANCE CRITERIA:
- A read of `tasks.jsonl` that returns lines 1 and 2 and then stops with an error at line 3, driven through a test seam over the reader and run through `RunDoctor`: doctor exits 1, and the report names line 3 and says the file could not be read in full (§5.3, §8.4)
- Same run: every check that consumes the file's lines fails with that read error in place of its normal verdict, one failure each, the same shape as today's failure when the file will not open. None of them passes over lines 1 and 2, even where those lines would pass every check (§5.3, §8.4)
- Same run: no result reports `tasks.jsonl not found` (§5.3, §8.4)
- A `.tick` directory whose `tasks.jsonl` is missing: each check that consumes the file's lines still reports `tasks.jsonl not found`, as today (§5.3)

STATUS: complete

SPEC CONTEXT: §1.2 names doctor's second defect: its reader stopped at an unreadable line and returned the prefix with no error, so every check ran over a partial file and doctor passed a store no command could open. §5.3 requires that a read which opened the file but was cut short fails, naming the line and saying the file could not be read in full; every line-consuming check fails with that read error (one error-severity result per check, Sequence uniqueness included), never a pass and never `tasks.jsonl not found`; the not-found result stays for a file that cannot be opened. The Cache check reads the bytes itself and is unchanged (§5.5; corrigendum 2026-09-29 scopes the "never not found" claim to line-consuming checks). §8.4 requires the path be driven through a test seam over the reader and exercised through `RunDoctor`.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/doctor/jsonl_reader.go:25 — `errIncompleteRead` sentinel ("tasks.jsonl could not be read in full")
  - internal/doctor/jsonl_reader.go:31-40 — `WithTasksOpener` / `openTasks`, the test seam over the reader's open
  - internal/doctor/jsonl_reader.go:49-59 — `ScanJSONLines`: open failure wrapped as `open tasks.jsonl: %w`; a mid-read failure from `jsonl.Lines` wrapped as `%w: %w` with `errIncompleteRead`, returning no lines
  - internal/jsonl/lines.go:33-36 — the shared reader yields `read line N: <err>` naming the line it could not read
  - internal/cli/doctor.go:28-29 — `RunDoctor` stores the scan outcome, error included, via `WithScan`, so no check re-reads the file
  - internal/doctor/jsonl_reader.go:86-103 — `WithScan` / `getJSONLines` / `getTaskRelationships` hand the stored error to every check; the no-scan fallback re-scans through the same opener seam
  - internal/doctor/helpers.go:30-41 — `linesUnavailableResult`: read error as an error-severity failure carrying the check's name when `errors.Is(err, errIncompleteRead)`, otherwise `fileNotFoundResult`
  - Error branches of all ten line-consuming checks route through `linesUnavailableResult`: jsonl_syntax.go:22, id_format.go:24, duplicate_id.go:28, orphaned_parent.go:20, orphaned_dependency.go:20, self_referential_dep.go:21, dependency_cycle.go:25, child_blocked_by_parent.go:25, parent_done_open_children.go:24, duplicate_seq.go:29 (all under internal/doctor/). The Cache check (internal/doctor/cache_staleness.go:28-37) is left alone, as §5.5 requires.
- Notes: The implementation differs from the original task commit (5eefe7c9), where each check re-read the file through the opener in the per-check fallback. Later analysis task 4-1 (774098c1) had `RunDoctor` store the scan outcome, error included, so the checks share one read. This is a sound change and no worse than what was written. A fault on only the first read can no longer let a later re-read pass. The "per-check fallback is covered" intent of §8.4 now means the stored-scan branch that every check takes under `RunDoctor`, and the RunDoctor-level tests exercise it. The detail rendered is `tasks.jsonl could not be read in full: read line 3: device went away`, which names the line and says the file could not be read in full. Every result is SeverityError, so Sequence uniqueness gets error severity here even though its own findings are warnings, matching the not-found shape. A file that cannot be opened, missing or otherwise, still maps to `tasks.jsonl not found`, as the spec says.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/doctor_incomplete_read_test.go:68-137 `TestDoctorIncompleteRead` drives `RunDoctor` with a `WithTasksOpener` seam. `cutShortReader` hands over two valid task lines and then fails with `device went away`, which `bufio.Reader.ReadBytes` surfaces at line 3. The store is set up with `setupListedProject`, which builds the cache from those two lines, so a normal run would pass every check.
    - AC1 (:76-87): exit code 1, stdout contains the full detail naming line 3 and "could not be read in full".
    - AC2 (:89-107): for each of the ten line-consuming checks, the report holds exactly one line, `✗ <check>: <detail>`. That rules out any pass and any second result. Cache stays `✓ Cache: OK`, and the summary is `10 issues found.`
    - AC3 (:109-117): stdout never contains `tasks.jsonl not found`.
    - AC4 (:119-137): with the file removed and no seam, each of the ten checks reports exactly `✗ <check>: tasks.jsonl not found`.
  - internal/cli/doctor_incomplete_read_test.go:150-243 `TestDoctorSharesOneScan` (added by 4-1) shows that a fault on only the first open still fails every check, and that `tasks.jsonl` is opened exactly once.
  - internal/doctor/jsonl_reader_test.go:238-252 and :298-312 show that the stored read error is returned without reopening the file.
  - Each test fails if the behaviour regresses. If the error were dropped, the checks would pass and the AC2 assertion would break. If it were mapped to not-found, AC2 and AC3 would break. If the whole shared result were downgraded to a warning, the exit-code assertion would break.
- Notes: Not over-tested. Each subtest targets one criterion, and the setup is minimal.

CODE QUALITY:
- Project conventions: Followed. The code uses stdlib `testing` with `t.Run` subtests and `t.Helper`, `%w` error wrapping, and ctx as the first parameter of `RunDoctor`. The seam is carried in the context, like the package's existing scan hand-off (`WithScan`).
- SOLID principles: Good. One helper (`linesUnavailableResult`) decides between the read-error and not-found results for all ten checks, so they cannot drift apart.
- Complexity: Low
- Modern idioms: Yes. It uses multi-`%w` wrapping, `iter.Seq2` range-over-func, `strings.SplitSeq` and `slices.Equal`.
- Readability: Good. Comments on `linesUnavailableResult`, `ScanJSONLines`, `WithTasksOpener`, `WithScan` and `jsonl.Lines` match the code.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
