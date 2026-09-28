# Specification: Oversized Text Field Bricks The Store

## Specification

### 1. Overview

#### 1.1 The defect

A task whose stored JSONL line reaches 65,536 bytes is accepted and written, and from then on every tick command that opens the store fails with `failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long`. That covers reads, writes and `tick rebuild` alike, including `update`, `remove` and `cancel` of the oversized task itself. The message names neither the line nor the task. Meanwhile `tick doctor` reports the store healthy. The only recovery today is hand-editing `.tick/tasks.jsonl`.

It doesn't matter which command wrote the line. `create`, `update` and `note add` commit the write, then exit 1 outside `--quiet` when their read-back fails. Status changes and `dep add` exit 0 with a normal success document. Cascades, `--blocks` and the Rule 6 done-parent reopen grow tasks the command never named, so `tick start <child>` can push a large parent over the line.

#### 1.2 Root causes

There are two defects, independent of each other, meeting at the same ceiling:

- **The store writes lines its own reader cannot read.** The store's reader parses `tasks.jsonl` with a `bufio.Scanner` at Go's default maximum token size (`go doc bufio.MaxScanTokenSize` → `MaxScanTokenSize = 64 * 1024`), so any line of 65,536 bytes or more stops it. Nobody chose that limit, and nothing on the write path knows about it:
  - Fields are bounded in characters: title ≤ 500, each note ≤ 2000.
  - The description is unbounded.
  - The encoded record is never measured.

  JSON escaping inflates text by up to 6 bytes per character (`<`, `>`, `&`, U+2028/U+2029, and control characters other than `\n`, `\r`, `\t`). Note, transition and `blocked_by` counts are unbounded. So no per-field character cap can keep a record under the ceiling. Every command that opens the store parses the whole file before acting, so one such line makes the entire store unreachable through tick.
- **Doctor treats a partial read as a full read.** Doctor's reader has the same ceiling and never checks the scanner's error. When it stops at an unreadable line, it returns the lines before that line with no error, and every check runs over that prefix as if it were the whole store. It passes a store no other command can open. It can also report false errors about references into the part it never read, for example `Orphaned dependencies` naming a task that exists after the cut.

  Separately, doctor's readability test is looser than the store's. Doctor asks only whether a line is valid JSON; the store needs a line that loads as a task. So a line such as `"priority":"high"` passes doctor and still makes every command fail.

#### 1.3 The fix

- Every reader of `tasks.jsonl` accepts a line of any length, and the store and doctor read through one shared line reader (§2). This alone stops any route from breaking the store by line length, and it reopens stores already over the old ceiling.
- Store read errors name the line and, where it can be recovered, the task (§3).
- `tick rebuild` leaves the existing cache in place until it knows it can build a new one (§4).
- Doctor judges every line the way tick loads it, and fails on anything it could not read (§5).
- The description gets a generous character cap. It serves as hygiene, not as the safety device (§6).

#### 1.4 Who meets these behaviours

Agents write the tasks, not people, so an agent is what meets a write-time refusal. Every refusal must say which field, what limit, and that nothing was saved. Agents also run `tick doctor` as a health check, and a false "healthy" tells an agent a store is fine when no command can open it. A passing doctor must therefore mean every tick command can open the store.

### 2. Reading `tasks.jsonl`

#### 2.1 No line-length ceiling

Every reader of `tasks.jsonl` accepts a line of any length the file holds. The read side imposes no per-line size limit — neither Go's default scanner limit (§1.2) nor any larger one chosen in its place. With it gone, no route can make the store unreadable by line length: the CLI's own writes, cascades onto tasks a command never named, status changes, `dep add`, `migrate`, hand edits, other tools, or lines written before this fix. A store already holding an over-ceiling line opens once tick is upgraded, with no migration or repair step, and the oversized task can then be shown, updated and removed like any other.

The ceiling bought no resource protection: every store read already loads the whole file into memory before parsing (`rg -n 'os.ReadFile\(s.jsonlPath\)' internal/storage/store.go` → `:159`, `:243`, `:360` — `ReadTasks`, `Rebuild`, `readAndEnsureFresh`).

The readers in scope are the production line scanners (`rg -n 'bufio.NewScanner' internal cmd -g '!*_test.go'` → `internal/storage/jsonl.go:95`, `internal/doctor/jsonl_reader.go:37`, `internal/migrate/beads/beads.go:84`). The first two read `tasks.jsonl` and move onto the shared reader (§2.2); the third reads an external file (§2.3).

#### 2.2 One shared line reader

The store and doctor read `tasks.jsonl` through one shared line reader, so the two cannot disagree about what the file holds and no future reader of the store can reintroduce a ceiling. It carries one definition of a line — the store's current one, preserved exactly, so commands behave as they do today on every store the old reader could read:

