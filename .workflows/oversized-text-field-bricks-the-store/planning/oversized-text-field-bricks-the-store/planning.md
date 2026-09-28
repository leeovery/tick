# Plan: Oversized Text Field Bricks The Store

## Phases

### Phase 1: Store reads lines of any length
status: draft

**Goal**: Remove the line-length ceiling from every reader of `tasks.jsonl` and from the beads importer. The store moves onto one shared line reader that keeps the store's current line definition exactly (§2.1–§2.2). The beads importer drops its ceiling and keeps its own line rules (§2.3). No route can then make the store unreadable by line length, and stores already over the old ceiling open again with no repair step. Store read errors name the line and, where the line carries one, the task (§3). `tick rebuild` parses `tasks.jsonl` before it touches `cache.db`, so a failed rebuild leaves the existing cache in place (§4).

**Why this order**: This is the bug itself: the store writes lines its own reader cannot read. It is the smallest fix for that root cause. It also builds the shared line reader that Phase 2 moves doctor onto, and it establishes the store's line-naming load errors that doctor's verdict must match.

**Acceptance**:
- [ ] Given a store whose `tasks.jsonl` already holds a task line of more than 65,536 bytes, `tick list`, `tick show <id>` and `tick rebuild` succeed. `tick update` and `tick remove` of the oversized task also succeed, with no migration or repair step.
- [ ] Given a store, a command that takes a task's stored line to 65,536 bytes or more (for example `tick update --description` or `tick note add`) exits 0 with its normal output. Every following command, including reads, writes and `tick rebuild`, opens the store.
- [ ] Given a parent task with a very large record, `tick start <child>` succeeds and the next `tick list` succeeds. `tick dep add` and status changes on a very large record likewise leave the next `tick list` succeeding.
- [ ] Given task lines of exactly 65,535 bytes, exactly 65,536 bytes and about 1 MiB (newline excluded), each is read. The ~1 MiB task is shown in full by `tick show` in toon, pretty and JSON output.
- [ ] Given a store with CRLF-terminated lines, empty lines and a final task line with no trailing newline, with or without one trailing `\r`, commands read every task exactly as they do today.
- [ ] Given an otherwise valid store whose file ends `…}\n\r`, commands succeed: the final `\r` is an empty, skipped line, not a whitespace-only one. Given a whitespace-only line, commands still fail, naming its line number.
- [ ] Given a line 2 holding task `tick-a1b2c3` with a wrong-typed field (`"priority":"high"`), every command fails with an error naming line 2, `tick-a1b2c3` and the decoder's reason. A malformed-JSON line, or a line with no string `id`, is named by line number alone.
- [ ] Given a store that fails to parse and an existing `cache.db`, `tick rebuild` exits 1 with that parse error and leaves `cache.db` byte-for-byte unchanged. `tick rebuild --verbose` logs `reading JSONL` before `deleting cache.db`. `tick rebuild` over a corrupt `cache.db` still succeeds.
- [ ] Given a `.beads/issues.jsonl` holding an issue line longer than 64 KiB, `tick migrate --from beads` reads that line and completes the import. Today the import aborts before anything is written.

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| oversized-text-field-bricks-the-store-1-1 | Shared line reader with no length ceiling | lines of exactly 65,535 bytes, exactly 65,536 bytes and ~1 MiB, newline excluded (§2.1, §8.1), CRLF-terminated lines read the same as LF (§2.2), final line with no trailing newline with and without one trailing `\r` (§2.2, Corrigendum), file ending `…}\n\r` has an empty skipped final line (§2.2, Corrigendum), empty lines skipped but counted in line numbering (§2.2), whitespace-only line not skipped and fails naming its line number (§2.2) |
| oversized-text-field-bricks-the-store-1-2 | Stores already over the old ceiling open and operate | line written before the fix opens with no migration or repair step (§2.1), ~1 MiB task renders in full in toon, pretty and JSON (§8.1) |
| oversized-text-field-bricks-the-store-1-3 | Writes that take their own task past the old ceiling report success | line crosses the ceiling through accumulated notes, each within the 2,000-character note cap (§1.2, §2.4) |
| oversized-text-field-bricks-the-store-1-4 | Status changes, dep add and cascades on a large record leave the store readable | cascade grows a parent the command never named (§1.1, §8.1) |
| oversized-text-field-bricks-the-store-1-5 | Beads importer reads issue lines of any length | keeps its own line rules (each line trimmed), not the shared reader (§2.3) |
| oversized-text-field-bricks-the-store-1-6 | Store read errors name the line and the task | malformed-JSON line named by line number alone (§3), line with no string `id` (missing, non-string, or not a JSON object) named by line number alone (§3) |
| oversized-text-field-bricks-the-store-1-7 | `tick rebuild` keeps the cache until it can build a new one | corrupt `cache.db` still recovered (§4, §8.3), `--verbose` logs `reading JSONL` before `deleting cache.db` (§4, §8.3) |

