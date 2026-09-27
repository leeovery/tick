# Investigation: Oversized Text Field Bricks The Store

## Symptoms

### Problem Description

**Expected behavior:**
Either a task record is refused on the write path with a clear error before it can make the store unreadable, or — once written — every store-reading command can still read it. `tick doctor` reports a store it could not fully read as unhealthy.

**Actual behavior:**
A task record whose JSONL line passes ~64 KiB is accepted and written (exit 0, new ID printed). From that moment every command that reads the store fails, `tick rebuild` included, so the tool offers no route back to a readable project. `tick doctor` meanwhile reports the project healthy.

### Manifestation

- `tick create "big" --description <70000 chars>` exits 0 and prints the new ID. The record lands in `.tick/tasks.jsonl` as a single 70,146-byte line.
- The SQLite cache is rebuilt from those same bytes, so the freshness hash matches and nothing looks wrong at write time.
- The next `tick list`, `tick stats` and `tick rebuild` all fail with:
  `Error: failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long`
- `tick doctor` prints `✓ JSONL syntax: OK` and `No issues found`, exit 0, on the same project. Cache-staleness passes too, since the cache genuinely was rebuilt from the bytes on disk.
- Only recovery is hand-editing `.tick/tasks.jsonl` — to recover tasks that are still well-formed JSON on disk.

### Reproduction Steps

1. In a tick project, run `tick create "big" --description <70000-char string>`.
2. Observe exit 0 and a new task ID; the JSONL line for the task is ~70 KiB.
3. Run `tick list` (or `stats`, or `rebuild`) — fails with `bufio.Scanner: token too long`.
4. Run `tick doctor` — reports `JSONL syntax: OK`, `No issues found`, exit 0.

**Reproducibility:** Always.

Alternative routes to the same over-ceiling line (from discovery, to be confirmed in analysis):
- Notes are capped at 2000 characters each but their count is unbounded — roughly 33 full-length notes push a task's line past the reader's ceiling with no description at all.
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

**Checkpoint depth:** {straight-through | check-ins}

### Code Trace

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
