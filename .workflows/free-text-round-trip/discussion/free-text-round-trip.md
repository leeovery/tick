# Discussion: Free Text Round Trip

## Context

Free-text fields on a task — the description and the notes — must survive an agent's read-edit-write cycle through the toon output format without a transformation step in either direction. An agent runs `tick show <id>`, takes the description text, edits it, and writes it back with `tick update --description`. Today that cycle does not close cleanly, and the failure that triggered this work was an agent abandoning the CLI altogether and reading `.tick/tasks.jsonl` directly to get a task's description as a plain string.

The existing escape hatch, `tick show --json`, works but is the wrong shape: it costs extra tokens and forces the agent to parse JSON syntax back off the string it wanted. The irony named during discovery — the token-efficient format exists for agents, and it is the format agents cannot use.

### Inherited position (from discovery)

Carried forward as working ground, not re-opened:

- **Scope is free text generally**, not descriptions alone. Notes were named as carrying the same problem, reaching output through the same formatter.
- **The primary fix is repairing the toon format** for round-trip fidelity, not adding a parallel raw output format.
- **Field extraction is a companion of interest** — `tick show <id> --field description` printing the bare string and nothing else: no header, no indentation, no JSON quoting.
- **A fourth output format was rejected.** `--raw` was pressed on and dropped: a format has to answer for every command in the CLI (`tick list --raw`, `tick stats --raw`) and carries an ongoing consistency burden across all of them. Field extraction was the alternative framing that avoids designing a format at all.
- **Work type is feature**, not bugfix — the indentation is deliberate design that defeats its own purpose, not a regression.

### Open tension carried in from discovery

TOON is an indentation-scoped format. Removing the indent from a free-text block requires some other means of marking where the block ends, and a description line resembling a section header is precisely the case a naive delimiter fails on. This is what made the work a discussion rather than a direct route to specification.

### Current state, measured

The description section prefixes every line of the description with two spaces and emits no terminator:

`sed -n '345,355p' internal/cli/toon_formatter.go` → `buildDescriptionSection` writes `description:` then `"\n  "` before each line produced by `strings.SplitSeq(desc, "\n")`.

The notes section takes a different path. Notes are encoded as a TOON tabular section through the `toon-go` library, and the library quotes and escapes any string containing a newline:

`grep -n "case '\\\\n':" "$(go list -m -f '{{.Dir}}' github.com/toon-format/toon-go)/internal/format/format.go"` → `QuoteString` maps `\n` to the two-character escape `\n`; `NeedsQuoting` returns true for any string containing `\n`, `\r` or `\t`.

Observed output for a task carrying a multi-line description and multi-line notes (rendered through `ToonFormatter.FormatTaskDetail`):

```
notes[3]{text,created}:
  single line note,"2026-09-16T12:00:00Z"
  "multi\nline\nnote","2026-09-16T12:00:00Z"
  "note, with comma and \"quotes\"","2026-09-16T12:00:00Z"

description:
  Line one
  
  Line two with trailing space   
    indented line
  notes[2]{text,created}:
  tabs	here
```

Three things follow from that output, and they change the shape of the problem discovery described:

1. **Notes already round-trip losslessly.** A multi-line note is emitted as one quoted, backslash-escaped TOON string. The data survives; what the agent must do is unquote and unescape — a transformation, but a standard one with an unambiguous result.
2. **The description transform is reversible but undeclared.** Stripping exactly two spaces from every line recovers the original, including blank lines (which render as a line of two spaces), preserved trailing whitespace, and originally-indented lines (which render at four). Nothing in the output says so, and no count or terminator marks where the block ends — it works today only because description is emitted last and runs to EOF.
3. **The "indistinguishable header" case is narrower than stated in discovery.** A description line reading `notes[2]{text,created}:` lands at column 2; a real section header sits at column 0. A parser that respects indentation can tell them apart. A parser that scans for the pattern anywhere on the line cannot.

So the defect is not data loss in the toon output. It is that reading free text out of it requires a per-field transformation the format does not announce, and that the two free-text fields use two different and mutually incompatible schemes.

