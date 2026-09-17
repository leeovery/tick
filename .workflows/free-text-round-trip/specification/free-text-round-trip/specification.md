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

---

## Working Notes
