# Plan: Free Text Round Trip

## Phases

### Phase 1: Task detail document becomes conformant TOON
status: draft

**Goal**: The document that `show`, `create`, `update`, `note add` and `note remove` all emit decodes with a standard TOON reader — the task's own fields at the top level with no wrapping key, tags and refs as library-produced inline lists, the description as one quoted value, and notes carrying a 1-based index column.

**Why this order**: It is the document five of the eight listed commands produce, and its header is the line a reader fails on before it sees anything else. Every later phase consumes it: the `changed` table becomes a section inside it, field selection projects from it, the round-trip fixture reads through it. It also deletes the mechanism behind every malformed section — hand-assembled string building and header surgery — establishing the pattern the remaining phases follow.

**Acceptance**:
- [ ] `tick show` output decodes with the project's TOON library for a task carrying every optional field (type, parent, closed, tags, refs, description, notes, children, blocked_by) and for one carrying none of them
- [ ] The task's own fields decode as top-level keys with no wrapping key, and no code strips a `[1]` from the detail document's header
- [ ] `tags` and `refs` decode as library-produced inline lists; a ref containing a comma decodes back as one value
- [ ] `description` decodes as a single string byte-identical to the stored value, for multi-line text carrying leading spaces, a colon and a leading dash
- [ ] The notes section carries a leading 1-based `index` column, present whether the section is filtered or not; JSON note objects carry the same index
- [ ] Which sections a document carries — always-present versus carried-only — is unchanged from today
- [ ] Pretty output for the same tasks is byte-identical to before the change
- [ ] README's `tick show` sample matches real output, and the `tick list` sample's empty-type row shows the quoted empty string the formatter emits
- [ ] Toon and JSON assertions for this document check decoded values rather than output text

### Phase 2: Status changes become one `changed` table
status: draft

**Goal**: Every command that moves a status returns a single `changed[N]{id,title,from,to,auto}` table, and `create`/`update` return one document carrying that table as a section rather than a document with transition lines appended after it.

**Why this order**: Phase 1 leaves `create` and `update` emitting a valid document with foreign lines stuck on the end — still unreadable as a stream — and this phase closes that, which requires Phase 1's document to nest into. It also carries the riskier of the two shared-code splits: `baseFormatter.FormatTransition` is embedded by both the toon and pretty formatters, so restructuring the toon form without moving pretty means splitting it deliberately.

**Acceptance**:
- [ ] `start`, `done`, `cancel` and `reopen` emit only the `changed` table, and it decodes
- [ ] A task whose status moved twice in one command carries exactly one row, reading from its pre-command status to its post-command status; a task that ends where it started carries no row
- [ ] `create` and `update` emit a single decodable document with `changed` as a section inside it, present with count 0 when no task's status moved
- [ ] `update`'s two independent cascade blocks collapse into the single table
- [ ] `show`, `note add` and `note remove` carry no `changed` section
- [ ] JSON emits the same `changed` list in place of the current `transition` object beside `cascaded`
- [ ] Pretty's single transition line and box-drawing cascade tree are byte-identical to before the change
- [ ] README's arrow transition, JSON transition and cascade samples match real output

### Phase 3: Stats and dependency-tree documents
status: draft

**Goal**: The remaining two hand-built single-object headers become top-level named fields, a focused dep tree identifies its task exactly as a task-detail document does on both branches, and both empty branches return the populated document emptied instead of an English sentence.

**Why this order**: These are the last two malformed headers and the last prose answer to a query. They share no code with Phases 1 and 2, so the phase is self-contained; it follows them because it reuses the named-field form Phase 1 establishes, and because its risk profile differs — the empty-branch fix lands in the dep-tree handler rather than a formatter, and pretty prints nothing at all on that branch unless handed its sentence explicitly.

