# Phase 1: Task detail document becomes conformant TOON — 6 tasks

## free-text-round-trip-1-1

### Task 1: Task fields become top-level named fields

**Problem**: The head of every task-detail document is hand-edited rather than encoded. `buildTaskSection` (`internal/cli/toon_formatter.go:260-295`) marshals the task's own fields as a one-element array under a `task` key and then deletes the `[1]` from the header with `strings.Replace(s, "task[1]", "task", 1)`, producing `task{id,title,status,priority,type,created,updated}:` — a table header with no table beneath it. A standard TOON reader fails on that line (`line 1: invalid unquoted key "task{id,title,status,priority,type,created,updated}"`) and never reaches any section below it, which is what sent an agent to read `.tick/tasks.jsonl` directly. Five commands emit this document: `show`, `create`, `update`, `note add`, `note remove`.

**Solution**: Emit the task's own fields as top-level named fields with no wrapping key, produced entirely by `toon.MarshalString` over an ordered `toon.Object`, and delete the header surgery. Which fields appear, and in what order, is unchanged.

**Outcome**: The toon detail document opens with `id: tick-a1b2` / `title: …` lines, decodes with `toon.DecodeString`, and no code path strips a `[1]` from the detail document's header.

**Do**:
1. `internal/cli/toon_formatter.go` — add `encodeToonFields(fields ...toon.Field) string`: marshals `toon.NewObject(fields...)` through `toon.MarshalString`, returning the encoded string, or `""` when marshalling fails.
2. `internal/cli/toon_formatter.go` — rewrite `buildTaskSection` to assemble the same ordered `[]toon.Field` it assembles today (`id`, `title`, `status`, `priority`; `type` when non-empty; `parent` when non-empty; `created`, `updated`; `closed` when non-nil) and return `encodeToonFields(fields...)`. Delete the `[]toon.Object` wrapper, the `task` key, the `strings.Replace` call and the `"task:"` fallback.
3. `internal/cli/toon_formatter.go` — in `FormatTaskDetail`, drop empty strings from `sections` before `strings.Join(sections, "\n\n")`.
4. `internal/cli/toon_decode_test.go` (new file) — add `decodeToonDoc(t *testing.T, doc string) map[string]any`: calls `toon.DecodeString`, `t.Fatalf`s with the document text on error, and fails when the decoded value is not a `map[string]any`. Mark it `t.Helper()`.
5. `internal/cli/toon_formatter_test.go` — replace the `task{…}` header and row assertions (lines 88, 143, 369, 440, 470, 602) with decoded-value assertions built on `decodeToonDoc`: assert the values of the keys the task carries and the absence of the keys it does not. Decoded numbers arrive as `float64`, so `priority` compares against `float64(1)`.

**Acceptance Criteria**:
- [ ] `FormatTaskDetail` output decodes without error via `toon.DecodeString` for a task carrying every optional field and for one carrying none
- [ ] The decoded document carries `id`, `title`, `status`, `priority`, `created` and `updated` as top-level keys, and carries no `task` key
- [ ] `type`, `parent` and `closed` appear in the decoded document exactly when the task carries them — identical presence rules to before the change
- [ ] Emitted field order is `id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`
- [ ] No `strings.Replace` call remains in `buildTaskSection`, and the head section is produced by a single `toon.MarshalString` call
- [ ] A title containing a comma, a colon and a leading dash decodes back byte-identical to the stored title
- [ ] `created` and `updated` decode as strings, not numbers
- [ ] `encodeToonFields` returns `""` for a value the encoder rejects, and `FormatTaskDetail` then emits the remaining sections with no leading blank line
- [ ] Pretty output and JSON output for the same `TaskDetail` are unchanged
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits the task's own fields as top-level named fields"` — decode `FormatTaskDetail` output; assert `id`, `title`, `status`, `priority`, `created`, `updated` values and that no `task` key exists
- `"it omits type, parent and closed when the task does not carry them"` — decoded map has none of those three keys
- `"it includes type, parent and closed when the task carries them"` — all three present with the expected values
- `"it round-trips a title containing a comma, a colon and a leading dash"` — decoded `title` equals the stored title byte-for-byte
- `"it emits created and updated as quoted timestamps"` — decoded values are `string`, equal to `task.FormatTimestamp` of the stored times
- `"it decodes the whole document when sections are joined by blank lines"` — full detail with blocked_by, children and notes present decodes to a single map carrying every section key
- `"it returns an empty string when the encoder rejects a field value"` — `encodeToonFields(toon.Field{Key: "x", Value: make(chan int)})` returns `""`
- `"it omits the head rather than emitting a blank line when the head cannot be encoded"` — a document whose head section came back empty starts with its first real section

