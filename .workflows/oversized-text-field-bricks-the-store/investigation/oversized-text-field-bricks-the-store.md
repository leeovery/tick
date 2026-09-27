# Investigation: Oversized Text Field Bricks The Store

## Symptoms

### Problem Description

**Expected behavior:**
Either a task record is refused on the write path with a clear error before it can make the store unreadable, or — once written — every store-reading command can still read it. `tick doctor` reports a store it could not fully read as unhealthy.

**Actual behavior:**
A task record whose JSONL line passes ~64 KiB is accepted and written (exit 0, new ID printed). From that moment every command that reads the store fails, `tick rebuild` included, so the tool offers no route back to a readable project. `tick doctor` meanwhile reports the project healthy.

### Manifestation

- `tick create "big" --description <70000 chars> --quiet` exits 0 and prints the new ID. The record lands in `.tick/tasks.jsonl` as a single 70,146-byte line.
- **Without `--quiet`** (the default for agents), the same `create` — or an `update` that grows a record past the ceiling — **exits 1** with `Error: failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long`, **yet the record was written**. The write commits; the read-back that renders the task detail is what fails. The caller is told the command failed when it succeeded, and the error blames parsing rather than the text it just submitted. (Measured on current main, 2026-09-27.)
- The SQLite cache is rebuilt from those same bytes, so the freshness hash matches and nothing looks wrong at write time.
- The next `tick list`, `tick stats`, `tick rebuild`, `tick show <any id>` — and every mutating command, including `update`/`remove`/`cancel` of the oversized task itself — fail with:
  `Error: failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long`
  The message names neither the line nor the task.
- `tick doctor` prints `✓ JSONL syntax: OK` and `No issues found`, exit 0, on the same project, as long as the oversized record is the last line and no `rebuild` has run. Cache-staleness passes too, since the cache genuinely was rebuilt from the bytes on disk.
- **`tick rebuild` deletes `cache.db` before it fails to parse**, so after one failed rebuild doctor reports `✗ Cache: cache.db not found` and suggests `Run tick rebuild to refresh cache` — the one command that cannot succeed.
- **When the oversized record is not the last line**, doctor checks only the lines before it and can report false errors: with task A blocked by task C, and task B (between them) grown past the ceiling, doctor reports `✗ Orphaned dependencies: <A> depends on non-existent task <C>` — C exists, on line 3. The real cause is never mentioned. (Measured on current main.)
- Only recovery is hand-editing `.tick/tasks.jsonl` — to recover tasks that are still well-formed JSON on disk.

### Reproduction Steps

1. In a tick project, run `tick create "big" --description <70000-char string>`.
2. Observe exit 0 and a new task ID; the JSONL line for the task is ~70 KiB.
3. Run `tick list` (or `stats`, or `rebuild`) — fails with `bufio.Scanner: token too long`.
4. Run `tick doctor` — reports `JSONL syntax: OK`, `No issues found`, exit 0.

**Reproducibility:** Always.

Alternative routes to the same over-ceiling line (measured on current main unless noted):
- Notes: capped at 2000 characters each, count unbounded. **32** full-length plain notes bricked the store (line 65,589 bytes); each `note add --quiet` exited 0. With escape-heavy text (`&` × 2000), **6** notes did it (line 72,423 bytes).
- Escape-heavy description: a description of 11,000 `<` characters — shorter than real descriptions already in this repo's store (up to 12,473 chars) — encodes to a 66,163-byte line and bricks the store.
- `tick migrate` (beads import): a write route with no size check and no title cap (see H2).
- Titles: the seed and discovery log say titles are uncapped — **incorrect**. `internal/task/task.go:34` sets `maxTitleLen = 500`, enforced by rune count at `task.go:185`. A title alone cannot reach the ceiling, though it contributes to a record's size.
- Transitions accumulate on the record over its lifetime.
- Lines produced outside tick's capped write path: an older tick binary, hand edits, other tools.

### Environment

