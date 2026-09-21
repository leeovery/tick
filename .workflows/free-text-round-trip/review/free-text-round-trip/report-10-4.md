TASK: free-text-round-trip-10-4 (tick-63d1bd) — The Field Selection Type Answers A Nil Receiver One Way

ACCEPTANCE CRITERIA:
- `var sel *FieldSelection; sel.Len()` returns 0 and `sel.Only()` returns `("", false)`; neither panics.
- All five methods of `*FieldSelection` answer a nil receiver: `includes` true, `Positions` nil, `ValidatePositions` a nil error, `Len` 0, `Only` `("", false)`.
- `bareFieldValue` contains no `sel == nil` check and still returns `("", false)` for a nil selection, which the existing `"it is not bare without a selection"` pins.
- A non-nil selection is unaffected: `Len` still counts distinct requested names and `Only` still reports the sole name.
- No shipped output changes: `tick show`, `create`, `update`, `note add`, `note remove` produce byte-identical documents under `--toon`, `--json`, `--pretty`, filtered and unfiltered.
- The diff is confined to `internal/cli/show_fields.go` and `internal/cli/show_fields_test.go`; no caller outside those files changes and no method is renamed, unexported or deleted.
- `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied.

STATUS: complete

SPEC CONTEXT: The specification (`.workflows/free-text-round-trip/specification/free-text-round-trip/specification.md`) does not name `FieldSelection` or the nil-receiver convention — this is a phase-10 consolidation task raised from an architecture finding against the `--field` selection machinery built in phase 4. The governing convention is the one phase 4 task 3 set when it deleted the exported `Selected` in favour of nil-safe `includes`: a nil `*FieldSelection` means "the whole document", and the type, not its callers, answers for that.

IMPLEMENTATION:
- Status: Implemented
- Location: `internal/cli/show_fields.go:178-194` (`Len` and `Only` nil guards plus their doc lines), `internal/cli/show_fields.go:118-131` (`bareFieldValue` with the `sel == nil` guard removed). Commit `98d0703d`, 2 files, +46/-5.
- Notes: All five methods now short-circuit on a nil receiver — `includes` (`:164-166`) true, `Positions` (`:171-176`) nil, `ValidatePositions` (`:284-287`) nil error, `Len` (`:180-185`) 0, `Only` (`:189-194`) `("", false)`. `bareFieldValue` is unchanged in result for every input: with a nil selection `sel.Only()` now returns false at `:119`, taking the same early return the deleted guard took; `sel.Positions(name)` at `:124` was already nil-safe. `Len` and `Only` remain exported and unrenamed, per Do step 3. I enumerated the `Len`/`Only` call sites across `internal/`: production has exactly one (`bareFieldValue` calling `Only` at `:119`); `Len` has no production caller, which the task records as a deliberate keep. The diff touches only `show_fields.go` and `show_fields_test.go`, so no formatter or handler changed. Behaviour for non-nil selections is untouched: `Len` still returns `len(s.names)`, `Only` still gates on `len(s.names) != 1`.

TESTS:
- Status: Adequate
- Coverage: `TestFieldSelectionNilReceiver` (`internal/cli/show_fields_test.go:717-754`) exercises all five methods on `var sel *FieldSelection` in three subtests matching the plan's micro acceptance: `Len()` is 0 (`:730-734`), `Only()` reports not-sole (`:736-741`), and `includes("title")`/`Positions("notes")`/`ValidatePositions(detail)` (`:743-753`). Both new assertions would fail loudly if the guards were dropped — the methods would panic on the nil dereference rather than silently return a wrong value. The existing pins the task names are present and unchanged: `"it is not bare without a selection"` (`:537-541`) still calls `bareFieldValue(detail, nil)` directly, which now depends on `Only`'s nil handling rather than the deleted guard, and `"it accepts a nil selection"` in `TestValidatePositions` (`:654-659`). Non-nil behaviour stays pinned by `TestParseShowArgs` at `:74-82`, `:99-105` and `:145-148`.
- Notes: The `detail` fixture at `:718-726` is fuller than the nil path strictly consumes (`ValidatePositions` returns before reading it), but it keeps the nil assertion non-vacuous against a document that does carry sections — a reasonable choice, not redundancy. No duplicated assertions and no mocking.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in the project's "it does X" phrasing, no testify, guard-clause style consistent with the sibling methods.
- SOLID principles: Good — the nil contract now lives in the type that documents it rather than in caller discipline, which is the point of the task.
- Complexity: Low — one added branch per method, one branch removed from `bareFieldValue`.
- Modern idioms: Yes.
- Readability: Good — each method's doc states what nil answers in one line, matching `includes` and `Positions`.
- Comment accuracy: The added doc lines hold against the code ("A nil selection is the whole document and requests no name"; "...and has no sole name"). No process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied." — `gofmt -l` over both changed files reports nothing, so the formatting half is settled; the suite, vet and lint halves need those three commands run at the change-set pass.
