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

---

## Working Notes
