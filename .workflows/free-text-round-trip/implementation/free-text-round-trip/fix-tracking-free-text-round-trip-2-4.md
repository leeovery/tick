## Attempt 1

ISSUES:
- `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:124-127` — no test asserts the JSON `changed` array's contents for a non-empty change set. The two JSON subtests (`detail_changes_test.go:100-132`) use `&StatusChanges{}` and `nil` only, and no other test in the package sets `TaskDetail.Changes` (verified: `grep "Changes:\|\.Changes\b\|StatusChanges{" internal/cli/*_test.go` matches only `detail_changes_test.go`). An implementation that ignored `detail.Changes.Rows` and always assigned `&[]jsonStatusChange{}` would pass the entire suite. §4.2 requires a JSON consumer to get the same structured answer as a toon one; if this wiring regresses, `tick create --json` (Task 5) reports `"changed": []` while `--toon` shows rows, and nothing fails until an agent acts on the empty list.
  FIX: Add a subtest to `/Users/leeovery/Code/tick/internal/cli/detail_changes_test.go` in `TestTaskDetailChangedSection`, e.g. `"it carries the changed rows in json"`, that renders `detailWithChanges(&StatusChanges{Rows: rows})` through `&JSONFormatter{}`, `json.Unmarshal`s into `map[string]any`, and asserts the `changed` list has two entries whose `id`, `title`, `from`, `to` and `auto` match the `rows` fixture already declared at line 46 — mirroring the toon subtest at lines 51-66 and reusing the same `parsed["changed"].([]any)` unpacking as lines 108-118.
  CONFIDENCE: high

NOTES:
- The "carries no changed section" end-to-end proofs cover toon only; the JSON and pretty nil paths are covered at formatter level instead. That composition is sound (`show` and both `note` paths pass nil, and each formatter's nil branch is unit-tested), so no end-to-end JSON variant is needed here.
- `toonSectionKeys` (`detail_changes_test.go:15-26`) splits the document on `"\n\n"`, which would mis-split a document whose description contained a blank line. The fixture's description has none, and a mis-split would fail loudly rather than pass silently, so it is not a defect — worth knowing if the helper is reused on richer fixtures later.
- `.tick/tasks.jsonl` is modified in the working tree; that is the workflow's own dogfooding state, not part of the change under review.
