TASK: free-text-round-trip-4-4 (tick-75bf6f) — JSON Renders The Filtered Document

ACCEPTANCE CRITERIA:
- Filtered JSON output parses as exactly one JSON object
- The object carries exactly the selected keys — `id` included only when asked for
- Key order is stable across runs for the same selection
- `notes` objects carry `index`, `text` and `created`, with `index` the note's 1-based position
- `tags` and `refs` selected on a task carrying none render `[]`, never `null` and never absent
- `description` selected on a task with no description renders `""`, and `type` renders `""`
- `parent` and `closed` selected on a task carrying neither are absent
- A selection whose every key would be absent produces zero bytes on stdout and exits zero — not `{}`
- Unfiltered `--json` output for `show`, `create`, `update`, `note add` and `note remove` is byte-identical to before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§9.7 requires the resolved format to apply to a filtered document "exactly as it applies to a full `tick show`"; §9.5 "You get exactly the fields you named, in both forms" (no `id` riding along); §9.6 gives the per-name empty rule — each empty name contributes what it would contribute to full output, and a selection whose every name prints nothing prints nothing at all and exits successfully; §4.2 requires each JSON note to carry its 1-based index so a consumer can call `note remove`; §11 forbids byte-level pinning in the machine formats, so the tests assert decoded values and emitted key order rather than full output strings.

IMPLEMENTATION:
- Status: Implemented (drifted from the plan's prescribed shape, deliberately and for the better)
- Location:
  - internal/cli/json_formatter.go:56-87 — `jsonMember`/`jsonObject` with a `MarshalJSON` that writes members in slice order
  - internal/cli/json_formatter.go:93-99 — `FormatTaskDetail` returns `""` when the object has no members
  - internal/cli/json_formatter.go:104-140 — `taskDetailJSONObject`, one ordered key table gated per key through `fields.includes`
  - internal/cli/json_formatter.go:150-169 — `toJSONStrings` / `toJSONNotes` (non-nil empty slices; `index` from the whole-section position)
  - internal/cli/show.go:83-91 — the empty-render guard; the document is printed only when non-empty
  - internal/cli/format.go:111-140 — `selectedSections` narrows each list section once, nil-safe for the unfiltered path
- Notes:
  - The plan's `map[string]any` was replaced by the ordered `jsonObject` under task 4-10 (commit 63b2e740), which the task record itself justifies from §9.7: alphabetical map keys made filtered order disagree with the full document's. The map's two required properties — "only these keys" and "selected but empty" — are both preserved, so this is a strict improvement over what 4-4 specified, not a loss. `map[string]any` no longer appears anywhere on `FormatTaskDetail`'s path.
  - Byte-identity of unfiltered output checked against the pre-task struct (`git show 78e1cb5f:internal/cli/json_formatter.go`): the emission order at json_formatter.go:117-137 is exactly `jsonTaskDetail`'s field order (id, title, status, priority, type, tags, refs, notes, description, parent, created, updated, closed, blocked_by, children, changed); `parent`/`closed` `omitempty` becomes the `!= ""` guards at :125 and :130; `changed`'s pointer-behind-omitempty becomes the `detail.Changes != nil` guard at :135, so a present-but-empty changed list still renders `[]`. `json.MarshalIndent` re-indents a `Marshaler`'s output and re-compacts it with HTML escaping, and the inner `json.Marshal` calls at :73/:77 already escape, so indentation and escaping are unchanged.
  - Only `show.go:83` ever sets `detail.Fields`; `create`, `update`, `note add` and `note remove` render through `outputMutationResult` (helpers.go:28-31) with a nil selection, so they take the full-document path.
  - `changed` is excluded from every filtered document because `includes` (show_fields.go:164-166) is false for a name absent from the registry, and `--field changed` is rejected at parse time (show_fields.go:223). Correct, and pinned by "it never carries the changed key".

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/json_formatter_test.go:1586-1814 — `TestJSONFilteredTaskDetail` covers every criterion at unit level: selected key set, `id` absent unless asked, scalar types, note `index`/`text`/`created`, empty tags/refs/notes as `[]` (`assertJSONEmptyArray` asserts non-nil and zero-length), empty `description` and `type` as `""`, absent `parent`/`closed` omitted, `""` when no key survives, `changed` never in a filtered document, full-document key set unchanged, document-order key emission for three selections, filtered order as the full order restricted to the selection, and the declared full order with and without `changed`.
  - internal/cli/list_show_test.go:1022-1086 — end-to-end through `runShow`: filtered object parses and carries only the selected keys, note `index` survives, empty tags/refs are `[]`, `--field parent,closed` gives empty stdout with exit 0, and key order is document order across two runs (the stronger replacement for the old identical-render check).
  - internal/cli/list_show_test.go:1231-1252 — `--json --field notes.2,title` narrows the section and keeps the real index 2.
  - internal/cli/conformance_test.go:1883-1886, 1218-1241 — the filtered JSON document is in the conformance inventory, and zero-bytes-for-a-vanishing-selection is asserted under every format spelling.
  - The pre-existing full-document assertions (json_formatter_test.go:86-986) are intact and unedited, which is what backs the "unfiltered output unchanged" criterion.
  - `jsonKeyOrder` (json_formatter_test.go:1818-1847) reads emitted order off the raw bytes via `json.Decoder.Token()`, so order assertions cannot be satisfied for free by map marshalling.
- Notes: Overlap between the unit, end-to-end and conformance layers is real but each layer answers a different question (renderer shape, handler wiring, inventory-wide conformance), and each was required by a different task. Not over-tested. No test would still pass if the behaviour it names broke.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.Helper()` on helpers, `t.TempDir()`-backed project fixtures, error wrapping unchanged.
- SOLID principles: Good. One ordered table decides both order and per-key presence, so the full and filtered paths cannot drift; `jsonObject` carries only the ordering concern and `taskDetailJSONObject` only the document's layout.
- Complexity: Low. `taskDetailJSONObject` is a flat sequence of `add` calls with three guards; `MarshalJSON` is a single loop.
- Modern idioms: Yes — generic `selectedItems`, `slices` package, `for i := range 5` elsewhere in the file.
- Readability: Good. The comments at :56-64, :89-92 and :101-103 each hold true against the code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands; reading shows gofmt-conformant formatting and no unused symbols left behind (`jsonTaskDetail` is fully removed, `bytes` and `encoding/json` are both used), but the suite result itself is not readable.
