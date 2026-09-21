TASK: free-text-round-trip-2-4 (tick-d7fc96) — The Detail Document Carries A Changed Section

ACCEPTANCE CRITERIA:
- A `TaskDetail` with `Changes != nil` renders `changed[N]{id,title,from,to,auto}:` in toon, positioned after `notes` and before `description`
- A `TaskDetail` with `Changes` non-nil and no rows renders `changed[0]{id,title,from,to,auto}:`, which decodes to an empty list
- A `TaskDetail` with `Changes == nil` renders no `changed` key in toon or JSON and appends nothing in pretty
- JSON renders `"changed": []` for the non-nil empty case and omits the key entirely for the nil case
- The full toon document decodes via `toon.DecodeString` with the `changed` section present, and the JSON document parses as a single object
- Pretty's rendered detail plus blocks is byte-for-byte what `Fprintln(detail)` followed by `Fprintln(block)` per block produced, including blank-line spacing
- `show`, `note add` and `note remove` output carries no `changed` section in any format
- The section's position in document order is the same for every caller
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: issues_found

SPEC CONTEXT:
§7.4 requires that where a command produces both a record and status changes the result is one document with the changes as a section inside it, and that the section is always present — `changed[0]{...}` when nothing moved — so a reader never branches on whether it exists. §7.3 requires `show`, `note add` and `note remove` to carry no `changed` section at all, even though all four mutating commands share one detail helper (§3.1). §4.1 keeps pretty's bytes as they are: relocating the trailing cascade output inside `FormatTaskDetail` must be a relocation, not a change. §5.2 fixes the rest of the document; the always-present rule belongs to `changed` alone.

IMPLEMENTATION:
- Status: Implemented (with two later, deliberate refactors on top)
- Location:
  - `internal/cli/format.go:106-108` — `Changes *StatusChanges` on `TaskDetail`, nil meaning no section
  - `internal/cli/format.go:142-151` — `StatusChanges{Blocks []CascadeResult}` with `Rows()` deriving the merged table
  - `internal/cli/toon_formatter.go:98-100` — `changed` section added after `notes` (`:94-96`) and before `description` (`:102-104`)
  - `internal/cli/toon_formatter.go:147-157` — `buildChangedSection`, count-zero header derived from `toonChangedRow` via `emptyToonSection` (`:327-334`)
  - `internal/cli/json_formatter.go:135-137` — `changed` member appended only when `Changes != nil`; `toJSONStatusChanges` (`:303-309`) always returns a non-nil slice, so the empty case marshals as `[]`
  - `internal/cli/pretty_formatter.go:231-235` — one `"\n"` + block per `Changes.Blocks` entry, nothing when there are none
  - `internal/cli/helpers.go:17-30` — `outputMutationResult` takes `changes *StatusChanges` and sets it on the detail; `printDocument` (`:34-41`) supplies the single trailing newline
  - Callers: `internal/cli/create.go:277-278` and `internal/cli/update.go:397-398` pass a non-nil `&StatusChanges{Blocks: blocks}`; `internal/cli/note.go:87` and `:145` pass nil; `internal/cli/show.go:64,81-82` builds its detail inline, leaving `Changes` nil by construction
- Notes: Two drifts from the task's "Do" list, both delivered by later plan tasks and both sound. (1) `StatusChanges` stores only `Blocks` and derives `Rows()` — task 2-7 ("Derive Each View Of A Command's Status Movements Instead Of Storing Both"); the two views can no longer disagree. (2) JSON renders through the ordered `jsonObject` in `taskDetailJSONObject` rather than the planned `jsonTaskDetail` struct with a `*[]jsonStatusChange` field — task 4-10 ("Filtered JSON Returns Its Keys In The Document's Order"); the nil/empty distinction the plan protected with a pointer is now carried by whether the member is appended at all, which is stronger. Pretty's byte equivalence holds: `detail + "\n" + block` through one `Fprintln` reproduces the old `Fprintln(detail)` + `Fprintln(block)` exactly, per block.

