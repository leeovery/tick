TASK: free-text-round-trip-5-5 — The Awkward-Task Fixture Round-Trips Byte-Identically

ACCEPTANCE CRITERIA:
- One task carries all three free-text carriers: a dash-leading title containing a comma, a multi-line description, and a dash-leading note
- The same task carries at least one kebab-case tag and one colon-bearing URL ref, so the document it produces contains the two sections the old hand-written builder wrote as unmarked indented items
- The description carries blank lines, a header-shaped line, an interior line ending in trailing spaces, an embedded double quote, a comma and a line beginning with a dash
- The note text begins with a dash and carries an embedded double quote and a comma
- `create` and `note add` both succeed for that content
- The stored title, description, tags, refs and note text are byte-identical to the fixture constants
- The decoded toon document's `title`, `description` and note `text` are byte-identical to the stored values
- The decoded toon document's `tags` and `refs` are element-wise byte-identical to the stored tags and refs, in the same order
- `tick show <id> --field title`, `--field description` and `--field notes.1` each print the stored value followed by exactly one newline
- Writing the decoded title and description back leaves the stored values byte-identical
- Re-adding the decoded note text stores a second note whose text is byte-identical to the first's, and the task then carries two notes
- Every fidelity assertion compares a stored value or a decoded value — none compares rendered output text
- A value carrying edge whitespace stores trimmed and stays trimmed across a write-back
- The test runs under `go test ./internal/cli` with no built binary and no network
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §11 part 2 requires one deliberately awkward task as a permanent fixture, round-tripped end to end — free text carrying newlines, quotes, commas, a leading dash, trailing spaces at the end of an interior line and a header-shaped line, spread across title, description and a note — written in, read out, decoded, written back, with the stored value asserted byte-for-byte (§2.2's bar). §2.1 makes both read paths first class, so the decoded document and `tick show --field` are both asserted. §6.1 is why the fixture also carries a tag and a ref: the old hand-written tags/refs sections emitted unmarked indented items that made the whole document undecodable. §9.2 fixes the bare form as "its own bytes followed by a single newline", and §9.3 extends it to `--field notes.N`. §11 part 3 requires assertions over decoded values rather than output text.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/round_trip_test.go:15-24 (fixture constants), :37-56 (`setupFixtureTask`), :59-80 (`storedFixture`, `toonString`), :82-280 (`TestAwkwardFixtureRoundTrip`, 16 subtests), :357-379 (`assertBareField`/`bareField`). Delivered in commit 22038583 as a single new file; later commits (00cf6c20, cabd1deb) added `TestHostileValueRoundTrip`, `TestRefusedCharacterRoundTrip` and the `bareField` split to the same file — those belong to other tasks.
- Notes: Every prescribed step is present. The fixture content is exactly the Go literals the plan's Context supplies: title `- read the header, carefully` (dash-leading, comma), description carrying a blank line, the header-shaped `Steps:` line, an interior line ending in three spaces, an embedded double quote and a comma, and note text `- retried "twice", then it stuck`. Tags are kebab-case and the ref is a colon-bearing URL with no comma or whitespace, which is all `task.ValidateRef` (internal/task/refs.go:17-33) and `task.ValidateTag` (internal/task/tags.go:24-35) permit.
  The write half is exercised through the real command surface: `create --quiet --description … --tags … --refs … -- <title>` and `note add <id> -- <text>` via `runTick` (internal/cli/end_of_flags_test.go:16), in-process with `t.TempDir` (internal/cli/create_test.go:19-30) — no built binary, no network.
  Both dash-leading values reach storage by paths I checked rather than assumed: `ValidateFlags` skips the argument after a value-taking flag (internal/cli/flags.go:154-156) and `flagScanner.value` consumes the following argument whatever its shape (internal/cli/flags.go:234-243), so `update --title "- read the header, carefully"` needs no marker; `RunNoteAdd` concatenates flag args with post-marker literals (internal/cli/note.go:41), so the dash-leading note arrives whole.
  The `--field` read path returns the bare value through `fmt.Fprintln` (internal/cli/show.go:70-74) whatever the format, and `notes.1` resolves through `barePositionValue` (internal/cli/show_fields.go:135-145), so the bare-form assertions exercise the real §9.2/§9.3 surface.
  One deliberate, documented divergence from the criteria's wording: the decoded-tags subtest compares against `slices.Sorted(...)` of the stored tags (round_trip_test.go:160-161) rather than stored order. The comment on line 160 is accurate — the detail query is `ORDER BY tag` (internal/cli/show.go:155) while storage keeps first-occurrence order (`DeduplicateTags`, internal/task/tags.go:37-40) — so decoded order is alphabetical by design and the criterion's "in the same order" cannot hold literally. The set is still asserted element-wise and byte-identically, which is what §6.1's defect class needs; tags carry no write-back. No loss, so no finding.

TESTS:
- Status: Adequate
- Coverage: All sixteen subtests the plan's Tests section names are present and named as given. Write half (round_trip_test.go:83-121): stored title, description, note text, tags and refs against the fixture constants. Read path A (:123-169): decoded `title`, `description`, first `notes` row's `text` with `index` 1, plus `tags` and `refs` as string lists. Read path B (:171-187): `--field title`, `--field description`, `--field notes.1`. Write-back (:189-241): title, description, and the note re-added as a second note with `Notes[1].Text == Notes[0].Text` and a count of two. Edge whitespace (:243-279): a title and description created with surrounding whitespace store trimmed and stay trimmed across a decoded write-back.
- Notes: The tests would fail if the guarantee broke. `decodeToonDoc` fatals on a document a real TOON reader rejects (internal/cli/toon_decode_test.go:17-28), so a regression to the old unmarked-item tags/refs form fails read path A outright; the byte-identity comparisons are `!=` against stored values, so any mangling of quotes, commas, the leading dash or the interior trailing spaces fails. `bareField` (round_trip_test.go:368-379) fatals when stdout carries no trailing newline and then compares the remainder to the stored value, so a value followed by two newlines or by none both fail — "exactly one newline" is genuinely pinned. No assertion compares rendered document text; the `--field` assertions compare a bare value, which §9.2 defines as not a document and which the plan's step 4 prescribes.
  Not over-tested: each subtest carries one distinct assertion from the plan's list, and the per-subtest `setupFixtureTask` rebuild buys isolation rather than redundancy. No overlap with `TestToonTaskDetailConformance` (internal/cli/toon_decode_test.go:184-278), which seeds JSONL directly and never exercises write → read → write-back.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run` subtests named "it does X", `t.TempDir` isolation via `setupTickProject`, `t.Helper()` on every helper, existing helpers (`readPersistedTasks`, `decodeToonDoc`, `runToonCommand`) reused rather than duplicated.
- SOLID principles: Good — `setupFixtureTask`, `storedFixture`, `toonString` and `bareField` each do one thing, and three of the four have since been reused by another test file.
- Complexity: Low
- Modern idioms: Yes — `slices.Equal`, `slices.Sorted(slices.Values(...))`, `strings.CutSuffix`.
- Readability: Good — the fixture's shape is stated once at the top and each subtest reads as a single claim.
- Comment accuracy: The file header comment (:11-14) and the tags-ordering comment (:160) both hold against the code and against `show.go:155`. No process-artifact references.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the toolchain run; reading cannot settle a suite result or a formatter's verdict. The same run settles whether the fixture's assertions actually hold end to end (that `create`/`note add` accept the content, that the stored and decoded values match byte-for-byte, and that the bare forms print one trailing newline) — reading confirms only that each assertion is present, correct in shape, and would fail if the behaviour it names broke.