**Edge Cases**:
- Task carrying none of `type`, `parent`, `closed` — presence rules must be identical to today's dynamic schema, not "always emit"
- `priority` is `0` for P0 tasks and must still be emitted; do not reach for `omitempty` struct tags, which would drop it
- Titles containing a comma, a colon, a leading dash or leading/trailing content that TOON quotes — quoting is the library's job, never the formatter's
- Timestamps must stay quoted strings so a decoder does not coerce them
- Marshal-error fallback: the encoder cannot fail for strings and ints, but the path exists; the fallback must leave the rest of the document decodable
- Sections are joined with a blank line between them; the decoder accepts that and it must stay verified, since the head is now scalar fields rather than a section header

**Context**:
> §5.2: "The task's own fields sit at the top level of the document, as named fields, with no wrapping key. They are peers of the collection sections rather than nested inside a `task:` scope." The worked example is:
> ```
> id: tick-a1b2
> title: Add retry to the sync worker
> status: in_progress
> priority: 2
> type: feature
> created: "2026-09-10T09:14:00Z"
> updated: "2026-09-17T16:00:00Z"
> ```
> §5.2 also fixes what this task must *not* change: "Which sections a document carries is unchanged by this work… `children`, `blocked_by` and `notes` are always present, carrying a count-zero header when empty, while `type`, `parent`, `closed`, `tags`, `refs` and `description` appear only when the task carries them."
>
> §5.3 rejected the one-row table form (`task[1]{id,title,…}:` with a single row) in favour of named fields: "Named fields put every value beside its name, so adding or reordering a field cannot break a positional read and a value containing a comma is safe without relying on quoting discipline."
>
> §5.4 rejected a wrapping key: `description` is a task field at the top level while `title` sat nested, the JSONL file already holds each task as a flat object, and §9's field flag falls out of the flat form.
>
> Verified against `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c`: a flat `toon.NewObject` of the head fields encodes one `key: value` line per field and the whole blank-line-joined document decodes back to a single `map[string]any`. Decoded numbers come back as `float64`.
>
> The specification does not say what a formatter should emit when `toon.MarshalString` fails. Returning `""` and skipping empty sections is chosen because it keeps every other section of the document decodable; `encodeToonSection`'s existing `name[0]:` fallback is left as it is.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §5.1, §5.2, §5.3, §5.4, §3.1

## free-text-round-trip-1-2

### Task 2: Tags and refs use the library's inline list form

**Problem**: `buildStringListSection` (`internal/cli/toon_formatter.go:320-328`) writes a count header and then one raw item per indented line:

```
tags[2]:
  has space
  plain
```

A TOON reader rejects this — an item on its own line carries a leading `- ` marker, and without it the decoder reports a length mismatch. The items are written raw as well, so a ref containing a comma comes back as two values and a URL's colon goes unquoted where the format's own rules would quote it. Two of the six sections in the detail document are therefore unreadable and lossy.

**Solution**: Encode both lists with the library's inline list form by handing the `[]string` to the existing `encodeToonSection` helper, which produces `tags[2]: has space,plain` with quoting decided by the encoder. Delete the hand-written builder rather than correcting its output.

**Outcome**: `tags` and `refs` decode as lists of strings whose items are byte-identical to the stored tags and refs, including items containing commas, spaces and colons; `buildStringListSection` no longer exists.

**Do**:
1. `internal/cli/toon_formatter.go` — delete `buildStringListSection`, `buildTagsSection` and `buildRefsSection`.
2. `internal/cli/toon_formatter.go` — in `FormatTaskDetail`, call `encodeToonSection("tags", detail.Tags)` and `encodeToonSection("refs", detail.Refs)` inside the existing `len(...) > 0` guards, so both sections stay omitted entirely when the task carries none.
3. `internal/cli/toon_formatter.go` — `encodeToonSection`'s doc comment says "an array of structs"; update it to cover scalar lists too.
4. `internal/cli/toon_formatter_test.go` — replace the `strings.Contains(result, "tags[2]:")` / `"  backend"` style assertions (lines ~480-560) with `decodeToonDoc` assertions on the decoded `tags` and `refs` slices; keep the two "omits the section when empty" subtests, asserting the decoded document has no `tags` / `refs` key.