TESTS:
- Status: Adequate, with one gap
- Coverage:
  - `internal/cli/detail_changes_test.go:57-99` — toon: rows present, count-zero header decoding to an empty list, absent when nil, and section order asserted as `id, blocked_by, children, notes, changed, description`
  - `internal/cli/detail_changes_test.go:101-164` — JSON: `changed` is `[]` and never null for the non-nil empty case, carries the rows for the populated case, and is absent (not null) for nil
  - `internal/cli/detail_changes_test.go:172-192` — pretty: one block and two blocks each asserted byte-for-byte against `body + "\n" + FormatCascadeTransition(block)`
  - `internal/cli/detail_changes_test.go:195-223` — end-to-end toon for `note add`, `note remove` and `show`, each decoded and asserted to carry no `changed` key; `internal/cli/conformance_test.go:1575` repeats the `note remove` assertion in the conformance inventory
  - `internal/cli/detail_changes_test.go:229-250` — `outputMutationResult` passes changes through to the rendered document
  - `internal/cli/toon_formatter_test.go:1347-1364`, `internal/cli/json_formatter_test.go:1744-1749` and `:1794-1807` — the `changed` key under the field-selection surface: rendered for a mutation document, dropped under a selection, and appended last in JSON key order
- Notes: The one planned micro-acceptance test that is not in the suite — "it appends nothing in pretty when the detail carries no blocks" — was written in this task's commit (828d2321) and deleted by task 2-7 (8e579a68) rather than rewritten against the new shape. See FINDINGS.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` "it does X" subtests, `t.TempDir()`-based fixtures, table-free focused assertions; formatter changes stay inside the `Formatter` interface and the handler signature is unchanged
- SOLID principles: Good — `StatusChanges` holds one representation and derives the other; each formatter owns its own rendering of it; `outputMutationResult` remains the single detail-rendering path for all four mutating commands
- Complexity: Low — one `nil` guard per formatter, one loop in pretty
- Modern idioms: Yes — generic `emptyToonSection[T]` derives the count-zero columns from the row struct's tags, so the header cannot drift from the row
- Readability: Good — the comments at `format.go:106-108`, `format.go:142-143` and `helpers.go:13-16` each state the nil/non-nil contract accurately and hold against the code
- Issues: None. The toon path gates the section on `Changes != nil` alone while JSON additionally routes it through `fields.includes("changed")`, but no reachable state has both a non-nil `Changes` and a non-nil `Fields` (`--field` exists only on `show`, which never sets `Changes`), so the two cannot diverge in output.

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/detail_changes_test.go:167 — `TestTaskDetailChangedSectionPretty` covers one block and two blocks but no longer covers a non-nil `Changes` carrying zero blocks; add a subtest asserting `f.FormatTaskDetail(detailWithChanges(&StatusChanges{})) == f.FormatTaskDetail(detailWithChanges(nil))`, restoring the case commit 8e579a68 deleted when `StatusChanges.Rows` became a derived method. — FAILS: that state ships on every `tick create` and `tick update` in a terminal that cascades nothing (`internal/cli/create.go:277` and `internal/cli/update.go:397` always build a non-nil `&StatusChanges{}`, with `Blocks` empty unless a parent reopened), and nothing observes it: a regression at `internal/cli/pretty_formatter.go:231-235` that emitted a stray `"\n"` or a `Cascaded:` header for an empty block list would leave the suite green. The only two end-to-end pretty assertions on mutation output — `internal/cli/create_test.go:1490-1501` and `internal/cli/update_test.go:1440-1450` — both exercise the cascading path and both use `strings.HasSuffix`, which an extra leading blank line satisfies; every other pretty detail assertion (`internal/cli/pretty_formatter_test.go`) uses a nil `Changes`, and pretty is not a conformance driver (`internal/cli/conformance_test.go:1571-1573` declares only toon and json).

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — requires executing the three commands from the repo root; verification here was by reading only.
