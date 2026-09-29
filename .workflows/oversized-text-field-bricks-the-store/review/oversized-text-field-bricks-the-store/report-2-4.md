TASK: No Tick Rebuild Advice While The Store Cannot Load (oversized-text-field-bricks-the-store-2-4, tick-87cccf)

ACCEPTANCE CRITERIA:
- A store with a stale cache and a line that fails to load: the Cache check still fails, reporting the stale cache. Its suggestion points at fixing the lines the JSONL syntax check names, and no suggestion in the report names `tick rebuild` (§5.5)
- A store with no `cache.db` and a line that fails to load: the Cache check still reports `cache.db not found`, its suggestion points at fixing the named lines, and no suggestion names `tick rebuild` (§5.5)
- A two-task store with one added line that is whitespace-only, `null`, `[]`, `{}` or a wrong-typed field such as `"priority":"high"`: no suggestion in the `tick doctor` report names `tick rebuild` (§5.1, §5.5, §8.4)
- A read of `tasks.jsonl` that stops partway, driven through Task 2-3's test seam and run through `RunDoctor`, with a stale cache: no suggestion names `tick rebuild` (§5.5)
- A store with a stale cache and a line that fails to load, where that line is then fixed by hand: the next `tick doctor` reports the stale cache with the usual `tick rebuild` advice (§5.5)
- A line that fails to load, with a `cache.db` whose stored hash matches the file's bytes, that line included: the Cache check passes. For the same file bytes and `cache.db`, the Cache check's verdict and details are the ones it gives today; only its suggestion changes (§5.5)
- A line that loads but repeats a value within its own `tags`, `refs` or `blocked_by` list, with a stale cache: the Cache check reports the stale cache with its usual `tick rebuild` advice, and that `tick rebuild` then fails, naming the task and the repeated value (§5.1, Corrigendum 2026-09-28 on §5.1)

STATUS: complete

SPEC CONTEXT: §5.5 requires that while any line fails to load (§5.1) or the read is incomplete (§5.3), no doctor output directs the user to `tick rebuild`; the cache check keeps reporting what it finds (stale / missing is a true finding) but its suggestion points at fixing the lines the JSONL check names. Everything else about the cache check (hashing raw bytes, verdict, details) is unchanged. The 2026-09-28 corrigendum on §5.1 keeps repeated list values out of doctor's scope: such a line loads, the cache check keeps its rebuild advice, and the rebuild fails naming task and value. The 2026-09-29 corrigenda on §5.3 confirm the cache check reads bytes itself and reports its own unreadable result when the bytes cannot be read.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/doctor/cache_staleness.go:50, :63, :74 — the three suggestion sites (cache.db not found, hash query failure, hash mismatch) now call refreshSuggestion; Details strings and Passed/Severity are untouched.
  - internal/doctor/cache_staleness.go:84-92 — refreshSuggestion returns the fix-lines advice when getJSONLines reports errIncompleteRead or loadFailures(lines) is non-empty, otherwise the original "Run `tick rebuild` to refresh cache".
  - internal/doctor/jsonl_syntax.go:37-45 — loadFailures is the same predicate the JSONL syntax check reports from, so "the lines the JSONL check names" and the cache check's trigger cannot disagree.
  - internal/cli/doctor.go:28-29 — RunDoctor scans once and passes the outcome via WithScan, so refreshSuggestion reuses the shared scan rather than reopening the file.
- Notes: The judgment is lazy (only on the failing branches), so a fresh cache still passes regardless of line loadability (criterion 6). The early `tasks.jsonl not found or unreadable` return at cache_staleness.go:28-37 never named rebuild and is unchanged. A non-incomplete scan error (open failure) falls through to rebuild advice; in practice the cache check's own os.ReadFile would have already failed in that case, so no reachable path advises rebuild while the store cannot load. Criterion 7 settles by reading: the dependencies/task_tags/task_refs tables carry (task_id, value) primary keys (internal/storage/cache.go:32-49) and the insert errors name task and value (internal/storage/cache.go:215, :221, :227), surfaced by Store.Rebuild's "failed to rebuild cache: %w" (internal/storage/store.go:262-263).

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/doctor_rebuild_advice_test.go:90-101 — criterion 1 (stale cache + `null`): exact Cache line and suggestion, no rebuild advice anywhere.
  - :103-110 — criterion 2 (no cache.db + `null`).
  - :112-141 — criterion 3, all five fixtures, each confirmed to fail `tick list` first, with exact Cache report asserted.
  - :143-157 — criterion 4 through runDoctorWithOpener/cutShortReader and RunDoctor, stale cache.
  - :159-173 — criterion 5, hand fix followed by rebuild advice, with a before-fix precondition.
  - :175-215 — criterion 7, tag/ref/blocker repeats: rebuild advice kept, rebuild exits 1 naming task and value.
  - internal/doctor/cache_staleness_test.go:530-541 — criterion 6 pass with a matching hash over a failing line; :543-568 compares the whole CheckResult for stale hash, missing key and missing cache.db, so details/verdict/severity are pinned and only the suggestion differs.
  - internal/cli/doctor_incomplete_read_test.go:201-220 additionally covers missing and stale caches when only the first read faults, which is the case that would catch refreshSuggestion rescanning instead of using the shared scan.
- Notes: Every assertion is exact-match on the Cache suggestion, so reverting refreshSuggestion to a constant would fail the suite. Minor overlap: cache_staleness_test.go:570-580 ("it keeps the rebuild advice when every line loads") asserts the same thing, with the same fixture shape, as the updated :214-229; harmless.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, t.Run "it ..." subtests, t.Helper on helpers, context-carried seams consistent with WithScan/WithTasksOpener)
- SOLID principles: Good — the loadability judgment is reused from the JSONL check rather than re-implemented.
- Complexity: Low
- Modern idioms: Yes (errors.Is on the wrapped sentinel, strings.SplitSeq/CutPrefix in test helpers)
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
