# Specification: Free Text Round Trip

## Specification

### 1. Purpose

`tick`'s toon output is the agent-facing format — the default whenever stdout is not a TTY. It does not parse. A standard TOON reader fails on the first line of `tick show` output and never reaches anything beneath it:

> `line 1: invalid unquoted key "task{id,title,status,priority,type,created,updated}"`

The failure that triggered this work was an agent abandoning the CLI and reading `.tick/tasks.jsonl` directly to obtain a task's description as a plain string. The file told it the truth without a decoding rule it had to already know.

The cause is uniform. Every section the formatter assembles by hand is malformed; every section it hands to the TOON library is correct.

| Written by | Sections | Parses |
|---|---|---|
| The library (`encodeToonSection`) | blockers, children, notes, priority breakdown, dep-tree edges | yes |
| Hand-assembled string building | task header, stats summary, dep-tree summary, tags, refs, description | no |

Each hand-rolled section exists because it wanted a shape the library does not produce directly — a singular object header, a list down the page, an unstructured text block. In every case the invented shape turned out to be invalid.

**This work delivers toon output that a standard TOON reader can read, for every command that returns data, and free text that survives an agent's read-edit-write cycle without a rule learned outside the output.**

Measured against `github.com/toon-format/toon-go@v0.0.0-20251202084852` (`grep toon-go go.mod`). Whether the TOON specification itself permits a single-object scope header was not verified — the claim is only that the reference Go implementation rejects it, and that the form is hand-constructed by tick rather than produced by the library.

### 2. Round-Trip Contract

#### 2.1 Both reading paths are in scope

An agent must be able to fetch one field bare, **and** must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback: the field flag (§9) does not excuse an ambiguous block in full output, and repairing the block does not remove the need for bare single-field output.

#### 2.2 The fidelity bar is byte-identity

Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was.

Existing whitespace trimming does not stand in the way of this. `create` and `update` both run a description through `TrimDescription` before storing (`grep -n 'TrimDescription(' internal/cli/create.go internal/cli/update.go` → `create.go:214`, `update.go:193`, `update.go:342`), which is `strings.TrimSpace` (`grep -n 'func TrimDescription' -A 2 internal/task/task.go` → `task.go:193-195`). No stored description can therefore carry leading or trailing whitespace, and the trim is idempotent over anything that came out of storage. The trimming behaviour is out of scope as a defect.

The bar does not hold for free of charge across all three free-text carriers. Note text and task titles are rejected before reaching storage when they begin with a dash — see §10. Reaching byte-identity for them requires that fix, which this work carries.

#### 2.3 What is not a defect

The current output is not lossy. The description block prefixes two spaces to every line, so blank lines emerge as two spaces and originally-indented lines at four; stripping exactly two from each line returns the original bytes. Note text passes through the library's tabular encoder, which quotes and escapes any string containing a newline, carriage return or tab, so a multi-line note survives as `"multi\nline\nnote"`.

The defect is that the two free-text fields use two different, mutually incompatible decoding rules, neither of which the output announces, and that the description block has no count and no terminator — it works today only because it is emitted last and runs to EOF (`sed -n '104,109p' internal/cli/toon_formatter.go`).

### 3. Output Inventory

#### 3.1 Output that must parse

Every command listed here emits output a standard TOON reader decodes, on **every** branch — the empty one included (§8).

| Output | Commands |
|---|---|
| Task detail | `show`, `create`, `update`, `note add`, `note remove` (all via `outputMutationResult`, `grep -n 'func outputMutationResult' internal/cli/helpers.go` → `helpers.go:16`) |
| Task list | `list`, `ready`, `blocked` |
| Stats | `stats` |
| Dependency graph | `dep tree` |
| Status change | `start`, `done`, `cancel`, `reopen` |

#### 3.2 Output that stays prose

`dep add`, `dep remove`, `remove`, `init`, and the general-purpose messages. These are confirmations of a command the caller issued: the caller already knows what it asked for and the exit code says whether it worked. Wrapping a one-line confirmation in a data format costs tokens to restate what the caller already knows and introduces a new way to fail.

The boundary this draws: a confirmation may be prose; an **answer to a query** may not. "No dependencies" is not a confirmation — it is the answer the caller ran the command to find out (§8).

#### 3.3 `doctor` and `migrate` are out of scope

Both bypass the formatter entirely and print straight to the terminal — `handleDoctor` documents this in its own comment (`sed -n '44,46p' internal/cli/doctor.go`), and `RunMigrate` presents through `migrate.Present` rather than a `Formatter` (`grep -n 'migrate.Present' internal/cli/migrate.go` → `migrate.go:124`). Neither honours the format flags.

The defect being fixed is output that *claims* to be machine-readable and is not — a header no reader accepts, a list missing its item markers. These two never claimed it. Bringing them in means building new formatter surface rather than repairing broken output, which is different work.

Cost accepted: an agent running the health check reads a paragraph and works out what to do from the words.

### 4. Formatter Scope

#### 4.1 Pretty is unchanged, everywhere