**Acceptance Criteria**:
- [ ] `tags` and `refs` are emitted as the library's inline form — `tags[2]: has space,plain` — with no indented item lines
- [ ] The decoded `tags` and `refs` slices are element-wise byte-identical to `detail.Tags` and `detail.Refs`, in the same order
- [ ] A ref containing a comma decodes back as one element, not two
- [ ] A ref containing a URL colon and a tag containing a space decode back unchanged
- [ ] A single-item list decodes to a one-element slice
- [ ] A task with no tags emits no `tags` key at all, and likewise for refs — unchanged from today
- [ ] `grep -n 'buildStringListSection\|buildTagsSection\|buildRefsSection' internal/cli/` returns nothing
- [ ] Pretty output for tags and refs is unchanged
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits tags as an inline list that decodes to the stored tags"` — `[]string{"backend", "ui"}` decodes back to the same two elements
- `"it keeps a ref containing a comma as one element"` — `"https://x.dev/a?b=1,2"` decodes to a single element equal to the stored ref
- `"it keeps a tag containing a space as one element"` — `"has space"` decodes unchanged
- `"it quotes a ref containing a URL colon"` — the ref decodes back byte-identical
- `"it emits a single-item refs list"` — one-element list decodes to one element
- `"it omits the tags section when the task has no tags"` — decoded document has no `tags` key
- `"it omits the refs section when the task has no refs"` — decoded document has no `refs` key
- `"it decodes the whole document when both tags and refs are present"` — full detail decodes and both keys carry their values

**Edge Cases**:
- Ref containing a comma — the defect that proves the hand-built form lossy; quoting must come from the encoder
- Tag containing a space, ref containing a colon (URL) — encoder decides quoting, the formatter never pre-quotes
- Single-item list — `tags[1]: plain`, still an inline list, not a bare scalar
- Empty tags / refs — the section is omitted entirely, not emitted as `tags[0]:`; this is the presence rule §5.2 preserves and the count-zero form belongs to `blocked_by`, `children` and `notes` only

**Context**:
> §6.1: "Both become the inline form, produced by the library: `tags[2]: has space,plain`. The deciding factor is that this deletes the hand-written builder rather than correcting its output, and quoting becomes the library's responsibility — so a comma-bearing ref is safe by construction. The alternative, one item per line each marked `- `, is not a form the encoder emits: it would have to be hand-built, preserving the exact class of defect this work removes."
>
> §6.1 accepts the cost: "a long refs list becomes a long line. The reader of toon output is an agent; a human reading tags reads pretty output, which renders them separately and is unchanged (§4.1)."
>
> Verified against the project's toon-go version: `encodeToonSection("tags", []string{"has space", "plain", "a,comma"})` yields `tags[3]: has space,plain,"a,comma"`, and `[]string{"https://x.dev/a?b=1,2", "PR #3"}` yields `refs[2]: "https://x.dev/a?b=1,2",PR #3`. Both decode back to the original elements. `encodeToonSection` is generic over the element type, so `[]string` needs no new helper.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §6.1, §5.2, §4.1

## free-text-round-trip-1-3

### Task 3: Description becomes one TOON-quoted value

**Problem**: `buildDescriptionSection` (`internal/cli/toon_formatter.go:346-355`) writes `description:` and then prefixes two spaces to every line of the stored text. TOON has no block form, so the section is unreadable however it is delimited; it also carries no count and no terminator and works today only because it is emitted last and runs to EOF. The result is that the two free-text fields in one document use two different, mutually incompatible decoding rules — notes are library-quoted, the description is an indented raw block — and the output announces neither. That is the defect that sent an agent to read the description out of `.tick/tasks.jsonl` instead.

**Solution**: Emit the description as a single TOON-quoted string produced by the library, exactly as `toon.MarshalString` renders a Go string — `description: "Fix it.\n\nSteps."` — using the `encodeToonFields` helper from Task 1. Its position in the document (last) and its omission when empty are unchanged.

**Outcome**: `description` decodes to a string byte-identical to the stored description for multi-line text carrying blank lines, indented lines, header-shaped lines, a leading dash, quotes, tabs and carriage returns, and all free text in the document obeys one decoding rule.

