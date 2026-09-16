# Discovery Session 001

Date: 2026-09-16
Work unit: free-text-round-trip

## Description (as of session)

Make free-text fields (descriptions, notes) survive an agent's read-edit-write round trip in the toon output format, with a field-extraction flag for raw single-field output.

## Seed

(none)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

The trigger was a live failure in agent use. A Claude session working with Tick needed a task's description as a plain string and could not get one from `tick show`, so it bypassed the CLI entirely and read `.tick/tasks.jsonl` directly. The available workaround — `tick show --json` — works but is the wrong shape for the case: it costs extra tokens and forces the agent to parse JSON syntax back off the string it wanted. The irony the user named: the token-efficient format exists for agents, and here it is the format agents cannot use.

The mechanism was confirmed in the source during shaping. `buildDescriptionSection` at `internal/cli/toon_formatter.go:346` emits `description:` and then writes every line of the description prefixed with two spaces. A hundred-line description therefore returns as a hundred lines each carrying a two-space prefix the agent must strip before it can work with the text. The block has no terminator either, so a description line that itself looks like a TOON section header is indistinguishable from a real one — the format is already ambiguous for free text, not merely inconvenient.

Two candidate directions were put to the user. The first repairs the toon format so free text survives intact. The second adds a raw output format alongside JSON. The raw-format option was pressed on: a fourth output format is not a narrow escape hatch — it has to answer for every command in the CLI (`tick list --raw`, `tick stats --raw`) and carries an ongoing consistency burden across all of them. The alternative framing offered was field extraction: `tick show <id> --field description` printing the bare string and nothing else, no header, no indentation, no JSON quoting — one field, no format to design, no cross-command surface.

The user settled both open questions. Scope covers free text generally, not descriptions alone — notes carry the same problem, reaching output through the same formatter. The primary fix is repairing the toon format for round-trip fidelity rather than adding a parallel raw format. Field extraction stands as a companion of interest, confirmed after clarification of what the flag would print.

The requirement underneath all of it is the round trip: an agent reads the text, edits it, and writes it back via `tick update --description` (or the notes equivalent) without a transformation step in either direction.

One design tension was surfaced and deliberately left unresolved for the next phase rather than settled here. TOON is an indentation-scoped format — removing the indent from a free-text block requires some other means of marking where the block ends, and a description line resembling a section header is precisely the case a naive delimiter fails on. That tension is what makes this a discussion before specification rather than a direct route to spec.

Work type was confirmed as a feature: one goal, one deliverable shape, one surface (the toon formatter, where descriptions and notes share an indentation path), with field extraction a companion to the same goal rather than an independent concern. It is not a bugfix — the indentation is deliberate design that defeats its own purpose, not a regression. It is not an epic — the two deliverables are not independent enough to split.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

(none)
