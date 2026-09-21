TASK: free-text-round-trip-4-10 (tick-46ece2) — Filtered JSON Returns Its Keys In The Document's Order

ACCEPTANCE CRITERIA:
- `tick show <id> --json --field type,tags` returns `type` before `tags`; `--field title,notes.2` returns `title` before `notes`; `--field status,title,id` returns `id`, `title`, `status`.
- For any selection, the filtered document's key order is the full document's order restricted to the selected keys.
- Full `tick show --json` output is byte-identical to today's, `changed` last when a mutation carries it.
- Each key's presence rule is written once: one `parent`/`closed` omit-when-absent guard, one empty-array shape, one `description` guard; `map[string]any` no longer appears on `FormatTaskDetail`'s path.
- The README's "in normal output order rather than the order they were typed" is true of JSON as well as toon and pretty.
- The existing key-set, shape and position tests in `json_formatter_test.go` and `list_show_test.go` pass unedited apart from the order subtest.

STATUS: issues_found

SPEC CONTEXT: §9.7 ("Interaction with the format flags") settles this: a multi-field request produces a document and "the resolved format applies to either exactly as it applies to a full `tick show`" — so a filtered JSON document must carry the same shape and ordering discipline a full one does. §9.6 fixes the two edges the rewrite had to preserve: a field the task does not carry prints what full output prints for it (nothing, for `parent`/`closed`), and "a selection whose every name prints nothing prints nothing at all and still exits successfully" — the `""` return. §9.5 keeps `id` out of a document that did not name it.

IMPLEMENTATION:
- Status: Implemented (later refined by tasks 10-1 and 4-12, which replaced the inline `selectedItems` calls with `detail.selectedSections()` and the string literals with the `field*` constants; the ordered-emission design of 4-10 is intact)
- Location:
  - `internal/cli/json_formatter.go:56-87` — `jsonMember`/`jsonObject` with a `MarshalJSON` that writes members in slice order.
  - `internal/cli/json_formatter.go:93-99` — `FormatTaskDetail`, returning `""` when no key survives.
  - `internal/cli/json_formatter.go:104-140` — the single ordered key table, each key gated once through `fields.includes` (`show_fields.go:164-166`, nil-safe).
- Notes:
  - Order emitted is `id, title, status, priority, type, tags, refs, notes, description, parent, created, updated, closed, blocked_by, children, changed` — identical to the order the deleted `jsonTaskDetail` struct declared, so the full document's key order is unchanged.
  - The fork is genuinely gone: `jsonTaskDetail`, `formatFilteredTaskDetailJSON` and `FieldSelection.Selected` have no remaining occurrences anywhere in the repo, and `map[string]any` appears in no non-test file under `internal/cli`.
  - Per-key rules are written once: the `parent` guard at `:125`, the `closed` guard at `:130`, the always-array shape via `toJSONStrings`/`toJSONNotes`/`toJSONRelated`, and `description` emitted unconditionally at `:124`.
  - `changed` (`:135-136`) is reachable only on a nil selection: `includes("changed")` is false for any non-nil `FieldSelection`, because `addValue` (`show_fields.go:218-239`) rejects any name absent from `showFields` and "changed" is not in it. Pinned by `json_formatter_test.go:1745` ("it never carries the changed key").
  - Byte-identity with the previous struct path follows from the marshalling chain: `marshalIndentJSON` (`json_formatter.go:405-411`) calls `json.MarshalIndent`, which marshals then re-indents the whole compact document, so a custom marshaler's compact `"k":v` members come out with the same 2-space indent and `": "` separator the struct path produced. The empty-array and empty-object forms survive `Indent` unchanged.