**Do**:
1. `internal/cli/toon_formatter.go` — delete `buildDescriptionSection`.
2. `internal/cli/toon_formatter.go` — in `FormatTaskDetail`, inside the existing `detail.Task.Description != ""` guard, append `encodeToonFields(toon.Field{Key: "description", Value: detail.Task.Description})`.
3. `internal/cli/toon_formatter_test.go` — rewrite the description assertions (the `"it formats show with all sections"` description block at lines ~109-120, `"it renders multiline description as indented lines"` at ~203, and `"it omits description section when empty"` at ~177) as decoded-value assertions: `decodeToonDoc(...)["description"]` equals the stored string, and the key is absent when the description is empty.
4. `internal/cli/toon_formatter_test.go` — add a byte-identity subtest driven by an awkward description fixture: a string containing `\n\n`, a line with leading spaces, a line ending `:`, a line beginning `- `, an embedded `"`, a tab and a `\r`.

**Acceptance Criteria**:
- [ ] The description is emitted as one `description: "…"` line produced by `toon.MarshalString`, with no indented block and no hand-written escaping
- [ ] The decoded `description` is byte-identical (`==` on the Go string) to `detail.Task.Description` for the awkward fixture
- [ ] Blank lines, interior leading spaces, a `Steps:` header-shaped line and a leading `- ` bullet all survive the round trip
- [ ] Embedded double quotes, tabs and carriage returns survive the round trip
- [ ] A task with an empty description emits no `description` key at all — unchanged from today
- [ ] The description stays the last section of the document
- [ ] The whole document decodes with the description present
- [ ] Pretty's indented description block is unchanged
- [ ] `grep -n 'buildDescriptionSection' internal/cli/` returns nothing
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it emits the description as one quoted value"` — decoded `description` equals the stored multi-line string
- `"it round-trips blank lines inside the description"` — text containing `\n\n` decodes byte-identical
- `"it round-trips interior lines with leading spaces"` — a line indented by two spaces decodes with its indentation intact
- `"it round-trips a header-shaped line"` — a line reading `Steps:` decodes unchanged and does not become a key
- `"it round-trips a line beginning with a dash"` — `- read the header` decodes unchanged
- `"it round-trips embedded quotes, tabs and carriage returns"` — `"`, `\t` and `\r` survive
- `"it omits the description section when the description is empty"` — decoded document has no `description` key
- `"it decodes the whole document with a description present"` — every section key decodes alongside it

**Edge Cases**:
- Blank lines inside the text, which the old block form rendered as two bare spaces
- Interior lines with leading spaces, which the old block form rendered at four
- A header-shaped line (`Steps:`) — inside a quoted value it is data, never a key
- A leading dash on a line, and a leading dash on the whole value
- Embedded `"`, tab and CR — escaping belongs to the encoder
- Empty description — the section is omitted, so there is no `description: ""` case in full output
- Edge whitespace on the whole value cannot occur: `create` and `update` run `TrimDescription` before storing; the fixture therefore carries its trailing spaces mid-text, not at the end

**Context**:
> §6.2: "It becomes a single TOON-quoted string, exactly as the library's `toon.MarshalString` produces from a Go string." Three reasons, in the order they carried: it is produced by the library rather than hand-assembled (a dash-list of lines "would have to be built by hand with our own quoting rules, which is the mechanism behind every malformed section this work removes"); the dash-list's advantage evaporates on real content (measured on a 13-line, 1090-character description, the quoted value is 1117 characters on one line, the dash-list 1185 characters over 14 lines with 13 of 13 lines needing quotes); and "it is the rule note text already obeys. One decoding rule for all free text in the output, which is what the reader needed and never had."
>
> §6.2 accepts the cost: "a long description is one long line and unpleasant to read in a terminal. A human reading a description reads pretty output, which this work does not touch." A third candidate — keeping the indented block with a terminator or line count — "cannot be conformant however it is delimited: it is a block form TOON has no concept of."
>
> §2.3 records that the current form is not lossy, only undecodable and unannounced: "the description block has no count and no terminator — it works today only because it is emitted last and runs to EOF."
>
> §2.2 sets the bar this task is measured against: "Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was."
>
> Verified against the project's toon-go version: a string carrying `\n\n`, two-space indentation, `"`, `\t` and `\r\n` encodes to one `description: "…"` line and decodes back byte-identical.
>
> §12.2 records that the `v1` / `tick-core` specification is owed a correction on exactly this point: it states "long text fields get their own unstructured sections" as a principle and prints the indented description block as its worked example (`sed -n '693,714p' .workflows/v1/specification/tick-core/specification.md`) — "`tick-core` states as a rule the exact thing §6.2 removes", which is plainly and load-bearingly wrong and is therefore amended rather than left to supersession. That is a completed work unit's artifact: it is corrected through the session's corrigendum route — the amendment presented and confirmed, the wrong claim replaced in place, a dated corrigendum entry recording what the document used to claim, and the document re-indexed — never by this task.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §6.2, §2.2, §2.3, §5.2, §12.2

