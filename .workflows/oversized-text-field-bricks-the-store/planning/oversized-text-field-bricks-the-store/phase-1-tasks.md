# Phase 1: Store reads lines of any length — 7 tasks

## oversized-text-field-bricks-the-store-1-1

### Task 1-1: Shared line reader with no length ceiling

**Problem**: One task line of 65,536 bytes or more makes the store unreadable. From then on every tick command that opens the store fails with `failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long`: reads, writes and `tick rebuild` alike. The message names neither the line nor the task, and the only recovery is hand-editing `.tick/tasks.jsonl`. The cause is the store's parser, a `bufio.Scanner` at Go's default maximum token size (`MaxScanTokenSize = 64 * 1024`). Nobody chose that limit and nothing on the write path knows about it. The title and note caps count characters, the description is unbounded, and JSON escaping inflates text by up to 6 bytes per character. Note, transition and `blocked_by` counts are unbounded too, so no per-field cap can keep a record under the limit. It bought no resource protection either, since every store read already loads the whole file into memory before parsing.

**Solution**: Build one shared line reader for `tasks.jsonl`. It accepts a line of any length and keeps the store's current definition of a line exactly. The store's parsing moves onto it. Phase 2 moves doctor onto the same reader.

**Outcome**: The store reads every line the file holds, whatever its length. On every store the old reader could read, commands behave exactly as they do today: the same lines are read, the same lines are skipped and the same line numbers are reported.

**Acceptance Criteria**:
- [ ] A store whose `tasks.jsonl` holds a task line of exactly 65,536 bytes (newline excluded): `tick list` succeeds and lists that task. Today it fails with `bufio.Scanner: token too long` (§1.1, §2.1)
- [ ] Task lines of exactly 65,535 bytes, exactly 65,536 bytes and about 1 MiB, newline excluded: the shared reader returns each line whole, and a store holding all three lists all three tasks (§2.1, §8.1)
- [ ] A store whose lines end in CRLF: every task reads exactly as it does from the same store with LF endings (§2.2)
- [ ] A store whose final task line has no trailing newline: that task is read. When that line ends in one `\r`, the `\r` is stripped and the task still reads (§2.2, Corrigendum 2026-09-28)
- [ ] An otherwise valid store whose file ends `…}\n\r`: `tick list` succeeds, because the final `\r` is an empty, skipped line, not a whitespace-only one (§2.2, Corrigendum 2026-09-28)
- [ ] A store with empty lines between its tasks: every task is read and the empty lines are skipped (§2.2)
- [ ] A store whose line 2 is empty and whose line 3 holds only whitespace: `tick list` fails, naming line 3. The skipped empty line still counts toward the numbering, and the whitespace-only line is not skipped (§2.2)
- [ ] The reader imposes no per-line size limit, neither Go's default scanner limit nor any larger one set in its place (§2.1)

**Do**:
- `internal/storage/jsonl.go`: `ParseJSONL` parses through the shared reader instead of its `bufio.Scanner` (line 95). All three store reads load the whole file and parse it through `ParseJSONL`, so they all move with it. Those reads are `ReadTasks`, `Rebuild` and `readAndEnsureFresh` (`internal/storage/store.go:159`, `:243`, `:360`).
- The line definition is the store's current one:
  - a line ends at `\n`, and one `\r` immediately before it is dropped;
  - a final line with no trailing newline is still read, and one `\r` at its end is dropped, as `bufio.ScanLines` does;
  - a line that is empty once its terminator is removed is skipped (the `line == ""` test at `internal/storage/jsonl.go:100`);
  - lines are numbered from 1, and skipped lines count toward the numbering.
- The ceiling is removed, not raised. No per-line size limit of any size replaces Go's default.
- Revisit `TestParseJSONL` and `TestReadJSONL` (`internal/storage/jsonl_test.go`) against the new reader (§8.6).

