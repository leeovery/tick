TASK: free-text-round-trip-4-11 (tick-f31314) — Each Field Is Declared Once In The Registry

ACCEPTANCE CRITERIA:
- `showSections` no longer exists; each list section's `noun`, `length` and `items` sit on its single `showFields` entry.
- `ValidatePositions` walks one ordered list of section names, so a list section added to the registry is range-checked without a second edit.
- `showFieldDescription` and `showFieldUnknown` are gone; `description` and the nine other single-value fields share one kind, and an unrecognised name is detected by the map lookup failing.
- One gate answers "does this name belong in the document", and it is nil-safe; no exported half survives to be reached for.
- A `TaskDetail` with nil `Fields` — what `create`, `update`, `note add` and `note remove` produce — renders in toon, pretty and JSON without panicking.
- Behaviour is unchanged: `go test ./...` passes with no test file edited beyond the four `Selected` call sites, and `go vet ./...` plus `golangci-lint run ./...` are clean.

STATUS: complete

SPEC CONTEXT: §9 governs `tick show --field`. §9.1 fixes the recognised vocabulary as the document's own names, with a positional suffix attaching only to list-holding sections and anything else taking the unrecognised-name error. §9.6 requires a non-zero exit and a message naming the range for a position that resolves to nothing (`notes.4` on two notes, `notes.0`, `tags.1` on a tag-less task), phrased as `note remove` phrases it. §9 also notes that `create`, `update`, `note add` and `note remove` emit the same detail document but take no field selection — the nil-`Fields` documents criterion 5 covers. This task changes no behaviour the spec describes; it consolidates the declarations behind it.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/show_fields.go:17-26` — `showField` now carries `bare`, `items`, `noun`, `length`, with `isList()` deriving the kind from `length != nil`; the `showFieldKind` enum, `showFieldUnknown`, `showFieldScalar`, `showFieldDescription` and `showFieldList` are all gone (`grep -rn --include='*.go' 'showFieldKind\|showFieldScalar\|showFieldDescription\|showFieldUnknown\|showFieldList\|showSections' .` → no hits).
  - `internal/cli/show_fields.go:50-66` — the five list entries now hold their own `noun`, `length` and (where the items are values) `items`; the ten single-value fields hold only `bare`.
  - `internal/cli/show_fields.go:68-70` — `showListSections` is the single ordered declaration of document order.
  - `internal/cli/show_fields.go:223` and `:229` — recognition is the map lookup's `ok`; a positional suffix additionally requires `field.isList()`.
  - `internal/cli/show_fields.go:162-166` — `includes` is the sole gate and is nil-safe; the exported `Selected` is deleted (`grep -rn --include='*.go' '\.Selected(' .` → no hits). Nothing outside package `cli` refers to `FieldSelection` (`grep -rn --include='*.go' 'FieldSelection' . | grep -v internal/cli/` → no hits).
  - `internal/cli/show_fields.go:284-298` — `ValidatePositions` walks `showListSections`, reading `noun`/`length` off the registry entry; the message format string is byte-identical to the one it replaced.
  - Gate call sites all route through `includes`: `internal/cli/json_formatter.go:111`, `internal/cli/pretty_formatter.go:147,187,191,195,199,207`, `internal/cli/toon_formatter.go:78,82,86,90,94,102,231` (seven of these arrived later, with task 4-12).
- Notes: Criterion 2 reads "without a second edit", and strictly a new list section still needs its name appended to `showListSections` — map iteration cannot supply the document order the error message depends on, which is why the plan's Do prescribed the ordered slice. The silent-divergence failure the task exists to close is closed by a drift guard rather than by deriving the order: `TestShowListSectionsMatchRegistry` (`internal/cli/show_fields_test.go:606-619`) fails the suite when a registry list entry is missing from `showListSections` or vice versa. Same pattern the project already uses for `commandFlags` versus the help registry. Nothing is left silently unchecked, so the criterion is met in substance.
- `bareFieldValue`'s `bare == nil` branch (`:127`) and `barePositionValue`'s `items == nil` branch (`:137`) were left as they stood, as the task directed.

TESTS:
- Status: Adequate
- Coverage:
  - `TestTaskDetailWithoutFieldSelection` (`internal/cli/show_fields_test.go:756-798`) — the one new test the task named. It asserts `detail.Fields` is nil, then renders through `ToonFormatter`, `PrettyFormatter` and `JSONFormatter`, checking each output carries the ID, title, description, tag, ref, note text, child ID and blocker ID. All three paths reach `includes` with a nil receiver (toon via `FormatTaskDetail`/`buildTaskSection`, pretty via `prettyDetailHeader`/`prettyDetailBlocks`, JSON via `taskDetailJSONObject`), so a non-nil-safe gate panics the test rather than merely changing output. Criterion 5 is pinned.
  - `TestShowListSectionsMatchRegistry` (`:606-619`) — extra over the task's test list, and it is what makes criterion 2 hold: it enumerates `showFields` entries with `isList()` true and compares the sorted set against `showListSections`.
  - Behaviour pins are untouched and still exercise the consolidated registry: `TestValidatePositions` (`:621-715`) keeps the exact messages including `notes.3 out of range: task has 2 note(s)`, plus `notes.0`, `notes.-1`, `tags.3`, `refs.2`, `children.2`, `blocked_by.2`, the zero-length `tags.1` case, first-failure-in-document-order and lowest-position-within-a-section. `TestParseShowArgs` (`:31`) covers recognition and rejection through the new `ok` lookup.
  - Test-file edits in the task's commit (49449a49) are exactly the four `Selected` → `includes` call sites (`show_fields_test.go:112`, `:129`, `:137`, `:167` at the time) plus the two new test functions — no expectation was rewritten to fit the refactor.
- Notes: `TestShowListSectionsMatchRegistry` compares sorted sets, so it guards membership but not position; document order is pinned only for the refs/notes pair by "it reports the first failure in document order whatever the argument order". That coverage shape predates this task (it guarded `showSections` identically) and pinning all ten pairs would be redundant. Not over-tested: the new nil-selection test asserts one property per format with no redundant setup.

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing`, `t.Run()` subtests, "it does X" subtest naming, registry-plus-drift-test pattern matching `commandFlags`/help. `slices.Sorted(slices.Values(...))` is the modernize-clean idiom.
- SOLID principles: Good. One type owns one field's description; the gate has one implementation; `isList()` is the single predicate over the kind distinction.
- Complexity: Low. `ValidatePositions` is two nested loops over explicit data; `addValue`'s branches are comma-separated-name and positional-suffix with no third case.
- Modern idioms: Yes. Map-lookup `ok` for recognition, generic `selectedItems`, `strings.SplitSeq`.
- Readability: Good. The doc comments on `showField` (`:12-16`), `showFields` (`:48-49`), `showListSections` (`:68-69`) and `includes` (`:162-163`) each hold against the code — the claim that `showListSections` is "the order the toon document renders them" matches `toon_formatter.go:78-95` (blocked_by, children, tags, refs, notes), and "items ... is nil for a section whose items are rows rather than values" matches the `blocked_by`/`children` entries. No process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Behaviour is unchanged: `go test ./...` passes with no test file edited beyond the four `Selected` call sites, and `go vet ./...` plus `golangci-lint run ./...` are clean." — the test-file-edit half is settled by reading the commit diff (four call-site edits plus two added tests, no expectation rewritten). The tooling half needs `go test ./...`, `go vet ./...` and `golangci-lint run ./...` executed to settle.
