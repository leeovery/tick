# Phase 2: Doctor passes only a store every command can open — 4 tasks

## oversized-text-field-bricks-the-store-2-1

### Task 2-1: Doctor reads tasks.jsonl through the shared line reader

**Problem**: Doctor still reads `tasks.jsonl` with its own `bufio.Scanner` (`internal/doctor/jsonl_reader.go:37`), which Phase 1 left in place. That scanner has the 65,536-byte ceiling the store used to have, and doctor never checks its error. When it meets a long line it stops and returns the lines before it with no error. Every check then runs over that prefix as if it were the whole store. Doctor passes a store it never finished reading, and it can report false errors about references into the part it never read, such as `Orphaned dependencies` naming a task that exists after the cut. Doctor's definition of a line also differs from the store's. It skips any whitespace-only line (`:44`), while the store reads that line and fails to load it. So doctor passes a store that `tick list` refuses.

**Solution**: Move doctor's reading of `tasks.jsonl` onto Phase 1's shared line reader (Task 1-1). Doctor then reads the same lines as the store, skips the same lines and numbers them the same way, with no line-length ceiling.

**Outcome**: Doctor reads every line of `tasks.jsonl`, whatever its length, and agrees with the store line for line: the same lines are read and the same line numbers are reported.

**Acceptance Criteria**:
- [ ] An otherwise valid store with a fresh cache and a task line over 64 KiB mid-file, where tasks before the long line name tasks after it as parent or blocker: `tick doctor` exits 0 and reports `No issues found`, with no orphaned-reference error for the tasks after the long line. Today doctor's read stops at the long line (§1.2, §5.4, §8.4)
- [ ] A store whose line 2 is empty and whose line 3 holds only whitespace: `tick doctor` exits 1 with a `JSONL syntax` failure naming line 3, the line `tick list` names in its own failure. The empty line is skipped but still counted. Today doctor skips the whitespace-only line and its JSONL check passes (§2.2, §8.1)
- [ ] A fixture carrying CRLF-terminated lines, empty lines and a whitespace-only line among valid tasks: doctor and the store read the same lines and report the same line numbers. Doctor's only JSONL failure names the line that `tick list` names (§2.2, §8.1)
- [ ] A valid store whose lines end in CRLF, with a fresh cache: `tick doctor` reports `No issues found`, as it does for the same store with LF endings (§2.2, §8.1)
- [ ] An otherwise valid store whose file ends `…}\n\r`: `tick list` succeeds, and a following `tick doctor` reports `No issues found`. Both treat the final `\r` as an empty, skipped line (§2.2, Corrigendum 2026-09-28 on §2.2)
- [ ] Doctor reads `tasks.jsonl` only through the shared line reader. No reader of `tasks.jsonl`, in doctor or in the CLI's doctor command, keeps its own scanner or a line-length ceiling (§2.1, §2.2; deferred from Task 1-1)

**Do**:
- `internal/doctor/jsonl_reader.go`: `ScanJSONLines` reads through the shared line reader instead of its `bufio.NewScanner` (line 37). Doctor's whitespace-only skip (the `TrimSpace(text) == ""` test at line 44) gives way to the store's rule: only a line that is empty once its terminator is removed is skipped (§2.2).
- Every doctor read of the file's lines reaches the file through `ScanJSONLines` today, so they all move with it. Those reads are `RunDoctor`'s pre-scan (`internal/cli/doctor.go:30`), the per-check fallback `getJSONLines` (`internal/doctor/jsonl_reader.go`) and `ParseTaskRelationships` (`internal/doctor/task_relationships.go:82`).
- Revisit `TestScanJSONLines` (`internal/doctor/jsonl_reader_test.go`), whose `it skips whitespace-only lines` no longer holds. Also revisit `TestJsonlSyntaxCheck`'s `it returns passing result when file contains only whitespace-only lines` (`internal/doctor/jsonl_syntax_test.go`) (§8.6).

**Context**:
> §2.2: "The store and doctor read `tasks.jsonl` through one shared line reader, so the two cannot disagree about what the file holds and no future reader of the store can reintroduce a ceiling." The reader carries the store's line definition. A line ends at `\n`, and one `\r` immediately before it is stripped. A final line with no trailing newline is still read, and one `\r` at its end is stripped. A line that is empty once its terminator is removed is skipped. Lines are numbered from 1, and skipped lines count toward the numbering. A whitespace-only line is not skipped. "Doctor's current rule of skipping any whitespace-only line … gives way to the store's …, because doctor's verdict must match what commands actually do (§5)."
>
> §5.4: "Because the whole file is now read, tasks after a long line are no longer invisible to them, so a mid-file oversized line produces no false orphan error." The relationship and hierarchy checks keep skipping a line that is not a JSON object or has no string `id` (`internal/doctor/task_relationships.go:26`).
>
> Doctor's Cache check compares the hash of the file's bytes with the one stored in `cache.db`. A `No issues found` scenario therefore needs a cache built from the file as it stands, for example by running `tick list` first.
>
> Sequencing within the phase: under this task a whitespace-only line reaches the existing JSONL syntax check, which fails it as invalid JSON. Task 2-2 changes that check to judge each line by loading it as a task and to report the loader's reason. Task 2-3 covers a read that stops partway (§5.3).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §2.1, §2.2, §5.4, §8.1, §8.4, §8.6, Corrigenda (2026-09-28)