**Context**:
> §2.2: "The store and doctor read `tasks.jsonl` through one shared line reader, so the two cannot disagree about what the file holds and no future reader of the store can reintroduce a ceiling." The reader carries the store's current definition of a line, "preserved exactly, so commands behave as they do today on every store the old reader could read". A whitespace-only line "is an ordinary line that must load as a task, and fails to". This was measured on current main: a trailing `   ` line makes `tick list` fail with `failed to parse line 3: unexpected end of JSON input`.
>
> The reader has to be reachable from `internal/doctor`, which today imports no other internal package. Doctor keeps its own scanner (`internal/doctor/jsonl_reader.go:37`), with its looser whitespace-only skip (`:44`), for the whole of this phase. Phase 2 moves doctor onto this reader, and its acceptance carries that deferral. In Phase 2, a read that stops partway must be reported with the line the reader could not read. That case is driven through a test seam over the reader (§5.3, §8.4).
>
> Sequencing within the phase: this task keeps today's `failed to parse line N:` error detail, and Task 1-6 adds the task ID to it (§3). The beads importer's scanner (`internal/migrate/beads/beads.go:84`) is Task 1-5, and that importer does not use this reader (§2.3). Task 1-2 covers the full command set on a store already over the old ceiling.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.1, §1.2, §2.1, §2.2, §8.1, §8.6, Corrigenda (2026-09-28)

## oversized-text-field-bricks-the-store-1-2

### Task 1-2: Stores already over the old ceiling open and operate

**Problem**: A store that already holds a line of 65,536 bytes or more cannot be reached through tick today. The line may have come from the CLI's own writes, a cascade, `migrate`, a hand edit or another tool. `list`, `show` and `rebuild` all fail, and so do `update` and `remove` of the oversized task itself, so hand-editing `.tick/tasks.jsonl` is the only recovery. The fix has to reopen such a store once tick is upgraded, with no migration or repair step, and the oversized task must then behave like any other. Task 1-1 makes the line readable. A line that size must also round-trip through the SQLite cache and render in every output format.

**Solution**: Run the full command set against a fixture store whose over-ceiling line was written straight into `tasks.jsonl`, the way a store written before the fix holds it. Send a task of about 1 MiB through the cache and all three output formats.

**Outcome**: Upgrading tick is the whole recovery. On a store over the old ceiling, `list`, `show` and `rebuild` succeed, and `update` and `remove` of the oversized task succeed. A task of about 1 MiB is shown in full in toon, pretty and JSON.

**Acceptance Criteria**:
- [ ] A fixture store whose `tasks.jsonl` holds a task line of more than 65,536 bytes among ordinary tasks: `tick list` lists every task, and `tick show <oversized-id>` shows the oversized task (§2.1, §8.1)
- [ ] Same fixture: `tick rebuild` succeeds and reports every task rebuilt (§2.1, §8.1)
- [ ] Same fixture, with no migration or repair step first: `tick update <oversized-id>` succeeds, and `tick remove <oversized-id>` succeeds and the task is gone from the next `tick list` (§2.1, §8.1)
- [ ] A task whose line is about 1 MiB: `tick show <id>` in toon, pretty and JSON output each carries the task's content in full, with no truncation, after the task has passed through the SQLite cache (§8.1)

**Context**:
> §2.1: "A store already holding an over-ceiling line opens once tick is upgraded, with no migration or repair step, and the oversized task can then be shown, updated and removed like any other."
>
> Phase 3 caps a description being set at 50,000 characters (§6.1). A description already stored over the cap is left as it is, reads normally, and does not block other changes to its task, such as a status change, a new title or a note (§6.2). If this fixture's bulk is an over-cap description, an update exercised here still passes after Phase 3, provided the update does not set a new over-cap description.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.1, §2.1, §6.2, §8.1

## oversized-text-field-bricks-the-store-1-3

### Task 1-3: Writes that take their own task past the old ceiling report success

**Problem**: `create`, `update` and `note add` can take a task's line to 65,536 bytes or more. Today they commit that write and then, outside `--quiet`, exit 1 when their read-back of the saved task fails. The agent is told a saved write failed, and every command after it fails too. The description is not the only way across the line. Note counts are unbounded, so a task crosses it through accumulated notes that each sit within the 2,000-character note cap. No per-field cap prevents that (§1.2, §2.4).

**Solution**: Exercise each write route that grows the task the command names against the fixed reader. A write that takes its own task across the old ceiling reports success with its normal output and leaves the store open.

**Outcome**: A `create`, `update` or `note add` whose write leaves the task's line at 65,536 bytes or more exits 0 with its normal output, so the reported result matches what was saved. Every command after it opens the store.

