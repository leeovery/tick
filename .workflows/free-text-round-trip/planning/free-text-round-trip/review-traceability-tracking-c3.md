# Review Tracking: Free Text Round Trip - Traceability

## Findings

### 1. The permanent round-trip fixture carries no tags and no refs

**Type**: Incomplete coverage
**Spec Reference**: §11, part 2 — "One deliberately awkward task becomes a permanent fixture, round-tripped end to end… That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect"
**Plan Reference**: Phase 5, task `free-text-round-trip-5-5` (The awkward-task fixture round-trips byte-identically)
**Move**: settled
**Change Type**: update-task

**Problem**:
The permanent fixture is the one test standing between this work and the next change quietly re-breaking the output — the specification credits it with catching three named defects: the description block, the tags item-marker and the refs quoting. The fixture task as planned carries a title, a description and a note and nothing else. A task with no tags and no refs emits neither section, so a future change that puts tags back to unmarked indented items leaves this test green: the document it reads never contains the section that would fail to decode. The guard that ships is narrower than the one the specification describes, and the two defects it was meant to pin are pinned nowhere end to end.

**Proposal**:
Put a tag and a ref on the same fixture task and assert them on read path A alongside the three free-text carriers. The specification decides this: §11 states the single test would have caught the tags item-marker defect and the refs comma defect, and a task carrying neither section exercises neither. The old hand-built `tags[2]:` form writes unmarked indented items, which makes the whole document undecodable, so read path A's `decodeToonDoc` is the assertion that catches it — but only if the fixture carries tags at all.

The values are constrained by the tool's own validation rather than by a choice: `task.ValidateTag` requires kebab-case (`internal/task/tags.go:24-35`) and `task.ValidateRef` rejects any ref containing a comma or whitespace (`internal/task/refs.go:17-34`). A comma-bearing ref therefore cannot be stored through the CLI at all, so that half of §6.1 stays where it already is — the formatter-level assertion in Phase 1 task `free-text-round-trip-1-2` — and the fixture's ref is a colon-bearing URL, which is the other half of §6.1's defect ("a URL's colon goes unquoted where the format's own rules would quote it"). No write-back is added for tags or refs: §11 names title, description and a note as the three free-text carriers, and the byte-identity bar of §2.2 is stated over those.