- A line ends at `\n`; one `\r` immediately before it is stripped, so CRLF files read the same as LF. A final line with no trailing newline is still read.
- Lines are numbered from 1, and every line counts toward the numbering, skipped ones included.
- A line that is empty once its terminator is removed is skipped.
- A line holding only whitespace is not skipped. It is an ordinary line that must load as a task, and fails to (§5.1) — as it does in the store today (measured on current main: a trailing `   ` line makes `tick list` fail with `failed to parse line 3: unexpected end of JSON input`).

Doctor's current rule of skipping any whitespace-only line (`rg -n 'TrimSpace\(text\) == ""' internal/doctor/jsonl_reader.go` → `:44`) gives way to the store's (`rg -n 'line == ""' internal/storage/jsonl.go` → `:100`), because doctor's verdict must match what commands actually do (§5).

#### 2.3 The beads importer's reader

`tick migrate --from beads` reads `.beads/issues.jsonl` — an external file, not the store — so it does not use the shared reader and keeps its own line rules (it trims each line). It drops the ceiling the same way: an `issues.jsonl` line of any length is read. Today a line of 65,536 bytes or more aborts the whole import before anything is written, with an error that names no line.

#### 2.4 Older tick binaries

`tasks.jsonl` is tracked in git, so collaborators on different tick versions can share one store. A binary from before this fix keeps the 65,536-byte ceiling, so a store written by a fixed tick that holds a longer line stays unreadable to it. The description cap (§6) makes plain-text records of that size rarer but cannot rule them out — escape inflation and unbounded note counts remain. Accepted as the cost of the fix; upgrading is the remedy.

### 3. Store read errors

Every error from reading `tasks.jsonl` into the store names the line that stopped it — 1-based, numbered per §2.2 — and, where the line carries one, the task: when the failing line is a JSON object whose `id` is a string, the error includes that ID. A line that is not valid JSON, or has no string `id`, is named by line alone. The loader's own reason follows. For example: `failed to parse tasks.jsonl: line 2 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int`.

Today a line that fails to decode is named by number but not by task, and the scanner's failure named neither (§1.1). The purpose is hand-edit recovery: the message alone tells the agent which record to open and what is wrong with it.

### 4. `tick rebuild` keeps the cache until it can build a new one

`tick rebuild` reads and parses `tasks.jsonl` before it touches `cache.db`. If the parse fails, rebuild exits 1 with the parse error (§3) and the existing `cache.db` is left exactly as it was. Only after a successful parse is the old cache removed and a fresh one built from the parsed tasks.

Today rebuild deletes the cache first and parses second (`rg -n '"deleting cache.db"|"reading JSONL"' internal/storage/store.go` → `:236` deleting, `:242` reading), so one failed rebuild leaves the project worse off: doctor then reports `cache.db not found` and advises running `tick rebuild` — the command that just failed.

Rebuild still recovers from a corrupt `cache.db`. Its `--verbose` log follows the new order: `reading JSONL` comes before `deleting cache.db`.

### 5. Doctor

#### 5.1 Each line judged the way tick loads it

Every line doctor reads through the shared reader (§2.2) must load as a task exactly as every tick command loads it — the same decoding, timestamp parsing included. A line that does not load fails the JSONL check, one failure per line, naming the line and giving the loader's reason. That covers:

- malformed JSON;
- a whitespace-only line;
- valid JSON of the wrong shape — `null`, `[]`, `{}`;
- a wrong-typed field, such as `"priority":"high"`.

Each of these makes every command fail today while doctor's JSONL check passes it (measured on current main with a built binary, each line added to a two-task store: `tick list` exits 1 naming the line; `tick doctor` prints `✓ JSONL syntax: OK`, flags only the stale cache, and advises `tick rebuild`).

Validation the loader does not itself enforce — enum membership (status, type) and value ranges (priority 0–4) — stays at write time and is not doctor's.

A doctor run that reports no errors therefore means every tick command can open the store. A valid store still reports `No issues found`.

#### 5.2 The check keeps its name

The check stays `JSONL syntax` in doctor's output, in the README's doctor "Checks for:" list (`README.md:396`) and in `tick help doctor` (`internal/cli/help.go:217`); none of those change. What changes is its failure detail: a line that fails to load reports the loader's reason in place of `invalid JSON`. Its suggestion stays a hand fix of the named line.

#### 5.3 A read doctor cannot complete

If reading `tasks.jsonl` stops before the end of the file — the file opened, but an error cut the read short — doctor reports a failure naming the line it could not read and saying the file could not be read in full. It never reports a pass over the lines it did read, and never reports `tasks.jsonl not found`. Every check that consumes the file's lines fails with that read error in place of its normal verdict, the same shape as today's failure when the file will not open. The not-found result (`rg -n 'func fileNotFoundResult' internal/doctor/helpers.go` → `:14`) stays, for a file that cannot be opened and nothing else.