**Acceptance Criteria**:
- [ ] A task whose line is under 65,536 bytes: `tick update <id> --description` with text that takes the line to 65,536 bytes or more exits 0 with its normal task detail output, carrying the new description (§1.1, §7)
- [ ] `tick create` with a description that makes the new task's line 65,536 bytes or more exits 0 with its normal task detail output (§1.1, §7)
- [ ] A task gains notes one `tick note add` at a time, each note within the 2,000-character cap. The `note add` that takes its line to 65,536 bytes or more exits 0 with its normal task detail output, and the new note is included (§1.2, §2.4, §7)
- [ ] After each of these writes, every following command opens the store: a read (`tick list`, `tick show <id>`), a further write to the grown task and `tick rebuild` all succeed (§2.1)

**Context**:
> §1.1: "`create`, `update` and `note add` commit the write, then exit 1 outside `--quiet` when their read-back fails." The read-back is the task detail these commands print once the write is committed. `outputMutationResult` (`internal/cli/helpers.go:17`) queries it back through the store. With `--quiet` it prints only the ID and skips that read, so the scenarios above run without `--quiet`.
>
> §7: "The post-write signal needs no change of its own." Once the reader accepts any line (Task 1-1), a write can no longer produce a line the reader refuses. The read-back then succeeds, and the inconsistency disappears.
>
> Phase 3 refuses a description being set over 50,000 characters, counted in characters after trimming (§6.1). The description scenarios here keep passing after Phase 3 only if their text crosses 65,536 bytes while staying within 50,000 characters. The spec gives two ways text's byte length outruns its character count. JSON escaping inflates `<`, `>`, `&`, U+2028/U+2029 and control characters other than `\n`, `\r`, `\t` by up to 6 bytes per character (§1.2). Multibyte characters count once toward the cap (§6.1).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.1, §1.2, §2.1, §2.4, §6.1, §7

## oversized-text-field-bricks-the-store-1-4

### Task 1-4: Status changes, dep add and cascades on a large record leave the store readable

**Problem**: Status changes and `dep add` exit 0 with a normal success document even when their write takes a task's line to 65,536 bytes or more. The agent is told the command succeeded, and then every following command fails. Some routes also grow a task the command never named. The start cascade, `--blocks` and the Rule 6 done-parent reopen each record a change on another task, so `tick start <child>` can push a large parent over the line. The agent never touched the parent, yet the store is now unreadable.

**Solution**: Exercise the status-change, `dep add` and cascade routes against the fixed reader, starting from a large record that the command's own growth takes across the old ceiling. The next command must open the store.

**Outcome**: A status change, a new dependency or a cascade that takes a task's line to 65,536 bytes or more still exits 0 with its normal output. The next `tick list` now succeeds, including when the grown task is one the command never named.

**Acceptance Criteria**:
- [ ] An open parent whose line sits close enough under 65,536 bytes that the start cascade's transition takes it to 65,536 or more: `tick start <child>` exits 0 and moves the parent to `in_progress`, and the next `tick list` succeeds and lists both tasks (§1.1, §8.1)
- [ ] A task whose line a new dependency takes to 65,536 bytes or more: `tick dep add` on that task exits 0 with its normal output, and the next `tick list` succeeds (§1.1, §8.1)
- [ ] A task whose line its own recorded transition takes to 65,536 bytes or more: each status change (`tick start`, `tick done`, `tick cancel`, `tick reopen`) exits 0 with its normal output, and the next `tick list` succeeds (§1.1, §8.1)
- [ ] A done parent whose line the Rule 6 reopen takes to 65,536 bytes or more: `tick create --parent <parent>` exits 0 and reopens the parent, and the next `tick list` succeeds (§1.1, §2.1)
- [ ] A task whose line a `--blocks` link takes to 65,536 bytes or more: `tick create --blocks <task>` exits 0, and the next `tick list` succeeds (§1.1, §2.1)

**Context**:
> §1.1: "Status changes and `dep add` exit 0 with a normal success document. Cascades, `--blocks` and the Rule 6 done-parent reopen grow tasks the command never named, so `tick start <child>` can push a large parent over the line." §2.1 lists cascades onto tasks a command never named, status changes and `dep add` among the routes that can no longer make the store unreadable by line length.
>
> §7: the post-write signal needs no change of its own. Once the reader accepts any line (Task 1-1), the read-back succeeds, and the gap between a reported success and an unreadable store disappears.
>
> Each of these routes grows the affected task's record. A status change or cascade appends a transition record, and `dep add` and `--blocks` add an entry to `blocked_by`. Transition and `blocked_by` counts stay unbounded (§7), so this growth never meets a cap.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.1, §2.1, §7, §8.1