## oversized-text-field-bricks-the-store-2-2

### Task 2-2: JSONL syntax check judges each line by loading it as a task

**Problem**: Doctor's JSONL check asks only whether a line is valid JSON, but the store needs a line that loads as a task. So `null`, `[]`, `{}` and a wrong-typed field such as `"priority":"high"` each make every command fail while doctor's JSONL check passes them. This was measured on current main with a built binary, each line added to a two-task store: `tick list` exits 1 naming the line, while `tick doctor` prints `✓ JSONL syntax: OK`, flags only the stale cache, and advises `tick rebuild`. A timestamp the loader cannot parse slips through the same way. Agents run `tick doctor` as a health check, and a false "healthy" tells an agent a store is fine when no command can open it.

**Solution**: Judge every line the shared reader returns by loading it as a task, exactly as every tick command loads it: the same decoding, timestamp parsing included. A line that does not load fails the JSONL check, one failure per line, naming the line and giving the loader's reason. The check keeps its name and its hand-fix suggestion.

**Outcome**: The JSONL check fails exactly the lines no tick command can load, and each line it fails is one `tick list` also fails on, named by the same line number. A valid store still reports `No issues found`.

**Acceptance Criteria**:
- [ ] A two-task store with one added line, where that line is whitespace-only, `null`, `[]`, `{}`, malformed JSON, a wrong-typed field such as `"priority":"high"`, or a `created` timestamp that does not parse: `tick doctor` exits 1 with a `JSONL syntax` failure that names that line and gives the loader's reason in place of today's `invalid JSON`. `tick list` on the same store fails, naming the same line (§5.1, §5.2, §8.4)
- [ ] A store with several lines that fail to load: the JSONL check reports one failure per line, each naming its own line (§5.1)
- [ ] A line that loads but carries a status or type outside the allowed values, or a priority outside 0–4: the JSONL check passes it, and doctor adds no enum or range check (§5.1)
- [ ] A line that loads but repeats a value within its own `tags`, `refs` or `blocked_by` list: the JSONL check passes it, and doctor adds no check for the repeat (§5.1, Corrigendum 2026-09-28 on §5.1)
- [ ] A valid store with a fresh cache: `tick doctor` reports `No issues found` (§5.1, §8.4)
- [ ] The check keeps its name and its suggestion. It is titled `JSONL syntax` in doctor's output whether it passes or fails, and each failure's suggestion stays a hand fix of the named line. The README's doctor section (its "Checks for:" list at `README.md:396` and the sample tested against it) and `tick help doctor` are unchanged (§5.2)
- [ ] A line that is not a JSON object (such as `[]` or `null`) or whose `id` is not a string: the relationship and hierarchy checks skip it as they do today, and the JSONL check reports it (§5.4)

**Do**:
- `internal/doctor/jsonl_syntax.go`: `JsonlSyntaxCheck.Run` fails a line that does not load as a task the way commands load it, in place of today's `json.Valid` test. Its failure detail gives the loader's reason where it says `invalid JSON` today (§5.1, §5.2).
- The check name `JSONL syntax` stays. `README.md:396` and `internal/cli/help.go:217` are unchanged (§5.2).
- Revisit `TestJsonlSyntaxCheck` (`internal/doctor/jsonl_syntax_test.go`), including `it does not validate JSON field names or values — only syntax` (§8.6).