- **Affected environments:** Any — local CLI, every platform the static binary ships for.
- **Browser/platform:** n/a
- **User conditions:** A project whose `tasks.jsonl` holds (or is about to hold) any single line over ~64 KiB.
- **Real-world headroom (this repo's own dogfood store, `.tick/tasks.jsonl`, measured 2026-09-27):** 263 records, median line 3,146 bytes, longest line 13,317 bytes (~20% of the 65,536-byte ceiling). The five largest records are all description-driven (10.8k–12.5k chars of description, titles ~50 chars, no notes, 2 transitions). Real descriptions in this store already reach ~12 KB.

### Impact

- **Severity:** High — a successful save leaves the whole project unusable through tick itself, and the diagnostic built to catch store damage reports it healthy.
- **Scope:** Any project where one task accumulates enough text (description, title, notes, transitions) to cross the ceiling. Not known to have bricked a real store — the maintainer's own projects have not hit it — but tick is open source, so stores in the wild are unknown; an already-over-ceiling store cannot be ruled out.
- **Who writes the text:** For the maintainer, tasks are written by agents only, never by hand. The party that meets any write-time refusal is therefore an agent, and the refusal has to be something an agent can read and act on.
- **Who reads `doctor`:** The maintainer does not run `tick doctor`; agents do, at times, as a health check. A false "healthy" tells an agent a store is fine when no other command can open it — the agent then has no signal pointing at the JSONL line as the cause.
- **Business impact:** Trust in the store — data is intact on disk but unreachable without manual JSONL surgery.

### References

- Seed: `seeds/2026-09-21-oversized-text-field-bricks-the-store.md` (inbox bug).
- Found by the `free-text-round-trip` review's change-set verification probing that specification's §10.3; source finding id `9-field-selection-10-free-text-that-begins-with-a-dash-1`.
- Pre-existing: reproduces identically with a binary built from `5a71cbc8` (before free-text-round-trip's first commit); no commit in that feature's range touches `internal/storage` or `internal/doctor`.
- Ground named by the report:
  - Scanner construction `internal/storage/jsonl.go:95`, error check `:112`.
  - Doctor's reader `internal/doctor/jsonl_reader.go:37-66`, returns at `:65` without consulting the scanner's error.
  - Read paths in `internal/storage/store.go` — `ReadTasks`, `Rebuild`, `readAndEnsureFresh` (used by `Mutate` and `Query`).
  - Existing note cap `maxNoteTextLen` in `internal/task/notes.go`.
  - Title/description validation in `internal/task/task.go` — trims, does not bound.
- Constraint noted by the report: if a cap were surfaced as a flag rather than a fixed constant, it lands in `commandFlags`, `tick help` and the README together (`TestCommandFlagsMatchHelp`, `TestREADMEDocumentsFieldSelection`).

---

## Analysis

### Hypotheses

**Checkpoint depth:** straight-through

- **H1: The store's reader gives up on any line over 64 KiB, and every command reads through it — rebuild included** [confirmed]
  Basis: `ParseJSONL` builds a `bufio.Scanner` with no `Buffer` call (`internal/storage/jsonl.go:95`); `ReadTasks`, `Rebuild` and `readAndEnsureFresh` all call it.
  Evidence: exact ceiling measured with a default `bufio.Scanner` — a line of 65,535 bytes (excluding its newline) reads; 65,536 bytes fails with `bufio.Scanner: token too long`. Every store read funnels through `ParseJSONL`: `Store.ReadTasks` (`store.go:151`, called from `cli/app.go:290` and `cli/dep_tree.go:25`), `Store.Rebuild` (`store.go:222`), and `readAndEnsureFresh` (`store.go:359`) under both `Mutate` (`store.go:182`) and `Query` (`store.go:279`). `Query` parses the JSONL on every call before it touches SQLite, so a fresh cache is no way around it. Reproduced on a binary built from current main: `list`, `stats`, `rebuild` and `show` of an unrelated task all fail identically. Because `Mutate` reads before it writes, no mutating command — including `remove` or `update` of the oversized task itself — can run either: there is no in-tool route to shrink or delete the record (measured: `update <big-id> --description short`, `remove <big-id> --force` and `cancel <big-id>` all fail with the same error). The error names neither the line number nor the task: `ParseJSONL` knows `lineNum` when the scanner stops, but the `scanner.Err()` branch (`jsonl.go:112-114`) wraps only the scanner's error, so the user is not told which record to hand-edit.
- **H2: Nothing on the write path measures the record, so the oversized line is committed and the cache is built to match it** [confirmed]
  Basis: `Mutate` marshals, writes, then rebuilds the cache from the in-memory tasks and the written bytes — never parses them back (`internal/storage/store.go:195-209`); description is unbounded by the v1 spec.
  Evidence: every write goes through `Store.Mutate` — `cli/create.go:187`, `cli/update.go:275`, `cli/transition.go:36`, `cli/note.go:68,124`, `cli/dep.go:78,156`, `cli/remove.go:184`, `migrate/store_creator.go:36`. `Mutate` (`store.go:175-216`) reads, applies the closure, `MarshalJSONL` (`jsonl.go:16-31`, one `json.Marshal` per task), `WriteJSONLRaw` atomically, then `cache.Rebuild(mutated, newRawJSONL)` from the in-memory slice — the written bytes are hashed but never re-scanned, so nothing on the write side meets the reader's ceiling. Field bounds applied before the closure: title ≤ 500 runes (`task.go:185`), note text ≤ 2000 runes each (`notes.go:57`), tags ≤ 30 bytes × 10, refs ≤ 200 bytes × 10; **description, note count, transition count and `blocked_by` count are unbounded**. The migrate route is looser still: `MigratedTask.Validate` (`migrate/migrate.go:43-54`) checks only that the title is non-empty — no 500-rune title cap — and the description passes through untouched. The post-write exit 1 is `outputMutationResult` (`cli/helpers.go:17-32`): outside `--quiet` it calls `queryShowData` → `Store.Query` → `readAndEnsureFresh`, which parses the just-written file and fails.
- **H3: Doctor reads only up to the first overlong line, drops it and everything after it, and checks that prefix as if it were the whole store** [confirmed]
  Basis: `ScanJSONLines` never consults `scanner.Err()` (`internal/doctor/jsonl_reader.go:37-66`); a `Scanner` stops at the first `ErrTooLong`, so later lines are lost too, not just the long one.
  Evidence: measured with a default `bufio.Scanner` over `a\n<65,536 x>\nb\n` — one line yielded, then `token too long`; `b` is never seen. `cli/doctor.go:30-33` scans once and hands the prefix to every check via `JSONLinesKey`; all line and relationship checks (`jsonl_syntax.go:20`, `id_format.go:23`, `duplicate_id.go:26`, `duplicate_seq.go:27`, and via `getTaskRelationships` the orphan, self-dep, cycle, child-blocked-by-parent and parent-done checks) read only that prefix. Reproduced: oversized task mid-file → false `Orphaned dependencies` error naming a task that exists after the cut. `ScanJSONLines` has never checked the scanner's error — nor did the per-check scanners it replaced in `d555c22f`. The only failure doctor's checks can express from the reader is `fileNotFoundResult` (`doctor/helpers.go:14-22`, "tasks.jsonl not found"), so a reader error surfaced naively would be reported as a missing file by every check. The cache check (`cache_staleness.go:22-80`) hashes the raw file bytes itself and is correct: it passes because the cache was built from those exact bytes.
- **H4: Per-field character caps cannot bound the line: a record crosses the ceiling with every field under its cap** [confirmed]
  Basis: caps count runes (`internal/task/task.go:185`, `internal/task/notes.go:57`) while the reader counts encoded bytes — UTF-8 width and JSON escaping inflate — and note count, transitions and dependencies are unbounded.
  Evidence: Go's `json.Marshal` escapes `<`, `>`, `&`, U+2028/U+2029 and control characters (other than `\n`, `\r`, `\t`) to 6-byte `\uXXXX` sequences; `"`, `\`, `\n`, `\r`, `\t` to 2 bytes; a non-ASCII rune is 2–4 bytes raw. Worst case is **6 encoded bytes per rune**. Measured on current main: 11,000 `<` in a description → 66,163-byte line (bricked); 6 notes of 2000 `&` → 72,423 bytes (bricked); 32 plain 2000-char notes → 65,589 bytes (bricked) with no description at all. Real text inflates little: this repo's descriptions over 500 chars encode at most ~1.08× their character count, with ≤ 2 `<>&` characters each. So a character cap sized for real text (the largest real description is 12,473 chars) still admits a record that bricks the store, and caps on individual fields leave the unbounded counts (notes, transitions, `blocked_by`) open regardless.
- **H5: Doctor's idea of a readable line is looser than the store's — a line that is valid JSON but not a valid task passes doctor and still bricks the store** [confirmed]
  Basis: surfaced mid-trace — doctor parses each line into `map[string]any` (`jsonl_reader.go:53-56`) and the syntax check asks only `json.Valid` (`jsonl_syntax.go:34`); the store unmarshals into the typed task (`jsonl.go:105`).
  Evidence: hand-edited line 2's `"priority":2` → `"priority":"high"`. `tick list` fails `failed to parse line 2: json: cannot unmarshal string into Go struct field taskJSON.priority of type int`; doctor reports `✓ JSONL syntax: OK`, flags only `✗ Cache: stale`, and suggests `tick rebuild` — which fails the same way. The v1 doctor-validation specification scopes this out on purpose ("Schema validation … happens at write time, not in doctor"), so this is a specified boundary rather than an oversight; it is the same doctor/store disagreement as H3 (doctor passes a store the store cannot open), reached by a different route.

Trace lines, in order:
1. Reproduce on current main with a built binary: >64 KiB description; notes-only route; a description under any plausible character cap that encodes past the ceiling.
2. Store read paths: every caller of `ParseJSONL` / `ReadJSONL` — confirm no command reads the store another way.
3. Write path: `Mutate` → `MarshalJSONL` → write → cache rebuild; every `Mutate` caller (create, update, note add, dep, transitions, migrate import) and the field validation each applies.
4. Doctor: `ScanJSONLines` and its wiring in `internal/cli/doctor.go`; what each check sees on a truncated prefix; the cache-staleness check.
5. Encoding width: `json.Marshal` escaping on task fields; worst-case line size under today's caps.
6. Blast radius: the beads importer's own scanner; existing tests at the reader boundaries.

### Code Trace

**Entry point:**
Any mutating command (`create`, `update`, `note add`, `start`/`done`/`cancel`/`reopen`, `dep add`, `migrate`) that leaves one task's encoded JSON at ≥ 65,536 bytes.

**Execution path — the write that succeeds:**
1. `internal/task/task.go:177-190` / `internal/task/notes.go:52-61` — field validation: title ≤ 500 runes, each note ≤ 2000 runes; description checked only for non-empty (`task.go:203-208`). No record-level bound anywhere. (`migrate/migrate.go:43-54`: title non-empty only.)
2. `internal/storage/store.go:175-216` `Store.Mutate` — `readAndEnsureFresh` (succeeds: the store is still readable), closure appends/edits the task.
3. `internal/storage/jsonl.go:16-31` `MarshalJSONL` — `json.Marshal` per task, HTML-safe escaping (`<`/`>`/`&` → 6 bytes). No size check.
4. `internal/storage/jsonl.go:44-85` `WriteJSONLRaw` → `writeAtomic` — temp file, fsync, rename. The oversized line is now the source of truth.
5. `internal/storage/store.go:209` `cache.Rebuild(mutated, newRawJSONL)` — cache populated from the in-memory slice; hash of the written bytes stored. Cache is fresh.
6. `internal/cli/helpers.go:17-32` `outputMutationResult` — with `--quiet`, prints the ID, exit 0. Otherwise `queryShowData` → `Store.Query` → step 7 below fails → exit 1 after a committed write.

**Execution path — every later command:**
7. `internal/storage/store.go:359-375` `readAndEnsureFresh` (via `Mutate`/`Query`), `store.go:151-170` `ReadTasks`, or `store.go:222-266` `Rebuild` (which first deletes `cache.db`, `store.go:235-238`) → `os.ReadFile` → `ParseJSONL`.
8. `internal/storage/jsonl.go:95` — `bufio.NewScanner` with Go's default `MaxScanTokenSize` (64 × 1024); `Scan` returns false at the first line of ≥ 65,536 bytes.
9. `internal/storage/jsonl.go:112-114` — `scanner.Err()` = `bufio.ErrTooLong`, wrapped as `error reading JSONL data: …` without the line number; `store.go` wraps as `failed to parse tasks.jsonl: …`. Command exits 1.

**Execution path — doctor:**
10. `internal/cli/doctor.go:30-33` → `internal/doctor/jsonl_reader.go:27-66` `ScanJSONLines` — same default scanner; `Scan` stops at the oversized line; `scanner.Err()` never consulted; returns the prefix with a nil error.
11. Every line/relationship check runs over the prefix via `JSONLinesKey`; the syntax check (`jsonl_syntax.go:19-56`) finds nothing malformed in it → `✓ JSONL syntax: OK`. Relationship checks can report false orphans for references into the unseen suffix.
12. `internal/doctor/cache_staleness.go:22-80` — hashes the whole raw file; matches the hash `Mutate` stored → `✓ Cache: OK` (or, after a failed `rebuild`, `✗ cache.db not found` → "Run `tick rebuild`").

**Key files involved:**
- `internal/storage/jsonl.go` — the reader with the unchosen 64 KiB ceiling (`:95`), the error that omits the line (`:112-114`), the marshal with no size check (`:16-31`).
- `internal/storage/store.go` — `Mutate` (write never meets the reader), `Rebuild` (deletes cache before parsing), `readAndEnsureFresh`/`ReadTasks` (every read funnels into `ParseJSONL`).
- `internal/doctor/jsonl_reader.go` — doctor's reader: same ceiling, scanner error dropped.
- `internal/doctor/helpers.go` — `fileNotFoundResult`, the only reader-failure result checks can emit.
- `internal/task/task.go`, `internal/task/notes.go`, `internal/migrate/migrate.go` — field caps in runes, no record-level bound, migrate looser than the CLI.
- `internal/cli/helpers.go` — post-write read-back that turns a committed write into exit 1.

### Root Cause

### Contributing Factors

### Why It Wasn't Caught

### Blast Radius

---

## Fix Direction

---

## Notes

- Discovery settled that the two halves are independent: a write-time cap stops tick producing an unreadable line going forward, but does not rescue a store that already holds one, does not cover lines produced outside the capped write path, and the doctor's silent pass hides all of those alike.
- Starting shape carried from the inbox report (subject to this investigation): no support for arbitrarily large text; a generous but finite limit enforced on the write path with a clear error, consistent with the existing note cap. Discovery added that the bound has to be reasoned about per record (the line the reader consumes), not only per field — or the reader's ceiling itself reconsidered.
- Open question carried from discovery: recovery for stores already over the ceiling.

### Prior decisions on this ground (knowledge base, 2026-09-27)

- **v1/tick-core specification** (Task Schema → Title and Description Limits): title required, max 500 characters, single line; description optional, **"No maximum length"**, newlines and markdown allowed. An unbounded description is a specified decision, not an omission — a description cap would contradict that specification.
- **increase-note-char-limit specification**: raised `maxNoteTextLen` 500 → 2000; Exclusions state the description is "intentionally unbounded (only empty/whitespace rejected via `ValidateDescriptionUpdate`)" and that neither JSONL nor the `task_notes` TEXT column enforces a length cap.
- **free-text-round-trip specification §10.3**: declines a stdin/file input path for descriptions; measured `ARG_MAX` = 1,048,576 and a 200 KB argument passing through a process call; states "an arbitrarily large description remains possible in principle". The argument channel carries far more than the store's reader can read back — the ceiling that bites is the reader's, not the shell's.
- **v1/doctor-validation specification**: doctor is "report, don't fix", a "safety net" that "catches what slipped through write-time validation or got corrupted"; error check #2 is "JSONL syntax errors — malformed JSON lines that can't be parsed"; exit 1 when any error is found. The spec calls doctor human-focused and says agents don't parse its output — in practice (see Impact) agents do run it.
