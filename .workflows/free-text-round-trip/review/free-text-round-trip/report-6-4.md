TASK: free-text-round-trip-6-4 — Field-Selection Documents And The Two Exemptions Are Covered

ACCEPTANCE CRITERIA:
- `tick show <id> --field description,notes --toon` decodes and carries exactly the two selected keys
- `tick show <id> --field description,notes.2 --toon` decodes with a one-row `notes` list whose `index` is `2`, alongside the whole description
- `tick show <id> --field notes --toon` decodes and carries `notes` alone
- The bare-value output equals the stored value's bytes followed by exactly one newline
- That output is byte-identical with no format flag, with `--toon`, with `--pretty` and with `--json`
- A selection whose every name prints nothing produces zero bytes and exits zero, identically in all four spellings
- `--quiet` combined with a selection exits non-zero with zero bytes on stdout
- The three exemptions are entries in `conformanceDocs` carrying their reasons, not absences
- The driver skips them rather than attempting a decode
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT: §11 counts conformance coverage in documents, requiring every document `show` produces — "a multi-field selection and one narrowed by position included" — to be decoded by a real TOON reader. §3.1 exempts exactly two outputs from the must-parse rule because they are not documents: the bare value of §9.2 and the zero-byte selection of §9.6. §9.7 keeps a bare value outside the format flags entirely; §9.8 refuses `--quiet` alongside a selection with a non-zero exit and nothing on stdout. §9.2 fixes the bare form as "its own bytes followed by a single newline", which is why a byte assertion rather than a decode is the check that reaches it.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/conformance_test.go:168-177 — `conformanceSelectedID`, `conformanceSelectedDescription` ("Line one\nLine two") and `conformanceSelectedTask()`, the shared seed carrying a description and the two `conformanceNotes`
  - internal/cli/conformance_test.go:360-382 — the three filtered `show` inventory entries: `--field description,notes`, `--field description,notes.2`, `--field notes`
  - internal/cli/conformance_test.go:383-399 — the three `NotADocument` entries (bare value, zero-byte selection, `--quiet` with a selection), each carrying its reason
  - internal/cli/conformance_test.go:1136-1165 — decoded-value subtests for the three filtered documents
  - internal/cli/conformance_test.go:1179-1290 — `TestFieldSelectionExemptions`
  - internal/cli/conformance_test.go:1167-1177 — `conformanceFormatSpellings` (nil, `--toon`, `--pretty`, `--json`) and `runConformanceShow`
  - internal/cli/conformance_test.go:1327-1335 — `assertConformanceKeys`, the exact-key assertion the three decode subtests rest on
  - Skip path: internal/cli/conformance_test.go:654-671 (`conformanceSkipReason` / `driveConformanceEntry`) — an entry carrying `NotADocument` is skipped before `Setup` is called, under both drivers
- Notes: the whole task lands in one test file; the commit (d337d1c0) touches nothing else, which matches a task whose deliverable is coverage. The production behaviour every assertion pins was traced and holds by reading: `RunShow` refuses `--quiet` with a selection before any output (internal/cli/show.go:45-47), takes the bare path before any formatter is reached so the four format spellings cannot differ (show.go:71-76), and prints nothing when the formatter returns an empty document (show.go:88-90). The zero-byte case is real in all three formats — toon's `buildTaskSection` returns "" when no field survives selection (internal/cli/toon_formatter.go:256-258) and `joinToonSections` drops empty sections (toon_formatter.go:286-294); JSON returns "" for an empty object (internal/cli/json_formatter.go:94-97) and omits `parent`/`closed` when absent (json_formatter.go:126-133); pretty returns "" when no group survives (internal/cli/pretty_formatter.go:224-226). A later commit (daacfe1b, "phase 6 comment corrections") removed the doc comment this task added above `conformanceSelectedTask` — a deliberate correction, not a loss.

TESTS:
- Status: Adequate
- Coverage: all eleven micro-acceptance tests named by the plan exist under the names the plan gives them. The three filtered documents are decoded through the same driver as every other inventory entry (`TestToonOutputConformance`, conformance_test.go:694) and additionally carry decoded-value assertions: exact-key sets, the whole description, and for the narrowed entry a single `notes` row whose `index` decodes as `float64(2)` with the second note's text and timestamp. The exemptions are asserted behaviourally (bytes, exit codes) rather than by decode, which is the only assertion that reaches output no decoder accepts.
- Notes: the assertions would fail if the behaviour broke — `assertConformanceKeys` compares the full sorted key set, so a field riding along unasked (§9.5) fails it, and a section silently dropped fails it too. The skip test (conformance_test.go:1268-1290) proves the driver does not reach `Setup` by observing a flag the fixture's own setup would set, which is the only observable the skip has. Two subtests overlap by design: the nil spelling inside "it prints the same bare bytes under every format flag" repeats what "it prints a bare value as its bytes plus one newline" asserts, and the same holds for the two zero-byte subtests — both pairs are prescribed by the plan's test list, and they carry different intents (the byte contract of §9.2 versus the format-flag invariance of §9.7), so this is not over-testing worth undoing.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run` subtests named "it does X", `t.Helper()` on every helper, fixtures seeded through `setupTickProjectWithTasks` on `t.TempDir()`.
- SOLID principles: Good — `conformanceEntry` was split out of `decodeConformanceEntry` so the exemption subtests can read an entry without running it, rather than duplicating the lookup.
- Complexity: Low
- Modern idioms: Yes — `slices.Concat` for the format spelling, `slices.Sorted(maps.Keys(...))` for the key-set comparison.
- Readability: Good — the exemption reasons read as prose a later reader can act on, and the inventory now reads as a complete account rather than one with silent holes.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settle by running the four commands from the repo root. This criterion also carries the decode criteria: reading confirms every section of a filtered document is produced by `toon.MarshalString` (the same library the test decodes with) and that the assertions are correctly shaped, but that the documents actually decode is observable only by running the suite.