## free-text-round-trip-1-4

### Task 4: Notes carry a 1-based index column

**Problem**: The notes table carries `{text,created}` and nothing that identifies a note. Position is the only handle a note has — there is no note ID, `note remove` takes a 1-based index (`internal/cli/note.go:127-131`), and notes stay an append-and-retract log with no edit command. So an agent that reads a filtered notes section sees `notes[1]` with one row, decides the note is wrong, runs `tick note remove <id> 1` and deletes the first note on the task. Even in full output it must count rows to work out what to pass to `note remove`.

**Solution**: Add a leading `index` column carrying each note's 1-based position to the toon notes table, and the same `index` field to each note object in JSON. The column is present in one shape always — full output and, once field selection lands, a filtered section — so there is no rule about when it appears.

**Outcome**: `notes[2]{index,text,created}:` rows decode with an `index` that is the number `tick note remove` accepts for that note, in toon and in JSON, and the count-zero header carries the column too.

**Do**:
1. `internal/cli/toon_formatter.go` — add `Index int` with tag `toon:"index"` as the first field of `toonNoteRow`.
2. `internal/cli/toon_formatter.go` — in `buildNotesSection`, set `Index: i + 1` on each row and change the empty-case literal to `notes[0]{index,text,created}:`.
3. `internal/cli/json_formatter.go` — add `Index int` with tag `json:"index"` as the first field of `jsonNote`, and set it to `i + 1` in `FormatTaskDetail`'s note loop.
4. `internal/cli/show.go:146` — order the notes query `ORDER BY created ASC, rowid ASC` so notes stored within the same second keep the order `note remove` indexes; notes are inserted into `task_notes` in JSONL order during cache rebuild (`internal/storage/cache.go:161`).
5. `internal/cli/toon_formatter_test.go` and `internal/cli/json_formatter_test.go` — update the notes subtests (`toon_formatter_test.go:608-665`, `json_formatter_test.go:940-995`) to assert the decoded `index` values alongside text and created, and to assert the count-zero toon header includes the column.

**Acceptance Criteria**:
- [ ] The toon notes header is `notes[N]{index,text,created}:` and each decoded row's `index` is its 1-based position, in order
- [ ] The empty toon notes section is `notes[0]{index,text,created}:` and decodes to an empty list
- [ ] Each JSON note object carries `index` with the same 1-based value
- [ ] `tick note remove <id> <index>` applied to the index shown for a note removes that note and no other, including on a task whose notes share a created timestamp
- [ ] Multi-line note text stays library-quoted and decodes byte-identical
- [ ] Pretty's notes block is unchanged — no index is rendered there
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it numbers notes from 1 in the toon notes table"` — two notes decode with `index` 1 and 2 in order
- `"it carries the index column on the empty notes section"` — decoded document has an empty `notes` list and the emitted header is `notes[0]{index,text,created}:`
- `"it numbers notes from 1 in json output"` — decoded JSON note objects carry `index` 1 and 2
- `"it keeps multi-line note text quoted"` — a note containing `\n` decodes back byte-identical with its index intact
- `"it keeps a note beginning with a dash intact"` — the note text decodes unchanged
- `"it matches the index note remove accepts"` — end-to-end: add three notes, read the index of the second from `tick show` output, `tick note remove <id> <that index>`, assert the remaining notes are the first and third
- `"it keeps insertion order for notes sharing a created timestamp"` — two notes added within the same second decode with indexes matching their JSONL order

**Edge Cases**:
- Count-zero header carries the column, so the schema is the same whether or not the task has notes
- Multi-line note text — already library-quoted via the tabular encoder; adding a column must not disturb that
- Note text beginning with a dash, which §10 makes writable later in the plan but which is already readable
- Notes sharing a `created` timestamp (both added inside one second): `show` orders by `created ASC`, so ties must be broken by insertion order or the index would not be the one `note remove` takes
- `index` decodes as `float64` through the generic decoder, like every other TOON number