**Context**:
> §5.1: "Every line doctor reads through the shared reader (§2.2) must load as a task exactly as every tick command loads it — the same decoding, timestamp parsing included. A line that does not load fails the JSONL check, one failure per line, naming the line and giving the loader's reason."
>
> Commands load a line in `storage.ParseJSONL` (`internal/storage/jsonl.go`) by decoding it into `task.Task`. Its `UnmarshalJSON` (`internal/task/task.go:109`) decodes into `taskJSON`, then parses `created`, `updated` and `closed`. `null` and `{}` fail on the empty `created` timestamp, `[]` fails in decoding, and a whitespace-only line fails with `unexpected end of JSON input`. For a wrong-typed field the store reports, for example, `failed to parse tasks.jsonl: line 2 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int` (§3, Task 1-6).
>
> §5.1: "Validation the loader does not itself enforce — enum membership (status, type) and value ranges (priority 0–4) — stays at write time and is not doctor's." A value repeated within one task's own `tags`, `refs` or `blocked_by` list also loads, but the cache rebuild every command runs refuses it on that list's uniqueness constraint. No tick write produces one; only a hand edit or another tool does, and it is kept outside this fix (Corrigendum 2026-09-28 on §5.1). What the Cache check advises for such a store is Task 2-4.
>
> §5.2: the check stays `JSONL syntax` because the v1 doctor-validation specification keeps that title while widening what it covers. Renaming it would change user-visible output and a tested README sample with no change in behaviour. `internal/cli/readme_samples_test.go` reads the doctor section's "Checks for:" list. The suggestion today is `Manual fix required`.
>
> Existing fixtures: the `RunDoctor` tests in `internal/cli/doctor_test.go` write task lines with no `created` or `updated`, such as `{"id":"tick-aaa111","title":"Test"}`. Those lines do not load as a task. Tests there that expect a passing JSONL check or `No issues found` need lines that load.
>
> Suggestions that name `tick rebuild` while a line fails to load are Task 2-4 (§5.5).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §1.4, §3, §5.1, §5.2, §5.4, §8.4, §8.6, Corrigenda (2026-09-28)

## oversized-text-field-bricks-the-store-2-3

### Task 2-3: Doctor fails a read of tasks.jsonl it cannot complete

**Problem**: When doctor's read of `tasks.jsonl` stops before the end of the file, doctor drops the reader's error and runs every check over the lines it did read. It passes a store no other command can open (§1.2). Surfacing that error through the existing per-check fallback would not help. `RunDoctor` keeps the pre-scanned lines only when the scan returns no error, and every check that then re-reads the file answers any error with `tasks.jsonl not found` (`fileNotFoundResult`, `internal/doctor/helpers.go:14`). That is false for a file that opened.

**Solution**: Report a read that opened the file but stopped partway as a failure of its own. Every check that consumes the file's lines fails with that read error in place of its normal verdict. The error names the line doctor could not read and says the file could not be read in full. The not-found result stays for a file that cannot be opened, and nothing else.

**Outcome**: A doctor run whose read of `tasks.jsonl` is cut short fails, naming the line it could not read. It never reports a pass over the lines it did read, and it never claims the file is missing. A missing file still reports `tasks.jsonl not found`.

**Acceptance Criteria**:
- [ ] A read of `tasks.jsonl` that returns lines 1 and 2 and then stops with an error at line 3, driven through a test seam over the reader and run through `RunDoctor`: doctor exits 1, and the report names line 3 and says the file could not be read in full (§5.3, §8.4)
- [ ] Same run: every check that consumes the file's lines fails with that read error in place of its normal verdict, one failure each, the same shape as today's failure when the file will not open. None of them passes over lines 1 and 2, even where those lines would pass every check (§5.3, §8.4)
- [ ] Same run: no result reports `tasks.jsonl not found` (§5.3, §8.4)
- [ ] A `.tick` directory whose `tasks.jsonl` is missing: each check that consumes the file's lines still reports `tasks.jsonl not found`, as today (§5.3)

**Do**:
- The work lives in three places. The first is `RunDoctor` (`internal/cli/doctor.go`), whose pre-scan puts the lines in the context only when the scan returns no error. The second is `getJSONLines` and `getTaskRelationships` (`internal/doctor/jsonl_reader.go`), the per-check fallback that re-reads the file when the context holds no lines. The third is each line-consuming check's error branch, which answers any error with `fileNotFoundResult` (`internal/doctor/helpers.go:14`).
- The not-found result stays, for a file that cannot be opened and nothing else (§5.3).
- The incomplete-read path is driven through a test seam over the reader and exercised through `RunDoctor`, not the reader alone, so the per-check fallback is covered (§8.4).

**Context**:
> §5.3: "If reading `tasks.jsonl` stops before the end of the file — the file opened, but an error cut the read short — doctor reports a failure naming the line it could not read and saying the file could not be read in full. It never reports a pass over the lines it did read, and never reports `tasks.jsonl not found`. Every check that consumes the file's lines fails with that read error in place of its normal verdict, the same shape as today's failure when the file will not open." Today's failure when the file will not open is one error-severity result per check, carrying the check's own name. That includes Sequence uniqueness, whose findings are otherwise warnings.
>
> The checks that consume the file's lines are every check `RunDoctor` registers except Cache: JSONL syntax, ID format, ID uniqueness, Orphaned parents, Orphaned dependencies, Self-referential dependencies, Dependency cycles, Child blocked by parent, Parent done with open children and Sequence uniqueness. The Cache check reads the file's bytes itself and hashes them (`internal/doctor/cache_staleness.go`). Its verdict is unchanged (§5.5).
>
> §5.3: "Once the ceiling is gone, no file on disk makes the reader fail this way, so the path is reached through a test seam over the reader (§8)."
>
> Keeping `tick rebuild` advice out of this report is Task 2-4 (§5.5).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §5.3, §5.5, §8.4

