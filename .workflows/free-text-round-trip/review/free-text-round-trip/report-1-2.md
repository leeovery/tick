TASK: free-text-round-trip-1-2 (tick-45148d) — Tags And Refs Use The Library Inline List Form

ACCEPTANCE CRITERIA:
- `tags` and `refs` are emitted as the library's inline form — `tags[2]: has space,plain` — with no indented item lines
- The decoded `tags` and `refs` slices are element-wise byte-identical to `detail.Tags` and `detail.Refs`, in the same order
- A ref containing a comma decodes back as one element, not two
- A ref containing a URL colon and a tag containing a space decode back unchanged
- A single-item list decodes to a one-element slice
- A task with no tags emits no `tags` key at all, and likewise for refs — unchanged from today
- `grep -n 'buildStringListSection\|buildTagsSection\|buildRefsSection' internal/cli/` returns nothing
- Pretty output for tags and refs is unchanged
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§6.1 requires tags and refs to become the library's inline list form (`tags[2]: has space,plain`), produced by the encoder rather than hand-built, so quoting is the library's responsibility and a comma-bearing ref is safe by construction; the hand-written builder is to be deleted, not corrected. §5.2 keeps the presence rule — `tags`, `refs`, `type`, `parent`, `closed`, `description` appear only when the task carries them, while `children`/`blocked_by`/`notes` always render (count-zero header per §8). §4.1 leaves pretty untouched, which is why the long-line cost of an inline list is accepted.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/toon_formatter.go:86-92 (`FormatTaskDetail` calls `encodeToonSection("tags", sections.Tags)` and `encodeToonSection("refs", sections.Refs)` inside the `len(detail.Tags) > 0` / `len(detail.Refs) > 0` guards); internal/cli/toon_formatter.go:336-341 (`encodeToonSection` doc comment now reads "structs become a tabular section, scalars an inline list. Quoting is the encoder's."); commit fd989b94 is the task's change, touching only toon_formatter.go, toon_formatter_test.go and toon_decode_test.go.
- Notes:
  - `grep -rn 'buildStringListSection\|buildTagsSection\|buildRefsSection' internal/` returns nothing — all three builders are gone.
  - Both sections still sit behind presence guards, so a task with no tags/refs emits no key (§5.2 preserved); the count-zero form remains confined to `blocked_by`, `children`, `notes` and `changed` via `emptyToonSection` (toon_formatter.go:325-334).
  - The guards read `len(detail.Tags)`/`len(detail.Refs)` while the encoded value is the narrowed `sections.Tags`/`sections.Refs` (format.go:125-140, added later by the field-selection phase). A narrowing that resolved to zero items would therefore reach the encoder — but `show` rejects an out-of-range position before formatting (`tags.3` on a two-tag task, `tags.1` on a tag-less task; internal/cli/show_fields_test.go:665-671, internal/cli/list_show_test.go:1361-1373), so no CLI input reaches that branch. No defect follows; recorded as context only.
  - Pretty is untouched by this commit and still renders tags as a comma-joined header line (pretty_formatter.go:158-160) and refs as a block (pretty_formatter.go:195-197).
  - README's show sample carries the new form — `tags[2]: auth,backend` / `refs[1]: "https://github.com/org/repo/issues/42"` (README.md:484-486), and readme_samples_test.go decodes the README fences, so the doc sample cannot drift from the encoder.

TESTS:
- Status: Adequate
- Coverage: internal/cli/toon_formatter_test.go:575-632 carries all eight tests the plan names, each asserting on the decoded document rather than on substrings: two-element tags (575), comma-bearing ref `https://x.dev/a?b=1,2` (582), tag with a space (589), URL-colon ref (596), single-item refs list (603, which additionally pins the literal `refs[1]: gh-123` inline form), tags omitted (613), refs omitted (619), and both present in one document (625). `decodeToonDoc` (toon_decode_test.go:17-28) fails the test if the document does not parse, so the old indented-item form could not survive any of these; `assertToonStringList` (toon_decode_test.go:90-111) compares element-wise with `slices.Equal`, which is the byte-identity criterion. End-to-end coverage exists too: round_trip_test.go:156-168 decodes tags and refs out of a real `tick show --toon` document, and conformance_test.go:1115-1116/1133 asserts the same keys present and absent through the CLI.
- Notes: The comma-bearing ref is covered at unit level only, which is the right level — `--refs` takes a comma-separated list at the CLI, so no comma-bearing ref can be created through the command line. No redundant assertions: each subtest pins a distinct edge case from §6.1, and the one remaining `strings.Contains` check (line 607) pins the inline form itself, which decoding alone cannot distinguish from other conformant shapes.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it …", `t.Helper()` on the decode helpers, error wrapping via the existing `*toonEncodeError`.
- SOLID principles: Good — two special-case builders collapse into the one generic `encodeToonSection`, so the section-encoding responsibility now sits in a single place.
- Complexity: Low — the change is net-negative in code (29 lines removed from the formatter, 21 of them the deleted builders).
- Modern idioms: Yes — the generic `encodeToonSection[T any]` handles `[]string` with no new helper, as the plan's verification against toon-go predicted.
- Readability: Good.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the toolchain; reading confirms the call sites, the deleted symbols and the test bodies, but not that the suite, vet and gofmt pass.
