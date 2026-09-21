TASK: free-text-round-trip-4-7 — Out-Of-Range Positions Are An Error (tick-b55ed5)

ACCEPTANCE CRITERIA:
- `--field notes.4` on a two-note task exits non-zero with `notes.4 out of range: task has 2 note(s)`
- `--field notes.0` exits non-zero with the same grammar, reporting the task's real note count
- `--field notes.-1` exits non-zero the same way
- `--field tags.1` on a tag-less task exits non-zero with `tags.1 out of range: task has 0 tag(s)`
- `--field children.2`, `--field blocked_by.1` and `--field refs.3` fail the same way against their own counts and nouns
- Stdout is empty on every one of those, including when another name in the selection would have rendered
- `--field tags` on a tag-less task still exits zero and prints what full output prints for it
- `--field notes` on a note-less task still exits zero with its count-zero header
- A non-numeric suffix still takes Task 1's unrecognised-name error, not this one
- The first reported failure is deterministic for a selection carrying several out-of-range positions
- An in-range position still renders the narrowed section, in all three formats
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §9.6 requires a position that names nothing to fail with a non-zero exit and a message naming the range — `notes.4` past the end, `notes.0` below the 1-based floor, and `tags.1` on a tag-less task all alike — while keeping the empty-field success case for a section named whole (`tags` on a tag-less task prints what full output prints and exits zero; `notes` prints its count-zero header). The message must reuse the grammar `note remove` already carries (`internal/cli/note.go:131`: `index %d out of range: task has %d note(s)`) rather than inventing a second one. §9.3 extends the rule to every list section with its own count. §9.1 keeps a non-numeric suffix on the unrecognised-name error.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/show_fields.go:284-298` — `(*FieldSelection).ValidatePositions(TaskDetail) error`, walking `showListSections` in document order and each section's positions ascending (`slices.Sorted(slices.Values(...))`, which copies rather than reordering the stored slice), returning on the first `pos < 1 || pos > length`.
  - `internal/cli/show_fields.go:293` — the message `"%s.%d out of range: task has %d %s"`, keeping the `out of range: task has N x` half verbatim from `note.go:131` and putting the selector as typed in front.
  - `internal/cli/show_fields.go:61-65` — per-section `noun`/`length` on the `showField` registry: `blocker(s)`, `child(ren)`, `tag(s)`, `ref(s)`, `note(s)`.
  - `internal/cli/show_fields.go:70` — `showListSections` fixes the document order `blocked_by, children, tags, refs, notes`, which matches the order `ToonFormatter.FormatTaskDetail` emits them in (`internal/cli/toon_formatter.go:78-95`), so the first reported failure is deterministic.
  - `internal/cli/show.go:67-69` — called immediately after `showDataToTaskDetail` and before the bare-value path (line 71), the quiet branch (line 78) and any rendering (line 84), so nothing reaches stdout on failure.
- Notes: `Positions(name)` returns nil for a section also named whole (`show_fields.go:171-176`), so `--field notes,notes.9` succeeds — deliberate, matching §9.1's "a section named both whole and by position comes back whole", and stated in the method's doc comment. `internal/cli/show.go:83` is the only assignment of `detail.Fields` in non-test code, so `RunShow` is the only path a selection reaches a formatter by and the single validation point covers it; `--field`/`--fields` are registered against `show` alone (`internal/cli/flags.go:71-73`). The empty-field success case is preserved by the formatters themselves — toon omits `tags`/`refs` when the task carries none (`toon_formatter.go:86,90`) and always emits the notes section (`toon_formatter.go:94`).

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/show_fields_test.go:621` `TestValidatePositions` — in-range positions on all five sections accepted; whole sections on an empty task accepted; nil selection accepted; a message table covering `notes.3`, `notes.0`, `notes.-1`, `tags.3`, `refs.2`, `children.2`, `blocked_by.2` asserting the exact string and noun; `tags.1` on an empty detail giving `task has 0 tag(s)`; document-order determinism regardless of argument order (`refs.9,notes.9` and `notes.9,refs.9` both report refs); the lowest out-of-range position within a section; positions ignored on a section also named whole.
  - `internal/cli/list_show_test.go:1303` `TestShowFieldPositionOutOfRange` — end-to-end over `App.Run`, asserting non-zero exit, empty stdout and exact `Error: ...` stderr for `notes.4`, `notes.0`, `notes.-1`, `tags.1` (tag-less project), `children.2`, `blocked_by.2`, `refs.3`; empty stdout for `title,notes.4` where another name would have rendered; determinism across three repeated runs; `notes.x` keeping the unrecognised-name error; `tags` on a tag-less task exiting zero with empty stdout; `notes` on a note-less task exiting zero with the count-zero header (`assertCountZeroSection`); `notes.2` rendering bare and exiting zero in `--toon`, `--pretty` and `--json`.
  - `internal/cli/show_fields_test.go:606` `TestShowListSectionsMatchRegistry` — drift guard: every list entry in `showFields` must appear in `showListSections`, so a new list section cannot be added without a noun, a length and an order slot.
  - In-range rendering in each format is covered outside this task's block too — `list_show_test.go:1233` (`--json --field notes.2,title`), `list_show_test.go:1274,1283` (pretty children/blocked_by narrowing), `json_formatter_test.go:1764` (`title,notes.2` key set).
- Notes: The unit and end-to-end layers overlap on `notes.0`, `notes.-1`, `children.2` and `blocked_by.2`, but they measure different things (message text versus exit code, stderr routing and empty stdout) and both layers were prescribed by the task. Each assertion would fail if the validator were removed or its ordering made nondeterministic. `blocked_by.1` from the criteria is exercised as `blocked_by.2` against a one-blocker task — same substance, since the project fixture carries one blocker.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` "it does X" subtests, `t.Helper()` on `assertRejected`/`richProject`/`bareProject`, `t.TempDir()`-backed fixtures via `setupTickProjectWithTasks`, error strings lowercase and unwrapped, no testify.
- SOLID principles: Good — the noun and length live on the field registry beside the rest of each field's description rather than in a parallel table, so one entry describes a section completely.
- Complexity: Low — two nested loops with a single early return.
- Modern idioms: Yes — `slices.Sorted(slices.Values(...))` (Go 1.23+ iterator form), non-mutating, consistent with `selectedItems`.
- Readability: Good — the doc comment states the ordering guarantee and the whole-section exemption, both of which the tests pin.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the three commands run from the repo root; reading cannot settle a suite pass, a vet report or formatting output.