### Phase 2: Doctor passes only a store every command can open
status: draft

**Goal**: Doctor reads `tasks.jsonl` through the shared line reader and judges every line by loading it as a task, exactly as commands do (§5.1). When the read cannot complete, the run fails instead of passing the lines it did read (§5.3). Doctor stops directing the user to `tick rebuild` while the store cannot load (§5.5). The result is that a doctor run with no errors means every tick command can open the store.

**Why this order**: Doctor's false pass is the second, independent root cause (§1.2). The fix uses Phase 1's shared reader, and its verdicts are checked against Phase 1's store behaviour: every doctor failure fixture must also fail `tick list` on the same line.

**Acceptance**:
- [ ] Given doctor's own line scanner, left in place by Phase 1, doctor reads `tasks.jsonl` only through Phase 1's shared line reader, so no reader of `tasks.jsonl` keeps its own scanner or line-length ceiling (deferred from Phase 1, task 1-1).
- [ ] Given a two-task store with one added line, the line being any of: whitespace-only, `null`, `[]`, `{}`, malformed JSON, or a wrong-typed field such as `"priority":"high"`:
  - `tick doctor` exits 1 with a `JSONL syntax` failure that names that line and gives the loader's reason.
  - `tick list` on the same store fails, naming the same line.
  - No suggestion in the doctor report names `tick rebuild`.
- [ ] Given a valid store, `tick doctor` reports `No issues found`.
- [ ] Given an otherwise valid store with a line over 64 KiB mid-file, `tick doctor` reports no issues. In particular it reports no orphaned-reference error for tasks that sit after the long line.
- [ ] Given a store carrying empty, whitespace-only and CRLF-terminated lines, doctor and the store read the same lines and report the same line numbers.
- [ ] Given an otherwise valid store whose file ends `…}\n\r`, `tick list` succeeds and a following `tick doctor` reports `No issues found`. Both treat the final `\r` as an empty, skipped line.
- [ ] Given a read of `tasks.jsonl` that stops partway (driven through a test seam over the reader and run through `RunDoctor`):
  - The report fails, naming the line it could not read and saying the file could not be read in full.
  - No check that consumes the file's lines passes.
  - Nothing reports `tasks.jsonl not found`.
  - No suggestion names `tick rebuild`.
- [ ] Given a stale cache plus a line that fails to load, the Cache check still fails, but its suggestion points at fixing the lines the JSONL check names. Once the line is fixed, the next `tick doctor` gives the usual `tick rebuild` advice for the still-stale cache.
- [ ] Given a missing `tasks.jsonl`, doctor still reports `tasks.jsonl not found`. The check is still titled `JSONL syntax` in doctor's output, and the README doctor "Checks for:" list, its tested sample and `tick help doctor` are unchanged.

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| oversized-text-field-bricks-the-store-2-1 | Doctor reads tasks.jsonl through the shared line reader | line over 64 KiB mid-file reports no issues and no orphaned-reference error for tasks after it (§1.2, §5.4, §8.4), whitespace-only line is read as a line rather than skipped and gets the same line number the store reports (§2.2, §8.1), empty lines skipped but counted in line numbering (§2.2, §8.1), CRLF-terminated lines read the same as LF (§2.2, §8.1), file ending `…}\n\r` has an empty skipped final line so `tick list` succeeds and doctor reports `No issues found` (§2.2, Corrigendum 2026-09-28 on §2.2) |
| oversized-text-field-bricks-the-store-2-2 | JSONL syntax check judges each line by loading it as a task | whitespace-only line (§5.1, §8.4), `null`, `[]` and `{}` (§5.1, §8.4), wrong-typed field such as `"priority":"high"` (§5.1, §8.4), malformed JSON (§5.1), unparseable timestamp (§5.1), several failing lines reported one failure each (§5.1), each fixture also fails `tick list` naming the same line (§8.4), enum membership (status, type) and priority range are not doctor's (§5.1), value repeated within one task's own `tags`, `refs` or `blocked_by` list passes the JSONL check and gets no check of its own (§5.1, Corrigendum 2026-09-28 on §5.1), valid store still reports `No issues found` (§5.1, §8.4), check keeps the `JSONL syntax` name in output, README and `tick help doctor`, and keeps a hand-fix suggestion (§5.2), relationship and hierarchy checks keep skipping lines that are not a JSON object or have no string `id` (§5.4) |
| oversized-text-field-bricks-the-store-2-3 | Doctor fails a read of tasks.jsonl it cannot complete | driven through a test seam over the reader and exercised through `RunDoctor`, not the reader alone (§5.3, §8.4), no line-consuming check passes over the lines read before the stop (§5.3, §8.4), nothing reports `tasks.jsonl not found` (§5.3, §8.4), a file that cannot be opened still reports `tasks.jsonl not found` (§5.3) |
| oversized-text-field-bricks-the-store-2-4 | No `tick rebuild` advice while the store cannot load | stale cache with a line that fails to load (§5.5), missing `cache.db` with a line that fails to load (§5.5), incomplete read (§5.5), no suggestion names `tick rebuild` for any §5.1 fixture (§8.4), once the line is fixed the next run gives the usual `tick rebuild` advice for a still-stale cache (§5.5), the cache check's verdict is unchanged, still from hashing the raw file bytes (§5.5), value repeated within one task's own `tags`, `refs` or `blocked_by` list: the Cache check reports the stale cache with its usual `tick rebuild` advice, and that rebuild fails naming the task and the repeated value (§5.1, Corrigendum 2026-09-28 on §5.1) |