**Context**:
> §6.3: "The notes section gains a leading `index` column carrying each note's 1-based position, present whether the section is filtered (§9.3) or not":
> ```
> notes[2]{index,text,created}:
>   1,Retried twice before it stuck,"2026-09-14T10:02:00Z"
>   2,"multi\nline\nnote","2026-09-16T08:30:00Z"
> ```
> "Position is the only handle a note has. There is no note ID; `note remove` takes a 1-based index; and notes stay an append-and-retract log (§6.4), so nothing else identifies one. Without the column, an agent that asked for the third note alone sees `notes[1]` with one row, decides the note is wrong, runs `tick note remove <id> 1`, and deletes the first note on the task." And: "One shape either way, so there is no rule about when the column appears."
>
> §4.2: "Each note also carries its 1-based index, for the reason the toon table does (§6.3): a consumer that asked for one note (§9.3) needs the note's real position before it can call `note remove`, and that need is the same whichever format it parses."
>
> §6.4 fixes what is out of scope here: "No `note edit` command is added by this work."
>
> §4.1 keeps pretty untouched, so the index is a machine-format column only.
>
> The specification does not discuss how notes are ordered on the read path. The query ordering change in step 4 is the minimum that makes the new column truthful — an index the caller cannot pass to `note remove` is worse than no index — and it is invisible in every case where timestamps differ.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §6.3, §6.4, §4.2, §4.1

## free-text-round-trip-1-5

### Task 5: Detail-document assertions check decoded values

**Problem**: The suite could not catch the defect this phase fixes. Its assertions compare output against a string written down alongside the code, so a malformed header passed for the tool's entire life — the test compared a wrong string to the same wrong string. Tasks 1 to 4 rewrote the assertions they each broke, but shape-pinned checks on the detail document remain: `format_integration_test.go` asserts `strings.Contains(stdout, "task{")` at lines 28, 304, 620 and 663, which is the malformed header used as a format-detection probe, and nothing anywhere decodes a whole `tick show` document end to end.

**Solution**: Replace the remaining shape-pinned detail-document assertions with decoded-value checks, and add a conformance test that runs `tick show` (and the two note commands, which emit the same document and reach their final shape in this phase) against a real store and decodes stdout with the project's TOON library.

**Outcome**: Every assertion on the task-detail document in toon and JSON checks a decoded value or key presence; the document is decoded end to end for a task carrying every optional field and for one carrying none; pretty's golden strings are untouched.

**Do**:
1. `internal/cli/format_integration_test.go` — replace the four `strings.Contains(stdout, "task{")` checks with `decodeToonDoc(t, stdout)` followed by assertions that `id` equals the expected task ID and that the always-present `notes` key exists (pretty carries neither, so the check still discriminates the formats).
2. `internal/cli/toon_decode_test.go` — add `TestToonTaskDetailConformance` with two subtests driven through `App.Run([]string{"tick", "--toon", "show", id})` on a project seeded by `setupTickProjectWithTasks`: one task carrying type, parent, closed, tags, refs, description, notes, children and blocked_by, and one carrying none of them.
3. `internal/cli/toon_decode_test.go` — assert per decoded document: every expected key present with its stored value, every unexpected key absent, `blocked_by`/`children`/`notes` present in both cases, and `notes` rows carrying `index`, `text`, `created`.
4. `internal/cli/toon_decode_test.go` — add subtests decoding `tick --toon note add <id> -- <text>` and `tick --toon note remove <id> 1` stdout the same way, since both emit this document through `outputMutationResult`.
5. Sweep and confirm: `grep -rn 'task{\|tags\[2\]:\|notes\[0\]{text\|"description:"' internal/cli/*_test.go` returns nothing, no toon or JSON test asserts a pinned full-document string for the detail document, and `internal/cli/pretty_formatter_test.go` and the pretty branches of `format_integration_test.go` are unmodified by this phase.