Today the reader's error is dropped. Surfaced through the existing per-check fallback, it would make every check report the file missing (§1.2).

Once the ceiling is gone, no file on disk makes the reader fail this way, so the path is reached through a test seam over the reader (§8).

#### 5.4 Relationship checks

"Partial" means lines the reader never reached. A line the reader did reach but that does not load is reported by the JSONL check (§5.1). The relationship and hierarchy checks keep their designed behaviour of skipping a line that is not a JSON object or has no string `id` (`rg -n 'jl.Parsed == nil' internal/doctor/task_relationships.go` → `:26`). Because the whole file is now read, tasks after a long line are no longer invisible to them, so a mid-file oversized line produces no false orphan error.

#### 5.5 No advice to rebuild a store that cannot load

While any line fails to load (§5.1) or the read is incomplete (§5.3), no doctor output directs the user to `tick rebuild`, which fails the same way until the line is fixed. The cache check still reports what it finds, since a stale or missing cache is a true finding. But its suggestion points at fixing the lines the JSONL check names, not at `tick rebuild`. Once those lines are fixed, the next doctor run gives the usual rebuild advice if the cache is still stale.

Everything else about the cache check is unchanged. It hashes the raw file bytes and its verdict is correct.

### 6. Description cap

#### 6.1 The cap

A task description is capped at **50,000 characters**. It is counted the way the title (500) and note (2000) caps are: Unicode characters, not bytes, after leading and trailing whitespace is trimmed. A description of exactly 50,000 characters is accepted, multibyte ones included; 50,001 is refused. The cap is a fixed constant alongside the title and note caps, not a flag, so the command flag registry, `tick help` and the README do not change.

The cap is hygiene, not the store's safety (§2.1). It keeps free text finite: without it an agent can write a description of any size that every later `tick show` returns in full. It sits well above real use so no legitimate description meets it (longest real description across the maintainer's tick stores: `jq -r '(.description // "") | length' ~/Code/*/.tick/tasks.jsonl | sort -n | tail -1` → `20785`); a description near it signals a task that wants breaking up. It is not sized against the 64 KiB line older binaries still enforce (§2.4).

#### 6.2 Where it is enforced

Every route that sets a description enforces it: `tick create --description`, `tick update --description`, and `tick migrate` (`rg -l 'TrimDescription\(' internal/cli internal/migrate -g '!*_test.go'` → `internal/cli/create.go`, `internal/cli/update.go`, `internal/migrate/migrate.go`). The check runs before anything is written, so a refused `create` or `update` exits 1 and leaves `tasks.jsonl` and the cache untouched.

The cap applies to a description being set. A description already stored over it is left as it is, reads normally (§2.1), and does not block other changes to its task — a status change, a new title, a note.

#### 6.3 The refusal

The refusal names the field, the limit, the length submitted, and that nothing was saved, so an agent knows how much to cut and that the write must be retried. For example: `description is 61,204 characters, over the 50,000-character limit; nothing was saved`. The four elements are required; the exact wording is the implementer's. On `create` and `update` it is the command's error, exit 1. On `migrate` it is the skipped issue's reason (§6.4).

#### 6.4 Migrate

`tick migrate` validates each issue before creating it. An issue whose description exceeds the cap is skipped and reported with its reason, exactly as the import already treats any other invalid issue (`rg -n 'if err := mt.Validate' internal/migrate/engine.go` → `:74`). The remaining issues import. The description is never truncated, and one oversized issue never aborts the import.

Migrate also adopts the CLI's title rules. Today it checks only that the title is non-empty. An issue whose title exceeds 500 characters or spans more than one line is now skipped the same way, with its reason.

`--dry-run` reports the same refusals as a real run, because validation happens before the dry run's no-op creator is reached.

### 7. Unchanged by this work

- **No record-level size bound.** Nothing on the write path measures a record's encoded size or refuses a write for its byte length. With the reader's ceiling gone (§2.1), such a bound would protect nothing.
- **Note, transition and `blocked_by` counts stay unbounded.** The title (500), note (2000), tag and ref caps are unchanged.
- **Title and note refusals keep their current messages.** They name the field and limit but not the submitted length or that nothing was saved; bringing them in line with §6.3 is not part of this work.
- **The post-write signal needs no change of its own.** Today a write that pushes a record over the ceiling is committed and then reported as failed by `create`, `update` and `note` (exit 1), or as plain success by status changes and `dep add` (§1.1). Once the reader accepts any line (§2.1), a write can no longer produce a line the reader refuses, so the read-back succeeds and both inconsistencies disappear.

---

## Working Notes