## oversized-text-field-bricks-the-store-1-5

### Task 1-5: Beads importer reads issue lines of any length

**Problem**: `tick migrate --from beads` reads `.beads/issues.jsonl` with its own `bufio.Scanner` at Go's default maximum token size. An `issues.jsonl` line of 65,536 bytes or more therefore aborts the whole import before anything is written, with an error that names no line. The file is external, not the store, so it keeps its own line rules and does not move onto the shared reader. Its ceiling has to go all the same.

**Solution**: Drop the beads importer's line-length ceiling the same way the store's was dropped. The importer keeps its own reader and its own line rules.

**Outcome**: `tick migrate --from beads` reads an `issues.jsonl` line of any length and completes the import. Its handling of blank and whitespace-only lines is unchanged.

**Acceptance Criteria**:
- [ ] A `.beads/issues.jsonl` holding an issue line longer than 64 KiB among ordinary issues: `tick migrate --from beads` reads that line and completes the import, and every issue is imported, the long one included. Today the import aborts before anything is written (§2.3, §8.1)
- [ ] A `.beads/issues.jsonl` with whitespace-only lines among its issues: those lines are skipped as they are today, not reported as failed entries, because the importer trims each line (§2.3)
- [ ] The importer imposes no per-line size limit, neither Go's default scanner limit nor any larger one set in its place (§2.1, §2.3)

**Do**:
- `internal/migrate/beads/beads.go`: `BeadsProvider.Tasks` drops the ceiling of its `bufio.NewScanner` (line 84). It keeps its own line rules: each line is trimmed, and a line empty after trimming is skipped. It does not use the shared reader (§2.3).

**Context**:
> §2.3: "`tick migrate --from beads` reads `.beads/issues.jsonl` — an external file, not the store — so it does not use the shared reader and keeps its own line rules (it trims each line). It drops the ceiling the same way: an `issues.jsonl` line of any length is read."
>
> Each imported issue is written into the store, so an issue whose tick record also passes 65,536 bytes depends on Task 1-1's reader to reopen the store after the import.
>
> Phase 3 has migrate skip an issue whose description is over 50,000 characters, or whose title is over 500 characters or spans more than one line (§6.4). The long issue here keeps importing after Phase 3 only if its line passes 64 KiB within those limits. The spec gives two ways text's byte length outruns its character count: JSON escaping of the source text, at up to 6 bytes per character (§1.2), and multibyte characters, which count once toward the cap (§6.1).

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §2.1, §2.3, §6.1, §6.4, §8.1

## oversized-text-field-bricks-the-store-1-6

### Task 1-6: Store read errors name the line and the task

**Problem**: When a line of `tasks.jsonl` fails to load, the store names the line by number but not the task. Today's wording is `failed to parse tasks.jsonl: failed to parse line 2: <reason>`. The old scanner failure named neither the line nor the task (§1.1). A store that will not load can only be recovered by hand-editing `tasks.jsonl`, and the agent doing that has only the error message to find the record and see what is wrong with it.

**Solution**: Make every error from reading `tasks.jsonl` into the store name the line that stopped it, 1-based and numbered as §2.2 defines. When the failing line is a JSON object whose `id` is a string, the error also names that task ID. The loader's own reason follows.

**Outcome**: The error from any command on a store that cannot load tells the agent which line to open, which task it holds when that can be recovered, and what the loader rejected.

**Acceptance Criteria**:
- [ ] A store whose line 2 holds task `tick-a1b2c3` with a wrong-typed field (`"priority":"high"`): `tick list` exits 1 with an error naming line 2, `tick-a1b2c3` and the decoder's reason, for example `failed to parse tasks.jsonl: line 2 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int` (§3, §8.2)
- [ ] Same store: a write such as `tick create` and `tick rebuild` each fail with the same line, task ID and reason (§3)
- [ ] A line carrying a string `id` whose `created` timestamp does not parse: the error names the line, that ID and the timestamp reason (§3)
- [ ] A malformed-JSON line or a whitespace-only line: the error names the line by number alone, with no task ID, followed by the reason (§3, §8.2)
- [ ] A line with no string `id` (the key is missing, the `id` is not a string, or the value is not a JSON object, such as `null` or `[]`): the error names the line by number alone, followed by the reason (§3)
- [ ] An empty line 2 followed by a wrong-typed task on line 3: the error names line 3, counting the skipped line (§2.2, §3)