**Acceptance Criteria**:
- [ ] `tick show --toon` output decodes via the TOON library for a task carrying every optional field and for one carrying none
- [ ] Both decoded documents carry `blocked_by`, `children` and `notes`; the all-fields one additionally carries `type`, `parent`, `closed`, `tags`, `refs` and `description`, and the bare one carries none of those six
- [ ] `note add` and `note remove` toon output decodes the same way
- [ ] No test asserts `task{` anywhere in the repository
- [ ] No toon or JSON assertion on the detail document compares against a pinned full-output string; each checks a decoded value or key presence
- [ ] `internal/cli/pretty_formatter_test.go` carries the same golden-string assertions as before this phase, none removed or weakened
- [ ] The JSON detail tests still assert through `json.Unmarshal` into `map[string]any`, as they already do
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

**Tests**:
- `"it decodes a show document carrying every optional field"` — all nine optional keys present with their stored values
- `"it decodes a show document carrying no optional fields"` — `type`, `parent`, `closed`, `tags`, `refs`, `description` all absent; `blocked_by`, `children`, `notes` present and empty
- `"it decodes note add output as the task detail document"` — decoded `notes` carries the added note at its index
- `"it decodes note remove output as the task detail document"` — decoded `notes` is one shorter and re-indexed from 1
- `"it decodes show output when stdout is not a TTY"` — replaces the non-TTY `task{` probe
- `"it decodes show output under --toon on a TTY"` — replaces the `--toon` override probe
- `"it keeps pretty show output on a TTY"` — unchanged `ID:` assertion, kept as the pretty-side probe

**Edge Cases**:
- Pretty's golden-string assertions stay: pretty output has no parser, so a decoded-value assertion does not exist for it and removing its golden strings would replace its only form of assertion with nothing
- The two format-detection probes must still tell toon from pretty after the header changes — key presence does that, a first-line string match would re-introduce a pinned shape
- `create` and `update` also emit this document but gain a `changed` section in Phase 2, so their end-to-end decode belongs there; `note add` and `note remove` never carry that section and reach their final shape here
- Sections whose shape did not change (`blocked_by`, `children`) still need decoded assertions, since the document around them changed and the old assertions read them positionally out of a `strings.Split(result, "\n\n")`
- There are no JSON full-document golden blobs in `create_test.go`, `update_test.go` or `note_test.go` to convert — those files assert persisted state, not rendered output; confirm rather than assume

**Context**:
> §11: "Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse. This alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content." And: "Rewritten assertions check decoded values, not output text. 'The notes section has two rows and the second row's text is X', not 'the output equals this blob'."
>
> §11 on why the old assertions could not catch it: "Its assertions compare output against a string written down alongside the code, so a malformed header passed for the tool's entire life: the test compared a wrong string to the same wrong string."
>
> §11 on golden strings: "No byte-level pinning is kept in the machine formats. Golden strings pin the exact output shape… but they are the mechanism that rotted into the defect this work undoes. Decoded-value assertions survive harmless reformatting while still failing when a section goes missing or a value is wrong. That trade is taken for toon and JSON. Pretty keeps golden-string assertions."
>
> §3.1 lists the five commands emitting this document and notes that `show` renders it inline (`internal/cli/show.go:52-64`) while `create`, `update`, `note add` and `note remove` share `outputMutationResult`.
>
> Phase 6 extends decode coverage to the whole must-parse inventory and every branch; this task covers the task-detail document only.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §3.1, §4.1

## free-text-round-trip-1-6

### Task 6: README samples match real output

**Problem**: The README prints the agent-facing output this phase replaces. Its `tick show` sample (`README.md:429-450`) shows the `task{…}:` header, the indented tags and refs lists and the indented description block — none of which the tool will produce — and it prints the sections in an order the formatter never emits (tags and refs before blocked_by and children). Its `tick list` sample (`README.md:396-401`) renders an empty `type` as a bare trailing comma where the formatter emits a quoted empty string. The README is live documentation someone reads to learn the tool, so leaving it describing output the tool does not produce is shipping a defect.

**Solution**: Replace both samples with the tool's actual output, and add a test that decodes the README's toon samples so a future reshaping cannot silently rot them again.

**Outcome**: The `tick show` and `tick list` samples in the README are byte-for-byte what the tool prints, and a test fails if either stops parsing as TOON.