What a terminal prints today is what it prints after this work — the single transition line, the box-drawing cascade tree, the indented description block, the dep-tree prose on empty branches. Pretty is the human surface; this work is about the agent surface, and nothing in pretty is broken by the standard being applied: it never claimed to be machine-readable, and a human reads it fine.

Two candidate changes were declined explicitly and are **not** in scope:

- Replicating §7's table shape in pretty. The current cascade tree nests by which task caused which, so a grandchild closing because its parent closed shows as three levels of indentation. Flattened into rows, every knock-on looks equally directly caused. The toon table drops that too, but an agent holds the parent/child links and can reconstruct the chain; a human reading a terminal cannot, which is why the tree exists.
- Adding the task's title to pretty's single transition line. A genuine improvement rather than a defect fix, and out of scope.

#### 4.2 JSON moves with toon

A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`).

#### 4.3 Two shared code paths must be split, not edited

Pretty being unchanged while toon and JSON move is not free — the code is shared in two places, and editing it in place would change pretty by accident:

- **The single transition line** comes from `baseFormatter.FormatTransition` (`grep -n 'func (b \*baseFormatter) FormatTransition' internal/cli/format.go` → `format.go:211`), embedded by both the toon and pretty formatters, so the two emit byte-identical text today. Restructuring the toon form requires splitting that method.
- **The dep-tree empty messages** are not produced by a formatter at all. They are set on the result in the shared graph builder (`grep -n 'No dependencies' internal/cli/dep_tree_graph.go` → `dep_tree_graph.go:177`, `dep_tree_graph.go:255`) and consumed by all three formatters, so removing them at source would strip pretty's message too. Pretty keeps its sentence; only the machine formats take the structured empty form.

### 5. Single-Object Sections Become Top-Level Named Fields

#### 5.1 The current shape and its cause

Three places in the output describe one thing rather than a list of things: the task's own fields at the head of `tick show` (and of `create`, `update`, `note add`, `note remove`), the counts summary in `tick stats`, and the chains/longest/blocked summary in `tick dep tree`. All three are malformed, and this is the line a reader fails on before it sees anything else.

The cause is a hand-edit. The value is marshalled as a one-element array and the `[1]` is then deleted from the header with a string replace to make it read as singular — `buildTaskSection` does it inline (`grep -n 'strings.Replace(s, "task\[1\]"' internal/cli/toon_formatter.go` → `toon_formatter.go:294`) and `encodeToonSingleObject` does it generically for stats and the dep-tree summary (`grep -n 'func encodeToonSingleObject' -A 10 internal/cli/toon_formatter.go` → `toon_formatter.go:371-379`). The result is a table header with no table beneath it, a shape TOON has no equivalent for.

#### 5.2 The required shape

**The task's own fields sit at the top level of the document, as named fields, with no wrapping key.** They are peers of the collection sections rather than nested inside a `task:` scope:

```
id: tick-a1b2
title: Add retry to the sync worker
status: in_progress
priority: 2
type: feature
created: "2026-09-10T09:14:00Z"
updated: "2026-09-17T16:00:00Z"

children[1]{id,title,status}:
  tick-9f3c,Parse the header,done

description: "Fix it.\n\nSteps."
```

**The same treatment applies to the other two single-object sites**: `tick stats`' counts and the dep-tree chains/longest/blocked summary become top-level named fields beside their tables.

#### 5.3 Why named fields rather than a one-row table

Two conformant options existed; both were encoded and decoded back with the project's TOON library, and both round-trip.

The one-row table (`task[1]{id,title,…}:` with a single row beneath) is three characters longer than today's broken output and is the library's own output with nothing stripped. Its cost is positional reading: a consumer counts commas across to the matching name in the header. The table layout earns its keep when many rows would otherwise repeat the field names — that is `tick list`'s case, not this one. With a single row it repeats nothing, so it compresses nothing, and it trades that for a positional read that can go wrong.

Named fields put every value beside its name, so adding or reordering a field cannot break a positional read and a value containing a comma is safe without relying on quoting discipline.

Neither form is an invention. TOON is a compact way of writing JSON: an object inside an object is written as the key with its fields indented beneath (the named-fields form), and a list of same-shaped objects is written as a header plus rows (the table form, which is what `tick list` uses). What tick invented was neither.

#### 5.4 Why no wrapping key

Measured: 256 characters flat against 276 wrapped, for the same content, both parsing and round-tripping.

Three reasons, the size being the least of them:

1. **It removes an inconsistency the wrapper created.** `description` is a task field and sits at the top level. `title` is a task field and sat nested. The same kind of thing at two different depths, for no reason beyond one of them being long.
2. **It mirrors storage.** The JSONL file already holds each task as a flat object — `{"id":…,"title":…,"status":…,"description":…}` — so the output stops inventing a grouping that exists nowhere else in the system.
3. **The field flag falls out of it.** `--field title,status` returns two lines with nothing wrapped around them, rather than a trimmed block (§9.4).

Given up: a consumer can no longer grab "the task's own fields" as one object without naming them. Nothing identified wants that.

---

## Working Notes
