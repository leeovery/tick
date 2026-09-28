# Specification: Oversized Text Field Bricks The Store

## Specification

### 1. Overview

#### 1.1 The defect

A task whose stored JSONL line reaches 65,536 bytes is accepted and written, and from then on every tick command that opens the store fails with `failed to parse tasks.jsonl: error reading JSONL data: bufio.Scanner: token too long` — reads, writes and `tick rebuild` alike, including `update`, `remove` and `cancel` of the oversized task itself. The message names neither the line nor the task. Meanwhile `tick doctor` reports the store healthy. The only recovery today is hand-editing `.tick/tasks.jsonl`.

The failure is independent of which command wrote the line. `create`, `update` and `note add` commit the write and then, outside `--quiet`, exit 1 when their read-back fails; status changes and `dep add` exit 0 with a normal success document. Cascades, `--blocks` and the Rule 6 done-parent reopen grow tasks the command never named, so `tick start <child>` can push a large parent over.

#### 1.2 Root causes

Two defects, independent of each other, meeting at the same ceiling:

- **The store writes lines its own reader cannot read.** The store's reader parses `tasks.jsonl` with a `bufio.Scanner` at Go's default maximum token size (`go doc bufio.MaxScanTokenSize` → `MaxScanTokenSize = 64 * 1024`), so any line of 65,536 bytes or more stops it. Nobody chose that limit and nothing on the write path knows of it: fields are bounded in characters (title ≤ 500, each note ≤ 2000), the description is unbounded, and the encoded record is never measured. JSON escaping inflates text up to 6 bytes per character (`<`, `>`, `&`, U+2028/U+2029, and control characters other than `\n`, `\r`, `\t`), and note, transition and `blocked_by` counts are unbounded — so no per-field character cap can keep a record under the ceiling. Every command that opens the store parses the whole file before acting, so one such line makes the entire store unreachable through tick.
- **Doctor treats a partial read as a full read.** Doctor's reader has the same ceiling and never consults the scanner's error. When it stops at an unreadable line it returns the lines before it with no error, and every check runs over that prefix as if it were the whole store — passing a store no other command can open, and able to report false errors about references into the part it never read (e.g. `Orphaned dependencies` naming a task that exists after the cut). Separately, doctor's readability test is looser than the store's: it asks only whether a line is valid JSON, while the store needs a line that loads as a task — so a line such as `"priority":"high"` passes doctor and still makes every command fail.

#### 1.3 The fix

- Every reader of `tasks.jsonl` accepts a line of any length; the store and doctor read through one shared line reader (§2). This alone makes the store unbrickable by line length from every route, and reopens stores already over the old ceiling.
- Store read errors name the line and, where recoverable, the task (§3).
- `tick rebuild` leaves the existing cache in place until it knows it can build a new one (§4).
- Doctor judges every line the way tick loads it, and fails on anything it could not read (§5).
- The description gets a generous character cap — hygiene, not the safety device (§6).

#### 1.4 Who meets these behaviours

Tasks are written by agents, not by hand, so the party that meets a write-time refusal is an agent: every refusal must say which field, what limit, and that nothing was saved. Agents also run `tick doctor` as a health check; a false "healthy" tells an agent a store is fine when no command can open it. A passing doctor must therefore mean every tick command can open the store.

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

---

## Working Notes