TESTS:
- Status: Adequate, with one gap (see FINDINGS)
- Coverage:
  - `json_formatter_test.go:1758` — filtered key order for `type,tags`, `title,notes.2`, `status,title,id`: the first acceptance criterion, exactly.
  - `json_formatter_test.go:1775` — filtered order equals the full document's order restricted to the selection, driven off the full render rather than a literal.
  - `json_formatter_test.go:1793` — full document's declared order, plus the `changed`-last case with `detail.Changes` set.
  - `json_formatter_test.go:1808` — `parent`/`closed` selected but absent drop out of a filtered document.
  - `list_show_test.go:1072` — CLI-level order across two runs, replacing the identical-render comparison that map marshalling satisfied for free.
  - `json_formatter_test.go:1818-1848` — `jsonKeyOrder` reads top-level keys from the raw bytes via `json.Decoder.Token()`, which is what makes any of the above able to fail; `assertJSONKeySet` (`:1850`) is left in place for the presence assertions.
  - The commit is additive on the test surface (`git show --numstat 63b2e740`: `json_formatter_test.go` 90 insertions / 0 deletions; `list_show_test.go` 8/5, the one subtest plus its import), so the sixth criterion holds.
- Notes:
  - Not over-tested: the four new subtests each pin a distinct claim (three fixed selections, the general restriction property, the full order, the absent-key case), and none duplicates an existing key-set assertion.
  - `"it orders a filtered document as the full document restricted to the selection"` computes `want` by filtering the full order down to the keys `got` actually carries, so it observes order only, not presence. That is the right division — presence is pinned by the `assertJSONKeySet` subtests above it — and is not a defect.

CODE QUALITY:
- Project conventions: Followed. Field names come from the `field*` constants (`show_fields.go:30-46`) rather than literals, matching the toon and pretty formatters' gating; stdlib `testing` with `t.Run` "it does X" subtests and `t.Helper()` on `jsonKeyOrder`.
- SOLID principles: Good. `jsonObject` does one thing (ordered emission) and knows nothing about tasks; `taskDetailJSONObject` holds the document's shape and nothing else; `FormatTaskDetail` is reduced to the empty-document decision.
- Complexity: Low. One linear table with three guards, replacing two parallel builders.
- Modern idioms: Yes. Generic `selectedItems`, `bytes.Buffer` accumulation, closure-based gate.
- Readability: Good. The key table reads as the document reads.
- Issues: None. `json.Marshal(member.key)` on a Go string cannot fail, but the error is propagated rather than ignored, which costs nothing and keeps the method honest.

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/json_formatter.go:66 — `jsonObject.MarshalJSON` is the only custom JSON marshaler under `internal/cli` (verified by grepping `MarshalJSON` across `internal`: the other four are `internal/task` and `internal/storage` types on the JSONL path, not this one), and it is the sole producer of the `tick show --json` detail document's raw bytes — yet nothing observes those bytes. Every detail test decodes before asserting (`json.Unmarshal`, or `jsonKeyOrder`'s `json.Decoder`), and both indentation assertions cover struct paths instead: `json_formatter_test.go:830` renders `FormatTaskList`, `json_formatter_test.go:1432` renders `FormatDepTree`. The README byte-for-byte samples do not reach it either — the only two JSON samples are `list` and `start` (`readme_samples_test.go:368` and `:376`). Fix: add a byte-level assertion to `json_formatter_test.go:1793`'s subtest, which already renders `richDetail()` in full and already imports `strings` — e.g. require the render to begin `"{\n  \"id\": \"tick-a1b2\",\n"`, which pins the 2-space indent and the `": "` separator the acceptance criterion's "byte-identical" depends on. — FAILS: the third acceptance criterion is unguarded going forward. Any later edit to `MarshalJSON` that changes spacing — emitting `": "` itself, or returning pre-indented bytes that `json.Indent` then re-indents — silently reshapes `tick show --json` for every downstream consumer with the whole suite green.

UNSETTLED:
- "Full `tick show --json` output is byte-identical to today's, `changed` last when a mutation carries it." — the `changed`-last half is settled by reading (`json_formatter.go:135-136` and `json_formatter_test.go:1793`'s second half). The byte-identity half is a comparison against the pre-change build, which reading cannot perform: it needs the full detail document rendered by `50b83319` (the commit before this task) diffed against the same document rendered by `HEAD` — e.g. build both and `diff <(old show tick-x --json) <(new show tick-x --json)` on a task carrying parent, closed, tags, refs, notes, blockers and children. Reading supports identity (same key order, same guards, same values, and `json.MarshalIndent` re-indents a custom marshaler's compact output exactly as it indents a struct's), but no test in the repo compares those bytes.