**Do**:
1. Build the binary to a scratch path and, in a throwaway `.tick` project, produce real `--toon` output for a task carrying type, tags, refs, one note and a multi-line description plus one blocker and no children; copy it into `README.md:429-450` verbatim. The target shape is in Context and the captured output must match it field for field — the sample task has no parent, so the document carries no `parent` line.
2. `README.md:396-401` — correct the empty-type row of the `tick list` sample to `tick-d5c6,Update docs,open,3,""`.
3. Leave the dep-tree sample (`README.md:302-307`) and the transition and cascade samples (`README.md:473-504`) alone — they are corrected by the phases that change them.
4. `internal/cli/readme_samples_test.go` (new file) — add `TestREADMEToonSamplesDecode`: locate `README.md` via `testutil.FindRepoRoot(t)`, collect every fenced block with an empty info string, strip a leading `$ tick …` prompt line from each, index the blocks by their first remaining line, then for each anchor in `{"tasks[3]{id,title,status,priority,type}:", "tasks[2]{id,title,status,priority,type}:", "id: tick-a1b2"}` fail if no block carries it and fail if `toon.DecodeString` rejects that block.

**Acceptance Criteria**:
- [ ] The `tick show` sample was copied from real tool output, not hand-written, and its section order is head fields, `blocked_by`, `children`, `tags`, `refs`, `notes`, `description`
- [ ] The sample's head is the top-level named fields `id`, `title`, `status`, `priority`, `type`, `created`, `updated` — no `task` key, no `[1]` and no `parent` line, matching the target shape in Context
- [ ] `tags` and `refs` in the sample are inline lists, and the refs URL is quoted as the encoder quotes it
- [ ] The sample's `description` is one quoted value with `\n` escapes, not an indented block
- [ ] The sample's notes table header is `notes[1]{index,text,created}:` and its row starts with `1`
- [ ] The `tick list` sample's empty-type row ends `,""`
- [ ] `TestREADMEToonSamplesDecode` passes, and fails if any of the three anchored blocks is removed or made unparseable
- [ ] The dep-tree, transition and cascade samples are unchanged by this task
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it decodes the README show sample"` — the block anchored on `id: tick-a1b2` parses as TOON
- `"it decodes the README list samples"` — both `tasks[N]{…}` blocks parse as TOON
- `"it fails when an anchored sample is missing"` — an anchor with no matching block is a test failure, not a silent pass
- `"it renders an empty type as a quoted empty string"` — the list sample's third row ends `,""`, matching what `FormatTaskList` emits for an empty `Type`

**Edge Cases**:
- The list sample's empty `type` renders as `""`: `toonTaskRow.Type` carries no `omitempty` and the library quotes an empty string — the suite's own golden row already shows `tick-a1b2,Setup Sanctum,done,1,""`
- Section order in the show sample must match the formatter's real order, which is not the order the README currently prints
- The anchor-based test must ignore ```json and ```bash blocks and the pretty-output blocks, and must strip the `$ tick …` prompt line that precedes some samples
- The dep-tree sample still carries the malformed `summary{chains,longest,blocked}:` header until Phase 3, so it must not be one of this task's anchors

**Context**:
> §12.1: the README "is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect." The samples this phase owns are the `tick show` full detail (`README.md:430-450` — "header, tags, refs and description block all change") and the `tick list` table (`README.md:396-401`), whose "printed row is nonetheless wrong and is corrected while the section is being rewritten: it renders an empty `type` as a bare trailing comma, where the formatter quotes it… `toonTaskRow.Type` carries no `omitempty` and the library quotes an empty string." The dep-tree summary header, the arrow transition, the JSON transition and the cascade samples are listed in the same table but change with §5.2's other sites and §7.2's table, which land in later phases.
>
> §12.1 also records README additions this work owes that are not this phase's: `--field`/`--fields` on `show` (§9) and `--` for dash-leading free text (§10.2).
>
> Target shape for the show sample, encoded with the project's toon-go version and verified to decode:
> ```
> id: tick-a1b2
> title: Setup auth
> status: in_progress
> priority: 1
> type: feature
> created: "2026-01-19T10:00:00Z"
> updated: "2026-01-19T14:30:00Z"
>
> blocked_by[1]{id,title,status}:
>   tick-c3d4,Database migrations,done
>
> children[0]{id,title,status}:
>
> tags[2]: backend,auth
>
> refs[1]: "https://github.com/org/repo/issues/42"
>
> notes[1]{index,text,created}:
>   1,Discussed approach with team,"2026-01-19T14:00:00Z"
>
> description: "Full task description here.\nCan be multiple lines."
> ```
>
> `testutil.FindRepoRoot(t)` returns the directory containing `go.mod` and is already used by `cmd/tick/build_test.go` and the `scripts/` tests.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §12.1, §5.2, §6.1, §6.2, §6.3
