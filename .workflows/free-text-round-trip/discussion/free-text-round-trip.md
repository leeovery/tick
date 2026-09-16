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

## Summary

### Open Threads

- Nothing decided yet.

### Current State

- The toon description block is a reversible but undeclared two-space indent with no terminator; notes are lossless TOON-quoted strings. The two free-text fields do not share a scheme.
- The write path trims surrounding whitespace, so an exact read does not guarantee an exact write-back.