## oversized-text-field-bricks-the-store-2-4

### Task 2-4: No `tick rebuild` advice while the store cannot load

**Problem**: While a line of `tasks.jsonl` fails to load, doctor's Cache check still finds the cache stale or missing and advises `` Run `tick rebuild` to refresh cache ``. That rebuild fails the same way until the line is fixed. Measured on current main, each line that makes every command fail gets a doctor report that flags the stale cache and advises `tick rebuild` (§5.1). An agent that follows the advice runs the one command the bad line guarantees will fail, and learns nothing about the line.

**Solution**: While any line fails to load (§5.1) or the read is incomplete (§5.3), the Cache check still reports what it finds, but its suggestion points at fixing the lines the JSONL check names instead of at `tick rebuild`. Everything else about the Cache check is unchanged.

**Outcome**: No doctor output directs the user to `tick rebuild` while the store cannot load. Once the named lines are fixed, the next doctor run gives the usual rebuild advice if the cache is still stale.

**Acceptance Criteria**:
- [ ] A store with a stale cache and a line that fails to load: the Cache check still fails, reporting the stale cache. Its suggestion points at fixing the lines the JSONL syntax check names, and no suggestion in the report names `tick rebuild` (§5.5)
- [ ] A store with no `cache.db` and a line that fails to load: the Cache check still reports `cache.db not found`, its suggestion points at fixing the named lines, and no suggestion names `tick rebuild` (§5.5)
- [ ] A two-task store with one added line that is whitespace-only, `null`, `[]`, `{}` or a wrong-typed field such as `"priority":"high"`: no suggestion in the `tick doctor` report names `tick rebuild`. Today the report advises `tick rebuild` for each (§5.1, §5.5, §8.4)
- [ ] A read of `tasks.jsonl` that stops partway, driven through Task 2-3's test seam and run through `RunDoctor`, with a stale cache: no suggestion names `tick rebuild` (§5.5)
- [ ] A store with a stale cache and a line that fails to load, where that line is then fixed by hand: the next `tick doctor` reports the stale cache with the usual `tick rebuild` advice (§5.5)
- [ ] A line that fails to load, with a `cache.db` whose stored hash matches the file's bytes, that line included: the Cache check passes. For the same file bytes and `cache.db`, the Cache check's verdict and details are the ones it gives today; only its suggestion changes (§5.5)
- [ ] A line that loads but repeats a value within its own `tags`, `refs` or `blocked_by` list, with a stale cache: the Cache check reports the stale cache with its usual `tick rebuild` advice, and that `tick rebuild` then fails, naming the task and the repeated value (§5.1, Corrigendum 2026-09-28 on §5.1)

**Do**:
- `internal/doctor/cache_staleness.go`: the Cache check is the only doctor check whose suggestions name `tick rebuild` (its `cache.db not found` and stale results, lines 49, 62 and 73). While any line fails to load or the read is incomplete, those suggestions point at fixing the lines the JSONL check names. The check's verdict and details, from hashing the raw file bytes, are unchanged (§5.5).

**Context**:
> §5.5: "While any line fails to load (§5.1) or the read is incomplete (§5.3), no doctor output directs the user to `tick rebuild`, which fails the same way until the line is fixed. The cache check still reports what it finds, since a stale or missing cache is a true finding. But its suggestion points at fixing the lines the JSONL check names, not at `tick rebuild`. Once those lines are fixed, the next doctor run gives the usual rebuild advice if the cache is still stale." And: "Everything else about the cache check is unchanged. It hashes the raw file bytes and its verdict is correct."
>
> Which lines fail to load is the JSONL check's judgment (Task 2-2), and an incomplete read is Task 2-3's. The Cache check does not consume the file's lines: it reads the file's bytes itself and hashes them. Under Phase 1, a rebuild that cannot parse the store exits 1 and leaves `cache.db` as it was (Task 1-7), but it keeps failing until the line is fixed.
>
> A value repeated within one task's own `tags`, `refs` or `blocked_by` list is not doctor's (Corrigendum 2026-09-28 on §5.1). The line loads and passes the JSONL check, so no line fails to load and the Cache check keeps its usual advice. The rebuild that advice leads to fails naming the task and the value, for example `failed to insert tag tick-fa204e -> x: … UNIQUE constraint failed: task_tags.task_id, task_tags.tag`. The dependency and ref inserts name the task and value the same way (`internal/storage/cache.go:215`, `:227`).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.4, §4, §5.1, §5.3, §5.5, §8.4, Corrigenda (2026-09-28)
