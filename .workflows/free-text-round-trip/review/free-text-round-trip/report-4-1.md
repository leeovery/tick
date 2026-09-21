TASK: free-text-round-trip-4-1 — Field And Fields Are Registered On Show And Parsed (tick-794c1f)

ACCEPTANCE CRITERIA:
- `--field` and `--fields` are both registered against `show` and against no other command; `tick create --field title` still fails with the unknown-flag error
- `TestCommandFlagsMatchHelp` passes with both spellings present in `show`'s help entry
- `tick show --field title tick-a1b2` and `tick show tick-a1b2 --field title` both resolve the same task
- `--field "title, status"` parses the same selection as `--field title,status`
- `--field title --field status` parses the same selection as `--field title,status`, and `--fields` composes with `--field`
- `--field title,title` yields a selection whose `Len()` is 1
- Every name in the registry parses whether or not the task carries it
- `--field titel`, `--field ""`, `--field "title,,status"`, `--field "title,"`, `--field notes.x`, `--field notes.1.2` and `--field title.1` each exit non-zero with the unrecognised-name error and nothing on stdout
- `--field` with no following argument exits non-zero with `--field requires a value`
- `--quiet` with any selection exits non-zero with nothing on stdout, and `--quiet` without one still prints the task ID
- `notes.2` parses as the `notes` section narrowed to position 2; `notes,notes.2` parses as `notes` taken whole
- A recognised selection produces stdout byte-identical to the same command without the flag
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §9 makes `--field` `show`'s flag and no other command's — it is `show`'s first command-specific flag. §9.1 makes `--fields` an alias, fixes the vocabulary to the names the output document uses (scalars plus the section keys), and sets the lenience rules: whitespace around a name is not part of it, repeated flags compose, a repeated name collapses and counts once, a section named both whole and by position comes back whole, and a positional suffix attaches only to a list section. §9.6 makes an unrecognised name — including a blank name from an empty value, a doubled comma or a trailing comma, and a non-numeric suffix — a non-zero exit, and keeps `notes.0`/`notes.-1` on the out-of-range path rather than the unrecognised path. §9.8 refuses `--quiet` alongside a selection with nothing on stdout. §12.1 records that registration in `commandFlags` is a constraint, since the unrecognised-name error cannot fire without it and `TestCommandFlagsMatchHelp` fails on a registered long flag with no help entry.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/flags.go:71-74 — `"show"` now carries `"--field": {TakesValue: true}` and `"--fields": {TakesValue: true}`; a grep for `field` in flags.go returns only these two lines, and `ready`/`blocked` are derived from `list` (flags.go:102-104), so no other command picks the flag up.
  - internal/cli/help.go:72-79 — `Usage` is `tick show <task-id> [flags]` and the single `flagInfo` is `{"--field, --fields", "<name,...>", "Select fields by name; a section may be narrowed with .N (e.g. notes.2)", false}`. `TestCommandFlagsMatchHelp` splits `Name` on `", "` and keeps `--`-prefixed parts (internal/cli/flag_validation_test.go:345-353), so both spellings match the registry in both directions.
  - internal/cli/show_fields.go:30-68 — the recognised-name registry: the nine scalars, `description`, and the five list sections, each declared once.
  - internal/cli/show_fields.go:149-239 — `FieldSelection` with `names` (first-seen order), `whole`, `positions`, plus `Positions` (nil for a name taken whole), `Len`, `Only`, and `addValue`, which splits on `,`, `TrimSpace`s each part, cuts at the first `.`, requires a list section as the base and `strconv.Atoi` on the suffix, and otherwise returns `unknown field %q for "show". Run 'tick help show' for usage.` with the name exactly as typed. `addWhole` deletes any positions already recorded; `addPosition` is a no-op once the section is whole (show_fields.go:202-216).
  - internal/cli/show_fields.go:248-282 — `parseShowArgs` walks `flagArgs` with `newFlagScanner`, accumulates every `--field`/`--fields` occurrence (attached `--field=x` and separated spellings both, via `flagScanner.value()` at internal/cli/flags.go:234-243), returns `--field requires a value` when no value follows, skips other flag-shaped arguments, takes the first non-flag argument as the ID, and falls back to `literals[0]` when none appeared.
  - internal/cli/show.go:38-47 — `RunShow` calls `parseShowArgs(flagArgs, literals)`, keeps `task ID is required`, and returns `--quiet cannot be combined with --field` when `fc.Quiet` and the selection is non-nil. Both run before `openStore` (show.go:49), so nothing reaches stdout.