**Current**:
```markdown
## free-text-round-trip-5-5

### Task 5: The awkward-task fixture round-trips byte-identically

**Problem**: Every test this work has added so far checks a shape — that a document decodes, that a section carries the right rows, that a flag parses. None checks the guarantee the work actually made, which is the round trip: read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was. §11 requires one deliberately awkward task as a permanent fixture for exactly that reason — a single test that would have caught the original description defect, the tags item-marker defect and the refs comma defect, none of which a parse check alone catches.

**Solution**: One permanent test that writes a fixture task whose title, description and note each carry text the old output mangled, reads every value back through both read paths — the decoded toon document and `tick show --field` — writes the decoded text back, and asserts the stored value byte-for-byte at each step.

**Outcome**: `internal/cli/round_trip_test.go` fails if any free-text carrier stops surviving write → read → decode → write-back, and it asserts stored values rather than output text.

**Do**:
1. `internal/cli/round_trip_test.go` (new file) — declare the three fixture constants given in Context and create the task through `App.Run` with `--description` before the marker and the dash-leading title after it, capturing the ID from `--quiet` output, then add the note with `tick note add <id> -- <text>`.
2. Assert the write half: `readPersistedTasks` shows `Title`, `Description` and `Notes[0].Text` equal to the fixture constants byte-for-byte.
3. Read path A: run `tick --toon show <id>`, decode with `decodeToonDoc` (added in Phase 1), and assert the decoded `title`, `description` and first `notes` row's `text` equal the stored values.
4. Read path B: run `tick show <id> --field title`, `--field description` and `--field notes.1`, strip exactly one trailing newline from each, and assert each equals the stored value.
5. Write the decoded values back — `tick update <id> --title <decoded title> --description <decoded description>` and `tick note add <id> -- <decoded note text>` — then assert from `readPersistedTasks` that the stored title and description are unchanged and that `Notes[1].Text` equals `Notes[0].Text`. Add a second subtest creating a task whose title and description carry edge whitespace, asserting they store trimmed and stay trimmed across a write-back.

**Acceptance Criteria**:
- [ ] One task carries all three free-text carriers: a dash-leading title containing a comma, a multi-line description, and a dash-leading note
- [ ] The description carries blank lines, a header-shaped line, an interior line ending in trailing spaces, an embedded double quote, a comma and a line beginning with a dash
- [ ] The note text begins with a dash and carries an embedded double quote and a comma
- [ ] `create` and `note add` both succeed for that content
- [ ] The stored title, description and note text are byte-identical to the fixture constants
- [ ] The decoded toon document's `title`, `description` and note `text` are byte-identical to the stored values
- [ ] `tick show <id> --field title`, `--field description` and `--field notes.1` each print the stored value followed by exactly one newline
- [ ] Writing the decoded title and description back leaves the stored values byte-identical
- [ ] Re-adding the decoded note text stores a second note whose text is byte-identical to the first's, and the task then carries two notes
- [ ] Every fidelity assertion compares a stored value or a decoded value — none compares rendered output text
- [ ] A value carrying edge whitespace stores trimmed and stays trimmed across a write-back
- [ ] The test runs under `go test ./internal/cli` with no built binary and no network
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it stores the awkward title byte-identically"` — the persisted title equals the fixture constant
- `"it stores the awkward description byte-identically"` — the persisted description equals the fixture constant
- `"it stores the awkward note byte-identically"` — the persisted note text equals the fixture constant
- `"it decodes the stored title out of the show document"` — decoded `title` equals the stored title
- `"it decodes the stored description out of the show document"` — decoded `description` equals the stored description
- `"it decodes the stored note text out of the show document"` — the first `notes` row's `text` equals the stored text, and its `index` reads 1
- `"it returns the stored title bare from --field"` — stdout equals the stored title plus one newline
- `"it returns the stored description bare from --field"` — stdout equals the stored description plus one newline
- `"it returns the stored note text bare from --field notes.1"` — stdout equals the stored note text plus one newline
- `"it writes the decoded title back unchanged"` — after `tick update --title`, the persisted title is byte-identical
- `"it writes the decoded description back unchanged"` — after `tick update --description`, the persisted description is byte-identical
- `"it writes the decoded note back as a second note with identical text"` — `Notes[1].Text == Notes[0].Text` and the task carries two notes
- `"it trims edge whitespace on the way in and keeps it trimmed"` — a title and description created with surrounding whitespace store trimmed, and a write-back of the read value stores the same bytes

**Edge Cases**:
- The title and the note text both begin with a dash, so the fixture cannot be written at all without Tasks 1 to 3
- Trailing spaces sit on an interior line, not at the end of a value: edge whitespace is trimmed on the way in and stays trimmed, so a value carrying it could never round-trip and is not what the bar covers
- The description carries blank lines, a header-shaped line, embedded quotes and commas — each the shape a hand-built section mangled before Phase 1
- The title must stay single-line, since `ValidateTitle` rejects newlines; the multi-line content lives in the description
- Note text is capped at 2000 characters, which the fixture is far under
- The note is written back by adding it again, since notes carry no edit command, and the assertion is made against the new note's stored text
- Both read paths are exercised — the decoded document and `tick show --field` — because §2.1 makes neither a fallback for the other
- The assertion reads the stored value rather than output text, so a reformatting of the document cannot make the test pass or fail spuriously
- The write-back of the title needs no marker because its value follows a registered value-taking flag; the note write-back does need one