### Phase 3: Description cap
status: draft

**Goal**: Cap the task description at 50,000 characters, counted after trimming (§6.1), on every route that sets one: create, update and migrate. The refusal tells an agent the field, the limit, the submitted length and that nothing was saved (§6.3). Migrate also adopts the CLI's title rules (§6.4).

**Why this order**: The cap is hygiene, not the store's safety device (§6.1), and nothing in it depends on Phases 1 and 2. It lands after the defect fixes so that the store's safety never depends on it.

**Acceptance**:
- [ ] Given `tick create --description` of exactly 50,000 characters, the task is created, including when the characters are multibyte. Given 50,001 characters, the command exits 1 with an error naming the description field, the 50,000 limit, the submitted length and that nothing was saved, and `tasks.jsonl` and the cache are unchanged.
- [ ] Given the same descriptions on `tick update --description`, the same boundary and refusal hold, and a refused update leaves `tasks.jsonl` unchanged.
- [ ] Given a 50,000-character description padded with leading and trailing whitespace, `create` and `update` accept it. The length reported in any refusal is the trimmed character count.
- [ ] Given a task whose stored description is already over 50,000 characters, a status change, an `update` that sets a new title and a `note add` each succeed, and the task reads normally.
- [ ] Given a migrate source where one issue's description is exactly 50,000 characters and another's is 50,001:
  - `tick migrate` imports the first.
  - It skips the second with a reason carrying the field, the limit, the submitted length and that nothing was saved.
  - The remaining issues import.
  - `--dry-run` reports the same skip.
- [ ] Given a migrate source with an issue whose title is over 500 characters or spans more than one line, that issue is skipped with its reason and the remaining issues import.
- [ ] `tick help`, the command flag registry and the README are unchanged, since the cap is a constant and not a flag. Title and note refusal messages keep their current wording.

#### Tasks

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| oversized-text-field-bricks-the-store-3-1 | Description cap on `tick create` | exactly 50,000 characters accepted (§6.1, §8.5), 50,001 characters refused (§6.1, §8.5), multibyte description of exactly 50,000 characters accepted (§6.1, §8.5), leading and trailing whitespace not counted toward the cap (§6.1, §8.5), reported length is the trimmed character count (§6.3), refused create leaves `tasks.jsonl` and the cache untouched (§6.2, §8.5), cap is a constant and not a flag, so the flag registry, `tick help` and README are unchanged (§6.1), title and note refusals keep their current messages (§7) |
| oversized-text-field-bricks-the-store-3-2 | Description cap on `tick update --description` | exactly 50,000 characters accepted and 50,001 refused (§6.1, §8.5), multibyte description of exactly 50,000 characters accepted (§6.1, §8.5), leading and trailing whitespace not counted toward the cap (§6.1, §8.5), refused update leaves `tasks.jsonl` and the cache unchanged (§6.2, §8.5), stored over-cap description left as it is and reads normally (§6.2), stored over-cap description does not block a status change, an `update` that sets a new title, or a `note add` (§6.2, §8.5) |
| oversized-text-field-bricks-the-store-3-3 | Migrate skips an issue whose description exceeds the cap | description of exactly 50,000 characters imports (§6.4, §8.5), description of 50,001 characters skipped with a reason carrying the field, the limit, the submitted length and that nothing was saved (§6.3, §6.4, §8.5), the other issues import and one oversized issue never aborts the import (§6.4, §8.5), description never truncated (§6.4), reported length is the trimmed character count (§6.1, §6.3), `--dry-run` reports the same skip as a real run (§6.4, §8.5) |
| oversized-text-field-bricks-the-store-3-4 | Migrate adopts the CLI's title rules | title over 500 characters skipped with its reason (§6.4, §8.5), title spanning more than one line skipped with its reason (§6.4, §8.5), the other issues import (§6.4), `--dry-run` reports the same skip (§6.4), skip reasons use the CLI's current title refusal messages (§7) |