- Notes:
  - Two deliberate divergences from the plan's wording, both sound. (1) The plan's signature `parseShowArgs(args []string)` is delivered as `parseShowArgs(flagArgs, literals []string)`, matching the end-of-flags split every handler now takes (internal/cli/app.go:201); it is what lets `tick show -- -weird-id` resolve a dash-leading ID, and the `--field -- title` case still reports the missing value (internal/cli/end_of_flags_test.go:656-666). (2) The plan's `Selected(name string) bool` is delivered as the unexported `includes` (show_fields.go:164); the only consumers are in-package formatters and tests, so nothing needs it exported.
  - The criterion "A recognised selection produces stdout byte-identical to the same command without the flag" was the transitional behaviour this task's own Solution scoped "until Task 2"; task 4-2 and later delivered the filtered document, so the criterion is superseded by the delivered change-set rather than unmet. The corresponding test is now `"it renders the named fields for a recognised selection"` (internal/cli/show_fields_test.go:234-246), which pins `--field title,status` to `Title:    Add login\nStatus:   open\n`.

TESTS:
- Status: Adequate
- Coverage: internal/cli/show_fields_test.go:31-222 covers the parse layer directly — nil selection with no flag, comma list order, plural spelling, composition across repeated flags, whitespace trimming, repeated-name and repeated-position collapse (within one value and across flags), `Len`/`Only` for one and several names, ID read past the flag value, every registered name via `maps.Keys(showFields)`, `notes.2` narrowing, `notes,notes.2` and `notes.2,notes` both coming back whole, `notes.0`/`notes.-1` accepted as positions, the eight unrecognised-name cases asserted against the exact message (including `notes.1.2` and `titel.1`), `--field`/`--fields` with no value, and an empty ID. internal/cli/show_fields_test.go:224-396 covers the command surface: exit code plus empty stdout for `titel`, the empty value, the doubled comma and the trailing comma; the missing value both bare and followed by `--json`; the `--quiet` refusal; `--quiet` alone still printing the ID; `--field closed` on an open task exiting zero; flag-before-ID resolution; and `tick create X --field title` failing with the unknown-flag error. internal/cli/flag_validation_test.go:97-105 pins `show`'s registry at exactly the two flags and drops `show` from the no-flag command list (flag_validation_test.go:144-148).
- Notes: The unrecognised-name and missing-value cases appear both as unit assertions on `parseShowArgs` and as command-level assertions via `runShow`. That is not redundancy — the acceptance criteria are about exit code and an empty stdout, which only the command-level run can observe, while the unit table is what pins the exact message. `runShow` runs with `IsTTY: true`, so these assertions are against pretty output, per the project's test convention.

CODE QUALITY:
- Project conventions: Followed. `fmt.Errorf` with no arguments matches every sibling `--x requires a value` (internal/cli/list.go:49, internal/cli/create.go:48); the sentence-shaped error mirrors `ValidateFlags`' own wording and ST1005 is disabled in .golangci.yml for exactly that reason. Tests are stdlib-only with "it does X" subtests and `t.Helper()` on `parseSelection`.
- SOLID principles: Good. Parsing, the selection type and the name registry sit in one file behind a single entry point; `RunShow` gained four lines and no branching on field semantics.
- Complexity: Low. `addValue` is one loop with two exits; `parseShowArgs` one scan with a single flag case.
- Modern idioms: Yes — `strings.SplitSeq`, `strings.Cut`, `slices.Contains` under go 1.26.
- Readability: Good. Names are declared once as constants and reused by the formatters, so a name cannot drift between the registry and the renderers.
- Issues: None worth acting on.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the three commands run; reading found nothing that would break them (no unused imports or symbols introduced, formatting consistent), but only an executing pass settles it.