**Context**:
> §11, part 2: "One deliberately awkward task becomes a permanent fixture, round-tripped end to end. Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. It carries that text in all three free-text carriers — title, description and a note — each taking what its shape allows: the title and the note text begin with a dash, and the description carries the multi-line content. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets. A note is written back by adding it again, since notes carry no edit (§6.4), and the assertion is made against the new note's stored text. Whitespace at the very start or end of a value is trimmed on the way in and stays trimmed, which is why the fixture carries its trailing spaces mid-text. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability."
>
> §2.2: "Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was."
>
> §2.1: "An agent must be able to fetch one field bare, **and** must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback: the field flag (§9) does not excuse an ambiguous block in full output, and repairing the block does not remove the need for bare single-field output." Both paths are therefore asserted.
>
> §9.2 fixes the bare form this test reads: "The value goes out as a line: its own bytes followed by a single newline." §9.3: "Asked for alone, `--field notes.2` prints that note's text bare, by the one-field rule."
>
> Fixture content, as Go string literals:
> ```go
> const fixtureTitle = `- read the header, carefully`
> const fixtureDescription = "Fix the parser.\n\nSteps:\n  - read the header   \n  - validate \"strictly\", then stop\nDone."
> const fixtureNoteText = `- retried "twice", then it stuck`
> ```
> The description's second bullet line ends in three spaces and is followed by two further lines, so the trailing whitespace is interior and survives; the value itself starts and ends on non-whitespace.
>
> Phase 1 added `decodeToonDoc` in `internal/cli/toon_decode_test.go`; `setupTickProject` and `readPersistedTasks` are in `internal/cli/create_test.go`; `runShow` in `internal/cli/list_show_test.go` constructs the `App` with `IsTTY: true`, so an explicit `--toon` is needed for the decoded read path.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §2.1, §2.2, §6.4, §9.2, §9.3, §10.2
```

**Proposed Text**:
```markdown
## free-text-round-trip-5-5

### Task 5: The awkward-task fixture round-trips byte-identically

**Problem**: Every test this work has added so far checks a shape — that a document decodes, that a section carries the right rows, that a flag parses. None checks the guarantee the work actually made, which is the round trip: read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was. §11 requires one deliberately awkward task as a permanent fixture for exactly that reason — a single test that would have caught the original description defect, the tags item-marker defect and the refs comma defect, none of which a parse check alone catches.

**Solution**: One permanent test that writes a fixture task whose title, description and note each carry text the old output mangled, and which also carries a tag and a ref so the sections the old builder wrote by hand are in the document too, reads every value back through both read paths — the decoded toon document and `tick show --field` — writes the decoded text back, and asserts the stored value byte-for-byte at each step.

**Outcome**: `internal/cli/round_trip_test.go` fails if any free-text carrier stops surviving write → read → decode → write-back, and if the tags or refs sections stop decoding to their stored values, and it asserts stored values rather than output text.

**Do**:
1. `internal/cli/round_trip_test.go` (new file) — declare the five fixture constants given in Context and create the task through `App.Run` with `--description`, `--tags` and `--refs` before the marker and the dash-leading title after it, capturing the ID from `--quiet` output, then add the note with `tick note add <id> -- <text>`.
2. Assert the write half: `readPersistedTasks` shows `Title`, `Description`, `Tags`, `Refs` and `Notes[0].Text` equal to the fixture constants byte-for-byte.
3. Read path A: run `tick --toon show <id>`, decode with `decodeToonDoc` (added in Phase 1), and assert the decoded `title`, `description`, `tags`, `refs` and first `notes` row's `text` equal the stored values.
4. Read path B: run `tick show <id> --field title`, `--field description` and `--field notes.1`, strip exactly one trailing newline from each, and assert each equals the stored value.
5. Write the decoded values back — `tick update <id> --title <decoded title> --description <decoded description>` and `tick note add <id> -- <decoded note text>` — then assert from `readPersistedTasks` that the stored title and description are unchanged and that `Notes[1].Text` equals `Notes[0].Text`. Add a second subtest creating a task whose title and description carry edge whitespace, asserting they store trimmed and stay trimmed across a write-back.

