TASK: free-text-round-trip-4-2 (tick-4d2f1d) — One Field Returns Its Bare Value

ACCEPTANCE CRITERIA:
- `--field description` prints the stored description followed by exactly one newline, with no indentation, no quoting and no header
- The same request under `--json`, under `--pretty` and under `--toon` produces byte-identical stdout
- A multi-line description goes out with its own newlines intact and no per-line prefix
- A value beginning with `- `, a value containing a colon, and a value that looks like a section header all go out unescaped
- `--field closed` on a task that is not closed, and `--field type` on a task with no type, print zero bytes and exit zero — not a blank line
- `--field id` on a partial-ID request prints the resolved full ID
- `--field title,title` takes the bare path
- `--field notes`, `--field tags`, `--field refs`, `--field children` and `--field blocked_by` do not take the bare path
- `--field title,status` does not take the bare path
- Every non-bare selection still renders the full document unchanged
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§9.2 splits the answer by how many fields resolve: one field that resolves to exactly one value goes out as its own bytes plus a single newline, as the bare task ID already does; a single name that is a list section is a document, because a list has no bare form and the flag is a projection rather than a single-value extractor. §9.4 makes the task's own fields selectable exactly like sections, with nothing wrapping the result. §9.6 makes an empty value an empty stream rather than a blank line, exiting zero. §9.7 exempts the bare form from `--json`/`--pretty`/`--toon` — a bare value is not a document, and honouring a format flag would re-quote the string the flag exists to hand over unquoted; §3.1 carries the same exemption into the must-parse inventory.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/show_fields.go:118 — `bareFieldValue(detail, sel)`: `sel.Only()` gates on exactly one name (nil-safe), then `barePositionValue` (the Task 6 position extension the plan anticipated), then the registry's `bare` renderer, which is nil for every list section.
  - internal/cli/show_fields.go:50-66 — the `showFields` registry carries `bare` for the ten scalars plus `description`; `priority` through `strconv.Itoa`, `created`/`updated` through `task.FormatTimestamp`, `closed` through `bareClosed` (show_fields.go:108) which yields `""` for an absent timestamp.
  - internal/cli/show.go:71-76 — `RunShow` calls `bareFieldValue` after `showDataToTaskDetail`, writes through `fmt.Fprintln(stdout, value)` only when the value is non-empty, and returns before the quiet branch and before the formatter.
  - internal/cli/show.go:44-46 — `--quiet` alongside a selection is refused before store access (§9.8), so the bare check preceding the quiet branch is not reachable with both set.
- Notes:
  - The code has legitimately moved past the task's text: positions (`notes.2`, `tags.1`) route through the same `bareFieldValue` as the plan's Context anticipated for Task 6, and `showFields` became a single registry under Task 4-11. Both are later tasks in the same phase; neither removes anything this task required.
  - Criterion "Every non-bare selection still renders the full document unchanged" was time-bound to the moment before Task 4-3 landed — non-bare selections now render the filtered document by design (§9.2). The substance this task owed — non-bare selections do not take the bare path — holds, and is asserted at internal/cli/list_show_test.go:866-879.
  - Bare and document forms stay consistent on absence: the toon document omits `type`, `parent` and `closed` when the task lacks them (internal/cli/toon_formatter.go:234-254) and the bare form prints zero bytes for the same three, matching §9.6.
  - The format-independent rendering of `created`/`updated` (stored timestamp form, not pretty's shortened form) is correct for a value handed over for write-back, and is the direct consequence of §9.7's bypass.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/show_fields_test.go:402 `TestBareFieldValue` — unit-covers all ten scalars plus `description` against a fixture detail, the absent optional (`closed` nil → `"", true`), the repeated name, all five list sections as not-bare, the two row-position sections (`children.1`, `blocked_by.1`) as not-bare, two names as not-bare, and the nil selection.
  - internal/cli/list_show_test.go:747 `TestShowBareField` — end-to-end through `runShow` (which drives `App.Run`, so global format flags really parse): exact-byte assertions for a single field, a multi-line description, a dash-leading title, a header-shaped/colon-carrying description, the `--json`/`--pretty`/`--toon` equality loop, absent `closed`, absent `type`, empty description, the partial-ID `--field id` resolution, `priority` as a number, `title,title`, and four non-bare selections.
  - internal/cli/conformance_test.go:1179-1270 — the inventory pins the bare bytes under every format spelling and declares both the bare value and the zero-byte selection exempt from the must-parse table (§3.1).
- Notes: The unit and end-to-end layers overlap on the repeated-name and absent-value cases, but the plan asked for both layers deliberately (bare-value semantics at the function, exact stdout bytes at the CLI boundary) and the e2e assertions are byte-exact rather than duplicate reformulations. Each bare test would fail if the feature broke: they compare whole stdout, so a stray blank line, an added header or a re-quoted value all fail.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run()` "it does X" subtests, `t.TempDir()`-based project fixtures, handler signature and `fmt.Fprintln` output convention unchanged; the bare path mirrors `Fprintln(stdout, id)` at internal/cli/helpers.go:18 as the spec cites.
- SOLID principles: Good — the registry keeps each field's bare rendering next to its list metadata, and `RunShow` gains one branch rather than a second route; positions extended `bareFieldValue` instead of adding a parallel path.
- Complexity: Low — `bareFieldValue` is three guarded returns; `barePositionValue` is two.
- Modern idioms: Yes — `slices.Sorted`/`slices.Compact` in `selectedItems`, generic over the item type.
- Readability: Good — the doc comments on `bareFieldValue` and `barePositionValue` state the ok-false conditions exactly as the code enforces them, and carry no process-artifact references.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settling it needs those three commands run at the repo root; reading shows formatted, vet-clean-looking code and assertions consistent with the implementation, but pass/fail is an execution result.