The write side has its own fidelity question, independent of format: `tick update --description` runs the value through `TrimDescription`, which is `strings.TrimSpace` (`internal/task/task.go:193`), so leading and trailing whitespace does not survive a write even when the read was exact.

`show` currently accepts no command-specific flags at all — `grep -n '"show":' internal/cli/flags.go` → `"show":        {},` — so `--field` would be the first, alongside the global `--quiet` which already prints a bare task ID and nothing else.

### References

- Discovery session log: `.workflows/free-text-round-trip/discovery/sessions/session-001.md`
- Manifest `description` (the work unit's carrier)
- `internal/cli/toon_formatter.go` — `buildDescriptionSection`, `buildNotesSection`, `FormatTaskDetail`
- `internal/cli/show.go` — `RunShow`, `queryShowData`
- `internal/cli/update.go` — `--description` / `--clear-description` parsing and validation
- `internal/cli/note.go` — `RunNoteAdd`, note text assembly from argv
- `internal/task/notes.go`, `internal/task/task.go` — `TrimNoteText`, `TrimDescription`, length validation
- `internal/cli/flags.go` — `commandFlags` registry and `globalFlagSet`

---

*Subtopics are documented below as they reach `decided` or accumulate enough exploration to capture.*

---

## Round Trip Contract

### Context

"Free text survives the round trip" needs a definition before any of the encoding subtopics can be judged against it. Two things had to be pinned: which reading paths the guarantee covers, and what fidelity it promises.

### Journey

The session opened by measuring the current output rather than taking discovery's account of it. Three findings reshaped the problem:

1. **Notes are not lossy.** A note containing newlines is emitted as a single TOON-quoted string with the newlines escaped (`"multi\nline\nnote"`). Nothing is lost; the reader unquotes a standard TOON string and the result is unambiguous. Discovery recorded notes as carrying the same problem as descriptions — they carry a different one.
2. **The description block is not lossy either.** Two spaces are prefixed to every line; blank lines emerge as two spaces, originally-indented lines at four. Stripping exactly two from each line returns the original bytes. The transform is reversible.
3. **What is missing is the declaration.** Nothing in the output says the description block is indented, and nothing marks where it ends — it works today only because the description is emitted last and runs to EOF.

So the defect is not data loss. It is that the two free-text fields use two different, mutually incompatible decoding rules and neither announces itself. That is why the agent in the triggering incident read `.tick/tasks.jsonl` directly: the file told it the truth without a rule it had to already know.

That led to the framing question — is the round trip the single-field fetch (`tick show <id> --field description`), or the whole-task read that an agent edits in place? The lean offered was the second, with the first as a fast path, on the grounds that an agent deciding *whether* to edit usually needs the surrounding context and a second call is the same token tax in a different currency.

### Decision

**Both reading paths are in scope, and both must work.** The user ruled both valid: an agent must be able to fetch one field bare, and must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback — the field flag does not excuse an ambiguous block in full output, and the block being fixed does not remove the need for bare single-field output.

This constrains the encoding subtopics directly: whatever the description block becomes, it must hand over text the reader can lift without a rule learned elsewhere, *and* a bare-field path must exist alongside it.

---

## Summary

### Key Insights

1. The toon output is not lossy for free text — it is undeclared. Both the description indent and the notes escaping are reversible; what is absent is anything in the output telling the reader which rule applies to which field.
2. The two free-text fields reached their current encodings by different routes and do not agree. Descriptions follow the v1 principle "long text fields get their own unstructured sections"; notes gained a timestamp and so became a tabular section, inheriting the TOON library's string quoting instead.

### Open Threads

- Whether changing the description encoding owes a correction to the v1 `tick-core` specification, which states the unstructured-section principle as a golden rule.

### Current State

- Resolved: both the single-field fetch and the whole-task read are in scope, and both must work.
- Uncertain: how the description block should encode multi-line text; whether notes need anything beyond what the TOON quoting already gives; the shape of the field-extraction flag; whether the write side needs an input path other than a command-line argument.