**Acceptance Criteria**:
- [ ] One task carries all three free-text carriers: a dash-leading title containing a comma, a multi-line description, and a dash-leading note
- [ ] The same task carries at least one kebab-case tag and one colon-bearing URL ref, so the document it produces contains the two sections the old hand-written builder wrote as unmarked indented items
- [ ] The description carries blank lines, a header-shaped line, an interior line ending in trailing spaces, an embedded double quote, a comma and a line beginning with a dash
- [ ] The note text begins with a dash and carries an embedded double quote and a comma
- [ ] `create` and `note add` both succeed for that content
- [ ] The stored title, description, tags, refs and note text are byte-identical to the fixture constants
- [ ] The decoded toon document's `title`, `description` and note `text` are byte-identical to the stored values
- [ ] The decoded toon document's `tags` and `refs` are element-wise byte-identical to the stored tags and refs, in the same order
- [ ] `tick show <id> --field title`, `--field description` and `--field notes.1` each print the stored value followed by exactly one newline
- [ ] Writing the decoded title and description back leaves the stored values byte-identical
- [ ] Re-adding the decoded note text stores a second note whose text is byte-identical to the first's, and the task then carries two notes
- [ ] Every fidelity assertion compares a stored value or a decoded value — none compares rendered output text
- [ ] A value carrying edge whitespace stores trimmed and stays trimmed across a write-back
- [ ] The test runs under `go test ./internal/cli` with no built binary and no network
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it stores the awkward title byte-identically"` — the persisted title equals the fixture constant
- `"it stores the awkward description byte-identically"` — the persisted description equals the fixture constant
- `"it stores the awkward note byte-identically"` — the persisted note text equals the fixture constant
- `"it stores the fixture tags and refs"` — the persisted `Tags` and `Refs` equal the fixture constants element for element
- `"it decodes the stored title out of the show document"` — decoded `title` equals the stored title
- `"it decodes the stored description out of the show document"` — decoded `description` equals the stored description
- `"it decodes the stored note text out of the show document"` — the first `notes` row's `text` equals the stored text, and its `index` reads 1
- `"it decodes the stored tags out of the show document"` — decoded `tags` equals the stored tags element for element
- `"it decodes the stored refs out of the show document"` — decoded `refs` equals the stored refs element for element, the URL's colon quoted by the library
- `"it returns the stored title bare from --field"` — stdout equals the stored title plus one newline
- `"it returns the stored description bare from --field"` — stdout equals the stored description plus one newline
- `"it returns the stored note text bare from --field notes.1"` — stdout equals the stored note text plus one newline
- `"it writes the decoded title back unchanged"` — after `tick update --title`, the persisted title is byte-identical
- `"it writes the decoded description back unchanged"` — after `tick update --description`, the persisted description is byte-identical
- `"it writes the decoded note back as a second note with identical text"` — `Notes[1].Text == Notes[0].Text` and the task carries two notes
- `"it trims edge whitespace on the way in and keeps it trimmed"` — a title and description created with surrounding whitespace store trimmed, and a write-back of the read value stores the same bytes

