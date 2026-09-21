TASK: free-text-round-trip-1-5 (tick-7389ba) — Detail-Document Assertions Check Decoded Values

ACCEPTANCE CRITERIA:
1. `tick show --toon` output decodes via the TOON library for a task carrying every optional field and for one carrying none
2. Both decoded documents carry `blocked_by`, `children` and `notes`; the all-fields one additionally carries `type`, `parent`, `closed`, `tags`, `refs` and `description`, and the bare one carries none of those six
3. `note add` and `note remove` toon output decodes the same way
4. No test asserts `task{` anywhere in the repository
5. No toon or JSON assertion on the detail document compares against a pinned full-output string; each checks a decoded value or key presence
6. `internal/cli/pretty_formatter_test.go` carries the same golden-string assertions as before this phase, none removed or weakened
7. The JSON detail tests still assert through `json.Unmarshal` into `map[string]any`, as they already do
8. `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT:
§11 part 1 requires every structured command's output to be decoded by a real TOON reader in the suite, failing the test if it will not parse — "this alone catches the entire class of defect this work exists to fix — a section nobody can read, whatever its content." §11 part 3 requires rewritten assertions to check decoded values, not output text: "'The notes section has two rows and the second row's text is X', not 'the output equals this blob'." §11 also states the trade explicitly: no byte-level pinning is kept in the machine formats (toon and JSON), while pretty keeps golden-string assertions because "pretty output has no parser, so a decoded-value assertion does not exist for it". §3.1 lists the five commands emitting the task-detail document; `show` renders it inline while `create`, `update`, `note add` and `note remove` share `outputMutationResult`. The task is scoped to the task-detail document only; Phase 6 extends decode coverage to the whole inventory.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/toon_decode_test.go:184-278` — `TestToonTaskDetailConformance` with the four required subtests, driven through `App.Run` against a store seeded by `setupTickProjectWithTasks`
  - `internal/cli/toon_decode_test.go:37-56` — `toonRows`, generalised out of the pre-existing `decodeToonNotes`
  - `internal/cli/toon_decode_test.go:72-79` — `assertToonKeysPresent`
  - `internal/cli/toon_decode_test.go:113-127` — `runToonCommand`
  - `internal/cli/toon_decode_test.go:129-155` — `assertToonNoteRows`, `assertToonRelatedRow`
  - `internal/cli/toon_decode_test.go:175-182` — `assertToonRowsEmpty`
  - `internal/cli/format_integration_test.go:27-29, 311-313, 621-623, 664-666` — the four former `strings.HasPrefix(stdout, "id: ")` / `task{` probes, now `decodeToonDoc` + `assertToonFields` + `assertToonKeysPresent(…, "notes")`
  - `internal/cli/toon_formatter_test.go:224` — count-zero `blocked_by`/`children` now additionally decoded (added by this task's commit c4ffd035)
- Notes:
  - Criterion 1 holds: both `show` branches decode through `toon.DecodeString` (`toon_decode_test.go:19`), which fails the test on a parse error.
  - Criterion 2 holds. Full task (`toon_decode_test.go:216-238`): `type`, `parent`, `closed`, `tags`, `refs`, `description` all asserted with their stored values; `blocked_by`, `children`, `notes` asserted via `assertToonRelatedRow`/`assertToonNoteRows`, both of which route through `toonRows` and `t.Fatalf` on a missing key. Bare task (`:240-255`): `assertToonKeysAbsent` covers the six, and `assertToonRowsEmpty(t, doc, "blocked_by", "children", "notes")` asserts presence-and-empty in one step.
  - Criterion 3 holds (`:257-277`). `note add` checks the third row's `index` and `text`; `note remove tick-aaa111 1` checks the surviving note is re-indexed from 1 — `RunNoteRemove` is 1-based (`internal/cli/note.go:105-133`), so `notes[1:]` is the correct expectation.
  - Criterion 4 holds repo-wide: `grep -rn 'task{' --include='*.go' .` outside `.workflows/` returns 0 hits; the only `*.md` hits are `.workflows/` planning and specification records quoting the old header deliberately.
  - Criterion 5 holds. `grep -n 'want := \`|expected := \`' ` across `toon_formatter_test.go`, `json_formatter_test.go`, `format_integration_test.go`, `detail_changes_test.go`, `note_test.go`, `create_test.go`, `update_test.go` and `show_fields_test.go` returns three hits, none a full detail document: the single-line description-section check (`toon_formatter_test.go:253`), a `FormatRemoval` prose string (`:486`), and an error-message string (`show_fields_test.go:195`). The edge case's claim that `create_test.go`, `update_test.go` and `note_test.go` hold no JSON golden blobs is confirmed rather than assumed.
  - Criterion 6 holds: `git show --stat` on all seven Phase 1 commits (eb88e700, fd989b94, cfcc96bb, 3376f7a9, c4ffd035, a29d0ca1, bef5f4cb) shows `internal/cli/pretty_formatter_test.go` in none of them. The pretty branches of `format_integration_test.go` are likewise untouched by c4ffd035's diff — the "TTY defaults to pretty" and "--pretty overrides piped default" probes keep their `ID:` assertion (`format_integration_test.go:638, 681`).
  - Criterion 7 holds: the JSON detail tests still parse into `map[string]any` before asserting (`json_formatter_test.go:88-90, 166-169, 198-201, …`).
  - The plan's Tests section proposed names like `"it decodes show output when stdout is not a TTY"`; the implementation kept the existing subtest names (`"non-TTY defaults to toon"`, `"--toon overrides TTY default"`) and rewrote only their bodies. The substance the criteria name — decoded-value probes replacing the shape-pinned ones — is delivered; the naming is a preference, not a criterion.

TESTS:
- Status: Adequate
- Coverage: All-optional-fields `show`, bare `show`, `note add`, `note remove`; plus the four rewritten format-detection probes in `format_integration_test.go`. Every assertion is a decoded value or key presence/absence.
- Notes:
  - The tests would fail if the feature broke. A regression to a hand-edited `task{…}` header fails `decodeToonDoc` at `toon_decode_test.go:21`; a dropped notes index column fails `assertToonNoteRows` at `:136-140`; a section silently emptied fails `assertToonRowsEmpty` / `assertToonRelatedRow`.
  - The probes still discriminate toon from pretty without re-pinning a shape: pretty's detail output carries neither a decodable `id` key nor a `notes` section, so `decodeToonDoc` + `assertToonKeysPresent(…, "notes")` separates the formats on structure rather than on a first-line string match — which is what the task's second edge case asked for.
  - Observation, not a finding: Phase 6 later added `TestToonDetailConformance` (`internal/cli/conformance_test.go:1099-1135`), whose first two subtests assert the same key set and the same values over the same `show` document as `TestToonTaskDetailConformance`'s first two, using the same helpers against a different fixture. The overlap is real but was anticipated by the plan ("Phase 6 extends decode coverage to the whole must-parse inventory and every branch; this task covers the task-detail document only"), nothing is broken by it, and the `note add`/`note remove` subtests here are not subsumed — the conformance table's note entries check row counts and the absence of `changed` (`conformance_test.go:1560-1576`) but not index or text values. Recorded so a later consolidation can decide, not raised as a defect in this task.
  - Not over-tested otherwise: no redundant assertions inside the four subtests, no mocking, and the fixture seeds the minimum five tasks needed to give the full task a parent, a blocker and a child.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` only, `t.Run()` subtests in the project's "it does X" phrasing, `t.TempDir()` isolation via `setupTickProjectWithTasks`, `t.Helper()` on every helper (`toon_decode_test.go:18, 32, 38, 59, 73, 82, 91, 115, 130, 145, 176`), and helper placement in a `_test.go` file per the project's convention.
- SOLID principles: Good. `decodeToonNotes` was refactored into the general `toonRows` rather than duplicated (`toon_decode_test.go:31-56`); the new assertion helpers each do one thing and are used 7 to 105 times across the package, so none is orphaned.
- Complexity: Low. No helper exceeds a single loop with one branch.
- Modern idioms: Yes. `slices.Equal` for the string-list comparison, variadic key helpers, `%q`/`%#v` verbs in failure messages that print the offending document.
- Readability: Good. Failure messages name the key and both values, and `decodeToonDoc` prints the whole document on a decode error — which is the diagnostic this work exists to produce.
- Issues: None. The comments on the added helpers (`toon_decode_test.go:30, 36, 113`) each state what the function does and hold true against it; none references a task id, phase or spec section.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settled only by running the four commands from the repository root. Reading confirms the added code compiles by inspection (every symbol it references resolves: `task.FormatTimestamp` at `internal/task/task.go:264`, `RelatedTask` at `internal/cli/format.go:88`, `setupTickProjectWithTasks` at `internal/cli/create_test.go:34`) and that no helper it adds is unused, which is the `unused` linter's most likely complaint in a test file — but neither compilation, test outcome nor formatting can be confirmed without execution.
