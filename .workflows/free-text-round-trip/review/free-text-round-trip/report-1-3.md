TASK: free-text-round-trip-1-3 (tick-a119e7) — Description Becomes One TOON-Quoted Value

ACCEPTANCE CRITERIA:
- The description is emitted as one `description: "…"` line produced by `toon.MarshalString`, with no indented block and no hand-written escaping
- The decoded `description` is byte-identical (`==` on the Go string) to `detail.Task.Description` for the awkward fixture
- Blank lines, interior leading spaces, a `Steps:` header-shaped line and a leading `- ` bullet all survive the round trip
- Embedded double quotes, tabs and carriage returns survive the round trip
- A task with an empty description emits no `description` key at all — unchanged from today
- The description stays the last section of the document
- The whole document decodes with the description present
- Pretty's indented description block is unchanged
- `grep -n 'buildDescriptionSection' internal/cli/` returns nothing
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §6.2 requires the description to become "a single TOON-quoted string, exactly as the library's `toon.MarshalString` produces from a Go string", deleting the hand-written indented block (§2.3: the block "has no count and no terminator — it works today only because it is emitted last and runs to EOF"). The deciding reasons are that quoting becomes the library's responsibility, that the dash-list alternative is larger and needs quotes on nearly every line anyway, and that it puts description text under the same decoding rule note text already obeys. §2.2 sets the bar: read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was. §6.2 accepts that a long description becomes one long line, because a human reads pretty output, which this work leaves alone. §12.2's `tick-core` amendment is explicitly routed away from this task.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/toon_formatter.go:102-104 — the description section, emitted last, inside the `detail.Task.Description != ""` guard, as `encodeToonFields(toon.Field{Key: fieldDescription, Value: detail.Task.Description})`
  - internal/cli/toon_formatter.go:266-272 — `encodeToonFields` marshals via `toon.MarshalString(toon.NewObject(fields...))`; all quoting and escaping is the library's
  - internal/cli/toon_formatter.go:106 — `doc.join(detail.Task.ID)` renders the collected sections, description last
  - `buildDescriptionSection` is gone: `grep -rn 'buildDescriptionSection' internal/ cmd/` returns nothing (exit 1). Its deletion is in commit cfcc96bb.
  - internal/cli/pretty_formatter.go:207-208 — pretty still splits the description into indented lines; the only change since is the field-selection guard added by task 10-1, not this task
- Notes: The delivered code uses the `fieldDescription` constant (internal/cli/show_fields.go:40) rather than the literal the plan's Do-step wrote, and carries the later `sel.includes(fieldDescription)` field-selection guard and the `toonDoc`/error-returning refactors from sibling tasks. Both are consistent with the surrounding code and with the task's intent. Round-trip fidelity is settled by reading the library as well as the tests: the encoder quotes on `\n`/`\r`/`\t`, on a leading `-`, on edge whitespace (`strings.TrimSpace(s) != s`), on `:` and on `"` (toon-go@v0.0.0-20251202084852 internal/format/format.go:27-89), and the decoder reverses exactly those escapes (internal/parse/parse.go:9-44), so encode-then-decode is byte-identical for every character class in the fixture.

TESTS:
- Status: Adequate
- Coverage: internal/cli/toon_formatter_test.go — `assertDescriptionRoundTrip` (53-64) decodes with the real library decoder (`toon.DecodeString` via `decodeToonDoc`, internal/cli/toon_decode_test.go:17-28) and compares with `!=` on the Go string, so each subtest is a byte-identity assertion, not a substring check. Subtests: one-quoted-value with the exact expected line and a single-line check (244-258), blank lines (260), interior leading spaces (264), header-shaped line plus `assertToonKeysAbsent(doc, "Steps")` (268-273), leading dash on a line (275) and on the whole value (279), quotes/tabs/CR (283), the awkward fixture (287), whole-document decode with every section key present (291-317), empty description omits the key and leaves 4 sections (220-242), and the all-sections test now asserting the decoded description (166-169). End-to-end corroboration exists in internal/cli/round_trip_test.go:91-95, 132-138, 204-215 (stored, decoded and written back byte-identically) and internal/cli/toon_decode_test.go:231.
- Notes: The awkward fixture (toon_formatter_test.go:17) carries every class the plan named — `\n\n`, a two-space indent, `Steps:`, `- read the header`, `"`, `\t`, `\r` — and its comment at :15-16 describes it accurately. The per-class subtests plus the combined fixture overlap by design (the plan asked for both); each named class is still distinguishable on failure, so this is not over-testing. No stale assertion of the old block form survives anywhere: the only `description:` expectations in tests are the quoted line at toon_formatter_test.go:253 and the key-order check at :1288.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests, `t.Helper()` on the new helper, "it does X" subtest names, and encoding delegated to the library as the rest of the file does.
- SOLID principles: Good — the formatter no longer owns a private text-assembly rule for one field; it hands the value to the encoder like every other field.
- Complexity: Low — a hand-rolled builder with its own loop was replaced by one call.
- Modern idioms: Yes.
- Readability: Good — the description site now reads identically to the other field sites.
- Issues: None. No comment in the changed code makes a claim the code falsifies.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the toolchain run; reading shows gofmt-shaped source and no vet-visible construct, but cleanliness is only settled by executing the three commands.
