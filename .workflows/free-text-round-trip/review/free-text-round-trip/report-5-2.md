TASK: free-text-round-trip-5-2 (tick-f3937c) — Post-Marker Text That Spells A Command Flag Is Text

ACCEPTANCE CRITERIA:
- `tick create -- --priority` exits zero and stores the title `--priority`
- `tick create -- --description foo` stores the title `--description`, leaves the description empty and ignores `foo`
- `tick create --priority 0 -- "- title"` stores priority 0 and the title `- title`
- `tick show <id> -- --field title` renders full output rather than a field selection, and resolves the same task
- `tick show -- <id>` resolves the task
- `tick update <id> --title x -- --json` stores the title `x` and renders in the resolved default format
- `tick remove -- --force` treats `--force` as a task ID and reports it unresolved rather than skipping confirmation
- `tick list -- --status` exits zero and lists tasks, ignoring the literal as a stray positional
- `tick note add <id> -- "- text"` stores `- text`, with the marker absent from the joined text
- A command with no positional argument behaves as it does today apart from no longer reading post-marker text as a flag
- A selection flag whose value would sit after the marker (`tick show <id> --field -- title`) still reports `--field requires a value`
- Every invocation that carries no marker produces byte-identical output and the same exit status as before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §10.2 requires that nothing after the end-of-flags marker is read as a flag — "not the command's own flags and not the global format flags" — so free text spelling a flag exactly (`--json`, `--priority`) is writable for the same reason a dash-leading string is. §10.1 insists the unknown-flag check itself stays intact for pre-marker arguments, so `tick update <id> --prioirty 3` still refuses. §9.1 fixes what a post-marker `--field` must not do: a literal never reaches the field-selection parser, so `tick show <id> -- --field title` is a request for a task, not a selection. Task 1 of this phase carried the boundary as a trailing literal count; this task pushes the split down into each command's own parser.

IMPLEMENTATION:
- Status: Implemented (with a later, deliberate refactor of the split site)
- Location:
  - internal/cli/app.go:43 — `flagArgs, literals := splitLiteralArgs(subArgs, flags.literals)` splits once, above dispatch; both halves are handed to every handler (app.go:118-141)
  - internal/cli/flags.go:181 — `splitLiteralArgs` clips flagArgs' capacity (`args[:boundary:boundary]`), so `handleReady`/`handleBlocked`'s prepend-append (app.go:219, app.go:232) cannot overwrite the first literal
  - internal/cli/create.go:39 — `parseCreateArgs(flagArgs, literals)`; flags scanned over flagArgs only, literals replayed through `addPositional` (create.go:104-106), preserving first-positional-wins
  - internal/cli/update.go:46 and update.go:116-118 — same shape, positional being the task ID
  - internal/cli/list.go:37 — `parseListFlags(flagArgs, _ []string)`; list has no positional, so literals are ignored exactly as a stray positional is
  - internal/cli/show_fields.go:248 — `parseShowArgs(flagArgs, literals)`; `--field`/`--fields` recognised in flagArgs only, and show_fields.go:274-276 uses a literal as the task ID only when no pre-marker ID was found
  - internal/cli/remove.go:21 and remove.go:39-41 — literals join the ID list in order through the same `addID` closure, keeping the lowercase dedupe
  - internal/cli/note.go:41, internal/cli/transition.go:17, internal/cli/dep.go:57 — index-reading commands concatenate the halves in order via `slices.Concat`