**Acceptance**:
- [ ] `tick stats` counts decode as top-level named fields beside the `by_priority` table
- [ ] `tick dep tree` chains/longest/blocked decode as top-level named fields beside the edges section
- [ ] `tick dep tree <id>` carries top-level `id`, `title` and `status` on both the populated and the empty branch
- [ ] Nothing blocked anywhere, and a named task with no dependencies either way, both emit the populated document emptied — summary fields reading zero, edge sections carrying count-zero headers — and both decode, in toon and JSON
- [ ] Pretty still prints `No dependencies found.` and `No dependencies.` on those branches
- [ ] No code path anywhere in the formatter strips a count from a section header
- [ ] README's dep-tree summary sample matches real output

### Phase 4: Field selection on `show`
status: draft

**Goal**: `tick show --field`/`--fields` returns one field's value bare with nothing around it, or a document carrying only the named sections, with list sections addressable by position.

**Why this order**: It is the case that started the work, and it is a projection of the document Phase 1 defines — the names it accepts are the names that document uses, and the bare form exists only because Phase 1 removed the wrapper. It depends on nothing from Phases 2 and 3, since `show` carries no `changed` section, so it follows once every document it could project is conformant.

**Acceptance**:
- [ ] `--field` and `--fields` are registered against `show` alone with matching help text, and the flag/help drift test passes
- [ ] A selection resolving to exactly one value prints that value's bytes followed by a single newline — no header, indentation or quoting — and ignores `--json`, `--pretty` and `--toon`
- [ ] A multi-field selection, or a single field naming a list section, prints a document identical to full output minus the sections not asked for, in normal section order, honouring the format flags in all three formats
- [ ] Positional selectors work on every list section, narrow only the section they name, and leave every other selected field whole
- [ ] Nothing rides along unasked, `id` included, in either form
- [ ] An empty field prints what full output prints for it and exits zero; unrecognised names, blank names and out-of-range positions exit non-zero with nothing on stdout
- [ ] `--quiet` combined with a field selection is refused with a non-zero exit and nothing on stdout
- [ ] README documents `--field`/`--fields`

### Phase 5: Dash-leading free text and the whitespace invariant
status: draft

**Goal**: Text an agent reads out of tick can be written back — dash-leading titles and note text are accepted, and every write path trims edge whitespace so byte-identity holds across all three free-text carriers.

**Why this order**: It is the write half of the round-trip contract and the read half now exists: Phases 1 to 4 make the text readable, this makes it writable. The byte-identity fixture that proves the contract needs both halves and the dash fix specifically, since the fixture's title and note both begin with a dash.

**Acceptance**:
- [ ] `--` is accepted on every command as the end-of-flags marker; nothing after it is read as a flag, including an argument that spells a global flag exactly
- [ ] `tick create -- "- title"` and `tick note add <id> "- text"` both succeed, and dash-leading note text that is not itself a global flag works with or without the marker
- [ ] Every invocation that works today produces identical output and exit status
- [ ] `tick migrate` trims edge whitespace from imported titles and descriptions; a whitespace-only value imports as empty and no import fails because of it
- [ ] Values already in storage are untouched — no pass rewrites stored records, and `show` emits stored bytes unmodified
- [ ] A permanent fixture task carrying newlines, quotes, commas, a leading dash, trailing spaces on an interior line and a header-shaped line — across title, description and a note — round-trips write → read → decode → write-back with the stored value byte-for-byte what it was
- [ ] README and the command help text present `--` as the canonical way to pass free text that may begin with a dash

### Phase 6: Conformance verification across the output inventory
status: draft

**Goal**: Every document the tool produces is decoded by a real TOON reader in the suite, on every branch, and no machine-format assertion pins output text.

**Why this order**: The coverage is counted in documents rather than commands, so it can only be enumerated once every document exists in its final shape. It is the guard that stops the next change re-introducing a hand-built section — the failure mode that produced this work in the first place.

**Acceptance**:
- [ ] Every command in the must-parse inventory has its output decoded by the TOON library in the suite, on every branch it can take, the emptied forms included
- [ ] Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes
- [ ] JSON output for each listed command is parsed and asserted by decoded value
- [ ] No toon or JSON test asserts against a pinned full-output string
- [ ] Pretty's golden-string assertions remain in place
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean
