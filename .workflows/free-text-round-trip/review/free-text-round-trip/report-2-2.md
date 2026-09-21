TASK: free-text-round-trip-2-2 (tick-ed0952) — Status Commands Emit The Changed Table In Toon

ACCEPTANCE CRITERIA:
- `tick start` on a task with no children emits `changed[1]{id,title,from,to,auto}:` and one row — never a bare arrow line
- A cascade emits one table containing the requested change and every knock-on, with the requested row first and the count matching the row total
- `auto` renders unquoted and decodes as a Go `bool`; the requested row is `false`, every cascaded row `true`
- A title containing a comma is quoted by the library and decodes back as one value
- An empty change set renders `changed[0]{id,title,from,to,auto}:` and decodes to an empty list
- `tick start --quiet` and its siblings print nothing at all
- Pretty output for `start`, `done`, `cancel` and `reopen` — the single line and the box-drawing tree — is byte-identical to before this task
- `internal/cli/pretty_formatter_test.go` still asserts the arrow line `tick-a3f2b7: open → in_progress` as a golden string, now through `FormatCascadeTransition`
- `grep -rn 'FormatTransition' internal/` returns nothing
- `CLAUDE.md`'s Formatter bullet lists the eight remaining methods and no longer names `FormatTransition`
- Every status command's toon stdout decodes via `toon.DecodeString`
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §7.2 requires one `changed[N]{id,title,from,to,auto}` table for every status-moving command, with the `auto` column carrying what the old trailing `(auto)` marker meant, a title column so no second lookup is needed, and a task appearing at most once. §7.3 keeps the per-command split — `done`, `start`, `cancel`, `reopen` return **only** the `changed` table. §4.1 and §4.3 require pretty's bytes to be unchanged and the shared `baseFormatter.FormatTransition` to be split rather than edited, since its `(id, oldStatus, newStatus)` signature carries no title.