- Notes: The plan's step 1 asked each handler to call `fc.SplitLiterals(args)`. The delivered commit (a3489a26) did exactly that; the follow-on task 5-7 (18d1deb2) hoisted the split into `App.Run` and removed `FormatConfig.SplitLiterals`/`Literals`. `grep -rn "Literals" internal/cli/*.go` finds no production leftovers, and CLAUDE.md records the hoisted signature as the project's handler contract. The substance this task required — flags matched against the pre-marker half only, the post-marker half joining positionals in order — is intact, so the divergence is an improvement carried forward, not a loss.
  Each criterion traced through the code:
  - `create -- --priority` → flagArgs empty, ValidateFlags sees nothing, title = `--priority`, priority left at its default 2
  - `create -- --description foo` → both literals hit `addPositional`; first wins, second dropped, description empty (a literal can never pair as a flag value)
  - `create --priority 0 -- "- title"` → flag half parses normally; literal becomes the title
  - `show <id> -- --field title` → id resolved from flagArgs, selection stays nil, full document rendered
  - `show -- <id>` → show_fields.go:274 picks up literals[0]
  - `update <id> --title x -- --json` → parseArgs (app.go:363-375) stops applying global flags at the marker, so `--json` never reaches `globalFlags`; the format resolves to the TTY default and the literal is a losing positional
  - `remove -- --force` → parseRemoveArgs sees `--force` only in the literal half, so force stays false and `store.ResolveID("--force")` errors before the confirmation path
  - `list -- --status` → literal ignored; identical output to a bare `list`
  - `note add <id> -- "- text"` → `flagScanLimit["note add"]` (flags.go:100) caps validation at the ID, and RunNoteAdd joins flagArgs+literals with the marker already dropped by parseArgs
  - `show <id> --field -- title` → the value would sit past the flag half, so `flagScanner.value()` returns ok=false and the parser reports `--field requires a value` (show_fields.go:257)

TESTS:
- Status: Adequate
- Coverage: `TestPostMarkerArgumentsAreNeverFlags` (internal/cli/end_of_flags_test.go:433-667) carries one subtest per acceptance criterion, driving the real `App.Run` through the per-command helpers and asserting against persisted JSONL rather than parser internals: post-marker `--priority` as a title, a post-marker `--description` swallowing nothing, pre-marker flags still parsing, the second post-marker positional ignored, `--field` as text on show, a post-marker ID on show, `--json` as text on update (asserted by `json.Valid(stdout)` being false, which a regression would flip), `--force` as an unresolved ID on remove with the task still present, `--status` ignored on list, the note-text join, and index-reading commands (`done`, `dep add`, `note remove`) receiving post-marker arguments in order. `--field requires a value` is pinned at end_of_flags_test.go:657-667. `TestSplitLiteralArgs` (end_of_flags_test.go:400) pins the boundary helper including the clamp.
- Notes: The existing parser suites were updated for the new two-argument signature only — the remove_test.go/show_fields_test.go diffs in a3489a26 pass `nil` literals and change no assertion, so no pre-existing guard was weakened. One benign overlap: `TestPostMarkerArgumentsAreNeverFlags/"it passes post-marker arguments to dep add in order"` (end_of_flags_test.go:625) repeats the invocation and assertion already made by `TestEndOfFlagsMarker/"it keeps the marker working on a command with a sub-subcommand"` (end_of_flags_test.go:172). Nothing breaks from either existing, so it is recorded here rather than raised as a finding.

CODE QUALITY:
- Project conventions: Followed — stdlib testing with `t.Run` "it does X" naming and `t.TempDir()`-backed fixtures; handler signature matches the contract CLAUDE.md records; error strings keep the project's lowercase-flag phrasing.
- SOLID principles: Good — the boundary is computed in one place (flags.go:181) and consumed by parsers that each own one command's grammar; `addPositional` on `createOpts`/`updateOpts` gives the flag half and the literal half a single definition of first-positional-wins rather than duplicating the rule.
- Complexity: Acceptable — the parsers are flat switch loops over `flagScanner`, with a single trailing `for range literals`.
- Modern idioms: Yes — `slices.Concat` for the index-reading commands, three-index slicing to clip capacity, `min` builtin, blank parameter names where a half is deliberately unused.
- Readability: Good — each parser's doc comment states the rule it implements ("Flags are recognised in flagArgs only; every literal is a positional"), and each statement holds against the code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Every invocation that carries no marker produces byte-identical output and the same exit status as before this task" — reading shows the no-marker path yields `literals == nil` and parser behaviour equivalent to the pre-task index loops, but byte-identity across the command surface needs the suite (and a diff against the pre-phase binary) run to settle.
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by executing those three commands.