**Do**:
- `internal/storage/jsonl.go`: `ParseJSONL`'s per-line failure, today `failed to parse line %d: %w`, carries the line number and, where the line holds one, the task ID. Every command sees it through the store's `failed to parse tasks.jsonl:` wrap in `ReadTasks`, `Rebuild` and `readAndEnsureFresh` (`internal/storage/store.go`).
- Revisit `TestParseJSONL`, including `it returns error for invalid JSON in bytes` (`internal/storage/jsonl_test.go`) (§8.6).

**Context**:
> §3: "Every error from reading `tasks.jsonl` into the store names the line that stopped it — 1-based, numbered per §2.2 — and, where the line carries one, the task: when the failing line is a JSON object whose `id` is a string, the error includes that ID. A line that is not valid JSON, or has no string `id`, is named by line alone. The loader's own reason follows." The purpose is hand-edit recovery: "the message alone tells the agent which record to open and what is wrong with it."
>
> Loading a line includes its timestamp parsing. `Task.UnmarshalJSON` (`internal/task/task.go:109`) decodes into `taskJSON` and then parses `created`, `updated` and `closed`. `null` and `{}` therefore fail on the empty `created` timestamp, and `[]` fails in decoding. The failure reason stays the loader's own.
>
> Phase 2's doctor reports the loader's reason for each line that fails to load (§5.1). Its fixtures are confirmed to fail `tick list` naming the same line (§8.4), so the store's line naming set here is what doctor's verdict is checked against.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.1, §2.2, §3, §5.1, §8.2, §8.4, §8.6

## oversized-text-field-bricks-the-store-1-7

### Task 1-7: `tick rebuild` keeps the cache until it can build a new one

**Problem**: `tick rebuild` deletes `cache.db` first and parses `tasks.jsonl` second (`internal/storage/store.go:236` logs `deleting cache.db`, `:242` logs `reading JSONL`). When the parse fails, one failed rebuild leaves the project worse off than before. Doctor then reports `cache.db not found` and advises running `tick rebuild`, the command that just failed.

**Solution**: Reorder rebuild so it reads and parses `tasks.jsonl` before it touches `cache.db`. The old cache is removed, and a fresh one built from the parsed tasks, only after a successful parse.

**Outcome**: A rebuild that cannot parse the store exits 1 with the parse error and leaves `cache.db` exactly as it was. A rebuild over a parseable store still replaces the cache, including a corrupt one.

**Acceptance Criteria**:
- [ ] A store that fails to parse (for example, a wrong-typed field on one line) with an existing `cache.db`: `tick rebuild` exits 1 with the parse error naming the line, and `cache.db` is byte-for-byte unchanged (§4, §8.3)
- [ ] `tick rebuild --verbose` on a valid store logs `reading JSONL` before `deleting cache.db` (§4, §8.3)
- [ ] A valid store whose `cache.db` is corrupt: `tick rebuild` succeeds, and the rebuilt cache answers queries (§4, §8.3)

**Do**:
- `internal/storage/store.go`: `Store.Rebuild` reads and parses `tasks.jsonl` before it deletes `cache.db`. Only after a successful parse is the old cache removed and a fresh one built from the parsed tasks.
- `internal/storage/store_test.go`: `it logs verbose messages during rebuild` (line 679) is updated to the new order. `it works when cache.db is corrupted` (line 571) still passes (§8.3).

**Context**:
> §4: "`tick rebuild` reads and parses `tasks.jsonl` before it touches `cache.db`. If the parse fails, rebuild exits 1 with the parse error (§3) and the existing `cache.db` is left exactly as it was. Only after a successful parse is the old cache removed and a fresh one built from the parsed tasks." It adds: "Rebuild still recovers from a corrupt `cache.db`. Its `--verbose` log follows the new order: `reading JSONL` comes before `deleting cache.db`."
>
> The parse error is whatever the store reports for the failing line. It names the line under Task 1-1, and Task 1-6 adds the task ID.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §3, §4, §8.3, §8.6
