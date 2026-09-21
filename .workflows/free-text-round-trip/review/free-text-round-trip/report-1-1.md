TASK: free-text-round-trip-1-1 (tick-36cfbf) — Task Fields Become Top-Level Named Fields

ACCEPTANCE CRITERIA:
- `FormatTaskDetail` output decodes without error via `toon.DecodeString` for a task carrying every optional field and for one carrying none
- The decoded document carries `id`, `title`, `status`, `priority`, `created` and `updated` as top-level keys, and carries no `task` key
- `type`, `parent` and `closed` appear in the decoded document exactly when the task carries them — identical presence rules to before the change
- Emitted field order is `id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`
- No `strings.Replace` call remains in `buildTaskSection`, and the head section is produced by a single `toon.MarshalString` call
- A title containing a comma, a colon and a leading dash decodes back byte-identical to the stored title
- `created` and `updated` decode as strings, not numbers
- `encodeToonFields` returns `""` for a value the encoder rejects, and `FormatTaskDetail` then emits the remaining sections with no leading blank line
- Pretty output and JSON output for the same `TaskDetail` are unchanged
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §5.1 names the cause — the head was marshalled as a one-element array and the `[1]` deleted from the header by a string replace, producing `task{…}:`, a table header with no table, which a standard TOON reader rejects on line 1. §5.2 fixes the required shape: the task's own fields sit at the top level as named fields with no wrapping key, peers of the collection sections, with `type`, `parent` and `closed` present only when the task carries them. §5.3 rejects the one-row table (positional reading, no compression at one row) and §5.4 rejects a wrapping key (it split task fields across two depths, storage is flat, and §9's field flag falls out of the flat form). §3.1 lists the five commands that emit this document and — added by later work in the same spec — requires that a value TOON cannot carry fails the command rather than vanishing from the document.

IMPLEMENTATION:
- Status: Implemented (drifted in one place, deliberately and soundly — see Notes)
- Location:
  - `internal/cli/toon_formatter.go:228` — `buildTaskSection` assembles the ordered `[]toon.Field` (`id`, `title`, `status`, `priority`; `type` when non-empty; `parent` when non-empty; `created`, `updated`; `closed` when non-nil) and returns `encodeToonFields(fields...)`; no wrapper object, no `task` key, no `strings.Replace`, no `"task:"` fallback.
  - `internal/cli/toon_formatter.go:266` — `encodeToonFields` marshals `toon.NewObject(fields...)` through a single `toon.MarshalString` call.
  - `internal/cli/toon_formatter.go:286` — `joinToonSections` drops empty sections before joining with a blank line, so an empty head produces no leading blank line.
  - `internal/cli/toon_formatter.go:71` — `FormatTaskDetail` adds the head through `toonDoc.add` and renders via `doc.join`.
  - `grep -rn 'task\[1\]\|encodeToonSingleObject\|task{' internal/ README.md` returns no hits: no code path strips a `[1]` from any header, and the README's detail sample (`README.md:471-472`) opens `id: tick-a1b2` / `title: Setup auth`.
- Notes: Two shapes moved past the task's literal wording, both under later tasks of this same plan and both sound.
  1. `encodeToonFields` and `buildTaskSection` now return `(string, error)` rather than `""` alone. The task's criterion 8 ("`FormatTaskDetail` then emits the remaining sections") was superseded by spec §3.1, which requires a value TOON cannot carry to fail the command rather than disappear from the document — implemented in phase 9 (`toonEncodeError`, `toon_formatter.go:370`; `toonDoc.add`, `toon_formatter.go:404`). The half of the criterion that still applies holds: `encodeToonFields` returns `""` on refusal (`toon_formatter.go:266-272`), and the empty-head-no-blank-line behaviour survives through `joinToonSections` on the field-selection path.
  2. `buildTaskSection` takes a `*FieldSelection` and gates each field through `sel.includes` (phase 4, §9). A nil selection includes every name (`show_fields.go:164-166`), so the unfiltered document is unchanged by the gate.
  Neither loses anything the task required in substance, and the head is still one `toon.MarshalString` call on the success path (the per-field re-marshal in `fieldRefusal`, `toon_formatter.go:276`, runs only after a refusal, to name the offending field).

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/toon_formatter_test.go:724` "it emits the task's own fields as top-level named fields" — decodes the full document, asserts `id`, `title`, `status`, `priority` (`float64(0)`), `created`, `updated`, asserts `task` absent, and asserts emitted key order equals `id,title,status,priority,type,parent,created,updated,closed` by reading the head's raw lines (the only way to check order, since a decoded map has none).
  - `:172` "it omits type, parent and closed when the task does not carry them" and `:634` "it includes type, parent and closed when the task carries them" — the two presence branches.
  - `:767` "it round-trips a title containing a comma, a colon and a leading dash" — `"- fix: retries, backoff and jitter"` decodes back byte-identical.
  - `:789` "it emits created and updated as quoted timestamps" — asserts the decoded values are `string` (type-asserted, so a number fails) and equal `task.FormatTimestamp`.
  - `:821` "it decodes the whole document when sections are joined by blank lines" — full detail with blocked_by, children and notes decodes to one map carrying every section key.
  - `:845` "it returns the encoder's error from encodeToonFields" — `""` plus a refusal naming the field (the evolved form of the planned empty-string test).
  - `:892` "it omits the head rather than emitting a blank line when no top-level field is selected" — empty head, then `joinToonSections` yields the notes section with no leading blank line.
  - `:116` "it formats show with all sections" — pins section composition (5 sections) and decodes each, covering the `parent`-present/`type`-absent mix.
  - `internal/cli/toon_decode_test.go:17` — `decodeToonDoc` helper exists as planned: `t.Helper()`, `t.Fatalf` with the document text on decode error and on a non-`map[string]any` value.
  - End-to-end: `internal/cli/conformance_test.go:345,353` run `show` on a task carrying every optional field and on one carrying none through the real CLI and decode the output, so criterion 1 is covered at command level as well as formatter level.
- Notes: No over-testing found. The overlap between `:116`, `:634` and `:724` is not redundant — they cover section composition, the optional-field branch and field order respectively. No test pins a full-output string for this document.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests in "it does X" form, `t.Helper()` on helpers, `fmt.Errorf`/wrapped errors via `toonEncodeError.Unwrap`, formatter helpers kept unexported in `internal/cli`.
- SOLID principles: Good — `encodeToonFields` is the single named-fields encoder and is reused by stats (`toon_formatter.go:113`), the dep-tree summary (`:186`), the focused dep-tree head (`:198`) and the description field (`:103`); `buildTaskSection` only decides which fields exist.
- Complexity: Low — one linear field-assembly function with a small `add` closure; presence rules read as three guarded calls.
- Modern idioms: Yes — generics on the section helpers, `reflect.TypeFor`, `strings.Cut`, `range` over int, `slices` in tests.
- Readability: Good — quoting, ordering and escaping are all the library's job, which is what the spec asked for.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands at the repository root; verification here was by reading.