**Edge Cases**:
- The title and the note text both begin with a dash, so the fixture cannot be written at all without Tasks 1 to 3
- Trailing spaces sit on an interior line, not at the end of a value: edge whitespace is trimmed on the way in and stays trimmed, so a value carrying it could never round-trip and is not what the bar covers
- The description carries blank lines, a header-shaped line, embedded quotes and commas — each the shape a hand-built section mangled before Phase 1
- The fixture carries tags and refs as well as the three free-text carriers: a task carrying neither section exercises neither, and the old `tags[2]:` form with unmarked indented items makes the whole document undecodable, which is what read path A fails on
- A comma-bearing ref cannot reach storage — `task.ValidateRef` (`internal/task/refs.go:17-34`) rejects a ref containing a comma or whitespace, and `task.ValidateTag` (`internal/task/tags.go:24-35`) requires kebab-case — so the fixture's ref is a colon-bearing URL and the comma case stays the formatter-level assertion of Phase 1 task `free-text-round-trip-1-2`
- Tags and refs carry no write-back: they are not free-text carriers, and the byte-identity bar of §2.2 is stated over the title, the description and note text
- The title must stay single-line, since `ValidateTitle` rejects newlines; the multi-line content lives in the description
- Note text is capped at 2000 characters, which the fixture is far under
- The note is written back by adding it again, since notes carry no edit command, and the assertion is made against the new note's stored text
- Both read paths are exercised — the decoded document and `tick show --field` — because §2.1 makes neither a fallback for the other
- The assertion reads the stored value rather than output text, so a reformatting of the document cannot make the test pass or fail spuriously
- The write-back of the title needs no marker because its value follows a registered value-taking flag; the note write-back does need one

**Context**:
> §11, part 2: "One deliberately awkward task becomes a permanent fixture, round-tripped end to end. Free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line, and a line that looks like a section header. It carries that text in all three free-text carriers — title, description and a note — each taking what its shape allows: the title and the note text begin with a dash, and the description carries the multi-line content. Write it in, read it out, decode it, write the decoded text back, and assert the stored value is byte-for-byte what it was — the bar §2.2 sets. A note is written back by adding it again, since notes carry no edit (§6.4), and the assertion is made against the new note's stored text. Whitespace at the very start or end of a value is trimmed on the way in and stays trimmed, which is why the fixture carries its trailing spaces mid-text. That single test would have caught the original description defect, the tags item-marker defect and the refs comma defect — and it is the only test that checks the guarantee this work actually made, which is the round trip (§2.2) rather than parseability."
>
> §2.2: "Read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was."
>
> §2.1: "An agent must be able to fetch one field bare, **and** must equally be able to run one `tick show` and lift usable free text out of the full output. Neither is the designated path with the other as a fallback: the field flag (§9) does not excuse an ambiguous block in full output, and repairing the block does not remove the need for bare single-field output." Both paths are therefore asserted.
>
> §6.1 is what the tag and the ref on this task are here for: today's form writes "a header followed by one raw item per indented line", which "a TOON reader rejects… an item written on its own line carries a leading `- ` marker, and without it the decoder reports a length mismatch", and "the items are also written raw, so a ref containing a comma comes back as two values rather than one, and a URL's colon goes unquoted where the format's own rules would quote it".
>
> §9.2 fixes the bare form this test reads: "The value goes out as a line: its own bytes followed by a single newline." §9.3: "Asked for alone, `--field notes.2` prints that note's text bare, by the one-field rule."
>
> Fixture content, as Go string literals:
> ```go
> const fixtureTitle = `- read the header, carefully`
> const fixtureDescription = "Fix the parser.\n\nSteps:\n  - read the header   \n  - validate \"strictly\", then stop\nDone."
> const fixtureNoteText = `- retried "twice", then it stuck`
> var fixtureTags = []string{"round-trip", "fixture"}
> var fixtureRefs = []string{"https://example.com/issues/42"}
> ```
> The description's second bullet line ends in three spaces and is followed by two further lines, so the trailing whitespace is interior and survives; the value itself starts and ends on non-whitespace. The tags are kebab-case and the ref carries no comma or whitespace because the tool's own validation allows nothing else; the ref's colon is the character the old raw form left unquoted.
>
> Phase 1 added `decodeToonDoc` in `internal/cli/toon_decode_test.go`; `setupTickProject` and `readPersistedTasks` are in `internal/cli/create_test.go`; `runShow` in `internal/cli/list_show_test.go` constructs the `App` with `IsTTY: true`, so an explicit `--toon` is needed for the decoded read path.

**Spec Reference**: `.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md` §11, §2.1, §2.2, §6.1, §6.4, §9.2, §9.3, §10.2
```

**Resolution**: Pending
**Notes**:

---