IMPLEMENTATION:
- Status: Implemented (subsequently extended by later tasks in the plan — refusal plumbing, row merging, the `StatusChanges` detail section — none of which undoes this task's shape)
- Location:
  - `internal/cli/toon_formatter.go:137-145` — `toonChangedRow` with `toon:"id"`, `toon:"title"`, `toon:"from"`, `toon:"to"` and `Auto bool \`toon:"auto"\``
  - `internal/cli/toon_formatter.go:146-156` — `buildChangedSection`, returning `emptyToonSection[toonChangedRow]("changed")` (which reflects the tagged field names into `changed[0]{id,title,from,to,auto}:` at `internal/cli/toon_formatter.go:327-334`) when the change set is empty, and the library-encoded section otherwise
  - `internal/cli/toon_formatter.go:158-162` — `ToonFormatter.FormatCascadeTransition` returns only the changed section; no arrow construction and no `TaskID == ""` early return remain
  - `internal/cli/format.go:206-226` — `StatusChange` and `CascadeResult.Changed()`, which seeds the requested row first with `Auto: c.PrimaryAuto` and appends every cascaded entry with `Auto: true`
  - `internal/cli/helpers.go:109-112` — `outputStatusChanges(stdout, fmtr, cr)` writes `FormatCascadeTransition` unconditionally; called from `internal/cli/transition.go:58` after the `fc.Quiet` early return at `internal/cli/transition.go:54-56`
  - `internal/cli/format.go:279-298` — the `Formatter` interface now declares eight methods and no `FormatTransition`; `baseFormatter` (`internal/cli/format.go:301-325`) and `StubFormatter` (`internal/cli/format.go:333-357`) carry none either
  - `internal/cli/pretty_formatter.go:286-302` — pretty's head line `fmt.Fprintf(&b, "%s: %s → %s", ...)` is byte-identical to the deleted `baseFormatter.FormatTransition` format string (confirmed against the deletion in commit 7ddbe8d4), and its `result.TaskID == ""` guard is intact
  - `CLAUDE.md:49` — Formatter bullet lists FormatTaskList, FormatTaskDetail, FormatCascadeTransition, FormatDepChange, FormatDepTree, FormatStats, FormatMessage, FormatRemoval (eight, matching the interface)
- Notes:
  - `grep -rn 'FormatTransition' internal/ CLAUDE.md` returns nothing; `jsonTransition` is likewise gone. The only `→` left in non-test `internal/cli` source is pretty's (`internal/cli/pretty_formatter.go:298,312`) plus the help-text command summaries (`internal/cli/help.go:105-128`), which §3.2 leaves as prose.
  - `create.go` and `update.go` no longer call `outputStatusChanges` — later plan tasks folded their changes into the detail document as `StatusChanges` (`internal/cli/format.go:143-152`), which is §7.4's end state, not drift from this task.
  - `CascadeResult.Changed()` routes through `statusChangeSet` (`internal/cli/transition.go:109-147`), which drops a task whose `From == To`. For an empty `CascadeResult{}` that yields zero rows and hence the count-zero header — exactly what `format_test.go`'s rewritten expectation asserts — and it implements §7.2's "a task that ends where it started carries no row".
  - The `*CascadeResult` deref at `internal/cli/transition.go:58` is safe: `Store.Mutate` (`internal/storage/store.go:174-214`) always invokes the closure, and the closure either sets the pointer or returns the error checked at `internal/cli/transition.go:49-51`.

TESTS:
- Status: Adequate
- Coverage:
  - All eight micro-acceptance tests named in the plan exist. `internal/cli/toon_decode_test.go:283-373` (`TestToonStatusChangeConformance`) covers the one-row table for `tick start`, the two-row cascade with the requested row first, `auto` false/true per row, the comma-bearing title `"Parse, the header"` round-tripping as one value, `buildChangedSection(nil)` equalling `changed[0]{id,title,from,to,auto}:` and decoding to zero rows, and `--quiet` producing empty stdout.
  - `auto` is asserted as a Go `bool` via a type assertion on the decoded value (`internal/cli/toon_decode_test.go:345-347`), which is what proves it was emitted unquoted.
  - Formatter-level coverage: `internal/cli/cascade_formatter_test.go:13-63` decodes downward, upward and single-change tables through `toon.DecodeString`.
  - Pretty goldens are intact: `internal/cli/pretty_formatter_test.go:335-342` asserts `tick-a3f2b7: open → in_progress` through `FormatCascadeTransition`; `internal/cli/transition_test.go:389-404` asserts the same line end-to-end under `IsTTY: true`; `internal/cli/helpers_test.go:314-331` and `internal/cli/cascade_formatter_test.go:66-141` keep the box-drawing tree goldens.
  - End-to-end decode for all four commands: `internal/cli/conformance_test.go:1440-1509` decodes `start`, `done`, `cancel` and `reopen`, each with and without a cascade.
  - `TestAllFormattersProduceConsistentTransitionOutput` is gone rather than weakened, as the task's edge-case note required.
  - Every toon assertion goes through `decodeToonDoc` → `toon.DecodeString` (`internal/cli/toon_decode_test.go:17-28`), so a table that stopped parsing would fail rather than pass on a substring match.
- Notes: The single-change toon table is asserted at three layers (`cascade_formatter_test.go:52-63` formatter, `helpers_test.go:295-312` helper wiring, `format_integration_test.go:82-104` and `toon_decode_test.go:289-305` end-to-end). The layers differ, so each would catch a distinct break; only `cascade_formatter_test.go:52-63` and the toon arm of `TestAllFormattersCascadeEmptyArrays` (`internal/cli/cascade_formatter_test.go:274-279`) assert the same thing at the same layer, and that second one earns its place by checking all three formatters agree on the no-cascade case.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` with `t.Run` "it does X" subtests and `t.Helper()` on helpers; pretty-format assertions use `IsTTY: true` or `--pretty`, toon assertions use `--toon`; `changed` built by the same count-zero-literal pattern `buildRelatedSection`/`buildNotesSection` already use.
- SOLID principles: Good — removing `FormatTransition` narrows the interface to the one status-output method (interface segregation), and the row shape lives in the toon formatter rather than in the shared base.
- Complexity: Low — `buildChangedSection` is a length check plus a slice map; `FormatCascadeTransition` is two lines.
- Modern idioms: Yes — generic `emptyToonSection[T]`, struct conversion `toonChangedRow(c)` in place of field-by-field copying.
- Readability: Good — the table's shape is readable from `toonChangedRow`'s tags alone.
- Issues: None. Comments in the changed code hold: the `baseFormatter` doc comment was updated to name only `FormatDepChange` and `FormatRemoval` (`internal/cli/format.go:301-303`), and no comment still describes an arrow line in toon.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs those three commands run at HEAD; reading confirms the code compiles logically and the assertions match the implementation, but not that the suite, vet and gofmt are clean.
